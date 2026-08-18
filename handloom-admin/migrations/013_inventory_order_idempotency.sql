-- Order-scoped inventory movements must be idempotent.
--
-- ReleaseStock and CommitStock guarded only on the product-level reserved_qty
-- total, never on which order held it, so a repeated commit silently consumed a
-- different order's reservation. reference_id was written on every order-scoped
-- mutation but never read back. This index makes "one movement of each type per
-- product per order" an enforced invariant rather than a convention, so the
-- application-level check in the repository has a backstop it cannot race.

-- Existing duplicates would block the index. The known source is a release that
-- ran twice for one order — the payment-failure rollback followed by a later
-- cancel — which was benign but does violate the rule from here on. Keep the
-- earliest row of each group; the later ones recorded no additional movement
-- that the balance did not already reflect.
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
