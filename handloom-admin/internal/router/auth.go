package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/handler"
)

// NewAuthRouter creates routes for the auth service
func NewAuthRouter(r *chi.Mux, h *handler.AuthHandler) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", h.Login)
		r.Post("/refresh", h.RefreshToken)
		r.Post("/logout", h.Logout)
		r.Post("/password/change", h.ChangePassword)
		r.Post("/password/reset-request", h.RequestPasswordReset)
		r.Post("/password/reset", h.ResetPassword)
	})
}
