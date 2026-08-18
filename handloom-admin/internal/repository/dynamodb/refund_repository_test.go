package dynamodb

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
)

func TestRefundRepository(t *testing.T) {
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testOrdersTable)

	repo := NewRefundRepository(wrapped)
	ctx := context.Background()

	newRefund := func(id, orderID string) *domain.Refund {
		return &domain.Refund{
			ID: id, OrderID: orderID, PaymentID: "pay_1", CustomerID: "cust_1",
			Amount: 2500, Status: domain.RefundStatusPending,
			Reason: domain.RefundReasonOutOfStock, MerchantRefundID: "mr_" + id,
			InitiatedAt: time.Now(),
		}
	}

	t.Run("round-trips a refund", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newRefund("ref_a", "order_1")))

		found, err := repo.GetByID(ctx, "ref_a")
		require.NoError(t, err)
		require.Equal(t, int64(2500), found.Amount)
		require.Equal(t, domain.RefundStatusPending, found.Status)
	})

	// A refund with no provider id yet must carry no GSI2 keys at all: DynamoDB
	// rejects an empty string on an indexed attribute, so blanks would fail the
	// write outright.
	t.Run("stores a refund that has not reached the provider yet", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newRefund("ref_b", "order_1")))

		found, err := repo.GetByID(ctx, "ref_b")
		require.NoError(t, err)
		require.Empty(t, found.ProviderRefundID)
	})

	t.Run("finds a refund by the provider id a webhook carries", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newRefund("ref_c", "order_2")))
		require.NoError(t, repo.SetProviderRefundID(ctx, "ref_c", "OMR_c"))

		found, err := repo.GetByProviderRefundID(ctx, "OMR_c")
		require.NoError(t, err)
		require.Equal(t, "ref_c", found.ID)
	})

	t.Run("reports nothing for an unknown provider id", func(t *testing.T) {
		_, err := repo.GetByProviderRefundID(ctx, "OMR_nope")
		require.Error(t, err)
	})

	// GSI1's ORDER# partition is shared with payments, so the listing has to
	// narrow or it returns them too.
	t.Run("lists only refunds for the order", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newRefund("ref_d", "order_3")))
		require.NoError(t, repo.Create(ctx, newRefund("ref_e", "order_3")))
		require.NoError(t, repo.Create(ctx, newRefund("ref_f", "order_4")))

		found, err := repo.ListByOrder(ctx, "order_3")
		require.NoError(t, err)
		require.Len(t, found, 2)
		for _, r := range found {
			require.Equal(t, "order_3", r.OrderID)
		}
	})

	t.Run("settles a pending refund", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newRefund("ref_g", "order_5")))

		require.NoError(t, repo.Settle(ctx, "ref_g", domain.RefundStatusCompleted, time.Now(), "", ""))

		found, err := repo.GetByID(ctx, "ref_g")
		require.NoError(t, err)
		require.Equal(t, domain.RefundStatusCompleted, found.Status)
		require.True(t, found.IsTerminal())
	})

	t.Run("records why a refund failed", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newRefund("ref_h", "order_6")))

		require.NoError(t, repo.Settle(ctx, "ref_h", domain.RefundStatusFailed, time.Now(),
			"REFUND_FAILED", "INSUFFICIENT_BALANCE"))

		found, err := repo.GetByID(ctx, "ref_h")
		require.NoError(t, err)
		require.Equal(t, domain.RefundStatusFailed, found.Status)
		require.Equal(t, "INSUFFICIENT_BALANCE", found.DetailedErrorCode)
	})

	// The gate the whole settlement hangs off: a second delivery must be
	// refused, not applied on top.
	t.Run("refuses to settle a refund twice", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newRefund("ref_i", "order_7")))

		require.NoError(t, repo.Settle(ctx, "ref_i", domain.RefundStatusCompleted, time.Now(), "", ""))
		err := repo.Settle(ctx, "ref_i", domain.RefundStatusCompleted, time.Now(), "", "")
		require.Error(t, err, "a replayed webhook must not settle an already-terminal refund")
	})

	// PhonePe retries and Lambda can process two deliveries at once.
	t.Run("exactly one of two concurrent settlements wins", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newRefund("ref_j", "order_8")))

		var wg sync.WaitGroup
		results := make([]error, 2)
		for i := range results {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				results[idx] = repo.Settle(ctx, "ref_j", domain.RefundStatusCompleted, time.Now(), "", "")
			}(i)
		}
		wg.Wait()

		won := 0
		for _, err := range results {
			if err == nil {
				won++
			}
		}
		require.Equal(t, 1, won, "the condition must pick exactly one winner")
	})
}
