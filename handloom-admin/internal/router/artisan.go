package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/handler"
)

// NewArtisanRouter creates routes for the artisan service
// Routes are mounted via h.Routes() which includes validation middleware
func NewArtisanRouter(r *chi.Mux, h *handler.ArtisanHandler) {
	r.Mount("/artisans", h.Routes())
}
