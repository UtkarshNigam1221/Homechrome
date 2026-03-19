// Package middleware provides HTTP middleware
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/response"
	"github.com/handloom/admin/pkg/slogx"
)

// Use exported ContextKey constants from interfaces.go for all context keys.
// This ensures SetUserInContext and GetUserIDFromContext use the same key type.

// RequestID adds a unique request ID to the context
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		ctx = slogx.SetRequestID(ctx, requestID)
		w.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logger logs HTTP requests
func Logger() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(ww, r)

			slog.InfoContext(r.Context(), "HTTP request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.statusCode,
				"duration", time.Since(start).String(),
				"user_agent", r.UserAgent(),
				"remote_ip", getRemoteIP(r),
			)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// Recoverer recovers from panics and returns a 500 error
func Recoverer() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					slog.ErrorContext(r.Context(), "Panic recovered", "panic", err)

					response.InternalError(w)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// Auth provides JWT authentication middleware
type Auth struct {
	authService domain.AuthService
}

// NewAuth creates a new Auth middleware
func NewAuth(authService domain.AuthService) *Auth {
	return &Auth{
		authService: authService,
	}
}

// Authenticate validates JWT token and sets user in context
func (a *Auth) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := extractBearerToken(r, "access_token")
		if err != nil {
			response.Unauthorized(w, err.Error())
			return
		}

		// Validate token
		claims, err := a.authService.ValidateToken(r.Context(), token)
		if err != nil {
			response.Unauthorized(w, "Invalid or expired token")
			return
		}

		// Set user in context using exported keys
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = slogx.SetUserID(ctx, claims.UserID)

		// Create a minimal user object from claims
		user := &domain.User{
			ID:          claims.UserID,
			Email:       claims.Email,
			Role:        claims.Role,
			Permissions: claims.Permissions,
		}
		ctx = context.WithValue(ctx, UserKey, user)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission creates a middleware that requires a specific permission
func (a *Auth) RequirePermission(permission string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := r.Context().Value(UserKey).(*domain.User)
			if !ok {
				response.Unauthorized(w, "Authentication required")
				return
			}

			// Admin has all permissions
			if user.Role == domain.UserRoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			// Check if user has the required permission
			hasPermission := false
			for _, p := range user.Permissions {
				if p == permission || p == "*" {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				response.Forbidden(w, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole creates a middleware that requires a specific role
func (a *Auth) RequireRole(roles ...domain.UserRole) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := r.Context().Value(UserKey).(*domain.User)
			if !ok {
				response.Unauthorized(w, "Authentication required")
				return
			}

			for _, role := range roles {
				if user.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			response.Forbidden(w, "Insufficient role")
		})
	}
}

// getRemoteIP extracts the real client IP
func getRemoteIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return strings.Split(r.RemoteAddr, ":")[0]
}

// GetUserFromContext retrieves the user from context
func GetUserFromContext(ctx context.Context) *domain.User {
	if user, ok := ctx.Value(UserKey).(*domain.User); ok {
		return user
	}
	return nil
}

// GetUserIDFromContext retrieves the user ID from context
func GetUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}
