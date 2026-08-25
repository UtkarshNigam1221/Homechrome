package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
)

func activeCoupon() *domain.Coupon {
	end := time.Now().Add(24 * time.Hour)
	return &domain.Coupon{
		ID:         "coupon_1",
		Code:       "FESTIVE20",
		Name:       "Festive 20",
		Type:       domain.CouponTypePercentage,
		Value:      2000, // 20.00%
		Audience:   domain.AudienceAll,
		ValidFrom:  time.Now().Add(-time.Hour),
		ValidUntil: &end,
		Status:     domain.CouponStatusActive,
	}
}

func TestCouponService_Validate(t *testing.T) {
	ctx := context.Background()

	setup := func(t *testing.T, c *domain.Coupon) *CouponService {
		t.Helper()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCouponRepository(ctrl)
		repo.EXPECT().GetByCode(gomock.Any(), "FESTIVE20").Return(c, nil).AnyTimes()
		repo.EXPECT().GetCustomerUsage(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(0, nil).AnyTimes()
		return NewCouponService(repo)
	}

	t.Run("a percentage coupon discounts the cart total", func(t *testing.T) {
		s := setup(t, activeCoupon())

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{
			CartTotal: 300000, CustomerID: "cust_1",
		})
		require.NoError(t, err)
		require.True(t, res.Valid)
		require.Equal(t, int64(60000), res.DiscountAmount) // 20% of ₹3000
	})

	t.Run("a fixed coupon discounts its face value", func(t *testing.T) {
		c := activeCoupon()
		c.Type = domain.CouponTypeFixed
		c.Value = 50000 // ₹500
		s := setup(t, c)

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{CartTotal: 300000})
		require.NoError(t, err)
		require.Equal(t, int64(50000), res.DiscountAmount)
	})

	t.Run("MaxDiscount caps a percentage", func(t *testing.T) {
		c := activeCoupon()
		c.MaxDiscount = 20000 // ₹200
		s := setup(t, c)

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{CartTotal: 300000})
		require.NoError(t, err)
		require.Equal(t, int64(20000), res.DiscountAmount)
	})

	// The allocator refuses a discount larger than the order rather than approximating
	// one, so the cap has to happen here, at the source.
	t.Run("a discount never exceeds the cart", func(t *testing.T) {
		c := activeCoupon()
		c.Type = domain.CouponTypeFixed
		c.Value = 500000 // ₹5000 off a ₹1000 cart
		s := setup(t, c)

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{CartTotal: 100000})
		require.NoError(t, err)
		require.Equal(t, int64(100000), res.DiscountAmount)
	})

	t.Run("rejects a cart below the minimum, saying how much more to add", func(t *testing.T) {
		c := activeCoupon()
		c.MinOrderValue = 300000
		s := setup(t, c)

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{CartTotal: 180000})
		require.NoError(t, err)
		require.False(t, res.Valid)
		require.Contains(t, res.ErrorMessage, "1,200",
			"the shortfall is the one rejection that raises order value")
	})

	t.Run("rejects an inactive coupon", func(t *testing.T) {
		c := activeCoupon()
		c.Status = domain.CouponStatusInactive
		s := setup(t, c)

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{CartTotal: 300000})
		require.NoError(t, err)
		require.False(t, res.Valid)
	})

	t.Run("rejects a coupon whose window has not opened", func(t *testing.T) {
		c := activeCoupon()
		c.ValidFrom = time.Now().Add(time.Hour)
		s := setup(t, c)

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{CartTotal: 300000})
		require.NoError(t, err)
		require.False(t, res.Valid)
	})

	t.Run("rejects an expired coupon", func(t *testing.T) {
		c := activeCoupon()
		past := time.Now().Add(-time.Hour)
		c.ValidUntil = &past
		s := setup(t, c)

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{CartTotal: 300000})
		require.NoError(t, err)
		require.False(t, res.Valid)
	})

	// Open-ended is the default for a coupon the operator ends by switching it off.
	// It must never age out on its own.
	t.Run("an open-ended coupon never expires", func(t *testing.T) {
		c := activeCoupon()
		c.ValidUntil = nil
		s := setup(t, c)

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{CartTotal: 300000})
		require.NoError(t, err)
		require.True(t, res.Valid)
	})

	t.Run("rejects an exhausted coupon", func(t *testing.T) {
		c := activeCoupon()
		c.UsageLimit = 5
		c.UsageCount = 5
		s := setup(t, c)

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{CartTotal: 300000})
		require.NoError(t, err)
		require.False(t, res.Valid)
	})

	t.Run("rejects a customer who has hit the per-user limit", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCouponRepository(ctrl)
		c := activeCoupon()
		c.UsagePerUser = 1
		repo.EXPECT().GetByCode(gomock.Any(), "FESTIVE20").Return(c, nil)
		repo.EXPECT().GetCustomerUsage(gomock.Any(), "cust_1", "coupon_1").Return(1, nil)
		s := NewCouponService(repo)

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{
			CartTotal: 300000, CustomerID: "cust_1",
		})
		require.NoError(t, err)
		require.False(t, res.Valid)
	})

	t.Run("rejects an unknown code", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCouponRepository(ctrl)
		repo.EXPECT().GetByCode(gomock.Any(), "NOPE").Return(nil, errors.NotFound("Coupon"))
		s := NewCouponService(repo)

		res, err := s.Validate(ctx, "NOPE", domain.CouponContext{CartTotal: 300000})
		require.NoError(t, err, "an unknown code is a rejection, not a server error")
		require.False(t, res.Valid)
	})
}

// The stacking gate. Buy-2-get-1 is a third off before any code, so a 20% coupon on top
// reaches 46.7% off. Whether that is allowed is the coupon's own decision.
func TestCouponService_Validate_OfferGate(t *testing.T) {
	ctx := context.Background()

	newService := func(t *testing.T, combines bool) *CouponService {
		t.Helper()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCouponRepository(ctrl)
		c := activeCoupon()
		c.CombinesWithOffers = combines
		repo.EXPECT().GetByCode(gomock.Any(), "FESTIVE20").Return(c, nil).AnyTimes()
		repo.EXPECT().GetCustomerUsage(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(0, nil).AnyTimes()
		return NewCouponService(repo)
	}

	t.Run("refuses to stack when the flag is off", func(t *testing.T) {
		s := newService(t, false)

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{
			CartTotal: 300000, HasAutomaticOffer: true,
		})
		require.NoError(t, err)
		require.False(t, res.Valid)
		require.Contains(t, res.ErrorMessage, "offer")
	})

	t.Run("stacks when the flag is on", func(t *testing.T) {
		s := newService(t, true)

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{
			CartTotal: 300000, HasAutomaticOffer: true,
		})
		require.NoError(t, err)
		require.True(t, res.Valid)
	})

	// The flag only bites when an offer is present. Without one it must change nothing.
	t.Run("the flag being off is irrelevant with no offer in the cart", func(t *testing.T) {
		s := newService(t, false)

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{
			CartTotal: 300000, HasAutomaticOffer: false,
		})
		require.NoError(t, err)
		require.True(t, res.Valid)
	})
}

func TestCouponService_Redeem(t *testing.T) {
	ctx := context.Background()

	t.Run("claims, records and counts, in that order", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCouponRepository(ctrl)

		repo.EXPECT().IncrementUsage(gomock.Any(), "coupon_1").Return(true, nil)
		repo.EXPECT().GetByID(gomock.Any(), "coupon_1").Return(activeCoupon(), nil)
		repo.EXPECT().RecordUsage(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, u *domain.CouponUsage) error {
				require.Equal(t, "order_9", u.OrderID)
				require.Equal(t, int64(60000), u.Discount)
				return nil
			})
		repo.EXPECT().IncrementCustomerUsage(gomock.Any(), "cust_1", "coupon_1").Return(nil)

		s := NewCouponService(repo)
		claimed, err := s.Redeem(ctx, "coupon_1", "order_9", "cust_1", 60000)
		require.NoError(t, err)
		require.True(t, claimed)
	})

	// Redemptions are counted at payment success, so a limited code can be exhausted
	// between an order being created and paid. The order keeps its quoted price; this
	// just reports that no slot was left.
	t.Run("reports exhaustion without recording anything", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCouponRepository(ctrl)

		repo.EXPECT().IncrementUsage(gomock.Any(), "coupon_1").Return(false, nil)
		// No RecordUsage, no IncrementCustomerUsage — gomock fails the test if either
		// is called, which is the assertion.

		s := NewCouponService(repo)
		claimed, err := s.Redeem(ctx, "coupon_1", "order_9", "cust_1", 60000)
		require.NoError(t, err, "exhaustion is an outcome, not an error")
		require.False(t, claimed)
	})
}

func TestCouponService_Update_ClearValidUntilPrecedence(t *testing.T) {
	ctx := context.Background()

	// Pins the contract stated in the comment above the ClearValidUntil branch in
	// coupon_service.go: when a PATCH sets both fields, ClearValidUntil wins because it
	// is checked second. Nothing else in this diff documents that ordering except a test.
	t.Run("ClearValidUntil wins when both are set in the same request", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCouponRepository(ctrl)
		existing := activeCoupon()
		repo.EXPECT().GetByID(gomock.Any(), "coupon_1").Return(existing, nil)
		repo.EXPECT().Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, c *domain.Coupon) error {
				require.Nil(t, c.ValidUntil, "ClearValidUntil must win over a same-request ValidUntil")
				return nil
			})

		newUntil := time.Now().Add(48 * time.Hour)
		s := NewCouponService(repo)
		coupon, err := s.Update(ctx, "coupon_1", domain.UpdateCouponRequest{
			ValidUntil:      &newUntil,
			ClearValidUntil: true,
		}, "admin_1")

		require.NoError(t, err)
		require.Nil(t, coupon.ValidUntil)
	})

	t.Run("ClearValidUntil alone makes a dated coupon open-ended", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCouponRepository(ctrl)
		existing := activeCoupon() // has a non-nil ValidUntil
		repo.EXPECT().GetByID(gomock.Any(), "coupon_1").Return(existing, nil)
		repo.EXPECT().Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, c *domain.Coupon) error {
				require.Nil(t, c.ValidUntil)
				return nil
			})

		s := NewCouponService(repo)
		coupon, err := s.Update(ctx, "coupon_1", domain.UpdateCouponRequest{
			ClearValidUntil: true,
		}, "admin_1")

		require.NoError(t, err)
		require.Nil(t, coupon.ValidUntil)
	})
}

func TestCouponService_Create_FieldMapping(t *testing.T) {
	ctx := context.Background()

	t.Run("maps the new fields and allows an open-ended ValidUntil", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCouponRepository(ctrl)
		repo.EXPECT().Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, c *domain.Coupon) error {
				require.Equal(t, "FESTIVE20", c.Code)
				require.Equal(t, domain.AudienceSpecificCustomer, c.Audience)
				require.Equal(t, "cust_1", c.CustomerID)
				require.True(t, c.CombinesWithOffers)
				require.Nil(t, c.ValidUntil, "a nil ValidUntil must survive Create as open-ended")
				return nil
			})

		s := NewCouponService(repo)
		coupon, err := s.Create(ctx, domain.CreateCouponRequest{
			Code:               "festive20",
			Name:               "Festive 20",
			Type:               domain.CouponTypePercentage,
			Value:              2000,
			Audience:           domain.AudienceSpecificCustomer,
			CustomerID:         "cust_1",
			CombinesWithOffers: true,
			ValidFrom:          time.Now(),
			ValidUntil:         nil,
		}, "admin_1")

		require.NoError(t, err)
		require.NotNil(t, coupon)
	})

	t.Run("rejects a ValidUntil before ValidFrom", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCouponRepository(ctrl)
		// No Create call expected: the date guard must reject before the repository is touched.

		until := time.Now()
		from := until.Add(time.Hour)
		s := NewCouponService(repo)
		coupon, err := s.Create(ctx, domain.CreateCouponRequest{
			Code:       "BADDATES",
			Name:       "Bad Dates",
			Type:       domain.CouponTypeFixed,
			Value:      10000,
			ValidFrom:  from,
			ValidUntil: &until,
		}, "admin_1")

		require.Error(t, err)
		require.Nil(t, coupon)
	})
}

func TestCouponService_GetByCode_Uppercases(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockCouponRepository(ctrl)
	repo.EXPECT().GetByCode(gomock.Any(), "FESTIVE20").Return(activeCoupon(), nil)

	s := NewCouponService(repo)
	coupon, err := s.GetByCode(context.Background(), "festive20")

	require.NoError(t, err)
	require.Equal(t, "FESTIVE20", coupon.Code)
}
