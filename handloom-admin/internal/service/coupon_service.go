// Package service implements the business logic layer
package service

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/metrics"
)

// coupon validation outcome label (repeated across validation branches).
const outcomeInvalid = "invalid"

// CouponService implements domain.CouponService
type CouponService struct {
	couponRepo domain.CouponRepository
}

// NewCouponService creates a new CouponService
func NewCouponService(couponRepo domain.CouponRepository) *CouponService {
	return &CouponService{
		couponRepo: couponRepo,
	}
}

// Create creates a new coupon
func (s *CouponService) Create(ctx context.Context, req domain.CreateCouponRequest, createdBy string) (*domain.Coupon, error) {
	// Check if coupon code already exists
	existing, err := s.couponRepo.GetByCode(ctx, strings.ToUpper(req.Code))
	if err == nil && existing != nil {
		return nil, errors.New(errors.ErrCodeAlreadyExists, "Coupon code already exists")
	}

	// Validate dates
	if req.ValidUntil.Before(req.ValidFrom) {
		return nil, errors.Validation("Valid until date must be after valid from date")
	}

	coupon := &domain.Coupon{
		ID:                   "coupon_" + uuid.New().String()[:8],
		Code:                 strings.ToUpper(req.Code),
		Name:                 req.Name,
		Description:          req.Description,
		Type:                 req.Type,
		Value:                req.Value,
		MinOrderValue:        req.MinOrderValue,
		MaxDiscount:          req.MaxDiscount,
		UsageLimit:           req.UsageLimit,
		UsagePerUser:         req.UsagePerUser,
		UsageCount:           0,
		ApplicableCategories: req.ApplicableCategories,
		ApplicableProducts:   req.ApplicableProducts,
		ExcludedCategories:   req.ExcludedCategories,
		ExcludedProducts:     req.ExcludedProducts,
		ValidFrom:            req.ValidFrom,
		ValidUntil:           req.ValidUntil,
		Status:               domain.CouponStatusActive,
	}
	coupon.CreatedBy = createdBy

	if err := s.couponRepo.Create(ctx, coupon); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Created coupon", "coupon_id", coupon.ID, "code", coupon.Code)
	return coupon, nil
}

// GetByID retrieves a coupon by ID
func (s *CouponService) GetByID(ctx context.Context, id string) (*domain.Coupon, error) {
	return s.couponRepo.GetByID(ctx, id)
}

// GetByCode retrieves a coupon by code
func (s *CouponService) GetByCode(ctx context.Context, code string) (*domain.Coupon, error) {
	return s.couponRepo.GetByCode(ctx, strings.ToUpper(code))
}

// Update updates an existing coupon
func (s *CouponService) Update(ctx context.Context, id string, req domain.UpdateCouponRequest, updatedBy string) (*domain.Coupon, error) {
	coupon, err := s.couponRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		coupon.Name = *req.Name
	}
	if req.Description != nil {
		coupon.Description = *req.Description
	}
	if req.MinOrderValue != nil {
		coupon.MinOrderValue = *req.MinOrderValue
	}
	if req.MaxDiscount != nil {
		coupon.MaxDiscount = *req.MaxDiscount
	}
	if req.UsageLimit != nil {
		coupon.UsageLimit = *req.UsageLimit
	}
	if req.UsagePerUser != nil {
		coupon.UsagePerUser = *req.UsagePerUser
	}
	if req.ApplicableCategories != nil {
		coupon.ApplicableCategories = req.ApplicableCategories
	}
	if req.ApplicableProducts != nil {
		coupon.ApplicableProducts = req.ApplicableProducts
	}
	if req.ExcludedCategories != nil {
		coupon.ExcludedCategories = req.ExcludedCategories
	}
	if req.ExcludedProducts != nil {
		coupon.ExcludedProducts = req.ExcludedProducts
	}
	if req.ValidFrom != nil {
		coupon.ValidFrom = *req.ValidFrom
	}
	if req.ValidUntil != nil {
		coupon.ValidUntil = *req.ValidUntil
	}
	if req.Status != nil {
		coupon.Status = *req.Status
	}

	coupon.UpdatedBy = updatedBy
	coupon.UpdatedAt = time.Now()

	if err := s.couponRepo.Update(ctx, coupon); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "Updated coupon", "coupon_id", id)
	return coupon, nil
}

// Delete deletes a coupon by ID
func (s *CouponService) Delete(ctx context.Context, id string) error {
	if err := s.couponRepo.Delete(ctx, id); err != nil {
		return err
	}

	slog.InfoContext(ctx, "Deleted coupon", "coupon_id", id)
	return nil
}

// List retrieves coupons with filters
func (s *CouponService) List(ctx context.Context, req domain.ListCouponsRequest) (*domain.ListCouponsResponse, error) {
	return s.couponRepo.List(ctx, req)
}

// Validate validates a coupon for an order
func (s *CouponService) Validate(ctx context.Context, code string, orderTotal int64, customerID string, lines []domain.CouponLine) (*domain.CouponValidationResult, error) {
	// Normalise the code label to keep metric cardinality bounded to real codes.
	codeLabel := strings.ToUpper(strings.TrimSpace(code))
	if codeLabel == "" {
		codeLabel = "empty"
	}

	outcome := "valid"
	defer func() {
		label := codeLabel
		if outcome != "valid" {
			label = "rejected"
		}
		metrics.Record(ctx, "coupon_applied", metrics.L{
			metrics.LabelCouponCode: label, metrics.LabelOutcome: outcome,
		})
	}()

	coupon, _ := s.couponRepo.GetByCode(ctx, strings.ToUpper(code))
	if coupon == nil {
		outcome = outcomeInvalid
		return s.invalidCouponResult(code, "Coupon not found"), nil
	}

	// Check status
	if coupon.Status != domain.CouponStatusActive {
		outcome = outcomeInvalid
		return &domain.CouponValidationResult{
			Valid:        false,
			Code:         code,
			ErrorMessage: "Coupon is not active",
		}, nil
	}

	// Check validity dates
	now := time.Now()
	if now.Before(coupon.ValidFrom) {
		outcome = outcomeInvalid
		return &domain.CouponValidationResult{
			Valid:        false,
			Code:         code,
			ErrorMessage: "Coupon is not yet valid",
		}, nil
	}
	if now.After(coupon.ValidUntil) {
		outcome = "expired"
		return &domain.CouponValidationResult{
			Valid:        false,
			Code:         code,
			ErrorMessage: "Coupon has expired",
		}, nil
	}

	// Check minimum order value
	if orderTotal < coupon.MinOrderValue {
		outcome = outcomeInvalid
		return &domain.CouponValidationResult{
			Valid:        false,
			Code:         code,
			ErrorMessage: "Order total does not meet minimum requirement",
		}, nil
	}

	// Check usage limits
	if coupon.UsageLimit > 0 && coupon.UsageCount >= coupon.UsageLimit {
		outcome = "limit_reached"
		return &domain.CouponValidationResult{
			Valid:        false,
			Code:         code,
			ErrorMessage: "Coupon usage limit reached",
		}, nil
	}

	// Check per-user usage limit
	if coupon.UsagePerUser > 0 && customerID != "" {
		userUsageCount, err := s.couponRepo.GetUserUsageCount(ctx, coupon.ID, customerID)
		if err == nil && userUsageCount >= coupon.UsagePerUser {
			outcome = "limit_reached"
			return &domain.CouponValidationResult{
				Valid:        false,
				Code:         code,
				ErrorMessage: "You have already used this coupon the maximum number of times",
			}, nil
		}
	}

	// Applicability. The lists were stored and editable long before anything read
	// them, so a coupon scoped to one category discounted every order.
	if reason := checkApplicability(coupon, lines); reason != "" {
		outcome = outcomeInvalid
		return &domain.CouponValidationResult{Valid: false, Code: code, ErrorMessage: reason}, nil
	}

	// Calculate discount
	var discount int64
	if coupon.Type == domain.CouponTypePercentage {
		discount = orderTotal * coupon.Value / 10000 // Value is percentage * 100
		if coupon.MaxDiscount > 0 && discount > coupon.MaxDiscount {
			discount = coupon.MaxDiscount
		}
	} else {
		discount = coupon.Value
	}
	// Never more than the order is worth. A percentage above 100, or a fixed amount
	// above the total, otherwise produces a discount the lines cannot carry — which
	// the allocator refuses outright rather than approximating.
	if discount > orderTotal {
		discount = orderTotal
	}
	if discount < 0 {
		discount = 0
	}

	return &domain.CouponValidationResult{
		Valid:          true,
		CouponID:       coupon.ID,
		Code:           coupon.Code,
		DiscountAmount: discount,
	}, nil
}

// Apply applies a coupon to an order
func (s *CouponService) Apply(ctx context.Context, couponID string, orderID string, customerID string, discount int64) error {
	coupon, err := s.couponRepo.GetByID(ctx, couponID)
	if err != nil {
		return err
	}

	// Record usage
	usage := &domain.CouponUsage{
		ID:         uuid.New().String()[:8],
		CouponID:   couponID,
		CouponCode: coupon.Code,
		OrderID:    orderID,
		CustomerID: customerID,
		Discount:   discount,
		CreatedAt:  time.Now(),
	}

	if err := s.couponRepo.RecordUsage(ctx, usage); err != nil {
		return err
	}

	// Increment usage count
	coupon.UsageCount++
	if err := s.couponRepo.Update(ctx, coupon); err != nil {
		slog.ErrorContext(ctx, "Failed to update coupon usage count", "error", err)
	}

	slog.InfoContext(ctx, "Applied coupon", "coupon_code", coupon.Code, "order_id", orderID, "discount", discount)
	return nil
}

// checkApplicability reports why coupon does not apply to these lines, or "" when it
// does. Empty lists mean "everything", and exclusion beats inclusion: an excluded item
// in the basket refuses the coupon rather than being quietly skipped, because a coupon
// that silently discounts less than it advertised reads as a pricing bug.
//
// Lines carry their own category so this needs no product read — which also keeps the
// coupon Lambda free of a Postgres pool, since products do not live in DynamoDB.
func checkApplicability(coupon *domain.Coupon, lines []domain.CouponLine) string {
	scoped := len(coupon.ApplicableProducts) > 0 || len(coupon.ExcludedProducts) > 0 ||
		len(coupon.ApplicableCategories) > 0 || len(coupon.ExcludedCategories) > 0
	if !scoped {
		return ""
	}
	if len(lines) == 0 {
		return "This coupon applies to specific items"
	}

	const notForThese = "This coupon does not apply to one or more items in your order"
	for _, line := range lines {
		if slices.Contains(coupon.ExcludedProducts, line.ProductID) ||
			slices.Contains(coupon.ExcludedCategories, line.CategoryID) {
			return notForThese
		}
	}

	// An inclusion list needs at least one line inside it, or the coupon is simply
	// not for this basket.
	if len(coupon.ApplicableProducts) > 0 || len(coupon.ApplicableCategories) > 0 {
		for _, line := range lines {
			if slices.Contains(coupon.ApplicableProducts, line.ProductID) ||
				slices.Contains(coupon.ApplicableCategories, line.CategoryID) {
				return ""
			}
		}
		return "This coupon applies to specific items"
	}
	return ""
}

func (s *CouponService) invalidCouponResult(code, message string) *domain.CouponValidationResult {
	return &domain.CouponValidationResult{
		Valid:        false,
		Code:         code,
		ErrorMessage: message,
	}
}

// Ensure interface compliance
var _ domain.CouponService = (*CouponService)(nil)
