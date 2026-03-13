package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/handler/store"
	"github.com/handloom/admin/internal/middleware"
)

// NewStoreCheckoutRouter creates routes for the store checkout service
// Routes are mounted at /api/v1/store/checkout/* with customer auth middleware
func NewStoreCheckoutRouter(r *chi.Mux, h *store.CheckoutHandler, customerAuth *middleware.CustomerAuth) {
	r.Route("/api/v1/store/checkout", func(r chi.Router) {
		r.Use(customerAuth.Authenticate)
		r.Mount("/", h.Routes())
	})
}
