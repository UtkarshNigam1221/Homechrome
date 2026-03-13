package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/handler/store"
)

// NewStoreCatalogRouter creates routes for the store catalog service
// Routes are mounted at /api/v1/store/catalog/* to match API Gateway paths
func NewStoreCatalogRouter(r *chi.Mux, h *store.CatalogHandler) {
	r.Mount("/api/v1/store/catalog", h.Routes())
}
