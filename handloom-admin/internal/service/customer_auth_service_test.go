package service

import (
	"context"
	stderrors "errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
)

// fakeCustomerTokenStore mimics the DynamoDB store's semantics — rows keyed by
// (customerID, tokenHash), TTL compared against wall clock — so rotation can be
// exercised for real instead of asserted against call expectations.
type fakeCustomerTokenStore struct {
	ttls map[string]int64 // customerID#tokenHash -> unix expiry

	// validateErr stands in for a throttled or timed-out GetItem. The real
	// store returns (false, wrapped-err) there, and callers must not read
	// that as "token revoked".
	validateErr error
}

func newFakeCustomerTokenStore() *fakeCustomerTokenStore {
	return &fakeCustomerTokenStore{ttls: map[string]int64{}}
}

// rowKey mirrors the real store's PK/SK pair (PK=CUST_TOKEN#<id>, SK=REFRESH_TOKEN#<hash>).
func rowKey(customerID, tokenHash string) string { return customerID + "#" + tokenHash }

func (f *fakeCustomerTokenStore) StoreToken(_ context.Context, customerID, tokenHash string, ttl int64) error {
	f.ttls[rowKey(customerID, tokenHash)] = ttl
	return nil
}

func (f *fakeCustomerTokenStore) ValidateToken(_ context.Context, customerID, tokenHash string) (bool, error) {
	if f.validateErr != nil {
		return false, f.validateErr
	}
	ttl, ok := f.ttls[rowKey(customerID, tokenHash)]
	if !ok {
		return false, nil
	}
	return ttl >= time.Now().Unix(), nil
}

func (f *fakeCustomerTokenStore) RevokeToken(_ context.Context, customerID, tokenHash string) error {
	delete(f.ttls, rowKey(customerID, tokenHash))
	return nil
}

func (f *fakeCustomerTokenStore) RevokeAllTokens(_ context.Context, customerID string) error {
	for k := range f.ttls {
		if strings.HasPrefix(k, customerID+"#") {
			delete(f.ttls, k)
		}
	}
	return nil
}

func (f *fakeCustomerTokenStore) RevokeTokensExpiringBefore(_ context.Context, customerID string, cutoff int64) error {
	prefix := customerID + "#"
	for k, ttl := range f.ttls {
		if strings.HasPrefix(k, prefix) && ttl <= cutoff {
			delete(f.ttls, k)
		}
	}
	return nil
}

var _ domain.CustomerTokenStore = (*fakeCustomerTokenStore)(nil)

func newCustomerAuthServiceForTest(t *testing.T) (*CustomerAuthService, *fakeCustomerTokenStore, *domain.Customer) {
	t.Helper()

	ctrl := gomock.NewController(t)
	customer := &domain.Customer{ID: "cust-1", Phone: "+919000000000", Status: domain.CustomerStatusActive}

	customerRepo := mocks.NewMockCustomerRepository(ctrl)
	customerRepo.EXPECT().GetByID(gomock.Any(), customer.ID).Return(customer, nil).AnyTimes()

	store := newFakeCustomerTokenStore()

	svc := NewCustomerAuthService(
		mocks.NewMockOTPRepository(ctrl),
		customerRepo,
		store,
		mocks.NewMockSMSGateway(ctrl),
		CustomerAuthConfig{
			JWTSecret:            "test-secret-key",
			AccessTokenDuration:  15 * time.Minute,
			RefreshTokenDuration: 7 * 24 * time.Hour,
			Issuer:               "test-issuer",
		},
	)

	return svc, store, customer
}

// A browser fires several authenticated requests at once; each 401s on the same
// expired access token and refreshes with the same refresh token. Rotation used
// to delete the old token immediately, so every straggler got "revoked" — and
// the handler clears both auth cookies on that path, logging the customer out.
// The straggler arrives after the winner has already rotated, which is what
// this models; DynamoDB serializes per-item, so the interesting ordering is
// sequential rather than genuinely simultaneous.
func TestCustomerAuthService_RefreshToken_RotatedTokenValidInGraceWindow(t *testing.T) {
	svc, store, customer := newCustomerAuthServiceForTest(t)
	ctx := context.Background()

	tokens, err := svc.generateTokenPair(customer)
	require.NoError(t, err)
	require.NoError(t, svc.storeRefreshToken(ctx, customer.ID, tokens.RefreshToken))

	// First refresh wins and rotates.
	_, first, err := svc.RefreshToken(ctx, tokens.RefreshToken)
	require.NoError(t, err)
	require.NotEqual(t, tokens.RefreshToken, first.RefreshToken)

	// Straggler still holding the pre-rotation token.
	_, second, err := svc.RefreshToken(ctx, tokens.RefreshToken)
	require.NoError(t, err, "overlapping refresh must not be rejected inside the grace window")
	require.NotEmpty(t, second.RefreshToken)

	// The rotated token is on a short leash, not kept for the full 7 days.
	oldTTL, ok := store.ttls[rowKey(customer.ID, hashSHA256(tokens.RefreshToken))]
	require.True(t, ok, "rotated token should be retained until its grace TTL")
	require.LessOrEqual(t, oldTTL, time.Now().Add(refreshGracePeriod).Unix())
	require.Greater(t, oldTTL, time.Now().Unix())
}

// The grace window must expire. Past it, a rotated token is dead.
func TestCustomerAuthService_RefreshToken_RejectedAfterGraceWindow(t *testing.T) {
	svc, store, customer := newCustomerAuthServiceForTest(t)
	ctx := context.Background()

	tokens, err := svc.generateTokenPair(customer)
	require.NoError(t, err)
	require.NoError(t, svc.storeRefreshToken(ctx, customer.ID, tokens.RefreshToken))

	_, _, err = svc.RefreshToken(ctx, tokens.RefreshToken)
	require.NoError(t, err)

	// Wind the grace TTL into the past rather than sleeping through it.
	store.ttls[rowKey(customer.ID, hashSHA256(tokens.RefreshToken))] = time.Now().Add(-time.Second).Unix()

	_, _, err = svc.RefreshToken(ctx, tokens.RefreshToken)
	require.Error(t, err, "a rotated token must stop working once its grace window lapses")
}

// The presented token is deleted outright, not put on the grace TTL rotation
// uses — logout is immediate for the token the customer is holding.
func TestCustomerAuthService_Logout_RevokesImmediately(t *testing.T) {
	svc, store, customer := newCustomerAuthServiceForTest(t)
	ctx := context.Background()

	tokens, err := svc.generateTokenPair(customer)
	require.NoError(t, err)
	require.NoError(t, svc.storeRefreshToken(ctx, customer.ID, tokens.RefreshToken))

	require.NoError(t, svc.Logout(ctx, customer.ID, tokens.RefreshToken))

	_, ok := store.ttls[rowKey(customer.ID, hashSHA256(tokens.RefreshToken))]
	require.False(t, ok, "logout must delete the token outright")

	_, _, err = svc.RefreshToken(ctx, tokens.RefreshToken)
	require.Error(t, err)
}

// Logout must also kill the predecessor left alive by a recent rotation. The
// caller only ever holds the successor, so revoking just that would leave a
// working refresh token behind for the rest of the grace window — after the
// customer explicitly asked to be logged out.
func TestCustomerAuthService_Logout_RevokesGracePredecessor(t *testing.T) {
	svc, store, customer := newCustomerAuthServiceForTest(t)
	ctx := context.Background()

	original, err := svc.generateTokenPair(customer)
	require.NoError(t, err)
	require.NoError(t, svc.storeRefreshToken(ctx, customer.ID, original.RefreshToken))

	_, rotated, err := svc.RefreshToken(ctx, original.RefreshToken)
	require.NoError(t, err)

	// Customer logs out holding only the rotated token.
	require.NoError(t, svc.Logout(ctx, customer.ID, rotated.RefreshToken))

	require.Empty(t, store.ttls, "logout must leave no usable token for the customer")

	_, _, err = svc.RefreshToken(ctx, original.RefreshToken)
	require.Error(t, err, "the pre-rotation token must not outlive logout")
}

// Logging out on one device must not end a session on another. A live
// session's TTL is the full 7-day refresh lifetime — far past the grace
// cutoff — so the sweep in Logout must leave it alone.
func TestCustomerAuthService_Logout_LeavesOtherDeviceSessionIntact(t *testing.T) {
	svc, _, customer := newCustomerAuthServiceForTest(t)
	ctx := context.Background()

	phone, err := svc.generateTokenPair(customer)
	require.NoError(t, err)
	require.NoError(t, svc.storeRefreshToken(ctx, customer.ID, phone.RefreshToken))

	laptop, err := svc.generateTokenPair(customer)
	require.NoError(t, err)
	require.NoError(t, svc.storeRefreshToken(ctx, customer.ID, laptop.RefreshToken))

	require.NoError(t, svc.Logout(ctx, customer.ID, phone.RefreshToken))

	_, _, err = svc.RefreshToken(ctx, phone.RefreshToken)
	require.Error(t, err, "the device that logged out must be logged out")

	_, _, err = svc.RefreshToken(ctx, laptop.RefreshToken)
	require.NoError(t, err, "an unrelated device's live session must survive another device's logout")
}

// A throttled or timed-out token lookup must surface as an infrastructure
// error, not as a revocation. The handler clears both auth cookies on terminal
// auth codes, so misclassifying here ends the session over a transient blip.
func TestCustomerAuthService_RefreshToken_StoreErrorIsNotRevocation(t *testing.T) {
	svc, store, customer := newCustomerAuthServiceForTest(t)
	ctx := context.Background()

	tokens, err := svc.generateTokenPair(customer)
	require.NoError(t, err)
	require.NoError(t, svc.storeRefreshToken(ctx, customer.ID, tokens.RefreshToken))

	store.validateErr = stderrors.New("dynamodb: ProvisionedThroughputExceededException")

	_, _, err = svc.RefreshToken(ctx, tokens.RefreshToken)
	require.Error(t, err)

	var appErr *errors.AppError
	if stderrors.As(err, &appErr) {
		require.NotEqual(t, errors.ErrCodeInvalidToken, appErr.Code,
			"a store failure must not be reported as a revoked token")
	}

	// The token is untouched — it works again once the store recovers.
	store.validateErr = nil
	_, _, err = svc.RefreshToken(ctx, tokens.RefreshToken)
	require.NoError(t, err)
}
