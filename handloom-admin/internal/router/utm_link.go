package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/handler"
)

// NewUTMLinkRouter creates routes for the UTM link service
// Routes are mounted via h.Routes() which includes validation middleware
func NewUTMLinkRouter(r *chi.Mux, h *handler.UTMLinkHandler) {
	r.Mount("/utm-links", h.Routes())
}
