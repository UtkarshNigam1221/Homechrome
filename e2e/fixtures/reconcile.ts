import { APIRequestContext, expect } from '@playwright/test';

import { getInventory, getLedger, InventoryTransaction } from './api';

/**
 * #230 case 35 — the invariant the whole stack exists to protect, and its top
 * priority: replaying every ledger movement for a product must land on the live
 * balance.
 *
 * Deliberately a shared helper rather than an assertion inside one spec. Any
 * scenario can break the ledger, so every scenario should check it; asserting
 * it in the one spec that happens to think about it is how the invariant rots.
 */

/** How each ledger type moves the two counters. Mirrors movementEffects.ts. */
function delta(row: InventoryTransaction): { onHand: number; reserved: number } {
  const q = row.quantity;
  switch (row.type) {
    case 'ADD':
    case 'RETURN':
      return { onHand: +q, reserved: 0 };
    case 'REMOVE':
      return { onHand: -q, reserved: 0 };
    case 'RESERVE':
      return { onHand: 0, reserved: +q };
    case 'RELEASE':
      return { onHand: 0, reserved: -q };
    // A dispatch and a write-off are arithmetically identical: both consume the
    // reservation and the goods together, so available_qty does not move.
    case 'COMMIT':
    case 'WRITE_OFF':
      return { onHand: -q, reserved: -q };
    // ADJUST is a stocktake: it sets quantity outright rather than moving it,
    // so it cannot be replayed as a delta. Handled by the caller below.
    case 'ADJUST':
      return { onHand: 0, reserved: 0 };
  }
}

/**
 * Replays the ledger and asserts it agrees with live inventory.
 *
 * Replay needs an opening balance because the ledger does not contain one:
 * ProductService.Create writes the inventory row with Quantity = InitialStock
 * and no transaction to match, so the first stock a product ever has is
 * invisible to the ledger. Summing deltas from zero therefore lands
 * `initialStock` short for every product in the system — #230 case 35 as
 * literally worded ("SUM(ledger deltas) equals inventory.quantity") cannot hold
 * until opening stock is ledgered.
 *
 * reserved_qty needs no such treatment: it genuinely starts at zero and every
 * change to it is a movement.
 *
 * A stocktake is the other discontinuity — ADJUST sets quantity outright rather
 * than moving it — so replay restarts from the newest one when there is one.
 */
export async function expectLedgerBalances(
  api: APIRequestContext,
  product: { id: string; initialStock: number },
  context = ''
): Promise<void> {
  const productId = product.id;
  const [inventory, ledger] = await Promise.all([
    getInventory(api, productId),
    getLedger(api, productId),
  ]);

  const ordered = ledger
    .slice()
    .sort((a, b) => a.created_at.localeCompare(b.created_at) || a.id.localeCompare(b.id));

  const lastAdjust = ordered.map((r) => r.type).lastIndexOf('ADJUST');
  const replayFrom = lastAdjust >= 0 ? lastAdjust : 0;

  let onHand = lastAdjust >= 0 ? ordered[lastAdjust]!.new_qty : product.initialStock;
  let reserved = 0;

  for (const row of ordered.slice(lastAdjust >= 0 ? replayFrom + 1 : 0)) {
    const d = delta(row);
    onHand += d.onHand;
    reserved += d.reserved;
  }

  const where = context ? ` (${context})` : '';
  expect(onHand, `ledger replay must equal on-hand for ${productId}${where}`).toBe(
    inventory.quantity
  );
  expect(reserved, `ledger replay must equal reserved for ${productId}${where}`).toBe(
    inventory.reserved_qty
  );
  expect(
    inventory.available_qty,
    `available_qty must be quantity - reserved_qty for ${productId}${where}`
  ).toBe(inventory.quantity - inventory.reserved_qty);
  expect(
    inventory.available_qty,
    `available_qty must never go negative for ${productId}${where}`
  ).toBeGreaterThanOrEqual(0);
}

/** Convenience for a whole fixture's products. */
export async function expectAllLedgersBalance(
  api: APIRequestContext,
  products: { id: string; initialStock: number }[],
  context = ''
): Promise<void> {
  for (const product of products) {
    await expectLedgerBalances(api, product, context);
  }
}
