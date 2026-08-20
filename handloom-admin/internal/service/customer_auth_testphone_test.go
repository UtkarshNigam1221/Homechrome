package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
)

const (
	allowlistedPhone = "+919999900001"
	testOTPCode      = "424242"
)

// recordingGateway fails the test if the SMS gateway is reached at all. The
// whole point of the bypass is that an allowlisted number costs nothing.
type recordingGateway struct{ calls int }

func (g *recordingGateway) SendOTP(context.Context, string, string) error {
	g.calls++
	return nil
}

func newAuthServiceWithBypass(t *testing.T, phones []string, otp string) (*CustomerAuthService, *mocks.MockOTPRepository, *recordingGateway) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	otpRepo := mocks.NewMockOTPRepository(ctrl)
	gw := &recordingGateway{}

	svc := NewCustomerAuthService(
		otpRepo,
		mocks.NewMockCustomerRepository(ctrl),
		mocks.NewMockCustomerTokenStore(ctrl),
		gw,
		CustomerAuthConfig{
			JWTSecret:  "test-secret",
			Issuer:     "handloom-store",
			TestPhones: phones,
			TestOTP:    otp,
		},
	)
	return svc, otpRepo, gw
}

func TestSendOTP_AllowlistedPhoneSkipsTheGateway(t *testing.T) {
	svc, otpRepo, gw := newAuthServiceWithBypass(t, []string{allowlistedPhone}, testOTPCode)

	var stored *domain.OTP
	otpRepo.EXPECT().Store(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, o *domain.OTP) error { stored = o; return nil })

	require.NoError(t, svc.SendOTP(context.Background(), allowlistedPhone))

	assert.Equal(t, 0, gw.calls, "an allowlisted phone must never reach the SMS gateway")
	require.NotNil(t, stored)
	assert.Equal(t, hashSHA256(testOTPCode), stored.CodeHash,
		"the stored hash must be of the fixed test code, so the suite can verify")
	assert.Equal(t, 0, stored.Attempts, "attempt limits stay the production path")
}

func TestSendOTP_NonAllowlistedPhoneStillSends(t *testing.T) {
	svc, otpRepo, gw := newAuthServiceWithBypass(t, []string{allowlistedPhone}, testOTPCode)

	var stored *domain.OTP
	otpRepo.EXPECT().Store(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, o *domain.OTP) error { stored = o; return nil })

	require.NoError(t, svc.SendOTP(context.Background(), "+919812345678"))

	assert.Equal(t, 1, gw.calls, "a real number must still go through MSG91")
	require.NotNil(t, stored)
	assert.NotEqual(t, hashSHA256(testOTPCode), stored.CodeHash,
		"a real number must get a random code, never the test one")
}

func TestSendOTP_MatchIsExactNotPrefix(t *testing.T) {
	svc, otpRepo, gw := newAuthServiceWithBypass(t, []string{allowlistedPhone}, testOTPCode)

	otpRepo.EXPECT().Store(gomock.Any(), gomock.Any()).Return(nil)

	// One trailing digit: a prefix match would let an arbitrary real number in.
	require.NoError(t, svc.SendOTP(context.Background(), allowlistedPhone+"7"))
	assert.Equal(t, 1, gw.calls, "allowlist must match the full E.164 string only")
}

func TestSendOTP_EmptyTestOTPFailsClosed(t *testing.T) {
	// Allowlist set, code not. A half-configured environment must not hand out
	// a guessable or empty OTP — it must behave as though the bypass is off.
	svc, otpRepo, gw := newAuthServiceWithBypass(t, []string{allowlistedPhone}, "")

	otpRepo.EXPECT().Store(gomock.Any(), gomock.Any()).Return(nil)

	require.NoError(t, svc.SendOTP(context.Background(), allowlistedPhone))
	assert.Equal(t, 1, gw.calls, "no test code configured means no bypass")
}

func TestConfigValidate_RefusesTheBypassInProduction(t *testing.T) {
	for _, env := range []string{"prod", "production"} {
		t.Run(env, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.App.Environment = env
			cfg.Store.TestPhones = []string{allowlistedPhone}
			cfg.Store.TestOTP = testOTPCode

			err := cfg.Validate()
			require.Error(t, err, "production must refuse to start with the bypass configured")
			assert.Contains(t, err.Error(), "STORE_TEST_PHONES")
		})
	}
}

func TestConfigValidate_AllowsTheBypassOutsideProduction(t *testing.T) {
	cfg := &config.Config{}
	cfg.App.Environment = "dev"
	cfg.Store.TestPhones = []string{allowlistedPhone}
	cfg.Store.TestOTP = testOTPCode

	require.NoError(t, cfg.Validate())
}
