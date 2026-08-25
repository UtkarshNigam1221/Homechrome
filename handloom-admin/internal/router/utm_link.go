package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/handler"
)

// NewUTMLinkRouter creates routes for the UTM link service.
// Mounted at /admin/utm-links to match the API Gateway resource path: the
// integration is a proxy, so the Lambda receives the full request path.
func NewUTMLinkRouter(r *chi.Mux, h *handler.UTMLinkHandler) {
	r.Route("/admin/utm-links", func(r chi.Router) {
		r.Mount("/", h.Routes())
	})
}
