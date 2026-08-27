package store

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
)

func advertisableCoupon() *domain.Coupon {
	end := time.Now().Add(30 * 24 * time.Hour)
	return &domain.Coupon{
		ID:            "coupon_1",
		Code:          "FESTIVE20",
		Name:          "Festive 20",
		Description:   "20% off everything",
		Type:          domain.CouponTypePercentage,
		Value:         2000,
		MinOrderValue: 100000,
		MaxDiscount:   50000,
		UsageLimit:    5,
		UsageCount:    4,
		UsagePerUser:  1,
		Audience:      domain.AudienceAll,
		CustomerID:    "cust_secret",
		ValidUntil:    &end,
		Status:        domain.CouponStatusActive,
	}
}

func serveCoupons(t *testing.T, svc domain.CouponService) *httptest.ResponseRecorder {
	t.Helper()
	h := NewCatalogHandler(nil, nil, nil, svc)
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/coupons", nil))
	return rec
}

// The payload reaches every visitor, so what it omits is a guarantee, not a saving.
func TestCatalogHandler_ListCoupons_WithholdsInternalFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := mocks.NewMockCouponService(ctrl)
	svc.EXPECT().ListPublic(gomock.Any()).
		Return([]*domain.Coupon{advertisableCoupon()}, nil)

	rec := serveCoupons(t, svc)
	require.Equal(t, http.StatusOK, rec.Code)
	// Derived, not literal: a max-age longer than ListPublic's filter window would put
	// expired coupons back on the banner, and this is what catches that drift.
	require.Equal(t,
		fmt.Sprintf("public, max-age=%d", int(domain.PublicCouponListTTL.Seconds())),
		rec.Header().Get("Cache-Control"))

	var body struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)

	got := body.Data[0]
	require.Equal(t, "FESTIVE20", got["code"])
	require.Equal(t, float64(2000), got["value"])

	for _, withheld := range []string{
		"id", "usage_count", "usage_limit", "usage_per_user", "customer_id",
		"batch_id", "search_key", "status", "audience", "combines_with_offers",
		"created_at", "updated_at",
	} {
		require.NotContains(t, got, withheld, "internal field must never reach a customer")
	}
}

// A dead coupon path must not blank the homepage.
func TestCatalogHandler_ListCoupons_ReadFailureIsAnEmptyList(t *testing.T) {
	ctrl := gomock.NewController(t)
	svc := mocks.NewMockCouponService(ctrl)
	svc.EXPECT().ListPublic(gomock.Any()).
		Return(nil, errors.Internal("dynamodb is down"))

	rec := serveCoupons(t, svc)
	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Success bool             `json:"success"`
		Data    []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.True(t, body.Success)
	require.Empty(t, body.Data)
}
