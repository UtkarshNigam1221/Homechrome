# Store Catalog - High-Level Design (HLD)

## 1. Overview

The Store Catalog service provides a public, read-only browsing layer over the admin catalog for the B2C storefront. It reuses the existing `ProductService`, `CategoryService`, and `InventoryService` from the admin domain, but applies storefront-specific transformations: filtering to only ACTIVE items, stripping sensitive fields (cost_price), mapping raw inventory counts to a boolean `in_stock` flag, and supporting slug-based URL lookups. No authentication is required.

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                              STORE CATALOG SYSTEM                                             │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

                                    ┌───────────────────┐
                                    │   Next.js Store   │
                                    │   (SSR + Client)  │
                                    └─────────┬─────────┘
                                              │
                                              │ HTTPS (public, no auth)
                                              ▼
                                    ┌───────────────────┐
                                    │   API Gateway /   │
                                    │   Chi Router      │
                                    └─────────┬─────────┘
                                              │
                                              ▼
                                    ┌───────────────────┐
                                    │  CatalogHandler   │
                                    │  (store/)         │
                                    │                   │
                                    │  - toStoreProduct │
                                    │  - toStoreCategory│
                                    │  - isUUID()       │
                                    └─────────┬─────────┘
                                              │
                         ┌────────────────────┼────────────────────┐
                         │                    │                    │
                         ▼                    ▼                    ▼
              ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
              │ ProductService  │  │ CategoryService │  │ InventoryService│
              │ (shared admin)  │  │ (shared admin)  │  │ (shared admin)  │
              └────────┬────────┘  └────────┬────────┘  └────────┬────────┘
                       │                    │                    │
                       └────────────────────┼────────────────────┘
                                            │
                                            ▼
                                 ┌─────────────────────┐
                                 │      DynamoDB       │
                                 │   handloom-core     │
                                 │   (Products,        │
                                 │   Categories,       │
                                 │   Inventory)        │
                                 └─────────────────────┘
```

---

## 3. Component Design

### 3.1 CatalogHandler (store/catalog_handler.go)

```
┌─────────────────────────────────────────────────────────────────────┐
│                      STORE CATALOG HANDLER                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  All routes are public (no auth middleware):                          │
│                                                                      │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐  │
│  │  ListCategories  │  │  GetCategory     │  │  ListProducts    │  │
│  │  GET /categories │  │  GET /categories │  │  GET /products   │  │
│  │                  │  │  /{idOrSlug}     │  │                  │  │
│  └────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘  │
│           │                     │                      │             │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐  │
│  │  SearchProducts  │  │  GetProduct      │  │ CheckAvailability│  │
│  │  GET /products   │  │  GET /products   │  │ GET /products    │  │
│  │  /search         │  │  /{idOrSlug}     │  │ /{id}/avail..   │  │
│  │  (alias)         │  │                  │  │                  │  │
│  └────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘  │
│           │                     │                      │             │
│           └─────────────────────┼──────────────────────┘             │
│                                 │                                    │
│                                 ▼                                    │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                    TRANSFORMATION LAYER                        │  │
│  │                                                               │  │
│  │  toStoreProduct(product, inStock)                             │  │
│  │    - Copies all public fields from domain.Product             │  │
│  │    - Excludes cost_price                                      │  │
│  │    - Maps available_qty > 0 to in_stock boolean               │  │
│  │                                                               │  │
│  │  toStoreProductFromRelations(productWithRelations)            │  │
│  │    - Converts full relation object                            │  │
│  │    - Populates category summary (id, name, slug)              │  │
│  │                                                               │  │
│  │  toStoreCategory(category)                                    │  │
│  │    - Copies all category fields (no sensitive fields)         │  │
│  │    - Includes denormalized product_count                      │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                    HELPER FUNCTIONS                            │  │
│  │                                                               │  │
│  │  parsePagination(r) — limit, cursor, sort_by, sort_order      │  │
│  │  isUUID(s)          — route UUID vs slug for id-or-slug paths │  │
│  │  findProductBySlug  — search + exact slug match               │  │
│  │  findCategoryBySlug — search + exact slug match               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 Store Response Types

```
┌─────────────────────────────────────────────────────────────────────┐
│                    STORE RESPONSE TYPES                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  StoreProduct (excludes cost_price, replaces qty with in_stock):    │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ id, name, slug, sku, description                              │  │
│  │ category_id, artisan_id                                       │  │
│  │ base_price, selling_price, currency (INR)                     │  │
│  │ dimensions, weight                                            │  │
│  │ allow_custom_dimensions, pricing_rule_id                      │  │
│  │ attributes (map), material, color, weave_type                 │  │
│  │ origin, craft_type                                            │  │
│  │ images[] (url, alt_text, is_primary, sort_order)              │  │
│  │ tags[]                                                        │  │
│  │ in_stock (bool)                 <-- derived from inventory    │  │
│  │ category (id, name, slug)       <-- populated on GetProduct   │  │
│  │ created_at, updated_at                                        │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  StoreCategory:                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ id, name, slug, description, image_url                        │  │
│  │ own_attributes[] (name, label, type, options, etc.)           │  │
│  │ product_count (denormalized)                                  │  │
│  │ status                                                        │  │
│  │ created_at, updated_at                                        │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  AvailabilityResponse:                                               │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ in_stock (bool)                                               │  │
│  │ available_quantity (int)                                       │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Model

### 4.1 DynamoDB Table Design (shared with admin)

```
┌─────────────────────────────────────────────────────────────────────┐
│                  TABLE: handloom-core                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  PRODUCT RECORDS                                                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ PK: PRODUCT#<product_id>                                      │  │
│  │ SK: METADATA                                                  │  │
│  │                                                               │  │
│  │ Attributes:                                                   │  │
│  │   - id, name, slug, sku, description                         │  │
│  │   - category_id, artisan_id                                  │  │
│  │   - base_price, selling_price, cost_price (hidden from store)│  │
│  │   - currency, dimensions, weight                             │  │
│  │   - allow_custom_dimensions, pricing_rule_id                 │  │
│  │   - attributes (map), material, color, weave_type            │  │
│  │   - origin, craft_type, images[], tags[]                     │  │
│  │   - quantity, reserved_qty, available_qty                    │  │
│  │   - low_stock_threshold, status                              │  │
│  │   - created_at, updated_at                                   │  │
│  │                                                               │  │
│  │ GSI1: PK=CATEGORY#<cat_id>, SK=PRODUCT#<prod_id>             │  │
│  │ GSI2: PK=PRODUCT#ALL, SK=PRODUCT#<prod_id>                   │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  CATEGORY RECORDS                                                    │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ PK: CATEGORY#<category_id>                                    │  │
│  │ SK: METADATA                                                  │  │
│  │                                                               │  │
│  │ Attributes:                                                   │  │
│  │   - id, name, slug, description, image_url                   │  │
│  │   - own_attributes[], product_count                          │  │
│  │   - status, created_at, updated_at                           │  │
│  │                                                               │  │
│  │ GSI1: PK=CATEGORY#ALL, SK=CATEGORY#<cat_id>                  │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  INVENTORY RECORDS                                                   │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ PK: INVENTORY#<product_id>                                    │  │
│  │ SK: METADATA                                                  │  │
│  │                                                               │  │
│  │ Attributes:                                                   │  │
│  │   - product_id, product_sku, product_name                    │  │
│  │   - quantity, reserved_qty, available_qty                    │  │
│  │   - low_stock_threshold, reorder_point                       │  │
│  │   - last_restock_at                                          │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  PRODUCT ATTRIBUTE INDEXES (for filtered queries)                    │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ PK: PRODUCT#<product_id>                                      │  │
│  │ SK: ATTR#<attr_name>#<attr_value>                             │  │
│  │                                                               │  │
│  │ GSI1: PK=ATTR#<category_id>#<attr_name>                      │  │
│  │       SK=<attr_value>#PRODUCT#<product_id>                    │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 4.2 Data Filtering Strategy

```
┌─────────────────────────────────────────────────────────────────────┐
│                    STORE vs ADMIN DATA FILTERING                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  The store catalog shares the same DynamoDB tables as the admin      │
│  catalog. Filtering is applied at the service/handler layer:         │
│                                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                                                               │  │
│  │  Admin API              Store API                             │  │
│  │  ─────────              ─────────                             │  │
│  │  All statuses    ───▶   ACTIVE only (hardcoded filter)        │  │
│  │  cost_price      ───▶   Excluded (not in StoreProduct)        │  │
│  │  quantity fields  ───▶   Mapped to in_stock boolean           │  │
│  │  ID lookups only  ───▶   ID + slug lookups                    │  │
│  │                                                               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 5. Security

```
┌─────────────────────────────────────────────────────────────────────┐
│                       SECURITY DESIGN                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Access Control:                                                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ - All catalog endpoints are fully public (no auth required)   │  │
│  │ - No write operations exposed                                 │  │
│  │ - Read-only access to ACTIVE items only                       │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Data Protection:                                                    │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ - cost_price excluded from all store responses                │  │
│  │ - Raw inventory quantities hidden (only in_stock boolean)     │  │
│  │ - INACTIVE/DRAFT products return 404 (not exposed)            │  │
│  │ - INACTIVE categories return 404                              │  │
│  │ - No admin metadata (created_by, updated_by) exposed         │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Input Validation:                                                   │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ - limit capped at 100 (prevents large scans)                  │  │
│  │ - attribute_filters parsed as JSON (invalid JSON ignored)     │  │
│  │ - Numeric params parsed with error handling (invalid ignored)  │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 6. Error Handling

```
┌─────────────────────────────────────────────────────────────────────┐
│                       ERROR HANDLING                                   │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Error Responses:                                                    │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ NOT_FOUND         │ Product or category not found / inactive│    │
│  │ BAD_REQUEST       │ Missing required path parameter         │    │
│  │ INTERNAL_ERROR    │ Database read failure                   │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  Graceful Degradation:                                               │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ - Invalid attribute_filters JSON: silently ignored           │    │
│  │ - Invalid numeric params (min_price, etc.): silently ignored │    │
│  │ - Missing inventory record: falls back to product available_ │    │
│  │   qty for availability check                                 │    │
│  │ - Slug lookup miss: returns 404 Not Found                   │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  Response Format:                                                    │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ {                                                           │    │
│  │   "success": false,                                        │    │
│  │   "error": {                                               │    │
│  │     "code": "NOT_FOUND",                                   │    │
│  │     "message": "Product not found"                         │    │
│  │   }                                                        │    │
│  │ }                                                           │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 7. Integration Points

```
┌─────────────────────────────────────────────────────────────────────┐
│                    INTEGRATION POINTS                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Shared Admin Services (reused, not duplicated):                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ - domain.ProductService.List()   — product listing + filters  │  │
│  │ - domain.ProductService.GetByID() — single product + relations│  │
│  │ - domain.CategoryService.List()  — category listing           │  │
│  │ - domain.CategoryService.GetByID() — single category          │  │
│  │ - domain.InventoryService.GetByProductID() — live stock data  │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Downstream Consumers:                                               │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ - Next.js store frontend (SSR for SEO + client-side filters)  │  │
│  │ - Cart Service (validates product existence + pricing)         │  │
│  │ - Checkout Service (reads product data for order creation)     │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  URL Slug Pattern:                                                   │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ - Frontend uses slugs for SEO-friendly URLs                   │  │
│  │   /c/banarasi-sarees -> GET /catalog/categories/banarasi-sarees│  │
│  │   /p/royal-blue-banarasi -> GET /catalog/products/royal-blue..│  │
│  │ - Handler detects UUID vs slug via uuid.Parse()               │  │
│  │ - UUID: direct GetByID lookup                                 │  │
│  │ - Slug: search + exact match (List with search=slug)          │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 8. Dependencies

```
┌─────────────────────────────────────────────────────────────────────┐
│                       DEPENDENCIES                                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  External Services:                                                  │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ - AWS DynamoDB (handloom-core table)                          │  │
│  │ - S3 / CloudFront (product/category image hosting)            │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Internal Interfaces:                                                │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ - domain.ProductService (List, GetByID)                       │  │
│  │ - domain.CategoryService (List, GetByID)                      │  │
│  │ - domain.InventoryService (GetByProductID)                    │  │
│  │ - domain.ProductRepository (underlying DynamoDB access)       │  │
│  │ - domain.CategoryRepository (underlying DynamoDB access)      │  │
│  │ - domain.InventoryRepository (underlying DynamoDB access)     │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Libraries:                                                          │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ - go-chi/chi — HTTP routing                                   │  │
│  │ - google/uuid — UUID parsing for id-vs-slug detection         │  │
│  │ - aws-sdk-go-v2 — DynamoDB client                             │  │
│  │ - encoding/json — attribute_filters query param parsing        │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```
