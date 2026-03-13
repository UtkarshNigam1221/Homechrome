package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/handler/store"
	"github.com/handloom/admin/internal/middleware"
)

// NewStoreOrdersRouter creates routes for the store orders service
// Routes are mounted at /api/v1/store/orders/* with customer auth middleware
func NewStoreOrdersRouter(r *chi.Mux, h *store.OrderHandler, customerAuth *middleware.CustomerAuth) {
	r.Route("/api/v1/store/orders", func(r chi.Router) {
		r.Use(customerAuth.Authenticate)
		r.Mount("/", h.Routes())
	})
}
