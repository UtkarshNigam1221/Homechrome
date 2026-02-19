# Backend Refactor & TDD Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor the handloom-admin Go backend to a clean golang-standards layout, fix 15 structural issues, and achieve 90%+ test coverage — all without breaking existing behavior.

**Architecture:** Test-First Foundation approach. Write comprehensive tests (service unit, handler HTTP, repo integration) against current code, then refactor with the safety net of tests. Domain god files (`entity.go`, `service.go`, `repository.go`) split into per-domain files. Dead code and artifacts removed.

**Tech Stack:** Go 1.24, Chi router, DynamoDB, gomock, testify, Google Wire, golangci-lint

---

## Phase 1: Service Unit Tests

Write tests for all 14 services against the CURRENT code. Each service task follows the same TDD pattern: create test file → write table-driven tests → run to verify they pass (we're testing existing code, not writing new code). Mock all repository interfaces using gomock mocks in `internal/mocks/`.

**Pattern conventions** (apply to ALL service tests):
- Package: `package service` (same package for whitebox access to unexported helpers)
- Imports: `testify/assert`, `testify/require`, `go.uber.org/mock/gomock`
- Setup: `gomock.NewController(t)` + `defer ctrl.Finish()` + mock repos + `logger.New(true)`
- Structure: table-driven `tests := []struct{ name; setup; input; wantErr; errCode }` with `t.Run`
- Assertions: `require.NoError`/`require.Error` for control flow, `assert.*` for values

### Task 1: Extend Auth Service Tests

**Files:**
- Modify: `internal/service/auth_service_test.go`

**Step 1: Add missing test cases for RequestPasswordReset and ResetPassword**

The existing test file covers Login, ValidateToken, RefreshToken, Logout, and ChangePassword. Add tests for the two missing methods:

```go
func TestAuthService_RequestPasswordReset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.New(true)

	svc := NewAuthService(userRepo, tokenStore, log, "test-secret", 15*time.Minute, 7*24*time.Hour, "test-issuer")
	ctx := context.Background()

	tests := []struct {
		name    string
		email   string
		setup   func()
		wantErr bool
	}{
		{
			name:  "existing user - stores reset token",
			email: "test@example.com",
			setup: func() {
				userRepo.EXPECT().GetByEmail(ctx, "test@example.com").Return(&domain.User{
					ID:    "user_123",
					Email: "test@example.com",
				}, nil)
				tokenStore.EXPECT().StorePasswordResetToken(ctx, "user_123", gomock.Any(), time.Hour).Return(nil)
			},
			wantErr: false,
		},
		{
			name:  "non-existent email - succeeds silently (no leak)",
			email: "noone@example.com",
			setup: func() {
				userRepo.EXPECT().GetByEmail(ctx, "noone@example.com").Return(nil, errors.NotFound("User"))
			},
			wantErr: false,
		},
		{
			name:  "repo error - returns error",
			email: "test@example.com",
			setup: func() {
				userRepo.EXPECT().GetByEmail(ctx, "test@example.com").Return(nil, errors.Internal("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := svc.RequestPasswordReset(ctx, tt.email)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAuthService_ResetPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.New(true)

	svc := NewAuthService(userRepo, tokenStore, log, "test-secret", 15*time.Minute, 7*24*time.Hour, "test-issuer")
	ctx := context.Background()

	tests := []struct {
		name    string
		req     domain.ResetPasswordRequest
		setup   func()
		wantErr bool
	}{
		{
			name: "valid reset token",
			req:  domain.ResetPasswordRequest{Token: "valid-token", NewPassword: "newpass123"},
			setup: func() {
				tokenStore.EXPECT().ValidatePasswordResetToken(ctx, "valid-token").Return("user_123", nil)
				userRepo.EXPECT().GetByID(ctx, "user_123").Return(&domain.User{ID: "user_123"}, nil)
				userRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
				tokenStore.EXPECT().RevokePasswordResetToken(ctx, "valid-token").Return(nil)
				tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "invalid reset token",
			req:  domain.ResetPasswordRequest{Token: "bad-token", NewPassword: "newpass123"},
			setup: func() {
				tokenStore.EXPECT().ValidatePasswordResetToken(ctx, "bad-token").Return("", errors.New(errors.ErrCodeInvalidToken, "expired"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := svc.ResetPassword(ctx, tt.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
```

**Step 2: Add edge case tests for Login (token store failures)**

```go
// Add to TestAuthService_Login test cases:
{
	name: "store refresh token fails - returns error",
	req: domain.LoginRequest{Email: "test@example.com", Password: "password123"},
	setup: func() {
		userRepo.EXPECT().GetByEmail(ctx, "test@example.com").Return(&domain.User{
			ID: "user_123", Email: "test@example.com",
			PasswordHash: string(hashedPassword), Status: domain.UserStatusActive,
			Role: domain.UserRoleAdmin, Permissions: []string{"*"},
		}, nil)
		tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(nil)
		tokenStore.EXPECT().StoreRefreshToken(ctx, "user_123", gomock.Any(), gomock.Any()).Return(errors.Internal("redis down"))
	},
	wantErr: true,
	errCode: "INTERNAL",
},
```

**Step 3: Run tests**

Run: `cd handloom-admin && go test -v -run TestAuthService ./internal/service/...`
Expected: ALL PASS

**Step 4: Commit**

```bash
git add internal/service/auth_service_test.go
git commit -m "test: extend auth service tests - password reset, edge cases"
```

---

### Task 2: Write User Service Tests

**Files:**
- Create: `internal/service/user_service_test.go`

**Step 1: Write comprehensive tests for all 6 UserService methods**

```go
package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
)

func newUserServiceTest(t *testing.T) (*UserService, *mocks.MockUserRepository, *mocks.MockTokenStore) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.New(true)
	svc := NewUserService(userRepo, tokenStore, log)
	return svc, userRepo, tokenStore
}

func TestUserService_Create(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		req       domain.CreateUserRequest
		createdBy string
		setup     func(*mocks.MockUserRepository, *mocks.MockTokenStore)
		wantErr   bool
		errCode   string
		validate  func(t *testing.T, user *domain.User)
	}{
		{
			name: "successful creation",
			req: domain.CreateUserRequest{
				Email: "new@example.com", Password: "password123",
				FirstName: "John", LastName: "Doe",
				Role: domain.UserRoleOperator, Permissions: []string{"read"},
			},
			createdBy: "admin_1",
			setup: func(ur *mocks.MockUserRepository, ts *mocks.MockTokenStore) {
				ur.EXPECT().GetByEmail(ctx, "new@example.com").Return(nil, errors.NotFound("User"))
				ur.EXPECT().Create(ctx, gomock.Any()).Return(nil)
			},
			validate: func(t *testing.T, user *domain.User) {
				assert.Contains(t, user.ID, "user_")
				assert.Equal(t, "new@example.com", user.Email)
				assert.Empty(t, user.PasswordHash, "password hash should be stripped")
				assert.Equal(t, domain.UserStatusPending, user.Status)
				assert.Equal(t, domain.UserRoleOperator, user.Role)
			},
		},
		{
			name: "duplicate email",
			req: domain.CreateUserRequest{
				Email: "existing@example.com", Password: "password123",
				FirstName: "Jane", LastName: "Doe", Role: domain.UserRoleOperator,
			},
			createdBy: "admin_1",
			setup: func(ur *mocks.MockUserRepository, ts *mocks.MockTokenStore) {
				ur.EXPECT().GetByEmail(ctx, "existing@example.com").Return(&domain.User{ID: "user_existing"}, nil)
			},
			wantErr: true,
			errCode: "ALREADY_EXISTS",
		},
		{
			name: "repo create error",
			req: domain.CreateUserRequest{
				Email: "new@example.com", Password: "password123",
				FirstName: "John", LastName: "Doe", Role: domain.UserRoleOperator,
			},
			createdBy: "admin_1",
			setup: func(ur *mocks.MockUserRepository, ts *mocks.MockTokenStore) {
				ur.EXPECT().GetByEmail(ctx, "new@example.com").Return(nil, errors.NotFound("User"))
				ur.EXPECT().Create(ctx, gomock.Any()).Return(errors.Internal("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, userRepo, tokenStore := newUserServiceTest(t)
			tt.setup(userRepo, tokenStore)
			user, err := svc.Create(ctx, tt.req, tt.createdBy)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, user)
				}
			}
		})
	}
}

func TestUserService_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("found - strips password hash", func(t *testing.T) {
		svc, userRepo, _ := newUserServiceTest(t)
		userRepo.EXPECT().GetByID(ctx, "user_123").Return(&domain.User{
			ID: "user_123", Email: "test@example.com", PasswordHash: "hashed",
		}, nil)
		user, err := svc.GetByID(ctx, "user_123")
		require.NoError(t, err)
		assert.Empty(t, user.PasswordHash)
	})

	t.Run("not found", func(t *testing.T) {
		svc, userRepo, _ := newUserServiceTest(t)
		userRepo.EXPECT().GetByID(ctx, "user_999").Return(nil, errors.NotFound("User"))
		_, err := svc.GetByID(ctx, "user_999")
		require.Error(t, err)
	})
}

func TestUserService_Update(t *testing.T) {
	ctx := context.Background()
	firstName := "Updated"

	t.Run("successful partial update", func(t *testing.T) {
		svc, userRepo, _ := newUserServiceTest(t)
		userRepo.EXPECT().GetByID(ctx, "user_123").Return(&domain.User{
			ID: "user_123", FirstName: "Original", Email: "test@example.com",
		}, nil)
		userRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
			assert.Equal(t, "Updated", u.FirstName)
			assert.Equal(t, "admin_1", u.UpdatedBy)
			return nil
		})
		user, err := svc.Update(ctx, "user_123", domain.UpdateUserRequest{FirstName: &firstName}, "admin_1")
		require.NoError(t, err)
		assert.Equal(t, "Updated", user.FirstName)
		assert.Empty(t, user.PasswordHash)
	})

	t.Run("user not found", func(t *testing.T) {
		svc, userRepo, _ := newUserServiceTest(t)
		userRepo.EXPECT().GetByID(ctx, "user_999").Return(nil, errors.NotFound("User"))
		_, err := svc.Update(ctx, "user_999", domain.UpdateUserRequest{FirstName: &firstName}, "admin_1")
		require.Error(t, err)
	})

	t.Run("update with password change", func(t *testing.T) {
		svc, userRepo, _ := newUserServiceTest(t)
		newPass := "newpassword123"
		userRepo.EXPECT().GetByID(ctx, "user_123").Return(&domain.User{ID: "user_123"}, nil)
		userRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
			assert.NotEmpty(t, u.PasswordHash, "password should be hashed")
			return nil
		})
		_, err := svc.Update(ctx, "user_123", domain.UpdateUserRequest{Password: &newPass}, "admin_1")
		require.NoError(t, err)
	})
}

func TestUserService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("successful delete revokes tokens", func(t *testing.T) {
		svc, userRepo, tokenStore := newUserServiceTest(t)
		userRepo.EXPECT().Delete(ctx, "user_123").Return(nil)
		tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(nil)
		err := svc.Delete(ctx, "user_123")
		require.NoError(t, err)
	})

	t.Run("delete - token revoke failure is non-fatal", func(t *testing.T) {
		svc, userRepo, tokenStore := newUserServiceTest(t)
		userRepo.EXPECT().Delete(ctx, "user_123").Return(nil)
		tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(errors.Internal("redis down"))
		err := svc.Delete(ctx, "user_123")
		require.NoError(t, err) // should still succeed
	})

	t.Run("repo delete error", func(t *testing.T) {
		svc, userRepo, _ := newUserServiceTest(t)
		userRepo.EXPECT().Delete(ctx, "user_999").Return(errors.NotFound("User"))
		err := svc.Delete(ctx, "user_999")
		require.Error(t, err)
	})
}

func TestUserService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("strips password hashes from all users", func(t *testing.T) {
		svc, userRepo, _ := newUserServiceTest(t)
		userRepo.EXPECT().List(ctx, gomock.Any()).Return(&domain.ListUsersResponse{
			Users: []*domain.User{
				{ID: "u1", PasswordHash: "hash1"},
				{ID: "u2", PasswordHash: "hash2"},
			},
			Pagination: domain.PaginationResponse{Limit: 20},
		}, nil)
		resp, err := svc.List(ctx, domain.ListUsersRequest{})
		require.NoError(t, err)
		for _, u := range resp.Users {
			assert.Empty(t, u.PasswordHash)
		}
	})
}

func TestUserService_UpdateStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("successful status update", func(t *testing.T) {
		svc, userRepo, _ := newUserServiceTest(t)
		userRepo.EXPECT().GetByID(ctx, "user_123").Return(&domain.User{
			ID: "user_123", Status: domain.UserStatusPending,
		}, nil)
		userRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, u *domain.User) error {
			assert.Equal(t, domain.UserStatusActive, u.Status)
			assert.Equal(t, "admin_1", u.UpdatedBy)
			assert.False(t, u.UpdatedAt.IsZero())
			return nil
		})
		err := svc.UpdateStatus(ctx, "user_123", domain.UserStatusActive, "admin_1")
		require.NoError(t, err)
	})
}
```

**Step 2: Run tests**

Run: `cd handloom-admin && go test -v -run TestUserService ./internal/service/...`
Expected: ALL PASS

**Step 3: Commit**

```bash
git add internal/service/user_service_test.go
git commit -m "test: add user service tests - full coverage for all 6 methods"
```

---

### Task 3: Write Category Service Tests

**Files:**
- Create: `internal/service/category_service_test.go`

**Test cases to cover** (follow the exact same pattern as Task 2):

| Method | Test Cases |
|--------|-----------|
| `Create` | success, slug generation from name, repo error |
| `GetByID` | found, not found |
| `Update` | partial update (name only), full update, not found |
| `Delete` | success, not found, repo error |
| `List` | returns results, empty results, with status filter |
| `AddAttribute` | success, category not found, duplicate attr name |
| `UpdateAttribute` | success, attr not found |
| `DeleteAttribute` | success, attr not found |
| `GetAttributes` | success, category not found |

Read `internal/service/category_service.go` first to understand the implementation details before writing tests.

**Step 1: Write tests** — follow the helper pattern from Task 2
**Step 2: Run** — `go test -v -run TestCategoryService ./internal/service/...`
**Step 3: Commit** — `git commit -m "test: add category service tests"`

---

### Task 4: Write Product Service Tests

**Files:**
- Create: `internal/service/product_service_test.go`

**Test cases to cover:**

| Method | Test Cases |
|--------|-----------|
| `Create` | success (generates ID, slug, sets Draft status), duplicate SKU, category not found, image finalization (temp→assets), repo error |
| `GetByID` | found with relations (category + inventory), not found |
| `Update` | partial update, image finalization on update, product not found, attribute index sync |
| `Delete` | success (deletes with attribute indexes), not found |
| `List` | with filters (category, status, price range, attribute filters), pagination |
| `GetAttributeFilterOptions` | returns distinct values |

**Important**: ProductService depends on `CategoryRepository`, `ProductRepository`, `InventoryRepository`, and `AssetFinalizer` — mock all of them.

Read `internal/service/product_service.go` first. This is the most complex service.

**Step 1: Write tests**
**Step 2: Run** — `go test -v -run TestProductService ./internal/service/...`
**Step 3: Commit** — `git commit -m "test: add product service tests"`

---

### Task 5: Extend Order Service Tests

**Files:**
- Modify: `internal/service/order_service_test.go`

Read the existing test file first. Add missing test cases:

| Method | Test Cases to Add |
|--------|-------------------|
| `Create` | customer not found, product not found, inventory insufficient, coupon validation, quote-based pricing |
| `UpdateStatus` | valid transitions (PENDING→CONFIRMED→PROCESSING→SHIPPED→DELIVERED), invalid transitions, inventory reservation on confirm, release on cancel |
| `AddNote` | success, order not found |
| `UpdateTracking` | success, order not found |
| `CancelOrder` | success (releases inventory), already cancelled, delivered (can't cancel) |
| `RefundOrder` | success, partial refund, already refunded |

**Step 1: Read existing tests, extend**
**Step 2: Run** — `go test -v -run TestOrderService ./internal/service/...`
**Step 3: Commit** — `git commit -m "test: extend order service tests - state machine, cancel, refund"`

---

### Task 6: Extend Pricing Service Tests

**Files:**
- Modify: `internal/service/pricing_service_test.go`

Read the existing file. Add tests for:

| Method | Test Cases to Add |
|--------|-------------------|
| `CreateRule` | all scope types (GLOBAL, CATEGORY, PRODUCT, MATERIAL), validation of required fields per scope |
| `CalculatePrice` | area-based calculation, length-based, tiered pricing, material multipliers, attribute surcharges, min order value check |
| `BulkCalculatePrice` | multiple configs, max 10 limit, mixed success/error |
| `GetDimensionOptions` | returns options with pricing attributes |
| `GetQuote` | found, not found, expired |

**Step 1: Read existing tests, extend**
**Step 2: Run** — `go test -v -run TestPricingService ./internal/service/...`
**Step 3: Commit** — `git commit -m "test: extend pricing service tests - calculations, bulk, quotes"`

---

### Task 7: Extend Inventory Service Tests

**Files:**
- Modify: `internal/service/inventory_service_test.go`

Read existing file. Add tests for:

| Method | Test Cases |
|--------|-----------|
| `GetByProductID` | found, not found |
| `AddStock` | success (updates product denormalized qty), zero quantity error |
| `RemoveStock` | success, insufficient stock error |
| `AdjustStock` | success, negative quantity error |
| `GetTransactions` | returns paginated transactions |
| `GetLowStockProducts` | returns only low stock items |

**Step 1-3: Read, write, run, commit** — `git commit -m "test: extend inventory service tests"`

---

### Task 8: Write Asset Service Tests

**Files:**
- Create: `internal/service/asset_service_test.go`

Read `internal/service/asset_service.go`. Tests:

| Method | Test Cases |
|--------|-----------|
| `GenerateUploadURL` | success (returns presigned URL + tmp key), invalid content type |
| `FinalizeIfTemp` | temp URL → copies to assets/, already permanent → returns as-is, S3 copy error |
| `DeleteAsset` | success, S3 error |

**Step 1-3: Write, run, commit** — `git commit -m "test: add asset service tests"`

---

### Task 9: Write Remaining Service Tests (Batch)

Write tests for the 6 simpler CRUD-only services. Each follows the same pattern.

**Files to create:**
- `internal/service/artisan_service_test.go`
- `internal/service/coupon_service_test.go`
- `internal/service/notification_service_test.go`
- `internal/service/customer_service_test.go`
- `internal/service/analytics_service_test.go`
- `internal/service/report_service_test.go`

For each service, read the implementation first, then write tests covering:
- All public methods: happy path + not found + repo error
- Any business logic (coupon validation, analytics date ranges)
- Status transitions where applicable

Read each service implementation file before writing its tests.

**Step 1: Write all 6 test files**
**Step 2: Run** — `go test -v ./internal/service/...`
**Step 3: Commit** — `git commit -m "test: add tests for artisan, coupon, notification, customer, analytics, report services"`

---

### Task 10: Write Bulk Service Tests

**Files:**
- Create: `internal/service/bulk_service_test.go`

This service is complex — it has import/export processors. Read `internal/service/bulk_service.go` and `internal/service/bulk_export_processor.go`.

| Method | Test Cases |
|--------|-----------|
| `CreateOperation` | success, validates entity type |
| `ImportProducts` | success (dry run), success (actual), CSV parse errors, validation errors per row |
| `ExportProducts` | success (CSV format), success (JSON format), empty result set |
| `UpdateInventoryBulk` | SET/ADD/SUBTRACT operations |
| `GetOperation` | found, not found |
| `ListOperations` | with filters |

**Step 1-3: Write, run, commit** — `git commit -m "test: add bulk service tests - import/export processors"`

---

### Task 11: Run Full Test Suite and Check Coverage

**Step 1: Run all service tests with coverage**

Run: `cd handloom-admin && go test -v -race -coverprofile=coverage.out ./internal/service/...`

**Step 2: Check coverage**

Run: `go tool cover -func=coverage.out | grep -E '^total:|service\.go'`
Expected: 90%+ for service layer

**Step 3: If coverage < 90%, identify gaps and add tests**

Run: `go tool cover -html=coverage.out -o coverage.html`
Open `coverage.html` in browser and add tests for any uncovered branches.

**Step 4: Commit coverage report**

```bash
echo "coverage.out" >> .gitignore
echo "coverage.html" >> .gitignore
git add internal/service/*_test.go .gitignore
git commit -m "test: achieve 90%+ coverage for service layer"
```

---

## Phase 2: Split Domain God Files

Pure file reorganization — no logic changes. Tests from Phase 1 validate nothing breaks.

### Task 12: Split entity.go into per-domain files

**Files:**
- Modify: `internal/domain/entity.go` (will be deleted at end)
- Create: `internal/domain/user.go`
- Create: `internal/domain/category.go`
- Create: `internal/domain/product.go`
- Create: `internal/domain/pricing.go`
- Create: `internal/domain/inventory.go`

**Step 1: Create `internal/domain/user.go`**

Move from `entity.go` lines 10-178:
- `UserRole` type + constants
- `UserStatus` type + constants
- `BaseEntity` struct (stays in a shared file — actually move to `common.go`)
- `User` struct + `TableName()` + `SetKeys()`

**Step 2: Create `internal/domain/common.go`**

Move shared types that don't belong to any single domain:
- `BaseEntity` struct (from `entity.go`)
- `PaginationRequest`, `PaginationResponse` (from `repository.go`)
- `Dimensions` struct (from `entity.go` — used by Product, Order, PriceQuote)

**Step 3: Create `internal/domain/category.go`**

Move from `entity.go`:
- `CategoryStatus` type + constants
- `AttributeType` type + constants
- `Category` struct + `TableName()` + `SetKeys()`
- `CategoryAttribute`, `AttributeOption`, `DimensionConfig` structs
- `CategoryAttributeValues` struct + `SetKeys()`
- `ProductImage` struct (used by Product but defined in category section — keep with product)

**Step 4: Create `internal/domain/product.go`**

Move from `entity.go`:
- `ProductStatus` type + constants
- `Product` struct + `TableName()` + `SetKeys()` + `NewProduct()` + `ApplyUpdate()`
- `ProductImage` struct
- `ProductAttributeIndex` struct + `SetKeys()`

**Step 5: Create `internal/domain/pricing.go`**

Move from `entity.go`:
- `PricingRuleScope`, `PricingType`, `PricingUnit`, `SurchargeType` types + constants
- `PricingRule` struct + `TableName()` + `SetKeys()`
- `AttributeSurcharge`, `PricingTier` structs
- `PriceQuote` struct + `TableName()` + `SetKeys()`
- `PriceBreakdown`, `SurchargeDetail` structs

**Step 6: Create `internal/domain/inventory.go`**

Move from `entity.go`:
- `InventoryTransactionType` type + constants
- `Inventory` struct + `TableName()` + `SetKeys()`
- `InventoryTransaction` struct + `TableName()` + `SetKeys()`

**Step 7: Delete `entity.go`**

**Step 8: Run tests**

Run: `cd handloom-admin && go test -v -race ./internal/service/...`
Expected: ALL PASS (imports are package-level, file moves within same package don't break anything)

**Step 9: Run build**

Run: `cd handloom-admin && go build ./...`
Expected: SUCCESS

**Step 10: Commit**

```bash
git add internal/domain/
git commit -m "refactor: split domain/entity.go into per-domain files"
```

---

### Task 13: Split service.go into per-domain files

**Files:**
- Modify: `internal/domain/service.go` (will be deleted)
- Modify: `internal/domain/user.go` (add UserService interface + DTOs)
- Create: `internal/domain/auth.go` (AuthService interface + DTOs)
- Modify: `internal/domain/category.go` (add CategoryService + DTOs)
- Modify: `internal/domain/product.go` (add ProductService + DTOs)
- Modify: `internal/domain/pricing.go` (add PricingService + DTOs)
- Modify: `internal/domain/inventory.go` (add InventoryService + DTOs)

**Step 1: Move AuthService + DTOs to `internal/domain/auth.go`**

Move lines 10-73 from `service.go`:
- `AuthService` interface
- `LoginRequest`, `LoginResponse`, `TokenPair`, `TokenClaims`
- `ChangePasswordRequest`, `ResetPasswordRequest`

**Step 2: Append UserService + DTOs to `internal/domain/user.go`**

Move lines 75-118 from `service.go`:
- `UserService` interface
- `CreateUserRequest`, `UpdateUserRequest`

**Step 3: Append CategoryService + DTOs to `internal/domain/category.go`**

Move lines 120-180 from `service.go`.

**Step 4: Append ProductService + DTOs to `internal/domain/product.go`**

Move lines 182-260 from `service.go`.

**Step 5: Append PricingService + DTOs to `internal/domain/pricing.go`**

Move lines 262-438 from `service.go`.

**Step 6: Append InventoryService + DTOs to `internal/domain/inventory.go`**

Move lines 449-500 from `service.go`. Include `AssetFinalizer` interface in `asset.go`.

**Step 7: Delete `service.go`**

**Step 8: Update `//go:generate` directive**

The old directive was `//go:generate mockgen -source=service.go`. Now split across files. Add directives to each file that contains interfaces:
- `user.go`: `//go:generate mockgen -source=user.go -destination=../mocks/user_mock.go -package=mocks`
- `auth.go`: `//go:generate mockgen -source=auth.go -destination=../mocks/auth_mock.go -package=mocks`
- etc.

**Alternative**: Keep a single mockgen command that targets the package rather than individual files. Use the `-package` flag with `reflect` mode:
```
//go:generate mockgen -destination=../mocks/service_mock.go -package=mocks github.com/handloom/admin/internal/domain AuthService,UserService,CategoryService,ProductService,PricingService,InventoryService,AssetFinalizer
```

Place this in `common.go`.

**Step 9: Run tests and build**

Run: `cd handloom-admin && go test -v -race ./internal/service/... && go build ./...`
Expected: ALL PASS

**Step 10: Commit**

```bash
git add internal/domain/
git commit -m "refactor: split domain/service.go into per-domain files"
```

---

### Task 14: Split repository.go into per-domain files

**Files:**
- Modify: `internal/domain/repository.go` (will be deleted)
- Modify: `internal/domain/user.go` (add UserRepository)
- Modify: `internal/domain/category.go` (add CategoryRepository)
- Modify: `internal/domain/product.go` (add ProductRepository)
- Modify: `internal/domain/pricing.go` (add PricingRuleRepository, PriceQuoteRepository)
- Modify: `internal/domain/inventory.go` (add InventoryRepository)
- Modify: `internal/domain/common.go` (PaginationRequest/Response already moved)

**Step 1: Move each repository interface + its list request/response types to the corresponding domain file**

Same pattern as Task 13. Move `UserRepository` + `ListUsersRequest` + `ListUsersResponse` to `user.go`, etc.

**Step 2: Delete `repository.go`**

**Step 3: Update `//go:generate` directive** — same approach as Task 13.

**Step 4: Run tests and build**

Run: `cd handloom-admin && go test -v -race ./internal/service/... && go build ./...`

**Step 5: Commit**

```bash
git add internal/domain/
git commit -m "refactor: split domain/repository.go into per-domain files"
```

---

### Task 15: Merge audit_repository.go and order_repository.go into their domain files

**Files:**
- Delete: `internal/domain/audit_repository.go` → merge into `internal/domain/audit.go`
- Delete: `internal/domain/order_repository.go` → merge into `internal/domain/order.go`

**Step 1: Append audit_repository.go contents to audit.go**

Move `AuditRepository`, `AuditService`, `ListAuditLogsRequest`, `ListAuditLogsResponse` into `audit.go`.

**Step 2: Append order_repository.go contents to order.go**

Move `OrderRepository`, `CustomerRepository`, `OrderService`, all order request/response DTOs into `order.go`.

**Step 3: Delete the old files**

**Step 4: Run tests and build**

**Step 5: Commit** — `git commit -m "refactor: consolidate audit and order domain files"`

---

### Task 16: Clean up asset.go — separate bulk.go and report.go

**Files:**
- Modify: `internal/domain/asset.go` (trim to only asset types)
- Modify: `internal/domain/bulk.go` (already exists — has BulkOperation types, keep + add BulkJob from asset.go)
- Create: `internal/domain/report.go` (move Report entity from asset.go)

**Step 1: Move BulkJob entity + BulkJobRepository + BulkJob DTOs from asset.go to bulk.go**

Keep only these in `asset.go`:
- `AssetType` constants
- `UploadAssetRequest`, `UploadURLResponse`, `DeleteAssetRequest`
- `AssetFinalizer` interface (if not moved in Task 13)

**Step 2: Create `internal/domain/report.go`**

Move from `asset.go`:
- `ReportType`, `ReportFormat`, `ReportStatus` types + constants
- `Report` struct + `TableName()` + `SetKeys()`
- `ReportRepository` interface
- `GenerateReportRequest`, `ListReportsRequest`, `ListReportsResponse`

**Step 3: Run tests and build**

**Step 4: Commit** — `git commit -m "refactor: separate bulk and report domain types from asset.go"`

---

## Phase 3: Code Quality Fixes

### Task 17: Fix Customer.TotalSpent float64 → int64 paise

**Files:**
- Modify: `internal/domain/order.go` line 175: `TotalSpent float64` → `TotalSpent int64`
- Search all references to `TotalSpent` and update accordingly

**Step 1: Find all references**

Run: `grep -rn "TotalSpent" internal/`

**Step 2: Update the type**

In `internal/domain/order.go`:
```go
// Before:
TotalSpent   float64        `json:"total_spent" dynamodbav:"total_spent"`
// After:
TotalSpent   int64          `json:"total_spent" dynamodbav:"total_spent"` // in paise
```

**Step 3: Update any code that sets/reads TotalSpent with float64 arithmetic**

Search in `internal/service/order_service.go` and `internal/repository/dynamodb/order_repository.go`.

**Step 4: Run tests** — `go test -v -race ./...`
**Step 5: Commit** — `git commit -m "fix: change Customer.TotalSpent from float64 to int64 paise"`

---

### Task 18: Use table name constants everywhere

**Files:**
- Modify: `internal/domain/constants.go` — add `TableAudit` and `TableAnalytics`
- Modify: all `TableName()` methods to use constants

**Step 1: Update constants.go**

```go
const (
	TableCore      = "handloom-core"
	TableOrders    = "handloom-orders"
	TableAudit     = "handloom-audit"
	TableAnalytics = "handloom-analytics"
)
```

**Step 2: Update all TableName() methods**

Find all: `grep -rn 'func.*TableName.*string' internal/domain/`

Replace hardcoded strings:
- `"handloom-core"` → `TableCore`
- `"handloom-orders"` → `TableOrders`
- `"handloom-audit"` → `TableAudit`
- `"handloom-analytics"` → `TableAnalytics`

**Step 3: Run tests and build**
**Step 4: Commit** — `git commit -m "refactor: use table name constants in all TableName() methods"`

---

### Task 19: Standardize BulkJobRepository vs BulkOperationRepository naming

**Files:**
- Modify: `internal/domain/bulk.go` — ensure consistent naming
- Modify: `internal/domain/asset.go` — if `BulkJobRepository` is here
- Modify: `internal/wire/providers.go` — rename `ProvideBulkOperationRepository`
- Modify: `internal/wire/wire.go` — update provider set
- Modify: `internal/repository/dynamodb/bulk_repository.go` — rename constructor

**Step 1: Decide on name** — use `BulkOperationRepository` (matches `BulkOperation` entity in `bulk.go`)

Note: There are TWO bulk systems — `BulkJob` (in `asset.go`) and `BulkOperation` (in `bulk.go`). Determine which is actually used by examining Wire providers and services. Consolidate to one.

**Step 2: Rename and update all references**
**Step 3: Run `make wire`**
**Step 4: Run tests and build**
**Step 5: Commit** — `git commit -m "refactor: standardize bulk repository naming"`

---

### Task 20: Remove dead ValidateQuery and dto/ package

**Files:**
- Modify: `internal/middleware/validation.go` — remove `ValidateQuery` function
- Delete: `internal/dto/` directory entirely

**Step 1: Verify ValidateQuery is unused**

Run: `grep -rn "ValidateQuery" internal/ --include="*.go" | grep -v "_test.go" | grep -v "validation.go"`
Expected: No results (confirms it's dead code)

**Step 2: Remove ValidateQuery from validation.go**

**Step 3: Verify dto/ is unused**

Run: `grep -rn '"github.com/handloom/admin/internal/dto"' internal/ --include="*.go"`
If any files import it, update them to use `domain` types directly.

**Step 4: Delete internal/dto/**

**Step 5: Run tests and build**
**Step 6: Commit** — `git commit -m "refactor: remove dead ValidateQuery and unused dto/ package"`

---

## Phase 4: Structural Fixes

### Task 21: Wire cmd/api/main.go to use Wire DI

**Files:**
- Modify: `internal/wire/wire.go` — add `InitializeApp` injector
- Modify: `internal/wire/providers.go` — add app-level provider if needed
- Modify: `cmd/api/main.go` — replace manual wiring with Wire

**Step 1: Add `InitializeApp` injector to `wire.go`**

```go
type AppDeps struct {
	Router chi.Router
	Logger *logger.Logger
}

func InitializeAppDeps(ctx context.Context, cfg *config.Config) (*AppDeps, error) {
	wire.Build(CoreSet, /* all provider sets */)
	return nil, nil
}
```

**Step 2: Run `make wire`**

**Step 3: Rewrite cmd/api/main.go to use the generated function**

**Step 4: Run tests** — `go test -v -race ./...`
**Step 5: Run locally** — `make run` and test a few endpoints
**Step 6: Commit** — `git commit -m "refactor: wire cmd/api/main.go to use Wire DI"`

---

### Task 22: Fix health check auth + cleanup artifacts

**Files:**
- Modify: `internal/router/common.go` — move `/health` to base router (unauthenticated)
- Delete: `internal/service/order_service_test.go.bak`
- Modify: `.gitignore` — add `uploads/`, `exports/`, `bin/`

**Step 1: In `router/common.go`, move `/health` from `NewAuthenticatedRouter` to `NewBaseRouter`**

The health endpoint should be available without auth for Lambda warm-up pings.

**Step 2: Delete stale artifacts**

```bash
rm internal/service/order_service_test.go.bak
rm -rf uploads/ exports/
```

**Step 3: Update .gitignore**

```
uploads/
exports/
bin/
*.bak
coverage.out
coverage.html
```

**Step 4: Clean up duplicate bin/ directories in Makefile**

Ensure Makefile uses single `bin/lambda/` output path.

**Step 5: Run tests and build**
**Step 6: Commit** — `git commit -m "fix: unauthenticate health endpoint, cleanup stale artifacts"`

---

## Phase 5: Handler Tests

### Task 23: Write handler test infrastructure

**Files:**
- Create: `internal/handler/testhelper_test.go`

Set up shared test infrastructure for handler tests:

```go
package handler

import (
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/logger"
)

type handlerTestSuite struct {
	ctrl    *gomock.Controller
	router  chi.Router
	log     *logger.Logger
}

func executeRequest(router chi.Router, method, path string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
```

**Step 1: Write the helper**
**Step 2: Commit** — `git commit -m "test: add handler test infrastructure"`

---

### Task 24: Write Auth Handler Tests

**Files:**
- Create: `internal/handler/auth_handler_test.go`

Test HTTP layer: request decoding, response format, status codes, cookie handling.

| Endpoint | Test Cases |
|----------|-----------|
| `POST /login` | 200 with cookies set, 401 invalid credentials, 400 validation error |
| `POST /refresh` | 200 new tokens, 401 invalid token |
| `POST /logout` | 200 success, 401 unauthenticated |
| `POST /change-password` | 200 success, 400 validation, 401 wrong password |

**Step 1-3: Write, run, commit** — `git commit -m "test: add auth handler HTTP tests"`

---

### Task 25: Write User Handler Tests

**Files:**
- Create: `internal/handler/user_handler_test.go`

| Endpoint | Test Cases |
|----------|-----------|
| `POST /users` | 201 created, 400 validation, 409 duplicate |
| `GET /users/:id` | 200 found, 404 not found |
| `PUT /users/:id` | 200 updated, 404 not found |
| `DELETE /users/:id` | 204 deleted, 404 not found |
| `GET /users` | 200 with pagination |

**Step 1-3: Write, run, commit** — `git commit -m "test: add user handler HTTP tests"`

---

### Task 26: Write Remaining Handler Tests (Batch)

Write handler tests for the remaining services following the same pattern. Focus on:
- Correct HTTP status codes for each scenario
- Response envelope format (`{success, data}` or `{success, error}`)
- Pagination metadata in list responses
- Validation error format

**Files to create:**
- `internal/handler/category_handler_test.go`
- `internal/handler/product_handler_test.go`
- `internal/handler/order_handler_test.go`
- `internal/handler/pricing_handler_test.go`

**Step 1-3: Write, run, commit** — `git commit -m "test: add handler tests for catalog, order, pricing"`

---

## Phase 6: Integration Tests

### Task 27: Write Product Repository Integration Tests

**Files:**
- Create: `internal/repository/dynamodb/product_repository_test.go`

```go
//go:build integration

package dynamodb

// Tests require DynamoDB Local running on :8000
// Run: make docker-up && go test -v -tags=integration ./internal/repository/dynamodb/...
```

Test the most complex repository:
- `CreateWithAttributeIndexes` — verifies product + indexes + inventory in single transaction
- `List` with attribute filters — verifies GSI queries work
- `FilterByAttributes` — verifies the ATTR# key pattern
- Pagination with cursor encoding/decoding

**Step 1-3: Write, run (requires `make docker-up`), commit**

---

### Task 28: Write Order Repository Integration Tests

**Files:**
- Create: `internal/repository/dynamodb/order_repository_test.go`

Test order-specific patterns:
- `Create` + `GetByID` roundtrip
- `GetByCustomer` — verifies GSI1 (CUSTOMER#) query
- `UpdateStatus` + status history records
- `AddNote` — appends to internal_notes list

**Step 1-3: Write, run, commit**

---

### Task 29: Final Coverage Check and Cleanup

**Step 1: Run full test suite**

Run: `cd handloom-admin && go test -v -race -coverprofile=coverage.out ./internal/...`

**Step 2: Generate coverage report**

Run: `go tool cover -func=coverage.out | tail -1`
Expected: `total: (statements) 90.0%+`

**Step 3: Run linter**

Run: `golangci-lint run`
Expected: No errors

**Step 4: Run build**

Run: `go build ./...`

**Step 5: Final commit**

```bash
git add -A
git commit -m "refactor: complete backend restructure with 90%+ test coverage

- Split domain god files (entity.go, service.go, repository.go) into per-domain files
- Fixed Customer.TotalSpent float64 -> int64 paise
- Used table name constants everywhere
- Standardized bulk repository naming
- Removed dead ValidateQuery and unused dto/ package
- Wired cmd/api/main.go to use Wire DI
- Fixed unauthenticated health endpoint
- Cleaned up stale artifacts
- Added comprehensive service, handler, and integration tests"
```

---

## File Change Summary

### Files Created (Tests)
- `internal/service/user_service_test.go`
- `internal/service/category_service_test.go`
- `internal/service/product_service_test.go`
- `internal/service/asset_service_test.go`
- `internal/service/artisan_service_test.go`
- `internal/service/coupon_service_test.go`
- `internal/service/notification_service_test.go`
- `internal/service/customer_service_test.go`
- `internal/service/analytics_service_test.go`
- `internal/service/report_service_test.go`
- `internal/service/bulk_service_test.go`
- `internal/handler/testhelper_test.go`
- `internal/handler/auth_handler_test.go`
- `internal/handler/user_handler_test.go`
- `internal/handler/category_handler_test.go`
- `internal/handler/product_handler_test.go`
- `internal/handler/order_handler_test.go`
- `internal/handler/pricing_handler_test.go`
- `internal/repository/dynamodb/product_repository_test.go`
- `internal/repository/dynamodb/order_repository_test.go`

### Files Created (Domain Splits)
- `internal/domain/common.go`
- `internal/domain/auth.go`
- `internal/domain/report.go`

### Files Modified
- `internal/domain/user.go` (new — from entity.go split)
- `internal/domain/category.go` (new — from entity.go split)
- `internal/domain/product.go` (new — from entity.go split)
- `internal/domain/pricing.go` (new — from entity.go split)
- `internal/domain/inventory.go` (new — from entity.go split)
- `internal/domain/order.go` (merged order_repository.go + float64 fix)
- `internal/domain/audit.go` (merged audit_repository.go)
- `internal/domain/asset.go` (trimmed — bulk/report moved out)
- `internal/domain/bulk.go` (received BulkJob from asset.go)
- `internal/domain/constants.go` (added TableAudit, TableAnalytics)
- `internal/middleware/validation.go` (removed ValidateQuery)
- `internal/wire/wire.go` (added InitializeApp)
- `internal/wire/providers.go` (renamed bulk provider)
- `internal/router/common.go` (unauthenticated health)
- `cmd/api/main.go` (Wire DI instead of manual)
- `Makefile` (single bin output)
- `.gitignore` (added uploads/, exports/, bin/, *.bak)
- `internal/service/auth_service_test.go` (extended)
- `internal/service/order_service_test.go` (extended)
- `internal/service/pricing_service_test.go` (extended)
- `internal/service/inventory_service_test.go` (extended)

### Files Deleted
- `internal/domain/entity.go`
- `internal/domain/service.go`
- `internal/domain/repository.go`
- `internal/domain/order_repository.go`
- `internal/domain/audit_repository.go`
- `internal/dto/` (entire directory)
- `internal/service/order_service_test.go.bak`
- `uploads/` directory
- `exports/` directory
