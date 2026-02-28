# Service Test Expansion Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add ~65 edge-case tests across 5 services (auth, category, inventory, user, product) to catch behavior regressions in error codes, security invariants, event publishing, cache invalidation, and data integrity.

**Architecture:** All tests are unit tests using gomock + testify. Two new spy types (spyPublisher, spyCache) replace NoopPublisher/noopCache when tests need to assert on side effects. Tests go in existing `*_test.go` files.

**Tech Stack:** Go 1.24, gomock, testify (assert/require), existing mocks in `internal/mocks/`

---

### Task 1: Add shared test spy types

**Files:**
- Create: `internal/service/test_helpers_test.go`

**Step 1: Write the test helper file**

```go
package service

import (
	"context"
	"fmt"

	"github.com/handloom/admin/internal/event"
)

// spyPublisher records published events for test assertions.
type spyPublisher struct {
	events []event.Event
	err    error // if set, Publish returns this error
}

func newSpyPublisher() *spyPublisher {
	return &spyPublisher{}
}

func newFailingPublisher(err error) *spyPublisher {
	return &spyPublisher{err: err}
}

func (s *spyPublisher) Publish(_ context.Context, e event.Event) error {
	if s.err != nil {
		return s.err
	}
	s.events = append(s.events, e)
	return nil
}

func (s *spyPublisher) hasEvent(t event.EventType) bool {
	for _, e := range s.events {
		if e.Type == t {
			return true
		}
	}
	return false
}

func (s *spyPublisher) eventCount() int {
	return len(s.events)
}

// spyCache records cache invalidation calls.
type spyCache struct {
	prefixes []string
}

func newSpyCache() *spyCache {
	return &spyCache{}
}

func (s *spyCache) DeletePrefix(prefix string) {
	s.prefixes = append(s.prefixes, prefix)
}

func (s *spyCache) calledWith(prefix string) bool {
	for _, p := range s.prefixes {
		if p == prefix {
			return true
		}
	}
	return false
}

func (s *spyCache) callCount() int {
	return len(s.prefixes)
}

// ptr is a generic helper for creating pointers to literals in tests.
func ptr[T any](v T) *T {
	return &v
}

// assertAppError verifies an error is an AppError with a specific code.
// Usage: assertAppError(t, err, errors.ErrCodeNotFound)
func assertAppErrorCode(t interface{ Helper(); Errorf(string, ...interface{}) }, err error, expectedCode string) {
	t.Helper()
	if err == nil {
		t.Errorf("expected error with code %s, got nil", expectedCode)
		return
	}
	got := err.Error()
	if !contains(got, expectedCode) {
		t.Errorf("expected error code %s in %q", expectedCode, got)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// Compile-time check: spyPublisher implements EventPublisher
var _ event.EventPublisher = (*spyPublisher)(nil)

// Compile-time check: spyCache implements CacheInvalidator
var _ CacheInvalidator = (*spyCache)(nil)

// Suppress unused import warning
var _ = fmt.Sprintf
```

**Step 2: Verify it compiles**

Run: `cd handloom-admin && go test -run NONE ./internal/service/...`
Expected: PASS (compiles but runs no tests)

**Step 3: Commit**

```bash
git add internal/service/test_helpers_test.go
git commit -m "test: add spy types for event/cache test assertions"
```

---

### Task 2: AuthService edge-case tests

**Files:**
- Modify: `internal/service/auth_service_test.go`

**Step 1: Add all new auth test cases**

Append the following tests after the existing `TestAuthService_ResetPassword` function:

```go
func TestAuthService_Login_ErrorCodes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.NewNoop()

	svc := NewAuthService(userRepo, tokenStore, log, "test-secret", 15*time.Minute, 7*24*time.Hour, "test-issuer")
	ctx := context.Background()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	t.Run("inactive user returns USER_INACTIVE error code", func(t *testing.T) {
		userRepo.EXPECT().GetByEmail(ctx, "inactive@example.com").Return(&domain.User{
			ID:           "user_123",
			Email:        "inactive@example.com",
			PasswordHash: string(hashedPassword),
			Status:       domain.UserStatusInactive,
		}, nil)

		_, err := svc.Login(ctx, domain.LoginRequest{Email: "inactive@example.com", Password: "password123"})

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeUserInactive, appErr.Code)
	})

	t.Run("pending user returns USER_INACTIVE error code", func(t *testing.T) {
		userRepo.EXPECT().GetByEmail(ctx, "pending@example.com").Return(&domain.User{
			ID:           "user_456",
			Email:        "pending@example.com",
			PasswordHash: string(hashedPassword),
			Status:       domain.UserStatusPending,
		}, nil)

		_, err := svc.Login(ctx, domain.LoginRequest{Email: "pending@example.com", Password: "password123"})

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeUserInactive, appErr.Code)
	})

	t.Run("update last login failure is non-fatal", func(t *testing.T) {
		userRepo.EXPECT().GetByEmail(ctx, "test@example.com").Return(&domain.User{
			ID: "user_123", Email: "test@example.com",
			PasswordHash: string(hashedPassword), Status: domain.UserStatusActive,
			Role: domain.UserRoleAdmin, Permissions: []string{"*"},
		}, nil)
		tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(nil)
		tokenStore.EXPECT().StoreRefreshToken(ctx, "user_123", gomock.Any(), gomock.Any()).Return(nil)
		userRepo.EXPECT().UpdateLastLogin(ctx, "user_123").Return(errors.Internal("audit db down"))

		result, err := svc.Login(ctx, domain.LoginRequest{Email: "test@example.com", Password: "password123"})

		require.NoError(t, err) // non-fatal
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Tokens.AccessToken)
	})

	t.Run("revoke previous tokens failure is non-fatal", func(t *testing.T) {
		userRepo.EXPECT().GetByEmail(ctx, "test@example.com").Return(&domain.User{
			ID: "user_123", Email: "test@example.com",
			PasswordHash: string(hashedPassword), Status: domain.UserStatusActive,
			Role: domain.UserRoleAdmin, Permissions: []string{"*"},
		}, nil)
		tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(errors.Internal("redis down"))
		tokenStore.EXPECT().StoreRefreshToken(ctx, "user_123", gomock.Any(), gomock.Any()).Return(nil)
		userRepo.EXPECT().UpdateLastLogin(ctx, "user_123").Return(nil)

		result, err := svc.Login(ctx, domain.LoginRequest{Email: "test@example.com", Password: "password123"})

		require.NoError(t, err) // revoke failure is non-fatal
		assert.NotNil(t, result)
	})
}

func TestAuthService_RefreshToken_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.NewNoop()

	svc := NewAuthService(userRepo, tokenStore, log, "test-secret", 15*time.Minute, 7*24*time.Hour, "test-issuer")
	ctx := context.Background()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &domain.User{
		ID: "user_123", Email: "test@example.com",
		PasswordHash: string(hashedPassword), Status: domain.UserStatusActive,
		Role: domain.UserRoleAdmin, Permissions: []string{"*"},
	}
	tokens, _ := svc.generateTokenPair(user)

	t.Run("inactive user returns USER_INACTIVE error code", func(t *testing.T) {
		inactiveUser := &domain.User{
			ID: "user_123", Email: "test@example.com",
			Status: domain.UserStatusInactive,
		}
		tokenStore.EXPECT().ValidateRefreshToken(ctx, "user_123", tokens.RefreshToken).Return(true, nil)
		userRepo.EXPECT().GetByID(ctx, "user_123").Return(inactiveUser, nil)

		_, err := svc.RefreshToken(ctx, tokens.RefreshToken)

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeUserInactive, appErr.Code)
	})

	t.Run("old token revoke failure is non-fatal", func(t *testing.T) {
		tokenStore.EXPECT().ValidateRefreshToken(ctx, "user_123", tokens.RefreshToken).Return(true, nil)
		userRepo.EXPECT().GetByID(ctx, "user_123").Return(user, nil)
		tokenStore.EXPECT().RevokeRefreshToken(ctx, "user_123", tokens.RefreshToken).Return(errors.Internal("redis error"))
		tokenStore.EXPECT().StoreRefreshToken(ctx, "user_123", gomock.Any(), gomock.Any()).Return(nil)

		newTokens, err := svc.RefreshToken(ctx, tokens.RefreshToken)

		require.NoError(t, err) // revoke failure is non-fatal
		assert.NotEmpty(t, newTokens.AccessToken)
	})
}

func TestAuthService_ChangePassword_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.NewNoop()

	svc := NewAuthService(userRepo, tokenStore, log, "test-secret", 15*time.Minute, 7*24*time.Hour, "test-issuer")
	ctx := context.Background()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("oldpassword"), bcrypt.DefaultCost)

	t.Run("wrong password returns BAD_REQUEST error code", func(t *testing.T) {
		user := &domain.User{
			ID: "user_123", PasswordHash: string(hashedPassword),
		}
		userRepo.EXPECT().GetByID(ctx, "user_123").Return(user, nil)

		err := svc.ChangePassword(ctx, "user_123", domain.ChangePasswordRequest{
			CurrentPassword: "wrongpassword",
			NewPassword:     "newpassword123",
		})

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeBadRequest, appErr.Code)
	})

	t.Run("user not found returns error", func(t *testing.T) {
		userRepo.EXPECT().GetByID(ctx, "user_999").Return(nil, errors.NotFound("User"))

		err := svc.ChangePassword(ctx, "user_999", domain.ChangePasswordRequest{
			CurrentPassword: "old",
			NewPassword:     "new",
		})

		require.Error(t, err)
	})

	t.Run("token revocation failure is non-fatal", func(t *testing.T) {
		user := &domain.User{
			ID: "user_123", PasswordHash: string(hashedPassword),
		}
		userRepo.EXPECT().GetByID(ctx, "user_123").Return(user, nil)
		userRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
		tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(errors.Internal("redis down"))

		err := svc.ChangePassword(ctx, "user_123", domain.ChangePasswordRequest{
			CurrentPassword: "oldpassword",
			NewPassword:     "newpassword123",
		})

		require.NoError(t, err) // token revoke failure is non-fatal
	})

	t.Run("revokes all tokens on success", func(t *testing.T) {
		user := &domain.User{
			ID: "user_123", PasswordHash: string(hashedPassword),
		}
		userRepo.EXPECT().GetByID(ctx, "user_123").Return(user, nil)
		userRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
		tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(nil)

		err := svc.ChangePassword(ctx, "user_123", domain.ChangePasswordRequest{
			CurrentPassword: "oldpassword",
			NewPassword:     "newpassword123",
		})

		require.NoError(t, err)
		// If we got here without gomock error, RevokeAllUserTokens was called
	})
}

func TestAuthService_RequestPasswordReset_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.NewNoop()

	svc := NewAuthService(userRepo, tokenStore, log, "test-secret", 15*time.Minute, 7*24*time.Hour, "test-issuer")
	ctx := context.Background()

	t.Run("token store failure returns error", func(t *testing.T) {
		userRepo.EXPECT().GetByEmail(ctx, "test@example.com").Return(&domain.User{
			ID:    "user_123",
			Email: "test@example.com",
		}, nil)
		tokenStore.EXPECT().StorePasswordResetToken(ctx, "user_123", gomock.Any(), time.Hour).Return(errors.Internal("store failed"))

		err := svc.RequestPasswordReset(ctx, "test@example.com")

		require.Error(t, err)
	})
}

func TestAuthService_ResetPassword_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.NewNoop()

	svc := NewAuthService(userRepo, tokenStore, log, "test-secret", 15*time.Minute, 7*24*time.Hour, "test-issuer")
	ctx := context.Background()

	t.Run("revocation failures are non-fatal", func(t *testing.T) {
		tokenStore.EXPECT().ValidatePasswordResetToken(ctx, "valid-token").Return("user_123", nil)
		userRepo.EXPECT().GetByID(ctx, "user_123").Return(&domain.User{ID: "user_123"}, nil)
		userRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
		tokenStore.EXPECT().RevokePasswordResetToken(ctx, "valid-token").Return(errors.Internal("revoke failed"))
		tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(errors.Internal("revoke failed"))

		err := svc.ResetPassword(ctx, domain.ResetPasswordRequest{Token: "valid-token", NewPassword: "newpass"})

		require.NoError(t, err) // both revocations are non-fatal
	})
}

func TestAuthService_ValidateToken_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.NewNoop()

	svc := NewAuthService(userRepo, tokenStore, log, "test-secret", 15*time.Minute, 7*24*time.Hour, "test-issuer")
	ctx := context.Background()

	t.Run("permissions defaults to empty slice when missing", func(t *testing.T) {
		// Generate a token without permissions claim
		user := &domain.User{
			ID:           "user_123",
			Email:        "test@example.com",
			PasswordHash: "hash",
			Status:       domain.UserStatusActive,
			Role:         domain.UserRoleAdmin,
			Permissions:  nil, // no permissions
		}

		tokens, err := svc.generateTokenPair(user)
		require.NoError(t, err)

		claims, err := svc.ValidateToken(ctx, tokens.AccessToken)
		require.NoError(t, err)

		assert.NotNil(t, claims.Permissions) // should be empty slice, not nil
		assert.Empty(t, claims.Permissions)
	})

	t.Run("expired token returns INVALID_TOKEN error code", func(t *testing.T) {
		// Create a service with 0 duration to generate already-expired tokens
		expiredSvc := NewAuthService(userRepo, tokenStore, log, "test-secret", -1*time.Second, 7*24*time.Hour, "test-issuer")

		user := &domain.User{
			ID: "user_123", Email: "test@example.com",
			PasswordHash: "hash", Status: domain.UserStatusActive,
			Role: domain.UserRoleAdmin, Permissions: []string{"*"},
		}

		tokens, err := expiredSvc.generateTokenPair(user)
		require.NoError(t, err)

		_, err = svc.ValidateToken(ctx, tokens.AccessToken)

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeInvalidToken, appErr.Code)
	})
}
```

**Step 2: Verify all new tests pass**

Run: `cd handloom-admin && go test -v -run "TestAuthService_(Login_ErrorCodes|RefreshToken_EdgeCases|ChangePassword_EdgeCases|RequestPasswordReset_EdgeCases|ResetPassword_EdgeCases|ValidateToken_EdgeCases)" ./internal/service/...`
Expected: All PASS

**Step 3: Run full auth test suite to verify no regressions**

Run: `cd handloom-admin && go test -v -run "TestAuthService" ./internal/service/...`
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/service/auth_service_test.go
git commit -m "test(auth): add edge-case tests for error codes, non-fatal failures, token security"
```

---

### Task 3: CategoryService edge-case tests

**Files:**
- Modify: `internal/service/category_service_test.go`

**Step 1: Add all new category test cases**

Append after the existing `TestGenerateSlug` function:

```go
func TestCategoryService_Create_EdgeCases(t *testing.T) {
	t.Run("image finalization failure prevents repo create", func(t *testing.T) {
		svc, mockCatRepo, _, mockFinalizer, ctx := setupCategoryTest(t)

		req := domain.CreateCategoryRequest{
			Name:     "Test",
			ImageURL: "tmp/category/bad.jpg",
		}

		mockFinalizer.EXPECT().
			FinalizeIfTemp(ctx, "tmp/category/bad.jpg").
			Return("", errors.Internal("S3 error"))

		// mockCatRepo.Create should NOT be called
		// (gomock will fail if unexpected call happens)

		cat, err := svc.Create(ctx, req, "admin_1")

		assert.Nil(t, cat)
		require.Error(t, err)
	})
}

func TestCategoryService_Delete_ErrorCodes(t *testing.T) {
	t.Run("products exist returns CATEGORY_HAS_PRODUCTS error code", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID:           "cat_123",
			ProductCount: 5,
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		err := svc.Delete(ctx, "cat_123")

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeCategoryHasProducts, appErr.Code)
	})
}

func TestCategoryService_AddAttribute_EdgeCases(t *testing.T) {
	t.Run("repo update failure returns error", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID:            "cat_123",
			OwnAttributes: []domain.CategoryAttribute{},
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		mockCatRepo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(errors.Internal("db error"))

		cat, err := svc.AddAttribute(ctx, "cat_123", domain.CategoryAttribute{
			Name: "color", Label: "Color",
		}, "admin_1")

		assert.Nil(t, cat)
		require.Error(t, err)
	})
}

func TestCategoryService_UpdateAttribute_EdgeCases(t *testing.T) {
	t.Run("preserves other attributes unchanged", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "color", Label: "Color", Type: "SELECT", Required: true},
				{Name: "size", Label: "Size", Type: "TEXT", Required: false},
				{Name: "pattern", Label: "Pattern", Type: "TEXT", Required: true},
			},
		}

		updatedAttr := domain.CategoryAttribute{
			Name: "size", Label: "Size Updated", Type: "SELECT", Required: true,
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		mockCatRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, cat *domain.Category) error {
				assert.Len(t, cat.OwnAttributes, 3)
				// First and third should be unchanged
				assert.Equal(t, "color", cat.OwnAttributes[0].Name)
				assert.Equal(t, "Color", cat.OwnAttributes[0].Label)
				assert.Equal(t, domain.AttributeType("SELECT"), cat.OwnAttributes[0].Type)
				// Second should be updated
				assert.Equal(t, "Size Updated", cat.OwnAttributes[1].Label)
				assert.Equal(t, domain.AttributeType("SELECT"), cat.OwnAttributes[1].Type)
				assert.True(t, cat.OwnAttributes[1].Required)
				// Third unchanged
				assert.Equal(t, "pattern", cat.OwnAttributes[2].Name)
				assert.Equal(t, "Pattern", cat.OwnAttributes[2].Label)
				return nil
			})

		cat, err := svc.UpdateAttribute(ctx, "cat_123", "size", updatedAttr, "admin_1")

		require.NoError(t, err)
		assert.NotNil(t, cat)
	})
}

func TestCategoryService_DeleteAttribute_EdgeCases(t *testing.T) {
	t.Run("first attribute deletion works", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "first", Label: "First"},
				{Name: "second", Label: "Second"},
				{Name: "third", Label: "Third"},
			},
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		mockCatRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, cat *domain.Category) error {
				assert.Len(t, cat.OwnAttributes, 2)
				assert.Equal(t, "second", cat.OwnAttributes[0].Name)
				assert.Equal(t, "third", cat.OwnAttributes[1].Name)
				return nil
			})

		err := svc.DeleteAttribute(ctx, "cat_123", "first", "admin_1")
		require.NoError(t, err)
	})

	t.Run("preserves remaining attributes", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "a", Label: "A", Type: "TEXT"},
				{Name: "b", Label: "B", Type: "SELECT"},
				{Name: "c", Label: "C", Type: "NUMBER"},
			},
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		mockCatRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, cat *domain.Category) error {
				assert.Len(t, cat.OwnAttributes, 2)
				// Verify full integrity of remaining attributes
				assert.Equal(t, "a", cat.OwnAttributes[0].Name)
				assert.Equal(t, domain.AttributeType("TEXT"), cat.OwnAttributes[0].Type)
				assert.Equal(t, "c", cat.OwnAttributes[1].Name)
				assert.Equal(t, domain.AttributeType("NUMBER"), cat.OwnAttributes[1].Type)
				return nil
			})

		err := svc.DeleteAttribute(ctx, "cat_123", "b", "admin_1")
		require.NoError(t, err)
	})
}

func TestGenerateSlug_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"consecutive dashes from special chars", "A--B--C", "a-b-c"},
		{"special chars creating dashes", "Hello!!!World", "helloworld"},
		{"mixed special and spaces", "Silk & Cotton Blend", "silk-cotton-blend"},
		{"unicode stripped", "Café Mocha", "caf-mocha"},
		{"only dashes", "---", ""},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlug(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
```

**Step 2: Verify all new tests pass**

Run: `cd handloom-admin && go test -v -run "TestCategoryService_(Create_EdgeCases|Delete_ErrorCodes|AddAttribute_EdgeCases|UpdateAttribute_EdgeCases|DeleteAttribute_EdgeCases)|TestGenerateSlug_EdgeCases" ./internal/service/...`
Expected: All PASS

**Step 3: Run full category test suite to verify no regressions**

Run: `cd handloom-admin && go test -v -run "TestCategoryService|TestGenerateSlug" ./internal/service/...`
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/service/category_service_test.go
git commit -m "test(category): add edge-case tests for error codes, attribute integrity, slug edge cases"
```

---

### Task 4: InventoryService edge-case tests

**Files:**
- Modify: `internal/service/inventory_service_test.go`

**Step 1: Add all new inventory test cases**

Append after the existing `TestInventoryService_GetLowStockProducts` function:

```go
func TestInventoryService_AddStock_EventPublishing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	log := logger.NewNoop()
	ctx := context.Background()

	t.Run("publishes RESTOCKED event", func(t *testing.T) {
		spy := newSpyPublisher()
		cache := newSpyCache()
		svc := NewInventoryService(mockInventoryRepo, cache, spy, log)

		transaction := &domain.InventoryTransaction{
			ID: "txn_1", ProductID: "prod_123",
			PreviousQty: 10, NewQty: 60,
		}

		mockInventoryRepo.EXPECT().
			AddStock(ctx, "prod_123", 50, "restock", "user_1").
			Return(transaction, nil)

		_, err := svc.AddStock(ctx, "prod_123", domain.AddStockRequest{
			Quantity: 50, Reason: "restock",
		}, "user_1")

		require.NoError(t, err)
		assert.True(t, spy.hasEvent(event.InventoryRestocked))
		assert.Equal(t, 1, spy.eventCount())
	})

	t.Run("event publish failure is non-fatal", func(t *testing.T) {
		spy := newFailingPublisher(errors.Internal("SNS down"))
		svc := NewInventoryService(mockInventoryRepo, noopCache{}, spy, log)

		transaction := &domain.InventoryTransaction{
			ID: "txn_1", PreviousQty: 10, NewQty: 60,
		}

		mockInventoryRepo.EXPECT().
			AddStock(ctx, "prod_123", 50, "restock", "user_1").
			Return(transaction, nil)

		result, err := svc.AddStock(ctx, "prod_123", domain.AddStockRequest{
			Quantity: 50, Reason: "restock",
		}, "user_1")

		require.NoError(t, err) // publish failure is non-fatal
		assert.NotNil(t, result)
	})
}

func TestInventoryService_RemoveStock_EventPublishing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	log := logger.NewNoop()
	ctx := context.Background()

	t.Run("publishes OUT_OF_STOCK event when available qty is zero", func(t *testing.T) {
		spy := newSpyPublisher()
		svc := NewInventoryService(mockInventoryRepo, noopCache{}, spy, log)

		transaction := &domain.InventoryTransaction{
			ID: "txn_1", PreviousQty: 5, NewQty: 0,
		}

		mockInventoryRepo.EXPECT().
			RemoveStock(ctx, "prod_123", 5, "sold out", "user_1").
			Return(transaction, nil)

		mockInventoryRepo.EXPECT().
			GetByProductID(ctx, "prod_123").
			Return(&domain.Inventory{
				ProductID: "prod_123", AvailableQty: 0, LowStockThreshold: 10,
			}, nil)

		_, err := svc.RemoveStock(ctx, "prod_123", domain.RemoveStockRequest{
			Quantity: 5, Reason: "sold out",
		}, "user_1")

		require.NoError(t, err)
		assert.True(t, spy.hasEvent(event.InventoryOutOfStock))
		assert.False(t, spy.hasEvent(event.InventoryLowStock)) // mutually exclusive
	})

	t.Run("publishes LOW_STOCK event when below threshold but not zero", func(t *testing.T) {
		spy := newSpyPublisher()
		svc := NewInventoryService(mockInventoryRepo, noopCache{}, spy, log)

		transaction := &domain.InventoryTransaction{
			ID: "txn_1", PreviousQty: 15, NewQty: 8,
		}

		mockInventoryRepo.EXPECT().
			RemoveStock(ctx, "prod_123", 7, "sale", "user_1").
			Return(transaction, nil)

		mockInventoryRepo.EXPECT().
			GetByProductID(ctx, "prod_123").
			Return(&domain.Inventory{
				ProductID: "prod_123", AvailableQty: 8, LowStockThreshold: 10,
			}, nil)

		_, err := svc.RemoveStock(ctx, "prod_123", domain.RemoveStockRequest{
			Quantity: 7, Reason: "sale",
		}, "user_1")

		require.NoError(t, err)
		assert.True(t, spy.hasEvent(event.InventoryLowStock))
		assert.False(t, spy.hasEvent(event.InventoryOutOfStock))
	})

	t.Run("no stock event when above threshold", func(t *testing.T) {
		spy := newSpyPublisher()
		svc := NewInventoryService(mockInventoryRepo, noopCache{}, spy, log)

		transaction := &domain.InventoryTransaction{
			ID: "txn_1", PreviousQty: 100, NewQty: 80,
		}

		mockInventoryRepo.EXPECT().
			RemoveStock(ctx, "prod_123", 20, "sale", "user_1").
			Return(transaction, nil)

		mockInventoryRepo.EXPECT().
			GetByProductID(ctx, "prod_123").
			Return(&domain.Inventory{
				ProductID: "prod_123", AvailableQty: 80, LowStockThreshold: 10,
			}, nil)

		_, err := svc.RemoveStock(ctx, "prod_123", domain.RemoveStockRequest{
			Quantity: 20, Reason: "sale",
		}, "user_1")

		require.NoError(t, err)
		assert.Equal(t, 0, spy.eventCount()) // no events
	})

	t.Run("event publish failure is non-fatal", func(t *testing.T) {
		spy := newFailingPublisher(errors.Internal("SNS down"))
		svc := NewInventoryService(mockInventoryRepo, noopCache{}, spy, log)

		transaction := &domain.InventoryTransaction{
			ID: "txn_1", PreviousQty: 5, NewQty: 0,
		}

		mockInventoryRepo.EXPECT().
			RemoveStock(ctx, "prod_123", 5, "sold", "user_1").
			Return(transaction, nil)

		mockInventoryRepo.EXPECT().
			GetByProductID(ctx, "prod_123").
			Return(&domain.Inventory{
				ProductID: "prod_123", AvailableQty: 0, LowStockThreshold: 10,
			}, nil)

		result, err := svc.RemoveStock(ctx, "prod_123", domain.RemoveStockRequest{
			Quantity: 5, Reason: "sold",
		}, "user_1")

		require.NoError(t, err) // non-fatal
		assert.NotNil(t, result)
	})
}

func TestInventoryService_CacheInvalidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	log := logger.NewNoop()
	publisher := event.NewNoopPublisher()
	ctx := context.Background()

	t.Run("AddStock invalidates product cache", func(t *testing.T) {
		cache := newSpyCache()
		svc := NewInventoryService(mockInventoryRepo, cache, publisher, log)

		mockInventoryRepo.EXPECT().
			AddStock(ctx, "prod_123", 10, "test", "user_1").
			Return(&domain.InventoryTransaction{PreviousQty: 0, NewQty: 10}, nil)

		_, err := svc.AddStock(ctx, "prod_123", domain.AddStockRequest{
			Quantity: 10, Reason: "test",
		}, "user_1")

		require.NoError(t, err)
		assert.True(t, cache.calledWith("prod:"))
	})

	t.Run("RemoveStock invalidates product cache", func(t *testing.T) {
		cache := newSpyCache()
		svc := NewInventoryService(mockInventoryRepo, cache, publisher, log)

		mockInventoryRepo.EXPECT().
			RemoveStock(ctx, "prod_123", 5, "test", "user_1").
			Return(&domain.InventoryTransaction{PreviousQty: 10, NewQty: 5}, nil)

		mockInventoryRepo.EXPECT().
			GetByProductID(ctx, "prod_123").
			Return(&domain.Inventory{AvailableQty: 50, LowStockThreshold: 5}, nil)

		_, err := svc.RemoveStock(ctx, "prod_123", domain.RemoveStockRequest{
			Quantity: 5, Reason: "test",
		}, "user_1")

		require.NoError(t, err)
		assert.True(t, cache.calledWith("prod:"))
	})

	t.Run("AdjustStock invalidates product cache", func(t *testing.T) {
		cache := newSpyCache()
		svc := NewInventoryService(mockInventoryRepo, cache, publisher, log)

		mockInventoryRepo.EXPECT().
			AdjustStock(ctx, "prod_123", 50, "audit", "user_1").
			Return(&domain.InventoryTransaction{PreviousQty: 10, NewQty: 50}, nil)

		_, err := svc.AdjustStock(ctx, "prod_123", domain.AdjustStockRequest{
			NewQuantity: 50, Reason: "audit",
		}, "user_1")

		require.NoError(t, err)
		assert.True(t, cache.calledWith("prod:"))
	})

	t.Run("nil cache does not panic", func(t *testing.T) {
		svc := NewInventoryService(mockInventoryRepo, nil, publisher, log)

		mockInventoryRepo.EXPECT().
			AddStock(ctx, "prod_123", 10, "test", "user_1").
			Return(&domain.InventoryTransaction{PreviousQty: 0, NewQty: 10}, nil)

		_, err := svc.AddStock(ctx, "prod_123", domain.AddStockRequest{
			Quantity: 10, Reason: "test",
		}, "user_1")

		require.NoError(t, err) // should not panic
	})
}

func TestInventoryService_ResultFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	log := logger.NewNoop()
	svc := NewInventoryService(mockInventoryRepo, noopCache{}, event.NewNoopPublisher(), log)
	ctx := context.Background()

	t.Run("AddStock result fields match transaction", func(t *testing.T) {
		mockInventoryRepo.EXPECT().
			AddStock(ctx, "prod_123", 25, "shipment", "user_1").
			Return(&domain.InventoryTransaction{
				ID: "txn_abc", PreviousQty: 75, NewQty: 100,
			}, nil)

		result, err := svc.AddStock(ctx, "prod_123", domain.AddStockRequest{
			Quantity: 25, Reason: "shipment",
		}, "user_1")

		require.NoError(t, err)
		assert.Equal(t, "prod_123", result.ProductID)
		assert.Equal(t, 75, result.PreviousQuantity)
		assert.Equal(t, 25, result.ChangeQuantity)
		assert.Equal(t, 100, result.NewQuantity)
		assert.Equal(t, 100, result.AvailableQty)
		assert.Equal(t, "txn_abc", result.TransactionID)
	})

	t.Run("RemoveStock result has negative change quantity", func(t *testing.T) {
		mockInventoryRepo.EXPECT().
			RemoveStock(ctx, "prod_123", 30, "damaged", "user_1").
			Return(&domain.InventoryTransaction{
				ID: "txn_def", PreviousQty: 100, NewQty: 70,
			}, nil)

		mockInventoryRepo.EXPECT().
			GetByProductID(ctx, "prod_123").
			Return(&domain.Inventory{AvailableQty: 70, LowStockThreshold: 5}, nil)

		result, err := svc.RemoveStock(ctx, "prod_123", domain.RemoveStockRequest{
			Quantity: 30, Reason: "damaged",
		}, "user_1")

		require.NoError(t, err)
		assert.Equal(t, -30, result.ChangeQuantity) // negative for removal
		assert.Equal(t, 100, result.PreviousQuantity)
		assert.Equal(t, 70, result.NewQuantity)
	})
}
```

**Step 2: Verify all new tests pass**

Run: `cd handloom-admin && go test -v -run "TestInventoryService_(AddStock_EventPublishing|RemoveStock_EventPublishing|CacheInvalidation|ResultFields)" ./internal/service/...`
Expected: All PASS

**Step 3: Run full inventory test suite**

Run: `cd handloom-admin && go test -v -run "TestInventoryService" ./internal/service/...`
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/service/inventory_service_test.go
git commit -m "test(inventory): add event publishing, cache invalidation, and result field tests"
```

---

### Task 5: UserService edge-case tests

**Files:**
- Modify: `internal/service/user_service_test.go`

**Step 1: Add all new user test cases**

Append after the existing `TestUserService_UpdateStatus` function:

```go
func TestUserService_Create_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockTokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.NewNoop()
	svc := NewUserService(mockUserRepo, mockTokenStore, log)
	ctx := context.Background()

	t.Run("new user starts with PENDING status", func(t *testing.T) {
		mockUserRepo.EXPECT().GetByEmail(ctx, "new@example.com").Return(nil, errors.NotFound("User"))
		mockUserRepo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, user *domain.User) error {
			assert.Equal(t, domain.UserStatusPending, user.Status)
			return nil
		})

		user, err := svc.Create(ctx, domain.CreateUserRequest{
			Email:    "new@example.com",
			Password: "password123",
			Role:     domain.UserRoleOperator,
		}, "admin_1")

		require.NoError(t, err)
		assert.Equal(t, domain.UserStatusPending, user.Status)
	})

	t.Run("password hash not in returned user", func(t *testing.T) {
		mockUserRepo.EXPECT().GetByEmail(ctx, "safe@example.com").Return(nil, errors.NotFound("User"))
		mockUserRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		user, err := svc.Create(ctx, domain.CreateUserRequest{
			Email:    "safe@example.com",
			Password: "password123",
		}, "admin_1")

		require.NoError(t, err)
		assert.Empty(t, user.PasswordHash) // security: never expose hash
	})

	t.Run("duplicate email returns ALREADY_EXISTS error code", func(t *testing.T) {
		mockUserRepo.EXPECT().GetByEmail(ctx, "dup@example.com").Return(&domain.User{
			ID: "user_existing", Email: "dup@example.com",
		}, nil)

		_, err := svc.Create(ctx, domain.CreateUserRequest{
			Email:    "dup@example.com",
			Password: "password123",
		}, "admin_1")

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeAlreadyExists, appErr.Code)
	})

	t.Run("repo create failure returns error", func(t *testing.T) {
		mockUserRepo.EXPECT().GetByEmail(ctx, "fail@example.com").Return(nil, errors.NotFound("User"))
		mockUserRepo.EXPECT().Create(ctx, gomock.Any()).Return(errors.Internal("db error"))

		user, err := svc.Create(ctx, domain.CreateUserRequest{
			Email:    "fail@example.com",
			Password: "password123",
		}, "admin_1")

		assert.Nil(t, user)
		require.Error(t, err)
	})
}

func TestUserService_GetByID_Security(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockTokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.NewNoop()
	svc := NewUserService(mockUserRepo, mockTokenStore, log)
	ctx := context.Background()

	t.Run("password hash cleared even when repo returns one", func(t *testing.T) {
		mockUserRepo.EXPECT().GetByID(ctx, "user_123").Return(&domain.User{
			ID:           "user_123",
			Email:        "test@example.com",
			PasswordHash: "$2a$10$somebcrypthash", // repo returns hash
		}, nil)

		user, err := svc.GetByID(ctx, "user_123")

		require.NoError(t, err)
		assert.Equal(t, "user_123", user.ID)
		assert.Empty(t, user.PasswordHash) // must be cleared
	})
}

func TestUserService_Update_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockTokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.NewNoop()
	svc := NewUserService(mockUserRepo, mockTokenStore, log)
	ctx := context.Background()

	t.Run("empty password skips hashing", func(t *testing.T) {
		existing := &domain.User{
			ID:           "user_123",
			PasswordHash: "original_hash",
		}

		emptyPass := ""
		req := domain.UpdateUserRequest{Password: &emptyPass}

		mockUserRepo.EXPECT().GetByID(ctx, "user_123").Return(existing, nil)
		mockUserRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, user *domain.User) error {
			// Password hash should remain unchanged since empty password was provided
			assert.Equal(t, "original_hash", user.PasswordHash)
			return nil
		})

		_, err := svc.Update(ctx, "user_123", req, "admin_1")
		require.NoError(t, err)
	})

	t.Run("permission update replaces entire list", func(t *testing.T) {
		existing := &domain.User{
			ID:          "user_123",
			Permissions: []string{"read", "write", "delete"},
		}

		req := domain.UpdateUserRequest{
			Permissions: []string{"read"}, // replace with single permission
		}

		mockUserRepo.EXPECT().GetByID(ctx, "user_123").Return(existing, nil)
		mockUserRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, user *domain.User) error {
			assert.Equal(t, []string{"read"}, user.Permissions) // old permissions gone
			return nil
		})

		_, err := svc.Update(ctx, "user_123", req, "admin_1")
		require.NoError(t, err)
	})

	t.Run("sets updated_at timestamp", func(t *testing.T) {
		existing := &domain.User{ID: "user_123"}

		newName := "Updated"
		req := domain.UpdateUserRequest{FirstName: &newName}

		mockUserRepo.EXPECT().GetByID(ctx, "user_123").Return(existing, nil)
		mockUserRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, user *domain.User) error {
			assert.False(t, user.UpdatedAt.IsZero()) // timestamp should be set
			return nil
		})

		_, err := svc.Update(ctx, "user_123", req, "admin_1")
		require.NoError(t, err)
	})

	t.Run("returned user has cleared password", func(t *testing.T) {
		existing := &domain.User{
			ID:           "user_123",
			PasswordHash: "some_hash",
		}

		newName := "Test"
		req := domain.UpdateUserRequest{FirstName: &newName}

		mockUserRepo.EXPECT().GetByID(ctx, "user_123").Return(existing, nil)
		mockUserRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		user, err := svc.Update(ctx, "user_123", req, "admin_1")

		require.NoError(t, err)
		assert.Empty(t, user.PasswordHash)
	})
}

func TestUserService_List_Security(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockTokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.NewNoop()
	svc := NewUserService(mockUserRepo, mockTokenStore, log)
	ctx := context.Background()

	t.Run("all users have password cleared in batch", func(t *testing.T) {
		mockUserRepo.EXPECT().List(ctx, gomock.Any()).Return(&domain.ListUsersResponse{
			Users: []*domain.User{
				{ID: "u1", PasswordHash: "hash1"},
				{ID: "u2", PasswordHash: "hash2"},
				{ID: "u3", PasswordHash: "hash3"},
			},
			Pagination: domain.PaginationResponse{Limit: 20},
		}, nil)

		resp, err := svc.List(ctx, domain.ListUsersRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 20},
		})

		require.NoError(t, err)
		assert.Len(t, resp.Users, 3)
		for _, u := range resp.Users {
			assert.Empty(t, u.PasswordHash, "user %s should have password cleared", u.ID)
		}
	})
}

func TestUserService_UpdateStatus_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockTokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.NewNoop()
	svc := NewUserService(mockUserRepo, mockTokenStore, log)
	ctx := context.Background()

	t.Run("sets updated_at timestamp", func(t *testing.T) {
		mockUserRepo.EXPECT().GetByID(ctx, "user_123").Return(&domain.User{
			ID:     "user_123",
			Status: domain.UserStatusPending,
		}, nil)

		mockUserRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, user *domain.User) error {
			assert.False(t, user.UpdatedAt.IsZero())
			assert.Equal(t, "admin_1", user.UpdatedBy)
			return nil
		})

		err := svc.UpdateStatus(ctx, "user_123", domain.UserStatusActive, "admin_1")
		require.NoError(t, err)
	})

	t.Run("repo update failure returns error", func(t *testing.T) {
		mockUserRepo.EXPECT().GetByID(ctx, "user_123").Return(&domain.User{
			ID: "user_123", Status: domain.UserStatusPending,
		}, nil)

		mockUserRepo.EXPECT().Update(ctx, gomock.Any()).Return(errors.Internal("db error"))

		err := svc.UpdateStatus(ctx, "user_123", domain.UserStatusActive, "admin_1")
		require.Error(t, err)
	})
}
```

**Step 2: Verify new tests pass**

Run: `cd handloom-admin && go test -v -run "TestUserService_(Create_EdgeCases|GetByID_Security|Update_EdgeCases|List_Security|UpdateStatus_EdgeCases)" ./internal/service/...`
Expected: All PASS

**Step 3: Run full user test suite**

Run: `cd handloom-admin && go test -v -run "TestUserService" ./internal/service/...`
Expected: All PASS

**Step 4: Commit**

```bash
git add internal/service/user_service_test.go
git commit -m "test(user): add security invariant, error code, and data integrity edge cases"
```

---

### Task 6: ProductService edge-case tests

**Files:**
- Modify: `internal/service/product_service_test.go`

**Step 1: Add a setup helper with spy publisher**

Add this helper function near the top of the file, after the existing `setupProductTest`:

```go
func setupProductTestWithSpy(t *testing.T) (
	*ProductService,
	*mocks.MockProductRepository,
	*mocks.MockCategoryRepository,
	*mocks.MockInventoryRepository,
	*mocks.MockAssetFinalizer,
	*spyPublisher,
	context.Context,
) {
	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	mockProdRepo := mocks.NewMockProductRepository(ctrl)
	mockCatRepo := mocks.NewMockCategoryRepository(ctrl)
	mockInvRepo := mocks.NewMockInventoryRepository(ctrl)
	mockFinalizer := mocks.NewMockAssetFinalizer(ctrl)
	log := logger.NewNoop()

	spy := newSpyPublisher()
	svc := NewProductService(mockProdRepo, mockCatRepo, mockInvRepo, mockFinalizer, spy, log)
	return svc, mockProdRepo, mockCatRepo, mockInvRepo, mockFinalizer, spy, context.Background()
}
```

**Step 2: Add all new product test cases**

Append after the existing `TestValidateRequiredAttributes` function:

```go
func TestProductService_Create_EventPublishing(t *testing.T) {
	t.Run("publishes PRODUCT_CREATED event", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, spy, ctx := setupProductTestWithSpy(t)

		category := &domain.Category{ID: "cat_123"}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().Create(ctx, gomock.Any(), gomock.Any()).Return(nil)
		mockCatRepo.EXPECT().IncrementProductCount(ctx, "cat_123", 1).Return(nil)

		_, err := svc.Create(ctx, domain.CreateProductRequest{
			Name: "Test", SKU: "TST-001", CategoryID: "cat_123",
		}, "admin_1")

		require.NoError(t, err)
		assert.True(t, spy.hasEvent(event.ProductCreated))
	})

	t.Run("event failure is non-fatal", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		t.Cleanup(func() { ctrl.Finish() })

		mockProdRepo := mocks.NewMockProductRepository(ctrl)
		mockCatRepo := mocks.NewMockCategoryRepository(ctrl)
		mockInvRepo := mocks.NewMockInventoryRepository(ctrl)
		mockFinalizer := mocks.NewMockAssetFinalizer(ctrl)
		log := logger.NewNoop()

		failPub := newFailingPublisher(errors.Internal("SNS down"))
		svc := NewProductService(mockProdRepo, mockCatRepo, mockInvRepo, mockFinalizer, failPub, log)
		ctx := context.Background()

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(&domain.Category{ID: "cat_123"}, nil)
		mockProdRepo.EXPECT().Create(ctx, gomock.Any(), gomock.Any()).Return(nil)
		mockCatRepo.EXPECT().IncrementProductCount(ctx, "cat_123", 1).Return(nil)

		product, err := svc.Create(ctx, domain.CreateProductRequest{
			Name: "Test", SKU: "TST-001", CategoryID: "cat_123",
		}, "admin_1")

		require.NoError(t, err) // non-fatal
		assert.NotNil(t, product)
	})
}

func TestProductService_Update_EventPublishing(t *testing.T) {
	t.Run("publishes PRODUCT_UPDATED event", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, spy, ctx := setupProductTestWithSpy(t)

		existing := &domain.Product{ID: "prod_123", CategoryID: "cat_123", Material: "silk"}
		category := &domain.Category{ID: "cat_123"}
		newName := "Updated Name"

		mockProdRepo.EXPECT().GetByID(ctx, "prod_123").Return(existing, nil)
		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		_, err := svc.Update(ctx, "prod_123", domain.UpdateProductRequest{
			Name: &newName,
		}, "admin_1")

		require.NoError(t, err)
		assert.True(t, spy.hasEvent(event.ProductUpdated))
	})
}

func TestProductService_Delete_EventPublishing(t *testing.T) {
	t.Run("publishes PRODUCT_DELETED event", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, spy, ctx := setupProductTestWithSpy(t)

		mockProdRepo.EXPECT().GetByID(ctx, "prod_123").Return(&domain.Product{
			ID: "prod_123", CategoryID: "cat_123",
		}, nil)
		mockProdRepo.EXPECT().Delete(ctx, "prod_123").Return(nil)
		mockCatRepo.EXPECT().IncrementProductCount(ctx, "cat_123", -1).Return(nil)

		err := svc.Delete(ctx, "prod_123")

		require.NoError(t, err)
		assert.True(t, spy.hasEvent(event.ProductDeleted))
	})
}

func TestProductService_Create_ErrorCodes(t *testing.T) {
	t.Run("missing category returns NOT_FOUND error code", func(t *testing.T) {
		svc, _, mockCatRepo, _, _, ctx := setupProductTest(t)

		mockCatRepo.EXPECT().GetByID(ctx, "cat_999").Return(nil, errors.NotFound("Category"))

		_, err := svc.Create(ctx, domain.CreateProductRequest{
			Name: "Test", CategoryID: "cat_999",
		}, "admin_1")

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeNotFound, appErr.Code)
	})

	t.Run("missing required attr returns VALIDATION error code", func(t *testing.T) {
		svc, _, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "pattern", Label: "Pattern", Searchable: true, Required: true},
			},
		}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)

		_, err := svc.Create(ctx, domain.CreateProductRequest{
			Name: "Test", CategoryID: "cat_123",
		}, "admin_1")

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeValidation, appErr.Code)
	})
}

func TestProductService_Create_Atomicity(t *testing.T) {
	t.Run("inventory created alongside product", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{ID: "cat_123"}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().
			Create(ctx, gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, product *domain.Product, inv *domain.Inventory) error {
				// Both product and inventory must be created
				assert.NotNil(t, product)
				assert.NotNil(t, inv)
				assert.Equal(t, product.ID, inv.ProductID)
				assert.Equal(t, 25, inv.Quantity)
				assert.Equal(t, 25, inv.AvailableQty)
				assert.Equal(t, 5, inv.LowStockThreshold)
				assert.Equal(t, "admin_1", inv.CreatedBy)
				return nil
			})
		mockCatRepo.EXPECT().IncrementProductCount(ctx, "cat_123", 1).Return(nil)

		_, err := svc.Create(ctx, domain.CreateProductRequest{
			Name: "Test", SKU: "TST-001", CategoryID: "cat_123",
			InitialStock: 25, LowStockThreshold: 5,
		}, "admin_1")

		require.NoError(t, err)
	})

	t.Run("category count increment failure propagates", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{ID: "cat_123"}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().Create(ctx, gomock.Any(), gomock.Any()).Return(nil)
		mockCatRepo.EXPECT().IncrementProductCount(ctx, "cat_123", 1).Return(errors.Internal("db error"))

		_, err := svc.Create(ctx, domain.CreateProductRequest{
			Name: "Test", CategoryID: "cat_123",
		}, "admin_1")

		require.Error(t, err) // should propagate
	})
}

func TestProductService_Delete_Cascade(t *testing.T) {
	t.Run("decrements category count", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		mockProdRepo.EXPECT().GetByID(ctx, "prod_123").Return(&domain.Product{
			ID: "prod_123", CategoryID: "cat_123",
		}, nil)
		mockProdRepo.EXPECT().Delete(ctx, "prod_123").Return(nil)
		mockCatRepo.EXPECT().
			IncrementProductCount(ctx, "cat_123", -1).
			Return(nil) // if this wasn't called, gomock would fail

		err := svc.Delete(ctx, "prod_123")
		require.NoError(t, err)
	})
}

func TestProductService_Update_EdgeCases(t *testing.T) {
	t.Run("slug regenerated when name changes", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		existing := &domain.Product{
			ID: "prod_123", Name: "Old Name", Slug: "old-name", CategoryID: "cat_123",
		}
		category := &domain.Category{ID: "cat_123"}
		newName := "Brand New Name"

		mockProdRepo.EXPECT().GetByID(ctx, "prod_123").Return(existing, nil)
		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, p *domain.Product) error {
			assert.Equal(t, "brand-new-name", p.Slug)
			return nil
		})

		product, err := svc.Update(ctx, "prod_123", domain.UpdateProductRequest{Name: &newName}, "admin_1")

		require.NoError(t, err)
		assert.Equal(t, "brand-new-name", product.Slug)
	})

	t.Run("slug unchanged when name not provided", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		existing := &domain.Product{
			ID: "prod_123", Name: "Original", Slug: "original", CategoryID: "cat_123",
		}
		category := &domain.Category{ID: "cat_123"}

		mockProdRepo.EXPECT().GetByID(ctx, "prod_123").Return(existing, nil)
		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, p *domain.Product) error {
			assert.Equal(t, "original", p.Slug) // unchanged
			return nil
		})

		newDesc := "updated desc"
		_, err := svc.Update(ctx, "prod_123", domain.UpdateProductRequest{Description: &newDesc}, "admin_1")

		require.NoError(t, err)
	})
}

func TestProductService_GetAttributeFilterOptions_EdgeCases(t *testing.T) {
	t.Run("only searchable attributes included", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "color", Searchable: true},
				{Name: "internal_code", Searchable: false},
				{Name: "pattern", Searchable: true},
				{Name: "notes", Searchable: false},
			},
		}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().
			GetAttributeFilterOptions(ctx, "cat_123", gomock.InAnyOrder([]string{"color", "pattern"})).
			Return(map[string][]string{
				"color":   {"red"},
				"pattern": {"floral"},
			}, nil)

		result, err := svc.GetAttributeFilterOptions(ctx, "cat_123")

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Contains(t, result, "color")
		assert.Contains(t, result, "pattern")
		assert.NotContains(t, result, "internal_code")
		assert.NotContains(t, result, "notes")
	})

	t.Run("empty category returns empty map not nil", func(t *testing.T) {
		svc, _, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{
			ID:            "cat_123",
			OwnAttributes: nil, // no attributes at all
		}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)

		result, err := svc.GetAttributeFilterOptions(ctx, "cat_123")

		require.NoError(t, err)
		assert.NotNil(t, result) // should be empty map, not nil
		assert.Len(t, result, 0)
	})

	t.Run("no searchable attributes returns empty map", func(t *testing.T) {
		svc, _, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "notes", Searchable: false},
			},
		}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)

		result, err := svc.GetAttributeFilterOptions(ctx, "cat_123")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 0)
	})
}

func TestProductService_ReorderProducts_EdgeCases(t *testing.T) {
	t.Run("duplicate IDs rejected", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{ID: "cat_123"}
		products := []*domain.Product{
			{ID: "prod_a", CategoryID: "cat_123"},
			{ID: "prod_b", CategoryID: "cat_123"},
		}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().GetByCategoryAll(ctx, "cat_123").Return(products, nil)

		_, err := svc.ReorderProducts(ctx, "cat_123", []string{"prod_a", "prod_a"})

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeValidation, appErr.Code)
		assert.Contains(t, err.Error(), "Duplicate")
	})

	t.Run("partial reorder assigns sequential to unranked", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{ID: "cat_123"}
		products := []*domain.Product{
			{ID: "prod_a", CategoryID: "cat_123"},
			{ID: "prod_b", CategoryID: "cat_123"},
			{ID: "prod_c", CategoryID: "cat_123"},
		}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().GetByCategoryAll(ctx, "cat_123").Return(products, nil)
		mockProdRepo.EXPECT().
			BatchUpdateSortOrder(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, prods []*domain.Product) error {
				assert.Len(t, prods, 3) // all 3 updated even though only 1 was ranked
				// prod_c requested first
				for _, p := range prods {
					if p.ID == "prod_c" {
						assert.Equal(t, 1, p.SortOrder)
					}
				}
				return nil
			})

		count, err := svc.ReorderProducts(ctx, "cat_123", []string{"prod_c"})
		require.NoError(t, err)
		assert.Equal(t, 3, count) // all products updated
	})
}

func TestValidateRequiredAttributes_EdgeCases(t *testing.T) {
	t.Run("non-searchable required attribute is NOT enforced", func(t *testing.T) {
		categoryAttrs := []domain.CategoryAttribute{
			{Name: "internal_notes", Label: "Notes", Searchable: false, Required: true},
		}

		// No attributes provided, but the required attr is non-searchable
		err := validateRequiredAttributes(nil, categoryAttrs)
		assert.NoError(t, err) // should pass: only searchable+required is enforced
	})

	t.Run("interface slice with empty strings filtered", func(t *testing.T) {
		result := normalizeToStringSlice([]interface{}{"a", "", "b"})
		assert.Equal(t, []string{"a", "b"}, result) // empty strings filtered
	})
}
```

**Step 2: Verify new tests pass**

Run: `cd handloom-admin && go test -v -run "TestProductService_(Create_EventPublishing|Update_EventPublishing|Delete_EventPublishing|Create_ErrorCodes|Create_Atomicity|Delete_Cascade|Update_EdgeCases|GetAttributeFilterOptions_EdgeCases|ReorderProducts_EdgeCases)|TestValidateRequiredAttributes_EdgeCases" ./internal/service/...`
Expected: All PASS

**Step 3: Run full product + all service tests**

Run: `cd handloom-admin && go test -v -race ./internal/service/...`
Expected: All PASS (including existing tests)

**Step 4: Commit**

```bash
git add internal/service/product_service_test.go
git commit -m "test(product): add event publishing, error codes, atomicity, and reorder edge cases"
```

---

### Task 7: Final verification

**Step 1: Run full test suite with race detector**

Run: `cd handloom-admin && go test -v -race -cover ./internal/service/...`
Expected: All PASS, coverage should increase

**Step 2: Verify test count increased**

Run: `cd handloom-admin && go test -v ./internal/service/... 2>&1 | grep -c "=== RUN"`
Expected: ~65 more test runs than before (was ~80, should be ~145+)
