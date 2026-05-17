# Shipping — Open Items, TODOs, Known Issues

Aggregated from 5 phase reviews + 5 simplify passes + convention audit. Items are grouped by severity. Last updated 2026-05-16, branch `delhivery`, tag `delhivery-complete`.

---

## 🚨 P0 — Critical pre-deploy blockers

These must land before customers see the live Delhivery flow.

| # | Item | Owner | Notes |
|---|------|-------|-------|
| P0-1 | **Delhivery production API token + base URL** | Ops | `DELHIVERY_API_TOKEN`, `DELHIVERY_BASE_URL`, `DELHIVERY_CLIENT_NAME`, `DELHIVERY_WEBHOOK_TOKEN` env values. Empty token = DevClient fallback (silent fake) — confirm prod has all 4 set. |
| P0-2 | **Pickup location name in Delhivery dashboard** | Ops | `DELHIVERY_PICKUP_LOCATION` (default `"Primary"`) must match an actual pickup location registered in Delhivery's UI; otherwise CreateShipment / CreateManifest will fail. |
| P0-3 | **Backend `POST /admin/orders/{id}/shipments?priority=...` not registered** | Backend | Frontend calls; backend 404s. Add to `internal/router/order.go`. Phase 4 surfaced. |
| P0-4 | **Backend `Order.shipment` field missing** | Backend | Frontend `OrderDetailPage` reads `order.shipment` (priority, AWB, manifest_id, COD details, NDR count, label URL). Section is gated `if (order.shipment)` so absence is silent — but flow doesn't work end-to-end without it. Add nested shipment view to Order serializer (or separate GET). |
| P0-5 | **Backend `cart.shipping_charge` not serialized** | Backend | Storefront `CartSummary` falls back to "Calculated at checkout" placeholder. Wire `RateTableService.Lookup` into cart aggregate at fetch time. |
| P0-6 | **Webhook idempotency / replay dedupe** | Backend | Two NDR webhook replays before state catches up would double-increment `ndr_count`. Add `processed_webhook_ids` cache keyed on `(awb, event.timestamp)` with 7-day TTL. |
| P0-7 | **Confirm 7-day return window + 3-attempt NDR limit** | Ops/PM | Defaults from spec. Both env-configurable (`RETURN_WINDOW_DAYS`, `NDR_REATTEMPT_LIMIT`). |

---

## 🔧 P1 — Important (post-launch within first sprint)

### Backend stubs returning 501 (frontend hits, UI shows graceful toast)

| Endpoint | Returns | What's missing |
|----------|---------|----------------|
| `POST /admin/shipping/shipments/{id}/ndr-action` | 501 | Wire `NDRService.HandleNDREvent` with admin-supplied action (`reattempt`, `rto`, `mark_contacted`) |
| `GET /admin/shipping/pickup-batches` | empty `[]any{}` | Implement when `Manifest` entity lands as a persisted record (Phase 5+ suggested) |
| `GET /admin/orders/{id}/returns` | empty `[]any{}` | Wire `ReturnRepository.ListByOrder(orderID)` to handler |
| `PATCH /admin/returns/{id}/cancel` | 501 | Implement `ReturnService.Cancel` (lookup by ID needs PK+SK or new GSI) |
| `POST /admin/returns/{id}/refund` | 501 | **Major** — integrate PhonePe refund flow; mark `Return.Status=REFUNDED`, set `RefundedAt`, restore inventory, publish `return.refunded` event |
| `/api/v1/store/track/by-awb/{awb}` (planned) | n/a | Storefront `shippingApi.trackByAWB` was deleted as dead code. Reintroduce when endpoint ships. Today track flow uses `/track/{orderNumber}`. |

### Reverse webhook routing

- `ReturnService.HandleReverseWebhook` is a logging stub. Real impl needs:
  - `ReturnRepository.GetByReverseAWB(awb)` (new method + likely a `return-awb-index` GSI on orders table).
  - Status mutation: REQUESTED → PICKED_UP (REVERSE_PICKED_UP event) → RECEIVED (REVERSE_DELIVERED event).
  - Publish `return.received` event when terminal.

### `ShippingRateRepository.ListAll` pagination

- Currently unpaginated `Scan`. Fine at 25 rate rows. Will silently truncate if pincode cache rows scale past 1 MB scan page. Add `LastEvaluatedKey` loop or move to `Query` against a fixed PK prefix.

### `OrderRepository.UpdateCODRemittance` audit attribute

- Doesn't set `updated_by` — only `updated_at`. Either pass `actorID` ("system" for cron) or accept that cron writes are unattributed.

---

## ⚡ P2 — Performance + concurrency (deferred from Phase 2 simplify)

Real perf concerns, not Phase 3-introduced. Acceptable for current scale (<100 orders/day). Revisit when volume grows.

| Item | Location | Impact | Fix |
|------|----------|--------|-----|
| `FetchRateMatrix` 75 sequential HTTP calls | `gateway/delhivery/client.go:fetchRate` × 75 | ~22–60 s per rate refresh cron | `errgroup` with `SetLimit(8)` |
| `CODReconciliationService` N+1 `GetByAWB` | `service/cod_reconciliation_service.go:reconcileEntry` | ~200 entries × 2 DDB calls = ~4 s+ | Bounded errgroup or `BatchGetItem` |
| `ManifestService.RunDailyBatch` serial UpdateStatus | `service/manifest_service.go:RunDailyBatch` | 500 sequential DDB writes after carrier call | Bounded errgroup or chunked `TransactWriteItems` |
| Shared 30s HTTP client timeout | `gateway/delhivery/client.go:NewClient` | Hot-path `CheckPincode` may block cart 30 s on Delhivery slowness | Per-call `ctx.WithTimeout(3*time.Second)` in `ShippingService.CheckServiceability` |
| `CheckoutService.Initiate` dup pincode read | `service/checkout_service.go` | Net 3 DDB reads on checkout hot path (pincode read twice) | `CheckServiceability` returns `PincodeZone` + `Lookup` accepts pre-resolved zone |
| `CODRemittanceRepository.ListByStatus` no cursor | `repository/dynamodb/cod_remittance_repository.go` | Truncates at `limit` — admin can't page past N remittances | Add cursor parameter mirroring `ReturnRepository.ListByStatus` |
| Single-row `BatchUpsert` retries on partial failure | `repository/dynamodb/shipping_rate_repository.go` | Inherited pattern — currently retries 5× with 100ms linear backoff | Already correct; mention only as known pattern |

---

## ♿ P3 — Frontend UX + accessibility polish

Mostly from Phase 4–5 reviews. Non-blocking but worth picking up.

### Admin frontend (Phase 4 deferrals)

- `ValidateJSONTyped[T]` refactor across admin handlers (admin shipping + order handlers use raw `json.NewDecoder`)
- Rename `Order.unit_price` → `unit_price_paise` (units in field names — codebase-wide consistency)
- `RateEditModal` use react-hook-form `key` pattern (already adopted in sibling components)
- Reset `priority` local state on `OrderDetailPage` shipment-create success
- `<CurrencyInput>` shared component (3 paise-inputs needed)
- `<FilterChipGroup>` shared component (CODRemittancePage chips; will recur)
- `<ListPage>` shell — title/subtitle/action/Card+Table pattern duplicated across 14+ pages
- `<RefundAmountModal>` generalized → `<AmountModal>` (rate edit + future fee edits)
- Per-row pending state on cancel/refund (currently shared mutation `isPending` shows spinner on all rows)
- `useUpdateMutation` hook (8+ sites duplicate `useMutation + toast + invalidate` pattern)
- Status map registry (`statusPresentation.ts`) — 3 separate label/color maps (OrderStatusBadge, TrackingTimeline, ReturnStatusBadge)
- `<PageHeader>` retrofit to 14 existing admin pages (Phase 4 simplify added it only to new pages)

### Storefront (Phase 5 deferrals)

- Pincode persistence across PDP → cart → checkout via Zustand `shippingStore` with persist middleware (currently in-component Map only, lost on page change)
- PincodeChecker localStorage persist for last-checked pincode
- Server Component migration for `app/account/orders/[id]/page.tsx` (currently client + `useEffect` fetch + skeleton state machine)
- `sortedImages` `useMemo` in `ProductDetailView` (pre-existing perf nit)
- `firstCourier` arbitrary pick in PincodeChecker — show min/range across `couriers[]` or document
- `isValidIndianPincode(s)` helper in `lib/validation` (regex duplicated in PincodeChecker and address forms)
- `TrackingTimeline` `aria-live` for newly-arrived scans (a11y — already has `aria-label` on `<ol>` after Phase 5 fix)
- `normalizeTracking(tracking)` helper to flatten `tracking?.shipment?.X ?? tracking?.X` fallbacks (5 sites in `track/page.tsx`)
- Convert `OrderSummary` shipping placeholder `"--"` → already done via `<ShippingLine>` extraction

---

## 🔭 P4 — Observability + ops

### Cron Lambda alarms have no SNS action

`infra/stacks/cron.go` creates `AWS::CloudWatch::Alarm` for each cron Lambda's `Errors > 0` over 1 hr. **No `AlarmAction` wired** — alarms enter `ALARM` state silently. No paging.

- Decision needed: which SNS topic should receive these? Ops on-call? Email-only ops list? PagerDuty integration?
- One-line CDK addition in each `awscloudwatch.NewAlarm` call: `AlarmActions: &[]awscloudwatch.IAlarmAction{...}`.

### Custom CloudWatch metrics

Phase 3 review suggested emitting:
- `ManifestedShipments` (cron-pickup-batch)
- `RemittancesMatched` / `RemittancesUnmatched` (cron-cod-remittance)
- `RatesRefreshed` (cron-rate-refresh)
- `WebhooksReceived` / `WebhooksUnknownAWB` / `WebhooksSignatureFailed`

Today, observability is via `slog` log inspection only.

### EventBridge schedule drift recovery

If AWS outage during the daily 02:30 UTC firing, EventBridge does not catch up. COD pull silently misses a day. Mitigate:
- Store `last_successful_run_at` per cron in DynamoDB.
- On next firing, COD pull queries from `last_successful_run_at` (not `now - 24h`).
- Same pattern for rate refresh + pickup batch.

### Async invoke payload schema

`cron-rate-refresh` ignores the EventBridge payload. When admin "Refresh rates" button gains zone-scoped option (e.g., "Refresh Zone A only"), need a typed payload:

```go
type RateRefreshRequest struct {
    Zone string `json:"zone,omitempty"` // empty = full matrix
}
```

Handler currently takes only `ctx context.Context` — extend signature.

### Phase 4 alarm action TODO

CronStack alarm comment: `// TODO(P4): wire SNS action to ops topic`. Track via this doc.

---

## 🧪 P5 — Test coverage gaps

Phase 4 (admin frontend) shipped 1 test file. Coverage on critical paths is light.

| Surface | Current | Gap |
|---------|---------|-----|
| `internal/gateway/delhivery/client.go` | 12 method tests + status_map (13 cases) | 70%+ coverage. Strong. |
| `internal/service/shipping_service.go` | Some via integration | Unit tests for `HandleWebhook` (signature fail, unknown AWB, reverse routing branch), `CheckServiceability` cache-hit vs cache-miss |
| `internal/service/manifest_service.go` | None | `RunDailyBatch` empty list, partial-failure path |
| `internal/service/ndr_service.go` | None | Boundary tests on `maxAttempts` — count 2, 3, 4 |
| `internal/service/cod_reconciliation_service.go` | None | Multi-UTR group, mixed-match scenarios, idempotent re-pull |
| `internal/service/rate_table_service.go` | None | Manual-override preservation |
| `internal/service/return_service.go` | None | Window expired, no shipping address, undelivered order |
| `internal/handler/cron/*` | 6 tests (happy + failure per handler) | OK |
| `handloom-admin-frontend/src/features/shipping/` | 1 (`RatesPage.test.tsx`) | At least 1 happy + 1 error per page — CODRemittancePage, PickupBatchPage, NDRQueuePage, ReturnsListPage, ReturnFormModal, OrderDetailPage shipping section |
| `homechrome-store/` Phase 5 | 0 new tests | TrackingTimeline (empty / multi-scan / unknown status), PincodeChecker (invalid regex / cache hit / cache miss), ShippingLine variants |
| LocalStack integration smoke | Skipped on this branch (Docker not running) | `make setup-local && make test-integration` should be re-run before deploy |
| CDK diff against prod | Not run (AWS session unavailable) | `make cdk-diff-prod` review before deploy |

---

## 🧹 P6 — Refactor opportunities (nice-to-have)

### Backend

- `handloom-admin/scripts/init-local-db.sh`: orders table `priority-status-index` and `awb-index` GSIs added inline; verify on fresh `make setup-local`
- `internal/repository/dynamodb/helpers.go batchDeleteKeys` lacks `UnprocessedItems` retry parity with new `BatchUpsert` in shipping repo
- `parseDelhiveryTime` simplified to return `time.Time` only (no error); review confirmed
- `internal/handler/shipping_admin_handler.go nextDayPickup9AMIST` consolidated into `cron.NextDayPickupSlotIST` (done Phase 3 simplify)
- `Manifest` as a persisted entity (PK=`MANIFEST#<id>`) for batch history / partial-failure reconciliation — Phase 3 review suggested
- Indian-holidays calendar (`internal/calendar/IsBusinessDay(t)`) for pickup-date computation — currently any cron firing day → next day, regardless of holidays
- `ShipmentRepository.UpdateStatus` extra `GetByOrderID` removed when Phase 2 changed signature to accept `priority` directly (done)

### Admin frontend

- Existing 14 admin pages still hand-roll page header — adopt `<PageHeader>`
- Status maps unified to a single registry
- `useUpdateMutation` hook
- `<CurrencyInput>` + `<FilterChipGroup>` shared components

### Storefront

- `<PolicySection>` component or data array for `return-policy/page.tsx` (6 `<h2>` repetitions)
- `formatDateTime` import alias inconsistency across files

---

## ✅ Fixed during phase reviews (audit trail)

### Phase 1
- ✅ `Servicable` typo → `Serviceable` (matches existing domain field)
- ✅ Field naming `WeightG`/`LengthCM` → `WeightGrams`/`LengthCm` (codebase convention)
- ✅ `VerifyWebhookSignature(http.Header)` instead of `map[string]string`
- ✅ `Shipment.UpdateStatus` doc warning about `priority_status` invariant
- ✅ Use `SKMetadata` constant, `EntityType*` constants
- ✅ `ReturnRepository.UpdateStatus` use `buildDynamicUpdate` helper
- ✅ `BatchUpsert` retry on `UnprocessedItems`
- ✅ `errors.Wrap` not `errors.Internal` (preserve cause)
- ✅ Config loaded `Delhivery` block (was struct without `Load()` population)
- ✅ `CalculateShipping` deleted (duplicated `RateTableService.Lookup`)
- ✅ `parseDelhiveryTime` dropped dead error return
- ✅ `CreateShipment`/`CreateReversePickup` extracted shared helpers

### Phase 2
- ✅ **`Shipment.UpdateStatus` atomic** `status + priority_status` write (was read-then-write)
- ✅ COD reconciliation idempotency (deterministic UUIDv5 by UTR + up-front aggregates)
- ✅ Webhook signature error returns `ErrCodeUnauthorized` → HTTP 401
- ✅ Reverse webhook routes to `ReturnService.HandleReverseWebhook` (was incorrectly mutating forward shipment)
- ✅ `BatchResult` failure tracking (`ShipmentMarkedIDs` + `FailedShipmentIDs`)
- ✅ Stubs return 501 instead of fake-success 200
- ✅ Pincode regex `^[1-9][0-9]{5}$` (rejects 0-prefix)
- ✅ Nil-publisher guards dropped (Wire always returns non-nil)
- ✅ `mapCourierEvent`/`mapReturnStatus` moved to `internal/gateway/courier/status_map.go`
- ✅ Duplicate webhook handlers collapsed into one
- ✅ `UpdateCODRemittance` added `attribute_exists(PK)` + `updated_at`
- ✅ Webhook body capped at 1 MB via `io.LimitReader`
- ✅ `TrackShipment` cache short-circuit covers all terminal states
- ✅ `ProcessRefund` returns `NotImplemented` (was fake-acking)

### Phase 3
- ✅ `nextBusinessDay9AM` renamed `nextDayPickup9AMIST` + Mon-Fri schedule
- ✅ Cron-pickup-batch timeout 5 → 10 min
- ✅ `LambdaInvoker.Invoke` doc clarified
- ✅ Failure-path tests added for COD + Pickup handlers
- ✅ **Cross-stack circular dep resolved** — APIStack uses `cronStack.RateRefreshFn.GrantInvoke()` + `FunctionName()` instead of hand-built ARN; `cronEnv` built in `main.go` (slim, no JWT/MSG91/PhonePe leak)
- ✅ `ReservedConcurrentExecutions: 1` on all 3 cron Lambdas
- ✅ `_ = cronStack` dead binding removed
- ✅ `nextBusinessDay9AM`/`nextDayPickup9AMIST` consolidated to `cron.NextDayPickupSlotIST`
- ✅ Makefile `build-lambdas-all` used by both deploy targets (DRY)

### Phase 4
- ✅ `formatPaiseExact` helper (2-decimal precision); rate editor switched to paise inputs (no decimal round-trip)
- ✅ `window.prompt` replaced with `<RefundAmountModal>`
- ✅ Badge variant map updated with 14 new statuses
- ✅ Specific React Query invalidation keys (per-orderId)
- ✅ NDR menu items disabled while mutation pending
- ✅ NaN guard on `ReturnFormModal.handleQty`
- ✅ Per-row pending state on cancel/refund
- ✅ `ConfirmModal` on destructive cancel
- ✅ `<PriorityBadge>` extracted (2 inline usages collapsed)
- ✅ `<PageHeader>` extracted for new admin pages
- ✅ `RateEditModal` `key` pattern (no `useEffect` reset)
- ✅ Backend `BatchResult` got JSON tags (snake_case in TS)
- ✅ `badge.ts` semantic rationale: CANCELLED→gray, REFUNDED→success

### Phase 5
- ✅ `NEXT_PUBLIC_RETURNS_ENABLED` flag dropped — runtime detection
- ✅ TrackingTimeline highlights only one current scan (was all matching)
- ✅ PincodeChecker `role="alert"` + `aria-invalid` + `role="status"` on result
- ✅ Track page 404 vs 5xx error branching
- ✅ Cart/checkout shipping line: "Calculated at checkout" / "Calculated next" placeholder (removed `FLAT_SHIPPING_ESTIMATE` magic number)
- ✅ TrackingTimeline ARIA (`aria-label` on `<ol>`, `aria-hidden` on icon)
- ✅ Static `return-policy` page (dropped unused `revalidate`)
- ✅ Consolidated `PincodeServiceability` → canonical `ServiceabilityResult`
- ✅ Consolidated `ReturnStatus` (single source in `types/index.ts`)
- ✅ Consolidated `TimelineScan` → `TrackingScan` (renamed `time` → `timestamp`)
- ✅ Deleted dead `trackByAWB` + `TrackingInfo`
- ✅ Shared `<Alert>` (added `success` variant) in PincodeChecker
- ✅ Shared `<Input>` in PincodeChecker
- ✅ `isAxiosError` instead of unsafe cast
- ✅ `<ShippingLine>` extracted for cart + checkout (no copy drift)
- ✅ PincodeChecker `useRef<Map>` memoize (no repeat network per PDP visit)
- ✅ `SUPPORT_EMAIL` constant in return-policy
- ✅ `CHECK_PINCODE` route moved to `SHIPPING` namespace

---

## Reference

- Full design spec: `docs/superpowers/specs/2026-05-15-delhivery-integration-design.md` (gitignored, on disk only)
- Phase plans 1–5: `docs/superpowers/plans/2026-05-16-delhivery-phase-{1..5}-*.md` (gitignored)
- Architecture: [hld.md](./hld.md)
- Schema: [database.md](./database.md)
- Flows: [user-flows.md](./user-flows.md)
- Git tags: `delhivery-phase-1` … `delhivery-phase-5`, `delhivery-complete`
