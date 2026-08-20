package dynamodb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
)

// GSI1's ORDER# partition holds payments and refunds. Descending, REFUND# sorts first,
// so a lookup leaning on a post-read filter finds nothing once a refund exists.
func TestPaymentRepository_GetByOrderID_WithRefundsPresent(t *testing.T) {
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testOrdersTable)

	payments := NewPaymentRepository(wrapped)
	refunds := NewRefundRepository(wrapped)
	ctx := context.Background()

	require.NoError(t, payments.Create(ctx, &domain.Payment{
		ID: "pay_1", OrderID: "order_1", CustomerID: "cust_1",
		Amount: 20000, Status: domain.PaymentStatusPaid,
		MerchantTransactionID: "txn_1", InitiatedAt: time.Now(),
	}))

	found, err := payments.GetByOrderID(ctx, "order_1")
	require.NoError(t, err)
	require.Equal(t, "pay_1", found.ID)

	require.NoError(t, refunds.Create(ctx, &domain.Refund{
		ID: "refund_1", OrderID: "order_1", PaymentID: "pay_1", CustomerID: "cust_1",
		Amount: 10000, Status: domain.RefundStatusPending,
		Reason: domain.RefundReasonOutOfStock, MerchantRefundID: "mref_1",
		InitiatedAt: time.Now(),
	}))

	found, err = payments.GetByOrderID(ctx, "order_1")
	require.NoError(t, err, "a refunded order must still find its payment")
	require.Equal(t, "pay_1", found.ID)
}
