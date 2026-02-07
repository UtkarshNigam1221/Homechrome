package middleware

import (
	"context"

	"github.com/handloom/admin/internal/domain"
)

// Context helper functions - consolidated from middleware.go and handler/helpers.go
// These are the canonical implementations that should be used everywhere.

// SetUserInContext sets the user in context.
// This also sets the user ID for convenience.
func SetUserInContext(ctx context.Context, user *domain.User) context.Context {
	ctx = context.WithValue(ctx, UserIDKey, user.ID)
	ctx = context.WithValue(ctx, UserKey, user)
	return ctx
}

// RequireUserID retrieves user ID from context and returns error if not found.
func RequireUserID(ctx context.Context) (string, error) {
	userID := GetUserIDFromContext(ctx)
	if userID == "" {
		return "", ErrUserNotInContext
	}
	return userID, nil
}

// ErrUserNotInContext is returned when user is expected but not found in context
var ErrUserNotInContext = &contextError{message: "user not found in context"}

// contextError is a simple error type for context-related errors
type contextError struct {
	message string
}

func (e *contextError) Error() string {
	return e.message
}

// GetCreatedBy returns the user ID suitable for "created_by" fields.
// Returns empty string if no user is authenticated.
func GetCreatedBy(ctx context.Context) string {
	if user := GetUserFromContext(ctx); user != nil {
		return user.ID
	}
	return GetUserIDFromContext(ctx)
}
