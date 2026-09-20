package router

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/gateway/webpush"
	"github.com/handloom/admin/internal/handler/store"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/internal/validator"
	apperrors "github.com/handloom/admin/pkg/errors"
)

const pushRouterJWTSecret = "store-push-router-test-secret"

// storeToken mints a real customer access token, signed with the secret the
// middleware validates against, so the cookie is parsed rather than faked.
func storeToken(t *testing.T, customerID string) string {
	t.Helper()
	now := time.Now()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   customerID,
		"phone": "+919000000000",
		"type":  "customer",
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}).SignedString([]byte(pushRouterJWTSecret))
	require.NoError(t, err)
	return token
}

// newStorePushServer mounts the storefront push routes exactly as the Lambda
// does, middleware included.
func newStorePushServer(t *testing.T, repo domain.PushSubscriptionRepository) *httptest.Server {
	t.Helper()
	ctrl := gomock.NewController(t)
	validation := middleware.NewValidation(validator.New(), middleware.ValidationConfig{})
	pushService := service.NewPushService(repo, webpush.NewDevClient(), mocks.NewMockAssetFinalizer(ctrl))
	authService := service.NewCustomerAuthService(nil, nil, nil, nil, service.CustomerAuthConfig{
		JWTSecret: pushRouterJWTSecret,
	})

	r := chi.NewMux()
	NewStorePushRouter(r,
		store.NewPushHandler(pushService, validation),
		middleware.NewCustomerAuth(authService))

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func postSubscribe(t *testing.T, srv *httptest.Server, token string) int {
	t.Helper()
	body, err := json.Marshal(domain.SubscribePushRequest{
		Endpoint: "https://fcm.googleapis.com/fcm/send/router",
		Keys:     domain.PushSubscriptionKeys{P256dh: "p", Auth: "a"},
	})
	require.NoError(t, err)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		srv.URL+"/api/v1/store/push/subscribe", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.AddCookie(&http.Cookie{Name: "store_token", Value: token})
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode
}

// The handler tests inject the context key by hand, so deleting the
// OptionalCustomer mount left them green while /subscribe linked nobody.
func TestNewStorePushRouter_LinksTheSignedInCustomer(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockPushSubscriptionRepository(ctrl)

	var saved *domain.PushSubscription
	repo.EXPECT().Save(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, sub *domain.PushSubscription) (bool, error) {
			saved = sub
			return false, nil
		})

	srv := newStorePushServer(t, repo)
	require.Equal(t, http.StatusCreated, postSubscribe(t, srv, storeToken(t, "cust_42")))
	require.NotNil(t, saved)
	require.Equal(t, "cust_42", saved.CustomerID)
}

// The same mount must still serve an anonymous visitor: the opt-in is offered
// before anyone signs in.
func TestNewStorePushRouter_StillServesAnonymousVisitors(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockPushSubscriptionRepository(ctrl)
	repo.EXPECT().GetByEndpoint(gomock.Any(), gomock.Any()).
		Return(nil, apperrors.NotFound("Push subscription"))

	var saved *domain.PushSubscription
	repo.EXPECT().Save(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, sub *domain.PushSubscription) (bool, error) {
			saved = sub
			return false, nil
		})

	srv := newStorePushServer(t, repo)
	require.Equal(t, http.StatusCreated, postSubscribe(t, srv, ""))
	require.NotNil(t, saved)
	require.Empty(t, saved.CustomerID)
}
