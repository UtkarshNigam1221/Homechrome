package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// CategoryService implements domain.CategoryService
type CategoryService struct {
	categoryRepo   domain.CategoryRepository
	productRepo    domain.ProductRepository
	assetFinalizer domain.AssetFinalizer
}

// NewCategoryService creates a new CategoryService
func NewCategoryService(
	categoryRepo domain.CategoryRepository,
	productRepo domain.ProductRepository,
	assetFinalizer domain.AssetFinalizer,
) *CategoryService {
	return &CategoryService{
		categoryRepo:   categoryRepo,
		productRepo:    productRepo,
		assetFinalizer: assetFinalizer,
	}
}

// finalizeImage resolves a tmp/ asset key into a permanent URL.
func (s *CategoryService) finalizeImage(ctx context.Context, url string) (string, error) {
	finalURL, err := s.assetFinalizer.FinalizeIfTemp(ctx, url)
	if err != nil {
		return "", errors.Wrap(err, "failed to finalize image")
	}
	return finalURL, nil
}

// findAttributeIndex returns the index of the attribute with the given name, or -1.
func findAttributeIndex(attrs []domain.CategoryAttribute, name string) int {
	for i, a := range attrs {
		if a.Name == name {
			return i
		}
	}
	return -1
}

// Create creates a new category
func (s *CategoryService) Create(ctx context.Context, req domain.CreateCategoryRequest, createdBy string) (*domain.Category, error) {
	if req.ImageURL != "" {
		finalURL, err := s.finalizeImage(ctx, req.ImageURL)
		if err != nil {
			return nil, err
		}
		req.ImageURL = finalURL
	}

	category := domain.NewCategory(req, "cat_"+uuid.New().String(), generateSlug(req.Name), createdBy)

	if err := s.categoryRepo.Create(ctx, category); err != nil {
		return nil, err
	}

	// Resize the newly added category image (if any).
	if category.ImageURL != "" {
		s.assetFinalizer.SyncImageVariants(ctx, nil, []string{category.ImageURL})
	}

	slog.InfoContext(ctx, "Created category", "category_id", category.ID)
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

	oldImageURL := category.ImageURL

	slug := category.Slug
	if req.Name != nil {
		slug = generateSlug(*req.Name)
	}
	category.ApplyUpdate(req, slug)

	if req.ImageURL != nil {
		finalURL, err := s.finalizeImage(ctx, *req.ImageURL)
		if err != nil {
			return nil, err
		}
		category.ImageURL = finalURL
	}
	category.UpdatedBy = updatedBy

	if err := s.categoryRepo.Update(ctx, category); err != nil {
		return nil, err
	}

	// Resize new image + cleanup old variants if image changed.
	if oldImageURL != category.ImageURL {
		var oldList, newList []string
		if oldImageURL != "" {
			oldList = []string{oldImageURL}
		}
		if category.ImageURL != "" {
			newList = []string{category.ImageURL}
		}
		s.assetFinalizer.SyncImageVariants(ctx, oldList, newList)
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

	if err := s.categoryRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Cleanup category image + variants (best-effort).
	if category.ImageURL != "" {
		s.assetFinalizer.SyncImageVariants(ctx, []string{category.ImageURL}, nil)
	}

	return nil
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

	if findAttributeIndex(category.OwnAttributes, attr.Name) != -1 {
		return nil, errors.Conflict("Attribute with this name already exists")
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

	idx := findAttributeIndex(category.OwnAttributes, attrName)
	if idx == -1 {
		return nil, errors.NotFound("Attribute")
	}
	category.OwnAttributes[idx] = attr

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

	idx := findAttributeIndex(category.OwnAttributes, attrName)
	if idx == -1 {
		return errors.NotFound("Attribute")
	}
	category.OwnAttributes = append(category.OwnAttributes[:idx], category.OwnAttributes[idx+1:]...)
	category.UpdatedBy = updatedBy

	if err := s.categoryRepo.Update(ctx, category); err != nil {
		return err
	}

	return nil
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

// Ensure interface compliance
var _ domain.CategoryService = (*CategoryService)(nil)
