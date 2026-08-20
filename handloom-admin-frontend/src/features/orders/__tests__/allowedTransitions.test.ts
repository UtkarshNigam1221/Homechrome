import { describe, expect, it } from 'vitest';

import { ALLOWED_TRANSITIONS, ORDER_STATUSES } from '@/features/orders/types';

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
