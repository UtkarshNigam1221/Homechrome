import { describe, expect, it } from 'vitest';

import { ALLOWED_TRANSITIONS, FORWARD_STATUSES, ORDER_STATUSES } from '@/features/orders/types';

describe('ALLOWED_TRANSITIONS', () => {
  // The backend allows CANCELLED from these, but that path records no reason. The
  // dropdown withholds it so cancelling goes through Cancel Order, which asks.
  it('never offers CANCELLED', () => {
    for (const status of ORDER_STATUSES) {
      expect(ALLOWED_TRANSITIONS[status]).not.toContain('CANCELLED');
    }
  });

  // The Update button is disabled on an empty list, so an order mid-fulfilment must
  // keep somewhere to go.
  it('leaves every pre-dispatch status a way forward', () => {
    for (const status of ['PENDING', 'CONFIRMED', 'PROCESSING'] as const) {
      expect(ALLOWED_TRANSITIONS[status].length).toBeGreaterThan(0);
    }
  });

  it('offers nothing out of a terminal status', () => {
    expect(ALLOWED_TRANSITIONS.CANCELLED).toEqual([]);
    expect(ALLOWED_TRANSITIONS.RETURNED).toEqual([]);
  });
});

describe('FORWARD_STATUSES', () => {
  // Mirrors forwardStatuses in internal/service/order_service.go. Drifting from it
  // means the page hides an option the backend allows, or offers one it refuses.
  it('covers the pre-dispatch moves only', () => {
    expect(FORWARD_STATUSES).toEqual(['CONFIRMED', 'PROCESSING', 'SHIPPED']);
  });

  // The goods are already gone by then, so recording their fate is never gated.
  it('never gates a post-dispatch or recovery move', () => {
    for (const status of ['DELIVERED', 'RETURNED', 'CANCELLED'] as const) {
      expect(FORWARD_STATUSES).not.toContain(status);
    }
  });

  // A failed payment must still leave the order somewhere to go: Cancel Order.
  it('leaves nothing forward from a pre-dispatch status', () => {
    for (const status of ['PENDING', 'CONFIRMED', 'PROCESSING'] as const) {
      const remaining = ALLOWED_TRANSITIONS[status].filter((s) => !FORWARD_STATUSES.includes(s));
      expect(remaining).toEqual([]);
    }
  });
});
