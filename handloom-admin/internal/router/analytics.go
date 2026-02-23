package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/handler"
)

// NewAnalyticsRouter creates routes for the analytics service
func NewAnalyticsRouter(r *chi.Mux, h *handler.AnalyticsHandler) {
	r.Route("/analytics", func(r chi.Router) {
		r.Get("/dashboard", h.GetDashboardStats)
		r.Get("/sales", h.GetSalesAnalytics)
		r.Get("/top-products", h.GetTopProducts)
		r.Get("/top-categories", h.GetTopCategories)
		r.Get("/customers", h.GetCustomerAnalytics)
		r.Get("/inventory", h.GetInventoryAnalytics)
		r.Get("/funnel", h.GetFunnelAnalytics)
		r.Get("/engagement", h.GetEngagementAnalytics)
	})
}
