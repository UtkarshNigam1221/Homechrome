package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/errors"
)

var testRule = domain.RateLimitRule{Cooldown: time.Minute, Max: 3, Window: time.Hour}

// fakeLimiter records what it was asked, and refuses when told to.
type fakeLimiter struct {
	keys   []string
	refuse bool
}

func (f *fakeLimiter) Claim(_ context.Context, key string, _ domain.RateLimitRule) error {
	f.keys = append(f.keys, key)
	if f.refuse {
		return errors.New(errors.ErrCodeRateLimited, "Too many requests")
	}
	return nil
}

func serve(t *testing.T, lim domain.RateLimiter, key string) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	reached := false
	h := middleware.RateLimit(lim, testRule, func(context.Context) string { return key })(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			reached = true
			w.WriteHeader(http.StatusOK)
		}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/otp/send", nil))
	return rec, reached
}

func TestRateLimit_AllowedRequestReachesTheHandler(t *testing.T) {
	lim := &fakeLimiter{}

	rec, reached := serve(t, lim, "otp_send:+919812345678")

	assert.True(t, reached)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, []string{"otp_send:+919812345678"}, lim.keys)
}

func TestRateLimit_RefusedRequestNeverReachesTheHandler(t *testing.T) {
	lim := &fakeLimiter{refuse: true}

	rec, reached := serve(t, lim, "otp_send:+919812345678")

	// The whole point of claiming here rather than in the service: the
	// expensive work must not run at all.
	assert.False(t, reached, "a refused request must not reach the handler")
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimit_EmptyKeySkipsTheLimiterEntirely(t *testing.T) {
	lim := &fakeLimiter{refuse: true}

	// An empty key is the exemption signal — allowlisted test phones cost
	// nothing, so they must not consume or be refused by a window.
	rec, reached := serve(t, lim, "")

	require.True(t, reached, "an exempt request must reach the handler")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, lim.keys, "an exempt request must not touch the limiter")
}
