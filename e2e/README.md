# e2e

End-to-end suite for the **deployed dev environment**. Real Lambdas, real
DynamoDB, real Neon, real PhonePe. Nothing here is mocked or stubbed — a suite
that mocks the seams cannot catch bugs that only live in them, and every defect
this exists for lived in a seam:

- **DynamoDB ↔ Postgres.** Orders live in one, inventory in the other, with no
  shared transaction. Every reserve, commit, release and write-off crosses it.
- **Two implementations of the money.** The refund amount is derived in Go and
  again in TypeScript for the admin preview.
- **Migrations on live data.** `013` and `014` re-key uniqueness on the ledger.

## Running it

Against dev, from CI: **Actions → E2E (dev) → Run workflow**. Also nightly.

Locally:

```bash
cp .env.example .env      # fill it in; never commit it
npm install
npx playwright install --with-deps chromium
npm test                  # everything
npm run test:api          # no browser — the inventory and refund specs
npm run test:ui           # the admin UI specs
npm run cleanup           # reap anything a crashed run left behind
```

## Prerequisite: the OTP short-circuit

A refund needs a paid order, a paid order needs a customer session, and a
customer session needs an OTP. The suite must send **zero** real SMS — MSG91 is
the only integration in the codebase that costs money per call.

So `SendOTP` short-circuits above the gateway for an exact-match E.164
allowlist, storing the hash of a fixed code instead of a random one:

```
STORE_TEST_PHONES=+919999900001     # exact match, never a prefix
STORE_TEST_OTP=<secret>
```

Everything downstream — hashing, TTL, attempt limits, verification — is the
production path, untouched. **The deployment under test must have both set**, or
`otp/verify` rejects and every refund spec fails.

Three guards keep it out of production:

1. `config.Validate()` refuses to start when either is set and the environment
   is prod. The Lambda dies on cold start.
2. CDK never sets the variables on the prod stack at all, so console or SSM
   drift cannot enable it without a code deploy.
3. Exact match on the full E.164 string, and an empty `STORE_TEST_OTP` disables
   the bypass regardless of the allowlist — a half-configured environment fails
   closed.

## Why the suite creates its own products

It never touches real catalog data. `AdjustStock` sets `quantity` without ever
writing `reserved_qty`, so compensating arithmetic through the stock API carries
any leak forward and would force falsifying the ledger. Deleting a product is a
real row delete that cascades `inventory` and `inventory_transactions`, so
deleting it deletes any leak with it. Restoration is exact and needs no
arithmetic.

Everything created is named `E2E-<runId>-…`. `scripts/cleanup.ts` keys on that
prefix, is idempotent, and runs as a CI post-step with `if: always()` because a
crashed spec skips its own teardown.

### What cannot be undone

- **Orders.** `OrderRepository` has no `Delete` at all. Cancel is the only
  lever, and it is needed anyway to release the reservation. Orders accumulate
  on the test customer permanently.
- **`customer.OrderCount` / `TotalSpent`.** Atomic DynamoDB `ADD`, with no
  decrement anywhere in the codebase.
- **Analytics.** Every run writes `orders_placed`, `payment_completed` and
  friends into `metric_counters`, which feeds the dev dashboards. Dev funnel
  figures are skewed by CI traffic. Accepted; no suppression is built.

## Layout

```
fixtures/    run-id, authenticated API clients, catalog create/destroy
helpers/     order placement (admin path and paid storefront path)
pages/       page objects; third-party DOM confined to one file each
specs/
  inventory/ duplicate lines, aggregate rejection, replay safety, cancel
  refunds/   second refund, cancel-partly-refunded, RBAC, preview, ledger
  admin-ui/  refund button visibility, ledger rendering
scripts/     cleanup
```

`api` specs are pure HTTP and need no browser; `admin-ui` specs drive Chromium.
The projects are separate so the cheap tier can run on its own.

## Coverage against #230

Fully covered: 1, 2, 3, 6, 8, 9, 10, 11, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
27, 28, 29, 30, 31, 32, 33, 34, 35, and the `limit` half of 14.

Covered elsewhere, deliberately not duplicated here:

- **13, 14** — reconciliation is not an endpoint or a UI panel. It is
  `scripts/reconcile-inventory`, a weekly Go CLI driven by
  `.github/workflows/inventory-reconciliation.yml` that exits non-zero when
  stock is held against nothing. Its semantics are pinned in
  `internal/repository/postgres/` against a disposable database, which is
  deterministic in a way a shared dev environment is not.
- **11, 33** — the migrations run once at deploy and cannot be re-applied. What
  is asserted is the invariant they leave behind: that the unique index carries
  `source_id`, that `source_id` is `NOT NULL` so nulls cannot defeat it, that a
  same-quantity replay is a no-op, that a different-quantity replay is a
  conflict, and that two refunds on one order both keep their stock movement.

Not covered, with the reason stated in the spec rather than omitted:

- **36** — a crash between the two datastores. See
  `specs/inventory/cross-store-gap.spec.ts`: a deployed Lambda cannot be killed
  mid-invocation, and adding a fault-injection seam would put a crash path into
  production code. The *detection* half is covered by reconciliation.
- **12, 15, 16** are browser specs and run only in the `admin-ui` project.

## Known divergence from #223

#223 Tier 1.1 says the refund **list** must stay readable by an `OPERATOR` or
their order page breaks on a 403. #232 moved `GET /admin/orders/{id}/refunds`
inside the admin group — "admin-only end to end, the read included".
`specs/refunds/rbac.spec.ts` asserts what #232 actually does. If the operator
order page is meant to show refunds, that is a product decision to settle on
#232, not a test to soften.

## Tracked defects

`test.fixme` with a ticket link, so they are visible rather than absent, and
flip to passing when fixed:

- **#222** — cancelling a paid order refunds nothing, including the remainder of
  a partially refunded one. Stock returns; the customer's money does not.
