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

func TestCouponService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCouponRepo := mocks.NewMockCouponRepository(ctrl)
	log := logger.NewNoop()
	service := NewCouponService(mockCouponRepo, log)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		req := domain.CreateCouponRequest{
			Code:          "summer20",
			Name:          "Summer Sale 20%",
			Description:   "20% off on all products",
			Type:          domain.CouponTypePercentage,
			Value:         2000, // 20% * 100
			MinOrderValue: 100000,
			MaxDiscount:   50000,
			UsageLimit:    100,
			UsagePerUser:  2,
			ValidFrom:     time.Now(),
			ValidUntil:    time.Now().AddDate(0, 3, 0),
		}

		mockCouponRepo.EXPECT().
			GetByCode(ctx, "SUMMER20").
			Return(nil, errors.NotFound("Coupon"))

		mockCouponRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, coupon *domain.Coupon) error {
				assert.Contains(t, coupon.ID, "coupon_")
				assert.Equal(t, "SUMMER20", coupon.Code)
				assert.Equal(t, "Summer Sale 20%", coupon.Name)
				assert.Equal(t, domain.CouponTypePercentage, coupon.Type)
				assert.Equal(t, int64(2000), coupon.Value)
				assert.Equal(t, domain.CouponStatusActive, coupon.Status)
				assert.Equal(t, 0, coupon.UsageCount)
				assert.Equal(t, "admin_123", coupon.CreatedBy)
				return nil
			})

		coupon, err := service.Create(ctx, req, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, coupon)
		assert.Equal(t, "SUMMER20", coupon.Code)
		assert.Equal(t, domain.CouponStatusActive, coupon.Status)
	})

	t.Run("duplicate code", func(t *testing.T) {
		req := domain.CreateCouponRequest{
			Code:       "EXISTING",
			Name:       "Existing Coupon",
			Type:       domain.CouponTypeFixed,
			Value:      10000,
			ValidFrom:  time.Now(),
			ValidUntil: time.Now().AddDate(0, 1, 0),
		}

		existing := &domain.Coupon{
			ID:   "coupon_existing",
			Code: "EXISTING",
		}

		mockCouponRepo.EXPECT().
			GetByCode(ctx, "EXISTING").
			Return(existing, nil)

		coupon, err := service.Create(ctx, req, "admin_123")

		assert.Nil(t, coupon)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("invalid dates - until before from", func(t *testing.T) {
		req := domain.CreateCouponRequest{
			Code:       "INVALID",
			Name:       "Invalid Dates",
			Type:       domain.CouponTypeFixed,
			Value:      10000,
			ValidFrom:  time.Now().AddDate(0, 1, 0),
			ValidUntil: time.Now(), // Before ValidFrom
		}

		mockCouponRepo.EXPECT().
			GetByCode(ctx, "INVALID").
			Return(nil, errors.NotFound("Coupon"))

		coupon, err := service.Create(ctx, req, "admin_123")

		assert.Nil(t, coupon)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Valid until date must be after valid from date")
	})
}

func TestCouponService_GetByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCouponRepo := mocks.NewMockCouponRepository(ctrl)
	log := logger.NewNoop()
	service := NewCouponService(mockCouponRepo, log)
	ctx := context.Background()

	t.Run("successful get", func(t *testing.T) {
		expected := &domain.Coupon{
			ID:   "coupon_abc123",
			Code: "SUMMER20",
			Name: "Summer Sale",
		}

		mockCouponRepo.EXPECT().
			GetByID(ctx, "coupon_abc123").
			Return(expected, nil)

		coupon, err := service.GetByID(ctx, "coupon_abc123")

		require.NoError(t, err)
		assert.Equal(t, "coupon_abc123", coupon.ID)
	})

	t.Run("not found", func(t *testing.T) {
		mockCouponRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Coupon"))

		coupon, err := service.GetByID(ctx, "nonexistent")

		assert.Nil(t, coupon)
		require.Error(t, err)
	})
}

func TestCouponService_GetByCode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCouponRepo := mocks.NewMockCouponRepository(ctrl)
	log := logger.NewNoop()
	service := NewCouponService(mockCouponRepo, log)
	ctx := context.Background()

	t.Run("successful get by code - uppercased", func(t *testing.T) {
		expected := &domain.Coupon{
			ID:   "coupon_abc123",
			Code: "SUMMER20",
		}

		mockCouponRepo.EXPECT().
			GetByCode(ctx, "SUMMER20").
			Return(expected, nil)

		coupon, err := service.GetByCode(ctx, "summer20")

		require.NoError(t, err)
		assert.Equal(t, "SUMMER20", coupon.Code)
	})
}

func TestCouponService_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCouponRepo := mocks.NewMockCouponRepository(ctrl)
	log := logger.NewNoop()
	service := NewCouponService(mockCouponRepo, log)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		existing := &domain.Coupon{
			ID:            "coupon_abc123",
			Code:          "SUMMER20",
			Name:          "Summer Sale",
			MinOrderValue: 100000,
			Status:        domain.CouponStatusActive,
		}

		newName := "Summer Sale Updated"
		newMinOrder := int64(200000)
		newStatus := domain.CouponStatusInactive
		req := domain.UpdateCouponRequest{
			Name:          &newName,
			MinOrderValue: &newMinOrder,
			Status:        &newStatus,
		}

		mockCouponRepo.EXPECT().
			GetByID(ctx, "coupon_abc123").
			Return(existing, nil)

		mockCouponRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, coupon *domain.Coupon) error {
				assert.Equal(t, "Summer Sale Updated", coupon.Name)
				assert.Equal(t, int64(200000), coupon.MinOrderValue)
				assert.Equal(t, domain.CouponStatusInactive, coupon.Status)
				assert.Equal(t, "admin_456", coupon.UpdatedBy)
				return nil
			})

		coupon, err := service.Update(ctx, "coupon_abc123", req, "admin_456")

		require.NoError(t, err)
		assert.NotNil(t, coupon)
		assert.Equal(t, "Summer Sale Updated", coupon.Name)
	})

	t.Run("coupon not found", func(t *testing.T) {
		newName := "Test"
		req := domain.UpdateCouponRequest{Name: &newName}

		mockCouponRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Coupon"))

		coupon, err := service.Update(ctx, "nonexistent", req, "admin_456")

		assert.Nil(t, coupon)
		require.Error(t, err)
	})
}

func TestCouponService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCouponRepo := mocks.NewMockCouponRepository(ctrl)
	log := logger.NewNoop()
	service := NewCouponService(mockCouponRepo, log)
	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		mockCouponRepo.EXPECT().
			Delete(ctx, "coupon_abc123").
			Return(nil)

		err := service.Delete(ctx, "coupon_abc123")
		require.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		mockCouponRepo.EXPECT().
			Delete(ctx, "nonexistent").
			Return(errors.NotFound("Coupon"))

		err := service.Delete(ctx, "nonexistent")
		require.Error(t, err)
	})
}

func TestCouponService_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCouponRepo := mocks.NewMockCouponRepository(ctrl)
	log := logger.NewNoop()
	service := NewCouponService(mockCouponRepo, log)
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		req := domain.ListCouponsRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 20},
		}

		expectedResponse := &domain.ListCouponsResponse{
			Coupons: []*domain.Coupon{
				{ID: "coupon_1", Code: "SUMMER20"},
				{ID: "coupon_2", Code: "WINTER30"},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockCouponRepo.EXPECT().
			List(ctx, req).
			Return(expectedResponse, nil)

		response, err := service.List(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Coupons, 2)
	})
}

func TestCouponService_Validate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCouponRepo := mocks.NewMockCouponRepository(ctrl)
	log := logger.NewNoop()
	service := NewCouponService(mockCouponRepo, log)
	ctx := context.Background()

	t.Run("coupon not found - returns invalid result", func(t *testing.T) {
		mockCouponRepo.EXPECT().
			GetByCode(ctx, "NOTFOUND").
			Return(nil, errors.NotFound("Coupon"))

		result, err := service.Validate(ctx, "notfound", 500000, "cust_123", nil)

		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, "Coupon not found", result.ErrorMessage)
	})

	t.Run("inactive coupon", func(t *testing.T) {
		coupon := &domain.Coupon{
			ID:     "coupon_1",
			Code:   "INACTIVE",
			Status: domain.CouponStatusInactive,
		}

		mockCouponRepo.EXPECT().
			GetByCode(ctx, "INACTIVE").
			Return(coupon, nil)

		result, err := service.Validate(ctx, "INACTIVE", 500000, "cust_123", nil)

		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, "Coupon is not active", result.ErrorMessage)
	})

	t.Run("not yet valid - future ValidFrom", func(t *testing.T) {
		coupon := &domain.Coupon{
			ID:         "coupon_1",
			Code:       "FUTURE",
			Status:     domain.CouponStatusActive,
			ValidFrom:  time.Now().Add(24 * time.Hour), // tomorrow
			ValidUntil: time.Now().Add(48 * time.Hour),
		}

		mockCouponRepo.EXPECT().
			GetByCode(ctx, "FUTURE").
			Return(coupon, nil)

		result, err := service.Validate(ctx, "FUTURE", 500000, "cust_123", nil)

		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, "Coupon is not yet valid", result.ErrorMessage)
	})

	t.Run("expired coupon", func(t *testing.T) {
		coupon := &domain.Coupon{
			ID:         "coupon_1",
			Code:       "EXPIRED",
			Status:     domain.CouponStatusActive,
			ValidFrom:  time.Now().Add(-48 * time.Hour),
			ValidUntil: time.Now().Add(-24 * time.Hour), // yesterday
		}

		mockCouponRepo.EXPECT().
			GetByCode(ctx, "EXPIRED").
			Return(coupon, nil)

		result, err := service.Validate(ctx, "EXPIRED", 500000, "cust_123", nil)

		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, "Coupon has expired", result.ErrorMessage)
	})

	t.Run("below minimum order value", func(t *testing.T) {
		coupon := &domain.Coupon{
			ID:            "coupon_1",
			Code:          "MIN500",
			Status:        domain.CouponStatusActive,
			ValidFrom:     time.Now().Add(-24 * time.Hour),
			ValidUntil:    time.Now().Add(24 * time.Hour),
			MinOrderValue: 500000, // 5000 INR
		}

		mockCouponRepo.EXPECT().
			GetByCode(ctx, "MIN500").
			Return(coupon, nil)

		result, err := service.Validate(ctx, "MIN500", 300000, "cust_123", nil) // 3000 INR < 5000 INR

		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, "Order total does not meet minimum requirement", result.ErrorMessage)
	})

	t.Run("usage limit reached", func(t *testing.T) {
		coupon := &domain.Coupon{
			ID:            "coupon_1",
			Code:          "LIMITED",
			Status:        domain.CouponStatusActive,
			ValidFrom:     time.Now().Add(-24 * time.Hour),
			ValidUntil:    time.Now().Add(24 * time.Hour),
			MinOrderValue: 0,
			UsageLimit:    5,
			UsageCount:    5, // Already used 5 times
		}

		mockCouponRepo.EXPECT().
			GetByCode(ctx, "LIMITED").
			Return(coupon, nil)

		result, err := service.Validate(ctx, "LIMITED", 500000, "cust_123", nil)

		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, "Coupon usage limit reached", result.ErrorMessage)
	})

	t.Run("per-user limit reached", func(t *testing.T) {
		coupon := &domain.Coupon{
			ID:            "coupon_1",
			Code:          "PERUSER",
			Status:        domain.CouponStatusActive,
			ValidFrom:     time.Now().Add(-24 * time.Hour),
			ValidUntil:    time.Now().Add(24 * time.Hour),
			MinOrderValue: 0,
			UsageLimit:    0, // unlimited
			UsagePerUser:  1,
			UsageCount:    3,
		}

		mockCouponRepo.EXPECT().
			GetByCode(ctx, "PERUSER").
			Return(coupon, nil)

		mockCouponRepo.EXPECT().
			GetUserUsageCount(ctx, "coupon_1", "cust_123").
			Return(1, nil) // Already used once

		result, err := service.Validate(ctx, "PERUSER", 500000, "cust_123", nil)

		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.ErrorMessage, "maximum number of times")
	})

	t.Run("valid percentage coupon - correct discount", func(t *testing.T) {
		coupon := &domain.Coupon{
			ID:            "coupon_1",
			Code:          "PERCENT10",
			Status:        domain.CouponStatusActive,
			Type:          domain.CouponTypePercentage,
			Value:         1000, // 10% * 100
			ValidFrom:     time.Now().Add(-24 * time.Hour),
			ValidUntil:    time.Now().Add(24 * time.Hour),
			MinOrderValue: 0,
			UsageLimit:    0,
			UsagePerUser:  0,
		}

		mockCouponRepo.EXPECT().
			GetByCode(ctx, "PERCENT10").
			Return(coupon, nil)

		result, err := service.Validate(ctx, "PERCENT10", 1000000, "cust_123", nil) // 10000 INR

		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, "coupon_1", result.CouponID)
		assert.Equal(t, "PERCENT10", result.Code)
		// 1000000 * 1000 / 10000 = 100000 (1000 INR)
		assert.Equal(t, int64(100000), result.DiscountAmount)
	})

	t.Run("percentage coupon with MaxDiscount cap", func(t *testing.T) {
		coupon := &domain.Coupon{
			ID:            "coupon_1",
			Code:          "PERCENT50",
			Status:        domain.CouponStatusActive,
			Type:          domain.CouponTypePercentage,
			Value:         5000, // 50% * 100
			MaxDiscount:   200000, // Max 2000 INR
			ValidFrom:     time.Now().Add(-24 * time.Hour),
			ValidUntil:    time.Now().Add(24 * time.Hour),
			MinOrderValue: 0,
			UsageLimit:    0,
			UsagePerUser:  0,
		}

		mockCouponRepo.EXPECT().
			GetByCode(ctx, "PERCENT50").
			Return(coupon, nil)

		result, err := service.Validate(ctx, "PERCENT50", 1000000, "cust_123", nil) // 10000 INR

		require.NoError(t, err)
		assert.True(t, result.Valid)
		// 1000000 * 5000 / 10000 = 500000, but capped at 200000
		assert.Equal(t, int64(200000), result.DiscountAmount)
	})

	t.Run("valid fixed coupon", func(t *testing.T) {
		coupon := &domain.Coupon{
			ID:            "coupon_1",
			Code:          "FLAT500",
			Status:        domain.CouponStatusActive,
			Type:          domain.CouponTypeFixed,
			Value:         50000, // 500 INR
			ValidFrom:     time.Now().Add(-24 * time.Hour),
			ValidUntil:    time.Now().Add(24 * time.Hour),
			MinOrderValue: 0,
			UsageLimit:    0,
			UsagePerUser:  0,
		}

		mockCouponRepo.EXPECT().
			GetByCode(ctx, "FLAT500").
			Return(coupon, nil)

		result, err := service.Validate(ctx, "FLAT500", 1000000, "cust_123", nil)

		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Equal(t, int64(50000), result.DiscountAmount)
	})

	t.Run("fixed coupon where value exceeds order total", func(t *testing.T) {
		coupon := &domain.Coupon{
			ID:            "coupon_1",
			Code:          "FLAT5000",
			Status:        domain.CouponStatusActive,
			Type:          domain.CouponTypeFixed,
			Value:         500000, // 5000 INR
			ValidFrom:     time.Now().Add(-24 * time.Hour),
			ValidUntil:    time.Now().Add(24 * time.Hour),
			MinOrderValue: 0,
			UsageLimit:    0,
			UsagePerUser:  0,
		}

		mockCouponRepo.EXPECT().
			GetByCode(ctx, "FLAT5000").
			Return(coupon, nil)

		result, err := service.Validate(ctx, "FLAT5000", 300000, "cust_123", nil) // 3000 INR < 5000 INR discount

		require.NoError(t, err)
		assert.True(t, result.Valid)
		// Discount capped at order total
		assert.Equal(t, int64(300000), result.DiscountAmount)
	})
}

func TestCouponService_Apply(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCouponRepo := mocks.NewMockCouponRepository(ctrl)
	log := logger.NewNoop()
	service := NewCouponService(mockCouponRepo, log)
	ctx := context.Background()

	t.Run("successful apply", func(t *testing.T) {
		coupon := &domain.Coupon{
			ID:         "coupon_1",
			Code:       "SUMMER20",
			UsageCount: 5,
		}

		mockCouponRepo.EXPECT().
			GetByID(ctx, "coupon_1").
			Return(coupon, nil)

		mockCouponRepo.EXPECT().
			RecordUsage(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, usage *domain.CouponUsage) error {
				assert.Equal(t, "coupon_1", usage.CouponID)
				assert.Equal(t, "SUMMER20", usage.CouponCode)
				assert.Equal(t, "order_123", usage.OrderID)
				assert.Equal(t, "cust_123", usage.CustomerID)
				assert.Equal(t, int64(100000), usage.Discount)
				return nil
			})

		mockCouponRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, c *domain.Coupon) error {
				assert.Equal(t, 6, c.UsageCount) // Incremented
				return nil
			})

		err := service.Apply(ctx, "coupon_1", "order_123", "cust_123", 100000)

		require.NoError(t, err)
	})

	t.Run("coupon not found for apply", func(t *testing.T) {
		mockCouponRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Coupon"))

		err := service.Apply(ctx, "nonexistent", "order_123", "cust_123", 100000)

		require.Error(t, err)
	})
}
