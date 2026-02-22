import type { Address, Dimensions } from '@/shared/types/common';

export type OrderStatus =
  | 'PENDING'
  | 'CONFIRMED'
  | 'PROCESSING'
  | 'SHIPPED'
  | 'DELIVERED'
  | 'CANCELLED'
  | 'RETURNED';
export type PaymentStatus = 'PENDING' | 'PAID' | 'FAILED' | 'REFUNDED';

export interface OrderItem {
  id: string;
  product_id: string;
  product_name: string;
  product_sku: string;
  quantity: number;
  unit_price: number;
  total_price: number;
  custom_dimensions?: Dimensions;
  attributes?: Record<string, unknown>;
}

export interface OrderNote {
  id: string;
  note: string;
  is_internal?: boolean;
  created_by: string;
  created_at: string;
}

export interface Order {
  id: string;
  order_number: string;
  customer_id: string;
  customer_name: string;
  customer_email: string;
  items: OrderItem[];
  subtotal: number;
  discount_amount: number;
  tax_amount: number;
  shipping_amount: number;
  total_amount: number;
  currency: string;
  payment_status: PaymentStatus;
  status: OrderStatus;
  shipping_address: Address;
  billing_address?: Address;
  tracking_number?: string;
  shipping_carrier?: string;
  internal_notes?: OrderNote[];
  coupon_code?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateOrderRequest {
  customer_id: string;
  items: {
    product_id: string;
    quantity: number;
    price_quote_id?: string;
    custom_dimensions?: Dimensions;
    attributes?: Record<string, unknown>;
  }[];
  shipping_address: Address;
  billing_address?: Address;
  notes?: string;
  coupon_code?: string;
}
