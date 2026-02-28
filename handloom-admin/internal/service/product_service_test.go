package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/event"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
)

func setupProductTest(t *testing.T) (
	*ProductService,
	*mocks.MockProductRepository,
	*mocks.MockCategoryRepository,
	*mocks.MockInventoryRepository,
	*mocks.MockAssetFinalizer,
	context.Context,
) {
	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	mockProdRepo := mocks.NewMockProductRepository(ctrl)
	mockCatRepo := mocks.NewMockCategoryRepository(ctrl)
	mockInvRepo := mocks.NewMockInventoryRepository(ctrl)
	mockFinalizer := mocks.NewMockAssetFinalizer(ctrl)
	log := logger.NewNoop()

	publisher := event.NewNoopPublisher()
	svc := NewProductService(mockProdRepo, mockCatRepo, mockInvRepo, mockFinalizer, publisher, log)
	return svc, mockProdRepo, mockCatRepo, mockInvRepo, mockFinalizer, context.Background()
}

func setupProductTestWithSpy(t *testing.T) (
	*ProductService,
	*mocks.MockProductRepository,
	*mocks.MockCategoryRepository,
	*mocks.MockInventoryRepository,
	*mocks.MockAssetFinalizer,
	*spyPublisher,
	context.Context,
) {
	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	mockProdRepo := mocks.NewMockProductRepository(ctrl)
	mockCatRepo := mocks.NewMockCategoryRepository(ctrl)
	mockInvRepo := mocks.NewMockInventoryRepository(ctrl)
	mockFinalizer := mocks.NewMockAssetFinalizer(ctrl)
	log := logger.NewNoop()

	spy := newSpyPublisher()
	svc := NewProductService(mockProdRepo, mockCatRepo, mockInvRepo, mockFinalizer, spy, log)
	return svc, mockProdRepo, mockCatRepo, mockInvRepo, mockFinalizer, spy, context.Background()
}

func TestProductService_Create(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{
			ID:   "cat_123",
			Name: "Silk Sarees",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "weave_pattern", Searchable: true, Required: false},
			},
		}

		req := domain.CreateProductRequest{
			Name:         "Banarasi Silk Saree",
			SKU:          "BSS-001",
			CategoryID:   "cat_123",
			BasePrice:    500000,
			SellingPrice: 450000,
			Material:     "silk",
			Color:        "red",
			InitialStock: 10,
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(category, nil)

		mockProdRepo.EXPECT().
			Create(ctx, gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, product *domain.Product, inv *domain.Inventory) error {
				assert.Contains(t, product.ID, "prod_")
				assert.Equal(t, "Banarasi Silk Saree", product.Name)
				assert.Equal(t, "banarasi-silk-saree", product.Slug)
				assert.Equal(t, "BSS-001", product.SKU)
				assert.Equal(t, domain.ProductStatusDraft, product.Status)
				assert.Equal(t, int64(450000), product.SellingPrice)
				// Inventory should be created with initial stock
				assert.Equal(t, 10, inv.Quantity)
				assert.Equal(t, 10, inv.AvailableQty)
				return nil
			})

		mockCatRepo.EXPECT().
			IncrementProductCount(ctx, "cat_123", 1).
			Return(nil)

		product, err := svc.Create(ctx, req, "admin_1")

		require.NoError(t, err)
		assert.NotNil(t, product)
		assert.Equal(t, "Banarasi Silk Saree", product.Name)
	})

	t.Run("creation with explicit status", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{
			ID:   "cat_123",
			Name: "Silk Sarees",
		}

		activeStatus := domain.ProductStatusActive
		req := domain.CreateProductRequest{
			Name:         "Active Product",
			SKU:          "ACT-001",
			CategoryID:   "cat_123",
			BasePrice:    100000,
			SellingPrice: 90000,
			Status:       &activeStatus,
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(category, nil)

		mockProdRepo.EXPECT().
			Create(ctx, gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, product *domain.Product, inv *domain.Inventory) error {
				assert.Equal(t, domain.ProductStatusActive, product.Status)
				return nil
			})

		mockCatRepo.EXPECT().
			IncrementProductCount(ctx, "cat_123", 1).
			Return(nil)

		product, err := svc.Create(ctx, req, "admin_1")

		require.NoError(t, err)
		assert.Equal(t, domain.ProductStatusActive, product.Status)
	})

	t.Run("category not found", func(t *testing.T) {
		svc, _, mockCatRepo, _, _, ctx := setupProductTest(t)

		req := domain.CreateProductRequest{
			Name:       "Test",
			SKU:        "TST-001",
			CategoryID: "cat_nonexistent",
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_nonexistent").
			Return(nil, errors.NotFound("Category"))

		product, err := svc.Create(ctx, req, "admin_1")

		assert.Nil(t, product)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Category not found")
	})

	t.Run("missing required attributes", func(t *testing.T) {
		svc, _, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "weave_pattern", Label: "Weave Pattern", Searchable: true, Required: true},
			},
		}

		req := domain.CreateProductRequest{
			Name:       "Test",
			SKU:        "TST-001",
			CategoryID: "cat_123",
			// No attributes provided — weave_pattern is required
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(category, nil)

		product, err := svc.Create(ctx, req, "admin_1")

		assert.Nil(t, product)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Missing required attributes")
	})

	t.Run("image finalization on create", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, mockFinalizer, ctx := setupProductTest(t)

		category := &domain.Category{ID: "cat_123"}

		req := domain.CreateProductRequest{
			Name:       "Test",
			SKU:        "TST-001",
			CategoryID: "cat_123",
			Material:   "cotton",
			Images: []domain.ProductImage{
				{URL: "tmp/product/img1.jpg"},
				{URL: "assets/product/img2.jpg"}, // already finalized
			},
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(category, nil)

		mockFinalizer.EXPECT().
			FinalizeIfTemp(ctx, "tmp/product/img1.jpg").
			Return("assets/product/img1.jpg", nil)

		mockFinalizer.EXPECT().
			FinalizeIfTemp(ctx, "assets/product/img2.jpg").
			Return("assets/product/img2.jpg", nil)

		mockProdRepo.EXPECT().
			Create(ctx, gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, product *domain.Product, _ *domain.Inventory) error {
				assert.Equal(t, "assets/product/img1.jpg", product.Images[0].URL)
				assert.Equal(t, "assets/product/img2.jpg", product.Images[1].URL)
				return nil
			})

		mockCatRepo.EXPECT().
			IncrementProductCount(ctx, "cat_123", 1).
			Return(nil)

		product, err := svc.Create(ctx, req, "admin_1")

		require.NoError(t, err)
		assert.NotNil(t, product)
	})

	t.Run("image finalization failure", func(t *testing.T) {
		svc, _, mockCatRepo, _, mockFinalizer, ctx := setupProductTest(t)

		category := &domain.Category{ID: "cat_123"}

		req := domain.CreateProductRequest{
			Name:       "Test",
			SKU:        "TST-001",
			CategoryID: "cat_123",
			Images:     []domain.ProductImage{{URL: "tmp/product/bad.jpg"}},
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(category, nil)

		mockFinalizer.EXPECT().
			FinalizeIfTemp(ctx, "tmp/product/bad.jpg").
			Return("", errors.Internal("S3 error"))

		product, err := svc.Create(ctx, req, "admin_1")

		assert.Nil(t, product)
		require.Error(t, err)
	})

	t.Run("repo error on create", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{ID: "cat_123"}

		req := domain.CreateProductRequest{
			Name:       "Test",
			SKU:        "TST-001",
			CategoryID: "cat_123",
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(category, nil)

		mockProdRepo.EXPECT().
			Create(ctx, gomock.Any(), gomock.Any()).
			Return(errors.Internal("db error"))

		product, err := svc.Create(ctx, req, "admin_1")

		assert.Nil(t, product)
		require.Error(t, err)
	})
}

func TestProductService_GetByID(t *testing.T) {
	t.Run("found with relations", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, mockInvRepo, _, ctx := setupProductTest(t)

		product := &domain.Product{
			ID:         "prod_123",
			Name:       "Silk Saree",
			CategoryID: "cat_123",
		}

		category := &domain.Category{
			ID:   "cat_123",
			Name: "Sarees",
			Slug: "sarees",
		}

		inventory := &domain.Inventory{
			ProductID:    "prod_123",
			Quantity:     50,
			AvailableQty: 45,
		}

		mockProdRepo.EXPECT().
			GetByID(ctx, "prod_123").
			Return(product, nil)

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(category, nil)

		mockInvRepo.EXPECT().
			GetByProductID(ctx, "prod_123").
			Return(inventory, nil)

		result, err := svc.GetByID(ctx, "prod_123")

		require.NoError(t, err)
		assert.Equal(t, "prod_123", result.Product.ID)
		assert.NotNil(t, result.Category)
		assert.Equal(t, "Sarees", result.Category.Name)
		assert.NotNil(t, result.Inventory)
		assert.Equal(t, 45, result.Inventory.AvailableQty)
	})

	t.Run("product not found", func(t *testing.T) {
		svc, mockProdRepo, _, _, _, ctx := setupProductTest(t)

		mockProdRepo.EXPECT().
			GetByID(ctx, "prod_nonexistent").
			Return(nil, errors.NotFound("Product"))

		result, err := svc.GetByID(ctx, "prod_nonexistent")

		assert.Nil(t, result)
		require.Error(t, err)
	})

	t.Run("category/inventory fetch errors are non-fatal", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, mockInvRepo, _, ctx := setupProductTest(t)

		product := &domain.Product{
			ID:         "prod_123",
			CategoryID: "cat_deleted",
		}

		mockProdRepo.EXPECT().
			GetByID(ctx, "prod_123").
			Return(product, nil)

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_deleted").
			Return(nil, errors.NotFound("Category"))

		mockInvRepo.EXPECT().
			GetByProductID(ctx, "prod_123").
			Return(nil, errors.NotFound("Inventory"))

		result, err := svc.GetByID(ctx, "prod_123")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Nil(t, result.Category)
		assert.Nil(t, result.Inventory)
	})
}

func TestProductService_Update(t *testing.T) {
	t.Run("successful partial update", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		existing := &domain.Product{
			ID:         "prod_123",
			Name:       "Old Name",
			CategoryID: "cat_123",
			Material:   "cotton",
		}

		category := &domain.Category{ID: "cat_123"}

		newName := "New Name"
		req := domain.UpdateProductRequest{
			Name: &newName,
		}

		mockProdRepo.EXPECT().
			GetByID(ctx, "prod_123").
			Return(existing, nil)

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(category, nil)

		mockProdRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, product *domain.Product) error {
				assert.Equal(t, "New Name", product.Name)
				assert.Equal(t, "new-name", product.Slug)
				assert.Equal(t, "admin_1", product.UpdatedBy)
				return nil
			})

		product, err := svc.Update(ctx, "prod_123", req, "admin_1")

		require.NoError(t, err)
		assert.Equal(t, "New Name", product.Name)
	})

	t.Run("product not found", func(t *testing.T) {
		svc, mockProdRepo, _, _, _, ctx := setupProductTest(t)

		mockProdRepo.EXPECT().
			GetByID(ctx, "prod_nonexistent").
			Return(nil, errors.NotFound("Product"))

		product, err := svc.Update(ctx, "prod_nonexistent", domain.UpdateProductRequest{}, "admin_1")

		assert.Nil(t, product)
		require.Error(t, err)
	})

	t.Run("update with image finalization", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, mockFinalizer, ctx := setupProductTest(t)

		existing := &domain.Product{
			ID:         "prod_123",
			CategoryID: "cat_123",
			Material:   "silk",
		}

		category := &domain.Category{ID: "cat_123"}

		req := domain.UpdateProductRequest{
			Images: []domain.ProductImage{{URL: "tmp/product/new.jpg"}},
		}

		mockProdRepo.EXPECT().
			GetByID(ctx, "prod_123").
			Return(existing, nil)

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(category, nil)

		mockFinalizer.EXPECT().
			FinalizeIfTemp(ctx, "tmp/product/new.jpg").
			Return("assets/product/new.jpg", nil)

		mockProdRepo.EXPECT().
			Update(ctx, gomock.Any()).
			Return(nil)

		product, err := svc.Update(ctx, "prod_123", req, "admin_1")

		require.NoError(t, err)
		assert.NotNil(t, product)
	})
}

func TestProductService_Delete(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		product := &domain.Product{
			ID:         "prod_123",
			CategoryID: "cat_123",
		}

		mockProdRepo.EXPECT().
			GetByID(ctx, "prod_123").
			Return(product, nil)

		mockProdRepo.EXPECT().
			Delete(ctx, "prod_123").
			Return(nil)

		mockCatRepo.EXPECT().
			IncrementProductCount(ctx, "cat_123", -1).
			Return(nil)

		err := svc.Delete(ctx, "prod_123")
		require.NoError(t, err)
	})

	t.Run("product not found", func(t *testing.T) {
		svc, mockProdRepo, _, _, _, ctx := setupProductTest(t)

		mockProdRepo.EXPECT().
			GetByID(ctx, "prod_nonexistent").
			Return(nil, errors.NotFound("Product"))

		err := svc.Delete(ctx, "prod_nonexistent")
		require.Error(t, err)
	})

	t.Run("delete with category count decrement", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		product := &domain.Product{
			ID:         "prod_123",
			CategoryID: "cat_deleted",
		}

		mockProdRepo.EXPECT().
			GetByID(ctx, "prod_123").
			Return(product, nil)

		mockProdRepo.EXPECT().
			Delete(ctx, "prod_123").
			Return(nil)

		mockCatRepo.EXPECT().
			IncrementProductCount(ctx, "cat_deleted", -1).
			Return(nil)

		err := svc.Delete(ctx, "prod_123")
		require.NoError(t, err)
	})
}

func TestProductService_List(t *testing.T) {
	t.Run("basic list", func(t *testing.T) {
		svc, mockProdRepo, _, _, _, ctx := setupProductTest(t)

		req := domain.ListProductsRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 20},
		}

		expected := &domain.ListProductsResponse{
			Products: []*domain.Product{
				{ID: "prod_1", Name: "Saree A"},
				{ID: "prod_2", Name: "Saree B"},
			},
			Pagination: domain.PaginationResponse{Limit: 20, HasMore: false},
		}

		mockProdRepo.EXPECT().
			List(ctx, req).
			Return(expected, nil)

		resp, err := svc.List(ctx, req)

		require.NoError(t, err)
		assert.Len(t, resp.Products, 2)
	})

	t.Run("list with attribute filters", func(t *testing.T) {
		svc, mockProdRepo, _, _, _, ctx := setupProductTest(t)

		catID := "cat_123"
		req := domain.ListProductsRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 10},
			CategoryID:        &catID,
			AttributeFilters:  map[string][]string{"color": {"red", "blue"}},
		}

		expected := &domain.ListProductsResponse{
			Products:   []*domain.Product{{ID: "prod_1"}},
			Pagination: domain.PaginationResponse{Limit: 10, HasMore: false},
		}

		mockProdRepo.EXPECT().
			List(ctx, req).
			Return(expected, nil)

		resp, err := svc.List(ctx, req)

		require.NoError(t, err)
		assert.Len(t, resp.Products, 1)
	})
}

func TestProductService_GetAttributeFilterOptions(t *testing.T) {
	t.Run("returns sorted searchable values", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "color", Searchable: true},
				{Name: "size", Searchable: false},   // not searchable
				{Name: "pattern", Searchable: true},
			},
		}

		repoValues := map[string][]string{
			"color":   {"red", "blue", "green"},
			"pattern": {"floral", "geometric"},
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(category, nil)

		mockProdRepo.EXPECT().
			GetAttributeFilterOptions(ctx, "cat_123", gomock.InAnyOrder([]string{"color", "pattern"})).
			Return(repoValues, nil)

		result, err := svc.GetAttributeFilterOptions(ctx, "cat_123")

		require.NoError(t, err)
		assert.Contains(t, result, "color")
		assert.Contains(t, result, "pattern")
		assert.NotContains(t, result, "size") // not searchable
		// Values should be sorted
		assert.Equal(t, []string{"blue", "green", "red"}, result["color"])
	})

	t.Run("category not found", func(t *testing.T) {
		svc, _, mockCatRepo, _, _, ctx := setupProductTest(t)

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_nonexistent").
			Return(nil, errors.NotFound("Category"))

		result, err := svc.GetAttributeFilterOptions(ctx, "cat_nonexistent")

		assert.Nil(t, result)
		require.Error(t, err)
	})
}

func TestNormalizeToStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{"string value", "hello", []string{"hello"}},
		{"empty string", "", nil},
		{"string slice", []string{"a", "b"}, []string{"a", "b"}},
		{"interface slice", []interface{}{"a", "b"}, []string{"a", "b"}},
		{"nil", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeToStringSlice(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProductService_ReorderProducts(t *testing.T) {
	t.Run("successful reorder", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{ID: "cat_123", Name: "Test"}

		existingProducts := []*domain.Product{
			{ID: "prod_a", CategoryID: "cat_123", SortOrder: 0},
			{ID: "prod_b", CategoryID: "cat_123", SortOrder: 0},
			{ID: "prod_c", CategoryID: "cat_123", SortOrder: 0},
		}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().GetByCategoryAll(ctx, "cat_123").Return(existingProducts, nil)
		mockProdRepo.EXPECT().
			BatchUpdateSortOrder(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, products []*domain.Product) error {
				assert.Len(t, products, 3)
				// Requested order: c, a, b
				for _, p := range products {
					switch p.ID {
					case "prod_c":
						assert.Equal(t, 1, p.SortOrder)
					case "prod_a":
						assert.Equal(t, 2, p.SortOrder)
					case "prod_b":
						assert.Equal(t, 3, p.SortOrder)
					}
				}
				return nil
			})

		count, err := svc.ReorderProducts(ctx, "cat_123", []string{"prod_c", "prod_a", "prod_b"})
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("product not in category", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{ID: "cat_123", Name: "Test"}
		existingProducts := []*domain.Product{
			{ID: "prod_a", CategoryID: "cat_123"},
		}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().GetByCategoryAll(ctx, "cat_123").Return(existingProducts, nil)

		_, err := svc.ReorderProducts(ctx, "cat_123", []string{"prod_a", "prod_unknown"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not belong to category")
	})

	t.Run("category not found", func(t *testing.T) {
		svc, _, mockCatRepo, _, _, ctx := setupProductTest(t)

		mockCatRepo.EXPECT().GetByID(ctx, "cat_999").Return(nil, errors.NotFound("Category"))

		_, err := svc.ReorderProducts(ctx, "cat_999", []string{"prod_a"})
		require.Error(t, err)
	})
}

func TestValidateRequiredAttributes(t *testing.T) {
	categoryAttrs := []domain.CategoryAttribute{
		{Name: "weave_pattern", Label: "Weave Pattern", Searchable: true, Required: true},
		{Name: "color_shade", Label: "Color Shade", Searchable: true, Required: false},
	}

	t.Run("all required provided", func(t *testing.T) {
		attrs := map[string]interface{}{
			"weave_pattern": "floral",
		}
		err := validateRequiredAttributes(attrs, categoryAttrs)
		assert.NoError(t, err)
	})

	t.Run("missing required", func(t *testing.T) {
		err := validateRequiredAttributes(nil, categoryAttrs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Weave Pattern")
	})

	t.Run("empty value for required", func(t *testing.T) {
		attrs := map[string]interface{}{
			"weave_pattern": "",
		}
		err := validateRequiredAttributes(attrs, categoryAttrs)
		require.Error(t, err)
	})
}

func TestProductService_Create_EventPublishing(t *testing.T) {
	t.Run("publishes PRODUCT_CREATED event", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, spy, ctx := setupProductTestWithSpy(t)

		category := &domain.Category{ID: "cat_123"}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().Create(ctx, gomock.Any(), gomock.Any()).Return(nil)
		mockCatRepo.EXPECT().IncrementProductCount(ctx, "cat_123", 1).Return(nil)

		_, err := svc.Create(ctx, domain.CreateProductRequest{
			Name: "Test", SKU: "TST-001", CategoryID: "cat_123",
		}, "admin_1")

		require.NoError(t, err)
		assert.True(t, spy.hasEvent(event.ProductCreated))
	})

	t.Run("event failure is non-fatal", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		t.Cleanup(func() { ctrl.Finish() })

		mockProdRepo := mocks.NewMockProductRepository(ctrl)
		mockCatRepo := mocks.NewMockCategoryRepository(ctrl)
		mockInvRepo := mocks.NewMockInventoryRepository(ctrl)
		mockFinalizer := mocks.NewMockAssetFinalizer(ctrl)
		log := logger.NewNoop()

		failPub := newFailingPublisher(errors.Internal("SNS down"))
		svc := NewProductService(mockProdRepo, mockCatRepo, mockInvRepo, mockFinalizer, failPub, log)
		ctx := context.Background()

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(&domain.Category{ID: "cat_123"}, nil)
		mockProdRepo.EXPECT().Create(ctx, gomock.Any(), gomock.Any()).Return(nil)
		mockCatRepo.EXPECT().IncrementProductCount(ctx, "cat_123", 1).Return(nil)

		product, err := svc.Create(ctx, domain.CreateProductRequest{
			Name: "Test", SKU: "TST-001", CategoryID: "cat_123",
		}, "admin_1")

		require.NoError(t, err) // non-fatal
		assert.NotNil(t, product)
	})
}

func TestProductService_Update_EventPublishing(t *testing.T) {
	t.Run("publishes PRODUCT_UPDATED event", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, spy, ctx := setupProductTestWithSpy(t)

		existing := &domain.Product{ID: "prod_123", CategoryID: "cat_123", Material: "silk"}
		category := &domain.Category{ID: "cat_123"}
		newName := "Updated Name"

		mockProdRepo.EXPECT().GetByID(ctx, "prod_123").Return(existing, nil)
		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		_, err := svc.Update(ctx, "prod_123", domain.UpdateProductRequest{
			Name: &newName,
		}, "admin_1")

		require.NoError(t, err)
		assert.True(t, spy.hasEvent(event.ProductUpdated))
	})
}

func TestProductService_Delete_EventPublishing(t *testing.T) {
	t.Run("publishes PRODUCT_DELETED event", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, spy, ctx := setupProductTestWithSpy(t)

		mockProdRepo.EXPECT().GetByID(ctx, "prod_123").Return(&domain.Product{
			ID: "prod_123", CategoryID: "cat_123",
		}, nil)
		mockProdRepo.EXPECT().Delete(ctx, "prod_123").Return(nil)
		mockCatRepo.EXPECT().IncrementProductCount(ctx, "cat_123", -1).Return(nil)

		err := svc.Delete(ctx, "prod_123")

		require.NoError(t, err)
		assert.True(t, spy.hasEvent(event.ProductDeleted))
	})
}

func TestProductService_Create_ErrorCodes(t *testing.T) {
	t.Run("missing category returns NOT_FOUND error code", func(t *testing.T) {
		svc, _, mockCatRepo, _, _, ctx := setupProductTest(t)

		mockCatRepo.EXPECT().GetByID(ctx, "cat_999").Return(nil, errors.NotFound("Category"))

		_, err := svc.Create(ctx, domain.CreateProductRequest{
			Name: "Test", CategoryID: "cat_999",
		}, "admin_1")

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeNotFound, appErr.Code)
	})

	t.Run("missing required attr returns VALIDATION error code", func(t *testing.T) {
		svc, _, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "pattern", Label: "Pattern", Searchable: true, Required: true},
			},
		}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)

		_, err := svc.Create(ctx, domain.CreateProductRequest{
			Name: "Test", CategoryID: "cat_123",
		}, "admin_1")

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeValidation, appErr.Code)
	})
}

func TestProductService_Create_Atomicity(t *testing.T) {
	t.Run("inventory created alongside product", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{ID: "cat_123"}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().
			Create(ctx, gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, product *domain.Product, inv *domain.Inventory) error {
				// Both product and inventory must be created
				assert.NotNil(t, product)
				assert.NotNil(t, inv)
				assert.Equal(t, product.ID, inv.ProductID)
				assert.Equal(t, 25, inv.Quantity)
				assert.Equal(t, 25, inv.AvailableQty)
				assert.Equal(t, 5, inv.LowStockThreshold)
				assert.Equal(t, "admin_1", inv.CreatedBy)
				return nil
			})
		mockCatRepo.EXPECT().IncrementProductCount(ctx, "cat_123", 1).Return(nil)

		_, err := svc.Create(ctx, domain.CreateProductRequest{
			Name: "Test", SKU: "TST-001", CategoryID: "cat_123",
			InitialStock: 25, LowStockThreshold: 5,
		}, "admin_1")

		require.NoError(t, err)
	})

	t.Run("category count increment failure propagates", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{ID: "cat_123"}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().Create(ctx, gomock.Any(), gomock.Any()).Return(nil)
		mockCatRepo.EXPECT().IncrementProductCount(ctx, "cat_123", 1).Return(errors.Internal("db error"))

		_, err := svc.Create(ctx, domain.CreateProductRequest{
			Name: "Test", CategoryID: "cat_123",
		}, "admin_1")

		require.Error(t, err) // should propagate
	})
}

func TestProductService_Delete_Cascade(t *testing.T) {
	t.Run("decrements category count", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		mockProdRepo.EXPECT().GetByID(ctx, "prod_123").Return(&domain.Product{
			ID: "prod_123", CategoryID: "cat_123",
		}, nil)
		mockProdRepo.EXPECT().Delete(ctx, "prod_123").Return(nil)
		mockCatRepo.EXPECT().
			IncrementProductCount(ctx, "cat_123", -1).
			Return(nil) // if this wasn't called, gomock would fail

		err := svc.Delete(ctx, "prod_123")
		require.NoError(t, err)
	})
}

func TestProductService_Update_EdgeCases(t *testing.T) {
	t.Run("slug regenerated when name changes", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		existing := &domain.Product{
			ID: "prod_123", Name: "Old Name", Slug: "old-name", CategoryID: "cat_123",
		}
		category := &domain.Category{ID: "cat_123"}
		newName := "Brand New Name"

		mockProdRepo.EXPECT().GetByID(ctx, "prod_123").Return(existing, nil)
		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, p *domain.Product) error {
			assert.Equal(t, "brand-new-name", p.Slug)
			return nil
		})

		product, err := svc.Update(ctx, "prod_123", domain.UpdateProductRequest{Name: &newName}, "admin_1")

		require.NoError(t, err)
		assert.Equal(t, "brand-new-name", product.Slug)
	})

	t.Run("slug unchanged when name not provided", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		existing := &domain.Product{
			ID: "prod_123", Name: "Original", Slug: "original", CategoryID: "cat_123",
		}
		category := &domain.Category{ID: "cat_123"}

		mockProdRepo.EXPECT().GetByID(ctx, "prod_123").Return(existing, nil)
		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, p *domain.Product) error {
			assert.Equal(t, "original", p.Slug) // unchanged
			return nil
		})

		newDesc := "updated desc"
		_, err := svc.Update(ctx, "prod_123", domain.UpdateProductRequest{Description: &newDesc}, "admin_1")

		require.NoError(t, err)
	})
}

func TestProductService_GetAttributeFilterOptions_EdgeCases(t *testing.T) {
	t.Run("only searchable attributes included", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "color", Searchable: true},
				{Name: "internal_code", Searchable: false},
				{Name: "pattern", Searchable: true},
				{Name: "notes", Searchable: false},
			},
		}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().
			GetAttributeFilterOptions(ctx, "cat_123", gomock.InAnyOrder([]string{"color", "pattern"})).
			Return(map[string][]string{
				"color":   {"red"},
				"pattern": {"floral"},
			}, nil)

		result, err := svc.GetAttributeFilterOptions(ctx, "cat_123")

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Contains(t, result, "color")
		assert.Contains(t, result, "pattern")
		assert.NotContains(t, result, "internal_code")
		assert.NotContains(t, result, "notes")
	})

	t.Run("empty category returns empty map not nil", func(t *testing.T) {
		svc, _, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{
			ID:            "cat_123",
			OwnAttributes: nil, // no attributes at all
		}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)

		result, err := svc.GetAttributeFilterOptions(ctx, "cat_123")

		require.NoError(t, err)
		assert.NotNil(t, result) // should be empty map, not nil
		assert.Len(t, result, 0)
	})

	t.Run("no searchable attributes returns empty map", func(t *testing.T) {
		svc, _, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "notes", Searchable: false},
			},
		}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)

		result, err := svc.GetAttributeFilterOptions(ctx, "cat_123")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result, 0)
	})
}

func TestProductService_ReorderProducts_EdgeCases(t *testing.T) {
	t.Run("duplicate IDs rejected", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{ID: "cat_123"}
		products := []*domain.Product{
			{ID: "prod_a", CategoryID: "cat_123"},
			{ID: "prod_b", CategoryID: "cat_123"},
		}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().GetByCategoryAll(ctx, "cat_123").Return(products, nil)

		_, err := svc.ReorderProducts(ctx, "cat_123", []string{"prod_a", "prod_a"})

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeValidation, appErr.Code)
		assert.Contains(t, err.Error(), "Duplicate")
	})

	t.Run("partial reorder assigns sequential to unranked", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{ID: "cat_123"}
		products := []*domain.Product{
			{ID: "prod_a", CategoryID: "cat_123"},
			{ID: "prod_b", CategoryID: "cat_123"},
			{ID: "prod_c", CategoryID: "cat_123"},
		}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().GetByCategoryAll(ctx, "cat_123").Return(products, nil)
		mockProdRepo.EXPECT().
			BatchUpdateSortOrder(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, prods []*domain.Product) error {
				assert.Len(t, prods, 3) // all 3 updated even though only 1 was ranked
				// prod_c requested first
				for _, p := range prods {
					if p.ID == "prod_c" {
						assert.Equal(t, 1, p.SortOrder)
					}
				}
				return nil
			})

		count, err := svc.ReorderProducts(ctx, "cat_123", []string{"prod_c"})
		require.NoError(t, err)
		assert.Equal(t, 3, count) // all products updated
	})
}

func TestValidateRequiredAttributes_EdgeCases(t *testing.T) {
	t.Run("non-searchable required attribute is NOT enforced", func(t *testing.T) {
		categoryAttrs := []domain.CategoryAttribute{
			{Name: "internal_notes", Label: "Notes", Searchable: false, Required: true},
		}

		// No attributes provided, but the required attr is non-searchable
		err := validateRequiredAttributes(nil, categoryAttrs)
		assert.NoError(t, err) // should pass: only searchable+required is enforced
	})

	t.Run("interface slice with empty strings filtered", func(t *testing.T) {
		result := normalizeToStringSlice([]interface{}{"a", "", "b"})
		assert.Equal(t, []string{"a", "b"}, result) // empty strings filtered
	})
}
