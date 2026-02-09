// Package service implements the business logic layer
package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
)

// DesignService implements domain.DesignService
type DesignService struct {
	designRepo   domain.DesignRepository
	categoryRepo domain.CategoryRepository
	logger       *logger.Logger
}

// NewDesignService creates a new DesignService
func NewDesignService(
	designRepo domain.DesignRepository,
	categoryRepo domain.CategoryRepository,
	logger *logger.Logger,
) *DesignService {
	return &DesignService{
		designRepo:   designRepo,
		categoryRepo: categoryRepo,
		logger:       logger,
	}
}

// Create creates a new design
func (s *DesignService) Create(ctx context.Context, req domain.CreateDesignRequest, createdBy string) (*domain.Design, error) {
	// Validate category exists
	_, err := s.categoryRepo.GetByID(ctx, req.CategoryID)
	if err != nil {
		return nil, errors.New(errors.ErrCodeNotFound, "Category not found")
	}

	design := &domain.Design{
		ID:          "design_" + uuid.New().String()[:8],
		Name:        req.Name,
		Slug:        generateSlug(req.Name),
		CategoryID:  req.CategoryID,
		Description: req.Description,
		Images:      req.Images,
		Attributes:  req.Attributes,
		Status:      domain.DesignStatusActive,
	}
	design.CreatedBy = createdBy

	if err := s.designRepo.Create(ctx, design); err != nil {
		return nil, err
	}

	s.logger.WithContext(ctx).Infof("Created design: %s", design.ID)
	return design, nil
}

// GetByID retrieves a design by ID
func (s *DesignService) GetByID(ctx context.Context, id string) (*domain.DesignWithCategory, error) {
	design, err := s.designRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	result := &domain.DesignWithCategory{
		Design: design,
	}

	// Get category info
	category, err := s.categoryRepo.GetByID(ctx, design.CategoryID)
	if err == nil {
		result.Category = &domain.CategorySummary{
			ID:   category.ID,
			Name: category.Name,
			Slug: category.Slug,
		}
	}

	return result, nil
}

// Update updates an existing design
func (s *DesignService) Update(ctx context.Context, id string, req domain.UpdateDesignRequest, updatedBy string) (*domain.Design, error) {
	design, err := s.designRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		design.Name = *req.Name
		design.Slug = generateSlug(*req.Name)
	}
	if req.Description != nil {
		design.Description = *req.Description
	}
	if req.Images != nil {
		design.Images = req.Images
	}
	if req.Attributes != nil {
		design.Attributes = req.Attributes
	}
	if req.Status != nil {
		design.Status = *req.Status
	}

	design.UpdatedBy = updatedBy
	design.UpdatedAt = time.Now()

	if err := s.designRepo.Update(ctx, design); err != nil {
		return nil, err
	}

	s.logger.WithContext(ctx).Infof("Updated design: %s", id)
	return design, nil
}

// Delete deletes a design by ID
func (s *DesignService) Delete(ctx context.Context, id string) error {
	design, err := s.designRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if design has products
	if design.ProductCount > 0 {
		return errors.New(errors.ErrCodeHasDependencies, "Cannot delete design with associated products")
	}

	if err := s.designRepo.Delete(ctx, id); err != nil {
		return err
	}

	s.logger.WithContext(ctx).Infof("Deleted design: %s", id)
	return nil
}

// List retrieves designs with filters
func (s *DesignService) List(ctx context.Context, req domain.ListDesignsRequest) (*domain.ListDesignsResponse, error) {
	return s.designRepo.List(ctx, req)
}

// Ensure interface compliance
var _ domain.DesignService = (*DesignService)(nil)
