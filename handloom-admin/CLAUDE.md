# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Handloom Admin is a Go serverless backend for the Homechrome handloom e-commerce platform. It powers both the admin dashboard and the B2C customer storefront. It runs as 22 Lambda services in dev (12 admin + 9 store + 1 migrator; event stack disabled) or 26 with event stack enabled (+ 4 workers), and a single monolithic server locally (port 8081). Go module: `github.com/handloom/admin`, Go 1.25.

## Common Commands

### Local Development
```bash
make setup-local      # One-command: docker-up + create-tables + init-s3 + create-events + seed-data
make run              # Start local API server on :8081 (monolith mode)
make run-watch        # Hot reload via air
make docker-up        # Start LocalStack (:4566) + DynamoDB Admin UI (:8001)
make deploy-local     # Build + deploy Lambdas to LocalStack (Lambda mode)
make redeploy-local   # Redeploy Lambda code only (faster)
make teardown-local   # Stop all Docker services and remove volumes
```

### Build
```bash
make build                # Build local binary → bin/handloom-api
make build-lambdas        # Build all Lambda binaries (linux/arm64)
make build-lambdas-active # Build only active lambdas: auth, user, catalog, asset
make build-workers        # Build all 4 worker Lambda binaries
```

### Test
```bash
make test                 # All tests: go test -v -race -cover ./...
make test-unit            # Unit tests only (internal/service/...)
make test-integration     # Integration tests (internal/repository/..., needs DynamoDB Local)
```
Run a single test:
```bash
go test -v -run TestFunctionName ./internal/service/...
```

### Code Generation
```bash
make wire             # Regenerate internal/wire/wire_gen.go (run after changing DI wiring)
make generate-mocks   # Regenerate mocks from domain/repository.go and domain/service.go
```

### Lint
```bash
golangci-lint run     # Uses .golangci.yml config
```

### Deploy
```bash
make cdk-deploy-dev   # Build active lambdas + CDK deploy (dev)
make cdk-deploy-prod  # Build active lambdas + CDK deploy (prod)
```

## Architecture

### Dual Entry Points
- **Local dev**: `cmd/api/main.go` — monolith that mounts all routers into one Chi mux
- **Lambda**: `cmd/lambda/<service>/main.go` — each service is a separate binary using `aws-lambda-go-api-proxy` to bridge API Gateway events to Chi

### Clean Architecture Layers
```
domain/ (entities + interfaces) ← handler/ → service/ → repository/dynamodb/
```
- `internal/domain/` — all entities, service interfaces (`service.go`), repository interfaces (`repository.go`, `order_repository.go`, `audit_repository.go`)
- `internal/handler/` — HTTP handlers (admin), one per domain; each exposes `Routes() chi.Router`
- `internal/handler/store/` — B2C store handlers (auth, catalog, cart, checkout, orders, tracking, profile, events, webhooks)
- `internal/service/` — business logic implementing domain service interfaces
- `internal/repository/dynamodb/` — DynamoDB implementations of domain repository interfaces
- `internal/repository/postgres/` — PostgreSQL implementations for catalog data (categories, products, inventory)
- `internal/router/` — mounts handlers onto Chi routers; `lambda.go` provides the Lambda adapter
- `internal/wire/` — Google Wire compile-time DI; one `Initialize*Deps()` per Lambda plus `InitializeMonolithDeps()` for the local `cmd/api` server
- `internal/gateway/` — External service integrations (PhonePe Standard Checkout v2 payments, Shiprocket shipping, MSG91 SMS/OTP). Each gateway uses a DevClient fallback when credentials are not configured (SMS: `MSG91_AUTH_KEY`/`MSG91_OTP_TEMPLATE_ID`, PhonePe: `PHONEPE_CLIENT_ID`/`PHONEPE_CLIENT_SECRET`, Shiprocket: `SHIPROCKET_EMAIL`/`SHIPROCKET_PASSWORD`).
- `internal/event/` — SNS event publisher + types; `handlers/` subdir for SQS consumer handlers
- `internal/middleware/` — RequestID, Logger, Recoverer, Auth (admin JWT), CustomerAuth (customer JWT), Validation (generics-based)
- `pkg/errors/` — typed `AppError` with error codes mapping to HTTP statuses
- `pkg/response/` — standard JSON envelope: `{success, data, meta}` or `{success, error: {code, message}}`

### Database Design
Hybrid storage: DynamoDB (7 tables) for core/transactional data + Neon PostgreSQL for catalog data.

#### DynamoDB Tables
All using PK/SK composite keys with GSIs:
| Table | Env Name | Content |
|-------|----------|---------|
| `handloom-core-{env}` | `DYNAMODB_CORE_TABLE` | Users, Pricing Rules, Coupons |
| `handloom-orders-{env}` | `DYNAMODB_ORDERS_TABLE` | Orders, Customers, PriceQuotes |
| `handloom-audit-{env}` | `DYNAMODB_AUDIT_TABLE` | Audit logs (TTL-based expiry) |
| `handloom-analytics-{env}` | `DYNAMODB_ANALYTICS_TABLE` | Dashboard metrics, Reports (TTL-based expiry) |
| `handloom-notifications-{env}` | `DYNAMODB_NOTIFICATIONS_TABLE` | Notifications |
| `handloom-sessions-{env}` | `DYNAMODB_SESSIONS_TABLE` | OTP, refresh tokens (TTL-based expiry) |
| `handloom-events-{env}` | `DYNAMODB_EVENTS_TABLE` | Raw tracking events (30-day TTL) |

Key patterns: `PK=USER#<id> SK=METADATA`, etc. Prices stored in **paise** (1 INR = 100 paise). Pagination is cursor-based (base64-encoded DynamoDB `ExclusiveStartKey`).

#### PostgreSQL (Catalog)
Categories, products, inventory stored in Neon PostgreSQL. Schema files in `migrations/*.sql`, auto-applied by the migrator Lambda on `cdk deploy` (see below). Locally, Docker applies `001_catalog_schema.sql` on first start. Repository implementations in `internal/repository/postgres/`.

**Config:** `POSTGRES_DSN` (local dev) or `NEON_CONNECTION_STRING` (Lambda — Neon PostgreSQL connection string). Connection pool: `pgxpool` (jackc/pgx v5).

**Tables:** `categories`, `category_attributes`, `category_attribute_options`, `products`, `product_attribute_values`, `product_images`, `inventory`, `inventory_transactions`

**Key patterns:**
- **Attribute filtering**: Dynamic `EXISTS` subqueries on `product_attribute_values` table (EAV pattern). Hardcoded fields (material, color) also stored as attribute rows for uniform filtering.
- **Full-text search**: `tsvector` generated column (`search_vector`) on products combining `name` (weight A) and `description` (weight B) with a GIN index. Queries use `websearch_to_tsquery` for relevance-ranked `ts_rank` ordering, with an `ILIKE` fallback for partial/substring matches. Trigram index on `name` kept for the ILIKE path.
- **Inventory locking**: `SELECT ... FOR UPDATE` within transactions to prevent race conditions on stock changes. Every mutation creates an `inventory_transaction` audit record.
- **Caching**: In-process TTL cache (`internal/cache/`, go-cache) wraps category and product repos. All cached entries (category items + lists, product items + attributes + lists) use a 1 hour TTL (`catListTTL`/`catItemTTL`/`prodItemTTL`/`prodAttrTTL`/`prodListTTL`). Invalidated on writes via `DeletePrefix`.
- **Pagination**: Base64-encoded integer offset cursors. Fetch `LIMIT+1` to detect HasMore.
- **Batch relation loading**: Product lists batch-load attributes and images via `WHERE product_id = ANY($1)` to avoid N+1 queries.

### Lambda Entry Point Pattern
Every Lambda follows the same structure:
1. `config.Load()` — env vars
2. `wire.Initialize*Deps(ctx, cfg)` — compile-time DI
3. `router.NewAuthenticatedRouter(...)` — Chi mux with auth middleware
4. `router.New*Router(r, deps.Handler)` — mount service routes
5. `router.NewLambdaAdapter(r).Start()` — start Lambda handler

### Asset Upload Flow
Uses a tmp-then-finalize S3 pattern:
1. `POST /admin/assets/upload-url` → returns presigned PUT URL to `tmp/{type}/{uuid}.ext`
2. Client uploads directly to S3
3. On entity save, `AssetService.FinalizeIfTemp()` copies `tmp/` → `assets/` and deletes tmp
4. S3 lifecycle auto-deletes unfinalized `tmp/` objects after 24 hours

### Admin Auth Strategy
- JWT in `access_token` HttpOnly cookie (primary), `Authorization: Bearer` header (fallback)
- JWT secret: `JWT_SECRET_KEY` env var, or fetched from AWS SSM at `/handloom/{env}/jwt-secret`
- Roles: `ADMIN` (bypasses permission checks), `OPERATOR`
- Cookie settings adapt: secure+SameSite=Lax for custom domain, secure+SameSite=None for Lambda URL, insecure for local

### Customer Auth Strategy (B2C Store)
- Phone number + OTP login via MSG91 SMS gateway (dev mode prints OTPs to console)
- Customer JWT in HttpOnly cookies (`CUSTOMER_JWT_SECRET`)
- `CustomerAuth` middleware authenticates store routes
- Token refresh at `/api/v1/store/auth/refresh`

### B2C Store Routes
Mounted at `/api/v1/store/*` in the monolith (`cmd/api/main.go`):
- `/auth/*` — OTP send/verify, refresh, logout (rate-limited: 30/min)
- `/catalog/*` — Public product/category browsing
- `/cart/*` — Cart CRUD (customer-authenticated)
- `/checkout/*` — Order placement + payment initiation (customer-authenticated)
- `/orders/*` — Customer order history (customer-authenticated)
- `/me/*` — Customer profile (customer-authenticated)
- `/track/*` — Public order tracking
- `/events/*` — Storefront analytics event ingestion (rate-limited)
- `/webhooks/*` — Payment callbacks (signature-verified)

### Event-Driven Architecture
- **Three publishers**: `SNSPublisher` (Lambda mode with events enabled, publishes to SNS), `LocalPublisher` (monolith mode, calls handler functions in-process), `NoopPublisher` (Lambda mode with events disabled — logs and discards). Selection is wired at compile time: Lambda injectors include `wire.LambdaPublisherSet` (branches on `EVENT_PUBLISHING_ENABLED`); `InitializeMonolithDeps` includes `wire.MonolithPublisherSet` (always LocalPublisher with the 4 event handlers).
- **Dev: event stack disabled** — EventStack (SNS + SQS + 4 worker Lambdas + EventBridge rule) is commented out in `infra/cmd/main.go` for cost savings. `nil` is passed to APIStack, which handles it gracefully (no `SNS_TOPIC_ARN`, no `EVENT_PUBLISHING_ENABLED` set). `NoopPublisher` is used when events are disabled. To re-enable: uncomment the EventStack block in `infra/cmd/main.go` and pass `eventStack` to APIStack.
- **SNS fan-out** (when enabled): Single `handloom-events-{env}` topic → 4 SQS queues with filter policies → 4 worker Lambdas
- **Workers**: `worker-notification`, `worker-report`, `worker-analytics`, `worker-audit` — entry points at `cmd/lambda/worker-*/main.go`, use `SQSBatchResponse` with `BatchItemFailures` for partial failure reporting
- **Event handlers**: `internal/event/handlers/` — each implements `EventHandler` interface (`CanHandle`/`Handle`) for `LocalPublisher` and `HandleSQSEvent` for Lambda mode
- **20 event types** across 7 categories: `order.*` (3), `payment.*` (3), `shipment.*` (3), `product.*` (3), `inventory.*` (3), `customer.*` (2), `admin.*` (2) — defined in `internal/event/types.go`
- **Filter policies**: notification gets `order/payment/shipment/customer.registered`, report gets `order/payment`, analytics gets `order/payment/product/inventory/customer`, audit gets all events
- **Fire-and-forget**: Event publishing errors are logged but never propagate to callers
- Config: `SNS_TOPIC_ARN`, `EVENT_PUBLISHING_ENABLED`

### Schema Migrations
- SQL files in `migrations/` are embedded via `go:embed` (`migrations/embed.go`) into the migrator Lambda
- `cmd/lambda/migrator/main.go` — connects to Neon PostgreSQL, creates `schema_migrations` tracking table, applies unapplied `.sql` files in filename order, each in a transaction
- CDK `triggers.Trigger` in `infra/stacks/database.go` invokes the migrator after Neon PostgreSQL is ready, re-runs when migration files change
- Uses `pg_advisory_lock` for concurrency safety
- Migration failure causes CDK deployment rollback
- To add a migration: create `migrations/NNN_description.sql`, then `make cdk-deploy-dev`

### Infrastructure (infra/)
AWS CDK in Go. Four stacks per environment: DatabaseStack, StorageStack, APIStack, EventStack — EventStack is currently disabled in dev (commented out in `infra/cmd/main.go`). All Lambdas use ARM64/128MB(dev)/256MB(prod)/provided.al2023. Lambda count: 22 in dev (12 admin + 9 store + 1 migrator), 26 with event stack (+ 4 workers).

Gateway credentials are propagated from the deploy-time shell environment to every Lambda's `Environment.Variables` via `gatewayEnvKeys` in `infra/stacks/api.go` (PhonePe + MSG91 + Shiprocket keys). Empty values fall through to each gateway's DevClient. `make cdk-deploy-{dev,prod}` sources `.env.{dev,prod}` first; the GitHub workflow injects `MSG91_AUTH_KEY` (secret) + `MSG91_OTP_TEMPLATE_ID` (variable) at the step level.

### Payment Integration (PhonePe Standard Checkout v2)
- **Auth**: OAuth token flow — `POST /v1/oauth/token` with `PHONEPE_CLIENT_ID` + `PHONEPE_CLIENT_SECRET` to obtain `O-Bearer` token
- **Payment initiation**: `POST /checkout/v2/pay` with `O-Bearer` token header
- **Status check**: `GET /checkout/v2/order/{id}/status` with `O-Bearer` token header
- **Env vars**: `PHONEPE_CLIENT_ID`, `PHONEPE_CLIENT_SECRET`, `PHONEPE_CLIENT_VERSION`, `PHONEPE_WEBHOOK_USERNAME`, `PHONEPE_WEBHOOK_PASSWORD`
- **Webhook auth**: `Authorization: SHA256(username:password)` header verification
- **Webhook events**: `checkout.order.completed`, `checkout.order.failed`
- **DevClient**: Used when `PHONEPE_CLIENT_ID` or `PHONEPE_CLIENT_SECRET` is empty (simulates successful payments)

### Dev Client Selection
External service clients no longer use `IsDevelopment()` to decide dev vs real mode. Each gateway checks if its specific credentials are configured:
- **SMS (MSG91)**: DevClient when `MSG91_AUTH_KEY` or `MSG91_OTP_TEMPLATE_ID` is empty
- **PhonePe**: DevClient when `PHONEPE_CLIENT_ID` or `PHONEPE_CLIENT_SECRET` is empty
- **Shiprocket**: DevClient when `SHIPROCKET_EMAIL` or `SHIPROCKET_PASSWORD` is empty

## Key Conventions

- **Handler validation**: Use `middleware.ValidateJSONTyped[T]` as Chi middleware, then `middleware.MustGetValidatedBody[T]` in handler
- **Error returns**: Services return `*errors.AppError`; handlers call `response.Error(w, err)`
- **Wire DI**: After adding/changing providers, run `make wire` to regenerate `wire_gen.go`
- **Mock generation**: After changing interfaces in `domain/`, run `make generate-mocks`
- **Import ordering**: enforced by goimports with local prefix `github.com/handloom/admin`
- **golangci-lint**: complexity thresholds — gocognit=30, gocyclo=25, dupl=200
