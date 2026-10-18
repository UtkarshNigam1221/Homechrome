package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/handler"
	"github.com/handloom/admin/internal/middleware"
)

// NewPushRouter mounts the admin Web Push console at /admin/push/*, ADMIN-only
// like users and audit: a broadcast reaches every customer and cannot be recalled.
func NewPushRouter(r *chi.Mux, h *handler.PushHandler, authMiddleware *middleware.Auth) {
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.Use(authMiddleware.RequireRole(domain.UserRoleAdmin))
		r.Mount("/admin/push", h.Routes())
	})
}
