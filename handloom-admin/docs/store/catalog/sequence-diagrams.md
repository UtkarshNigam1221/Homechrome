# Store Catalog - Sequence Diagrams

## Overview
This document contains sequence diagrams for public storefront catalog operations.

---

## 1. List Products Sequence

```
┌────────┐          ┌──────────────┐          ┌────────────┐          ┌────────────┐          ┌─────────────┐
│ Client │          │ CatalogHandler│          │ Product    │          │ Product    │          │ (Transform) │
│        │          │ (store)      │          │ Service    │          │ Repo (DB)  │          │ toStore*()  │
└───┬────┘          └──────┬───────┘          └─────┬──────┘          └─────┬──────┘          └──────┬──────┘
    │                      │                        │                       │                        │
    │ GET /catalog/products│                        │                       │                        │
    │ ?category_id=<uuid>  │                        │                       │                        │
    │ &material=Silk       │                        │                       │                        │
    │ &min_price=500000    │                        │                       │                        │
    │─────────────────────▶│                        │                       │                        │
    │                      │                        │                       │                        │
    │                      │  Parse query params    │                       │                        │
    │                      │  Set status=ACTIVE     │                       │                        │
    │                      │──────────┐             │                       │                        │
    │                      │          │             │                       │                        │
    │                      │◀─────────┘             │                       │                        │
    │                      │                        │                       │                        │
    │                      │  List(req{             │                       │                        │
    │                      │    status=ACTIVE,      │                       │                        │
    │                      │    category_id,        │                       │                        │
    │                      │    material,           │                       │                        │
    │                      │    min_price})          │                       │                        │
    │                      │───────────────────────▶│                       │                        │
    │                      │                        │                       │                        │
    │                      │                        │  List(req)            │                        │
    │                      │                        │  (DynamoDB Query/Scan)│                        │
    │                      │                        │──────────────────────▶│                        │
    │                      │                        │                       │                        │
    │                      │                        │  []*Product,          │                        │
    │                      │                        │  Pagination            │                        │
    │                      │                        │◀──────────────────────│                        │
    │                      │                        │                       │                        │
    │                      │  ListProductsResponse  │                       │                        │
    │                      │  (products, pagination)│                       │                        │
    │                      │◀───────────────────────│                       │                        │
    │                      │                        │                       │                        │
    │                      │  For each product:     │                       │                        │
    │                      │  toStoreProduct(p,     │                       │                        │
    │                      │    available_qty > 0)  │                       │                        │
    │                      │──────────────────────────────────────────────▶│                        │
    │                      │                        │                       │                        │
    │                      │  []*StoreProduct       │                       │                        │
    │                      │  (cost_price stripped, │                       │                        │
    │                      │   in_stock set)        │                       │                        │
    │                      │◀──────────────────────────────────────────────│                        │
    │                      │                        │                       │                        │
    │ 200 OK               │                        │                       │                        │
    │ {success: true,      │                        │                       │                        │
    │  data: [...],        │                        │                       │                        │
    │  meta: {limit,       │                        │                       │                        │
    │   next_cursor,       │                        │                       │                        │
    │   has_more}}         │                        │                       │                        │
    │◀─────────────────────│                        │                       │                        │
    │                      │                        │                       │                        │
```

---

## 2. Get Product by Slug Sequence

```
┌────────┐          ┌──────────────┐          ┌────────────┐          ┌────────────┐          ┌────────────┐
│ Client │          │ CatalogHandler│          │ Product    │          │ Product    │          │ Category   │
│        │          │ (store)      │          │ Service    │          │ Repo (DB)  │          │ Repo (DB)  │
└───┬────┘          └──────┬───────┘          └─────┬──────┘          └─────┬──────┘          └─────┬──────┘
    │                      │                        │                       │                       │
    │ GET /catalog/products│                        │                       │                       │
    │ /royal-blue-banarasi-│                        │                       │                       │
    │ silk-saree           │                        │                       │                       │
    │─────────────────────▶│                        │                       │                       │
    │                      │                        │                       │                       │
    │                      │  isUUID("royal-blue..") │                       │                       │
    │                      │  → false (is a slug)   │                       │                       │
    │                      │──────────┐             │                       │                       │
    │                      │          │             │                       │                       │
    │                      │◀─────────┘             │                       │                       │
    │                      │                        │                       │                       │
    │                      │  findProductBySlug:    │                       │                       │
    │                      │  List(status=ACTIVE,   │                       │                       │
    │                      │   search=slug,         │                       │                       │
    │                      │   limit=100)           │                       │                       │
    │                      │───────────────────────▶│                       │                       │
    │                      │                        │                       │                       │
    │                      │                        │  List(req)            │                       │
    │                      │                        │──────────────────────▶│                       │
    │                      │                        │                       │                       │
    │                      │                        │  Search results       │                       │
    │                      │                        │◀──────────────────────│                       │
    │                      │                        │                       │                       │
    │                      │  Products[]            │                       │                       │
    │                      │◀───────────────────────│                       │                       │
    │                      │                        │                       │                       │
    │                      │  Find exact slug match │                       │                       │
    │                      │  in results            │                       │                       │
    │                      │──────────┐             │                       │                       │
    │                      │          │             │                       │                       │
    │                      │◀─────────┘             │                       │                       │
    │                      │                        │                       │                       │
    │                      │  GetByID(product.ID)   │                       │                       │
    │                      │  (full relations)      │                       │                       │
    │                      │───────────────────────▶│                       │                       │
    │                      │                        │                       │                       │
    │                      │                        │  GetByID(id)          │                       │
    │                      │                        │──────────────────────▶│                       │
    │                      │                        │                       │                       │
    │                      │                        │  Product              │                       │
    │                      │                        │◀──────────────────────│                       │
    │                      │                        │                       │                       │
    │                      │                        │  GetByID(category_id) │                       │
    │                      │                        │──────────────────────────────────────────────▶│
    │                      │                        │                       │                       │
    │                      │                        │  Category             │                       │
    │                      │                        │◀──────────────────────────────────────────────│
    │                      │                        │                       │                       │
    │                      │  ProductWithRelations  │                       │                       │
    │                      │  {product, category,   │                       │                       │
    │                      │   inventory}           │                       │                       │
    │                      │◀───────────────────────│                       │                       │
    │                      │                        │                       │                       │
    │                      │  toStoreProductFrom    │                       │                       │
    │                      │  Relations(pwr)        │                       │                       │
    │                      │──────────┐             │                       │                       │
    │                      │          │             │                       │                       │
    │                      │◀─────────┘             │                       │                       │
    │                      │                        │                       │                       │
    │ 200 OK               │                        │                       │                       │
    │ {success: true,      │                        │                       │                       │
    │  data: {             │                        │                       │                       │
    │    ...product,       │                        │                       │                       │
    │    in_stock: true,   │                        │                       │                       │
    │    category: {       │                        │                       │                       │
    │      id, name, slug  │                        │                       │                       │
    │    }                 │                        │                       │                       │
    │  }}                  │                        │                       │                       │
    │◀─────────────────────│                        │                       │                       │
    │                      │                        │                       │                       │
```

---

## 3. Check Availability Sequence

```
┌────────┐          ┌──────────────┐          ┌────────────┐          ┌────────────┐          ┌─────────────┐
│ Client │          │ CatalogHandler│          │ Product    │          │ Inventory  │          │ Inventory   │
│        │          │ (store)      │          │ Service    │          │ Service    │          │ Repo (DB)   │
└───┬────┘          └──────┬───────┘          └─────┬──────┘          └─────┬──────┘          └──────┬──────┘
    │                      │                        │                       │                        │
    │ GET /catalog/products│                        │                       │                        │
    │ /<product-uuid>/     │                        │                       │                        │
    │ availability         │                        │                       │                        │
    │─────────────────────▶│                        │                       │                        │
    │                      │                        │                       │                        │
    │                      │  Validate ID not empty │                       │                        │
    │                      │──────────┐             │                       │                        │
    │                      │          │             │                       │                        │
    │                      │◀─────────┘             │                       │                        │
    │                      │                        │                       │                        │
    │                      │  GetByID(productID)    │                       │                        │
    │                      │───────────────────────▶│                       │                        │
    │                      │                        │                       │                        │
    │                      │  ProductWithRelations  │                       │                        │
    │                      │  (verify ACTIVE)       │                       │                        │
    │                      │◀───────────────────────│                       │                        │
    │                      │                        │                       │                        │
    │                      │  Check status=ACTIVE   │                       │                        │
    │                      │──────────┐             │                       │                        │
    │                      │          │             │                       │                        │
    │                      │◀─────────┘             │                       │                        │
    │                      │                        │                       │                        │
    │                      │  GetByProductID(id)    │                       │                        │
    │                      │──────────────────────────────────────────────▶│                        │
    │                      │                        │                       │                        │
    │                      │                        │                       │  GetByProductID(id)    │
    │                      │                        │                       │──────────────────────▶│
    │                      │                        │                       │                        │
    │                      │                        │                       │  Inventory record      │
    │                      │                        │                       │  (or NotFound)         │
    │                      │                        │                       │◀──────────────────────│
    │                      │                        │                       │                        │
    │                      │  Inventory              │                       │                        │
    │                      │  {available_qty: 7}    │                       │                        │
    │                      │◀──────────────────────────────────────────────│                        │
    │                      │                        │                       │                        │
    │                      │  [If NotFound: use      │                       │                        │
    │                      │   pwr.AvailableQty]    │                       │                        │
    │                      │──────────┐             │                       │                        │
    │                      │          │             │                       │                        │
    │                      │◀─────────┘             │                       │                        │
    │                      │                        │                       │                        │
    │ 200 OK               │                        │                       │                        │
    │ {success: true,      │                        │                       │                        │
    │  data: {             │                        │                       │                        │
    │    in_stock: true,   │                        │                       │                        │
    │    available_quantity:│                        │                       │                        │
    │    7                 │                        │                       │                        │
    │  }}                  │                        │                       │                        │
    │◀─────────────────────│                        │                       │                        │
    │                      │                        │                       │                        │
```

---

## 4. List Categories Sequence

```
┌────────┐          ┌──────────────┐          ┌────────────┐          ┌────────────┐
│ Client │          │ CatalogHandler│          │ Category   │          │ Category   │
│        │          │ (store)      │          │ Service    │          │ Repo (DB)  │
└───┬────┘          └──────┬───────┘          └─────┬──────┘          └─────┬──────┘
    │                      │                        │                       │
    │ GET /catalog/         │                        │                       │
    │ categories            │                        │                       │
    │ ?search=pashmina     │                        │                       │
    │─────────────────────▶│                        │                       │
    │                      │                        │                       │
    │                      │  Parse pagination      │                       │
    │                      │  Set status=ACTIVE     │                       │
    │                      │  Set search param      │                       │
    │                      │──────────┐             │                       │
    │                      │          │             │                       │
    │                      │◀─────────┘             │                       │
    │                      │                        │                       │
    │                      │  List(req{             │                       │
    │                      │    status=ACTIVE,      │                       │
    │                      │    search="pashmina"}) │                       │
    │                      │───────────────────────▶│                       │
    │                      │                        │                       │
    │                      │                        │  List(req)            │
    │                      │                        │  (DynamoDB Query)     │
    │                      │                        │──────────────────────▶│
    │                      │                        │                       │
    │                      │                        │  []*Category,         │
    │                      │                        │  Pagination            │
    │                      │                        │◀──────────────────────│
    │                      │                        │                       │
    │                      │  ListCategoriesResponse│                       │
    │                      │◀───────────────────────│                       │
    │                      │                        │                       │
    │                      │  For each: toStore     │                       │
    │                      │  Category(c)           │                       │
    │                      │──────────┐             │                       │
    │                      │          │             │                       │
    │                      │◀─────────┘             │                       │
    │                      │                        │                       │
    │ 200 OK               │                        │                       │
    │ {success: true,      │                        │                       │
    │  data: [...],        │                        │                       │
    │  meta: {limit,       │                        │                       │
    │   next_cursor,       │                        │                       │
    │   has_more}}         │                        │                       │
    │◀─────────────────────│                        │                       │
    │                      │                        │                       │
```
