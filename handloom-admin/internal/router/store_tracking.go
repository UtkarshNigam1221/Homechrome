package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/handler/store"
)

// NewStoreTrackingRouter creates routes for the store tracking service
// Routes are mounted at /api/v1/store/track/* (public, no auth middleware)
func NewStoreTrackingRouter(r *chi.Mux, h *store.TrackingHandler) {
	r.Mount("/api/v1/store/track", h.Routes())
}
