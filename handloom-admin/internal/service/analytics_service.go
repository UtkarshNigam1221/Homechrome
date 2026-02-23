// Package service implements the business logic layer
package service

import (
	"context"
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

	// Aggregate funnel step counts across all days
	stepTotals := map[string]int{
		"product_views": 0,
		"add_to_cart":   0,
		"checkout":      0,
		"payment":       0,
		"completed":     0,
	}
	stepOrder := []string{"product_views", "add_to_cart", "checkout", "payment", "completed"}

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
	if stepTotals["product_views"] > 0 {
		overallConversion = float64(stepTotals["completed"]) / float64(stepTotals["product_views"]) * 100
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

// GetEngagementAnalytics retrieves user engagement analytics aggregated across a date range
func (s *AnalyticsService) GetEngagementAnalytics(ctx context.Context, startDate, endDate string) (*domain.EngagementAnalytics, error) {
	rows, err := s.analyticsRepo.GetDailyAggregates(ctx, "ENGAGEMENT", startDate, endDate)
	if err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to get engagement aggregates")
		return nil, err
	}

	var totalSessions int
	var totalDuration int
	var totalBounces int
	deviceCounts := map[string]int{}
	pageViewCounts := map[string]int{}

	for _, row := range rows {
		if v, ok := row["sessions"]; ok {
			if val, ok := v.(float64); ok {
				totalSessions += int(val)
			}
		}
		if v, ok := row["total_duration"]; ok {
			if val, ok := v.(float64); ok {
				totalDuration += int(val)
			}
		}
		if v, ok := row["bounces"]; ok {
			if val, ok := v.(float64); ok {
				totalBounces += int(val)
			}
		}
		// Aggregate device counts
		if v, ok := row["devices"]; ok {
			if devices, ok := v.(map[string]interface{}); ok {
				for device, count := range devices {
					if c, ok := count.(float64); ok {
						deviceCounts[device] += int(c)
					}
				}
			}
		}
		// Aggregate page view counts
		if v, ok := row["pages"]; ok {
			if pages, ok := v.(map[string]interface{}); ok {
				for page, count := range pages {
					if c, ok := count.(float64); ok {
						pageViewCounts[page] += int(c)
					}
				}
			}
		}
	}

	// Compute averages
	var avgDuration int
	var bounceRate float64
	if totalSessions > 0 {
		avgDuration = totalDuration / totalSessions
		bounceRate = float64(totalBounces) / float64(totalSessions) * 100
	}

	// Device breakdown as percentages
	deviceBreakdown := map[string]float64{}
	totalDevices := 0
	for _, count := range deviceCounts {
		totalDevices += count
	}
	if totalDevices > 0 {
		for device, count := range deviceCounts {
			deviceBreakdown[device] = float64(count) / float64(totalDevices) * 100
		}
	}

	// Top pages sorted by views (top 10)
	topPages := make([]domain.PageStats, 0, len(pageViewCounts))
	for path, views := range pageViewCounts {
		topPages = append(topPages, domain.PageStats{Path: path, Views: views})
	}
	// Sort descending by views
	for i := 0; i < len(topPages); i++ {
		for j := i + 1; j < len(topPages); j++ {
			if topPages[j].Views > topPages[i].Views {
				topPages[i], topPages[j] = topPages[j], topPages[i]
			}
		}
	}
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

// Ensure interface compliance
var _ domain.AnalyticsService = (*AnalyticsService)(nil)
