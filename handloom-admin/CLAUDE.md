# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Handloom Admin is a Go serverless backend for the Homechrome handloom e-commerce platform. It powers both the admin dashboard and the B2C customer storefront. It runs as 26 Lambda services in production (12 admin + 9 store + 4 event workers + 1 migrator) and a single monolithic server locally (port 8081). Go module: `github.com/handloom/admin`, Go 1.24.

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
- `internal/wire/` — Google Wire compile-time DI; one `Initialize*Deps()` per Lambda
- `internal/gateway/` — External service integrations (PhonePe payments, Shiprocket shipping, MSG91 SMS/OTP)
- `internal/event/` — SNS event publisher + types; `handlers/` subdir for SQS consumer handlers
- `internal/middleware/` — RequestID, Logger, Recoverer, Auth (admin JWT), CustomerAuth (customer JWT), Validation (generics-based)
- `pkg/errors/` — typed `AppError` with error codes mapping to HTTP statuses
- `pkg/response/` — standard JSON envelope: `{success, data, meta}` or `{success, error: {code, message}}`

### Database Design
Hybrid storage: DynamoDB (7 tables) for core/transactional data + PostgreSQL (RDS) for catalog data.

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
Categories, products, inventory stored in PostgreSQL (RDS). Schema files in `migrations/*.sql`, auto-applied by the migrator Lambda on `cdk deploy` (see below). Locally, Docker applies `001_catalog_schema.sql` on first start. Repository implementations in `internal/repository/postgres/`.

**Config:** `POSTGRES_DSN` (local dev) or `RDS_SECRET_ARN` + `RDS_ENDPOINT` + `RDS_PORT` + `RDS_DATABASE` (Lambda — credentials resolved from Secrets Manager). Connection pool: `pgxpool` (jackc/pgx v5).

**Tables:** `categories`, `category_attributes`, `category_attribute_options`, `products`, `product_attribute_values`, `product_images`, `inventory`, `inventory_transactions`

**Key patterns:**
- **Attribute filtering**: Dynamic `EXISTS` subqueries on `product_attribute_values` table (EAV pattern). Hardcoded fields (material, color) also stored as attribute rows for uniform filtering.
- **Full-text search**: `tsvector` generated column (`search_vector`) on products combining `name` (weight A) and `description` (weight B) with a GIN index. Queries use `websearch_to_tsquery` for relevance-ranked `ts_rank` ordering, with an `ILIKE` fallback for partial/substring matches. Trigram index on `name` kept for the ILIKE path.
- **Inventory locking**: `SELECT ... FOR UPDATE` within transactions to prevent race conditions on stock changes. Every mutation creates an `inventory_transaction` audit record.
- **Caching**: In-process TTL cache (`internal/cache/`, go-cache) wraps category (2-5 min) and product (2 min) repos. Invalidated on writes. Product lists are NOT cached.
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
- **Dual publisher**: `SNSPublisher` (Lambda mode, publishes to SNS) vs `LocalPublisher` (monolith mode, calls handler functions in-process). Controlled by `EVENT_PUBLISHING_ENABLED` env var.
- **SNS fan-out**: Single `handloom-events-{env}` topic → 4 SQS queues with filter policies → 4 worker Lambdas
- **Workers**: `worker-notification`, `worker-report`, `worker-analytics`, `worker-audit` — entry points at `cmd/lambda/worker-*/main.go`, use `SQSBatchResponse` with `BatchItemFailures` for partial failure reporting
- **Event handlers**: `internal/event/handlers/` — each implements `EventHandler` interface (`CanHandle`/`Handle`) for `LocalPublisher` and `HandleSQSEvent` for Lambda mode
- **20 event types** across 7 categories: `order.*` (3), `payment.*` (3), `shipment.*` (3), `product.*` (3), `inventory.*` (3), `customer.*` (2), `admin.*` (2) — defined in `internal/event/types.go`
- **Filter policies**: notification gets `order/payment/shipment/customer.registered`, report gets `order/payment`, analytics gets `order/payment/product/inventory/customer`, audit gets all events
- **Fire-and-forget**: Event publishing errors are logged but never propagate to callers
- Config: `SNS_TOPIC_ARN`, `EVENT_PUBLISHING_ENABLED`

### Schema Migrations
- SQL files in `migrations/` are embedded via `go:embed` (`migrations/embed.go`) into the migrator Lambda
- `cmd/lambda/migrator/main.go` — connects to RDS, creates `schema_migrations` tracking table, applies unapplied `.sql` files in filename order, each in a transaction
- CDK `triggers.Trigger` in `infra/stacks/database.go` invokes the migrator after RDS is ready, re-runs when migration files change
- Uses `pg_advisory_lock` for concurrency safety
- Migration failure causes CDK deployment rollback
- To add a migration: create `migrations/NNN_description.sql`, then `make cdk-deploy-dev`

### Infrastructure (infra/)
AWS CDK in Go. Four stacks per environment: DatabaseStack, StorageStack, APIStack, EventStack. All Lambdas use ARM64/128MB(dev)/256MB(prod)/provided.al2023.

## Key Conventions

- **Handler validation**: Use `middleware.ValidateJSONTyped[T]` as Chi middleware, then `middleware.MustGetValidatedBody[T]` in handler
- **Error returns**: Services return `*errors.AppError`; handlers call `response.Error(w, err)`
- **Wire DI**: After adding/changing providers, run `make wire` to regenerate `wire_gen.go`
- **Mock generation**: After changing interfaces in `domain/`, run `make generate-mocks`
- **Import ordering**: enforced by goimports with local prefix `github.com/handloom/admin`
- **golangci-lint**: complexity thresholds — gocognit=30, gocyclo=25, dupl=200
