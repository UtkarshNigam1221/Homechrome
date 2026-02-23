package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/handler/store"
	"github.com/handloom/admin/internal/middleware"
)

// NewStoreCartRouter creates routes for the store cart service
// Routes are mounted at /api/v1/store/cart/* with customer auth middleware
func NewStoreCartRouter(r *chi.Mux, h *store.CartHandler, customerAuth *middleware.CustomerAuth) {
	r.Route("/api/v1/store/cart", func(r chi.Router) {
		r.Use(customerAuth.Authenticate)
		r.Mount("/", h.Routes())
	})
}
