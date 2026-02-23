package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/handler/store"
)

// NewStoreWebhooksRouter creates routes for the store webhooks service
// Routes are mounted at /api/v1/store/webhooks/* (public, signature-verified in service)
func NewStoreWebhooksRouter(r *chi.Mux, h *store.WebhookHandler) {
	r.Mount("/api/v1/store/webhooks", h.Routes())
}
