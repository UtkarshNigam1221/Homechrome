package domain

import (
	"context"
	"time"
)

//go:generate mockgen -source=token_store.go -destination=../mocks/token_store_mock.go -package=mocks

// TokenStore defines the interface for token storage operations
type TokenStore interface {
	// StoreRefreshToken stores a refresh token for a user
	StoreRefreshToken(ctx context.Context, userID string, token string, expiry time.Duration) error

	// ValidateRefreshToken validates if a refresh token exists and is valid
	ValidateRefreshToken(ctx context.Context, userID string, token string) (bool, error)

	// RevokeRefreshToken revokes a specific refresh token
	RevokeRefreshToken(ctx context.Context, userID string, token string) error

	// RevokeAllUserTokens revokes all tokens for a user
	RevokeAllUserTokens(ctx context.Context, userID string) error

	// StorePasswordResetToken stores a password reset token
	StorePasswordResetToken(ctx context.Context, userID string, token string, expiry time.Duration) error

	// ValidatePasswordResetToken validates a password reset token and returns the user ID
	ValidatePasswordResetToken(ctx context.Context, token string) (string, error)

	// RevokePasswordResetToken revokes a password reset token
	RevokePasswordResetToken(ctx context.Context, token string) error
}
