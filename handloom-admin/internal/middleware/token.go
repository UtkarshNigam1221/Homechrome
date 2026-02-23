package middleware

import (
	"fmt"
	"net/http"
	"strings"
)

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
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	return parts[1], nil
}
