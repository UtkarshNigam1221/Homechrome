-- Order-scoped inventory movements must be idempotent: Release/CommitStock guarded
-- only on the reserved_qty total, so a repeat consumed another order's reservation.

-- This index makes one movement per (product, order, type) an enforced invariant,
-- backstopping the repository check against races.

-- Existing duplicates would block the index; the known source is a double release
-- (payment-failure rollback, then cancel). Keep the earliest row of each group.
DELETE FROM inventory_transactions a
USING inventory_transactions b
WHERE a.reference_type = 'ORDER'
  AND b.reference_type = 'ORDER'
  AND a.product_id = b.product_id
  AND a.reference_id = b.reference_id
  AND a.type = b.type
  AND (a.created_at, a.id) > (b.created_at, b.id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_inv_txn_order_scoped
  ON inventory_transactions (product_id, reference_id, type)
  WHERE reference_type = 'ORDER';
