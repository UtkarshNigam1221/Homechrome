package dynamodb

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// isRateLimited separates the refusal from a transport or table error, which
// the caller must never swallow into a 429.
func isRateLimited(t *testing.T, err error) bool {
	t.Helper()
	appErr, ok := errors.AsAppError(err)
	return ok && appErr.Code == errors.ErrCodeRateLimited
}

func TestRateLimiter_Claim(t *testing.T) {
	wrappedClient, rawClient := testWrappedClient(t)
	skipIfNoLocal(t, rawClient)
	setupTestTable(t, rawClient, testSessionsTable)
	defer cleanupTestTable(t, rawClient, testSessionsTable)

	ctx := context.Background()

	// repoAt pins the clock, so the cooldown and the window can be crossed
	// without the test sleeping through them.
	base := time.Date(2026, 8, 21, 10, 0, 0, 0, time.UTC)
	repoAt := func(offset time.Duration) *RateLimiter {
		r := NewRateLimiter(wrappedClient)
		r.now = func() time.Time { return base.Add(offset) }
		return r
	}

	t.Run("first send for a phone is allowed", func(t *testing.T) {
		require.NoError(t, repoAt(0).Claim(ctx, "+919000000001", domain.OTPSendRule))
	})

	t.Run("a second send inside the cooldown is refused", func(t *testing.T) {
		phone := "+919000000002"
		require.NoError(t, repoAt(0).Claim(ctx, phone, domain.OTPSendRule))

		err := repoAt(domain.OTPSendRule.Cooldown-time.Second).Claim(ctx, phone, domain.OTPSendRule)
		require.Error(t, err)
		assert.True(t, isRateLimited(t, err), "got %v", err)

		// A refusal is the caller's fault, not ours: it must reach the client
		// as 429 so the storefront can show a wait message rather than a crash.
		appErr, ok := errors.AsAppError(err)
		require.True(t, ok)
		assert.Equal(t, http.StatusTooManyRequests, appErr.HTTPStatus)
	})

	t.Run("a send after the cooldown is allowed", func(t *testing.T) {
		phone := "+919000000003"
		require.NoError(t, repoAt(0).Claim(ctx, phone, domain.OTPSendRule))
		require.NoError(t, repoAt(domain.OTPSendRule.Cooldown).Claim(ctx, phone, domain.OTPSendRule))
	})

	t.Run("sends beyond the hourly cap are refused even after the cooldown", func(t *testing.T) {
		phone := "+919000000004"
		for i := range domain.OTPSendRule.Max {
			offset := time.Duration(i) * domain.OTPSendRule.Cooldown
			require.NoError(t, repoAt(offset).Claim(ctx, phone, domain.OTPSendRule), "send %d should be allowed", i+1)
		}

		over := time.Duration(domain.OTPSendRule.Max) * domain.OTPSendRule.Cooldown
		err := repoAt(over).Claim(ctx, phone, domain.OTPSendRule)
		require.Error(t, err)
		assert.True(t, isRateLimited(t, err), "got %v", err)
	})

	t.Run("each phone has its own window", func(t *testing.T) {
		require.NoError(t, repoAt(0).Claim(ctx, "+919000000005", domain.OTPSendRule))
		// A different number must be unaffected by the one above, or one
		// attacker locks out every customer.
		require.NoError(t, repoAt(0).Claim(ctx, "+919000000006", domain.OTPSendRule))
	})
}
