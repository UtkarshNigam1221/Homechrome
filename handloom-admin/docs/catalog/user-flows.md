# Catalog Lambda - User Flows

## Overview
This document describes the user flows for the Catalog Lambda service, covering Categories, Designs, and Products management.

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
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Fill basic info:│
│ - SKU           │
│ - Name          │
│ - Description   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Select Category │◀────────────┐
│ from dropdown   │             │
└────────┬────────┘             │
         │                      │
         ├── Need new ──────────┤
         │   category?          │
         │                      │
         ▼                      │
┌─────────────────┐    ┌────────┴────────┐
│ Select Design   │    │ Create Category │
│ (optional)      │    │ (side modal)    │
└────────┬────────┘    └─────────────────┘
         │
         ▼
┌─────────────────┐
│ Set Pricing:    │
│ - Base price    │
│ - Tax category  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Upload Images:  │
│ - Primary image │
│ - Gallery       │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Add Attributes: │
│ - Material      │
│ - Weight        │
│ - Dimensions    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Set Initial     │
│ Inventory       │
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
           │ Create        │         │ Show errors   │
           │ Product       │         │ Highlight     │
           │ in DB         │         │ fields        │
           └───────┬───────┘         └───────────────┘
                   │
                   ▼
           ┌───────────────┐
           │ Create Audit  │
           │ Log Entry     │
           └───────┬───────┘
                   │
                   ▼
           ┌───────────────┐
           │ Show Success  │
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
│ View Category   │
│ Tree Structure  │
└────────┬────────┘
         │
         ├─────────── Add New ──────────┐
         │                              │
         ├─────────── Edit ─────────────┤
         │                              │
         ├─────────── Delete ───────────┤
         │                              │
         ├─────────── Reorder ──────────┤
         │                              │
         ▼                              ▼
┌─────────────────┐          ┌─────────────────┐
│ ADD NEW:        │          │ EDIT:           │
│ - Enter name    │          │ - Update name   │
│ - Add slug      │          │ - Change parent │
│ - Select parent │          │ - Update image  │
│ - Upload image  │          │ - Edit desc     │
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
              │ Invalidate    │
              │ Cache         │
              └───────┬───────┘
                      │
                      ▼
              ┌───────────────┐
              │ Refresh Tree  │
              │ View          │
              └───────┬───────┘
                      │
                      ▼
                 ┌────────┐
                 │  END   │
                 └────────┘
```

---

## 3. Product Search & Filter Flow

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
│ (page 1, 10/pg) │
└────────┬────────┘
         │
         ├─────────── Search ───────────┐
         │                              │
         ├─────────── Filter ───────────┤
         │                              │
         ├─────────── Sort ─────────────┤
         │                              │
         ▼                              ▼
┌─────────────────┐          ┌─────────────────┐
│ SEARCH:         │          │ FILTER BY:      │
│ Enter keyword   │          │ - Category      │
│ (name, SKU)     │          │ - Price range   │
│                 │          │ - Status        │
│ Debounce 300ms  │          │ - Stock level   │
└────────┬────────┘          └────────┬────────┘
         │                            │
         └────────────┬───────────────┘
                      │
                      ▼
              ┌───────────────┐
              │ Build Query   │
              │ Parameters    │
              └───────┬───────┘
                      │
                      ▼
              ┌───────────────┐
              │ Fetch Results │
              │ from API      │
              └───────┬───────┘
                      │
                      ▼
              ┌───────────────┐
              │ Display       │
              │ Results       │
              │ with          │
              │ Pagination    │
              └───────┬───────┘
                      │
                      ├────── No Results ─────┐
                      │                       │
                      ▼                       ▼
              ┌───────────────┐       ┌───────────────┐
              │ Show products │       │ Show "No      │
              │ Click for     │       │ products      │
              │ details       │       │ found"        │
              └───────────────┘       └───────────────┘
```

---

## 4. Bulk Product Upload Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         BULK PRODUCT UPLOAD FLOW                             │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────┐
    │  START   │
    └────┬─────┘
         │
         ▼
┌─────────────────┐
│ Navigate to     │
│ Bulk Operations │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Download CSV    │
│ Template        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Fill template   │
│ with product    │
│ data offline    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Upload CSV      │
│ file            │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Client-side     │
│ Validation:     │
│ - File format   │
│ - Column headers│
│ - Row count     │
└────────┬────────┘
         │
         ├── Invalid ─────────────────┐
         │                            │
         ▼                            ▼
┌─────────────────┐          ┌───────────────┐
│ Upload to S3    │          │ Show error    │
│ Get presigned   │          │ "Invalid      │
│ URL             │          │ format"       │
└────────┬────────┘          └───────────────┘
         │
         ▼
┌─────────────────┐
│ Submit Import   │
│ Job             │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Job Processing: │
│ - Validate rows │
│ - Create/Update │
│   products      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Show Progress:  │
│ - Total rows    │
│ - Processed     │
│ - Errors        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Job Complete:   │
│ - Success count │
│ - Error file    │
│   download      │
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  END   │
    └────────┘
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
│ Gallery         │
│ (Card View)     │
└────────┬────────┘
         │
         ├─────── Add New Design ──────┐
         │                             │
         ▼                             ▼
┌─────────────────┐          ┌─────────────────┐
│ VIEW DESIGN:    │          │ CREATE DESIGN:  │
│ - Design images │          │ - Name          │
│ - Linked        │          │ - Code          │
│   products      │          │ - Category      │
│ - Category      │          │ - Description   │
└────────┬────────┘          │ - Upload images │
         │                   └────────┬────────┘
         │                            │
         ├── Edit ────────────────────┤
         │                            │
         ├── Delete ──────────────────┤
         │                            │
         ├── Link to Product ─────────┤
         │                            │
         ▼                            ▼
┌─────────────────┐          ┌─────────────────┐
│ LINK DESIGN:    │          │ SAVE DESIGN:    │
│ - Select from   │          │ - Validate      │
│   unlinked      │          │ - Save to DB    │
│   products      │          │ - Show success  │
│ - Save link     │          │                 │
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

## State Diagram - Product Lifecycle

```
                         ┌─────────────────┐
                         │     DRAFT       │
                         │ (Not Published) │
                         └────────┬────────┘
                                  │
                            Publish
                                  │
                                  ▼
     ┌───────────┐       ┌─────────────────┐       ┌───────────┐
     │   OUT OF  │◀──────│     ACTIVE      │──────▶│  ARCHIVED │
     │   STOCK   │       │  (Published)    │       │           │
     └─────┬─────┘       └────────┬────────┘       └───────────┘
           │                      │                       ▲
           │              Update Stock                    │
           │                      │                  Archive
           └──────────────────────┴───────────────────────┘
                         Restock
```
