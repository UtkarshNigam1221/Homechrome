package store

import (
	"log/slog"
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
	r.With(middleware.ValidateJSONTyped[ValidateCouponRequest](h.validation)).
		Post("/validate-coupon", h.ValidateCoupon)
	r.Get("/coupons", h.ListCoupons)

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

// ValidateCouponRequest is a customer asking what a code would save them.
// No cart total: the server computes it. A figure from a browser is never trusted.
type ValidateCouponRequest struct {
	Code string `json:"code" validate:"required"`
}

// ValidateCoupon previews a coupon against the customer's current cart.
// POST /api/v1/store/checkout/validate-coupon
func (h *CheckoutHandler) ValidateCoupon(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())
	req := middleware.MustGetValidatedBody[ValidateCouponRequest](r.Context())

	result, err := h.checkoutService.PreviewCoupon(r.Context(), customerID, req.Code)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, response.Response{Success: true, Data: result})
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

// StoreCouponOffer is one coupon priced against the customer's cart. Embeds the same
// trimmed DTO the banner uses, so neither surface can leak what the other withholds.
type StoreCouponOffer struct {
	StoreCoupon
	Eligible       bool   `json:"eligible"`
	DiscountAmount int64  `json:"discount_amount"`
	Reason         string `json:"reason,omitempty"`
}

// ListCoupons handles GET /api/v1/store/checkout/coupons — the coupon picker.
// Per customer and per cart, so it is never cached.
func (h *CheckoutHandler) ListCoupons(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())

	w.Header().Set("Cache-Control", "no-store")

	offers, err := h.checkoutService.ListCoupons(r.Context(), customerID)
	if err != nil {
		// The customer can still type a code, and Validate is the authority.
		slog.WarnContext(r.Context(), "Coupon picker unavailable", "error", err)
		offers = nil
	}

	out := make([]StoreCouponOffer, 0, len(offers))
	for _, o := range offers {
		out = append(out, StoreCouponOffer{
			StoreCoupon:    toStoreCoupon(o.Coupon),
			Eligible:       o.Eligible,
			DiscountAmount: o.DiscountAmount,
			Reason:         o.Reason,
		})
	}

	response.Success(w, out)
}
