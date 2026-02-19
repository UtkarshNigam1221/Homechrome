package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/handler"
	"github.com/handloom/admin/internal/middleware"
)

// NewAuditRouter creates routes for the audit service
func NewAuditRouter(r *chi.Mux, h *handler.AuditHandler, authMiddleware *middleware.Auth) {
	r.Route("/audit", func(r chi.Router) {
		// Audit logs require admin role
		r.Use(authMiddleware.RequireRole(domain.UserRoleAdmin))
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)
		r.Get("/entity/{type}/{id}", h.GetByEntity)
		r.Get("/user/{id}", h.GetByUser)
	})
}
