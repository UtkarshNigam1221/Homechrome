package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/handler"
)

// NewAuthRouter creates routes for the auth service
// Routes are mounted at /admin/auth/* to match API Gateway paths
func NewAuthRouter(r *chi.Mux, h *handler.AuthHandler) {
	// Mount at /admin/auth to match API Gateway resource path
	r.Route("/admin/auth", func(r chi.Router) {
		r.Post("/login", h.Login)
		r.Post("/refresh", h.RefreshToken)
		r.Post("/logout", h.Logout)
		r.Post("/password/change", h.ChangePassword)
		r.Post("/password/reset-request", h.RequestPasswordReset)
		r.Post("/password/reset", h.ResetPassword)
	})
}
