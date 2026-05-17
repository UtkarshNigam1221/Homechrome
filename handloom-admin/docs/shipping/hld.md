# Shipping — High-Level Design

## Goals

- Replace Shiprocket aggregator with **Delhivery direct courier** for forward shipments, reverse pickups, COD reconciliation, NDR handling.
- Keep shipping logic **carrier-agnostic** at the service layer so a future second carrier (Bluedart, DTDC, …) can drop in without touching services.
- **Asynchronous-first**: manifest, COD pull, and rate refresh run on EventBridge cron; admin actions trigger Lambda invokes for long work.
- **Cost discipline**: rate matrix cached in DynamoDB, pincode zone cached with 7-day TTL, daily batch consolidates carrier calls.

## Component map

```
┌────────────────────────────────────────────────────────────────┐
│ Storefront (homechrome-store, Next.js 16)                      │
│   PDP            → <PincodeChecker> → /catalog/check-pincode    │
│   /track         → ShippingService.TrackShipment                │
│   /cart, /checkout → cart.shipping_charge from backend          │
│   /account/orders → <ReturnStatusBadge>                         │
│   /return-policy (static)                                       │
└────────────────────────────────────────────────────────────────┘
                              │ HTTPS
                              ▼
┌────────────────────────────────────────────────────────────────┐
│ Admin Frontend (handloom-admin-frontend, React 19 + Vite)      │
│   /shipping/rates           → ShippingAdminHandler              │
│   /shipping/cod-remittance  →   ↓                               │
│   /shipping/pickups         →   ↓                               │
│   /shipping/ndr-queue       →   ↓                               │
│   /shipping/returns         → OrderHandler.ReturnRoutes         │
│   /orders/[id] (extended)   → +PriorityToggle +shipping section │
└────────────────────────────────────────────────────────────────┘
                              │ HTTPS
                              ▼
┌────────────────────────────────────────────────────────────────────────────┐
│ Backend (handloom-admin, Go 1.25 on AWS Lambda / monolith locally)         │
│                                                                            │
│  handler/ (HTTP boundary)                                                  │
│    store/{webhook,catalog,tracking}_handler.go                             │
│    shipping_admin_handler.go (admin)                                       │
│    order_handler.go (admin — return routes)                                │
│    cron/{pickup_batch,cod_remittance,rate_refresh}_handler.go              │
│         ▼                                                                  │
│  service/ (business logic)                                                 │
│    ShippingService — courier.Gateway + pincode cache + rate lookup         │
│    ManifestService — per-order + daily batch                               │
│    NDRService — auto re-attempt + escalation                               │
│    CODReconciliationService — daily UTR matching                           │
│    RateTableService — weekly matrix refresh (manual override-aware)        │
│    ReturnService — admin-initiated reverse pickup                          │
│         ▼                                                                  │
│  repository/dynamodb/ (persistence)                                        │
│    ShipmentRepository (orders table)                                       │
│    ReturnRepository (orders table)                                         │
│    ShippingRateRepository (shipping table)                                 │
│    PincodeRepository (shipping table)                                      │
│    CODRemittanceRepository (shipping table)                                │
│    OrderRepository.UpdateCODRemittance (extended)                          │
│         ▼                                                                  │
│  gateway/courier/ (carrier-agnostic interface + canonical types)           │
│         │                                                                  │
│         ▼                                                                  │
│  gateway/delhivery/ (HTTP client + DevClient + status mapping)             │
└────────────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                  ┌──────────────────────┐
                  │ Delhivery REST API   │
                  │ track.delhivery.com  │
                  └──────────────────────┘
```

## Carrier-agnostic interface

`internal/gateway/courier/courier.go` exposes `Gateway`:

```go
type Gateway interface {
    CheckPincode(ctx, pincode) (*PincodeInfo, error)
    FetchRateMatrix(ctx) ([]RateRow, error)
    CreateShipment(ctx, req) (*CreateShipmentResult, error)
    GenerateLabel(ctx, awb) (string, error)
    CreateManifest(ctx, awbs, pickupDate) (*ManifestResult, error)
    SchedulePickup(ctx, manifestID, pickupLocation, pickupDate) error
    TrackByAWB(ctx, awb) (*TrackingInfo, error)
    ReAttemptDelivery(ctx, awb, action) error
    CreateReversePickup(ctx, req) (*ReversePickupResult, error)
    FetchCODRemittances(ctx, from, to) ([]RemittanceRow, error)
    VerifyWebhookSignature(headers http.Header, body []byte) error
    ParseWebhook(body []byte) (*WebhookEvent, error)
}
```

Implementations:
- `delhivery.Client` (production) — HTTP client against Delhivery REST API, token-authed.
- `delhivery.DevClient` (local dev) — returns deterministic fake data when `DELHIVERY_API_TOKEN=""`.

Canonical event enum `courier.ShipmentEvent` is the wire format between gateway and services. `courier.ToShipmentStatus(event)` + `courier.ToReturnStatus(event)` map to domain status types — carrier translation lives in `courier/status_map.go`, services stay carrier-blind.

Wire DI selects implementation at compile time:

```go
func ProvideDelhiveryGateway(cfg *config.Config) courier.Gateway {
    if cfg.Delhivery.APIToken == "" {
        return delhivery.NewDevClient()
    }
    return delhivery.NewClient(delhivery.Config{...})
}
```

## Lambda topology

Phase 3 deploys 3 cron Lambdas alongside 22 existing admin/store/migrator Lambdas → **25 total in dev**, 29 with event stack enabled (+4 workers).

| Lambda | Memory | Timeout | Trigger | Service |
|--------|--------|---------|---------|---------|
| `cron-pickup-batch` | 256 MB | 10 min | EventBridge `cron(30 11 * * MON-FRI *)` | ManifestService.RunDailyBatch |
| `cron-cod-remittance` | 256 MB | 5 min | EventBridge `cron(30 2 * * ? *)` | CODReconciliationService.RunDailyPull |
| `cron-rate-refresh` | 256 MB | 10 min | EventBridge `cron(30 21 ? * SAT *)` + admin async invoke | RateTableService.Refresh |

All cron Lambdas:
- ARM64, `provided.al2023` runtime, `bootstrap` handler.
- `ReservedConcurrentExecutions: 1` (prevents EventBridge replay races).
- CloudWatch error alarm on `Errors > 0` over 1 hr window (no SNS action yet — open item).
- Slim `EnvVars` map (only DDB tables + Delhivery + event topic) — no JWT/MSG91/PhonePe leakage (least-privilege boundary).

## CDK stacks

```
infra/
├── cmd/main.go            — entry, instantiates stacks in order: Database → Storage → Cron → API
└── stacks/
    ├── database.go        — DDB tables (8 incl. handloom-shipping-{env}) + GSIs + Postgres trigger
    ├── storage.go         — S3 bucket
    ├── cron.go            — 3 cron Lambdas + EventBridge rules + CloudWatch alarms (NEW Phase 3)
    └── api.go             — 22 service Lambdas + API Gateway + IAM
```

Stack ordering matters: `APIStack` receives `cronStack.RateRefreshFn` so admin order Lambda gets `GrantInvoke` + `FunctionName()` from CDK tokens (no string concat for ARN). Phase 3 simplify resolved this.

## Webhook flow

```
Delhivery POST → API Gateway → store-webhooks Lambda
   → handleCourierWebhook (one handler, routes /delhivery + /delhivery/reverse)
       → io.ReadAll(io.LimitReader(r.Body, 1<<20))  // 1MB cap
       → ShippingService.HandleWebhook(ctx, body, r.Header)
           → courier.VerifyWebhookSignature  // HMAC-SHA256 via subtle.ConstantTimeCompare
               → fail → AppError{Unauthorized} → HTTP 401
           → courier.ParseWebhook → WebhookEvent
           → if ev.IsReverse: ReturnService.HandleReverseWebhook (stub)
           → else (forward):
               → ShipmentRepository.GetByAWB (awb-index GSI lookup)
                   → not found: log + ACK 200 (replay protection)
               → newStatus = courier.ToShipmentStatus(ev.Status)
               → if newStatus == sh.Status: ACK (state-based dedupe)
               → ev.Status == NDR: publish event.ShipmentUpdated (NDR-specific data)
               → ShipmentRepository.UpdateStatus (atomic status + priority_status)
               → terminal-state side-effects (DELIVERED → order, RTO → order)
       → ACK 200 {status: ok}
```

**Open gap:** No idempotency key store (replays with same content arrive within ms could double-fire NDR counters before state catches up). Mitigation: add `processed_webhook_ids` cache keyed on `(awb, event.id)` with 7-day TTL.

## Webhook signature

Delhivery sends `X-Delhivery-Signature: <hex(hmac_sha256(WebhookToken, body))>`. Verification uses `crypto/subtle.ConstantTimeCompare` to avoid timing attacks. Failure returns `errors.New(ErrCodeUnauthorized, ...)` → HTTP 401.

## Configuration

Backend env vars (`DelhiveryConfig` in `internal/config/config.go`):

| Var | Default | Purpose |
|-----|---------|---------|
| `DELHIVERY_API_TOKEN` | `""` | Production API token; empty → DevClient |
| `DELHIVERY_BASE_URL` | `https://track.delhivery.com` | API root |
| `DELHIVERY_CLIENT_NAME` | `""` | Merchant identifier sent with rate quotes |
| `DELHIVERY_WEBHOOK_TOKEN` | `""` | HMAC secret for inbound webhook verification |
| `DELHIVERY_PICKUP_LOCATION` | `Primary` | Warehouse name registered in Delhivery dashboard |
| `NDR_REATTEMPT_LIMIT` | `3` | NDRService escalation threshold |
| `RETURN_WINDOW_DAYS` | `7` | Days post-delivery during which return can be created |
| `DYNAMODB_SHIPPING_TABLE` | `handloom-shipping-local` | New shipping table name |
| `RATE_REFRESH_LAMBDA_NAME` | `""` | Async target for admin "Refresh rates" — empty → sync fallback |

Propagated to all Lambdas via `commonEnv` in `infra/stacks/api.go`. Cron Lambdas use slim `cronEnv` built in `infra/cmd/main.go` (no JWT, no MSG91, no PhonePe — least-privilege).

## DI summary

Wire generates injectors per Lambda binary:

- `InitializeStoreCheckoutDeps`, `InitializeStoreWebhooksDeps`, `InitializeStoreTrackingDeps`, `InitializeStoreCatalogDeps`, `InitializeOrderDeps`, `InitializeMonolithDeps` (existing, extended with new providers)
- `InitializeCronPickupBatchDeps`, `InitializeCronCODRemittanceDeps`, `InitializeCronRateRefreshDeps` (new Phase 3)

Provider sets `RepositorySet` + `ServiceSet` include the 5 new services + 4 new repos. Shiprocket gateway + service are deleted.

## Frontend integration

### Admin (handloom-admin-frontend)

- `src/features/shipping/` — Rates, COD Remittance, Pickups, NDR Queue + admin handlers
- `src/features/returns/` — ReturnFormModal + ReturnsListPage
- `src/features/orders/` extended — PriorityToggle, shipping section on OrderDetailPage
- `<PageHeader>` extracted to `src/shared/components/layout/`
- `<PriorityBadge>` extracted to `src/features/shipping/components/`
- `formatPaiseExact` helper in `src/shared/utils/currency.ts`

### Storefront (homechrome-store)

- `src/types/shipping.ts` — TrackingEvent, TrackingScan
- `src/lib/shipping-api.ts` — checkPincode (re-uses canonical `ServiceabilityResult`)
- `src/components/tracking/TrackingTimeline.tsx`
- `src/app/p/[slug]/PincodeChecker.tsx` (client component, in-component Map cache)
- `src/app/return-policy/page.tsx` (static server component)
- `src/components/orders/ReturnStatusBadge.tsx`
- `src/components/checkout/ShippingLine.tsx` — used by cart + checkout

## Performance characteristics

| Operation | Frequency | Latency budget |
|-----------|-----------|----------------|
| Pincode check (cache hit) | Per cart/checkout | <50 ms DDB GetItem |
| Pincode check (cache miss) | First customer for a pincode (or 7d expiry) | ~500 ms Delhivery + DDB write |
| Rate lookup at checkout | Per checkout | <50 ms (2 DDB GetItems: pincode + rate) |
| Shipment creation | Per order ship | ~1–2 s Delhivery + ~50ms DDB write |
| Tracking (cache hit, terminal) | Per /track page view | <50 ms DDB read |
| Tracking (cache miss, non-terminal) | Per /track > 30min | ~500 ms Delhivery + DDB write |
| Daily pickup batch | 1× daily Mon-Fri | up to 10 min Lambda (500 sequential DDB writes after carrier call) |
| COD reconciliation | 1× daily | up to 5 min (N+1 on GetByAWB — Phase 2 deferred concurrency) |
| Rate refresh | 1× weekly + on-demand | up to 10 min (75 sequential Delhivery calls — deferred concurrency) |

## Known deferred performance items

- `FetchRateMatrix` 75 sequential calls — bounded errgroup recommended (Phase 2 simplify)
- `CODReconciliationService.processUTRGroup` N+1 on AWB lookup — bounded errgroup recommended
- `ManifestService.RunDailyBatch` 500 sequential UpdateStatus — bounded errgroup or TransactWriteItems
- 30s shared HTTP client timeout (per-call `ctx.WithTimeout` better — Phase 2 deferral)
