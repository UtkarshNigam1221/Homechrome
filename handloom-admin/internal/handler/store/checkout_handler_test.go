package store

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/mocks"
)

// Reuses the public DTO, so the picker cannot leak what the banner withholds.
func TestCheckoutHandler_ListCoupons(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := mocks.NewMockCheckoutService(ctrl)
	svc.EXPECT().ListCoupons(gomock.Any(), "cust_1").Return([]*domain.CouponOffer{
		{
			Coupon:         advertisableCoupon(),
			Eligible:       false,
			DiscountAmount: 0,
			Reason:         "Add ₹500 more to use this coupon",
		},
	}, nil)

	h := NewCheckoutHandler(svc, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/coupons", nil)
	req = req.WithContext(
		context.WithValue(req.Context(), middleware.CustomerIDKey, "cust_1"))
	h.Routes().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))

	var body struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)

	got := body.Data[0]
	require.Equal(t, false, got["eligible"])
	require.Equal(t, "Add ₹500 more to use this coupon", got["reason"])
	require.NotContains(t, got, "usage_count")
	require.NotContains(t, got, "customer_id")
}
