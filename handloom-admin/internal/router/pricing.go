package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/handler"
)

// NewPricingRouter creates routes for the pricing service
func NewPricingRouter(r *chi.Mux, h *handler.PricingHandler) {
	// Public pricing routes (no auth required)
	r.Route("/pricing", func(r chi.Router) {
		r.Mount("/", h.PublicRoutes())
	})

	// Admin pricing routes
	r.Route("/admin/pricing", func(r chi.Router) {
		r.Mount("/", h.Routes())
	})
}

// NewPricingAdminRouter creates admin-only pricing routes
func NewPricingAdminRouter(r *chi.Mux, h *handler.PricingHandler) {
	r.Route("/pricing", func(r chi.Router) {
		r.Mount("/", h.Routes())
	})
}

// NewPricingPublicRouter creates public pricing routes (no auth)
func NewPricingPublicRouter(r *chi.Mux, h *handler.PricingHandler) {
	r.Route("/pricing", func(r chi.Router) {
		r.Mount("/", h.PublicRoutes())
	})
}
