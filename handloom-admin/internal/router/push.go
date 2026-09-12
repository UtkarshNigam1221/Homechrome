package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/handler"
	"github.com/handloom/admin/internal/middleware"
)

// NewPushRouter creates routes for the admin Web Push console, mounted at
// /admin/push/* to match the API Gateway resource paths.
//
// Broadcasting to every subscriber and reading subscriber records are both
// operator-only, so these routes sit inside their own authenticated group
// rather than on the public router the storefront push routes share.
func NewPushRouter(r *chi.Mux, h *handler.PushHandler, authMiddleware *middleware.Auth) {
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.Mount("/admin/push", h.Routes())
	})
}
