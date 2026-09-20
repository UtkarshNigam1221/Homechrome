package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/webpush"
	"github.com/handloom/admin/internal/middleware"
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
		svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

		repo.EXPECT().
			Save(ctx, gomock.Any()).
			DoAndReturn(func(_ context.Context, sub *domain.PushSubscription) (bool, error) {
				assert.Equal(t, domain.PushSubscriptionActive, sub.Status)
				assert.Equal(t, "Mozilla/5.0", sub.UserAgent)
				return true, nil
			})

		sub, err := svc.Subscribe(ctx, domain.SubscribePushRequest{
			Endpoint: "https://fcm.googleapis.com/fcm/send/a",
			Keys:     domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
		}, "Mozilla/5.0")

		require.NoError(t, err)
		assert.Equal(t, domain.PushSubscriptionActive, sub.Status)

		payload, ok := gw.sentTo("https://fcm.googleapis.com/fcm/send/a")
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
		svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

		repo.EXPECT().Save(ctx, gomock.Any()).Return(false, nil)

		_, err := svc.Subscribe(ctx, domain.SubscribePushRequest{
			Endpoint: "https://fcm.googleapis.com/fcm/send/a",
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
		gw.failWith["https://fcm.googleapis.com/fcm/send/a"] = errors.New("push service unreachable")
		svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

		repo.EXPECT().Save(ctx, gomock.Any()).Return(true, nil)

		sub, err := svc.Subscribe(ctx, domain.SubscribePushRequest{
			Endpoint: "https://fcm.googleapis.com/fcm/send/a",
			Keys:     domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
		}, "")

		require.NoError(t, err)
		assert.NotNil(t, sub)
	})
}

func TestPushService_SubscribeEndpointAllowlist(t *testing.T) {
	ctx := context.Background()

	accepted := []string{
		"https://fcm.googleapis.com/fcm/send/abc",
		"https://android.googleapis.com/gcm/send/abc",
		"https://updates.push.services.mozilla.com/wpush/v2/abc",
		"https://wns2-par02p.notify.windows.com/w/?token=abc",
		"https://web.push.apple.com/abc",
		"https://FCM.googleapis.com/fcm/send/abc",
	}

	// Each of these would otherwise have the backend open a VAPID-signed
	// connection to a host the anonymous caller picked.
	rejected := []string{
		"http://fcm.googleapis.com/fcm/send/abc", // downgraded to plaintext
		"https://127.0.0.1:8080/admin",
		"https://169.254.169.254/latest/meta-data/",
		"https://internal-svc.local/",
		"https://evil.example.com/fcm.googleapis.com",
		"https://fcm.googleapis.com.evil.example.com/abc", // suffix must be on a label boundary
		"ftp://fcm.googleapis.com/abc",
		"",
		// Userinfo is dropped when Go dials, so these would all reach one real
		// device under distinct row keys — unbounded duplicates of one victim.
		"https://anything@fcm.googleapis.com/fcm/send/abc",
		"https://a:b@fcm.googleapis.com/fcm/send/abc",
		"https://fcm.googleapis.com:8443/fcm/send/abc",
		"https://fcm.googleapis.com/fcm/send/" + strings.Repeat("a", 512),
	}

	req := func(endpoint string) domain.SubscribePushRequest {
		return domain.SubscribePushRequest{
			Endpoint: endpoint,
			Keys:     domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
		}
	}

	for _, endpoint := range accepted {
		t.Run("accepts "+endpoint, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mocks.NewMockPushSubscriptionRepository(ctrl)
			repo.EXPECT().Save(ctx, gomock.Any()).Return(false, nil)

			_, err := NewPushService(repo, newFakeGateway(), mocks.NewMockAssetFinalizer(ctrl)).Subscribe(ctx, req(endpoint), "")
			require.NoError(t, err)
		})
	}

	for _, endpoint := range rejected {
		t.Run("rejects "+endpoint, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// No Save, no Send: a rejected endpoint must not be stored or reached.
			repo := mocks.NewMockPushSubscriptionRepository(ctrl)
			gw := newFakeGateway()

			_, err := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl)).Subscribe(ctx, req(endpoint), "")

			var appErr *apperrors.AppError
			require.ErrorAs(t, err, &appErr)
			assert.Equal(t, apperrors.ErrCodeValidation, appErr.Code)
			assert.Zero(t, gw.sentCount())
		})
	}
}

func TestPushService_SendTest(t *testing.T) {
	ctx := context.Background()

	t.Run("delivers only to the endpoint named in the request", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mocks.NewMockPushSubscriptionRepository(ctrl)
		gw := newFakeGateway()
		svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

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
		svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

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
		svc := NewPushService(repo, newFakeGateway(), mocks.NewMockAssetFinalizer(ctrl))

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
		finalizer := mocks.NewMockAssetFinalizer(ctrl)
		finalizer.EXPECT().FinalizeIfTemp(gomock.Any(), "").Return("", nil).AnyTimes()
		svc := NewPushService(repo, gw, finalizer)

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
		finalizer := mocks.NewMockAssetFinalizer(ctrl)
		finalizer.EXPECT().FinalizeIfTemp(gomock.Any(), "").Return("", nil).AnyTimes()
		svc := NewPushService(repo, gw, finalizer)

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
		finalizer := mocks.NewMockAssetFinalizer(ctrl)
		finalizer.EXPECT().FinalizeIfTemp(gomock.Any(), "").Return("", nil).AnyTimes()
		svc := NewPushService(repo, newFakeGateway(), finalizer)

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
		finalizer := mocks.NewMockAssetFinalizer(ctrl)
		finalizer.EXPECT().FinalizeIfTemp(gomock.Any(), "").Return("", nil).AnyTimes()
		svc := NewPushService(repo, newFakeGateway(), finalizer)

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
		finalizer := mocks.NewMockAssetFinalizer(ctrl)
		finalizer.EXPECT().FinalizeIfTemp(gomock.Any(), "").Return("", nil).AnyTimes()
		svc := NewPushService(repo, gw, finalizer)

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

func TestPushService_BroadcastFinalizesImages(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	gw := newFakeGateway()
	finalizer := mocks.NewMockAssetFinalizer(ctrl)

	repo.EXPECT().ListActive(ctx).Return([]*domain.PushSubscription{
		activeSub("https://fcm.googleapis.com/fcm/send/a"),
	}, nil)
	repo.EXPECT().SaveBroadcast(ctx, gomock.Any()).Return(nil)

	// A tmp key must become a permanent URL before it reaches a device: tmp/
	// is deleted after a day, and the notification outlives that.
	finalizer.EXPECT().
		FinalizeIfTemp(ctx, "tmp/image/banner.jpg").
		Return("https://cdn.homechrome.in/assets/image/banner.jpg", nil)
	finalizer.EXPECT().
		FinalizeIfTemp(ctx, "tmp/image/thumb.jpg").
		Return("https://cdn.homechrome.in/assets/image/thumb.jpg", nil)

	svc := NewPushService(repo, gw, finalizer)

	result, err := svc.Broadcast(ctx, domain.BroadcastPushRequest{
		Title: "Festive drop",
		Body:  "Now live.",
		Image: "tmp/image/banner.jpg",
		Icon:  "tmp/image/thumb.jpg",
	}, "usr_1")
	require.NoError(t, err)

	payload, ok := gw.sentTo("https://fcm.googleapis.com/fcm/send/a")
	require.True(t, ok)

	var decoded domain.PushPayload
	require.NoError(t, json.Unmarshal(payload, &decoded))
	assert.Equal(t, "https://cdn.homechrome.in/assets/image/banner.jpg", decoded.Image,
		"a device cannot fetch a tmp/ key")
	assert.Equal(t, "https://cdn.homechrome.in/assets/image/thumb.jpg", decoded.Icon)

	// History must record the permanent URLs too, or the audit row rots in a day.
	assert.Equal(t, "https://cdn.homechrome.in/assets/image/banner.jpg", result.Broadcast.Image)
	assert.Equal(t, "https://cdn.homechrome.in/assets/image/thumb.jpg", result.Broadcast.Icon)
}

func TestPushService_BroadcastFailsWhenFinalizeFails(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	finalizer := mocks.NewMockAssetFinalizer(ctrl)

	repo.EXPECT().ListActive(ctx).Return([]*domain.PushSubscription{
		activeSub("https://fcm.googleapis.com/fcm/send/a"),
	}, nil)
	finalizer.EXPECT().
		FinalizeIfTemp(ctx, "tmp/image/banner.jpg").
		Return("", errors.New("s3 copy failed"))

	svc := NewPushService(repo, newFakeGateway(), finalizer)

	// Better to refuse than to fan out to every device with a dead image.
	_, err := svc.Broadcast(ctx, domain.BroadcastPushRequest{
		Title: "Festive drop", Body: "Now live.", Image: "tmp/image/banner.jpg",
	}, "usr_1")
	require.Error(t, err)
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

func TestSubscribeRecordsTheSignedInCustomer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	gw := newFakeGateway()
	svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

	var saved *domain.PushSubscription
	repo.EXPECT().Save(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, sub *domain.PushSubscription) (bool, error) {
			saved = sub
			return true, nil
		})

	ctx := context.WithValue(context.Background(), middleware.CustomerIDKey, "cust_42")
	_, err := svc.Subscribe(ctx, domain.SubscribePushRequest{
		Endpoint: "https://fcm.googleapis.com/fcm/send/abc",
		Keys:     domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
	}, "Mozilla/5.0")

	require.NoError(t, err)
	require.Equal(t, "cust_42", saved.CustomerID)
}

func TestSubscribeWithoutASignedInCustomerStaysAnonymous(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	gw := newFakeGateway()
	svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

	var saved *domain.PushSubscription
	repo.EXPECT().Save(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, sub *domain.PushSubscription) (bool, error) {
			saved = sub
			return true, nil
		})

	_, err := svc.Subscribe(context.Background(), domain.SubscribePushRequest{
		Endpoint: "https://fcm.googleapis.com/fcm/send/abc",
		Keys:     domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
	}, "Mozilla/5.0")

	require.NoError(t, err)
	require.Empty(t, saved.CustomerID)
}

func TestLinkCustomerRequiresASignedInCustomer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	gw := newFakeGateway()
	svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

	// No customer in context: nothing may be written, or an anonymous caller
	// could claim another shopper's device.
	err := svc.LinkCustomer(context.Background(), "https://fcm.googleapis.com/fcm/send/abc")
	require.Error(t, err)
}

func TestLinkCustomerAttachesTheDevice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	gw := newFakeGateway()
	svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

	repo.EXPECT().
		LinkCustomer(gomock.Any(), "https://fcm.googleapis.com/fcm/send/abc", "cust_42").
		Return(nil)

	ctx := context.WithValue(context.Background(), middleware.CustomerIDKey, "cust_42")
	require.NoError(t, svc.LinkCustomer(ctx, "https://fcm.googleapis.com/fcm/send/abc"))
}

func TestLinkCustomerRejectsAnUnknownPushService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	gw := newFakeGateway()
	svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

	ctx := context.WithValue(context.Background(), middleware.CustomerIDKey, "cust_42")
	err := svc.LinkCustomer(ctx, "https://evil.example.com/hook")
	require.Error(t, err)
}

func TestNotifyCustomerReachesEveryDevice(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	gw := newFakeGateway()
	svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

	repo.EXPECT().ListByCustomer(gomock.Any(), "cust_1").Return([]*domain.PushSubscription{
		{ID: "a", Endpoint: "https://fcm.googleapis.com/fcm/send/a", Status: domain.PushSubscriptionActive},
		{ID: "b", Endpoint: "https://fcm.googleapis.com/fcm/send/b", Status: domain.PushSubscriptionActive},
	}, nil)

	delivered, err := svc.NotifyCustomer(context.Background(), "cust_1", domain.PushPayload{
		Title: "Your order has shipped",
		Body:  "HL-1 is on its way.",
		URL:   "/account/orders/order_1",
	})

	require.NoError(t, err)
	require.Equal(t, 2, delivered)
	require.Equal(t, 2, gw.sentCount())
}

func TestNotifyCustomerWithNoDevicesIsNotAnError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	gw := newFakeGateway()
	svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

	repo.EXPECT().ListByCustomer(gomock.Any(), "cust_1").Return(nil, nil)

	delivered, err := svc.NotifyCustomer(context.Background(), "cust_1", domain.PushPayload{
		Title: "Your order has shipped",
		Body:  "HL-1 is on its way.",
	})

	require.NoError(t, err)
	require.Zero(t, delivered)
}

func TestNotifyCustomerRequiresACustomer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	gw := newFakeGateway()
	svc := NewPushService(repo, gw, mocks.NewMockAssetFinalizer(ctrl))

	// No ListByCustomer expectation: an empty id must never become a query that
	// could match the anonymous partition.
	_, err := svc.NotifyCustomer(context.Background(), "", domain.PushPayload{Title: "x", Body: "y"})
	require.Error(t, err)
}
