# Guest Cart Feature Design

Allow unauthenticated (guest) users to build and manage a shopping cart. Authentication is only required at checkout.

## Context

The existing cart implementation requires `CustomerAuth` middleware on all cart routes, and the storefront frontend blocks cart access for unauthenticated users. However, the backend already has infrastructure for guest carts: the `Cart` domain entity has a `SessionID` field, the DynamoDB key pattern supports `CART#<session_id>`, and a `MergeGuestCart` service method exists.

## Decisions

- **Server-side guest cart** stored in DynamoDB with a `guest_session` HttpOnly cookie (30-day TTL). Chosen over client-side localStorage for cross-tab persistence and simpler frontend.
- **Merge strategy on login**: combine guest + existing authenticated cart, keeping the higher quantity per product (existing `MergeGuestCart` behavior).
- **Full cart access for guests**: view cart page, update quantities, remove items. Auth gate only at checkout.
- **Identical UX**: no login prompts, no banners. Guest and authenticated add-to-cart flow is the same.
- **Guest session cookie is unsigned**: accepted risk given low sensitivity of cart data (product IDs and quantities). No PII is exposed.
- **Rate limiting on guest cart creation deferred**: existing infrastructure does not rate-limit cart routes. Will be addressed as a follow-up if abuse is observed.

## Design

### 1. OptionalCartAuth Middleware

New file: `internal/middleware/optional_cart_auth.go`

Resolves cart identity without requiring authentication:

1. Extract and validate `store_token` cookie. If valid JWT, set `CustomerIDKey` in context (authenticated path). Do NOT generate or set a `guest_session` cookie.
2. Else, read `guest_session` cookie. If present, set `GuestSessionKey` in context.
3. Else, generate a UUID v4, set `GuestSessionKey` in context, and write a `guest_session` HttpOnly cookie in the response (30-day MaxAge, same secure/sameSite/domain settings as existing store cookies via `cookieSettings()`).

New context key and helper:

```go
GuestSessionKey // new context key in context_keys.go

// GetCartIdentityFromContext returns the cart owner identifier and whether the user is a guest.
// Authenticated: ("<customerID>", false)
// Guest: ("<sessionID>", true)
func GetCartIdentityFromContext(ctx context.Context) (cartOwner string, isGuest bool)
```

The existing `CustomerAuth.Authenticate` middleware is not modified.

### 2. Cart Service Changes

**Parameter additions**: All `CartService` methods (`GetCart`, `AddItem`, `UpdateItemQuantity`, `RemoveItem`, `ClearCart`) gain an `isGuest bool` parameter alongside the existing `customerID` parameter (renamed to `cartOwner`). The `cartPK()` helper already just prepends `"CART#"`, so it works for both customer IDs and session IDs.

**`recalculateAndGetCart` signature change**: `recalculateAndGetCart(ctx, pk, cartOwner string, isGuest bool)`. Sets `header.CustomerID = cartOwner` if `!isGuest`, or `header.SessionID = cartOwner` if `isGuest`. This prevents guest session UUIDs from being written into the `CustomerID` field.

All callers of `recalculateAndGetCart` must thread the `isGuest` parameter:
- `AddItem` → `s.recalculateAndGetCart(ctx, pk, cartOwner, isGuest)`
- `UpdateItemQuantity` → `s.recalculateAndGetCart(ctx, pk, cartOwner, isGuest)`
- `RemoveItem` → `s.recalculateAndGetCart(ctx, pk, cartOwner, isGuest)`

**CartService interface change** in `internal/domain/store_service.go` (not `service.go`):

```go
type CartService interface {
    GetCart(ctx context.Context, cartOwner string, isGuest bool) (*CartWithItems, error)
    AddItem(ctx context.Context, cartOwner string, isGuest bool, req AddCartItemRequest) (*CartWithItems, error)
    UpdateItemQuantity(ctx context.Context, cartOwner string, isGuest bool, productID string, quantity int) (*CartWithItems, error)
    RemoveItem(ctx context.Context, cartOwner string, isGuest bool, productID string) (*CartWithItems, error)
    ClearCart(ctx context.Context, cartOwner string) error
    MergeGuestCart(ctx context.Context, customerID string, items []AddCartItemRequest) (*CartWithItems, error)
    MergeGuestSession(ctx context.Context, customerID, guestSessionID string) error
}
```

**New method — `MergeGuestSession`**:

```go
MergeGuestSession(ctx context.Context, customerID, guestSessionID string) error
```

- Reads all items from `CART#<guestSessionID>`
- If empty, returns (no-op)
- Converts items to `[]AddCartItemRequest` and calls existing `MergeGuestCart` logic
- Deletes the guest cart partition (`CART#<guestSessionID>`) via `ClearCart`

### 3. Cart Handler Changes

All handlers in `internal/handler/store/cart_handler.go` replace:

```go
customerID := middleware.GetCustomerIDFromContext(r.Context())
```

with:

```go
cartOwner, isGuest := middleware.GetCartIdentityFromContext(r.Context())
```

And pass `isGuest` through to service calls.

The `MergeGuestCart` handler additionally clears the `guest_session` cookie after a successful merge.

Split handler routes into two methods:
- `CRUDRoutes() chi.Router` — GET cart, POST items, PATCH items, DELETE items, DELETE cart
- `MergeRoute() chi.Router` — POST merge (stays auth-required)

### 4. Cart Router Changes

`internal/router/store_cart.go`:

Cart routes use `OptionalCartAuth` for guest-accessible CRUD:

```go
r.Route("/api/v1/store/cart", func(r chi.Router) {
    r.Use(optionalCartAuth.Resolve)
    r.Mount("/", h.CRUDRoutes())
})
```

The `/merge` endpoint is removed (see Section 10) — merge is handled server-side in `VerifyOTP`.

### 5. Auth Handler — Merge on Login

`internal/handler/store/auth_handler.go`:

- **New dependency**: `CartService` injected alongside `CustomerAuthService`.
- **`AuthHandler` constructor change**: `NewAuthHandler(customerAuthService, cartService, validation)` — new parameter.
- **`VerifyOTP` change**: After successful OTP verification, check for `guest_session` cookie on the request. If present:
  1. Call `cartService.MergeGuestSession(ctx, customer.ID, guestSessionID)`
  2. Clear the `guest_session` cookie in the response (MaxAge: -1)
- Merge errors are logged but do not fail the login (fire-and-forget, consistent with event publishing pattern).

### 6. Wire Changes

Specific files to update:
- `internal/wire/store_auth.go` — add `CartService` provider to store-auth Lambda deps
- `internal/wire/monolith.go` (or equivalent) — add `CartService` to monolith deps
- Run `make wire` to regenerate `wire_gen.go`
- Run `make generate-mocks` to regenerate mocks for the updated `CartService` interface

### 7. Frontend — useCart Hook

`homechrome-store/src/hooks/useCart.ts`:

- Remove `if (!isAuthenticated) return` guard from `fetchCart`. Always fetch cart on mount.
- Keep `isAuthenticated` in the `useCallback` dependency array so the cart re-fetches on auth state changes.
- All operations (`addItem`, `updateQuantity`, `removeItem`, `clearCart`) unchanged — same API calls, backend resolves identity from cookies.
- The existing `useEffect` on `fetchCart` already re-fetches the cart when `isAuthenticated` changes (login or logout), picking up the merged cart post-login or an empty cart post-logout.

**Logout behavior**: On logout, `isAuthenticated` changes to `false`, triggering `fetchCart`. The backend receives no `store_token` and no `guest_session` cookie (both cleared), so it generates a new empty guest session. The UI shows an empty cart — correct behavior.

### 8. Frontend — Cart Page

`homechrome-store/src/app/cart/page.tsx`:

- Remove auth redirect guard (currently redirects to `/login?redirect=/cart`).
- Add auth check only on the "Proceed to Checkout" button: if `!isAuthenticated`, redirect to `/login?redirect=/cart`.

### 9. Frontend — Product Detail Page

`homechrome-store/src/app/p/[slug]/ProductDetailView.tsx`:

- Remove any auth check before add-to-cart. Button works identically for guests and authenticated users.

### 10. Remove `/merge` Endpoint

The `POST /cart/merge` endpoint was designed for client-driven merge (frontend sends guest items in the request body). With the new server-side merge-on-login in `VerifyOTP`, the frontend never calls `/merge` explicitly. Remove the endpoint and the `MergeGuestCart` handler method. The `MergeGuestCart` service method is kept as it is reused internally by `MergeGuestSession`.

## Known Limitations

- **Cart item TTL drift**: Individual `CartItem` TTL is set at creation time and not refreshed on subsequent reads or other item additions. A guest who adds item A on day 1 and item B on day 25 will see item A expire on day 31 while item B survives. This is a pre-existing issue, not introduced by this feature. A follow-up could refresh all item TTLs during `recalculateAndGetCart`.
- **Unsigned guest session cookie**: The `guest_session` cookie value is a plain UUID, not HMAC-signed. An attacker could guess/set another guest's session ID and view their cart contents (product IDs and quantities only — no PII). Accepted risk given low sensitivity.
- **No rate limiting on guest cart creation**: A bot could generate many guest carts by sending requests without cookies. Deferred to follow-up if abuse is observed.

## What Does Not Change

- `Cart` domain entity — already has `SessionID` field and dual-key `SetKeys()`
- `CartRepository` — already key-agnostic (operates on PK strings)
- DynamoDB table schema — no migration needed
- `CustomerAuth` middleware — untouched
- Checkout routes — remain authenticated-only

## File Change Summary

| Layer | File | Change |
|-------|------|--------|
| Middleware | `internal/middleware/optional_cart_auth.go` | New — `OptionalCartAuth` + `GetCartIdentityFromContext` |
| Middleware | `internal/middleware/context_keys.go` | Add `GuestSessionKey` |
| Domain | `internal/domain/store_service.go` | Update `CartService` interface: add `isGuest` param, add `MergeGuestSession` |
| Service | `internal/service/cart_service.go` | Add `isGuest` param to all methods + `recalculateAndGetCart`, add `MergeGuestSession` |
| Handler | `internal/handler/store/cart_handler.go` | Use `GetCartIdentityFromContext`, split into `CRUDRoutes`/remove merge handler |
| Handler | `internal/handler/store/auth_handler.go` | Inject `CartService`, merge guest cart in `VerifyOTP`, clear `guest_session` cookie |
| Router | `internal/router/store_cart.go` | Two sibling route groups: guest CRUD + authenticated merge |
| Wire | `internal/wire/store_auth.go` + monolith | Wire `CartService` into `AuthHandler` |
| Frontend | `src/hooks/useCart.ts` | Remove `isAuthenticated` guard from `fetchCart` |
| Frontend | `src/app/cart/page.tsx` | Remove auth redirect, gate only checkout button |
| Frontend | `src/app/p/[slug]/ProductDetailView.tsx` | Remove auth check on add-to-cart |
