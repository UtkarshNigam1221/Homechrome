package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
)

// CategoryService implements domain.CategoryService
type CategoryService struct {
	categoryRepo domain.CategoryRepository
	productRepo  domain.ProductRepository
	logger       *logger.Logger
}

// NewCategoryService creates a new CategoryService
func NewCategoryService(
	categoryRepo domain.CategoryRepository,
	productRepo domain.ProductRepository,
	logger *logger.Logger,
) *CategoryService {
	return &CategoryService{
		categoryRepo: categoryRepo,
		productRepo:  productRepo,
		logger:       logger,
	}
}

// Create creates a new category
func (s *CategoryService) Create(ctx context.Context, req domain.CreateCategoryRequest, createdBy string) (*domain.Category, error) {
	category := &domain.Category{
		ID:            "cat_" + uuid.New().String()[:8],
		Name:          req.Name,
		Slug:          generateSlug(req.Name),
		Description:   req.Description,
		ImageURL:      req.ImageURL,
		OwnAttributes: req.OwnAttributes,
		Status:        domain.CategoryStatusActive,
	}
	category.CreatedBy = createdBy

	if err := s.categoryRepo.Create(ctx, category); err != nil {
		return nil, err
	}

	s.logger.WithContext(ctx).Infof("Created category: %s", category.ID)
	return category, nil
}

// GetByID retrieves a category by ID
func (s *CategoryService) GetByID(ctx context.Context, id string) (*domain.Category, error) {
	return s.categoryRepo.GetByID(ctx, id)
}

// Update updates an existing category
func (s *CategoryService) Update(ctx context.Context, id string, req domain.UpdateCategoryRequest, updatedBy string) (*domain.Category, error) {
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		category.Name = *req.Name
		category.Slug = generateSlug(*req.Name)
	}
	if req.Description != nil {
		category.Description = *req.Description
	}
	if req.ImageURL != nil {
		category.ImageURL = *req.ImageURL
	}
	if req.Status != nil {
		category.Status = *req.Status
	}
	if req.OwnAttributes != nil {
		category.OwnAttributes = req.OwnAttributes
	}
	category.UpdatedBy = updatedBy

	if err := s.categoryRepo.Update(ctx, category); err != nil {
		return nil, err
	}

	return category, nil
}

// Delete deletes a category by ID
func (s *CategoryService) Delete(ctx context.Context, id string) error {
	category, err := s.categoryRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Check for products
	if category.ProductCount > 0 {
		return errors.New(errors.ErrCodeCategoryHasProducts, "Cannot delete category with existing products")
	}

	return s.categoryRepo.Delete(ctx, id)
}

// List retrieves categories with filters
func (s *CategoryService) List(ctx context.Context, req domain.ListCategoriesRequest) (*domain.ListCategoriesResponse, error) {
	return s.categoryRepo.List(ctx, req)
}

// AddAttribute adds a new attribute to a category
func (s *CategoryService) AddAttribute(ctx context.Context, categoryID string, attr domain.CategoryAttribute, updatedBy string) (*domain.Category, error) {
	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	// Check if attribute already exists
	for _, existing := range category.OwnAttributes {
		if existing.Name == attr.Name {
			return nil, errors.Conflict("Attribute with this name already exists")
		}
	}

	category.OwnAttributes = append(category.OwnAttributes, attr)
	category.UpdatedBy = updatedBy

	if err := s.categoryRepo.Update(ctx, category); err != nil {
		return nil, err
	}

	return category, nil
}

// UpdateAttribute updates an existing attribute
func (s *CategoryService) UpdateAttribute(ctx context.Context, categoryID string, attrName string, attr domain.CategoryAttribute, updatedBy string) (*domain.Category, error) {
	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	found := false
	for i, existing := range category.OwnAttributes {
		if existing.Name == attrName {
			category.OwnAttributes[i] = attr
			found = true
			break
		}
	}

	if !found {
		return nil, errors.NotFound("Attribute")
	}

	category.UpdatedBy = updatedBy

	if err := s.categoryRepo.Update(ctx, category); err != nil {
		return nil, err
	}

	return category, nil
}

// DeleteAttribute removes an attribute from a category
func (s *CategoryService) DeleteAttribute(ctx context.Context, categoryID string, attrName string, updatedBy string) error {
	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return err
	}

	found := false
	var newAttrs []domain.CategoryAttribute
	for _, existing := range category.OwnAttributes {
		if existing.Name == attrName {
			found = true
			continue
		}
		newAttrs = append(newAttrs, existing)
	}

	if !found {
		return errors.NotFound("Attribute")
	}

	category.OwnAttributes = newAttrs
	category.UpdatedBy = updatedBy

	return s.categoryRepo.Update(ctx, category)
}

// GetAttributes retrieves all attributes for a category
func (s *CategoryService) GetAttributes(ctx context.Context, categoryID string) (*domain.CategoryAttributesResponse, error) {
	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	return &domain.CategoryAttributesResponse{
		OwnAttributes: category.OwnAttributes,
		TotalCount:    len(category.OwnAttributes),
	}, nil
}

// generateSlug creates a URL-friendly slug from a name
func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Remove special characters
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// Ensure interface compliance
var _ domain.CategoryService = (*CategoryService)(nil)
