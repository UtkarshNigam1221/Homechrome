import type { Address, Dimensions } from '@/shared/types/common';

export type OrderStatus =
  | 'PENDING'
  | 'CONFIRMED'
  | 'PROCESSING'
  | 'SHIPPED'
  | 'DELIVERED'
  | 'CANCELLED'
  | 'RETURNED';

export const ORDER_STATUSES: OrderStatus[] = [
  'PENDING',
  'CONFIRMED',
  'PROCESSING',
  'SHIPPED',
  'DELIVERED',
  'CANCELLED',
  'RETURNED',
];

/**
 * Mirrors validTransitions in internal/service/order_service.go. The backend
 * rejects anything else with a 400, so offering the full status list just
 * produced failing picks.
 *
 * Keep in sync with the Go map — it is the source of truth.
 */
export const ALLOWED_TRANSITIONS: Record<OrderStatus, OrderStatus[]> = {
  PENDING: ['CONFIRMED', 'CANCELLED'],
  CONFIRMED: ['PROCESSING', 'SHIPPED', 'CANCELLED'],
  PROCESSING: ['SHIPPED', 'CANCELLED'],
  SHIPPED: ['DELIVERED', 'RETURNED'],
  DELIVERED: ['RETURNED'],
  CANCELLED: [],
  RETURNED: [],
};
export type PaymentStatus =
  | 'PENDING'
  | 'PAID'
  | 'FAILED'
  | 'REFUNDED'
  // Money went back for some lines while the rest of the order still stands.
  | 'PARTIALLY_REFUNDED';

export interface OrderItem {
  id: string;
  product_id: string;
  product_name: string;
  product_sku: string;
  quantity: number;
  // How much of this line has already gone back. Only completed refunds count.
  refunded_quantity: number;
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
  tracking_url?: string;
  internal_notes?: OrderNote[];
  coupon_code?: string;
  created_at: string;
  updated_at: string;
}

export interface ProviderPaymentStatus {
  order_id: string;
  merchant_txn_id: string;
  provider_order_id: string;
  provider_state: string;
  local_status: string;
  amount: number;
  payment_mode?: string;
  transaction_id?: string;
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

export type RefundStatus = 'PENDING' | 'COMPLETED' | 'FAILED';

export type RefundReason =
  | 'OUT_OF_STOCK'
  | 'DAMAGED'
  | 'CUSTOMER_REQUEST'
  | 'PRICING_ERROR'
  | 'OTHER';

// Bounded server-side so it can label a metric. Anything needing explanation
// belongs in an order note.
export const REFUND_REASON_LABELS: Record<RefundReason, string> = {
  OUT_OF_STOCK: 'Out of stock',
  DAMAGED: 'Damaged',
  CUSTOMER_REQUEST: 'Customer request',
  PRICING_ERROR: 'Pricing error',
  OTHER: 'Other',
};

export interface RefundItem {
  order_item_id: string;
  product_id: string;
  product_name: string;
  quantity: number;
  amount: number;
  // True returns the units to sale; false writes them off.
  restock: boolean;
}

export interface Refund {
  id: string;
  order_id: string;
  payment_id: string;
  customer_id: string;
  amount: number;
  status: RefundStatus;
  reason: RefundReason;
  items: RefundItem[];
  merchant_refund_id: string;
  provider_refund_id?: string;
  error_code?: string;
  detailed_error_code?: string;
  initiated_at: string;
  completed_at?: string;
  created_by: string;
}

// Lines and quantities only — the server derives the money and rejects any
// amount a client sends.
export interface CreateRefundRequest {
  reason: RefundReason;
  items: {
    order_item_id: string;
    quantity: number;
    restock: boolean;
  }[];
}
