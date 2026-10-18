package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/handler/store"
	"github.com/handloom/admin/internal/middleware"
)

// NewStorePushRouter mounts the public storefront push routes at
// /api/v1/store/push/*; the handler applies its own per-IP limiter.
func NewStorePushRouter(r *chi.Mux, h *store.PushHandler, customerAuth *middleware.CustomerAuth) {
	r.Group(func(r chi.Router) {
		// Optional, not required: anonymous visitors must still be able to opt
		// in, but a signed-in shopper's devices get linked to their orders.
		r.Use(customerAuth.OptionalCustomer)
		r.Mount("/api/v1/store/push", h.Routes())
	})
}
