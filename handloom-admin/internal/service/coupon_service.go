// Package service implements the business logic layer
package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/metrics"
)

// coupon validation outcome label (repeated across validation branches).
const outcomeInvalid = "invalid"

// minPayableAmount is the smallest total the payment gateway will accept, in paise.
// A discount that cleared the cart made InitiatePayment(Amount: 0) fail: stock was
// released, the sale was lost, and the PENDING order row survived at total 0. A coupon
// problem must never cost a sale, so the discount yields instead — ₹1 charged beats
// nothing charged.
const minPayableAmount int64 = 100

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
	// No duplicate-code pre-check here: the repository's transactional condition on the
	// code pointer item settles that. A read-then-decide check here would race it.

	// Validate dates. ValidUntil is nil-able (open-ended), so only compare when set.
	if req.ValidUntil != nil && req.ValidUntil.Before(req.ValidFrom) {
		return nil, errors.Validation("Valid until date must be after valid from date")
	}

	coupon := &domain.Coupon{
		ID:                 "coupon_" + uuid.New().String()[:8],
		Code:               strings.ToUpper(req.Code),
		Name:               req.Name,
		Description:        req.Description,
		Type:               req.Type,
		Value:              req.Value,
		MinOrderValue:      req.MinOrderValue,
		MaxDiscount:        req.MaxDiscount,
		UsageLimit:         req.UsageLimit,
		UsagePerUser:       req.UsagePerUser,
		UsageCount:         0,
		Audience:           req.Audience,
		CustomerID:         req.CustomerID,
		CombinesWithOffers: req.CombinesWithOffers,
		ValidFrom:          req.ValidFrom,
		ValidUntil:         req.ValidUntil,
		Status:             domain.CouponStatusActive,
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
	// Captured before any field is applied, so the guard below catches a code change
	// however it might slip in — see the assertion after the field updates.
	originalCode := coupon.Code

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
	if req.CombinesWithOffers != nil {
		coupon.CombinesWithOffers = *req.CombinesWithOffers
	}
	if req.ValidFrom != nil {
		coupon.ValidFrom = *req.ValidFrom
	}
	if req.ValidUntil != nil {
		coupon.ValidUntil = req.ValidUntil
	}
	// A nil ValidUntil in a PATCH body is indistinguishable from an omitted one, so
	// "make this open-ended" needs its own signal. Checked after ValidUntil, so
	// ClearValidUntil always wins when both are set in the same request — that
	// precedence is the contract; do not reorder these two branches.
	if req.ClearValidUntil {
		coupon.ValidUntil = nil
	}
	if req.Status != nil {
		coupon.Status = *req.Status
	}

	// CouponRepository.Update never repoints the code pointer item, so a code change here
	// would silently orphan it — the coupon would stop resolving by GetByCode while still
	// existing by ID. UpdateCouponRequest has no Code field today, so this can't fire; it
	// exists so that if one is ever added, the oversight fails loudly here instead of
	// shipping a coupon nobody can look up by its own code.
	if coupon.Code != originalCode {
		return nil, errors.New(errors.ErrCodeInternal,
			"coupon code cannot be changed via Update; the code pointer index would be orphaned")
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

// Validate checks a coupon against a cart. Every rejection carries a reason the
// storefront shows verbatim, so the wording lives in one place.
//
// Note what is absent: cart contents. Coupons carry no item scoping, so eligibility is a
// function of the total and the customer. That is what lets the wallet evaluate every
// candidate coupon against one shared context.
func (s *CouponService) Validate(
	ctx context.Context,
	code string,
	cc domain.CouponContext,
) (*domain.CouponValidationResult, error) {
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

	reject := func(o, message string) (*domain.CouponValidationResult, error) {
		outcome = o
		return &domain.CouponValidationResult{
			Valid: false, Code: code, ErrorMessage: message,
		}, nil
	}

	coupon, err := s.couponRepo.GetByCode(ctx, code)
	if err != nil || coupon == nil {
		return reject(outcomeInvalid, "That code isn't valid")
	}

	if coupon.Status != domain.CouponStatusActive {
		return reject(outcomeInvalid, "This coupon is no longer available")
	}

	now := time.Now()
	if now.Before(coupon.ValidFrom) {
		return reject(outcomeInvalid, "This coupon isn't active yet")
	}
	// A nil ValidUntil is open-ended and never expires — the operator's off switch is
	// how such a coupon ends.
	if coupon.ValidUntil != nil && now.After(*coupon.ValidUntil) {
		return reject("expired", "This coupon has expired")
	}

	if cc.CartTotal < coupon.MinOrderValue {
		short := coupon.MinOrderValue - cc.CartTotal
		return reject(outcomeInvalid, fmt.Sprintf("Add %s more to use this coupon", formatPaise(short)))
	}

	if coupon.UsageLimit > 0 && coupon.UsageCount >= coupon.UsageLimit {
		return reject("limit_reached", "This coupon has been fully claimed")
	}

	if coupon.UsagePerUser > 0 && cc.CustomerID != "" {
		used, usageErr := s.couponRepo.GetCustomerUsage(ctx, cc.CustomerID, coupon.ID)
		if usageErr == nil && used >= coupon.UsagePerUser {
			return reject("limit_reached", "You've already used this coupon")
		}
	}

	// The stacking gate. Off unless the operator turned it on, because buy-2-get-1 is a
	// third off before any code and 20% on top of it reaches 46.7%.
	if cc.HasAutomaticOffer && !coupon.CombinesWithOffers {
		return reject(outcomeInvalid, "This code can't be used with the offer already in your cart")
	}

	discount, trimmed := computeCouponDiscount(coupon, cc.CartTotal)

	result := &domain.CouponValidationResult{
		Valid:          true,
		CouponID:       coupon.ID,
		Code:           coupon.Code,
		DiscountAmount: discount,
	}
	// Applied, not refused. The code is worth more than the order, so it pays out what
	// it can and says so, rather than zeroing a total the gateway would then reject.
	if trimmed > 0 {
		result.Notice = fmt.Sprintf(
			"This code is worth more than your order, so we've taken off %s and left %s to pay",
			formatPaise(discount), formatPaise(minPayableAmount))
	}
	return result, nil
}

// computeCouponDiscount returns the discount to apply, plus how much of the coupon's
// face value had to be given up to leave something payable.
//
// The ceiling is the cart less minPayableAmount, not the cart itself: allocateDiscount
// refuses a discount larger than the order, and the gateway refuses a zero total. Both
// are reachable without malice — a FIXED coupon worth more than the cart (₹500 off a
// ₹450 cart), or a percentage coupon stored before the 100% ceiling existed.
func computeCouponDiscount(coupon *domain.Coupon, cartTotal int64) (discount, trimmed int64) {
	var raw int64
	if coupon.Type == domain.CouponTypePercentage {
		raw = cartTotal * coupon.Value / 10000 // Value is percentage × 100
		if coupon.MaxDiscount > 0 && raw > coupon.MaxDiscount {
			raw = coupon.MaxDiscount
		}
	} else {
		raw = coupon.Value
	}
	raw = max(raw, 0)

	ceiling := max(cartTotal-minPayableAmount, 0)
	if raw > ceiling {
		return ceiling, raw - ceiling
	}
	return raw, 0
}

// formatPaise renders paise as rupees with Indian digit grouping, for customer-facing
// messages. 120000 -> "₹1,200".
func formatPaise(paise int64) string {
	rupees := paise / 100
	s := strconv.FormatInt(rupees, 10)
	if len(s) <= 3 {
		return "₹" + s
	}
	// Indian grouping: last three digits, then pairs.
	head, tail := s[:len(s)-3], s[len(s)-3:]
	var parts []string
	for len(head) > 2 {
		parts = append([]string{head[len(head)-2:]}, parts...)
		head = head[:len(head)-2]
	}
	if head != "" {
		parts = append([]string{head}, parts...)
	}
	return "₹" + strings.Join(parts, ",") + "," + tail
}

// Redeem records one redemption of a paid order.
//
// The claim comes first and is conditional: if it fails, the coupon is exhausted and
// nothing else is written. Callers must treat claimed=false as an expected outcome —
// because redemptions are counted at payment success, a limited code can run out between
// an order being created and paid, and the order still keeps the price it was quoted.
func (s *CouponService) Redeem(
	ctx context.Context,
	couponID, orderID, customerID string,
	discount int64,
) (bool, error) {
	claimed, err := s.couponRepo.IncrementUsage(ctx, couponID)
	if err != nil {
		return false, err
	}
	if !claimed {
		slog.WarnContext(ctx, "Coupon exhausted before the order was paid",
			"coupon_id", couponID, keyOrderID, orderID)
		return false, nil
	}

	coupon, err := s.couponRepo.GetByID(ctx, couponID)
	if err != nil {
		return true, err
	}

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
		return true, err
	}

	if customerID != "" {
		perUserClaimed, usageErr := s.couponRepo.IncrementCustomerUsage(
			ctx, customerID, couponID, coupon.UsagePerUser)
		if usageErr != nil {
			return true, usageErr
		}
		if !perUserClaimed {
			// The order keeps the price it was quoted, so this does not undo anything.
			// What it does is stop the counter drifting past the limit unnoticed: the
			// customer got through the initiate-to-payment window twice, and that is
			// worth seeing rather than absorbing.
			slog.WarnContext(ctx, "Per-customer coupon allowance already spent at payment success",
				"coupon_id", couponID, "customer_id", customerID, keyOrderID, orderID,
				"usage_per_user", coupon.UsagePerUser)
		}
	}

	slog.InfoContext(ctx, "Redeemed coupon",
		"coupon_code", coupon.Code, keyOrderID, orderID, "discount", discount)
	return true, nil
}

// Ensure interface compliance
var _ domain.CouponService = (*CouponService)(nil)
