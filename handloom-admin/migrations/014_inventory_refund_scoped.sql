-- A refund moves stock for an order that has already moved it.
--
-- Migration 013 made "one movement of each type per product per order" an
-- enforced invariant. That is right for the order lifecycle — an order reserves
-- once, dispatches once, releases once — but a refund is not part of it. An
-- order can be refunded line by line over days, so the second refund of the same
-- product was deduped into the first: the money went back and the stock never
-- moved.
--
-- source_id names what caused the movement when that is something other than the
-- order itself. The reference stays the order, so the ledger and the
-- orphan-reservation pairing still see one story per order.
--
-- NOT NULL DEFAULT '' rather than nullable: Postgres treats NULLs as distinct in
-- a unique index, so a nullable column would quietly stop deduplicating the
-- order-lifecycle movements this index exists to protect.
ALTER TABLE inventory_transactions
  ADD COLUMN IF NOT EXISTS source_id text NOT NULL DEFAULT '';

DROP INDEX IF EXISTS idx_inv_txn_order_scoped;

CREATE UNIQUE INDEX IF NOT EXISTS idx_inv_txn_order_scoped
  ON inventory_transactions (product_id, reference_id, type, source_id)
  WHERE reference_type = 'ORDER';
