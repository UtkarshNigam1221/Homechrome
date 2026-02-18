package service

import (
	"context"
	"testing"
	"time"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAnalyticsService_GetDashboardStats(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAnalyticsRepo := mocks.NewMockAnalyticsRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	log := logger.NewNoop()

	service := NewAnalyticsService(mockAnalyticsRepo, mockOrderRepo, mockProductRepo, mockInventoryRepo, log)
	ctx := context.Background()

	t.Run("successful get dashboard stats", func(t *testing.T) {
		expectedStats := &domain.DashboardStats{
			TodayOrders:    15,
			TodayRevenue:   1500000,
			WeekOrders:     80,
			WeekRevenue:    8000000,
			MonthOrders:    320,
			MonthRevenue:   32000000,
			TotalOrders:    5000,
			TotalRevenue:   500000000,
			TotalCustomers: 1200,
			TotalProducts:  350,
			LowStockCount:  12,
			OutOfStockCount: 3,
		}

		mockAnalyticsRepo.EXPECT().
			GetDashboardStats(ctx).
			Return(expectedStats, nil)

		stats, err := service.GetDashboardStats(ctx)

		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, 15, stats.TodayOrders)
		assert.Equal(t, int64(1500000), stats.TodayRevenue)
		assert.Equal(t, 5000, stats.TotalOrders)
	})

	t.Run("repo error returns empty stats - graceful degradation", func(t *testing.T) {
		mockAnalyticsRepo.EXPECT().
			GetDashboardStats(ctx).
			Return(nil, errors.Internal("Database error"))

		stats, err := service.GetDashboardStats(ctx)

		// Should NOT return an error - graceful degradation
		require.NoError(t, err)
		assert.NotNil(t, stats)
		// Stats should be empty/zero
		assert.Equal(t, 0, stats.TodayOrders)
		assert.Equal(t, int64(0), stats.TodayRevenue)
		assert.Equal(t, 0, stats.TotalOrders)
	})
}

func TestAnalyticsService_GetSalesAnalytics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAnalyticsRepo := mocks.NewMockAnalyticsRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	log := logger.NewNoop()

	service := NewAnalyticsService(mockAnalyticsRepo, mockOrderRepo, mockProductRepo, mockInventoryRepo, log)
	ctx := context.Background()

	t.Run("with valid request", func(t *testing.T) {
		startDate := time.Now().AddDate(0, -1, 0)
		endDate := time.Now()

		req := domain.SalesAnalyticsRequest{
			Period:    "weekly",
			StartDate: startDate,
			EndDate:   endDate,
		}

		expectedAnalytics := &domain.SalesAnalytics{
			Period:            "weekly",
			StartDate:         startDate,
			EndDate:           endDate,
			TotalSales:        5000000,
			TotalOrders:       120,
			AverageOrderValue: 41666,
		}

		mockAnalyticsRepo.EXPECT().
			GetSalesAnalytics(ctx, "weekly", startDate, endDate).
			Return(expectedAnalytics, nil)

		result, err := service.GetSalesAnalytics(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "weekly", result.Period)
		assert.Equal(t, 120, result.TotalOrders)
	})

	t.Run("defaults applied when period is empty", func(t *testing.T) {
		startDate := time.Now().AddDate(0, -1, 0)
		endDate := time.Now()

		req := domain.SalesAnalyticsRequest{
			Period:    "", // empty - should default to "daily"
			StartDate: startDate,
			EndDate:   endDate,
		}

		mockAnalyticsRepo.EXPECT().
			GetSalesAnalytics(ctx, "daily", startDate, endDate).
			Return(&domain.SalesAnalytics{Period: "daily"}, nil)

		result, err := service.GetSalesAnalytics(ctx, req)

		require.NoError(t, err)
		assert.Equal(t, "daily", result.Period)
	})

	t.Run("end date before start date - end date defaults to now", func(t *testing.T) {
		startDate := time.Now().AddDate(0, -1, 0)
		endDate := time.Now().AddDate(0, -2, 0) // Before start

		req := domain.SalesAnalyticsRequest{
			Period:    "daily",
			StartDate: startDate,
			EndDate:   endDate, // Invalid
		}

		mockAnalyticsRepo.EXPECT().
			GetSalesAnalytics(ctx, "daily", startDate, gomock.Any()).
			Return(&domain.SalesAnalytics{Period: "daily"}, nil)

		result, err := service.GetSalesAnalytics(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("zero start date defaults to last month", func(t *testing.T) {
		req := domain.SalesAnalyticsRequest{
			Period:    "daily",
			StartDate: time.Time{}, // zero value
			EndDate:   time.Now(),
		}

		// EndDate < StartDate (zero) so EndDate gets set to now
		// Then StartDate zero -> defaults to last month
		mockAnalyticsRepo.EXPECT().
			GetSalesAnalytics(ctx, "daily", gomock.Any(), gomock.Any()).
			Return(&domain.SalesAnalytics{Period: "daily"}, nil)

		result, err := service.GetSalesAnalytics(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestAnalyticsService_GetTopProducts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAnalyticsRepo := mocks.NewMockAnalyticsRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	log := logger.NewNoop()

	service := NewAnalyticsService(mockAnalyticsRepo, mockOrderRepo, mockProductRepo, mockInventoryRepo, log)
	ctx := context.Background()

	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()

	t.Run("valid limit", func(t *testing.T) {
		expectedProducts := []domain.TopProduct{
			{ProductID: "prod_1", ProductName: "Silk Saree", UnitsSold: 100, Revenue: 5000000},
			{ProductID: "prod_2", ProductName: "Cotton Kurta", UnitsSold: 80, Revenue: 3200000},
		}

		mockAnalyticsRepo.EXPECT().
			GetTopProducts(ctx, 5, startDate, endDate).
			Return(expectedProducts, nil)

		result, err := service.GetTopProducts(ctx, 5, startDate, endDate)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "prod_1", result[0].ProductID)
	})

	t.Run("zero limit defaults to 10", func(t *testing.T) {
		mockAnalyticsRepo.EXPECT().
			GetTopProducts(ctx, 10, startDate, endDate).
			Return([]domain.TopProduct{}, nil)

		result, err := service.GetTopProducts(ctx, 0, startDate, endDate)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("negative limit defaults to 10", func(t *testing.T) {
		mockAnalyticsRepo.EXPECT().
			GetTopProducts(ctx, 10, startDate, endDate).
			Return([]domain.TopProduct{}, nil)

		result, err := service.GetTopProducts(ctx, -5, startDate, endDate)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("limit over 50 clamped to 50", func(t *testing.T) {
		mockAnalyticsRepo.EXPECT().
			GetTopProducts(ctx, 50, startDate, endDate).
			Return([]domain.TopProduct{}, nil)

		result, err := service.GetTopProducts(ctx, 100, startDate, endDate)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestAnalyticsService_GetTopCategories(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAnalyticsRepo := mocks.NewMockAnalyticsRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	log := logger.NewNoop()

	service := NewAnalyticsService(mockAnalyticsRepo, mockOrderRepo, mockProductRepo, mockInventoryRepo, log)
	ctx := context.Background()

	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()

	t.Run("valid limit", func(t *testing.T) {
		expectedCategories := []domain.TopCategory{
			{CategoryID: "cat_1", CategoryName: "Sarees", UnitsSold: 200, Revenue: 10000000},
		}

		mockAnalyticsRepo.EXPECT().
			GetTopCategories(ctx, 5, startDate, endDate).
			Return(expectedCategories, nil)

		result, err := service.GetTopCategories(ctx, 5, startDate, endDate)

		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Sarees", result[0].CategoryName)
	})

	t.Run("zero limit defaults to 10", func(t *testing.T) {
		mockAnalyticsRepo.EXPECT().
			GetTopCategories(ctx, 10, startDate, endDate).
			Return([]domain.TopCategory{}, nil)

		result, err := service.GetTopCategories(ctx, 0, startDate, endDate)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("limit over 50 clamped to 50", func(t *testing.T) {
		mockAnalyticsRepo.EXPECT().
			GetTopCategories(ctx, 50, startDate, endDate).
			Return([]domain.TopCategory{}, nil)

		result, err := service.GetTopCategories(ctx, 100, startDate, endDate)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestAnalyticsService_GetCustomerAnalytics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAnalyticsRepo := mocks.NewMockAnalyticsRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	log := logger.NewNoop()

	service := NewAnalyticsService(mockAnalyticsRepo, mockOrderRepo, mockProductRepo, mockInventoryRepo, log)
	ctx := context.Background()

	t.Run("successful get customer analytics", func(t *testing.T) {
		startDate := time.Now().AddDate(0, -1, 0)
		endDate := time.Now()

		expectedAnalytics := &domain.CustomerAnalytics{
			TotalCustomers:           1200,
			NewCustomers:             150,
			ReturningCustomers:       800,
			AverageOrdersPerCustomer: 3.5,
		}

		mockAnalyticsRepo.EXPECT().
			GetCustomerAnalytics(ctx, startDate, endDate).
			Return(expectedAnalytics, nil)

		result, err := service.GetCustomerAnalytics(ctx, startDate, endDate)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1200, result.TotalCustomers)
		assert.Equal(t, 150, result.NewCustomers)
	})

	t.Run("repo error", func(t *testing.T) {
		startDate := time.Now().AddDate(0, -1, 0)
		endDate := time.Now()

		mockAnalyticsRepo.EXPECT().
			GetCustomerAnalytics(ctx, startDate, endDate).
			Return(nil, errors.Internal("Database error"))

		result, err := service.GetCustomerAnalytics(ctx, startDate, endDate)

		assert.Nil(t, result)
		require.Error(t, err)
	})
}

func TestAnalyticsService_GetInventoryAnalytics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAnalyticsRepo := mocks.NewMockAnalyticsRepository(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	log := logger.NewNoop()

	service := NewAnalyticsService(mockAnalyticsRepo, mockOrderRepo, mockProductRepo, mockInventoryRepo, log)
	ctx := context.Background()

	t.Run("successful get inventory analytics", func(t *testing.T) {
		expectedAnalytics := &domain.InventoryAnalytics{
			TotalProducts:       350,
			TotalInventoryValue: 50000000,
			LowStockCount:       12,
			OutOfStockCount:     3,
		}

		mockAnalyticsRepo.EXPECT().
			GetInventoryAnalytics(ctx).
			Return(expectedAnalytics, nil)

		result, err := service.GetInventoryAnalytics(ctx)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 350, result.TotalProducts)
		assert.Equal(t, 12, result.LowStockCount)
	})

	t.Run("repo error", func(t *testing.T) {
		mockAnalyticsRepo.EXPECT().
			GetInventoryAnalytics(ctx).
			Return(nil, errors.Internal("Database error"))

		result, err := service.GetInventoryAnalytics(ctx)

		assert.Nil(t, result)
		require.Error(t, err)
	})
}
