# Coupon Public Offers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a cached public offers banner to the storefront homepage and a live coupon picker to checkout, both showing only coupons whose eligibility rules the server actually enforces.

**Architecture:** Two new store endpoints. `GET /api/v1/store/catalog/coupons` reads the existing `GSI1PK = COUPON#ALL` partition filtered to `ACTIVE` + `audience = ALL`, drops anything expiring inside the cache window in Go, and returns a trimmed DTO — served by `store-catalog`, which already stamps `Cache-Control: public, max-age=3600` on every GET. `GET /api/v1/store/checkout/coupons` takes the same candidate set, joins one bulk per-customer usage query, and annotates each coupon with its real saving against the customer's cart. No new index, no new Lambda, no backend cache.

**Tech Stack:** Go 1.25, Chi, DynamoDB (aws-sdk-go-v2), Google Wire, gomock/testify; Next.js 16 + React 19 + Mantine on the storefront; Playwright for e2e.

**Spec:** `docs/superpowers/specs/2026-08-28-coupon-public-offers-design.md`

## Global Constraints

- **Base branch:** `feat/coupons-phase-1` (#275). Branch this work as `feat/coupon-public-offers`. Do not branch off `main` — the coupons table, service and repository do not exist there.
- **Money is paise.** `int64` everywhere. `Coupon.Value` is percentage × 100 for `PERCENTAGE`, paise for `FIXED`.
- **Only `audience = ALL` coupons may be advertised.** `CouponService.Validate` does not read `coupon.Audience`; enforcement is #255. Nothing in this plan surfaces `FIRST_ORDER`, `RETURNING` or `SPECIFIC_CUSTOMER` coupons.
- **`valid_until` window checks happen in Go, never in a `FilterExpression`.** `attributevalue` marshals `time.Time` as `time.RFC3339Nano`, which trims trailing zeros, so the stored string is variable-width and compares wrong inside a single second.
- **Comments cap at two lines.** Repo convention.
- **Store DTOs are explicit structs, never `*domain.Coupon`.** `usage_count`, `usage_limit`, `usage_per_user`, `customer_id`, `id`, `batch_id`, `search_key`, `status`, `audience`, `combines_with_offers` and timestamps must not appear in any store payload.
- **Coupon read failures return `200` with an empty list**, logged at warn. A dead coupon path must not blank the homepage or block a checkout.
- **Repository tests need DynamoDB Local.** Run `make docker-up` first. `skipIfNoLocal` (`internal/repository/dynamodb/testhelper_test.go:144`) silently skips when it is absent unless `CI` is set — always run these with `CI=1` so a skip becomes a failure.
- **After changing `internal/domain/coupon.go` or `internal/domain/store_service.go`:** run `make generate-mocks` so local builds and tests see the new interface methods. **Never `git add internal/mocks/`** — `.gitignore:77` ignores `internal/mocks/*_mock.go` and `ci.yml:92` regenerates them; no mock is tracked. After changing Wire providers or an `Initialize*Deps` build list: run `make wire`, and DO commit the regenerated `wire_gen.go`, which is tracked.

---

## File Structure

**Backend — `handloom-admin/`**

| file | responsibility |
|---|---|
| `internal/domain/coupon.go` | Modify: add `ListPublic` + `GetCustomerUsageAll` to `CouponRepository`, `ListPublic` + `ListForCart` to `CouponService`, and the `CouponOffer` type |
| `internal/repository/dynamodb/coupon_repository.go` | Modify: implement both reads |
| `internal/repository/dynamodb/coupon_repository_test.go` | Modify: cover both reads against DynamoDB Local |
| `internal/service/coupon_service.go` | Modify: extract `evaluate`, implement `ListPublic` + `ListForCart` |
| `internal/service/coupon_service_test.go` | Modify: cover the new methods; existing `Validate` tests stay untouched |
| `internal/handler/store/catalog_coupons.go` | Create: public offers DTO, handler, and the `StoreCoupon` mapper |
| `internal/handler/store/catalog_coupons_test.go` | Create: withheld-field and error-path assertions |
| `internal/handler/store/catalog_handler.go` | Modify: hold `domain.CouponService`, register `GET /coupons` |
| `internal/handler/store/checkout_handler.go` | Modify: register `GET /coupons`, add the picker handler and its DTO |
| `internal/handler/store/checkout_handler_test.go` | Modify or create: picker payload assertions |
| `internal/domain/store_service.go` | Modify: add `ListCoupons` to `CheckoutService` |
| `internal/service/checkout_service.go` | Modify: implement `ListCoupons` |
| `internal/service/checkout_service_test.go` | Modify: cover `ListCoupons` |
| `internal/wire/providers.go` | Modify: `ProvideStoreCatalogHandler` gains `*service.CouponService` |
| `internal/wire/wire.go` | Modify: `InitializeStoreCatalogDeps` builds the coupon repo + service |

**Storefront — `homechrome-store/`**

| file | responsibility |
|---|---|
| `src/types/index.ts` | Modify: `PublicCoupon` and `CouponOffer` types |
| `src/lib/routes.ts` | Modify: `CATALOG.COUPONS` and `CHECKOUT.COUPONS` path constants |
| `src/components/catalog/OffersBanner.tsx` | Create: the green banner, a presentational component |
| `src/app/page.tsx` | Modify: fetch public coupons with `revalidate: 3600` |
| `src/app/HomeView.tsx` | Modify: accept and render `coupons` |
| `src/components/checkout/CouponPicker.tsx` | Create: the checkout dropdown |
| `src/app/checkout/ReviewStep.tsx` | Modify: mount the picker beside the existing code input |

**E2E — `e2e/`**

| file | responsibility |
|---|---|
| `specs/coupons/public-offers.spec.ts` | Create: public list and picker coverage |

---

### Task 1: Repository — `ListPublic`

**Files:**
- Modify: `handloom-admin/internal/domain/coupon.go` (the `CouponRepository` interface, after `List`)
- Modify: `handloom-admin/internal/repository/dynamodb/coupon_repository.go`
- Test: `handloom-admin/internal/repository/dynamodb/coupon_repository_test.go`

**Interfaces:**
- Consumes: `QueryAll[T]` (`internal/repository/dynamodb/pagination.go:231`), consts `exprPK`, `nameStatus`, `attrStatus`, `valStatus` (`internal/repository/dynamodb/constants.go`), test helpers `testWrappedClient`, `skipIfNoLocal`, `setupTestTable`, `testCouponsTable`, `newTestCoupon` (`coupon_repository_test.go:16`)
- Produces: `domain.PublicCouponListTTL` and `CouponRepository.ListPublic(ctx) ([]*domain.Coupon, error)`, consumed by Tasks 3 and 4

**Where the TTL const lives:** `internal/domain/coupon.go`, not the repository package. The repository filters by it and the handler's `Cache-Control` must agree with it; putting it in `domain` lets both read one value without a handler importing a repository.

- [ ] **Step 1: Write the failing test**

Append to `handloom-admin/internal/repository/dynamodb/coupon_repository_test.go`:

```go
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

	got, err := repo.ListPublic(ctx)
	require.NoError(t, err)

	ids := make([]string, 0, len(got))
	for _, c := range got {
		ids = append(ids, c.ID)
	}
	require.ElementsMatch(t, []string{"coupon_open", "coupon_live"}, ids,
		"only ACTIVE + ALL coupons valid past the cache window may be advertised")
}

// The stored valid_until is RFC3339Nano, which trims trailing zeros — so a whole
// second and a fraction later are different widths. Fails if the window check
// ever moves into a DynamoDB string comparison.
func TestCouponRepository_ListPublic_SubSecondExpiry(t *testing.T) {
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testCouponsTable)

	repo := NewCouponRepository(wrapped)
	ctx := context.Background()
	base := time.Now().Add(2 * domain.PublicCouponListTTL).Truncate(time.Second)

	whole := publicCoupon("coupon_whole", "WHOLE10")
	wholeEnd := base
	whole.ValidUntil = &wholeEnd

	fraction := publicCoupon("coupon_fraction", "FRACTION10")
	fractionEnd := base.Add(time.Microsecond)
	fraction.ValidUntil = &fractionEnd

	require.NoError(t, repo.Create(ctx, whole))
	require.NoError(t, repo.Create(ctx, fraction))

	got, err := repo.ListPublic(ctx)
	require.NoError(t, err)
	require.Len(t, got, 2, "both are valid well past the window and must both survive")
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd handloom-admin && make docker-up
CI=1 go test -run 'TestCouponRepository_ListPublic' ./internal/repository/dynamodb/ -v
```

Expected: compile failure — `repo.ListPublic undefined` and `undefined: domain.PublicCouponListTTL`.

- [ ] **Step 3: Add the const and the interface method**

In `handloom-admin/internal/domain/coupon.go`, directly above `// Coupon represents a discount coupon`:

```go
// PublicCouponListTTL is how long a public coupon payload may be cached. ListPublic
// drops coupons expiring inside it, so a cached payload cannot advertise a dead code.
// Lives here because the repository filters by it and the handler's Cache-Control
// must match it.
const PublicCouponListTTL = time.Hour
```

Then inside `CouponRepository`, directly after the `List` method:

```go
	// ListPublic returns coupons safe to advertise: ACTIVE, audience ALL, and valid
	// past the cache window. Not paginated — a banner's worth is small by design.
	ListPublic(ctx context.Context) ([]*Coupon, error)
```

- [ ] **Step 4: Implement it**

In `handloom-admin/internal/repository/dynamodb/coupon_repository.go`, above `func (r *CouponRepository) List(`:

```go
// ListPublic reads the advertisable coupons. Status and audience filter in DynamoDB;
// the validity window is checked in Go — see the comment inside.
func (r *CouponRepository) ListPublic(ctx context.Context) ([]*domain.Coupon, error) {
	all, err := QueryAll[domain.Coupon](ctx, r.client.db, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.couponsTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk"),
		FilterExpression:       aws.String("#status = :status AND #audience = :audience"),
		ExpressionAttributeNames: map[string]string{
			nameStatus: attrStatus,
			// Named rather than inline: cheaper than being wrong about the reserved list.
			"#audience": "audience",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK:      &types.AttributeValueMemberS{Value: "COUPON#ALL"},
			valStatus:   &types.AttributeValueMemberS{Value: string(domain.CouponStatusActive)},
			":audience": &types.AttributeValueMemberS{Value: string(domain.AudienceAll)},
		},
		ScanIndexForward: aws.Bool(false),
	}, "Failed to list public coupons")
	if err != nil {
		return nil, err
	}

	// valid_until marshals as RFC3339Nano, which trims trailing zeros, so the stored
	// string is variable-width and compares wrong inside one second. Filtered here.
	now := time.Now()
	horizon := now.Add(domain.PublicCouponListTTL)
	live := make([]*domain.Coupon, 0, len(all))
	for _, c := range all {
		if now.Before(c.ValidFrom) {
			continue
		}
		if c.ValidUntil != nil && c.ValidUntil.Before(horizon) {
			continue
		}
		live = append(live, c)
	}
	return live, nil
}
```

- [ ] **Step 5: Run the tests to verify they pass**

```bash
CI=1 go test -run 'TestCouponRepository_ListPublic' ./internal/repository/dynamodb/ -v
```

Expected: both PASS, no SKIP lines.

- [ ] **Step 6: Prove the window predicate is load-bearing**

Temporarily change `c.ValidUntil.Before(horizon)` to `c.ValidUntil.Before(now)` and re-run. Expected: `TestCouponRepository_ListPublic` FAILS because `coupon_soon` appears. Restore the line and confirm PASS again.

- [ ] **Step 7: Regenerate mocks and commit**

```bash
cd handloom-admin && make generate-mocks
go build ./... && golangci-lint run internal/...
git add internal/domain/coupon.go internal/repository/dynamodb/coupon_repository.go \
        internal/repository/dynamodb/coupon_repository_test.go
git commit -m "feat(coupons): read the coupons safe to advertise"
```

---

### Task 2: Repository — `GetCustomerUsageAll`

**Files:**
- Modify: `handloom-admin/internal/domain/coupon.go` (the `CouponRepository` interface, after `GetCustomerUsage`)
- Modify: `handloom-admin/internal/repository/dynamodb/coupon_repository.go`
- Test: `handloom-admin/internal/repository/dynamodb/coupon_repository_test.go`

**Interfaces:**
- Consumes: `QueryAll[T]`, `exprPK`, `domain.CouponUseCounter`, `CouponRepository.IncrementCustomerUsage`
- Produces: `CouponRepository.GetCustomerUsageAll(ctx, customerID string) (map[string]int, error)`, consumed by Task 3

- [ ] **Step 1: Write the failing test**

Append to `handloom-admin/internal/repository/dynamodb/coupon_repository_test.go`:

```go
// One query for every count a customer holds. Validate's GetCustomerUsage is a GetItem
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
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd handloom-admin && CI=1 go test -run TestCouponRepository_GetCustomerUsageAll ./internal/repository/dynamodb/ -v
```

Expected: compile failure — `repo.GetCustomerUsageAll undefined`.

- [ ] **Step 3: Add the interface method**

In `handloom-admin/internal/domain/coupon.go`, inside `CouponRepository`, directly after `GetCustomerUsage`:

```go
	// GetCustomerUsageAll returns every per-coupon count this customer holds, keyed by
	// coupon id. One query, so pricing M candidates costs one read rather than M.
	GetCustomerUsageAll(ctx context.Context, customerID string) (map[string]int, error)
```

- [ ] **Step 4: Implement it**

In `handloom-admin/internal/repository/dynamodb/coupon_repository.go`, directly after `GetCustomerUsage`:

```go
// GetCustomerUsageAll reads the whole CUSTOMER#<id> counter partition in one query.
// Keys come from the item's coupon_id, which IncrementCustomerUsage always sets.
func (r *CouponRepository) GetCustomerUsageAll(
	ctx context.Context, customerID string,
) (map[string]int, error) {
	counters, err := QueryAll[domain.CouponUseCounter](ctx, r.client.db, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.couponsTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK:    &types.AttributeValueMemberS{Value: "CUSTOMER#" + customerID},
			":prefix": &types.AttributeValueMemberS{Value: "USE#"},
		},
	}, "Failed to read coupon usage")
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int, len(counters))
	for _, c := range counters {
		counts[c.CouponID] = c.Count
	}
	return counts, nil
}
```

- [ ] **Step 5: Run the test to verify it passes**

```bash
CI=1 go test -run TestCouponRepository_GetCustomerUsageAll ./internal/repository/dynamodb/ -v
```

Expected: PASS, no SKIP.

- [ ] **Step 6: Regenerate mocks and commit**

```bash
cd handloom-admin && make generate-mocks
go build ./... && golangci-lint run internal/...
git add internal/domain/coupon.go internal/repository/dynamodb/coupon_repository.go \
        internal/repository/dynamodb/coupon_repository_test.go
git commit -m "feat(coupons): read every usage counter a customer holds in one query"
```

---

### Task 3: Service — extract `evaluate`, add `ListPublic` and `ListForCart`

**Files:**
- Modify: `handloom-admin/internal/domain/coupon.go` (the `CouponService` interface, plus a new `CouponOffer` type)
- Modify: `handloom-admin/internal/service/coupon_service.go:180-266` (`Validate`)
- Test: `handloom-admin/internal/service/coupon_service_test.go`

**Interfaces:**
- Consumes: `CouponRepository.ListPublic` (Task 1), `CouponRepository.GetCustomerUsageAll` (Task 2), `computeCouponDiscount` and `minPayableAmount` (`coupon_service.go`)
- Produces: `domain.CouponOffer{Coupon *Coupon; Eligible bool; DiscountAmount int64; Reason string}`, `CouponService.ListPublic(ctx) ([]*Coupon, error)`, `CouponService.ListForCart(ctx, cc CouponContext) ([]*CouponOffer, error)` — consumed by Tasks 4 and 5

**Why the extraction:** `Validate`'s branches each build a `CouponValidationResult`. The picker needs the same verdicts for M coupons without re-reading a usage counter per coupon. Extracting the post-lookup body means one function computes both, so the picker cannot promise a saving `Validate` then refuses. **Every existing `TestCouponService_Validate` subtest must pass unmodified — that is this task's real gate.**

- [ ] **Step 1: Write the failing tests**

Append to `handloom-admin/internal/service/coupon_service_test.go`:

```go
func TestCouponService_ListPublic(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockCouponRepository(ctrl)

	repo.EXPECT().ListPublic(gomock.Any()).Return([]*domain.Coupon{activeCoupon()}, nil)

	got, err := NewCouponService(repo).ListPublic(ctx)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "FESTIVE20", got[0].Code)
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

	setup := func(t *testing.T, counts map[string]int) *CouponService {
		t.Helper()
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockCouponRepository(ctrl)
		repo.EXPECT().ListPublic(gomock.Any()).
			Return([]*domain.Coupon{activeCoupon(), withMinimum()}, nil)
		// Exactly once for the whole list — the N+1 guard.
		repo.EXPECT().GetCustomerUsageAll(gomock.Any(), "cust_1").Return(counts, nil).Times(1)
		return NewCouponService(repo)
	}

	t.Run("annotates each coupon against the cart", func(t *testing.T) {
		s := setup(t, nil)

		offers, err := s.ListForCart(ctx, domain.CouponContext{
			CartTotal: 100000, CustomerID: "cust_1",
		})
		require.NoError(t, err)
		require.Len(t, offers, 2)

		require.True(t, offers[0].Eligible, "eligible coupons sort first")
		require.Equal(t, "FESTIVE20", offers[0].Coupon.Code)
		require.Equal(t, int64(20000), offers[0].DiscountAmount)
		require.Empty(t, offers[0].Reason)

		require.False(t, offers[1].Eligible)
		require.Equal(t, "BIG500", offers[1].Coupon.Code)
		require.Zero(t, offers[1].DiscountAmount)
		require.Equal(t, "Add ₹1,000 more to use this coupon", offers[1].Reason,
			"the reason must be the message Validate would return for the same cart")
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
}
```

For the second subtest to mean anything, `activeCoupon()` must carry a per-user cap. Add one line to the existing `activeCoupon` helper (`coupon_service_test.go:16`), after `Value: 2000,`:

```go
		UsagePerUser: 1,
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd handloom-admin && go test -run 'TestCouponService_List' ./internal/service/ -v
```

Expected: compile failure — `s.ListPublic undefined`, `s.ListForCart undefined`, `undefined: domain.CouponOffer`.

- [ ] **Step 3: Add the domain type and interface methods**

In `handloom-admin/internal/domain/coupon.go`, directly above `// ==================== COUPON SERVICE ====================`:

```go
// CouponOffer is one coupon priced against a specific cart. Reason carries the
// customer-facing message when Eligible is false, so a picker can say why.
type CouponOffer struct {
	Coupon         *Coupon `json:"coupon"`
	Eligible       bool    `json:"eligible"`
	DiscountAmount int64   `json:"discount_amount"`
	Reason         string  `json:"reason,omitempty"`
}
```

Inside `CouponService`, directly after `Validate`:

```go
	// ListPublic returns the coupons safe to advertise, with no cart context.
	ListPublic(ctx context.Context) ([]*Coupon, error)

	// ListForCart prices every public coupon against one cart. Eligible offers come
	// first, best saving down; ineligible ones follow, each carrying its reason.
	ListForCart(ctx context.Context, cc CouponContext) ([]*CouponOffer, error)
}
```

Remember to delete the old closing `}` of the interface so it is not doubled.

- [ ] **Step 4: Extract `evaluate` out of `Validate`**

Replace the body of `Validate` in `handloom-admin/internal/service/coupon_service.go` from `coupon, err := s.couponRepo.GetByCode(ctx, code)` to the end of the function with:

```go
	coupon, err := s.couponRepo.GetByCode(ctx, code)
	if err != nil || coupon == nil {
		return reject(outcomeInvalid, "That code isn't valid")
	}

	used := 0
	if coupon.UsagePerUser > 0 && cc.CustomerID != "" {
		if n, usageErr := s.couponRepo.GetCustomerUsage(ctx, cc.CustomerID, coupon.ID); usageErr == nil {
			used = n
		}
	}

	result := evaluate(coupon, cc, used)
	if !result.Valid {
		outcome = result.Outcome
	}
	return result, nil
}

// evaluate is the whole eligibility rule, given a coupon and how many times this
// customer has already redeemed it. Validate and ListForCart share it so they cannot
// disagree about what a code is worth.
func evaluate(coupon *domain.Coupon, cc domain.CouponContext, used int) *domain.CouponValidationResult {
	reject := func(o, message string) *domain.CouponValidationResult {
		return &domain.CouponValidationResult{
			Valid: false, Code: coupon.Code, Outcome: o, ErrorMessage: message,
		}
	}

	if coupon.Status != domain.CouponStatusActive {
		return reject(outcomeInvalid, "This coupon is no longer available")
	}

	now := time.Now()
	if now.Before(coupon.ValidFrom) {
		return reject(outcomeInvalid, "This coupon isn't active yet")
	}
	// A nil ValidUntil is open-ended and never expires — the operator's off switch is
	// how such a coupon ends.
	if coupon.ValidUntil != nil && now.After(*coupon.ValidUntil) {
		return reject("expired", "This coupon has expired")
	}

	if cc.CartTotal < coupon.MinOrderValue {
		short := coupon.MinOrderValue - cc.CartTotal
		return reject(outcomeInvalid, fmt.Sprintf("Add %s more to use this coupon", formatPaise(short)))
	}

	if coupon.UsageLimit > 0 && coupon.UsageCount >= coupon.UsageLimit {
		return reject("limit_reached", "This coupon has been fully claimed")
	}

	if coupon.UsagePerUser > 0 && used >= coupon.UsagePerUser {
		return reject("limit_reached", "You've already used this coupon")
	}

	// The stacking gate. Off unless the operator turned it on, because buy-2-get-1 is a
	// third off before any code and 20% on top of it reaches 46.7%.
	if cc.HasAutomaticOffer && !coupon.CombinesWithOffers {
		return reject(outcomeInvalid, "This code can't be used with the offer already in your cart")
	}

	discount, trimmed := computeCouponDiscount(coupon, cc.CartTotal)

	result := &domain.CouponValidationResult{
		Valid:          true,
		CouponID:       coupon.ID,
		Code:           coupon.Code,
		DiscountAmount: discount,
	}
	// Applied, not refused. The code is worth more than the order, so it pays out what
	// it can and says so, rather than zeroing a total the gateway would then reject.
	if trimmed > 0 {
		result.Notice = fmt.Sprintf(
			"This code is worth more than your order, so we've taken off %s and left %s to pay",
			formatPaise(discount), formatPaise(minPayableAmount))
	}
	return result
}
```

`Validate` keeps its metric `defer` and its own `reject` closure untouched above this point.

`evaluate` needs to report which outcome label the metric should record, so add one field to `CouponValidationResult` in `handloom-admin/internal/domain/coupon.go`:

```go
	// Outcome is the metric label for a rejection ("invalid", "expired",
	// "limit_reached"). Internal — never serialized to a customer.
	Outcome string `json:"-"`
```

- [ ] **Step 5: Run the existing `Validate` tests — the regression gate**

```bash
cd handloom-admin && go test -run TestCouponService_Validate ./internal/service/ -v
```

Expected: every subtest PASSES, with no edits to any of them. If one fails, the extraction changed behaviour — fix `evaluate`, not the test.

- [ ] **Step 6: Implement the two new service methods**

Append to `handloom-admin/internal/service/coupon_service.go`:

```go
// ListPublic returns the coupons safe to advertise.
func (s *CouponService) ListPublic(ctx context.Context) ([]*domain.Coupon, error) {
	return s.couponRepo.ListPublic(ctx)
}

// ListForCart prices every public coupon against one cart, in one usage read.
func (s *CouponService) ListForCart(
	ctx context.Context, cc domain.CouponContext,
) ([]*domain.CouponOffer, error) {
	coupons, err := s.couponRepo.ListPublic(ctx)
	if err != nil {
		return nil, err
	}

	// One query for every counter this customer holds. Asking per coupon would be a
	// read per candidate.
	var used map[string]int
	if cc.CustomerID != "" {
		if counts, usageErr := s.couponRepo.GetCustomerUsageAll(ctx, cc.CustomerID); usageErr == nil {
			used = counts
		} else {
			slog.WarnContext(ctx, "Coupon usage counters unavailable", "error", usageErr)
		}
	}

	offers := make([]*domain.CouponOffer, 0, len(coupons))
	for _, c := range coupons {
		v := evaluate(c, cc, used[c.ID])
		offers = append(offers, &domain.CouponOffer{
			Coupon:         c,
			Eligible:       v.Valid,
			DiscountAmount: v.DiscountAmount,
			Reason:         v.ErrorMessage,
		})
	}

	// Eligible first, best saving down. Ineligible keep index order, which is
	// newest-first out of GSI1.
	sort.SliceStable(offers, func(i, j int) bool {
		if offers[i].Eligible != offers[j].Eligible {
			return offers[i].Eligible
		}
		return offers[i].DiscountAmount > offers[j].DiscountAmount
	})
	return offers, nil
}
```

Add `"sort"` to the import block.

- [ ] **Step 7: Run the new tests to verify they pass**

```bash
go test -run 'TestCouponService_List' ./internal/service/ -v
```

Expected: PASS. Then the whole package:

```bash
go test -count=1 ./internal/service/
```

- [ ] **Step 8: Regenerate mocks and commit**

```bash
cd handloom-admin && make generate-mocks
go build ./... && golangci-lint run internal/...
git add internal/domain/coupon.go internal/service/coupon_service.go \
        internal/service/coupon_service_test.go
git commit -m "feat(coupons): price every public coupon against a cart in one usage read"
```

---

### Task 4: Public offers endpoint on `store-catalog`

**Files:**
- Create: `handloom-admin/internal/handler/store/catalog_coupons.go`
- Create: `handloom-admin/internal/handler/store/catalog_coupons_test.go`
- Modify: `handloom-admin/internal/handler/store/catalog_handler.go:21-53`
- Modify: `handloom-admin/internal/wire/providers.go:706-712`
- Modify: `handloom-admin/internal/wire/wire.go:468-486`

**Interfaces:**
- Consumes: `CouponService.ListPublic` (Task 3), `middleware.CatalogCacheControl` (`internal/middleware/cache_control.go:11`), `response.Success`
- Produces: `GET /api/v1/store/catalog/coupons`, returning `[]StoreCoupon`. Consumed by Task 6.

**Cache header:** `CatalogHandler.Routes()` already applies `middleware.CatalogCacheControl("public, max-age=3600")` to every GET it serves, so the new route inherits the right header with no work. This deliberately drops the spec's `stale-while-revalidate=60`, which would need a route-scoped middleware for no measurable gain.

- [ ] **Step 1: Write the failing test**

Create `handloom-admin/internal/handler/store/catalog_coupons_test.go`:

```go
package store

import (
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
	h.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/coupons", nil))
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
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd handloom-admin && go test -run TestCatalogHandler_ListCoupons ./internal/handler/store/ -v
```

Expected: compile failure — `NewCatalogHandler` takes 3 arguments, not 4.

- [ ] **Step 3: Create the DTO, mapper and handler**

Create `handloom-admin/internal/handler/store/catalog_coupons.go`:

```go
package store

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/response"
)

// StoreCoupon is the public face of a coupon. An explicit struct rather than the
// entity: usage counters advertise scarcity, and customer_id is another customer's id.
type StoreCoupon struct {
	Code          string     `json:"code"`
	Name          string     `json:"name"`
	Description   string     `json:"description,omitempty"`
	Type          string     `json:"type"`
	Value         int64      `json:"value"`
	MinOrderValue int64      `json:"min_order_value"`
	MaxDiscount   int64      `json:"max_discount,omitempty"`
	ValidUntil    *time.Time `json:"valid_until,omitempty"`
}

func toStoreCoupon(c *domain.Coupon) StoreCoupon {
	return StoreCoupon{
		Code:          c.Code,
		Name:          c.Name,
		Description:   c.Description,
		Type:          string(c.Type),
		Value:         c.Value,
		MinOrderValue: c.MinOrderValue,
		MaxDiscount:   c.MaxDiscount,
		ValidUntil:    c.ValidUntil,
	}
}

// ListCoupons handles GET /api/v1/store/catalog/coupons — the public offers banner.
// A read failure is an empty banner, never a broken homepage.
func (h *CatalogHandler) ListCoupons(w http.ResponseWriter, r *http.Request) {
	coupons, err := h.couponService.ListPublic(r.Context())
	if err != nil {
		slog.WarnContext(r.Context(), "Public coupon list unavailable", "error", err)
		coupons = nil
	}

	out := make([]StoreCoupon, 0, len(coupons))
	for _, c := range coupons {
		out = append(out, toStoreCoupon(c))
	}

	response.Success(w, out)
}
```

- [ ] **Step 4: Wire the service into `CatalogHandler`**

In `handloom-admin/internal/handler/store/catalog_handler.go`, add the field, the constructor parameter, and the route:

```go
type CatalogHandler struct {
	productService   domain.ProductService
	categoryService  domain.CategoryService
	inventoryService domain.InventoryService
	couponService    domain.CouponService
}

// NewCatalogHandler creates a new CatalogHandler.
func NewCatalogHandler(
	ps domain.ProductService,
	cs domain.CategoryService,
	is domain.InventoryService,
	cps domain.CouponService,
) *CatalogHandler {
	return &CatalogHandler{
		productService:   ps,
		categoryService:  cs,
		inventoryService: is,
		couponService:    cps,
	}
}
```

And inside `Routes()`, after the `availability` line:

```go
	r.Get("/coupons", h.ListCoupons)
```

- [ ] **Step 5: Run the test to verify it passes**

```bash
go test -run TestCatalogHandler_ListCoupons ./internal/handler/store/ -v
```

Expected: both PASS.

- [ ] **Step 6: Update Wire and regenerate**

In `handloom-admin/internal/wire/providers.go`, change `ProvideStoreCatalogHandler`:

```go
func ProvideStoreCatalogHandler(
	productService *service.ProductService,
	categoryService *service.CategoryService,
	inventoryService *service.InventoryService,
	couponService *service.CouponService,
) *store.CatalogHandler {
	return store.NewCatalogHandler(productService, categoryService, inventoryService, couponService)
}
```

In `handloom-admin/internal/wire/wire.go`, inside `InitializeStoreCatalogDeps`'s `wire.Build`, add two providers next to the existing repository lines:

```go
		ProvideCouponRepository,
		ProvideCouponService,
```

Then:

```bash
cd handloom-admin && make wire && go build ./...
```

Expected: `wire_gen.go` regenerates with no error. If Wire reports a missing provider, `ProvideCouponService`'s own dependencies (`internal/wire/providers.go:309`) are not in the build list — add what it asks for.

- [ ] **Step 7: Verify the monolith serves the route**

```bash
cd handloom-admin && make run
```

In a second shell:

```bash
curl -si localhost:8081/api/v1/store/catalog/coupons | head -20
```

Expected: `200`, `Cache-Control: public, max-age=3600`, and `{"success":true,"data":[...]}`. An empty `data` array is correct on a fresh local DB.

- [ ] **Step 8: Commit**

```bash
cd handloom-admin && golangci-lint run internal/...
git add internal/handler/store/catalog_coupons.go internal/handler/store/catalog_coupons_test.go \
        internal/handler/store/catalog_handler.go internal/wire/
git commit -m "feat(store): serve the public offers list from the catalog Lambda"
```

---

### Task 5: Checkout coupon picker endpoint

**Files:**
- Modify: `handloom-admin/internal/domain/store_service.go:38-45`
- Modify: `handloom-admin/internal/service/checkout_service.go:261-272`
- Test: `handloom-admin/internal/service/checkout_service_test.go`
- Modify: `handloom-admin/internal/handler/store/checkout_handler.go:30-38`
- Test: `handloom-admin/internal/handler/store/checkout_handler_test.go`

**Interfaces:**
- Consumes: `CouponService.ListForCart` (Task 3), `toStoreCoupon` (Task 4), `cartService.GetCart` as used by `PreviewCoupon`, `middleware.GetCustomerIDFromContext`
- Produces: `CheckoutService.ListCoupons(ctx, customerID string) ([]*domain.CouponOffer, error)` and `GET /api/v1/store/checkout/coupons` returning `[]StoreCouponOffer`. Consumed by Task 7.

- [ ] **Step 1: Write the failing service test**

Append to `handloom-admin/internal/service/checkout_service_test.go`. The file has no shared constructor helper — every test calls `NewCheckoutService` inline with the same six arguments (see `checkout_service_test.go:192`), so this does too. `ListCoupons` touches only the cart and coupon services, so the other four are unused mocks:

```go
// The picker prices against the server's cart, never a figure from a browser.
func TestCheckoutService_ListCoupons(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	cartSvc := mocks.NewMockCartService(ctrl)
	cartSvc.EXPECT().GetCart(gomock.Any(), "cust_1", false).Return(&domain.CartResult{
		Cart: &domain.Cart{Subtotal: 250000},
	}, nil)

	couponSvc := mocks.NewMockCouponService(ctrl)
	couponSvc.EXPECT().ListForCart(gomock.Any(), domain.CouponContext{
		CartTotal:         250000,
		CustomerID:        "cust_1",
		HasAutomaticOffer: false,
	}).Return([]*domain.CouponOffer{
		{Coupon: &domain.Coupon{Code: "FESTIVE20"}, Eligible: true, DiscountAmount: 50000},
	}, nil)

	svc := NewCheckoutService(
		cartSvc,
		mocks.NewMockOrderRepository(ctrl),
		mocks.NewMockPaymentService(ctrl),
		mocks.NewMockInventoryRepository(ctrl),
		mocks.NewMockCustomerRepository(ctrl),
		couponSvc,
	)

	offers, err := svc.ListCoupons(ctx, "cust_1")
	require.NoError(t, err)
	require.Len(t, offers, 1)
	require.Equal(t, int64(50000), offers[0].DiscountAmount)
}
```

If any of those four mock constructor names differ, copy them verbatim from the `NewCheckoutService` call at `checkout_service_test.go:192` — the argument order there is authoritative.

- [ ] **Step 2: Run it to verify it fails**

```bash
cd handloom-admin && go test -run TestCheckoutService_ListCoupons ./internal/service/ -v
```

Expected: compile failure — `s.ListCoupons undefined`.

- [ ] **Step 3: Add the interface method**

In `handloom-admin/internal/domain/store_service.go`, inside `CheckoutService`, after `PreviewCoupon`:

```go
	// ListCoupons prices every public coupon against the customer's current cart, so
	// the picker can show a real saving or a real reason on each.
	ListCoupons(ctx context.Context, customerID string) ([]*CouponOffer, error)
```

- [ ] **Step 4: Implement it**

In `handloom-admin/internal/service/checkout_service.go`, directly after `PreviewCoupon`:

```go
// ListCoupons prices the public coupons against this customer's cart. Same cart source
// as PreviewCoupon, so the picker and a typed code agree.
func (s *CheckoutService) ListCoupons(
	ctx context.Context, customerID string,
) ([]*domain.CouponOffer, error) {
	cart, err := s.cartService.GetCart(ctx, customerID, false)
	if err != nil {
		return nil, err
	}

	return s.couponService.ListForCart(ctx, domain.CouponContext{
		CartTotal:         cart.Cart.Subtotal,
		CustomerID:        customerID,
		HasAutomaticOffer: false, // Phase 4 sets this
	})
}
```

- [ ] **Step 5: Run it to verify it passes**

```bash
go test -run TestCheckoutService_ListCoupons ./internal/service/ -v
```

Expected: PASS.

- [ ] **Step 6: Write the failing handler test**

Append to `handloom-admin/internal/handler/store/checkout_handler_test.go` (create the file with `package store` and the imports from `catalog_coupons_test.go` if it does not exist). The handler reads the customer id via `middleware.GetCustomerIDFromContext`, which looks up `middleware.CustomerIDKey` — an exported `ContextKey` (`internal/middleware/interfaces.go:58`), so the test seeds it with a plain `context.WithValue`:

```go
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
	req := httptest.NewRequest(http.MethodGet, "/coupons", nil)
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
```

This needs `"context"` and `"github.com/handloom/admin/internal/middleware"` in the test file's imports.

- [ ] **Step 7: Run it to verify it fails**

```bash
go test -run TestCheckoutHandler_ListCoupons ./internal/handler/store/ -v
```

Expected: `404` from the router, because the route does not exist.

- [ ] **Step 8: Add the route, DTO and handler**

In `handloom-admin/internal/handler/store/checkout_handler.go`, inside `Routes()` after the `validate-coupon` line:

```go
	r.Get("/coupons", h.ListCoupons)
```

And at the end of the file:

```go
// StoreCouponOffer is one coupon priced against the customer's cart. Embeds the same
// trimmed DTO the banner uses, so neither surface can leak what the other withholds.
type StoreCouponOffer struct {
	StoreCoupon
	Eligible       bool   `json:"eligible"`
	DiscountAmount int64  `json:"discount_amount"`
	Reason         string `json:"reason,omitempty"`
}

// ListCoupons handles GET /api/v1/store/checkout/coupons — the coupon picker.
// Per customer and per cart, so it is never cached.
func (h *CheckoutHandler) ListCoupons(w http.ResponseWriter, r *http.Request) {
	customerID := middleware.GetCustomerIDFromContext(r.Context())

	w.Header().Set("Cache-Control", "no-store")

	offers, err := h.checkoutService.ListCoupons(r.Context(), customerID)
	if err != nil {
		// The customer can still type a code, and Validate is the authority.
		slog.WarnContext(r.Context(), "Coupon picker unavailable", "error", err)
		offers = nil
	}

	out := make([]StoreCouponOffer, 0, len(offers))
	for _, o := range offers {
		out = append(out, StoreCouponOffer{
			StoreCoupon:    toStoreCoupon(o.Coupon),
			Eligible:       o.Eligible,
			DiscountAmount: o.DiscountAmount,
			Reason:         o.Reason,
		})
	}

	response.Success(w, out)
}
```

Add `"log/slog"` to the import block.

- [ ] **Step 9: Run the tests to verify they pass**

```bash
cd handloom-admin && make generate-mocks
go test -run 'TestCheckoutHandler_ListCoupons|TestCheckoutService_ListCoupons' ./internal/handler/store/ ./internal/service/ -v
go test -count=1 ./internal/...
```

Expected: all PASS.

- [ ] **Step 10: Commit**

```bash
cd handloom-admin && go build ./... && golangci-lint run internal/...
git add internal/domain/store_service.go internal/service/checkout_service.go \
        internal/service/checkout_service_test.go internal/handler/store/checkout_handler.go \
        internal/handler/store/checkout_handler_test.go
git commit -m "feat(store): price every public coupon against the cart at checkout"
```

---

### Task 6: Storefront — offers banner on the homepage

**Files:**
- Modify: `homechrome-store/src/types/index.ts:203` (after `CouponValidationResult`)
- Modify: `homechrome-store/src/lib/routes.ts:28-36`
- Create: `homechrome-store/src/components/catalog/OffersBanner.tsx`
- Modify: `homechrome-store/src/app/page.tsx`
- Modify: `homechrome-store/src/app/HomeView.tsx:24-33`

**Interfaces:**
- Consumes: `GET /api/v1/store/catalog/coupons` (Task 4), `formatPrice` (`src/lib/utils.ts:16`), `API_BASE` (`src/lib/constants`)
- Produces: `PublicCoupon` type, consumed by Task 7

There is no test runner in `homechrome-store` (no vitest, no jest, no config), so this task's gate is `npm run lint`, `npx tsc --noEmit`, and looking at the page.

- [ ] **Step 1: Add the types**

In `homechrome-store/src/types/index.ts`, after `CouponValidationResult`:

```ts
// A coupon the store advertises. Cached and served to every visitor, so it carries
// no usage counters and no customer id — see StoreCoupon on the backend.
export interface PublicCoupon {
  code: string;
  name: string;
  description?: string;
  type: 'PERCENTAGE' | 'FIXED';
  value: number; // percentage × 100, or paise for FIXED
  min_order_value: number; // paise
  max_discount?: number; // paise
  valid_until?: string;
}

// A public coupon priced against the current cart. reason explains a false eligible.
export interface CouponOffer extends PublicCoupon {
  eligible: boolean;
  discount_amount: number; // paise
  reason?: string;
}
```

- [ ] **Step 2: Add the route constants**

In `homechrome-store/src/lib/routes.ts`, add to the `CATALOG` block:

```ts
    COUPONS: '/api/v1/store/catalog/coupons',
```

and to the `CHECKOUT` block:

```ts
    COUPONS: '/api/v1/store/checkout/coupons',
```

- [ ] **Step 3: Create the banner component**

Create `homechrome-store/src/components/catalog/OffersBanner.tsx`:

```tsx
import { Box, Container, Group, Text } from '@mantine/core';

import { PublicCoupon } from '@/types';

import { formatPrice } from '@/lib/utils';

// "20% off" or "₹500 off", plus the minimum when there is one.
function offerLabel(coupon: PublicCoupon): string {
  const off =
    coupon.type === 'PERCENTAGE'
      ? `${coupon.value / 100}% off`
      : `${formatPrice(coupon.value)} off`;
  return coupon.min_order_value > 0
    ? `${off} above ${formatPrice(coupon.min_order_value)}`
    : off;
}

interface OffersBannerProps {
  coupons: PublicCoupon[];
}

export default function OffersBanner({ coupons }: OffersBannerProps) {
  if (coupons.length === 0) return null;

  return (
    <Box component="section" bg="teal.7" py="xs">
      <Container size="xl">
        <Group justify="center" gap="lg" wrap="wrap">
          {coupons.map((coupon) => (
            <Text key={coupon.code} c="white" fz="sm" fw={500}>
              {offerLabel(coupon)} with{' '}
              <Text span fw={700}>
                {coupon.code}
              </Text>
            </Text>
          ))}
        </Group>
      </Container>
    </Box>
  );
}
```

- [ ] **Step 4: Fetch on the homepage**

In `homechrome-store/src/app/page.tsx`, add the import and fetch, following the two existing ones exactly:

```tsx
async function getPublicCoupons(): Promise<PublicCoupon[]> {
  try {
    const res = await fetch(`${API_BASE}${ROUTES.CATALOG.COUPONS}`, {
      next: { revalidate: 3600 },
    });
    if (!res.ok) return [];
    const json = await res.json();
    return json.data || [];
  } catch {
    return [];
  }
}
```

Add `PublicCoupon` to the `@/types` import, then extend the page body:

```tsx
export default async function HomePage() {
  const [categories, products, coupons] = await Promise.all([
    getCategories(),
    getFeaturedProducts(),
    getPublicCoupons(),
  ]);

  return <HomeView categories={categories} products={products} coupons={coupons} />;
}
```

- [ ] **Step 5: Render it**

In `homechrome-store/src/app/HomeView.tsx`, extend the props and mount the banner above the hero `Box`:

```tsx
interface HomeViewProps {
  categories: Category[];
  products: Product[];
  coupons: PublicCoupon[];
}

export default function HomeView({ categories, products, coupons }: HomeViewProps) {
  return (
    <>
      <HomePageTracker />

      <OffersBanner coupons={coupons} />
```

Add both imports: `OffersBanner` from `@/components/catalog/OffersBanner`, and `PublicCoupon` alongside `Category, Product` from `@/types`.

- [ ] **Step 6: Verify**

```bash
cd homechrome-store && npm run lint && npx tsc --noEmit
```

Expected: lint clean; `tsc` reports **only** the 3 pre-existing `.webp` errors recorded in #275 and nothing new.

Then, with the backend running (`cd handloom-admin && make run`):

```bash
cd homechrome-store && npm run dev
```

Create an `ACTIVE`, `audience = ALL` coupon in the admin dashboard, then load `localhost:3000`. Expected: a teal strip above the hero naming the code. Delete the coupon, wait out the revalidate window or restart `npm run dev`, and confirm the strip disappears.

- [ ] **Step 7: Commit**

```bash
cd homechrome-store
git add src/types/index.ts src/lib/routes.ts src/components/catalog/OffersBanner.tsx \
        src/app/page.tsx src/app/HomeView.tsx
git commit -m "feat(store): advertise the running offers on the homepage"
```

---

### Task 7: Storefront — coupon picker at checkout

**Files:**
- Create: `homechrome-store/src/components/checkout/CouponPicker.tsx`
- Modify: `homechrome-store/src/app/checkout/ReviewStep.tsx`

**Interfaces:**
- Consumes: `GET /api/v1/store/checkout/coupons` (Task 5), `CouponOffer` type (Task 6), `api` (`src/lib/api`), `ROUTES.CHECKOUT.COUPONS`, `formatPrice`, and the existing `apply` callback from `useCoupon` (`src/hooks/useCoupon.ts`)
- Produces: nothing downstream

- [ ] **Step 1: Create the picker**

Create `homechrome-store/src/components/checkout/CouponPicker.tsx`:

```tsx
'use client';

import { Button, Menu, Stack, Text } from '@mantine/core';
import { useCallback, useEffect, useState } from 'react';

import api from '@/lib/api';
import { ROUTES } from '@/lib/routes';
import { formatPrice } from '@/lib/utils';
import { CouponOffer } from '@/types';

interface CouponPickerProps {
  // Refetched whenever this moves: every saving below is a function of the subtotal.
  subtotal: number;
  onApply: (code: string, discount: number) => void;
}

export default function CouponPicker({ subtotal, onApply }: CouponPickerProps) {
  const [offers, setOffers] = useState<CouponOffer[]>([]);

  useEffect(() => {
    let current = true;
    api
      .get<CouponOffer[]>(ROUTES.CHECKOUT.COUPONS)
      .then(({ data }) => {
        if (current) setOffers(data || []);
      })
      .catch(() => {
        // Not being able to list is not the same as there being none. The code input
        // beside this still works, and Validate is the authority.
      });
    return () => {
      current = false;
    };
  }, [subtotal]);

  const apply = useCallback(
    (offer: CouponOffer) => onApply(offer.code, offer.discount_amount),
    [onApply],
  );

  if (offers.length === 0) return null;

  return (
    <Menu position="bottom-start" withinPortal>
      <Menu.Target>
        <Button variant="subtle" size="compact-sm">
          View available offers ({offers.filter((o) => o.eligible).length})
        </Button>
      </Menu.Target>
      <Menu.Dropdown>
        {offers.map((offer) => (
          <Menu.Item
            key={offer.code}
            disabled={!offer.eligible}
            onClick={() => apply(offer)}
          >
            <Stack gap={2}>
              <Text fz="sm" fw={600}>
                {offer.code}
                {offer.eligible ? ` — saves ${formatPrice(offer.discount_amount)}` : ''}
              </Text>
              <Text fz="xs" c="dimmed">
                {offer.eligible ? offer.name : offer.reason}
              </Text>
            </Stack>
          </Menu.Item>
        ))}
      </Menu.Dropdown>
    </Menu>
  );
}
```

- [ ] **Step 2: Mount it**

`ReviewStep` has `couponCode`, `couponDiscount`, `onCouponApplied` and `onCouponRemoved` but no subtotal, so add one prop. In `homechrome-store/src/app/checkout/ReviewStep.tsx`, extend `ReviewStepProps`:

```tsx
  subtotal: number;
```

Destructure it alongside the existing props, and render the picker inside the existing Coupon `Stack` (`ReviewStep.tsx:61-69`), directly under `<CouponInput />`:

```tsx
          <CouponPicker subtotal={subtotal} onApply={onCouponApplied} />
```

Add `import CouponPicker from '@/components/checkout/CouponPicker';` beside the `CouponInput` import.

Then in `homechrome-store/src/app/checkout/page.tsx`, pass it to `<ReviewStep` (line 81) using the same value `OrderSummary` already receives at line 100:

```tsx
              subtotal={cart.cart.subtotal}
```

`onCouponApplied` already has the `(code: string, discount: number) => void` shape `CouponInput` calls it with, which is exactly `CouponPicker`'s `onApply` — no adapter needed.

- [ ] **Step 3: Verify**

```bash
cd homechrome-store && npm run lint && npx tsc --noEmit
```

Expected: clean, apart from the 3 known `.webp` errors.

With backend and storefront running, log in, add items, open checkout. Expected: an offers button listing each public coupon — eligible ones showing the saving and applying on click, ineligible ones greyed with their reason. Change the cart quantity and confirm the savings recompute.

- [ ] **Step 4: Commit**

```bash
cd homechrome-store
git add src/components/checkout/CouponPicker.tsx src/app/checkout/ReviewStep.tsx
git commit -m "feat(store): let customers pick a coupon instead of guessing a code"
```

---

### Task 8: E2E coverage

**Files:**
- Create: `e2e/specs/coupons/public-offers.spec.ts`

**Interfaces:**
- Consumes: the coupon helpers `e2e/helpers/coupon.ts` and API fixture `e2e/fixtures/api.ts` created by #285; `GET /api/v1/store/catalog/coupons` (Task 4) and `GET /api/v1/store/checkout/coupons` (Task 5)
- Produces: nothing downstream

**Read `e2e/specs/coupons/validation.spec.ts` first.** It establishes how a coupon is created, how the storefront session is authenticated, and how a cart is seeded. Reuse those helpers; do not re-implement them.

`e2e/playwright.config.ts` enumerates spec directories in `testMatch`. #285 adds `coupons` to it. Confirm with `npx playwright test --list | wc -l` before and after adding this file — the count must rise by the number of tests here. A spec collected by nothing passes silently.

- [ ] **Step 1: Write the specs**

Create `e2e/specs/coupons/public-offers.spec.ts`:

```ts
import { APIRequestContext, expect, test } from '@playwright/test';

import { adminClient } from '../../fixtures/api';
import { destroyCatalog, seedCatalog, SeededCatalog } from '../../fixtures/catalog';
import { Coupon, createCoupon, deleteCoupon } from '../../helpers/coupon';
import { customerClient, prepareCheckout } from '../../helpers/order';

/**
 * The two customer-facing coupon lists, over real HTTP. The service layer proves
 * each branch against mocks; nothing proved the endpoints reach them, that the
 * public payload withholds what it must, or that the picker prices against the
 * server's cart. No payment, so nothing costs anything at the gateway.
 */
test.describe('coupon offers', () => {
  let admin: APIRequestContext;
  let store: APIRequestContext;
  let catalog: SeededCatalog | undefined;
  let coupon: Coupon | undefined;

  // One ₹300 cart for the file. Both endpoints read the cart and never write.
  const unitPrice = 30000;

  test.beforeAll(async () => {
    admin = await adminClient();
    store = await customerClient();
    catalog = await seedCatalog(admin, [50]);
    await prepareCheckout(store, [{ productId: catalog.products[0]!.id, quantity: 1 }]);
  });

  test.afterEach(async () => {
    await deleteCoupon(admin, coupon);
    coupon = undefined;
  });

  test.afterAll(async () => {
    await destroyCatalog(admin, catalog);
  });

  test('an ACTIVE public coupon is advertised, without its internals', async () => {
    coupon = await createCoupon(admin, { value: 1000 });

    const res = await store.get('/api/v1/store/catalog/coupons');
    expect(res.ok()).toBeTruthy();

    const { data } = await res.json();
    const advertised = data.find((c: { code: string }) => c.code === coupon!.code);
    expect(advertised, 'a running ACTIVE/ALL coupon must appear').toBeTruthy();
    expect(advertised.value).toBe(1000);

    // The payload reaches every visitor, so what it omits is a guarantee.
    for (const withheld of [
      'id',
      'usage_count',
      'usage_limit',
      'usage_per_user',
      'customer_id',
      'status',
      'audience',
    ]) {
      expect(advertised, `${withheld} must never reach a customer`).not.toHaveProperty(
        withheld,
      );
    }
  });

  test('an INACTIVE coupon is not advertised', async () => {
    coupon = await createCoupon(admin, { value: 1000 });

    const patched = await admin.patch(`/admin/coupons/${coupon.id}`, {
      data: { status: 'INACTIVE' },
    });
    expect(patched.ok()).toBeTruthy();

    const res = await store.get('/api/v1/store/catalog/coupons');
    const { data } = await res.json();
    expect(data.map((c: { code: string }) => c.code)).not.toContain(coupon.code);
  });

  test('the picker reports a cart below the minimum as ineligible', async () => {
    // ₹500 minimum against the ₹300 cart above.
    coupon = await createCoupon(admin, { value: 1000, minOrderValue: 50000 });

    const res = await store.get('/api/v1/store/checkout/coupons');
    expect(res.ok()).toBeTruthy();

    const { data } = await res.json();
    const offer = data.find((o: { code: string }) => o.code === coupon!.code);
    expect(offer, 'an ineligible coupon is still listed, with its reason').toBeTruthy();
    expect(offer.eligible).toBe(false);
    expect(offer.discount_amount).toBe(0);
    expect(offer.reason).toContain('more to use this coupon');
  });

  test('the picker prices an eligible coupon against the server cart', async () => {
    // ₹100 minimum, comfortably under the ₹300 cart.
    coupon = await createCoupon(admin, { value: 1000, minOrderValue: 10000 });

    const res = await store.get('/api/v1/store/checkout/coupons');
    const { data } = await res.json();
    const offer = data.find((o: { code: string }) => o.code === coupon!.code);

    expect(offer.eligible).toBe(true);
    // 10% of the ₹300 cart, computed by the server — never by the browser.
    expect(offer.discount_amount).toBe(unitPrice / 10);
  });
});
```

`createCoupon` already generates a per-run code (`helpers/coupon.ts:37`), so two concurrent runs against shared dev cannot fight over one usage limit. Do not hardcode a code.

- [ ] **Step 2: Confirm the specs are collected**

```bash
cd e2e && npx playwright test --list | tail -5
```

Expected: the four new tests appear by name.

- [ ] **Step 3: Run them against dev**

```bash
cd e2e && npx playwright test specs/coupons/public-offers.spec.ts
```

Expected: 4 passed, 0 skipped. **These need the branch deployed to dev** — dispatch `Deploy Backend` with `environment: dev` from this branch first, per the deploy flow in the root `CLAUDE.md`.

- [ ] **Step 4: Commit**

```bash
cd e2e && git add specs/coupons/public-offers.spec.ts
git commit -m "test(e2e): cover the public offers list and the checkout picker"
```

---

## Final verification

- [ ] `cd handloom-admin && go build ./... && go vet ./... && golangci-lint run internal/...` — clean
- [ ] `cd handloom-admin && make docker-up && CI=1 go test -count=1 -race ./internal/repository/dynamodb/` — passes, and the coupon tests genuinely run rather than skip
- [ ] `cd handloom-admin && go test -count=1 ./internal/...` — passes. Postgres-backed tests **skip silently** without `POSTGRES_DSN`; "green" excludes them
- [ ] `cd handloom-admin && make wire && git diff --exit-code internal/wire/wire_gen.go` — no drift
- [ ] `cd handloom-admin && make generate-mocks && go build ./... && go test -count=1 ./internal/...` — mocks are gitignored, so the check is that a fresh regeneration still compiles and passes, not that the tree is unchanged
- [ ] `cd homechrome-store && npm run lint && npx tsc --noEmit` — clean apart from the 3 known `.webp` errors
- [ ] `cd handloom-admin && make cdk-diff-dev` — no infrastructure change. This plan adds no Lambda, no table, no index, and no API Gateway resource; a diff here means something went wrong
- [ ] Manual: an `ACTIVE`/`ALL` coupon appears on the homepage banner and in the checkout picker; a `FIRST_ORDER` coupon appears in **neither**
- [ ] `git log --oneline feat/coupons-phase-1..HEAD` — one commit per task, no fixups
