# Product Status on Creation & Category Product Ranking

**Date:** 2026-02-22
**Status:** Approved
**Scope:** Bug fix for product status on creation + category-level product ranking with drag-and-drop admin UI

## Problem

### Bug: Product status ignored on creation
The admin product form allows selecting a status (DRAFT/ACTIVE/INACTIVE) during creation, but the backend `CreateProductRequest` has no `Status` field. `NewProduct()` force-sets status to `DRAFT`. The frontend sends the status value but it's silently ignored.

### Feature: Product ranking within categories
There is no way for admins to control the display order of products within a category. Products appear in arbitrary order (DynamoDB internal ordering by product ID). Admins need to control which products appear first on the B2C storefront's category pages.

## Decision

### Status Fix
Add `Status *ProductStatus` to `CreateProductRequest`. Default to `DRAFT` if not provided.

### Ranking: sort_order field on Product (Approach 1)
Add an integer `SortOrder` field to the Product entity. Encode it into `GSI1SK` so DynamoDB returns products pre-sorted by rank when querying by category. This is optimal for a read-heavy system — zero application-level sorting on reads.

**Alternatives considered:**
- **Separate ranking table** (array of product IDs per category): Single write on reorder, but requires two-step reads and complex pagination. Poor for read-heavy.
- **Fractional ordering** (float sort_order): Single write per move, but floating-point precision issues and periodic rebalancing needed.

## Design

### Data Model Changes

#### Product Entity
```go
type Product struct {
    // ... existing fields ...
    SortOrder int `json:"sort_order" dynamodbav:"sort_order"`
}
```

#### GSI1SK Encoding
Current: `GSI1SK = "PRODUCT#" + p.ID`
New: `GSI1SK = "RANK#" + zeroPad(p.SortOrder, 6) + "#" + p.ID`

Example: `RANK#000001#abc123`

- Zero-padded to 6 digits (supports up to 999,999 products per category)
- `#<id>` suffix guarantees uniqueness if two products share the same sort_order
- DynamoDB string sort order: `RANK#000001` < `RANK#000002` — natural ascending order

#### Default Sort Order
- New products: `SortOrder = 0` (appear first — lowest number = highest priority)
- Existing products without explicit ranking: treated as `SortOrder = 0`, ordered by ID as tiebreaker (existing behavior preserved)

### CreateProductRequest Change

```go
type CreateProductRequest struct {
    // ... existing fields ...
    Status *ProductStatus `json:"status,omitempty" validate:"omitempty,oneof=ACTIVE INACTIVE DRAFT"`
}
```

`NewProduct()` uses `req.Status` if provided, defaults to `ProductStatusDraft`.

### Reorder API

**Endpoint:** `PUT /admin/products/categories/{categoryId}/reorder`

**Request:**
```json
{
  "product_ids": ["id1", "id2", "id3"]
}
```

**Behavior:**
1. Validates all product IDs belong to the given category
2. Assigns sequential `SortOrder` values: position 0 → SortOrder=1, position 1 → SortOrder=2, etc.
3. Updates each product's `SortOrder` field and `GSI1SK` key in a DynamoDB transaction
4. DynamoDB transactions support max 100 items. For categories with >100 products, use multiple sequential transactions (acceptable since reordering is an infrequent admin operation)

**Response:**
```json
{
  "success": true,
  "data": { "reordered": 3 }
}
```

### B2C Storefront Impact

**None.** The store catalog handler queries GSI1 (`GSI1PK=CATEGORY#<id>`) which now returns products sorted by `GSI1SK=RANK#...`. Products automatically appear in ranked order on category pages. No store code changes needed.

### Admin Frontend

**Drag-and-drop UI:**
- New "Manage Order" view accessible from the products list when filtered by a specific category
- Uses `@dnd-kit/core` + `@dnd-kit/sortable` (lightweight, accessible, React-friendly)
- Shows product name, SKU, image thumbnail, status in a sortable list
- "Save Order" button sends the reordered product ID list to the reorder endpoint
- Optimistic update with rollback on failure

**Products table enhancement:**
- Show `Sort Order` column when viewing products within a specific category
- Products listed in sort_order ascending by default

## File Changes

### Backend

| File | Change |
|------|--------|
| `internal/domain/entity.go` | Add `SortOrder` field to Product, update `SetKeys()` to encode rank in GSI1SK, update `NewProduct()` to accept status |
| `internal/domain/service.go` | Add `Status` to `CreateProductRequest`, add `ReorderProductsRequest` struct |
| `internal/service/product_service.go` | Update `Create()` to pass status, add `ReorderProducts()` method |
| `internal/repository/dynamodb/product_repository.go` | Add `BatchUpdateSortOrder()`, update `CreateWithAttributeIndexes()` for new GSI1SK format |
| `internal/handler/product_handler.go` | Add `Reorder` handler + route |

### Frontend

| File | Change |
|------|--------|
| `package.json` | Add `@dnd-kit/core`, `@dnd-kit/sortable`, `@dnd-kit/utilities` |
| `src/features/products/api.ts` | Add `reorderProducts()` API call |
| `src/features/products/types.ts` | Add `sort_order` to Product type, add `ReorderProductsRequest` |
| `src/features/products/components/ProductRankingModal/` | New drag-and-drop ranking modal component |
| `src/features/products/components/ProductList.tsx` (or equivalent) | Add "Manage Order" button when category is selected |

## Migration

Existing products have no `SortOrder` field (defaults to 0 in Go). Their `GSI1SK` is `PRODUCT#<id>`. Two options:

1. **Lazy migration**: When admin first reorders a category, all products in that category get assigned sequential sort_order values. Until then, products with `GSI1SK=PRODUCT#<id>` sort after `RANK#...` items (since `P` > `R` is false — actually `P` < `R`, so old items sort before new ones).

   Wait — `P` (0x50) < `R` (0x52), so `PRODUCT#...` sorts BEFORE `RANK#...`. This means un-migrated products would appear first, which is acceptable as a starting state.

2. **Explicit migration script**: Batch update all existing products with sequential sort_order values per category. Cleaner but requires a one-time script.

**Recommendation:** Lazy migration. When the reorder endpoint is called for a category, it assigns sort_order to ALL products in that category (not just the ones in the request). This naturally migrates data on first use.

## Implementation Order

1. Backend: Add `SortOrder` field + update `SetKeys()` + GSI1SK encoding
2. Backend: Fix `CreateProductRequest` to accept `Status`
3. Backend: Add `ReorderProducts` service method + repository batch update
4. Backend: Add reorder handler + route
5. Frontend: Add `@dnd-kit` dependencies
6. Frontend: Add reorder API call
7. Frontend: Build drag-and-drop ranking modal
8. Frontend: Wire "Manage Order" button into products list
