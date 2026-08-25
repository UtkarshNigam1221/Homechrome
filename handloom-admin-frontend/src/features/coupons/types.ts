// Mirrors domain.CouponType. No FREE_SHIPPING — that never existed on the backend.
export type CouponType = 'PERCENTAGE' | 'FIXED';
export type CouponStatus = 'ACTIVE' | 'INACTIVE' | 'EXPIRED';

// Mirrors domain.CouponAudience. Exactly one applies; CustomerID is set only for
// SPECIFIC_CUSTOMER.
export type CouponAudience = 'ALL' | 'FIRST_ORDER' | 'RETURNING' | 'SPECIFIC_CUSTOMER';

// Mirrors domain.Coupon. No applicable_categories/applicable_products — coupons carry
// no item scoping on the backend.
export interface Coupon {
  id: string;
  code: string;
  name: string;
  description?: string;
  type: CouponType;
  value: number; // percentage * 100, or a fixed amount in paise
  min_order_value: number;
  max_discount?: number; // percentage type only
  usage_limit: number; // 0 = unlimited
  usage_per_user: number; // 0 = unlimited
  usage_count: number;
  audience: CouponAudience;
  customer_id?: string;
  combines_with_offers: boolean;
  valid_from: string;
  valid_until?: string | null; // absent OR null = open-ended
  status: CouponStatus;
  created_at: string;
  updated_at: string;
}

// Mirrors domain.CreateCouponRequest field for field.
export interface CreateCouponRequest {
  code: string;
  name: string;
  description?: string;
  type: CouponType;
  value: number;
  min_order_value: number;
  max_discount?: number;
  usage_limit: number;
  usage_per_user: number;
  audience: CouponAudience;
  customer_id?: string;
  combines_with_offers: boolean;
  valid_from: string;
  valid_until?: string | null; // absent OR null = open-ended
}

// Mirrors domain.UpdateCouponRequest. Narrower than CreateCouponRequest on purpose:
// the backend has no path to change code, type, value, audience or customer_id.
//
// clear_valid_until exists because a PATCH body's null is indistinguishable from an
// omitted field once decoded — see coupon_service.go — so it's the only way to open-end.
export interface UpdateCouponRequest {
  name?: string;
  description?: string;
  min_order_value?: number;
  max_discount?: number;
  usage_limit?: number;
  usage_per_user?: number;
  combines_with_offers?: boolean;
  valid_from?: string;
  valid_until?: string;
  clear_valid_until?: boolean;
  status?: CouponStatus;
}
