# Guest Cart Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow unauthenticated users to build and manage a shopping cart, with server-side merge on login.

**Architecture:** New `OptionalCartAuth` middleware resolves cart identity from either JWT or guest session cookie. Cart service methods gain an `isGuest` parameter to correctly set `CustomerID` vs `SessionID` on the cart header. On OTP verification, the auth handler merges the guest cart into the customer's cart and clears the guest cookie. Frontend removes auth guards from cart access, gating only checkout.

**Tech Stack:** Go 1.25 (Chi, DynamoDB), React 19 (Next.js 16, Zustand), Google Wire DI

**Spec:** `docs/superpowers/specs/2026-03-18-guest-cart-design.md`

---

### Task 1: Add GuestSessionKey context key

**Files:**
- Modify: `handloom-admin/internal/middleware/interfaces.go:38-62`

- [ ] **Step 1: Add GuestSessionKey constant**

In `internal/middleware/interfaces.go`, add `GuestSessionKey` to the context key constants block:

```go
// GuestSessionKey stores the guest cart session ID
GuestSessionKey ContextKey = "guest_session_id"
```

Add it after the `CustomerKey` constant (line 61).

- [ ] **Step 2: Commit**

```bash
cd handloom-admin
git add internal/middleware/interfaces.go
git commit -m "feat(cart): add GuestSessionKey context key"
```

---

### Task 2: Create OptionalCartAuth middleware

**Files:**
- Create: `handloom-admin/internal/middleware/optional_cart_auth.go`

- [ ] **Step 1: Write the OptionalCartAuth middleware**

Create `internal/middleware/optional_cart_auth.go`:

```go
package middleware

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/logger"
)

const guestSessionCookieName = "guest_session"
const guestSessionMaxAge = 30 * 24 * time.Hour

// OptionalCartAuth resolves cart identity without requiring authentication.
// Authenticated users get CustomerIDKey set; guests get GuestSessionKey set.
type OptionalCartAuth struct {
	customerAuthService domain.CustomerAuthService
	logger              *logger.Logger
}

// NewOptionalCartAuth creates a new OptionalCartAuth middleware.
func NewOptionalCartAuth(
	customerAuthService domain.CustomerAuthService,
	logger *logger.Logger,
) *OptionalCartAuth {
	return &OptionalCartAuth{
		customerAuthService: customerAuthService,
		logger:              logger,
	}
}

// Resolve is the middleware handler that resolves cart identity.
func (m *OptionalCartAuth) Resolve(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 1. Try authenticated path: extract and validate store_token
		if token, err := extractBearerToken(r, "store_token"); err == nil {
			if claims, err := m.customerAuthService.ValidateCustomerToken(ctx, token); err == nil && claims.CustomerID != "" {
				ctx = context.WithValue(ctx, CustomerIDKey, claims.CustomerID)
				ctx = logger.SetUserID(ctx, claims.CustomerID)
				ctx = context.WithValue(ctx, CustomerKey, &domain.Customer{
					ID:    claims.CustomerID,
					Phone: claims.Phone,
					Email: claims.Email,
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// 2. Try existing guest session cookie
		if cookie, err := r.Cookie(guestSessionCookieName); err == nil && cookie.Value != "" {
			ctx = context.WithValue(ctx, GuestSessionKey, cookie.Value)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// 3. Generate new guest session
		sessionID := uuid.New().String()
		ctx = context.WithValue(ctx, GuestSessionKey, sessionID)

		secure, sameSite, cookieDomain := guestCookieSettings()
		http.SetCookie(w, &http.Cookie{
			Name:     guestSessionCookieName,
			Value:    sessionID,
			Path:     "/",
			Domain:   cookieDomain,
			HttpOnly: true,
			Secure:   secure,
			SameSite: sameSite,
			MaxAge:   int(guestSessionMaxAge / time.Second),
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// guestCookieSettings returns cookie settings matching store auth cookies.
func guestCookieSettings() (secure bool, sameSite http.SameSite, domain string) {
	if d := os.Getenv("COOKIE_DOMAIN"); d != "" {
		return true, http.SameSiteLaxMode, d
	}
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		return true, http.SameSiteNoneMode, ""
	}
	return false, http.SameSiteLaxMode, ""
}

// GetCartIdentityFromContext returns the cart owner identifier and whether the user is a guest.
// Authenticated: (customerID, false). Guest: (sessionID, true).
func GetCartIdentityFromContext(ctx context.Context) (cartOwner string, isGuest bool) {
	if customerID, ok := ctx.Value(CustomerIDKey).(string); ok && customerID != "" {
		return customerID, false
	}
	if sessionID, ok := ctx.Value(GuestSessionKey).(string); ok && sessionID != "" {
		return sessionID, true
	}
	return "", true
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd handloom-admin && go build ./internal/middleware/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/middleware/optional_cart_auth.go
git commit -m "feat(cart): add OptionalCartAuth middleware with guest session cookie"
```

---

### Task 3: Update CartService interface and implementation

**Files:**
- Modify: `handloom-admin/internal/domain/store_service.go:26-34`
- Modify: `handloom-admin/internal/service/cart_service.go`

- [ ] **Step 1: Update CartService interface**

In `internal/domain/store_service.go`, replace the `CartService` interface (lines 27-34):

```go
// CartService defines shopping cart operations
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

- [ ] **Step 2: Update recalculateAndGetCart signature**

In `internal/service/cart_service.go`, update `recalculateAndGetCart` (lines 187-215) to accept `isGuest bool` and conditionally set `CustomerID` vs `SessionID`:

```go
func (s *CartService) recalculateAndGetCart(ctx context.Context, pk, cartOwner string, isGuest bool) (*domain.CartWithItems, error) {
	cart, err := s.cartRepo.GetCart(ctx, pk)
	if err != nil {
		return nil, err
	}

	var subtotal int64
	for _, item := range cart.Items {
		subtotal += item.TotalPrice
	}

	header := cart.Cart
	if isGuest {
		header.SessionID = cartOwner
		header.CustomerID = ""
	} else {
		header.CustomerID = cartOwner
	}
	header.ItemCount = len(cart.Items)
	header.Subtotal = subtotal
	header.Currency = "INR"
	header.UpdatedAt = time.Now()
	header.TTL = cartTTL()
	header.EntityType = "CART"
	header.PK = pk
	header.SK = "METADATA"

	if err := s.cartRepo.UpdateCartHeader(ctx, header); err != nil {
		return nil, err
	}

	cart.Cart = header
	return cart, nil
}
```

- [ ] **Step 3: Update GetCart**

Change signature from `GetCart(ctx, customerID)` to `GetCart(ctx, cartOwner, isGuest)`:

```go
func (s *CartService) GetCart(ctx context.Context, cartOwner string, isGuest bool) (*domain.CartWithItems, error) {
	return s.cartRepo.GetCart(ctx, cartPK(cartOwner))
}
```

Note: `isGuest` is unused in `GetCart` since it just reads, but the interface requires it for consistency.

- [ ] **Step 4: Update AddItem**

Change signature and thread `isGuest`:

```go
func (s *CartService) AddItem(ctx context.Context, cartOwner string, isGuest bool, req domain.AddCartItemRequest) (*domain.CartWithItems, error) {
	product, err := s.productRepo.GetByID(ctx, req.ProductID)
	if err != nil {
		return nil, err
	}
	if product.Status != domain.ProductStatusActive {
		return nil, errors.BadRequest("Product is not available")
	}

	if err := s.validateStock(ctx, req.ProductID, req.Quantity); err != nil {
		return nil, err
	}

	pk := cartPK(cartOwner)

	item := &domain.CartItem{
		ProductID:    req.ProductID,
		ProductName:  product.Name,
		ProductSKU:   product.SKU,
		ProductImage: primaryImage(product.Images),
		CategoryID:   product.CategoryID,
		Quantity:     req.Quantity,
		UnitPrice:    product.SellingPrice,
		TotalPrice:   product.SellingPrice * int64(req.Quantity),
		IsCustomSize: req.Dimensions != nil,
		Dimensions:   req.Dimensions,
		QuoteID:      req.QuoteID,
		AddedAt:      time.Now(),
		TTL:          cartTTL(),
	}
	item.SetKeys(pk)

	if err := s.cartRepo.PutCartItem(ctx, item); err != nil {
		return nil, err
	}

	s.logger.WithContext(ctx).Infof("Added item %s to cart for %s", req.ProductID, cartOwner)

	return s.recalculateAndGetCart(ctx, pk, cartOwner, isGuest)
}
```

- [ ] **Step 5: Update UpdateItemQuantity**

```go
func (s *CartService) UpdateItemQuantity(ctx context.Context, cartOwner string, isGuest bool, productID string, quantity int) (*domain.CartWithItems, error) {
	if quantity == 0 {
		return s.RemoveItem(ctx, cartOwner, isGuest, productID)
	}

	if err := s.validateStock(ctx, productID, quantity); err != nil {
		return nil, err
	}

	pk := cartPK(cartOwner)

	cart, err := s.cartRepo.GetCart(ctx, pk)
	if err != nil {
		return nil, err
	}

	unitPrice, found := findItemUnitPrice(cart.Items, productID)
	if !found {
		return nil, errors.NotFound("Cart item not found")
	}

	if err := s.cartRepo.UpdateCartItem(ctx, pk, productID, quantity, unitPrice*int64(quantity)); err != nil {
		return nil, err
	}

	s.logger.WithContext(ctx).Infof("Updated item %s quantity to %d for %s", productID, quantity, cartOwner)

	return s.recalculateAndGetCart(ctx, pk, cartOwner, isGuest)
}
```

- [ ] **Step 6: Update RemoveItem**

```go
func (s *CartService) RemoveItem(ctx context.Context, cartOwner string, isGuest bool, productID string) (*domain.CartWithItems, error) {
	pk := cartPK(cartOwner)

	if err := s.cartRepo.DeleteCartItem(ctx, pk, productID); err != nil {
		return nil, err
	}

	s.logger.WithContext(ctx).Infof("Removed item %s from cart for %s", productID, cartOwner)

	return s.recalculateAndGetCart(ctx, pk, cartOwner, isGuest)
}
```

- [ ] **Step 7: Update MergeGuestCart internal calls**

`MergeGuestCart` calls `AddItem` internally. Update the call to pass `isGuest: false` since merge always targets the customer's cart:

```go
func (s *CartService) MergeGuestCart(ctx context.Context, customerID string, items []domain.AddCartItemRequest) (*domain.CartWithItems, error) {
	pk := cartPK(customerID)

	existingCart, err := s.cartRepo.GetCart(ctx, pk)
	if err != nil {
		return nil, err
	}

	existingQty := make(map[string]int, len(existingCart.Items))
	for _, item := range existingCart.Items {
		existingQty[item.ProductID] = item.Quantity
	}

	for _, req := range items {
		if qty, exists := existingQty[req.ProductID]; exists && qty >= req.Quantity {
			continue
		}
		if _, err := s.AddItem(ctx, customerID, false, req); err != nil {
			s.logger.WithContext(ctx).Warnf("Failed to merge item %s: %v", req.ProductID, err)
			continue
		}
	}

	s.logger.WithContext(ctx).Infof("Merged guest cart (%d items) for customer %s", len(items), customerID)

	return s.cartRepo.GetCart(ctx, pk)
}
```

- [ ] **Step 8: Add MergeGuestSession method**

Add after `MergeGuestCart`:

```go
// MergeGuestSession reads items from a guest cart and merges them into the customer's cart.
func (s *CartService) MergeGuestSession(ctx context.Context, customerID, guestSessionID string) error {
	guestCart, err := s.cartRepo.GetCart(ctx, cartPK(guestSessionID))
	if err != nil {
		return err
	}

	if len(guestCart.Items) == 0 {
		return nil
	}

	items := make([]domain.AddCartItemRequest, len(guestCart.Items))
	for i, item := range guestCart.Items {
		items[i] = domain.AddCartItemRequest{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}

	if _, err := s.MergeGuestCart(ctx, customerID, items); err != nil {
		return err
	}

	return s.ClearCart(ctx, guestSessionID)
}
```

- [ ] **Step 9: Update interface compliance check**

The `var _ domain.CartService = (*CartService)(nil)` at the bottom already ensures compile-time checking.

- [ ] **Step 10: Verify it compiles**

```bash
cd handloom-admin && go build ./internal/domain/... ./internal/service/...
```

This will fail because callers (handler, checkout service) haven't been updated yet. That's expected — we fix callers in the next tasks.

- [ ] **Step 11: Commit**

```bash
git add internal/domain/store_service.go internal/service/cart_service.go
git commit -m "feat(cart): add isGuest param to CartService methods + MergeGuestSession"
```

---

### Task 4: Update cart handler

**Files:**
- Modify: `handloom-admin/internal/handler/store/cart_handler.go`

- [ ] **Step 1: Replace Routes() with CRUDRoutes() and delete MergeGuestCart handler**

Replace the `Routes()` method with `CRUDRoutes()`, update all handlers to use `GetCartIdentityFromContext` instead of `GetCustomerIDFromContext`, and **delete the `MergeGuestCart` handler method entirely** (lines 122-134) — the `/merge` endpoint is removed per spec Section 10:

```go
// CRUDRoutes returns cart CRUD routes (used with OptionalCartAuth middleware).
func (h *CartHandler) CRUDRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.GetCart)
	r.With(middleware.ValidateJSONTyped[domain.AddCartItemRequest](h.validation)).Post("/items", h.AddItem)
	r.With(middleware.ValidateJSONTyped[domain.UpdateCartItemRequest](h.validation)).Patch("/items/{productID}", h.UpdateQuantity)
	r.Delete("/items/{productID}", h.RemoveItem)
	r.Delete("/", h.ClearCart)

	return r
}
```

- [ ] **Step 2: Update all handler methods to use GetCartIdentityFromContext**

Replace each handler to use `cartOwner, isGuest := middleware.GetCartIdentityFromContext(r.Context())` and pass `isGuest` to service calls:

```go
func (h *CartHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	cartOwner, isGuest := middleware.GetCartIdentityFromContext(r.Context())

	cart, err := h.cartService.GetCart(r.Context(), cartOwner, isGuest)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, cart)
}

func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	cartOwner, isGuest := middleware.GetCartIdentityFromContext(r.Context())
	req := middleware.MustGetValidatedBody[domain.AddCartItemRequest](r.Context())

	cart, err := h.cartService.AddItem(r.Context(), cartOwner, isGuest, *req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, cart)
}

func (h *CartHandler) UpdateQuantity(w http.ResponseWriter, r *http.Request) {
	cartOwner, isGuest := middleware.GetCartIdentityFromContext(r.Context())
	productID := chi.URLParam(r, "productID")
	req := middleware.MustGetValidatedBody[domain.UpdateCartItemRequest](r.Context())

	cart, err := h.cartService.UpdateItemQuantity(r.Context(), cartOwner, isGuest, productID, req.Quantity)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, cart)
}

func (h *CartHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	cartOwner, isGuest := middleware.GetCartIdentityFromContext(r.Context())
	productID := chi.URLParam(r, "productID")

	cart, err := h.cartService.RemoveItem(r.Context(), cartOwner, isGuest, productID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, cart)
}

func (h *CartHandler) ClearCart(w http.ResponseWriter, r *http.Request) {
	cartOwner, _ := middleware.GetCartIdentityFromContext(r.Context())

	if err := h.cartService.ClearCart(r.Context(), cartOwner); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Cart cleared successfully"})
}
```

- [ ] **Step 3: Verify it compiles**

```bash
cd handloom-admin && go build ./internal/handler/store/...
```

- [ ] **Step 4: Commit**

```bash
git add internal/handler/store/cart_handler.go
git commit -m "feat(cart): update cart handlers to support guest identity"
```

---

### Task 5: Update cart router for guest access

**Files:**
- Modify: `handloom-admin/internal/router/store_cart.go`

- [ ] **Step 1: Update router to use OptionalCartAuth**

Replace the entire file:

```go
package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/handler/store"
	"github.com/handloom/admin/internal/middleware"
)

// NewStoreCartRouter creates routes for the store cart service.
// Cart CRUD routes use OptionalCartAuth (guest + authenticated access).
func NewStoreCartRouter(r *chi.Mux, h *store.CartHandler, optionalCartAuth *middleware.OptionalCartAuth) {
	r.Route("/api/v1/store/cart", func(r chi.Router) {
		r.Use(optionalCartAuth.Resolve)
		r.Mount("/", h.CRUDRoutes())
	})
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/router/store_cart.go
git commit -m "feat(cart): switch cart router to OptionalCartAuth middleware"
```

---

### Task 6: Update auth handler for merge-on-login

**Files:**
- Modify: `handloom-admin/internal/handler/store/auth_handler.go`

- [ ] **Step 1: Add CartService dependency to AuthHandler**

Update the struct and constructor:

```go
type AuthHandler struct {
	customerAuthService domain.CustomerAuthService
	cartService         domain.CartService
	validation          *middleware.Validation
}

func NewAuthHandler(customerAuthService domain.CustomerAuthService, cartService domain.CartService, validation *middleware.Validation) *AuthHandler {
	return &AuthHandler{
		customerAuthService: customerAuthService,
		cartService:         cartService,
		validation:          validation,
	}
}
```

- [ ] **Step 2: Add logger import and update VerifyOTP for guest merge**

Add `"github.com/handloom/admin/pkg/logger"` to imports (only needed if not already there — actually the auth handler doesn't have a logger, so we'll use `log.Printf` or add one. Per the spec, merge errors are logged but don't fail login. We'll keep it simple and use the existing pattern.)

Actually, looking at the auth handler, it doesn't have a logger field. The spec says "merge errors are logged but do not fail the login." We'll add a logger dependency:

```go
type AuthHandler struct {
	customerAuthService domain.CustomerAuthService
	cartService         domain.CartService
	validation          *middleware.Validation
	logger              *logger.Logger
}

func NewAuthHandler(customerAuthService domain.CustomerAuthService, cartService domain.CartService, validation *middleware.Validation, logger *logger.Logger) *AuthHandler {
	return &AuthHandler{
		customerAuthService: customerAuthService,
		cartService:         cartService,
		validation:          validation,
		logger:              logger,
	}
}
```

Add `"github.com/handloom/admin/pkg/logger"` to the import block.

- [ ] **Step 3: Update VerifyOTP to merge guest cart and clear cookie**

Replace the `VerifyOTP` method:

```go
func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := middleware.MustGetValidatedBody[domain.VerifyOTPRequest](ctx)

	customer, tokens, isNewCustomer, err := h.customerAuthService.VerifyOTP(ctx, req.Phone, req.Code)
	if err != nil {
		response.Error(w, err)
		return
	}

	h.setStoreCookies(w, tokens)

	// Merge guest cart if guest_session cookie is present
	if cookie, err := r.Cookie("guest_session"); err == nil && cookie.Value != "" {
		if mergeErr := h.cartService.MergeGuestSession(ctx, customer.ID, cookie.Value); mergeErr != nil {
			h.logger.WithContext(ctx).Warnf("Failed to merge guest cart for customer %s: %v", customer.ID, mergeErr)
		}
		// Clear the guest_session cookie
		secure, sameSite, domain := cookieSettings()
		http.SetCookie(w, &http.Cookie{
			Name:     "guest_session",
			Value:    "",
			Path:     "/",
			Domain:   domain,
			HttpOnly: true,
			Secure:   secure,
			SameSite: sameSite,
			MaxAge:   -1,
		})
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"customer":        customer,
		"is_new_customer": isNewCustomer,
	})
}
```

- [ ] **Step 4: Verify it compiles**

```bash
cd handloom-admin && go build ./internal/handler/store/...
```

- [ ] **Step 5: Commit**

```bash
git add internal/handler/store/auth_handler.go
git commit -m "feat(cart): merge guest cart on login in VerifyOTP handler"
```

---

### Task 7: Update Wire configuration and providers

**Files:**
- Modify: `handloom-admin/internal/wire/providers.go:666-672` (ProvideStoreAuthHandler)
- Modify: `handloom-admin/internal/wire/providers.go:684-691` (ProvideStoreCartHandler — no change needed)
- Modify: `handloom-admin/internal/wire/wire.go:415-421` (StoreAuthDeps)
- Modify: `handloom-admin/internal/wire/wire.go:430-436` (StoreCartDeps)
- Modify: `handloom-admin/internal/wire/wire.go:487-503` (InitializeStoreAuthDeps)
- Modify: `handloom-admin/internal/wire/wire.go:524-544` (InitializeStoreCartDeps)

- [ ] **Step 1: Add ProvideOptionalCartAuth provider**

In `internal/wire/providers.go`, add after `ProvideCustomerAuthMiddleware` (around line 757):

```go
// ProvideOptionalCartAuth creates the OptionalCartAuth middleware
func ProvideOptionalCartAuth(
	customerAuthService *service.CustomerAuthService,
	log *logger.Logger,
) *middleware.OptionalCartAuth {
	return middleware.NewOptionalCartAuth(customerAuthService, log)
}
```

- [ ] **Step 2: Update ProvideStoreAuthHandler**

In `internal/wire/providers.go`, update `ProvideStoreAuthHandler` (lines 667-672) to pass `CartService` and `Logger`:

```go
func ProvideStoreAuthHandler(
	customerAuthService *service.CustomerAuthService,
	cartService *service.CartService,
	validation *middleware.Validation,
	log *logger.Logger,
) *store.AuthHandler {
	return store.NewAuthHandler(customerAuthService, cartService, validation, log)
}
```

- [ ] **Step 3: Update StoreAuthDeps to include CartService providers**

In `internal/wire/wire.go`, update `StoreAuthDeps` (lines 416-421):

```go
type StoreAuthDeps struct {
	Config                 *config.Config
	Logger                 *logger.Logger
	Handler                *store.AuthHandler
	CustomerAuthMiddleware *middleware.CustomerAuth
}
```

No change to the struct — the deps struct doesn't need `CartService` directly since it's wired through the handler.

- [ ] **Step 4: Update InitializeStoreAuthDeps wire.Build**

In `internal/wire/wire.go`, update `InitializeStoreAuthDeps` (lines 488-503) to include cart-related providers so `CartService` can be injected into the auth handler:

```go
func InitializeStoreAuthDeps(ctx context.Context, cfg *config.Config) (*StoreAuthDeps, error) {
	wire.Build(
		CoreSet,
		ProvideValidator,
		ProvideValidation,
		ProvideOTPRepository,
		ProvideCustomerRepository,
		ProvideCustomerTokenStore,
		ProvideEventPublisher,
		ProvideCustomerAuthService,
		// Cart deps needed for guest merge in auth handler
		ProvideCartRepository,
		ProvideProductRepository,
		ProvideInventoryRepository,
		ProvideCartService,
		ProvideStoreAuthHandler,
		ProvideCustomerAuthMiddleware,
		wire.Struct(new(StoreAuthDeps), "*"),
	)
	return nil, nil
}
```

- [ ] **Step 5: Update StoreCartDeps to use OptionalCartAuth**

In `internal/wire/wire.go`, replace `CustomerAuthMiddleware` with `OptionalCartAuth` in `StoreCartDeps`:

```go
type StoreCartDeps struct {
	Config              *config.Config
	Logger              *logger.Logger
	Handler             *store.CartHandler
	OptionalCartAuth    *middleware.OptionalCartAuth
}
```

- [ ] **Step 6: Update InitializeStoreCartDeps**

Replace the wire.Build to use `ProvideOptionalCartAuth` instead of `ProvideCustomerAuthMiddleware`:

```go
func InitializeStoreCartDeps(ctx context.Context, cfg *config.Config) (*StoreCartDeps, error) {
	wire.Build(
		CoreSet,
		ProvideValidator,
		ProvideValidation,
		ProvideCartRepository,
		ProvideProductRepository,
		ProvideInventoryRepository,
		ProvideOTPRepository,
		ProvideCustomerRepository,
		ProvideCustomerTokenStore,
		ProvideEventPublisher,
		ProvideCustomerAuthService,
		ProvideCartService,
		ProvideStoreCartHandler,
		ProvideOptionalCartAuth,
		wire.Struct(new(StoreCartDeps), "*"),
	)
	return nil, nil
}
```

- [ ] **Step 7: Verify it compiles (pre-wire)**

```bash
cd handloom-admin && go build ./internal/wire/...
```

This may fail because wire_gen.go is stale. That's fine — next step regenerates it.

- [ ] **Step 8: Commit wire source changes**

```bash
git add internal/wire/providers.go internal/wire/wire.go
git commit -m "feat(cart): update Wire config for OptionalCartAuth and auth handler cart injection"
```

---

### Task 8: Update store-cart Lambda entry point

**Files:**
- Modify: `handloom-admin/cmd/lambda/store-cart/main.go:32`

- [ ] **Step 1: Update router call to use OptionalCartAuth**

Change line 32 from:
```go
router.NewStoreCartRouter(r, deps.Handler, deps.CustomerAuthMiddleware)
```
to:
```go
router.NewStoreCartRouter(r, deps.Handler, deps.OptionalCartAuth)
```

- [ ] **Step 2: Commit**

```bash
git add cmd/lambda/store-cart/main.go
git commit -m "feat(cart): update store-cart Lambda to use OptionalCartAuth"
```

---

### Task 9: Update monolith entry point

**Files:**
- Modify: `handloom-admin/cmd/api/main.go`

- [ ] **Step 1: Create OptionalCartAuth in monolith main**

In `cmd/api/main.go`, after the line creating `customerAuthMiddleware` (line 254), add:

```go
optionalCartAuth := middleware.NewOptionalCartAuth(customerAuthService, log)
```

- [ ] **Step 2: Update createRouter signature and call**

Add `optionalCartAuth *middleware.OptionalCartAuth` parameter to `createRouter` function signature (after `customerAuthMiddleware`).

Update the call site (around line 257-267) to pass `optionalCartAuth`.

- [ ] **Step 3: Update createRouter body — cart routes**

In the `createRouter` body, change the cart mounting (around line 391) from:

```go
r.Mount("/cart", storeCartHandler.Routes())
```

to extract cart out of the customer-authenticated group and mount it separately:

```go
// Guest-accessible cart routes
r.Group(func(r chi.Router) {
	r.Use(optionalCartAuth.Resolve)
	r.Mount("/cart", storeCartHandler.CRUDRoutes())
})
```

This must be OUTSIDE the `customerAuthMiddleware.Authenticate` group and INSIDE the `/api/v1/store` route.

- [ ] **Step 4: Update storeAuthHandler construction**

Update the `NewAuthHandler` call (around line 243) to pass `cartService` and `log`:

```go
storeAuthHandler := store.NewAuthHandler(customerAuthService, cartService, validation, log)
```

- [ ] **Step 5: Verify it compiles**

```bash
cd handloom-admin && go build ./cmd/api/...
```

- [ ] **Step 6: Commit**

```bash
git add cmd/api/main.go
git commit -m "feat(cart): update monolith for guest cart access and auth merge"
```

---

### Task 10: Fix callers — CheckoutService

**Files:**
- Modify: `handloom-admin/internal/service/checkout_service.go` (likely calls CartService methods)

The `CheckoutService` calls `CartService.GetCart` and `CartService.ClearCart`. Since checkout is always authenticated, pass `isGuest: false`.

- [ ] **Step 1: Find and update CartService.GetCart calls in CheckoutService**

Search for calls:
```bash
cd handloom-admin && grep -n 'cartService\.' internal/service/checkout_service.go
```

Only `GetCart`, `AddItem`, `UpdateItemQuantity`, and `RemoveItem` gained the `isGuest` parameter. `ClearCart` did NOT change signature. Update only the affected calls — add `false` for `isGuest` since checkout is always authenticated. For example, `s.cartService.GetCart(ctx, customerID)` becomes `s.cartService.GetCart(ctx, customerID, false)`.

- [ ] **Step 2: Verify it compiles**

```bash
cd handloom-admin && go build ./internal/service/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/service/checkout_service.go
git commit -m "fix(cart): update CheckoutService cart calls with isGuest=false"
```

---

### Task 11: Regenerate Wire and Mocks

**Files:**
- Regenerate: `handloom-admin/internal/wire/wire_gen.go`
- Regenerate: mock files

- [ ] **Step 1: Run wire generation**

```bash
cd handloom-admin && make wire
```

Expected: `wire_gen.go` regenerated successfully.

- [ ] **Step 2: Run mock generation**

```bash
cd handloom-admin && make generate-mocks
```

Expected: Mocks regenerated for updated `CartService` interface.

- [ ] **Step 3: Verify full build**

```bash
cd handloom-admin && go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/wire/wire_gen.go internal/mocks/
git commit -m "chore: regenerate Wire and mocks for guest cart changes"
```

---

### Task 12: Fix unit tests

**Files:**
- Modify: test files that call CartService methods (likely `internal/service/cart_service_test.go`, handler tests)

- [ ] **Step 1: Find all test files calling CartService methods**

```bash
cd handloom-admin && grep -rn 'GetCart\|AddItem\|UpdateItemQuantity\|RemoveItem\|MergeGuestCart' --include='*_test.go' internal/
```

- [ ] **Step 2: Update test calls to include isGuest parameter**

For each test calling cart service methods, add the `isGuest` parameter. Existing tests for authenticated flows should pass `false`. Add at least one test for guest flow passing `true`.

- [ ] **Step 2b: Add tests for OptionalCartAuth middleware**

Create `internal/middleware/optional_cart_auth_test.go` with tests for all 3 code paths:
1. Valid `store_token` cookie → sets `CustomerIDKey` in context, no `guest_session` cookie set
2. No `store_token`, existing `guest_session` cookie → sets `GuestSessionKey` from cookie value
3. No `store_token`, no `guest_session` cookie → generates UUID, sets `GuestSessionKey`, writes `guest_session` cookie

- [ ] **Step 2c: Add test for MergeGuestSession**

In `internal/service/cart_service_test.go`, add a test for `MergeGuestSession`:
- Set up a guest cart with items, call `MergeGuestSession`, verify items merged into customer cart, guest cart cleared

- [ ] **Step 2d: Add test for VerifyOTP merge-on-login**

In auth handler tests, verify that `VerifyOTP` calls `MergeGuestSession` when `guest_session` cookie is present and clears the cookie in the response.

- [ ] **Step 3: Run tests**

```bash
cd handloom-admin && make test-unit
```

Expected: All tests pass.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "test: update cart tests for isGuest parameter"
```

---

### Task 13: Frontend — Remove auth guard from useCart hook

**Files:**
- Modify: `homechrome-store/src/hooks/useCart.ts:24-25`

- [ ] **Step 1: Remove the isAuthenticated guard from fetchCart**

Change `fetchCart` to always fetch:

```typescript
const fetchCart = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await api.get<CartWithItems>('/api/v1/store/cart');
      updateCart(data);
    } catch {
      /* ignore — may be network error or empty guest cart */
    } finally {
      setLoading(false);
    }
  }, [isAuthenticated, updateCart]);
```

Key changes:
- Remove `if (!isAuthenticated) return;` (line 25)
- Keep `isAuthenticated` in the dependency array so cart re-fetches on login/logout state changes

- [ ] **Step 2: Commit**

```bash
cd homechrome-store
git add src/hooks/useCart.ts
git commit -m "feat(cart): allow guest users to fetch cart"
```

---

### Task 14: Frontend — Update cart page

**Files:**
- Modify: `homechrome-store/src/app/cart/page.tsx`

- [ ] **Step 1: Remove unauthenticated redirect, gate only checkout**

Replace the entire cart page. Remove the `!isAuthenticated` block (lines 28-62). Keep loading state and empty cart state. Add auth check on the checkout button instead:

```tsx
'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';

import CartItemComponent from '@/components/cart/CartItem';
import CartSummary from '@/components/cart/CartSummary';
import { useCart } from '@/hooks/useCart';
import { useAuthStore } from '@/stores/auth';

export default function CartPage() {
  const { cart, loading, updateQuantity, removeItem } = useCart();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const isAuthLoading = useAuthStore((s) => s.isLoading);

  // Loading state
  if (isAuthLoading || loading) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
        <h1 className="text-3xl font-bold text-foreground">Shopping Cart</h1>
        <div className="mt-8 flex items-center justify-center py-16">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        </div>
      </div>
    );
  }

  // Empty cart
  const items = cart?.items || [];
  if (items.length === 0) {
    return (
      <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
        <h1 className="text-3xl font-bold text-foreground">Shopping Cart</h1>
        <div className="mt-8 flex flex-col items-center justify-center py-16 text-center">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            strokeWidth={1}
            stroke="currentColor"
            className="h-16 w-16 text-muted/50"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M15.75 10.5V6a3.75 3.75 0 1 0-7.5 0v4.5m11.356-1.993 1.263 12c.07.665-.45 1.243-1.119 1.243H4.25a1.125 1.125 0 0 1-1.12-1.243l1.264-12A1.125 1.125 0 0 1 5.513 7.5h12.974c.576 0 1.059.435 1.119 1.007ZM8.625 10.5a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm7.5 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Z"
            />
          </svg>
          <h2 className="mt-4 text-lg font-medium text-foreground">
            Your cart is empty
          </h2>
          <p className="mt-2 text-sm text-muted">
            Browse our collection and add some beautiful textiles.
          </p>
          <Link
            href="/products"
            className="mt-6 rounded-lg bg-primary px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-primary-dark"
          >
            Start Shopping
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
      <h1 className="text-3xl font-bold text-foreground">Shopping Cart</h1>

      <div className="mt-8 grid grid-cols-1 gap-8 lg:grid-cols-3">
        {/* Cart items */}
        <div className="space-y-4 lg:col-span-2">
          {items.map((item) => (
            <CartItemComponent
              key={item.product_id}
              item={item}
              onUpdateQuantity={async (productId, qty) => {
                await updateQuantity(productId, qty);
              }}
              onRemove={async (productId) => {
                await removeItem(productId);
              }}
            />
          ))}
        </div>

        {/* Summary */}
        <div>
          <div className="sticky top-32">
            <CartSummary
              subtotal={cart?.cart.subtotal || 0}
              itemCount={cart?.cart.item_count || 0}
              isAuthenticated={isAuthenticated}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Update CartSummary to gate checkout button**

In `homechrome-store/src/components/cart/CartSummary.tsx`, add `isAuthenticated` prop and gate the checkout link:

Add to the interface:
```typescript
interface CartSummaryProps {
  subtotal: number;
  itemCount: number;
  isAuthenticated: boolean;
}
```

Update the checkout button section:
```tsx
<div className="mt-6">
  {isAuthenticated ? (
    <Link href="/checkout">
      <Button variant="primary" size="lg" className="w-full" disabled={itemCount === 0}>
        Proceed to Checkout
      </Button>
    </Link>
  ) : (
    <Link href="/login?redirect=/cart">
      <Button variant="primary" size="lg" className="w-full" disabled={itemCount === 0}>
        Sign in to Checkout
      </Button>
    </Link>
  )}
</div>
```

- [ ] **Step 3: Commit**

```bash
cd homechrome-store
git add src/app/cart/page.tsx src/components/cart/CartSummary.tsx
git commit -m "feat(cart): remove auth gate from cart page, gate only checkout"
```

---

### Task 15: Frontend — Remove auth check from product detail add-to-cart

**Files:**
- Modify: `homechrome-store/src/app/p/[slug]/ProductDetailView.tsx:52-56`

- [ ] **Step 1: Remove the auth redirect from handleAddToCart**

In `ProductDetailView.tsx`, remove the auth check at lines 53-56:

```typescript
const handleAddToCart = async () => {
    setLoading(true);
    try {
      await addItem(product.id, quantity);
      track('add_to_cart', {
        product_id: product.id,
        product_name: product.name,
        category_id: product.category_id,
        price: product.selling_price,
        quantity,
      });
    } catch (err) {
      console.error('Failed to add to cart:', err);
    } finally {
      setLoading(false);
    }
  };
```

The `isAuthenticated` import from `useAuthStore` can be removed if no longer used elsewhere in the component. Check if `isAuthenticated` is used in the JSX — it is not (the cart qty display and increment/decrement don't check auth). Remove the `isAuthenticated` line (line 33) and the `useAuthStore` import if unused.

- [ ] **Step 2: Commit**

```bash
cd homechrome-store
git add src/app/p/[slug]/ProductDetailView.tsx
git commit -m "feat(cart): allow guests to add to cart from product page"
```

---

### Task 16: Run full build and tests

- [ ] **Step 1: Run backend tests**

```bash
cd handloom-admin && make test
```

- [ ] **Step 2: Run frontend build check**

```bash
cd homechrome-store && npm run build
```

- [ ] **Step 3: Fix any failures**

Address any compilation or test failures.

- [ ] **Step 4: Final commit if any fixes**

```bash
git add -A
git commit -m "fix: address build/test issues from guest cart feature"
```
