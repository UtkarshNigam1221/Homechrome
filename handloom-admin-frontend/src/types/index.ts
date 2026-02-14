// ============================================================================
// Common Types
// ============================================================================

export interface PaginationParams {
  limit?: number;
  cursor?: string;
  sort_by?: string;
  sort_order?: 'ASC' | 'DESC';
}

export interface PaginationResponse {
  limit: number;
  next_cursor?: string;
  has_more: boolean;
}

export interface ApiResponse<T> {
  success: boolean;
  data: T;
  message?: string;
  error?: {
    code: string;
    message: string;
  };
}

export interface ListResponse<T> {
  items: T[];
  pagination: PaginationResponse;
}

// ============================================================================
// User & Auth Types
// ============================================================================

export type UserRole = 'ADMIN' | 'OPERATOR';
export type UserStatus = 'ACTIVE' | 'INACTIVE' | 'PENDING';

export interface User {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  phone?: string;
  role: UserRole;
  permissions?: string[];
  status: UserStatus;
  last_login_at?: string;
  created_at: string;
  updated_at: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}


export interface CreateUserRequest {
  email: string;
  password: string;
  first_name: string;
  last_name: string;
  phone?: string;
  role: UserRole;
  permissions?: string[];
  status?: UserStatus;
}

// ============================================================================
// Category Types
// ============================================================================

export type CategoryStatus = 'ACTIVE' | 'INACTIVE';
export type AttributeType =
  | 'SELECT'
  | 'MULTI_SELECT'
  | 'TEXT'
  | 'NUMBER'
  | 'BOOLEAN'
  | 'DIMENSION'
  | 'DIMENSION_RANGE';

export interface AttributeOption {
  value: string;
  label: string;
  surcharge?: number;
}

export interface CategoryAttribute {
  name: string;
  label: string;
  type: AttributeType;
  required: boolean;
  searchable: boolean; // When true, indexed for filtering and shown in filter UI
  display_order: number;
  options?: AttributeOption[];
}

export interface Category {
  id: string;
  name: string;
  slug: string;
  description?: string;
  image_url?: string;
  own_attributes?: CategoryAttribute[];
  status: CategoryStatus;
  product_count: number;
  created_at: string;
  updated_at: string;
}

export interface CreateCategoryRequest {
  name: string;
  description?: string;
  image_url?: string;
  own_attributes?: CategoryAttribute[];
  status?: CategoryStatus;
}

export interface ProductImage {
  url: string;
  alt_text?: string;
  is_primary?: boolean;
  sort_order?: number;
}

// ============================================================================
// Product Types
// ============================================================================

export type ProductStatus = 'ACTIVE' | 'INACTIVE' | 'DRAFT';

export interface Dimensions {
  length: number;
  width: number;
  height?: number;
  unit: string;
}

export interface Product {
  id: string;
  name: string;
  sku: string;
  slug: string;
  description?: string;
  category_id: string;
  artisan_id?: string;
  base_price: number;
  selling_price: number;
  cost_price?: number;
  currency: string;
  dimensions?: Dimensions;
  weight?: number;
  allow_custom_dimensions: boolean;
  pricing_rule_id?: string;
  attributes?: Record<string, unknown>;
  material?: string;
  color?: string;
  weave_type?: string;
  origin?: string;
  craft_type?: string;
  images?: ProductImage[];
  tags?: string[];
  quantity: number;
  reserved_qty: number;
  available_qty: number;
  low_stock_threshold: number;
  status: ProductStatus;
  created_at: string;
  updated_at: string;
}

export interface CreateProductRequest {
  name: string;
  sku: string;
  description?: string;
  category_id: string;
  artisan_id?: string;
  base_price: number;
  selling_price: number;
  cost_price?: number;
  currency?: string;
  dimensions?: Dimensions;
  weight?: number;
  allow_custom_dimensions?: boolean;
  pricing_rule_id?: string;
  attributes?: Record<string, unknown>;
  material?: string;
  color?: string;
  weave_type?: string;
  origin?: string;
  craft_type?: string;
  images?: ProductImage[];
  tags?: string[];
  initial_stock?: number;
  low_stock_threshold?: number;
  status?: ProductStatus;
}

// ============================================================================
// Order Types
// ============================================================================

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
  sku: string;
  quantity: number;
  unit_price: number;
  total_price: number;
  custom_dimensions?: Dimensions;
  attributes?: Record<string, unknown>;
}

export interface Address {
  type?: 'billing' | 'shipping';
  name: string;
  street: string;
  city: string;
  state: string;
  postal_code: string;
  country: string;
  phone?: string;
  is_default?: boolean;
}

export interface OrderNote {
  id: string;
  note: string;
  is_internal: boolean;
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
  discount: number;
  tax: number;
  shipping_cost: number;
  total_price: number;
  currency: string;
  payment_status: PaymentStatus;
  order_status: OrderStatus;
  shipping_address: Address;
  billing_address?: Address;
  tracking_number?: string;
  carrier?: string;
  notes?: OrderNote[];
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

// ============================================================================
// Customer Types
// ============================================================================

export type CustomerStatus = 'ACTIVE' | 'INACTIVE' | 'SUSPENDED';

export interface Customer {
  id: string;
  email: string;
  phone?: string;
  name: string;
  first_name?: string;
  last_name?: string;
  addresses?: Address[];
  status: CustomerStatus;
  order_count: number;
  total_spent: number;
  last_order_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateCustomerRequest {
  email: string;
  phone?: string;
  name: string;
  addresses?: Address[];
}

export interface UpdateCustomerRequest extends Partial<CreateCustomerRequest> {
  status?: CustomerStatus;
}

// ============================================================================
// Artisan Types
// ============================================================================

export type ArtisanStatus = 'ACTIVE' | 'INACTIVE' | 'SUSPENDED';

export interface BankDetails {
  account_name: string;
  account_number: string;
  bank_name: string;
  ifsc_code: string;
  upi_id?: string;
}

export interface Location {
  city: string;
  state: string;
  country: string;
}

export interface Artisan {
  id: string;
  name: string;
  email?: string;
  phone: string;
  craft_type?: string;
  skills?: string[];
  location: Location;
  bio?: string;
  profile_image?: string;
  bank_details?: BankDetails;
  status: ArtisanStatus;
  product_count: number;
  total_earnings: number;
  created_at: string;
  updated_at: string;
}

export interface CreateArtisanRequest {
  name: string;
  email?: string;
  phone: string;
  craft_type?: string;
  skills?: string[];
  location: Location;
  bio?: string;
  bank_details?: BankDetails;
}

export interface UpdateArtisanRequest extends Partial<CreateArtisanRequest> {
  status?: ArtisanStatus;
}

// ============================================================================
// Coupon Types
// ============================================================================

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

// ============================================================================
// Pricing Types
// ============================================================================

export type ScopeType = 'GLOBAL' | 'CATEGORY' | 'SUBCATEGORY' | 'PRODUCT' | 'MATERIAL';
export type PricingType = 'AREA_BASED' | 'LENGTH_BASED' | 'FIXED' | 'TIERED' | 'FORMULA';
export type PricingUnit =
  | 'SQ_INCH'
  | 'SQ_FOOT'
  | 'SQ_CM'
  | 'SQ_METER'
  | 'INCH'
  | 'CM'
  | 'FOOT'
  | 'METER';

export interface PricingRule {
  id: string;
  name: string;
  description?: string;
  scope_type: ScopeType;
  scope_id?: string;
  category_id?: string;
  pricing_type: PricingType;
  base_price: number;
  price_per_unit?: number;
  unit?: PricingUnit;
  material_multipliers?: Record<string, number>;
  attribute_surcharges?: {
    attribute_name: string;
    attribute_value: string;
    surcharge_type: 'FIXED' | 'PERCENTAGE';
    surcharge_value: number;
  }[];
  min_area?: number;
  max_area?: number;
  min_order_value?: number;
  priority: number;
  is_active: boolean;
  valid_from?: string;
  valid_until?: string;
  created_at: string;
  updated_at: string;
}

export interface CreatePricingRuleRequest {
  name: string;
  description?: string;
  scope_type: ScopeType;
  category_id?: string;
  pricing_type: PricingType;
  base_price: number;
  price_per_unit?: number;
  unit?: string;
  min_area?: number;
  max_area?: number;
  priority: number;
  is_active: boolean;
}

export type UpdatePricingRuleRequest = Partial<CreatePricingRuleRequest>;

export interface CalculatePriceRequest {
  category_id: string;
  product_id?: string;
  dimensions: {
    length: number;
    width: number;
    height?: number;
    unit: string;
  };
  attributes?: Record<string, string>;
  quantity: number;
}

export interface PriceBreakdown {
  area: number;
  area_unit: string;
  base_cost: number;
  material_cost: number;
  surcharges_total: number;
  subtotal_per_unit: number;
  quantity: number;
  total: number;
}

export interface CalculatePriceResponse {
  price_breakdown: PriceBreakdown;
  formatted_price: {
    subtotal: string;
    total: string;
    currency: string;
  };
  pricing_rule_id: string;
}

// ============================================================================
// Inventory Types
// ============================================================================

export type TransactionType = 'ADD' | 'REMOVE' | 'RESERVE' | 'RELEASE' | 'ADJUST';

export interface Inventory {
  product_id: string;
  product_name: string;
  sku: string;
  quantity: number;
  reserved_qty: number;
  available_qty: number;
  low_stock_threshold: number;
  is_low_stock: boolean;
  last_updated_at: string;
}

export interface InventoryTransaction {
  id: string;
  product_id: string;
  type: TransactionType;
  quantity: number;
  previous_qty: number;
  new_qty: number;
  reason?: string;
  reference_id?: string;
  created_by: string;
  created_at: string;
}

// ============================================================================
// Notification Types
// ============================================================================

export type NotificationType = 'ORDER' | 'PRODUCT' | 'SYSTEM' | 'PROMOTION' | 'ALERT';
export type NotificationStatus = 'UNREAD' | 'READ' | 'ARCHIVED';

export interface Notification {
  id: string;
  user_id: string;
  type: NotificationType;
  title: string;
  message: string;
  data?: Record<string, unknown>;
  priority?: 'low' | 'normal' | 'high';
  status: NotificationStatus;
  read_at?: string;
  created_at: string;
}

// ============================================================================
// Analytics Types
// ============================================================================

export interface DashboardStats {
  total_revenue: number;
  total_orders: number;
  total_customers: number;
  active_products: number;
  low_stock_count: number;
  pending_orders: number;
  revenue_change?: number;
  orders_change?: number;
}

export interface SalesAnalytics {
  period: string;
  data: {
    date: string;
    revenue: number;
    orders: number;
    average_order_value: number;
  }[];
  totals: {
    revenue: number;
    orders: number;
    average_order_value: number;
  };
}

export interface TopProduct {
  product_id: string;
  product_name: string;
  sku: string;
  quantity_sold: number;
  revenue: number;
  image_url?: string;
}

export interface TopCategory {
  category_id: string;
  category_name: string;
  product_count: number;
  revenue: number;
}

// ============================================================================
// Bulk Operation Types
// ============================================================================

export type BulkOperationType = 'IMPORT' | 'UPDATE' | 'EXPORT';
export type BulkEntityType = 'PRODUCT' | 'INVENTORY' | 'PRICE' | 'CUSTOMER';
export type BulkOperationStatus = 'PENDING' | 'PROCESSING' | 'COMPLETED' | 'FAILED' | 'CANCELLED';

export interface BulkOperation {
  id: string;
  type: BulkOperationType;
  entity_type: BulkEntityType;
  status: BulkOperationStatus;
  total_records: number;
  total_count: number;
  success_count: number;
  failure_count: number;
  error_count: number;
  file_url?: string;
  output_file_url: string;
  error_file_url: string;
  created_by: string;
  created_at: string;
  completed_at?: string;
}

// ============================================================================
// Report Types
// ============================================================================

export type ReportType = 'SALES' | 'INVENTORY' | 'ORDERS' | 'CUSTOMERS' | 'PRODUCTS' | 'ARTISANS';
export type ReportFormat = 'CSV' | 'EXCEL' | 'PDF';
export type ReportStatus = 'PENDING' | 'PROCESSING' | 'COMPLETED' | 'FAILED';

export interface Report {
  id: string;
  type: ReportType;
  status: ReportStatus;
  format: ReportFormat;
  filters?: Record<string, unknown>;
  file_url?: string;
  created_by: string;
  created_at: string;
  completed_at?: string;
}

// ============================================================================
// Asset Types (tmp/ → assets/ S3 flow, no DynamoDB records)
// ============================================================================

export type AssetType = 'IMAGE' | 'VIDEO' | 'DOCUMENT';

export interface UploadURLResponse {
  upload_url: string;
  tmp_key: string;
  tmp_url: string;
  expires_at: string;
}

// ============================================================================
// Audit Types
// ============================================================================

export interface AuditLog {
  id: string;
  action: string;
  entity_type: string;
  entity_id: string;
  user_id: string;
  user_email: string;
  changes?: {
    field: string;
    old_value: unknown;
    new_value: unknown;
  }[];
  ip_address?: string;
  user_agent?: string;
  created_at: string;
}
