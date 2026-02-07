# Catalog Lambda - High-Level Design (HLD)

## 1. Overview

The Catalog Lambda service manages the product catalog including Categories, Designs, and Products. It provides CRUD operations, hierarchical category management, and product search capabilities.

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
   │  - List/Tree        │       │   - CRUD            │       │   - CRUD            │
   │  - CRUD             │       │   - Search          │       │   - Link Products   │
   │  - Reorder          │       │   - Bulk ops        │       │                     │
   └──────────┬──────────┘       └──────────┬──────────┘       └──────────┬──────────┘
              │                              │                              │
              └───────────────────────┬──────┴──────────────────────────────┘
                                      │
                                      ▼
                            ┌─────────────────────┐
                            │   Catalog Service   │
                            │   (Business Logic)  │
                            └──────────┬──────────┘
                                       │
              ┌────────────────────────┼────────────────────────┐
              │                        │                        │
              ▼                        ▼                        ▼
   ┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
   │     DynamoDB        │  │        S3           │  │    CloudWatch       │
   │  (Products, Cats,   │  │   (Product Images)  │  │   (Logs, Metrics)   │
   │   Designs)          │  │                     │  │                     │
   └─────────────────────┘  └─────────────────────┘  └─────────────────────┘
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
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘                 │   │
│  └─────────┼────────────────┼────────────────┼──────────────────────────┘   │
│            │                │                │                              │
│            └────────────────┼────────────────┘                              │
│                             ▼                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        Service Layer                                 │   │
│  │                                                                      │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │                    CatalogService                              │ │   │
│  │  │  - CreateCategory()      - CreateProduct()                    │ │   │
│  │  │  - GetCategoryTree()     - UpdateProduct()                    │ │   │
│  │  │  - UpdateCategory()      - DeleteProduct()                    │ │   │
│  │  │  - DeleteCategory()      - SearchProducts()                   │ │   │
│  │  │  - CreateDesign()        - GetProductsByCategory()            │ │   │
│  │  │  - LinkDesignToProduct() - BulkCreateProducts()               │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                             │                                               │
│                             ▼                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                      Repository Layer                                │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │   │
│  │  │  Category   │  │  Product    │  │  Design     │                 │   │
│  │  │  Repository │  │  Repository │  │  Repository │                 │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘                 │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Model

### 4.1 DynamoDB Table Design

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          TABLE: handloom-admin                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  CATEGORY RECORDS                                                            │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: CATEGORY#<category_id>                                        │      │
│  │ SK: METADATA                                                      │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - name                    - slug (GSI1-PK)                     │      │
│  │   - description             - parent_id                          │      │
│  │   - image_url               - path (ancestry path)               │      │
│  │   - depth                   - sort_order                         │      │
│  │   - status                  - product_count                      │      │
│  │   - created_at              - updated_at                         │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  PRODUCT RECORDS                                                             │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: PRODUCT#<product_id>                                          │      │
│  │ SK: METADATA                                                      │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - sku (GSI2-PK)           - name                               │      │
│  │   - description             - category_id (GSI3-PK)              │      │
│  │   - design_id               - base_price                         │      │
│  │   - images[]                - attributes{}                       │      │
│  │   - status (GSI3-SK)        - created_at                         │      │
│  │   - updated_at              - created_by                         │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  DESIGN RECORDS                                                              │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: DESIGN#<design_id>                                            │      │
│  │ SK: METADATA                                                      │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - name                    - code (GSI4-PK)                     │      │
│  │   - description             - category_id                        │      │
│  │   - image_urls[]            - status                             │      │
│  │   - created_at              - updated_at                         │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  GLOBAL SECONDARY INDEXES                                                    │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ GSI1: slug-index         (category slug lookups)                  │      │
│  │ GSI2: sku-index          (product SKU lookups)                    │      │
│  │ GSI3: category-status    (products by category + status)         │      │
│  │ GSI4: design-code-index  (design code lookups)                   │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 4.2 Category Hierarchy Design

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        CATEGORY HIERARCHY                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Example Hierarchy:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  Sarees (depth=0, path="/")                                         │   │
│  │    ├── Silk Sarees (depth=1, path="/sarees/")                       │   │
│  │    │     ├── Kanchipuram (depth=2, path="/sarees/silk-sarees/")     │   │
│  │    │     └── Banarasi (depth=2, path="/sarees/silk-sarees/")        │   │
│  │    └── Cotton Sarees (depth=1, path="/sarees/")                     │   │
│  │          └── Handloom Cotton (depth=2, path="/sarees/cotton-sarees/")│   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Adjacency List Pattern:                                                     │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ Category ID │ Name           │ Parent ID    │ Depth │ Path        │      │
│  ├─────────────┼────────────────┼──────────────┼───────┼─────────────┤      │
│  │ cat_001     │ Sarees         │ null         │ 0     │ /           │      │
│  │ cat_002     │ Silk Sarees    │ cat_001      │ 1     │ /cat_001/   │      │
│  │ cat_003     │ Kanchipuram    │ cat_002      │ 2     │ /cat_001/...│      │
│  └───────────────────────────────────────────────────────────────────┘      │
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
│  CloudFront Distribution:                                                    │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  cdn.handloom.com ──▶ S3 Origin                                     │   │
│  │                                                                      │   │
│  │  Cache Policy:                                                       │   │
│  │    - TTL: 1 year (images are immutable)                             │   │
│  │    - Compress: enabled                                              │   │
│  │    - Origin Shield: enabled                                         │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Search Design

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          SEARCH ARCHITECTURE                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Current Implementation (DynamoDB):                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  Search Strategy:                                                    │   │
│  │    1. SKU exact match (GSI2 query)                                  │   │
│  │    2. SKU prefix match (GSI2 begins_with)                           │   │
│  │    3. Name contains (Scan with filter)                              │   │
│  │    4. Merge and deduplicate results                                 │   │
│  │                                                                      │   │
│  │  Limitations:                                                        │   │
│  │    - No full-text search                                            │   │
│  │    - Case-sensitive matching                                        │   │
│  │    - No relevance scoring                                           │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Future Implementation (OpenSearch):                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  ┌─────────────┐      ┌─────────────┐      ┌─────────────┐         │   │
│  │  │  DynamoDB   │─────▶│  DynamoDB   │─────▶│  OpenSearch │         │   │
│  │  │  (Source)   │      │  Streams    │      │  (Search)   │         │   │
│  │  └─────────────┘      └─────────────┘      └─────────────┘         │   │
│  │                                                                      │   │
│  │  Features:                                                           │   │
│  │    - Full-text search with fuzzy matching                           │   │
│  │    - Faceted search (by category, price range)                      │   │
│  │    - Relevance scoring                                              │   │
│  │    - Auto-suggestions                                               │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Caching Strategy

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          CACHING DESIGN                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  Browser Cache:                                                      │   │
│  │    - Product list: 5 minutes                                        │   │
│  │    - Category tree: 15 minutes                                      │   │
│  │    - Product details: 10 minutes                                    │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  API Gateway Cache (future):                                         │   │
│  │    - GET /categories/tree: 30 minutes                               │   │
│  │    - GET /products (by category): 5 minutes                         │   │
│  │    - Cache key includes: path + query params                        │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Cache Invalidation:                                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  On category change:                                                 │   │
│  │    - Invalidate category tree cache                                 │   │
│  │    - Invalidate affected product listings                           │   │
│  │                                                                      │   │
│  │  On product change:                                                  │   │
│  │    - Invalidate product detail cache                                │   │
│  │    - Invalidate category product listing                            │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Validation Rules

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          VALIDATION RULES                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Product Validation:                                                         │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ Field         │ Rules                                             │      │
│  ├───────────────┼───────────────────────────────────────────────────┤      │
│  │ SKU           │ Required, unique, alphanumeric, 3-50 chars       │      │
│  │ Name          │ Required, 3-200 chars                            │      │
│  │ Description   │ Optional, max 5000 chars                         │      │
│  │ Category      │ Required, must exist                             │      │
│  │ Base Price    │ Required, > 0, max 2 decimals                    │      │
│  │ Images        │ At least 1 required, max 10                      │      │
│  │ Status        │ draft | active | archived                        │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  Category Validation:                                                        │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ Field         │ Rules                                             │      │
│  ├───────────────┼───────────────────────────────────────────────────┤      │
│  │ Name          │ Required, 2-100 chars, unique at same level      │      │
│  │ Slug          │ Auto-generated, unique globally                  │      │
│  │ Parent        │ Optional, must exist, max depth 3                │      │
│  │ Sort Order    │ Optional, integer >= 0                           │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  Design Validation:                                                          │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ Field         │ Rules                                             │      │
│  ├───────────────┼───────────────────────────────────────────────────┤      │
│  │ Name          │ Required, 3-100 chars                            │      │
│  │ Code          │ Required, unique, alphanumeric                   │      │
│  │ Category      │ Optional, must exist if provided                 │      │
│  │ Images        │ At least 1 required                              │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 9. Error Handling

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
│  │ CAT012 │ Category has subcategories (cannot delete)                 │   │
│  │ CAT013 │ Max category depth exceeded                                │   │
│  │ CAT014 │ Circular category reference                                │   │
│  │ CAT015 │ Slug already exists                                        │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Design Errors:                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ CAT020 │ Design not found                                           │   │
│  │ CAT021 │ Design code already exists                                 │   │
│  │ CAT022 │ Design linked to products (cannot delete)                  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 10. Scalability & Performance

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
│  │ • GSI count: 4 (for various access patterns)                        │   │
│  │ • Batch operations: Max 25 items                                    │   │
│  │ • Pagination: Max 1000 items per query                              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Performance Optimizations:                                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Sparse indexes for optional attributes                            │   │
│  │ • Projection expressions to reduce data transfer                    │   │
│  │ • Parallel scans for bulk operations                                │   │
│  │ • Connection pooling (reuse DynamoDB client)                        │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 11. Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          DEPENDENCIES                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  External Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • AWS DynamoDB - Data storage                                       │   │
│  │ • AWS S3 - Image storage                                            │   │
│  │ • AWS CloudFront - Image CDN                                        │   │
│  │ • AWS CloudWatch - Logging & monitoring                             │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Internal Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Auth Lambda - Authentication                                      │   │
│  │ • Inventory Lambda - Stock management                               │   │
│  │ • Asset Lambda - Image processing                                   │   │
│  │ • Audit Lambda - Change logging                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Libraries:                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • aws-sdk-go-v2/dynamodb - DynamoDB client                          │   │
│  │ • aws-sdk-go-v2/s3 - S3 client                                      │   │
│  │ • go-chi/chi - HTTP routing                                         │   │
│  │ • google/uuid - ID generation                                       │   │
│  │ • gosimple/slug - Slug generation                                   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```
