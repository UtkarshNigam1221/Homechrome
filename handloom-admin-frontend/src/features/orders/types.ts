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
 * The statuses the dropdown offers. A subset of validTransitions in
 * internal/service/order_service.go: the backend rejects anything outside that map
 * with a 400, so offering more just produced failing picks.
 *
 * CANCELLED is deliberately absent even though the backend allows it. Cancelling
 * that way records no reason — the metric gets the literal "status_update" — so
 * cancellation goes through Cancel Order, which asks for one. Do not add it back to
 * "match the Go map"; the divergence is the point.
 */
export const ALLOWED_TRANSITIONS: Record<OrderStatus, OrderStatus[]> = {
  PENDING: ['CONFIRMED'],
  CONFIRMED: ['PROCESSING', 'SHIPPED'],
  PROCESSING: ['SHIPPED'],
  SHIPPED: ['DELIVERED', 'RETURNED'],
  DELIVERED: ['RETURNED'],
  CANCELLED: [],
  RETURNED: [],
};
/**
 * The moves the backend gates on payment, mirroring forwardStatuses in
 * internal/service/order_service.go. DELIVERED and RETURNED are absent: the goods
 * have already gone, so recording what happened to them is never blocked.
 */
export const FORWARD_STATUSES: OrderStatus[] = ['CONFIRMED', 'PROCESSING', 'SHIPPED'];

export type PaymentStatus =
  | 'PENDING'
  // Written by the gateway path, not the checkout one — the union has to carry
  // them or a status test against them only typechecks by widening to string.
  | 'INITIATED'
  | 'SUCCESS'
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
  // What the payment says has actually gone back. Authoritative — a client
  // summing its own refund rows can drift from it if a settlement half completed.
  refunded_amount?: number;
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

// Prices lines without raising a refund, so it carries no reason: the reason labels
// a refund, it does not affect what the lines are worth.
export interface PreviewRefundRequest {
  items: { order_item_id: string; quantity: number }[];
}

// What a requested set of lines would cost. Derived server-side; the client never
// sends an amount, so this is the only figure a screen should show.
export interface RefundPreview {
  total: number;
  is_final: boolean;
  lines: RefundItem[];
  breakdown: {
    line_value: number;
    discount: number;
    tax: number;
    // Zero until the refund that clears the order, and carries the residual when not.
    shipping: number;
  };
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
  // Resolved server-side; created_by alone is an opaque user id.
  created_by_name?: string;
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
