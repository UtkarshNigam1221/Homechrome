# Backend Refactor & TDD Design

**Date**: 2026-02-19
**Approach**: Test-First Foundation (Approach A) — write comprehensive tests first, then refactor with safety net
**Target Structure**: golang-standards/project-layout with cleaned-up internals

## Scope

Full cleanup of 15 structural issues across 5 phases. 90%+ line coverage target.

## Phase 1: Lock Behavior with Tests

Write tests against the CURRENT code structure before moving anything.

### Layer 1 — Service Unit Tests (`internal/service/`)

One `<service>_test.go` per service, table-driven with `t.Run()` subtests.
Mock all repository interfaces using mockgen mocks in `internal/mocks/`.
Target: every public method has happy path + all error branches + edge cases.

| Service | File | Notes |
|---------|------|-------|
| auth | `auth_service_test.go` | Extend existing — add token refresh, password reset, edge cases |
| user | `user_service_test.go` | New — CRUD + role transitions + status changes |
| category | `category_service_test.go` | New — CRUD + attribute handling |
| product | `product_service_test.go` | New — variants, attributes, image finalization, category validation |
| pricing | `pricing_service_test.go` | Extend existing — rule engine, bulk calc, scope validation |
| inventory | `inventory_service_test.go` | Extend existing — stock adjustments, low-stock alerts, transactions |
| order | `order_service_test.go` | Extend existing — state machine, cancel, refund, notes |
| audit | `audit_service_test.go` | Review and extend coverage |
| asset | `asset_service_test.go` | New — presigned URL generation, finalize temp→assets flow |
| analytics | `analytics_service_test.go` | New — aggregation queries, date range handling |
| artisan | `artisan_service_test.go` | New — CRUD + status management |
| bulk | `bulk_service_test.go` | New — import/export processors, job state management |
| coupon | `coupon_service_test.go` | New — validation rules, apply logic, expiry |
| notification | `notification_service_test.go` | New — CRUD + read status |
| customer | `customer_service_test.go` | New — CRUD + order history |
| report | `report_service_test.go` | New — generation logic, export formats |

### Layer 2 — Handler HTTP Tests (`internal/handler/`)

One `<handler>_test.go` per handler using `httptest.NewRecorder` + `chi.NewRouter`.
Mock service interfaces. Verify: HTTP status codes, response envelope format, validation errors, auth errors.

### Layer 3 — Repository Integration Tests (`internal/repository/dynamodb/`)

Build tag: `//go:build integration`. Requires DynamoDB Local via Docker.
Existing: `user_repository_test.go`, `token_store_test.go`.
Add: `product_repository_test.go`, `order_repository_test.go` (most complex key patterns).

### Test Conventions
- Table-driven tests with `t.Run()` subtests
- `testify/assert` + `testify/require`
- Mock expectations set per-test, not globally
- Colocated: `<name>_test.go` next to `<name>.go`
- Integration tests: `//go:build integration` tag

## Phase 2: Split Domain God Files

Pure file reorganization — no logic changes. Run `make test` after each split.

### Target `internal/domain/` structure

```
internal/domain/
├── constants.go     # TableCore, TableOrders, TableAudit, TableAnalytics, SKMetadata
├── common.go        # PaginationRequest, PaginatedResult[T], CursorPagination
├── token_store.go   # TokenStore interface (unchanged)
├── user.go          # User + UserRepository + UserService + DTOs
├── category.go      # Category, CategoryAttribute + CategoryRepository + CategoryService + DTOs
├── product.go       # Product, ProductAttributeIndex + ProductRepository + ProductService + DTOs
├── pricing.go       # PricingRule, PriceQuote + repos + PricingService + DTOs
├── inventory.go     # Inventory, InventoryTransaction + repo + InventoryService + DTOs
├── auth.go          # AuthService + LoginRequest, TokenPair, etc. (extracted from service.go)
├── order.go         # Order, Customer, Address + repos + OrderService + DTOs (consolidate order_repository.go)
├── artisan.go       # (unchanged)
├── audit.go         # AuditLog + AuditRepository + AuditService (merge audit_repository.go)
├── asset.go         # AssetType + AssetService, AssetFinalizer (trimmed)
├── bulk.go          # BulkJob + BulkJobRepository + BulkService + DTOs
├── report.go        # Report + ReportRepository + ReportService
├── coupon.go        # (unchanged)
├── notification.go  # (unchanged)
├── analytics.go     # (unchanged)
```

Files to delete after merge:
- `entity.go` (split into user/category/product/pricing/inventory)
- `service.go` (split into per-domain files)
- `repository.go` (split into per-domain files)
- `order_repository.go` (merged into order.go)
- `audit_repository.go` (merged into audit.go)

## Phase 3: Code Quality Fixes

Each fix is a test-verified change. Run tests after each.

| # | Issue | Fix |
|---|-------|-----|
| 1 | `Customer.TotalSpent float64` | Change to `int64` paise, update all references |
| 2 | Hardcoded `"handloom-core"` in `TableName()` | Use `domain.TableCore` constant |
| 3 | Missing `TableAudit`, `TableAnalytics` constants | Add to `constants.go` |
| 4 | `BulkJobRepository` vs `BulkOperationRepository` | Standardize to `BulkJobRepository` everywhere |
| 5 | Dead `ValidateQuery` function | Remove from `middleware/validation.go` |
| 6 | `dto/` package duplicates domain types | Delete `internal/dto/`, move unique types to domain or handler-local |

## Phase 4: Structural Fixes

| # | Issue | Fix |
|---|-------|-----|
| 1 | `cmd/api/main.go` manual wiring (333 lines) | Create Wire injector `InitializeApp` for monolith |
| 2 | Health check behind auth in `NewAuthenticatedRouter` | Move `/health` to base router (unauthenticated) |
| 3 | Duplicate `bin/lambda/` vs `bin/lambdas/` | Clean Makefile to use single `bin/lambda/` output |
| 4 | Stale `.bak` file, `uploads/`, `exports/` | Delete artifacts, add to `.gitignore` |

## Phase 5: Telemetry Integration

Wire `pkg/telemetry/` into entry points:
- `cmd/api/main.go`: Initialize OTLP tracer, defer shutdown
- `cmd/lambda/*/main.go`: Initialize tracer with Lambda-appropriate config
- Add tracing middleware to router chain

## Execution Strategy

- **Approach**: Test-First Foundation — all tests written in Phase 1 before any refactoring
- **Safety**: `make test` runs after every atomic change in Phases 2-5
- **Commits**: One commit per logical change (e.g., "split entity.go into per-domain files")
- **Wire**: Run `make wire` after any DI-related changes
- **Mocks**: Run `make generate-mocks` after any interface changes
- **Coverage**: Target 90%+ line coverage across service layer
