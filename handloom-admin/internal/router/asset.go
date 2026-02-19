package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/handler"
)

// NewAssetRouter creates routes for the asset service
// Routes are mounted at /admin/assets to match API Gateway paths
func NewAssetRouter(r *chi.Mux, h *handler.AssetHandler) {
	r.Mount("/admin/assets", h.Routes())
}
