# Inventory reconciliation — stranded reservations

**Signal:** `inventory_orphan_reserved_units` above zero, or the log line
`Reservations held with no dispatch or release`.

## What it means

A reservation is settled when the same order either dispatches the stock
(`COMMIT`) or gives it back (`RELEASE`). One that does neither is stock held
against an order that never shipped and never cancelled. Those units are
unsellable, and **no order transition will free them** — from `SHIPPED` the only
routes are `DELIVERED`, which has no inventory effect, and `RETURNED`, which
adds stock without touching `reserved_qty`.

Note what this is not: `inventory_mutation_failed` says a movement failed.
This says what is actually stuck right now, whether or not anyone saw the
failure that caused it.

## Check

```
GET /admin/inventory/reconciliation?min_age=24h
```

```json
{
  "order_count": 1,
  "stranded_units": 4,
  "reservations": [
    { "product_id": "prod_x", "sku": "SKU-1", "order_id": "order_dd90",
      "quantity": 4, "reserved_at": "2026-08-17T00:57:28+05:30" }
  ]
}
```

`min_age` defaults to 24h. Reservations younger than that are usually customers
mid-payment, not drift. The result is capped at 500 orders; hitting the cap is
logged as `truncated`, and means the problem is systemic rather than a handful
of stuck orders.

Per product, the same thing is visible in the admin UI: Inventory → the product's
**Stock history**, where an unsettled reservation is badged **Still held**. That
view only compares the page in front of you, so use the endpoint for the real
answer.

## Diagnose before fixing

Find what happened to the order:

```sql
SELECT type, quantity, previous_qty, new_qty, created_at
FROM inventory_transactions
WHERE reference_id = 'order_dd90' AND reference_type = 'ORDER'
ORDER BY created_at;
```

Then check the order itself. Three cases:

| Order status | Meaning | Action |
|---|---|---|
| `PENDING` | Checkout never completed; no reaper exists yet | Release the reservation |
| `CANCELLED` | The release failed at the time | Release the reservation |
| `SHIPPED`/`DELIVERED` | The dispatch failed, so stock was never decremented | **Do not release** — commit it, or the units come back twice |

That last row is the one to be careful about. Releasing a reservation whose
order actually shipped returns stock that physically left the warehouse.

## Fix

Releasing is idempotent per `(product_id, order_id)` and writes a ledger row, so
it is safe to retry and leaves a trace:

```
POST /admin/orders/{order_id}/cancel
```

for an order that should be cancelled. There is deliberately no bulk "release
all orphans" endpoint: each case needs the status check above, and a blanket
release would silently corrupt the `SHIPPED` case.

## If the number keeps growing

Recurring drift is not something to keep clearing by hand. The usual causes:

- No reaper for abandoned `PENDING` checkouts — the known permanent-leak path
- A failing `ReleaseStock` on the payment-failure rollback, which would also be
  raising `inventory_mutation_failed` with reason `release`

`release_unreserved` is not this. That reason means a release found nothing left
to release, which is the benign double-release case and does not strand stock.
