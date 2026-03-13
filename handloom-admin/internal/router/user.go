package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/handler"
	"github.com/handloom/admin/internal/middleware"
)

// NewUserRouter creates routes for the user service
// Routes are mounted at /admin/users/* to match API Gateway paths
func NewUserRouter(r *chi.Mux, h *handler.UserHandler, authMiddleware *middleware.Auth) {
	// Mount at /admin/users to match API Gateway resource path
	r.Route("/admin/users", func(r chi.Router) {
		// All user routes require admin role
		r.Use(authMiddleware.RequireRole(domain.UserRoleAdmin))
		r.Mount("/", h.Routes())
	})
}
