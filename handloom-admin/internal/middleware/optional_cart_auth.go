package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/response"
)

const guestSessionCookieName = "guest_session"
const guestSessionMaxAge = 30 * 24 * time.Hour

// OptionalCartAuth resolves cart identity without requiring authentication.
// Authenticated users get CustomerIDKey set; guests get GuestSessionKey set.
type OptionalCartAuth struct {
	customerAuthService domain.CustomerAuthService
}

// NewOptionalCartAuth creates a new OptionalCartAuth middleware.
func NewOptionalCartAuth(
	customerAuthService domain.CustomerAuthService,
) *OptionalCartAuth {
	return &OptionalCartAuth{
		customerAuthService: customerAuthService,
	}
}

// Resolve is the middleware handler that resolves cart identity.
func (m *OptionalCartAuth) Resolve(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 1. Try authenticated path: extract and validate store_token.
		if token, err := extractBearerToken(r, "store_token"); err == nil {
			claims, validateErr := m.customerAuthService.ValidateCustomerToken(ctx, token)
			if validateErr == nil && claims.CustomerID != "" {
				ctx = setCustomerContext(ctx, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			// Token is invalid/expired. If a refresh cookie exists, the user
			// intends to stay authenticated — return 401 so the frontend's
			// axios interceptor can refresh the token and retry.
			// If no refresh cookie, the session is over — fall through to guest.
			if _, refreshErr := r.Cookie("store_refresh"); refreshErr == nil {
				response.Unauthorized(w, "Token expired")
				return
			}
		}

		// 2. Try existing guest session cookie (must be a valid UUID)
		if cookie, err := r.Cookie(guestSessionCookieName); err == nil && cookie.Value != "" {
			if _, parseErr := uuid.Parse(cookie.Value); parseErr == nil {
				ctx = context.WithValue(ctx, GuestSessionKey, cookie.Value)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// 3. Generate new guest session
		sessionID := uuid.New().String()
		ctx = context.WithValue(ctx, GuestSessionKey, sessionID)

		secure, sameSite, cookieDomain := AuthCookieSettings()
		//nolint:gosec // G124: Secure flag is environment-conditional, not omitted.
		http.SetCookie(w, &http.Cookie{
			Name:     guestSessionCookieName,
			Value:    sessionID,
			Path:     "/",
			Domain:   cookieDomain,
			HttpOnly: true,
			Secure:   secure,
			SameSite: sameSite,
			MaxAge:   int(guestSessionMaxAge / time.Second),
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetCartIdentityFromContext returns the cart owner identifier and whether the user is a guest.
// Authenticated: (customerID, false). Guest: (sessionID, true).
func GetCartIdentityFromContext(ctx context.Context) (cartOwner string, isGuest bool) {
	if customerID, ok := ctx.Value(CustomerIDKey).(string); ok && customerID != "" {
		return customerID, false
	}
	if sessionID, ok := ctx.Value(GuestSessionKey).(string); ok && sessionID != "" {
		return sessionID, true
	}
	return "", true
}
