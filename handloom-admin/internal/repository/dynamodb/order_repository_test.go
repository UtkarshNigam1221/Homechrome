package dynamodb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
)

// Settlement writes through an UpdateExpression, and `items` is a DynamoDB reserved
// word — a mistake no mock can catch, because the parser is server-side.
func TestOrderRepository_ApplyRefundSettlement(t *testing.T) {
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testOrdersTable)

	repo := NewOrderRepository(wrapped)
	ctx := context.Background()

	seed := func(t *testing.T, id string, status domain.PaymentStatus) *domain.Order {
		t.Helper()
		order := &domain.Order{
			ID: id, OrderNumber: "HL-" + id, CustomerID: "cust_1",
			Status: domain.OrderStatusConfirmed, PaymentStatus: status,
			Items: []domain.OrderItem{
				{ID: "item_a", ProductID: "prod_a", Quantity: 2, UnitPrice: 10000},
			},
			Subtotal: 20000, TotalAmount: 20000,
		}
		require.NoError(t, repo.Create(ctx, order))
		return order
	}

	t.Run("writes the refunded quantities and the payment status", func(t *testing.T) {
		order := seed(t, "order_settle", domain.PaymentStatusPaid)
		order.Items[0].RefundedQuantity = 1

		require.NoError(t, repo.ApplyRefundSettlement(ctx, order.ID, order.Items,
			domain.PaymentStatusPartiallyRefunded))

		stored, err := repo.GetByID(ctx, order.ID)
		require.NoError(t, err)
		require.Equal(t, 1, stored.Items[0].RefundedQuantity)
		require.Equal(t, domain.PaymentStatusPartiallyRefunded, stored.PaymentStatus)
	})

	// The reason it is a targeted write: a whole-order PutItem would revert whatever
	// an admin changed between the read and the settlement.
	t.Run("leaves the fulfillment status alone", func(t *testing.T) {
		order := seed(t, "order_keeps_status", domain.PaymentStatusPaid)
		require.NoError(t, repo.UpdateStatus(ctx, order.ID, domain.OrderStatusShipped, "admin_1"))

		require.NoError(t, repo.ApplyRefundSettlement(ctx, order.ID, order.Items,
			domain.PaymentStatusPartiallyRefunded))

		stored, err := repo.GetByID(ctx, order.ID)
		require.NoError(t, err)
		require.Equal(t, domain.OrderStatusShipped, stored.Status, "the refund must not move it back")
	})

	// Two settlements can land in the opposite order to the totals they derived.
	t.Run("a partial settlement cannot overwrite a full one", func(t *testing.T) {
		order := seed(t, "order_no_regress", domain.PaymentStatusPaid)
		require.NoError(t, repo.ApplyRefundSettlement(ctx, order.ID, order.Items,
			domain.PaymentStatusRefunded))

		require.NoError(t, repo.ApplyRefundSettlement(ctx, order.ID, order.Items,
			domain.PaymentStatusPartiallyRefunded), "refused, not an error the webhook should see")

		stored, err := repo.GetByID(ctx, order.ID)
		require.NoError(t, err)
		require.Equal(t, domain.PaymentStatusRefunded, stored.PaymentStatus)
	})
}
