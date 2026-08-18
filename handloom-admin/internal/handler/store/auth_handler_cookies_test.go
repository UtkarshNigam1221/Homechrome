package store

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
)

func refreshCookieValue(t *testing.T, rec *httptest.ResponseRecorder) (string, bool) {
	t.Helper()

	for _, c := range rec.Result().Cookies() {
		if c.Name == cookieStoreRefresh {
			return c.Value, true
		}
	}
	return "", false
}

// Writing the straggler's empty refresh token would wipe the cookie the
// rotation winner just set — the very logout the grace window exists to stop.
func TestSetStoreCookies_LeavesRefreshCookieAloneWhenNoRefreshTokenIssued(t *testing.T) {
	h := &AuthHandler{}
	rec := httptest.NewRecorder()

	h.setStoreCookies(rec, &domain.TokenPair{AccessToken: "access-only"})

	_, ok := refreshCookieValue(t, rec)
	require.False(t, ok, "no refresh token issued means store_refresh must not be written at all")

	var access string
	for _, c := range rec.Result().Cookies() {
		if c.Name == cookieStoreToken {
			access = c.Value
		}
	}
	require.Equal(t, "access-only", access, "the access cookie must still be refreshed")
}

func TestSetStoreCookies_WritesRefreshCookieWhenTokenIssued(t *testing.T) {
	h := &AuthHandler{}
	rec := httptest.NewRecorder()

	h.setStoreCookies(rec, &domain.TokenPair{AccessToken: "access", RefreshToken: "refresh"})

	value, ok := refreshCookieValue(t, rec)
	require.True(t, ok)
	require.Equal(t, "refresh", value)
}
