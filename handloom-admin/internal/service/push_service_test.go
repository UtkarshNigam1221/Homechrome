package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/webpush"
	"github.com/handloom/admin/internal/mocks"
	apperrors "github.com/handloom/admin/pkg/errors"
)

// fakeGateway records every delivery and replays a scripted error per endpoint.
type fakeGateway struct {
	mu        sync.Mutex
	sent      map[string][]byte
	failWith  map[string]error
	publicKey string
}

func newFakeGateway() *fakeGateway {
	return &fakeGateway{
		sent:      map[string][]byte{},
		failWith:  map[string]error{},
		publicKey: "test-public-key",
	}
}

func (f *fakeGateway) PublicKey() string { return f.publicKey }

func (f *fakeGateway) Send(_ context.Context, sub webpush.Subscription, payload []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err, ok := f.failWith[sub.Endpoint]; ok {
		return err
	}
	f.sent[sub.Endpoint] = payload
	return nil
}

func (f *fakeGateway) sentTo(endpoint string) ([]byte, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	payload, ok := f.sent[endpoint]
	return payload, ok
}

func (f *fakeGateway) sentCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sent)
}

func activeSub(endpoint string) *domain.PushSubscription {
	return &domain.PushSubscription{
		ID:       domain.PushEndpointID(endpoint),
		Endpoint: endpoint,
		Keys:     domain.PushSubscriptionKeys{P256dh: "p256dh-value", Auth: "auth-value"},
		Status:   domain.PushSubscriptionActive,
	}
}

func TestPushService_Subscribe(t *testing.T) {
	ctx := context.Background()

	t.Run("a new endpoint gets a welcome push", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockPushSubscriptionRepository(ctrl)
		gw := newFakeGateway()
		svc := NewPushService(repo, gw)

		repo.EXPECT().
			Save(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, sub *domain.PushSubscription) (bool, error) {
				assert.Equal(t, domain.PushSubscriptionActive, sub.Status)
				assert.Equal(t, "Mozilla/5.0", sub.UserAgent)
				return true, nil
			})

		sub, err := svc.Subscribe(ctx, domain.SubscribePushRequest{
			Endpoint: "https://push.example.com/a",
			Keys:     domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
		}, "Mozilla/5.0")

		require.NoError(t, err)
		assert.Equal(t, domain.PushSubscriptionActive, sub.Status)

		payload, ok := gw.sentTo("https://push.example.com/a")
		require.True(t, ok, "a first-time subscriber should receive the welcome push")

		var decoded domain.PushPayload
		require.NoError(t, json.Unmarshal(payload, &decoded))
		assert.Equal(t, "Welcome to Homechrome", decoded.Title)
		assert.Equal(t, defaultPushIcon, decoded.Icon, "an unset icon falls back to the storefront default")
	})

	t.Run("re-subscribing an existing endpoint sends nothing", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockPushSubscriptionRepository(ctrl)
		gw := newFakeGateway()
		svc := NewPushService(repo, gw)

		repo.EXPECT().Save(ctx, gomock.Any()).Return(false, nil)

		_, err := svc.Subscribe(ctx, domain.SubscribePushRequest{
			Endpoint: "https://push.example.com/a",
			Keys:     domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
		}, "")

		require.NoError(t, err)
		assert.Zero(t, gw.sentCount(), "a repeat subscribe must not re-send the welcome push")
	})

	t.Run("a failed welcome push does not fail the subscription", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockPushSubscriptionRepository(ctrl)
		gw := newFakeGateway()
		gw.failWith["https://push.example.com/a"] = errors.New("push service unreachable")
		svc := NewPushService(repo, gw)

		repo.EXPECT().Save(ctx, gomock.Any()).Return(true, nil)

		sub, err := svc.Subscribe(ctx, domain.SubscribePushRequest{
			Endpoint: "https://push.example.com/a",
			Keys:     domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
		}, "")

		require.NoError(t, err)
		assert.NotNil(t, sub)
	})
}

func TestPushService_SendTest(t *testing.T) {
	ctx := context.Background()

	t.Run("delivers only to the endpoint named in the request", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockPushSubscriptionRepository(ctrl)
		gw := newFakeGateway()
		svc := NewPushService(repo, gw)

		repo.EXPECT().
			GetByEndpoint(ctx, "https://push.example.com/mine").
			Return(activeSub("https://push.example.com/mine"), nil)

		require.NoError(t, svc.SendTest(ctx, "https://push.example.com/mine"))

		// The whole point of the endpoint-scoped test route: a storefront
		// visitor testing their own notifications must never fan out.
		assert.Equal(t, 1, gw.sentCount())
		_, ok := gw.sentTo("https://push.example.com/mine")
		assert.True(t, ok)
	})

	t.Run("refuses an endpoint that is not subscribed", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockPushSubscriptionRepository(ctrl)
		gw := newFakeGateway()
		svc := NewPushService(repo, gw)

		inactive := activeSub("https://push.example.com/dead")
		inactive.Status = domain.PushSubscriptionInactive
		repo.EXPECT().GetByEndpoint(ctx, "https://push.example.com/dead").Return(inactive, nil)

		err := svc.SendTest(ctx, "https://push.example.com/dead")

		require.Error(t, err)
		assert.Zero(t, gw.sentCount())
	})

	t.Run("propagates a missing subscription", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockPushSubscriptionRepository(ctrl)
		svc := NewPushService(repo, newFakeGateway())

		repo.EXPECT().
			GetByEndpoint(ctx, "https://push.example.com/gone").
			Return(nil, apperrors.NotFound("Push subscription"))

		err := svc.SendTest(ctx, "https://push.example.com/gone")

		require.Error(t, err)
		assert.True(t, apperrors.IsNotFound(err))
	})
}

func TestPushService_Broadcast(t *testing.T) {
	ctx := context.Background()

	t.Run("counts successes and failures and records the outcome", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockPushSubscriptionRepository(ctrl)
		gw := newFakeGateway()
		gw.failWith["https://push.example.com/b"] = errors.New("transient failure")
		svc := NewPushService(repo, gw)

		repo.EXPECT().ListActive(ctx).Return([]*domain.PushSubscription{
			activeSub("https://push.example.com/a"),
			activeSub("https://push.example.com/b"),
			activeSub("https://push.example.com/c"),
		}, nil)

		var recorded *domain.PushBroadcast
		repo.EXPECT().
			SaveBroadcast(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, b *domain.PushBroadcast) error {
				recorded = b
				return nil
			})

		result, err := svc.Broadcast(ctx, domain.BroadcastPushRequest{
			Title: "New arrivals",
			Body:  "Fresh Chanderi just landed",
		}, "admin_1")

		require.NoError(t, err)
		assert.Equal(t, 3, result.TotalTargeted)
		assert.Equal(t, 2, result.SuccessCount)
		assert.Equal(t, 1, result.FailureCount)
		assert.Equal(t, domain.PushBroadcastPartial, result.Status)

		require.NotNil(t, recorded)
		assert.Equal(t, "admin_1", recorded.SentBy)
		assert.Equal(t, "/", recorded.URL, "an unset click-through target defaults to the storefront root")
	})

	t.Run("deactivates endpoints the push service reports as gone", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockPushSubscriptionRepository(ctrl)
		gw := newFakeGateway()
		gw.failWith["https://push.example.com/dead"] = webpush.ErrSubscriptionGone
		svc := NewPushService(repo, gw)

		repo.EXPECT().ListActive(ctx).Return([]*domain.PushSubscription{
			activeSub("https://push.example.com/live"),
			activeSub("https://push.example.com/dead"),
		}, nil)

		// Without this, every future broadcast keeps retrying a dead endpoint.
		repo.EXPECT().Deactivate(ctx, "https://push.example.com/dead").Return(nil)
		repo.EXPECT().SaveBroadcast(ctx, gomock.Any()).Return(nil)

		result, err := svc.Broadcast(ctx, domain.BroadcastPushRequest{
			Title: "Sale", Body: "20% off",
		}, "admin_1")

		require.NoError(t, err)
		assert.Equal(t, 1, result.SuccessCount)
		assert.Equal(t, 1, result.FailureCount)
	})

	t.Run("no subscribers is a success, not a failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockPushSubscriptionRepository(ctrl)
		svc := NewPushService(repo, newFakeGateway())

		repo.EXPECT().ListActive(ctx).Return([]*domain.PushSubscription{}, nil)
		repo.EXPECT().SaveBroadcast(ctx, gomock.Any()).Return(nil)

		result, err := svc.Broadcast(ctx, domain.BroadcastPushRequest{
			Title: "Hello", Body: "World",
		}, "admin_1")

		require.NoError(t, err)
		assert.Equal(t, 0, result.TotalTargeted)
		assert.Equal(t, domain.PushBroadcastSuccess, result.Status)
	})

	t.Run("losing the audit row still reports the real delivery counts", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockPushSubscriptionRepository(ctrl)
		svc := NewPushService(repo, newFakeGateway())

		repo.EXPECT().ListActive(ctx).Return([]*domain.PushSubscription{
			activeSub("https://push.example.com/a"),
		}, nil)
		repo.EXPECT().SaveBroadcast(ctx, gomock.Any()).Return(errors.New("dynamo unavailable"))

		result, err := svc.Broadcast(ctx, domain.BroadcastPushRequest{
			Title: "Hello", Body: "World",
		}, "admin_1")

		// The push already went out — failing the call here would tell the
		// operator nothing was sent and invite a duplicate broadcast.
		require.NoError(t, err)
		assert.Equal(t, 1, result.SuccessCount)
	})

	t.Run("fans out beyond the concurrency limit", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockPushSubscriptionRepository(ctrl)
		gw := newFakeGateway()
		svc := NewPushService(repo, gw)

		total := broadcastConcurrency*2 + 5
		subs := make([]*domain.PushSubscription, 0, total)
		for i := range total {
			subs = append(subs, activeSub("https://push.example.com/"+string(rune('a'+i%26))+string(rune('0'+i/26))))
		}

		repo.EXPECT().ListActive(ctx).Return(subs, nil)
		repo.EXPECT().SaveBroadcast(ctx, gomock.Any()).Return(nil)

		result, err := svc.Broadcast(ctx, domain.BroadcastPushRequest{
			Title: "Hello", Body: "World",
		}, "admin_1")

		require.NoError(t, err)
		assert.Equal(t, total, result.TotalTargeted)
		assert.Equal(t, total, result.SuccessCount)
		assert.Equal(t, domain.PushBroadcastSuccess, result.Status)
	})
}

func TestPushEndpointID(t *testing.T) {
	a := domain.PushEndpointID("https://push.example.com/a")
	b := domain.PushEndpointID("https://push.example.com/b")

	assert.Len(t, a, 64, "a sha256 hex digest is 64 characters")
	assert.NotEqual(t, a, b)
	assert.Equal(t, a, domain.PushEndpointID("https://push.example.com/a"),
		"the same endpoint must hash to the same key, or re-subscribing duplicates the row")
}

func TestBroadcastStatus(t *testing.T) {
	tests := []struct {
		name                       string
		targeted, success, failure int
		want                       domain.PushBroadcastStatus
	}{
		{"no subscribers", 0, 0, 0, domain.PushBroadcastSuccess},
		{"all delivered", 3, 3, 0, domain.PushBroadcastSuccess},
		{"some delivered", 3, 2, 1, domain.PushBroadcastPartial},
		{"none delivered", 3, 0, 3, domain.PushBroadcastFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, broadcastStatus(tt.targeted, tt.success, tt.failure))
		})
	}
}
