// Package service implements the business logic layer
package service

import (
	"context"
	"sort"
	"time"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/logger"
)

// AnalyticsService implements domain.AnalyticsService
type AnalyticsService struct {
	analyticsRepo domain.AnalyticsRepository
	orderRepo     domain.OrderRepository
	productRepo   domain.ProductRepository
	inventoryRepo domain.InventoryRepository
	logger        *logger.Logger
}

// NewAnalyticsService creates a new AnalyticsService
func NewAnalyticsService(
	analyticsRepo domain.AnalyticsRepository,
	orderRepo domain.OrderRepository,
	productRepo domain.ProductRepository,
	inventoryRepo domain.InventoryRepository,
	logger *logger.Logger,
) *AnalyticsService {
	return &AnalyticsService{
		analyticsRepo: analyticsRepo,
		orderRepo:     orderRepo,
		productRepo:   productRepo,
		inventoryRepo: inventoryRepo,
		logger:        logger,
	}
}

// GetDashboardStats retrieves overall dashboard statistics
func (s *AnalyticsService) GetDashboardStats(ctx context.Context) (*domain.DashboardStats, error) {
	stats, err := s.analyticsRepo.GetDashboardStats(ctx)
	if err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to get dashboard stats")
		// Return empty stats instead of error for better UX
		return &domain.DashboardStats{}, nil
	}
	return stats, nil
}

// GetSalesAnalytics retrieves sales analytics for a period
func (s *AnalyticsService) GetSalesAnalytics(ctx context.Context, req domain.SalesAnalyticsRequest) (*domain.SalesAnalytics, error) {
	// Validate request
	if req.EndDate.Before(req.StartDate) {
		req.EndDate = time.Now()
	}
	if req.StartDate.IsZero() {
		req.StartDate = time.Now().AddDate(0, -1, 0) // Default to last month
	}

	// Default period
	if req.Period == "" {
		req.Period = "daily"
	}

	return s.analyticsRepo.GetSalesAnalytics(ctx, req.Period, req.StartDate, req.EndDate)
}

// GetTopProducts retrieves top selling products
func (s *AnalyticsService) GetTopProducts(ctx context.Context, limit int, startDate, endDate time.Time) ([]domain.TopProduct, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	return s.analyticsRepo.GetTopProducts(ctx, limit, startDate, endDate)
}

// GetTopCategories retrieves top performing categories
func (s *AnalyticsService) GetTopCategories(ctx context.Context, limit int, startDate, endDate time.Time) ([]domain.TopCategory, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	return s.analyticsRepo.GetTopCategories(ctx, limit, startDate, endDate)
}

// GetCustomerAnalytics retrieves customer analytics
func (s *AnalyticsService) GetCustomerAnalytics(ctx context.Context, startDate, endDate time.Time) (*domain.CustomerAnalytics, error) {
	return s.analyticsRepo.GetCustomerAnalytics(ctx, startDate, endDate)
}

// GetInventoryAnalytics retrieves inventory analytics
func (s *AnalyticsService) GetInventoryAnalytics(ctx context.Context) (*domain.InventoryAnalytics, error) {
	return s.analyticsRepo.GetInventoryAnalytics(ctx)
}

// GetFunnelAnalytics retrieves conversion funnel analytics aggregated across a date range
func (s *AnalyticsService) GetFunnelAnalytics(ctx context.Context, startDate, endDate string) (*domain.FunnelAnalytics, error) {
	rows, err := s.analyticsRepo.GetDailyAggregates(ctx, "FUNNEL", startDate, endDate)
	if err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to get funnel aggregates")
		return nil, err
	}

	// Aggregate funnel step counts across all days.
	// Field names must match the dynamodbav tags on funnelAggregate in analytics_aggregator.go.
	stepTotals := map[string]int{
		"page_views":        0,
		"product_views":     0,
		"add_to_carts":      0,
		"checkouts_started": 0,
		"orders_created":    0,
	}
	stepOrder := []string{"page_views", "product_views", "add_to_carts", "checkouts_started", "orders_created"}

	for _, row := range rows {
		for _, step := range stepOrder {
			if v, ok := row[step]; ok {
				switch val := v.(type) {
				case float64:
					stepTotals[step] += int(val)
				case int:
					stepTotals[step] += val
				}
			}
		}
	}

	// Build funnel steps with conversion rates
	steps := make([]domain.FunnelStep, 0, len(stepOrder))
	for i, step := range stepOrder {
		fs := domain.FunnelStep{
			Name:  step,
			Count: stepTotals[step],
		}
		if i > 0 && stepTotals[stepOrder[i-1]] > 0 {
			fs.Rate = float64(stepTotals[step]) / float64(stepTotals[stepOrder[i-1]]) * 100
		}
		steps = append(steps, fs)
	}

	var overallConversion float64
	if stepTotals["page_views"] > 0 {
		overallConversion = float64(stepTotals["orders_created"]) / float64(stepTotals["page_views"]) * 100
	}

	return &domain.FunnelAnalytics{
		Period: domain.DateRange{
			StartDate: startDate,
			EndDate:   endDate,
		},
		Steps:             steps,
		OverallConversion: overallConversion,
	}, nil
}

// GetEngagementAnalytics retrieves user engagement analytics aggregated across a date range.
// Field names must match the dynamodbav tags on engagementAggregate in analytics_aggregator.go.
// Device breakdown is sourced from CUSTOMERS aggregates (by_device_type field).
func (s *AnalyticsService) GetEngagementAnalytics(ctx context.Context, startDate, endDate string) (*domain.EngagementAnalytics, error) {
	rows, err := s.analyticsRepo.GetDailyAggregates(ctx, "ENGAGEMENT", startDate, endDate)
	if err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to get engagement aggregates")
		return nil, err
	}

	var totalSessions int
	var totalBounces int
	var durationSum float64
	var durationDays int
	pageViewCounts := map[string]int{}

	for _, row := range rows {
		daySessions := extractMapInt(row, "total_sessions")
		totalSessions += daySessions

		totalBounces += extractMapInt(row, "bounce_count")

		// avg_session_duration is already an average for that day, so we
		// compute a weighted average across days using total_sessions as weight.
		if avgDur := extractMapFloat64(row, "avg_session_duration"); avgDur > 0 && daySessions > 0 {
			durationSum += avgDur * float64(daySessions)
			durationDays += daySessions
		}

		// top_pages is a list of {path, views} structs written by the aggregator
		if v, ok := row["top_pages"]; ok {
			if items, ok := v.([]interface{}); ok {
				for _, item := range items {
					if m, ok := item.(map[string]interface{}); ok {
						path := extractMapString(m, "path")
						views := extractMapInt(m, "views")
						if path != "" {
							pageViewCounts[path] += views
						}
					}
				}
			}
		}
	}

	// Compute weighted average session duration across all days
	var avgDuration int
	var bounceRate float64
	if totalSessions > 0 {
		bounceRate = float64(totalBounces) / float64(totalSessions) * 100
	}
	if durationDays > 0 {
		avgDuration = int(durationSum / float64(durationDays))
	}

	// Device breakdown from CUSTOMERS aggregates (by_device_type field)
	deviceBreakdown := map[string]float64{}
	custRows, err := s.analyticsRepo.GetDailyAggregates(ctx, "CUSTOMERS", startDate, endDate)
	if err == nil {
		deviceCounts := map[string]int{}
		for _, row := range custRows {
			if v, ok := row["by_device_type"]; ok {
				if devices, ok := v.(map[string]interface{}); ok {
					for device, count := range devices {
						deviceCounts[device] += toInt(count)
					}
				}
			}
		}
		totalDevices := 0
		for _, count := range deviceCounts {
			totalDevices += count
		}
		if totalDevices > 0 {
			for device, count := range deviceCounts {
				deviceBreakdown[device] = float64(count) / float64(totalDevices) * 100
			}
		}
	}

	// Top pages sorted by views (top 10)
	topPages := make([]domain.PageStats, 0, len(pageViewCounts))
	for path, views := range pageViewCounts {
		topPages = append(topPages, domain.PageStats{Path: path, Views: views})
	}
	sort.Slice(topPages, func(i, j int) bool { return topPages[i].Views > topPages[j].Views })
	if len(topPages) > 10 {
		topPages = topPages[:10]
	}

	return &domain.EngagementAnalytics{
		Period: domain.DateRange{
			StartDate: startDate,
			EndDate:   endDate,
		},
		TotalSessions:      totalSessions,
		AvgSessionDuration: avgDuration,
		BounceRate:         bounceRate,
		DeviceBreakdown:    deviceBreakdown,
		TopPages:           topPages,
	}, nil
}

// ---------------------------------------------------------------------------
// Map extraction helpers for untyped DynamoDB aggregate data
// ---------------------------------------------------------------------------

func extractMapString(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func extractMapInt(m map[string]interface{}, key string) int {
	v, ok := m[key]
	if !ok {
		return 0
	}
	return toInt(v)
}

func extractMapFloat64(m map[string]interface{}, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}

// Ensure interface compliance
var _ domain.AnalyticsService = (*AnalyticsService)(nil)
