package router

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"

	"github.com/handloom/admin/internal/handler/store"
	"github.com/handloom/admin/internal/middleware"
)

// NewStorePushRouter creates routes for storefront push subscription management.
// Routes are mounted at /api/v1/store/push/* (public, no required auth) so
// anonymous visitors can opt in. Rate-limited per IP because subscribe and test
// both trigger outbound work, and the test route sends a real notification.
func NewStorePushRouter(r *chi.Mux, h *store.PushHandler, customerAuth *middleware.CustomerAuth) {
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(20, time.Minute))
		// Optional, not required: anonymous visitors must still be able to opt
		// in, but a signed-in shopper's devices get linked to their orders.
		r.Use(customerAuth.OptionalCustomer)
		r.Mount("/api/v1/store/push", h.Routes())
	})
}
