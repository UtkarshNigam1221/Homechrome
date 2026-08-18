# Refund Flow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let an admin refund a customer in full or per line item, moving real money through PhonePe, with per-line inventory disposition and idempotent webhook settlement.

**Architecture:** A new `Refund` entity in the orders DynamoDB table, a dedicated `RefundService` that persists before it calls the gateway, two new PhonePe client methods, and a new atomic `WriteOffStock` PostgreSQL operation. Settlement arrives on the existing PhonePe webhook route and is gated by a single conditional update on the refund record, so concurrent deliveries apply exactly once. The stub `POST /admin/orders/{id}/refund` — which marks money refunded without moving any — is deleted.

**Tech Stack:** Go 1.25, Chi, DynamoDB (aws-sdk-go-v2), PostgreSQL (pgx v5), Google Wire, mockgen, testify. Frontends: React 19 + TypeScript (admin), Next.js 16 (storefront).

**Spec:** `docs/superpowers/specs/2026-08-18-refund-flow-design.md`

## Global Constraints

- All money is `int64` **paise** (1 INR = 100 paise). Never float.
- The client never sends an amount. The server derives it. A client-supplied amount is not read.
- Services return `*errors.AppError`; handlers call `response.Error(w, err)`.
- Handler validation is `middleware.ValidateJSONTyped[T]` as Chi middleware, then `middleware.MustGetValidatedBody[T]` in the handler.
- Metric label **keys** must come from the `metrics.Label*` vocabulary in `pkg/metrics/labels.go`. `LabelReason` and `LabelGateway` already exist; no new keys are needed. Label **values** must be bounded.
- Import ordering is goimports with local prefix `github.com/handloom/admin`.
- `internal/mocks/*_mock.go` is **gitignored and generated** — never commit it, always regenerate with `make generate-mocks` after changing a `domain/` interface.
- Run `make wire` after changing Wire providers; commit the regenerated `internal/wire/wire_gen.go`.
- golangci-lint thresholds: gocognit=30, gocyclo=25, dupl=200. Run `golangci-lint run ./...` before each commit.
- Per `CLAUDE.md`, a backend API change ships with both frontends updated **in the same PR**.
- Commit messages: Conventional Commits, first line under 70 chars.

## Reference: the corrected model

```
create   →  Refund{PENDING} persisted  →  PhonePe /payments/v2/refund
                                            ├─ transport error → Refund{FAILED}, no inventory effect
                                            └─ ok → store ProviderRefundID → inventory effect
settle   →  webhook pg.refund.completed / pg.refund.failed
                └─ conditional update on Refund (status = PENDING) is the ONE gate
                     └─ winner applies: payment total, item quantities, order payment status, notification
recheck  →  GET /payments/v2/refund/{merchantRefundId}/status → same settlement path
```

Inventory, at creation time only:

| Order status | Line marked | Effect |
|---|---|---|
| PENDING / CONFIRMED / PROCESSING | restock | `ReleaseStock` |
| PENDING / CONFIRMED / PROCESSING | write off | `WriteOffStock` |
| SHIPPED / DELIVERED | either | none — `RETURNED` owns post-dispatch restocking |

## File Structure

**Create:**
- `internal/domain/refund.go` — `Refund`, `RefundItem`, `RefundStatus`, `RefundReason`, `CreateRefundRequest`, `RefundWebhookEvent`, `RefundRepository`, `RefundService` interfaces
- `internal/domain/refund_test.go` — key-pattern test
- `internal/repository/dynamodb/refund_repository.go` — `RefundRepository` implementation
- `internal/repository/dynamodb/refund_repository_test.go` — DynamoDB Local integration test
- `internal/service/refund_amount.go` — pure amount-derivation helpers
- `internal/service/refund_amount_test.go` — table-driven money tests
- `internal/service/refund_service.go` — `RefundService`
- `internal/service/refund_service_test.go` — mocked service tests
- `handloom-admin-frontend/src/features/orders/RefundModal.tsx` — refund composer
- `handloom-admin-frontend/src/features/orders/components/OrderDetailPage/RefundSection.tsx` — refund list + modal trigger

**Modify:**
- `internal/domain/entity.go` — `PaymentStatusPartiallyRefunded`, `InventoryTransactionTypeWriteOff`
- `internal/domain/order.go` — `OrderItem.RefundedQuantity`
- `internal/domain/repository.go` — `InventoryRepository.WriteOffStock`
- `internal/domain/store_repository.go` — `PaymentRepository.AddRefundAmount`
- `internal/domain/order_repository.go` — remove `OrderService.RefundOrder`
- `internal/domain/store_service.go` — remove `PaymentService.RefundPayment`
- `internal/validator/custom_rules.go` — `PARTIALLY_REFUNDED`
- `internal/repository/postgres/inventory_repository.go` — `WriteOffStock`
- `internal/repository/dynamodb/payment_repository.go` — `AddRefundAmount`
- `internal/gateway/phonepe/types.go`, `client.go`, `dev_client.go` — refund methods
- `internal/service/order_service.go` — remove `RefundOrder`
- `internal/service/payment_service.go` — remove `RefundPayment`
- `internal/handler/order_handler.go` — refund routes replace the stub
- `internal/handler/request_types.go` — remove `RefundOrderRequest`
- `internal/handler/store/webhook_handler.go` — route `pg.refund.*`
- `internal/router/order.go` — admin-only refund subrouter
- `internal/wire/providers.go`, `wire.go` — refund providers
- `Makefile` — mockgen line for `refund.go`
- `handloom-admin-frontend/src/features/orders/types.ts`, `api.ts`, `src/shared/constants/routes.ts`
- `handloom-admin-frontend/src/features/inventory/types.ts` — `'WRITE_OFF'`
- `homechrome-store/src/types/index.ts`, `src/lib/utils.ts`

---

## Task 1: Domain model for refunds

Adds every type the rest of the plan refers to. Nothing else compiles without it.

**Files:**
- Create: `internal/domain/refund.go`
- Test: `internal/domain/refund_test.go`
- Modify: `internal/domain/entity.go:120` (payment statuses), `internal/domain/entity.go:132` (ledger types)
- Modify: `internal/domain/order.go:94-113` (`OrderItem`)
- Modify: `internal/validator/custom_rules.go:100-106`
- Modify: `Makefile:127-139` (generate-mocks)

**Interfaces:**
- Consumes: nothing.
- Produces: `domain.Refund`, `domain.RefundItem`, `domain.RefundStatus` (`RefundStatusPending`/`Completed`/`Failed`), `domain.RefundReason` (5 values), `domain.CreateRefundRequest`, `domain.RefundItemRequest`, `domain.RefundWebhookEvent`, `domain.RefundRepository`, `domain.RefundService`, `domain.PaymentStatusPartiallyRefunded`, `domain.InventoryTransactionTypeWriteOff`, `OrderItem.RefundedQuantity int`.

- [ ] **Step 1: Write the failing test**

Create `internal/domain/refund_test.go`:

```go
package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRefund_SetKeys(t *testing.T) {
	initiated := time.Date(2026, 8, 18, 10, 30, 0, 0, time.UTC)
	r := &Refund{
		ID:               "refund_abc",
		OrderID:          "order_123",
		MerchantRefundID: "mref_xyz",
		InitiatedAt:      initiated,
	}

	r.SetKeys()

	require.Equal(t, "REFUND#refund_abc", r.PK)
	require.Equal(t, SKMetadata, r.SK)
	require.Equal(t, "ORDER#order_123", r.GSI1PK, "refunds for an order must share the order's GSI1 partition")
	require.Equal(t, "REFUND#2026-08-18T10:30:00Z", r.GSI1SK)
	require.Equal(t, "REFUND_TXN", r.GSI2PK)
	require.Equal(t, "mref_xyz", r.GSI2SK)
	require.Equal(t, "REFUND", r.EntityType)
	require.Equal(t, TableOrders, r.TableName())
}

func TestRefund_GSI1SKSortsChronologically(t *testing.T) {
	// GSI1SK ordering is what makes ListByOrderID return refunds oldest-first
	// without a client-side sort.
	early := &Refund{ID: "a", OrderID: "o", InitiatedAt: time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)}
	late := &Refund{ID: "b", OrderID: "o", InitiatedAt: time.Date(2026, 8, 18, 11, 0, 0, 0, time.UTC)}
	early.SetKeys()
	late.SetKeys()

	require.Less(t, early.GSI1SK, late.GSI1SK)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/domain/... -run TestRefund -v`
Expected: FAIL — `undefined: Refund`.

- [ ] **Step 3: Add the payment status and the ledger type**

In `internal/domain/entity.go`, extend the `PaymentStatus` block (currently lines 114-121):

```go
const (
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusPaid      PaymentStatus = "PAID"
	PaymentStatusFailed    PaymentStatus = "FAILED"
	PaymentStatusRefunded  PaymentStatus = "REFUNDED"
	PaymentStatusInitiated PaymentStatus = "INITIATED"
	PaymentStatusSuccess   PaymentStatus = "SUCCESS"
	// PaymentStatusPartiallyRefunded means at least one refund settled but the
	// order total has not been fully refunded. The order keeps its fulfilment
	// status — the unrefunded remainder still ships.
	PaymentStatusPartiallyRefunded PaymentStatus = "PARTIALLY_REFUNDED"
)
```

And extend the `InventoryTransactionType` block (currently lines 126-133):

```go
const (
	InventoryTransactionTypeAdd      InventoryTransactionType = "ADD"
	InventoryTransactionTypeRemove   InventoryTransactionType = "REMOVE"
	InventoryTransactionTypeReserve  InventoryTransactionType = "RESERVE"
	InventoryTransactionTypeRelease  InventoryTransactionType = "RELEASE"
	InventoryTransactionTypeAdjust   InventoryTransactionType = "ADJUST"
	InventoryTransactionTypeCommit   InventoryTransactionType = "COMMIT"
	// InventoryTransactionTypeWriteOff records units destroyed or lost: the
	// reservation is released AND on-hand drops, so available_qty is unchanged.
	// Arithmetically identical to COMMIT, semantically distinct — the ledger
	// must be able to tell a dispatch from a write-off.
	InventoryTransactionTypeWriteOff InventoryTransactionType = "WRITE_OFF"
)
```

- [ ] **Step 4: Add `RefundedQuantity` to `OrderItem`**

In `internal/domain/order.go`, in the `OrderItem` struct's `// Pricing` block (currently lines 108-112), append:

```go
	// Pricing
	UnitPrice  int64 `json:"unit_price" dynamodbav:"unit_price"`
	Quantity   int   `json:"quantity" dynamodbav:"quantity"`
	TotalPrice int64 `json:"total_price" dynamodbav:"total_price"`

	// RefundedQuantity is how many units of this line have been refunded by a
	// refund that actually SETTLED. Units held by a still-PENDING refund are
	// not counted here — the service adds those separately when it caps a new
	// refund. Written only by refund settlement.
	RefundedQuantity int `json:"refunded_quantity,omitempty" dynamodbav:"refunded_quantity,omitempty"`
```

- [ ] **Step 5: Create the refund domain file**

Create `internal/domain/refund.go`:

```go
package domain

import (
	"context"
	"time"
)

//go:generate mockgen -source=refund.go -destination=../mocks/refund_mock.go -package=mocks

// ==================== REFUND ENTITY ====================

// RefundStatus is the lifecycle state of a single refund attempt. PhonePe has
// no "accepted" state — a refund goes PENDING → terminal.
type RefundStatus string

const (
	RefundStatusPending   RefundStatus = "PENDING"
	RefundStatusCompleted RefundStatus = "COMPLETED"
	RefundStatusFailed    RefundStatus = "FAILED"
)

// RefundReason is a bounded set so it can label a metric without unbounded
// cardinality. Free text belongs in an order note, not here.
type RefundReason string

const (
	RefundReasonOutOfStock      RefundReason = "OUT_OF_STOCK"
	RefundReasonDamaged         RefundReason = "DAMAGED"
	RefundReasonCustomerRequest RefundReason = "CUSTOMER_REQUEST"
	RefundReasonPricingError    RefundReason = "PRICING_ERROR"
	RefundReasonOther           RefundReason = "OTHER"
)

// RefundItem is one refunded order line.
type RefundItem struct {
	OrderItemID string `json:"order_item_id" dynamodbav:"order_item_id"`
	ProductID   string `json:"product_id" dynamodbav:"product_id"`
	Quantity    int    `json:"quantity" dynamodbav:"quantity"`
	Amount      int64  `json:"amount" dynamodbav:"amount"` // paise, server-derived
	// Restock true returns the units to sale; false writes them off. Only read
	// for orders that have not been dispatched.
	Restock bool `json:"restock" dynamodbav:"restock"`
}

// Refund is one refund attempt against one order. A separate entity rather
// than fields on Payment because partial refunds mean many refunds per payment.
type Refund struct {
	ID         string `json:"id" dynamodbav:"id"`
	PK         string `json:"-" dynamodbav:"PK"`
	SK         string `json:"-" dynamodbav:"SK"`
	GSI1PK     string `json:"-" dynamodbav:"GSI1PK"`
	GSI1SK     string `json:"-" dynamodbav:"GSI1SK"`
	GSI2PK     string `json:"-" dynamodbav:"GSI2PK"`
	GSI2SK     string `json:"-" dynamodbav:"GSI2SK"`
	EntityType string `json:"-" dynamodbav:"entity_type"`

	OrderID    string `json:"order_id" dynamodbav:"order_id"`
	PaymentID  string `json:"payment_id" dynamodbav:"payment_id"`
	CustomerID string `json:"customer_id" dynamodbav:"customer_id"`

	Amount int64        `json:"amount" dynamodbav:"amount"` // paise
	Status RefundStatus `json:"status" dynamodbav:"status"`
	Reason RefundReason `json:"reason" dynamodbav:"reason"`
	Items  []RefundItem `json:"items" dynamodbav:"items"`

	// MerchantRefundID is ours, unique per attempt. ProviderRefundID is
	// PhonePe's and is empty until initiation returns — the webhook correlates
	// on it, so a lost initiation response is recovered via the status
	// endpoint, which is keyed on MerchantRefundID.
	MerchantRefundID string `json:"merchant_refund_id" dynamodbav:"merchant_refund_id"`
	ProviderRefundID string `json:"provider_refund_id,omitempty" dynamodbav:"provider_refund_id,omitempty"`

	ErrorCode         string `json:"error_code,omitempty" dynamodbav:"error_code,omitempty"`
	DetailedErrorCode string `json:"detailed_error_code,omitempty" dynamodbav:"detailed_error_code,omitempty"`

	InitiatedAt time.Time  `json:"initiated_at" dynamodbav:"initiated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" dynamodbav:"completed_at,omitempty"`
	CreatedBy   string     `json:"created_by" dynamodbav:"created_by"`

	BaseEntity
}

// TableName returns the DynamoDB table name for Refund.
func (r *Refund) TableName() string {
	return TableOrders
}

// SetKeys sets the DynamoDB keys for Refund. GSI2 is written for operational
// lookup by merchant refund ID; no repository method reads it today because
// re-check is keyed on our own refund ID.
func (r *Refund) SetKeys() {
	r.PK = "REFUND#" + r.ID
	r.SK = SKMetadata
	r.GSI1PK = "ORDER#" + r.OrderID
	r.GSI1SK = "REFUND#" + r.InitiatedAt.Format("2006-01-02T15:04:05Z")
	r.GSI2PK = "REFUND_TXN"
	r.GSI2SK = r.MerchantRefundID
	r.EntityType = "REFUND"
}

// ==================== REFUND REQUEST TYPES ====================

// CreateRefundRequest is the admin's refund request. Note the absence of an
// amount: the server derives it from the lines.
type CreateRefundRequest struct {
	Reason RefundReason        `json:"reason" validate:"required,oneof=OUT_OF_STOCK DAMAGED CUSTOMER_REQUEST PRICING_ERROR OTHER"`
	Items  []RefundItemRequest `json:"items" validate:"required,min=1,dive"`
}

// RefundItemRequest is one requested line.
type RefundItemRequest struct {
	OrderItemID string `json:"order_item_id" validate:"required"`
	Quantity    int    `json:"quantity" validate:"required,gt=0"`
	Restock     bool   `json:"restock"`
}

// RefundWebhookEvent is a provider-agnostic refund settlement event. The
// webhook handler translates the provider payload into this struct.
//
// Correlation: PhonePe does not echo our MerchantRefundID, so the pair
// (OriginalMerchantOrderID, ProviderRefundID) is what identifies the refund.
type RefundWebhookEvent struct {
	OriginalMerchantOrderID string
	ProviderRefundID        string
	Amount                  int64
	ErrorCode               string
	DetailedErrorCode       string
}

// ==================== REFUND INTERFACES ====================

// RefundRepository defines refund data access operations.
type RefundRepository interface {
	Create(ctx context.Context, refund *Refund) error
	GetByID(ctx context.Context, id string) (*Refund, error)

	// ListByOrderID returns every refund for an order, oldest first.
	ListByOrderID(ctx context.Context, orderID string) ([]*Refund, error)

	// SetProviderRefundID records PhonePe's refund ID after initiation returns.
	SetProviderRefundID(ctx context.Context, id string, providerRefundID string) error

	// SettleIfPending moves a refund from PENDING to a terminal status under a
	// ConditionExpression. It is the single gate every downstream settlement
	// effect runs behind: of two concurrent webhook deliveries exactly one
	// succeeds and the other returns ErrCodeConflict.
	SettleIfPending(ctx context.Context, id string, status RefundStatus, updates map[string]interface{}) error
}

// RefundService defines refund operations.
type RefundService interface {
	Create(ctx context.Context, orderID string, req CreateRefundRequest, createdBy string) (*Refund, error)
	ListByOrder(ctx context.Context, orderID string) ([]*Refund, error)
	RecheckStatus(ctx context.Context, refundID string) (*Refund, error)
	HandleRefundCompleted(ctx context.Context, evt RefundWebhookEvent) error
	HandleRefundFailed(ctx context.Context, evt RefundWebhookEvent) error
}
```

- [ ] **Step 6: Add `PARTIALLY_REFUNDED` to the validator**

In `internal/validator/custom_rules.go`, replace the `validPaymentStatuses` map (lines 100-106):

```go
// Payment status constants
var validPaymentStatuses = map[string]bool{
	statusPending:         true,
	"PAID":                true,
	"FAILED":              true,
	"REFUNDED":            true,
	"PARTIALLY_REFUNDED":  true,
	"CANCELED":            true,
}
```

- [ ] **Step 7: Register the new mock source**

In `Makefile`, inside the `generate-mocks` target, add a line after the `order_repository.go` line:

```makefile
	mockgen -source=internal/domain/refund.go -destination=internal/mocks/refund_mock.go -package=mocks
```

- [ ] **Step 8: Regenerate mocks**

Run: `make generate-mocks`
Expected: no error. `internal/mocks/refund_mock.go` now exists (gitignored — do not `git add` it).

- [ ] **Step 9: Run the test to verify it passes**

Run: `go test ./internal/domain/... -run TestRefund -v`
Expected: PASS, both subtests.

- [ ] **Step 10: Verify the tree still builds and lints**

Run: `go build ./... && golangci-lint run ./internal/domain/... ./internal/validator/...`
Expected: no output.

- [ ] **Step 11: Commit**

```bash
git add internal/domain/refund.go internal/domain/refund_test.go \
        internal/domain/entity.go internal/domain/order.go \
        internal/validator/custom_rules.go Makefile
git commit -m "feat(refunds): add the refund domain model"
```

---

## Task 2: `WriteOffStock` in the inventory repository

Write-off is two mutations — release the reservation and reduce on-hand. As
`ReleaseStock` then `RemoveStock` it is not atomic: a crash between them leaves
the units released and back on sale, the exact opposite of a write-off.

**Files:**
- Modify: `internal/domain/repository.go:262-266` (add to `InventoryRepository`)
- Modify: `internal/repository/postgres/inventory_repository.go` (add after `CommitStock`, which ends at line 358)
- Test: `internal/repository/postgres/inventory_repository_test.go`

**Interfaces:**
- Consumes: `domain.InventoryTransactionTypeWriteOff` (Task 1).
- Produces: `InventoryRepository.WriteOffStock(ctx context.Context, productID string, quantity int, orderID string) (*InventoryTransaction, error)`.

- [ ] **Step 1: Write the failing test**

Append to `internal/repository/postgres/inventory_repository_test.go`:

```go
func TestInventoryRepository_WriteOffStock(t *testing.T) {
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

	t.Run("write-off decrements quantity and reserved, leaving available unchanged", func(t *testing.T) {
		productID := newProduct(t, 100, 10)

		txn, err := repo.WriteOffStock(ctx, productID, 4, "order_abc")
		require.NoError(t, err)

		qty, reserved, available := readInventory(t, pool, productID)
		require.Equal(t, 96, qty)
		require.Equal(t, 6, reserved)
		require.Equal(t, 90, available, "available_qty must not move on write-off")

		require.Equal(t, domain.InventoryTransactionTypeWriteOff, txn.Type)
		require.Equal(t, 4, txn.Quantity)
		require.Equal(t, 100, txn.PreviousQty)
		require.Equal(t, 96, txn.NewQty)
		require.Equal(t, "order_abc", txn.ReferenceID)
	})

	t.Run("write-off is distinguishable from a dispatch in the ledger", func(t *testing.T) {
		productID := newProduct(t, 50, 10)

		_, err := repo.CommitStock(ctx, productID, 2, "order_def")
		require.NoError(t, err)
		_, err = repo.WriteOffStock(ctx, productID, 3, "order_def")
		require.NoError(t, err)

		var commits, writeOffs int
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT COUNT(*) FILTER (WHERE type = $2), COUNT(*) FILTER (WHERE type = $3)
			 FROM inventory_transactions WHERE product_id = $1`,
			productID,
			string(domain.InventoryTransactionTypeCommit),
			string(domain.InventoryTransactionTypeWriteOff),
		).Scan(&commits, &writeOffs))
		require.Equal(t, 1, commits)
		require.Equal(t, 1, writeOffs)
	})

	t.Run("write-off beyond reserved is rejected and changes nothing", func(t *testing.T) {
		productID := newProduct(t, 100, 3)

		_, err := repo.WriteOffStock(ctx, productID, 5, "order_ghi")
		require.Error(t, err)
		require.ErrorContains(t, err, "insufficient stock")

		qty, reserved, available := readInventory(t, pool, productID)
		require.Equal(t, 100, qty)
		require.Equal(t, 3, reserved)
		require.Equal(t, 97, available)
	})

	t.Run("write-off on a missing product returns not found", func(t *testing.T) {
		_, err := repo.WriteOffStock(ctx, "does-not-exist", 1, "order_jkl")
		require.Error(t, err)
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/repository/postgres/... -run TestInventoryRepository_WriteOffStock -v`
Expected: FAIL to compile — `repo.WriteOffStock undefined`.

Note: these tests need PostgreSQL. `newTestPool` reads `POSTGRES_DSN` and falls back to `postgres://postgres:postgres@localhost:5432/handloom?sslmode=disable`; it **skips** when the database is unreachable and `CI` is unset, and **fails** when `CI` is set. Start it with `make setup-local` if the run skips.

- [ ] **Step 3: Add the method to the interface**

In `internal/domain/repository.go`, immediately after the `CommitStock` entry in `InventoryRepository`:

```go
	// WriteOffStock destroys reserved units: the reservation is released and
	// on-hand drops by the same amount, so available_qty is unchanged. Used
	// when a refunded line cannot be returned to sale. Atomic — doing it as
	// ReleaseStock followed by RemoveStock would put the units back on sale if
	// the process died in between.
	WriteOffStock(ctx context.Context, productID string, quantity int, orderID string) (*InventoryTransaction, error)
```

- [ ] **Step 4: Regenerate mocks**

Run: `make generate-mocks`
Expected: no error.

- [ ] **Step 5: Implement `WriteOffStock`**

In `internal/repository/postgres/inventory_repository.go`, immediately after `CommitStock`:

```go
// WriteOffStock destroys reserved units. Arithmetically identical to
// CommitStock — reserved_qty -= q and quantity -= q, so available_qty holds —
// but it writes a WRITE_OFF ledger row so a write-off is never mistaken for a
// dispatch.
func (r *InventoryRepository) WriteOffStock(ctx context.Context, productID string, quantity int, orderID string) (*domain.InventoryTransaction, error) {
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

		// Same two guards as CommitStock: reservedQty is the real invariant,
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
			Type:          domain.InventoryTransactionTypeWriteOff,
			Quantity:      quantity,
			PreviousQty:   currentQty,
			NewQty:        newQty,
			Reason:        fmt.Sprintf("WRITE-OFF order %s", orderID),
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

- [ ] **Step 6: Run the test to verify it passes**

Run: `go test ./internal/repository/postgres/... -run TestInventoryRepository_WriteOffStock -v`
Expected: PASS, all four subtests.

- [ ] **Step 7: Run the whole repository package for regressions**

Run: `go test ./internal/repository/... `
Expected: `ok`.

- [ ] **Step 8: Lint and commit**

```bash
golangci-lint run ./internal/repository/postgres/... ./internal/domain/...
git add internal/domain/repository.go \
        internal/repository/postgres/inventory_repository.go \
        internal/repository/postgres/inventory_repository_test.go
git commit -m "feat(inventory): add atomic WriteOffStock"
```

---

## Task 3: Amount derivation

Pure integer arithmetic, isolated from every repository and gateway so the money
rules are testable on their own. This is the only place a refund amount is ever
computed.

**Files:**
- Create: `internal/service/refund_amount.go`
- Test: `internal/service/refund_amount_test.go`

**Interfaces:**
- Consumes: `domain.Order`, `domain.OrderItem`, `domain.Refund`, `domain.RefundStatusPending` (Task 1).
- Produces, all unexported within `package service`:
  - `type refundLine struct { item domain.OrderItem; quantity int }`
  - `prorate(amount, part, whole int64) int64`
  - `openRefundTotal(existing []*domain.Refund) int64`
  - `refundedQuantities(order *domain.Order, existing []*domain.Refund) map[string]int`
  - `isFinalRefund(order *domain.Order, lines []refundLine, alreadyRefundedQty map[string]int) bool`
  - `computeRefundAmount(order *domain.Order, lines []refundLine, alreadyRefundedQty map[string]int, alreadyRefundedAmount int64) int64`
  - `distributeRefundAmount(order *domain.Order, lines []refundLine, total int64) []int64`

- [ ] **Step 1: Write the failing test**

Create `internal/service/refund_amount_test.go`:

```go
package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
)

// orderWithLines builds an order whose Subtotal is the sum of its lines, so
// proration denominators are realistic.
func orderWithLines(discount, tax, shipping int64, items ...domain.OrderItem) *domain.Order {
	var subtotal int64
	for _, it := range items {
		subtotal += it.UnitPrice * int64(it.Quantity)
	}
	return &domain.Order{
		Items:          items,
		Subtotal:       subtotal,
		DiscountAmount: discount,
		TaxAmount:      tax,
		ShippingAmount: shipping,
		TotalAmount:    subtotal - discount + tax + shipping,
	}
}

func item(id string, unitPrice int64, quantity, refunded int) domain.OrderItem {
	return domain.OrderItem{
		ID:               id,
		ProductID:        "prod_" + id,
		UnitPrice:        unitPrice,
		Quantity:         quantity,
		TotalPrice:       unitPrice * int64(quantity),
		RefundedQuantity: refunded,
	}
}

func TestProrate(t *testing.T) {
	tests := []struct {
		name                string
		amount, part, whole int64
		want                int64
	}{
		{"zero amount", 0, 500, 1000, 0},
		{"zero whole is not a division by zero", 100, 500, 0, 0},
		{"exact half rounds up", 1, 1, 2, 1},
		{"below half rounds down", 10, 334, 1000, 3},
		{"whole share", 1000, 1000, 1000, 1000},
		{"typical share", 10000, 30000, 100000, 3000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, prorate(tt.amount, tt.part, tt.whole))
		})
	}
}

func TestComputeRefundAmount_SingleLine(t *testing.T) {
	// ₹1000 subtotal, ₹100 discount, ₹50 shipping → ₹950 total.
	order := orderWithLines(10000, 0, 5000,
		item("oi_1", 30000, 2, 0),
		item("oi_2", 40000, 1, 0),
	)

	lines := []refundLine{{item: order.Items[0], quantity: 1}}

	// 30000 line − prorated discount round(10000 × 30000/100000) = 3000 → 27000.
	// Shipping is retained: the parcel still ships.
	got := computeRefundAmount(order, lines, refundedQuantities(order, nil), 0)
	require.Equal(t, int64(27000), got)
}

func TestComputeRefundAmount_MultipleLines(t *testing.T) {
	order := orderWithLines(10000, 0, 5000,
		item("oi_1", 30000, 2, 0),
		item("oi_2", 40000, 1, 0),
	)

	lines := []refundLine{
		{item: order.Items[0], quantity: 1}, // 30000 − 3000
		{item: order.Items[1], quantity: 1}, // 40000 − 4000
	}

	got := computeRefundAmount(order, lines, refundedQuantities(order, nil), 0)
	require.Equal(t, int64(63000), got)
}

func TestComputeRefundAmount_TaxIsProratedToo(t *testing.T) {
	// CheckoutService sets tax to zero today; the term exists so the formula
	// stays correct if tax starts being charged.
	order := orderWithLines(0, 10000, 0, item("oi_1", 50000, 2, 0))

	lines := []refundLine{{item: order.Items[0], quantity: 1}}

	// 50000 + prorated tax round(10000 × 50000/100000) = 5000 → 55000.
	got := computeRefundAmount(order, lines, refundedQuantities(order, nil), 0)
	require.Equal(t, int64(55000), got)
}

func TestComputeRefundAmount_FinalRefundIncludesShipping(t *testing.T) {
	order := orderWithLines(10000, 0, 5000, item("oi_1", 100000, 1, 0))

	lines := []refundLine{{item: order.Items[0], quantity: 1}}

	// Clears the last unrefunded unit → the whole order total, shipping included.
	got := computeRefundAmount(order, lines, refundedQuantities(order, nil), 0)
	require.Equal(t, order.TotalAmount, got)
	require.Equal(t, int64(95000), got)
}

func TestComputeRefundAmount_FinalRefundAbsorbsProrationResidual(t *testing.T) {
	// Three lines whose prorated discounts each round up. Without residual
	// absorption the three refunds would sum to 991 against a 990 total, so
	// the order could never reach fully-refunded.
	order := orderWithLines(10, 0, 0,
		item("oi_1", 334, 1, 0),
		item("oi_2", 333, 1, 0),
		item("oi_3", 333, 1, 0),
	)
	require.Equal(t, int64(990), order.TotalAmount)

	first := computeRefundAmount(order, []refundLine{{item: order.Items[0], quantity: 1}},
		refundedQuantities(order, nil), 0)
	require.Equal(t, int64(331), first)

	// oi_1 has settled.
	order.Items[0].RefundedQuantity = 1
	second := computeRefundAmount(order, []refundLine{{item: order.Items[1], quantity: 1}},
		refundedQuantities(order, nil), first)
	require.Equal(t, int64(330), second)

	// oi_2 has settled; the third clears the order and absorbs the residual.
	order.Items[1].RefundedQuantity = 1
	third := computeRefundAmount(order, []refundLine{{item: order.Items[2], quantity: 1}},
		refundedQuantities(order, nil), first+second)
	require.Equal(t, int64(329), third)

	require.Equal(t, order.TotalAmount, first+second+third,
		"refunds against an order must sum to its total exactly")
}

func TestIsFinalRefund(t *testing.T) {
	order := orderWithLines(0, 0, 0,
		item("oi_1", 100, 2, 0),
		item("oi_2", 100, 1, 0),
	)

	t.Run("not final while units remain", func(t *testing.T) {
		lines := []refundLine{{item: order.Items[0], quantity: 2}}
		require.False(t, isFinalRefund(order, lines, refundedQuantities(order, nil)))
	})

	t.Run("final when the request clears every remaining unit", func(t *testing.T) {
		lines := []refundLine{
			{item: order.Items[0], quantity: 2},
			{item: order.Items[1], quantity: 1},
		}
		require.True(t, isFinalRefund(order, lines, refundedQuantities(order, nil)))
	})

	t.Run("final counting units already settled", func(t *testing.T) {
		settled := orderWithLines(0, 0, 0,
			item("oi_1", 100, 2, 2),
			item("oi_2", 100, 1, 0),
		)
		lines := []refundLine{{item: settled.Items[1], quantity: 1}}
		require.True(t, isFinalRefund(settled, lines, refundedQuantities(settled, nil)))
	})
}

func TestRefundedQuantities_CountsPendingButNotFailed(t *testing.T) {
	order := orderWithLines(0, 0, 0, item("oi_1", 100, 5, 1))

	existing := []*domain.Refund{
		{Status: domain.RefundStatusPending, Items: []domain.RefundItem{{OrderItemID: "oi_1", Quantity: 2}}},
		{Status: domain.RefundStatusFailed, Items: []domain.RefundItem{{OrderItemID: "oi_1", Quantity: 2}}},
		// A COMPLETED refund is already reflected in OrderItem.RefundedQuantity;
		// counting it again would double-count.
		{Status: domain.RefundStatusCompleted, Items: []domain.RefundItem{{OrderItemID: "oi_1", Quantity: 1}}},
	}

	got := refundedQuantities(order, existing)
	require.Equal(t, 3, got["oi_1"], "1 settled + 2 pending")
}

func TestOpenRefundTotal_ExcludesFailed(t *testing.T) {
	existing := []*domain.Refund{
		{Status: domain.RefundStatusPending, Amount: 100},
		{Status: domain.RefundStatusCompleted, Amount: 250},
		{Status: domain.RefundStatusFailed, Amount: 900},
	}
	require.Equal(t, int64(350), openRefundTotal(existing))
}

func TestDistributeRefundAmount_SumsToTheTotal(t *testing.T) {
	order := orderWithLines(10, 0, 0,
		item("oi_1", 334, 1, 0),
		item("oi_2", 333, 1, 0),
		item("oi_3", 333, 1, 0),
	)

	t.Run("per-line amounts sum to a partial total", func(t *testing.T) {
		lines := []refundLine{
			{item: order.Items[0], quantity: 1},
			{item: order.Items[1], quantity: 1},
		}
		total := computeRefundAmount(order, lines, refundedQuantities(order, nil), 0)

		amounts := distributeRefundAmount(order, lines, total)
		require.Len(t, amounts, 2)
		require.Equal(t, total, amounts[0]+amounts[1])
		require.Equal(t, int64(331), amounts[0])
	})

	t.Run("the last line absorbs the residual on a final refund", func(t *testing.T) {
		lines := []refundLine{
			{item: order.Items[0], quantity: 1},
			{item: order.Items[1], quantity: 1},
			{item: order.Items[2], quantity: 1},
		}
		total := computeRefundAmount(order, lines, refundedQuantities(order, nil), 0)
		require.Equal(t, order.TotalAmount, total)

		amounts := distributeRefundAmount(order, lines, total)
		var sum int64
		for _, a := range amounts {
			sum += a
		}
		require.Equal(t, total, sum, "line amounts must reconcile against the refund total")
		require.Equal(t, int64(329), amounts[2], "residual lands on the last line")
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/... -run 'TestProrate|TestComputeRefundAmount|TestIsFinalRefund|TestRefundedQuantities|TestOpenRefundTotal|TestDistributeRefundAmount' -v`
Expected: FAIL to compile — `undefined: prorate`.

- [ ] **Step 3: Implement the helpers**

Create `internal/service/refund_amount.go`:

```go
package service

import "github.com/handloom/admin/internal/domain"

// refundLine pairs an order item with the number of its units being refunded.
type refundLine struct {
	item     domain.OrderItem
	quantity int
}

// prorate returns amount × part / whole, rounded half-up, in integer paise.
// All three arguments are non-negative in every call site (order money fields
// and line subtotals), which is what makes the +whole/2 rounding correct.
func prorate(amount, part, whole int64) int64 {
	if whole <= 0 || amount == 0 {
		return 0
	}
	return (amount*part + whole/2) / whole
}

// openRefundTotal sums the money already committed to this order by refunds
// that have not failed — settled and in-flight alike. A FAILED refund moved no
// money, so it does not count against the order total.
func openRefundTotal(existing []*domain.Refund) int64 {
	var total int64
	for _, r := range existing {
		if r.Status == domain.RefundStatusFailed {
			continue
		}
		total += r.Amount
	}
	return total
}

// refundedQuantities returns, per order-item ID, how many units are already
// spoken for: units written to OrderItem.RefundedQuantity at settlement, plus
// units held by refunds still PENDING. COMPLETED refunds are deliberately not
// re-counted — settlement already folded them into RefundedQuantity.
func refundedQuantities(order *domain.Order, existing []*domain.Refund) map[string]int {
	quantities := make(map[string]int, len(order.Items))
	for _, it := range order.Items {
		quantities[it.ID] = it.RefundedQuantity
	}
	for _, r := range existing {
		if r.Status != domain.RefundStatusPending {
			continue
		}
		for _, ri := range r.Items {
			quantities[ri.OrderItemID] += ri.Quantity
		}
	}
	return quantities
}

// isFinalRefund reports whether this request clears the last unrefunded unit of
// every line on the order.
func isFinalRefund(order *domain.Order, lines []refundLine, alreadyRefundedQty map[string]int) bool {
	requested := make(map[string]int, len(lines))
	for _, l := range lines {
		requested[l.item.ID] += l.quantity
	}
	for _, it := range order.Items {
		if alreadyRefundedQty[it.ID]+requested[it.ID] != it.Quantity {
			return false
		}
	}
	return true
}

// computeRefundAmount derives the refund total in paise.
//
// Per line: unit_price × quantity, less the line's prorated share of the order
// discount, plus its prorated share of tax. Shipping is refunded only by the
// refund that clears the last unrefunded unit, because a partial refund still
// ships a parcel.
//
// That final refund is computed as "whatever is left of the order total"
// rather than by the per-line formula, which absorbs the paise that per-line
// round-half-up leaves behind. Without it the refunds against an order sum to
// a few paise off the total and the order never reaches fully-refunded.
func computeRefundAmount(order *domain.Order, lines []refundLine, alreadyRefundedQty map[string]int, alreadyRefundedAmount int64) int64 {
	if isFinalRefund(order, lines, alreadyRefundedQty) {
		return order.TotalAmount - alreadyRefundedAmount
	}

	var total int64
	for _, l := range lines {
		lineSubtotal := l.item.UnitPrice * int64(l.quantity)
		total += lineSubtotal -
			prorate(order.DiscountAmount, lineSubtotal, order.Subtotal) +
			prorate(order.TaxAmount, lineSubtotal, order.Subtotal)
	}
	return total
}

// distributeRefundAmount splits total across lines so the per-line amounts
// stored on the refund reconcile against the refund's own total exactly. Each
// line gets the per-line formula; the last line takes whatever is left, which
// is where a final refund's shipping and absorbed residual land.
func distributeRefundAmount(order *domain.Order, lines []refundLine, total int64) []int64 {
	amounts := make([]int64, len(lines))
	var sum int64
	for i, l := range lines {
		lineSubtotal := l.item.UnitPrice * int64(l.quantity)
		amounts[i] = lineSubtotal -
			prorate(order.DiscountAmount, lineSubtotal, order.Subtotal) +
			prorate(order.TaxAmount, lineSubtotal, order.Subtotal)
		sum += amounts[i]
	}
	if len(amounts) > 0 {
		amounts[len(amounts)-1] += total - sum
	}
	return amounts
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/service/... -run 'TestProrate|TestComputeRefundAmount|TestIsFinalRefund|TestRefundedQuantities|TestOpenRefundTotal|TestDistributeRefundAmount' -v`
Expected: PASS, every subtest.

- [ ] **Step 5: Lint and commit**

```bash
golangci-lint run ./internal/service/...
git add internal/service/refund_amount.go internal/service/refund_amount_test.go
git commit -m "feat(refunds): derive refund amounts server-side"
```

---

## Task 4: PhonePe refund gateway methods

**Files:**
- Modify: `internal/gateway/phonepe/types.go` (append)
- Modify: `internal/gateway/phonepe/client.go` (append; also add a shared request helper)
- Modify: `internal/gateway/phonepe/dev_client.go` (`Gateway` interface + `DevClient` methods)
- Test: `internal/gateway/phonepe/client_test.go` (append)

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces on `phonepe.Gateway` (so both `Client` and `DevClient` implement them):
  - `InitiateRefund(ctx context.Context, merchantRefundID, originalMerchantOrderID string, amount int64) (*RefundResponse, error)`
  - `CheckRefundStatus(ctx context.Context, merchantRefundID string) (*RefundStatusResponse, error)`
- Also produces the types `phonepe.RefundRequest`, `phonepe.RefundResponse`, `phonepe.RefundStatusResponse`, `phonepe.RefundWebhookPayload`, `phonepe.RefundWebhookOrder`, and the constant `phonepe.StateFailed`.

- [ ] **Step 1: Write the failing test**

Append to `internal/gateway/phonepe/client_test.go`:

```go
func TestClient_InitiateRefund_Success(t *testing.T) {
	server := httptest.NewServer(fakeTokenThenHandler(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/payments/v2/refund", r.URL.Path)
		assert.Equal(t, "O-Bearer test-token", r.Header.Get("Authorization"))

		var body RefundRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "mref_1", body.MerchantRefundID)
		assert.Equal(t, "txn_original", body.OriginalMerchantOrderID)
		assert.Equal(t, int64(2500), body.Amount)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(RefundResponse{
			RefundID: "OMR456",
			Amount:   2500,
			State:    "PENDING",
		})
	}))
	defer server.Close()

	client := NewClient(Config{ClientID: "C", ClientSecret: "S", ClientVersion: "1", BaseURL: server.URL})

	resp, err := client.InitiateRefund(context.Background(), "mref_1", "txn_original", 2500)
	require.NoError(t, err)
	assert.Equal(t, "OMR456", resp.RefundID)
	assert.Equal(t, int64(2500), resp.Amount)
	assert.Equal(t, "PENDING", resp.State)
}

func TestClient_InitiateRefund_Failure(t *testing.T) {
	server := httptest.NewServer(fakeTokenThenHandler(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(PayErrorResponse{Code: "BAD_REQUEST", Message: "amount exceeds order"})
	}))
	defer server.Close()

	client := NewClient(Config{ClientID: "C", ClientSecret: "S", ClientVersion: "1", BaseURL: server.URL})

	_, err := client.InitiateRefund(context.Background(), "mref_1", "txn_original", 2500)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "BAD_REQUEST")
	assert.Contains(t, err.Error(), "amount exceeds order")
}

func TestClient_CheckRefundStatus_Success(t *testing.T) {
	server := httptest.NewServer(fakeTokenThenHandler(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/payments/v2/refund/mref_1/status", r.URL.Path)
		assert.Equal(t, "O-Bearer test-token", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(RefundStatusResponse{
			OriginalMerchantOrderID: "txn_original",
			RefundID:                "OMR456",
			Amount:                  2500,
			State:                   StateCompleted,
		})
	}))
	defer server.Close()

	client := NewClient(Config{ClientID: "C", ClientSecret: "S", ClientVersion: "1", BaseURL: server.URL})

	resp, err := client.CheckRefundStatus(context.Background(), "mref_1")
	require.NoError(t, err)
	assert.Equal(t, "OMR456", resp.RefundID)
	assert.Equal(t, StateCompleted, resp.State)
	assert.Equal(t, "txn_original", resp.OriginalMerchantOrderID)
}

func TestDevClient_RefundCompletesImmediately(t *testing.T) {
	dev := NewDevClient("http://localhost:3000/checkout/confirmation")

	resp, err := dev.InitiateRefund(context.Background(), "mref_1", "txn_original", 2500)
	require.NoError(t, err)
	assert.Equal(t, StateCompleted, resp.State)
	assert.NotEmpty(t, resp.RefundID, "dev refunds still need an ID for webhook correlation")
	assert.Equal(t, int64(2500), resp.Amount)

	status, err := dev.CheckRefundStatus(context.Background(), "mref_1")
	require.NoError(t, err)
	assert.Equal(t, StateCompleted, status.State)
	assert.Equal(t, resp.RefundID, status.RefundID)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/gateway/phonepe/... -run Refund -v`
Expected: FAIL to compile — `undefined: RefundRequest`.

- [ ] **Step 3: Add the types**

Append to `internal/gateway/phonepe/types.go`:

```go
// --- Refund (Standard Checkout v2) ---

// RefundRequest is the payload for initiating a refund.
type RefundRequest struct {
	MerchantRefundID        string `json:"merchantRefundId"`
	OriginalMerchantOrderID string `json:"originalMerchantOrderId"`
	Amount                  int64  `json:"amount"` // in paise
}

// RefundResponse is the response from the refund initiation API.
type RefundResponse struct {
	RefundID string `json:"refundId"`
	Amount   int64  `json:"amount"`
	State    string `json:"state"` // PENDING
}

// RefundStatusResponse is the response from the refund status API.
type RefundStatusResponse struct {
	OriginalMerchantOrderID string `json:"originalMerchantOrderId"`
	RefundID                string `json:"refundId"`
	Amount                  int64  `json:"amount"`
	State                   string `json:"state"` // COMPLETED, PENDING, FAILED
	ErrorCode               string `json:"errorCode,omitempty"`
	DetailedErrorCode       string `json:"detailedErrorCode,omitempty"`
}

// RefundWebhookPayload is the callback payload for pg.refund.* events.
type RefundWebhookPayload struct {
	Event   string             `json:"event"` // pg.refund.completed, pg.refund.failed
	Payload RefundWebhookOrder `json:"payload"`
}

// RefundWebhookOrder holds the refund details in a webhook callback.
//
// Note the absence of our merchantRefundId: PhonePe does not echo it, so
// correlation is on originalMerchantOrderId plus PhonePe's refundId.
type RefundWebhookOrder struct {
	OriginalMerchantOrderID string `json:"originalMerchantOrderId"`
	RefundID                string `json:"refundId"`
	Amount                  int64  `json:"amount"`
	State                   string `json:"state"` // COMPLETED, FAILED
	Timestamp               int64  `json:"timestamp"`
	ErrorCode               string `json:"errorCode,omitempty"`
	DetailedErrorCode       string `json:"detailedErrorCode,omitempty"`
}
```

- [ ] **Step 4: Add the failed-state constant and the shared request helper**

In `internal/gateway/phonepe/client.go`, extend the const block at the top:

```go
const (
	contentTypeJSON = "application/json"
	// StateCompleted is the PhonePe payment-state value indicating success.
	StateCompleted = "COMPLETED"
	// StateFailed is the PhonePe state value indicating a terminal failure.
	StateFailed = "FAILED"
)
```

Then append to `internal/gateway/phonepe/client.go`:

```go
// doJSON sends an authenticated JSON request to PhonePe and decodes a 200
// response into out. body and out may be nil.
//
// The two payment methods above predate this helper and keep their own
// inlined transport; they are not rewritten here because their error strings
// are asserted by existing tests.
func (c *Client) doJSON(ctx context.Context, method, path string, body, out any) error {
	token, err := c.getToken(ctx)
	if err != nil {
		return err
	}

	var reader io.Reader
	if body != nil {
		raw, marshalErr := json.Marshal(body)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal request: %w", marshalErr)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.config.BaseURL+path, reader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("Authorization", "O-Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call PhonePe: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp PayErrorResponse
		if unmarshalErr := json.Unmarshal(respBody, &errResp); unmarshalErr == nil && errResp.Code != "" {
			return fmt.Errorf("PhonePe %s %s failed: %s - %s", method, path, errResp.Code, errResp.Message)
		}
		return fmt.Errorf("PhonePe %s %s failed (status %d): %s", method, path, resp.StatusCode, string(respBody))
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	return nil
}

// InitiateRefund starts a refund against a completed payment. The refund is
// asynchronous: a successful call returns state PENDING and settlement arrives
// by webhook.
func (c *Client) InitiateRefund(ctx context.Context, merchantRefundID, originalMerchantOrderID string, amount int64) (*RefundResponse, error) {
	reqBody := RefundRequest{
		MerchantRefundID:        merchantRefundID,
		OriginalMerchantOrderID: originalMerchantOrderID,
		Amount:                  amount,
	}

	var resp RefundResponse
	if err := c.doJSON(ctx, http.MethodPost, "/payments/v2/refund", reqBody, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CheckRefundStatus reads the current state of a refund. Keyed on our
// merchantRefundID, which is the recovery path when the initiation response
// was lost and we never stored PhonePe's refundId.
func (c *Client) CheckRefundStatus(ctx context.Context, merchantRefundID string) (*RefundStatusResponse, error) {
	var resp RefundStatusResponse
	path := fmt.Sprintf("/payments/v2/refund/%s/status", merchantRefundID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
```

- [ ] **Step 5: Extend the `Gateway` interface and `DevClient`**

In `internal/gateway/phonepe/dev_client.go`, extend the interface:

```go
// Gateway defines the methods that PaymentService and RefundService use from
// the PhonePe client. Both Client (real) and DevClient (local dev) implement it.
type Gateway interface {
	InitiatePayment(ctx context.Context, merchantTxnID, customerID string, amount int64, orderID string) (string, error)
	CheckPaymentStatus(ctx context.Context, merchantTxnID string) (*StatusResponse, error)
	InitiateRefund(ctx context.Context, merchantRefundID, originalMerchantOrderID string, amount int64) (*RefundResponse, error)
	CheckRefundStatus(ctx context.Context, merchantRefundID string) (*RefundStatusResponse, error)
	VerifyWebhookSignature(username, password, authHeader string) bool
}
```

And append the `DevClient` implementations:

```go
func (d *DevClient) InitiateRefund(_ context.Context, merchantRefundID, originalMerchantOrderID string, amount int64) (*RefundResponse, error) {
	fmt.Printf("\n╔══════════════════════════════════════════════════╗\n")
	fmt.Printf("║  DEV PHONEPE: refund %s  ║\n", merchantRefundID)
	fmt.Printf("║  Original order: %s  ║\n", originalMerchantOrderID)
	fmt.Printf("║  Amount: ₹%.2f                                  ║\n", float64(amount)/100)
	fmt.Printf("╚══════════════════════════════════════════════════╝\n\n")

	return &RefundResponse{
		RefundID: "DEV-REFUND-" + merchantRefundID,
		Amount:   amount,
		State:    StateCompleted,
	}, nil
}

func (d *DevClient) CheckRefundStatus(_ context.Context, merchantRefundID string) (*RefundStatusResponse, error) {
	return &RefundStatusResponse{
		RefundID: "DEV-REFUND-" + merchantRefundID,
		State:    StateCompleted,
	}, nil
}
```

- [ ] **Step 6: Run the test to verify it passes**

Run: `go test ./internal/gateway/phonepe/... -v`
Expected: PASS, including the pre-existing payment tests.

- [ ] **Step 7: Lint and commit**

```bash
golangci-lint run ./internal/gateway/...
git add internal/gateway/phonepe/
git commit -m "feat(phonepe): add refund initiation and status APIs"
```

---

## Task 5: Refund repository and the payment refund counter

**Files:**
- Create: `internal/repository/dynamodb/refund_repository.go`
- Test: `internal/repository/dynamodb/refund_repository_test.go`
- Modify: `internal/domain/store_repository.go:16-22` (`PaymentRepository`)
- Modify: `internal/repository/dynamodb/payment_repository.go` (append before the interface assertion)

**Interfaces:**
- Consumes: `domain.Refund`, `domain.RefundStatus`, `domain.RefundRepository` (Task 1).
- Produces:
  - `dynamodb.NewRefundRepository(client *Client) *RefundRepository` implementing `domain.RefundRepository`
  - `PaymentRepository.AddRefundAmount(ctx context.Context, id string, amountPaise int64) (int64, error)` on the `domain.PaymentRepository` interface, returning the new running total.

- [ ] **Step 1: Write the failing test**

Create `internal/repository/dynamodb/refund_repository_test.go`:

```go
package dynamodb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
	apperrors "github.com/handloom/admin/pkg/errors"
)

func newRefund(id, orderID string, initiatedAt time.Time) *domain.Refund {
	return &domain.Refund{
		ID:               id,
		OrderID:          orderID,
		PaymentID:        "pay_1",
		CustomerID:       "cust_1",
		Amount:           2500,
		Status:           domain.RefundStatusPending,
		Reason:           domain.RefundReasonOutOfStock,
		MerchantRefundID: "mref_" + id,
		Items: []domain.RefundItem{
			{OrderItemID: "oi_1", ProductID: "prod_1", Quantity: 1, Amount: 2500, Restock: false},
		},
		InitiatedAt: initiatedAt,
		CreatedBy:   "admin_1",
	}
}

func TestRefundRepository_CreateAndGet(t *testing.T) {
	wrappedClient, rawClient := testWrappedClient(t)
	skipIfNoLocal(t, rawClient)
	setupTestTable(t, rawClient, testOrdersTable)
	defer cleanupTestTable(t, rawClient, testOrdersTable)

	repo := NewRefundRepository(wrappedClient)
	ctx := context.Background()

	refund := newRefund("refund_1", "order_1", time.Now().UTC())
	require.NoError(t, repo.Create(ctx, refund))

	got, err := repo.GetByID(ctx, "refund_1")
	require.NoError(t, err)
	assert.Equal(t, "order_1", got.OrderID)
	assert.Equal(t, int64(2500), got.Amount)
	assert.Equal(t, domain.RefundStatusPending, got.Status)
	assert.Equal(t, domain.RefundReasonOutOfStock, got.Reason)
	require.Len(t, got.Items, 1)
	assert.Equal(t, "oi_1", got.Items[0].OrderItemID)
	assert.False(t, got.Items[0].Restock)

	_, err = repo.GetByID(ctx, "refund_missing")
	require.Error(t, err)
	assert.True(t, apperrors.IsNotFound(err))
}

func TestRefundRepository_ListByOrderID(t *testing.T) {
	wrappedClient, rawClient := testWrappedClient(t)
	skipIfNoLocal(t, rawClient)
	setupTestTable(t, rawClient, testOrdersTable)
	defer cleanupTestTable(t, rawClient, testOrdersTable)

	repo := NewRefundRepository(wrappedClient)
	ctx := context.Background()

	base := time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)
	require.NoError(t, repo.Create(ctx, newRefund("refund_b", "order_1", base.Add(time.Hour))))
	require.NoError(t, repo.Create(ctx, newRefund("refund_a", "order_1", base)))
	require.NoError(t, repo.Create(ctx, newRefund("refund_other", "order_2", base)))

	got, err := repo.ListByOrderID(ctx, "order_1")
	require.NoError(t, err)
	require.Len(t, got, 2, "must not leak refunds from another order")
	assert.Equal(t, "refund_a", got[0].ID, "oldest first")
	assert.Equal(t, "refund_b", got[1].ID)

	none, err := repo.ListByOrderID(ctx, "order_with_no_refunds")
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestRefundRepository_SettleIfPending_IsTheGate(t *testing.T) {
	wrappedClient, rawClient := testWrappedClient(t)
	skipIfNoLocal(t, rawClient)
	setupTestTable(t, rawClient, testOrdersTable)
	defer cleanupTestTable(t, rawClient, testOrdersTable)

	repo := NewRefundRepository(wrappedClient)
	ctx := context.Background()
	completedAt := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)

	require.NoError(t, repo.Create(ctx, newRefund("refund_1", "order_1", time.Now().UTC())))

	// First delivery wins.
	require.NoError(t, repo.SettleIfPending(ctx, "refund_1", domain.RefundStatusCompleted,
		map[string]interface{}{"completed_at": completedAt}))

	// Second delivery of the same event must lose, not double-apply.
	err := repo.SettleIfPending(ctx, "refund_1", domain.RefundStatusCompleted,
		map[string]interface{}{"completed_at": completedAt})
	require.Error(t, err)
	appErr, ok := apperrors.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrCodeConflict, appErr.Code)

	got, err := repo.GetByID(ctx, "refund_1")
	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusCompleted, got.Status)
	require.NotNil(t, got.CompletedAt)
}

func TestRefundRepository_SetProviderRefundID(t *testing.T) {
	wrappedClient, rawClient := testWrappedClient(t)
	skipIfNoLocal(t, rawClient)
	setupTestTable(t, rawClient, testOrdersTable)
	defer cleanupTestTable(t, rawClient, testOrdersTable)

	repo := NewRefundRepository(wrappedClient)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newRefund("refund_1", "order_1", time.Now().UTC())))
	require.NoError(t, repo.SetProviderRefundID(ctx, "refund_1", "OMR456"))

	got, err := repo.GetByID(ctx, "refund_1")
	require.NoError(t, err)
	assert.Equal(t, "OMR456", got.ProviderRefundID)
	assert.Equal(t, domain.RefundStatusPending, got.Status, "storing the provider ID must not settle the refund")
}

func TestPaymentRepository_AddRefundAmount(t *testing.T) {
	wrappedClient, rawClient := testWrappedClient(t)
	skipIfNoLocal(t, rawClient)
	setupTestTable(t, rawClient, testOrdersTable)
	defer cleanupTestTable(t, rawClient, testOrdersTable)

	repo := NewPaymentRepository(wrappedClient)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &domain.Payment{
		ID:                    "pay_1",
		OrderID:               "order_1",
		CustomerID:            "cust_1",
		Amount:                10000,
		Currency:              "INR",
		Status:                domain.PaymentStatusPaid,
		Provider:              domain.PaymentProviderPhonePe,
		MerchantTransactionID: "txn_1",
		InitiatedAt:           time.Now().UTC(),
	}))

	// ADD initialises an absent attribute to 0, so the first call returns the
	// increment itself.
	total, err := repo.AddRefundAmount(ctx, "pay_1", 2500)
	require.NoError(t, err)
	assert.Equal(t, int64(2500), total)

	// Concurrent settlements of different refunds must not lose an increment,
	// which is why this is ADD and not read-modify-write.
	total, err = repo.AddRefundAmount(ctx, "pay_1", 4000)
	require.NoError(t, err)
	assert.Equal(t, int64(6500), total)

	got, err := repo.GetByID(ctx, "pay_1")
	require.NoError(t, err)
	assert.Equal(t, int64(6500), got.RefundAmount)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/repository/dynamodb/... -run 'TestRefundRepository|TestPaymentRepository_AddRefundAmount' -v`
Expected: FAIL to compile — `undefined: NewRefundRepository`.

Note: these tests need DynamoDB Local. `skipIfNoLocal` skips them when it is not reachable; start it with `make docker-up`.

- [ ] **Step 3: Add `AddRefundAmount` to the payment repository interface**

In `internal/domain/store_repository.go`, extend `PaymentRepository`:

```go
// PaymentRepository defines payment data access operations
type PaymentRepository interface {
	Create(ctx context.Context, payment *Payment) error
	GetByID(ctx context.Context, id string) (*Payment, error)
	GetByOrderID(ctx context.Context, orderID string) (*Payment, error)
	GetByMerchantTxnID(ctx context.Context, merchantTxnID string) (*Payment, error)
	UpdateStatus(ctx context.Context, id string, status PaymentStatus, updates map[string]interface{}) error

	// AddRefundAmount atomically adds amountPaise to the payment's running
	// refund total and returns the new total. DynamoDB ADD rather than
	// read-modify-write, so concurrent settlements of different refunds
	// against the same payment cannot lose an increment.
	AddRefundAmount(ctx context.Context, id string, amountPaise int64) (int64, error)
}
```

- [ ] **Step 4: Implement `AddRefundAmount`**

In `internal/repository/dynamodb/payment_repository.go`, add `"strconv"` to the imports and insert this before the interface assertion at the bottom:

```go
// AddRefundAmount atomically adds amountPaise to refund_amount and returns the
// new total. ADD initialises the attribute to 0 when absent, so the first call
// against a never-refunded payment returns the increment itself.
func (r *PaymentRepository) AddRefundAmount(ctx context.Context, id string, amountPaise int64) (int64, error) {
	out, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "PAYMENT#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
		UpdateExpression: aws.String("ADD refund_amount :amount"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":amount": &types.AttributeValueMemberN{Value: strconv.FormatInt(amountPaise, 10)},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
		ReturnValues:        types.ReturnValueUpdatedNew,
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return 0, errors.NotFound("Payment not found")
		}
		return 0, errors.Wrap(err, "Failed to add refund amount")
	}

	raw, ok := out.Attributes["refund_amount"].(*types.AttributeValueMemberN)
	if !ok {
		return 0, errors.Internal("refund_amount missing from UpdateItem response")
	}
	total, err := strconv.ParseInt(raw.Value, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to parse refund_amount")
	}
	return total, nil
}
```

- [ ] **Step 5: Implement the refund repository**

Create `internal/repository/dynamodb/refund_repository.go`:

```go
package dynamodb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

const entityTypeRefund = "REFUND"

// RefundRepository implements domain.RefundRepository
type RefundRepository struct {
	client *Client
}

// NewRefundRepository creates a new RefundRepository
func NewRefundRepository(client *Client) *RefundRepository {
	return &RefundRepository{client: client}
}

// Create persists a refund. Called before the gateway is contacted: a refund
// that leaves the building without a local record is unreconcilable.
func (r *RefundRepository) Create(ctx context.Context, refund *domain.Refund) error {
	now := time.Now()
	refund.CreatedAt = now
	refund.UpdatedAt = now
	refund.SetKeys()

	av, err := attributevalue.MarshalMap(refund)
	if err != nil {
		return errors.Internal("Failed to marshal refund")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.ordersTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.New(errors.ErrCodeAlreadyExists, "Refund already exists")
		}
		return errors.Wrap(err, "Failed to create refund")
	}

	return nil
}

// GetByID retrieves a refund by ID
func (r *RefundRepository) GetByID(ctx context.Context, id string) (*domain.Refund, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "REFUND#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get refund")
	}

	if result.Item == nil {
		return nil, errors.NotFound("Refund not found")
	}

	var refund domain.Refund
	if err := attributevalue.UnmarshalMap(result.Item, &refund); err != nil {
		return nil, errors.Internal("Failed to unmarshal refund")
	}

	return &refund, nil
}

// ListByOrderID returns every refund for an order, oldest first. GSI1 is shared
// with orders and payments, so results are filtered to REFUND entities.
func (r *RefundRepository) ListByOrderID(ctx context.Context, orderID string) ([]*domain.Refund, error) {
	result, err := r.client.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.client.ordersTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :pk AND begins_with(GSI1SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			exprPK:    &types.AttributeValueMemberS{Value: "ORDER#" + orderID},
			":prefix": &types.AttributeValueMemberS{Value: "REFUND#"},
		},
		ScanIndexForward: aws.Bool(true), // oldest first
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to query refunds by order ID")
	}

	refunds := make([]*domain.Refund, 0, len(result.Items))
	for _, item := range result.Items {
		var refund domain.Refund
		if err := attributevalue.UnmarshalMap(item, &refund); err != nil {
			continue
		}
		if refund.EntityType != entityTypeRefund {
			continue
		}
		refunds = append(refunds, &refund)
	}

	return refunds, nil
}

// SetProviderRefundID records PhonePe's refund ID after initiation returns. It
// deliberately does not touch status — the refund stays PENDING until it settles.
func (r *RefundRepository) SetProviderRefundID(ctx context.Context, id string, providerRefundID string) error {
	_, err := r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "REFUND#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
		UpdateExpression: aws.String("SET provider_refund_id = :prid, updated_at = :now"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":prid": &types.AttributeValueMemberS{Value: providerRefundID},
			exprNow: &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Refund not found")
		}
		return errors.Wrap(err, "Failed to set provider refund ID")
	}
	return nil
}

// SettleIfPending moves a refund from PENDING to a terminal status. The
// ConditionExpression is the authority behind which every settlement effect
// runs: of two concurrent webhook deliveries exactly one succeeds here and the
// other gets ErrCodeConflict, so the payment total, item quantities, order
// status and notification are applied once.
func (r *RefundRepository) SettleIfPending(ctx context.Context, id string, status domain.RefundStatus, updates map[string]interface{}) error {
	du, err := buildDynamicUpdate(string(status), updates)
	if err != nil {
		return err
	}
	du.AttrValues[":pending"] = &types.AttributeValueMemberS{Value: string(domain.RefundStatusPending)}

	_, err = r.client.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.client.ordersTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "REFUND#" + id},
			"SK": &types.AttributeValueMemberS{Value: skMetadata},
		},
		UpdateExpression:          aws.String(du.Expression),
		ExpressionAttributeNames:  du.AttrNames,
		ExpressionAttributeValues: du.AttrValues,
		ConditionExpression:       aws.String("attribute_exists(PK) AND #status = :pending"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.New(errors.ErrCodeConflict, "Refund is not pending")
		}
		return errors.Wrap(err, "Failed to settle refund")
	}
	return nil
}

// Ensure interface compliance
var _ domain.RefundRepository = (*RefundRepository)(nil)
```

- [ ] **Step 6: Regenerate mocks**

Run: `make generate-mocks`
Expected: no error (`PaymentRepository` gained a method).

- [ ] **Step 7: Run the test to verify it passes**

Run: `go test ./internal/repository/dynamodb/... -run 'TestRefundRepository|TestPaymentRepository_AddRefundAmount' -v`
Expected: PASS. If every test SKIPs, DynamoDB Local is not running — start it with `make docker-up` and rerun.

- [ ] **Step 8: Build, lint, commit**

```bash
go build ./... && golangci-lint run ./internal/repository/dynamodb/... ./internal/domain/...
git add internal/domain/store_repository.go \
        internal/repository/dynamodb/refund_repository.go \
        internal/repository/dynamodb/refund_repository_test.go \
        internal/repository/dynamodb/payment_repository.go
git commit -m "feat(refunds): add the refund repository"
```

---

## Task 6: `RefundService.Create`

Persist first, call the gateway second, move stock third. A refund that reaches
PhonePe with no local record is unreconcilable; a PENDING record whose gateway
call never happened is recoverable through the status endpoint.

**Files:**
- Create: `internal/service/refund_service.go`
- Test: `internal/service/refund_service_test.go`

**Interfaces:**
- Consumes: `domain.RefundRepository`, `domain.OrderRepository`, `domain.PaymentRepository`, `domain.InventoryRepository`, `domain.AuditService`, `phonepe.Gateway`, and the Task 3 helpers.
- Produces:
  - `service.NewRefundService(refundRepo, orderRepo, paymentRepo, inventoryRepo, auditService, phonePe) *RefundService`
  - `(*RefundService).Create(ctx context.Context, orderID string, req domain.CreateRefundRequest, createdBy string) (*domain.Refund, error)`
  - `(*RefundService).ListByOrder(ctx context.Context, orderID string) ([]*domain.Refund, error)`
  - unexported: `resolveRefundLines`, `applyRefundInventoryEffect`

Note: `NewRefundService` gains one more parameter in Task 7 (the notifier). The
interface assertion `var _ domain.RefundService` is added in Task 7, once the
settlement methods exist.

- [ ] **Step 1: Write the failing test**

Create `internal/service/refund_service_test.go`:

```go
package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/phonepe"
	"github.com/handloom/admin/internal/mocks"
	apperrors "github.com/handloom/admin/pkg/errors"
)

// fakeGateway is a hand-written phonepe.Gateway double. The gateway interface
// lives outside internal/domain, so there is no generated mock for it.
type fakeGateway struct {
	refundResp *phonepe.RefundResponse
	refundErr  error
	statusResp *phonepe.RefundStatusResponse
	statusErr  error

	gotMerchantRefundID string
	gotOriginalOrderID  string
	gotAmount           int64
	refundCalls         int
}

func (f *fakeGateway) InitiatePayment(context.Context, string, string, int64, string) (string, error) {
	return "", nil
}

func (f *fakeGateway) CheckPaymentStatus(context.Context, string) (*phonepe.StatusResponse, error) {
	return nil, nil
}

func (f *fakeGateway) InitiateRefund(_ context.Context, merchantRefundID, originalMerchantOrderID string, amount int64) (*phonepe.RefundResponse, error) {
	f.refundCalls++
	f.gotMerchantRefundID = merchantRefundID
	f.gotOriginalOrderID = originalMerchantOrderID
	f.gotAmount = amount
	return f.refundResp, f.refundErr
}

func (f *fakeGateway) CheckRefundStatus(context.Context, string) (*phonepe.RefundStatusResponse, error) {
	return f.statusResp, f.statusErr
}

func (f *fakeGateway) VerifyWebhookSignature(string, string, string) bool { return true }

type refundFixture struct {
	svc        *RefundService
	refundRepo *mocks.MockRefundRepository
	orderRepo  *mocks.MockOrderRepository
	payRepo    *mocks.MockPaymentRepository
	invRepo    *mocks.MockInventoryRepository
	audit      *mocks.MockAuditService
	gateway    *fakeGateway
}

func newRefundFixture(t *testing.T) *refundFixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	f := &refundFixture{
		refundRepo: mocks.NewMockRefundRepository(ctrl),
		orderRepo:  mocks.NewMockOrderRepository(ctrl),
		payRepo:    mocks.NewMockPaymentRepository(ctrl),
		invRepo:    mocks.NewMockInventoryRepository(ctrl),
		audit:      mocks.NewMockAuditService(ctrl),
		gateway: &fakeGateway{
			refundResp: &phonepe.RefundResponse{RefundID: "OMR456", Amount: 27000, State: "PENDING"},
		},
	}
	f.svc = NewRefundService(f.refundRepo, f.orderRepo, f.payRepo, f.invRepo, f.audit, f.gateway)
	return f
}

// testOrder: ₹1000 subtotal, ₹100 discount, ₹50 shipping, ₹950 total.
func testOrder(status domain.OrderStatus) *domain.Order {
	return &domain.Order{
		ID:         "order_1",
		CustomerID: "cust_1",
		Status:     status,
		Items: []domain.OrderItem{
			{ID: "oi_1", ProductID: "prod_1", UnitPrice: 30000, Quantity: 2, TotalPrice: 60000},
			{ID: "oi_2", ProductID: "prod_2", UnitPrice: 40000, Quantity: 1, TotalPrice: 40000},
		},
		Subtotal:       100000,
		DiscountAmount: 10000,
		ShippingAmount: 5000,
		TotalAmount:    95000,
		PaymentStatus:  domain.PaymentStatusPaid,
	}
}

func testPayment() *domain.Payment {
	return &domain.Payment{
		ID:                    "pay_1",
		OrderID:               "order_1",
		CustomerID:            "cust_1",
		Amount:                95000,
		Status:                domain.PaymentStatusPaid,
		MerchantTransactionID: "txn_original",
	}
}

func oneLineRequest(restock bool) domain.CreateRefundRequest {
	return domain.CreateRefundRequest{
		Reason: domain.RefundReasonOutOfStock,
		Items:  []domain.RefundItemRequest{{OrderItemID: "oi_1", Quantity: 1, Restock: restock}},
	}
}

func TestRefundService_Create_HappyPath(t *testing.T) {
	f := newRefundFixture(t)
	ctx := context.Background()

	f.orderRepo.EXPECT().GetByID(ctx, "order_1").Return(testOrder(domain.OrderStatusConfirmed), nil)
	f.payRepo.EXPECT().GetByOrderID(ctx, "order_1").Return(testPayment(), nil)
	f.refundRepo.EXPECT().ListByOrderID(ctx, "order_1").Return(nil, nil)

	var persisted *domain.Refund
	f.refundRepo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, r *domain.Refund) error {
			persisted = r
			return nil
		})
	f.refundRepo.EXPECT().SetProviderRefundID(ctx, gomock.Any(), "OMR456").Return(nil)
	f.invRepo.EXPECT().WriteOffStock(ctx, "prod_1", 1, "order_1").Return(nil, nil)
	f.audit.EXPECT().Log(ctx, gomock.Any(), "ORDER", "order_1", "admin_1", gomock.Any(), gomock.Any()).Return(nil)

	refund, err := f.svc.Create(ctx, "order_1", oneLineRequest(false), "admin_1")
	require.NoError(t, err)

	// 30000 line − prorated discount 3000 = 27000. Shipping retained.
	assert.Equal(t, int64(27000), refund.Amount)
	assert.Equal(t, domain.RefundStatusPending, refund.Status)
	assert.Equal(t, "pay_1", refund.PaymentID)
	assert.Equal(t, "OMR456", refund.ProviderRefundID)
	require.Len(t, refund.Items, 1)
	assert.Equal(t, int64(27000), refund.Items[0].Amount)
	assert.Equal(t, "prod_1", refund.Items[0].ProductID)

	// Persisted as PENDING before the gateway was called.
	require.NotNil(t, persisted)
	assert.Equal(t, domain.RefundStatusPending, persisted.Status)
	assert.NotEmpty(t, persisted.MerchantRefundID)

	assert.Equal(t, 1, f.gateway.refundCalls)
	assert.Equal(t, "txn_original", f.gateway.gotOriginalOrderID,
		"PhonePe correlates on the payment's merchant transaction ID")
	assert.Equal(t, int64(27000), f.gateway.gotAmount)
}

func TestRefundService_Create_GatewayFailureMarksFailedAndSkipsInventory(t *testing.T) {
	f := newRefundFixture(t)
	f.gateway.refundResp = nil
	f.gateway.refundErr = errors.New("connection reset")
	ctx := context.Background()

	f.orderRepo.EXPECT().GetByID(ctx, "order_1").Return(testOrder(domain.OrderStatusConfirmed), nil)
	f.payRepo.EXPECT().GetByOrderID(ctx, "order_1").Return(testPayment(), nil)
	f.refundRepo.EXPECT().ListByOrderID(ctx, "order_1").Return(nil, nil)
	f.refundRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)
	f.refundRepo.EXPECT().SettleIfPending(ctx, gomock.Any(), domain.RefundStatusFailed, gomock.Any()).Return(nil)
	// No SetProviderRefundID, no inventory call, no audit.

	_, err := f.svc.Create(ctx, "order_1", oneLineRequest(false), "admin_1")
	require.Error(t, err)
}

func TestRefundService_Create_RestockReleasesInsteadOfWritingOff(t *testing.T) {
	f := newRefundFixture(t)
	ctx := context.Background()

	f.orderRepo.EXPECT().GetByID(ctx, "order_1").Return(testOrder(domain.OrderStatusProcessing), nil)
	f.payRepo.EXPECT().GetByOrderID(ctx, "order_1").Return(testPayment(), nil)
	f.refundRepo.EXPECT().ListByOrderID(ctx, "order_1").Return(nil, nil)
	f.refundRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)
	f.refundRepo.EXPECT().SetProviderRefundID(ctx, gomock.Any(), "OMR456").Return(nil)
	f.invRepo.EXPECT().ReleaseStock(ctx, "prod_1", 1, "order_1").Return(nil, nil)
	f.audit.EXPECT().Log(ctx, gomock.Any(), "ORDER", "order_1", "admin_1", gomock.Any(), gomock.Any()).Return(nil)

	_, err := f.svc.Create(ctx, "order_1", oneLineRequest(true), "admin_1")
	require.NoError(t, err)
}

func TestRefundService_Create_PostDispatchMovesNoStock(t *testing.T) {
	for _, status := range []domain.OrderStatus{domain.OrderStatusShipped, domain.OrderStatusDelivered} {
		t.Run(string(status), func(t *testing.T) {
			f := newRefundFixture(t)
			ctx := context.Background()

			f.orderRepo.EXPECT().GetByID(ctx, "order_1").Return(testOrder(status), nil)
			f.payRepo.EXPECT().GetByOrderID(ctx, "order_1").Return(testPayment(), nil)
			f.refundRepo.EXPECT().ListByOrderID(ctx, "order_1").Return(nil, nil)
			f.refundRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)
			f.refundRepo.EXPECT().SetProviderRefundID(ctx, gomock.Any(), "OMR456").Return(nil)
			f.audit.EXPECT().Log(ctx, gomock.Any(), "ORDER", "order_1", "admin_1", gomock.Any(), gomock.Any()).Return(nil)
			// RETURNED owns post-dispatch restocking: no inventory call here.

			_, err := f.svc.Create(ctx, "order_1", oneLineRequest(true), "admin_1")
			require.NoError(t, err)
		})
	}
}

func TestRefundService_Create_RejectsUnpaidOrder(t *testing.T) {
	f := newRefundFixture(t)
	ctx := context.Background()

	payment := testPayment()
	payment.Status = domain.PaymentStatusPending

	f.orderRepo.EXPECT().GetByID(ctx, "order_1").Return(testOrder(domain.OrderStatusPending), nil)
	f.payRepo.EXPECT().GetByOrderID(ctx, "order_1").Return(payment, nil)

	_, err := f.svc.Create(ctx, "order_1", oneLineRequest(false), "admin_1")
	require.Error(t, err)
	appErr, ok := apperrors.AsAppError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrCodeValidation, appErr.Code)
	assert.Equal(t, 0, f.gateway.refundCalls)
}

func TestRefundService_Create_RejectsQuantityBeyondRemainder(t *testing.T) {
	f := newRefundFixture(t)
	ctx := context.Background()

	order := testOrder(domain.OrderStatusConfirmed)
	order.Items[0].RefundedQuantity = 1 // one of the two units already refunded

	f.orderRepo.EXPECT().GetByID(ctx, "order_1").Return(order, nil)
	f.payRepo.EXPECT().GetByOrderID(ctx, "order_1").Return(testPayment(), nil)
	f.refundRepo.EXPECT().ListByOrderID(ctx, "order_1").Return(nil, nil)

	req := domain.CreateRefundRequest{
		Reason: domain.RefundReasonDamaged,
		Items:  []domain.RefundItemRequest{{OrderItemID: "oi_1", Quantity: 2}},
	}

	_, err := f.svc.Create(ctx, "order_1", req, "admin_1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")
	assert.Equal(t, 0, f.gateway.refundCalls)
}

func TestRefundService_Create_CountsPendingRefundsAgainstTheRemainder(t *testing.T) {
	f := newRefundFixture(t)
	ctx := context.Background()

	f.orderRepo.EXPECT().GetByID(ctx, "order_1").Return(testOrder(domain.OrderStatusConfirmed), nil)
	f.payRepo.EXPECT().GetByOrderID(ctx, "order_1").Return(testPayment(), nil)
	// Both units of oi_1 are already held by an in-flight refund.
	f.refundRepo.EXPECT().ListByOrderID(ctx, "order_1").Return([]*domain.Refund{
		{
			ID:     "refund_prior",
			Status: domain.RefundStatusPending,
			Amount: 54000,
			Items:  []domain.RefundItem{{OrderItemID: "oi_1", Quantity: 2}},
		},
	}, nil)

	_, err := f.svc.Create(ctx, "order_1", oneLineRequest(false), "admin_1")
	require.Error(t, err)
	assert.Equal(t, 0, f.gateway.refundCalls)
}

func TestRefundService_Create_RejectsUnknownOrderItem(t *testing.T) {
	f := newRefundFixture(t)
	ctx := context.Background()

	f.orderRepo.EXPECT().GetByID(ctx, "order_1").Return(testOrder(domain.OrderStatusConfirmed), nil)
	f.payRepo.EXPECT().GetByOrderID(ctx, "order_1").Return(testPayment(), nil)
	f.refundRepo.EXPECT().ListByOrderID(ctx, "order_1").Return(nil, nil)

	req := domain.CreateRefundRequest{
		Reason: domain.RefundReasonOther,
		Items:  []domain.RefundItemRequest{{OrderItemID: "oi_not_on_this_order", Quantity: 1}},
	}

	_, err := f.svc.Create(ctx, "order_1", req, "admin_1")
	require.Error(t, err)
	assert.Equal(t, 0, f.gateway.refundCalls)
}

func TestRefundService_ListByOrder(t *testing.T) {
	f := newRefundFixture(t)
	ctx := context.Background()

	want := []*domain.Refund{{ID: "refund_1"}}
	f.refundRepo.EXPECT().ListByOrderID(ctx, "order_1").Return(want, nil)

	got, err := f.svc.ListByOrder(ctx, "order_1")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/... -run TestRefundService -v`
Expected: FAIL to compile — `undefined: NewRefundService`.

- [ ] **Step 3: Implement the service**

Create `internal/service/refund_service.go`:

```go
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/phonepe"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/metrics"
)

// auditActionRefundInitiated is the audit action recorded when an admin starts
// a refund.
const auditActionRefundInitiated = "REFUND_INITIATED"

// RefundService implements domain.RefundService.
//
// Refunds are their own concern with their own lifecycle. Folding them into
// OrderService — already the largest service in the codebase — would make it
// worse.
type RefundService struct {
	refundRepo    domain.RefundRepository
	orderRepo     domain.OrderRepository
	paymentRepo   domain.PaymentRepository
	inventoryRepo domain.InventoryRepository
	auditService  domain.AuditService
	phonePe       phonepe.Gateway
}

// NewRefundService creates a new RefundService.
func NewRefundService(
	refundRepo domain.RefundRepository,
	orderRepo domain.OrderRepository,
	paymentRepo domain.PaymentRepository,
	inventoryRepo domain.InventoryRepository,
	auditService domain.AuditService,
	phonePe phonepe.Gateway,
) *RefundService {
	return &RefundService{
		refundRepo:    refundRepo,
		orderRepo:     orderRepo,
		paymentRepo:   paymentRepo,
		inventoryRepo: inventoryRepo,
		auditService:  auditService,
		phonePe:       phonePe,
	}
}

// Create validates the requested lines, derives the amount, persists a PENDING
// refund, initiates it with PhonePe and applies the inventory effect.
//
// Ordering matters: the record is persisted before the gateway is called. A
// refund that leaves the building with no local record cannot be reconciled;
// a PENDING record whose gateway call never happened is recovered through
// RecheckStatus.
func (s *RefundService) Create(ctx context.Context, orderID string, req domain.CreateRefundRequest, createdBy string) (*domain.Refund, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	payment, err := s.paymentRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if payment.Status != domain.PaymentStatusPaid && payment.Status != domain.PaymentStatusSuccess {
		return nil, errors.Validation("Order has no successful payment to refund")
	}

	existing, err := s.refundRepo.ListByOrderID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	alreadyQty := refundedQuantities(order, existing)
	lines, err := resolveRefundLines(order, req.Items, alreadyQty)
	if err != nil {
		return nil, err
	}

	alreadyAmount := openRefundTotal(existing)
	amount := computeRefundAmount(order, lines, alreadyQty, alreadyAmount)
	if amount <= 0 {
		return nil, errors.Validation("Refund amount must be greater than zero")
	}
	if alreadyAmount+amount > order.TotalAmount {
		return nil, errors.Validation("Refund would exceed the order total")
	}

	lineAmounts := distributeRefundAmount(order, lines, amount)
	items := make([]domain.RefundItem, len(lines))
	for i, l := range lines {
		items[i] = domain.RefundItem{
			OrderItemID: l.item.ID,
			ProductID:   l.item.ProductID,
			Quantity:    l.quantity,
			Amount:      lineAmounts[i],
			Restock:     req.Items[i].Restock,
		}
	}

	refund := &domain.Refund{
		ID:               "refund_" + uuid.New().String()[:8],
		OrderID:          order.ID,
		PaymentID:        payment.ID,
		CustomerID:       order.CustomerID,
		Amount:           amount,
		Status:           domain.RefundStatusPending,
		Reason:           req.Reason,
		Items:            items,
		MerchantRefundID: "mref_" + uuid.New().String(),
		InitiatedAt:      time.Now(),
		CreatedBy:        createdBy,
	}

	if err := s.refundRepo.Create(ctx, refund); err != nil {
		return nil, err
	}

	// originalMerchantOrderID is the merchantOrderId we sent at checkout.
	resp, err := s.phonePe.InitiateRefund(ctx, refund.MerchantRefundID, payment.MerchantTransactionID, refund.Amount)
	if err != nil {
		slog.ErrorContext(ctx, "Refund initiation failed at the gateway",
			"refund_id", refund.ID, "order_id", order.ID, "error", err)
		if settleErr := s.refundRepo.SettleIfPending(ctx, refund.ID, domain.RefundStatusFailed,
			map[string]interface{}{"error_code": "GATEWAY_ERROR"}); settleErr != nil {
			slog.ErrorContext(ctx, "Failed to mark refund failed", "refund_id", refund.ID, "error", settleErr)
		}
		metrics.Record(ctx, "refund_failed", metrics.L{
			metrics.LabelReason:  string(refund.Reason),
			metrics.LabelGateway: gatewayPhonePe,
		})
		return nil, errors.Wrap(err, "Failed to initiate refund with the payment provider")
	}

	refund.ProviderRefundID = resp.RefundID
	if err := s.refundRepo.SetProviderRefundID(ctx, refund.ID, resp.RefundID); err != nil {
		// The money is already moving. Losing the provider ID costs webhook
		// correlation, not the refund — RecheckStatus recovers it.
		slog.ErrorContext(ctx, "Failed to store provider refund ID",
			"refund_id", refund.ID, "provider_refund_id", resp.RefundID, "error", err)
	}

	s.applyRefundInventoryEffect(ctx, order, refund)

	if err := s.auditService.Log(ctx, auditActionRefundInitiated, "ORDER", order.ID, createdBy, nil,
		map[string]interface{}{
			"refund_id": refund.ID,
			"amount":    refund.Amount,
			"reason":    string(refund.Reason),
			"items":     refund.Items,
		}); err != nil {
		slog.ErrorContext(ctx, "Failed to write refund audit log", "refund_id", refund.ID, "error", err)
	}

	metrics.Record(ctx, "refund_initiated", metrics.L{
		metrics.LabelReason:  string(refund.Reason),
		metrics.LabelGateway: gatewayPhonePe,
	})

	return refund, nil
}

// ListByOrder returns every refund recorded against an order, oldest first.
func (s *RefundService) ListByOrder(ctx context.Context, orderID string) ([]*domain.Refund, error) {
	return s.refundRepo.ListByOrderID(ctx, orderID)
}

// resolveRefundLines matches each requested line to an order item and checks it
// against the units not already spoken for. The returned slice is parallel to
// req, so the caller can read Restock back by index.
func resolveRefundLines(order *domain.Order, req []domain.RefundItemRequest, alreadyRefundedQty map[string]int) ([]refundLine, error) {
	byID := make(map[string]domain.OrderItem, len(order.Items))
	for _, it := range order.Items {
		byID[it.ID] = it
	}

	lines := make([]refundLine, 0, len(req))
	requested := make(map[string]int, len(req))

	for _, r := range req {
		item, ok := byID[r.OrderItemID]
		if !ok {
			return nil, errors.Validation(fmt.Sprintf("Order item not found on this order: %s", r.OrderItemID))
		}
		requested[r.OrderItemID] += r.Quantity

		remaining := item.Quantity - alreadyRefundedQty[r.OrderItemID]
		if requested[r.OrderItemID] > remaining {
			return nil, errors.Validation(fmt.Sprintf(
				"Refund quantity for %s exceeds the %d unrefunded unit(s) remaining", item.ProductName, remaining))
		}

		lines = append(lines, refundLine{item: item, quantity: r.Quantity})
	}

	return lines, nil
}

// applyRefundInventoryEffect moves stock for a refund on an order that has not
// been dispatched. Dispatch is the dividing line because CommitStock consumes
// the reservation at SHIPPED; letting a refund also restock after that would
// double-count against the RETURNED transition, which owns post-dispatch
// restocking.
//
// The pre-dispatch statuses are whitelisted rather than the post-dispatch ones
// blacklisted: CANCELLED already released its reservation and RETURNED already
// restocked, so a new status defaulting to "no stock movement" is the safe way
// to be wrong.
//
// Failures are logged and counted, not returned: the refund itself has already
// been accepted by the gateway.
func (s *RefundService) applyRefundInventoryEffect(ctx context.Context, order *domain.Order, refund *domain.Refund) {
	switch order.Status {
	case domain.OrderStatusPending, domain.OrderStatusConfirmed, domain.OrderStatusProcessing:
	default:
		return
	}

	for _, item := range refund.Items {
		var err error
		if item.Restock {
			_, err = s.inventoryRepo.ReleaseStock(ctx, item.ProductID, item.Quantity, order.ID)
		} else {
			_, err = s.inventoryRepo.WriteOffStock(ctx, item.ProductID, item.Quantity, order.ID)
		}
		if err != nil {
			slog.ErrorContext(ctx, "Failed to apply refund inventory effect",
				keyProductID, item.ProductID, "restock", item.Restock,
				"refund_id", refund.ID, "error", err)
			metrics.Record(ctx, "inventory_mutation_failed", metrics.L{metrics.LabelReason: "refund"})
		}
	}
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/service/... -run TestRefundService -v`
Expected: PASS, every subtest.

- [ ] **Step 5: Run the full service suite for regressions**

Run: `go test ./internal/service/...`
Expected: `ok`.

- [ ] **Step 6: Lint and commit**

```bash
golangci-lint run ./internal/service/...
git add internal/service/refund_service.go internal/service/refund_service_test.go \
        internal/service/refund_amount.go internal/service/refund_amount_test.go
git commit -m "feat(refunds): create refunds against PhonePe"
```

---

## Task 7: Settlement and re-check

Idempotency is built in from the start rather than retrofitted. PhonePe retries
webhooks and Lambda can process two deliveries concurrently, so the refund
record's own conditional update is the single gate every downstream effect runs
behind.

**Files:**
- Modify: `internal/service/refund_service.go` (add the notifier dependency and the settlement methods)
- Modify: `internal/service/refund_service_test.go` (fixture gains the notifier; new tests)

**Interfaces:**
- Consumes: Task 6's `RefundService`, `domain.RefundWebhookEvent`, `phonepe.StateCompleted` / `phonepe.StateFailed`.
- Produces:
  - `NewRefundService(refundRepo, orderRepo, paymentRepo, inventoryRepo, auditService, notifier, phonePe)` — **one new parameter**, `notifier orderNotifier`, inserted before `phonePe`
  - `type orderNotifier interface { SendOrderNotification(ctx context.Context, order *domain.Order, trigger domain.NotificationTrigger, createdBy string) error }`
  - `(*RefundService).HandleRefundCompleted(ctx, domain.RefundWebhookEvent) error`
  - `(*RefundService).HandleRefundFailed(ctx, domain.RefundWebhookEvent) error`
  - `(*RefundService).RecheckStatus(ctx, refundID string) (*domain.Refund, error)`
  - `var _ domain.RefundService = (*RefundService)(nil)`

- [ ] **Step 1: Write the failing test**

Append to `internal/service/refund_service_test.go`:

```go
// fakeNotifier records refund notifications instead of persisting them.
type fakeNotifier struct {
	calls    int
	lastFor  string
	lastWith domain.NotificationTrigger
	err      error
}

func (f *fakeNotifier) SendOrderNotification(_ context.Context, order *domain.Order, trigger domain.NotificationTrigger, _ string) error {
	f.calls++
	f.lastFor = order.ID
	f.lastWith = trigger
	return f.err
}

func settledRefund(status domain.RefundStatus) *domain.Refund {
	return &domain.Refund{
		ID:               "refund_1",
		OrderID:          "order_1",
		PaymentID:        "pay_1",
		CustomerID:       "cust_1",
		Amount:           27000,
		Status:           status,
		Reason:           domain.RefundReasonOutOfStock,
		MerchantRefundID: "mref_1",
		ProviderRefundID: "OMR456",
		Items:            []domain.RefundItem{{OrderItemID: "oi_1", ProductID: "prod_1", Quantity: 1, Amount: 27000}},
	}
}

func completedEvent() domain.RefundWebhookEvent {
	return domain.RefundWebhookEvent{
		OriginalMerchantOrderID: "txn_original",
		ProviderRefundID:        "OMR456",
		Amount:                  27000,
	}
}

func TestRefundService_HandleRefundCompleted_PartialRefund(t *testing.T) {
	f := newRefundFixture(t)
	ctx := context.Background()

	f.payRepo.EXPECT().GetByMerchantTxnID(ctx, "txn_original").Return(testPayment(), nil)
	f.refundRepo.EXPECT().ListByOrderID(ctx, "order_1").Return([]*domain.Refund{settledRefund(domain.RefundStatusPending)}, nil)
	f.refundRepo.EXPECT().SettleIfPending(ctx, "refund_1", domain.RefundStatusCompleted, gomock.Any()).Return(nil)
	f.payRepo.EXPECT().AddRefundAmount(ctx, "pay_1", int64(27000)).Return(int64(27000), nil)
	f.orderRepo.EXPECT().GetByID(ctx, "order_1").Return(testOrder(domain.OrderStatusProcessing), nil)

	var updated *domain.Order
	f.orderRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, o *domain.Order) error {
			updated = o
			return nil
		})

	require.NoError(t, f.svc.HandleRefundCompleted(ctx, completedEvent()))

	require.NotNil(t, updated)
	assert.Equal(t, domain.PaymentStatusPartiallyRefunded, updated.PaymentStatus)
	assert.Equal(t, domain.OrderStatusProcessing, updated.Status,
		"fulfilment status must not change — the remainder still ships")
	assert.Equal(t, 1, updated.Items[0].RefundedQuantity)
	assert.Equal(t, 0, updated.Items[1].RefundedQuantity)

	assert.Equal(t, 1, f.notifier.calls)
	assert.Equal(t, domain.NotificationTriggerRefund, f.notifier.lastWith)
}

func TestRefundService_HandleRefundCompleted_FullRefund(t *testing.T) {
	f := newRefundFixture(t)
	ctx := context.Background()

	full := settledRefund(domain.RefundStatusPending)
	full.Amount = 95000

	f.payRepo.EXPECT().GetByMerchantTxnID(ctx, "txn_original").Return(testPayment(), nil)
	f.refundRepo.EXPECT().ListByOrderID(ctx, "order_1").Return([]*domain.Refund{full}, nil)
	f.refundRepo.EXPECT().SettleIfPending(ctx, "refund_1", domain.RefundStatusCompleted, gomock.Any()).Return(nil)
	f.payRepo.EXPECT().AddRefundAmount(ctx, "pay_1", int64(95000)).Return(int64(95000), nil)
	f.orderRepo.EXPECT().GetByID(ctx, "order_1").Return(testOrder(domain.OrderStatusProcessing), nil)

	var updated *domain.Order
	f.orderRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, o *domain.Order) error {
			updated = o
			return nil
		})
	f.payRepo.EXPECT().UpdateStatus(ctx, "pay_1", domain.PaymentStatusRefunded, gomock.Any()).Return(nil)

	require.NoError(t, f.svc.HandleRefundCompleted(ctx, completedEvent()))
	require.NotNil(t, updated)
	assert.Equal(t, domain.PaymentStatusRefunded, updated.PaymentStatus)
}

func TestRefundService_HandleRefundCompleted_DuplicateDeliveryAppliesOnce(t *testing.T) {
	f := newRefundFixture(t)
	ctx := context.Background()

	f.payRepo.EXPECT().GetByMerchantTxnID(ctx, "txn_original").Return(testPayment(), nil)
	f.refundRepo.EXPECT().ListByOrderID(ctx, "order_1").Return([]*domain.Refund{settledRefund(domain.RefundStatusPending)}, nil)
	// The loser of the race: the conditional update fails.
	f.refundRepo.EXPECT().SettleIfPending(ctx, "refund_1", domain.RefundStatusCompleted, gomock.Any()).
		Return(apperrors.New(apperrors.ErrCodeConflict, "Refund is not pending"))
	// No AddRefundAmount, no order Update, no notification — gomock fails the
	// test if any of them is called.

	require.NoError(t, f.svc.HandleRefundCompleted(ctx, completedEvent()),
		"a losing duplicate is a no-op, not an error")
	assert.Equal(t, 0, f.notifier.calls)
}

func TestRefundService_HandleRefundCompleted_AlreadyTerminalIsANoOp(t *testing.T) {
	f := newRefundFixture(t)
	ctx := context.Background()

	f.payRepo.EXPECT().GetByMerchantTxnID(ctx, "txn_original").Return(testPayment(), nil)
	f.refundRepo.EXPECT().ListByOrderID(ctx, "order_1").Return([]*domain.Refund{settledRefund(domain.RefundStatusCompleted)}, nil)
	// Not even SettleIfPending: the read-side check short-circuits first.

	require.NoError(t, f.svc.HandleRefundCompleted(ctx, completedEvent()))
	assert.Equal(t, 0, f.notifier.calls)
}

func TestRefundService_HandleRefundCompleted_CorrelatesOnProviderRefundID(t *testing.T) {
	f := newRefundFixture(t)
	ctx := context.Background()

	other := settledRefund(domain.RefundStatusPending)
	other.ID = "refund_other"
	other.ProviderRefundID = "OMR_SOMETHING_ELSE"

	target := settledRefund(domain.RefundStatusPending)

	f.payRepo.EXPECT().GetByMerchantTxnID(ctx, "txn_original").Return(testPayment(), nil)
	f.refundRepo.EXPECT().ListByOrderID(ctx, "order_1").Return([]*domain.Refund{other, target}, nil)
	f.refundRepo.EXPECT().SettleIfPending(ctx, "refund_1", domain.RefundStatusCompleted, gomock.Any()).Return(nil)
	f.payRepo.EXPECT().AddRefundAmount(ctx, "pay_1", int64(27000)).Return(int64(27000), nil)
	f.orderRepo.EXPECT().GetByID(ctx, "order_1").Return(testOrder(domain.OrderStatusProcessing), nil)
	f.orderRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

	require.NoError(t, f.svc.HandleRefundCompleted(ctx, completedEvent()))
}

func TestRefundService_HandleRefundFailed(t *testing.T) {
	f := newRefundFixture(t)
	ctx := context.Background()

	f.payRepo.EXPECT().GetByMerchantTxnID(ctx, "txn_original").Return(testPayment(), nil)
	f.refundRepo.EXPECT().ListByOrderID(ctx, "order_1").Return([]*domain.Refund{settledRefund(domain.RefundStatusPending)}, nil)

	var updates map[string]interface{}
	f.refundRepo.EXPECT().SettleIfPending(ctx, "refund_1", domain.RefundStatusFailed, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, _ domain.RefundStatus, u map[string]interface{}) error {
			updates = u
			return nil
		})
	// Order and payment are deliberately untouched, and inventory is NOT
	// reversed — see the accepted gaps in the design doc.

	evt := completedEvent()
	evt.ErrorCode = "REFUND_DECLINED"
	evt.DetailedErrorCode = "INSUFFICIENT_MERCHANT_BALANCE"

	require.NoError(t, f.svc.HandleRefundFailed(ctx, evt))
	assert.Equal(t, "REFUND_DECLINED", updates["error_code"])
	assert.Equal(t, "INSUFFICIENT_MERCHANT_BALANCE", updates["detailed_error_code"])
	assert.Equal(t, 0, f.notifier.calls)
}

func TestRefundService_RecheckStatus_SettlesAndRecoversProviderID(t *testing.T) {
	f := newRefundFixture(t)
	ctx := context.Background()

	// The initiation response was lost, so ProviderRefundID was never stored.
	lost := settledRefund(domain.RefundStatusPending)
	lost.ProviderRefundID = ""

	f.gateway.statusResp = &phonepe.RefundStatusResponse{
		OriginalMerchantOrderID: "txn_original",
		RefundID:                "OMR456",
		Amount:                  27000,
		State:                   phonepe.StateCompleted,
	}

	f.refundRepo.EXPECT().GetByID(ctx, "refund_1").Return(lost, nil)
	f.refundRepo.EXPECT().SetProviderRefundID(ctx, "refund_1", "OMR456").Return(nil)
	f.refundRepo.EXPECT().SettleIfPending(ctx, "refund_1", domain.RefundStatusCompleted, gomock.Any()).Return(nil)
	f.payRepo.EXPECT().AddRefundAmount(ctx, "pay_1", int64(27000)).Return(int64(27000), nil)
	f.orderRepo.EXPECT().GetByID(ctx, "order_1").Return(testOrder(domain.OrderStatusProcessing), nil)
	f.orderRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
	f.refundRepo.EXPECT().GetByID(ctx, "refund_1").Return(settledRefund(domain.RefundStatusCompleted), nil)

	got, err := f.svc.RecheckStatus(ctx, "refund_1")
	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusCompleted, got.Status)
}

func TestRefundService_RecheckStatus_TerminalRefundIsNotReprocessed(t *testing.T) {
	f := newRefundFixture(t)
	ctx := context.Background()

	f.refundRepo.EXPECT().GetByID(ctx, "refund_1").Return(settledRefund(domain.RefundStatusCompleted), nil)
	// The gateway is not called at all.

	got, err := f.svc.RecheckStatus(ctx, "refund_1")
	require.NoError(t, err)
	assert.Equal(t, domain.RefundStatusCompleted, got.Status)
}
```

Then update `refundFixture` and `newRefundFixture` in the same file to carry the notifier:

```go
type refundFixture struct {
	svc        *RefundService
	refundRepo *mocks.MockRefundRepository
	orderRepo  *mocks.MockOrderRepository
	payRepo    *mocks.MockPaymentRepository
	invRepo    *mocks.MockInventoryRepository
	audit      *mocks.MockAuditService
	notifier   *fakeNotifier
	gateway    *fakeGateway
}

func newRefundFixture(t *testing.T) *refundFixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	f := &refundFixture{
		refundRepo: mocks.NewMockRefundRepository(ctrl),
		orderRepo:  mocks.NewMockOrderRepository(ctrl),
		payRepo:    mocks.NewMockPaymentRepository(ctrl),
		invRepo:    mocks.NewMockInventoryRepository(ctrl),
		audit:      mocks.NewMockAuditService(ctrl),
		notifier:   &fakeNotifier{},
		gateway: &fakeGateway{
			refundResp: &phonepe.RefundResponse{RefundID: "OMR456", Amount: 27000, State: "PENDING"},
		},
	}
	f.svc = NewRefundService(f.refundRepo, f.orderRepo, f.payRepo, f.invRepo, f.audit, f.notifier, f.gateway)
	return f
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/... -run TestRefundService -v`
Expected: FAIL to compile — too many arguments to `NewRefundService`, `undefined: f.svc.HandleRefundCompleted`.

- [ ] **Step 3: Add the notifier dependency**

In `internal/service/refund_service.go`, add the interface above the struct, add the field, and add the constructor parameter:

```go
// orderNotifier is the slice of NotificationService that RefundService needs.
// Declared here, on the consumer side, so settlement can be tested without a
// notification repository.
type orderNotifier interface {
	SendOrderNotification(ctx context.Context, order *domain.Order, trigger domain.NotificationTrigger, createdBy string) error
}
```

```go
type RefundService struct {
	refundRepo    domain.RefundRepository
	orderRepo     domain.OrderRepository
	paymentRepo   domain.PaymentRepository
	inventoryRepo domain.InventoryRepository
	auditService  domain.AuditService
	notifier      orderNotifier
	phonePe       phonepe.Gateway
}

// NewRefundService creates a new RefundService.
func NewRefundService(
	refundRepo domain.RefundRepository,
	orderRepo domain.OrderRepository,
	paymentRepo domain.PaymentRepository,
	inventoryRepo domain.InventoryRepository,
	auditService domain.AuditService,
	notifier orderNotifier,
	phonePe phonepe.Gateway,
) *RefundService {
	return &RefundService{
		refundRepo:    refundRepo,
		orderRepo:     orderRepo,
		paymentRepo:   paymentRepo,
		inventoryRepo: inventoryRepo,
		auditService:  auditService,
		notifier:      notifier,
		phonePe:       phonePe,
	}
}
```

- [ ] **Step 4: Implement settlement and re-check**

Append to `internal/service/refund_service.go`:

```go
// HandleRefundCompleted settles a refund from a pg.refund.completed webhook.
func (s *RefundService) HandleRefundCompleted(ctx context.Context, evt domain.RefundWebhookEvent) error {
	refund, err := s.findRefundForEvent(ctx, evt)
	if err != nil {
		return err
	}
	return s.settleCompleted(ctx, refund)
}

// HandleRefundFailed settles a refund from a pg.refund.failed webhook.
func (s *RefundService) HandleRefundFailed(ctx context.Context, evt domain.RefundWebhookEvent) error {
	refund, err := s.findRefundForEvent(ctx, evt)
	if err != nil {
		return err
	}
	return s.settleFailed(ctx, refund, evt.ErrorCode, evt.DetailedErrorCode)
}

// RecheckStatus asks PhonePe for a refund's current state and applies the same
// settlement logic. This is the escape hatch for a webhook that never arrived,
// and the recovery path when the initiation response was lost so
// ProviderRefundID was never stored — the status endpoint is keyed on
// MerchantRefundID, which we always have.
func (s *RefundService) RecheckStatus(ctx context.Context, refundID string) (*domain.Refund, error) {
	refund, err := s.refundRepo.GetByID(ctx, refundID)
	if err != nil {
		return nil, err
	}
	if refund.Status != domain.RefundStatusPending {
		return refund, nil
	}

	status, err := s.phonePe.CheckRefundStatus(ctx, refund.MerchantRefundID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read refund status from the payment provider")
	}

	if refund.ProviderRefundID == "" && status.RefundID != "" {
		if setErr := s.refundRepo.SetProviderRefundID(ctx, refund.ID, status.RefundID); setErr != nil {
			slog.ErrorContext(ctx, "Failed to store recovered provider refund ID",
				"refund_id", refund.ID, "error", setErr)
		}
		refund.ProviderRefundID = status.RefundID
	}

	switch status.State {
	case phonepe.StateCompleted:
		if err := s.settleCompleted(ctx, refund); err != nil {
			return nil, err
		}
	case phonepe.StateFailed:
		if err := s.settleFailed(ctx, refund, status.ErrorCode, status.DetailedErrorCode); err != nil {
			return nil, err
		}
	default:
		// Still PENDING at the provider. Nothing to apply.
		return refund, nil
	}

	return s.refundRepo.GetByID(ctx, refundID)
}

// findRefundForEvent resolves the refund a webhook refers to. PhonePe does not
// echo our merchantRefundId, so the payment's merchant transaction ID gives us
// the order and PhonePe's refundId picks the refund out of that order's — an
// order can have several.
func (s *RefundService) findRefundForEvent(ctx context.Context, evt domain.RefundWebhookEvent) (*domain.Refund, error) {
	payment, err := s.paymentRepo.GetByMerchantTxnID(ctx, evt.OriginalMerchantOrderID)
	if err != nil {
		return nil, err
	}

	refunds, err := s.refundRepo.ListByOrderID(ctx, payment.OrderID)
	if err != nil {
		return nil, err
	}

	for _, r := range refunds {
		if r.ProviderRefundID != "" && r.ProviderRefundID == evt.ProviderRefundID {
			return r, nil
		}
	}

	return nil, errors.NotFound("Refund not found for the provider refund ID")
}

// settleCompleted applies a successful refund to the payment, the order items
// and the order's payment status.
//
// The status read below is only an optimisation. The authority is
// SettleIfPending's ConditionExpression: of two concurrent deliveries exactly
// one gets past it, and every effect after it therefore runs exactly once.
func (s *RefundService) settleCompleted(ctx context.Context, refund *domain.Refund) error {
	if refund.Status != domain.RefundStatusPending {
		return nil
	}

	now := time.Now()
	if err := s.refundRepo.SettleIfPending(ctx, refund.ID, domain.RefundStatusCompleted,
		map[string]interface{}{"completed_at": now}); err != nil {
		if appErr, ok := errors.AsAppError(err); ok && appErr.Code == errors.ErrCodeConflict {
			slog.InfoContext(ctx, "Refund settlement lost the race, skipping", "refund_id", refund.ID)
			return nil
		}
		return err
	}

	// ADD, not read-modify-write: concurrent settlements of different refunds
	// against the same payment must not lose an increment.
	newTotal, err := s.paymentRepo.AddRefundAmount(ctx, refund.PaymentID, refund.Amount)
	if err != nil {
		return err
	}

	order, err := s.orderRepo.GetByID(ctx, refund.OrderID)
	if err != nil {
		return err
	}

	refundedByItem := make(map[string]int, len(refund.Items))
	for _, it := range refund.Items {
		refundedByItem[it.OrderItemID] += it.Quantity
	}
	for i := range order.Items {
		order.Items[i].RefundedQuantity += refundedByItem[order.Items[i].ID]
	}

	fullyRefunded := newTotal >= order.TotalAmount
	if fullyRefunded {
		order.PaymentStatus = domain.PaymentStatusRefunded
	} else {
		order.PaymentStatus = domain.PaymentStatusPartiallyRefunded
	}
	// Order.Status is deliberately untouched: the unrefunded remainder ships.
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return err
	}

	if fullyRefunded {
		if err := s.paymentRepo.UpdateStatus(ctx, refund.PaymentID, domain.PaymentStatusRefunded,
			map[string]interface{}{"refunded_at": now}); err != nil {
			slog.ErrorContext(ctx, "Failed to mark payment refunded", "payment_id", refund.PaymentID, "error", err)
		}
	}

	if err := s.notifier.SendOrderNotification(ctx, order, domain.NotificationTriggerRefund, "system"); err != nil {
		slog.ErrorContext(ctx, "Failed to send refund notification", "refund_id", refund.ID, "error", err)
	}

	metrics.Record(ctx, "refund_completed", metrics.L{
		metrics.LabelReason:  string(refund.Reason),
		metrics.LabelGateway: gatewayPhonePe,
	})

	return nil
}

// settleFailed records a refund the provider declined. The order and the
// payment are left untouched, and the inventory effect applied at creation is
// deliberately NOT reversed: a write-off reflects a physical fact that a
// payment failure does not change.
func (s *RefundService) settleFailed(ctx context.Context, refund *domain.Refund, errorCode, detailedErrorCode string) error {
	if refund.Status != domain.RefundStatusPending {
		return nil
	}

	if err := s.refundRepo.SettleIfPending(ctx, refund.ID, domain.RefundStatusFailed,
		map[string]interface{}{
			"error_code":          errorCode,
			"detailed_error_code": detailedErrorCode,
		}); err != nil {
		if appErr, ok := errors.AsAppError(err); ok && appErr.Code == errors.ErrCodeConflict {
			return nil
		}
		return err
	}

	metrics.Record(ctx, "refund_failed", metrics.L{
		metrics.LabelReason:  string(refund.Reason),
		metrics.LabelGateway: gatewayPhonePe,
	})

	return nil
}

// Ensure interface compliance
var _ domain.RefundService = (*RefundService)(nil)
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./internal/service/... -run TestRefundService -v`
Expected: PASS, every subtest including the Task 6 ones.

- [ ] **Step 6: Run the full service suite**

Run: `go test ./internal/service/...`
Expected: `ok`.

- [ ] **Step 7: Lint and commit**

```bash
golangci-lint run ./internal/service/...
git add internal/service/refund_service.go internal/service/refund_service_test.go
git commit -m "feat(refunds): settle refunds idempotently"
```

---

## Task 8: Webhook settlement routing and wiring

Refund settlement arrives on the **existing** `/api/v1/store/webhooks/phonepe`
route under the SHA256 `Authorization` verification already implemented. No new
endpoint, no new credentials, no CDK change.

**Files:**
- Modify: `internal/handler/store/webhook_handler.go`
- Test: `internal/handler/store/webhook_handler_test.go` (create)
- Modify: `internal/wire/providers.go` (refund repository + service + webhook handler providers)
- Modify: `internal/wire/wire.go` (`InitializeStoreWebhooksDeps`, `InitializeMonolithDeps`)
- Regenerate: `internal/wire/wire_gen.go`

**Interfaces:**
- Consumes: `domain.RefundService` (Task 1), `service.NewRefundService` (Tasks 6-7), `dynamodb.NewRefundRepository` (Task 5), `phonepe.RefundWebhookPayload` (Task 4).
- Produces:
  - `store.NewWebhookHandler(paymentService domain.PaymentService, refundService domain.RefundService, phonePe phonepe.Gateway, webhookUsername, webhookPassword string) *WebhookHandler` — **one new parameter**, `refundService`, second
  - `wire.ProvideRefundRepository(client *dynamodb.Client) domain.RefundRepository`
  - `wire.ProvideRefundService(...) *service.RefundService`

- [ ] **Step 1: Write the failing test**

Create `internal/handler/store/webhook_handler_test.go`:

```go
package store

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/phonepe"
	"github.com/handloom/admin/internal/mocks"
)

// postWebhook sends a raw body to the PhonePe webhook route. Credentials are
// left empty so signature verification is skipped, as it is in local dev.
func postWebhook(t *testing.T, h *WebhookHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/phonepe", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	h.Routes().ServeHTTP(rec, req)
	return rec
}

func newWebhookFixture(t *testing.T) (*WebhookHandler, *mocks.MockPaymentService, *mocks.MockRefundService) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	paymentSvc := mocks.NewMockPaymentService(ctrl)
	refundSvc := mocks.NewMockRefundService(ctrl)
	h := NewWebhookHandler(paymentSvc, refundSvc, phonepe.NewDevClient(""), "", "")
	return h, paymentSvc, refundSvc
}

func TestPhonePeWebhook_RoutesRefundCompleted(t *testing.T) {
	h, _, refundSvc := newWebhookFixture(t)

	var got domain.RefundWebhookEvent
	refundSvc.EXPECT().HandleRefundCompleted(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, evt domain.RefundWebhookEvent) error {
			got = evt
			return nil
		})

	rec := postWebhook(t, h, `{
		"event": "pg.refund.completed",
		"payload": {
			"originalMerchantOrderId": "txn_original",
			"refundId": "OMR456",
			"amount": 27000,
			"state": "COMPLETED"
		}
	}`)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "txn_original", got.OriginalMerchantOrderID)
	assert.Equal(t, "OMR456", got.ProviderRefundID)
	assert.Equal(t, int64(27000), got.Amount)
}

func TestPhonePeWebhook_RoutesRefundFailed(t *testing.T) {
	h, _, refundSvc := newWebhookFixture(t)

	var got domain.RefundWebhookEvent
	refundSvc.EXPECT().HandleRefundFailed(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, evt domain.RefundWebhookEvent) error {
			got = evt
			return nil
		})

	rec := postWebhook(t, h, `{
		"event": "pg.refund.failed",
		"payload": {
			"originalMerchantOrderId": "txn_original",
			"refundId": "OMR456",
			"amount": 27000,
			"state": "FAILED",
			"errorCode": "REFUND_DECLINED",
			"detailedErrorCode": "INSUFFICIENT_MERCHANT_BALANCE"
		}
	}`)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "REFUND_DECLINED", got.ErrorCode)
	assert.Equal(t, "INSUFFICIENT_MERCHANT_BALANCE", got.DetailedErrorCode)
}

func TestPhonePeWebhook_StillRoutesPaymentEvents(t *testing.T) {
	h, paymentSvc, _ := newWebhookFixture(t)

	var got domain.PaymentWebhookEvent
	paymentSvc.EXPECT().HandlePaymentSuccess(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, evt domain.PaymentWebhookEvent) error {
			got = evt
			return nil
		})

	rec := postWebhook(t, h, `{
		"event": "checkout.order.completed",
		"payload": {
			"merchantOrderId": "txn_original",
			"state": "COMPLETED",
			"paymentDetails": [{"transactionId": "T1", "paymentMode": "UPI_INTENT", "state": "COMPLETED"}]
		}
	}`)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "txn_original", got.MerchantTxnID)
	assert.Equal(t, "T1", got.TransactionID)
	assert.Equal(t, "UPI_INTENT", got.PaymentMode)
}

func TestPhonePeWebhook_UnknownEventIsAcknowledgedAndIgnored(t *testing.T) {
	h, _, _ := newWebhookFixture(t)
	// Neither service is called; gomock fails the test if either is.

	rec := postWebhook(t, h, `{"event": "pg.something.new", "payload": {}}`)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestPhonePeWebhook_MalformedBodyIsAcknowledged(t *testing.T) {
	h, _, _ := newWebhookFixture(t)

	rec := postWebhook(t, h, `not json`)
	require.Equal(t, http.StatusOK, rec.Code, "PhonePe must never be told to retry a body we cannot parse")
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/handler/store/... -run TestPhonePeWebhook -v`
Expected: FAIL to compile — too few arguments to `NewWebhookHandler`.

- [ ] **Step 3: Rewrite the webhook handler to branch on the event family**

Replace the body of `internal/handler/store/webhook_handler.go` below the imports with:

```go
// PhonePe webhook event names.
const (
	eventPaymentCompleted = "checkout.order.completed"
	eventPaymentFailed    = "checkout.order.failed"
	eventPaymentPending   = "checkout.order.pending"
	eventRefundCompleted  = "pg.refund.completed"
	eventRefundFailed     = "pg.refund.failed"
)

// WebhookHandler handles incoming webhook callbacks from external providers.
type WebhookHandler struct {
	paymentService  domain.PaymentService
	refundService   domain.RefundService
	phonePe         phonepe.Gateway
	webhookUsername string
	webhookPassword string
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(
	paymentService domain.PaymentService,
	refundService domain.RefundService,
	phonePe phonepe.Gateway,
	webhookUsername, webhookPassword string,
) *WebhookHandler {
	return &WebhookHandler{
		paymentService:  paymentService,
		refundService:   refundService,
		phonePe:         phonePe,
		webhookUsername: webhookUsername,
		webhookPassword: webhookPassword,
	}
}

// Routes returns the webhook routes. These routes are unauthenticated;
// verification is performed via provider-specific signature checks.
func (h *WebhookHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/phonepe", h.PhonePeWebhook)

	return r
}

// PhonePeWebhook handles incoming PhonePe Standard Checkout webhook callbacks.
// It reads the raw body, verifies the signature, then dispatches on the event
// name. Payment and refund events carry different payload shapes, so the body
// is decoded twice: once for the envelope, once for the family-specific payload.
// Always returns 200 OK — a non-200 makes PhonePe retry, which is wrong for
// anything we cannot parse.
func (h *WebhookHandler) PhonePeWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.ErrorContext(ctx, "failed to read PhonePe webhook body", "error", err)
		response.JSON(w, http.StatusOK, map[string]string{response.KeyStatus: response.KeyError})
		return
	}
	defer func() { _ = r.Body.Close() }()

	if !h.verify(ctx, r) {
		response.JSON(w, http.StatusOK, map[string]string{response.KeyStatus: response.KeyError})
		return
	}

	var envelope struct {
		Event string `json:"event"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		slog.ErrorContext(ctx, "Failed to parse PhonePe webhook envelope", "error", err)
		response.JSON(w, http.StatusOK, map[string]string{response.KeyStatus: response.KeyError})
		return
	}

	switch envelope.Event {
	case eventRefundCompleted, eventRefundFailed:
		h.handleRefundEvent(ctx, envelope.Event, body)
	case eventPaymentCompleted, eventPaymentFailed, eventPaymentPending:
		h.handlePaymentEvent(ctx, envelope.Event, body)
	default:
		slog.WarnContext(ctx, "Unhandled PhonePe webhook event", "event", envelope.Event)
	}

	response.JSON(w, http.StatusOK, map[string]string{response.KeyStatus: "ok"})
}

// verify checks the PhonePe Authorization header. Verification is skipped when
// credentials are not configured, which is the local-dev case.
func (h *WebhookHandler) verify(ctx context.Context, r *http.Request) bool {
	if h.webhookUsername == "" || h.webhookPassword == "" {
		slog.WarnContext(ctx, "PhonePe webhook signature verification SKIPPED - credentials not configured")
		return true
	}
	if !h.phonePe.VerifyWebhookSignature(h.webhookUsername, h.webhookPassword, r.Header.Get("Authorization")) {
		slog.ErrorContext(ctx, "Invalid PhonePe webhook signature")
		return false
	}
	return true
}

// handlePaymentEvent decodes a checkout.order.* payload and dispatches it.
func (h *WebhookHandler) handlePaymentEvent(ctx context.Context, event string, body []byte) {
	var payload phonepe.WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.ErrorContext(ctx, "Failed to parse PhonePe payment payload", "error", err)
		return
	}

	slog.InfoContext(ctx, "Received PhonePe payment webhook",
		"event", event,
		"merchant_order_id", payload.Payload.MerchantOrderID,
		"state", payload.Payload.State,
	)

	evt := domain.PaymentWebhookEvent{MerchantTxnID: payload.Payload.MerchantOrderID}
	if len(payload.Payload.PaymentDetails) > 0 {
		detail := payload.Payload.PaymentDetails[0]
		evt.TransactionID = detail.TransactionID
		evt.PaymentMode = detail.PaymentMode
	}

	var err error
	switch event {
	case eventPaymentCompleted:
		err = h.paymentService.HandlePaymentSuccess(ctx, evt)
	case eventPaymentFailed:
		err = h.paymentService.HandlePaymentFailure(ctx, evt)
	case eventPaymentPending:
		err = h.paymentService.HandlePaymentPending(ctx, evt)
	}
	if err != nil {
		slog.ErrorContext(ctx, "failed to handle payment webhook", "event", event, "error", err)
	}
}

// handleRefundEvent decodes a pg.refund.* payload and dispatches it.
func (h *WebhookHandler) handleRefundEvent(ctx context.Context, event string, body []byte) {
	var payload phonepe.RefundWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.ErrorContext(ctx, "Failed to parse PhonePe refund payload", "error", err)
		return
	}

	slog.InfoContext(ctx, "Received PhonePe refund webhook",
		"event", event,
		"original_merchant_order_id", payload.Payload.OriginalMerchantOrderID,
		"provider_refund_id", payload.Payload.RefundID,
		"state", payload.Payload.State,
	)

	evt := domain.RefundWebhookEvent{
		OriginalMerchantOrderID: payload.Payload.OriginalMerchantOrderID,
		ProviderRefundID:        payload.Payload.RefundID,
		Amount:                  payload.Payload.Amount,
		ErrorCode:               payload.Payload.ErrorCode,
		DetailedErrorCode:       payload.Payload.DetailedErrorCode,
	}

	var err error
	switch event {
	case eventRefundCompleted:
		err = h.refundService.HandleRefundCompleted(ctx, evt)
	case eventRefundFailed:
		err = h.refundService.HandleRefundFailed(ctx, evt)
	}
	if err != nil {
		slog.ErrorContext(ctx, "failed to handle refund webhook", "event", event, "error", err)
	}
}
```

Add `"context"` to the import block; the rest of the imports are unchanged.

- [ ] **Step 4: Add the Wire providers**

In `internal/wire/providers.go`, add the repository provider next to `ProvidePaymentRepository`:

```go
// ProvideRefundRepository creates a new RefundRepository
func ProvideRefundRepository(client *dynamodb.Client) domain.RefundRepository {
	return dynamodb.NewRefundRepository(client)
}
```

Add the service provider next to `ProvidePaymentService`:

```go
// ProvideRefundService creates a new RefundService
func ProvideRefundService(
	refundRepo domain.RefundRepository,
	orderRepo domain.OrderRepository,
	paymentRepo domain.PaymentRepository,
	inventoryRepo domain.InventoryRepository,
	auditService *service.AuditService,
	notificationService *service.NotificationService,
	phonePe phonepe.Gateway,
) *service.RefundService {
	return service.NewRefundService(refundRepo, orderRepo, paymentRepo, inventoryRepo, auditService, notificationService, phonePe)
}
```

Update the webhook handler provider:

```go
func ProvideStoreWebhookHandler(
	paymentService *service.PaymentService,
	refundService *service.RefundService,
	phonePe phonepe.Gateway,
	cfg *config.Config,
) *store.WebhookHandler {
	return store.NewWebhookHandler(paymentService, refundService, phonePe, cfg.Store.PhonePeWebhookUsername, cfg.Store.PhonePeWebhookPassword)
}
```

And add both new providers to `ServiceSet` / `RepositorySet` alongside their payment counterparts:

```go
	ProvideRefundRepository,   // in RepositorySet, next to ProvidePaymentRepository
	ProvideRefundService,      // in ServiceSet, next to ProvidePaymentService
```

- [ ] **Step 5: Extend the injectors**

In `internal/wire/wire.go`, `InitializeStoreWebhooksDeps` gains the refund graph. Settlement needs the notification service, which this Lambda did not previously wire — that is the one genuinely new dependency in this design:

```go
// InitializeStoreWebhooksDeps creates Store Webhooks Lambda dependencies
func InitializeStoreWebhooksDeps(ctx context.Context, cfg *config.Config) (*StoreWebhooksDeps, error) {
	wire.Build(
		CoreSet,
		ProvidePaymentRepository,
		ProvideOrderRepository,
		ProvideCustomerRepository,
		ProvideInventoryRepository,
		ProvideCartRepository,
		ProvideProductRepository,
		ProvideRefundRepository,
		ProvideAuditRepository,
		ProvideNotificationRepository,
		ProvideUserRepository,
		ProvideCartService,
		ProvideAuditService,
		ProvideNotificationService,
		ProvidePhonePeGateway,
		ProvidePaymentService,
		ProvideRefundService,
		ProvideStoreWebhookHandler,
		wire.Struct(new(StoreWebhooksDeps), "*"),
	)
	return nil, nil
}
```

`InitializeMonolithDeps` already wires the audit, notification and user repositories for its own handlers; add `ProvideRefundRepository` and `ProvideRefundService` to its `wire.Build` list next to `ProvidePaymentService`.

- [ ] **Step 6: Regenerate Wire**

Run: `make wire`
Expected: `wire_gen.go` regenerated with no error. If Wire reports an unused provider, a provider was added to an injector whose output struct does not need it — remove it from that injector.

- [ ] **Step 7: Run the test to verify it passes**

Run: `go test ./internal/handler/store/... -run TestPhonePeWebhook -v`
Expected: PASS, all five tests.

- [ ] **Step 8: Build and smoke-test the monolith**

```bash
go build ./...
go test ./...
```
Expected: build clean. `TestSearcher_Hybrid_ReturnsKnownProduct` in `cmd/embedder/embedder` is a known pre-existing failure on `main` (see the inventory backlog doc) — every other package must pass.

- [ ] **Step 9: Lint and commit**

```bash
golangci-lint run ./internal/...
git add internal/handler/store/webhook_handler.go internal/handler/store/webhook_handler_test.go \
        internal/wire/providers.go internal/wire/wire.go internal/wire/wire_gen.go
git commit -m "feat(refunds): settle PhonePe refund webhooks"
```

---

## Task 9: Admin refund API, and deleting the stub

`POST /admin/orders/{id}/refund` (singular) sets `PaymentStatus = REFUNDED`
with no gateway call. Leaving it in place is a route that silently lies about
money having moved, so it is removed rather than deprecated.

**Files:**
- Modify: `internal/handler/order_handler.go`
- Test: `internal/handler/order_handler_test.go` (create)
- Modify: `internal/handler/request_types.go:51-55` (delete `RefundOrderRequest`)
- Modify: `internal/domain/order_repository.go` (delete `OrderService.RefundOrder`)
- Modify: `internal/service/order_service.go:502-526` (delete `RefundOrder`)
- Modify: `internal/service/order_service_test.go:685-…` (delete `TestOrderService_RefundOrder`)
- Modify: `internal/domain/store_service.go:78` (delete `PaymentService.RefundPayment`)
- Modify: `internal/service/payment_service.go:391-394` (delete `RefundPayment`)
- Modify: `internal/wire/providers.go` (`ProvideOrderHandler`)
- Modify: `internal/wire/wire.go` (`InitializeOrderDeps`)

**Interfaces:**
- Consumes: `domain.RefundService` (Task 1), `service.RefundService` (Tasks 6-7).
- Produces:
  - `handler.NewOrderHandler(orderService domain.OrderService, paymentService domain.PaymentService, refundService domain.RefundService, auth *middleware.Auth, validation *middleware.Validation) *OrderHandler` — **two new parameters**
  - Routes: `GET|POST /admin/orders/{id}/refunds`, `POST /admin/orders/{id}/refunds/{refundID}/recheck`
- Removes: `domain.OrderService.RefundOrder`, `domain.PaymentService.RefundPayment`, `handler.RefundOrderRequest`, `OrderHandler.Refund`.

The role guard lives in `OrderHandler.Routes()` rather than in
`internal/router/order.go` because both the monolith (`cmd/api/main.go:168`) and
the order Lambda (`cmd/lambda/order/main.go:30`) mount `OrderHandler.Routes()`
directly — putting it here means neither entry point can forget it, and neither
`NewOrderRouter` nor any `cmd/` file changes.

- [ ] **Step 1: Write the failing test**

Create `internal/handler/order_handler_test.go`:

```go
package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/internal/validator"
)

// asUser wraps a router so every request carries the given authenticated user,
// standing in for the Authenticate middleware the real routers apply upstream.
func asUser(next http.Handler, user *domain.User) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), middleware.UserKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newOrderHandlerFixture(t *testing.T) (*OrderHandler, *mocks.MockRefundService) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	refundSvc := mocks.NewMockRefundService(ctrl)
	h := NewOrderHandler(
		mocks.NewMockOrderService(ctrl),
		mocks.NewMockPaymentService(ctrl),
		refundSvc,
		middleware.NewAuth(nil), // RequireRole reads only the context user
		middleware.NewValidation(validator.New(), middleware.ValidationConfig{}),
	)
	return h, refundSvc
}

func TestOrderHandler_CreateRefund_RequiresAdmin(t *testing.T) {
	h, _ := newOrderHandlerFixture(t)
	// No EXPECT on the refund service: an OPERATOR must never reach it.

	body := `{"reason":"OUT_OF_STOCK","items":[{"order_item_id":"oi_1","quantity":1,"restock":false}]}`
	req := httptest.NewRequest(http.MethodPost, "/order_1/refunds", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	asUser(h.Routes(), &domain.User{ID: "user_1", Role: domain.UserRoleOperator}).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestOrderHandler_CreateRefund_AdminSucceeds(t *testing.T) {
	h, refundSvc := newOrderHandlerFixture(t)

	var gotOrderID, gotCreatedBy string
	var gotReq domain.CreateRefundRequest
	refundSvc.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, orderID string, req domain.CreateRefundRequest, createdBy string) (*domain.Refund, error) {
			gotOrderID, gotReq, gotCreatedBy = orderID, req, createdBy
			return &domain.Refund{ID: "refund_1", Amount: 27000, Status: domain.RefundStatusPending}, nil
		})

	body := `{"reason":"OUT_OF_STOCK","items":[{"order_item_id":"oi_1","quantity":1,"restock":false}]}`
	req := httptest.NewRequest(http.MethodPost, "/order_1/refunds", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	asUser(h.Routes(), &domain.User{ID: "admin_1", Role: domain.UserRoleAdmin}).ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Equal(t, "order_1", gotOrderID)
	assert.Equal(t, "admin_1", gotCreatedBy)
	assert.Equal(t, domain.RefundReasonOutOfStock, gotReq.Reason)
	require.Len(t, gotReq.Items, 1)
	assert.Equal(t, 1, gotReq.Items[0].Quantity)
}

func TestOrderHandler_CreateRefund_RejectsClientSuppliedAmount(t *testing.T) {
	h, refundSvc := newOrderHandlerFixture(t)

	// An "amount" in the body is not part of CreateRefundRequest, so it is
	// discarded rather than honoured. Money is never a client input.
	refundSvc.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _ string, req domain.CreateRefundRequest, _ string) (*domain.Refund, error) {
			assert.Len(t, req.Items, 1)
			return &domain.Refund{ID: "refund_1", Amount: 27000}, nil
		})

	body := `{"reason":"OTHER","amount":9999999,"items":[{"order_item_id":"oi_1","quantity":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/order_1/refunds", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	asUser(h.Routes(), &domain.User{ID: "admin_1", Role: domain.UserRoleAdmin}).ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
}

func TestOrderHandler_CreateRefund_RejectsUnknownReason(t *testing.T) {
	h, _ := newOrderHandlerFixture(t)
	// Validation rejects it before the service is reached.

	body := `{"reason":"BECAUSE_I_SAID_SO","items":[{"order_item_id":"oi_1","quantity":1}]}`
	req := httptest.NewRequest(http.MethodPost, "/order_1/refunds", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	asUser(h.Routes(), &domain.User{ID: "admin_1", Role: domain.UserRoleAdmin}).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestOrderHandler_ListRefunds(t *testing.T) {
	h, refundSvc := newOrderHandlerFixture(t)

	refundSvc.EXPECT().ListByOrder(gomock.Any(), "order_1").
		Return([]*domain.Refund{{ID: "refund_1"}}, nil)

	req := httptest.NewRequest(http.MethodGet, "/order_1/refunds", nil)
	rec := httptest.NewRecorder()

	asUser(h.Routes(), &domain.User{ID: "admin_1", Role: domain.UserRoleAdmin}).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "refund_1")
}

func TestOrderHandler_RecheckRefund(t *testing.T) {
	h, refundSvc := newOrderHandlerFixture(t)

	refundSvc.EXPECT().RecheckStatus(gomock.Any(), "refund_1").
		Return(&domain.Refund{ID: "refund_1", Status: domain.RefundStatusCompleted}, nil)

	req := httptest.NewRequest(http.MethodPost, "/order_1/refunds/refund_1/recheck", nil)
	rec := httptest.NewRecorder()

	asUser(h.Routes(), &domain.User{ID: "admin_1", Role: domain.UserRoleAdmin}).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "COMPLETED")
}

func TestOrderHandler_StubRefundRouteIsGone(t *testing.T) {
	h, _ := newOrderHandlerFixture(t)

	// POST /{id}/refund (singular) marked money refunded without moving any.
	// It is deleted, not deprecated — no route should answer here.
	req := httptest.NewRequest(http.MethodPost, "/order_1/refund", bytes.NewBufferString(`{"amount":1,"reason":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	asUser(h.Routes(), &domain.User{ID: "admin_1", Role: domain.UserRoleAdmin}).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/handler/... -run TestOrderHandler -v`
Expected: FAIL to compile — too few arguments to `NewOrderHandler`.

- [ ] **Step 3: Rework the order handler**

In `internal/handler/order_handler.go`, replace the struct, constructor and `Routes`:

```go
// OrderHandler handles order-related requests
type OrderHandler struct {
	orderService   domain.OrderService
	paymentService domain.PaymentService
	refundService  domain.RefundService
	auth           *middleware.Auth
	validation     *middleware.Validation
}

// NewOrderHandler creates a new OrderHandler
func NewOrderHandler(
	orderService domain.OrderService,
	paymentService domain.PaymentService,
	refundService domain.RefundService,
	auth *middleware.Auth,
	validation *middleware.Validation,
) *OrderHandler {
	return &OrderHandler{
		orderService:   orderService,
		paymentService: paymentService,
		refundService:  refundService,
		auth:           auth,
		validation:     validation,
	}
}

// Routes returns the order routes
func (h *OrderHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.With(middleware.ValidateJSONTyped[domain.CreateOrderRequest](h.validation)).Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.Get("/{id}/payment-status", h.CheckPaymentStatus)
	r.With(middleware.ValidateJSONTyped[UpdateOrderStatusRequest](h.validation)).Patch("/{id}/status", h.UpdateStatus)
	r.With(middleware.ValidateJSONTyped[AddOrderNoteRequest](h.validation)).Post("/{id}/notes", h.AddNote)
	r.With(middleware.ValidateJSONTyped[UpdateTrackingRequest](h.validation)).Patch("/{id}/tracking", h.UpdateTracking)
	r.With(middleware.ValidateJSONTyped[CancelOrderRequest](h.validation)).Post("/{id}/cancel", h.Cancel)

	// Refunds move real money, so they are ADMIN-only — the same guard the
	// audit router applies. Mounted here rather than in internal/router so
	// both the monolith and the order Lambda inherit it automatically.
	r.Group(func(r chi.Router) {
		r.Use(h.auth.RequireRole(domain.UserRoleAdmin))
		r.Get("/{id}/refunds", h.ListRefunds)
		r.With(middleware.ValidateJSONTyped[domain.CreateRefundRequest](h.validation)).Post("/{id}/refunds", h.CreateRefund)
		r.Post("/{id}/refunds/{refundID}/recheck", h.RecheckRefund)
	})

	return r
}
```

Then replace the old `Refund` handler (currently lines 187-200) with the three new ones:

```go
// CreateRefund initiates a refund for specific order lines. The request carries
// line IDs and quantities only — the server derives the amount.
func (h *OrderHandler) CreateRefund(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	req := middleware.MustGetValidatedBody[domain.CreateRefundRequest](ctx)

	refund, err := h.refundService.Create(ctx, id, req, getUserIDFromContext(ctx))
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, refund)
}

// ListRefunds returns every refund recorded against an order.
func (h *OrderHandler) ListRefunds(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	refunds, err := h.refundService.ListByOrder(ctx, id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, refunds)
}

// RecheckRefund forces a provider status re-check. This is the escape hatch for
// a webhook that never arrived.
func (h *OrderHandler) RecheckRefund(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	refundID := chi.URLParam(r, "refundID")

	refund, err := h.refundService.RecheckStatus(ctx, refundID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, refund)
}
```

- [ ] **Step 4: Delete the stub**

In `internal/handler/request_types.go`, delete `RefundOrderRequest` (lines 51-55) and its comment.

In `internal/domain/order_repository.go`, delete the last entry of `OrderService`:

```go
	// RefundOrder initiates a refund for an order
	RefundOrder(ctx context.Context, id string, amount int64, reason string, updatedBy string) error
```

In `internal/service/order_service.go`, delete the whole `RefundOrder` method (lines 502-526).

In `internal/service/order_service_test.go`, delete `TestOrderService_RefundOrder` (starts at line 685).

In `internal/domain/store_service.go`, delete from `PaymentService`:

```go
	RefundPayment(ctx context.Context, paymentID string, amount int64, reason string) error
```

In `internal/service/payment_service.go`, delete:

```go
// RefundPayment is a placeholder for refund functionality
func (s *PaymentService) RefundPayment(_ context.Context, _ string, _ int64, _ string) error {
	return errors.New(errors.ErrCodeInternal, "Refund functionality is not implemented yet")
}
```

- [ ] **Step 5: Update the handler provider and the order injector**

In `internal/wire/providers.go`:

```go
func ProvideOrderHandler(
	orderService *service.OrderService,
	paymentService *service.PaymentService,
	refundService *service.RefundService,
	auth *middleware.Auth,
	validation *middleware.Validation,
) *handler.OrderHandler {
	return handler.NewOrderHandler(orderService, paymentService, refundService, auth, validation)
}
```

In `internal/wire/wire.go`, `InitializeOrderDeps` gains everything `RefundService` needs. It already wires the order, customer, product, inventory, payment and cart repositories plus the auth middleware; add:

```go
		ProvideRefundRepository,
		ProvideAuditRepository,
		ProvideNotificationRepository,
		ProvideAuditService,
		ProvideNotificationService,
		ProvideRefundService,
```

(`ProvideUserRepository` is already in that injector for the auth middleware, so `ProvideNotificationService` finds its user repository.)

- [ ] **Step 6: Regenerate mocks and Wire**

```bash
make generate-mocks
make wire
```
Expected: both succeed. `make generate-mocks` is needed because `OrderService` and `PaymentService` each lost a method.

- [ ] **Step 7: Run the test to verify it passes**

Run: `go test ./internal/handler/... -run TestOrderHandler -v`
Expected: PASS, all seven tests.

- [ ] **Step 8: Build and run the whole backend suite**

```bash
go build ./...
go test ./...
```
Expected: build clean; every package passes except the known pre-existing `cmd/embedder/embedder` failure.

Also confirm the stub is gone everywhere:

```bash
grep -rn "RefundOrder\|RefundPayment" --include=*.go internal/ cmd/
```
Expected: no matches.

- [ ] **Step 9: Lint and commit**

```bash
golangci-lint run ./internal/...
git add internal/handler/ internal/domain/ internal/service/ internal/wire/
git commit -m "feat(refunds): add the admin refund API, drop the stub"
```

---

## Task 10: Frontends

Per `CLAUDE.md`, both frontends ship in the same PR as the API change. The
storefront only needs to learn the new payment status; the admin app gets the
refund UI.

**Files:**
- Modify: `handloom-admin-frontend/src/features/orders/types.ts`
- Modify: `handloom-admin-frontend/src/features/orders/api.ts`
- Modify: `handloom-admin-frontend/src/shared/constants/routes.ts:46-56`
- Modify: `handloom-admin-frontend/src/features/inventory/types.ts:1`
- Create: `handloom-admin-frontend/src/features/orders/refundPreview.ts`
- Test: `handloom-admin-frontend/src/features/orders/__tests__/refundPreview.test.ts`
- Create: `handloom-admin-frontend/src/features/orders/components/RefundSection.tsx`
- Modify: `handloom-admin-frontend/src/features/orders/components/OrderDetailPage/OrderDetailPage.tsx`
- Modify: `homechrome-store/src/types/index.ts:166-172`
- Modify: `homechrome-store/src/lib/utils.ts:40`

**Interfaces:**
- Consumes: the API from Task 9 (`GET|POST /admin/orders/{id}/refunds`, `POST /admin/orders/{id}/refunds/{refundID}/recheck`) and the `Refund` JSON shape from Task 1.
- Produces: `Refund`, `RefundItem`, `RefundStatus`, `RefundReason`, `CreateRefundPayload` types; `ordersApi.listRefunds`, `ordersApi.createRefund`, `ordersApi.recheckRefund`; `refundPreview()`; `<RefundSection order={order} />`.

- [ ] **Step 1: Write the failing test**

Create `handloom-admin-frontend/src/features/orders/__tests__/refundPreview.test.ts`:

```ts
import { describe, expect, it } from 'vitest';

import { refundPreview } from '../refundPreview';
import type { Order } from '../types';

// ₹1000 subtotal, ₹100 discount, ₹50 shipping → ₹950 total.
const order = {
  id: 'order_1',
  subtotal: 100000,
  discount_amount: 10000,
  tax_amount: 0,
  shipping_amount: 5000,
  total_amount: 95000,
  items: [
    { id: 'oi_1', product_id: 'prod_1', product_name: 'A', product_sku: 'A', unit_price: 30000, quantity: 2, total_price: 60000 },
    { id: 'oi_2', product_id: 'prod_2', product_name: 'B', product_sku: 'B', unit_price: 40000, quantity: 1, total_price: 40000 },
  ],
} as unknown as Order;

describe('refundPreview', () => {
  it('prorates the discount across a partial refund and retains shipping', () => {
    const preview = refundPreview(order, { oi_1: 1 }, {});
    expect(preview.lineSubtotal).toBe(30000);
    expect(preview.discount).toBe(3000);
    expect(preview.shipping).toBe(0);
    expect(preview.total).toBe(27000);
    expect(preview.isFinal).toBe(false);
  });

  it('includes shipping and matches the order total on a final refund', () => {
    const preview = refundPreview(order, { oi_1: 2, oi_2: 1 }, {});
    expect(preview.isFinal).toBe(true);
    expect(preview.shipping).toBe(5000);
    expect(preview.total).toBe(95000);
  });

  it('counts already-refunded units when deciding a refund is final', () => {
    const preview = refundPreview(order, { oi_2: 1 }, { oi_1: 2 });
    expect(preview.isFinal).toBe(true);
  });

  it('caps a line at the unrefunded remainder', () => {
    expect(remainingQuantity(order, 'oi_1', { oi_1: 1 })).toBe(1);
    expect(remainingQuantity(order, 'oi_2', {})).toBe(1);
  });
});

// Imported here so the cap helper is covered by the same suite.
import { remainingQuantity } from '../refundPreview';
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd handloom-admin-frontend && npm run test -- refundPreview`
Expected: FAIL — cannot resolve `../refundPreview`.

- [ ] **Step 3: Add the preview helpers**

Create `handloom-admin-frontend/src/features/orders/refundPreview.ts`:

```ts
import type { Order } from './types';

/** Quantities keyed by order item ID. */
export type QuantityMap = Record<string, number>;

export interface RefundPreview {
  lineSubtotal: number;
  discount: number;
  tax: number;
  shipping: number;
  total: number;
  isFinal: boolean;
}

/** Integer paise, rounded half-up — mirrors the server's prorate(). */
function prorate(amount: number, part: number, whole: number): number {
  if (whole <= 0 || amount === 0) return 0;
  return Math.floor((amount * part + Math.floor(whole / 2)) / whole);
}

/** Units of a line that have not been refunded yet. */
export function remainingQuantity(order: Order, itemId: string, alreadyRefunded: QuantityMap): number {
  const item = order.items.find((i) => i.id === itemId);
  if (!item) return 0;
  return item.quantity - (alreadyRefunded[itemId] ?? 0);
}

/**
 * refundPreview mirrors the server's amount derivation for display only. The
 * server recomputes the amount and its number is the one that moves — this
 * exists so the admin sees what they are about to do, not to decide it.
 */
export function refundPreview(order: Order, requested: QuantityMap, alreadyRefunded: QuantityMap): RefundPreview {
  let lineSubtotal = 0;
  let discount = 0;
  let tax = 0;

  for (const item of order.items) {
    const quantity = requested[item.id] ?? 0;
    if (quantity <= 0) continue;
    const subtotal = item.unit_price * quantity;
    lineSubtotal += subtotal;
    discount += prorate(order.discount_amount, subtotal, order.subtotal);
    tax += prorate(order.tax_amount, subtotal, order.subtotal);
  }

  const isFinal = order.items.every(
    (item) => (alreadyRefunded[item.id] ?? 0) + (requested[item.id] ?? 0) === item.quantity
  );

  // A final refund is "the rest of the order", which is what folds in shipping
  // and absorbs the paise that per-line rounding leaves behind.
  if (isFinal) {
    let alreadyRefundedAmount = 0;
    for (const item of order.items) {
      const settled = alreadyRefunded[item.id] ?? 0;
      if (settled <= 0) continue;
      const subtotal = item.unit_price * settled;
      alreadyRefundedAmount +=
        subtotal -
        prorate(order.discount_amount, subtotal, order.subtotal) +
        prorate(order.tax_amount, subtotal, order.subtotal);
    }
    return {
      lineSubtotal,
      discount,
      tax,
      shipping: order.shipping_amount,
      total: order.total_amount - alreadyRefundedAmount,
      isFinal,
    };
  }

  return { lineSubtotal, discount, tax, shipping: 0, total: lineSubtotal - discount + tax, isFinal };
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd handloom-admin-frontend && npm run test -- refundPreview`
Expected: PASS, all four cases.

- [ ] **Step 5: Add the types**

In `handloom-admin-frontend/src/features/orders/types.ts`, extend `PaymentStatus`, extend `OrderItem`, and append the refund types:

```ts
export type PaymentStatus = 'PENDING' | 'PAID' | 'FAILED' | 'REFUNDED' | 'PARTIALLY_REFUNDED';

export interface OrderItem {
  id: string;
  product_id: string;
  product_name: string;
  product_sku: string;
  quantity: number;
  unit_price: number;
  total_price: number;
  refunded_quantity?: number;
  custom_dimensions?: Dimensions;
  attributes?: Record<string, unknown>;
}

export type RefundStatus = 'PENDING' | 'COMPLETED' | 'FAILED';

export type RefundReason =
  | 'OUT_OF_STOCK'
  | 'DAMAGED'
  | 'CUSTOMER_REQUEST'
  | 'PRICING_ERROR'
  | 'OTHER';

export const REFUND_REASONS: { value: RefundReason; label: string }[] = [
  { value: 'OUT_OF_STOCK', label: 'Out of stock' },
  { value: 'DAMAGED', label: 'Damaged' },
  { value: 'CUSTOMER_REQUEST', label: 'Customer request' },
  { value: 'PRICING_ERROR', label: 'Pricing error' },
  { value: 'OTHER', label: 'Other' },
];

export interface RefundItem {
  order_item_id: string;
  product_id: string;
  quantity: number;
  amount: number;
  restock: boolean;
}

export interface Refund {
  id: string;
  order_id: string;
  payment_id: string;
  customer_id: string;
  amount: number;
  status: RefundStatus;
  reason: RefundReason;
  items: RefundItem[];
  merchant_refund_id: string;
  provider_refund_id?: string;
  error_code?: string;
  detailed_error_code?: string;
  initiated_at: string;
  completed_at?: string;
  created_by: string;
}

export interface CreateRefundPayload {
  reason: RefundReason;
  items: { order_item_id: string; quantity: number; restock: boolean }[];
}
```

In `handloom-admin-frontend/src/features/inventory/types.ts`, add the write-off ledger type:

```ts
export type TransactionType = 'ADD' | 'REMOVE' | 'RESERVE' | 'RELEASE' | 'ADJUST' | 'COMMIT' | 'WRITE_OFF';
```

- [ ] **Step 6: Replace the API client's refund call**

In `handloom-admin-frontend/src/shared/constants/routes.ts`, replace the `REFUND` entry inside `ORDERS`:

```ts
    REFUNDS: (id: string) => `/admin/orders/${id}/refunds`,
    REFUND_RECHECK: (id: string, refundId: string) =>
      `/admin/orders/${id}/refunds/${refundId}/recheck`,
```

In `handloom-admin-frontend/src/features/orders/api.ts`, replace the `refund` method (the old `POST /refund` route no longer exists):

```ts
  listRefunds: async (id: string): Promise<Refund[]> => {
    const response = await apiClient.get<Refund[]>(ROUTES.ORDERS.REFUNDS(id));
    return response.data ?? [];
  },

  createRefund: async (id: string, payload: CreateRefundPayload): Promise<Refund> => {
    const response = await apiClient.post<Refund>(ROUTES.ORDERS.REFUNDS(id), payload);
    return response.data;
  },

  recheckRefund: async (id: string, refundId: string): Promise<Refund> => {
    const response = await apiClient.post<Refund>(ROUTES.ORDERS.REFUND_RECHECK(id, refundId));
    return response.data;
  },
```

Add `Refund` and `CreateRefundPayload` to the file's type imports.

- [ ] **Step 7: Build the refund UI**

Create `handloom-admin-frontend/src/features/orders/components/RefundSection.tsx`:

```tsx
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useMemo, useState } from 'react';
import toast from 'react-hot-toast';

import { Badge, Button, Modal, Select } from '@/shared/components/ui';
import { useAuthStore } from '@/shared/stores/authStore';
import { formatCurrency, getErrorMessage } from '@/shared/utils';

import { ordersApi } from '../api';
import { refundPreview, remainingQuantity, type QuantityMap } from '../refundPreview';
import { REFUND_REASONS, type Order, type RefundReason, type RefundStatus } from '../types';

const statusVariant: Record<RefundStatus, 'warning' | 'success' | 'danger'> = {
  PENDING: 'warning',
  COMPLETED: 'success',
  FAILED: 'danger',
};

interface RefundSectionProps {
  order: Order;
}

export function RefundSection({ order }: RefundSectionProps) {
  const { user } = useAuthStore();
  const queryClient = useQueryClient();
  const [isOpen, setIsOpen] = useState(false);
  const [reason, setReason] = useState<RefundReason>('OUT_OF_STOCK');
  const [quantities, setQuantities] = useState<QuantityMap>({});
  const [restock, setRestock] = useState<Record<string, boolean>>({});

  const isAdmin = user?.role === 'ADMIN';
  const postDispatch = order.status === 'SHIPPED' || order.status === 'DELIVERED';

  const { data: refunds = [] } = useQuery({
    queryKey: ['order', order.id, 'refunds'],
    queryFn: () => ordersApi.listRefunds(order.id),
    enabled: isAdmin,
  });

  // Units held by an in-flight refund are unavailable too, not just settled ones.
  const alreadyRefunded = useMemo<QuantityMap>(() => {
    const map: QuantityMap = {};
    for (const item of order.items) map[item.id] = item.refunded_quantity ?? 0;
    for (const refund of refunds) {
      if (refund.status !== 'PENDING') continue;
      for (const line of refund.items) map[line.order_item_id] = (map[line.order_item_id] ?? 0) + line.quantity;
    }
    return map;
  }, [order.items, refunds]);

  const preview = useMemo(
    () => refundPreview(order, quantities, alreadyRefunded),
    [order, quantities, alreadyRefunded]
  );

  const createMutation = useMutation({
    mutationFn: () =>
      ordersApi.createRefund(order.id, {
        reason,
        items: Object.entries(quantities)
          .filter(([, quantity]) => quantity > 0)
          .map(([order_item_id, quantity]) => ({
            order_item_id,
            quantity,
            restock: restock[order_item_id] ?? false,
          })),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['order', order.id] });
      queryClient.invalidateQueries({ queryKey: ['order', order.id, 'refunds'] });
      toast.success('Refund initiated');
      setIsOpen(false);
      setQuantities({});
      setRestock({});
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  const recheckMutation = useMutation({
    mutationFn: (refundId: string) => ordersApi.recheckRefund(order.id, refundId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['order', order.id, 'refunds'] });
      toast.success('Refund status refreshed');
    },
    onError: (error) => toast.error(getErrorMessage(error)),
  });

  // Hidden entirely for non-admins, matching the server guard.
  if (!isAdmin) return null;

  const hasSelection = Object.values(quantities).some((quantity) => quantity > 0);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium text-gray-900">Refunds</h3>
        <Button variant="secondary" onClick={() => setIsOpen(true)}>
          Refund
        </Button>
      </div>

      {refunds.length === 0 ? (
        <p className="text-sm text-gray-500">No refunds on this order.</p>
      ) : (
        <ul className="divide-y divide-gray-200">
          {refunds.map((refund) => (
            <li key={refund.id} className="flex items-center justify-between py-3">
              <div>
                <p className="text-sm font-medium text-gray-900">{formatCurrency(refund.amount)}</p>
                <p className="text-xs text-gray-500">
                  {refund.reason} · {refund.created_by} · {new Date(refund.initiated_at).toLocaleString()}
                </p>
                {refund.error_code && <p className="text-xs text-red-600">{refund.error_code}</p>}
              </div>
              <div className="flex items-center gap-2">
                <Badge variant={statusVariant[refund.status]}>{refund.status}</Badge>
                {refund.status === 'PENDING' && (
                  <Button
                    variant="ghost"
                    onClick={() => recheckMutation.mutate(refund.id)}
                    disabled={recheckMutation.isPending}
                  >
                    Re-check
                  </Button>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}

      <Modal isOpen={isOpen} onClose={() => setIsOpen(false)} title="Refund order" size="lg">
        <div className="space-y-4">
          {postDispatch && (
            <p className="rounded bg-amber-50 p-3 text-sm text-amber-800">
              This order has been dispatched, so the refund moves money only. To return stock, mark
              the order RETURNED.
            </p>
          )}

          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-gray-500">
                <th className="py-2">Item</th>
                <th>Ordered</th>
                <th>Refunded</th>
                <th>Refund qty</th>
                {!postDispatch && <th>Stock</th>}
              </tr>
            </thead>
            <tbody>
              {order.items.map((item) => {
                const remaining = remainingQuantity(order, item.id, alreadyRefunded);
                return (
                  <tr key={item.id} className="border-t border-gray-100">
                    <td className="py-2">{item.product_name}</td>
                    <td>{item.quantity}</td>
                    <td>{alreadyRefunded[item.id] ?? 0}</td>
                    <td>
                      <input
                        type="number"
                        min={0}
                        max={remaining}
                        value={quantities[item.id] ?? 0}
                        disabled={remaining === 0}
                        onChange={(e) =>
                          setQuantities((prev) => ({
                            ...prev,
                            [item.id]: Math.max(0, Math.min(remaining, Number(e.target.value))),
                          }))
                        }
                        className="w-20 rounded border border-gray-300 px-2 py-1"
                      />
                    </td>
                    {!postDispatch && (
                      <td>
                        <label className="flex items-center gap-2 text-xs">
                          <input
                            type="checkbox"
                            checked={restock[item.id] ?? false}
                            onChange={(e) =>
                              setRestock((prev) => ({ ...prev, [item.id]: e.target.checked }))
                            }
                          />
                          Return to sale
                        </label>
                      </td>
                    )}
                  </tr>
                );
              })}
            </tbody>
          </table>

          <Select
            label="Reason"
            value={reason}
            onChange={(e) => setReason(e.target.value as RefundReason)}
            options={REFUND_REASONS.map((r) => ({ value: r.value, label: r.label }))}
          />

          <dl className="space-y-1 rounded bg-gray-50 p-3 text-sm">
            <div className="flex justify-between">
              <dt>Line subtotal</dt>
              <dd>{formatCurrency(preview.lineSubtotal)}</dd>
            </div>
            <div className="flex justify-between">
              <dt>Discount</dt>
              <dd>−{formatCurrency(preview.discount)}</dd>
            </div>
            <div className="flex justify-between">
              <dt>Shipping</dt>
              <dd>{preview.isFinal ? formatCurrency(preview.shipping) : 'retained'}</dd>
            </div>
            <div className="flex justify-between font-medium">
              <dt>Total</dt>
              <dd>{formatCurrency(preview.total)}</dd>
            </div>
          </dl>
          <p className="text-xs text-gray-500">
            The server recomputes this total; the figure above is an estimate.
          </p>

          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setIsOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => createMutation.mutate()}
              disabled={!hasSelection || createMutation.isPending}
            >
              Refund {formatCurrency(preview.total)}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
```

Render it on the order detail page. In
`handloom-admin-frontend/src/features/orders/components/OrderDetailPage/OrderDetailPage.tsx`,
import `RefundSection` from `../RefundSection` and place `<RefundSection order={order} />`
inside the detail layout, next to `<OrderNotes />`.

- [ ] **Step 8: Update the storefront**

In `homechrome-store/src/types/index.ts`, extend the `PaymentStatus` union (lines 166-172):

```ts
export type PaymentStatus =
  | 'PENDING'
  | 'INITIATED'
  | 'PAID'
  | 'SUCCESS'
  | 'FAILED'
  | 'PARTIALLY_REFUNDED'
  | 'REFUNDED';
```

In `homechrome-store/src/lib/utils.ts`, add a colour next to the `REFUNDED` entry (line 40):

```ts
  PARTIALLY_REFUNDED: 'gray',
  REFUNDED: 'gray',
```

- [ ] **Step 9: Verify both frontends**

```bash
cd handloom-admin-frontend && npm run check && npm run test
cd ../homechrome-store && npm run lint && npm run build
```
Expected: typecheck, lint, format and tests all clean; the storefront builds.

- [ ] **Step 10: Commit**

```bash
git add handloom-admin-frontend/src homechrome-store/src
git commit -m "feat(refunds): add the admin refund UI"
```

---

## Verification

Run after the last task, from the repository root:

```bash
cd handloom-admin
go build ./...
go test ./...                     # only cmd/embedder/embedder may fail (pre-existing)
golangci-lint run ./...
make wire && git diff --exit-code internal/wire/wire_gen.go   # wire_gen must be committed and current

cd ../handloom-admin-frontend && npm run check && npm run test
cd ../homechrome-store && npm run lint && npm run build
```

Manual end-to-end, locally, with the PhonePe DevClient (no credentials configured):

1. `cd handloom-admin && make setup-local && make run`
2. Place an order through the storefront and pay — the DevClient completes it.
3. As an ADMIN user, `POST /admin/orders/{id}/refunds` with one line, `restock: false`.
4. Confirm: a PENDING refund exists, `inventory_transactions` has a `WRITE_OFF` row for the order, and `reserved_qty` and `quantity` both dropped while `available_qty` held.
5. `POST /admin/orders/{id}/refunds/{refundID}/recheck` — the DevClient reports COMPLETED, the refund settles, the order goes `PARTIALLY_REFUNDED`, and `OrderItem.refunded_quantity` incremented.
6. Re-check again — nothing changes. That is the idempotency gate.
7. Refund the remaining lines — the order reaches `REFUNDED`, and the refund amounts sum exactly to `total_amount`.
8. As an OPERATOR user, the refund routes return 403 and the UI section is absent.

## Out of scope

Deliberately not built here; the design doc records the reasoning.

- Reversing a refund's inventory effect when the provider later fails it. A write-off reflects a physical fact that a payment failure does not change.
- Automatic retry of failed refunds. The admin creates a new one, which sidesteps provider-side idempotency entirely.
- Adjusting `Customer.TotalSpent`. It is documented as gross order value.
- Split payments. An order has one successful payment today.
- Customer-initiated refund requests, RMA, and automatic refunds on cancellation.
- The inventory-integrity backlog (`docs/superpowers/specs/2026-08-18-inventory-integrity-backlog.md`). Item 1 — ledger-keyed idempotency for order-scoped mutations — would also make `WriteOffStock` idempotent, but this plan does not depend on it: a refund's write-off happens once, at creation, behind a record that is created once.
