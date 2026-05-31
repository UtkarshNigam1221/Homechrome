// Package middleware provides HTTP middleware
package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/response"
	"github.com/handloom/admin/pkg/slogx"
	"github.com/handloom/admin/pkg/telemetry"
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

// Logger logs HTTP requests with status-based severity and surfaces the
// active trace ID on the response so the client can quote it in tickets.
// At the end of every request it also force-flushes the OTel providers so
// metrics and any buffered telemetry ship before the Lambda runtime freezes
// the process. Outside Lambda the flush is a near no-op against a live
// batch processor.
func Logger() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			if span := trace.SpanFromContext(r.Context()); span.SpanContext().IsValid() {
				w.Header().Set("X-Trace-ID", span.SpanContext().TraceID().String())
			}

			next.ServeHTTP(ww, r)

			level := slog.LevelInfo
			switch {
			case ww.statusCode >= 500:
				level = slog.LevelError
			case ww.statusCode >= 400:
				level = slog.LevelWarn
			}
			slog.Log(r.Context(), level, "request_completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.statusCode,
				"duration_ms", time.Since(start).Milliseconds(),
				"remote_ip", getRemoteIP(r),
			)

			flushCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			telemetry.ForceFlush(flushCtx)
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
