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
	t.Run("a public coupon lands in the listing partition", func(t *testing.T) {
		end := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
		c := &Coupon{ID: "coupon_abc", Code: "FESTIVE20", Audience: AudienceAll, ValidUntil: &end}
		c.SetKeys()

		require.Equal(t, "COUPON#coupon_abc", c.PK)
		require.Equal(t, SKMetadata, c.SK)
		require.Equal(t, "COUPON#ALL", c.GSI1PK)
		require.Equal(t, "2026-12-31T23:59:59Z#coupon_abc", c.GSI1SK)
	})

	t.Run("a named-customer coupon lands in that customer's partition", func(t *testing.T) {
		end := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
		c := &Coupon{
			ID: "coupon_xyz", Code: "WINBACK5",
			Audience: AudienceSpecificCustomer, CustomerID: "cust_9", ValidUntil: &end,
		}
		c.SetKeys()

		require.Equal(t, "CUSTOMER_COUPON#cust_9", c.GSI1PK,
			"a personal code must not sit in the listing partition")
	})

	// An open-ended coupon must still fall inside the wallet's `GSI1SK >= now` range
	// query, forever. Sorting it under a far-future sentinel is what achieves that.
	t.Run("an open-ended coupon sorts under a far-future sentinel", func(t *testing.T) {
		c := &Coupon{ID: "coupon_open", Code: "ALWAYS", Audience: AudienceAll, ValidUntil: nil}
		c.SetKeys()

		require.Equal(t, "9999-12-31T23:59:59Z#coupon_open", c.GSI1SK)
	})

	// The sort key is a range boundary, so it must compare lexicographically in the
	// same order as the timestamps it encodes.
	t.Run("sort keys order the same way the dates do", func(t *testing.T) {
		early := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
		late := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)

		a := &Coupon{ID: "a", Audience: AudienceAll, ValidUntil: &early}
		b := &Coupon{ID: "b", Audience: AudienceAll, ValidUntil: &late}
		open := &Coupon{ID: "c", Audience: AudienceAll}
		a.SetKeys()
		b.SetKeys()
		open.SetKeys()

		require.Less(t, a.GSI1SK, b.GSI1SK)
		require.Less(t, b.GSI1SK, open.GSI1SK, "open-ended must sort last")
	})

	// UTC, not local: the same instant must produce one key regardless of where the
	// process runs, or a coupon changes partitions when the server's zone changes.
	t.Run("the sort key is UTC regardless of the input zone", func(t *testing.T) {
		ist := time.FixedZone("IST", 5*60*60+30*60)
		end := time.Date(2026, 6, 1, 5, 30, 0, 0, ist) // 2026-06-01T00:00:00Z

		c := &Coupon{ID: "coupon_tz", Audience: AudienceAll, ValidUntil: &end}
		c.SetKeys()

		require.Equal(t, "2026-06-01T00:00:00Z#coupon_tz", c.GSI1SK)
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
