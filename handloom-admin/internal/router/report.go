package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/handler"
)

// NewReportRouter creates routes for the report service
// Routes are mounted via h.Routes() which includes validation middleware
func NewReportRouter(r *chi.Mux, h *handler.ReportHandler) {
	r.Mount("/reports", h.Routes())
}
