package store

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/webpush"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/internal/validator"
	apperrors "github.com/handloom/admin/pkg/errors"
)

// newPushTestServer mounts the storefront push routes. An optional customer id
// stands in for what OptionalCustomer would have put on the context.
func newPushTestServer(
	t *testing.T, repo domain.PushSubscriptionRepository, customerID ...string,
) *httptest.Server {
	t.Helper()
	validation := middleware.NewValidation(validator.New(), middleware.ValidationConfig{})
	// These routes never broadcast, so the finalizer is wired but never called.
	finalizer := mocks.NewMockAssetFinalizer(gomock.NewController(t))
	svc := service.NewPushService(repo, webpush.NewDevClient(), finalizer)

	var routes http.Handler = NewPushHandler(svc, validation).Routes()
	if len(customerID) == 1 && customerID[0] != "" {
		routes = withCustomer(routes, customerID[0])
	}

	srv := httptest.NewServer(routes)
	t.Cleanup(srv.Close)
	return srv
}

// withCustomer injects a signed-in customer the way OptionalCustomer does.
func withCustomer(next http.Handler, customerID string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), middleware.CustomerIDKey, customerID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// anonymousLookup stands in for the read that stops an unauthenticated
// re-subscribe from clearing a link: here the endpoint is not stored yet.
func anonymousLookup(repo *mocks.MockPushSubscriptionRepository) {
	repo.EXPECT().GetByEndpoint(gomock.Any(), gomock.Any()).
		Return(nil, apperrors.NotFound("Push subscription")).AnyTimes()
}

// do sends req and returns the status and the fully-read body, so callers
// never hold an open response.
func do(t *testing.T, req *http.Request) (int, string) {
	t.Helper()
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(body)
}

func postJSON(t *testing.T, url string, body any) (int, string) {
	t.Helper()
	encoded, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(encoded))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	return do(t, req)
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	require.NoError(t, err)
	return do(t, req)
}

// The public router must expose no way to reach another visitor's device. An
// unauthenticated broadcast route was the defect this whole surface is shaped
// to avoid, so assert its absence rather than trusting the route list by eye.
func TestPushRoutes_ExposeNoBroadcastSurface(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	srv := newPushTestServer(t, repo)

	for _, path := range []string{"/broadcast", "/subscribers", "/broadcasts"} {
		t.Run(path, func(t *testing.T) {
			status, _ := postJSON(t, srv.URL+path, map[string]string{"title": "x", "body": "y"})
			assert.Equal(t, http.StatusNotFound, status,
				"%s must not exist on the public storefront router", path)
		})
	}

	// GET too — a subscriber dump is the other half of the same leak.
	status, _ := get(t, srv.URL+"/subscribers")
	assert.Equal(t, http.StatusNotFound, status)
}

func TestPushSubscribe(t *testing.T) {
	t.Run("stores a valid subscription", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockPushSubscriptionRepository(ctrl)
		anonymousLookup(repo)
		repo.EXPECT().
			Save(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, sub *domain.PushSubscription) (bool, error) {
				assert.Equal(t, "https://fcm.googleapis.com/fcm/send/abc", sub.Endpoint)
				assert.Equal(t, "p256dh-value", sub.Keys.P256dh)
				return false, nil
			})

		srv := newPushTestServer(t, repo)
		status, _ := postJSON(t, srv.URL+"/subscribe", domain.SubscribePushRequest{
			Endpoint: "https://fcm.googleapis.com/fcm/send/abc",
			Keys:     domain.PushSubscriptionKeys{P256dh: "p256dh-value", Auth: "auth-value"},
			Device:   &domain.PushDeviceInfo{Browser: "Chrome", OS: "macOS"},
		})

		require.Equal(t, http.StatusCreated, status)
	})

	t.Run("never echoes the subscription keys back", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockPushSubscriptionRepository(ctrl)
		anonymousLookup(repo)
		repo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(false, nil)

		srv := newPushTestServer(t, repo)
		status, body := postJSON(t, srv.URL+"/subscribe", domain.SubscribePushRequest{
			Endpoint: "https://fcm.googleapis.com/fcm/send/abc",
			Keys:     domain.PushSubscriptionKeys{P256dh: "p256dh-value", Auth: "auth-value"},
		})
		require.Equal(t, http.StatusCreated, status)

		// endpoint + keys together are the full credential to push to a device.
		assert.NotContains(t, body, "p256dh-value")
		assert.NotContains(t, body, "auth-value")
	})

	t.Run("rejects a payload with no keys", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockPushSubscriptionRepository(ctrl)
		srv := newPushTestServer(t, repo)

		status, _ := postJSON(t, srv.URL+"/subscribe", map[string]any{
			"endpoint": "https://fcm.googleapis.com/fcm/send/abc",
		})

		assert.Equal(t, http.StatusBadRequest, status)
	})

	t.Run("rejects an endpoint that is not a URL", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockPushSubscriptionRepository(ctrl)
		srv := newPushTestServer(t, repo)

		status, _ := postJSON(t, srv.URL+"/subscribe", map[string]any{
			"endpoint": "not-a-url",
			"keys":     map[string]string{"p256dh": "p", "auth": "a"},
		})

		assert.Equal(t, http.StatusBadRequest, status)
	})
}

func TestPushUnsubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	repo.EXPECT().Deactivate(gomock.Any(), "https://fcm.googleapis.com/fcm/send/abc").Return(nil)

	srv := newPushTestServer(t, repo)
	status, _ := postJSON(t, srv.URL+"/unsubscribe", domain.UnsubscribePushRequest{
		Endpoint: "https://fcm.googleapis.com/fcm/send/abc",
	})

	assert.Equal(t, http.StatusOK, status)
}

func TestPushUnlink(t *testing.T) {
	t.Run("an anonymous caller cannot unlink a device", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockPushSubscriptionRepository(ctrl)
		srv := newPushTestServer(t, repo)

		// No repo expectation: without a session this must not reach storage.
		status, _ := postJSON(t, srv.URL+"/unlink", domain.LinkPushRequest{
			Endpoint: "https://fcm.googleapis.com/fcm/send/abc",
		})
		assert.Equal(t, http.StatusUnauthorized, status)
	})

	t.Run("a signed-in caller detaches the device", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockPushSubscriptionRepository(ctrl)
		repo.EXPECT().
			UnlinkCustomer(gomock.Any(), "https://fcm.googleapis.com/fcm/send/abc").
			Return(nil)

		srv := newPushTestServer(t, repo, "cust_42")
		status, _ := postJSON(t, srv.URL+"/unlink", domain.LinkPushRequest{
			Endpoint: "https://fcm.googleapis.com/fcm/send/abc",
		})
		assert.Equal(t, http.StatusOK, status)
	})
}

func TestPushVapidKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	srv := newPushTestServer(t, repo)

	status, raw := get(t, srv.URL+"/vapid-key")
	require.Equal(t, http.StatusOK, status)

	var body struct {
		Data struct {
			PublicKey string `json:"public_key"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(raw), &body))

	// The dev gateway has no key, and the storefront reads the empty string as
	// "push is unavailable" and hides the opt-in rather than offering one that
	// can only fail.
	assert.Empty(t, body.Data.PublicKey)
}
