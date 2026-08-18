# Inventory Integrity — Deferred Work

**Context:** `fix/inventory-lifecycle` added the missing `CommitStock` operation and
hooked it at `SHIPPED`, restocked at `RETURNED`, aligned the two cancel paths, threaded
the order ID into the reservation ledger, and made failed inventory mutations alertable.

Two independent reviews of that branch found defects it does not address. The blocking
ones were fixed on the branch. This document tracks the rest so they are not lost.

**Deployment context that shapes the priorities below:** production holds no
inventory, orders, or products, and dev inventory data is disposable. Anything whose
damage depended on pre-existing data is therefore not urgent. Anything reachable from
an empty database on day one still is.

---

## 1. Order-scoped inventory mutations are not idempotent

**Severity:** high — the largest remaining correctness gap.

`reserved_qty` is a single aggregate per product (`inventory.product_id` is `UNIQUE`).
`ReleaseStock` and `CommitStock` guard only on that total:

```go
if reservedQty < quantity || currentQty < quantity {
```

Neither verifies that *this* `orderID` actually holds the quantity being consumed.
`reference_id` is written to `inventory_transactions` on every mutation but is **never
read back** — no query in the repo filters on it.

Consequence: a repeated or spurious commit/release silently consumes a *different*
order's reservation whenever the product has other open orders. The guard only fires
when the product has exactly one outstanding order — which is precisely the case the
unit tests cover, and precisely the case that does not matter.

Concrete failure. `prod_A`: `quantity=10, reserved_qty=5` (order X holds 2, order Y holds 3).

1. `UpdateStatus(X, SHIPPED)` runs twice — a retried request, or two admins.
2. First commit: `reserved 5→3`, `quantity 10→8`.
3. Second commit: `reserved(3) >= 2` and `quantity(8) >= 2`, so it **passes**.
   `reserved 3→1`, `quantity 8→6`.
4. Order X decremented twice; order Y's reservation eaten.
5. Y ships later → `reserved(1) < 3` → refused → swallowed → Y goes `SHIPPED` with no
   decrement, and `reserved_qty` is stranded at 1 permanently.

Two triggers make step 1 realistic, neither exotic:

- `OrderRepository.GetByID` is a DynamoDB `GetItem` with **no `ConsistentRead`**, so a
  client-timeout-then-retry on a request that actually succeeded re-reads the old status
  and re-runs the whole transition.
- `orderRepo.Update` is an unconditional `PutItem` guarded only by
  `attribute_exists(PK)` — no optimistic concurrency on status, so two in-flight
  requests never collide.

`FOR UPDATE` serialises the two commits correctly. It does not make them idempotent.

**Fix shape:** inside the mutation's existing transaction, after `FOR UPDATE`,
short-circuit on an existing ledger row:

```sql
SELECT 1 FROM inventory_transactions
WHERE product_id = $1 AND reference_id = $2 AND type = $3
```

Back it with a unique partial index on `(product_id, reference_id, type)` for the
order-scoped types, so it is enforced rather than merely checked. The original plan named
this as the real fix and scoped it out; it turned out to be load-bearing for the
operation that plan added.

## 2. Multi-item orders mutate stock per item, with no rollback

**Severity:** high — depends on 1 for a clean fix.

`applyInventoryEffect` iterates items independently and swallows per-item errors. An
order with `prod_A` and `prod_B` where A commits and B fails leaves the order `SHIPPED`,
A committed, B's reservation stranded, and nothing recording which is which.

Worse, there is no recovery path: from `SHIPPED`, `validTransitions` allows only
`DELIVERED` (no inventory effect) and `RETURNED` (`AddStock`, which never touches
`reserved_qty`). Those units are unrecoverable by any reachable operation.

All items of one order live in the same PostgreSQL database, so the fix is a repository
method taking the whole set in one transaction — `CommitOrderStock(ctx, orderID, items)`
— all-or-nothing per order rather than per item. That also collapses item 1's
idempotency check into a single guarded operation.

## 3. Returns reuse `AddStock`, corrupting `last_restock_at`

**Severity:** medium.

`AddStock` unconditionally stamps `last_restock_at = now`. A customer return is not a
supplier replenishment, but it now overwrites that field, which is exposed through the
inventory API. "When did we last replenish this SKU" becomes wrong for any returned
product, and any reorder logic keyed off it is misled.

The original plan weighed one trade-off of reusing `AddStock` — returns land in the
ledger as type `ADD`, distinguishable only by the reason prefix — but not this one. A
dedicated return operation fixes both, and gives the distinct ledger type that a
commit-aware restock (below) needs.

## 4. `RETURNED` restock assumes a commit that may not have happened

**Severity:** medium *given an empty database*; would be high with live data.

`AddStock(+q)` is unconditional and assumes the matching `−q` already happened. The code
comment states the assumption — "RETURNED is reachable only from SHIPPED/DELIVERED, both
post-commit" — but the commit is deliberately best-effort, so *post-SHIPPED* is not
*post-commit*. If the commit failed (transient error, or a corrupted row rejected by the
guard), the return inflates stock: `quantity` rises with no offsetting decrement.

Note the direction. Before this branch a failed commit merely stranded stock, which
under-sells. Now it can inflate, which over-sells and produces orders that cannot be
fulfilled.

The branch closed the *retry* route into this state by moving stock movement after the
order write. The remaining route is a genuinely failed commit.

**Fix:** gate the restock on the commit having happened — restock only what was actually
committed for that order, read from `COMMIT` ledger rows for `reference_id = order.ID`.
Depends on item 1's ledger indexing and reads better with item 3's distinct type.

There was also a deploy-time variant of this — orders already in `SHIPPED`/`DELIVERED`
from before the fix, shipped without a decrement, would inflate on return. **Not
applicable here:** no such orders exist in any environment.

## 5. Frontend `TransactionType` union is missing `'COMMIT'`

**Status:** landed in PR #186 (`fix/inventory-stock-lifecycle`). Kept for the record.

**Severity:** low, but `CLAUDE.md` requires frontends to be updated in the same PR as the
API change.

```ts
// handloom-admin-frontend/src/features/inventory/types.ts:1
export type TransactionType = 'ADD' | 'REMOVE' | 'RESERVE' | 'RELEASE' | 'ADJUST';
```

The backend now returns `"COMMIT"` through `getInventoryTransactions`. No component
switches on the value today, so nothing breaks at runtime — the type is simply false.

## 6. `inventory_mutation_failed` is observable but not actionable

**Severity:** low.

The metric fires with a bounded `reason` label (`commit`/`release`/`restock`), which is
correct for cardinality — but it names neither the product nor the order, so recovery
means correlating a counter against `slog` lines by timestamp. There is no reconciliation
query and no repair job.

Threading the order ID into the ledger made per-order reconciliation *possible*; nothing
consumes it yet. A reconciliation query in the runbook, or an audit row emitted alongside
the metric, would close this.

## 7. The ledger has no admin UI

**Severity:** low as correctness, medium as operability — it is the only way to see
inventory history without a psql session.

`inventory_transactions` is the single source of truth for how a product's stock reached
its current value, and today nothing renders it. `InventoryPage.tsx` shows balances only.
Diagnosing drift means opening a SQL client against dev, or Neon against production.

The backend and the client method already exist and are unused:

| Piece | Location | State |
|---|---|---|
| Endpoint | `GET /admin/products/{id}/inventory/transactions` | live, paginated, admin-auth (`product_handler.go:52`) |
| Service | `InventoryService.GetTransactions` | live |
| API client | `productsApi.getInventoryTransactions` | defined, **zero call sites** (`features/products/api.ts:81`) |
| Route constant | `ROUTES.PRODUCTS.INVENTORY_TRANSACTIONS` | defined (`shared/constants/routes.ts:72`) |
| Types | `InventoryTransaction`, `TransactionType` | defined, `COMMIT` included since #186 |
| Component | — | **missing — this is the whole task** |

So the work is a table component plus wiring, not new plumbing.

**Shape:** a history panel on the product detail page, or a drawer from `InventoryPage`
rows. Columns `created_at`, `type`, `quantity`, `previous_qty` → `new_qty`,
`reference_id`, `reason`, `created_by`. Existing `Table` and `Modal` from
`components/common/` cover it; React Query for paging, same as every other list.

**Worth getting right:**

- Link `reference_id` to the order when `reference_type` is `ORDER`. That link is what
  turns the table into a diagnostic rather than a log dump.
- Label the columns per type. `PreviousQty`/`NewQty` track `reserved_qty` for
  `RESERVE`/`RELEASE` and `quantity` for `ADD`/`REMOVE`/`ADJUST`/`COMMIT` — see the
  pre-existing-issues table below. A single "Before → After" header silently means two
  different things.
- A `RESERVE` with no later `COMMIT` or `RELEASE` is the drift signature. Surfacing it
  visually is most of item 6's value without building a reconciliation job.

**Depends on:** nothing. Can ship independently of items 1–4.

---

## Pre-existing, not caused by this branch

Found during review, present on `main`, untouched here. Listed so they are on the record.

| Issue | Note |
|---|---|
| `AdjustStock` can drive `available_qty` negative | Sets `quantity` without touching `reserved_qty`; a stocktake to a value below `reserved_qty` yields negative `available_qty`. Unrepairable through the API — `InventoryRepository.Update` has no service or handler call site, so only manual SQL fixes `reserved_qty`. |
| No optimistic concurrency on order writes | Unconditional `PutItem`; status transitions are a TOCTOU against a non-consistent read. Root enabler of item 1. |
| Webhook idempotency is read-then-write | `resolvePayment` compares status, then `UpdateStatus` writes with no status precondition. Duplicate PhonePe deliveries can both pass. No dedup on event ID either. |
| `HandlePaymentFailure` leaves the order actionable | Releases stock but leaves the order `PENDING`/`PENDING`, so it still appears as a normal order an admin can confirm and ship — against a reservation it no longer holds. |
| `OrderStatusHistory` is never written | The entity is defined; there are zero call sites. No transition audit trail exists. |
| No reaper for abandoned reservations | A `PENDING` order holds its reservation forever. After this branch, the only remaining permanent-leak path. |
| Checkout is not idempotent | N rapid Pay Now clicks create N orders and N reservations. |
| Admin `Create` swallows reservation failures | The availability check and the reservation are far apart, and a failed reservation still returns a created order. |
| `TestSearcher_Hybrid_ReturnsKnownProduct` fails | `cmd/embedder/embedder`. Pre-existing on `origin/main`, unrelated to inventory. Now the only red test in the suite. |
| `PreviousQty`/`NewQty` semantics differ by ledger type | `RESERVE`/`RELEASE` track `reserved_qty`; `ADD`/`REMOVE`/`ADJUST`/`COMMIT` track `quantity`. |

## No longer applicable

`docs/superpowers/runbooks/inventory-reserved-qty-reset.md` existed to reset reservations
leaked by the pre-fix code. With no orders in production and disposable dev data, there
was nothing to reset and never would be — the leak it repaired is historical. Deleted.

## Suggested order

1. Ledger-keyed idempotency (item 1) — unblocks 2 and 4.
2. One transaction per order (item 2).
3. Commit-gated restock (item 4) and a dedicated return operation (item 3), together.
4. Frontend union (item 5) — done in #186.
5. Ledger view in the admin UI (item 7) — independent, ships any time; makes item 6
   largely unnecessary if the drift signature is visible in the table.
6. Reconciliation query (item 6).
