// Unit conversion between the coupon form and the wire shapes. One factor, one owner:
// see toStoredAmount. Both create and update mapping live here so the two can't drift.
import type {
  Coupon,
  CouponAudience,
  CouponStatus,
  CouponType,
  CreateCouponRequest,
  UpdateCouponRequest,
} from '../types';

export interface CouponFormValues {
  code: string;
  name: string;
  description: string;
  type: CouponType;
  value: number; // entered percentage points, or entered rupees for a fixed amount
  minOrderValue: number; // entered rupees, 0 = no minimum
  maxDiscount: number; // entered rupees, 0 = no cap (percentage coupons only)
  usageLimit: number; // 0 = unlimited
  usagePerUser: number; // 0 = unlimited
  audience: CouponAudience;
  customerId: string; // only meaningful, and only sent, for SPECIFIC_CUSTOMER
  combinesWithOffers: boolean;
  validFrom: string; // YYYY-MM-DD
  noEndDate: boolean;
  expiryDate: string; // YYYY-MM-DD, ignored when noEndDate is set
  status: CouponStatus;
}

// validFrom is left blank rather than defaulted to "today" — that's the operator's
// clock, a component concern this pure module shouldn't bake in.
export const defaultCouponFormValues: CouponFormValues = {
  code: '',
  name: '',
  description: '',
  type: 'PERCENTAGE',
  value: 10,
  minOrderValue: 0,
  maxDiscount: 0,
  usageLimit: 0,
  usagePerUser: 0,
  audience: 'ALL',
  customerId: '',
  combinesWithOffers: false,
  validFrom: '',
  noEndDate: false,
  expiryDate: '',
  status: 'ACTIVE',
};

// Percentage x100 or paise — both are the entered number x100. Rounded, not truncated,
// so a fractional entry (₹19.999, 12.5%) doesn't quietly lose value.
export function toStoredAmount(entered: number): number {
  return Math.round((entered || 0) * 100);
}

// Inverse of toStoredAmount, for reading a stored value back into the form.
export function fromStoredAmount(stored: number | undefined | null): number {
  return stored ? stored / 100 : 0;
}

// Turns a date-only string into a midnight-UTC instant via plain concatenation — never
// through `Date`, which is how this class of code picks up an off-by-one-day TZ bug.
export function dateOnlyToInstant(dateOnly: string): string {
  return `${dateOnly}T00:00:00.000Z`;
}

// Inverse of dateOnlyToInstant: sliced, not parsed, for the same reason.
export function instantToDateOnly(instant: string): string {
  return instant.slice(0, 10);
}

// customer_id is sent only when the audience actually names a customer — required
// there, rejected otherwise.
function customerIdFor(form: CouponFormValues): string | undefined {
  return form.audience === 'SPECIFIC_CUSTOMER' ? form.customerId : undefined;
}

// The cap only means something on a percentage coupon.
function maxDiscountFor(form: CouponFormValues): number | undefined {
  return form.type === 'PERCENTAGE' && form.maxDiscount
    ? toStoredAmount(form.maxDiscount)
    : undefined;
}

export function toCreateRequest(form: CouponFormValues): CreateCouponRequest {
  return {
    code: form.code.trim().toUpperCase(),
    name: form.name.trim(),
    description: form.description.trim() || undefined,
    type: form.type,
    value: toStoredAmount(form.value),
    min_order_value: toStoredAmount(form.minOrderValue),
    max_discount: maxDiscountFor(form),
    usage_limit: form.usageLimit || 0,
    usage_per_user: form.usagePerUser || 0,
    audience: form.audience,
    customer_id: customerIdFor(form),
    combines_with_offers: form.combinesWithOffers ?? false,
    valid_from: dateOnlyToInstant(form.validFrom),
    // Explicit null, not an omitted key — kept consistent with the update path below,
    // where the distinction is load-bearing.
    valid_until: form.noEndDate ? null : dateOnlyToInstant(form.expiryDate),
  };
}

// Code, type, value, audience and customer_id are absent: the API has no path to
// change them post-creation, so sending them would promise something the backend drops.
//
// Every field below is a concrete value, never omitted — an omitted key means "leave
// alone" on UpdateCouponRequest's pointer fields, not "clear it" (see types.ts).
export function toUpdateRequest(form: CouponFormValues): UpdateCouponRequest {
  const req: UpdateCouponRequest = {
    name: form.name.trim(),
    description: form.description.trim(),
    min_order_value: toStoredAmount(form.minOrderValue),
    // Unlike toCreateRequest, no type gate: type can't change on edit, and an edited
    // FIXED coupon's form.maxDiscount is already 0 (no cap either way).
    max_discount: toStoredAmount(form.maxDiscount),
    usage_limit: form.usageLimit || 0,
    usage_per_user: form.usagePerUser || 0,
    combines_with_offers: form.combinesWithOffers ?? false,
    valid_from: dateOnlyToInstant(form.validFrom),
    status: form.status,
  };

  // valid_until: null would be indistinguishable from omitted once decoded — see
  // UpdateCouponRequest — so "make this open-ended" needs its own flag instead.
  if (form.noEndDate) {
    req.clear_valid_until = true;
  } else {
    req.valid_until = dateOnlyToInstant(form.expiryDate);
  }

  return req;
}

// Inverse of the create/update mappers, for populating the form on edit. Code, type,
// audience and customer_id are read-only once a coupon exists but still shown.
export function couponToFormValues(coupon: Coupon): CouponFormValues {
  return {
    code: coupon.code,
    name: coupon.name,
    description: coupon.description ?? '',
    type: coupon.type,
    value: fromStoredAmount(coupon.value),
    minOrderValue: fromStoredAmount(coupon.min_order_value),
    maxDiscount: coupon.type === 'PERCENTAGE' ? fromStoredAmount(coupon.max_discount) : 0,
    usageLimit: coupon.usage_limit || 0,
    usagePerUser: coupon.usage_per_user || 0,
    audience: coupon.audience,
    customerId: coupon.customer_id ?? '',
    combinesWithOffers: coupon.combines_with_offers,
    validFrom: instantToDateOnly(coupon.valid_from),
    noEndDate: coupon.valid_until === null,
    expiryDate: coupon.valid_until ? instantToDateOnly(coupon.valid_until) : '',
    status: coupon.status,
  };
}
