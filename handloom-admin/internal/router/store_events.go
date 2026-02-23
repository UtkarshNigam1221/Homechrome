package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/handler/store"
)

// NewStoreEventsRouter creates routes for the store events service
// Routes are mounted at /api/v1/store/events/* (public, no auth middleware)
func NewStoreEventsRouter(r *chi.Mux, h *store.EventsHandler) {
	r.Mount("/api/v1/store/events", h.Routes())
}
