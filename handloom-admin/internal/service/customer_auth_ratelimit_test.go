package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// realPhone is any number off the test allowlist, so it takes the paying path.
const realPhone = "+919812345678"

// The key function is the single owner of the exemption rule: the middleware
// claims on what it returns, and SendOTP picks the code by the same rule.

func TestOTPSendRateLimitKey_RealNumberIsLimited(t *testing.T) {
	svc, _, _ := newAuthServiceWithBypass(t, []string{allowlistedPhone}, testOTPCode)

	assert.Equal(t, "otp_send:"+realPhone, svc.OTPSendRateLimitKey(realPhone))
}

func TestOTPSendRateLimitKey_AllowlistedNumberIsExempt(t *testing.T) {
	svc, _, _ := newAuthServiceWithBypass(t, []string{allowlistedPhone}, testOTPCode)

	// Empty means exempt. An allowlisted number never reaches MSG91, so it
	// costs nothing and the e2e suite must be able to loop on it.
	assert.Empty(t, svc.OTPSendRateLimitKey(allowlistedPhone))
}

func TestOTPSendRateLimitKey_FailsClosedWhenTheBypassIsHalfConfigured(t *testing.T) {
	// Allowlist set, code not — isTestPhone reports false, so the number pays
	// for its SMS and must therefore be limited like any other.
	svc, _, _ := newAuthServiceWithBypass(t, []string{allowlistedPhone}, "")

	assert.Equal(t, "otp_send:"+allowlistedPhone, svc.OTPSendRateLimitKey(allowlistedPhone))
}
