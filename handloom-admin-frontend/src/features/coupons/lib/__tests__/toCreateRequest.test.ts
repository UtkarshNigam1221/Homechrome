import { describe, expect, it } from 'vitest';

import type { Coupon } from '../../types';
import {
  type CouponFormValues,
  couponToFormValues,
  defaultCouponFormValues,
  fromStoredAmount,
  instantToDateOnly,
  toCreateRequest,
  toStoredAmount,
  toUpdateRequest,
} from '../toCreateRequest';

const baseForm: CouponFormValues = {
  ...defaultCouponFormValues,
  code: 'test10',
  name: 'Test coupon',
  value: 10,
  validFrom: '2026-01-01',
  expiryDate: '2026-12-31',
};

describe('toStoredAmount / fromStoredAmount', () => {
  it('stores a percentage as percentage x100', () => {
    expect(toStoredAmount(12.5)).toBe(1250);
  });

  it('stores a fixed amount in paise', () => {
    expect(toStoredAmount(149.5)).toBe(14950);
  });

  it('rounds a fractional entry rather than truncating it', () => {
    // 19.999 * 100 = 1999.9 — truncating would give 1999, rounding gives 2000.
    expect(toStoredAmount(19.999)).toBe(2000);
    // 10.005 * 100 = 1000.5 — truncating would give 1000, rounding gives 1001.
    expect(toStoredAmount(10.005)).toBe(1001);
  });

  it('round-trips a stored integer back to the entered figure', () => {
    expect(fromStoredAmount(1250)).toBe(12.5);
    expect(fromStoredAmount(toStoredAmount(19.99))).toBe(19.99);
  });

  it('treats an absent stored value as zero', () => {
    expect(fromStoredAmount(undefined)).toBe(0);
    expect(fromStoredAmount(null)).toBe(0);
  });
});

describe('date conversion is TZ-independent', () => {
  // No `Date` object involved, so these hold under any TZ the test runner uses.
  it('turns a date-only string into a midnight-UTC instant', () => {
    expect(toCreateRequest({ ...baseForm, validFrom: '2026-01-15' }).valid_from).toBe(
      '2026-01-15T00:00:00.000Z'
    );
  });

  it('pulls the calendar date back out of an instant', () => {
    expect(instantToDateOnly('2026-01-15T00:00:00.000Z')).toBe('2026-01-15');
    expect(instantToDateOnly('2026-01-15T18:30:00.000Z')).toBe('2026-01-15');
  });
});

describe('toCreateRequest: amounts and limits', () => {
  it('drops the cap on a fixed-amount coupon', () => {
    const req = toCreateRequest({ ...baseForm, type: 'FIXED', value: 100, maxDiscount: 50 });
    expect(req.max_discount).toBeUndefined();
  });

  it('keeps the cap on a percentage coupon', () => {
    const req = toCreateRequest({ ...baseForm, type: 'PERCENTAGE', maxDiscount: 600 });
    expect(req.max_discount).toBe(60000);
  });

  it('sends blank usage limits as 0 rather than omitting them', () => {
    const req = toCreateRequest({ ...baseForm, usageLimit: 0, usagePerUser: 0 });
    expect(req.usage_limit).toBe(0);
    expect(req.usage_per_user).toBe(0);
    expect('usage_limit' in req).toBe(true);
    expect('usage_per_user' in req).toBe(true);
  });

  it('sends a blank minimum order value as 0', () => {
    expect(toCreateRequest({ ...baseForm, minOrderValue: 0 }).min_order_value).toBe(0);
  });
});

describe('audience and validity', () => {
  it('sends the chosen audience', () => {
    const req = toCreateRequest({ ...baseForm, audience: 'RETURNING' });
    expect(req.audience).toBe('RETURNING');
  });

  it('sends customer_id only for a named-customer coupon', () => {
    const named = toCreateRequest({
      ...baseForm,
      audience: 'SPECIFIC_CUSTOMER',
      customerId: 'cust_1',
    });
    expect(named.customer_id).toBe('cust_1');

    const publicCoupon = toCreateRequest({
      ...baseForm,
      audience: 'ALL',
      customerId: 'cust_1',
    });
    expect(publicCoupon.customer_id).toBeUndefined();
  });

  // Open-ended must be an explicit null, not an omitted field — omitted is
  // indistinguishable from "unchanged" on the update path.
  it('sends valid_until as null when the coupon is open-ended', () => {
    const req = toCreateRequest({ ...baseForm, noEndDate: true, expiryDate: '2026-12-31' });
    expect(req.valid_until).toBeNull();
  });

  it('sends the end date when there is one', () => {
    const req = toCreateRequest({ ...baseForm, noEndDate: false, expiryDate: '2026-12-31' });
    expect(req.valid_until).not.toBeNull();
    expect(req.valid_until).toBe('2026-12-31T00:00:00.000Z');
  });

  // Off unless deliberately turned on: buy-2-get-1 is a third off before any code.
  it('defaults combines_with_offers to false', () => {
    const req = toCreateRequest(baseForm);
    expect(req.combines_with_offers).toBe(false);
  });

  it('sends combines_with_offers true when the operator opts in', () => {
    const req = toCreateRequest({ ...baseForm, combinesWithOffers: true });
    expect(req.combines_with_offers).toBe(true);
  });
});

describe('toUpdateRequest', () => {
  it('has no path to change code, type, value, audience or customer_id', () => {
    const req = toUpdateRequest({
      ...baseForm,
      audience: 'SPECIFIC_CUSTOMER',
      customerId: 'cust_1',
    }) as Record<string, unknown>;

    expect(req).not.toHaveProperty('code');
    expect(req).not.toHaveProperty('type');
    expect(req).not.toHaveProperty('value');
    expect(req).not.toHaveProperty('audience');
    expect(req).not.toHaveProperty('customer_id');
  });

  it('clears the end date with an explicit flag, not a null value', () => {
    const req = toUpdateRequest({ ...baseForm, noEndDate: true });
    expect(req.clear_valid_until).toBe(true);
    expect(req.valid_until).toBeUndefined();
  });

  it('sends the end date, with clear_valid_until unset, when there is one', () => {
    const req = toUpdateRequest({ ...baseForm, noEndDate: false, expiryDate: '2026-12-31' });
    expect(req.valid_until).toBe('2026-12-31T00:00:00.000Z');
    expect(req.clear_valid_until).toBeUndefined();
  });

  // Every UpdateCouponRequest field is a pointer — an omitted key means "leave alone",
  // so a real clear has to arrive as "" or 0, not a missing key.
  it('sends a cleared description as an empty string, not an omitted key', () => {
    const req = toUpdateRequest({ ...baseForm, description: '  ' }) as Record<string, unknown>;
    expect(req).toHaveProperty('description', '');
  });

  it('sends a dropped cap as 0, not an omitted key', () => {
    const req = toUpdateRequest({
      ...baseForm,
      type: 'PERCENTAGE',
      maxDiscount: 0,
    }) as Record<string, unknown>;
    expect(req).toHaveProperty('max_discount', 0);
  });
});

describe('round-trip stability', () => {
  const coupon: Coupon = {
    id: 'coupon_1',
    code: 'FEST20',
    name: 'Festive 20',
    description: 'Twenty percent off, capped at ₹600',
    type: 'PERCENTAGE',
    value: 2000, // 20.00%
    min_order_value: 100000, // ₹1,000
    max_discount: 60000, // ₹600 cap
    usage_limit: 5,
    usage_per_user: 1,
    usage_count: 0,
    audience: 'ALL',
    combines_with_offers: false,
    valid_from: '2026-01-01T00:00:00.000Z',
    valid_until: '2026-12-31T00:00:00.000Z',
    status: 'ACTIVE',
    created_at: '2026-01-01T00:00:00.000Z',
    updated_at: '2026-01-01T00:00:00.000Z',
  };

  it('a freshly created coupon comes back through the form unchanged', () => {
    const form: CouponFormValues = {
      ...defaultCouponFormValues,
      code: 'FEST20',
      name: 'Festive 20',
      description: 'Twenty percent off, capped at ₹600',
      type: 'PERCENTAGE',
      value: 20,
      minOrderValue: 1000,
      maxDiscount: 600,
      usageLimit: 5,
      usagePerUser: 1,
      audience: 'ALL',
      validFrom: '2026-01-01',
      expiryDate: '2026-12-31',
    };
    const req = toCreateRequest(form);
    const formAgain = couponToFormValues({
      ...coupon,
      code: req.code,
      name: req.name,
      description: req.description,
      value: req.value,
      min_order_value: req.min_order_value,
      max_discount: req.max_discount,
      valid_from: req.valid_from,
      valid_until: req.valid_until,
    });
    expect(toCreateRequest(formAgain)).toEqual(req);
  });

  // An asymmetric conversion would rescale a coupon's figures on every save; a single
  // round trip wouldn't show that, so this drives it through several cycles.
  it('an edited-and-saved coupon does not drift across several save cycles', () => {
    let current = coupon;

    for (let cycle = 0; cycle < 5; cycle++) {
      const form = couponToFormValues(current);
      const update = toUpdateRequest(form);

      current = {
        ...current,
        name: update.name ?? current.name,
        description: update.description ?? current.description,
        min_order_value: update.min_order_value ?? current.min_order_value,
        max_discount: update.max_discount,
        usage_limit: update.usage_limit ?? current.usage_limit,
        usage_per_user: update.usage_per_user ?? current.usage_per_user,
        combines_with_offers: update.combines_with_offers ?? current.combines_with_offers,
        valid_from: update.valid_from ?? current.valid_from,
        valid_until: update.clear_valid_until ? null : (update.valid_until ?? current.valid_until),
        status: update.status ?? current.status,
      };

      // value, type and audience are immutable on update — they must never move.
      expect(current.value).toBe(coupon.value);
      expect(current.type).toBe(coupon.type);
      expect(current.audience).toBe(coupon.audience);
    }

    expect(current.min_order_value).toBe(coupon.min_order_value);
    expect(current.max_discount).toBe(coupon.max_discount);
    expect(current.valid_from).toBe(coupon.valid_from);
    expect(current.valid_until).toBe(coupon.valid_until);
  });

  it('an open-ended coupon stays open-ended across save cycles', () => {
    let current: Coupon = { ...coupon, valid_until: null };

    for (let cycle = 0; cycle < 3; cycle++) {
      const form = couponToFormValues(current);
      expect(form.noEndDate).toBe(true);
      const update = toUpdateRequest(form);
      current = {
        ...current,
        valid_until: update.clear_valid_until ? null : (update.valid_until ?? null),
      };
    }

    expect(current.valid_until).toBeNull();
  });
});
