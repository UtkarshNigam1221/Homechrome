package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// AuthCookieSettings returns the Secure, SameSite, and Domain values for all
// auth cookies (admin, store, and guest-session). Single source of truth so
// the attributes can't drift between login handlers and the cart middleware.
//   - COOKIE_DOMAIN set (custom domain, same-site): Secure + Lax + Domain
//   - Lambda without custom domain (cross-origin): Secure + None (third-party cookies)
//   - Local dev: insecure + Lax (Vite proxy, same-origin)
func AuthCookieSettings() (secure bool, sameSite http.SameSite, domain string) {
	if d := os.Getenv("COOKIE_DOMAIN"); d != "" {
		return true, http.SameSiteLaxMode, d
	}
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		return true, http.SameSiteNoneMode, ""
	}
	return false, http.SameSiteLaxMode, ""
}

// extractBearerToken reads a JWT from the named cookie, falling back to the Authorization header.
func extractBearerToken(r *http.Request, cookieName string) (string, error) {
	if cookie, err := r.Cookie(cookieName); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authentication required")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", fmt.Errorf("invalid authorization header format")
	}

	return parts[1], nil
}
