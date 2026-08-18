# Inventory Lifecycle Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `quantity` decrement when goods are dispatched, so successful sales stop leaking their reservations and `available_qty` stops ratcheting permanently downward.

**Architecture:** The `inventory` table tracks `quantity` (physical stock), `reserved_qty` (committed to undispatched orders), and `available_qty` (a stored column always rewritten as `quantity - reserved_qty`). Reserve and Release exist and are correct. The missing third operation is **Commit** — dispatch — which decrements `quantity` and `reserved_qty` by the same amount, leaving `available_qty` unchanged. This plan adds `CommitStock`, hooks it at `SHIPPED`, restocks at `RETURNED`, and fixes three adjacent defects found alongside it.

**Tech Stack:** Go 1.25, pgx/v5, PostgreSQL, go.uber.org/mock (mockgen), testify.

## Global Constraints

- Prices are in paise; quantities are plain ints. No conversion in this work.
- Every inventory mutation happens inside a single `pgx.BeginFunc` transaction, with `SELECT ... FOR UPDATE` on the `inventory` row before any write.
- Every inventory mutation writes exactly one `inventory_transactions` ledger row via `insertInventoryTransaction`, inside that same transaction.
- `available_qty` is a stored column, never generated. Every writer must recompute and set it explicitly.
- Services return `*errors.AppError`; use `errors.New(errors.ErrCodeInsufficientStock, ...)`, `errors.NotFound(...)`, `errors.Wrap(err, ...)` from `github.com/handloom/admin/pkg/errors`.
- Run `make generate-mocks` after any change to an interface in `internal/domain/`.
- Repository tests live in package `postgres_test` and call `newTestPool(t)`, which skips when PostgreSQL is unreachable.
- Metric names must be low cardinality. Never use a product ID or order ID as a metric label.

## Reference: the corrected model

| Operation | `quantity` | `reserved_qty` | `available_qty` | Trigger |
|---|---|---|---|---|
| Reserve | — | +q | −q | checkout initiate |
| Release | — | −q | +q | cancel, payment failure |
| **Commit** | **−q** | **−q** | — | **`SHIPPED` (new)** |
| Restock | +q | — | +q | `RETURNED` (new) |

`CANCELLED` is reachable only from `PENDING`/`CONFIRMED`/`PROCESSING` (`internal/service/order_service.go:492-497`), so Release is always pre-dispatch and can never double-decrement a committed reservation.

---

## File Structure

| File | Responsibility | Change |
|---|---|---|
| `internal/domain/entity.go` | Ledger type constants | Add `InventoryTransactionTypeCommit` |
| `internal/domain/repository.go` | `InventoryRepository` interface | Add `CommitStock` |
| `internal/mocks/repository_mock.go` | Generated mock | Regenerate |
| `internal/repository/postgres/inventory_repository.go` | Inventory persistence | Add `CommitStock` |
| `internal/repository/postgres/inventory_repository_test.go` | Repository tests | **Create** |
| `internal/service/order_service.go` | Order status machine | Commit at SHIPPED, restock at RETURNED, cancel accepts PROCESSING, failure metrics |
| `internal/service/order_service_test.go` | Service tests | Add cases |
| `internal/service/checkout_service.go` | Checkout flow | Ledger reference ID |
| `internal/service/payment_service.go` | Payment webhooks | Failure metric |
| `docs/superpowers/runbooks/inventory-reserved-qty-reset.md` | Dev reset procedure | **Create** |

---

## Task 1: Add `CommitStock` to the inventory repository

**Files:**
- Modify: `internal/domain/entity.go:127-131`
- Modify: `internal/domain/repository.go:250-272`
- Modify: `internal/repository/postgres/inventory_repository.go`
- Create: `internal/repository/postgres/inventory_repository_test.go`
- Regenerate: `internal/mocks/repository_mock.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `CommitStock(ctx context.Context, productID string, quantity int, orderID string) (*domain.InventoryTransaction, error)` on `domain.InventoryRepository`, and the constant `domain.InventoryTransactionTypeCommit` with value `"COMMIT"`. Tasks 2 and 6 depend on both.

- [ ] **Step 1: Write the failing test**

Create `internal/repository/postgres/inventory_repository_test.go`. The helpers `newTestPool`, `seedCategory` and `newTestProduct` already exist in this package in `product_repository_test.go` — do not redefine them.

```go
package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/repository/postgres"
)

// seedInventory inserts an inventory row directly, bypassing product-create
// semantics so the starting numbers are explicit.
func seedInventory(t *testing.T, pool *pgxpool.Pool, productID string, quantity, reserved int) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO inventory (id, product_id, quantity, reserved_qty, available_qty)
		 VALUES ($1, $2, $3, $4, $5)`,
		uuid.New().String(), productID, quantity, reserved, quantity-reserved,
	)
	require.NoError(t, err)
}

// readInventory returns the current quantity, reserved_qty and available_qty.
func readInventory(t *testing.T, pool *pgxpool.Pool, productID string) (qty, reserved, available int) {
	t.Helper()
	err := pool.QueryRow(context.Background(),
		`SELECT quantity, reserved_qty, available_qty FROM inventory WHERE product_id = $1`,
		productID,
	).Scan(&qty, &reserved, &available)
	require.NoError(t, err)
	return
}

func TestInventoryRepository_CommitStock(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewInventoryRepository(pool)
	ctx := context.Background()
	category := seedCategory(t, pool)

	newProduct := func(t *testing.T, quantity, reserved int) string {
		t.Helper()
		p := newTestProduct(category.ID)
		require.NoError(t, postgres.NewProductRepository(pool).Create(ctx, p, nil))
		t.Cleanup(func() {
			_, _ = pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, p.ID)
		})
		_, err := pool.Exec(ctx, `DELETE FROM inventory WHERE product_id = $1`, p.ID)
		require.NoError(t, err)
		seedInventory(t, pool, p.ID, quantity, reserved)
		return p.ID
	}

	t.Run("commit decrements quantity and reserved, leaving available unchanged", func(t *testing.T) {
		productID := newProduct(t, 100, 10)

		txn, err := repo.CommitStock(ctx, productID, 4, "order_abc")
		require.NoError(t, err)

		qty, reserved, available := readInventory(t, pool, productID)
		require.Equal(t, 96, qty)
		require.Equal(t, 6, reserved)
		require.Equal(t, 90, available, "available_qty must not move on commit")

		require.Equal(t, domain.InventoryTransactionTypeCommit, txn.Type)
		require.Equal(t, 4, txn.Quantity)
		require.Equal(t, 100, txn.PreviousQty)
		require.Equal(t, 96, txn.NewQty)
		require.Equal(t, "order_abc", txn.ReferenceID)
	})

	t.Run("commit writes exactly one ledger row", func(t *testing.T) {
		productID := newProduct(t, 50, 5)

		_, err := repo.CommitStock(ctx, productID, 2, "order_def")
		require.NoError(t, err)

		var count int
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM inventory_transactions WHERE product_id = $1 AND type = $2`,
			productID, string(domain.InventoryTransactionTypeCommit),
		).Scan(&count))
		require.Equal(t, 1, count)
	})

	t.Run("commit beyond reserved is rejected and changes nothing", func(t *testing.T) {
		productID := newProduct(t, 100, 3)

		_, err := repo.CommitStock(ctx, productID, 5, "order_ghi")
		require.Error(t, err)
		require.ErrorContains(t, err, "insufficient stock")

		qty, reserved, available := readInventory(t, pool, productID)
		require.Equal(t, 100, qty)
		require.Equal(t, 3, reserved)
		require.Equal(t, 97, available)
	})

	t.Run("commit on a missing product returns not found", func(t *testing.T) {
		_, err := repo.CommitStock(ctx, "does-not-exist", 1, "order_jkl")
		require.Error(t, err)
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd handloom-admin && go test ./internal/repository/postgres/ -run TestInventoryRepository_CommitStock -v
```

Expected: compile failure — `repo.CommitStock undefined` and `domain.InventoryTransactionTypeCommit undefined`.

If instead every subtest reports `SKIP: postgres not available`, start the database first with `make setup-local`. A skip is not a passing test.

- [ ] **Step 3: Add the ledger type constant**

In `internal/domain/entity.go`, add to the existing const block at lines 127-131:

```go
	InventoryTransactionTypeCommit  InventoryTransactionType = "COMMIT"
```

- [ ] **Step 4: Add the method to the interface**

In `internal/domain/repository.go`, inside the `InventoryRepository` interface, immediately after the `ReleaseStock` declaration:

```go
	// CommitStock converts a reservation into a dispatch. The goods have left
	// the warehouse, so quantity and reserved_qty both drop by the same amount
	// and available_qty is unchanged.
	CommitStock(ctx context.Context, productID string, quantity int, orderID string) (*InventoryTransaction, error)
```

- [ ] **Step 5: Regenerate the mocks**

```bash
cd handloom-admin && make generate-mocks
```

Expected: `internal/mocks/repository_mock.go` gains a `CommitStock` method on `MockInventoryRepository`.

- [ ] **Step 6: Implement `CommitStock`**

In `internal/repository/postgres/inventory_repository.go`, immediately after `ReleaseStock`:

```go
// CommitStock converts a reservation into a dispatch within a transaction.
// Both quantity and reserved_qty drop by the same amount, so available_qty is
// unchanged: the units were already unavailable when reserved, and are now
// physically gone as well.
func (r *InventoryRepository) CommitStock(ctx context.Context, productID string, quantity int, orderID string) (*domain.InventoryTransaction, error) {
	var txn *domain.InventoryTransaction

	err := pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		var currentQty, reservedQty int
		err := tx.QueryRow(ctx,
			`SELECT quantity, reserved_qty FROM inventory WHERE product_id = $1 FOR UPDATE`,
			productID,
		).Scan(&currentQty, &reservedQty)
		if err != nil {
			if err == pgx.ErrNoRows {
				return errors.NotFound("Inventory not found")
			}
			return errors.Wrap(err, "failed to lock inventory row")
		}

		// Both guards matter. reservedQty is the real invariant, but a row
		// corrupted by the historical leak can violate reservedQty <= quantity,
		// and driving quantity negative would be worse than refusing.
		if reservedQty < quantity || currentQty < quantity {
			return errors.New(errors.ErrCodeInsufficientStock, "insufficient stock")
		}

		now := time.Now()
		newQty := currentQty - quantity
		newReserved := reservedQty - quantity
		newAvailable := newQty - newReserved

		updQB := querybuilder.Update("inventory").
			Set(ColQuantity, newQty).
			Set(ColReservedQty, newReserved).
			Set(ColAvailableQty, newAvailable).
			Set(ColUpdatedAt, now).
			Where(ColProductID, productID)

		updSQL, updArgs := updQB.Build()
		_, err = tx.Exec(ctx, updSQL, updArgs...)
		if err != nil {
			return errors.Wrap(err, "failed to update inventory")
		}

		txn = &domain.InventoryTransaction{
			ID:            "inv_txn_" + uuid.New().String()[:8],
			ProductID:     productID,
			Type:          domain.InventoryTransactionTypeCommit,
			Quantity:      quantity,
			PreviousQty:   currentQty,
			NewQty:        newQty,
			Reason:        fmt.Sprintf("ORDER %s", orderID),
			ReferenceType: inventoryRefTypeOrder,
			ReferenceID:   orderID,
			CreatedAt:     now,
		}

		return insertInventoryTransaction(ctx, tx, txn)
	})
	if err != nil {
		return nil, err
	}
	return txn, nil
}
```

Note `PreviousQty`/`NewQty` track `quantity` here, matching `AddStock` and `AdjustStock`. `ReserveStock` and `ReleaseStock` track `reserved_qty` in those fields instead — that inconsistency is pre-existing and out of scope.

- [ ] **Step 7: Run the test to verify it passes**

```bash
cd handloom-admin && go test ./internal/repository/postgres/ -run TestInventoryRepository_CommitStock -v
```

Expected: all four subtests PASS.

- [ ] **Step 8: Verify the whole package still builds and lints**

```bash
cd handloom-admin && go build ./... && golangci-lint run
```

Expected: no errors.

- [ ] **Step 9: Commit**

```bash
git add internal/domain/entity.go internal/domain/repository.go \
        internal/mocks/repository_mock.go \
        internal/repository/postgres/inventory_repository.go \
        internal/repository/postgres/inventory_repository_test.go
git commit -m "feat(inventory): add CommitStock for dispatch

Converts a reservation into a dispatch: quantity and reserved_qty both drop
by the same amount, so available_qty is unchanged. This is the operation
that was missing from the lifecycle."
```

---

## Task 2: Commit stock when an order ships

**Files:**
- Modify: `internal/service/order_service.go:322-344`
- Test: `internal/service/order_service_test.go`

**Interfaces:**
- Consumes: `CommitStock` and `domain.InventoryTransactionTypeCommit` from Task 1.
- Produces: no new signatures. `OrderService.UpdateStatus` gains a `SHIPPED` inventory effect.

- [ ] **Step 1: Write the failing test**

Append to `internal/service/order_service_test.go`. Mirror the existing construction in `TestOrderService_Create` (lines 17-33) for the mocks and service.

```go
func TestOrderService_UpdateStatus_Inventory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)
	mockProductRepo := mocks.NewMockProductRepository(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	mockPriceQuoteRepo := mocks.NewMockPriceQuoteRepository(ctrl)
	mockPricingService := mocks.NewMockPricingService(ctrl)

	service := NewOrderService(
		mockOrderRepo,
		mockCustomerRepo,
		mockProductRepo,
		mockInventoryRepo,
		mockPriceQuoteRepo,
		mockPricingService,
	)
	ctx := context.Background()

	t.Run("shipping commits stock for every item", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_123",
			Status: domain.OrderStatusConfirmed,
			Items: []domain.OrderItem{
				{ProductID: "prod_123", Quantity: 2},
				{ProductID: "prod_456", Quantity: 1},
			},
		}

		mockOrderRepo.EXPECT().GetByID(gomock.Any(), "order_123").Return(order, nil)
		mockOrderRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		mockInventoryRepo.EXPECT().
			CommitStock(gomock.Any(), "prod_123", 2, "order_123").
			Return(&domain.InventoryTransaction{}, nil)
		mockInventoryRepo.EXPECT().
			CommitStock(gomock.Any(), "prod_456", 1, "order_123").
			Return(&domain.InventoryTransaction{}, nil)

		err := service.UpdateStatus(ctx, "order_123", domain.OrderStatusShipped, "admin_123")
		require.NoError(t, err)
	})

	t.Run("delivery has no inventory effect", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_789",
			Status: domain.OrderStatusShipped,
			Items:  []domain.OrderItem{{ProductID: "prod_123", Quantity: 2}},
		}

		mockOrderRepo.EXPECT().GetByID(gomock.Any(), "order_789").Return(order, nil)
		mockOrderRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		// No inventory expectation: gomock fails the test on any unexpected call.

		err := service.UpdateStatus(ctx, "order_789", domain.OrderStatusDelivered, "admin_123")
		require.NoError(t, err)
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd handloom-admin && go test ./internal/service/ -run TestOrderService_UpdateStatus_Inventory -v
```

Expected: FAIL on the first subtest — `missing call(s) to CommitStock`, because nothing commits yet.

- [ ] **Step 3: Add the SHIPPED case and clean up DELIVERED**

In `internal/service/order_service.go`, replace the `OrderStatusShipped` and `OrderStatusDelivered` cases in the `UpdateStatus` switch (lines 322-331):

```go
	case domain.OrderStatusShipped:
		now := time.Now()
		order.ShippedAt = &now
		// Goods have left the warehouse: convert each reservation into a
		// dispatch. available_qty is unaffected — these units were already
		// unavailable while reserved.
		for _, item := range order.Items {
			if _, commitErr := s.inventoryRepo.CommitStock(ctx, item.ProductID, item.Quantity, order.ID); commitErr != nil {
				slog.ErrorContext(ctx, "Failed to commit stock", keyProductID, item.ProductID, "error", commitErr)
			}
		}
	case domain.OrderStatusDelivered:
		now := time.Now()
		order.DeliveredAt = &now
		// No inventory effect: stock was committed at dispatch.
```

The swallowed error here matches the existing `CANCELLED` branch and is made observable in Task 6.

- [ ] **Step 4: Run the test to verify it passes**

```bash
cd handloom-admin && go test ./internal/service/ -run TestOrderService_UpdateStatus_Inventory -v
```

Expected: both subtests PASS.

- [ ] **Step 5: Run the full service suite for regressions**

```bash
cd handloom-admin && go test ./internal/service/ -v
```

Expected: PASS. Pre-existing tests asserting no inventory calls on SHIPPED will now fail — those assertions encoded the bug and should be updated to expect `CommitStock`.

- [ ] **Step 6: Commit**

```bash
git add internal/service/order_service.go internal/service/order_service_test.go
git commit -m "fix(inventory): commit stock when an order ships

quantity was never decremented by any order transition, so every successful
sale leaked its reservation permanently and available_qty ratcheted down
until the product became silently unbuyable.

Commits at SHIPPED rather than DELIVERED: stock leaves the warehouse at
dispatch, so committing at delivery would leave quantity overstating
physical stock for the whole transit window."
```

---

## Task 3: Restock when an order is returned

**Files:**
- Modify: `internal/service/order_service.go` (the `UpdateStatus` switch)
- Test: `internal/service/order_service_test.go`

**Interfaces:**
- Consumes: `AddStock(ctx, productID string, quantity int, reason string, userID string) (*domain.InventoryTransaction, error)` — already on `domain.InventoryRepository`.
- Produces: no new signatures.

`RETURNED` is reachable from `SHIPPED` and `DELIVERED` (`order_service.go:494-495`), both post-commit, so `quantity += q` returns the count to its pre-dispatch value. `reserved_qty` is untouched — it was already cleared at commit.

`AddStock` is used rather than a new method: it already performs `quantity += q, available_qty += q` atomically with a ledger row. The trade-off is that returns are recorded as ledger type `ADD` and are distinguishable only by reason prefix.

- [ ] **Step 1: Write the failing test**

Append inside `TestOrderService_UpdateStatus_Inventory` from Task 2:

```go
	t.Run("return restocks every item", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_ret",
			Status: domain.OrderStatusDelivered,
			Items: []domain.OrderItem{
				{ProductID: "prod_123", Quantity: 2},
			},
		}

		mockOrderRepo.EXPECT().GetByID(gomock.Any(), "order_ret").Return(order, nil)
		mockOrderRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		mockInventoryRepo.EXPECT().
			AddStock(gomock.Any(), "prod_123", 2, "RETURN order_ret", "admin_123").
			Return(&domain.InventoryTransaction{}, nil)

		err := service.UpdateStatus(ctx, "order_ret", domain.OrderStatusReturned, "admin_123")
		require.NoError(t, err)
	})
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd handloom-admin && go test ./internal/service/ -run TestOrderService_UpdateStatus_Inventory/return_restocks -v
```

Expected: FAIL — `missing call(s) to AddStock`.

- [ ] **Step 3: Add the RETURNED case**

In the same `UpdateStatus` switch, after the `OrderStatusCancelled` case:

```go
	case domain.OrderStatusReturned:
		// Goods are back. RETURNED is reachable only from SHIPPED/DELIVERED,
		// both post-commit, so quantity returns to its pre-dispatch value and
		// reserved_qty is already clear.
		for _, item := range order.Items {
			reason := fmt.Sprintf("RETURN order %s", order.ID)
			if _, addErr := s.inventoryRepo.AddStock(ctx, item.ProductID, item.Quantity, reason, updatedBy); addErr != nil {
				slog.ErrorContext(ctx, "Failed to restock returned item", keyProductID, item.ProductID, "error", addErr)
			}
		}
```

Confirm `fmt` is imported in this file; add it if not.

- [ ] **Step 4: Run the test to verify it passes**

```bash
cd handloom-admin && go test ./internal/service/ -run TestOrderService_UpdateStatus_Inventory -v
```

Expected: all three subtests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/service/order_service.go internal/service/order_service_test.go
git commit -m "feat(inventory): restock returned orders

RETURNED previously had no inventory effect, so returned goods never went
back on sale. Returns are assumed resellable and go straight back to
available stock with no inspection step."
```

---

## Task 4: Make the two cancel paths agree

**Files:**
- Modify: `internal/service/order_service.go:397-402`
- Test: `internal/service/order_service_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: `OrderService.CancelOrder` accepts `PROCESSING` in addition to `PENDING` and `CONFIRMED`.

`CancelOrder` accepts only `PENDING`/`CONFIRMED`, but `validTransitions` allows `PROCESSING → CANCELLED` (`order_service.go:494`). An admin pressing "Cancel Order" on a `PROCESSING` order gets an error, while setting its status to `CANCELLED` on the same order succeeds and releases stock. Same intent, two outcomes.

The storefront only offers Cancel for `PENDING`/`CONFIRMED`, so this does not widen what customers can cancel.

- [ ] **Step 1: Write the failing test**

Append to `internal/service/order_service_test.go`, constructing mocks and service exactly as in Task 2 Step 1:

```go
	t.Run("cancelling a processing order releases stock", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_proc",
			Status: domain.OrderStatusProcessing,
			Items:  []domain.OrderItem{{ProductID: "prod_123", Quantity: 3}},
		}

		mockOrderRepo.EXPECT().GetByID(gomock.Any(), "order_proc").Return(order, nil)
		mockOrderRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		mockInventoryRepo.EXPECT().
			ReleaseStock(gomock.Any(), "prod_123", 3, "order_proc").
			Return(&domain.InventoryTransaction{}, nil)

		err := service.CancelOrder(ctx, "order_proc", "out of stock", "admin_123")
		require.NoError(t, err)
	})

	t.Run("cancelling a shipped order is rejected", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_ship",
			Status: domain.OrderStatusShipped,
			Items:  []domain.OrderItem{{ProductID: "prod_123", Quantity: 1}},
		}

		mockOrderRepo.EXPECT().GetByID(gomock.Any(), "order_ship").Return(order, nil)

		err := service.CancelOrder(ctx, "order_ship", "too late", "admin_123")
		require.Error(t, err)
	})
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd handloom-admin && go test ./internal/service/ -run "TestOrderService.*cancelling_a_processing" -v
```

Expected: FAIL — the processing case returns "Order cannot be canceled in current status".

- [ ] **Step 3: Widen the guard**

In `internal/service/order_service.go`, replace the status guard at lines 397-402:

```go
	// Cancellable up to dispatch. Mirrors validTransitions, which allows
	// PENDING/CONFIRMED/PROCESSING -> CANCELLED; the two paths previously
	// disagreed about PROCESSING.
	if order.Status != domain.OrderStatusPending &&
		order.Status != domain.OrderStatusConfirmed &&
		order.Status != domain.OrderStatusProcessing {
		cancelErr := errors.BadRequest("Order cannot be canceled in current status")
		span.EndWithError(cancelErr)
		return cancelErr
	}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd handloom-admin && go test ./internal/service/ -v
```

Expected: PASS, including both new subtests.

- [ ] **Step 5: Commit**

```bash
git add internal/service/order_service.go internal/service/order_service_test.go
git commit -m "fix(orders): allow cancelling a PROCESSING order

CancelOrder accepted only PENDING/CONFIRMED while validTransitions allowed
PROCESSING -> CANCELLED, so the Cancel action failed on orders whose status
could still be set to CANCELLED directly."
```

---

## Task 5: Use the order ID as the ledger reference at checkout

**Files:**
- Modify: `internal/service/checkout_service.go:84-97`, `:145-150`, `:274-282`
- Test: `internal/service/checkout_service_test.go` (create if absent)

**Interfaces:**
- Consumes: nothing new.
- Produces: `releaseReservedItems(ctx context.Context, orderID string, items []domain.CartItem)` — signature changed, gains `orderID`.

Checkout reserves with the literal string `"checkout"` (`checkout_service.go:87`) while every release passes `order.ID`. `ReleaseStock` does not match on reference, so releases still work — but the ledger cannot be joined reserve-to-release by order, which is exactly the query needed to detect a leak.

The order ID is currently generated at step 9, after reservation at step 5. Hoist it.

- [ ] **Step 1: Write the failing test**

`internal/service/checkout_service_test.go` does not exist — checkout currently has **no test coverage at all**, despite being the money path. Create it.

All five mocks already exist: `MockCartService` and `MockPaymentService` in `internal/mocks/store_service_mock.go`, `MockOrderRepository` and `MockCustomerRepository` in `internal/mocks/order_repository_mock.go`, `MockInventoryRepository` in `internal/mocks/repository_mock.go`.

```go
package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
)

func TestCheckoutService_Initiate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCartService := mocks.NewMockCartService(ctrl)
	mockOrderRepo := mocks.NewMockOrderRepository(ctrl)
	mockPaymentService := mocks.NewMockPaymentService(ctrl)
	mockInventoryRepo := mocks.NewMockInventoryRepository(ctrl)
	mockCustomerRepo := mocks.NewMockCustomerRepository(ctrl)

	service := NewCheckoutService(
		mockCartService,
		mockOrderRepo,
		mockPaymentService,
		mockInventoryRepo,
		mockCustomerRepo,
	)
	ctx := context.Background()

	t.Run("reserves against the order id, not the literal checkout", func(t *testing.T) {
		customer := &domain.Customer{
			ID:        "cust_123",
			FirstName: "Test",
			LastName:  "Customer",
			Phone:     "+919999900001",
			Addresses: []domain.Address{{ID: "addr_1"}},
		}

		cart := &domain.CartWithItems{
			Cart:  &domain.Cart{Subtotal: 50000},
			Items: []domain.CartItem{{ProductID: "prod_123", Quantity: 2}},
		}

		var reservedRef, createdOrderID string

		mockCustomerRepo.EXPECT().GetByID(gomock.Any(), "cust_123").Return(customer, nil)
		mockCartService.EXPECT().GetCart(gomock.Any(), "cust_123", false).Return(cart, nil)

		mockInventoryRepo.EXPECT().
			ReserveStock(gomock.Any(), "prod_123", 2, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ int, ref string) (*domain.InventoryTransaction, error) {
				reservedRef = ref
				return &domain.InventoryTransaction{}, nil
			})

		mockOrderRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, o *domain.Order) error {
				createdOrderID = o.ID
				return nil
			})

		mockPaymentService.EXPECT().
			InitiatePayment(gomock.Any(), gomock.Any()).
			Return(&domain.PaymentResponse{
				PaymentID:     "pay_1",
				RedirectURL:   "https://sandbox.example/pay",
				MerchantTxnID: "txn_1",
			}, nil)

		result, err := service.Initiate(ctx, "cust_123", domain.CheckoutRequest{ShippingAddressID: "addr_1"})
		require.NoError(t, err)
		require.NotNil(t, result)

		require.NotEqual(t, "checkout", reservedRef, "reservation must not use the literal placeholder")
		require.Equal(t, createdOrderID, reservedRef, "reserve and release must share the order id")
	})
}
```

If a mock method signature differs from the above, run `go test` and correct against the compiler error — do not weaken the two final assertions, which are the point of the task.

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd handloom-admin && go test ./internal/service/ -run TestCheckoutService_Initiate -v
```

Expected: FAIL on `reservation must not use the literal placeholder` — `reservedRef` is `"checkout"`.

- [ ] **Step 3: Hoist the order ID above the reservation loop**

In `internal/service/checkout_service.go`, immediately before the comment `// 5. Reserve inventory for each item`:

```go
	// Generated up front so the reservation ledger rows carry the order ID and
	// can be joined against their matching release.
	orderID := uuid.New().String()
```

- [ ] **Step 4: Pass it through the three call sites**

Reservation loop (line 87):

```go
		_, reserveErr := s.inventoryRepo.ReserveStock(ctx, item.ProductID, item.Quantity, orderID)
```

Both rollback calls — inside the reservation loop and after a failed `orderRepo.Create`:

```go
			s.releaseReservedItems(ctx, orderID, reservedItems)
```

The order struct literal, replacing `ID: uuid.New().String()`:

```go
		ID:              orderID,
```

And the helper at line 275:

```go
// releaseReservedItems releases inventory for items that were previously reserved
func (s *CheckoutService) releaseReservedItems(ctx context.Context, orderID string, items []domain.CartItem) {
	for _, item := range items {
		_, err := s.inventoryRepo.ReleaseStock(ctx, item.ProductID, item.Quantity, orderID)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to release reserved stock", keyProductID, item.ProductID, "error", err)
			// Continue releasing other items even if one fails
		}
	}
}
```

- [ ] **Step 5: Run the tests to verify they pass**

```bash
cd handloom-admin && go test ./internal/service/ -v && go build ./...
```

Expected: PASS, no build errors. `grep -n '"checkout"' internal/service/checkout_service.go` should return nothing.

- [ ] **Step 6: Commit**

```bash
git add internal/service/checkout_service.go internal/service/checkout_service_test.go
git commit -m "fix(inventory): reserve against the order id at checkout

Checkout reserved with the literal string \"checkout\" while every release
passed the order ID, so reserve and release rows could not be joined and a
leaked reservation was undetectable from the ledger."
```

---

## Task 6: Make inventory failures observable

**Files:**
- Modify: `internal/service/order_service.go` (three sites: SHIPPED commit, RETURNED restock, CANCELLED release, plus `CancelOrder`)
- Modify: `internal/service/payment_service.go:345-347`
- Modify: `internal/service/checkout_service.go:278-280`
- Test: `internal/service/order_service_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: metric `inventory_mutation_failed`, labelled by `metrics.LabelReason` with one of `commit`, `release`, `restock`.

Every inventory call outside a transaction logs its error and discards it. A failed release still reports a successful cancel, so leaks are silent.

**These errors stay non-fatal — deliberately.** The order write (DynamoDB) and the inventory write (PostgreSQL) are separate stores with no shared transaction, and in every one of these paths the order status is persisted *before* the inventory call. Returning the error would leave the order already cancelled while telling the caller it failed; a retry then hits the status guard and the stock is never released. Reordering does not fix it either — it just moves the window. The correct fix is an idempotent, ledger-checked release keyed on `(product_id, order_id)`, which Task 5 makes possible but which is out of scope here.

So: keep them non-fatal, make them **loud**. A metric can alert; a swallowed error cannot.

- [ ] **Step 1: Write the failing test**

Append inside `TestOrderService_UpdateStatus_Inventory`:

```go
	t.Run("a failed commit does not fail the shipment", func(t *testing.T) {
		order := &domain.Order{
			ID:     "order_fail",
			Status: domain.OrderStatusConfirmed,
			Items:  []domain.OrderItem{{ProductID: "prod_123", Quantity: 2}},
		}

		mockOrderRepo.EXPECT().GetByID(gomock.Any(), "order_fail").Return(order, nil)
		mockOrderRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
		mockInventoryRepo.EXPECT().
			CommitStock(gomock.Any(), "prod_123", 2, "order_fail").
			Return(nil, errors.New(errors.ErrCodeInsufficientStock, "insufficient stock"))

		err := service.UpdateStatus(ctx, "order_fail", domain.OrderStatusShipped, "admin_123")
		require.NoError(t, err, "inventory failure must not block the status change")
		require.Equal(t, domain.OrderStatusShipped, order.Status)
	})
```

- [ ] **Step 2: Run the test to verify it passes already**

```bash
cd handloom-admin && go test ./internal/service/ -run TestOrderService_UpdateStatus_Inventory -v
```

Expected: PASS. This test pins the non-fatal contract so a later change cannot silently make it fatal. The metric itself is verified by inspection in Step 4, not by assertion — there is no metrics test harness in this package.

- [ ] **Step 3: Add the metric at each site**

Follow the existing call style in this file, e.g. `order_service.go:340-343`. In the SHIPPED case:

```go
			if _, commitErr := s.inventoryRepo.CommitStock(ctx, item.ProductID, item.Quantity, order.ID); commitErr != nil {
				slog.ErrorContext(ctx, "Failed to commit stock", keyProductID, item.ProductID, "error", commitErr)
				metrics.Record(ctx, "inventory_mutation_failed", metrics.L{metrics.LabelReason: "commit"})
			}
```

In the RETURNED case, the same shape with `"restock"`. In the CANCELLED case of `UpdateStatus` and in the release loop of `CancelOrder`, the same shape with `"release"`. In `payment_service.go:345-347` and `checkout_service.go:278-280`, the same shape with `"release"`.

Do not add the product or order ID as a metric label — that would make cardinality unbounded. Both are already in the structured log.

- [ ] **Step 4: Verify every site is covered**

```bash
cd handloom-admin && grep -c "inventory_mutation_failed" internal/service/order_service.go internal/service/payment_service.go internal/service/checkout_service.go
```

Expected: `order_service.go:4`, `payment_service.go:1`, `checkout_service.go:1`.

- [ ] **Step 5: Run the full backend suite**

```bash
cd handloom-admin && make test && golangci-lint run
```

Expected: PASS, no lint errors.

- [ ] **Step 6: Commit**

```bash
git add internal/service/order_service.go internal/service/payment_service.go \
        internal/service/checkout_service.go internal/service/order_service_test.go
git commit -m "feat(inventory): alert on failed inventory mutations

Every inventory call outside a transaction logged its error and discarded it,
so a failed release reported a successful cancel and leaked silently.

Kept non-fatal on purpose: orders live in DynamoDB and inventory in Postgres
with no shared transaction, and the status is persisted first, so returning
the error would report failure on an already-cancelled order and a retry
would hit the status guard. Idempotent ledger-checked release is the real
fix; this makes the failure visible until then."
```

---

## Task 7: Reset the leaked reservations in dev

**Files:**
- Create: `docs/superpowers/runbooks/inventory-reserved-qty-reset.md`

**Interfaces:**
- Consumes: all preceding tasks must be deployed first.
- Produces: nothing consumed by later tasks.

Existing dev data already carries leaked reservations, and the fix does not repair them: leaked reservations belong to delivered orders that will never transition again. The catalog is dev-only and disposable, so a blanket reset is appropriate.

- [ ] **Step 1: Write the runbook**

Create `docs/superpowers/runbooks/inventory-reserved-qty-reset.md`:

````markdown
# Runbook: reset leaked `reserved_qty` (dev)

**When:** once, after the inventory lifecycle fix is deployed to dev.
**Why:** before the fix, `quantity` was never decremented and successful sales
leaked their reservations permanently. Those reservations belong to orders that
will never transition again, so they do not self-heal.

**Do not run this against an environment holding real orders.** It erases
legitimate reservations along with leaked ones.

## 1. Deploy the fix first

Confirm the fix is live before resetting, or new leaks accrue in the gap:

```bash
gh workflow run deploy-backend.yml -f environment=dev
```

## 2. Cancel open orders

Any order in `PENDING`, `CONFIRMED` or `PROCESSING` holds a legitimate
reservation that this reset would erase. Its later cancellation would then call
`ReleaseStock` against a zeroed `reserved_qty`, fail the insufficient-stock
guard, and — now that the failure is no longer silent — surface as an
`inventory_mutation_failed` metric.

Cancel them from the admin UI (Orders → filter by status → Cancel), or confirm
there are none.

## 3. Measure before changing anything

```sql
SELECT COUNT(*) AS affected_products, SUM(reserved_qty) AS total_leaked
FROM inventory
WHERE reserved_qty > 0;
```

Record the output in the change ticket.

## 4. Reset

```sql
UPDATE inventory SET reserved_qty = 0, available_qty = quantity;
```

## 5. Verify

```sql
SELECT COUNT(*) FROM inventory WHERE reserved_qty <> 0 OR available_qty <> quantity;
```

Expected: `0`.

## 6. Confirm the fix holds

On the storefront, place an order against a test product, mark it SHIPPED in
admin, and check the product:

```sql
SELECT quantity, reserved_qty, available_qty FROM inventory WHERE product_id = '<id>';
```

Expected: `quantity` down by the ordered amount, `reserved_qty` back to 0,
`available_qty` equal to `quantity`. The ledger should show one `RESERVE` and
one `COMMIT` row sharing the same `reference_id`:

```sql
SELECT type, quantity, reference_id FROM inventory_transactions
WHERE product_id = '<id>' ORDER BY created_at DESC LIMIT 5;
```
````

- [ ] **Step 2: Commit**

```bash
git add docs/superpowers/runbooks/inventory-reserved-qty-reset.md
git commit -m "docs(runbook): reset leaked reserved_qty in dev"
```

- [ ] **Step 3: Execute the runbook**

Follow it against dev. This is a manual step; record the step 3 measurement in the PR description.

---

## Verification

After all tasks:

```bash
cd handloom-admin && make test && golangci-lint run && make build-lambdas-active
```

Manual check against local, per Task 7 step 6: order → ship → confirm `quantity` fell, `reserved_qty` returned to zero, `available_qty` held steady, and the ledger shows a joinable `RESERVE`/`COMMIT` pair.

## Out of scope

Tracked in the spec's findings section, not addressed here:

- **Checkout is not idempotent** — N rapid Pay Now clicks create N orders and N reservations.
- **No sweeper for abandoned checkouts** — a `PENDING` order holds its reservation forever. After this work, that is the only remaining permanent-leak path.
- **Idempotent ledger-checked release** — the real fix for the cross-store failure window described in Task 6.
- **`PreviousQty`/`NewQty` semantics differ across ledger types** — `RESERVE`/`RELEASE` track `reserved_qty`; `ADD`/`REMOVE`/`ADJUST`/`COMMIT` track `quantity`.
