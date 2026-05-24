// Package service implements the business logic layer
package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/event"
	"github.com/handloom/admin/pkg/errors"
)

// ProductService implements domain.ProductService
type ProductService struct {
	productRepo    domain.ProductRepository
	categoryRepo   domain.CategoryRepository
	inventoryRepo  domain.InventoryRepository
	assetFinalizer domain.AssetFinalizer
	publisher      event.EventPublisher
}

// NewProductService creates a new ProductService
func NewProductService(
	productRepo domain.ProductRepository,
	categoryRepo domain.CategoryRepository,
	inventoryRepo domain.InventoryRepository,
	assetFinalizer domain.AssetFinalizer,
	publisher event.EventPublisher,
) *ProductService {
	return &ProductService{
		productRepo:    productRepo,
		categoryRepo:   categoryRepo,
		inventoryRepo:  inventoryRepo,
		assetFinalizer: assetFinalizer,
		publisher:      publisher,
	}
}

// Create creates a new product
func (s *ProductService) Create(ctx context.Context, req domain.CreateProductRequest, createdBy string) (*domain.Product, error) {
	// Validate category exists
	category, err := s.categoryRepo.GetByID(ctx, req.CategoryID)
	if err != nil {
		return nil, errors.New(errors.ErrCodeNotFound, "Category not found")
	}

	// Validate required searchable attributes are provided
	if err := validateRequiredAttributes(req.Attributes, category.OwnAttributes); err != nil {
		return nil, err
	}

	// Finalize any tmp/ image keys to permanent assets/ URLs
	for i, img := range req.Images {
		finalURL, err := s.assetFinalizer.FinalizeIfTemp(ctx, img.URL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to finalize image")
		}
		req.Images[i].URL = finalURL
	}

	if req.VideoURL != "" {
		finalURL, err := s.assetFinalizer.FinalizeIfTemp(ctx, req.VideoURL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to finalize video")
		}
		req.VideoURL = finalURL
	}
	if req.VideoPosterURL != "" {
		finalURL, err := s.assetFinalizer.FinalizeIfTemp(ctx, req.VideoPosterURL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to finalize video poster")
		}
		req.VideoPosterURL = finalURL
	}

	product := domain.NewProduct(req, "prod_"+uuid.New().String(), generateSlug(req.Name), createdBy)

	// Build inventory record to include in the same transaction
	inventory := &domain.Inventory{
		ID:                product.ID,
		ProductID:         product.ID,
		Quantity:          req.InitialStock,
		AvailableQty:      req.InitialStock,
		LowStockThreshold: req.LowStockThreshold,
	}
	inventory.CreatedBy = createdBy

	// Create product and inventory atomically
	if err := s.productRepo.Create(ctx, product, inventory); err != nil {
		return nil, err
	}

	// Increment category product count
	if err := s.categoryRepo.IncrementProductCount(ctx, category.ID, 1); err != nil {
		return nil, errors.Wrap(err, "failed to increment category product count")
	}

	// Generate image variants for all new product images (best-effort).
	s.assetFinalizer.SyncImageVariants(ctx, nil, imageURLs(product.Images))

	if pubErr := s.publisher.Publish(ctx, event.New(event.ProductCreated, product)); pubErr != nil {
		slog.ErrorContext(ctx, "Failed to publish product.created event", "error", pubErr)
	}

	slog.InfoContext(ctx, "Created product", keyProductID, product.ID)
	return product, nil
}

// imageURLs extracts the URL field from a ProductImage slice.
func imageURLs(images []domain.ProductImage) []string {
	urls := make([]string, 0, len(images))
	for _, img := range images {
		if img.URL != "" {
			urls = append(urls, img.URL)
		}
	}
	return urls
}

// GetByID retrieves a product by ID
func (s *ProductService) GetByID(ctx context.Context, id string) (*domain.ProductWithRelations, error) {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	result := &domain.ProductWithRelations{
		Product: product,
	}

	// Fetch category and inventory concurrently — they are independent
	var category *domain.Category
	var inventory *domain.Inventory

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var catErr error
		category, catErr = s.categoryRepo.GetByID(gctx, product.CategoryID)
		return catErr
	})
	g.Go(func() error {
		var invErr error
		inventory, invErr = s.inventoryRepo.GetByProductID(gctx, product.ID)
		return invErr
	})
	// Errors from enrichment calls are non-fatal; ignore them so partial
	// data is still returned (mirrors the prior sequential behavior).
	_ = g.Wait()

	if category != nil {
		result.Category = &domain.CategorySummary{
			ID:   category.ID,
			Name: category.Name,
			Slug: category.Slug,
		}
	}
	if inventory != nil {
		result.Inventory = inventory
	}

	return result, nil
}

// Update updates an existing product
func (s *ProductService) Update(ctx context.Context, id string, req domain.UpdateProductRequest, updatedBy string) (*domain.Product, error) {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Snapshot existing image URLs before mutation so we can diff after the write.
	oldImageURLs := imageURLs(product.Images)

	// Get category for attribute validation
	category, err := s.categoryRepo.GetByID(ctx, product.CategoryID)
	if err != nil {
		return nil, err
	}

	// Finalize any tmp/ image keys in the update request
	for i, img := range req.Images {
		finalURL, err := s.assetFinalizer.FinalizeIfTemp(ctx, img.URL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to finalize image")
		}
		req.Images[i].URL = finalURL
	}

	// Capture old asset URLs BEFORE applying the update so we can delete them
	// AFTER the DB write succeeds (avoids orphaned deletes on DB failure).
	var oldVideoURL, oldPosterURL string
	if req.VideoURL != nil {
		oldVideoURL = product.VideoURL
		newURL, err := s.assetFinalizer.FinalizeIfTemp(ctx, *req.VideoURL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to finalize video")
		}
		*req.VideoURL = newURL
	}
	if req.VideoPosterURL != nil {
		oldPosterURL = product.VideoPosterURL
		newURL, err := s.assetFinalizer.FinalizeIfTemp(ctx, *req.VideoPosterURL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to finalize video poster")
		}
		*req.VideoPosterURL = newURL
	}

	// Apply updates
	product.ApplyUpdate(req)
	if req.Name != nil {
		product.Slug = generateSlug(*req.Name)
	}

	// Validate required searchable attributes (after applying updates)
	if err := validateRequiredAttributes(product.Attributes, category.OwnAttributes); err != nil {
		return nil, err
	}

	product.UpdatedBy = updatedBy
	product.UpdatedAt = time.Now()

	// Update product (repository handles attribute value sync)
	if err := s.productRepo.Update(ctx, product); err != nil {
		return nil, err
	}

	// DB write succeeded — now it is safe to delete replaced S3 assets.
	if oldVideoURL != "" && oldVideoURL != product.VideoURL {
		if delErr := s.assetFinalizer.DeleteAsset(ctx, oldVideoURL); delErr != nil {
			slog.WarnContext(ctx, "Failed to delete old product video", "url", oldVideoURL, "error", delErr)
		}
	}
	if oldPosterURL != "" && oldPosterURL != product.VideoPosterURL {
		if delErr := s.assetFinalizer.DeleteAsset(ctx, oldPosterURL); delErr != nil {
			slog.WarnContext(ctx, "Failed to delete old product video poster", "url", oldPosterURL, "error", delErr)
		}
	}

	// Resize newly added product images + clean up variants for removed ones.
	s.assetFinalizer.SyncImageVariants(ctx, oldImageURLs, imageURLs(product.Images))

	if pubErr := s.publisher.Publish(ctx, event.New(event.ProductUpdated, product)); pubErr != nil {
		slog.ErrorContext(ctx, "Failed to publish product.updated event", "error", pubErr)
	}

	slog.InfoContext(ctx, "Updated product", keyProductID, id)
	return product, nil
}

// Delete deletes a product by ID
func (s *ProductService) Delete(ctx context.Context, id string) error {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete product (CASCADE removes inventory, attribute values, images)
	if err := s.productRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Decrement category count before revalidation so the storefront sees the
	// updated count when it next fetches the category.
	if err := s.categoryRepo.IncrementProductCount(ctx, product.CategoryID, -1); err != nil {
		return errors.Wrap(err, "failed to decrement category product count")
	}

	// Best-effort cleanup of S3 assets — never fail the delete on cleanup errors.
	s.deleteProductAssets(ctx, product)

	if pubErr := s.publisher.Publish(ctx, event.New(event.ProductDeleted, map[string]interface{}{
		keyProductID: id,
	})); pubErr != nil {
		slog.ErrorContext(ctx, "Failed to publish product.deleted event", "error", pubErr)
	}

	slog.InfoContext(ctx, "Deleted product", keyProductID, id)
	return nil
}

// deleteProductAssets removes all S3 assets associated with a product.
// Images go through SyncImageVariants so resized variants are cleaned up too.
// Video + poster use DeleteAsset (no variants to worry about).
// Errors are logged as warnings — they never propagate to the caller.
func (s *ProductService) deleteProductAssets(ctx context.Context, p *domain.Product) {
	// Images + variants
	s.assetFinalizer.SyncImageVariants(ctx, imageURLs(p.Images), nil)

	// Video + poster
	for _, url := range []string{p.VideoURL, p.VideoPosterURL} {
		if url == "" {
			continue
		}
		if err := s.assetFinalizer.DeleteAsset(ctx, url); err != nil {
			slog.WarnContext(ctx, "Failed to delete product asset", "url", url, "error", err)
		}
	}
}

// List retrieves products with filters
func (s *ProductService) List(ctx context.Context, req domain.ListProductsRequest) (*domain.ListProductsResponse, error) {
	return s.productRepo.List(ctx, req)
}

// GetAttributeFilterOptions returns all distinct values for each searchable attribute in a category.
func (s *ProductService) GetAttributeFilterOptions(ctx context.Context, categoryID string) (map[string][]string, error) {
	// Get category to know which attributes are searchable
	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	// Collect searchable attribute names
	var attrNames []string
	for _, attr := range category.OwnAttributes {
		if attr.Searchable {
			attrNames = append(attrNames, attr.Name)
		}
	}

	if len(attrNames) == 0 {
		return map[string][]string{}, nil
	}

	// Query distinct values from the product_attribute_values table
	result, err := s.productRepo.GetAttributeFilterOptions(ctx, categoryID, attrNames)
	if err != nil {
		return nil, err
	}

	// Sort each attribute's values
	for _, values := range result {
		sort.Strings(values)
	}

	return result, nil
}

// normalizeToStringSlice converts various types to a slice of strings for attribute validation.
func normalizeToStringSlice(value interface{}) []string {
	switch v := value.(type) {
	case string:
		if v != "" {
			return []string{v}
		}
	case []string:
		return v
	case []interface{}:
		var result []string
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}

// validateRequiredAttributes validates that all required searchable attributes are provided
func validateRequiredAttributes(productAttrs map[string]interface{}, categoryAttrs []domain.CategoryAttribute) error {
	var missingAttrs []string

	for _, attr := range categoryAttrs {
		if !attr.Required || !attr.Searchable {
			continue
		}

		// Check if attribute is provided
		hasValue := false
		if productAttrs != nil {
			if value, exists := productAttrs[attr.Name]; exists {
				// Check if value is non-empty
				values := normalizeToStringSlice(value)
				if len(values) > 0 {
					hasValue = true
				}
			}
		}

		if !hasValue {
			missingAttrs = append(missingAttrs, attr.Label)
		}
	}

	if len(missingAttrs) > 0 {
		return errors.New(errors.ErrCodeValidation, fmt.Sprintf("Missing required attributes: %v", missingAttrs))
	}

	return nil
}

// ReorderProducts sets the display order for products in a category.
// productIDs is the desired order — position 0 gets SortOrder=1, etc.
func (s *ProductService) ReorderProducts(ctx context.Context, categoryID string, productIDs []string) (int, error) {
	// Validate category exists
	if _, err := s.categoryRepo.GetByID(ctx, categoryID); err != nil {
		return 0, errors.New(errors.ErrCodeNotFound, "Category not found")
	}

	// Fetch all products in the category to validate ownership
	existing, err := s.productRepo.GetByCategoryAll(ctx, categoryID)
	if err != nil {
		return 0, err
	}

	existingMap := make(map[string]*domain.Product, len(existing))
	for _, p := range existing {
		existingMap[p.ID] = p
	}

	// Check for duplicate IDs in the request
	seen := make(map[string]bool, len(productIDs))
	for _, id := range productIDs {
		if seen[id] {
			return 0, errors.New(errors.ErrCodeValidation, "Duplicate product ID: "+id)
		}
		seen[id] = true
	}

	// Validate all requested IDs belong to this category
	for _, id := range productIDs {
		if _, ok := existingMap[id]; !ok {
			return 0, errors.New(errors.ErrCodeValidation, "Product "+id+" does not belong to category "+categoryID)
		}
	}

	// Build updates: requested IDs get sequential sort_order starting at 1
	var toUpdate []*domain.Product
	assigned := make(map[string]bool)

	for i, id := range productIDs {
		p := existingMap[id]
		p.SortOrder = i + 1
		toUpdate = append(toUpdate, p)
		assigned[id] = true
	}

	// Any products not in the request list get sort_order after the ranked ones
	nextOrder := len(productIDs) + 1
	for _, p := range existing {
		if !assigned[p.ID] {
			p.SortOrder = nextOrder
			toUpdate = append(toUpdate, p)
			nextOrder++
		}
	}

	if err := s.productRepo.BatchUpdateSortOrder(ctx, toUpdate); err != nil {
		return 0, err
	}

	slog.InfoContext(ctx, "Reordered products", "count", len(toUpdate), "category_id", categoryID)
	return len(toUpdate), nil
}

// Ensure interface compliance
var _ domain.ProductService = (*ProductService)(nil)
