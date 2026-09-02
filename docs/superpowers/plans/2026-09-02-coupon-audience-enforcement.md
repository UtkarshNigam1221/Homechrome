# Coupon Audience Enforcement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `Coupon.Audience` actually restrict who can redeem a coupon, and let an operator create targeted coupons from the admin form by entering a customer's phone number.

**Architecture:** `evaluate` — the shared eligibility rule `Validate` and `ListForCart` both call — gains an audience branch and an `orderCount *int` parameter where `nil` means "not resolved" and targeted audiences reject on it. `CouponService` takes a `CustomerRepository` and resolves `Customer.OrderCount` only when the audience needs it, so an `ALL` coupon costs no extra read. The same dependency resolves a phone number to a customer id when a `SPECIFIC_CUSTOMER` coupon is created.

**Tech Stack:** Go 1.25, Chi, DynamoDB (aws-sdk-go-v2), Google Wire, gomock/testify. Admin frontend: React 19, TypeScript, react-hook-form + zod, Vitest.

**Spec:** `docs/superpowers/specs/2026-09-02-coupon-audience-enforcement-design.md`

## Global Constraints

- **Branch:** `feat/coupon-audience-enforcement`, already created off `main` and carrying the spec commit. Do not switch branches.
- **Money is paise**, `int64`. `Coupon.Value` is percentage × 100 for `PERCENTAGE`, paise for `FIXED`.
- **`SPECIFIC_CUSTOMER` and unresolved-audience rejections must return the SAME customer-facing message as an unknown code**, byte for byte. A distinct message turns the refusal into an enumeration oracle. This plan makes that string a shared constant so the two cannot drift.
- **`FIRST_ORDER` and `RETURNING` rejections DO name their reason.** Exact strings: `"This code is for first orders only"` and `"This code is for returning customers"`.
- **Fail closed.** An unresolved order count, a failed customer read, or an empty `CustomerID` on a targeted coupon all reject. Never fall through to a permissive default.
- **Phones are E.164 on the wire and in storage** — `+91` plus ten digits. `GetByPhone` is an exact-match `GetItem` with no normalisation of its own.
- **Comments cap at TWO lines.** Repo convention. Do not assume this plan's own comments comply — count them, trim if needed, and say so in your report.
- **Go: import ordering** via goimports, local prefix `github.com/handloom/admin`. `golangci-lint run internal/...` must report 0 issues. Thresholds gocognit=30, gocyclo=25, dupl=200 — `evaluate` grows in Task 1, so check its complexity actually clears rather than assuming.
- **Use `httptest.NewRequestWithContext`, never `httptest.NewRequest`** — the `noctx` linter rejects the latter.
- **Never `git add internal/mocks/`.** `.gitignore:77` ignores `internal/mocks/*_mock.go` and `ci.yml` regenerates them. Run `make generate-mocks` so local builds compile, but do not stage those files. `internal/wire/wire_gen.go` IS tracked and must be committed when `make wire` changes it.
- **Run Go tests with `-count=1`.** A cached `ok` is not evidence. Repository tests additionally need `CI=1` and DynamoDB Local; this plan adds no repository tests.
- `internal/repository/postgres` fails locally with `postgres not available` on every case. Pre-existing and environmental — ignore it.
- **Admin frontend gate:** `npm run check` (typecheck + lint + format:check) and `npm run test`. Both must pass.
- **No infrastructure changes.** Nothing under any `infra/` directory.

---

## File Structure

**Backend — `handloom-admin/`**

| file | responsibility |
|---|---|
| `internal/service/coupon_service.go` | Modify: the audience rule in `evaluate`, order-count resolution in `Validate`, phone resolution in `Create`, the `normalizePhone` helper, two new message/label constants |
| `internal/service/coupon_service_test.go` | Modify: audience cases, zero-read assertions, phone normalisation cases |
| `internal/domain/coupon.go` | Modify: `CreateCouponRequest.CustomerID` → `CustomerPhone` |
| `internal/wire/providers.go` | Modify: `ProvideCouponService` gains the customer repository |
| `internal/wire/wire.go` | Modify: two build lists gain `ProvideCustomerRepository` |

**Admin frontend — `handloom-admin-frontend/`**

| file | responsibility |
|---|---|
| `src/features/coupons/types.ts` | Modify: `CreateCouponRequest.customer_id` → `customer_phone` |
| `src/features/coupons/lib/couponSchema.ts` | Modify: phone field and its conditional requirement; delete the stale trailing comment |
| `src/features/coupons/lib/toCreateRequest.ts` | Modify: `customerPhone` form field, sent as `customer_phone` |
| `src/features/coupons/lib/__tests__/couponSchema.test.ts` | Modify: phone validation cases |
| `src/features/coupons/lib/__tests__/toCreateRequest.test.ts` | Modify: mapping cases |
| `src/features/coupons/components/CouponFormModal.tsx` | Modify: enable the three options, drop the "(Phase 3)" copy, render the phone input conditionally |

---

### Task 1: The audience rule in `evaluate`

**Files:**
- Modify: `handloom-admin/internal/service/coupon_service.go`
- Test: `handloom-admin/internal/service/coupon_service_test.go`

**Interfaces:**
- Consumes: `domain.CouponAudience` constants `AudienceAll`, `AudienceFirstOrder`, `AudienceReturning`, `AudienceSpecificCustomer` (`internal/domain/coupon.go:34-37`); `domain.CouponContext{CartTotal, CustomerID, HasAutomaticOffer}`; the existing `reject` closure inside `evaluate`
- Produces: `evaluate(c *domain.Coupon, cc domain.CouponContext, used int, orderCount *int) *domain.CouponValidationResult`, plus package constants `msgCodeInvalid` and `outcomeAudience` — all consumed by Task 2

**State after this task:** `Validate` and `ListForCart` pass `nil`, so `FIRST_ORDER` and `RETURNING` coupons reject for everyone and `SPECIFIC_CUSTOMER` works only on an exact id match. That is strictly *more* restrictive than today, never less, and no targeted coupon can exist yet — the admin form has disabled those options since Phase 1. Task 2 makes them work properly.

- [ ] **Step 1: Write the failing tests**

Append to `handloom-admin/internal/service/coupon_service_test.go`:

```go
// audienceCoupon is an otherwise-valid coupon targeted at one audience.
func audienceCoupon(a domain.CouponAudience) *domain.Coupon {
	c := activeCoupon()
	c.Audience = a
	return c
}

func intPtr(n int) *int { return &n }

// The rule, exercised directly rather than through Validate, so no repository
// behaviour can mask a branch.
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

		unknown := &domain.CouponValidationResult{
			Valid: false, Code: "WHATEVER", Outcome: outcomeInvalid,
			ErrorMessage: msgCodeInvalid,
		}
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
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd handloom-admin && go test -count=1 -run TestEvaluate_Audience ./internal/service/ -v
```

Expected: compile failure — `evaluate` takes 3 arguments not 4, and `msgCodeInvalid` / `outcomeAudience` are undefined.

- [ ] **Step 3: Add the two constants and share the invalid-code message**

In `handloom-admin/internal/service/coupon_service.go`, directly below the existing `const outcomeInvalid = "invalid"` (line 21):

```go
// outcomeAudience labels a rejection the customer is not always told the reason for,
// so the funnel can still see it. Outcome is json:"-" and never reaches a customer.
const outcomeAudience = "audience"

// msgCodeInvalid is shared by the unknown-code rejection and every rejection that must
// not confirm a code exists. One constant, so the two cannot drift apart.
const msgCodeInvalid = "That code isn't valid"
```

Then replace the inline literal in `Validate` (line 212) so it uses the constant:

```go
	coupon, err := s.couponRepo.GetByCode(ctx, code)
	if err != nil || coupon == nil {
		return reject(outcomeInvalid, msgCodeInvalid)
	}
```

- [ ] **Step 4: Add the audience branch to `evaluate`**

Change the signature and insert the branch. In `handloom-admin/internal/service/coupon_service.go`, `evaluate` currently begins at line 231. The new signature:

```go
func evaluate(
	coupon *domain.Coupon,
	cc domain.CouponContext,
	used int,
	orderCount *int,
) *domain.CouponValidationResult {
```

Insert this block immediately **after** the `ValidUntil` expiry check and immediately **before** the `cc.CartTotal < coupon.MinOrderValue` check:

```go
	// A nil orderCount means it was not resolved, so a targeted audience refuses rather
	// than assuming — zero would read as "first order", the permissive case.
	switch coupon.Audience {
	case domain.AudienceFirstOrder:
		if orderCount == nil {
			return reject(outcomeAudience, msgCodeInvalid)
		}
		if *orderCount != 0 {
			return reject(outcomeAudience, "This code is for first orders only")
		}
	case domain.AudienceReturning:
		if orderCount == nil {
			return reject(outcomeAudience, msgCodeInvalid)
		}
		if *orderCount < 1 {
			return reject(outcomeAudience, "This code is for returning customers")
		}
	case domain.AudienceSpecificCustomer:
		// Same message as an unknown code: a distinct one would confirm this code is real.
		if cc.CustomerID == "" || coupon.CustomerID != cc.CustomerID {
			return reject(outcomeAudience, msgCodeInvalid)
		}
	}
```

`domain.AudienceAll` is deliberately absent from the switch — it has no rule, and adding an empty case would invite someone to put one there.

- [ ] **Step 5: Update the two existing call sites to compile**

In `Validate`, pass `nil` for now (Task 2 resolves it):

```go
	result := evaluate(coupon, cc, used, nil)
```

In `ListForCart` (around line 431), pass `nil` permanently:

```go
		// nil, and it stays nil: ListPublic returns only audience=ALL, so no candidate
		// can need a count. If that ever changes, nil under-shows rather than over-promises.
		v := evaluate(c, cc, used[c.ID], nil)
```

- [ ] **Step 6: Run the tests to verify they pass**

```bash
cd handloom-admin && go test -count=1 -run TestEvaluate_Audience ./internal/service/ -v
```

Expected: every subtest PASSES.

- [ ] **Step 7: Confirm the existing suite is untouched**

```bash
go test -count=1 ./internal/service/ -v -run 'TestCouponService_Validate|TestCouponService_Redeem|TestCouponService_ListForCart|TestCouponService_ListPublic'
```

Expected: every pre-existing subtest passes **without you editing any of them**. If one fails, the branch was inserted in the wrong place or the shared constant changed a message — fix the code, never the test.

- [ ] **Step 8: Prove the branch is load-bearing**

Temporarily change `*orderCount != 0` to `*orderCount == 0` and re-run `TestEvaluate_Audience`. Expected: the "FIRST_ORDER passes on a first order" and "names its reason" subtests both fail. Restore and confirm they pass. Report what you saw.

- [ ] **Step 9: Check complexity and commit**

```bash
cd handloom-admin && go build ./... && golangci-lint run internal/...
```

Expected: 0 issues. `evaluate` gained a switch, so confirm `gocognit` (threshold 30) still passes rather than assuming.

```bash
git add internal/service/coupon_service.go internal/service/coupon_service_test.go
git commit -m "feat(coupons): make the audience decide who a code is for"
```

---

### Task 2: Resolve the order count

**Files:**
- Modify: `handloom-admin/internal/service/coupon_service.go`
- Modify: `handloom-admin/internal/wire/providers.go:309-313`
- Modify: `handloom-admin/internal/wire/wire.go`
- Test: `handloom-admin/internal/service/coupon_service_test.go`

**Interfaces:**
- Consumes: `evaluate(..., orderCount *int)`, `msgCodeInvalid`, `outcomeAudience` from Task 1; `domain.CustomerRepository.GetByID(ctx, id) (*domain.Customer, error)` (`internal/domain/order_repository.go:66`); `domain.Customer.OrderCount int` (`internal/domain/order.go:204`); `mocks.NewMockCustomerRepository` (already generated, `internal/mocks/order_repository_mock.go:201`)
- Produces: `NewCouponService(couponRepo domain.CouponRepository, customerRepo domain.CustomerRepository) *CouponService` — consumed by Task 3 and by Wire

- [ ] **Step 1: Write the failing tests**

Append to `handloom-admin/internal/service/coupon_service_test.go`:

```go
// setupAudienceService wires a coupon and an optional customer into a real service.
// customer == nil means GetByID fails, which must reject rather than grant.
func setupAudienceService(
	t *testing.T, c *domain.Coupon, customer *domain.Customer,
) (*CouponService, *mocks.MockCustomerRepository) {
	t.Helper()
	ctrl := gomock.NewController(t)

	couponRepo := mocks.NewMockCouponRepository(ctrl)
	couponRepo.EXPECT().GetByCode(gomock.Any(), gomock.Any()).Return(c, nil).AnyTimes()
	couponRepo.EXPECT().GetCustomerUsage(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(0, nil).AnyTimes()

	customerRepo := mocks.NewMockCustomerRepository(ctrl)
	if customer != nil {
		customerRepo.EXPECT().GetByID(gomock.Any(), "cust_1").Return(customer, nil).AnyTimes()
	} else {
		customerRepo.EXPECT().GetByID(gomock.Any(), gomock.Any()).
			Return(nil, errors.Internal("customer store is down")).AnyTimes()
	}

	return NewCouponService(couponRepo, customerRepo), customerRepo
}

func TestCouponService_Validate_Audience(t *testing.T) {
	ctx := context.Background()
	cc := domain.CouponContext{CartTotal: 100000, CustomerID: "cust_1"}

	t.Run("FIRST_ORDER passes for a customer with no paid orders", func(t *testing.T) {
		s, _ := setupAudienceService(t,
			audienceCoupon(domain.AudienceFirstOrder),
			&domain.Customer{ID: "cust_1", OrderCount: 0})

		res, err := s.Validate(ctx, "FESTIVE20", cc)
		require.NoError(t, err)
		require.True(t, res.Valid)
	})

	t.Run("FIRST_ORDER refuses a customer who has bought before", func(t *testing.T) {
		s, _ := setupAudienceService(t,
			audienceCoupon(domain.AudienceFirstOrder),
			&domain.Customer{ID: "cust_1", OrderCount: 2})

		res, err := s.Validate(ctx, "FESTIVE20", cc)
		require.NoError(t, err)
		require.False(t, res.Valid)
		require.Equal(t, "This code is for first orders only", res.ErrorMessage)
	})

	t.Run("RETURNING refuses a first-time buyer", func(t *testing.T) {
		s, _ := setupAudienceService(t,
			audienceCoupon(domain.AudienceReturning),
			&domain.Customer{ID: "cust_1", OrderCount: 0})

		res, err := s.Validate(ctx, "FESTIVE20", cc)
		require.NoError(t, err)
		require.False(t, res.Valid)
		require.Equal(t, "This code is for returning customers", res.ErrorMessage)
	})

	// A read failure must not become a discount.
	t.Run("a failed customer read refuses the coupon", func(t *testing.T) {
		s, _ := setupAudienceService(t, audienceCoupon(domain.AudienceFirstOrder), nil)

		res, err := s.Validate(ctx, "FESTIVE20", cc)
		require.NoError(t, err, "a coupon problem is a result, not an error")
		require.False(t, res.Valid)
		require.Equal(t, msgCodeInvalid, res.ErrorMessage)
	})

	// The guard against reintroducing a per-validate customer read.
	t.Run("an ALL coupon reads no customer at all", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		couponRepo := mocks.NewMockCouponRepository(ctrl)
		couponRepo.EXPECT().GetByCode(gomock.Any(), gomock.Any()).
			Return(audienceCoupon(domain.AudienceAll), nil)
		couponRepo.EXPECT().GetCustomerUsage(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(0, nil).AnyTimes()

		customerRepo := mocks.NewMockCustomerRepository(ctrl)
		customerRepo.EXPECT().GetByID(gomock.Any(), gomock.Any()).Times(0)

		res, err := NewCouponService(couponRepo, customerRepo).Validate(ctx, "FESTIVE20", cc)
		require.NoError(t, err)
		require.True(t, res.Valid)
	})

	// SPECIFIC_CUSTOMER is an id comparison; it needs no order history.
	t.Run("SPECIFIC_CUSTOMER reads no customer either", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		c := audienceCoupon(domain.AudienceSpecificCustomer)
		c.CustomerID = "cust_1"

		couponRepo := mocks.NewMockCouponRepository(ctrl)
		couponRepo.EXPECT().GetByCode(gomock.Any(), gomock.Any()).Return(c, nil)
		couponRepo.EXPECT().GetCustomerUsage(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(0, nil).AnyTimes()

		customerRepo := mocks.NewMockCustomerRepository(ctrl)
		customerRepo.EXPECT().GetByID(gomock.Any(), gomock.Any()).Times(0)

		res, err := NewCouponService(couponRepo, customerRepo).Validate(ctx, "FESTIVE20", cc)
		require.NoError(t, err)
		require.True(t, res.Valid)
	})

	// #255 asks us to confirm this rather than assume it: RecordPurchase runs in
	// HandlePaymentSuccess, so OrderCount is still 0 while an order sits unpaid.
	t.Run("an unpaid order leaves a first-order code usable", func(t *testing.T) {
		s, _ := setupAudienceService(t,
			audienceCoupon(domain.AudienceFirstOrder),
			&domain.Customer{ID: "cust_1", OrderCount: 0})

		res, err := s.Validate(ctx, "FESTIVE20", cc)
		require.NoError(t, err)
		require.True(t, res.Valid,
			"validation happens at checkout, before RecordPurchase; this is the accepted overshoot")
	})
}

// Pricing M candidates must not cost M customer reads.
func TestCouponService_ListForCart_ReadsNoCustomer(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	couponRepo := mocks.NewMockCouponRepository(ctrl)
	couponRepo.EXPECT().ListPublic(gomock.Any(), gomock.Any()).
		Return([]*domain.Coupon{
			audienceCoupon(domain.AudienceAll),
			audienceCoupon(domain.AudienceAll),
			audienceCoupon(domain.AudienceAll),
		}, nil)
	couponRepo.EXPECT().GetCustomerUsageAll(gomock.Any(), "cust_1").
		Return(map[string]int{}, nil)

	customerRepo := mocks.NewMockCustomerRepository(ctrl)
	customerRepo.EXPECT().GetByID(gomock.Any(), gomock.Any()).Times(0)

	offers, err := NewCouponService(couponRepo, customerRepo).
		ListForCart(ctx, domain.CouponContext{CartTotal: 100000, CustomerID: "cust_1"})
	require.NoError(t, err)
	require.Len(t, offers, 3)
}
```

The `ListPublic` expectation takes two arguments deliberately: the **repository** method is `ListPublic(ctx context.Context, cutoff time.Time) ([]*Coupon, error)` (`internal/domain/coupon.go:211`), because Phase 1's final review made the cutoff caller-supplied. Do not confuse it with the **service** method of the same name, which takes only a context (`internal/domain/coupon.go:282`). The mock here is `MockCouponRepository`.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd handloom-admin && go test -count=1 -run 'TestCouponService_Validate_Audience|TestCouponService_ListForCart_ReadsNoCustomer' ./internal/service/ -v
```

Expected: compile failure — `NewCouponService` takes 1 argument, not 2.

- [ ] **Step 3: Add the dependency**

In `handloom-admin/internal/service/coupon_service.go`, replace the struct and constructor (lines 31-38):

```go
// CouponService implements domain.CouponService
type CouponService struct {
	couponRepo   domain.CouponRepository
	customerRepo domain.CustomerRepository
}

// NewCouponService creates a new CouponService. The customer repository resolves order
// history for a targeted audience, and a phone number to a customer id on create.
func NewCouponService(
	couponRepo domain.CouponRepository,
	customerRepo domain.CustomerRepository,
) *CouponService {
	return &CouponService{
		couponRepo:   couponRepo,
		customerRepo: customerRepo,
	}
}
```

- [ ] **Step 4: Resolve the count in `Validate`**

In `Validate`, between the existing `used` block and the `evaluate` call, insert:

```go
	// Resolved only when the audience actually needs it, so an ALL coupon — every coupon
	// on the banner and in the picker — costs no extra read.
	var orderCount *int
	if needsOrderCount(coupon.Audience) && cc.CustomerID != "" {
		if customer, custErr := s.customerRepo.GetByID(ctx, cc.CustomerID); custErr == nil {
			orderCount = &customer.OrderCount
		} else {
			slog.WarnContext(ctx, "Coupon audience unresolved", "error", custErr)
		}
	}

	result := evaluate(coupon, cc, used, orderCount)
```

Delete the old `result := evaluate(coupon, cc, used, nil)` line from Task 1. Add the helper beside `evaluate`:

```go
// needsOrderCount reports whether an audience is decided by order history.
// SPECIFIC_CUSTOMER is an id comparison, and ALL has no rule.
func needsOrderCount(a domain.CouponAudience) bool {
	return a == domain.AudienceFirstOrder || a == domain.AudienceReturning
}
```

On a failed read, `orderCount` stays `nil` and `evaluate` rejects. That is the fail-closed path from Task 1, reached deliberately.

- [ ] **Step 5: Run the tests to verify they pass**

```bash
go test -count=1 -run 'TestCouponService_Validate_Audience|TestCouponService_ListForCart_ReadsNoCustomer' ./internal/service/ -v
```

Expected: every subtest PASSES.

- [ ] **Step 6: Update Wire**

In `handloom-admin/internal/wire/providers.go`, replace `ProvideCouponService` (lines 309-313):

```go
func ProvideCouponService(
	couponRepo domain.CouponRepository,
	customerRepo domain.CustomerRepository,
) *service.CouponService {
	return service.NewCouponService(couponRepo, customerRepo)
}
```

`ProvideCouponService` appears in five build lists in `internal/wire/wire.go`. Three already provide `ProvideCustomerRepository` and need no change: `InitializeOrderDeps`, `InitializeStoreCheckoutDeps`, `InitializeStoreWebhooksDeps`. `InitializeMonolithDeps` composes `RepositorySet` (which contains `ProvideCustomerRepository`) and `ServiceSet` (which contains `ProvideCouponService`), so it resolves automatically.

**Add `ProvideCustomerRepository` to exactly these two build lists:**
- `InitializeCouponDeps`
- `InitializeStoreCatalogDeps`

Then:

```bash
cd handloom-admin && make generate-mocks && make wire && go build ./...
```

Expected: `wire_gen.go` regenerates with no error. If Wire reports a missing provider, add exactly what it names and report what you added — do not add providers speculatively.

- [ ] **Step 7: Run the whole suite**

```bash
go test -count=1 ./internal/... 2>&1 | grep -v "^ok" | head -20
golangci-lint run internal/...
```

Expected: only `internal/repository/postgres` failures, every one reading `postgres not available`. Lint 0 issues.

- [ ] **Step 8: Commit**

```bash
git add internal/service/coupon_service.go internal/service/coupon_service_test.go internal/wire/
git commit -m "feat(coupons): resolve order history only when the audience needs it"
```

---

### Task 3: A phone number on create

**Files:**
- Modify: `handloom-admin/internal/domain/coupon.go:305`
- Modify: `handloom-admin/internal/service/coupon_service.go`
- Test: `handloom-admin/internal/service/coupon_service_test.go`

**Interfaces:**
- Consumes: `NewCouponService(couponRepo, customerRepo)` from Task 2; `domain.CustomerRepository.GetByPhone(ctx, phone) (*domain.Customer, error)` (`internal/domain/order_repository.go:72`); `errors.Validation(message string) *AppError`
- Produces: `CreateCouponRequest.CustomerPhone string` and the unexported `normalizePhone(raw string) (string, error)` — consumed by Task 4's frontend contract

**Why server-side:** `GetByPhone` is an exact-match `GetItem` on `PK = CUSTOMER_PHONE#<phone>` (`internal/repository/dynamodb/customer_repository.go:148-155`) with no normalisation. Storage holds E.164 — the storefront sends `+91` plus ten digits from `homechrome-store/src/app/login/page.tsx`. A number pasted from a support ticket in any other shape would otherwise report "customer doesn't exist" for a customer who does.

- [ ] **Step 1: Write the failing tests**

Append to `handloom-admin/internal/service/coupon_service_test.go`:

```go
// Every shape an operator might paste has to reach the same stored E.164 string.
func TestNormalizePhone(t *testing.T) {
	for _, tc := range []struct {
		name, in, want string
	}{
		{"bare ten digits", "9876543210", "+919876543210"},
		{"spaced with country code", "+91 98765 43210", "+919876543210"},
		{"hyphenated with a leading zero", "098765-43210", "+919876543210"},
		{"country code without a plus", "919876543210", "+919876543210"},
		{"parenthesised", "(+91) 98765-43210", "+919876543210"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizePhone(tc.in)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}

	for _, tc := range []struct{ name, in string }{
		{"too short", "98765"},
		{"too long", "98765432101234"},
		{"empty", ""},
		{"letters only", "not-a-number"},
	} {
		t.Run("rejects "+tc.name, func(t *testing.T) {
			_, err := normalizePhone(tc.in)
			require.Error(t, err)
		})
	}
}

func TestCouponService_Create_SpecificCustomer(t *testing.T) {
	ctx := context.Background()

	newReq := func(phone string) domain.CreateCouponRequest {
		return domain.CreateCouponRequest{
			Code:          "APOLOGY50",
			Name:          "Apology",
			Type:          domain.CouponTypeFixed,
			Value:         50000,
			Audience:      domain.AudienceSpecificCustomer,
			CustomerPhone: phone,
			ValidFrom:     time.Now(),
		}
	}

	t.Run("stores the resolved customer id, never the phone", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		couponRepo := mocks.NewMockCouponRepository(ctrl)
		customerRepo := mocks.NewMockCustomerRepository(ctrl)

		customerRepo.EXPECT().GetByPhone(gomock.Any(), "+919876543210").
			Return(&domain.Customer{ID: "cust_42", Phone: "+919876543210"}, nil)

		var saved *domain.Coupon
		couponRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, c *domain.Coupon) error {
				saved = c
				return nil
			})

		_, err := NewCouponService(couponRepo, customerRepo).
			Create(ctx, newReq("+91 98765 43210"), "admin_1")
		require.NoError(t, err)
		require.Equal(t, "cust_42", saved.CustomerID)
		require.NotContains(t, saved.CustomerID, "+91", "the phone is not the identity")
	})

	t.Run("an unresolvable number is a validation error, not a saved coupon", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		couponRepo := mocks.NewMockCouponRepository(ctrl)
		customerRepo := mocks.NewMockCustomerRepository(ctrl)

		customerRepo.EXPECT().GetByPhone(gomock.Any(), "+919999999999").
			Return(nil, errors.NotFound("Customer not found"))
		couponRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

		_, err := NewCouponService(couponRepo, customerRepo).
			Create(ctx, newReq("9999999999"), "admin_1")
		require.Error(t, err)
	})

	t.Run("a malformed number never reaches the repository", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		couponRepo := mocks.NewMockCouponRepository(ctrl)
		customerRepo := mocks.NewMockCustomerRepository(ctrl)

		customerRepo.EXPECT().GetByPhone(gomock.Any(), gomock.Any()).Times(0)
		couponRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

		_, err := NewCouponService(couponRepo, customerRepo).
			Create(ctx, newReq("98765"), "admin_1")
		require.Error(t, err)
	})

	// An ALL coupon must not touch the customer store or require a phone.
	t.Run("a non-targeted coupon resolves nothing", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		couponRepo := mocks.NewMockCouponRepository(ctrl)
		customerRepo := mocks.NewMockCustomerRepository(ctrl)

		customerRepo.EXPECT().GetByPhone(gomock.Any(), gomock.Any()).Times(0)

		var saved *domain.Coupon
		couponRepo.EXPECT().Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, c *domain.Coupon) error {
				saved = c
				return nil
			})

		req := newReq("")
		req.Audience = domain.AudienceAll
		_, err := NewCouponService(couponRepo, customerRepo).Create(ctx, req, "admin_1")
		require.NoError(t, err)
		require.Empty(t, saved.CustomerID)
	})
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd handloom-admin && go test -count=1 -run 'TestNormalizePhone|TestCouponService_Create_SpecificCustomer' ./internal/service/ -v
```

Expected: compile failure — `normalizePhone` undefined and `CreateCouponRequest` has no `CustomerPhone` field.

- [ ] **Step 3: Change the request field**

In `handloom-admin/internal/domain/coupon.go`, replace line 305 (`CustomerID` on `CreateCouponRequest`) with:

```go
	// A phone, not an id: that is what an operator has. CouponService.Create resolves it
	// to Coupon.CustomerID before the write, so a coupon never stores a mutable number.
	CustomerPhone string `json:"customer_phone,omitempty" validate:"required_if=Audience SPECIFIC_CUSTOMER"`
```

`CustomerID` is **removed** from `CreateCouponRequest`. Do not keep both — two ways to express one thing needs a rule about which wins. `domain.Coupon.CustomerID` (line 74) does not change.

- [ ] **Step 4: Add `normalizePhone`**

At the bottom of `handloom-admin/internal/service/coupon_service.go`, beside `formatPaise`:

```go
// normalizePhone turns anything an operator might paste into the E.164 form storage
// actually holds. GetByPhone is an exact-match read, so this is the whole contract.
func normalizePhone(raw string) (string, error) {
	var digits strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	d := digits.String()

	// A pasted +91 leaves twelve digits, and a domestic 0-prefix eleven. Both have to
	// go before the length check, or a valid number reads as a missing customer.
	switch {
	case len(d) == 12 && strings.HasPrefix(d, "91"):
		d = d[2:]
	case len(d) == 11 && strings.HasPrefix(d, "0"):
		d = d[1:]
	}

	if len(d) != 10 {
		return "", errors.Validation("Enter a 10-digit Indian mobile number")
	}
	return "+91" + d, nil
}
```

`strings` is already imported in this file.

- [ ] **Step 5: Resolve in `Create`**

In `Create`, immediately after the existing `ValidUntil` date check and **before** the `coupon := &domain.Coupon{...}` literal:

```go
	// Resolved before the write, so a coupon never reaches storage bound to nothing.
	customerID := ""
	if req.Audience == domain.AudienceSpecificCustomer {
		phone, phoneErr := normalizePhone(req.CustomerPhone)
		if phoneErr != nil {
			return nil, phoneErr
		}
		customer, custErr := s.customerRepo.GetByPhone(ctx, phone)
		if custErr != nil {
			return nil, errors.Validation("No customer with that number")
		}
		customerID = customer.ID
	}
```

Then change the struct literal's `CustomerID` field from `req.CustomerID` to:

```go
		CustomerID:         customerID,
```

- [ ] **Step 6: Run the tests to verify they pass**

```bash
go test -count=1 -run 'TestNormalizePhone|TestCouponService_Create_SpecificCustomer' ./internal/service/ -v
```

Expected: every subtest PASSES.

- [ ] **Step 7: Confirm nothing else referenced the removed field**

```bash
cd handloom-admin && grep -rn "CustomerID" --include="*.go" internal/handler/ internal/service/ | grep -i "createcoupon\|req\.CustomerID"
go build ./... && go test -count=1 ./internal/... 2>&1 | grep -v "^ok" | head
golangci-lint run internal/...
```

Expected: the grep returns nothing (`req.CustomerID` is gone), the build passes, and only `postgres not available` failures remain. Lint 0 issues.

- [ ] **Step 8: Commit**

```bash
git add internal/domain/coupon.go internal/service/coupon_service.go internal/service/coupon_service_test.go
git commit -m "feat(coupons): mint a targeted coupon from a phone number"
```

---

### Task 4: Unlock the admin form

**Files:**
- Modify: `handloom-admin-frontend/src/features/coupons/types.ts`
- Modify: `handloom-admin-frontend/src/features/coupons/lib/couponSchema.ts`
- Modify: `handloom-admin-frontend/src/features/coupons/lib/toCreateRequest.ts`
- Modify: `handloom-admin-frontend/src/features/coupons/components/CouponFormModal.tsx:20-27`
- Test: `handloom-admin-frontend/src/features/coupons/lib/__tests__/couponSchema.test.ts`
- Test: `handloom-admin-frontend/src/features/coupons/lib/__tests__/toCreateRequest.test.ts`

**Interfaces:**
- Consumes: the backend contract from Task 3 — `POST /admin/coupons` now takes `customer_phone` and no longer accepts `customer_id`
- Produces: nothing downstream

**The edit-path trap, and how to avoid it.** `couponToFormValues` populates the form from a stored coupon, which carries `customer_id` and no phone. If the schema simply required `customerPhone` whenever the audience is `SPECIFIC_CUSTOMER`, **editing an existing targeted coupon would become unsavable** — the form would load with an empty phone and fail validation on a field the operator cannot meaningfully fill, for a value the update request does not even send. So `CouponFormValues` keeps **both** fields: `customerId` (read-only, populated on edit, never sent) and `customerPhone` (create-only). The requirement is conditional on `customerId` being empty, which is exactly "we are creating".

- [ ] **Step 1: Update the request type**

In `handloom-admin-frontend/src/features/coupons/types.ts`, in `CreateCouponRequest`, replace the `customer_id` field with:

```ts
  // A phone, not an id — the server resolves it to a customer. Sent only for
  // SPECIFIC_CUSTOMER, where the backend requires it.
  customer_phone?: string;
```

Leave `Coupon.customer_id` alone: the entity still carries an id, and the list still reads it.

- [ ] **Step 2: Write the failing schema tests**

The existing file already has what you need: a `const validForm: CouponFormValues` baseline at line 9 and a helper `errorsFor(overrides: Partial<CouponFormValues>): string[]` at line 28 that returns the zod messages. Use `errorsFor`; do not call `safeParse` directly, and do not add a second baseline object.

First add `customerPhone: '',` to the `validForm` baseline, beside `customerId: ''`. Then append:

```ts
describe('couponSchema: audience targeting', () => {
  it('requires a phone when creating a single-customer coupon', () => {
    expect(
      errorsFor({ audience: 'SPECIFIC_CUSTOMER', customerId: '', customerPhone: '' })
    ).toContain('Enter the customer’s 10-digit mobile number');
  });

  it('accepts a ten-digit phone', () => {
    expect(
      errorsFor({ audience: 'SPECIFIC_CUSTOMER', customerId: '', customerPhone: '9876543210' })
    ).toEqual([]);
  });

  it('accepts a phone the operator spaced out', () => {
    expect(
      errorsFor({ audience: 'SPECIFIC_CUSTOMER', customerId: '', customerPhone: '+91 98765 43210' })
    ).toEqual([]);
  });

  it('rejects a phone that is not ten digits', () => {
    expect(
      errorsFor({ audience: 'SPECIFIC_CUSTOMER', customerId: '', customerPhone: '98765' })
    ).toContain('Enter the customer’s 10-digit mobile number');
  });

  // An existing targeted coupon has an id and no phone, and its update sends neither.
  // Requiring one would make such a coupon unsavable on edit.
  it('does not demand a phone when a customer is already bound', () => {
    expect(
      errorsFor({ audience: 'SPECIFIC_CUSTOMER', customerId: 'cust_42', customerPhone: '' })
    ).toEqual([]);
  });

  it('ignores the phone entirely for the other audiences', () => {
    for (const audience of ['ALL', 'FIRST_ORDER', 'RETURNING'] as const) {
      expect(errorsFor({ audience, customerId: '', customerPhone: '' })).toEqual([]);
    }
  });
});
```

Note the apostrophe in the message is a typographic `’`, matching the string in Step 4. A straight `'` will not match.

- [ ] **Step 3: Run them to verify they fail**

```bash
cd handloom-admin-frontend && npm run test -- couponSchema
```

Expected: failures — `customerPhone` is not a field on the schema.

- [ ] **Step 4: Update the schema**

In `handloom-admin-frontend/src/features/coupons/lib/couponSchema.ts`, add the field beside `customerId`:

```ts
    customerId: z.string(),
    customerPhone: z.string(),
```

Replace the existing `SPECIFIC_CUSTOMER` refine with:

```ts
  // A phone is required only when no customer is bound yet — that is, on create. An
  // existing targeted coupon has an id and no phone, and its update sends neither.
  .refine(
    (data) =>
      data.audience !== 'SPECIFIC_CUSTOMER' ||
      data.customerId.trim().length > 0 ||
      /^\d{10}$/.test(data.customerPhone.replace(/\D/g, '')),
    {
      message: 'Enter the customer’s 10-digit mobile number',
      path: ['customerPhone'],
    }
  )
```

Delete the stale trailing comment block at the end of the file — the one beginning "Nothing enforces an audience yet". It is now false, and a false comment is worse than none.

- [ ] **Step 5: Run them to verify they pass**

```bash
npm run test -- couponSchema
```

Expected: PASS.

- [ ] **Step 6: Write the failing mapper tests**

Append to `handloom-admin-frontend/src/features/coupons/lib/__tests__/toCreateRequest.test.ts`:

```ts
describe('customer_phone mapping', () => {
  it('sends the phone for a single-customer coupon', () => {
    const req = toCreateRequest(
      validForm({ audience: 'SPECIFIC_CUSTOMER', customerPhone: '9876543210' })
    );
    expect(req.customer_phone).toBe('9876543210');
  });

  it('omits it for every other audience', () => {
    for (const audience of ['ALL', 'FIRST_ORDER', 'RETURNING'] as const) {
      const req = toCreateRequest(validForm({ audience, customerPhone: '9876543210' }));
      expect(req.customer_phone).toBeUndefined();
    }
  });

  // The server owns normalisation; the form must not send a half-cleaned string that
  // looks normalised but isn't.
  it('sends what was typed, unaltered', () => {
    const req = toCreateRequest(
      validForm({ audience: 'SPECIFIC_CUSTOMER', customerPhone: '+91 98765 43210' })
    );
    expect(req.customer_phone).toBe('+91 98765 43210');
  });
});
```

- [ ] **Step 7: Run them to verify they fail, then update the mapper**

```bash
npm run test -- toCreateRequest
```

Expected: failures — `customerPhone` is not on `CouponFormValues`.

In `toCreateRequest.ts`: add `customerPhone: string;` to `CouponFormValues` beside `customerId`, add `customerPhone: '',` to `defaultCouponFormValues`, add `customerPhone: '',` to the object `couponToFormValues` returns (a stored coupon has no phone), and replace the `customerIdFor` helper with:

```ts
// customer_phone is sent only when the audience names a customer. The server
// normalises and resolves it; sending a cleaned string would split that ownership.
function customerPhoneFor(form: CouponFormValues): string | undefined {
  return form.audience === 'SPECIFIC_CUSTOMER' ? form.customerPhone : undefined;
}
```

In `toCreateRequest`, replace `customer_id: customerIdFor(form),` with `customer_phone: customerPhoneFor(form),`.

`toUpdateRequest` does not change — it already sends neither field.

- [ ] **Step 8: Run them to verify they pass**

```bash
npm run test -- toCreateRequest
```

Expected: PASS.

- [ ] **Step 9: Unlock the form and add the phone input**

In `handloom-admin-frontend/src/features/coupons/components/CouponFormModal.tsx`, replace the options and hint (lines 20-27) with:

```ts
const audienceOptions: SelectOption[] = [
  { value: 'ALL', label: 'Everyone' },
  { value: 'FIRST_ORDER', label: 'First order only' },
  { value: 'RETURNING', label: 'Returning customers' },
  { value: 'SPECIFIC_CUSTOMER', label: 'One specific customer' },
];
```

Delete `audienceHint` and its use at the audience `Select`.

Add `const audience = watch('audience');` beside the existing `const type = watch('type');`, then render the phone input directly after the audience `Select`, visible only when it is needed:

```tsx
        {audience === 'SPECIFIC_CUSTOMER' && !isEditing && (
          <Input
            label="Customer mobile number"
            placeholder="98765 43210"
            hint="The number the customer signs in with. +91 is assumed."
            error={errors.customerPhone?.message}
            {...register('customerPhone')}
          />
        )}
```

`InputProps` (`src/shared/components/ui/Input.tsx:4-10`) extends the native input props and supports `label`, `error`, `hint`, `leftIcon` and `rightIcon`, so the JSX above is correct as written.

- [ ] **Step 10: Write the component test**

This is the one requirement the schema and mapper tests cannot reach: that the field *appears* only for the right audience, and that the three options are genuinely selectable. There is a direct precedent to copy — `src/features/products/components/ProductFormModal/__tests__/ProductFormModal.test.tsx`, which mounts a form modal inside a `QueryClientProvider` and `vi.mock`s its api module. Follow its setup.

Create `handloom-admin-frontend/src/features/coupons/components/__tests__/CouponFormModal.test.tsx` covering exactly four things:

1. **The phone field is absent for `ALL`.** Render for creation, leave the audience at its default, assert no "Customer mobile number" field is in the document.
2. **Selecting `SPECIFIC_CUSTOMER` reveals it.** Change the audience select to `SPECIFIC_CUSTOMER`, then assert the field appears.
3. **None of the three targeting options is disabled.** Query the audience select's options and assert `FIRST_ORDER`, `RETURNING` and `SPECIFIC_CUSTOMER` are all enabled — this is the regression guard on the unlock, and it fails if someone reinstates `disabled: true`.
4. **No option label mentions a phase.** Assert the rendered select contains no text matching `/Phase \d/`. Cheap, and it catches the stale copy coming back.

Mock `@/features/coupons/api` the way the precedent mocks its own api module, so no request is attempted.

- [ ] **Step 11: Verify the whole frontend**

```bash
cd handloom-admin-frontend && npm run check && npm run test
```

Expected: typecheck, lint and format all clean; every test passes, including the pre-existing coupon tests unmodified.

- [ ] **Step 12: Commit**

```bash
cd handloom-admin-frontend
git add src/features/coupons/
git commit -m "feat(coupons): let an operator target a coupon by phone number"
```

---

## Final verification

- [ ] `cd handloom-admin && go build ./... && go vet ./... && golangci-lint run internal/...` — clean, 0 issues
- [ ] `cd handloom-admin && go test -count=1 ./internal/...` — only `postgres not available` failures
- [ ] `cd handloom-admin && make wire && git diff --exit-code internal/wire/wire_gen.go` — no drift
- [ ] `cd handloom-admin && make generate-mocks && go build ./... && go test -count=1 ./internal/service/` — a fresh mock regeneration still compiles and passes (mocks are gitignored, so a tree diff proves nothing)
- [ ] `cd handloom-admin-frontend && npm run check && npm run test` — clean
- [ ] `cd handloom-admin && make cdk-diff-dev` — **no infrastructure change**. This plan adds no Lambda, table, index or route; a diff here means something went wrong
- [ ] Every pre-existing `TestCouponService_Validate`, `TestCouponService_Redeem`, `TestCouponService_ListPublic` and `TestCouponService_ListForCart` subtest passes **unmodified**
- [ ] Manual, against a local monolith: create an `ALL` coupon and confirm it still applies at checkout; create a `FIRST_ORDER` coupon and confirm a customer with a paid order is refused with "This code is for first orders only"
- [ ] `git log --oneline main..HEAD` — one commit per task plus the spec, no fixups

## Deliberately not in this plan

- **Storefront segmentation** — no segmented banner, and `SPECIFIC_CUSTOMER` coupons stay out of the checkout picker. `ListPublic` keeps filtering to `audience = ALL`, so no cached payload can leak a targeted code. Scoped out.
- **E2E specs.** `e2e/specs/coupons/` exists now, but driving a customer to `OrderCount >= 1` needs a real paid order and therefore real payment — the constraint that kept `usage_limit` exhaustion out of Phase 1's suite. Worth a follow-up that reuses `helpers/paid-order.ts` if the UAT payment cost is acceptable.
- **Showing the phone instead of a raw `customer_id` in the coupon list.** Needs a customer lookup on read.
- **Any anti-farming mechanism.** A new phone number can claim a `WELCOME` code again; `UsagePerUser` caps it and the new `audience` outcome makes the cost measurable.
