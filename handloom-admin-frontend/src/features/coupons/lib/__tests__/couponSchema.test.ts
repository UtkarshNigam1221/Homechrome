import { describe, expect, it } from 'vitest';

import { couponSchema } from '../couponSchema';
import type { CouponFormValues } from '../toCreateRequest';

// Coupon.Value had no upper bound anywhere, so 150% stored happily and then produced a
// discount larger than the cart. The Go validator refuses it; this is the same rule in
// the form, so the operator sees a field error rather than a 400.
const validForm: CouponFormValues = {
  code: 'SAVE20',
  name: 'Save 20',
  description: '',
  type: 'PERCENTAGE',
  value: 20,
  minOrderValue: 0,
  maxDiscount: 0,
  usageLimit: 0,
  usagePerUser: 0,
  audience: 'ALL',
  customerId: '',
  combinesWithOffers: false,
  validFrom: '2026-01-01',
  noEndDate: false,
  expiryDate: '2026-12-31',
  status: 'ACTIVE',
};

function errorsFor(overrides: Partial<CouponFormValues>): string[] {
  const result = couponSchema.safeParse({ ...validForm, ...overrides });
  return result.success ? [] : result.error.issues.map((i) => i.message);
}

describe('couponSchema: the percentage ceiling', () => {
  it('accepts the baseline form', () => {
    expect(errorsFor({})).toEqual([]);
  });

  it('rejects a percentage above 100', () => {
    expect(errorsFor({ type: 'PERCENTAGE', value: 150 })).toContain(
      'A percentage discount cannot exceed 100%'
    );
  });

  it('rejects 100.01%', () => {
    expect(errorsFor({ type: 'PERCENTAGE', value: 100.01 })).toContain(
      'A percentage discount cannot exceed 100%'
    );
  });

  it('accepts exactly 100%', () => {
    expect(errorsFor({ type: 'PERCENTAGE', value: 100 })).toEqual([]);
  });

  // A fixed coupon's value is rupees, where 5000 is a routine ₹5,000 off.
  it('leaves a fixed amount above 100 alone', () => {
    expect(errorsFor({ type: 'FIXED', value: 5000 })).toEqual([]);
  });

  it('still rejects a zero value', () => {
    expect(errorsFor({ value: 0 })).toContain('Must be greater than 0');
  });
});
