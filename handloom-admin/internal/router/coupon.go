package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/handler"
)

// NewCouponRouter creates routes for the coupon service
// Routes are mounted via h.Routes() which includes validation middleware
func NewCouponRouter(r *chi.Mux, h *handler.CouponHandler) {
	r.Mount("/coupons", h.Routes())
}
