# Catalog Lambda - High-Level Design (HLD)

## 1. Overview

The Catalog Lambda service manages the product catalog including flat Categories (with custom searchable attributes), Designs, and Products. It provides CRUD operations, dynamic attribute management, searchable attribute indexing, pre-computed filter options, and integrated inventory management.

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    CATALOG SYSTEM                                            │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

                                    ┌───────────────────┐
                                    │   React Frontend  │
                                    │   (Admin Portal)  │
                                    └─────────┬─────────┘
                                              │
                                              │ HTTPS
                                              ▼
                                    ┌───────────────────┐
                                    │   API Gateway     │
                                    │   (REST API)      │
                                    └─────────┬─────────┘
                                              │
              ┌───────────────────────────────┼───────────────────────────────┐
              │                               │                               │
              ▼                               ▼                               ▼
   ┌─────────────────────┐       ┌─────────────────────┐       ┌─────────────────────┐
   │  Category Handler   │       │   Product Handler   │       │   Design Handler    │
   │  - List             │       │   - CRUD            │       │   - CRUD            │
   │  - CRUD             │       │   - Filter Options  │       │   - By Category     │
   │  - Attribute CRUD   │       │   - Inventory Mgmt  │       │                     │
   └──────────┬──────────┘       └──────────┬──────────┘       └──────────┬──────────┘
              │                              │                              │
              └───────────────┬──────────────┼──────────────────────────────┘
                              │              │
                              ▼              ▼
                    ┌─────────────────┐ ┌──────────────────┐
                    │ Category Svc    │ │ Product Svc      │
                    │ - Flat CRUD     │ │ - CRUD + Index   │
                    │ - Attr Mgmt    │ │ - Filter Options │
                    └────────┬────────┘ └────────┬─────────┘
                             │                   │
              ┌──────────────┼───────────────────┼──────────────────────┐
              │              │                   │                      │
              ▼              ▼                   ▼                      ▼
   ┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐ ┌──────────────────┐
   │   Category Repo  │ │   Product Repo   │ │   Design Repo    │ │  Inventory Repo  │
   │                  │ │   + Attr Index   │ │                  │ │  + Transactions  │
   └────────┬─────────┘ └────────┬─────────┘ └────────┬─────────┘ └────────┬─────────┘
            │                    │                     │                    │
            └────────────────────┼─────────────────────┼────────────────────┘
                                 │                     │
                                 ▼                     ▼
                       ┌─────────────────────┐  ┌─────────────────────┐
                       │     DynamoDB        │  │    CloudWatch       │
                       │  (Single Table)     │  │   (Logs, Metrics)   │
                       └─────────────────────┘  └─────────────────────┘
```

---

## 3. Component Design

### 3.1 Service Layer Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CATALOG SERVICE LAYER                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         Handler Layer                                │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │   │
│  │  │  Category   │  │  Product    │  │  Design     │                 │   │
│  │  │  Handler    │  │  Handler    │  │  Handler    │                 │   │
│  │  │  + Attr API │  │  + Filter   │  │             │                 │   │
│  │  │             │  │  + Inventory│  │             │                 │   │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘                 │   │
│  └─────────┼────────────────┼────────────────┼──────────────────────────┘   │
│            │                │                │                              │
│  ┌─────────┼────────────────┼────────────────┼──────────────────────────┐   │
│  │         ▼                ▼                ▼     Service Layer        │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │   │
│  │  │  Category   │  │  Product    │  │  Design     │                 │   │
│  │  │  Service    │  │  Service    │  │  Service    │                 │   │
│  │  │             │  │             │  │             │                 │   │
│  │  │ - Create    │  │ - Create    │  │ - Create    │                 │   │
│  │  │ - GetByID   │  │   + Index   │  │ - GetByID   │                 │   │
│  │  │ - Update    │  │   + AttrVals│  │ - Update    │                 │   │
│  │  │ - Delete    │  │ - Update    │  │ - Delete    │                 │   │
│  │  │ - List      │  │   + Reindex │  │ - List      │                 │   │
│  │  │ - AddAttr   │  │ - Delete    │  │             │                 │   │
│  │  │ - UpdAttr   │  │   + Cleanup │  │             │                 │   │
│  │  │ - DelAttr   │  │ - List      │  │             │                 │   │
│  │  │ - GetAttrs  │  │ - FilterOpt │  │             │                 │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘                 │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      Repository Layer                                │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌───────────┐ │   │
│  │  │  Category   │  │  Product    │  │  Design     │  │ Inventory │ │   │
│  │  │  Repository │  │  Repository │  │  Repository │  │ Repository│ │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └───────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Model

### 4.1 DynamoDB Single Table Design

All entities reside in a single DynamoDB table (`handloom-core`) using a composite primary key (PK, SK) pattern.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          TABLE: handloom-core                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  CATEGORY RECORDS                                                            │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: CATEGORY#<category_id>                                        │      │
│  │ SK: METADATA                                                      │      │
│  │                                                                   │      │
│  │ GSI1PK: CATEGORY#ALL        GSI1SK: CATEGORY#<id>                │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - name, slug, description, image_url                           │      │
│  │   - own_attributes[] (name, label, type, required, searchable,   │      │
│  │                       display_order, options[])                   │      │
│  │   - status (ACTIVE/INACTIVE)                                     │      │
│  │   - product_count (denormalized)                                 │      │
│  │   - created_at, updated_at, created_by, updated_by              │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  CATEGORY ATTRIBUTE VALUES (pre-computed filter options)                     │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: CATEGORY#<category_id>                                        │      │
│  │ SK: ATTR_VALUES                                                   │      │
│  │                                                                   │      │
│  │ Top-level String Set fields:                                      │      │
│  │   - attr_material: SS["cotton", "silk"]                          │      │
│  │   - attr_color: SS["red", "blue", "green"]                      │      │
│  │   - attr_weave_type: SS["jacquard", "plain"]                    │      │
│  │                                                                   │      │
│  │ Updated atomically via DynamoDB ADD on each product create/update │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  DESIGN RECORDS                                                              │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: DESIGN#<design_id>                                            │      │
│  │ SK: METADATA                                                      │      │
│  │                                                                   │      │
│  │ GSI1PK: CATEGORY#<cat_id>   GSI1SK: DESIGN#<id>                 │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - name, slug, category_id, description                        │      │
│  │   - images[], attributes[] (name, values[])                      │      │
│  │   - status, product_count                                        │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  PRODUCT RECORDS                                                             │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: PRODUCT#<product_id>                                          │      │
│  │ SK: METADATA                                                      │      │
│  │                                                                   │      │
│  │ GSI1PK: CATEGORY#<cat_id>   GSI1SK: PRODUCT#<id>                │      │
│  │ GSI2PK: PRODUCT#ALL         GSI2SK: PRODUCT#<id>                │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - name, slug, sku, description                                 │      │
│  │   - design_id, category_id, artisan_id                           │      │
│  │   - base_price, selling_price, cost_price, currency (prices in   │      │
│  │     paise)                                                        │      │
│  │   - dimensions{}, weight                                         │      │
│  │   - attributes{} (flexible map for category-specific data)       │      │
│  │   - material, color, weave_type (common indexed fields)          │      │
│  │   - images[], tags[]                                              │      │
│  │   - quantity, reserved_qty, available_qty, low_stock_threshold   │      │
│  │   - status (ACTIVE/INACTIVE/DRAFT)                               │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  PRODUCT SKU INDEX (uniqueness enforcement)                                 │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: SKU#<sku>                                                     │      │
│  │ SK: METADATA                                                      │      │
│  │                                                                   │      │
│  │ Attributes: product_id, entity_type (PRODUCT_SKU)                 │      │
│  │                                                                   │      │
│  │ Created atomically with product via TransactWriteItems            │      │
│  │ with attribute_not_exists(PK) to guarantee SKU uniqueness.        │      │
│  │ Enables O(1) SKU lookup: GetItem → product_id → GetByID.         │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  PRODUCT ATTRIBUTE INDEX (for searchable attribute filtering)                │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: PRODUCT#<product_id>                                          │      │
│  │ SK: ATTR#<attr_name>#<attr_value>                                 │      │
│  │                                                                   │      │
│  │ GSI1PK: ATTR#<category_id>#<attr_name>                           │      │
│  │ GSI1SK: <attr_value>#PRODUCT#<product_id>                        │      │
│  │                                                                   │      │
│  │ Attributes: product_id, category_id, attr_name, attr_value       │      │
│  │                                                                   │      │
│  │ Enables queries like:                                             │      │
│  │   "All products in category X with material=silk"                 │      │
│  │   Query GSI1 where PK=ATTR#cat_id#material, SK begins_with silk  │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  INVENTORY RECORDS                                                           │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: INVENTORY#<product_id>                                        │      │
│  │ SK: METADATA                                                      │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - product_id, product_sku, product_name                        │      │
│  │   - quantity, reserved_qty, available_qty                        │      │
│  │   - low_stock_threshold, reorder_point, last_restock_at          │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  INVENTORY TRANSACTIONS                                                      │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: INVENTORY#<product_id>                                        │      │
│  │ SK: TXN#<timestamp>#<txn_id>                                      │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - product_id, type (ADD/REMOVE/RESERVE/RELEASE/ADJUST)         │      │
│  │   - quantity, previous_qty, new_qty, reason                      │      │
│  │   - reference_type, reference_id                                  │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  GLOBAL SECONDARY INDEXES                                                    │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ GSI1 (GSI1PK + GSI1SK):                                          │      │
│  │   - List all categories: PK=CATEGORY#ALL                         │      │
│  │   - Designs by category: PK=CATEGORY#<id>, SK begins DESIGN#     │      │
│  │   - Products by category: PK=CATEGORY#<id>, SK begins PRODUCT#   │      │
│  │   - Products by attribute: PK=ATTR#<cat>#<attr>, SK begins <val> │      │
│  │   - Users by email: PK=USER_EMAIL, SK=<email>                    │      │
│  │                                                                   │      │
│  │ GSI2 (GSI2PK + GSI2SK):                                          │      │
│  │   - All products: PK=PRODUCT#ALL, SK begins PRODUCT#              │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 4.2 Attribute Indexing Design

Categories are **flat** (no hierarchy). Each category defines its own set of attributes via `own_attributes[]`. Attributes with `searchable: true` are indexed to enable efficient product filtering.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        ATTRIBUTE INDEXING                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  When a product is created/updated with searchable attributes:              │
│                                                                              │
│  1. ProductAttributeIndex records (per attribute per product):               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Product: prod_001, Category: cat_sarees                             │   │
│  │ Attributes: {material: "silk", color: "red"}                        │   │
│  │                                                                      │   │
│  │ Creates 2 index records:                                             │   │
│  │   PK=PRODUCT#prod_001  SK=ATTR#material#silk                        │   │
│  │     GSI1PK=ATTR#cat_sarees#material  GSI1SK=silk#PRODUCT#prod_001   │   │
│  │                                                                      │   │
│  │   PK=PRODUCT#prod_001  SK=ATTR#color#red                            │   │
│  │     GSI1PK=ATTR#cat_sarees#color  GSI1SK=red#PRODUCT#prod_001       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  2. CategoryAttributeValues record (pre-computed distinct values):          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ PK=CATEGORY#cat_sarees  SK=ATTR_VALUES                              │   │
│  │                                                                      │   │
│  │ attr_material: SS["cotton", "silk"]   <-- DynamoDB String Set       │   │
│  │ attr_color: SS["blue", "red"]         <-- DynamoDB String Set       │   │
│  │                                                                      │   │
│  │ Updated atomically using DynamoDB ADD on top-level SS fields.        │   │
│  │ Single write operation. Creates item if not exists.                  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Filter Options Read Path:                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  GET /products/filter-options/{categoryId}                          │   │
│  │    1. GetItem(PK=CATEGORY#<id>, SK=METADATA) → get searchable set  │   │
│  │    2. GetItem(PK=CATEGORY#<id>, SK=ATTR_VALUES) → read SS fields   │   │
│  │    3. Filter to searchable attrs only, sort values                  │   │
│  │    4. Return { "material": ["cotton","silk"], "color": ["red"] }    │   │
│  │                                                                      │   │
│  │  Result: 2 DynamoDB reads total (vs N queries per attribute before) │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Product Images Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        IMAGE MANAGEMENT                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Upload Flow:                                                                │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │   Client ──▶ Get Presigned URL ──▶ Upload to S3 ──▶ Confirm        │   │
│  │                                                                      │   │
│  │   1. Request presigned URL from API                                 │   │
│  │   2. Upload directly to S3 (bypass Lambda)                          │   │
│  │   3. Confirm upload, trigger image processing                       │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  S3 Bucket Structure:                                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  handloom-assets/                                                   │   │
│  │    ├── products/                                                    │   │
│  │    │     ├── original/    (full resolution)                         │   │
│  │    │     ├── large/       (1200x1200)                               │   │
│  │    │     ├── medium/      (600x600)                                 │   │
│  │    │     └── thumb/       (150x150)                                 │   │
│  │    ├── categories/                                                  │   │
│  │    └── designs/                                                     │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Search & Filtering Design

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          SEARCH & FILTERING                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Product Listing with Filters:                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  Standard Filters (on product fields):                              │   │
│  │    - category_id: GSI1 query (PK=CATEGORY#<id>, SK begins PRODUCT#)│   │
│  │    - all products: GSI2 query (PK=PRODUCT#ALL)                     │   │
│  │    - status, min_price, max_price, in_stock, low_stock: post-filter│   │
│  │    - search: name/SKU contains (DynamoDB filter on GSI query)      │   │
│  │    - material, color: direct field filters                         │   │
│  │    - sku lookup: GetItem PK=SKU#<sku> (O(1), no scan)             │   │
│  │                                                                      │   │
│  │  Dynamic Attribute Filters (attribute_filters param):               │   │
│  │    - Uses ProductAttributeIndex + GSI1                              │   │
│  │    - Query: PK=ATTR#<cat_id>#<attr_name>, SK begins_with <value>  │   │
│  │    - Intersect product IDs across multiple attribute filters        │   │
│  │    - Batch fetch matching products                                  │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Filter Options (pre-computed):                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  GET /products/filter-options/{categoryId}                          │   │
│  │    - Reads CategoryAttributeValues record (single GetItem)          │   │
│  │    - Returns distinct values per searchable attribute               │   │
│  │    - Frontend uses these to populate filter dropdowns               │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Validation Rules

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          VALIDATION RULES                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Product Validation:                                                         │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ Field         │ Rules                                             │      │
│  ├───────────────┼───────────────────────────────────────────────────┤      │
│  │ Name          │ Required                                          │      │
│  │ SKU           │ Required, unique                                  │      │
│  │ Design ID     │ Required, must exist                              │      │
│  │ Category ID   │ Required, must exist                              │      │
│  │ Base Price    │ Required, > 0 (in paise)                          │      │
│  │ Selling Price │ Required, > 0 (in paise)                          │      │
│  │ Status        │ DRAFT | ACTIVE | INACTIVE                        │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  Category Validation:                                                        │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ Field         │ Rules                                             │      │
│  ├───────────────┼───────────────────────────────────────────────────┤      │
│  │ Name          │ Required                                          │      │
│  │ Slug          │ Auto-generated from name, unique globally         │      │
│  │ Status        │ ACTIVE | INACTIVE                                 │      │
│  │ Delete guard  │ Cannot delete if products exist                   │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  Design Validation:                                                          │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ Field         │ Rules                                             │      │
│  ├───────────────┼───────────────────────────────────────────────────┤      │
│  │ Name          │ Required                                          │      │
│  │ Category ID   │ Required, must exist                              │      │
│  │ Slug          │ Auto-generated from name                          │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  Inventory Validation:                                                       │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ Field         │ Rules                                             │      │
│  ├───────────────┼───────────────────────────────────────────────────┤      │
│  │ Quantity      │ Required, > 0 (for add/remove)                    │      │
│  │ New Quantity  │ >= 0 (for adjust)                                 │      │
│  │ Reason        │ Required for all operations                       │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Error Handling

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          ERROR CODES                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Product Errors:                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ CAT001 │ Product not found                                          │   │
│  │ CAT002 │ SKU already exists                                         │   │
│  │ CAT003 │ Invalid category                                           │   │
│  │ CAT004 │ Product has pending orders (cannot delete)                 │   │
│  │ CAT005 │ Invalid product status transition                          │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Category Errors:                                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ CAT010 │ Category not found                                         │   │
│  │ CAT011 │ Category has products (cannot delete)                      │   │
│  │ CAT015 │ Slug already exists                                        │   │
│  │ CAT016 │ Attribute not found                                        │   │
│  │ CAT017 │ Attribute name already exists                              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Design Errors:                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ CAT020 │ Design not found                                           │   │
│  │ CAT021 │ Design slug already exists                                 │   │
│  │ CAT022 │ Design linked to products (cannot delete)                  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 9. Scalability & Performance

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     SCALABILITY CONSIDERATIONS                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Lambda Configuration:                                                       │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Memory: 512 MB                                                    │   │
│  │ • Timeout: 30 seconds                                               │   │
│  │ • Concurrent executions: 200 (reserved)                             │   │
│  │ • Provisioned concurrency: 10                                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  DynamoDB Configuration:                                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Capacity: On-demand                                               │   │
│  │ • GSI count: 2 (GSI1 multi-purpose, GSI2 all-products)             │   │
│  │ • Batch operations: Max 25 items per BatchWriteItem                 │   │
│  │ • Atomic ADD for String Sets (CategoryAttributeValues)              │   │
│  │ • TransactWriteItems for product + attribute index creation         │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Performance Optimizations:                                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Pre-computed filter options (2 reads vs N GSI queries)            │   │
│  │ • Atomic ADD on top-level SS fields (single write per product op)  │   │
│  │ • Denormalized product counts on categories and designs            │   │
│  │ • Denormalized inventory fields on products                        │   │
│  │ • Projection expressions to reduce data transfer                    │   │
│  │ • Connection pooling (reuse DynamoDB client across invocations)    │   │
│  │ • Cursor-based pagination (DynamoDB-native ExclusiveStartKey)     │   │
│  │ • O(1) SKU lookup via SKU# uniqueness item (replaces table Scan)  │   │
│  │ • GSI2 PRODUCT#ALL for all-products listing (single partition)    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 10. Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          DEPENDENCIES                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  External Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • AWS DynamoDB - Data storage (single table)                       │   │
│  │ • AWS S3 - Image storage                                            │   │
│  │ • AWS CloudWatch - Logging & monitoring                             │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Libraries:                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • aws-sdk-go-v2/dynamodb - DynamoDB client                          │   │
│  │ • go-chi/chi - HTTP routing                                         │   │
│  │ • google/uuid - ID generation                                       │   │
│  │ • gosimple/slug - Slug generation                                   │   │
│  │ • google/wire - Dependency injection                                │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```
