import { describe, expect, it } from 'vitest';

import type { InventoryTransaction, TransactionType } from '@/features/inventory/types';

import { openReservationIDs } from '../openReservations';

function row(
  type: TransactionType,
  referenceID?: string,
  referenceType = 'ORDER'
): InventoryTransaction {
  return {
    id: `${type}-${referenceID ?? 'none'}`,
    product_id: 'prod_1',
    type,
    quantity: 1,
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
