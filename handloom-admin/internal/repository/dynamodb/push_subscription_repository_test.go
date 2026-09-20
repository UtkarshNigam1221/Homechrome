package dynamodb

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
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

// pointerExists reports whether the PUSH_CUST# row that makes a device
// reachable from ListByCustomer is still in the table.
func pointerExists(t *testing.T, raw *dynamodb.Client, customerID, endpoint string) bool {
	t.Helper()
	out, err := raw.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(testNotificationsTable),
		Key:       custPointerKey(customerID, domain.PushEndpointID(endpoint)),
	})
	require.NoError(t, err)
	return out.Item != nil
}

func TestUnlinkCustomer(t *testing.T) {
	repo, raw := newPushRepo(t)
	ctx := context.Background()
	endpoint := "https://fcm.googleapis.com/fcm/send/shared-tablet"

	_, err := repo.Save(ctx, &domain.PushSubscription{
		Endpoint:   endpoint,
		Keys:       domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
		CustomerID: "cust_a",
		Status:     domain.PushSubscriptionActive,
		CreatedAt:  time.Now(),
	})
	require.NoError(t, err)
	require.True(t, pointerExists(t, raw, "cust_a", endpoint))

	require.NoError(t, repo.UnlinkCustomer(ctx, endpoint))

	// Both halves, not just one: the field alone leaves ListByCustomer still
	// reaching the device, and the pointer alone leaves the next re-subscribe
	// re-creating it.
	sub, getErr := repo.GetByEndpoint(ctx, endpoint)
	require.NoError(t, getErr)
	require.Empty(t, sub.CustomerID)
	require.False(t, pointerExists(t, raw, "cust_a", endpoint))

	got, listErr := repo.ListByCustomer(ctx, "cust_a")
	require.NoError(t, listErr)
	require.Empty(t, got)

	t.Run("unlinking an unknown endpoint is a no-op", func(t *testing.T) {
		require.NoError(t, repo.UnlinkCustomer(ctx, "https://fcm.googleapis.com/fcm/send/nothing"))
	})

	t.Run("unlinking an already anonymous device is a no-op", func(t *testing.T) {
		require.NoError(t, repo.UnlinkCustomer(ctx, endpoint))
	})
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

	t.Run("re-linking a device to another customer moves it, not copies it", func(t *testing.T) {
		// A shared device: the first customer must stop receiving notifications
		// for it the moment the second signs in.
		shared := &domain.PushSubscription{
			Endpoint:   "https://fcm.googleapis.com/fcm/send/shared",
			Keys:       domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
			CustomerID: "cust_a",
			Status:     domain.PushSubscriptionActive,
			CreatedAt:  time.Now(),
		}
		_, err := repo.Save(ctx, shared)
		require.NoError(t, err)

		shared.CustomerID = "cust_b"
		_, err = repo.Save(ctx, shared)
		require.NoError(t, err)

		gotB, err := repo.ListByCustomer(ctx, "cust_b")
		require.NoError(t, err)
		require.Len(t, gotB, 1)
		require.Equal(t, shared.Endpoint, gotB[0].Endpoint)

		gotA, err := repo.ListByCustomer(ctx, "cust_a")
		require.NoError(t, err)
		require.Empty(t, gotA)
	})

	t.Run("a sign-out in between must not let the old owner survive the hand-off", func(t *testing.T) {
		// Shopper A links, signs out (which unlinks), then Shopper B links the
		// same device — A's pointer must not outlive either step.
		handoff := &domain.PushSubscription{
			Endpoint:   "https://fcm.googleapis.com/fcm/send/handoff",
			Keys:       domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
			CustomerID: "cust_handoff_a",
			Status:     domain.PushSubscriptionActive,
			CreatedAt:  time.Now(),
		}
		_, err := repo.Save(ctx, handoff)
		require.NoError(t, err)

		require.NoError(t, repo.UnlinkCustomer(ctx, handoff.Endpoint))

		handoff.CustomerID = "cust_handoff_b"
		_, err = repo.Save(ctx, handoff)
		require.NoError(t, err)

		gotB, err := repo.ListByCustomer(ctx, "cust_handoff_b")
		require.NoError(t, err)
		require.Len(t, gotB, 1)
		require.Equal(t, handoff.Endpoint, gotB[0].Endpoint)

		gotA, err := repo.ListByCustomer(ctx, "cust_handoff_a")
		require.NoError(t, err)
		require.Empty(t, gotA)
	})
}
