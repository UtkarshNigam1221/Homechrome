package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/handler"
	"github.com/handloom/admin/internal/middleware"
)

// NewPushRouter creates routes for the admin Web Push console, mounted at
// /admin/push/* to match the API Gateway resource paths.
//
// ADMIN-only, like user and audit: a broadcast reaches every customer's lock
// screen and cannot be recalled, which is a wider blast radius than the rest of
// the console. The storefront's own push routes are public and live elsewhere.
func NewPushRouter(r *chi.Mux, h *handler.PushHandler, authMiddleware *middleware.Auth) {
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.Use(authMiddleware.RequireRole(domain.UserRoleAdmin))
		r.Mount("/admin/push", h.Routes())
	})
}
