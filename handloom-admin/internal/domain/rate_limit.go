package domain

import (
	"context"
	"time"
)

// RateLimitRule is one caller's limits. Rules are values so the limiter never
// learns about its callers.
type RateLimitRule struct {
	// Cooldown is the minimum gap between claims. Zero disables it.
	Cooldown time.Duration
	// Max is how many claims a window allows.
	Max int
	// Window is fixed from the first claim of the window; it does not slide.
	Window time.Duration
}

// RateLimiter records uses of a rate-limited action.
type RateLimiter interface {
	// Claim records one use of key, returning an ErrCodeRateLimited error when
	// the cooldown or the window cap would be exceeded. Refusals cost nothing.
	Claim(ctx context.Context, key string, rule RateLimitRule) error
}

// OTPSendRule caps OTP sends per recipient. MSG91 bills per SMS and the send
// endpoint is public and pre-auth, so the limit that matters is per-number.
var OTPSendRule = RateLimitRule{
	Cooldown: OTPSendCooldown,
	Max:      OTPSendsPerWindow,
	Window:   OTPSendWindow,
}
