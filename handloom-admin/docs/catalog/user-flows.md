# Catalog Lambda - User Flows

## Overview
This document describes the user flows for the Catalog Lambda service, covering Categories (flat with custom attributes), Designs, Products, and Inventory management.

---

## 1. Create Product Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CREATE PRODUCT FLOW                                │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Navigate to     │
│ Products > New  │
│ (click Add      │
│  Product button)│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Fill basic info:│
│ - Name          │
│ - SKU           │
│ - Description   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Select Category │
│ from flat list  │
│ dropdown        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Dynamic fields  │
│ appear based on │
│ category's      │
│ own_attributes: │
│ - SELECT fields │
│   with options  │
│ - TEXT inputs   │
│ - NUMBER inputs │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Select Design   │
│ (filtered by    │
│  category)      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Set Pricing:    │
│ - Base price    │
│ - Selling price │
│ - Cost price    │
│ (all in paise)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Upload Images:  │
│ - Primary image │
│ - Gallery       │
│ (via presigned  │
│  S3 URLs)       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Set Inventory:  │
│ - Initial stock │
│ - Low stock     │
│   threshold     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│ Review &        │────▶│ Validation      │
│ Submit          │     │ Check           │
└─────────────────┘     └────────┬────────┘
                                 │
                    ┌────────────┴────────────┐
                    │                         │
                    ▼                         ▼
           ┌───────────────┐         ┌───────────────┐
           │   VALID       │         │   INVALID     │
           └───────┬───────┘         └───────┬───────┘
                   │                         │
                   ▼                         ▼
           ┌───────────────┐         ┌───────────────┐
           │ Backend:      │         │ Show errors   │
           │ 1. Create     │         │ Highlight     │
           │    product    │         │ fields        │
           │ 2. Create     │         └───────────────┘
           │    attr index │
           │    records    │
           │ 3. Update     │
           │    ATTR_VALUES│
           │ 4. Create     │
           │    inventory  │
           │ 5. Increment  │
           │    counts     │
           └───────┬───────┘
                   │
                   ▼
           ┌───────────────┐
           │ Show Success  │
           │ Toast         │
           │ Redirect to   │
           │ Product List  │
           └───────┬───────┘
                   │
                   ▼
              ┌────────┐
              │  END   │
              └────────┘
```

---

## 2. Manage Categories Flow

Categories are flat (no tree/hierarchy). The admin manages a simple list of categories, each with its own set of custom attributes.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          CATEGORY MANAGEMENT FLOW                            │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Navigate to     │
│ Categories      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ View Flat       │
│ Category List   │
│ (table with     │
│  search, status │
│  filter,        │
│  pagination)    │
└────────┬────────┘
         │
         ├─────────── Add New ──────────┐
         │                              │
         ├─────────── Edit ─────────────┤
         │                              │
         ├─────────── Delete ───────────┤
         │                              │
         ├─────────── Manage Attrs ─────┤
         │                              │
         ▼                              ▼
┌─────────────────┐          ┌─────────────────┐
│ ADD NEW:        │          │ EDIT:           │
│ - Enter name    │          │ - Update name   │
│ - Auto slug     │          │ - Update desc   │
│ - Add desc      │          │ - Update image  │
│ - Upload image  │          │ - Change status │
│ - Add attrs     │          │                 │
│   (optional)    │          │                 │
└────────┬────────┘          └────────┬────────┘
         │                            │
         └────────────┬───────────────┘
                      │
                      ▼
              ┌───────────────┐
              │ Save Changes  │
              └───────┬───────┘
                      │
                      ▼
              ┌───────────────┐
              │ Refresh List  │
              │ Show Toast    │
              └───────┬───────┘
                      │
                      ▼
                 ┌────────┐
                 │  END   │
                 └────────┘


DELETE FLOW:
┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│ Click Delete  │────▶│ Confirm       │────▶│ Check: has    │
│ on category   │     │ Modal         │     │ products?     │
└───────────────┘     └───────────────┘     └───────┬───────┘
                                                    │
                                       ┌────────────┴────────────┐
                                       │                         │
                                       ▼                         ▼
                              ┌───────────────┐         ┌───────────────┐
                              │ No products   │         │ Has products  │
                              │ → Delete OK   │         │ → Show error  │
                              └───────────────┘         └───────────────┘
```

---

## 3. Category Attribute Management Flow

Each category can define custom attributes that products must provide. Searchable attributes are indexed for efficient filtering on the products page.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     ATTRIBUTE MANAGEMENT FLOW                                │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Open Category   │
│ Form Modal      │
│ (Attributes Tab)│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ View existing   │
│ attributes list │
│ (name, type,    │
│  required,      │
│  searchable)    │
└────────┬────────┘
         │
         ├─────── Add Attribute ──────┐
         │                            │
         ├─────── Edit Attribute ─────┤
         │                            │
         ├─────── Delete Attribute ───┤
         │                            │
         ▼                            ▼
┌─────────────────┐          ┌─────────────────┐
│ ADD ATTRIBUTE:  │          │ EDIT ATTRIBUTE: │
│ - Name (key)    │          │ - Label         │
│ - Label         │          │ - Required flag │
│ - Type:         │          │ - Searchable    │
│   SELECT,       │          │   flag          │
│   MULTI_SELECT, │          │ - Display order │
│   TEXT, NUMBER,  │          │ - Options (for  │
│   BOOLEAN       │          │   SELECT type)  │
│ - Required flag │          │   - value       │
│ - Searchable    │          │   - label       │
│   flag          │          │   - surcharge   │
│ - Display order │          │                 │
│ - Options (for  │          │                 │
│   SELECT type)  │          │                 │
└────────┬────────┘          └────────┬────────┘
         │                            │
         └────────────┬───────────────┘
                      │
                      ▼
              ┌───────────────┐
              │ POST/PATCH to │
              │ /categories/  │
              │ {id}/         │
              │ attributes/   │
              │ {attrName}    │
              └───────┬───────┘
                      │
                      ▼
              ┌───────────────┐
              │ Show Success  │
              │ Refresh attrs │
              └───────┬───────┘
                      │
                      ▼
                 ┌────────┐
                 │  END   │
                 └────────┘

Note: When searchable=true, the attribute values are:
  1. Indexed in ProductAttributeIndex records (GSI-queryable)
  2. Accumulated in CategoryAttributeValues (pre-computed sets)
  3. Shown as filter dropdowns in the Product Search sidebar
```

---

## 4. Product Search & Filter Flow

The products page supports category-based filtering with dynamic attribute filter dropdowns that appear based on the selected category's searchable attributes.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        PRODUCT SEARCH & FILTER FLOW                          │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Navigate to     │
│ Products List   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Load products   │
│ (page 1)        │
│ No filters      │
└────────┬────────┘
         │
         ├─────────── Search ───────────┐
         │                              │
         ├─────────── Category ─────────┤
         │           Filter             │
         ├─────────── Status ───────────┤
         │           Filter             │
         ▼                              ▼
┌─────────────────┐          ┌─────────────────────────┐
│ SEARCH:         │          │ SELECT CATEGORY:        │
│ Enter keyword   │          │ Choose from flat list   │
│ (name, SKU)     │          │ dropdown                │
│                 │          │                         │
│ Debounce 300ms  │          │ On selection:           │
└────────┬────────┘          │ 1. Fetch filter options │
         │                   │    GET /products/       │
         │                   │    filter-options/      │
         │                   │    {categoryId}         │
         │                   │                         │
         │                   │ 2. Fetch category attrs │
         │                   │    GET /categories/     │
         │                   │    {id}/attributes      │
         │                   │                         │
         │                   │ 3. Render dynamic       │
         │                   │    filter sidebar:      │
         │                   │    - Dropdown per       │
         │                   │      searchable attr    │
         │                   │    - Options from       │
         │                   │      filter-options     │
         │                   │      response           │
         │                   └────────────┬────────────┘
         │                                │
         └────────────┬───────────────────┘
                      │
                      ▼
              ┌───────────────┐
              │ User selects  │
              │ attribute     │
              │ filter values │
              │ (e.g. color=  │
              │ red, material │
              │ = silk)       │
              └───────┬───────┘
                      │
                      ▼
              ┌───────────────┐
              │ Build Query:  │
              │ category_id=  │
              │ cat_123       │
              │ &attribute_   │
              │ filters=      │
              │ {"color":     │
              │  ["red"],     │
              │  "material":  │
              │  ["silk"]}    │
              └───────┬───────┘
                      │
                      ▼
              ┌───────────────┐
              │ Fetch Results │
              │ GET /products │
              │ with filters  │
              └───────┬───────┘
                      │
                      ▼
              ┌───────────────┐
              │ Display       │
              │ filtered      │
              │ results with  │
              │ pagination    │
              └───────┬───────┘
                      │
                      ├────── No Results ─────┐
                      │                       │
                      ▼                       ▼
              ┌───────────────┐       ┌───────────────┐
              │ Show products │       │ Show "No      │
              │ table with    │       │ products      │
              │ details       │       │ found"        │
              └───────────────┘       └───────────────┘
```

---

## 5. Design Management Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          DESIGN MANAGEMENT FLOW                              │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Navigate to     │
│ Designs         │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ View Design     │
│ List/Gallery    │
│ (filter by      │
│  category,      │
│  status, search)│
└────────┬────────┘
         │
         ├─────── Add New Design ──────┐
         │                             │
         ▼                             ▼
┌─────────────────┐          ┌─────────────────┐
│ VIEW DESIGN:    │          │ CREATE DESIGN:  │
│ - Design images │          │ - Name          │
│ - Category      │          │ - Category      │
│ - Attributes    │          │ - Description   │
│ - Product count │          │ - Upload images │
│                 │          │ - Attributes    │
│                 │          │   (name: values)│
└────────┬────────┘          └────────┬────────┘
         │                            │
         ├── Edit ────────────────────┤
         │                            │
         ├── Delete ──────────────────┤
         │                            │
         ▼                            ▼
┌─────────────────┐          ┌─────────────────┐
│ EDIT DESIGN:    │          │ SAVE DESIGN:    │
│ - Update name   │          │ - Validate      │
│ - Update desc   │          │ - Save to DB    │
│ - Change images │          │ - Show success  │
│ - Update attrs  │          │                 │
│ - Change status │          │                 │
└────────┬────────┘          └────────┬────────┘
         │                            │
         └────────────┬───────────────┘
                      │
                      ▼
                 ┌────────┐
                 │  END   │
                 └────────┘
```

---

## 6. Inventory Management Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        INVENTORY MANAGEMENT FLOW                             │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Navigate to     │
│ Product Detail  │
│ or Inventory tab│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ View Inventory: │
│ - Quantity      │
│ - Reserved      │
│ - Available     │
│ - Low stock?    │
│ - Last restock  │
└────────┬────────┘
         │
         ├─────── Add Stock ──────────┐
         │                            │
         ├─────── Remove Stock ───────┤
         │                            │
         ├─────── Adjust Stock ───────┤
         │                            │
         ├─────── View History ───────┤
         │                            │
         ▼                            ▼
┌─────────────────┐          ┌─────────────────┐
│ ADD STOCK:      │          │ ADJUST STOCK:   │
│ Enter quantity  │          │ Enter new total │
│ Enter reason    │          │ Enter reason    │
│ Submit          │          │ Submit          │
└────────┬────────┘          └────────┬────────┘
         │                            │
         └────────────┬───────────────┘
                      │
                      ▼
              ┌───────────────┐
              │ API creates   │
              │ transaction   │
              │ record        │
              │ (PK=INVENTORY │
              │ #id,          │
              │ SK=TXN#...)   │
              └───────┬───────┘
                      │
                      ▼
              ┌───────────────┐
              │ Show updated  │
              │ inventory     │
              │ + transaction │
              │   result      │
              └───────┬───────┘
                      │
                      ▼
                 ┌────────┐
                 │  END   │
                 └────────┘
```

---

## State Diagram - Product Lifecycle

```
                         ┌─────────────────┐
                         │     DRAFT       │
                         │ (Not Published) │
                         └────────┬────────┘
                                  │
                            Activate
                                  │
                                  ▼
                         ┌─────────────────┐
                         │     ACTIVE      │
                         │  (Published)    │
                         └────────┬────────┘
                                  │
                             Deactivate
                                  │
                                  ▼
                         ┌─────────────────┐
                         │   INACTIVE      │
                         │  (Hidden)       │
                         └─────────────────┘

Note: Status transitions are simple: DRAFT → ACTIVE ↔ INACTIVE
Products can be reactivated from INACTIVE back to ACTIVE.
Deletion removes the product entirely (with all indexes,
inventory records, and transaction history).
```
