# E2E Test Suite — Design

**Date:** 2026-08-09
**Status:** Approved for planning
**Scope:** Two sequential PRs — (1) inventory lifecycle fix, (2) `e2e/` Playwright suite

## Goal

A pre-production gate. Run one suite against the deployed dev environment, see it green, then promote to prod. Exhaustive coverage of both the B2C storefront and the admin dashboard. The suite must cost nothing in SMS, and must not leave state that affects a later run or a human's dev testing.

## Why Phase 1 exists

Investigation for this suite surfaced a production defect in the inventory lifecycle. It has to be fixed before the tests are written, because the suite's inventory assertions would otherwise all encode broken behavior.

---

# Phase 1 — Inventory lifecycle fix

## The defect

`inventory` tracks three columns: `quantity` (physical stock), `reserved_qty` (committed to orders not yet dispatched), and `available_qty` (a stored column, always rewritten as `quantity - reserved_qty`).

Correct model — three operations:

| Operation | `quantity` | `reserved_qty` | `available_qty` |
|---|---|---|---|
| Reserve — checkout initiate | — | +q | −q |
| Release — cancel / payment failure | — | −q | +q |
| Commit — goods dispatched | **−q** | **−q** | — |

Reserve and Release are implemented and correct. **Commit does not exist.**

`internal/service/order_service.go:326-330`:

```go
case domain.OrderStatusDelivered:
	now := time.Now()
	order.DeliveredAt = &now
	// Release reserved stock (it's now sold)
	// In real app, this would decrement actual stock
```

Two comments, no code. `internal/service/payment_service.go:345` is the only other `ReleaseStock` call site and it lives in `releaseOrderInventory` — the failure path.

### Consequence

`quantity` is never decremented by any order. It changes only via manual admin add/remove/adjust. Every **successful** sale therefore leaks its reservation permanently, and since `available_qty = quantity - reserved_qty`, availability ratchets downward forever while the admin UI reports full stock.

Once cumulative successful sales of a product reach its `quantity`, `available_qty` hits zero and the product becomes silently unbuyable — `validateStock` (`internal/service/cart_service.go:223`) rejects every add-to-cart — with the admin seeing stock on hand and no explanation.

Abandoned checkouts leak identically, and there is no expiry sweeper: grepping `infra/stacks/*.go` for scheduled rules returns nothing, and none of the 22 Lambdas is an order-expiry worker.

## Fix

### 1.1 Add `CommitStock`

New method on `domain.InventoryRepository` (`internal/domain/repository.go:250-272`):

```go
// CommitStock converts a reservation into a dispatch: the goods have left the
// warehouse, so both physical quantity and the reservation drop by the same
// amount, leaving available_qty unchanged.
CommitStock(ctx context.Context, productID string, quantity int, orderID string) (*InventoryTransaction, error)
```

PostgreSQL implementation mirrors `ReserveStock` (`internal/repository/postgres/inventory_repository.go:233-293`): `SELECT ... FOR UPDATE` on the `inventory` row, a single `UPDATE` setting `quantity`, `reserved_qty` and `available_qty`, and one ledger row — all inside one `pgx.BeginFunc` transaction.

Add the ledger type in `internal/domain/entity.go:127-131`:

```go
InventoryTransactionTypeCommit InventoryTransactionType = "COMMIT"
```

**Do not** compose this from `RemoveStock` + `ReleaseStock`. The arithmetic is equivalent, but they are two independent PG transactions that can half-apply, and they would write two misleading ledger rows instead of one honest `COMMIT`.

Guard: if `reserved_qty < quantity`, return `ErrCodeInsufficientStock` rather than driving the column negative.

### 1.2 Hook Commit at `SHIPPED`

In the `UpdateStatus` switch (`internal/service/order_service.go:322-344`), add a `case domain.OrderStatusShipped` branch that loops `CommitStock` over `order.Items`, and delete the two dead comments under `OrderStatusDelivered`.

**Why SHIPPED, not DELIVERED.** Stock leaves your possession at dispatch. Committing at delivery would leave `quantity` overstating physical stock for the entire transit window, so a warehouse count cannot reconcile and low-stock alerts fire days late. Customer-facing behavior is identical either way, since Commit does not move `available_qty`. The existing TODO sits under `DELIVERED` only because that is where the original author stopped.

`DELIVERED` gets no inventory effect.

### 1.3 Restock at `RETURNED`

`RETURNED` is reachable from `SHIPPED` and `DELIVERED` (`internal/service/order_service.go:494-495`), both post-commit. Add a `case domain.OrderStatusReturned` branch restoring `quantity += q` per item.

Use the existing `AddStock` with a structured reason (`RETURN order <id>`) rather than adding a new repository method — `AddStock` already does exactly `quantity += q, available_qty += q` atomically with a ledger row. Net effect across ship → deliver → return returns `quantity` to its original value.

Accepted trade-off: returns are assumed resellable and go straight back on sale without an inspection step. Return rate is queryable by reason prefix, not by ledger type.

### 1.4 Make cancel consistent and non-silent

Cancel-before-ship already releases correctly, via both `CancelOrder` (`order_service.go:415-421`) and the `CANCELLED` branch of `UpdateStatus` (`:333-338`). `CANCELLED` is unreachable from `SHIPPED`/`DELIVERED`, so release is always pre-dispatch and cannot double-decrement a committed reservation. `CancelOrder` guards on status (`:397`), so double-cancel is rejected. **No behavioral change needed here.**

Two defects around it to fix:

- **Inconsistent cancel rules.** `CancelOrder` accepts only `PENDING`/`CONFIRMED` (`:397-398`), but `validTransitions` allows `PROCESSING → CANCELLED` (`:494`). An admin pressing "Cancel Order" on a `PROCESSING` order gets an error, while setting its status to `CANCELLED` succeeds and releases stock. Align `CancelOrder` to accept `PROCESSING`.
- **Swallowed release errors.** `ReleaseStock` failures are logged and discarded in all three call sites (`order_service.go:418-420`, `payment_service.go:345-347`, `checkout_service.go:278-280`), so a failed release still reports a successful cancel and leaks silently. Propagate the error, or at minimum emit a metric that alerts.

### 1.5 Fix the ledger reference ID

`checkout_service.go:87` reserves with the literal string `"checkout"`, while every release passes `order.ID`. `ReleaseStock` does not match on reference, so the release works — but the ledger cannot be joined reserve-to-release by order, which is exactly the query needed to verify this fix holds. Pass `order.ID`.

Note the ordering constraint: the reservation is currently taken *before* the order is created (`checkout_service.go:86-97`, order created at `:147`). Either generate the order ID up front, or move reservation after order creation and extend the existing rollback.

### 1.6 Reset leaked reservations

Existing data in dev already carries leaked reservations, and fixing the code forward does not repair it — leaked reservations belong to delivered orders that will never transition again.

The catalog is dev-only and disposable, so:

```sql
UPDATE inventory SET reserved_qty = 0, available_qty = quantity;
```

**Precondition:** cancel any genuinely open orders (`PENDING`/`CONFIRMED`/`PROCESSING`) first, then run the reset. Otherwise those orders hold reservations that the reset erases, and their later cancellation calls `ReleaseStock` against a zeroed `reserved_qty` — which returns `ErrCodeInsufficientStock` and, once 1.4 has stopped swallowing that error, fails the cancel outright.

Run this **after** the rest of Phase 1 has deployed, so no new leak accrues between the reset and the fix.

### 1.7 Tests

Go unit tests in `internal/service/order_service_test.go` and `internal/repository/postgres/`, covering: commit at SHIPPED decrements both counters and holds `available_qty` steady; DELIVERED is inventory-neutral; RETURNED restores `quantity`; cancel from each of PENDING/CONFIRMED/PROCESSING releases exactly once; commit against insufficient `reserved_qty` errors rather than going negative.

---

# Phase 2 — The `e2e/` suite

## Placement and stack

A new top-level `e2e/` directory in this monorepo, not a separate GitHub repository. The monorepo already houses three independent projects with one CI system; a separate repo would duplicate secrets and CI setup and require cross-repo dispatch to trigger, for no ownership benefit.

**Playwright** with TypeScript. Multi-origin (storefront and admin are different hosts) and parallel execution are both first-class, which Cypress handles poorly.

## Target and trigger

Runs against the **deployed dev environment** — real dev URLs, real Lambdas, real DynamoDB and Neon. This is what makes it a meaningful pre-prod gate.

Triggered by **manual `workflow_dispatch`**, matching the existing manual prod-promotion flow: deploy to dev → run E2E → confirm green → run `deploy-all.yml` with `environment=prod`.

## Repository layout

```
e2e/
  package.json                 # own npm project, not in any workspace
  playwright.config.ts         # projects: storefront, admin; chromium + mobile-chrome
  .env.example                 # STORE_URL, ADMIN_URL, API_URL, test credentials
  fixtures/
    auth.ts                    # storageState per role: customer, admin
    catalog.ts                 # creates/destroys E2E-<runId> category + products
    run-id.ts                  # unique per run, tags every entity
  pages/
    store/
    admin/
    phonepe-sandbox.ts         # third-party DOM isolated to one file
  specs/
    store/
    admin/
  scripts/
    cleanup.ts                 # idempotent reaper for anything E2E-prefixed
```

Authentication uses Playwright `storageState`: log in once in global setup, reuse the cookie jar across specs. This matters beyond speed — `/api/v1/store/auth/otp/send` has no throttle in Lambda mode, so per-spec login would hammer an unprotected endpoint.

## OTP short-circuit

The suite must send zero real SMS. MSG91 is the only integration in the codebase that costs money.

The existing `sms.DevClient` (`internal/gateway/sms/client.go:36-50`) is the wrong seam. It is selected by *credential absence* (`internal/wire/providers.go:548`), not intent, so any bypass keyed on that condition would silently activate in production the day the secret failed to propagate. It also stubs only delivery — the code stays random, so a test cannot know it.

**Design:** an explicit allowlist, short-circuiting before the gateway.

```
STORE_TEST_PHONES = "+919999900001"      # exact-match E.164 allowlist
STORE_TEST_OTP    = "<GitHub secret>"     # never committed
```

In `SendOTP` (`internal/service/customer_auth_service.go:94`): if the phone is in the allowlist, store the hash of `STORE_TEST_OTP` and return without touching `smsGateway`. Everything else — hashing, TTL, attempt limits, verification — is unchanged, so the tested path stays the production path.

Three independent guards:

1. **Startup assertion** — config validation calls `log.Fatal` if `env == "prod"` and the allowlist is non-empty. Fails the Lambda cold start, loudly and immediately.
2. **CDK never sets the variable on the prod stack** — absent entirely, not set-to-empty, so console or SSM drift cannot enable it without a code deploy.
3. **Exact-match on full E.164 only** — never a prefix or pattern.

Real phone numbers in dev continue to hit the real MSG91 gateway, so manual dev testing is unaffected.

## State restoration

`AdjustStock` cannot restore inventory. `inventory_repository.go:380` computes `availableQty := newQuantity - reservedQty` and never writes `reserved_qty`, so it carries any leak forward by construction. Compensating arithmetic through that endpoint would force falsifying `quantity` and the audit ledger.

**The suite therefore never touches real catalog products.** It creates its own, and `DELETE /admin/products/{id}` is a real row delete (`internal/repository/postgres/product_repository.go:471-479`) that cascades `inventory`, `inventory_transactions`, `product_attribute_values` and `product_images`. Deleting the product deletes any leak with it. Restoration becomes exact and needs no arithmetic.

```
setup    → create E2E category + products (prefixed E2E-<runId>-), stock them
tests    → buy only E2E products
teardown → cancel every order created   (releases the reservation)
         → delete E2E products          (cascades inventory + ledger)
         → delete E2E category          (blocked while product_count > 0 — order matters)
```

`scripts/cleanup.ts` is deliberately separate from Playwright teardown: a crashed spec skips its own teardown. It is idempotent, runs as a CI post-step with `if: always()`, and can be run by hand.

### What cannot be undone

Stated explicitly rather than papered over. Building delete paths for these would be a worse trade than leaving them.

- **Orders** — `domain.OrderRepository` (`internal/domain/order_repository.go:9-37`) has no `Delete` method at all. Cancel is the only lever, and it is needed anyway to release the reservation. Orders accumulate on the test customer permanently.
- **`customer.OrderCount` / `TotalSpent`** — atomic DynamoDB `ADD`, with no decrement anywhere in the codebase.
- **`metric_counters`** — retention is a `pg_cron` job (`migrations/008_metrics_retention_cron.sql`) that is a no-op unless pg_cron was manually enabled in the Neon console.

**Accepted:** analytics pollution. Every run writes `orders_placed`, `payment_completed`, `product_viewed` and similar into `metric_counters`, which feeds the `/dashboards` funnel. Dev funnel and conversion figures will be skewed by CI traffic, and anyone reading dev dashboards must discount it. No suppression mechanism will be built.

## Test accounts

Two, since the suite spans both applications.

**Storefront customer** — a single fixed phone on the OTP allowlist. Deliberately *not* fresh per run: `CustomerService.Delete` (`internal/service/customer_service.go:120-140`) is a soft delete that refuses outright once the customer has ≥1 order, so per-run customers would accrete undeletable records forever. Consequence: this account can never exercise `customer_first_purchase`, which fires once and burns.

**Admin user** — one seeded `ADMIN`, following the pattern in `scripts/seed-remote.go`. Authorization coverage is out of scope; note that only `/users` is role-gated client-side, so every other destructive admin action is reachable by `OPERATOR` and relies entirely on backend enforcement that this suite will not verify.

## Payment

Every purchase spec drives the **real PhonePe sandbox UI** end to end. Dev is genuinely sandboxed (`internal/config/config.go:167` → `api-preprod.phonepe.com/apis/pg-sandbox`), so this costs nothing.

Requires PhonePe sandbox test-instrument details as CI secrets. PhonePe's hosted page is third-party DOM that can change without notice, so all interaction with it is confined to `pages/phonepe-sandbox.ts` — a DOM change is then a one-file fix rather than a suite-wide rewrite.

## Coverage

### Storefront

| Area | Specs |
|---|---|
| Catalog | home; `/categories`; `/c/[slug]` with attribute, price and in-stock filters plus infinite scroll; `/products` with search; PDP gallery, lightbox, video and spec accordion; Spotlight ⌘K; mobile nav drawer |
| SEO | `sitemap.xml`; `robots.txt`; OG and meta tags; JSON-LD on PDP and category |
| Cart | add from PDP, product card and carousel; quantity ±; remove; MiniCart drawer; mobile FAB; guest cart merge on login; out-of-stock rejected |
| Auth | OTP send → verify; wrong OTP; 3-attempt lockout; resend with 30s countdown; logout; protected-route redirect preserving `?redirect=`; refresh-token rotation across concurrent tabs |
| Checkout | address add with validation (phone `^[6-9]\d{9}$`, PIN `^\d{6}$`); saved-address select; review step; Pay Now → sandbox → confirmation poll → PAID; payment-failure path; empty-cart redirect |
| Account | profile edit; order history; order detail with status timeline; cancel restricted to PENDING/CONFIRMED; address CRUD |
| Tracking | `/track` with valid and invalid order numbers |
| Static | legal pages; contact page |

### Admin

Login, session persistence and logout · categories CRUD including attributes and SELECT options · products CRUD with image upload and stock-adjust-on-edit · drag-and-drop ranking · product filters and attribute facets · inventory low-stock list and add/remove/adjust with reason · orders list with filters and status transitions · order detail (status, tracking, payment check, note, cancel) · customers CRUD with addresses and `?id=` deep-link · coupons CRUD · pricing rules CRUD · reports generate and history · notifications mark-read · users CRUD with activate/deactivate · settings profile and password change.

### Excluded

**`/dashboards/*`.** Gated by a second, independent authentication system — Neon Auth via Google OAuth plus an email allowlist (`src/features/dashboards/components/NeonAuthGate.tsx`). Automating Google sign-in is brittle and generally contrary to their terms for CI. Revisit only if a service-account path becomes available.

### Regression guards for known defects

Two specs assert correct behavior against bugs that Phase 1 does not fix. They land as `test.fixme()` with ticket links, so they are tracked rather than silently absent, and flip to passing when fixed.

1. **Checkout is not idempotent.** `POST /api/v1/store/checkout/initiate` (`internal/handler/store/checkout_handler.go:42-59`) has no idempotency key and no in-flight-order check. N rapid clicks produce N orders, N payments and N reservations. Spec: double-click Pay Now, assert one order.
2. **`/otp/send` is unthrottled in every deployed environment.** The 30/min limiter exists only in `cmd/api/main.go:125`, the local monolith; the Lambda router `internal/router/store_auth.go` mounts the auth handler with no limiter, and dev and prod both run Lambda mode. Spec: flood `/otp/send`, assert throttling.

The refresh-token rotation spec directly covers the work in progress on `fix/store-refresh-token-race`.

---

# Out of scope — findings requiring separate tickets

Surfaced during this investigation. None is addressed by either phase.

1. **Production PhonePe points at the sandbox.** `handloom-admin/infra/cmd/config.go` sets the `prod` block's `PhonePeBaseURL` to `https://api-preprod.phonepe.com/apis/pg-sandbox`, carrying the comment `// live PhonePe — VERIFY` and byte-identical to the `dev` block. If prod has been deployed from this config, production checkouts transact against sandbox: customers are not charged, and orders are marked paid against test money. **Verify before the next prod deploy.**
2. **`/api/v1/store/auth/otp/send` is public and unthrottled in production.** Per item 2 above. Combined with the absence of any per-phone cooldown or resend cap in `SendOTP`, this is an open SMS-pumping vector billing real money to your MSG91 account.
3. **`CUSTOMER_JWT_SECRET` has an insecure default.** `internal/config/config.go` falls back to the literal `"customer-secret-change-in-production"` when unset, yielding forgeable customer tokens with no startup warning.
4. **The audit log is never written.** `AuditService.Log` (`internal/service/audit_service.go:28`) has zero production callers — only its own test file. `handloom-audit-{env}` is written by nothing, and the admin audit screens read a permanently empty table.
5. **No sweeper for abandoned checkouts.** A customer who initiates checkout and closes the tab leaves a `PENDING` order holding its reservation forever. After Phase 1 this is the only remaining permanent-leak path.
6. **Duplicate middleware in the storefront.** Both `homechrome-store/src/middleware.ts` (auth gate) and `homechrome-store/middleware.ts` (CloudFront geo enrichment) exist. Next.js resolves a single middleware file and prefers `src/` when that directory is present, so the geo middleware is very likely dead — meaning country, city, latitude and longitude never reach the backend. The two matchers are disjoint (`/checkout|/account` versus `/api`), so they must be merged rather than swapped.
7. **Missing 404 handling in the storefront.** There is no `not-found.tsx` anywhere; missing products and categories render an inline "not found" body with HTTP 200.
8. **Admin frontend documentation is stale.** `.claude/CLAUDE.md` describes `src/pages/`, `src/api/` and routes defined in `App.tsx`. The project was refactored to a feature-sliced layout: routes live in `src/app/routes.tsx`, pages at `src/features/<feature>/components/*Page.tsx`, API modules at `src/features/<feature>/api.ts`.
9. **Backend documentation describes a removed integration.** `handloom-admin/CLAUDE.md` and `README.md` document a Shiprocket gateway with a DevClient. It does not exist — `internal/gateway/` contains only `phonepe/` and `sms/`, and `internal/handler/store/tracking_handler.go:91` states the integration was removed. Residual `SHIPROCKET_*` environment variables are inert. The same file's claim that `/auth/*` is rate-limited at 30/min is also wrong in every deployed environment.
