import { describe, expect, it } from 'vitest';

import type { CouponFormValues } from '../toCreateRequest';
import { toCreateRequest, toFormValues } from '../toCreateRequest';

const base: CouponFormValues = {
  code: 'summer20',
  name: 'Summer Sale',
  type: 'PERCENTAGE',
  value: 20,
  valid_from: '2026-06-01',
  valid_until: '2026-06-30',
  status: 'ACTIVE',
};

describe('toCreateRequest', () => {
  // The bug this replaces: the form sent a raw percentage and rupees, under field
  // names the API does not have, so every create failed validation.
  it('sends a percentage as percentage times 100', () => {
    expect(toCreateRequest({ ...base, type: 'PERCENTAGE', value: 20 }).value).toBe(2000);
  });

  it('sends a fixed amount in paise', () => {
    expect(toCreateRequest({ ...base, type: 'FIXED', value: 100 }).value).toBe(10000);
  });

  it('rounds a fractional entry rather than truncating it', () => {
    expect(toCreateRequest({ ...base, type: 'FIXED', value: 99.995 }).value).toBe(10000);
    expect(toCreateRequest({ ...base, type: 'PERCENTAGE', value: 12.5 }).value).toBe(1250);
  });

  it('uppercases the code', () => {
    expect(toCreateRequest(base).code).toBe('SUMMER20');
  });

  it('converts money constraints to paise', () => {
    const got = toCreateRequest({ ...base, min_order_value: 3000, max_discount: 500 });
    expect(got.min_order_value).toBe(300000);
    expect(got.max_discount).toBe(50000);
  });

  // A cap on a fixed amount is meaningless — the amount already is the cap.
  it('drops the discount cap on a fixed-amount coupon', () => {
    expect(
      toCreateRequest({ ...base, type: 'FIXED', max_discount: 500 }).max_discount
    ).toBeUndefined();
  });

  it('treats a blank limit as unlimited rather than omitting it', () => {
    const got = toCreateRequest(base);
    expect(got.usage_limit).toBe(0);
    expect(got.usage_per_user).toBe(0);
  });

  // A same-day coupon must be usable on that day, not expire as it starts. Asserted
  // in local time: the boundaries are local midnights, so pinning UTC strings here
  // would pass in IST and fail on a UTC runner.
  it('spans a whole day for a same-day coupon', () => {
    const got = toCreateRequest({ ...base, valid_from: '2026-06-01', valid_until: '2026-06-01' });
    const from = new Date(got.valid_from);
    const until = new Date(got.valid_until);

    expect(until.getTime()).toBeGreaterThan(from.getTime());
    expect(from.getDate()).toBe(1);
    expect(until.getDate()).toBe(1);
    expect(from.getHours()).toBe(0);
    expect(until.getHours()).toBe(23);
    // Just under 24h: 00:00:00 to 23:59:59.
    expect(until.getTime() - from.getTime()).toBe(86399 * 1000);
  });

  it('omits an empty description instead of sending an empty string', () => {
    expect(toCreateRequest({ ...base, description: '' }).description).toBeUndefined();
  });
});

describe('toFormValues', () => {
  // An asymmetric conversion would rescale the coupon on every save, so the round
  // trip matters more than either direction on its own.
  it('round-trips a coupon unchanged', () => {
    const stored = {
      id: 'c1',
      code: 'SUMMER20',
      name: 'Summer Sale',
      description: 'June promo',
      type: 'PERCENTAGE' as const,
      value: 2000,
      min_order_value: 300000,
      max_discount: 50000,
      usage_limit: 100,
      usage_per_user: 1,
      usage_count: 7,
      applicable_categories: ['cat_sarees'],
      applicable_products: [],
      excluded_categories: [],
      excluded_products: ['p_1'],
      valid_from: '2026-06-01T00:00:00Z',
      valid_until: '2026-06-30T23:59:59Z',
      status: 'ACTIVE' as const,
      created_at: '2026-05-01T00:00:00Z',
      updated_at: '2026-05-01T00:00:00Z',
    };

    const resent = toCreateRequest(toFormValues(stored));

    expect(resent.value).toBe(stored.value);
    expect(resent.min_order_value).toBe(stored.min_order_value);
    expect(resent.max_discount).toBe(stored.max_discount);
    expect(resent.usage_limit).toBe(stored.usage_limit);
    expect(resent.usage_per_user).toBe(stored.usage_per_user);
    expect(resent.applicable_categories).toEqual(stored.applicable_categories);
    expect(resent.excluded_products).toEqual(stored.excluded_products);
    expect(resent.code).toBe(stored.code);
    expect(resent.name).toBe(stored.name);
  });

  it('survives repeated edit-and-save cycles without drifting', () => {
    let request = toCreateRequest({ ...base, type: 'FIXED', value: 1234.56 });
    const firstValue = request.value;
    for (let i = 0; i < 5; i++) {
      request = toCreateRequest(toFormValues({ ...request, id: 'c', usage_count: 0 } as never));
    }
    expect(request.value).toBe(firstValue);
  });
});
