# Coupon Public Offers — Design

**Date:** 2026-08-28
**Status:** Approved for planning
**Scope:** One PR on top of `feat/coupons-phase-1` (#275) — a cached public offers list and a live checkout coupon picker
**Depends on:** #275 merged. Nothing here needs #285 (e2e), but the new specs land in the `specs/coupons/` directory #285 creates.

## Goal

Two customer-facing coupon reads that do not exist today:

1. A **green offers banner** on the homepage, showing currently-running public coupons. Same payload for guests and logged-in customers, served from cache, zero DynamoDB reads on the hot path.
2. A **coupon picker** in checkout, listing the codes this customer can use against the cart in front of them, with the real saving on each.

Today the storefront has exactly one coupon call: `POST /api/v1/store/checkout/validate-coupon`, which takes a code the customer typed. Nothing lists coupons to a customer.

## Access patterns, as scoped

The four reads originally raised — active coupons, public coupons, coupons for a user, one coupon's detail — collapse into two, because they all feed one screen each:

| # | pattern | surface | shape |
|---|---|---|---|
| 1 | public coupons currently running | homepage banner | cached list, no customer context |
| 2 | coupons usable on this cart | checkout picker | live list, per customer, per cart |
| 3 | one coupon by code | apply / `Validate` | exists, unchanged |
| 4 | admin list, filtered and paged | admin dashboard | exists, unchanged (`055d2ac`) |

## Decisions

### Public means `audience = ALL`, for now

`CouponService.Validate` (`internal/service/coupon_service.go:180`) checks status, validity window, `min_order_value`, `usage_limit`, `usage_per_user` and the stacking flag. It never reads `coupon.Audience`. A `SPECIFIC_CUSTOMER` coupon is therefore redeemable by anyone who learns the code, and a `FIRST_ORDER` coupon by a returning customer.

Audience enforcement is #255 (Phase 2). Until it lands, **nothing advertises a coupon whose eligibility rule the server does not enforce.** Both new endpoints filter to `audience = ALL`. Segmented banners (`FIRST_ORDER` for guests and zero-order customers, `RETURNING` for customers with `OrderCount > 0`) and `SPECIFIC_CUSTOMER` coupons in the picker are deliberately deferred to #255.

`Customer.OrderCount` (`internal/domain/order.go:204`) already exists and is incremented atomically by `CustomerRepository.RecordPurchase`, so the segment test is a field read when #255 arrives — no orders query, no new counter.

### Reuse `GSI1`, add no index

`ListPublic` queries the existing `GSI1PK = COUPON#ALL` partition with an equality filter. Two alternatives were considered and rejected:

- **A second GSI keyed by audience** (`COUPON#PUBLIC#<AUDIENCE>` / `valid_until`) gives key-condition-only reads and hands Phase 5's wallet back the expiry range `055d2ac` removed. It also duplicates every coupon write into a third index and moves index entries on status or audience edits. The banner read happens once per cache TTL; the query shape is not the cost.
- **A precomputed snapshot item** holding the trimmed list as JSON would be one `GetItem`. But a coupon expiring involves no write, so the snapshot decays on its own and validity must be rechecked at read — which is the query below, plus an invalidation bug.

Revisit the second GSI when #255 needs per-audience reads *and* the banner is uncached. Not before.

### The expiry check stays in Go

`attributevalue` marshals `time.Time` as `time.RFC3339Nano`, and the SDK's own comment warns the format *"removes trailing zeros from the seconds field"* (`attributevalue@v1.20.61/encode.go:28`, the version in `go.mod`). So `valid_until` is a variable-width string: `"…10:00:00Z"` vs `"…10:00:00.5Z"` compares wrong, because `.` (0x2E) sorts before `Z` (0x5A). This is the same trap `055d2ac` fixed for `GSI1SK` by using a fixed-width nine-digit layout.

`ValidUntil` is also `*time.Time` with `omitempty`, so it is absent on open-ended coupons and a filter would need `attribute_not_exists` alongside the comparison.

Both reasons point the same way: filter `status` and `audience` in DynamoDB (equality, safe on strings), and evaluate the window in Go on a `*time.Time`.

### One eligibility rule, two callers

Everything in `Validate` after `GetByCode` is extracted:

```go
func evaluate(c *domain.Coupon, cc domain.CouponContext, usedByCustomer int) *domain.CouponValidationResult
```

`Validate` becomes `GetByCode` → `GetCustomerUsage` → `evaluate`. The picker does one bulk usage query → `evaluate` per candidate. The picker cannot promise a saving `Validate` then refuses, because there is one function computing both. Existing `Validate` tests passing unchanged is the regression guard on the extraction.

## Endpoints

| endpoint | Lambda | auth | cache |
|---|---|---|---|
| `GET /api/v1/store/catalog/coupons` | `store-catalog` | none | `public, max-age=3600` |
| `GET /api/v1/store/checkout/coupons` | `store-checkout` | customer JWT | `no-store` |

`storeRoutes` in `infra/stacks/api.go:674` maps one path segment per Lambda under a `{proxy+}` resource, so a top-level `/api/v1/store/coupons` would need a new map entry and a new API Gateway resource. Mounting the public list under `catalog` costs zero infra: it is browse-surface data, served by the Lambda the ISR-cached homepage already calls. `store-catalog` gains `CouponService` in its Wire deps (`make wire`).

The picker goes on `StoreCheckoutHandler`, which already holds `domain.CouponService` (`internal/service/checkout_service.go:24`) and already resolves the customer's cart. No new wiring.

### Public response

```json
{
  "success": true,
  "data": [
    {
      "code": "WELCOME10",
      "name": "10% off your order",
      "description": "Valid on all handloom sarees",
      "type": "PERCENTAGE",
      "value": 1000,
      "min_order_value": 100000,
      "max_discount": 50000,
      "valid_until": "2026-09-30T18:29:59Z"
    }
  ]
}
```

`value` follows the existing convention: percentage × 100, or paise for `FIXED`. Money fields stay paise; the storefront already formats them.

Order is index order, newest first. No server-side "best offer" ranking — a percentage and a fixed amount are not comparable without a cart.

### Picker response

Same fields per coupon, plus:

```json
{ "eligible": true, "discount_amount": 25000, "reason": "" }
```

`reason` carries the customer-facing message from `evaluate` when `eligible` is false — *"Add ₹500 more to use this coupon"*, *"You've already used this coupon"* — so the picker shows a code and what stands between the customer and it. Eligible coupons first, sorted by `discount_amount` descending; ineligible ones after, in index order.

### Fields withheld from both DTOs, and why

This is a store-facing DTO, not a GSI projection change. `GSI1` stays `ProjectionType_ALL`: the admin list needs whole items, and the read volume this design adds is roughly two queries per hour.

| withheld | reason |
|---|---|
| `usage_count`, `usage_limit`, `usage_per_user` | tells a customer a code is nearly exhausted; scrapeable scarcity data |
| `customer_id` | on a `SPECIFIC_CUSTOMER` coupon this is **another customer's id**. Privacy, not optimization — guaranteed at the DTO even though today's filter excludes those coupons |
| `id`, `batch_id`, `search_key`, `status`, `audience`, `combines_with_offers`, `created_at`, `updated_at` | internal |

## Repository

```go
// ListPublic returns coupons safe to advertise: ACTIVE, audience ALL, and valid
// past the cache window. See the design note on RFC3339Nano before moving the
// expiry check into the filter.
ListPublic(ctx context.Context) ([]*Coupon, error)

// GetCustomerUsageAll returns every per-coupon count for one customer, keyed by
// coupon id. One query instead of a GetItem per candidate.
GetCustomerUsageAll(ctx context.Context, customerID string) (map[string]int, error)
```

`ListPublic`:

```
Query GSI1
  KeyConditionExpression: GSI1PK = :pk          (COUPON#ALL)
  FilterExpression:       #status = :active AND audience = :all
  ScanIndexForward:       false
```

Read via `QueryAll`, not `QueryPage` — this returns a set, not a page, and a banner's worth of public coupons is small by construction. Then in Go, drop any coupon where `ValidFrom` is in the future, or where `ValidUntil` is non-nil and falls **inside the cache window**: `validUntil.Before(now.Add(publicListTTL))`.

That window predicate, not a bare "already expired" check, is what makes a 1-hour TTL safe: a cached payload can never advertise a coupon that has already expired by the time someone reads it. Cost is that a coupon expiring in 30 minutes leaves the banner up to an hour early — the correct trade for a payload served to everyone from cache.

`PublicCouponListTTL` is one exported const in `internal/domain/coupon.go` — the repository filters by it and a handler test asserts `max-age` equals it, so neither can move alone without failing. It lives in `domain` rather than the repository package so the assertion does not make a handler import a repository. A TTL that lives in two places drifts, and the drift is silent: a longer `max-age` than the filter window puts expired coupons back on the banner.

`GetCustomerUsageAll`:

```
Query
  KeyConditionExpression: PK = :pk AND begins_with(SK, :prefix)   (CUSTOMER#<id>, USE#)
```

Keys come from the item's own `coupon_id` attribute, not from parsing the `USE#` prefix off `SK`. `CouponUseCounter.SetKeys` (`internal/domain/coupon.go:164`) keys counters `CUSTOMER#<id>` / `USE#<couponID>` precisely so one query returns them all. `Validate`'s existing `GetCustomerUsage` is a `GetItem` per coupon; for a picker of M candidates that would be M reads.

## Caching

- Endpoint sets `Cache-Control: public, max-age=3600`. `CatalogHandler.Routes()` already applies `middleware.CatalogCacheControl("public, max-age=3600")` to every GET it serves, so the route inherits the header with no work. No `stale-while-revalidate`: it would need a route-scoped middleware for no measurable gain.
- Storefront homepage fetch uses `next: { revalidate: 3600 }`, the value already on `src/app/page.tsx` and `src/app/layout.tsx`. No new number enters the codebase.
- Picker sets `Cache-Control: no-store`.
- **No backend cache.** `CLAUDE.md` and `docs/` describe an in-process TTL cache at `internal/cache/` wrapping the catalog repos; it does not exist — no such directory, no `go-cache` in `go.mod`, no `Cache` in `internal/repository/`. The docs are stale and should be corrected separately. Building one to protect a read that now happens twice an hour would be backwards.

**Accepted staleness:** an operator flipping a coupon to `INACTIVE` reaches no cached payload, so it stays advertised for up to an hour. Bounded: the picker is live and `Validate` is the authority, so the customer gets *"This coupon is no longer available"* on apply, never a wrong price. If that support call proves real, the fix is a revalidation hook on coupon write, not a shorter TTL.

## Error handling

Both endpoints return `200` with an empty list on any read failure, logging at warn. A dead coupon path must not blank the homepage or block a checkout — the customer can still type a code, and `Validate` remains the authority. This follows the precedent already in `checkout_service.go:246`: *"A coupon lookup failure must not cost the sale."*

## Test plan

**Repository, against DynamoDB local** (`internal/repository/dynamodb/coupon_repository_test.go`):

- `ListPublic` returns only `ACTIVE` coupons with `audience = ALL` — an `INACTIVE` one and a `FIRST_ORDER` one are both absent
- an open-ended coupon (`ValidUntil == nil`, so the attribute is absent) **is** returned
- an already-expired coupon is absent
- a coupon expiring inside the TTL window is absent
- a coupon with `ValidFrom` in the future is absent
- the RFC3339Nano trap, explicitly: one coupon expiring on a whole second and one a fraction later, both classified correctly. This test fails if the expiry check is ever pushed into a DynamoDB string comparison
- `GetCustomerUsageAll` returns every counter for one customer and nothing belonging to another

**Service** (`internal/service/coupon_service_test.go`):

- every existing `Validate` test passes unchanged after the `evaluate` extraction
- the picker asserts **exactly one** usage query for M candidates, via mock call count
- an ineligible candidate carries the same `reason` string `Validate` would return for the same cart

**Handler** (`internal/handler/store/`):

- serialization test asserting `usage_count`, `usage_limit`, `usage_per_user` and `customer_id` are absent from both payloads
- `Cache-Control` is set as specified on each endpoint
- a repository error yields `200` and an empty list, not a `5xx`

**E2E** (`e2e/specs/coupons/`, the directory #285 creates):

- public list returns a created `ACTIVE`/`ALL` coupon and omits an `INACTIVE` one
- picker against a cart below `min_order_value` returns the coupon as ineligible with the shortfall message; above it, eligible with the computed saving

**Storefront:** no test runner exists (no vitest, no jest, no config), so the banner and picker UI ship verified by reading and by the e2e specs hitting the endpoints beneath them. Same gap recorded in #275.

## Out of scope

- Audience segmentation, the `RETURNING` banner, and `SPECIFIC_CUSTOMER` coupons in the picker — #255
- Fixing the audience enforcement hole in `Validate` — #255, and the reason this design advertises `audience = ALL` only
- A second GSI, a snapshot item, or any change to `GSI1`'s `ALL` projection
- Correcting the stale `internal/cache/` documentation in `CLAUDE.md` and `docs/` — real, unrelated, separate commit
- Any backend cache layer
- Auto-apply or best-coupon selection. The picker lists and annotates; the customer chooses

## Risks

| risk | mitigation |
|---|---|
| A deactivated coupon stays advertised up to an hour | `Validate` refuses it on apply with a customer-facing message; picker is live |
| The `evaluate` extraction changes `Validate`'s behaviour | Existing `Validate` tests must pass unmodified; that is the gate |
| Wiring `CouponService` into `store-catalog` grows that Lambda's cold start | One service and one repository, both small; measure after deploy rather than pre-optimising |
| Public payload leaks a field later added to `Coupon` | The DTO is explicit, not a struct copy, and a serialization test asserts the withheld set |
