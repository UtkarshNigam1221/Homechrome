package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
)

func setupCategoryTest(t *testing.T) (
	*CategoryService,
	*mocks.MockCategoryRepository,
	*mocks.MockProductRepository,
	*mocks.MockAssetFinalizer,
	context.Context,
) {
	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	mockCatRepo := mocks.NewMockCategoryRepository(ctrl)
	mockProdRepo := mocks.NewMockProductRepository(ctrl)
	mockFinalizer := mocks.NewMockAssetFinalizer(ctrl)
	mockFinalizer.EXPECT().SyncImageVariants(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	svc := NewCategoryService(mockCatRepo, mockProdRepo, mockFinalizer)
	return svc, mockCatRepo, mockProdRepo, mockFinalizer, context.Background()
}

func TestCategoryService_Create(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		req := domain.CreateCategoryRequest{
			Name:        "Silk Sarees",
			Description: "Handwoven silk sarees",
		}

		mockCatRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, cat *domain.Category) error {
				assert.Contains(t, cat.ID, "cat_")
				assert.Equal(t, "Silk Sarees", cat.Name)
				assert.Equal(t, "silk-sarees", cat.Slug)
				assert.Equal(t, "Handwoven silk sarees", cat.Description)
				assert.Equal(t, domain.CategoryStatusActive, cat.Status)
				assert.Equal(t, "admin_1", cat.CreatedBy)
				return nil
			})

		cat, err := svc.Create(ctx, req, "admin_1")

		require.NoError(t, err)
		assert.NotNil(t, cat)
		assert.Equal(t, "Silk Sarees", cat.Name)
		assert.Equal(t, "silk-sarees", cat.Slug)
	})

	t.Run("creation with image finalizes tmp URL", func(t *testing.T) {
		svc, mockCatRepo, _, mockFinalizer, ctx := setupCategoryTest(t)

		req := domain.CreateCategoryRequest{
			Name:     "Cotton",
			ImageURL: "tmp/category/abc.jpg",
		}

		mockFinalizer.EXPECT().
			FinalizeIfTemp(ctx, "tmp/category/abc.jpg").
			Return("assets/category/abc.jpg", nil)

		mockCatRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, cat *domain.Category) error {
				assert.Equal(t, "assets/category/abc.jpg", cat.ImageURL)
				return nil
			})

		cat, err := svc.Create(ctx, req, "admin_1")

		require.NoError(t, err)
		assert.Equal(t, "assets/category/abc.jpg", cat.ImageURL)
	})

	t.Run("creation with image finalize failure", func(t *testing.T) {
		svc, _, _, mockFinalizer, ctx := setupCategoryTest(t)

		req := domain.CreateCategoryRequest{
			Name:     "Cotton",
			ImageURL: "tmp/category/abc.jpg",
		}

		mockFinalizer.EXPECT().
			FinalizeIfTemp(ctx, "tmp/category/abc.jpg").
			Return("", errors.Internal("S3 error"))

		cat, err := svc.Create(ctx, req, "admin_1")

		assert.Nil(t, cat)
		require.Error(t, err)
	})

	t.Run("creation with attributes", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		attrs := []domain.CategoryAttribute{
			{Name: "color", Label: "Color", Type: "SELECT", Required: true},
			{Name: "material", Label: "Material", Type: "TEXT"},
		}
		req := domain.CreateCategoryRequest{
			Name:          "Fabrics",
			OwnAttributes: attrs,
		}

		mockCatRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, cat *domain.Category) error {
				assert.Len(t, cat.OwnAttributes, 2)
				assert.Equal(t, "color", cat.OwnAttributes[0].Name)
				return nil
			})

		cat, err := svc.Create(ctx, req, "admin_1")

		require.NoError(t, err)
		assert.Len(t, cat.OwnAttributes, 2)
	})

	t.Run("repo error on create", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		req := domain.CreateCategoryRequest{Name: "Test"}

		mockCatRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(errors.Internal("db error"))

		cat, err := svc.Create(ctx, req, "admin_1")

		assert.Nil(t, cat)
		require.Error(t, err)
	})
}

func TestCategoryService_GetByID(t *testing.T) {
	t.Run("successful get", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		expected := &domain.Category{
			ID:   "cat_123",
			Name: "Silk Sarees",
			Slug: "silk-sarees",
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(expected, nil)

		cat, err := svc.GetByID(ctx, "cat_123")

		require.NoError(t, err)
		assert.Equal(t, "cat_123", cat.ID)
		assert.Equal(t, "Silk Sarees", cat.Name)
	})

	t.Run("not found", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_nonexistent").
			Return(nil, errors.NotFound("Category"))

		cat, err := svc.GetByID(ctx, "cat_nonexistent")

		assert.Nil(t, cat)
		require.Error(t, err)
	})
}

func TestCategoryService_Update(t *testing.T) {
	t.Run("successful update with name", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID:   "cat_123",
			Name: "Old Name",
			Slug: "old-name",
		}

		newName := "New Name"
		req := domain.UpdateCategoryRequest{
			Name: &newName,
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		mockCatRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, cat *domain.Category) error {
				assert.Equal(t, "New Name", cat.Name)
				assert.Equal(t, "new-name", cat.Slug)
				assert.Equal(t, "admin_1", cat.UpdatedBy)
				return nil
			})

		cat, err := svc.Update(ctx, "cat_123", req, "admin_1")

		require.NoError(t, err)
		assert.Equal(t, "New Name", cat.Name)
		assert.Equal(t, "new-name", cat.Slug)
	})

	t.Run("update with image finalizes tmp URL", func(t *testing.T) {
		svc, mockCatRepo, _, mockFinalizer, ctx := setupCategoryTest(t)

		existing := &domain.Category{ID: "cat_123", Name: "Test"}

		imgURL := "tmp/category/new.jpg"
		req := domain.UpdateCategoryRequest{ImageURL: &imgURL}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		mockFinalizer.EXPECT().
			FinalizeIfTemp(ctx, "tmp/category/new.jpg").
			Return("assets/category/new.jpg", nil)

		mockCatRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, cat *domain.Category) error {
				assert.Equal(t, "assets/category/new.jpg", cat.ImageURL)
				return nil
			})

		cat, err := svc.Update(ctx, "cat_123", req, "admin_1")

		require.NoError(t, err)
		assert.Equal(t, "assets/category/new.jpg", cat.ImageURL)
	})

	t.Run("update image finalize failure", func(t *testing.T) {
		svc, mockCatRepo, _, mockFinalizer, ctx := setupCategoryTest(t)

		existing := &domain.Category{ID: "cat_123", Name: "Test"}

		imgURL := "tmp/category/bad.jpg"
		req := domain.UpdateCategoryRequest{ImageURL: &imgURL}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		mockFinalizer.EXPECT().
			FinalizeIfTemp(ctx, "tmp/category/bad.jpg").
			Return("", errors.Internal("S3 error"))

		cat, err := svc.Update(ctx, "cat_123", req, "admin_1")

		assert.Nil(t, cat)
		require.Error(t, err)
	})

	t.Run("update with status change", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID:     "cat_123",
			Name:   "Test",
			Status: domain.CategoryStatusActive,
		}

		status := domain.CategoryStatusInactive
		req := domain.UpdateCategoryRequest{Status: &status}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		mockCatRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, cat *domain.Category) error {
				assert.Equal(t, domain.CategoryStatusInactive, cat.Status)
				return nil
			})

		cat, err := svc.Update(ctx, "cat_123", req, "admin_1")

		require.NoError(t, err)
		assert.Equal(t, domain.CategoryStatusInactive, cat.Status)
	})

	t.Run("update - category not found", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_nonexistent").
			Return(nil, errors.NotFound("Category"))

		cat, err := svc.Update(ctx, "cat_nonexistent", domain.UpdateCategoryRequest{}, "admin_1")

		assert.Nil(t, cat)
		require.Error(t, err)
	})
}

func TestCategoryService_Delete(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID:           "cat_123",
			ProductCount: 0,
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		mockCatRepo.EXPECT().
			Delete(ctx, "cat_123").
			Return(nil)

		err := svc.Delete(ctx, "cat_123")
		require.NoError(t, err)
	})

	t.Run("delete category with products - rejected", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID:           "cat_123",
			ProductCount: 5,
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		err := svc.Delete(ctx, "cat_123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot delete category with existing products")
	})

	t.Run("delete - category not found", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_nonexistent").
			Return(nil, errors.NotFound("Category"))

		err := svc.Delete(ctx, "cat_nonexistent")
		require.Error(t, err)
	})
}

func TestCategoryService_List(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		req := domain.ListCategoriesRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 20},
		}

		expectedResp := &domain.ListCategoriesResponse{
			Categories: []*domain.Category{
				{ID: "cat_1", Name: "Silk"},
				{ID: "cat_2", Name: "Cotton"},
			},
			Pagination: domain.PaginationResponse{Limit: 20, HasMore: false},
		}

		mockCatRepo.EXPECT().
			List(ctx, req).
			Return(expectedResp, nil)

		resp, err := svc.List(ctx, req)

		require.NoError(t, err)
		assert.Len(t, resp.Categories, 2)
	})

	t.Run("list with status filter", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		status := domain.CategoryStatusActive
		req := domain.ListCategoriesRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 10},
			Status:            &status,
		}

		expectedResp := &domain.ListCategoriesResponse{
			Categories: []*domain.Category{
				{ID: "cat_1", Name: "Silk", Status: domain.CategoryStatusActive},
			},
			Pagination: domain.PaginationResponse{Limit: 10, HasMore: false},
		}

		mockCatRepo.EXPECT().
			List(ctx, req).
			Return(expectedResp, nil)

		resp, err := svc.List(ctx, req)

		require.NoError(t, err)
		assert.Len(t, resp.Categories, 1)
	})
}

func TestCategoryService_AddAttribute(t *testing.T) {
	t.Run("successful add", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID:            "cat_123",
			OwnAttributes: []domain.CategoryAttribute{},
		}

		newAttr := domain.CategoryAttribute{
			Name:     "color",
			Label:    "Color",
			Type:     "SELECT",
			Required: true,
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		mockCatRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, cat *domain.Category) error {
				assert.Len(t, cat.OwnAttributes, 1)
				assert.Equal(t, "color", cat.OwnAttributes[0].Name)
				assert.Equal(t, "admin_1", cat.UpdatedBy)
				return nil
			})

		cat, err := svc.AddAttribute(ctx, "cat_123", newAttr, "admin_1")

		require.NoError(t, err)
		assert.Len(t, cat.OwnAttributes, 1)
	})

	t.Run("duplicate attribute name - rejected", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "color", Label: "Color"},
			},
		}

		newAttr := domain.CategoryAttribute{Name: "color", Label: "Different Label"}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		cat, err := svc.AddAttribute(ctx, "cat_123", newAttr, "admin_1")

		assert.Nil(t, cat)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("category not found", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_nonexistent").
			Return(nil, errors.NotFound("Category"))

		cat, err := svc.AddAttribute(ctx, "cat_nonexistent", domain.CategoryAttribute{Name: "test"}, "admin_1")

		assert.Nil(t, cat)
		require.Error(t, err)
	})
}

func TestCategoryService_UpdateAttribute(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "color", Label: "Color", Type: "SELECT"},
				{Name: "size", Label: "Size", Type: "TEXT"},
			},
		}

		updatedAttr := domain.CategoryAttribute{
			Name:     "color",
			Label:    "Color",
			Type:     "MULTI_SELECT",
			Required: true,
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		mockCatRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, cat *domain.Category) error {
				assert.Equal(t, "Color", cat.OwnAttributes[0].Label)
				assert.Equal(t, domain.AttributeType("MULTI_SELECT"), cat.OwnAttributes[0].Type)
				assert.True(t, cat.OwnAttributes[0].Required)
				// Second attribute unchanged
				assert.Equal(t, "size", cat.OwnAttributes[1].Name)
				return nil
			})

		cat, err := svc.UpdateAttribute(ctx, "cat_123", "color", updatedAttr, "admin_1")

		require.NoError(t, err)
		assert.NotNil(t, cat)
	})

	t.Run("attribute not found", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "color", Label: "Color"},
			},
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		cat, err := svc.UpdateAttribute(ctx, "cat_123", "nonexistent", domain.CategoryAttribute{}, "admin_1")

		assert.Nil(t, cat)
		require.Error(t, err)
	})

	t.Run("category not found", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_nonexistent").
			Return(nil, errors.NotFound("Category"))

		cat, err := svc.UpdateAttribute(ctx, "cat_nonexistent", "color", domain.CategoryAttribute{}, "admin_1")

		assert.Nil(t, cat)
		require.Error(t, err)
	})
}

func TestCategoryService_DeleteAttribute(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "color", Label: "Color"},
				{Name: "size", Label: "Size"},
			},
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		mockCatRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, cat *domain.Category) error {
				assert.Len(t, cat.OwnAttributes, 1)
				assert.Equal(t, "size", cat.OwnAttributes[0].Name)
				assert.Equal(t, "admin_1", cat.UpdatedBy)
				return nil
			})

		err := svc.DeleteAttribute(ctx, "cat_123", "color", "admin_1")
		require.NoError(t, err)
	})

	t.Run("attribute not found", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "color", Label: "Color"},
			},
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		err := svc.DeleteAttribute(ctx, "cat_123", "nonexistent", "admin_1")
		require.Error(t, err)
	})

	t.Run("category not found", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_nonexistent").
			Return(nil, errors.NotFound("Category"))

		err := svc.DeleteAttribute(ctx, "cat_nonexistent", "color", "admin_1")
		require.Error(t, err)
	})
}

func TestCategoryService_GetAttributes(t *testing.T) {
	t.Run("successful get attributes", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "color", Label: "Color"},
				{Name: "size", Label: "Size"},
			},
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		resp, err := svc.GetAttributes(ctx, "cat_123")

		require.NoError(t, err)
		assert.Len(t, resp.OwnAttributes, 2)
		assert.Equal(t, 2, resp.TotalCount)
	})

	t.Run("empty attributes", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID:            "cat_123",
			OwnAttributes: nil,
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		resp, err := svc.GetAttributes(ctx, "cat_123")

		require.NoError(t, err)
		assert.Empty(t, resp.OwnAttributes)
		assert.Equal(t, 0, resp.TotalCount)
	})

	t.Run("category not found", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_nonexistent").
			Return(nil, errors.NotFound("Category"))

		resp, err := svc.GetAttributes(ctx, "cat_nonexistent")

		assert.Nil(t, resp)
		require.Error(t, err)
	})
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple name", "Silk Sarees", "silk-sarees"},
		{"special characters", "Women's Wear!", "womens-wear"},
		{"multiple spaces", "Hand  Woven  Silk", "hand-woven-silk"},
		{"leading/trailing spaces", " Trim Me ", "trim-me"},
		{"numbers", "Category 123", "category-123"},
		{"already lowercase", "cotton fabrics", "cotton-fabrics"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlug(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCategoryService_Create_EdgeCases(t *testing.T) {
	t.Run("image finalization failure prevents repo create", func(t *testing.T) {
		svc, _, _, mockFinalizer, ctx := setupCategoryTest(t)

		req := domain.CreateCategoryRequest{
			Name:     "Test",
			ImageURL: "tmp/category/bad.jpg",
		}

		mockFinalizer.EXPECT().
			FinalizeIfTemp(ctx, "tmp/category/bad.jpg").
			Return("", errors.Internal("S3 error"))

		// mockCatRepo.Create should NOT be called
		// (gomock will fail if unexpected call happens)

		cat, err := svc.Create(ctx, req, "admin_1")

		assert.Nil(t, cat)
		require.Error(t, err)
	})
}

func TestCategoryService_Delete_ErrorCodes(t *testing.T) {
	t.Run("products exist returns CATEGORY_HAS_PRODUCTS error code", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID:           "cat_123",
			ProductCount: 5,
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		err := svc.Delete(ctx, "cat_123")

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeCategoryHasProducts, appErr.Code)
	})
}

func TestCategoryService_AddAttribute_EdgeCases(t *testing.T) {
	t.Run("repo update failure returns error", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID:            "cat_123",
			OwnAttributes: []domain.CategoryAttribute{},
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		mockCatRepo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(errors.Internal("db error"))

		cat, err := svc.AddAttribute(ctx, "cat_123", domain.CategoryAttribute{
			Name: "color", Label: "Color",
		}, "admin_1")

		assert.Nil(t, cat)
		require.Error(t, err)
	})
}

func TestCategoryService_UpdateAttribute_EdgeCases(t *testing.T) {
	t.Run("preserves other attributes unchanged", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "color", Label: "Color", Type: "SELECT", Required: true},
				{Name: "size", Label: "Size", Type: "TEXT", Required: false},
				{Name: "pattern", Label: "Pattern", Type: "TEXT", Required: true},
			},
		}

		updatedAttr := domain.CategoryAttribute{
			Name: "size", Label: "Size Updated", Type: "SELECT", Required: true,
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		mockCatRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, cat *domain.Category) error {
				assert.Len(t, cat.OwnAttributes, 3)
				// First and third should be unchanged
				assert.Equal(t, "color", cat.OwnAttributes[0].Name)
				assert.Equal(t, "Color", cat.OwnAttributes[0].Label)
				assert.Equal(t, domain.AttributeType("SELECT"), cat.OwnAttributes[0].Type)
				// Second should be updated
				assert.Equal(t, "Size Updated", cat.OwnAttributes[1].Label)
				assert.Equal(t, domain.AttributeType("SELECT"), cat.OwnAttributes[1].Type)
				assert.True(t, cat.OwnAttributes[1].Required)
				// Third unchanged
				assert.Equal(t, "pattern", cat.OwnAttributes[2].Name)
				assert.Equal(t, "Pattern", cat.OwnAttributes[2].Label)
				return nil
			})

		cat, err := svc.UpdateAttribute(ctx, "cat_123", "size", updatedAttr, "admin_1")

		require.NoError(t, err)
		assert.NotNil(t, cat)
	})
}

func TestCategoryService_DeleteAttribute_EdgeCases(t *testing.T) {
	t.Run("first attribute deletion works", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "first", Label: "First"},
				{Name: "second", Label: "Second"},
				{Name: "third", Label: "Third"},
			},
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		mockCatRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, cat *domain.Category) error {
				assert.Len(t, cat.OwnAttributes, 2)
				assert.Equal(t, "second", cat.OwnAttributes[0].Name)
				assert.Equal(t, "third", cat.OwnAttributes[1].Name)
				return nil
			})

		err := svc.DeleteAttribute(ctx, "cat_123", "first", "admin_1")
		require.NoError(t, err)
	})

	t.Run("preserves remaining attributes", func(t *testing.T) {
		svc, mockCatRepo, _, _, ctx := setupCategoryTest(t)

		existing := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "a", Label: "A", Type: "TEXT"},
				{Name: "b", Label: "B", Type: "SELECT"},
				{Name: "c", Label: "C", Type: "NUMBER"},
			},
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(existing, nil)

		mockCatRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, cat *domain.Category) error {
				assert.Len(t, cat.OwnAttributes, 2)
				// Verify full integrity of remaining attributes
				assert.Equal(t, "a", cat.OwnAttributes[0].Name)
				assert.Equal(t, domain.AttributeType("TEXT"), cat.OwnAttributes[0].Type)
				assert.Equal(t, "c", cat.OwnAttributes[1].Name)
				assert.Equal(t, domain.AttributeType("NUMBER"), cat.OwnAttributes[1].Type)
				return nil
			})

		err := svc.DeleteAttribute(ctx, "cat_123", "b", "admin_1")
		require.NoError(t, err)
	})
}

func TestGenerateSlug_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"consecutive dashes from special chars", "A--B--C", "a-b-c"},
		{"special chars creating dashes", "Hello!!!World", "helloworld"},
		{"mixed special and spaces", "Silk & Cotton Blend", "silk-cotton-blend"},
		{"unicode stripped", "Café Mocha", "caf-mocha"},
		{"only dashes", "---", ""},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlug(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
