package middleware

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/slogx"
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

		// 1. Try authenticated path: extract and validate store_token
		if token, err := extractBearerToken(r, "store_token"); err == nil {
			if claims, err := m.customerAuthService.ValidateCustomerToken(ctx, token); err == nil && claims.CustomerID != "" {
				ctx = context.WithValue(ctx, CustomerIDKey, claims.CustomerID)
				ctx = slogx.SetUserID(ctx, claims.CustomerID)
				ctx = context.WithValue(ctx, CustomerKey, &domain.Customer{
					ID:    claims.CustomerID,
					Phone: claims.Phone,
					Email: claims.Email,
				})
				next.ServeHTTP(w, r.WithContext(ctx))
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

		secure, sameSite, cookieDomain := guestCookieSettings()
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

// guestCookieSettings returns cookie settings matching store auth cookies.
// IMPORTANT: Must stay in sync with cookieSettings() in handler/store/auth_handler.go.
func guestCookieSettings() (secure bool, sameSite http.SameSite, domain string) {
	if d := os.Getenv("COOKIE_DOMAIN"); d != "" {
		return true, http.SameSiteLaxMode, d
	}
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		return true, http.SameSiteNoneMode, ""
	}
	return false, http.SameSiteLaxMode, ""
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
