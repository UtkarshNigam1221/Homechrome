// Mirrors domain.Coupon in handloom-admin. Field names match the API exactly rather
// than going through a mapping layer, which is a second place for them to drift —
// and drift here silently failed every create.
export type CouponType = 'PERCENTAGE' | 'FIXED';
export type CouponStatus = 'ACTIVE' | 'INACTIVE' | 'EXPIRED';

export interface Coupon {
  id: string;
  code: string;
  name: string;
  description?: string;
  type: CouponType;
  // Percentage * 100 for PERCENTAGE, paise for FIXED. Both are the entered value
  // times 100, which is why one conversion covers each.
  value: number;
  min_order_value: number;
  max_discount?: number;
  usage_limit: number;
  usage_per_user: number;
  usage_count: number;
  applicable_categories?: string[];
  applicable_products?: string[];
  excluded_categories?: string[];
  excluded_products?: string[];
  valid_from: string;
  valid_until: string;
  status: CouponStatus;
  created_at: string;
  updated_at: string;
}

export interface CreateCouponRequest {
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
  status?: CouponStatus;
}
