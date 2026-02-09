package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/handler"
)

// NewCatalogRouter creates routes for the catalog service (categories, designs, products)
// Routes are mounted at /admin/* to match API Gateway paths
func NewCatalogRouter(
	r *chi.Mux,
	categoryHandler *handler.CategoryHandler,
	designHandler *handler.DesignHandler,
	productHandler *handler.ProductHandler,
) {
	// Mount at /admin/* to match API Gateway resource paths
	r.Route("/admin/categories", func(r chi.Router) {
		r.Mount("/", categoryHandler.Routes())
	})

	r.Route("/admin/designs", func(r chi.Router) {
		r.Mount("/", designHandler.Routes())
	})

	r.Route("/admin/products", func(r chi.Router) {
		r.Mount("/", productHandler.Routes())
	})
}
