# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Handloom Admin is a Go serverless backend for a handloom e-commerce admin panel (Homechrome brand). It runs as 14 independent AWS Lambda microservices in production and a single monolithic server locally. Go module: `github.com/handloom/admin`, Go 1.24.

## Common Commands

### Local Development
```bash
make setup-local      # One-command: docker-up + create-tables + init-s3 + seed-data
make run              # Start local API server on :8080 (monolith mode)
make run-watch        # Hot reload via air
make docker-up        # Start LocalStack (:4566) + DynamoDB Admin UI (:8001)
make deploy-local     # Build + deploy Lambdas to LocalStack (Lambda mode)
make redeploy-local   # Redeploy Lambda code only (faster)
make teardown-local   # Stop all Docker services and remove volumes
```

### Build
```bash
make build                # Build local binary → bin/handloom-api
make build-lambdas        # Build all 14 Lambda binaries (linux/arm64)
make build-lambdas-active # Build only active 4: auth, user, catalog, asset
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
- `internal/handler/` — HTTP handlers, one per domain; each exposes `Routes() chi.Router`
- `internal/service/` — business logic implementing domain service interfaces
- `internal/repository/dynamodb/` — DynamoDB implementations of domain repository interfaces
- `internal/router/` — mounts handlers onto Chi routers; `lambda.go` provides the Lambda adapter
- `internal/wire/` — Google Wire compile-time DI; one `Initialize*Deps()` per Lambda
- `internal/middleware/` — RequestID, Logger, Recoverer, Auth (JWT from cookie or Bearer header), Validation (generics-based)
- `pkg/errors/` — typed `AppError` with error codes mapping to HTTP statuses
- `pkg/response/` — standard JSON envelope: `{success, data, meta}` or `{success, error: {code, message}}`

### DynamoDB Single-Table Design
Four tables, all using PK/SK composite keys with GSIs:
| Table | Env Name | Content |
|-------|----------|---------|
| `handloom-core-{env}` | `DYNAMODB_CORE_TABLE` | Users, Categories, Products, Inventory, Pricing, Artisans, Coupons |
| `handloom-orders-{env}` | `DYNAMODB_ORDERS_TABLE` | Orders, Customers, PriceQuotes |
| `handloom-audit-{env}` | `DYNAMODB_AUDIT_TABLE` | Audit logs (TTL-based expiry) |
| `handloom-analytics-{env}` | `DYNAMODB_ANALYTICS_TABLE` | Dashboard metrics (TTL-based expiry) |

Key patterns: `PK=USER#<id> SK=METADATA`, `PK=PRODUCT#<id> SK=METADATA`, etc. Prices stored in **paise** (1 INR = 100 paise). Pagination is cursor-based (base64-encoded DynamoDB `ExclusiveStartKey`).

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

### Auth Strategy
- JWT in `access_token` HttpOnly cookie (primary), `Authorization: Bearer` header (fallback)
- JWT secret: `JWT_SECRET_KEY` env var, or fetched from AWS SSM at `/handloom/{env}/jwt-secret`
- Roles: `ADMIN` (bypasses permission checks), `OPERATOR`
- Cookie settings adapt: secure+SameSite=Lax for custom domain, secure+SameSite=None for Lambda URL, insecure for local

### Infrastructure (infra/)
AWS CDK in Go. Three stacks per environment: DatabaseStack, StorageStack, APIStack. Currently only 4 of 14 services are active in CDK (auth, user, catalog, asset). All Lambdas use ARM64/128MB(dev)/256MB(prod)/provided.al2023.

## Key Conventions

- **Handler validation**: Use `middleware.ValidateJSONTyped[T]` as Chi middleware, then `middleware.MustGetValidatedBody[T]` in handler
- **Error returns**: Services return `*errors.AppError`; handlers call `response.Error(w, err)`
- **Wire DI**: After adding/changing providers, run `make wire` to regenerate `wire_gen.go`
- **Mock generation**: After changing interfaces in `domain/`, run `make generate-mocks`
- **Import ordering**: enforced by goimports with local prefix `github.com/handloom/admin`
- **golangci-lint**: complexity thresholds — gocognit=30, gocyclo=25, dupl=200
