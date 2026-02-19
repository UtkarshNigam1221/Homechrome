// Package service implements the business logic layer
package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/logger"
)

// ArtisanService implements domain.ArtisanService
type ArtisanService struct {
	artisanRepo domain.ArtisanRepository
	logger      *logger.Logger
}

// NewArtisanService creates a new ArtisanService
func NewArtisanService(artisanRepo domain.ArtisanRepository, logger *logger.Logger) *ArtisanService {
	return &ArtisanService{
		artisanRepo: artisanRepo,
		logger:      logger,
	}
}

// Create creates a new artisan
func (s *ArtisanService) Create(ctx context.Context, req domain.CreateArtisanRequest, createdBy string) (*domain.Artisan, error) {
	artisan := &domain.Artisan{
		ID:              "artisan_" + uuid.New().String()[:8],
		Name:            req.Name,
		Email:           req.Email,
		Phone:           req.Phone,
		ProfileImage:    req.ProfileImage,
		Bio:             req.Bio,
		Address:         req.Address,
		Location:        req.Location,
		CraftTypes:      req.CraftTypes,
		Specializations: req.Specializations,
		Experience:      req.Experience,
		BankDetails:     req.BankDetails,
		CommissionRate:  req.CommissionRate,
		ProductCount:    0,
		TotalSales:      0,
		TotalEarnings:   0,
		PendingPayout:   0,
		Status:          domain.ArtisanStatusPending,
	}
	artisan.CreatedBy = createdBy

	if err := s.artisanRepo.Create(ctx, artisan); err != nil {
		return nil, err
	}

	s.logger.WithContext(ctx).Infof("Created artisan: %s", artisan.ID)
	return artisan, nil
}

// GetByID retrieves an artisan by ID
func (s *ArtisanService) GetByID(ctx context.Context, id string) (*domain.Artisan, error) {
	return s.artisanRepo.GetByID(ctx, id)
}

// Update updates an existing artisan
func (s *ArtisanService) Update(ctx context.Context, id string, req domain.UpdateArtisanRequest, updatedBy string) (*domain.Artisan, error) {
	artisan, err := s.artisanRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		artisan.Name = *req.Name
	}
	if req.Email != nil {
		artisan.Email = *req.Email
	}
	if req.Phone != nil {
		artisan.Phone = *req.Phone
	}
	if req.ProfileImage != nil {
		artisan.ProfileImage = *req.ProfileImage
	}
	if req.Bio != nil {
		artisan.Bio = *req.Bio
	}
	if req.Address != nil {
		artisan.Address = req.Address
	}
	if req.Location != nil {
		artisan.Location = *req.Location
	}
	if req.CraftTypes != nil {
		artisan.CraftTypes = req.CraftTypes
	}
	if req.Specializations != nil {
		artisan.Specializations = req.Specializations
	}
	if req.Experience != nil {
		artisan.Experience = *req.Experience
	}
	if req.BankDetails != nil {
		artisan.BankDetails = req.BankDetails
	}
	if req.CommissionRate != nil {
		artisan.CommissionRate = *req.CommissionRate
	}
	if req.Status != nil {
		artisan.Status = *req.Status
	}

	artisan.UpdatedBy = updatedBy
	artisan.UpdatedAt = time.Now()

	if err := s.artisanRepo.Update(ctx, artisan); err != nil {
		return nil, err
	}

	s.logger.WithContext(ctx).Infof("Updated artisan: %s", id)
	return artisan, nil
}

// Delete deletes an artisan by ID
func (s *ArtisanService) Delete(ctx context.Context, id string) error {
	if err := s.artisanRepo.Delete(ctx, id); err != nil {
		return err
	}

	s.logger.WithContext(ctx).Infof("Deleted artisan: %s", id)
	return nil
}

// List retrieves artisans with filters
func (s *ArtisanService) List(ctx context.Context, req domain.ListArtisansRequest) (*domain.ListArtisansResponse, error) {
	return s.artisanRepo.List(ctx, req)
}

// UpdateStatus updates artisan status
func (s *ArtisanService) UpdateStatus(ctx context.Context, id string, status domain.ArtisanStatus, updatedBy string) error {
	artisan, err := s.artisanRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	artisan.Status = status
	artisan.UpdatedBy = updatedBy
	artisan.UpdatedAt = time.Now()

	if err := s.artisanRepo.Update(ctx, artisan); err != nil {
		return err
	}

	s.logger.WithContext(ctx).Infof("Updated artisan status: %s -> %s", id, status)
	return nil
}

// GetPayouts retrieves payouts for an artisan
func (s *ArtisanService) GetPayouts(ctx context.Context, artisanID string, pagination domain.PaginationRequest) (*domain.ListArtisanPayoutsResponse, error) {
	return s.artisanRepo.GetPayouts(ctx, artisanID, pagination)
}

// CreatePayout creates a payout for an artisan
func (s *ArtisanService) CreatePayout(ctx context.Context, req domain.CreatePayoutRequest, createdBy string) (*domain.ArtisanPayout, error) {
	// Verify artisan exists
	artisan, err := s.artisanRepo.GetByID(ctx, req.ArtisanID)
	if err != nil {
		return nil, err
	}

	payout := &domain.ArtisanPayout{
		ID:            "payout_" + uuid.New().String()[:8],
		ArtisanID:     req.ArtisanID,
		Amount:        req.Amount,
		Status:        "PENDING",
		PaymentMethod: req.PaymentMethod,
		PeriodStart:   req.PeriodStart,
		PeriodEnd:     req.PeriodEnd,
		OrderIDs:      req.OrderIDs,
		OrderCount:    len(req.OrderIDs),
		CreatedAt:     time.Now(),
		CreatedBy:     createdBy,
	}

	if err := s.artisanRepo.CreatePayout(ctx, payout); err != nil {
		return nil, err
	}

	// Update pending payout amount
	artisan.PendingPayout -= req.Amount
	if artisan.PendingPayout < 0 {
		artisan.PendingPayout = 0
	}
	_ = s.artisanRepo.Update(ctx, artisan)

	s.logger.WithContext(ctx).Infof("Created payout for artisan %s: %d", req.ArtisanID, req.Amount)
	return payout, nil
}

// GetProducts retrieves products for an artisan
func (s *ArtisanService) GetProducts(ctx context.Context, artisanID string, pagination domain.PaginationRequest) (*domain.ListProductsResponse, error) {
	return s.artisanRepo.GetProducts(ctx, artisanID, pagination)
}

// Search searches artisans by query
func (s *ArtisanService) Search(ctx context.Context, query string, pagination domain.PaginationRequest) (*domain.ListArtisansResponse, error) {
	return s.artisanRepo.Search(ctx, query, pagination)
}

// Ensure interface compliance
var _ domain.ArtisanService = (*ArtisanService)(nil)
