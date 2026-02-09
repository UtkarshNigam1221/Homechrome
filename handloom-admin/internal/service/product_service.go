// Package service implements the business logic layer
package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
)

// ProductService implements domain.ProductService
type ProductService struct {
	productRepo   domain.ProductRepository
	categoryRepo  domain.CategoryRepository
	designRepo    domain.DesignRepository
	inventoryRepo domain.InventoryRepository
	logger        *logger.Logger
}

// NewProductService creates a new ProductService
func NewProductService(
	productRepo domain.ProductRepository,
	categoryRepo domain.CategoryRepository,
	designRepo domain.DesignRepository,
	inventoryRepo domain.InventoryRepository,
	logger *logger.Logger,
) *ProductService {
	return &ProductService{
		productRepo:   productRepo,
		categoryRepo:  categoryRepo,
		designRepo:    designRepo,
		inventoryRepo: inventoryRepo,
		logger:        logger,
	}
}

// Create creates a new product
func (s *ProductService) Create(ctx context.Context, req domain.CreateProductRequest, createdBy string) (*domain.Product, error) {
	// Validate category exists and get its attributes
	category, err := s.categoryRepo.GetByID(ctx, req.CategoryID)
	if err != nil {
		return nil, errors.New(errors.ErrCodeNotFound, "Category not found")
	}

	// Validate required searchable attributes are provided
	if err := validateRequiredAttributes(req.Attributes, category.OwnAttributes); err != nil {
		return nil, err
	}

	// Validate design exists
	design, err := s.designRepo.GetByID(ctx, req.DesignID)
	if err != nil {
		return nil, errors.New(errors.ErrCodeNotFound, "Design not found")
	}

	product := domain.NewProduct(req, "prod_"+uuid.New().String()[:8], generateSlug(req.Name), createdBy)

	// Extract searchable attributes for indexing
	searchableAttrs := extractSearchableAttributes(product, category.OwnAttributes)

	// Create product with attribute indexes
	if err := s.productRepo.CreateWithAttributeIndexes(ctx, product, searchableAttrs); err != nil {
		return nil, err
	}

	// Add attribute values to the stored distinct value sets for this category
	if len(searchableAttrs) > 0 {
		if err := s.productRepo.AddAttributeValues(ctx, product.CategoryID, searchableAttrs); err != nil {
			s.logger.WithContext(ctx).WithError(err).Error("Failed to update attribute value sets")
		}
	}

	// Create inventory record
	inventory := &domain.Inventory{
		ID:                product.ID,
		ProductID:         product.ID,
		ProductSKU:        product.SKU,
		ProductName:       product.Name,
		Quantity:          req.InitialStock,
		AvailableQty:      req.InitialStock,
		LowStockThreshold: req.LowStockThreshold,
	}
	inventory.CreatedBy = createdBy

	if err := s.inventoryRepo.Create(ctx, inventory); err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to create inventory record")
	}

	// Increment counts
	_ = s.categoryRepo.IncrementProductCount(ctx, category.ID, 1)
	_ = s.designRepo.IncrementProductCount(ctx, design.ID, 1)

	s.logger.WithContext(ctx).Infof("Created product: %s", product.ID)
	return product, nil
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

	// Get category info
	category, err := s.categoryRepo.GetByID(ctx, product.CategoryID)
	if err == nil {
		result.Category = &domain.CategorySummary{
			ID:   category.ID,
			Name: category.Name,
			Slug: category.Slug,
		}
	}

	// Get design info
	design, err := s.designRepo.GetByID(ctx, product.DesignID)
	if err == nil {
		result.Design = &domain.DesignSummary{
			ID:   design.ID,
			Name: design.Name,
			Slug: design.Slug,
		}
	}

	// Get inventory info
	inventory, err := s.inventoryRepo.GetByProductID(ctx, product.ID)
	if err == nil {
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

	// Get category for attribute management
	category, err := s.categoryRepo.GetByID(ctx, product.CategoryID)
	if err != nil {
		return nil, err
	}

	// Extract old searchable attributes before update
	oldSearchableAttrs := extractSearchableAttributes(product, category.OwnAttributes)

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

	// Extract new searchable attributes after update
	newSearchableAttrs := extractSearchableAttributes(product, category.OwnAttributes)

	// Update product with attribute index sync
	if err := s.productRepo.UpdateWithAttributeIndexes(ctx, product, oldSearchableAttrs, newSearchableAttrs); err != nil {
		return nil, err
	}

	// Add new attribute values to the stored distinct value sets
	if len(newSearchableAttrs) > 0 {
		if err := s.productRepo.AddAttributeValues(ctx, product.CategoryID, newSearchableAttrs); err != nil {
			s.logger.WithContext(ctx).WithError(err).Error("Failed to update attribute value sets")
		}
	}

	s.logger.WithContext(ctx).Infof("Updated product: %s", id)
	return product, nil
}

// Delete deletes a product by ID
func (s *ProductService) Delete(ctx context.Context, id string) error {
	product, err := s.productRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Get category for attribute cleanup
	category, err := s.categoryRepo.GetByID(ctx, product.CategoryID)
	if err != nil {
		// If category not found, still try to delete product
		if err := s.productRepo.Delete(ctx, id); err != nil {
			return err
		}
	} else {
		// Extract searchable attributes for cleanup
		searchableAttrs := extractSearchableAttributes(product, category.OwnAttributes)

		// Delete product with attribute indexes
		if err := s.productRepo.DeleteWithAttributeIndexes(ctx, id, product.SKU, searchableAttrs); err != nil {
			return err
		}
	}

	// Delete inventory record and its transactions
	if err := s.inventoryRepo.DeleteByProductID(ctx, id); err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to delete inventory for product")
	}

	// Decrement counts
	_ = s.categoryRepo.IncrementProductCount(ctx, product.CategoryID, -1)
	_ = s.designRepo.IncrementProductCount(ctx, product.DesignID, -1)

	s.logger.WithContext(ctx).Infof("Deleted product: %s", id)
	return nil
}

// List retrieves products with filters
func (s *ProductService) List(ctx context.Context, req domain.ListProductsRequest) (*domain.ListProductsResponse, error) {
	// If attribute filters are provided and category is specified, use the optimized filter method
	if len(req.AttributeFilters) > 0 && req.CategoryID != nil {
		return s.productRepo.FilterByAttributes(ctx, *req.CategoryID, req.AttributeFilters, req.PaginationRequest)
	}
	return s.productRepo.List(ctx, req)
}

// GetAttributeFilterOptions returns all distinct values for each searchable attribute in a category.
// Reads from the pre-computed CategoryAttributeValues record — a single DynamoDB GetItem.
func (s *ProductService) GetAttributeFilterOptions(ctx context.Context, categoryID string) (map[string][]string, error) {
	// Get category to know which attributes are searchable
	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	// Build set of searchable attribute names
	searchableSet := make(map[string]bool)
	for _, attr := range category.OwnAttributes {
		if attr.Searchable {
			searchableSet[attr.Name] = true
		}
	}

	// Read the stored distinct value sets (single GetItem)
	allValues, err := s.productRepo.GetAttributeValues(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	// Filter to only searchable attributes and sort the values
	result := make(map[string][]string)
	for attrName, values := range allValues {
		if !searchableSet[attrName] {
			continue
		}
		if len(values) > 0 {
			sort.Strings(values)
			result[attrName] = values
		}
	}

	return result, nil
}

// extractSearchableAttributes extracts searchable attribute values from a product
// Returns a map of attribute name -> list of values to index
func extractSearchableAttributes(product *domain.Product, categoryAttrs []domain.CategoryAttribute) map[string][]string {
	result := make(map[string][]string)

	// Build a set of searchable attribute names
	searchableAttrs := make(map[string]bool)
	for _, attr := range categoryAttrs {
		if attr.Searchable {
			searchableAttrs[attr.Name] = true
		}
	}

	// Fixed fields that are always indexed regardless of category definition
	fixedFields := map[string]string{
		"material":   product.Material,
		"color":      product.Color,
		"weave_type": product.WeaveType,
		"origin":     product.Origin,
		"craft_type": product.CraftType,
	}
	for name, value := range fixedFields {
		searchableAttrs[name] = true
		if value != "" {
			result[name] = []string{value}
		}
	}

	// Extract values from product's dynamic Attributes map
	if product.Attributes != nil {
		for attrName := range searchableAttrs {
			if value, exists := product.Attributes[attrName]; exists {
				values := normalizeToStringSlice(value)
				if len(values) > 0 {
					result[attrName] = values
				}
			}
		}
	}

	return result
}

// normalizeToStringSlice converts various types to a slice of strings
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
	case int, int64, float64:
		return []string{fmt.Sprintf("%v", v)}
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

// Ensure interface compliance
var _ domain.ProductService = (*ProductService)(nil)
