export type CouponType = 'PERCENTAGE' | 'FIXED_AMOUNT' | 'FREE_SHIPPING';
export type CouponStatus = 'ACTIVE' | 'INACTIVE' | 'EXPIRED';

export interface Coupon {
  id: string;
  code: string;
  type: CouponType;
  discount_value: number;
  max_uses?: number;
  used_count: number;
  min_order_value?: number;
  max_discount?: number;
  applicable_categories?: string[];
  applicable_products?: string[];
  expiry_date?: string;
  status: CouponStatus;
  created_at: string;
  updated_at: string;
}

export interface CreateCouponRequest {
  code: string;
  type: CouponType;
  discount_value: number;
  max_uses?: number;
  min_order_value?: number;
  max_discount?: number;
  applicable_categories?: string[];
  applicable_products?: string[];
  expiry_date?: string;
  status?: CouponStatus;
}
