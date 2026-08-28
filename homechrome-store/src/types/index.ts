// Customer
export interface Customer {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  phone: string;
  phone_verified: boolean;
  status: 'ACTIVE' | 'INACTIVE' | 'BLOCKED';
  addresses: Address[];
  created_at: string;
  updated_at: string;
}

export interface Address {
  id: string;
  first_name: string;
  last_name: string;
  phone: string;
  address_line1: string;
  address_line2?: string;
  city: string;
  state: string;
  postal_code: string;
  country: string;
  is_default?: boolean;
}

// Product & Category
export interface AttributeOption {
  value: string;
  label: string;
  surcharge?: number;
}

export interface CategoryAttribute {
  name: string;
  label: string;
  type: string;
  required: boolean;
  searchable: boolean;
  display_order: number;
  options?: AttributeOption[];
}

export interface Category {
  id: string;
  name: string;
  slug: string;
  description: string;
  image_url: string;
  parent_id?: string;
  own_attributes?: CategoryAttribute[];
  status: 'ACTIVE' | 'INACTIVE';
  product_count: number;
}

export interface Product {
  id: string;
  name: string;
  slug: string;
  description: string;
  sku: string;
  category_id: string;
  selling_price: number; // in paise
  base_price: number;    // in paise — canonical field name from API
  mrp: number;           // alias for base_price; kept for backward-compat with product detail pages
  images: ProductImage[];
  video_url?: string;
  video_poster_url?: string;
  status: 'ACTIVE' | 'DRAFT' | 'ARCHIVED';
  in_stock: boolean;
  color?: string;
  material?: string;
  weave_type?: string;
  origin?: string;
  craft_type?: string;
  weight?: number; // in grams
  dimensions?: { length: number; width: number; height?: number; unit: string };
  attributes: Record<string, string | string[]>; // API sends arrays for multi-value attrs
  allow_custom_dimensions: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface ProductImage {
  url: string;
  alt_text: string;
  is_primary: boolean;
  sort_order: number;
}

// Cart
export interface Cart {
  customer_id: string;
  item_count: number;
  subtotal: number; // paise
  currency: string;
}

export interface CartItem {
  product_id: string;
  product_name: string;
  product_sku: string;
  product_image: string;
  quantity: number;
  unit_price: number; // paise
  total_price: number; // paise
  is_custom_size: boolean;
}

export interface CartWithItems {
  cart: Cart;
  items: CartItem[];
}

// Order
export interface Order {
  id: string;
  order_number: string;
  customer_id: string;
  items: OrderItem[];
  item_count: number;
  subtotal: number;
  discount_amount: number;
  tax_amount: number;
  shipping_amount: number;
  total_amount: number;
  currency: string;
  status: OrderStatus;
  payment_status: PaymentStatus;
  shipping_address: Address;
  tracking_number?: string;
  tracking_url?: string;
  shipping_carrier?: string;
  created_at: string;
  shipped_at?: string;
  delivered_at?: string;
  cancelled_at?: string;
  /**
   * Notes the shop left on this order. The backend keeps the `internal_notes`
   * wire name (it is one shared entity with the admin) but filters out
   * everything flagged internal before it reaches us, so anything present here
   * was deliberately shared.
   */
  internal_notes?: OrderNote[];
}

export interface OrderNote {
  id: string;
  note: string;
  is_internal: boolean;
  created_at: string;
}

export type OrderStatus =
  | 'PENDING'
  | 'CONFIRMED'
  | 'PROCESSING'
  | 'SHIPPED'
  | 'DELIVERED'
  | 'CANCELLED'
  | 'RETURNED'
  | 'REFUNDED';

export type PaymentStatus =
  | 'PENDING'
  | 'INITIATED'
  | 'PAID'
  | 'SUCCESS'
  | 'FAILED'
  | 'REFUNDED'
  | 'PARTIALLY_REFUNDED';

export interface OrderItem {
  id: string;
  product_id: string;
  product_name: string;
  product_sku: string;
  product_image?: string;
  unit_price: number;
  quantity: number;
  total_price: number;
}

// Checkout
export interface CheckoutResult {
  order: Order;
  redirect_url: string;
  merchant_txn_id: string;
  // Explains why a submitted code didn't apply. The order still goes through
  // at full price — empty when a coupon applied cleanly or none was sent.
  coupon_notice?: string;
}

// Advisory preview for a coupon code against the current cart. Checkout
// re-validates and wins — the two can disagree if the cart changed since.
export interface CouponValidationResult {
  valid: boolean;
  code: string;
  discount_amount?: number; // paise
  error_message?: string;
}

// Pagination
