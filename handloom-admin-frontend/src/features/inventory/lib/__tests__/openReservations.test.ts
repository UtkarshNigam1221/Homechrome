import { describe, expect, it } from 'vitest';

import type { InventoryTransaction, TransactionType } from '@/features/inventory/types';

import { openReservationIDs } from '../openReservations';

function row(
  type: TransactionType,
  referenceID?: string,
  referenceType = 'ORDER',
  quantity = 1
): InventoryTransaction {
  return {
    id: `${type}-${referenceID ?? 'none'}-${quantity}`,
    product_id: 'prod_1',
    type,
    quantity,
    previous_qty: 0,
    new_qty: 1,
    reference_type: referenceType,
    reference_id: referenceID,
    created_by: '',
    created_at: '2026-08-19T00:00:00Z',
  };
}

describe('openReservationIDs', () => {
  it('flags a reservation with no dispatch or release', () => {
    expect(openReservationIDs([row('RESERVE', 'order_1')])).toEqual(new Set(['order_1']));
  });

  it('clears a reservation the same order dispatched', () => {
    const rows = [row('RESERVE', 'order_1'), row('COMMIT', 'order_1')];
    expect(openReservationIDs(rows).size).toBe(0);
  });

  // A refunded line's goods are written off, not dispatched and not released.
  // Leaving WRITE_OFF out flagged every fully refunded order as stock stuck in
  // limbo — false alarms in the one report that exists to be believed.
  it('clears a reservation the same order wrote off', () => {
    const rows = [row('RESERVE', 'order_1'), row('WRITE_OFF', 'order_1')];
    expect(openReservationIDs(rows).size).toBe(0);
  });

  it('clears a reservation the same order released', () => {
    const rows = [row('RESERVE', 'order_1'), row('RELEASE', 'order_1')];
    expect(openReservationIDs(rows).size).toBe(0);
  });

  it('does not let one order settle another', () => {
    const rows = [row('RESERVE', 'order_1'), row('COMMIT', 'order_2')];
    expect(openReservationIDs(rows)).toEqual(new Set(['order_1']));
  });

  it('ignores movements that are not order-scoped', () => {
    const rows = [row('ADD', undefined, 'USER'), row('ADJUST', undefined, 'USER')];
    expect(openReservationIDs(rows).size).toBe(0);
  });

  it('is order-independent, so a page starting mid-history still settles', () => {
    const newestFirst = [row('COMMIT', 'order_1'), row('RESERVE', 'order_1')];
    expect(openReservationIDs(newestFirst).size).toBe(0);
  });
});

// Presence-based settlement called a partly-released order settled, which is the
// drift the report exists to catch.
describe('openReservationIDs partial settlement', () => {
  it('still flags an order that released only some of what it holds', () => {
    const rows = [row('RESERVE', 'order_1', 'ORDER', 3), row('RELEASE', 'order_1', 'ORDER', 1)];

    expect(openReservationIDs(rows)).toEqual(new Set(['order_1']));
  });

  it('clears an order once every unit is accounted for', () => {
    const rows = [
      row('RESERVE', 'order_1', 'ORDER', 3),
      row('RELEASE', 'order_1', 'ORDER', 1),
      row('WRITE_OFF', 'order_1', 'ORDER', 2),
    ];

    expect(openReservationIDs(rows).size).toBe(0);
  });
});
