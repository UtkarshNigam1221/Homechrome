# Coupon Audience Enforcement — Design

**Date:** 2026-09-02
**Status:** Approved for planning
**Scope:** One PR on `main` — enforce `Coupon.Audience` in the validator, and unlock the admin form's targeting options
**Issue:** #255, part of #251 — Phase 2 of five
**Depends on:** Phase 1 (#275, #285, #286), all merged

## Goal

Phase 1 gave coupons an `Audience` and never enforced it. `CouponService.Validate` checks status, validity window, `min_order_value`, `usage_limit`, `usage_per_user` and the stacking flag; it does not read `coupon.Audience` at all. So today:

- a `FIRST_ORDER` code works for a customer with fifty orders
- a `RETURNING` code works for a first-time buyer
- **a `SPECIFIC_CUSTOMER` coupon minted for one person is redeemable by anyone who learns the code**

The repository filters `audience = ALL` for the public list (`internal/repository/dynamodb/coupon_repository.go:286`), which is what keeps targeted coupons off the banner and out of the picker. It does nothing on the redeem path. This phase closes that, and unlocks the three targeting options the admin form currently disables.

## Note on the issue text

#255's body is stale. It says "add `FirstOrderOnly bool` to `Coupon`". Phase 1 replaced independent first-order/returning flags with a single `CouponAudience` enum (`internal/domain/coupon.go:31-38`) precisely so that "first order AND returning" — unsatisfiable — could not be expressed. The field exists; only enforcement is missing. #255 also predates the offers banner and the checkout picker entirely.

The admin form compounds the confusion: `CouponFormModal.tsx:22-27` labels all three targeting options "(Phase 3)" and hints "targeting arrives in Phase 3". The epic's numbering makes this Phase 2. The copy is what's wrong, and this phase removes it.

## Scope

**In:** backend enforcement, and the admin form changes needed to create targeted coupons.

**Out:** storefront segmentation. Phase 1 deferred a segmented banner (`FIRST_ORDER` for guests and zero-order customers, `RETURNING` for those with orders) and `SPECIFIC_CUSTOMER` coupons in the checkout picker to this issue. Both stay deferred. After this phase an operator can create a targeted coupon and hand the code out directly; customers never see it listed. That is deliberate — the public list keeps filtering to `audience = ALL`, so no cached payload can leak a targeted code.

## The rules

| audience | rule | customer sees on rejection |
|---|---|---|
| `ALL` | always passes | — |
| `FIRST_ORDER` | `OrderCount == 0` | "This code is for first orders only" |
| `RETURNING` | `OrderCount >= 1` | "This code is for returning customers" |
| `SPECIFIC_CUSTOMER` | `coupon.CustomerID == cc.CustomerID` | "That code isn't valid" |
| any targeted audience, empty `CustomerID` | reject | "That code isn't valid" |

`SPECIFIC_CUSTOMER` returns the message byte-identical to the unknown-code rejection at `coupon_service.go:212`. A stranger who guesses a real targeted code must not be able to tell it apart from a typo, or the refusal itself becomes an enumeration oracle. `FIRST_ORDER` and `RETURNING` say what they mean, because knowing the rule helps the customer and reveals nothing worth having.

An empty `CustomerID` rejects rather than skipping the check. A targeted coupon's validity is undefined without an identity, and fail-closed is the only sane default on a discount path. Only one caller can reach this: `POST /admin/coupons/validate` takes `customer_id` as an optional body field (`internal/handler/request_types.go:55-60`). Every storefront path runs behind `CustomerAuth` and always has an id.

### The signal, and its ordering subtlety

`Customer.OrderCount` (`internal/domain/order.go:204`) is maintained atomically by `CustomerRepository.RecordPurchase`, which runs in `HandlePaymentSuccess` — so it counts **paid** orders. That is the right signal, and it is what makes a genuine first order still read as `OrderCount == 0` at validation time, because validation happens at checkout and `RecordPurchase` has not run yet.

#255 asks us to confirm that rather than assume it. A test pins it.

The same ordering creates an accepted overshoot: a customer who initiates two checkouts before paying either has `OrderCount == 0` for both, so a `FIRST_ORDER` coupon prices into both orders. This is the same structural overshoot Phase 1 documented for `usage_limit` — counting at payment success plus honouring a quoted price makes it unavoidable. `UsagePerUser` caps it.

## Where the order count comes from

`CouponService` gains a second dependency, `customerRepo domain.CustomerRepository`. It currently holds only `couponRepo` (`internal/service/coupon_service.go:31-33`).

`Validate` resolves the count **only** when `coupon.Audience` is `FIRST_ORDER` or `RETURNING` *and* `cc.CustomerID != ""` — one `CustomerRepository.GetByID`. `SPECIFIC_CUSTOMER` needs no read: it is a string comparison against `coupon.CustomerID`. `ALL` needs no read.

That conditionality matters for cost. `ListPublic` returns only `audience = ALL` coupons, so `ListForCart` pricing M candidates for the checkout picker triggers **zero** customer reads. This design does not reintroduce the per-candidate read Phase 1 removed by adding `GetCustomerUsageAll`.

A failed customer read rejects. It does not fall through to a permissive default.

`ListForCart` passes `nil` and resolves nothing. Its candidates come from `ListPublic`, which returns only `audience = ALL`, so no candidate can need a count. Should a future change ever let a targeted coupon into that list, `nil` means the targeted audiences reject — the picker would under-show rather than over-promise, which is the right direction to fail.

### The alternative, and why not

The other option was adding the count to `CouponContext` (`internal/domain/coupon.go:181-185`) and having callers populate it. It keeps `CouponService` on one dependency, but `CustomerOrderCount int` has a zero value of `0`, which means "first order", which is the *permissive* case. Any of the four construction sites — `coupon_handler.go:140`, `checkout_service.go:239`, `:267`, `:284` — that failed to set it would silently grant every `WELCOME` code to every existing customer, which is the exact bug this phase exists to fix. Guarding it with `*int` and nil-means-reject recovers the fail-closed semantics but still leaves four places to get right.

Audience enforcement is a security control. It should not depend on callers remembering to populate a field whose default is "grant it".

A hybrid — `*int` in the context, the service resolving only when nil — would let a caller that already loaded the customer skip a read. `resolveCoupon` does not currently load the customer, so that saving is hypothetical. Not building it.

## `evaluate` stays pure

```go
func evaluate(
	c *domain.Coupon,
	cc domain.CouponContext,
	used int,
	orderCount *int,
) *domain.CouponValidationResult
```

`nil` means "not resolved", and the targeted audiences reject on it. Fail-closed becomes a property of the type rather than a rule every caller must remember. This matches the `ValidUntil *time.Time` precedent already in the same file, where nil carries a specific meaning rather than a zero.

`Validate` resolves the count before calling `evaluate`, so `evaluate` remains a pure function of its arguments and stays trivially testable — the property that made Phase 1's extraction worth doing.

### Branch position

The audience check goes **after** status and the validity window, **before** `min_order_value`.

A dead coupon is dead regardless of who is asking, so status and window come first. But between "Add ₹500 more to use this coupon" and "This code is for first orders only", the second is the one the customer can act on: a cart can change, an order history cannot. Getting told to add ₹500 and then still being refused is the worse experience.

One accepted cost of resolving before `evaluate`: a targeted coupon that turns out to be expired still costs one `GetByID`. One read on a rejected code is worth keeping `evaluate` pure.

## Metrics

A new `Outcome` value, `"audience"`, for all four rejection rows. `Outcome` is tagged `json:"-"` (`internal/domain/coupon.go:339-341`), so the `SPECIFIC_CUSTOMER` rejection stays indistinguishable to the customer while remaining fully visible in the funnel.

That visibility is the point. It makes the accepted farming limitation measurable: if `audience` rejections on a `FIRST_ORDER` code climb, that is real signal about whether the limitation ever needs revisiting.

## Admin form

**`FIRST_ORDER` and `RETURNING`:** enable them, drop the "(Phase 3)" suffixes and the `audienceHint` line. Neither needs a new field. Audience stays locked on edit — the backend has no path to change it, and Phase 1 already enforces that in the form.

**`SPECIFIC_CUSTOMER`** needs a customer identity, and the form has no field for one. The backend already rejects the attempt: `CreateCouponRequest.CustomerID` carries `validate:"required_if=Audience SPECIFIC_CUSTOMER"` (`internal/domain/coupon.go:305`).

The operator knows the customer's phone number, not an internal id — customers authenticate by phone OTP. So the form takes a phone.

### Phone in, id stored

**The form** shows a phone input only when `SPECIFIC_CUSTOMER` is selected, with a fixed `+91` prefix rendered beside it and a 10-digit constraint — the same shape `homechrome-store/src/app/login/page.tsx` uses, so an operator cannot produce a format the backend will not find.

**The request** gains `CustomerPhone string` with `validate:"required_if=Audience SPECIFIC_CUSTOMER"`, and `CustomerID` is **removed from `CreateCouponRequest`**. `domain.Coupon.CustomerID` does not change — only the request shape does.

This is a breaking change to `POST /admin/coupons`, and the admin frontend is its only client; both move in the same PR. It fails loudly rather than quietly: a stale client sending `customer_id` has that field ignored, then trips `required_if` on the absent `CustomerPhone` and gets a `400`. Keeping both fields and accepting either would leave two ways to express one thing and a rule about which wins.

**Normalisation happens on the server.** `CustomerRepository.GetByPhone` is an exact-match `GetItem` on `PK = CUSTOMER_PHONE#<phone>` (`internal/repository/dynamodb/customer_repository.go:148-155`) with no normalisation of any kind, and the stored value is E.164: the storefront sends `+91` + ten digits (`login/page.tsx`, `sendOTP(\`+91${cleaned}\`)`), and `CustomerPhoneIndex.SetKeys` keys the pointer on that string verbatim (`internal/domain/order.go:256`).

So the server normalises, in this order:

1. strip every non-digit
2. if twelve digits remain and they start with `91`, drop that `91`
3. if eleven digits remain and they start with `0`, drop that `0`
4. require exactly ten digits — anything else is a `400` naming the format expected
5. prepend `+91`

Steps 2 and 3 are what make `+91 98765 43210` and `098765-43210` resolve to the same customer as `9876543210`. Without step 2, stripping non-digits from a pasted `+91…` leaves twelve digits and fails the length check — which reads to the operator as "customer doesn't exist" for a number that plainly does.

Normalising only in the form would mean a number pasted from a support ticket fails for formatting reasons alone — the failure mode is silent and reads as "customer doesn't exist", which is the worst possible message to show an operator holding a phone number that plainly does exist.

**Resolution stores the id, never the phone.** `CouponService.Create` normalises, calls `GetByPhone`, and sets `coupon.CustomerID = customer.ID` before the repository write — so the transactional code-uniqueness guard Phase 1 built is unaffected, and a coupon never reaches storage with an unresolved customer. There is a profile path that updates `Customer.Phone`, so a number can move between people; the id is the identity. A coupon must stay bound to the human even if they change SIM.

**Not found → `400`, "No customer with that number".** Specific, because the reader is an authenticated operator. The enumeration concern applies to the storefront, not here.

This is the second use of the `customerRepo` dependency: order count at validate, phone resolution at create.

## Error handling

| condition | behaviour |
|---|---|
| customer read fails during validate | reject the coupon; do not grant |
| customer not found during validate | reject; treat as unresolved |
| phone does not resolve at create | `400` with a specific message |
| phone fails normalisation at create | `400` naming the format expected |
| coupon read fails on the public list or picker | unchanged from Phase 1 — `200` with an empty list, logged at warn |

## Test plan

**Service** (`internal/service/coupon_service_test.go`):

- each audience, both sides of its boundary: `FIRST_ORDER` with `OrderCount` 0 and 1; `RETURNING` with 0 and 1; `SPECIFIC_CUSTOMER` matching and mismatched
- the `SPECIFIC_CUSTOMER` mismatch message is asserted **equal to** the unknown-code message, by comparing the two results rather than pinning the same literal in two places — a test that pins the literal twice passes while they drift apart
- empty `CustomerID` rejects for all three targeted audiences
- **an `ALL` coupon triggers zero customer reads**, asserted by mock call count. This is the guard against reintroducing a per-candidate read
- `ListForCart` with M candidates: zero customer reads, same assertion
- a customer read failure rejects rather than granting
- a customer with an unpaid pending order still validates a `FIRST_ORDER` coupon — pins #255's ordering subtlety and documents the accepted overshoot
- every existing `TestCouponService_Validate` and `TestCouponService_Redeem` subtest passes **unmodified**. That, not the new tests, is the evidence enforcement did not change behaviour for `ALL` coupons

**Create path:**

- `+91 98765 43210`, `098765-43210` and `9876543210` all resolve to the same customer id
- a phone that normalises to the wrong length is a `400`, not a lookup attempt
- an unresolvable phone is a `400` naming it
- the stored coupon carries `CustomerID`, never a phone

**Admin frontend** (`handloom-admin-frontend`, has vitest):

- the phone field appears only for `SPECIFIC_CUSTOMER` and is required then
- audience remains locked on edit
- the three options are no longer disabled and no longer say "Phase 3"

**Not covered:** e2e. `e2e/specs/coupons/` now exists (#285 merged), so specs for targeted coupons are cheap to add and worth a follow-up, but driving a customer to `OrderCount >= 1` needs a real paid order and therefore real payment — the same constraint that kept `usage_limit` exhaustion out of Phase 1's suite.

## Accepted limitations

- **New-account farming is not solved.** A phone-authenticated customer can register a new number and claim a `WELCOME` code again. Perfect prevention is not available; `UsagePerUser` plus a modest discount caps the damage. #255 explicitly asks whether that is acceptable rather than building device fingerprinting: it is, and the new `audience` metric makes the cost measurable.
- **`FIRST_ORDER` can over-grant across concurrent unpaid checkouts**, as described above.
- **The coupon list still shows a raw `customer_id`**, which tells an operator nothing. Rendering the phone needs a customer lookup on read. Out of scope.
- **No storefront surface for targeted coupons.** Codes are handed out directly until the segmentation work lands.
