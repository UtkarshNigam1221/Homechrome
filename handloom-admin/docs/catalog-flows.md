# Catalog Flows: End-to-End Reference

Complete trace of every category, product, and inventory flow across both frontends,
the backend handler/service/repository layers, and the in-process cache.

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Storefront Flows](#2-storefront-flows)
3. [Admin Frontend Flows](#3-admin-frontend-flows)
4. [Backend Layers](#4-backend-layers)
5. [Caching](#5-caching)
6. [Attribute Filtering Deep Dive](#6-attribute-filtering-deep-dive)
7. [Inventory & Stock](#7-inventory--stock)
8. [Known Gotchas](#8-known-gotchas)

---

## 1. Architecture Overview

```
Storefront (Next.js)          Admin SPA (React/Vite)
  |                              |
  | SSR: fetch()                 | Axios → /admin/*
  | Client: Axios → /api/*      |
  |   (Next.js rewrites)        | (Vite proxy in dev)
  v                              v
─────────────── Backend (Go / Chi) ──────────────
  Handler (store/ or admin)
    → Service
      → CachedRepo (in-process go-cache)
        → PostgreSQL repo
          → PostgreSQL (categories, products, inventory tables)
```

**Key points:**
- Storefront SSR uses raw `fetch()` with `NEXT_PUBLIC_API_URL` (server-to-server).
- Storefront client components use Axios with empty baseURL → Next.js rewrites
  `/api/:path*` → `${NEXT_PUBLIC_API_URL}/api/:path*`.
- Admin SPA uses Axios with Vite proxy `/admin/*` → `localhost:8081` in dev.
- Both Axios clients auto-unwrap the `{success, data}` envelope.
- All prices are in **paise** (1 INR = 100 paise).
- Pagination is cursor-based (base64-encoded integer offset).

---

## 2. Storefront Flows

### 2.1 Categories List — `/categories`

```
page.tsx (SSR)
  fetch GET ${API_BASE}/api/v1/store/catalog/categories
    revalidate: 60s (ISR)
  → renders CategoryCard grid
```

| Layer | What happens |
|-------|-------------|
| **Handler** `store/catalog_handler.go:ListCategories` | Forces `Status=ACTIVE`, parses pagination + search, calls `categoryService.List(ctx, req)` |
| **Service** `category_service.go:List` | Pass-through to repo |
| **CachedCategoryRepo** | Cache key `cat:list` (only for unfiltered first page: status=nil, search="", cursor=""). TTL 1h |
| **CategoryRepo** `List` | `SELECT * FROM categories WHERE status=$1 [AND name ILIKE ...] ORDER BY created_at DESC LIMIT n+1 OFFSET m` |
| **Response** | `[]*StoreCategory` — includes `own_attributes` field but **List does NOT load attributes from DB** (only `GetByID` does) |

**Important:** `CategoryRepository.List()` does NOT call `fetchAttributes()`. Categories returned from `List` will have `own_attributes: null`. This is fine for the categories grid page (no attributes needed), but the FilterSidebar on category detail pages needs attributes — it gets them via the category fetched by `GetByID` in the slug lookup.

### 2.2 Category Detail — `/c/[slug]`

```
page.tsx (SSR)
  1. fetch GET ${API_BASE}/api/v1/store/catalog/categories/${slug}  → Category
  2. fetch GET ${API_BASE}/api/v1/store/catalog/products?category_id=${id}&filters...  → Product[]
  → renders <CategoryProductsView category={cat} products={products} />

CategoryProductsView.tsx (Client)
  On mount:
    3. GET /api/v1/store/catalog/products/filter-options/${category.id}  → filterOptions
  On filter change (debounced 300ms):
    4. GET /api/v1/store/catalog/products?category_id=${id}&filters...  → Product[]
    5. router.push() updates URL search params
```

**Step 1 — Get category by slug:**

| Layer | What happens |
|-------|-------------|
| **Handler** `store/catalog_handler.go:GetCategory` | Detects non-UUID → calls `findCategoryBySlug()` |
| **findCategoryBySlug** | Calls `categoryService.List(ctx, {Slug: slug, Status: ACTIVE, Limit: 1})` |
| **CategoryRepo.List** | `WHERE slug = $1 AND status = 'ACTIVE' LIMIT 2` — returns category **without** `own_attributes` |
| **Handler (continued)** | Re-fetches via `categoryService.GetByID(ctx, found.ID)` to include attributes |
| **CachedCategoryRepo.GetByID** | Cache key `cat:{id}`, TTL 1h. On miss: `SELECT * FROM categories WHERE id=$1` + `fetchAttributes()` |
| **Response** | `StoreCategory` with `own_attributes` populated |

**Step 2 — Get products (SSR, with filter params from URL):**

| Layer | What happens |
|-------|-------------|
| **page.tsx** | Reads `searchParams`, forwards `min_price`, `max_price`, `in_stock`, `attribute_filters` to fetch URL |
| **Handler** `store/catalog_handler.go:ListProducts` | Forces `Status=ACTIVE`, parses all query params including `in_stock` and `attribute_filters` JSON |
| **Service** `product_service.go:List` | Pass-through to repo |
| **CachedProductRepo** | Cache key `prod:list:{md5(request)}`, TTL 1h |
| **ProductRepo.List** | Complex query — see [Section 6](#6-attribute-filtering-deep-dive) |
| **Response** | `[]*StoreProduct` — `CostPrice` stripped, `InStock = AvailableQty > 0` |

**Step 3 — Fetch filter options (client-side, on mount):**

| Layer | What happens |
|-------|-------------|
| **Client** | `api.get(/api/v1/store/catalog/products/filter-options/${categoryId})` |
| **Handler** `store/catalog_handler.go:GetFilterOptions` | Calls `productService.GetAttributeFilterOptions(ctx, categoryID)` |
| **Service** `product_service.go:GetAttributeFilterOptions` | Gets category via `categoryRepo.GetByID` → collects `attr.Name` where `attr.Searchable == true` → calls `productRepo.GetAttributeFilterOptions(ctx, categoryID, attrNames)` → sorts values |
| **CachedProductRepo** | Cache key `prod:attr:{categoryId}`, TTL 1h |
| **ProductRepo** | Per attribute: `SELECT DISTINCT attribute_value FROM product_attribute_values WHERE product_id IN (SELECT id FROM products WHERE category_id=$1) AND attribute_name=$2 ORDER BY attribute_value` |
| **Response** | `map[string][]string` e.g. `{"color":["blue","green"],"material":["cotton","silk"]}` |

**Step 4 — FilterSidebar renders attribute checkboxes:**

The `FilterSidebar` component requires **both**:
- `filterOptions` (from step 3) — the available values
- `categoryAttributes` (from `category.own_attributes`) — metadata (label, searchable, display_order)

It filters to `attr.searchable && filterOptions?.[attr.name]?.length > 0` and renders checkbox groups.

**Step 5 — On filter change (client-side re-fetch):**

```
handleFiltersChange(newFilters)
  → setFilters(newFilters)
  → router.push(`/c/${slug}?${filtersToParams(newFilters)}`)

useEffect([filters])  (debounced 300ms, with AbortController)
  → api.get(/api/v1/store/catalog/products?category_id=${id}&min_price=...&in_stock=...&attribute_filters=...)
  → setProducts(response)
```

URL params: `min_price`, `max_price`, `in_stock`, `attribute_filters` (JSON string).

### 2.3 All Products — `/products`

```
page.tsx (SSR)
  fetch GET ${API_BASE}/api/v1/store/catalog/products?search=${q}&min_price=...&...
  → renders <ProductsView products={products} initialSearch={search} />

ProductsView.tsx (Client)
  On filter change (debounced 300ms):
    GET /api/v1/store/catalog/products?search=${q}&min_price=...&in_stock=...
    router.push() updates URL
  On search submit:
    router.push(/products?search=${q}&filters...)
```

Same backend flow as category products, but:
- No `category_id` param (all categories)
- No attribute filters (no single category context)
- Has search bar with `search` query param
- FilterSidebar has NO `filterOptions` or `categoryAttributes` props (only price + in-stock)

### 2.4 Product Detail — `/p/[slug]`

```
page.tsx (SSR)
  fetch GET ${API_BASE}/api/v1/store/catalog/products/${slug}
  → renders <ProductDetailView product={product} />
```

| Layer | What happens |
|-------|-------------|
| **Handler** `store/catalog_handler.go:GetProduct` | Non-UUID → `findProductBySlug()` (List with Slug filter, Limit 1) → re-fetch via `GetByID` for full relations |
| **Service** `product_service.go:GetByID` | Fetches product, category summary, inventory → returns `ProductWithRelations` |
| **CachedProductRepo.GetByID** | Cache key `prod:{id}`, TTL 1h |
| **ProductRepo.GetByID** | `SELECT p.*, COALESCE(i.*) FROM products p LEFT JOIN inventory i WHERE p.id=$1` + `loadProductRelations()` (batch attributes + images) |
| **Response** | `StoreProduct` with category summary, images, attributes, `in_stock` boolean |

### 2.5 Product Availability — `/products/{id}/availability`

```
ProductDetailView.tsx or Cart (client)
  GET /api/v1/store/catalog/products/${id}/availability
```

| Layer | What happens |
|-------|-------------|
| **Handler** `store/catalog_handler.go:CheckAvailability` | Verifies product exists + ACTIVE, then calls `inventoryService.GetByProductID(ctx, id)` |
| **InventoryRepo** | `SELECT * FROM inventory WHERE product_id=$1` |
| **Fallback** | If inventory record not found, uses denormalized `Product.AvailableQty` |
| **Response** | `{in_stock: bool, available_quantity: int}` |

---

## 3. Admin Frontend Flows

### 3.1 Categories CRUD

**API module:** `handloom-admin-frontend/src/features/categories/api.ts`

| Action | Endpoint | React Query Key | Invalidates |
|--------|----------|-----------------|-------------|
| List | `GET /admin/categories?limit&cursor&status` | `['categories', {params}]` | — |
| Get | `GET /admin/categories/{id}` | — | — |
| Create | `POST /admin/categories` | — | `['categories']` |
| Update | `PATCH /admin/categories/{id}` | — | `['categories']` |
| Delete | `DELETE /admin/categories/{id}` | — | `['categories']` |
| Get Attributes | `GET /admin/categories/{id}/attributes` | `['category-attributes', id]` | — |
| Add Attribute | `POST /admin/categories/{id}/attributes` | — | `['categories']` |
| Update Attribute | `PATCH /admin/categories/{id}/attributes/{name}` | — | `['categories']` |
| Delete Attribute | `DELETE /admin/categories/{id}/attributes/{name}` | — | `['categories']` |

**Backend path:** `handler/category_handler.go` → `service/category_service.go` → `CachedCategoryRepo` → `CategoryRepo` (PostgreSQL).

**Admin handler differences from store:**
- No `Status=ACTIVE` filter forced — admin sees all statuses
- Delete validated: fails if `category.ProductCount > 0`

### 3.2 Products CRUD + Filtering

**API module:** `handloom-admin-frontend/src/features/products/api.ts`

| Action | Endpoint | React Query Key | Invalidates |
|--------|----------|-----------------|-------------|
| List | `GET /admin/products?limit&cursor&category_id&status&search&min_price&max_price&in_stock&low_stock&attribute_filters` | `['products', {params}]` | — |
| Get | `GET /admin/products/{id}` | — | — |
| Create | `POST /admin/products` | — | `['products']` |
| Update | `PATCH /admin/products/{id}` | — | `['products']` |
| Delete | `DELETE /admin/products/{id}` | — | `['products']` |
| Filter Options | `GET /admin/products/filter-options/{categoryId}` | `['product-filter-options', categoryId]` | — |
| Reorder | `PUT /admin/products/categories/{categoryId}/reorder` | — | `['products']`, `['products-for-ranking']` |

**Admin list has extra filter:** `low_stock` — adds `WHERE i.available_qty <= i.low_stock_threshold` to SQL.

**Admin sees all statuses** — no forced `Status=ACTIVE` like the store handler.

**`attribute_filters` param:** JSON-stringified on the frontend before sending:
```ts
if (params.attribute_filters && Object.keys(params.attribute_filters).length > 0) {
  apiParams.attribute_filters = JSON.stringify(params.attribute_filters);
}
```

### 3.3 Inventory Management

**API module:** `handloom-admin-frontend/src/features/inventory/api.ts` and `features/products/api.ts`

| Action | Endpoint | Invalidates |
|--------|----------|-------------|
| Get Inventory | `GET /admin/products/{id}/inventory` | — |
| Add Stock | `POST /admin/products/{id}/inventory/add` | `['products']`, `['products-inventory']`, `['low-stock']` |
| Remove Stock | `POST /admin/products/{id}/inventory/remove` | same |
| Adjust Stock | `POST /admin/products/{id}/inventory/adjust` | same |
| Transactions | `GET /admin/products/{id}/inventory/transactions?limit&cursor` | — |
| Low Stock List | `GET /admin/inventory/low-stock?limit&cursor` | `['low-stock']` |

---

## 4. Backend Layers

### 4.1 Handler Layer

**Store handlers** (`internal/handler/store/catalog_handler.go`):
- Force `Status = ACTIVE` on all list/get queries
- Strip `CostPrice` from product responses (→ `StoreProduct`)
- Replace raw inventory qty with `InStock` boolean (`AvailableQty > 0`)
- HTTP cache header: `Cache-Control: public, max-age=3600`

**Admin handlers** (`internal/handler/product_handler.go`, `category_handler.go`):
- No status restriction (admin sees all)
- Return full product data including `CostPrice`, raw inventory quantities
- Auth required (JWT middleware)

### 4.2 Service Layer

**CategoryService** (`internal/service/category_service.go`):
- `Create`: generates slug, finalizes image URL (S3 tmp → assets), inserts
- `Update`: updates fields + replaces all attributes (delete + re-insert in tx)
- `Delete`: validates `ProductCount == 0` before delete
- `List`: pass-through to repo

**ProductService** (`internal/service/product_service.go`):
- `Create`: validates category exists, validates required attributes, finalizes images, creates product + inventory atomically, increments category product count, publishes `product.created` event
- `Update`: validates required attributes, replaces attribute values + images, publishes `product.updated`
- `Delete`: cascade deletes, decrements category count, publishes `product.deleted`
- `List`: pass-through to repo
- `GetByID`: fetches product, joins category summary + inventory → `ProductWithRelations`
- `GetAttributeFilterOptions`: gets category → collects searchable attr names → queries distinct values from `product_attribute_values`
- `ReorderProducts`: validates all IDs in category, assigns `sort_order = 1..n`, batch updates

**InventoryService** (`internal/service/inventory_service.go`):
- All mutations use `SELECT ... FOR UPDATE` row locking
- Publishes events: `inventory.restocked`, `inventory.out_of_stock`, `inventory.low_stock`
- Cache invalidation: calls `cache.DeletePrefix("prod:")` after any stock mutation

### 4.3 Repository Layer

**CategoryRepository** (`internal/repository/postgres/category_repository.go`):
- `GetByID`: loads category + calls `fetchAttributes()` (joins `category_attributes` + `category_attribute_options`)
- `List`: loads categories **without** attributes (no `fetchAttributes` call)
- `Create`/`Update`: transactional — insert/update category + delete old attributes + insert new attributes + options

**ProductRepository** (`internal/repository/postgres/product_repository.go`):
- All product queries `LEFT JOIN inventory` to include live stock data
- `List`: complex query builder — see [Section 6](#6-attribute-filtering-deep-dive)
- `GetByID`: single product query + `loadProductRelations()` (batch loads `product_attribute_values` + `product_images`)
- `Create`: transactional — insert product + attribute values + images + inventory record
- `Update`: transactional — update product + delete/re-insert attribute values + delete/re-insert images
- Hardcoded attributes (`material`, `color`, `weave_type`, `origin`, `craft_type`) are stored as both:
  - Direct fields on `Product` struct (set from first attribute value on load)
  - Rows in `product_attribute_values` table

**InventoryRepository** (`internal/repository/postgres/inventory_repository.go`):
- `AddStock`: tx → lock row → `quantity += n`, `available = quantity - reserved` → insert transaction log
- `RemoveStock`: tx → lock row → validate `available >= n` → `quantity -= n` → insert transaction
- `AdjustStock`: tx → lock row → `quantity = n`, `available = n - reserved` → insert transaction
- `ReserveStock`: tx → lock row → validate `available >= n` → `reserved += n`, `available -= n` → insert transaction
- `ReleaseStock`: tx → lock row → validate `reserved >= n` → `reserved -= n`, `available += n` → insert transaction

---

## 5. Caching

All caching is **in-process** using `go-cache` (TTL-based, with prefix deletion). Runs per-Lambda-instance (no shared cache across instances).

### 5.1 Category Cache

**File:** `internal/repository/postgres/cached_category_repo.go`

| Key Pattern | Value | TTL | When Set | When Invalidated |
|-------------|-------|-----|----------|------------------|
| `cat:{id}` | `*Category` (with attributes) | 1h | `GetByID` cache miss | Create, Update, Delete, IncrementProductCount |
| `cat:list` | `*ListCategoriesResponse` | 1h | `List` cache miss (only unfiltered first page: status=nil, search="", cursor="") | Create, Update, Delete, IncrementProductCount |

### 5.2 Product Cache

**File:** `internal/repository/postgres/cached_product_repo.go`

| Key Pattern | Value | TTL | When Set | When Invalidated |
|-------------|-------|-----|----------|------------------|
| `prod:{id}` | `*Product` | 1h | `GetByID` cache miss | Update, Delete |
| `prod:sku:{sku}` | `*Product` | 1h | `GetBySKU` cache miss | Update, Delete |
| `prod:cat:{categoryId}:all` | `[]*Product` | 1h | `GetByCategoryAll` cache miss | Any product CRUD in that category |
| `prod:attr:{categoryId}` | `map[string][]string` | 1h | `GetAttributeFilterOptions` cache miss | Any product CRUD in that category |
| `prod:list:{md5}` | `*ListProductsResponse` | 1h | `List` cache miss (all queries cached) | Any product CRUD (all `prod:list:*` keys deleted) |

**List cache key generation:**
- Canonical JSON of all `ListProductsRequest` fields (attribute filter values sorted for determinism)
- MD5 hash of JSON → `prod:list:{hex}`

### 5.3 Inventory Cache Invalidation

Inventory mutations (Add/Remove/Adjust) call `cache.DeletePrefix("prod:")` — this invalidates ALL product cache keys including individual products, lists, and filter options. This is conservative but prevents stale `in_stock` data.

### 5.4 Cache Flow Diagram

```
Request → CachedRepo.GetByID("prod_123")
  ├─ cache.Get("prod:prod_123") → HIT → return cached
  └─ cache.Get("prod:prod_123") → MISS
       → ProductRepo.GetByID("prod_123") → SELECT + loadRelations
       → cache.Set("prod:prod_123", product, 1h)
       → return product

Mutation → CachedRepo.Update(product)
  → ProductRepo.Update(product)  (SQL transaction)
  → cache.Delete("prod:{id}")
  → cache.Delete("prod:sku:{sku}")
  → cache.DeletePrefix("prod:cat:{categoryId}")  (all category-specific keys)
  → cache.Delete("prod:attr:{categoryId}")
  → cache.DeletePrefix("prod:list:")  (ALL list cache keys)
```

---

## 6. Attribute Filtering Deep Dive

### 6.1 How Attributes Are Stored

```
categories
  └── category_attributes (name, label, type, required, searchable, display_order)
        └── category_attribute_options (value, label, surcharge)

products
  └── product_attribute_values (product_id, attribute_name, attribute_value)
  └── product_images (product_id, url, alt_text, sort_order)
  └── inventory (product_id, quantity, reserved_qty, available_qty, ...)
```

Hardcoded attributes (`material`, `color`, `weave_type`, `origin`, `craft_type`) exist as:
1. Direct columns on `Product` struct (set from first `product_attribute_values` row on load)
2. Rows in `product_attribute_values` table (for filtering via EXISTS subqueries)

### 6.2 ProductRepository.List — Query Building

**File:** `internal/repository/postgres/product_repository.go:List`

```sql
SELECT p.*, COALESCE(i.quantity, 0) as quantity, ...
FROM products p
LEFT JOIN inventory i ON i.product_id = p.id
WHERE
  -- Static filters (AND-ed, each optional)
  [p.category_id = $1]
  [AND p.status = $2]
  [AND p.slug = $3]
  [AND p.selling_price >= $min]
  [AND p.selling_price <= $max]
  [AND i.available_qty > 0]                          -- in_stock=true
  [AND i.available_qty <= i.low_stock_threshold]     -- low_stock=true (admin only)

  -- Full-text search (optional)
  [AND p.search_vector @@ websearch_to_tsquery('english', $search)]

  -- Attribute filters (one EXISTS per attribute, AND-ed)
  [AND EXISTS (
    SELECT 1 FROM product_attribute_values v
    WHERE v.product_id = p.id
    AND v.attribute_name = 'material'
    AND v.attribute_value = ANY($values_array)
  )]
  [AND EXISTS (...)]  -- repeated per attribute

ORDER BY
  [ts_rank(...) DESC, p.sort_order, p.id]  -- if searching
  [p.sort_order, p.id]                      -- otherwise

LIMIT $limit+1 OFFSET $offset
```

**Filter semantics:**
- Multiple values for **same** attribute → OR (e.g., material IN ['cotton', 'silk'])
- Multiple **different** attributes → AND (e.g., material=cotton AND color=red)

**Material/Color merging:**
```go
// product_repository.go:List
attrFilters := make(map[string][]string)
for k, v := range req.AttributeFilters { attrFilters[k] = v }
if req.Material != nil { attrFilters["material"] = []string{*req.Material} }
if req.Color != nil    { attrFilters["color"]    = []string{*req.Color} }
```

The `material` and `color` query params are merged into `AttributeFilters` before query building.

### 6.3 Frontend → Backend Filter Flow

**Category page filter change:**
```
User checks "Cotton" under Material
  → FilterSidebar.handleAttributeToggle("material", "cotton")
  → onFiltersChange({...filters, attributeFilters: {material: ["cotton"]}})
  → CategoryProductsView.handleFiltersChange(newFilters)
    → setFilters(newFilters)
    → router.push("/c/sarees?attribute_filters=%7B%22material%22%3A%5B%22cotton%22%5D%7D")
    → useEffect triggers (300ms debounce)
      → api.get("/api/v1/store/catalog/products?category_id=cat_123&attribute_filters={\"material\":[\"cotton\"]}")
        → Next.js rewrites to ${API_BASE}/api/v1/store/catalog/products?...
          → Handler parses attribute_filters JSON
          → Adds EXISTS subquery for material
          → Returns filtered products
```

### 6.4 Filter Options Flow

```
CategoryProductsView mounts
  → api.get("/api/v1/store/catalog/products/filter-options/${categoryId}")
    → Handler calls productService.GetAttributeFilterOptions(ctx, categoryId)
      → categoryRepo.GetByID(categoryId) → category with own_attributes
      → collect attr.Name where attr.Searchable == true
      → productRepo.GetAttributeFilterOptions(ctx, categoryId, ["color", "material", ...])
        → Per attribute: SELECT DISTINCT attribute_value ... ORDER BY attribute_value
      → sort.Strings each value list
    → Response: {"color": ["blue", "green"], "material": ["cotton", "silk"]}

FilterSidebar receives:
  filterOptions = {"color": ["blue", "green"], "material": ["cotton", "silk"]}
  categoryAttributes = category.own_attributes (from SSR category fetch)

Renders checkbox groups:
  attributeSections = categoryAttributes
    .filter(attr => attr.searchable && filterOptions[attr.name]?.length > 0)
    .sort((a, b) => a.display_order - b.display_order)
```

---

## 7. Inventory & Stock

### 7.1 Data Model

```sql
inventory (
  id UUID PRIMARY KEY,
  product_id UUID UNIQUE REFERENCES products(id) ON DELETE CASCADE,
  quantity INT DEFAULT 0,           -- total stock
  reserved_qty INT DEFAULT 0,       -- reserved for orders
  available_qty INT DEFAULT 0,      -- quantity - reserved_qty
  low_stock_threshold INT DEFAULT 5,
  last_restock_at TIMESTAMP,
  created_at TIMESTAMP,
  updated_at TIMESTAMP,
  created_by VARCHAR,
  updated_by VARCHAR
)

inventory_transactions (
  id UUID PRIMARY KEY,
  product_id UUID REFERENCES products(id),
  type VARCHAR,                     -- ADD, REMOVE, RESERVE, RELEASE, ADJUST
  quantity INT,                     -- amount changed
  previous_qty INT,                -- stock before
  new_qty INT,                     -- stock after
  reason TEXT,
  reference_id VARCHAR,            -- order ID for reserve/release
  created_by VARCHAR,
  created_at TIMESTAMP
)
```

### 7.2 Stock Mutation Flows

All mutations use `SELECT ... FOR UPDATE` row-level locking to prevent race conditions.

**Add Stock:**
```
Admin clicks "Add Stock" → POST /admin/products/{id}/inventory/add {quantity: 50, reason: "PO-123"}
  → InventoryService.AddStock
    → InventoryRepo.AddStock (in tx):
        SELECT quantity, reserved_qty FROM inventory WHERE product_id=$1 FOR UPDATE
        newQty = current + 50
        available = newQty - reserved
        UPDATE inventory SET quantity=$newQty, available_qty=$available, last_restock_at=NOW()
        INSERT INTO inventory_transactions (type=ADD, ...)
    → Publish inventory.restocked event
    → cache.DeletePrefix("prod:")  ← invalidates all product caches
```

**Reserve Stock (on order placement):**
```
Customer places order → CheckoutService
  → InventoryRepo.ReserveStock(productId, qty, orderId):
      SELECT ... FOR UPDATE
      validate available >= qty
      reserved += qty, available -= qty
      INSERT transaction (type=RESERVE, reference_id=orderId)
```

**Release Stock (on order cancellation):**
```
Admin cancels order → OrderService
  → InventoryRepo.ReleaseStock(productId, qty, orderId):
      SELECT ... FOR UPDATE
      validate reserved >= qty
      reserved -= qty, available += qty
      INSERT transaction (type=RELEASE, reference_id=orderId)
```

### 7.3 Stock in Product Queries

Every product query LEFT JOINs inventory:
```sql
SELECT p.*,
  COALESCE(i.quantity, 0) as quantity,
  COALESCE(i.reserved_qty, 0) as reserved_qty,
  COALESCE(i.available_qty, 0) as available_qty,
  COALESCE(i.low_stock_threshold, 0) as low_stock_threshold
FROM products p
LEFT JOIN inventory i ON i.product_id = p.id
```

The store handler converts `AvailableQty > 0` → `InStock: true/false` (hides raw numbers from customers).

### 7.4 In-Stock Filter

When `in_stock=true` is passed as a query param:
```sql
WHERE ... AND i.available_qty > 0
```

This filters out products where the LEFT JOINed inventory has `available_qty <= 0` (or no inventory record, since `COALESCE(i.available_qty, 0) = 0`).

---

## 8. Known Gotchas

### 8.1 Category List Does NOT Load Attributes

`CategoryRepository.List()` only scans the `categories` table — it does NOT call `fetchAttributes()`. Only `GetByID()` loads the `category_attributes` + `category_attribute_options` join.

**Impact:** The store `ListCategories` endpoint returns categories with `own_attributes: null`. The store `GetCategory` slug path works because it re-fetches via `GetByID` after the slug lookup.

**Impact on FilterSidebar:** `category.own_attributes` in `CategoryProductsView` comes from the SSR `getCategory(slug)` call in `page.tsx`, which hits `GetCategory` → `GetByID` → attributes loaded. This is correct.

### 8.2 Hardcoded Attributes Live in Two Places

`material`, `color`, `weave_type`, `origin`, `craft_type` are stored both as:
1. Direct fields on the `Product` struct (populated from first `product_attribute_values` row)
2. Rows in `product_attribute_values` table

Filtering always uses the `product_attribute_values` table (via EXISTS subqueries). The `material` and `color` query params are merged into `AttributeFilters` at the repository level.

**Potential bug:** If a product has `material` set as a direct field but no corresponding row in `product_attribute_values`, the filter won't find it. Ensure product creation/update always writes hardcoded attributes to both places.

### 8.3 FilterSidebar Requires Both filterOptions AND categoryAttributes

The `FilterSidebar` only renders attribute checkboxes when:
```ts
categoryAttributes?.filter(attr => attr.searchable && filterOptions?.[attr.name]?.length)
```

Both conditions must be true:
1. The attribute must exist in `category.own_attributes` with `searchable: true`
2. The attribute must have values in the `filterOptions` response

If the filter-options API returns a key that doesn't match any `own_attributes` entry, it won't render.

### 8.4 Product List Cache is Aggressive

All product list queries are cached by MD5 hash of the full request. Any product CRUD operation invalidates ALL `prod:list:*` keys (conservative).

**Impact:** After creating/updating/deleting a product, all cached list results are cleared. This is correct but means the first request after a mutation will be slow (cache miss → full SQL query).

### 8.5 In-Process Cache is Per-Lambda-Instance

The `go-cache` is in-memory per Lambda instance. Different Lambda instances may serve stale data for up to 1 hour (cache TTL) after a mutation handled by a different instance.

**Mitigation:** TTL is 1h, which is acceptable for catalog data. For real-time stock, the `CheckAvailability` endpoint bypasses the cache.

### 8.6 ISR Cache on Next.js (60 seconds)

Storefront SSR pages use `next: { revalidate: 60 }` — stale data can be served for up to 60 seconds after a backend change. Client-side re-fetches (via Axios) bypass the ISR cache.

### 8.7 Attribute Filter Values Are Case-Sensitive

Filter values must match the exact case stored in `product_attribute_values.attribute_value`. The backend does not normalize case when filtering.

### 8.8 Empty attributeFilters Param

If `attribute_filters={}` (empty JSON object) is sent, the backend correctly generates zero EXISTS subqueries — no attribute filtering applied. The store handler only parses `attribute_filters` if the query param is non-empty.

### 8.9 StoreProduct Type Mismatch

The storefront `Product` TypeScript type has `mrp: number` but the backend `StoreProduct` sends `base_price`. Ensure the frontend maps `base_price` → `mrp` or the types are consistent.

### 8.10 Slug Lookup Double-Fetch

Both `GetCategory` and `GetProduct` on the store handler do a double-fetch for slug lookups:
1. `List` with `Slug` filter to find the entity
2. `GetByID` to load full relations (attributes, images, inventory)

This is intentional — `List` is optimized for scanning and doesn't load nested relations.
