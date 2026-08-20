import { describe, expect, it } from 'vitest';

import { claimedByLiveRefunds, unrefundedQuantity } from '@/features/orders/lib/refundClaims';

const line = (id: string, quantity: number, refunded = 0) => ({
  id,
  quantity,
  refunded_quantity: refunded,
});

const refund = (status: string, amount: number, items: [string, number][]) => ({
  status,
  amount,
  items: items.map(([order_item_id, quantity]) => ({ order_item_id, quantity })),
});

describe('unrefundedQuantity', () => {
  it('is what the line has left to refund', () => {
    expect(unrefundedQuantity(line('a', 3, 1))).toBe(2);
  });

  it('never goes below zero', () => {
    expect(unrefundedQuantity(line('a', 1, 3))).toBe(0);
  });

  it('counts units a refund still in flight has claimed', () => {
    expect(unrefundedQuantity(line('a', 3), { a: 2 })).toBe(1);
  });

  it('takes whichever claim is larger, settled or in flight', () => {
    expect(unrefundedQuantity(line('a', 5, 3), { a: 1 })).toBe(2);
    expect(unrefundedQuantity(line('a', 5, 1), { a: 3 })).toBe(2);
  });
});

describe('claimedByLiveRefunds', () => {
  it('counts a pending refund against the lines it names', () => {
    const { claims, amount } = claimedByLiveRefunds([refund('PENDING', 1000, [['a', 1]])]);

    expect(claims).toEqual({ a: 1 });
    expect(amount).toBe(1000);
  });

  it('frees the units of a refund that failed', () => {
    const { claims, amount } = claimedByLiveRefunds([refund('FAILED', 1000, [['a', 2]])]);

    expect(claims).toEqual({});
    expect(amount).toBe(0);
  });

  it('sums across refunds and lines', () => {
    const { claims, amount } = claimedByLiveRefunds([
      refund('COMPLETED', 1000, [['a', 1]]),
      refund('PENDING', 500, [
        ['a', 1],
        ['b', 2],
      ]),
    ]);

    expect(claims).toEqual({ a: 2, b: 2 });
    expect(amount).toBe(1500);
  });

  // A settlement that half-completed leaves the payment ahead of the rows, and the
  // remainder has to be the pessimistic reading of the two.
  it('takes the payment figure when it is ahead of the rows', () => {
    expect(claimedByLiveRefunds([refund('PENDING', 1000, [['a', 1]])], 2500).amount).toBe(2500);
    expect(claimedByLiveRefunds([refund('PENDING', 3000, [['a', 1]])], 2500).amount).toBe(3000);
  });
});
