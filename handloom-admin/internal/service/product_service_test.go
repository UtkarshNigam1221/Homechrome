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
			CreateWithAttributeIndexes(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, product *domain.Product, attrs map[string][]string, inv *domain.Inventory) error {
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

		mockProdRepo.EXPECT().
			AddAttributeValues(ctx, "cat_123", gomock.Any()).
			Return(nil)

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
			CreateWithAttributeIndexes(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, product *domain.Product, attrs map[string][]string, inv *domain.Inventory) error {
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
			CreateWithAttributeIndexes(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, product *domain.Product, _ map[string][]string, _ *domain.Inventory) error {
				assert.Equal(t, "assets/product/img1.jpg", product.Images[0].URL)
				assert.Equal(t, "assets/product/img2.jpg", product.Images[1].URL)
				return nil
			})

		mockProdRepo.EXPECT().
			AddAttributeValues(ctx, "cat_123", gomock.Any()).
			Return(nil)

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
			CreateWithAttributeIndexes(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
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
			UpdateWithAttributeIndexes(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, product *domain.Product, old, new map[string][]string) error {
				assert.Equal(t, "New Name", product.Name)
				assert.Equal(t, "new-name", product.Slug)
				assert.Equal(t, "admin_1", product.UpdatedBy)
				return nil
			})

		mockProdRepo.EXPECT().
			AddAttributeValues(ctx, "cat_123", gomock.Any()).
			Return(nil)

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
			UpdateWithAttributeIndexes(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil)

		mockProdRepo.EXPECT().
			AddAttributeValues(ctx, "cat_123", gomock.Any()).
			Return(nil)

		product, err := svc.Update(ctx, "prod_123", req, "admin_1")

		require.NoError(t, err)
		assert.NotNil(t, product)
	})
}

func TestProductService_Delete(t *testing.T) {
	t.Run("successful delete with attribute cleanup", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, mockInvRepo, _, ctx := setupProductTest(t)

		product := &domain.Product{
			ID:         "prod_123",
			SKU:        "BSS-001",
			CategoryID: "cat_123",
			Material:   "silk",
		}

		category := &domain.Category{
			ID: "cat_123",
			OwnAttributes: []domain.CategoryAttribute{
				{Name: "weave_pattern", Searchable: true},
			},
		}

		mockProdRepo.EXPECT().
			GetByID(ctx, "prod_123").
			Return(product, nil)

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(category, nil)

		mockProdRepo.EXPECT().
			DeleteWithAttributeIndexes(ctx, "prod_123", "BSS-001", gomock.Any()).
			Return(nil)

		mockInvRepo.EXPECT().
			DeleteByProductID(ctx, "prod_123").
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

	t.Run("delete when category missing - falls back to simple delete", func(t *testing.T) {
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

		mockProdRepo.EXPECT().
			Delete(ctx, "prod_123").
			Return(nil)

		mockInvRepo.EXPECT().
			DeleteByProductID(ctx, "prod_123").
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

	t.Run("list with attribute filters uses FilterByAttributes", func(t *testing.T) {
		svc, mockProdRepo, _, _, _, ctx := setupProductTest(t)

		catID := "cat_123"
		req := domain.ListProductsRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 10},
			CategoryID:        &catID,
			AttributeFilters:  map[string][]string{"color": {"red", "blue"}},
		}

		expected := &domain.ListProductsResponse{
			Products: []*domain.Product{{ID: "prod_1"}},
			Pagination: domain.PaginationResponse{Limit: 10, HasMore: false},
		}

		mockProdRepo.EXPECT().
			FilterByAttributes(ctx, "cat_123", req.AttributeFilters, req.PaginationRequest).
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

		allValues := map[string][]string{
			"color":   {"red", "blue", "green"},
			"size":    {"S", "M", "L"},
			"pattern": {"floral", "geometric"},
		}

		mockCatRepo.EXPECT().
			GetByID(ctx, "cat_123").
			Return(category, nil)

		mockProdRepo.EXPECT().
			GetAttributeValues(ctx, "cat_123").
			Return(allValues, nil)

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

func TestExtractSearchableAttributes(t *testing.T) {
	product := &domain.Product{
		Material:  "silk",
		Color:     "red",
		WeaveType: "jacquard",
		Origin:    "Varanasi",
		CraftType: "handloom",
		Attributes: map[string]interface{}{
			"pattern": "floral",
		},
	}

	categoryAttrs := []domain.CategoryAttribute{
		{Name: "pattern", Searchable: true},
		{Name: "style", Searchable: false}, // not searchable
	}

	result := extractSearchableAttributes(product, categoryAttrs)

	assert.Contains(t, result, "material")
	assert.Equal(t, []string{"silk"}, result["material"])
	assert.Contains(t, result, "color")
	assert.Contains(t, result, "pattern")
	assert.Equal(t, []string{"floral"}, result["pattern"])
	assert.NotContains(t, result, "style")
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
		{"integer", 42, []string{"42"}},
		{"float64", 3.14, []string{"3.14"}},
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
