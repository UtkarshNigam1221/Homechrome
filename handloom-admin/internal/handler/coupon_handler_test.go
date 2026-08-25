package handler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/validator"
)

// The validate endpoint takes a cart total, not a product list. Coupons carry no item
// scoping, so a line list would be a field nothing reads — which is exactly how the old
// product_ids parameter came to exist.
func TestValidateCouponRequest_Shape(t *testing.T) {
	body := `{"code":"FESTIVE20","cart_total":300000,"customer_id":"cust_1","has_automatic_offer":true}`

	var req ValidateCouponRequest
	require.NoError(t, json.Unmarshal([]byte(body), &req))

	require.Equal(t, "FESTIVE20", req.Code)
	require.Equal(t, int64(300000), req.CartTotal)
	require.Equal(t, "cust_1", req.CustomerID)
	require.True(t, req.HasAutomaticOffer)
}

// validRedeemCouponRequest satisfies every field's tag on its own, so each subtest below
// only has to break one field.
func validRedeemCouponRequest() RedeemCouponRequest {
	return RedeemCouponRequest{
		CouponID:   "coupon_1",
		OrderID:    "order_1",
		CustomerID: "cust_1",
		Discount:   500,
	}
}

// RedeemCouponRequest replaced ApplyCouponRequest, which required CustomerID and a
// positive Discount. These exercise the actual validate tags through the same Validator
// the middleware runs in production, not by asserting on the tag strings themselves.
//
// A missing CustomerID matters beyond a bare validation gap: CouponService.Redeem only
// bumps the per-customer usage counter when customerID != "", so a redemption that skips
// this check would count against the coupon's global limit while silently never counting
// against usage_per_user — exactly how a customer slips past that limit.
func TestRedeemCouponRequest_Validation(t *testing.T) {
	v := validator.New()
	ctx := context.Background()

	t.Run("a valid request passes", func(t *testing.T) {
		req := validRedeemCouponRequest()
		require.NoError(t, v.Validate(ctx, &req))
	})

	t.Run("an empty customer id is rejected", func(t *testing.T) {
		req := validRedeemCouponRequest()
		req.CustomerID = ""
		require.Error(t, v.Validate(ctx, &req),
			"an empty CustomerID would let a redemption skip the per-customer usage counter")
	})

	t.Run("a zero discount is rejected", func(t *testing.T) {
		req := validRedeemCouponRequest()
		req.Discount = 0
		require.Error(t, v.Validate(ctx, &req))
	})
}
