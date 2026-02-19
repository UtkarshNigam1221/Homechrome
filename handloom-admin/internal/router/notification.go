package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/handler"
)

// NewNotificationRouter creates routes for the notification service
// Routes are mounted via h.Routes() which includes validation middleware
func NewNotificationRouter(r *chi.Mux, h *handler.NotificationHandler) {
	r.Mount("/notifications", h.Routes())
}
