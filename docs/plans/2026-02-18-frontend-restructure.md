# Frontend Directory Restructure — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Restructure `handloom-admin-frontend/src/` from flat pages/components layout to feature-based architecture with aggressive file splitting and dead code removal.

**Architecture:** Feature-based directory structure where each business domain (products, orders, etc.) is a self-contained module with its own api, types, and components. Shared infrastructure (UI primitives, hooks, stores, utils) lives in `shared/`. App shell (routes, providers) lives in `app/`.

**Tech Stack:** React 19, TypeScript, Vite 7, React Router v6, React Query, Zustand, react-hook-form + zod

**Design doc:** `docs/plans/2026-02-18-frontend-restructure-design.md`

---

## Cross-Feature Type Dependencies

Before starting, note these types that cross feature boundaries:

| Type | Defined in | Also used by |
|------|-----------|-------------|
| `Address` | orders | customers |
| `Dimensions` | products | orders |
| `Category`, `CategoryAttribute` | categories | products, pricing |
| `Product` | products | inventory |
| `User` | auth | settings |

**Rule:** `Address` and `Dimensions` move to `shared/types/common.ts` since they're used across 2+ features. For other cross-feature types, import from the owning feature (e.g., `import type { Category } from '@/features/categories/types'`).

## Cross-Feature API Dependencies

| Feature file | Imports APIs from |
|-------------|-------------------|
| `ProductFormModal` | categoriesApi, artisansApi, productsApi |
| `PricingRuleFormModal` | categoriesApi, pricingApi |
| `DashboardPage` | analyticsApi, inventoryApi, ordersApi |
| `InventoryPage` | inventoryApi, productsApi |
| `SettingsPage` | authApi, usersApi |
| `BulkImportModal` | bulkApi + raw apiClient |

**Rule:** Import from other features' api.ts files directly (e.g., `import { categoriesApi } from '@/features/categories/api'`).

---

## Task 1: Create Directory Skeleton

**Files:** Create directories only — no files yet.

**Step 1: Create all directories**

```bash
cd handloom-admin-frontend/src
mkdir -p app
mkdir -p shared/{api,components/{ui,loading,layout},hooks,stores,types,utils}
mkdir -p features/{auth,products,orders,categories,customers,artisans,coupons,pricing,inventory,dashboard,analytics,reports,bulk,notifications,settings}/components
mkdir -p features/products/components/ProductFormModal
mkdir -p features/orders/components/OrderDetailPage
```

**Step 2: Commit**

```bash
git add -A
git commit -m "chore: create feature-based directory skeleton"
```

---

## Task 2: Create shared/types/common.ts

**Files:**
- Create: `src/shared/types/common.ts`

**Step 1: Write the shared types file**

Extract from `src/types/index.ts` lines 1-31 (PaginationParams, PaginationResponse, ApiResponse, ListResponse) plus Address (lines 227-237) and Dimensions (lines 134-139):

```typescript
// Common types shared across all features

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

// Shared across orders + customers
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

// Shared across products + orders
export interface Dimensions {
  length: number;
  width: number;
  height?: number;
  unit: string;
}
```

**Step 2: Commit**

```bash
git add src/shared/types/common.ts
git commit -m "feat: add shared/types/common.ts with cross-feature types"
```

---

## Task 3: Create All Feature Type Files

**Files:** Create 14 `features/*/types.ts` files by splitting `src/types/index.ts`.

Each file extracts the relevant section from the monolith. Listed below are the exact type names per feature.

### features/auth/types.ts
Types: `UserRole`, `UserStatus`, `User`, `LoginRequest`, `CreateUserRequest`, `ChangePasswordRequest`, `ResetPasswordRequest`

Note: Add `ChangePasswordRequest` and `ResetPasswordRequest` interfaces even though they're not in the current types file — they're used inline in auth API but having them typed is better practice. Actually, skip these since YAGNI — the auth API uses inline params.

```typescript
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
```

### features/categories/types.ts
Types: `CategoryStatus`, `AttributeType`, `AttributeOption`, `CategoryAttribute`, `Category`, `CreateCategoryRequest`, `ProductImage`

Note: `ProductImage` is defined between category and product sections. It's used only by Product types, but since it relates to the image_url pattern shared with categories, keep it in products. Actually — `ProductImage` is only used by `Product` and `CreateProductRequest`, so move it to `features/products/types.ts`.

```typescript
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
  searchable: boolean;
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
```

### features/products/types.ts
Types: `ProductStatus`, `ProductImage`, `Product`, `CreateProductRequest`

```typescript
import type { Dimensions } from '@/shared/types/common';

export interface ProductImage {
  url: string;
  alt_text?: string;
  is_primary?: boolean;
  sort_order?: number;
}

export type ProductStatus = 'ACTIVE' | 'INACTIVE' | 'DRAFT';

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
```

### features/orders/types.ts
Types: `OrderStatus`, `PaymentStatus`, `OrderItem`, `OrderNote`, `Order`, `CreateOrderRequest`

Note: `Address` and `Dimensions` are imported from `@/shared/types/common`.

```typescript
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
  sku: string;
  quantity: number;
  unit_price: number;
  total_price: number;
  custom_dimensions?: Dimensions;
  attributes?: Record<string, unknown>;
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
```

### features/customers/types.ts
```typescript
import type { Address } from '@/shared/types/common';

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
```

### features/artisans/types.ts
```typescript
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
```

### features/coupons/types.ts
```typescript
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
```

### features/pricing/types.ts
```typescript
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
```

### features/inventory/types.ts
```typescript
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
```

### features/notifications/types.ts
```typescript
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
```

### features/analytics/types.ts
```typescript
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
```

### features/bulk/types.ts
```typescript
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
```

### features/reports/types.ts
```typescript
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
```

### features/assets/types.ts (shared asset types used by BulkImportModal and ProductFormModal)

Actually — assets don't have their own page. `UploadURLResponse` and `AssetType` are used by `ImageUpload` component and bulk upload. Put them in `shared/types/common.ts`:

Add to `shared/types/common.ts`:
```typescript
export type AssetType = 'IMAGE' | 'VIDEO' | 'DOCUMENT';

export interface UploadURLResponse {
  upload_url: string;
  tmp_key: string;
  tmp_url: string;
  expires_at: string;
}
```

### features/audit/types.ts (no page — audit is used by settings? Let's check)

`AuditLog` is only used in `auditApi` which is not imported by any page. It's dead API code but keep the types in case they're needed. Put audit types in shared since audit spans all features:

Add to `shared/types/common.ts`:
```typescript
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
```

### features/dashboard/types.ts and features/settings/types.ts

Dashboard and settings don't define their own types — they import from other features. No types.ts needed for these features.

**Step: Commit all type files**

```bash
git add src/shared/types/ src/features/*/types.ts
git commit -m "feat: split types/index.ts into per-feature type modules"
```

---

## Task 4: Create shared/api/client.ts

**Files:**
- Create: `src/shared/api/client.ts` (copy existing `src/api/client.ts` + add `normalizeListResponse` export)

**Step 1: Copy and extend**

Copy `src/api/client.ts` to `src/shared/api/client.ts`. Add `normalizeListResponse` (from `src/api/index.ts` lines 43-50) as an exported function. Update the import of `useAuthStore` to new path.

```typescript
// At the top, change:
import { useAuthStore } from '@/stores/authStore';
// To:
import { useAuthStore } from '@/shared/stores/authStore';

// At the bottom, after getErrorMessage, add:
import type { ListResponse } from '@/shared/types/common';

export function normalizeListResponse<T>(data: Record<string, unknown>, key: string): ListResponse<T> {
  const items = (data[key] || data.items || data.data || []) as T[];
  const pagination = (data.pagination as ListResponse<T>['pagination']) || {
    limit: 10,
    has_more: false,
  };
  return { items, pagination };
}
```

**Step 2: Commit**

```bash
git add src/shared/api/client.ts
git commit -m "feat: create shared/api/client.ts with normalizeListResponse"
```

---

## Task 5: Create All Feature API Files

**Files:** Create 14 `features/*/api.ts` files by splitting `src/api/index.ts`.

Each feature's api.ts follows this pattern:
```typescript
import apiClient, { normalizeListResponse } from '@/shared/api/client';
import type { ListResponse, PaginationParams } from '@/shared/types/common';
import type { /* domain types */ } from './types';
// Cross-feature type imports if needed:
// import type { Category } from '@/features/categories/types';
```

Copy each API namespace verbatim from `src/api/index.ts`. The exact line ranges are:

| Feature | Source lines | API object name |
|---------|-------------|-----------------|
| auth | `src/api/auth.ts` (whole file) | `authApi` |
| settings | 56-86 | `usersApi` |
| categories | 91-142 | `categoriesApi` |
| products | 147-234 | `productsApi` |
| inventory | 239-264 | `inventoryApi` |
| orders | 269-316 | `ordersApi` |
| customers | 321-361 | `customersApi` |
| artisans | 366-408 | `artisansApi` |
| coupons | 413-464 | `couponsApi` |
| pricing | 469-522 | `pricingApi` |
| notifications | 527-566 | `notificationsApi` |
| analytics | 571-618 | `analyticsApi` |
| bulk | 624-683 | `bulkApi` |
| reports | 689-753 | `reportsApi` |

Plus two API objects that don't have their own feature page:
- `assetsApi` (lines 758-779) → `shared/api/assets.ts`
- `auditApi` (lines 784-816) → `shared/api/audit.ts`

### features/auth/api.ts
```typescript
import type { User } from './types';
import type { LoginRequest } from './types';
import apiClient from '@/shared/api/client';

export const authApi = {
  // Copy verbatim from src/api/auth.ts
};
```

### Import adjustments per feature API file

Each feature api.ts must:
1. Import `apiClient` (and `normalizeListResponse` if it uses list endpoints) from `@/shared/api/client`
2. Import `ListResponse`, `PaginationParams` from `@/shared/types/common` if used
3. Import domain types from `./types`
4. Import cross-feature types from `@/features/<other>/types` as needed

Notable cross-feature API type imports:
- `features/products/api.ts` needs `Inventory`, `InventoryTransaction` from `@/features/inventory/types` (for getInventory, getInventoryTransactions)
- `features/inventory/api.ts` needs `Inventory` from its own types
- `features/customers/api.ts` needs `Order` from `@/features/orders/types` (for getOrders)
- `features/artisans/api.ts` needs `Product` from `@/features/products/types` (for getProducts)
- `shared/api/assets.ts` needs `UploadURLResponse` from `@/shared/types/common`
- `shared/api/audit.ts` needs `AuditLog` from `@/shared/types/common`

**Step: Commit all API files**

```bash
git add src/features/*/api.ts src/shared/api/
git commit -m "feat: split api/index.ts into per-feature API modules"
```

---

## Task 6: Move Shared Infrastructure

**Files to move (with updated imports):**

| From | To |
|------|-----|
| `src/stores/authStore.ts` | `src/shared/stores/authStore.ts` |
| `src/stores/uiStore.ts` | `src/shared/stores/uiStore.ts` |
| `src/stores/__tests__/authStore.test.ts` | `src/shared/stores/__tests__/authStore.test.ts` |
| `src/stores/__tests__/uiStore.test.ts` | `src/shared/stores/__tests__/uiStore.test.ts` |
| `src/hooks/useDebounce.ts` | `src/shared/hooks/useDebounce.ts` |
| `src/hooks/useCursorPagination.ts` | `src/shared/hooks/useCursorPagination.ts` |
| `src/hooks/index.ts` | `src/shared/hooks/index.ts` |
| `src/hooks/__tests__/useDebounce.test.ts` | `src/shared/hooks/__tests__/useDebounce.test.ts` |
| `src/utils/currency.ts` | `src/shared/utils/currency.ts` |
| `src/utils/badge.ts` | `src/shared/utils/badge.ts` |
| `src/utils/chartColors.ts` | `src/shared/utils/chartColors.ts` |

**Step 1: Move files**

```bash
cd handloom-admin-frontend/src
# Stores
cp -r stores/__tests__ shared/stores/
cp stores/authStore.ts shared/stores/
cp stores/uiStore.ts shared/stores/
# Hooks
cp -r hooks/__tests__ shared/hooks/
cp hooks/useDebounce.ts shared/hooks/
cp hooks/useCursorPagination.ts shared/hooks/
cp hooks/index.ts shared/hooks/
# Utils
cp utils/currency.ts shared/utils/
cp utils/badge.ts shared/utils/
cp utils/chartColors.ts shared/utils/
```

**Step 2: Create shared/utils/index.ts barrel**

```typescript
export { formatCurrency } from './currency';
export { getStatusBadgeVariant } from './badge';
export { CHART_COLORS, PIE_COLORS } from './chartColors';
```

**Step 3: Commit**

```bash
git add src/shared/stores/ src/shared/hooks/ src/shared/utils/
git commit -m "feat: move stores, hooks, utils to shared/"
```

---

## Task 7: Move and Split Shared Components

### 7a: Move UI components to shared/components/ui/

**Files to move:**

| From (`components/common/`) | To (`shared/components/ui/`) |
|------|-----|
| Button.tsx | Button.tsx |
| Input.tsx | Input.tsx |
| Select.tsx | Select.tsx |
| Badge.tsx | Badge.tsx |
| Card.tsx | Card.tsx |
| Modal.tsx | Modal.tsx |
| Table.tsx | Table.tsx |
| Tabs.tsx | Tabs.tsx |
| Pagination.tsx | Pagination.tsx |
| ImageUpload.tsx | ImageUpload.tsx |

```bash
cd handloom-admin-frontend/src
for f in Button Input Select Badge Card Modal Table Tabs Pagination ImageUpload; do
  cp components/common/${f}.tsx shared/components/ui/${f}.tsx
done
```

**Create `shared/components/ui/index.ts`** — same exports as current `components/common/index.ts` but WITHOUT Loading exports, ErrorBoundary, or AttributeFilterSidebar. Also remove `getStatusBadgeVariant` proxy re-export (consumers will import from `@/shared/utils/badge` directly).

### 7b: Split Loading.tsx into shared/components/loading/

**Create 4 files from `components/common/Loading.tsx`:**

1. `shared/components/loading/LoadingSpinner.tsx` — lines 1-24 (LoadingSpinner + props)
2. `shared/components/loading/LoadingOverlay.tsx` — lines 26-39 (LoadingOverlay) + lines 194-210 (InlineLoading, LoadingBar). Import LoadingSpinner from `./LoadingSpinner`.
3. `shared/components/loading/PageLoading.tsx` — lines 41-54 (PageLoading) + lines 178-191 (DataLoading). Import LoadingSpinner from `./LoadingSpinner`.
4. `shared/components/loading/Skeleton.tsx` — lines 56-319 (Skeleton, CardSkeleton, TableSkeleton, StatsSkeleton, ChartSkeleton, FormSkeleton, ListItemSkeleton, DashboardSkeleton, TablePageSkeleton)

**Create `shared/components/loading/index.ts`** re-exporting all components and their prop types.

### 7c: Move layout components

```bash
cp components/layout/Header.tsx shared/components/layout/Header.tsx
cp components/layout/Sidebar.tsx shared/components/layout/Sidebar.tsx
cp components/layout/MainLayout.tsx shared/components/layout/MainLayout.tsx
cp components/layout/index.ts shared/components/layout/index.ts
```

### 7d: Move ErrorBoundary

```bash
cp components/common/ErrorBoundary.tsx shared/components/ErrorBoundary.tsx
```

**Step: Update internal imports within shared/components/**

- `shared/components/ui/ImageUpload.tsx` imports `apiClient` from `@/api/client` → change to `@/shared/api/client`
- `shared/components/ui/ImageUpload.tsx` imports types from `@/types` → change to `@/shared/types/common`
- `shared/components/layout/Header.tsx` imports from `@/stores/authStore` → `@/shared/stores/authStore`
- `shared/components/layout/Header.tsx` imports from `@/stores/uiStore` → `@/shared/stores/uiStore`
- `shared/components/layout/Sidebar.tsx` imports from `@/stores/uiStore` → `@/shared/stores/uiStore`

**Step: Commit**

```bash
git add src/shared/components/
git commit -m "feat: move and split shared components (ui, loading, layout)"
```

---

## Task 8: Split App.tsx into app/ Directory

**Files:**
- Create: `src/app/providers.tsx`
- Create: `src/app/routes.tsx`
- Create: `src/app/App.tsx`

### app/providers.tsx

```typescript
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { Toaster } from 'react-hot-toast';

import { ErrorBoundary } from '@/shared/components/ErrorBoundary';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 5 * 60 * 1000,
    },
  },
});

export function Providers({ children }: { children: ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <ErrorBoundary>
        {children}
      </ErrorBoundary>
      <Toaster
        position="top-right"
        toastOptions={{
          duration: 4000,
          className: 'bg-white text-gray-800 shadow-lg rounded-lg px-4 py-3',
          success: {
            iconTheme: { primary: '#10b981', secondary: '#fff' },
          },
          error: {
            iconTheme: { primary: '#ef4444', secondary: '#fff' },
          },
        }}
      />
    </QueryClientProvider>
  );
}
```

### app/routes.tsx

```typescript
import type { ComponentType } from 'react';
import { lazy, Suspense } from 'react';
import { Navigate, Outlet, Route, Routes } from 'react-router-dom';

import { LoadingOverlay, PageLoading } from '@/shared/components/loading';
import { MainLayout } from '@/shared/components/layout';
import { useAuthStore } from '@/shared/stores/authStore';
// Eagerly loaded (critical path)
import { LoginPage } from '@/features/auth';

function withSuspense<P extends object>(LazyComponent: ComponentType<P>) {
  return function SuspenseWrapper(props: P) {
    return (
      <Suspense fallback={<PageLoading message="Loading page..." />}>
        <LazyComponent {...props} />
      </Suspense>
    );
  };
}

// Lazy loaded feature pages
const Dashboard = withSuspense(lazy(() => import('@/features/dashboard').then((m) => ({ default: m.DashboardPage }))));
const Categories = withSuspense(lazy(() => import('@/features/categories').then((m) => ({ default: m.CategoriesPage }))));
const Products = withSuspense(lazy(() => import('@/features/products').then((m) => ({ default: m.ProductsPage }))));
const Orders = withSuspense(lazy(() => import('@/features/orders').then((m) => ({ default: m.OrdersPage }))));
const OrderDetail = withSuspense(lazy(() => import('@/features/orders').then((m) => ({ default: m.OrderDetailPage }))));
const Customers = withSuspense(lazy(() => import('@/features/customers').then((m) => ({ default: m.CustomersPage }))));
const Artisans = withSuspense(lazy(() => import('@/features/artisans').then((m) => ({ default: m.ArtisansPage }))));
const PricingRules = withSuspense(lazy(() => import('@/features/pricing').then((m) => ({ default: m.PricingRulesPage }))));
const Coupons = withSuspense(lazy(() => import('@/features/coupons').then((m) => ({ default: m.CouponsPage }))));
const Inventory = withSuspense(lazy(() => import('@/features/inventory').then((m) => ({ default: m.InventoryPage }))));
const Analytics = withSuspense(lazy(() => import('@/features/analytics').then((m) => ({ default: m.AnalyticsPage }))));
const Reports = withSuspense(lazy(() => import('@/features/reports').then((m) => ({ default: m.ReportsPage }))));
const Notifications = withSuspense(lazy(() => import('@/features/notifications').then((m) => ({ default: m.NotificationsPage }))));
const BulkOperations = withSuspense(lazy(() => import('@/features/bulk').then((m) => ({ default: m.BulkOperationsPage }))));
const Users = withSuspense(lazy(() => import('@/features/settings').then((m) => ({ default: m.UsersPage }))));
const Settings = withSuspense(lazy(() => import('@/features/settings').then((m) => ({ default: m.SettingsPage }))));
const NotFound = withSuspense(lazy(() => import('@/shared/components/NotFoundPage').then((m) => ({ default: m.NotFoundPage }))));

function ProtectedRoute() {
  const { isAuthenticated, isLoading } = useAuthStore();
  if (isLoading) return <LoadingOverlay message="Loading..." />;
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return <Outlet />;
}

function AdminRoute() {
  const { user } = useAuthStore();
  if (user?.role !== 'ADMIN') return <Navigate to="/" replace />;
  return <Outlet />;
}

function PublicRoute() {
  const { isAuthenticated, isLoading } = useAuthStore();
  if (isLoading) return <LoadingOverlay message="Loading..." />;
  if (isAuthenticated) return <Navigate to="/" replace />;
  return <Outlet />;
}

export function AppRoutes() {
  return (
    <Routes>
      <Route element={<PublicRoute />}>
        <Route path="/login" element={<LoginPage />} />
      </Route>
      <Route element={<ProtectedRoute />}>
        <Route element={<MainLayout />}>
          <Route path="/" element={<Dashboard />} />
          <Route path="/categories" element={<Categories />} />
          <Route path="/products" element={<Products />} />
          <Route path="/inventory" element={<Inventory />} />
          <Route path="/orders" element={<Orders />} />
          <Route path="/orders/:id" element={<OrderDetail />} />
          <Route path="/customers" element={<Customers />} />
          <Route path="/artisans" element={<Artisans />} />
          <Route path="/pricing" element={<PricingRules />} />
          <Route path="/coupons" element={<Coupons />} />
          <Route path="/analytics" element={<Analytics />} />
          <Route path="/reports" element={<Reports />} />
          <Route path="/bulk" element={<BulkOperations />} />
          <Route path="/notifications" element={<Notifications />} />
          <Route element={<AdminRoute />}>
            <Route path="/users" element={<Users />} />
          </Route>
          <Route path="/settings" element={<Settings />} />
          <Route path="/settings/*" element={<Settings />} />
        </Route>
      </Route>
      <Route path="*" element={<NotFound />} />
    </Routes>
  );
}
```

Note: Move `NotFoundPage.tsx` to `src/shared/components/NotFoundPage.tsx` since it's not a feature.

### app/App.tsx

```typescript
import { useEffect } from 'react';
import { BrowserRouter } from 'react-router-dom';

import { authApi } from '@/features/auth/api';
import { useAuthStore } from '@/shared/stores/authStore';
import { Providers } from './providers';
import { AppRoutes } from './routes';

function App() {
  useEffect(() => {
    const { login, logout } = useAuthStore.getState();
    authApi
      .getCurrentUser()
      .then((user) => login(user))
      .catch(() => logout());
  }, []);

  return (
    <Providers>
      <BrowserRouter>
        <AppRoutes />
      </BrowserRouter>
    </Providers>
  );
}

export default App;
```

**Update `src/main.tsx`** to import from `./app/App` instead of `./App`.

**Step: Commit**

```bash
git add src/app/ src/main.tsx
git commit -m "feat: split App.tsx into app/ directory (providers, routes, shell)"
```

---

## Task 9: Move Page Files into Features

Move each page's component files into `features/<domain>/components/`. Update imports in each file.

### Import rewrite rules for all page files:

| Old import | New import |
|-----------|-----------|
| `from '@/api'` | `from '@/features/<own-domain>/api'` (or cross-feature: `from '@/features/<other>/api'`) |
| `from '@/api/client'` | `from '@/shared/api/client'` |
| `from '@/components/common'` | `from '@/shared/components/ui'` and/or `from '@/shared/components/loading'` |
| `from '@/hooks'` | `from '@/shared/hooks'` |
| `from '@/types'` | `from './types'` or `from '@/features/<other>/types'` or `from '@/shared/types/common'` |
| `from '@/utils/currency'` | `from '@/shared/utils/currency'` |
| `from '@/utils/chartColors'` | `from '@/shared/utils/chartColors'` |
| `from '@/utils/badge'` | `from '@/shared/utils/badge'` |
| `from '@/stores/authStore'` | `from '@/shared/stores/authStore'` |
| `from '@/stores/uiStore'` | `from '@/shared/stores/uiStore'` |
| `from './ProductFormModal'` | `from './ProductFormModal'` (unchanged — relative) |

### Per-feature file moves:

**features/auth/**
- `pages/auth/LoginPage.tsx` → `features/auth/components/LoginPage.tsx`
- Import rewrites: `@/api` → `@/features/auth/api`, `@/components/common` → `@/shared/components/ui`, `@/stores/authStore` → `@/shared/stores/authStore`

**features/products/**
- `pages/products/ProductsPage.tsx` → `features/products/components/ProductsPage.tsx`
- `pages/products/ProductFormModal.tsx` → `features/products/components/ProductFormModal/ProductFormModal.tsx`
- `components/common/AttributeFilterSidebar.tsx` → `features/products/components/AttributeFilterSidebar.tsx`
- Import rewrites for ProductsPage: `@/api` → split between `@/features/products/api` and `@/features/categories/api`, `@/types` → `@/features/categories/types` (for CategoryAttribute) + `@/features/products/types` (for Product)
- Import rewrites for ProductFormModal: `@/api` → `@/features/products/api`, `@/features/categories/api`, `@/features/artisans/api`; `@/types` → `@/features/categories/types` + `@/features/products/types`

**features/orders/**
- `pages/orders/OrdersPage.tsx` → `features/orders/components/OrdersPage.tsx`
- `pages/orders/OrderDetailPage.tsx` → `features/orders/components/OrderDetailPage/OrderDetailPage.tsx`

**features/categories/**
- `pages/categories/CategoriesPage.tsx` → `features/categories/components/CategoriesPage.tsx`
- `pages/categories/CategoryFormModal.tsx` → `features/categories/components/CategoryFormModal.tsx`

**features/customers/**
- `pages/customers/CustomersPage.tsx` → `features/customers/components/CustomersPage.tsx`
- `pages/customers/CustomerFormModal.tsx` → `features/customers/components/CustomerFormModal.tsx`

**features/artisans/**
- `pages/artisans/ArtisansPage.tsx` → `features/artisans/components/ArtisansPage.tsx`
- `pages/artisans/ArtisanFormModal.tsx` → `features/artisans/components/ArtisanFormModal.tsx`

**features/coupons/**
- `pages/coupons/CouponsPage.tsx` → `features/coupons/components/CouponsPage.tsx`
- `pages/coupons/CouponFormModal.tsx` → `features/coupons/components/CouponFormModal.tsx`

**features/pricing/**
- `pages/pricing/PricingRulesPage.tsx` → `features/pricing/components/PricingRulesPage.tsx`
- `pages/pricing/PricingRuleFormModal.tsx` → `features/pricing/components/PricingRuleFormModal.tsx`

**features/inventory/**
- `pages/inventory/InventoryPage.tsx` → `features/inventory/components/InventoryPage.tsx`
- `pages/inventory/StockAdjustmentModal.tsx` → `features/inventory/components/StockAdjustmentModal.tsx`

**features/dashboard/**
- `pages/dashboard/DashboardPage.tsx` → `features/dashboard/components/DashboardPage.tsx`
- Import rewrites: `@/api` → `@/features/analytics/api`, `@/features/inventory/api`, `@/features/orders/api`

**features/analytics/**
- `pages/analytics/AnalyticsPage.tsx` → `features/analytics/components/AnalyticsPage.tsx`

**features/reports/**
- `pages/reports/ReportsPage.tsx` → `features/reports/components/ReportsPage.tsx`

**features/bulk/**
- `pages/bulk/BulkOperationsPage.tsx` → `features/bulk/components/BulkOperationsPage.tsx`
- `pages/bulk/BulkImportModal.tsx` → `features/bulk/components/BulkImportModal.tsx`
- `pages/bulk/BulkExportModal.tsx` → `features/bulk/components/BulkExportModal.tsx`

**features/notifications/**
- `pages/notifications/NotificationsPage.tsx` → `features/notifications/components/NotificationsPage.tsx`

**features/settings/**
- `pages/settings/SettingsPage.tsx` → `features/settings/components/SettingsPage.tsx`
- `pages/settings/UsersPage.tsx` → `features/settings/components/UsersPage.tsx`
- `pages/settings/UserFormModal.tsx` → `features/settings/components/UserFormModal.tsx`
- Import rewrites for SettingsPage: `@/api` → split between `@/features/auth/api`, `@/features/settings/api`
- Import rewrites for UsersPage/UserFormModal: `@/api` → `@/features/settings/api`, `@/types` → `@/features/auth/types`

**Step: Commit**

```bash
git add src/features/
git commit -m "feat: move all page components into features/<domain>/components/"
```

---

## Task 10: Create Feature Barrel Exports

Each feature needs an `index.ts` that exports page components for lazy loading in routes.tsx.

```typescript
// features/auth/index.ts
export { LoginPage } from './components/LoginPage';

// features/products/index.ts
export { ProductsPage } from './components/ProductsPage';

// features/orders/index.ts
export { OrdersPage } from './components/OrdersPage';
export { OrderDetailPage } from './components/OrderDetailPage';

// features/categories/index.ts
export { CategoriesPage } from './components/CategoriesPage';

// features/customers/index.ts
export { CustomersPage } from './components/CustomersPage';

// features/artisans/index.ts
export { ArtisansPage } from './components/ArtisansPage';

// features/coupons/index.ts
export { CouponsPage } from './components/CouponsPage';

// features/pricing/index.ts
export { PricingRulesPage } from './components/PricingRulesPage';

// features/inventory/index.ts
export { InventoryPage } from './components/InventoryPage';

// features/dashboard/index.ts
export { DashboardPage } from './components/DashboardPage';

// features/analytics/index.ts
export { AnalyticsPage } from './components/AnalyticsPage';

// features/reports/index.ts
export { ReportsPage } from './components/ReportsPage';

// features/bulk/index.ts
export { BulkOperationsPage } from './components/BulkOperationsPage';

// features/notifications/index.ts
export { NotificationsPage } from './components/NotificationsPage';

// features/settings/index.ts
export { SettingsPage } from './components/SettingsPage';
export { UsersPage } from './components/UsersPage';
```

**Step: Commit**

```bash
git add src/features/*/index.ts
git commit -m "feat: add barrel exports for all feature modules"
```

---

## Task 11: Split Large Components

### 11a: ProductFormModal → subdirectory

Read `features/products/components/ProductFormModal/ProductFormModal.tsx` and extract the attribute rendering section into `AttributeFields.tsx`.

`AttributeFields.tsx` should accept:
- `attributes: CategoryAttribute[]`
- `control: Control<FormData>` (from react-hook-form)
- Render the dynamic form fields for each attribute type (SELECT, MULTI_SELECT, TEXT, NUMBER, BOOLEAN, DIMENSION, DIMENSION_RANGE)

Create `features/products/components/ProductFormModal/index.ts`:
```typescript
export { ProductFormModal } from './ProductFormModal';
```

### 11b: OrderDetailPage → subdirectory

Read `features/orders/components/OrderDetailPage/OrderDetailPage.tsx` and extract:

1. **OrderNotes.tsx** — The notes list and "add note" form
2. **OrderTimeline.tsx** — The status history/timeline section

Create `features/orders/components/OrderDetailPage/index.ts`:
```typescript
export { OrderDetailPage } from './OrderDetailPage';
```

**Step: Commit**

```bash
git add src/features/products/components/ProductFormModal/
git add src/features/orders/components/OrderDetailPage/
git commit -m "refactor: split ProductFormModal and OrderDetailPage into sub-components"
```

---

## Task 12: Delete Old Structure and Dead Code

**Step 1: Delete dead code**

```bash
cd handloom-admin-frontend/src
rm utils/api.ts
rm utils/__tests__/api.test.ts
rm assets/react.svg
rmdir components/forms 2>/dev/null || true
rmdir components/tables 2>/dev/null || true
rmdir contexts 2>/dev/null || true
```

**Step 2: Delete old directories (now replaced by shared/ and features/)**

```bash
rm -rf api/
rm -rf types/
rm -rf components/
rm -rf pages/
rm -rf stores/
rm -rf hooks/
rm -rf utils/
rm -rf assets/
rm App.tsx
```

**Step 3: Commit**

```bash
git add -A
git commit -m "chore: delete old directory structure and dead code"
```

---

## Task 13: Verify Build

**Step 1: Run full check**

```bash
cd handloom-admin-frontend
npm run check
```

Run: `npm run check` (typecheck + lint + format:check)
Expected: PASS with zero errors

**Step 2: Fix any import errors**

If typecheck fails, the errors will be import-related. Fix each one by checking the import rewrite rules from Task 9.

**Step 3: Run lint:fix + format**

```bash
npm run lint:fix && npm run format
```

**Step 4: Run tests**

```bash
npm run test
```

Expected: All 5 existing test files pass (they moved to shared/ but paths updated).

**Step 5: Run dev server**

```bash
npm run dev
```

Manually verify the app loads and navigate to a few pages to confirm lazy loading works.

**Step 6: Final commit**

```bash
git add -A
git commit -m "fix: resolve import errors from directory restructure"
```

---

## Task 14: Final Cleanup Commit

**Step 1: Squash or rebase if desired**

All commits from this migration can optionally be squashed into a single commit for a cleaner history:

```
refactor: restructure frontend to feature-based architecture

- Split api/index.ts (816 lines) into 14 feature-scoped API modules
- Split types/index.ts (681 lines) into per-feature type files
- Split Loading.tsx (319 lines) into 4 focused files
- Split App.tsx (224 lines) into app/ (providers, routes, shell)
- Split ProductFormModal (722 lines) into sub-components
- Split OrderDetailPage (535 lines) into sub-components
- Move shared code to shared/ (components/ui, loading, layout, hooks, stores, utils)
- Delete dead code (extractItems util, empty dirs, unused react.svg)
- All imports updated to feature-based @/ paths
```

**Step 2: Verify one final time**

```bash
npm run check && npm run test
```
