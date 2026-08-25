package handler

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
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
