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

func TestArtisanService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockArtisanRepo := mocks.NewMockArtisanRepository(ctrl)
	log := logger.NewNoop()
	service := NewArtisanService(mockArtisanRepo, log)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		req := domain.CreateArtisanRequest{
			Name:           "Ravi Kumar",
			Phone:          "+919876543210",
			Location:       "Jaipur, Rajasthan",
			CraftTypes:     []string{"Block Printing", "Handloom"},
			CommissionRate: 15.0,
			Experience:     10,
		}

		mockArtisanRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, artisan *domain.Artisan) error {
				assert.Contains(t, artisan.ID, "artisan_")
				assert.Equal(t, "Ravi Kumar", artisan.Name)
				assert.Equal(t, "+919876543210", artisan.Phone)
				assert.Equal(t, "Jaipur, Rajasthan", artisan.Location)
				assert.Equal(t, domain.ArtisanStatusPending, artisan.Status)
				assert.Equal(t, 15.0, artisan.CommissionRate)
				assert.Equal(t, 10, artisan.Experience)
				assert.Equal(t, 0, artisan.ProductCount)
				assert.Equal(t, int64(0), artisan.TotalSales)
				assert.Equal(t, "admin_123", artisan.CreatedBy)
				return nil
			})

		artisan, err := service.Create(ctx, req, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, artisan)
		assert.Contains(t, artisan.ID, "artisan_")
		assert.Equal(t, domain.ArtisanStatusPending, artisan.Status)
	})

	t.Run("repo error on create", func(t *testing.T) {
		req := domain.CreateArtisanRequest{
			Name:       "Ravi Kumar",
			Phone:      "+919876543210",
			Location:   "Jaipur, Rajasthan",
			CraftTypes: []string{"Handloom"},
		}

		mockArtisanRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(errors.Internal("Database error"))

		artisan, err := service.Create(ctx, req, "admin_123")

		assert.Nil(t, artisan)
		require.Error(t, err)
	})
}

func TestArtisanService_GetByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockArtisanRepo := mocks.NewMockArtisanRepository(ctrl)
	log := logger.NewNoop()
	service := NewArtisanService(mockArtisanRepo, log)
	ctx := context.Background()

	t.Run("successful get by ID", func(t *testing.T) {
		expected := &domain.Artisan{
			ID:       "artisan_abc123",
			Name:     "Ravi Kumar",
			Phone:    "+919876543210",
			Location: "Jaipur, Rajasthan",
			Status:   domain.ArtisanStatusActive,
		}

		mockArtisanRepo.EXPECT().
			GetByID(ctx, "artisan_abc123").
			Return(expected, nil)

		artisan, err := service.GetByID(ctx, "artisan_abc123")

		require.NoError(t, err)
		assert.NotNil(t, artisan)
		assert.Equal(t, "artisan_abc123", artisan.ID)
		assert.Equal(t, "Ravi Kumar", artisan.Name)
	})

	t.Run("artisan not found", func(t *testing.T) {
		mockArtisanRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Artisan"))

		artisan, err := service.GetByID(ctx, "nonexistent")

		assert.Nil(t, artisan)
		require.Error(t, err)
	})
}

func TestArtisanService_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockArtisanRepo := mocks.NewMockArtisanRepository(ctrl)
	log := logger.NewNoop()
	service := NewArtisanService(mockArtisanRepo, log)
	ctx := context.Background()

	t.Run("successful full update", func(t *testing.T) {
		existing := &domain.Artisan{
			ID:             "artisan_abc123",
			Name:           "Ravi Kumar",
			Phone:          "+919876543210",
			Location:       "Jaipur, Rajasthan",
			CraftTypes:     []string{"Block Printing"},
			CommissionRate: 15.0,
			Experience:     10,
			Status:         domain.ArtisanStatusPending,
		}

		newName := "Ravi Sharma"
		newPhone := "+919876543211"
		newLocation := "Varanasi, UP"
		newExperience := 12
		newCommission := 18.0

		req := domain.UpdateArtisanRequest{
			Name:           &newName,
			Phone:          &newPhone,
			Location:       &newLocation,
			CraftTypes:     []string{"Handloom", "Block Printing"},
			Experience:     &newExperience,
			CommissionRate: &newCommission,
		}

		mockArtisanRepo.EXPECT().
			GetByID(ctx, "artisan_abc123").
			Return(existing, nil)

		mockArtisanRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, artisan *domain.Artisan) error {
				assert.Equal(t, "Ravi Sharma", artisan.Name)
				assert.Equal(t, "+919876543211", artisan.Phone)
				assert.Equal(t, "Varanasi, UP", artisan.Location)
				assert.Equal(t, 12, artisan.Experience)
				assert.Equal(t, 18.0, artisan.CommissionRate)
				assert.Equal(t, "admin_456", artisan.UpdatedBy)
				return nil
			})

		artisan, err := service.Update(ctx, "artisan_abc123", req, "admin_456")

		require.NoError(t, err)
		assert.NotNil(t, artisan)
		assert.Equal(t, "Ravi Sharma", artisan.Name)
	})

	t.Run("partial update - only name", func(t *testing.T) {
		existing := &domain.Artisan{
			ID:             "artisan_abc123",
			Name:           "Ravi Kumar",
			Phone:          "+919876543210",
			Location:       "Jaipur, Rajasthan",
			CommissionRate: 15.0,
			Status:         domain.ArtisanStatusActive,
		}

		newName := "Ravi Kumar Updated"
		req := domain.UpdateArtisanRequest{
			Name: &newName,
		}

		mockArtisanRepo.EXPECT().
			GetByID(ctx, "artisan_abc123").
			Return(existing, nil)

		mockArtisanRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, artisan *domain.Artisan) error {
				assert.Equal(t, "Ravi Kumar Updated", artisan.Name)
				assert.Equal(t, "+919876543210", artisan.Phone)
				assert.Equal(t, "Jaipur, Rajasthan", artisan.Location)
				assert.Equal(t, 15.0, artisan.CommissionRate)
				return nil
			})

		artisan, err := service.Update(ctx, "artisan_abc123", req, "admin_456")

		require.NoError(t, err)
		assert.NotNil(t, artisan)
	})

	t.Run("artisan not found", func(t *testing.T) {
		newName := "Test"
		req := domain.UpdateArtisanRequest{
			Name: &newName,
		}

		mockArtisanRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Artisan"))

		artisan, err := service.Update(ctx, "nonexistent", req, "admin_456")

		assert.Nil(t, artisan)
		require.Error(t, err)
	})
}

func TestArtisanService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockArtisanRepo := mocks.NewMockArtisanRepository(ctrl)
	log := logger.NewNoop()
	service := NewArtisanService(mockArtisanRepo, log)
	ctx := context.Background()

	t.Run("successful deletion", func(t *testing.T) {
		mockArtisanRepo.EXPECT().
			Delete(ctx, "artisan_abc123").
			Return(nil)

		err := service.Delete(ctx, "artisan_abc123")

		require.NoError(t, err)
	})

	t.Run("artisan not found", func(t *testing.T) {
		mockArtisanRepo.EXPECT().
			Delete(ctx, "nonexistent").
			Return(errors.NotFound("Artisan"))

		err := service.Delete(ctx, "nonexistent")

		require.Error(t, err)
	})
}

func TestArtisanService_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockArtisanRepo := mocks.NewMockArtisanRepository(ctrl)
	log := logger.NewNoop()
	service := NewArtisanService(mockArtisanRepo, log)
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		req := domain.ListArtisansRequest{
			PaginationRequest: domain.PaginationRequest{
				Limit: 20,
			},
		}

		expectedResponse := &domain.ListArtisansResponse{
			Artisans: []*domain.Artisan{
				{ID: "artisan_1", Name: "Ravi Kumar"},
				{ID: "artisan_2", Name: "Sita Devi"},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockArtisanRepo.EXPECT().
			List(ctx, req).
			Return(expectedResponse, nil)

		response, err := service.List(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Artisans, 2)
	})
}

func TestArtisanService_UpdateStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockArtisanRepo := mocks.NewMockArtisanRepository(ctrl)
	log := logger.NewNoop()
	service := NewArtisanService(mockArtisanRepo, log)
	ctx := context.Background()

	t.Run("successful status update", func(t *testing.T) {
		existing := &domain.Artisan{
			ID:     "artisan_abc123",
			Name:   "Ravi Kumar",
			Status: domain.ArtisanStatusPending,
		}

		mockArtisanRepo.EXPECT().
			GetByID(ctx, "artisan_abc123").
			Return(existing, nil)

		mockArtisanRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, artisan *domain.Artisan) error {
				assert.Equal(t, domain.ArtisanStatusActive, artisan.Status)
				assert.Equal(t, "admin_123", artisan.UpdatedBy)
				return nil
			})

		err := service.UpdateStatus(ctx, "artisan_abc123", domain.ArtisanStatusActive, "admin_123")

		require.NoError(t, err)
	})

	t.Run("artisan not found", func(t *testing.T) {
		mockArtisanRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Artisan"))

		err := service.UpdateStatus(ctx, "nonexistent", domain.ArtisanStatusActive, "admin_123")

		require.Error(t, err)
	})
}

func TestArtisanService_GetPayouts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockArtisanRepo := mocks.NewMockArtisanRepository(ctrl)
	log := logger.NewNoop()
	service := NewArtisanService(mockArtisanRepo, log)
	ctx := context.Background()

	t.Run("successful get payouts", func(t *testing.T) {
		pagination := domain.PaginationRequest{Limit: 20}

		expectedResponse := &domain.ListArtisanPayoutsResponse{
			Payouts: []*domain.ArtisanPayout{
				{ID: "payout_1", ArtisanID: "artisan_abc123", Amount: 500000},
				{ID: "payout_2", ArtisanID: "artisan_abc123", Amount: 300000},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockArtisanRepo.EXPECT().
			GetPayouts(ctx, "artisan_abc123", pagination).
			Return(expectedResponse, nil)

		response, err := service.GetPayouts(ctx, "artisan_abc123", pagination)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Payouts, 2)
	})
}

func TestArtisanService_CreatePayout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockArtisanRepo := mocks.NewMockArtisanRepository(ctrl)
	log := logger.NewNoop()
	service := NewArtisanService(mockArtisanRepo, log)
	ctx := context.Background()

	t.Run("successful payout creation", func(t *testing.T) {
		artisan := &domain.Artisan{
			ID:            "artisan_abc123",
			Name:          "Ravi Kumar",
			PendingPayout: 1000000, // 10000 INR
		}

		req := domain.CreatePayoutRequest{
			ArtisanID:     "artisan_abc123",
			Amount:        500000, // 5000 INR
			PaymentMethod: "BANK_TRANSFER",
			PeriodStart:   time.Now().AddDate(0, -1, 0),
			PeriodEnd:     time.Now(),
			OrderIDs:      []string{"order_1", "order_2"},
		}

		mockArtisanRepo.EXPECT().
			GetByID(ctx, "artisan_abc123").
			Return(artisan, nil)

		mockArtisanRepo.EXPECT().
			CreatePayout(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, payout *domain.ArtisanPayout) error {
				assert.Contains(t, payout.ID, "payout_")
				assert.Equal(t, "artisan_abc123", payout.ArtisanID)
				assert.Equal(t, int64(500000), payout.Amount)
				assert.Equal(t, "PENDING", payout.Status)
				assert.Equal(t, "BANK_TRANSFER", payout.PaymentMethod)
				assert.Equal(t, 2, payout.OrderCount)
				assert.Equal(t, "admin_123", payout.CreatedBy)
				return nil
			})

		mockArtisanRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, a *domain.Artisan) error {
				assert.Equal(t, int64(500000), a.PendingPayout) // 1000000 - 500000
				return nil
			})

		payout, err := service.CreatePayout(ctx, req, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, payout)
		assert.Contains(t, payout.ID, "payout_")
	})

	t.Run("artisan not found", func(t *testing.T) {
		req := domain.CreatePayoutRequest{
			ArtisanID:     "nonexistent",
			Amount:        500000,
			PaymentMethod: "BANK_TRANSFER",
			PeriodStart:   time.Now().AddDate(0, -1, 0),
			PeriodEnd:     time.Now(),
		}

		mockArtisanRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Artisan"))

		payout, err := service.CreatePayout(ctx, req, "admin_123")

		assert.Nil(t, payout)
		require.Error(t, err)
	})

	t.Run("pending payout goes to zero when amount exceeds", func(t *testing.T) {
		artisan := &domain.Artisan{
			ID:            "artisan_abc123",
			Name:          "Ravi Kumar",
			PendingPayout: 300000, // 3000 INR
		}

		req := domain.CreatePayoutRequest{
			ArtisanID:     "artisan_abc123",
			Amount:        500000, // 5000 INR > pending
			PaymentMethod: "BANK_TRANSFER",
			PeriodStart:   time.Now().AddDate(0, -1, 0),
			PeriodEnd:     time.Now(),
			OrderIDs:      []string{"order_1"},
		}

		mockArtisanRepo.EXPECT().
			GetByID(ctx, "artisan_abc123").
			Return(artisan, nil)

		mockArtisanRepo.EXPECT().
			CreatePayout(ctx, gomock.Any()).
			Return(nil)

		mockArtisanRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, a *domain.Artisan) error {
				assert.Equal(t, int64(0), a.PendingPayout) // Clamped to 0
				return nil
			})

		payout, err := service.CreatePayout(ctx, req, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, payout)
	})
}

func TestArtisanService_GetProducts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockArtisanRepo := mocks.NewMockArtisanRepository(ctrl)
	log := logger.NewNoop()
	service := NewArtisanService(mockArtisanRepo, log)
	ctx := context.Background()

	t.Run("successful get products", func(t *testing.T) {
		pagination := domain.PaginationRequest{Limit: 20}

		expectedResponse := &domain.ListProductsResponse{
			Products: []*domain.Product{
				{ID: "prod_1", Name: "Silk Saree"},
				{ID: "prod_2", Name: "Cotton Kurta"},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockArtisanRepo.EXPECT().
			GetProducts(ctx, "artisan_abc123", pagination).
			Return(expectedResponse, nil)

		response, err := service.GetProducts(ctx, "artisan_abc123", pagination)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Products, 2)
	})
}

func TestArtisanService_Search(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockArtisanRepo := mocks.NewMockArtisanRepository(ctrl)
	log := logger.NewNoop()
	service := NewArtisanService(mockArtisanRepo, log)
	ctx := context.Background()

	t.Run("successful search", func(t *testing.T) {
		pagination := domain.PaginationRequest{Limit: 20}

		expectedResponse := &domain.ListArtisansResponse{
			Artisans: []*domain.Artisan{
				{ID: "artisan_1", Name: "Ravi Kumar", Location: "Jaipur, Rajasthan"},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockArtisanRepo.EXPECT().
			Search(ctx, "Ravi", pagination).
			Return(expectedResponse, nil)

		response, err := service.Search(ctx, "Ravi", pagination)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Artisans, 1)
		assert.Equal(t, "Ravi Kumar", response.Artisans[0].Name)
	})
}
