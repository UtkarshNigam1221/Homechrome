package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/handler/store"
	"github.com/handloom/admin/internal/middleware"
)

// NewStoreAuthRouter creates routes for the store auth service
// Routes are mounted at /api/v1/store/auth/* to match API Gateway paths
func NewStoreAuthRouter(r *chi.Mux, h *store.AuthHandler, customerAuth *middleware.CustomerAuth) {
	r.Mount("/api/v1/store/auth", h.Routes(customerAuth.Authenticate))
}
