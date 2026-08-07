package store

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/response"
)

// CheckoutHandler handles checkout-related requests for the storefront.
type CheckoutHandler struct {
	checkoutService domain.CheckoutService
	validation      *middleware.Validation
}

// NewCheckoutHandler creates a new CheckoutHandler.
func NewCheckoutHandler(
	cs domain.CheckoutService,
	v *middleware.Validation,
) *CheckoutHandler {
	return &CheckoutHandler{
		checkoutService: cs,
		validation:      v,
	}
}

// Routes returns the checkout routes.
// All routes require customer auth (applied at the router level).
func (h *CheckoutHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.With(middleware.ValidateJSONTyped[domain.CheckoutRequest](h.validation)).Post("/initiate", h.Initiate)
	r.Get("/payment-status/{orderID}", h.GetPaymentStatus)

	return r
}

// Initiate handles POST /initiate - creates an order and initiates payment.
func (h *CheckoutHandler) Initiate(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())

	req := middleware.MustGetValidatedBody[domain.CheckoutRequest](r.Context())

	result, err := h.checkoutService.Initiate(r.Context(), customerID, *req)
	if err != nil {
		response.Error(w, err)
		return
	}

	stripInternal(result.Order)

	response.JSON(w, http.StatusCreated, response.Response{
		Success: true,
		Data:    result,
	})
}

// GetPaymentStatus handles GET /payment-status/{orderID} - returns payment status for an order.
func (h *CheckoutHandler) GetPaymentStatus(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())

	orderID := chi.URLParam(r, "orderID")

	result, err := h.checkoutService.GetPaymentStatus(r.Context(), customerID, orderID)
	if err != nil {
		response.Error(w, err)
		return
	}

	stripInternal(result.Order)

	response.Success(w, result)
}
