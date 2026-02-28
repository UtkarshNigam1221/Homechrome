# Service Test Expansion Design

**Date:** 2026-02-28
**Scope:** Expand edge-case test coverage for AuthService, CategoryService, InventoryService, UserService, ProductService
**Approach:** Full coverage (65 new test cases)

## Context

All 5 services already have comprehensive unit tests covering happy paths and basic error cases. This expansion adds tests that catch behavior changes in three categories:

1. **Business logic contracts** — Exact error codes, exact event types, exact return values
2. **Security invariants** — Password never leaked, tokens always revoked, inactive users blocked
3. **Data integrity** — Atomic operations, cache invalidation, cascade effects, boundary values

## Infrastructure Changes

### Mock EventPublisher

Add a gomock-based `MockEventPublisher` to replace `event.NewNoopPublisher()` in tests that need to verify event types/payloads. Check if `EventPublisher` interface exists in domain; if so, generate mock via mockgen. Otherwise, create a simple spy implementation.

### Mock CacheInvalidator

Replace `noopCache` struct in inventory tests with a gomock-based mock for `CacheInvalidator` interface to verify `DeletePrefix` calls.

### Error Code Assertions

Pattern for all error code tests:
```go
var appErr *errors.AppError
require.ErrorAs(t, err, &appErr)
assert.Equal(t, errors.ErrCodeNotFound, appErr.Code)
```

## AuthService (13 new tests)

### Error Codes
- `Login/inactive_user_returns_FORBIDDEN` — exact `ErrCodeForbidden`
- `RefreshToken/inactive_user_returns_USER_INACTIVE` — exact code after valid token but inactive user
- `ChangePassword/wrong_password_returns_BAD_REQUEST` — `ErrCodeBadRequest`
- `ChangePassword/user_not_found_returns_error` — `ErrCodeNotFound`

### Security
- `Login/revokes_previous_sessions_before_new_tokens` — assert `RevokeAllUserTokens` called before `StoreRefreshToken`
- `ChangePassword/revokes_all_tokens_on_success` — verify `RevokeAllUserTokens` called
- `ValidateToken/invalid_signing_method_rejected` — RSA-signed token rejected
- `ValidateToken/missing_permissions_defaults_to_empty_slice` — claims without permissions

### Non-Fatal Error Handling
- `Login/update_last_login_failure_is_non_fatal` — `UpdateLastLogin` fails, still returns tokens
- `RefreshToken/old_token_revoke_failure_is_non_fatal` — `RevokeRefreshToken` fails, still returns new tokens
- `ChangePassword/token_revocation_failure_is_non_fatal` — password changed even if token revoke fails
- `ResetPassword/revocation_failures_are_non_fatal` — both revocations can fail
- `RequestPasswordReset/token_store_failure_returns_error` — `StorePasswordResetToken` failure propagates

## CategoryService (8 new tests)

### Business Logic
- `Create/slug_collapses_consecutive_dashes` — "Multiple---Words" -> "multiple-words"
- `Create/image_finalization_failure_prevents_repo_create` — repo.Create NOT called
- `DeleteAttribute/first_attribute_deletion_works` — boundary: index 0
- `GetAttributes/nil_attributes_returns_zero_count` — TotalCount=0

### Error Codes
- `Delete/products_exist_returns_CATEGORY_HAS_PRODUCTS_code` — exact error code

### Data Integrity
- `AddAttribute/repo_update_failure_returns_error` — attribute added but repo fails
- `UpdateAttribute/preserves_other_attributes_unchanged` — unmodified attrs still present
- `DeleteAttribute/preserves_remaining_attributes` — no accidental data loss

## InventoryService (11 new tests)

### Event Publishing (MockEventPublisher)
- `AddStock/publishes_RESTOCKED_event` — verify `InventoryRestocked` type
- `RemoveStock/publishes_OUT_OF_STOCK_event_when_zero` — qty=0 triggers event
- `RemoveStock/publishes_LOW_STOCK_event_when_below_threshold` — qty > 0 but <= threshold
- `RemoveStock/out_of_stock_and_low_stock_mutually_exclusive` — qty=0 does NOT also trigger low stock

### Non-Fatal Publishing
- `AddStock/event_publish_failure_is_non_fatal` — publisher fails, still returns success
- `RemoveStock/event_publish_failure_is_non_fatal`

### Cache Invalidation (MockCacheInvalidator)
- `AddStock/invalidates_product_cache` — `DeletePrefix` called
- `RemoveStock/invalidates_product_cache`
- `AdjustStock/invalidates_product_cache`

### Other
- `GetByProductID/not_found_returns_specific_error` — exact error code
- `AddStock/result_fields_match_transaction` — verify PreviousQty, NewQty, ChangeQty mapping

## UserService (13 new tests)

### Security
- `Create/password_hash_not_in_returned_user` — explicit `PasswordHash == ""`
- `GetByID/password_hash_cleared_even_when_repo_returns_one` — mock with hash, verify empty
- `Update/returned_user_has_cleared_password` — verify after update
- `List/all_users_have_password_cleared` — multi-user list, all hashes empty

### Error Codes
- `Create/duplicate_email_returns_ALREADY_EXISTS` — exact code
- `Create/repo_create_failure_returns_error` — propagation
- `UpdateStatus/repo_failure_returns_error` — propagation

### Business Logic
- `Create/new_user_starts_with_PENDING_status` — explicit `UserStatusPending`
- `Update/empty_password_skips_hashing` — `Password: ptr("")` doesn't rehash
- `Update/permission_update_replaces_entire_list` — old permissions gone
- `Delete/token_revocation_after_delete_order` — delete first, then revoke

### Timestamps
- `Update/sets_updated_at_timestamp` — non-zero time.Time
- `UpdateStatus/sets_updated_at_timestamp`

## ProductService (20 new tests)

### Event Publishing
- `Create/publishes_PRODUCT_CREATED_event` — verify type
- `Update/publishes_PRODUCT_UPDATED_event` — verify type
- `Delete/publishes_PRODUCT_DELETED_event` — verify type
- `Create/event_failure_is_non_fatal` — publish fails, still returns product

### Error Codes
- `Create/missing_category_returns_NOT_FOUND_code`
- `Create/missing_required_attr_returns_VALIDATION_ERROR_code`

### Atomicity & Cascade
- `Create/inventory_created_alongside_product` — both passed to repo
- `Create/category_count_increment_failure_propagates`
- `Delete/category_count_decremented` — `IncrementProductCount(-1)` called

### Image Finalization
- `Create/multi_image_fails_on_first_error` — 3 images, 2nd fails
- `Update/image_finalization_on_update` — new images finalized

### Attribute Validation
- `Update/removing_required_attribute_fails`
- `Create/non_searchable_required_not_enforced` — required but not searchable skipped

### Reorder
- `ReorderProducts/duplicate_ids_rejected`
- `ReorderProducts/partial_reorder_assigns_sequential_to_unranked`
- `ReorderProducts/returns_correct_update_count`
- `ReorderProducts/category_not_found_returns_error`

### Other
- `Update/slug_regenerated_when_name_changes`
- `Update/slug_unchanged_when_name_nil`
- `GetAttributeFilterOptions/only_searchable_attributes_included`
- `GetAttributeFilterOptions/empty_category_returns_empty_map`

## Verification

After implementation:
```bash
make test-unit   # All existing + new tests pass
```

No new dependencies. All tests use existing gomock + testify patterns.
