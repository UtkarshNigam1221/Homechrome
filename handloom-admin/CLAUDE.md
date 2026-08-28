# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Handloom Admin is a Go serverless backend for the Homechrome handloom e-commerce platform. It powers both the admin dashboard and the B2C customer storefront. It runs as 23 Lambda services in dev (13 admin + 9 store + 1 migrator), and a single monolithic server locally (port 8081). Go module: `github.com/handloom/admin`, Go 1.25.

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
- `internal/middleware/` — RequestID, Logger, Recoverer, Auth (admin JWT), CustomerAuth (customer JWT), Validation (generics-based)
- `pkg/errors/` — typed `AppError` with error codes mapping to HTTP statuses
- `pkg/response/` — standard JSON envelope: `{success, data, meta}` or `{success, error: {code, message}}`

### Database Design
Hybrid storage: DynamoDB (6 tables) for core/transactional data + Neon PostgreSQL for catalog data.

#### DynamoDB Tables
All using PK/SK composite keys with GSIs:
| Table | Env Name | Content |
|-------|----------|---------|
| `handloom-core-{env}` | `DYNAMODB_CORE_TABLE` | Users, Pricing Rules, UTM Links |
| `handloom-orders-{env}` | `DYNAMODB_ORDERS_TABLE` | Orders, Customers, PriceQuotes |
| `handloom-audit-{env}` | `DYNAMODB_AUDIT_TABLE` | Audit logs (TTL-based expiry) |
| `handloom-notifications-{env}` | `DYNAMODB_NOTIFICATIONS_TABLE` | Notifications |
| `handloom-sessions-{env}` | `DYNAMODB_SESSIONS_TABLE` | OTP, refresh tokens (TTL-based expiry) |
| `handloom-coupons-{env}` | `DYNAMODB_COUPONS_TABLE` | Coupons, code pointer index, per-customer redemption counters, redemption history, bulk batches |

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

### Schema Migrations
- SQL files in `migrations/` are embedded via `go:embed` (`migrations/embed.go`) into the migrator Lambda
- `cmd/lambda/migrator/main.go` — connects to Neon PostgreSQL, creates `schema_migrations` tracking table, applies unapplied `.sql` files in filename order, each in a transaction
- CDK `triggers.Trigger` in `infra/stacks/database.go` invokes the migrator after Neon PostgreSQL is ready, re-runs when migration files change
- Uses `pg_advisory_lock` for concurrency safety
- Migration failure causes CDK deployment rollback
- To add a migration: create `migrations/NNN_description.sql`, then `make cdk-deploy-dev`

### Infrastructure (infra/)
AWS CDK in Go. Stacks per environment: LogsStack, DatabaseStack, StorageStack, EmbedderStack, MetricsStack, APIStack. All Lambdas use ARM64/128MB(dev)/256MB(prod)/provided.al2023. Lambda count: 23 in dev (13 admin + 9 store + 1 migrator).

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

## Git Workflow

**Branch naming**: `fix/` (bug fixes), `feat/` (features), `chore/` (housekeeping), `docs/` (documentation only). Prefix all branches with the type.

**Strategy**: Rebase before merging to main (via `git rebase main` then force-push). Squash fixup commits locally before pushing (`git rebase -i` to mark commits as `squash` or `fixup`). One logical commit per PR unless the PR is a rollup of independent changes.

**Commit messages**: Conventional format — `type(scope): description` (e.g., `fix(search): match SKUs in product queries`, `feat(store): infinite scroll on catalog`). Keep the first line under 70 chars. Body details go below a blank line if needed.

## Database Migrations

**PostgreSQL safety**: Migrations run in `cmd/lambda/migrator/` as part of `cdk deploy`. Always test locally first:
1. Add migration file `migrations/NNN_description.sql`
2. Run `make teardown-local && make setup-local` to apply it
3. Test thoroughly with integration tests (`make test-integration`)
4. Only then commit and deploy

**Rollback**: The migrator runs each migration in a transaction. If a migration fails, it rolls back and `cdk deploy` fails — this prevents partial deployments. To rollback a deployed migration, you must create a new reverse migration (e.g., `migrations/013_undo_012.sql`) and deploy it.

**Schema safety**: Avoid `DROP TABLE` in migrations (data loss risk). Use `DROP TABLE IF EXISTS` with caution. For backward-compatible changes, prefer adding columns with defaults over renaming/removing. Test migrations in both directions (up + down) for complex changes.

**DynamoDB**: No migrations needed — schema is implicitly defined by the code. Changes to key patterns or table structure require manual table recreation (use `make reset-db` locally; in prod, coordinate with the team to avoid data loss).

## Common Debugging

**LocalStack issues**: If `make deploy-local` fails, check:
- `docker ps` — LocalStack should be running
- `docker logs localstack_main` — view LocalStack logs
- `curl http://localhost:4566/health` — check LocalStack health
- `make teardown-local && make setup-local` — full reset (wipes data)

**DynamoDB inspection**: Use the DynamoDB Admin UI at `http://localhost:8001` (running when `make docker-up` is active). Query items directly or inspect table structure.

**PostgreSQL**: Connect locally with `psql $POSTGRES_DSN` (must have `postgres` command installed). Inspect schema: `\dt` (tables), `\d <table>` (columns), `\di` (indexes). Slow queries: enable `log_statement = 'all'` in Docker Compose and check logs.

**Lambda logs locally**: `make deploy-local` logs appear in the terminal. In AWS: `aws logs tail /aws/lambda/<function-name> --follow --region ap-south-1`.

**JWT debugging**: Decode tokens at https://jwt.io (copy the `access_token` cookie value). Verify expiry and claims match expectations. For custom claims, check `handler/auth.go` / `internal/service/auth_service.go`.

**Race conditions**: Run tests with `-race` flag (default in `make test`). If races appear in integration tests, add `SELECT ... FOR UPDATE` in the relevant transaction or increase lock contention test coverage.

## Dependency Updates

**Go modules**: `go get -u <module>` updates to the latest version matching `go.mod` constraints. Always run `go mod tidy` after. Test locally with `make test` before committing.

**Security patches**: Subscribe to Go security advisories (https://pkg.go.dev/vuln/). If a vulnerability is found, `go get -u <module>` and redeploy. For Lambda, this triggers a new CDK build.

**Major version updates**: Test extensively in a local environment before upgrading (e.g., updating Chi router, pgx driver). Check for breaking changes in the module's changelog.

## Performance Profiling

**CPU profiling**: Use pprof via `http://localhost:6060/debug/pprof/` when running monolith (`make run`). Get 30-second CPU profile: `go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30`. Inspect with `top`, `list <function>`.

**Memory profiling**: `go tool pprof http://localhost:6060/debug/pprof/heap` — check allocations and identify leaks.

**Lambda profiling**: Use CloudWatch Lambda Insights or add `log.Printf()` statements around expensive operations. Monitor cold start time with `Init Duration` metric in CloudWatch.

**Database query optimization**: Check `cmd/embedder/main.go` and catalog search queries for N+1 problems. Use `EXPLAIN ANALYZE` in PostgreSQL to inspect query plans. For DynamoDB, ensure GSI scans don't exceed provisioned capacity.

## Key Conventions

- **Handler validation**: Use `middleware.ValidateJSONTyped[T]` as Chi middleware, then `middleware.MustGetValidatedBody[T]` in handler
- **Error returns**: Services return `*errors.AppError`; handlers call `response.Error(w, err)`
- **Wire DI**: After adding/changing providers, run `make wire` to regenerate `wire_gen.go`
- **Mock generation**: After changing interfaces in `domain/`, run `make generate-mocks`
- **Import ordering**: enforced by goimports with local prefix `github.com/handloom/admin`
- **golangci-lint**: complexity thresholds — gocognit=30, gocyclo=25, dupl=200

## Embedder Secrets

The embedder Lambda + its callers (catalog + backfill Lambdas) need **one** SSM SecureString provisioned **out-of-band** (CDK does not create it):

| Parameter | Purpose | Rotation |
|---|---|---|
| `/handloom/{env}/embedder-auth-key` | HMAC shared secret between the embedder server-side `/embed` route and the catalog/backfill Lambdas. | Manual. Generate a new value, update SSM, then force a cold start on all consumer Lambdas (e.g. `aws lambda update-function-configuration --function-name <fn> --environment Variables={...}` to bump an env var). |

`POSTGRES_DSN` is **not** in SSM. It's passed to all Lambdas (including the embedder) as a plain env var by CDK at deploy time, sourced from:
- `handloom-admin/.env.{env}` for local `make cdk-deploy-{env}` (the Makefile target does `set -a && . ./.env.{env} && set +a`).
- `secrets.BACKEND_ENV_{ENV}` written to `.env.deploy` by the `deploy-backend.yml` workflow.

### Provisioning the auth key

```bash
ENV=dev
aws ssm put-parameter \
  --name "/handloom/${ENV}/embedder-auth-key" \
  --type SecureString \
  --value "$(openssl rand -hex 32)" \
  --region ap-south-1
```

## Embedder First-Deploy (fresh environment)

The embedder Lambda's container image is built by CDK during `cdk synth` from `cmd/embedder/Dockerfile` (`Code_FromAssetImage`). CDK pushes the image to its bootstrap ECR repo automatically — no custom ECR repo, no manual `docker push`, no S3 cache bucket.

The Dockerfile copies in three runtime assets (model + tokenizer + ONNX runtime) that live under `cmd/embedder/assets/`. The `prepare-embedder-assets` Make target (a prereq of `cdk-deploy-{env}`) runs `scripts/bootstrap-embedder-assets.sh` which populates them. First run is slow (~10 min — downloads from HuggingFace + exports to ONNX + quantizes); subsequent runs hit `~/.cache/handloom-embedder/` and finish in seconds.

One-command flow per environment:

```bash
ENV=dev  # or prod

# Step 1 — provision the auth key SSM SecureString (one time per env)
aws ssm put-parameter --name "/handloom/${ENV}/embedder-auth-key" \
  --type SecureString --value "$(openssl rand -hex 32)" --region ap-south-1

# Step 2 — deploy everything. CDK synth depends on prepare-embedder-assets,
# which runs the bootstrap script. Bootstrap is idempotent + cached.
cd handloom-admin && make cdk-deploy-${ENV}
```

For subsequent embedder code changes: just run `make cdk-deploy-${ENV}`. CDK content-hashes the Docker build context; rebuilds + redeploys the Lambda image only when something under `cmd/embedder/**` actually changed.
