package dynamodb

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
)

func newPushRepo(t *testing.T) (*PushSubscriptionRepository, *dynamodb.Client) {
	t.Helper()
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testNotificationsTable)
	t.Cleanup(func() { cleanupTestTable(t, raw, testNotificationsTable) })
	return NewPushSubscriptionRepository(wrapped), raw
}

func TestListByCustomer(t *testing.T) {
	repo, _ := newPushRepo(t)
	ctx := context.Background()
	mine := &domain.PushSubscription{
		Endpoint:   "https://fcm.googleapis.com/fcm/send/mine",
		Keys:       domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
		CustomerID: "cust_1",
		Status:     domain.PushSubscriptionActive,
		CreatedAt:  time.Now(),
	}
	theirs := &domain.PushSubscription{
		Endpoint:   "https://fcm.googleapis.com/fcm/send/theirs",
		Keys:       domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
		CustomerID: "cust_2",
		Status:     domain.PushSubscriptionActive,
		CreatedAt:  time.Now(),
	}
	anonymous := &domain.PushSubscription{
		Endpoint:  "https://fcm.googleapis.com/fcm/send/anon",
		Keys:      domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
		Status:    domain.PushSubscriptionActive,
		CreatedAt: time.Now(),
	}
	for _, sub := range []*domain.PushSubscription{mine, theirs, anonymous} {
		_, err := repo.Save(ctx, sub)
		require.NoError(t, err)
	}

	t.Run("returns only that customer's devices", func(t *testing.T) {
		got, err := repo.ListByCustomer(ctx, "cust_1")
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, mine.Endpoint, got[0].Endpoint)
	})

	t.Run("a customer with no devices is empty, not an error", func(t *testing.T) {
		got, err := repo.ListByCustomer(ctx, "cust_nobody")
		require.NoError(t, err)
		require.Empty(t, got)
	})

	// A device that unsubscribed must not receive an order update; leaving the
	// pointer behind while filtering on read keeps deactivation a single write.
	t.Run("skips a device that has since unsubscribed", func(t *testing.T) {
		require.NoError(t, repo.Deactivate(ctx, mine.Endpoint))
		got, err := repo.ListByCustomer(ctx, "cust_1")
		require.NoError(t, err)
		require.Empty(t, got)
	})
}
