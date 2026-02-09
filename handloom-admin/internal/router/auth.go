package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/handler"
)

// NewAuthRouter creates routes for the auth service
// Routes are mounted at /admin/auth/* to match API Gateway paths
func NewAuthRouter(r *chi.Mux, h *handler.AuthHandler) {
	r.Mount("/admin/auth", h.Routes())
}
