package store

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/logger"
	"github.com/handloom/admin/pkg/response"
)

// CartHandler handles cart-related requests for the storefront.
type CartHandler struct {
	cartService domain.CartService
	validation  *middleware.Validation
	logger      *logger.Logger
}

// NewCartHandler creates a new CartHandler.
func NewCartHandler(
	cartService domain.CartService,
	validation *middleware.Validation,
	logger *logger.Logger,
) *CartHandler {
	return &CartHandler{
		cartService: cartService,
		validation:  validation,
		logger:      logger,
	}
}

// Routes returns the cart routes.
// All routes require customer auth (applied at the router level).
func (h *CartHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.GetCart)
	r.With(middleware.ValidateJSONTyped[domain.AddCartItemRequest](h.validation)).Post("/items", h.AddItem)
	r.With(middleware.ValidateJSONTyped[domain.UpdateCartItemRequest](h.validation)).Patch("/items/{productID}", h.UpdateQuantity)
	r.Delete("/items/{productID}", h.RemoveItem)
	r.Delete("/", h.ClearCart)
	r.With(middleware.ValidateJSONTyped[domain.MergeCartRequest](h.validation)).Post("/merge", h.MergeGuestCart)

	return r
}

// GetCart handles GET / — returns the customer's cart with all items.
func (h *CartHandler) GetCart(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())

	cart, err := h.cartService.GetCart(r.Context(), customerID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, cart)
}

// AddItem handles POST /items — adds a product to the cart.
func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())

	req := middleware.MustGetValidatedBody[domain.AddCartItemRequest](r.Context())

	cart, err := h.cartService.AddItem(r.Context(), customerID, *req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, cart)
}

// UpdateQuantity handles PATCH /items/{productID} — updates item quantity.
func (h *CartHandler) UpdateQuantity(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())

	productID := chi.URLParam(r, "productID")

	req := middleware.MustGetValidatedBody[domain.UpdateCartItemRequest](r.Context())

	cart, err := h.cartService.UpdateItemQuantity(r.Context(), customerID, productID, req.Quantity)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, cart)
}

// RemoveItem handles DELETE /items/{productID} — removes an item from the cart.
func (h *CartHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())

	productID := chi.URLParam(r, "productID")

	cart, err := h.cartService.RemoveItem(r.Context(), customerID, productID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, cart)
}

// ClearCart handles DELETE / — removes all items from the cart.
func (h *CartHandler) ClearCart(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())

	if err := h.cartService.ClearCart(r.Context(), customerID); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Cart cleared successfully"})
}

// MergeGuestCart handles POST /merge — merges guest cart items into the customer's cart.
func (h *CartHandler) MergeGuestCart(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())

	req := middleware.MustGetValidatedBody[domain.MergeCartRequest](r.Context())

	cart, err := h.cartService.MergeGuestCart(r.Context(), customerID, req.Items)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Success(w, cart)
}
