package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/handler/store"
)

// NewStorePushRouter mounts the public storefront push routes at
// /api/v1/store/push/*; the handler applies its own per-IP limiter.
func NewStorePushRouter(r *chi.Mux, h *store.PushHandler) {
	r.Mount("/api/v1/store/push", h.Routes())
}
