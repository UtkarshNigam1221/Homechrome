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
  customerPhone: '',
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

describe('couponSchema: audience targeting', () => {
  it('requires a phone when creating a single-customer coupon', () => {
    expect(
      errorsFor({ audience: 'SPECIFIC_CUSTOMER', customerId: '', customerPhone: '' })
    ).toContain('Enter the customer’s 10-digit mobile number');
  });

  it('accepts a ten-digit phone', () => {
    expect(
      errorsFor({ audience: 'SPECIFIC_CUSTOMER', customerId: '', customerPhone: '9876543210' })
    ).toEqual([]);
  });

  // Every shape an operator might paste, now that the field imposes no length cap: a
  // cap could only truncate a country-coded paste into a different, plausible number.
  it('accepts the shapes an operator actually pastes', () => {
    for (const customerPhone of [
      '9876543210',
      '+91 98765 43210',
      '919876543210',
      '098765-43210',
      '+919876543210',
    ]) {
      expect(errorsFor({ audience: 'SPECIFIC_CUSTOMER', customerId: '', customerPhone })).toEqual(
        []
      );
    }
  });

  // A doubly-prefixed paste strips to a stray-zero number no record can match. The
  // leading-digit rule refuses it at the field instead of after a round-trip.
  it('rejects a doubly-prefixed number rather than sending it', () => {
    expect(
      errorsFor({ audience: 'SPECIFIC_CUSTOMER', customerId: '', customerPhone: '+910987654321' })
    ).toContain('An Indian mobile number starts with 6, 7, 8, or 9');
  });

  // Ten digits but the wrong leading one gets its own message — not the length message,
  // which would tell an operator who already typed ten digits that they typed too few.
  it('rejects a number that cannot be an Indian mobile', () => {
    expect(
      errorsFor({ audience: 'SPECIFIC_CUSTOMER', customerId: '', customerPhone: '1234567890' })
    ).toContain('An Indian mobile number starts with 6, 7, 8, or 9');
  });

  it('rejects a phone that is not ten digits', () => {
    expect(
      errorsFor({ audience: 'SPECIFIC_CUSTOMER', customerId: '', customerPhone: '98765' })
    ).toContain('Enter the customer’s 10-digit mobile number');
  });

  // An existing targeted coupon has an id and no phone, and its update sends neither.
  // Requiring one would make such a coupon unsavable on edit.
  it('does not demand a phone when a customer is already bound', () => {
    expect(
      errorsFor({ audience: 'SPECIFIC_CUSTOMER', customerId: 'cust_42', customerPhone: '' })
    ).toEqual([]);
  });

  it('ignores the phone entirely for the other audiences', () => {
    for (const audience of ['ALL', 'FIRST_ORDER', 'RETURNING'] as const) {
      expect(errorsFor({ audience, customerId: '', customerPhone: '' })).toEqual([]);
    }
  });
});
