# Migrate Logging to slog Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the DI-injected `*logger.Logger` (zerolog wrapper) with Go's stdlib `log/slog` using a global default logger, eliminating logger DI from ~35 structs and their Wire providers.

**Architecture:** Initialize `slog.SetDefault()` once in each entry point (`cmd/api/main.go` and each `cmd/lambda/*/main.go`). Replace all `s.logger.WithContext(ctx).Infof(...)` calls with `slog.InfoContext(ctx, ...)`. Use a custom `slog.Handler` that extracts `request_id` and `user_id` from context automatically. Remove `*logger.Logger` from all struct fields, constructors, Wire providers, and Wire deps.

**Tech Stack:** Go stdlib `log/slog` (available since Go 1.21, project uses Go 1.25)

---

## Migration Strategy

This is a large mechanical refactoring (~90 files, ~35 structs, ~100 test file occurrences of `logger.NewNoop()`). The plan is ordered to keep the codebase compiling at each commit:

1. **Build the new slog infrastructure** (handler, context helpers, initialization)
2. **Migrate callers layer by layer** (services → handlers → middleware → event handlers → entry points)
3. **Remove old logger package and Wire plumbing**

Each "migrate layer" task is independent by package — subagents can work on them without conflicts.

**Note on compilation**: From Task 5 through Task 10, the full `go build ./...` will be broken because signatures change before all callers are updated. Each task verifies its own package compiles. Full build is restored in Task 10.

**Log format change**: Switching from zerolog to slog changes the JSON log schema (e.g., `"level":"info"` → `"level":"INFO"`, different timestamp format). Update any CloudWatch Insights queries or log-based alarms after migration.

## File Structure

| Action | File | Purpose |
|--------|------|---------|
| Create | `pkg/slogx/handler.go` | Custom slog.Handler that enriches log records with request_id/user_id from context |
| Create | `pkg/slogx/setup.go` | `Setup(debug bool)` — configures and sets `slog.SetDefault()` |
| Create | `pkg/slogx/context.go` | Context helpers: `SetRequestID`, `GetRequestID`, `SetUserID`, `GetUserID` |
| Create | `pkg/slogx/handler_test.go` | Tests for context-enriching handler |
| Create | `pkg/slogx/setup_test.go` | Tests for Setup() |
| Modify | `internal/middleware/middleware.go` | RequestID, Logger, Recoverer — use slog |
| Modify | `internal/router/common.go` | Remove `*logger.Logger` from `NewBaseRouter` and `NewAuthenticatedRouter` |
| Modify | `internal/event/publisher.go` | Remove logger from `LocalPublisher` struct and `ProvideEventPublisher` |
| Modify | All ~35 struct files | Remove `logger *logger.Logger` field, constructor param, and calls |
| Modify | `internal/wire/providers.go` | Remove ProvideLogger + logger params from all providers |
| Modify | `internal/wire/wire.go` | Remove Logger from all Deps structs and CoreSet |
| Modify | `cmd/api/main.go` | Call `slogx.Setup(debug)` at startup, remove logger plumbing |
| Modify | `cmd/lambda/*/main.go` (all 26) | Call `slogx.Setup(debug)` at startup |
| Delete | `pkg/logger/logger.go` | Old logger package |

---

### Task 1: Create slogx package — context helpers

**Files:**
- Create: `handloom-admin/pkg/slogx/context.go`

These context helpers replace the ones in `pkg/logger/logger.go` (lines 125-161). They must use the same context key type and values so they're compatible during the gradual migration.

- [ ] **Step 1: Create context helpers**

```go
package slogx

import "context"

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	userIDKey    contextKey = "user_id"
)

// SetRequestID stores the request ID in context.
func SetRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// SetUserID stores the user ID in context.
func SetUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// GetUserID retrieves the user ID from context.
func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey).(string); ok {
		return id
	}
	return ""
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd handloom-admin && go build ./pkg/slogx/...
```

- [ ] **Step 3: Commit**

```bash
git add pkg/slogx/context.go
git commit -m "feat(logging): add slogx context helpers"
```

---

### Task 2: Create slogx package — context-enriching handler

**Files:**
- Create: `handloom-admin/pkg/slogx/handler.go`
- Create: `handloom-admin/pkg/slogx/handler_test.go`

This handler wraps any `slog.Handler` and automatically injects `request_id` and `user_id` from the context into every log record. This replaces the `WithContext(ctx)` pattern.

- [ ] **Step 1: Write the handler test**

```go
package slogx

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestContextHandler_AddsRequestID(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, nil)
	h := NewContextHandler(inner)
	logger := slog.New(h)

	ctx := SetRequestID(context.Background(), "req-123")
	logger.InfoContext(ctx, "test message")

	var record map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record["request_id"] != "req-123" {
		t.Errorf("expected request_id=req-123, got %v", record["request_id"])
	}
}

func TestContextHandler_AddsUserID(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, nil)
	h := NewContextHandler(inner)
	logger := slog.New(h)

	ctx := SetUserID(context.Background(), "user-456")
	logger.InfoContext(ctx, "test message")

	var record map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record["user_id"] != "user-456" {
		t.Errorf("expected user_id=user-456, got %v", record["user_id"])
	}
}

func TestContextHandler_NoContextValues(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, nil)
	h := NewContextHandler(inner)
	logger := slog.New(h)

	logger.InfoContext(context.Background(), "test message")

	var record map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if _, exists := record["request_id"]; exists {
		t.Error("request_id should not be present when not in context")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd handloom-admin && go test ./pkg/slogx/... -v -run TestContextHandler
```

Expected: FAIL — `NewContextHandler` not defined.

- [ ] **Step 3: Implement the handler**

```go
package slogx

import (
	"context"
	"log/slog"
)

// ContextHandler wraps a slog.Handler and enriches each record with
// request_id and user_id from context.
type ContextHandler struct {
	inner slog.Handler
}

// NewContextHandler creates a handler that extracts context values into log attributes.
func NewContextHandler(inner slog.Handler) *ContextHandler {
	return &ContextHandler{inner: inner}
}

func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := GetRequestID(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	if id := GetUserID(ctx); id != "" {
		r.AddAttrs(slog.String("user_id", id))
	}
	return h.inner.Handle(ctx, r)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{inner: h.inner.WithGroup(name)}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd handloom-admin && go test ./pkg/slogx/... -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/slogx/handler.go pkg/slogx/handler_test.go
git commit -m "feat(logging): add slog context-enriching handler"
```

---

### Task 3: Create slogx package — Setup function

**Files:**
- Create: `handloom-admin/pkg/slogx/setup.go`
- Create: `handloom-admin/pkg/slogx/setup_test.go`

- [ ] **Step 1: Write the setup test**

```go
package slogx

import (
	"log/slog"
	"testing"
)

func TestSetup_Debug(t *testing.T) {
	Setup(true)
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("debug mode should enable debug level")
	}
}

func TestSetup_Production(t *testing.T) {
	Setup(false)
	if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("production mode should not enable debug level")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd handloom-admin && go test ./pkg/slogx/... -v -run TestSetup
```

- [ ] **Step 3: Implement Setup**

```go
package slogx

import (
	"log/slog"
	"os"
)

// Setup configures the global slog default logger.
// Debug mode: text handler with source info, debug level.
// Production: JSON handler, info level.
func Setup(debug bool) {
	var level slog.Level
	var inner slog.Handler

	if debug {
		level = slog.LevelDebug
		inner = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     level,
			AddSource: true,
		})
	} else {
		level = slog.LevelInfo
		inner = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	}

	handler := NewContextHandler(inner)
	slog.SetDefault(slog.New(handler))
}
```

- [ ] **Step 4: Run tests**

```bash
cd handloom-admin && go test ./pkg/slogx/... -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/slogx/setup.go pkg/slogx/setup_test.go
git commit -m "feat(logging): add slogx.Setup for global logger initialization"
```

---

### Task 4: Bridge — make both loggers work during migration

**Files:**
- Modify: `handloom-admin/pkg/logger/logger.go`

During migration, both `pkg/logger` and `slogx` coexist. Update the context helper functions in `pkg/logger` to delegate to `pkg/slogx` so context values set by either package are shared.

- [ ] **Step 1: Update pkg/logger context helpers to use slogx keys**

In `pkg/logger/logger.go`, update `SetRequestID`, `GetRequestID`, `SetUserID`, `GetUserID` to delegate to `slogx`:

```go
import "github.com/handloom/admin/pkg/slogx"

func SetRequestID(ctx context.Context, id string) context.Context {
	return slogx.SetRequestID(ctx, id)
}

func GetRequestID(ctx context.Context) string {
	return slogx.GetRequestID(ctx)
}

func SetUserID(ctx context.Context, id string) context.Context {
	return slogx.SetUserID(ctx, id)
}

func GetUserID(ctx context.Context) string {
	return slogx.GetUserID(ctx)
}
```

Also update `WithContext` to read from the same keys. Check if the `contextKey` type in both packages is compatible — since they're both `string`-typed context keys with the same string values (`"request_id"`, `"user_id"`), the values will be shared.

**IMPORTANT:** Verify that the context key types are compatible. `pkg/logger` defines `type contextKey string` and `pkg/slogx` defines the same. Since Go context keys include the type in comparison, these are DIFFERENT types. To fix this, either:
- (a) Make `pkg/logger` delegate to `pkg/slogx` functions (which use `slogx.contextKey`), OR
- (b) Export the key type from `slogx` and use it in both.

Option (a) is cleanest. Replace the `SetRequestID`/`GetRequestID`/`SetUserID`/`GetUserID` functions in `pkg/logger` to call through to `pkg/slogx`.

- [ ] **Step 2: Verify tests still pass**

```bash
cd handloom-admin && go test ./pkg/logger/... ./pkg/slogx/... -v
```

- [ ] **Step 3: Run full test suite**

```bash
cd handloom-admin && make test-unit
```

- [ ] **Step 4: Commit**

```bash
git add pkg/logger/logger.go
git commit -m "refactor(logging): bridge pkg/logger context helpers to slogx"
```

---

### Task 5: Migrate middleware to slog

**Files:**
- Modify: `handloom-admin/internal/middleware/middleware.go`
- Modify: `handloom-admin/internal/middleware/customer_auth.go`
- Modify: `handloom-admin/internal/middleware/optional_cart_auth.go`

The middleware is the first layer to migrate because it's where request_id and user_id are set in context.

- [ ] **Step 1: Migrate RequestID middleware**

In `middleware.go`, update the `RequestID` function. Replace:
```go
ctx = logger.SetRequestID(ctx, requestID)
```
with:
```go
ctx = slogx.SetRequestID(ctx, requestID)
```

Update the import from `"github.com/handloom/admin/pkg/logger"` to `"github.com/handloom/admin/pkg/slogx"`.

- [ ] **Step 2: Migrate Logger middleware**

Replace the `Logger` middleware function. It currently takes `*logger.Logger` parameter. Change to:

```go
func Logger() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(ww, r)

			slog.InfoContext(r.Context(), "HTTP request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.statusCode,
				"duration", time.Since(start).String(),
				"user_agent", r.UserAgent(),
				"remote_ip", getRemoteIP(r),
			)
		})
	}
}
```

Note: `Logger()` no longer takes a `*logger.Logger` param. All callers must be updated (entry points).

- [ ] **Step 3: Migrate Recoverer middleware**

Replace the `Recoverer` function. Currently takes `*logger.Logger`. Change to:

```go
func Recoverer() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					slog.ErrorContext(r.Context(), "Panic recovered", "panic", err)
					response.InternalError(w)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 4: Migrate Auth middleware**

In `middleware.go`, update the `Auth` struct — remove `logger *logger.Logger` field and constructor param. Replace `a.logger.WithContext(ctx)...` calls with `slog.ErrorContext(ctx, ...)`. Update `logger.SetUserID` to `slogx.SetUserID`.

- [ ] **Step 5: Migrate CustomerAuth middleware**

In `customer_auth.go`, remove `logger *logger.Logger` field from `CustomerAuth` struct. Remove from `NewCustomerAuth` constructor. Replace `logger.SetUserID` with `slogx.SetUserID`.

- [ ] **Step 6: Migrate OptionalCartAuth middleware**

In `optional_cart_auth.go`, remove `logger *logger.Logger` field from `OptionalCartAuth` struct. Remove from `NewOptionalCartAuth` constructor. Replace `logger.SetUserID` with `slogx.SetUserID`.

- [ ] **Step 7: Migrate router/common.go**

`internal/router/common.go` has `NewBaseRouter(cfg Config, log *logger.Logger, addHealthCheck bool)` and `NewAuthenticatedRouter(cfg Config, log *logger.Logger, authMiddleware *middleware.Auth)`. Remove the `log *logger.Logger` parameter from both. They pass `log` to `middleware.Logger(log)` and `middleware.Recoverer(log)` — update those calls to `middleware.Logger()` and `middleware.Recoverer()`.

- [ ] **Step 8: Verify middleware + router compile**

```bash
cd handloom-admin && go build ./internal/middleware/... ./internal/router/...
```

- [ ] **Step 9: Commit**

```bash
git add internal/middleware/ internal/router/
git commit -m "refactor(logging): migrate middleware and router to slog"
```

---

### Task 6: Migrate all services to slog

**Files:**
- Modify: All 20 service files in `handloom-admin/internal/service/`

This is the largest task. For each service file:
1. Remove `logger *logger.Logger` from the struct
2. Remove `logger *logger.Logger` from the constructor params and body
3. Replace all `s.logger.WithContext(ctx).Infof("...", args)` with `slog.InfoContext(ctx, "...", args...)`
4. Replace all `s.logger.WithContext(ctx).Warnf("...", args)` with `slog.WarnContext(ctx, "...", args...)`
5. Replace all `s.logger.WithContext(ctx).Errorf("...", args)` with `slog.ErrorContext(ctx, "...", args...)`
6. Replace all `s.logger.WithContext(ctx).WithError(err).Error("...")` with `slog.ErrorContext(ctx, "...", "error", err)`
7. Remove `"github.com/handloom/admin/pkg/logger"` import, add `"log/slog"` import

**slog call pattern translation:**

| Old (zerolog wrapper) | New (slog) |
|---|---|
| `s.logger.WithContext(ctx).Infof("Added item %s for %s", itemID, owner)` | `slog.InfoContext(ctx, "Added item", "item_id", itemID, "cart_owner", owner)` |
| `s.logger.WithContext(ctx).Warnf("Failed to merge item %s: %v", id, err)` | `slog.WarnContext(ctx, "Failed to merge item", "product_id", id, "error", err)` |
| `s.logger.WithContext(ctx).WithError(err).Error("msg")` | `slog.ErrorContext(ctx, "msg", "error", err)` |
| `s.logger.WithContext(ctx).WithField("k", v).Info("msg")` | `slog.InfoContext(ctx, "msg", "k", v)` |
| `s.logger.WithContext(ctx).WithFields(m).Info("msg")` | Convert map entries to alternating key-value args |

**IMPORTANT**: `slog` uses structured key-value pairs, not `printf`-style formatting. Convert `Infof("User %s logged in", id)` to `slog.InfoContext(ctx, "User logged in", "user_id", id)` — NOT `slog.InfoContext(ctx, fmt.Sprintf(...))`.

**Service files to modify (20):**
1. `auth_service.go`
2. `user_service.go`
3. `category_service.go`
4. `product_service.go`
5. `inventory_service.go`
6. `pricing_service.go`
7. `order_service.go`
8. `customer_service.go`
9. `analytics_service.go`
10. `notification_service.go`
11. `coupon_service.go`
12. `asset_service.go`
13. `report_service.go`
14. `audit_service.go`
15. `cart_service.go`
16. `checkout_service.go`
17. `customer_auth_service.go`
18. `payment_service.go`
19. `shipping_service.go`
20. `analytics_aggregator.go`

- [ ] **Step 1: Migrate all service files**

For each file, apply the pattern above. Use search-and-replace where possible.

- [ ] **Step 1b: Migrate service test files — remove `logger.NewNoop()`**

There are ~100 occurrences of `logger.NewNoop()` across test files (primarily in `internal/service/*_test.go`). For each test file:
- Remove `logger.NewNoop()` from service constructor calls (e.g., `NewCartService(repo, prodRepo, invRepo, logger.NewNoop())` becomes `NewCartService(repo, prodRepo, invRepo)`)
- Remove `"github.com/handloom/admin/pkg/logger"` import
- Find all: `grep -rn 'logger.NewNoop()' internal/service/ --include='*_test.go'`

Also check and update test files in `internal/event/`:
- `internal/event/publisher_test.go` — `NewLocalPublisher` call
- `internal/event/handlers/handlers_test.go` — handler constructor calls

- [ ] **Step 2: Verify services compile**

```bash
cd handloom-admin && go build ./internal/service/...
```

- [ ] **Step 3: Run unit tests**

```bash
cd handloom-admin && make test-unit
```

- [ ] **Step 4: Commit**

```bash
git add internal/service/
git commit -m "refactor(logging): migrate all services to slog"
```

---

### Task 7: Migrate all handlers to slog

**Files:**
- Modify: All 11 admin handler files in `handloom-admin/internal/handler/`
- Modify: All 7 store handler files in `handloom-admin/internal/handler/store/`

Same pattern as Task 6 — remove `logger` field, constructor param, replace calls.

**Admin handlers (11):** `auth_handler.go`, `user_handler.go`, `category_handler.go`, `product_handler.go`, `inventory_handler.go`, `pricing_handler.go`, `order_handler.go`, `customer_handler.go`, `audit_handler.go`, `notification_handler.go`, `analytics_handler.go`, `report_handler.go`, `asset_handler.go`

**Store handlers (7):** `auth_handler.go`, `cart_handler.go`, `checkout_handler.go`, `events_handler.go`, `profile_handler.go`, `tracking_handler.go`, `webhook_handler.go`

**Note:** Some handlers have `h.logger` (not `s.logger`). The receiver variable may differ — check each file.

- [ ] **Step 1: Migrate all handler files**

- [ ] **Step 2: Verify handlers compile**

```bash
cd handloom-admin && go build ./internal/handler/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/handler/
git commit -m "refactor(logging): migrate all handlers to slog"
```

---

### Task 8: Migrate event handlers to slog

**Files:**
- Modify: `handloom-admin/internal/event/handlers/analytics.go`
- Modify: `handloom-admin/internal/event/handlers/audit.go`
- Modify: `handloom-admin/internal/event/handlers/notification.go`
- Modify: `handloom-admin/internal/event/handlers/report.go`
- Modify: `handloom-admin/internal/event/publisher.go` — `LocalPublisher` struct has `log *logger.Logger` field, `NewLocalPublisher` takes it

Same pattern. Remove `logger` field from all event handler structs and `LocalPublisher`, replace calls.

- [ ] **Step 1: Migrate event handler files**

- [ ] **Step 2: Verify compilation**

```bash
cd handloom-admin && go build ./internal/event/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/event/
git commit -m "refactor(logging): migrate event handlers to slog"
```

---

### Task 9: Update Wire providers — remove logger from all providers

**Files:**
- Modify: `handloom-admin/internal/wire/providers.go`
- Modify: `handloom-admin/internal/wire/wire.go`

- [ ] **Step 1: Remove ProvideLogger from providers.go**

Delete the `ProvideLogger` function (lines 33-36).

- [ ] **Step 2: Remove logger param from ALL service providers**

For every `Provide*Service` function, remove the `log *logger.Logger` parameter and the corresponding argument in the constructor call. Example:

Before:
```go
func ProvideCartService(
	cartRepo domain.CartRepository,
	productRepo domain.ProductRepository,
	inventoryRepo domain.InventoryRepository,
	log *logger.Logger,
) *service.CartService {
	return service.NewCartService(cartRepo, productRepo, inventoryRepo, log)
}
```

After:
```go
func ProvideCartService(
	cartRepo domain.CartRepository,
	productRepo domain.ProductRepository,
	inventoryRepo domain.InventoryRepository,
) *service.CartService {
	return service.NewCartService(cartRepo, productRepo, inventoryRepo)
}
```

Apply to ALL ~20 service providers and ALL ~15 handler providers.

- [ ] **Step 3: Remove logger param from ALL middleware and event providers**

Update `ProvideAuthMiddleware`, `ProvideCustomerAuthMiddleware`, `ProvideOptionalCartAuth` — remove `log *logger.Logger`.

Also update `ProvideEventPublisher` — it takes `log *logger.Logger` even though `SNSPublisher`/`NoopPublisher` don't use it. Remove the param.

- [ ] **Step 4: Remove Logger from all Deps structs in wire.go**

Every `*Deps` struct has `Logger *logger.Logger`. Remove that field from ALL of them.

- [ ] **Step 5: Remove ProvideLogger from CoreSet in wire.go**

Find the `CoreSet` wire.ProviderSet and remove `ProvideLogger` from it.

- [ ] **Step 6: Remove unused logger import from providers.go**

Remove `"github.com/handloom/admin/pkg/logger"` from imports.

- [ ] **Step 7: Commit (pre-wire regen)**

```bash
git add internal/wire/providers.go internal/wire/wire.go
git commit -m "refactor(logging): remove logger DI from Wire providers and deps"
```

---

### Task 10: Update entry points and regenerate Wire

**Files:**
- Modify: `handloom-admin/cmd/api/main.go`
- Modify: All `handloom-admin/cmd/lambda/*/main.go` (26 files)
- Regenerate: `handloom-admin/internal/wire/wire_gen.go`

- [ ] **Step 1: Update monolith main.go**

In `cmd/api/main.go`:
1. Add `"github.com/handloom/admin/pkg/slogx"` import
2. After `cfg := config.Load()`, add: `slogx.Setup(cfg.App.Debug)`
3. Remove `log := logger.New(cfg.App.Debug)` and all uses of `log` variable
4. Replace `logger.New(cfg.App.Debug)` with nothing — services/handlers no longer take logger
5. Update `middleware.Logger(log)` to `middleware.Logger()` (no param)
6. Update `middleware.Recoverer(log)` to `middleware.Recoverer()` (no param)
7. Remove logger from all `New*Handler`, `New*Service`, `New*Middleware` calls
8. Replace any remaining `log.Info(...)` / `log.Fatalf(...)` with `slog.Info(...)` / fatal with `slog.Error(...)` + `os.Exit(1)` (slog has no Fatal — use `slog.Error` + `os.Exit(1)`)

- [ ] **Step 2: Update API Lambda main.go files (22 non-worker Lambdas)**

Each API Lambda follows the same pattern. For each `cmd/lambda/*/main.go` (excluding worker-* Lambdas):
1. Add `"github.com/handloom/admin/pkg/slogx"` and `"log/slog"` imports
2. Replace `log := logger.New(cfg.App.Debug)` with `slogx.Setup(cfg.App.Debug)`
3. Replace `log.Info("Starting ...")` with `slog.Info("Starting ...")`
4. Replace `log.Fatalf("Failed: %v", err)` with `slog.Error("Failed", "error", err); os.Exit(1)`
5. Remove `deps.Logger` references (no longer in deps struct)
6. Update `router.NewBaseRouter(routerCfg, deps.Logger, ...)` → `router.NewBaseRouter(routerCfg, ...)`

```bash
find cmd/lambda -name 'main.go' ! -path '*/worker-*/*' | sort
```

- [ ] **Step 3: Update worker Lambda main.go files (4 files)**

Worker Lambdas (`worker-analytics`, `worker-audit`, `worker-notification`, `worker-report`) do NOT use Wire — they construct dependencies manually. For each:
1. Replace `log := logger.New(...)` with `slogx.Setup(...)`
2. Remove `log` from handler constructor calls (e.g., `handlers.NewAnalyticsHandler(repo, log)` → `handlers.NewAnalyticsHandler(repo)`)
3. Replace `log.Info/Fatalf` with `slog.Info/Error + os.Exit`

- [ ] **Step 4: Update createRouter function in cmd/api/main.go**

The `createRouter` function takes `log *logger.Logger` as a parameter. Remove it. Update the call site to not pass `log`. Inside `createRouter`, the `middleware.Logger()` and `middleware.Recoverer()` calls should already be parameterless after Task 5.

- [ ] **Step 4: Regenerate Wire**

```bash
cd handloom-admin && make wire
```

- [ ] **Step 5: Full build**

```bash
cd handloom-admin && go build ./...
```

- [ ] **Step 6: Run all tests**

```bash
cd handloom-admin && make test
```

- [ ] **Step 7: Commit**

```bash
git add cmd/ internal/wire/wire_gen.go internal/router/
git commit -m "refactor(logging): update entry points to slogx.Setup, regenerate Wire"
```

---

### Task 11: Delete old logger package

**Files:**
- Delete: `handloom-admin/pkg/logger/logger.go`

- [ ] **Step 1: Verify no imports remain**

```bash
cd handloom-admin && grep -r '"github.com/handloom/admin/pkg/logger"' --include='*.go' .
```

Expected: no matches. If any remain, fix them first.

- [ ] **Step 2: Delete the old package**

```bash
rm -rf pkg/logger/
```

- [ ] **Step 3: Remove zerolog dependency**

```bash
go mod tidy
```

This should remove `github.com/rs/zerolog` from `go.mod` if nothing else uses it.

- [ ] **Step 4: Full build + tests**

```bash
cd handloom-admin && go build ./... && make test
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor(logging): remove pkg/logger and zerolog dependency"
```

---

### Task 12: Update mocks and verify

**Files:**
- Regenerate: mocks

- [ ] **Step 1: Regenerate mocks**

```bash
cd handloom-admin && make generate-mocks
```

- [ ] **Step 2: Fix any test compilation errors**

Service constructor mocks may reference the old `logger` parameter. Update test files.

- [ ] **Step 3: Run full test suite**

```bash
cd handloom-admin && make test
```

- [ ] **Step 4: Run linter**

```bash
cd handloom-admin && golangci-lint run
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore: regenerate mocks and fix tests after slog migration"
```
