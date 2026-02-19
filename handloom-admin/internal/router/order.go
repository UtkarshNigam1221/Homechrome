package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/handler"
)

// NewOrderRouter creates routes for the order service
func NewOrderRouter(
	r *chi.Mux,
	orderHandler *handler.OrderHandler,
	customerHandler *handler.CustomerHandler,
) {
	// Orders
	r.Route("/orders", func(r chi.Router) {
		r.Mount("/", orderHandler.Routes())
	})

	// Customers — mounted via Routes() which includes validation middleware
	r.Mount("/customers", customerHandler.Routes())
}
