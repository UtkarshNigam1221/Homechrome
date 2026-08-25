package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/handler"
)

// NewCouponRouter creates routes for the coupon service.
// Mounted at /admin/* to match API Gateway resource paths — the Lambda receives the full
// path, so mounting at /coupons would 404 every request.
func NewCouponRouter(r *chi.Mux, h *handler.CouponHandler) {
	r.Route("/admin/coupons", func(r chi.Router) {
		r.Mount("/", h.Routes())
	})
}
