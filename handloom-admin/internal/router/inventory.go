package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/handler"
)

// NewInventoryRouter creates routes for the inventory service
func NewInventoryRouter(r *chi.Mux, h *handler.InventoryHandler) {
	r.Route("/inventory", func(r chi.Router) {
		r.Mount("/", h.Routes())
	})
}
