# Store Catalog - User Flows

## Overview
This document describes the user flows for public product and category browsing on the B2C storefront.

---

## 1. Browse Categories Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         BROWSE CATEGORIES FLOW                               │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Customer visits  │
│ /categories page │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ GET /catalog/    │
│ categories       │
│ (limit=20)       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Server filters  │
│ status=ACTIVE   │
│ (hardcoded)     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Display category│
│ grid:           │
│ - Image         │
│ - Name          │
│ - Product count │
└────────┬────────┘
         │
         ├──────────── Search ───────────────┐
         │                                    │
         ▼                                    ▼
┌─────────────────┐                  ┌───────────────┐
│ Customer        │                  │ GET /catalog/  │
│ scrolls down    │                  │ categories     │
│                 │                  │ ?search=       │
└────────┬────────┘                  │ "banarasi"     │
         │                           └───────┬───────┘
         ▼                                    │
┌─────────────────┐                           ▼
│ Has more?       │                  ┌───────────────┐
│ Load next page  │                  │ Display       │
│ (cursor-based)  │                  │ filtered      │
└────────┬────────┘                  │ results       │
         │                           └───────────────┘
         ▼
┌─────────────────┐
│ Customer clicks │
│ a category      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Navigate to     │
│ /c/{slug}       │
│ (products page) │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## 2. Browse Products with Filters Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     BROWSE PRODUCTS WITH FILTERS FLOW                        │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Customer lands  │
│ on category page│
│ /c/banarasi-    │
│ sarees          │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ GET /catalog/    │
│ categories/      │
│ banarasi-sarees  │
│ (get cat details)│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ GET /catalog/    │
│ products         │
│ ?category_id=    │
│ <cat-uuid>       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Display product │
│ grid with       │
│ filter sidebar  │
└────────┬────────┘
         │
         ├──────── Apply Filters ──────────┐
         │                                  │
         ▼                                  ▼
┌─────────────────┐              ┌─────────────────────┐
│ Customer browses│              │ Customer selects:    │
│ without filters │              │ - Material: "Silk"   │
│                 │              │ - Price: 5000-15000  │
└────────┬────────┘              │ - Color: "Blue"      │
         │                       └──────────┬──────────┘
         │                                  │
         │                                  ▼
         │                       ┌─────────────────────┐
         │                       │ GET /catalog/products│
         │                       │ ?category_id=<uuid>  │
         │                       │ &material=Silk       │
         │                       │ &min_price=500000    │
         │                       │ &max_price=1500000   │
         │                       │ &color=Blue          │
         │                       └──────────┬──────────┘
         │                                  │
         │                                  ▼
         │                       ┌─────────────────────┐
         │                       │ Display filtered     │
         │                       │ products             │
         │                       │ (ACTIVE only)        │
         │                       └──────────┬──────────┘
         │                                  │
         │◀─────────────────────────────────┘
         │
         ├──────── Search ─────────────────┐
         │                                  │
         ▼                                  ▼
┌─────────────────┐              ┌─────────────────────┐
│ Scroll for more │              │ GET /catalog/products│
│ (cursor-based   │              │ /search?search=      │
│ pagination)     │              │ "jangla silk"        │
└────────┬────────┘              └──────────┬──────────┘
         │                                  │
         ▼                                  ▼
┌─────────────────┐              ┌─────────────────────┐
│ Customer clicks │              │ Display search       │
│ a product card  │              │ results              │
└────────┬────────┘              └─────────────────────┘
         │
         ▼
┌─────────────────┐
│ Navigate to     │
│ /p/{slug}       │
│ (product detail)│
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## 3. View Product Detail Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        VIEW PRODUCT DETAIL FLOW                              │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Customer visits  │
│ /p/royal-blue-   │
│ banarasi-silk-   │
│ saree            │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ GET /catalog/    │
│ products/        │
│ royal-blue-      │
│ banarasi-silk-   │
│ saree            │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Handler detects │
│ slug (not UUID) │
│ → search + match│
└────────┬────────┘
         │
         ├──────────── Not Found ────────────┐
         │                                    │
         ▼                                    ▼
┌─────────────────┐                  ┌───────────────┐
│ Product found   │                  │ Show 404 page │
│ and ACTIVE      │                  └───────────────┘
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Fetch full      │
│ product with    │
│ relations       │
│ (category info) │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Display product │
│ detail page:    │
│ - Image gallery │
│ - Name, price   │
│ - Description   │
│ - Attributes    │
│ - Material      │
│ - Origin info   │
│ - In stock badge│
│ - Category link │
└────────┬────────┘
         │
         ├──── Check Availability ───────────┐
         │                                    │
         ▼                                    ▼
┌─────────────────┐              ┌─────────────────────┐
│ Customer clicks │              │ GET /catalog/products│
│ "Add to Cart"   │              │ /{id}/availability   │
│ (→ cart flow)   │              │ → {in_stock: true,   │
└────────┬────────┘              │    available_qty: 7} │
         │                       └─────────────────────┘
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```

---

## 4. Check Product Availability Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     CHECK PRODUCT AVAILABILITY FLOW                          │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Customer on     │
│ product detail  │
│ page            │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ GET /catalog/    │
│ products/{id}/   │
│ availability     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Verify product  │
│ exists and      │
│ status=ACTIVE   │
└────────┬────────┘
         │
         ├──────────── Not Found ────────────┐
         │                                    │
         ▼                                    ▼
┌─────────────────┐                  ┌───────────────┐
│ Fetch live      │                  │ Return 404    │
│ inventory from  │                  └───────────────┘
│ Inventory table │
└────────┬────────┘
         │
         ├──────── No Inventory Record ──────┐
         │                                    │
         ▼                                    ▼
┌─────────────────┐                  ┌───────────────┐
│ Return live     │                  │ Fall back to  │
│ availability:   │                  │ product's     │
│ {in_stock,      │                  │ denormalized  │
│ available_qty}  │                  │ available_qty │
└────────┬────────┘                  └───────┬───────┘
         │                                    │
         └────────────────────────────────────┘
         │
         ▼
┌─────────────────┐
│ Display on UI:  │
│                 │
│ ┌─────────────┐ │
│ │ In Stock    │ │ ← in_stock=true
│ │ (7 left)    │ │
│ └─────────────┘ │
│                 │
│ ┌─────────────┐ │
│ │ Out of Stock│ │ ← in_stock=false
│ └─────────────┘ │
│                 │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
```
