package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/handler/store"
	"github.com/handloom/admin/internal/middleware"
)

// NewStoreProfileRouter creates routes for the store profile service
// Routes are mounted at /api/v1/store/me/* with customer auth middleware
func NewStoreProfileRouter(r *chi.Mux, h *store.ProfileHandler, customerAuth *middleware.CustomerAuth) {
	r.Route("/api/v1/store/me", func(r chi.Router) {
		r.Use(customerAuth.Authenticate)
		r.Mount("/", h.Routes())
	})
}
