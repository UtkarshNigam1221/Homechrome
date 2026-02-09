package service

import (
	"context"
	"testing"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDesignService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDesignRepo := mocks.NewMockDesignRepository(ctrl)
	mockCategoryRepo := mocks.NewMockCategoryRepository(ctrl)
	log := logger.NewNoop()
	service := NewDesignService(mockDesignRepo, mockCategoryRepo, log)
	ctx := context.Background()

	t.Run("successful design creation", func(t *testing.T) {
		req := domain.CreateDesignRequest{
			Name:        "Ikkat Diamond Pattern",
			CategoryID:  "cat_123",
			Description: "Traditional Ikkat weaving pattern with diamond motifs",
			Images: []domain.ProductImage{
				{URL: "https://example.com/image1.jpg", IsPrimary: true},
			},
			Attributes: []domain.DesignAttribute{
				{Name: "pattern_type", Values: []string{"geometric"}},
				{Name: "origin", Values: []string{"Andhra Pradesh"}},
			},
		}

		category := &domain.Category{
			ID:   "cat_123",
			Name: "Ikkat Sarees",
		}

		mockCategoryRepo.EXPECT().
			GetByID(ctx, req.CategoryID).
			Return(category, nil)

		mockDesignRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, design *domain.Design) error {
				assert.Equal(t, req.Name, design.Name)
				assert.Equal(t, "ikkat-diamond-pattern", design.Slug)
				assert.Equal(t, req.CategoryID, design.CategoryID)
				assert.Equal(t, domain.DesignStatusActive, design.Status)
				return nil
			})

		design, err := service.Create(ctx, req, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, design)
		assert.Equal(t, req.Name, design.Name)
	})

	t.Run("category not found", func(t *testing.T) {
		req := domain.CreateDesignRequest{
			Name:       "Test Design",
			CategoryID: "nonexistent_cat",
		}

		mockCategoryRepo.EXPECT().
			GetByID(ctx, req.CategoryID).
			Return(nil, errors.NotFound("Category not found"))

		design, err := service.Create(ctx, req, "admin_123")

		assert.Nil(t, design)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Category not found")
	})
}

func TestDesignService_GetByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDesignRepo := mocks.NewMockDesignRepository(ctrl)
	mockCategoryRepo := mocks.NewMockCategoryRepository(ctrl)
	log := logger.NewNoop()
	service := NewDesignService(mockDesignRepo, mockCategoryRepo, log)
	ctx := context.Background()

	t.Run("successful get by ID with category", func(t *testing.T) {
		expectedDesign := &domain.Design{
			ID:          "design_123",
			Name:        "Test Design",
			Slug:        "test-design",
			CategoryID:  "cat_123",
			Description: "Test description",
			Status:      domain.DesignStatusActive,
		}

		category := &domain.Category{
			ID:   "cat_123",
			Name: "Sarees",
			Slug: "sarees",
		}

		mockDesignRepo.EXPECT().
			GetByID(ctx, "design_123").
			Return(expectedDesign, nil)

		mockCategoryRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(category, nil)

		result, err := service.GetByID(ctx, "design_123")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, expectedDesign.ID, result.Design.ID)
		assert.NotNil(t, result.Category)
		assert.Equal(t, "cat_123", result.Category.ID)
	})

	t.Run("design not found", func(t *testing.T) {
		mockDesignRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Design not found"))

		result, err := service.GetByID(ctx, "nonexistent")

		assert.Nil(t, result)
		require.Error(t, err)
	})
}

func TestDesignService_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDesignRepo := mocks.NewMockDesignRepository(ctrl)
	mockCategoryRepo := mocks.NewMockCategoryRepository(ctrl)
	log := logger.NewNoop()
	service := NewDesignService(mockDesignRepo, mockCategoryRepo, log)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		existingDesign := &domain.Design{
			ID:          "design_123",
			Name:        "Original Name",
			Slug:        "original-name",
			CategoryID:  "cat_123",
			Description: "Original description",
			Status:      domain.DesignStatusActive,
		}

		newName := "Updated Design Name"
		newDescription := "Updated description"
		req := domain.UpdateDesignRequest{
			Name:        &newName,
			Description: &newDescription,
		}

		mockDesignRepo.EXPECT().
			GetByID(ctx, "design_123").
			Return(existingDesign, nil)

		mockDesignRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, design *domain.Design) error {
				assert.Equal(t, newName, design.Name)
				assert.Equal(t, "updated-design-name", design.Slug)
				assert.Equal(t, newDescription, design.Description)
				return nil
			})

		design, err := service.Update(ctx, "design_123", req, "admin_456")

		require.NoError(t, err)
		assert.NotNil(t, design)
	})

	t.Run("update status", func(t *testing.T) {
		existingDesign := &domain.Design{
			ID:         "design_123",
			Name:       "Test Design",
			Status:     domain.DesignStatusActive,
			CategoryID: "cat_123",
		}

		newStatus := domain.DesignStatusInactive
		req := domain.UpdateDesignRequest{
			Status: &newStatus,
		}

		mockDesignRepo.EXPECT().
			GetByID(ctx, "design_123").
			Return(existingDesign, nil)

		mockDesignRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, design *domain.Design) error {
				assert.Equal(t, domain.DesignStatusInactive, design.Status)
				return nil
			})

		design, err := service.Update(ctx, "design_123", req, "admin_456")

		require.NoError(t, err)
		assert.NotNil(t, design)
	})
}

func TestDesignService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDesignRepo := mocks.NewMockDesignRepository(ctrl)
	mockCategoryRepo := mocks.NewMockCategoryRepository(ctrl)
	log := logger.NewNoop()
	service := NewDesignService(mockDesignRepo, mockCategoryRepo, log)
	ctx := context.Background()

	t.Run("successful delete", func(t *testing.T) {
		existingDesign := &domain.Design{
			ID:           "design_123",
			Name:         "Test Design",
			CategoryID:   "cat_123",
			ProductCount: 0, // No products
		}

		mockDesignRepo.EXPECT().
			GetByID(ctx, "design_123").
			Return(existingDesign, nil)

		mockDesignRepo.EXPECT().
			Delete(ctx, "design_123").
			Return(nil)

		err := service.Delete(ctx, "design_123")

		require.NoError(t, err)
	})

	t.Run("cannot delete design with products", func(t *testing.T) {
		existingDesign := &domain.Design{
			ID:           "design_123",
			Name:         "Test Design",
			CategoryID:   "cat_123",
			ProductCount: 5, // Has products
		}

		mockDesignRepo.EXPECT().
			GetByID(ctx, "design_123").
			Return(existingDesign, nil)

		err := service.Delete(ctx, "design_123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "products")
	})

	t.Run("design not found", func(t *testing.T) {
		mockDesignRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Design not found"))

		err := service.Delete(ctx, "nonexistent")

		require.Error(t, err)
	})
}

func TestDesignService_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDesignRepo := mocks.NewMockDesignRepository(ctrl)
	mockCategoryRepo := mocks.NewMockCategoryRepository(ctrl)
	log := logger.NewNoop()
	service := NewDesignService(mockDesignRepo, mockCategoryRepo, log)
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		req := domain.ListDesignsRequest{
			PaginationRequest: domain.PaginationRequest{
				Limit: 20,
			},
		}

		expectedResponse := &domain.ListDesignsResponse{
			Designs: []*domain.Design{
				{ID: "design_1", Name: "Design 1"},
				{ID: "design_2", Name: "Design 2"},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockDesignRepo.EXPECT().
			List(ctx, req).
			Return(expectedResponse, nil)

		response, err := service.List(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Designs, 2)
	})

	t.Run("list by category", func(t *testing.T) {
		categoryID := "cat_123"
		req := domain.ListDesignsRequest{
			PaginationRequest: domain.PaginationRequest{
				Limit: 10,
			},
			CategoryID: &categoryID,
		}

		expectedResponse := &domain.ListDesignsResponse{
			Designs: []*domain.Design{
				{ID: "design_1", Name: "Design 1", CategoryID: categoryID},
			},
			Pagination: domain.PaginationResponse{
				Limit:   10,
				HasMore: false,
			},
		}

		mockDesignRepo.EXPECT().
			List(ctx, req).
			Return(expectedResponse, nil)

		response, err := service.List(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Designs, 1)
	})
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "Test Design",
			expected: "test-design",
		},
		{
			name:     "name with special characters",
			input:    "Ikkat Diamond Pattern!",
			expected: "ikkat-diamond-pattern",
		},
		{
			name:     "name with multiple spaces",
			input:    "Multiple   Spaces   Here",
			expected: "multiple-spaces-here",
		},
		{
			name:     "name with numbers",
			input:    "Design 123 Pattern",
			expected: "design-123-pattern",
		},
		{
			name:     "name with leading/trailing spaces",
			input:    "  Trimmed Name  ",
			expected: "trimmed-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlug(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
