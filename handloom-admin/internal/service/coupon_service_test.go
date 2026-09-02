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
	// one, and the gateway refuses a zero total, so the cap has to happen here.
	// Was 100000 (the whole cart) before the payable floor existed: a discount equal to
	// the cart zeroed the total and killed the payment after the order row was written.
	t.Run("a discount never exceeds the cart, less the payable floor", func(t *testing.T) {
		c := activeCoupon()
		c.Type = domain.CouponTypeFixed
		c.Value = 500000 // ₹5000 off a ₹1000 cart
		s := setup(t, c)

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{CartTotal: 100000})
		require.NoError(t, err)
		require.Equal(t, int64(99900), res.DiscountAmount)
		require.True(t, res.Valid, "a coupon problem must never refuse the sale")
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

// A discount that clears the cart used to zero TotalAmount, which made
// InitiatePayment(Amount: 0) fail: stock released, sale lost, and a PENDING order left
// behind at total 0. The coupon succeeded and the sale died anyway — the one real breach
// of "a coupon problem never fails a checkout". So the discount yields to the gateway's
// floor and the coupon stays valid.
func TestCouponService_Validate_LeavesSomethingToPay(t *testing.T) {
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

	// ₹500 off a ₹450 cart. No malice needed and no invalid data — just a fixed-amount
	// coupon meeting a small order.
	t.Run("a fixed coupon worth more than the cart leaves the floor payable", func(t *testing.T) {
		c := activeCoupon()
		c.Type = domain.CouponTypeFixed
		c.Value = 50000 // ₹500
		s := setup(t, c)

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{CartTotal: 45000})
		require.NoError(t, err)
		require.True(t, res.Valid)
		require.Equal(t, int64(44900), res.DiscountAmount)
		require.Equal(t, int64(100), 45000-res.DiscountAmount, "₹1 must remain payable")
		require.NotEmpty(t, res.Notice, "the shortfall has to reach the customer")
	})

	// A percentage above 100 is refused at creation now, but coupons created before that
	// ceiling existed are still in the table, so the arithmetic has to hold anyway.
	t.Run("a percentage above 100 cannot clear the cart", func(t *testing.T) {
		c := activeCoupon()
		c.Value = 15000 // 150.00%
		s := setup(t, c)

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{CartTotal: 300000})
		require.NoError(t, err)
		require.True(t, res.Valid)
		require.Equal(t, int64(299900), res.DiscountAmount)
		require.NotEmpty(t, res.Notice)
	})

	t.Run("a coupon worth exactly the cart still leaves the floor", func(t *testing.T) {
		c := activeCoupon()
		c.Type = domain.CouponTypeFixed
		c.Value = 300000
		s := setup(t, c)

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{CartTotal: 300000})
		require.NoError(t, err)
		require.Equal(t, int64(299900), res.DiscountAmount)
	})

	// The floor must not creep into ordinary coupons: a 20% code on a ₹3,000 cart is
	// nowhere near the cart total and must be untouched, notice included.
	t.Run("an ordinary coupon is untouched and carries no notice", func(t *testing.T) {
		s := setup(t, activeCoupon())

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{CartTotal: 300000})
		require.NoError(t, err)
		require.Equal(t, int64(60000), res.DiscountAmount)
		require.Empty(t, res.Notice, "a coupon that fits must say nothing")
	})

	// A cart smaller than the floor can support no discount at all. The coupon still
	// validates — refusing would be the failure mode this whole fix removes.
	t.Run("a cart at or below the floor yields no discount but still validates", func(t *testing.T) {
		c := activeCoupon()
		c.Type = domain.CouponTypeFixed
		c.Value = 50000
		s := setup(t, c)

		res, err := s.Validate(ctx, "FESTIVE20", domain.CouponContext{CartTotal: 100})
		require.NoError(t, err)
		require.True(t, res.Valid)
		require.Zero(t, res.DiscountAmount)
		require.NotEmpty(t, res.Notice)
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
		repo.EXPECT().
			IncrementCustomerUsage(gomock.Any(), "cust_1", "coupon_1", 0).
			Return(true, nil)

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

// usage_per_user is claimed under a condition, so a second redemption by the same
// customer reports not-claimed instead of silently pushing the counter past the limit.
// The order still keeps the price it was quoted — the point is that the counter stays
// truthful and the overshoot is visible.
func TestCouponService_Redeem_PerCustomerLimit(t *testing.T) {
	ctx := context.Background()

	c := activeCoupon()
	c.UsagePerUser = 1

	ctrl := gomock.NewController(t)
	repo := mocks.NewMockCouponRepository(ctrl)
	repo.EXPECT().IncrementUsage(gomock.Any(), "coupon_1").Return(true, nil).Times(2)
	repo.EXPECT().GetByID(gomock.Any(), "coupon_1").Return(c, nil).Times(2)
	repo.EXPECT().RecordUsage(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	// The limit reaches the repository, which is what lets the condition exist at all.
	gomock.InOrder(
		repo.EXPECT().
			IncrementCustomerUsage(gomock.Any(), "cust_1", "coupon_1", 1).
			Return(true, nil),
		repo.EXPECT().
			IncrementCustomerUsage(gomock.Any(), "cust_1", "coupon_1", 1).
			Return(false, nil),
	)

	s := NewCouponService(repo)

	claimed, err := s.Redeem(ctx, "coupon_1", "order_1", "cust_1", 60000)
	require.NoError(t, err)
	require.True(t, claimed)

	// Second order by the same customer, paid inside the same initiate-to-payment
	// window. The redemption is still recorded and the order is untouched.
	claimed, err = s.Redeem(ctx, "coupon_1", "order_2", "cust_1", 60000)
	require.NoError(t, err, "a spent per-customer allowance is an outcome, not an error")
	require.True(t, claimed, "the order keeps the price it was quoted")
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

func TestCouponService_ListPublic(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockCouponRepository(ctrl)

	before := time.Now()
	var gotCutoff time.Time
	repo.EXPECT().ListPublic(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, cutoff time.Time) ([]*domain.Coupon, error) {
			gotCutoff = cutoff
			return []*domain.Coupon{activeCoupon()}, nil
		})

	got, err := NewCouponService(repo).ListPublic(ctx)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "FESTIVE20", got[0].Code)
	// The cached banner's cutoff must clear the whole cache window, unlike the picker's.
	require.WithinDuration(t, before.Add(domain.PublicCouponListTTL), gotCutoff, time.Second)
}

func TestCouponService_ListForCart(t *testing.T) {
	ctx := context.Background()

	// 20% off, no minimum. A second coupon needs ₹2,000 in the cart.
	withMinimum := func() *domain.Coupon {
		c := activeCoupon()
		c.ID = "coupon_2"
		c.Code = "BIG500"
		c.Type = domain.CouponTypeFixed
		c.Value = 50000
		c.MinOrderValue = 200000
		return c
	}

	// Local cap: TestCouponService_Redeem asserts against activeCoupon() unchanged.
	withUsagePerUser := func() *domain.Coupon {
		c := activeCoupon()
		c.UsagePerUser = 1
		return c
	}

	setup := func(t *testing.T, counts map[string]int) *CouponService {
		t.Helper()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCouponRepository(ctrl)
		repo.EXPECT().ListPublic(gomock.Any(), gomock.Any()).
			Return([]*domain.Coupon{withUsagePerUser(), withMinimum()}, nil)
		// Exactly once for the whole list — the N+1 guard.
		repo.EXPECT().GetCustomerUsageAll(gomock.Any(), "cust_1").Return(counts, nil).Times(1)
		return NewCouponService(repo)
	}

	t.Run("annotates each coupon against the cart", func(t *testing.T) {
		s := setup(t, nil)
		cc := domain.CouponContext{CartTotal: 100000, CustomerID: "cust_1"}

		offers, err := s.ListForCart(ctx, cc)
		require.NoError(t, err)
		require.Len(t, offers, 2)

		require.True(t, offers[0].Eligible, "eligible coupons sort first")
		require.Equal(t, "FESTIVE20", offers[0].Coupon.Code)
		require.Equal(t, int64(20000), offers[0].DiscountAmount)
		require.Empty(t, offers[0].Reason)

		require.False(t, offers[1].Eligible)
		require.Equal(t, "BIG500", offers[1].Coupon.Code)
		require.Zero(t, offers[1].DiscountAmount)
		// Cross-checked against evaluate, not a pinned literal, so they cannot drift.
		want := evaluate(offers[1].Coupon, cc, 0, nil)
		require.False(t, want.Valid)
		require.Equal(t, want.ErrorMessage, offers[1].Reason)
	})

	t.Run("a spent per-user allowance is reported, not hidden", func(t *testing.T) {
		s := setup(t, map[string]int{"coupon_1": 1})

		offers, err := s.ListForCart(ctx, domain.CouponContext{
			CartTotal: 100000, CustomerID: "cust_1",
		})
		require.NoError(t, err)

		byCode := map[string]*domain.CouponOffer{}
		for _, o := range offers {
			byCode[o.Coupon.Code] = o
		}
		require.False(t, byCode["FESTIVE20"].Eligible)
		require.Equal(t, "You've already used this coupon", byCode["FESTIVE20"].Reason)
	})

	t.Run("a failed usage-counter read is an error, not a silent zero", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCouponRepository(ctrl)
		repo.EXPECT().ListPublic(gomock.Any(), gomock.Any()).Return([]*domain.Coupon{activeCoupon()}, nil)
		boom := errors.NotFound("usage counters")
		repo.EXPECT().GetCustomerUsageAll(gomock.Any(), "cust_1").Return(nil, boom)

		offers, err := NewCouponService(repo).ListForCart(ctx, domain.CouponContext{
			CartTotal: 100000, CustomerID: "cust_1",
		})
		require.Equal(t, boom, err, "a read failure must surface, not read every cap as unused")
		require.Nil(t, offers)
	})

	t.Run("sorts eligible best-saving-first, then keeps ineligible in list order", func(t *testing.T) {
		eligBig := func() *domain.Coupon {
			c := activeCoupon()
			c.ID, c.Code = "coupon_big", "BIGSAVE"
			c.Type, c.Value = domain.CouponTypeFixed, 80000 // ₹800 off
			return c
		}
		eligSmall := func() *domain.Coupon {
			c := activeCoupon()
			c.ID, c.Code = "coupon_small", "SMALLSAVE"
			c.Type, c.Value = domain.CouponTypeFixed, 20000 // ₹200 off
			return c
		}
		// Both need ₹10,000 in the cart, which a ₹3,000 cart never reaches.
		inelig1 := func() *domain.Coupon {
			c := activeCoupon()
			c.ID, c.Code, c.MinOrderValue = "coupon_in1", "TOOFAR1", 1000000
			return c
		}
		inelig2 := func() *domain.Coupon {
			c := activeCoupon()
			c.ID, c.Code, c.MinOrderValue = "coupon_in2", "TOOFAR2", 1000000
			return c
		}

		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCouponRepository(ctrl)
		// Unsorted, ineligible entries split apart: catches an unstable sort.
		repo.EXPECT().ListPublic(gomock.Any(), gomock.Any()).
			Return([]*domain.Coupon{inelig1(), eligSmall(), inelig2(), eligBig()}, nil)

		offers, err := NewCouponService(repo).ListForCart(ctx, domain.CouponContext{CartTotal: 300000})
		require.NoError(t, err)
		require.Len(t, offers, 4)

		codes := make([]string, len(offers))
		for i, o := range offers {
			codes[i] = o.Coupon.Code
		}
		require.Equal(t, []string{"BIGSAVE", "SMALLSAVE", "TOOFAR1", "TOOFAR2"}, codes,
			"eligible first by largest saving, then ineligible in ListPublic's original order")

		require.True(t, offers[0].Eligible)
		require.True(t, offers[1].Eligible)
		require.False(t, offers[2].Eligible)
		require.False(t, offers[3].Eligible)
	})

	t.Run("no public coupons costs no usage read", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCouponRepository(ctrl)
		repo.EXPECT().ListPublic(gomock.Any(), gomock.Any()).Return(nil, nil)
		// Times(0) is the assertion: gomock fails the test if the read happens.
		repo.EXPECT().GetCustomerUsageAll(gomock.Any(), gomock.Any()).Times(0)

		offers, err := NewCouponService(repo).ListForCart(ctx, domain.CouponContext{
			CartTotal: 100000, CustomerID: "cust_1",
		})
		require.NoError(t, err)
		require.Empty(t, offers)
		require.NotNil(t, offers, "an empty list must encode as [], not null")
	})

	// The live picker must not inherit the banner's horizon and hide a valid coupon.
	t.Run("uses a live cutoff, not the banner's cache-window horizon", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCouponRepository(ctrl)

		before := time.Now()
		var gotCutoff time.Time
		repo.EXPECT().ListPublic(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, cutoff time.Time) ([]*domain.Coupon, error) {
				gotCutoff = cutoff
				return []*domain.Coupon{activeCoupon()}, nil
			})
		repo.EXPECT().GetCustomerUsageAll(gomock.Any(), "cust_1").Return(nil, nil)

		_, err := NewCouponService(repo).ListForCart(ctx, domain.CouponContext{
			CartTotal: 100000, CustomerID: "cust_1",
		})
		require.NoError(t, err)
		require.WithinDuration(t, before, gotCutoff, time.Second)
		require.Less(t, gotCutoff.Sub(before), domain.PublicCouponListTTL,
			"the picker's cutoff must not extend the banner's cache-window margin")
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

// audienceCoupon is an otherwise-valid coupon targeted at one audience.
func audienceCoupon(a domain.CouponAudience) *domain.Coupon {
	c := activeCoupon()
	c.Audience = a
	return c
}

func intPtr(n int) *int { return &n }

// The rule, exercised directly rather than through Validate, so no repository
// behavior can mask a branch.
func TestEvaluate_Audience(t *testing.T) {
	cc := domain.CouponContext{CartTotal: 100000, CustomerID: "cust_1"}

	t.Run("ALL ignores the order count entirely", func(t *testing.T) {
		for _, oc := range []*int{nil, intPtr(0), intPtr(7)} {
			res := evaluate(audienceCoupon(domain.AudienceAll), cc, 0, oc)
			require.True(t, res.Valid, "an ALL coupon must not depend on order history")
		}
	})

	t.Run("FIRST_ORDER passes on a first order", func(t *testing.T) {
		res := evaluate(audienceCoupon(domain.AudienceFirstOrder), cc, 0, intPtr(0))
		require.True(t, res.Valid)
	})

	t.Run("FIRST_ORDER names its reason to a returning customer", func(t *testing.T) {
		res := evaluate(audienceCoupon(domain.AudienceFirstOrder), cc, 0, intPtr(1))
		require.False(t, res.Valid)
		require.Equal(t, "This code is for first orders only", res.ErrorMessage)
		require.Equal(t, outcomeAudience, res.Outcome)
	})

	t.Run("RETURNING passes once there is an order", func(t *testing.T) {
		res := evaluate(audienceCoupon(domain.AudienceReturning), cc, 0, intPtr(1))
		require.True(t, res.Valid)
	})

	t.Run("RETURNING names its reason to a first-time buyer", func(t *testing.T) {
		res := evaluate(audienceCoupon(domain.AudienceReturning), cc, 0, intPtr(0))
		require.False(t, res.Valid)
		require.Equal(t, "This code is for returning customers", res.ErrorMessage)
		require.Equal(t, outcomeAudience, res.Outcome)
	})

	t.Run("SPECIFIC_CUSTOMER passes for its own customer", func(t *testing.T) {
		c := audienceCoupon(domain.AudienceSpecificCustomer)
		c.CustomerID = "cust_1"
		res := evaluate(c, cc, 0, nil)
		require.True(t, res.Valid, "no order count is needed to match an id")
	})

	// The refusal must be indistinguishable from a typo, or it confirms the code exists.
	t.Run("SPECIFIC_CUSTOMER is silent to anyone else", func(t *testing.T) {
		c := audienceCoupon(domain.AudienceSpecificCustomer)
		c.CustomerID = "cust_someone_else"
		res := evaluate(c, cc, 0, nil)
		require.False(t, res.Valid)
		require.Equal(t, msgCodeInvalid, res.ErrorMessage)
		require.Equal(t, outcomeAudience, res.Outcome,
			"the customer cannot tell, but the funnel must")
	})

	// The shared constant makes drift impossible, but only while both paths use it.
	// This compares the two outputs, so reinstating an inline literal anywhere fails.
	t.Run("its refusal is identical to an unknown code's", func(t *testing.T) {
		mine := audienceCoupon(domain.AudienceSpecificCustomer)
		mine.CustomerID = "cust_someone_else"
		targeted := evaluate(mine, cc, 0, nil)

		unknown := &domain.CouponValidationResult{ErrorMessage: msgCodeInvalid}
		require.Equal(t, unknown.ErrorMessage, targeted.ErrorMessage,
			"a distinct message would confirm the code is real")
	})

	t.Run("an unresolved order count rejects rather than assuming", func(t *testing.T) {
		for _, a := range []domain.CouponAudience{
			domain.AudienceFirstOrder, domain.AudienceReturning,
		} {
			res := evaluate(audienceCoupon(a), cc, 0, nil)
			require.False(t, res.Valid, "%s must not pass on an unresolved count", a)
			require.Equal(t, msgCodeInvalid, res.ErrorMessage)
		}
	})

	t.Run("a targeted coupon rejects when there is no customer at all", func(t *testing.T) {
		anon := domain.CouponContext{CartTotal: 100000}
		for _, a := range []domain.CouponAudience{
			domain.AudienceFirstOrder, domain.AudienceReturning,
			domain.AudienceSpecificCustomer,
		} {
			res := evaluate(audienceCoupon(a), anon, 0, nil)
			require.False(t, res.Valid, "%s must not pass without an identity", a)
			require.Equal(t, msgCodeInvalid, res.ErrorMessage)
		}
	})

	// Order history cannot change; a cart can. So the audience refusal is the more
	// useful one when both apply.
	t.Run("audience is reported before the cart minimum", func(t *testing.T) {
		c := audienceCoupon(domain.AudienceFirstOrder)
		c.MinOrderValue = 500000
		small := domain.CouponContext{CartTotal: 1000, CustomerID: "cust_1"}
		res := evaluate(c, small, 0, intPtr(3))
		require.Equal(t, "This code is for first orders only", res.ErrorMessage)
	})

	// A dead coupon is dead for everyone, so status still wins over audience.
	t.Run("status is reported before audience", func(t *testing.T) {
		c := audienceCoupon(domain.AudienceFirstOrder)
		c.Status = domain.CouponStatusInactive
		res := evaluate(c, cc, 0, intPtr(3))
		require.Equal(t, "This coupon is no longer available", res.ErrorMessage)
	})
}
