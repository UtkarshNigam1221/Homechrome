import { describe, expect, it } from 'vitest';

import type { InventoryTransaction, TransactionType } from '@/features/inventory/types';

import { balancesAfter, movementEffect, recordedCounter } from '../movementEffects';

function row(type: TransactionType, quantity: number, prev = 0, next = 0): InventoryTransaction {
  return {
    id: 'x',
    product_id: 'prod_1',
    type,
    quantity,
    previous_qty: prev,
    new_qty: next,
    created_by: '',
    created_at: '2026-08-19T00:00:00Z',
  };
}

describe('movementEffect', () => {
  // An omitted counter renders as a dash, so absence is the meaningful assertion.
  it('adds to stock on hand for a restock and a return, leaving reserved alone', () => {
    expect(movementEffect(row('ADD', 5)).onHand).toBe(5);
    expect(movementEffect(row('ADD', 5)).reserved).toBeUndefined();
    expect(movementEffect(row('RETURN', 3)).onHand).toBe(3);
    expect(movementEffect(row('RETURN', 3)).reserved).toBeUndefined();
  });

  it('moves only the reservation for reserve and release', () => {
    expect(movementEffect(row('RESERVE', 4)).reserved).toBe(4);
    expect(movementEffect(row('RESERVE', 4)).onHand).toBeUndefined();
    expect(movementEffect(row('RELEASE', 4)).reserved).toBe(-4);
    expect(movementEffect(row('RELEASE', 4)).onHand).toBeUndefined();
  });

  // The ledger row records only the on-hand pair for a dispatch, but reserved_qty
  // falls by the same amount — that is why available_qty is unchanged.
  it('reports a dispatch against both counters', () => {
    expect(movementEffect(row('COMMIT', 3, 50, 47))).toMatchObject({ onHand: -3, reserved: -3 });
  });

  // A recount is the one type whose direction the type name does not imply.
  it('takes a recount direction from the before/after pair', () => {
    expect(movementEffect(row('ADJUST', 2, 50, 48)).onHand).toBe(-2);
    expect(movementEffect(row('ADJUST', 2, 48, 50)).onHand).toBe(2);
  });

  // A write-off moves the same two counters a dispatch does, so the only thing
  // separating them in the ledger is the label.
  it('reports a write-off against both counters, but never calls it a dispatch', () => {
    const effect = movementEffect(row('WRITE_OFF', 2, 50, 48));

    expect(effect).toMatchObject({ onHand: -2, reserved: -2 });
    expect(effect.label).toBe('Written off');
  });

  it('names the counter the recorded before/after pair belongs to', () => {
    expect(recordedCounter('RESERVE')).toBe('reserved');
    expect(recordedCounter('RELEASE')).toBe('reserved');
    expect(recordedCounter('COMMIT')).toBe('onHand');
    expect(recordedCounter('ADD')).toBe('onHand');
    expect(recordedCounter('WRITE_OFF')).toBe('onHand');
  });
});

describe('balancesAfter', () => {
  // Newest first, as the API returns them. Read oldest to newest the history is:
  // reserve 3 against 50 on hand, dispatch those 3, then recount 47 down to 45.
  const history = [row('ADJUST', 2, 47, 45), row('COMMIT', 3, 50, 47), row('RESERVE', 3, 0, 3)];

  it('reconstructs both counters at every step', () => {
    const balances = balancesAfter(history, { onHand: 45, reserved: 0 }, true);

    expect(balances[0]).toEqual({ onHand: 45, reserved: 0 });
    // The dispatch cleared the reservation as it took the stock.
    expect(balances[1]).toEqual({ onHand: 47, reserved: 0 });
    // Before it, the 3 units were held against untouched stock — the figure the
    // row itself never records.
    expect(balances[2]).toEqual({ onHand: 50, reserved: 3 });
  });

  it('gives up on a row whose recorded figure disagrees', () => {
    const balances = balancesAfter(history, { onHand: 999, reserved: 0 }, true);
    expect(balances[0]).toBeNull();
  });

  it('derives nothing once the anchor is not the newest movement', () => {
    expect(balancesAfter(history, { onHand: 45, reserved: 0 }, false)).toEqual([null, null, null]);
  });

  // Why the recorded-counter check cannot replace the anchor: a reserve and its
  // release net to zero on reserved, so it passes while the derived one is invented.
  it('does not report a reserved figure invented by a stale anchor', () => {
    // Page 2 holds one ADD. Newer, unseen: a RESERVE of 5 still outstanding.
    const page2 = [row('ADD', 10, 40, 50)];
    const currentWithNewerReserve = { onHand: 50, reserved: 5 };

    // Reserved right after that ADD was 0, not 5. Unanchored, nothing is claimed.
    expect(balancesAfter(page2, currentWithNewerReserve, false)).toEqual([null]);
  });
});

// The reconstruction walks back from the product's current levels, so it is only
// meaningful while the newest row shown is the newest movement.
describe('balancesAfter anchoring', () => {
  const history = [row('ADJUST', 2, 47, 45), row('COMMIT', 3, 50, 47)];

  it('reconstructs nothing when the page is not the newest', () => {
    expect(balancesAfter(history, { onHand: 45, reserved: 0 }, false)).toEqual([null, null]);
  });

  it('still reconstructs on the newest page', () => {
    expect(balancesAfter(history, { onHand: 45, reserved: 0 }, true)[0]).toEqual({
      onHand: 45,
      reserved: 0,
    });
  });
});
