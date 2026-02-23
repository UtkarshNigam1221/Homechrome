package dynamodb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenStore_StoreRefreshToken(t *testing.T) {
	wrappedClient, rawClient := testWrappedClient(t)
	skipIfNoLocal(t, rawClient)
	setupTestTable(t, rawClient, testSessionsTable)
	defer cleanupTestTable(t, rawClient, testSessionsTable)

	store := NewTokenStore(wrappedClient)
	ctx := context.Background()

	t.Run("store and validate refresh token", func(t *testing.T) {
		userID := "user_token123"
		token := "refresh_token_abc123"
		expiry := 24 * time.Hour

		err := store.StoreRefreshToken(ctx, userID, token, expiry)
		require.NoError(t, err)

		// Validate the token
		valid, err := store.ValidateRefreshToken(ctx, userID, token)
		require.NoError(t, err)
		assert.True(t, valid)
	})

	t.Run("invalid token returns false", func(t *testing.T) {
		userID := "user_token456"
		token := "valid_token"
		wrongToken := "wrong_token"

		err := store.StoreRefreshToken(ctx, userID, token, 24*time.Hour)
		require.NoError(t, err)

		valid, err := store.ValidateRefreshToken(ctx, userID, wrongToken)
		require.NoError(t, err)
		assert.False(t, valid)
	})
}

func TestTokenStore_RevokeRefreshToken(t *testing.T) {
	wrappedClient, rawClient := testWrappedClient(t)
	skipIfNoLocal(t, rawClient)
	setupTestTable(t, rawClient, testSessionsTable)
	defer cleanupTestTable(t, rawClient, testSessionsTable)

	store := NewTokenStore(wrappedClient)
	ctx := context.Background()

	t.Run("revoke token", func(t *testing.T) {
		userID := "user_revoke123"
		token := "revoke_token_abc"

		err := store.StoreRefreshToken(ctx, userID, token, 24*time.Hour)
		require.NoError(t, err)

		// Verify token is valid
		valid, err := store.ValidateRefreshToken(ctx, userID, token)
		require.NoError(t, err)
		assert.True(t, valid)

		// Revoke the token
		err = store.RevokeRefreshToken(ctx, userID, token)
		require.NoError(t, err)

		// Token should no longer be valid
		valid, err = store.ValidateRefreshToken(ctx, userID, token)
		require.NoError(t, err)
		assert.False(t, valid)
	})
}

func TestTokenStore_RevokeAllUserTokens(t *testing.T) {
	wrappedClient, rawClient := testWrappedClient(t)
	skipIfNoLocal(t, rawClient)
	setupTestTable(t, rawClient, testSessionsTable)
	defer cleanupTestTable(t, rawClient, testSessionsTable)

	store := NewTokenStore(wrappedClient)
	ctx := context.Background()

	t.Run("revoke all user tokens", func(t *testing.T) {
		userID := "user_revokeall123"
		token1 := "token_1"
		token2 := "token_2"
		token3 := "token_3"

		// Store multiple tokens
		err := store.StoreRefreshToken(ctx, userID, token1, 24*time.Hour)
		require.NoError(t, err)
		err = store.StoreRefreshToken(ctx, userID, token2, 24*time.Hour)
		require.NoError(t, err)
		err = store.StoreRefreshToken(ctx, userID, token3, 24*time.Hour)
		require.NoError(t, err)

		// Verify tokens are valid
		valid, _ := store.ValidateRefreshToken(ctx, userID, token1)
		assert.True(t, valid)
		valid, _ = store.ValidateRefreshToken(ctx, userID, token2)
		assert.True(t, valid)
		valid, _ = store.ValidateRefreshToken(ctx, userID, token3)
		assert.True(t, valid)

		// Revoke all tokens
		err = store.RevokeAllUserTokens(ctx, userID)
		require.NoError(t, err)

		// All tokens should be invalid
		valid, _ = store.ValidateRefreshToken(ctx, userID, token1)
		assert.False(t, valid)
		valid, _ = store.ValidateRefreshToken(ctx, userID, token2)
		assert.False(t, valid)
		valid, _ = store.ValidateRefreshToken(ctx, userID, token3)
		assert.False(t, valid)
	})
}

func TestTokenStore_PasswordResetToken(t *testing.T) {
	wrappedClient, rawClient := testWrappedClient(t)
	skipIfNoLocal(t, rawClient)
	setupTestTable(t, rawClient, testSessionsTable)
	defer cleanupTestTable(t, rawClient, testSessionsTable)

	store := NewTokenStore(wrappedClient)
	ctx := context.Background()

	t.Run("store and validate password reset token", func(t *testing.T) {
		userID := "user_reset123"
		token := "reset_token_abc123"
		expiry := 1 * time.Hour

		err := store.StorePasswordResetToken(ctx, userID, token, expiry)
		require.NoError(t, err)

		// Validate the token - should return the user ID
		returnedUserID, err := store.ValidatePasswordResetToken(ctx, token)
		require.NoError(t, err)
		assert.Equal(t, userID, returnedUserID)
	})

	t.Run("invalid password reset token", func(t *testing.T) {
		_, err := store.ValidatePasswordResetToken(ctx, "nonexistent_token")
		require.Error(t, err)
	})

	t.Run("revoke password reset token", func(t *testing.T) {
		userID := "user_reset_revoke"
		token := "reset_revoke_token"

		err := store.StorePasswordResetToken(ctx, userID, token, 1*time.Hour)
		require.NoError(t, err)

		// Verify token is valid
		_, err = store.ValidatePasswordResetToken(ctx, token)
		require.NoError(t, err)

		// Revoke the token
		err = store.RevokePasswordResetToken(ctx, token)
		require.NoError(t, err)

		// Token should no longer be valid
		_, err = store.ValidatePasswordResetToken(ctx, token)
		require.Error(t, err)
	})
}

func TestTokenStore_TokenWithSameUserMultipleSessions(t *testing.T) {
	wrappedClient, rawClient := testWrappedClient(t)
	skipIfNoLocal(t, rawClient)
	setupTestTable(t, rawClient, testSessionsTable)
	defer cleanupTestTable(t, rawClient, testSessionsTable)

	store := NewTokenStore(wrappedClient)
	ctx := context.Background()

	t.Run("multiple sessions for same user", func(t *testing.T) {
		userID := "user_multisession"

		// Simulate multiple login sessions (different devices)
		session1Token := "session1_token"
		session2Token := "session2_token"
		session3Token := "session3_token"

		err := store.StoreRefreshToken(ctx, userID, session1Token, 24*time.Hour)
		require.NoError(t, err)
		err = store.StoreRefreshToken(ctx, userID, session2Token, 24*time.Hour)
		require.NoError(t, err)
		err = store.StoreRefreshToken(ctx, userID, session3Token, 24*time.Hour)
		require.NoError(t, err)

		// All sessions should be valid
		valid, _ := store.ValidateRefreshToken(ctx, userID, session1Token)
		assert.True(t, valid)
		valid, _ = store.ValidateRefreshToken(ctx, userID, session2Token)
		assert.True(t, valid)
		valid, _ = store.ValidateRefreshToken(ctx, userID, session3Token)
		assert.True(t, valid)

		// Revoke just one session
		err = store.RevokeRefreshToken(ctx, userID, session2Token)
		require.NoError(t, err)

		// Only session2 should be invalid
		valid, _ = store.ValidateRefreshToken(ctx, userID, session1Token)
		assert.True(t, valid)
		valid, _ = store.ValidateRefreshToken(ctx, userID, session2Token)
		assert.False(t, valid)
		valid, _ = store.ValidateRefreshToken(ctx, userID, session3Token)
		assert.True(t, valid)
	})
}
