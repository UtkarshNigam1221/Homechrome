# Product Status Fix & Category Product Ranking — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix the bug where product status is ignored on creation, and add category-level product ranking with a drag-and-drop admin UI so admins control product display order on the B2C storefront.

**Architecture:** Add `Status` field to `CreateProductRequest` and `SortOrder` field to `Product`. Encode `SortOrder` into `GSI1SK` so DynamoDB returns products pre-sorted by rank per category. New `PUT /admin/products/categories/{categoryId}/reorder` endpoint accepts an ordered list of product IDs. Frontend gets a drag-and-drop modal using `@dnd-kit`. The B2C storefront benefits automatically — no store code changes needed.

**Tech Stack:** Go 1.24, DynamoDB (transactions), Chi router, React 19, @dnd-kit/core + @dnd-kit/sortable, React Query, Zustand

---

## Task 1: Fix product status on creation — backend

**Files:**
- Modify: `internal/domain/service.go:206-229` — add Status field to CreateProductRequest
- Modify: `internal/domain/entity.go:354-387` — update NewProduct() to use req.Status
- Modify: `internal/service/product_service_test.go:40-95` — update test to verify status passthrough

**Step 1: Add Status field to CreateProductRequest**

In `internal/domain/service.go`, add this field at the end of `CreateProductRequest` (after line 228, before the closing brace):

```go
Status            *ProductStatus         `json:"status,omitempty" validate:"omitempty,oneof=ACTIVE INACTIVE DRAFT"`
```

**Step 2: Update NewProduct() to use the status**

In `internal/domain/entity.go`, replace line 383:

```go
// Old:
Status:                ProductStatusDraft,

// New:
Status:                productStatusOrDefault(req.Status),
```

Add this helper function after `NewProduct()` (after line 387):

```go
// productStatusOrDefault returns the given status or DRAFT if nil.
func productStatusOrDefault(s *ProductStatus) ProductStatus {
	if s != nil {
		return *s
	}
	return ProductStatusDraft
}
```

**Step 3: Add test for status passthrough**

In `internal/service/product_service_test.go`, add a new subtest inside `TestProductService_Create` (after the "successful creation" subtest around line 95):

```go
t.Run("creation with explicit status", func(t *testing.T) {
    svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

    category := &domain.Category{
        ID:   "cat_123",
        Name: "Silk Sarees",
    }

    activeStatus := domain.ProductStatusActive
    req := domain.CreateProductRequest{
        Name:         "Active Product",
        SKU:          "ACT-001",
        CategoryID:   "cat_123",
        BasePrice:    100000,
        SellingPrice: 90000,
        Status:       &activeStatus,
    }

    mockCatRepo.EXPECT().
        GetByID(ctx, "cat_123").
        Return(category, nil)

    mockProdRepo.EXPECT().
        CreateWithAttributeIndexes(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
        DoAndReturn(func(ctx context.Context, product *domain.Product, attrs map[string][]string, inv *domain.Inventory) error {
            assert.Equal(t, domain.ProductStatusActive, product.Status)
            return nil
        })

    mockProdRepo.EXPECT().
        AddAttributeValues(ctx, "cat_123", gomock.Any()).
        Return(nil)

    mockCatRepo.EXPECT().
        IncrementProductCount(ctx, "cat_123", 1).
        Return(nil)

    product, err := svc.Create(ctx, req, "admin_1")

    require.NoError(t, err)
    assert.Equal(t, domain.ProductStatusActive, product.Status)
})
```

**Step 4: Run tests**

```bash
go test -v -run TestProductService_Create ./internal/service/...
```

Expected: All 3 subtests pass. The existing "successful creation" test still asserts `ProductStatusDraft` (because no status is set in the request), and the new test asserts `ProductStatusActive`.

**Step 5: Commit**

```bash
git add internal/domain/service.go internal/domain/entity.go internal/service/product_service_test.go
git commit -m "fix: accept status field on product creation

CreateProductRequest now has an optional Status field. NewProduct()
uses it if provided, defaults to DRAFT. This fixes the bug where the
admin form's status dropdown was silently ignored on creation."
```

---

## Task 2: Add SortOrder field to Product entity

**Files:**
- Modify: `internal/domain/entity.go:275-352` — add SortOrder field, update SetKeys()

**Step 1: Add SortOrder field to Product struct**

In `internal/domain/entity.go`, add this field after line 333 (after the `Status` field, before `BaseEntity`):

```go
SortOrder int `json:"sort_order" dynamodbav:"sort_order"`
```

**Step 2: Update SetKeys() to encode sort_order in GSI1SK**

Replace the `SetKeys()` method (lines 344-352) with:

```go
// SetKeys sets the DynamoDB keys for Product
func (p *Product) SetKeys() {
	p.PK = "PRODUCT#" + p.ID
	p.SK = "METADATA"
	p.GSI1PK = "CATEGORY#" + p.CategoryID
	p.GSI1SK = fmt.Sprintf("RANK#%06d#%s", p.SortOrder, p.ID)
	p.GSI2PK = "PRODUCT#ALL"
	p.GSI2SK = "PRODUCT#" + p.ID
	p.EntityType = "PRODUCT"
}
```

Add `"fmt"` to the import list at the top of `entity.go` if not already present.

**Step 3: Update the repository's listByCategory GSI1 query**

In `internal/repository/dynamodb/product_repository.go`, find `listByCategory` (line 121-133). Change the key expression from `BeginsWith("PRODUCT#")` to `BeginsWith("RANK#")`:

```go
func (r *ProductRepository) listByCategory(ctx context.Context, categoryID string, limit int, cursor string, req domain.ListProductsRequest) (*domain.ListProductsResponse, error) {
	keyExpr := expression.Key("GSI1PK").Equal(expression.Value("CATEGORY#" + categoryID)).
		And(expression.Key("GSI1SK").BeginsWith("RANK#"))
```

**Step 4: Run full test suite to verify no regressions**

```bash
go test -v -race ./internal/...
```

Expected: All existing tests pass. Some tests that assert `GSI1SK = "PRODUCT#..."` will need updating — fix those in the test expectations to use the new `RANK#000000#...` format.

**Step 5: Commit**

```bash
git add internal/domain/entity.go internal/repository/dynamodb/product_repository.go
git commit -m "feat: add SortOrder field to Product, encode rank in GSI1SK

Products now have a SortOrder field (default 0). GSI1SK is encoded as
RANK#<zero-padded-sort-order>#<id> so DynamoDB returns products
pre-sorted by rank when querying by category."
```

---

## Task 3: Add ReorderProducts backend service + repository

**Files:**
- Modify: `internal/domain/service.go` — add ReorderProductsRequest, update ProductService interface
- Modify: `internal/domain/repository.go` — add BatchUpdateSortOrder to ProductRepository interface
- Modify: `internal/service/product_service.go` — add ReorderProducts method
- Modify: `internal/repository/dynamodb/product_repository.go` — add BatchUpdateSortOrder method

**Step 1: Add ReorderProductsRequest to domain/service.go**

After `UpdateProductRequest` (after line 253), add:

```go
// ReorderProductsRequest contains data for reordering products within a category
type ReorderProductsRequest struct {
	ProductIDs []string `json:"product_ids" validate:"required,min=1"`
}
```

Add `ReorderProducts` to the `ProductService` interface (around line 199):

```go
// ReorderProducts sets the sort order for products in a category
ReorderProducts(ctx context.Context, categoryID string, productIDs []string) (int, error)
```

**Step 2: Add BatchUpdateSortOrder to repository interface**

In `internal/domain/repository.go`, add to the `ProductRepository` interface (around line 130):

```go
// BatchUpdateSortOrder updates sort_order and GSI1SK for multiple products in a transaction
BatchUpdateSortOrder(ctx context.Context, products []*Product) error

// GetByCategoryAll retrieves all product IDs in a category (for reorder validation)
GetByCategoryAll(ctx context.Context, categoryID string) ([]*Product, error)
```

**Step 3: Implement BatchUpdateSortOrder in the DynamoDB repository**

In `internal/repository/dynamodb/product_repository.go`, add:

```go
// BatchUpdateSortOrder updates sort_order and GSI1SK for multiple products.
// DynamoDB transactions are limited to 100 items, so this batches into
// multiple transactions if needed.
func (r *ProductRepository) BatchUpdateSortOrder(ctx context.Context, products []*domain.Product) error {
	const maxBatchSize = 100

	for i := 0; i < len(products); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(products) {
			end = len(products)
		}

		batch := products[i:end]
		var transactItems []types.TransactWriteItem

		for _, p := range batch {
			p.SetKeys()
			p.UpdatedAt = time.Now()

			transactItems = append(transactItems, types.TransactWriteItem{
				Update: &types.Update{
					TableName: aws.String(r.client.catalogTable),
					Key: map[string]types.AttributeValue{
						"PK": &types.AttributeValueMemberS{Value: p.PK},
						"SK": &types.AttributeValueMemberS{Value: p.SK},
					},
					UpdateExpression: aws.String("SET sort_order = :so, GSI1SK = :gsi1sk, updated_at = :ua"),
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":so":     &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", p.SortOrder)},
						":gsi1sk": &types.AttributeValueMemberS{Value: p.GSI1SK},
						":ua":     &types.AttributeValueMemberS{Value: p.UpdatedAt.Format(time.RFC3339Nano)},
					},
				},
			})
		}

		_, err := r.client.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
			TransactItems: transactItems,
		})
		if err != nil {
			return errors.Internal(err)
		}
	}

	return nil
}

// GetByCategoryAll retrieves all products in a category (unpaginated, for reordering).
func (r *ProductRepository) GetByCategoryAll(ctx context.Context, categoryID string) ([]*domain.Product, error) {
	keyExpr := expression.Key("GSI1PK").Equal(expression.Value("CATEGORY#" + categoryID)).
		And(expression.Key("GSI1SK").BeginsWith("RANK#"))

	expr, err := expression.NewBuilder().WithKeyCondition(keyExpr).Build()
	if err != nil {
		return nil, errors.Internal(err)
	}

	var allProducts []*domain.Product
	var lastKey map[string]types.AttributeValue

	for {
		input := &dynamodb.QueryInput{
			TableName:                 aws.String(r.client.catalogTable),
			IndexName:                 aws.String("GSI1"),
			KeyConditionExpression:    expr.KeyCondition(),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			ExclusiveStartKey:         lastKey,
		}

		result, err := r.client.db.Query(ctx, input)
		if err != nil {
			return nil, errors.Internal(err)
		}

		var batch []*domain.Product
		if err := attributevalue.UnmarshalListOfMaps(result.Items, &batch); err != nil {
			return nil, errors.Internal(err)
		}
		allProducts = append(allProducts, batch...)

		if result.LastEvaluatedKey == nil {
			break
		}
		lastKey = result.LastEvaluatedKey
	}

	return allProducts, nil
}
```

**Step 4: Implement ReorderProducts service method**

In `internal/service/product_service.go`, add:

```go
// ReorderProducts sets the display order for products in a category.
// productIDs is the desired order — position 0 gets SortOrder=1, etc.
func (s *ProductService) ReorderProducts(ctx context.Context, categoryID string, productIDs []string) (int, error) {
	// Validate category exists
	if _, err := s.categoryRepo.GetByID(ctx, categoryID); err != nil {
		return 0, errors.New(errors.ErrCodeNotFound, "Category not found")
	}

	// Fetch all products in the category to validate ownership
	existing, err := s.productRepo.GetByCategoryAll(ctx, categoryID)
	if err != nil {
		return 0, err
	}

	existingMap := make(map[string]*domain.Product, len(existing))
	for _, p := range existing {
		existingMap[p.ID] = p
	}

	// Validate all requested IDs belong to this category
	for _, id := range productIDs {
		if _, ok := existingMap[id]; !ok {
			return 0, errors.New(errors.ErrCodeValidation, "Product "+id+" does not belong to category "+categoryID)
		}
	}

	// Build updates: requested IDs get sequential sort_order starting at 1
	var toUpdate []*domain.Product
	assigned := make(map[string]bool)

	for i, id := range productIDs {
		p := existingMap[id]
		p.SortOrder = i + 1
		toUpdate = append(toUpdate, p)
		assigned[id] = true
	}

	// Any products not in the request list get sort_order after the ranked ones
	nextOrder := len(productIDs) + 1
	for _, p := range existing {
		if !assigned[p.ID] {
			p.SortOrder = nextOrder
			toUpdate = append(toUpdate, p)
			nextOrder++
		}
	}

	if err := s.productRepo.BatchUpdateSortOrder(ctx, toUpdate); err != nil {
		return 0, err
	}

	s.logger.WithContext(ctx).Infof("Reordered %d products in category %s", len(toUpdate), categoryID)
	return len(toUpdate), nil
}
```

**Step 5: Run tests**

```bash
go test -v -race ./internal/service/...
```

**Step 6: Commit**

```bash
git add internal/domain/service.go internal/domain/repository.go internal/service/product_service.go internal/repository/dynamodb/product_repository.go
git commit -m "feat: add ReorderProducts service and BatchUpdateSortOrder repository

Admins can now reorder products within a category. The service validates
all product IDs belong to the category, assigns sequential sort_order
values, and batch-updates via DynamoDB transactions."
```

---

## Task 4: Add ReorderProducts test

**Files:**
- Modify: `internal/service/product_service_test.go` — add TestProductService_ReorderProducts
- Modify: `internal/mocks/` — regenerate mocks if needed

**Step 1: Regenerate mocks (if interface changed)**

```bash
make generate-mocks
```

**Step 2: Add test**

In `internal/service/product_service_test.go`, add:

```go
func TestProductService_ReorderProducts(t *testing.T) {
	t.Run("successful reorder", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{ID: "cat_123", Name: "Test"}

		existingProducts := []*domain.Product{
			{ID: "prod_a", CategoryID: "cat_123", SortOrder: 0},
			{ID: "prod_b", CategoryID: "cat_123", SortOrder: 0},
			{ID: "prod_c", CategoryID: "cat_123", SortOrder: 0},
		}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().GetByCategoryAll(ctx, "cat_123").Return(existingProducts, nil)
		mockProdRepo.EXPECT().
			BatchUpdateSortOrder(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, products []*domain.Product) error {
				assert.Len(t, products, 3)
				// Requested order: c, a, b
				for _, p := range products {
					switch p.ID {
					case "prod_c":
						assert.Equal(t, 1, p.SortOrder)
					case "prod_a":
						assert.Equal(t, 2, p.SortOrder)
					case "prod_b":
						assert.Equal(t, 3, p.SortOrder)
					}
				}
				return nil
			})

		count, err := svc.ReorderProducts(ctx, "cat_123", []string{"prod_c", "prod_a", "prod_b"})
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("product not in category", func(t *testing.T) {
		svc, mockProdRepo, mockCatRepo, _, _, ctx := setupProductTest(t)

		category := &domain.Category{ID: "cat_123", Name: "Test"}
		existingProducts := []*domain.Product{
			{ID: "prod_a", CategoryID: "cat_123"},
		}

		mockCatRepo.EXPECT().GetByID(ctx, "cat_123").Return(category, nil)
		mockProdRepo.EXPECT().GetByCategoryAll(ctx, "cat_123").Return(existingProducts, nil)

		_, err := svc.ReorderProducts(ctx, "cat_123", []string{"prod_a", "prod_unknown"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not belong to category")
	})

	t.Run("category not found", func(t *testing.T) {
		svc, _, mockCatRepo, _, _, ctx := setupProductTest(t)

		mockCatRepo.EXPECT().GetByID(ctx, "cat_999").Return(nil, errors.NotFound("Category"))

		_, err := svc.ReorderProducts(ctx, "cat_999", []string{"prod_a"})
		require.Error(t, err)
	})
}
```

**Step 3: Run tests**

```bash
go test -v -run TestProductService_ReorderProducts ./internal/service/...
```

Expected: All 3 subtests pass.

**Step 4: Commit**

```bash
git add internal/service/product_service_test.go internal/mocks/
git commit -m "test: add ReorderProducts service tests"
```

---

## Task 5: Add reorder HTTP handler and route

**Files:**
- Modify: `internal/handler/product_handler.go:39-58` — add Reorder handler + route

**Step 1: Add route**

In `internal/handler/product_handler.go`, inside the `Routes()` method, add after the Delete route (line 48, before the inventory routes):

```go
r.With(middleware.ValidateJSONTyped[domain.ReorderProductsRequest](h.validation)).Put("/categories/{categoryId}/reorder", h.Reorder)
```

**Step 2: Add handler method**

Add this method to `ProductHandler`:

```go
// Reorder handles PUT /admin/products/categories/{categoryId}/reorder
func (h *ProductHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	categoryID := chi.URLParam(r, "categoryId")
	if categoryID == "" {
		response.BadRequest(w, "Category ID is required")
		return
	}

	req := middleware.MustGetValidatedBody[domain.ReorderProductsRequest](ctx)

	count, err := h.productService.ReorderProducts(ctx, categoryID, req.ProductIDs)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]int{"reordered": count})
}
```

**Step 3: Run build to verify compilation**

```bash
go build ./cmd/api/...
```

**Step 4: Commit**

```bash
git add internal/handler/product_handler.go
git commit -m "feat: add PUT /admin/products/categories/{categoryId}/reorder endpoint"
```

---

## Task 6: Wire regeneration and full test

**Files:**
- Modify: `internal/wire/wire.go` (if needed)
- Modify: `internal/wire/wire_gen.go` (regenerated)

**Step 1: Regenerate mocks (interface changed)**

```bash
make generate-mocks
```

**Step 2: Regenerate wire**

```bash
make wire
```

**Step 3: Run full test suite**

```bash
go test -v -race -cover ./...
```

**Step 4: Build all binaries**

```bash
make build && make build-lambdas
```

**Step 5: Commit**

```bash
git add -f internal/wire/wire_gen.go internal/mocks/
git commit -m "chore: regenerate wire and mocks for reorder feature"
```

---

## Task 7: Frontend — add @dnd-kit dependency + reorder API

**Files:**
- Modify: `handloom-admin-frontend/package.json` — add @dnd-kit deps
- Modify: `handloom-admin-frontend/src/features/products/api.ts` — add reorder call
- Modify: `handloom-admin-frontend/src/features/products/types.ts` — add sort_order + reorder type

**Step 1: Install @dnd-kit**

```bash
cd ../handloom-admin-frontend && npm install @dnd-kit/core @dnd-kit/sortable @dnd-kit/utilities
```

**Step 2: Update types.ts**

In `handloom-admin-frontend/src/features/products/types.ts`, add `sort_order` to the `Product` interface (after `low_stock_threshold`):

```typescript
sort_order: number;
```

Add at the bottom of the file:

```typescript
export interface ReorderProductsRequest {
  product_ids: string[];
}
```

**Step 3: Add reorder API call**

In `handloom-admin-frontend/src/features/products/api.ts`, add to the `productsApi` object:

```typescript
reorder: async (categoryId: string, productIds: string[]) => {
  const response = await apiClient.put(`/admin/products/categories/${categoryId}/reorder`, {
    product_ids: productIds,
  });
  return response.data;
},
```

**Step 4: Commit**

```bash
git add package.json package-lock.json src/features/products/api.ts src/features/products/types.ts
git commit -m "feat(frontend): add @dnd-kit deps and reorder API"
```

---

## Task 8: Frontend — build drag-and-drop ranking modal

**Files:**
- Create: `handloom-admin-frontend/src/features/products/components/ProductRankingModal.tsx`

**Step 1: Create the ranking modal component**

Create `handloom-admin-frontend/src/features/products/components/ProductRankingModal.tsx`:

```tsx
import {
  closestCenter,
  DndContext,
  type DragEndEvent,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { GripVertical } from 'lucide-react';
import { useEffect, useState } from 'react';
import toast from 'react-hot-toast';

import { getErrorMessage } from '@/shared/api/client';
import { Badge, Button, Modal } from '@/shared/components/ui';
import { getStatusBadgeVariant } from '@/shared/utils/badge';
import { formatCurrency } from '@/shared/utils/currency';

import { productsApi } from '../api';
import type { Product } from '../types';

interface ProductRankingModalProps {
  isOpen: boolean;
  onClose: () => void;
  categoryId: string;
  categoryName: string;
}

function SortableProduct({ product }: { product: Product }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: product.id,
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`flex items-center gap-3 rounded-lg border bg-white p-3 ${isDragging ? 'shadow-lg' : ''}`}
    >
      <button
        type="button"
        className="cursor-grab touch-none text-gray-400 hover:text-gray-600"
        {...attributes}
        {...listeners}
      >
        <GripVertical className="h-5 w-5" />
      </button>

      {product.images?.[0] && (
        <img
          src={product.images[0].url}
          alt={product.name}
          className="h-10 w-10 rounded object-cover"
        />
      )}

      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-gray-900">{product.name}</p>
        <p className="text-xs text-gray-500">{product.sku}</p>
      </div>

      <div className="text-right text-sm text-gray-600">{formatCurrency(product.selling_price)}</div>

      <Badge variant={getStatusBadgeVariant(product.status)}>{product.status}</Badge>
    </div>
  );
}

export function ProductRankingModal({
  isOpen,
  onClose,
  categoryId,
  categoryName,
}: ProductRankingModalProps) {
  const queryClient = useQueryClient();
  const [orderedProducts, setOrderedProducts] = useState<Product[]>([]);

  const { data: productsData, isLoading } = useQuery({
    queryKey: ['products-for-ranking', categoryId],
    queryFn: () => productsApi.list({ category_id: categoryId, limit: 100 }),
    enabled: isOpen && !!categoryId,
  });

  useEffect(() => {
    if (productsData?.items) {
      // Sort by existing sort_order, then by name as fallback
      const sorted = [...productsData.items].sort((a, b) => {
        if (a.sort_order !== b.sort_order) return a.sort_order - b.sort_order;
        return a.name.localeCompare(b.name);
      });
      setOrderedProducts(sorted);
    }
  }, [productsData]);

  const reorderMutation = useMutation({
    mutationFn: () => productsApi.reorder(categoryId, orderedProducts.map((p) => p.id)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['products'] });
      queryClient.invalidateQueries({ queryKey: ['products-for-ranking'] });
      toast.success('Product order saved');
      onClose();
    },
    onError: (error) => {
      toast.error(getErrorMessage(error));
    },
  });

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  );

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    setOrderedProducts((items) => {
      const oldIndex = items.findIndex((i) => i.id === active.id);
      const newIndex = items.findIndex((i) => i.id === over.id);
      return arrayMove(items, oldIndex, newIndex);
    });
  }

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`Manage Order — ${categoryName}`} size="lg">
      <div className="space-y-4">
        <p className="text-sm text-gray-500">
          Drag products to set their display order. The order here determines how products appear on
          the storefront.
        </p>

        {isLoading ? (
          <div className="py-8 text-center text-gray-500">Loading products...</div>
        ) : orderedProducts.length === 0 ? (
          <div className="py-8 text-center text-gray-500">No products in this category.</div>
        ) : (
          <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
            <SortableContext
              items={orderedProducts.map((p) => p.id)}
              strategy={verticalListSortingStrategy}
            >
              <div className="max-h-[60vh] space-y-2 overflow-y-auto">
                {orderedProducts.map((product, index) => (
                  <div key={product.id} className="flex items-center gap-2">
                    <span className="w-6 text-right text-xs text-gray-400">{index + 1}</span>
                    <div className="flex-1">
                      <SortableProduct product={product} />
                    </div>
                  </div>
                ))}
              </div>
            </SortableContext>
          </DndContext>
        )}

        <div className="flex justify-end gap-3 border-t pt-4">
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button
            onClick={() => reorderMutation.mutate()}
            loading={reorderMutation.isPending}
            disabled={orderedProducts.length === 0}
          >
            Save Order
          </Button>
        </div>
      </div>
    </Modal>
  );
}
```

**Step 2: Commit**

```bash
git add src/features/products/components/ProductRankingModal.tsx
git commit -m "feat(frontend): add drag-and-drop product ranking modal"
```

---

## Task 9: Frontend — wire ranking modal into products page

**Files:**
- Modify: `handloom-admin-frontend/src/features/products/components/ProductsPage.tsx`

**Step 1: Import the modal and add state**

Add import at the top of `ProductsPage.tsx`:

```typescript
import { ArrowUpDown } from 'lucide-react';
```

Add to existing lucide-react import line. Then import the modal:

```typescript
import { ProductRankingModal } from './ProductRankingModal';
```

**Step 2: Add state variable**

Inside the `ProductsPage` component, add:

```typescript
const [showRankingModal, setShowRankingModal] = useState(false);
```

**Step 3: Add "Manage Order" button**

Find the toolbar area (near the "Add Product" button). Add this button next to it, shown only when a category filter is active:

```tsx
{categoryFilter && (
  <Button
    variant="secondary"
    icon={<ArrowUpDown className="h-4 w-4" />}
    onClick={() => setShowRankingModal(true)}
  >
    Manage Order
  </Button>
)}
```

**Step 4: Add modal at the bottom of the component return**

Before the closing fragment or parent div, add:

```tsx
{categoryFilter && (
  <ProductRankingModal
    isOpen={showRankingModal}
    onClose={() => setShowRankingModal(false)}
    categoryId={categoryFilter}
    categoryName={categories.find((c) => c.id === categoryFilter)?.name || ''}
  />
)}
```

**Step 5: Verify it builds**

```bash
npm run typecheck
```

**Step 6: Commit**

```bash
git add src/features/products/components/ProductsPage.tsx
git commit -m "feat(frontend): wire ranking modal into products page

Shows 'Manage Order' button when a category filter is selected.
Opens the drag-and-drop ranking modal for the selected category."
```

---

## Task 10: Fix existing tests and verify everything works

**Files:**
- Modify: any test files that break due to GSI1SK format change

**Step 1: Run full backend tests**

```bash
cd ../handloom-admin && go test -v -race -cover ./...
```

Fix any test assertions that check for `GSI1SK = "PRODUCT#..."` — update them to expect `RANK#000000#...`.

**Step 2: Run frontend checks**

```bash
cd ../handloom-admin-frontend && npm run check
```

Fix any lint/type errors.

**Step 3: Run frontend build**

```bash
npm run build
```

**Step 4: Final commit**

```bash
git add -A
git commit -m "fix: update test assertions for new GSI1SK format"
```
