package domain

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/validator"
)

// GSI1PK decides which index partition a coupon lands in, and that decision is what
// keeps thousands of personal codes out of the admin listing partition.
func TestCoupon_SetKeys(t *testing.T) {
	made := func(y int, mo time.Month, d, h, mi, sec int) time.Time {
		return time.Date(y, mo, d, h, mi, sec, 0, time.UTC)
	}

	t.Run("a public coupon lands in the listing partition", func(t *testing.T) {
		c := &Coupon{ID: "coupon_abc", Code: "FESTIVE20", Audience: AudienceAll}
		c.CreatedAt = made(2026, 12, 31, 23, 59, 59)
		c.SetKeys()

		require.Equal(t, "COUPON#coupon_abc", c.PK)
		require.Equal(t, SKMetadata, c.SK)
		require.Equal(t, "COUPON#ALL", c.GSI1PK)
		require.Equal(t, "2026-12-31T23:59:59.000000000Z#coupon_abc", c.GSI1SK)
	})

	t.Run("a named-customer coupon lands in that customer's partition", func(t *testing.T) {
		c := &Coupon{
			ID: "coupon_xyz", Code: "WINBACK5",
			Audience: AudienceSpecificCustomer, CustomerID: "cust_9",
		}
		c.SetKeys()

		require.Equal(t, "CUSTOMER_COUPON#cust_9", c.GSI1PK,
			"a personal code must not sit in the listing partition")
	})

	// The list queries this index descending, so the key has to compare
	// lexicographically in the same order as the timestamps it encodes — otherwise
	// "newest first" is whatever order the strings happen to fall in.
	t.Run("sort keys order the same way the created dates do", func(t *testing.T) {
		early := &Coupon{ID: "a", Audience: AudienceAll}
		early.CreatedAt = made(2026, 1, 2, 0, 0, 0)
		late := &Coupon{ID: "b", Audience: AudienceAll}
		late.CreatedAt = made(2026, 11, 30, 0, 0, 0)
		early.SetKeys()
		late.SetKeys()

		require.Less(t, early.GSI1SK, late.GSI1SK)
	})

	// Two coupons created in the same second must still order, or a page boundary can
	// drop or repeat one. Nano precision is what separates them.
	t.Run("coupons created in the same second still order", func(t *testing.T) {
		base := made(2026, 5, 1, 12, 0, 0)
		first := &Coupon{ID: "first", Audience: AudienceAll}
		first.CreatedAt = base
		second := &Coupon{ID: "second", Audience: AudienceAll}
		second.CreatedAt = base.Add(time.Microsecond)
		first.SetKeys()
		second.SetKeys()

		require.NotEqual(t, first.GSI1SK, second.GSI1SK)
		require.Less(t, first.GSI1SK, second.GSI1SK)
	})

	// UTC, not local: the same instant must produce one key regardless of where the
	// process runs, or the list reorders when the server's zone changes.
	t.Run("the sort key is UTC regardless of the input zone", func(t *testing.T) {
		ist := time.FixedZone("IST", 5*60*60+30*60)
		c := &Coupon{ID: "coupon_tz", Audience: AudienceAll}
		c.CreatedAt = time.Date(2026, 6, 1, 5, 30, 0, 0, ist) // 2026-06-01T00:00:00Z
		c.SetKeys()

		require.Equal(t, "2026-06-01T00:00:00.000000000Z#coupon_tz", c.GSI1SK)
	})

	// search_key is what the list's DynamoDB filter matches on, and contains() is
	// case-sensitive — so the stored copy has to be lowered, code and name both.
	t.Run("search_key lowercases the code and the name", func(t *testing.T) {
		c := &Coupon{ID: "coupon_s", Code: "WELCOME10", Name: "Welcome Offer", Audience: AudienceAll}
		c.SetKeys()

		require.Equal(t, "welcome10 welcome offer", c.SearchKey)
	})
}

func TestCouponUseCounter_SetKeys(t *testing.T) {
	u := &CouponUseCounter{}
	u.SetKeys("cust_1", "coupon_abc")

	require.Equal(t, "CUSTOMER#cust_1", u.PK)
	require.Equal(t, "USE#coupon_abc", u.SK)
}

// validCreateCouponRequest returns a request that satisfies every field's tag on its
// own, so each subtest below only has to vary Audience/CustomerID.
func validCreateCouponRequest() CreateCouponRequest {
	return CreateCouponRequest{
		Code:      "FESTIVE20",
		Name:      "Festive 20",
		Type:      CouponTypePercentage,
		Value:     2000,
		Audience:  AudienceAll,
		ValidFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

// Value carried no upper bound, so 150% stored happily and then produced a discount
// larger than any cart. The ceiling only applies to PERCENTAGE — a FIXED coupon's value
// is paise, where 10000 is a routine ₹100.
func TestCreateCouponRequest_ValueCeiling(t *testing.T) {
	v := validator.New()
	ctx := context.Background()

	t.Run("a percentage above 100% is rejected", func(t *testing.T) {
		req := validCreateCouponRequest()
		req.Type = CouponTypePercentage
		req.Value = 10001 // 100.01%
		require.Error(t, v.Validate(ctx, &req),
			"a discount over 100% cannot be honored by any cart")
	})

	t.Run("150% is rejected", func(t *testing.T) {
		req := validCreateCouponRequest()
		req.Type = CouponTypePercentage
		req.Value = 15000
		require.Error(t, v.Validate(ctx, &req))
	})

	t.Run("exactly 100% is allowed", func(t *testing.T) {
		req := validCreateCouponRequest()
		req.Type = CouponTypePercentage
		req.Value = 10000
		require.NoError(t, v.Validate(ctx, &req),
			"a full-value coupon is a legitimate giveaway; the payable floor handles it")
	})

	t.Run("a fixed amount far above the percentage ceiling is allowed", func(t *testing.T) {
		req := validCreateCouponRequest()
		req.Type = CouponTypeFixed
		req.Value = 5000000 // ₹50,000 off
		require.NoError(t, v.Validate(ctx, &req),
			"the ceiling is a percentage rule; paise have no such bound")
	})

	t.Run("a zero value is still rejected", func(t *testing.T) {
		req := validCreateCouponRequest()
		req.Value = 0
		require.Error(t, v.Validate(ctx, &req))
	})
}

// These exercise the actual validate tags through the same Validator the middleware
// runs in production (internal/validator.Service wrapping go-playground/validator),
// rather than asserting on the tag strings themselves.
func TestCreateCouponRequest_AudienceValidation(t *testing.T) {
	v := validator.New()
	ctx := context.Background()

	t.Run("a valid audience passes", func(t *testing.T) {
		req := validCreateCouponRequest()
		require.NoError(t, v.Validate(ctx, &req))
	})

	t.Run("an unknown audience string is rejected", func(t *testing.T) {
		req := validCreateCouponRequest()
		req.Audience = CouponAudience("BANANA")
		require.Error(t, v.Validate(ctx, &req),
			"a typo'd audience must not fail open to ALL")
	})

	t.Run("SPECIFIC_CUSTOMER with an empty CustomerID is rejected", func(t *testing.T) {
		req := validCreateCouponRequest()
		req.Audience = AudienceSpecificCustomer
		req.CustomerID = ""
		require.Error(t, v.Validate(ctx, &req),
			"otherwise SetKeys would produce an unreachable GSI1PK of just \"CUSTOMER_COUPON#\"")
	})

	t.Run("SPECIFIC_CUSTOMER with a customer id passes", func(t *testing.T) {
		req := validCreateCouponRequest()
		req.Audience = AudienceSpecificCustomer
		req.CustomerID = "cust_9"
		require.NoError(t, v.Validate(ctx, &req))
	})

	t.Run("a non-specific audience with an empty CustomerID passes", func(t *testing.T) {
		req := validCreateCouponRequest()
		req.Audience = AudienceReturning
		req.CustomerID = ""
		require.NoError(t, v.Validate(ctx, &req),
			"CustomerID must stay optional for every audience except SPECIFIC_CUSTOMER")
	})
}
