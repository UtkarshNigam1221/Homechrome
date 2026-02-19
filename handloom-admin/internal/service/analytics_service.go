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

// Ensure interface compliance
var _ domain.AnalyticsService = (*AnalyticsService)(nil)
