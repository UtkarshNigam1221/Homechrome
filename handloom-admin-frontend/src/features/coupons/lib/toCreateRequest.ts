import type { Coupon, CouponStatus, CouponType, CreateCouponRequest } from '../types';

// Percentage is stored as percentage * 100, fixed amounts as paise. Both are the
// entered number times 100, which is why one factor covers each.
const TO_MINOR = 100;

export interface CouponFormValues {
  code: string;
  name: string;
  description?: string;
  type: CouponType;
  value: number;
  min_order_value?: number;
  max_discount?: number;
  usage_limit?: number;
  usage_per_user?: number;
  applicable_categories?: string[];
  applicable_products?: string[];
  excluded_categories?: string[];
  excluded_products?: string[];
  valid_from: string;
  valid_until: string;
  status: CouponStatus;
}

// toCreateRequest maps the form's units to the API's. Kept out of the component so
// the conversion is testable: the admin previously sent percentages and rupees under
// the wrong field names, and nothing caught it.
export function toCreateRequest(data: CouponFormValues): CreateCouponRequest {
  return {
    code: data.code.toUpperCase(),
    name: data.name,
    description: data.description || undefined,
    type: data.type,
    value: Math.round(data.value * TO_MINOR),
    min_order_value: data.min_order_value ? Math.round(data.min_order_value * TO_MINOR) : 0,
    // A cap only means anything on a percentage.
    max_discount:
      data.type === 'PERCENTAGE' && data.max_discount
        ? Math.round(data.max_discount * TO_MINOR)
        : undefined,
    usage_limit: data.usage_limit || 0,
    usage_per_user: data.usage_per_user || 0,
    applicable_categories: data.applicable_categories,
    applicable_products: data.applicable_products,
    excluded_categories: data.excluded_categories,
    excluded_products: data.excluded_products,
    // Whole days sent as instants: start of the first day to the end of the last, or
    // a same-day coupon would expire the moment it began.
    valid_from: new Date(`${data.valid_from}T00:00:00`).toISOString(),
    valid_until: new Date(`${data.valid_until}T23:59:59`).toISOString(),
    status: data.status,
  };
}

// toFormValues is the inverse: what an existing coupon looks like in the form. The
// same factor in the other direction, so a coupon edited and saved unchanged must
// come back byte-identical — an asymmetric conversion would scale it every save.
export function toFormValues(coupon: Coupon): CouponFormValues {
  const day = (iso: string | undefined): string =>
    iso ? new Date(iso).toISOString().split('T')[0] : new Date().toISOString().split('T')[0];

  return {
    code: coupon.code,
    name: coupon.name ?? '',
    description: coupon.description ?? '',
    type: coupon.type,
    value: coupon.value / TO_MINOR,
    min_order_value: coupon.min_order_value ? coupon.min_order_value / TO_MINOR : 0,
    max_discount: coupon.max_discount ? coupon.max_discount / TO_MINOR : 0,
    usage_limit: coupon.usage_limit ?? 0,
    usage_per_user: coupon.usage_per_user ?? 0,
    applicable_categories: coupon.applicable_categories ?? [],
    applicable_products: coupon.applicable_products ?? [],
    excluded_categories: coupon.excluded_categories ?? [],
    excluded_products: coupon.excluded_products ?? [],
    valid_from: day(coupon.valid_from),
    valid_until: day(coupon.valid_until),
    status: coupon.status,
  };
}
