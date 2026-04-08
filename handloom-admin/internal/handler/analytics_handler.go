package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/pkg/response"
)

// AnalyticsHandler handles analytics-related HTTP requests
type AnalyticsHandler struct {
	analyticsService *service.AnalyticsService
}

// NewAnalyticsHandler creates a new AnalyticsHandler
func NewAnalyticsHandler(analyticsService *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
	}
}

// GetDashboardStats retrieves overall dashboard statistics
// GET /admin/analytics/dashboard
func (h *AnalyticsHandler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.analyticsService.GetDashboardStats(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, stats)
}

// GetSalesAnalytics retrieves sales analytics for a period
// GET /admin/analytics/sales
func (h *AnalyticsHandler) GetSalesAnalytics(w http.ResponseWriter, r *http.Request) {
	req := domain.SalesAnalyticsRequest{
		Period: r.URL.Query().Get("period"),
	}

	if start := r.URL.Query().Get("start_date"); start != "" {
		if t, err := time.Parse(domain.DateLayout, start); err == nil {
			req.StartDate = t
		}
	}

	if end := r.URL.Query().Get("end_date"); end != "" {
		if t, err := time.Parse(domain.DateLayout, end); err == nil {
			req.EndDate = t
		}
	}

	// Default dates
	if req.StartDate.IsZero() {
		req.StartDate = time.Now().AddDate(0, -1, 0)
	}
	if req.EndDate.IsZero() {
		req.EndDate = time.Now()
	}

	analytics, err := h.analyticsService.GetSalesAnalytics(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, analytics)
}

// GetTopProducts retrieves top selling products
// GET /admin/analytics/top-products
func (h *AnalyticsHandler) GetTopProducts(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	var startDate, endDate time.Time
	if start := r.URL.Query().Get("start_date"); start != "" {
		if t, err := time.Parse(domain.DateLayout, start); err == nil {
			startDate = t
		}
	}
	if end := r.URL.Query().Get("end_date"); end != "" {
		if t, err := time.Parse(domain.DateLayout, end); err == nil {
			endDate = t
		}
	}

	// Default to last 30 days
	if startDate.IsZero() {
		startDate = time.Now().AddDate(0, 0, -30)
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}

	products, err := h.analyticsService.GetTopProducts(r.Context(), limit, startDate, endDate)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"products":   products,
		"start_date": startDate.Format(domain.DateLayout),
		"end_date":   endDate.Format(domain.DateLayout),
	})
}

// GetTopCategories retrieves top performing categories
// GET /admin/analytics/top-categories
func (h *AnalyticsHandler) GetTopCategories(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	var startDate, endDate time.Time
	if start := r.URL.Query().Get("start_date"); start != "" {
		if t, err := time.Parse(domain.DateLayout, start); err == nil {
			startDate = t
		}
	}
	if end := r.URL.Query().Get("end_date"); end != "" {
		if t, err := time.Parse(domain.DateLayout, end); err == nil {
			endDate = t
		}
	}

	// Default to last 30 days
	if startDate.IsZero() {
		startDate = time.Now().AddDate(0, 0, -30)
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}

	categories, err := h.analyticsService.GetTopCategories(r.Context(), limit, startDate, endDate)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"categories": categories,
		"start_date": startDate.Format(domain.DateLayout),
		"end_date":   endDate.Format(domain.DateLayout),
	})
}

// GetCustomerAnalytics retrieves customer analytics
// GET /admin/analytics/customers
func (h *AnalyticsHandler) GetCustomerAnalytics(w http.ResponseWriter, r *http.Request) {
	var startDate, endDate time.Time
	if start := r.URL.Query().Get("start_date"); start != "" {
		if t, err := time.Parse(domain.DateLayout, start); err == nil {
			startDate = t
		}
	}
	if end := r.URL.Query().Get("end_date"); end != "" {
		if t, err := time.Parse(domain.DateLayout, end); err == nil {
			endDate = t
		}
	}

	// Default to last 30 days
	if startDate.IsZero() {
		startDate = time.Now().AddDate(0, 0, -30)
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}

	analytics, err := h.analyticsService.GetCustomerAnalytics(r.Context(), startDate, endDate)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, analytics)
}

// GetInventoryAnalytics retrieves inventory analytics
// GET /admin/analytics/inventory
func (h *AnalyticsHandler) GetInventoryAnalytics(w http.ResponseWriter, r *http.Request) {
	analytics, err := h.analyticsService.GetInventoryAnalytics(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, analytics)
}

// GetFunnelAnalytics retrieves conversion funnel analytics
// GET /admin/analytics/funnel
func (h *AnalyticsHandler) GetFunnelAnalytics(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Default to last 7 days if not provided
	if startDate == "" || endDate == "" {
		endDate = time.Now().Format(domain.DateLayout)
		startDate = time.Now().AddDate(0, 0, -7).Format(domain.DateLayout)
	}

	result, err := h.analyticsService.GetFunnelAnalytics(r.Context(), startDate, endDate)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// GetEngagementAnalytics retrieves user engagement analytics
// GET /admin/analytics/engagement
func (h *AnalyticsHandler) GetEngagementAnalytics(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Default to last 7 days if not provided
	if startDate == "" || endDate == "" {
		endDate = time.Now().Format(domain.DateLayout)
		startDate = time.Now().AddDate(0, 0, -7).Format(domain.DateLayout)
	}

	result, err := h.analyticsService.GetEngagementAnalytics(r.Context(), startDate, endDate)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}
