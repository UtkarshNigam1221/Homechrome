package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/handler/store"
	"github.com/handloom/admin/internal/middleware"
)

// NewStoreCartRouter creates routes for the store cart service.
// Cart CRUD routes use OptionalCartAuth (guest + authenticated access).
func NewStoreCartRouter(r *chi.Mux, h *store.CartHandler, optionalCartAuth *middleware.OptionalCartAuth) {
	r.Route("/api/v1/store/cart", func(r chi.Router) {
		r.Use(optionalCartAuth.Resolve)
		r.Mount("/", h.CRUDRoutes())
	})
}
