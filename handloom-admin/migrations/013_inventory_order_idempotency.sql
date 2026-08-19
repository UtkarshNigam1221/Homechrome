-- Order-scoped inventory movements must be idempotent: Release/CommitStock guarded
-- only on the reserved_qty total, so a repeat consumed another order's reservation.

-- This index makes one movement per (product, order, type) an enforced invariant,
-- backstopping the repository check against races.

-- Supports the dedup self-join below. idx_inv_txn_product_time is
-- (product_id, created_at DESC), which cannot serve this predicate.
CREATE INDEX IF NOT EXISTS idx_inv_txn_order_dedup
  ON inventory_transactions (product_id, reference_id, type)
  WHERE reference_type = 'ORDER';

-- Duplicates would block the unique index. Copied out before deletion, not
-- discarded: a duplicate RELEASE did move the balance, so this is the evidence.
CREATE TABLE IF NOT EXISTS inventory_transactions_dedup_013 (
  LIKE inventory_transactions INCLUDING DEFAULTS INCLUDING INDEXES
);

WITH doomed AS (
  -- DISTINCT: in a group of three, the last row matches both earlier rows and
  -- would otherwise be copied twice.
  SELECT DISTINCT a.id
  FROM inventory_transactions a
  JOIN inventory_transactions b
    ON  a.product_id   = b.product_id
    AND a.reference_id = b.reference_id
    AND a.type         = b.type
    AND b.reference_type = 'ORDER'
  WHERE a.reference_type = 'ORDER'
    AND (a.created_at, a.id) > (b.created_at, b.id)
)
INSERT INTO inventory_transactions_dedup_013
SELECT t.* FROM inventory_transactions t JOIN doomed d ON d.id = t.id;

-- No RAISE NOTICE here: the migrator's pool sets no OnNotice hook, so pgx drops
-- notices. Count with: SELECT count(*) FROM inventory_transactions_dedup_013;

DELETE FROM inventory_transactions t
USING inventory_transactions_dedup_013 d
WHERE t.id = d.id;

-- Takes ACCESS EXCLUSIVE for the build, blocking reserve/commit inserts. Fine at
-- current volume; on a large table build it manually with CONCURRENTLY first.
CREATE UNIQUE INDEX IF NOT EXISTS idx_inv_txn_order_scoped
  ON inventory_transactions (product_id, reference_id, type)
  WHERE reference_type = 'ORDER';

-- Same columns and predicate as the unique index above, so it is pure write
-- overhead once that exists. It only had to support the dedup delete.
DROP INDEX IF EXISTS idx_inv_txn_order_dedup;
