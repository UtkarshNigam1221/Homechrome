package dynamodb

import (
	"context"
	"sync"
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

	// The reason Update is an UpdateItem rather than a PutItem. CouponService reads the
	// coupon, applies the edit and writes it back; the PhonePe webhook increments
	// usage_count atomically in the same window. A whole-item put carries the count the
	// service read and erases whatever landed in between.
	t.Run("an increment between the read and the write survives", func(t *testing.T) {
		c := newTestCoupon("coupon_race_edit", "RACEEDIT")
		require.NoError(t, repo.Create(ctx, c))

		// The service's read.
		read, err := repo.GetByID(ctx, "coupon_race_edit")
		require.NoError(t, err)
		require.Equal(t, 0, read.UsageCount)

		// A payment succeeds while the operator has the form open.
		claimed, err := repo.IncrementUsage(ctx, "coupon_race_edit")
		require.NoError(t, err)
		require.True(t, claimed)

		// The service's write, carrying the stale count it read above.
		read.Name = "Renamed mid-flight"
		require.NoError(t, repo.Update(ctx, read))

		after, err := repo.GetByID(ctx, "coupon_race_edit")
		require.NoError(t, err)
		require.Equal(t, 1, after.UsageCount,
			"an admin edit must not roll back a redemption the webhook already counted")
		require.Equal(t, "Renamed mid-flight", after.Name, "the edit must still land")
	})

	// Everything the request cannot change must come back untouched, and everything it
	// can must actually move — an UpdateItem that names too few fields fails silently.
	t.Run("writes every mutable field and leaves the immutable ones alone", func(t *testing.T) {
		c := newTestCoupon("coupon_fields", "FIELDS")
		c.Description = "Original"
		c.MaxDiscount = 50000
		c.UsageLimit = 10
		require.NoError(t, repo.Create(ctx, c))

		read, err := repo.GetByID(ctx, "coupon_fields")
		require.NoError(t, err)

		newEnd := time.Date(2027, 3, 1, 18, 29, 59, 0, time.UTC)
		read.Name = "Edited name"
		read.Description = "Edited description"
		read.MinOrderValue = 250000
		read.MaxDiscount = 75000
		read.UsageLimit = 20
		read.UsagePerUser = 3
		read.CombinesWithOffers = true
		read.ValidUntil = &newEnd
		read.Status = domain.CouponStatusInactive
		read.UpdatedBy = "user_admin"
		require.NoError(t, repo.Update(ctx, read))

		after, err := repo.GetByID(ctx, "coupon_fields")
		require.NoError(t, err)
		require.Equal(t, "Edited name", after.Name)
		require.Equal(t, "Edited description", after.Description)
		require.Equal(t, int64(250000), after.MinOrderValue)
		require.Equal(t, int64(75000), after.MaxDiscount)
		require.Equal(t, 20, after.UsageLimit)
		require.Equal(t, 3, after.UsagePerUser)
		require.True(t, after.CombinesWithOffers)
		require.NotNil(t, after.ValidUntil)
		require.True(t, newEnd.Equal(*after.ValidUntil))
		require.Equal(t, domain.CouponStatusInactive, after.Status)
		require.Equal(t, "user_admin", after.UpdatedBy)

		require.Equal(t, "FIELDS", after.Code, "the code is immutable — its pointer is not repointed")
		require.Equal(t, domain.CouponTypePercentage, after.Type)
		require.Equal(t, int64(2000), after.Value)
		require.Equal(t, domain.AudienceAll, after.Audience)

		// GSI1SK follows created_at, so an edit must leave it exactly where it was.
		// Moving it would reorder the admin list on an unrelated change, and a cursor
		// issued mid-page would then skip or repeat rows.
		require.Equal(t, read.GSI1SK, after.GSI1SK, "an edit must not move the sort key")
		require.NotContains(t, after.GSI1SK, "2027-03-01",
			"the sort key is created_at, not the expiry")
		require.Equal(t, "COUPON#ALL", after.GSI1PK)
		require.Equal(t, "COUPON", after.EntityType)
	})

	// Clearing a field has to remove the attribute, matching the omitempty shape Create
	// writes, rather than parking a zero in it.
	t.Run("clearing the expiry makes the coupon open-ended", func(t *testing.T) {
		c := newTestCoupon("coupon_clear", "CLEARME")
		require.NoError(t, repo.Create(ctx, c))

		read, err := repo.GetByID(ctx, "coupon_clear")
		require.NoError(t, err)
		require.NotNil(t, read.ValidUntil)

		read.ValidUntil = nil
		read.Description = ""
		read.MaxDiscount = 0
		require.NoError(t, repo.Update(ctx, read))

		after, err := repo.GetByID(ctx, "coupon_clear")
		require.NoError(t, err)
		require.Nil(t, after.ValidUntil, "open-ended must not become a zero time")
		require.Empty(t, after.Description)
		require.Zero(t, after.MaxDiscount)
		require.Equal(t, read.GSI1SK, after.GSI1SK,
			"clearing the expiry changes no key — the sort key is created_at")
	})

	t.Run("reports an unknown id as not found", func(t *testing.T) {
		ghost := newTestCoupon("coupon_ghost", "GHOST")
		err := repo.Update(ctx, ghost)
		require.Error(t, err)

		appErr, ok := errors.AsAppError(err)
		require.True(t, ok)
		require.Equal(t, errors.ErrCodeNotFound, appErr.Code)
	})
}

// Deleting a coupon must free its code. The pointer item is what Create's
// attribute_not_exists(PK) tests, so a Delete that leaves it behind burns the code
// permanently — "Coupon code already exists" for a coupon that does not.
func TestCouponRepository_Delete(t *testing.T) {
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testCouponsTable)

	repo := NewCouponRepository(wrapped)
	ctx := context.Background()

	t.Run("frees the code for a new coupon", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newTestCoupon("coupon_d1", "REUSEME")))
		require.NoError(t, repo.Delete(ctx, "coupon_d1"))

		_, err := repo.GetByCode(ctx, "REUSEME")
		require.Error(t, err, "the pointer must go with the coupon")

		require.NoError(t, repo.Create(ctx, newTestCoupon("coupon_d2", "REUSEME")),
			"the same code must be usable again by a different coupon")

		found, err := repo.GetByCode(ctx, "REUSEME")
		require.NoError(t, err)
		require.Equal(t, "coupon_d2", found.ID, "the code must resolve to the new coupon")
	})

	t.Run("reports an unknown id as not found", func(t *testing.T) {
		err := repo.Delete(ctx, "coupon_nope")
		require.Error(t, err)

		appErr, ok := errors.AsAppError(err)
		require.True(t, ok)
		require.Equal(t, errors.ErrCodeNotFound, appErr.Code)
	})

	// Deleting one coupon must not disturb another's pointer, which the transaction's
	// key set decides — a wrong PK here would silently unhook a live code.
	t.Run("leaves another coupon's code alone", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newTestCoupon("coupon_keep", "KEEPME")))
		require.NoError(t, repo.Create(ctx, newTestCoupon("coupon_drop", "DROPME")))
		require.NoError(t, repo.Delete(ctx, "coupon_drop"))

		found, err := repo.GetByCode(ctx, "KEEPME")
		require.NoError(t, err)
		require.Equal(t, "coupon_keep", found.ID)
	})
}

func TestCouponRepository_List(t *testing.T) {
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testCouponsTable)

	repo := NewCouponRepository(wrapped)
	ctx := context.Background()

	active := newTestCoupon("coupon_l1", "LISTONE")
	// Distinctive so a name search can be told apart from a code search — every other
	// coupon here keeps newTestCoupon's identical default name.
	active.Name = "Diwali Handloom Sale"
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

	t.Run("searches by name", func(t *testing.T) {
		res, err := repo.List(ctx, domain.ListCouponsRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 50},
			Search:            "diwali",
		})
		require.NoError(t, err)
		require.Len(t, res.Coupons, 1)
		require.Equal(t, "coupon_l1", res.Coupons[0].ID)
	})

	// The needle is lowered before it reaches DynamoDB, because contains() is
	// case-sensitive and search_key stores a lowercased copy. Upper-case input is the
	// case that would silently return nothing if either half were dropped.
	t.Run("searches case-insensitively", func(t *testing.T) {
		res, err := repo.List(ctx, domain.ListCouponsRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 50},
			Search:            "DIWALI",
		})
		require.NoError(t, err)
		require.Len(t, res.Coupons, 1)
		require.Equal(t, "coupon_l1", res.Coupons[0].ID)
	})

	// Newest first, out of the index rather than a sort in Go. These were created in
	// order, so the listing has to come back reversed.
	t.Run("returns newest first", func(t *testing.T) {
		res, err := repo.List(ctx, domain.ListCouponsRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 50},
		})
		require.NoError(t, err)
		require.Len(t, res.Coupons, 3)
		require.Equal(t, []string{"coupon_l3", "coupon_l2", "coupon_l1"},
			[]string{res.Coupons[0].ID, res.Coupons[1].ID, res.Coupons[2].ID},
			"the index supplies the order; a stable one is what makes the cursor safe")
	})

	t.Run("paginates", func(t *testing.T) {
		res, err := repo.List(ctx, domain.ListCouponsRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 2},
		})
		require.NoError(t, err)
		require.Len(t, res.Coupons, 2)
		require.True(t, res.Pagination.HasMore)
	})

	// The cursor is a DynamoDB LastEvaluatedKey now, not an offset into a slice the
	// repository had already read. Following it has to reach the rest exactly once.
	t.Run("the cursor walks the rest without repeating", func(t *testing.T) {
		first, err := repo.List(ctx, domain.ListCouponsRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 2},
		})
		require.NoError(t, err)
		require.Len(t, first.Coupons, 2)
		require.NotEmpty(t, first.Pagination.NextCursor, "a cursor is the point of paging in the index")

		second, err := repo.List(ctx, domain.ListCouponsRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 2, Cursor: first.Pagination.NextCursor},
		})
		require.NoError(t, err)
		require.Len(t, second.Coupons, 1, "three public coupons, so the second page holds the last")

		seen := map[string]bool{}
		for _, c := range append(first.Coupons, second.Coupons...) {
			require.False(t, seen[c.ID], "coupon %s came back on both pages", c.ID)
			seen[c.ID] = true
		}
		require.Len(t, seen, 3, "every public coupon reached exactly one page")
	})

	// A filter runs after the read in DynamoDB, so the first round can come back
	// short. The page still has to be full when the index holds enough rows.
	t.Run("a filtered page is still filled", func(t *testing.T) {
		status := domain.CouponStatusActive
		res, err := repo.List(ctx, domain.ListCouponsRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 2},
			Status:            &status,
		})
		require.NoError(t, err)
		// coupon_l1 and coupon_l3 are ACTIVE; coupon_l2 is not.
		require.Len(t, res.Coupons, 2, "the shortfall from the filter has to be re-queried")
		for _, c := range res.Coupons {
			require.Equal(t, domain.CouponStatusActive, c.Status)
		}
	})
}

func TestCouponRepository_IncrementUsage(t *testing.T) {
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testCouponsTable)

	repo := NewCouponRepository(wrapped)
	ctx := context.Background()

	t.Run("claims up to the limit and then refuses", func(t *testing.T) {
		c := newTestCoupon("coupon_lim", "LIMIT2")
		c.UsageLimit = 2
		require.NoError(t, repo.Create(ctx, c))

		for i := 1; i <= 2; i++ {
			claimed, err := repo.IncrementUsage(ctx, "coupon_lim")
			require.NoError(t, err)
			require.True(t, claimed, "claim %d should succeed", i)
		}

		claimed, err := repo.IncrementUsage(ctx, "coupon_lim")
		require.NoError(t, err, "exhaustion is an outcome, not an error")
		require.False(t, claimed)
	})

	t.Run("treats a zero limit as unlimited", func(t *testing.T) {
		c := newTestCoupon("coupon_unl", "UNLIMITED")
		c.UsageLimit = 0
		require.NoError(t, repo.Create(ctx, c))

		for i := 0; i < 5; i++ {
			claimed, err := repo.IncrementUsage(ctx, "coupon_unl")
			require.NoError(t, err)
			require.True(t, claimed)
		}
	})

	// The reason this method exists. A read-modify-write lets racers all read
	// usage_count == 0, all decide they are under the limit, and all succeed —
	// giving a single-use code away many times over. 16 racers (not 2) so a
	// read-modify-write implementation would have to win a 16-way interleaving
	// lottery to slip through undetected.
	t.Run("lets exactly one of many concurrent claims win on a single-use code", func(t *testing.T) {
		c := newTestCoupon("coupon_race", "ONESHOT")
		c.UsageLimit = 1
		require.NoError(t, repo.Create(ctx, c))

		const racers = 16
		var (
			wg     sync.WaitGroup
			mu     sync.Mutex
			claims int
		)
		for i := 0; i < racers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				claimed, err := repo.IncrementUsage(context.Background(), "coupon_race")
				if err == nil && claimed {
					mu.Lock()
					claims++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()

		require.Equal(t, 1, claims, "a coupon with usage_limit 1 must be claimable once")
	})
}

func TestCouponRepository_CustomerUsage(t *testing.T) {
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testCouponsTable)

	repo := NewCouponRepository(wrapped)
	ctx := context.Background()

	t.Run("reports zero for a customer who has never used it", func(t *testing.T) {
		n, err := repo.GetCustomerUsage(ctx, "cust_new", "coupon_any")
		require.NoError(t, err, "no counter yet is zero, not an error")
		require.Equal(t, 0, n)
	})

	t.Run("counts per customer per coupon", func(t *testing.T) {
		// limit 0 is unlimited, so every claim succeeds.
		for _, args := range [][2]string{
			{"cust_1", "coupon_a"}, {"cust_1", "coupon_a"},
			{"cust_1", "coupon_b"}, {"cust_2", "coupon_a"},
		} {
			claimed, err := repo.IncrementCustomerUsage(ctx, args[0], args[1], 0)
			require.NoError(t, err)
			require.True(t, claimed)
		}

		n, err := repo.GetCustomerUsage(ctx, "cust_1", "coupon_a")
		require.NoError(t, err)
		require.Equal(t, 2, n)

		n, err = repo.GetCustomerUsage(ctx, "cust_1", "coupon_b")
		require.NoError(t, err)
		require.Equal(t, 1, n)

		n, err = repo.GetCustomerUsage(ctx, "cust_2", "coupon_a")
		require.NoError(t, err)
		require.Equal(t, 1, n)
	})

	// The counter only moves at payment success, so the window between a customer
	// initiating twice and paying both is wide open — seconds or minutes, not a race.
	// The condition is what closes it.
	t.Run("refuses a second claim against a limit of one", func(t *testing.T) {
		claimed, err := repo.IncrementCustomerUsage(ctx, "cust_once", "coupon_once", 1)
		require.NoError(t, err)
		require.True(t, claimed)

		claimed, err = repo.IncrementCustomerUsage(ctx, "cust_once", "coupon_once", 1)
		require.NoError(t, err, "a spent allowance is an outcome, not an error")
		require.False(t, claimed)

		n, err := repo.GetCustomerUsage(ctx, "cust_once", "coupon_once")
		require.NoError(t, err)
		require.Equal(t, 1, n, "the refused claim must not have moved the counter")
	})

	t.Run("claims up to a higher limit and then refuses", func(t *testing.T) {
		for i := 1; i <= 3; i++ {
			claimed, err := repo.IncrementCustomerUsage(ctx, "cust_three", "coupon_three", 3)
			require.NoError(t, err)
			require.True(t, claimed, "claim %d should succeed", i)
		}

		claimed, err := repo.IncrementCustomerUsage(ctx, "cust_three", "coupon_three", 3)
		require.NoError(t, err)
		require.False(t, claimed)
	})

	t.Run("treats a zero limit as unlimited", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			claimed, err := repo.IncrementCustomerUsage(ctx, "cust_unl", "coupon_unl", 0)
			require.NoError(t, err)
			require.True(t, claimed)
		}

		n, err := repo.GetCustomerUsage(ctx, "cust_unl", "coupon_unl")
		require.NoError(t, err)
		require.Equal(t, 5, n)
	})

	// Same reasoning as the global limit's race test: a read-then-write guard would let
	// every racer read 0, decide it is under the limit, and all succeed.
	t.Run("lets exactly one of many concurrent claims win on a limit of one", func(t *testing.T) {
		const racers = 16
		var (
			wg     sync.WaitGroup
			mu     sync.Mutex
			claims int
		)
		for i := 0; i < racers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				claimed, err := repo.IncrementCustomerUsage(
					context.Background(), "cust_race", "coupon_race_pu", 1)
				if err == nil && claimed {
					mu.Lock()
					claims++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()

		require.Equal(t, 1, claims, "usage_per_user 1 must be claimable once")
	})
}

// publicCoupon builds an advertisable coupon, then lets each case break one rule.
func publicCoupon(id, code string) *domain.Coupon {
	c := newTestCoupon(id, code)
	c.Audience = domain.AudienceAll
	c.Status = domain.CouponStatusActive
	return c
}

func TestCouponRepository_ListPublic(t *testing.T) {
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testCouponsTable)

	repo := NewCouponRepository(wrapped)
	ctx := context.Background()
	now := time.Now()

	inactive := publicCoupon("coupon_inactive", "INACTIVE1")
	inactive.Status = domain.CouponStatusInactive

	firstOrder := publicCoupon("coupon_first", "FIRSTONLY")
	firstOrder.Audience = domain.AudienceFirstOrder

	expired := publicCoupon("coupon_expired", "GONE10")
	expiredAt := now.Add(-time.Hour)
	expired.ValidUntil = &expiredAt

	insideWindow := publicCoupon("coupon_soon", "SOON10")
	soon := now.Add(domain.PublicCouponListTTL / 2)
	insideWindow.ValidUntil = &soon

	notYet := publicCoupon("coupon_future", "LATER10")
	notYet.ValidFrom = now.Add(24 * time.Hour)

	openEnded := publicCoupon("coupon_open", "FOREVER10")
	openEnded.ValidUntil = nil

	live := publicCoupon("coupon_live", "LIVE10")

	for _, c := range []*domain.Coupon{
		inactive, firstOrder, expired, insideWindow, notYet, openEnded, live,
	} {
		require.NoError(t, repo.Create(ctx, c))
	}

	got, err := repo.ListPublic(ctx, now.Add(domain.PublicCouponListTTL))
	require.NoError(t, err)

	ids := make([]string, 0, len(got))
	for _, c := range got {
		ids = append(ids, c.ID)
	}
	require.ElementsMatch(t, []string{"coupon_open", "coupon_live"}, ids,
		"only ACTIVE + ALL coupons valid past the cache window may be advertised")
}

// The live picker's cutoff (now) must not hide a coupon Validate would still accept,
// even though the cached banner's cutoff (now+TTL) rightly does.
func TestCouponRepository_ListPublic_CutoffIsCallerControlled(t *testing.T) {
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testCouponsTable)

	repo := NewCouponRepository(wrapped)
	ctx := context.Background()
	now := time.Now()

	expiringSoon := publicCoupon("coupon_expiring_soon", "SOON5")
	soon := now.Add(30 * time.Minute)
	expiringSoon.ValidUntil = &soon
	require.NoError(t, repo.Create(ctx, expiringSoon))

	banner, err := repo.ListPublic(ctx, now.Add(domain.PublicCouponListTTL))
	require.NoError(t, err)
	require.Empty(t, banner, "a coupon expiring inside the cache window must not be cached")

	picker, err := repo.ListPublic(ctx, now)
	require.NoError(t, err)
	ids := make([]string, 0, len(picker))
	for _, c := range picker {
		ids = append(ids, c.ID)
	}
	require.Contains(t, ids, "coupon_expiring_soon",
		"the live picker must show what Validate would accept typed into the box beside it")
}

// One coupon sits just inside the cache horizon (must be dropped) and one just
// outside it with a stray microsecond, proving the RFC3339Nano trap is guarded.
func TestCouponRepository_ListPublic_SubSecondExpiry(t *testing.T) {
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testCouponsTable)

	repo := NewCouponRepository(wrapped)
	ctx := context.Background()
	now := time.Now()

	justInside := publicCoupon("coupon_inside", "INSIDE10")
	insideEnd := now.Add(domain.PublicCouponListTTL - time.Minute).Truncate(time.Second)
	justInside.ValidUntil = &insideEnd

	justOutside := publicCoupon("coupon_outside", "OUTSIDE10")
	outsideEnd := now.Add(domain.PublicCouponListTTL + time.Minute).Add(time.Microsecond)
	justOutside.ValidUntil = &outsideEnd

	require.NoError(t, repo.Create(ctx, justInside))
	require.NoError(t, repo.Create(ctx, justOutside))

	got, err := repo.ListPublic(ctx, now.Add(domain.PublicCouponListTTL))
	require.NoError(t, err)

	ids := make([]string, 0, len(got))
	for _, c := range got {
		ids = append(ids, c.ID)
	}
	require.ElementsMatch(t, []string{"coupon_outside"}, ids,
		"only the coupon clearing the cache window may be advertised")
}

// One query for every count a customer holds. Validates GetCustomerUsage is a GetItem
// per coupon, which a picker of M candidates would pay M times.
func TestCouponRepository_GetCustomerUsageAll(t *testing.T) {
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testCouponsTable)

	repo := NewCouponRepository(wrapped)
	ctx := context.Background()

	claimed, err := repo.IncrementCustomerUsage(ctx, "cust_1", "coupon_a", 0)
	require.NoError(t, err)
	require.True(t, claimed)

	_, err = repo.IncrementCustomerUsage(ctx, "cust_1", "coupon_a", 0)
	require.NoError(t, err)
	_, err = repo.IncrementCustomerUsage(ctx, "cust_1", "coupon_b", 0)
	require.NoError(t, err)
	_, err = repo.IncrementCustomerUsage(ctx, "cust_2", "coupon_a", 0)
	require.NoError(t, err)

	counts, err := repo.GetCustomerUsageAll(ctx, "cust_1")
	require.NoError(t, err)
	require.Equal(t, map[string]int{"coupon_a": 2, "coupon_b": 1}, counts,
		"another customer's counters must not leak in")

	empty, err := repo.GetCustomerUsageAll(ctx, "cust_never")
	require.NoError(t, err)
	require.Empty(t, empty, "never used is an empty map, not an error")
}
