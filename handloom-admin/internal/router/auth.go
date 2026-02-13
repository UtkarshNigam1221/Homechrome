package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/handler"
	"github.com/handloom/admin/internal/middleware"
)

// NewAuthRouter creates routes for the auth service
// Routes are mounted at /admin/auth/* to match API Gateway paths
func NewAuthRouter(r *chi.Mux, h *handler.AuthHandler, authMiddleware *middleware.Auth) {
	r.Mount("/admin/auth", h.Routes(authMiddleware.Authenticate))
}
