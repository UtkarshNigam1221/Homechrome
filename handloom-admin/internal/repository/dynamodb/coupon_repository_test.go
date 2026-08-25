package dynamodb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

func newTestCoupon(id, code string) *domain.Coupon {
	end := time.Now().Add(30 * 24 * time.Hour)
	return &domain.Coupon{
		ID:         id,
		Code:       code,
		Name:       "Test coupon",
		Type:       domain.CouponTypePercentage,
		Value:      2000, // 20.00%
		Audience:   domain.AudienceAll,
		ValidFrom:  time.Now().Add(-time.Hour),
		ValidUntil: &end,
		Status:     domain.CouponStatusActive,
	}
}

// Code lookup is a pointer item rather than a GSI, which is what lets it also be the
// uniqueness guard — a GSI can refuse nothing, and is not strongly consistent.
func TestCouponRepository_CodeIndex(t *testing.T) {
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testCouponsTable)

	repo := NewCouponRepository(wrapped)
	ctx := context.Background()

	t.Run("finds a coupon through the pointer", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newTestCoupon("coupon_a", "FESTIVE20")))

		found, err := repo.GetByCode(ctx, "FESTIVE20")
		require.NoError(t, err)
		require.Equal(t, "coupon_a", found.ID)
		require.Equal(t, int64(2000), found.Value)
	})

	t.Run("normalises the code to upper case", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newTestCoupon("coupon_case", "MiXeD10")))

		found, err := repo.GetByCode(ctx, "mixed10")
		require.NoError(t, err)
		require.Equal(t, "coupon_case", found.ID)
	})

	t.Run("refuses a code another coupon holds", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newTestCoupon("coupon_b", "SUMMER15")))

		err := repo.Create(ctx, newTestCoupon("coupon_c", "SUMMER15"))
		require.Error(t, err, "two coupons must not share a code")

		appErr, ok := errors.AsAppError(err)
		require.True(t, ok)
		require.Equal(t, errors.ErrCodeAlreadyExists, appErr.Code)
	})

	// Coupon and pointer are one transaction. A refused code must leave no coupon
	// behind, or the guard has a hole exactly where it exists to have none.
	t.Run("creates neither the coupon nor the pointer when the code is taken", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newTestCoupon("coupon_x", "SHARED")))

		err := repo.Create(ctx, newTestCoupon("coupon_y", "SHARED"))
		require.Error(t, err)

		_, getErr := repo.GetByID(ctx, "coupon_y")
		require.Error(t, getErr, "the rejected coupon must not exist")
	})

	t.Run("reports an unknown code as not found", func(t *testing.T) {
		_, err := repo.GetByCode(ctx, "NOSUCHCODE")
		require.Error(t, err)

		appErr, ok := errors.AsAppError(err)
		require.True(t, ok)
		require.Equal(t, errors.ErrCodeNotFound, appErr.Code)
	})

	// An open-ended coupon has no ValidUntil. It must round-trip as nil rather than as
	// a zero time, which would read as "expired in year 1".
	t.Run("round-trips an open-ended coupon", func(t *testing.T) {
		c := newTestCoupon("coupon_open", "ALWAYS")
		c.ValidUntil = nil
		require.NoError(t, repo.Create(ctx, c))

		found, err := repo.GetByCode(ctx, "ALWAYS")
		require.NoError(t, err)
		require.Nil(t, found.ValidUntil, "open-ended must not become a zero time")
	})
}

func TestCouponRepository_Update(t *testing.T) {
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testCouponsTable)

	repo := NewCouponRepository(wrapped)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newTestCoupon("coupon_u", "EDITME")))

	t.Run("keeps the code pointer resolvable after an edit", func(t *testing.T) {
		c, err := repo.GetByID(ctx, "coupon_u")
		require.NoError(t, err)

		c.Name = "Renamed"
		require.NoError(t, repo.Update(ctx, c))

		found, err := repo.GetByCode(ctx, "EDITME")
		require.NoError(t, err)
		require.Equal(t, "Renamed", found.Name)
	})
}

func TestCouponRepository_List(t *testing.T) {
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testCouponsTable)

	repo := NewCouponRepository(wrapped)
	ctx := context.Background()

	active := newTestCoupon("coupon_l1", "LISTONE")
	inactive := newTestCoupon("coupon_l2", "LISTTWO")
	inactive.Status = domain.CouponStatusInactive
	fixed := newTestCoupon("coupon_l3", "LISTTHREE")
	fixed.Type = domain.CouponTypeFixed

	// A personal code. It must never appear in the admin listing partition, or a batch
	// of a thousand would bury the handful of public promos.
	personal := newTestCoupon("coupon_l4", "PERSONAL1")
	personal.Audience = domain.AudienceSpecificCustomer
	personal.CustomerID = "cust_1"

	for _, c := range []*domain.Coupon{active, inactive, fixed, personal} {
		require.NoError(t, repo.Create(ctx, c))
	}

	t.Run("returns the public coupons", func(t *testing.T) {
		res, err := repo.List(ctx, domain.ListCouponsRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 50},
		})
		require.NoError(t, err)
		require.Len(t, res.Coupons, 3, "the personal code must not be listed")
	})

	t.Run("filters by status", func(t *testing.T) {
		status := domain.CouponStatusInactive
		res, err := repo.List(ctx, domain.ListCouponsRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 50},
			Status:            &status,
		})
		require.NoError(t, err)
		require.Len(t, res.Coupons, 1)
		require.Equal(t, "coupon_l2", res.Coupons[0].ID)
	})

	t.Run("filters by type", func(t *testing.T) {
		ct := domain.CouponTypeFixed
		res, err := repo.List(ctx, domain.ListCouponsRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 50},
			Type:              &ct,
		})
		require.NoError(t, err)
		require.Len(t, res.Coupons, 1)
		require.Equal(t, "coupon_l3", res.Coupons[0].ID)
	})

	t.Run("searches by code and name", func(t *testing.T) {
		res, err := repo.List(ctx, domain.ListCouponsRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 50},
			Search:            "listthree",
		})
		require.NoError(t, err)
		require.Len(t, res.Coupons, 1)
		require.Equal(t, "coupon_l3", res.Coupons[0].ID)
	})

	t.Run("paginates", func(t *testing.T) {
		res, err := repo.List(ctx, domain.ListCouponsRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 2},
		})
		require.NoError(t, err)
		require.Len(t, res.Coupons, 2)
		require.True(t, res.Pagination.HasMore)
	})
}
