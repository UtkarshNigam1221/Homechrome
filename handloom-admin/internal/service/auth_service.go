// Package service implements the business logic layer
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
)

// AuthService implements domain.AuthService
type AuthService struct {
	userRepo             domain.UserRepository
	tokenStore           domain.TokenStore
	logger               *logger.Logger
	jwtSecret            []byte
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
	issuer               string
}

// NewAuthService creates a new AuthService
func NewAuthService(
	userRepo domain.UserRepository,
	tokenStore domain.TokenStore,
	logger *logger.Logger,
	jwtSecret string,
	accessTokenDuration time.Duration,
	refreshTokenDuration time.Duration,
	issuer string,
) *AuthService {
	return &AuthService{
		userRepo:             userRepo,
		tokenStore:           tokenStore,
		logger:               logger,
		jwtSecret:            []byte(jwtSecret),
		accessTokenDuration:  accessTokenDuration,
		refreshTokenDuration: refreshTokenDuration,
		issuer:               issuer,
	}
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*domain.LoginResponse, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.New(errors.ErrCodeInvalidCredentials, "Invalid email or password")
		}
		return nil, err
	}

	// Check user status
	if user.Status != domain.UserStatusActive {
		return nil, errors.New(errors.ErrCodeUserInactive, "User account is not active")
	}

	// Verify password
	if bcryptErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); bcryptErr != nil {
		return nil, errors.New(errors.ErrCodeInvalidCredentials, "Invalid email or password")
	}

	// Revoke any existing refresh tokens before generating new ones (single-session enforcement)
	if revokeErr := s.tokenStore.RevokeAllUserTokens(ctx, user.ID); revokeErr != nil {
		s.logger.WithContext(ctx).WithError(revokeErr).Warn("Failed to revoke existing tokens")
	}

	// Generate tokens
	tokens, err := s.generateTokenPair(user)
	if err != nil {
		return nil, err
	}

	// Store refresh token — fail the login if we can't persist the session
	if err := s.tokenStore.StoreRefreshToken(ctx, user.ID, tokens.RefreshToken, s.refreshTokenDuration); err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to store refresh token")
		return nil, errors.Internal("Failed to create session")
	}

	// Update last login (best-effort, non-critical)
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.WithContext(ctx).WithError(err).Warn("Failed to update last login")
	}

	s.logger.WithContext(ctx).Infof("User logged in: %s", user.ID)

	// Remove sensitive data
	user.PasswordHash = ""

	return &domain.LoginResponse{
		User:   user,
		Tokens: tokens,
	}, nil
}

// RefreshToken refreshes an access token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	// Validate refresh token
	claims, err := s.validateRefreshToken(refreshToken)
	if err != nil {
		return nil, errors.New(errors.ErrCodeInvalidToken, "Invalid refresh token")
	}

	// Check if refresh token is stored
	valid, err := s.tokenStore.ValidateRefreshToken(ctx, claims.UserID, refreshToken)
	if err != nil || !valid {
		return nil, errors.New(errors.ErrCodeInvalidToken, "Refresh token has been revoked")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}

	// Check user status
	if user.Status != domain.UserStatusActive {
		return nil, errors.New(errors.ErrCodeUserInactive, "User account is not active")
	}

	// Generate new token pair
	tokens, err := s.generateTokenPair(user)
	if err != nil {
		return nil, err
	}

	// Revoke old refresh token — if this fails, the old token remains valid (token reuse risk)
	if err := s.tokenStore.RevokeRefreshToken(ctx, claims.UserID, refreshToken); err != nil {
		s.logger.WithContext(ctx).WithError(err).Warn("Failed to revoke old refresh token")
	}

	// Store new refresh token — fail if we can't persist the session
	if err := s.tokenStore.StoreRefreshToken(ctx, user.ID, tokens.RefreshToken, s.refreshTokenDuration); err != nil {
		s.logger.WithContext(ctx).WithError(err).Error("Failed to store refresh token")
		return nil, errors.Internal("Failed to create session")
	}

	return tokens, nil
}

// Logout invalidates a user's tokens
func (s *AuthService) Logout(ctx context.Context, userID string) error {
	if err := s.tokenStore.RevokeAllUserTokens(ctx, userID); err != nil {
		return err
	}

	s.logger.WithContext(ctx).Infof("User logged out: %s", userID)
	return nil
}

// ValidateToken validates an access token and returns claims
func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (*domain.TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New(errors.ErrCodeInvalidToken, "Invalid token signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, errors.New(errors.ErrCodeInvalidToken, "Invalid token")
	}

	if !token.Valid {
		return nil, errors.New(errors.ErrCodeInvalidToken, "Token is not valid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New(errors.ErrCodeInvalidToken, "Invalid token claims")
	}

	// Extract claims
	userID, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	roleStr, _ := claims["role"].(string)
	permissionsInterface, _ := claims["permissions"].([]interface{})

	permissions := make([]string, len(permissionsInterface))
	for i, p := range permissionsInterface {
		permissions[i], _ = p.(string)
	}

	return &domain.TokenClaims{
		UserID:      userID,
		Email:       email,
		Role:        domain.UserRole(roleStr),
		Permissions: permissions,
	}, nil
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(ctx context.Context, userID string, req domain.ChangePasswordRequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Verify current password
	if bcryptErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); bcryptErr != nil {
		return errors.New(errors.ErrCodeBadRequest, "Current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.Internal("Failed to hash password")
	}

	// Update password
	user.PasswordHash = string(hashedPassword)
	user.UpdatedBy = userID

	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// Revoke all existing tokens to force re-login
	if err := s.tokenStore.RevokeAllUserTokens(ctx, userID); err != nil {
		s.logger.WithContext(ctx).WithError(err).Warn("Failed to revoke tokens after password change")
	}

	s.logger.WithContext(ctx).Infof("Password changed for user: %s", userID)
	return nil
}

// RequestPasswordReset initiates password reset flow
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		// Don't reveal if email exists
		if errors.IsNotFound(err) {
			s.logger.WithContext(ctx).Infof("Password reset requested for non-existent email: %s", email)
			return nil
		}
		return err
	}

	// Generate reset token
	resetToken, err := generateSecureToken(32)
	if err != nil {
		return errors.Internal("Failed to generate reset token")
	}

	// Store reset token (valid for 1 hour)
	if err := s.tokenStore.StorePasswordResetToken(ctx, user.ID, resetToken, time.Hour); err != nil {
		return err
	}

	// TODO: Send password reset email
	// This should be handled by a notification service
	s.logger.WithContext(ctx).Infof("Password reset token generated for user: %s", user.ID)

	return nil
}

// ResetPassword resets password with token
func (s *AuthService) ResetPassword(ctx context.Context, req domain.ResetPasswordRequest) error {
	// Validate reset token and get user ID
	userID, err := s.tokenStore.ValidatePasswordResetToken(ctx, req.Token)
	if err != nil {
		return errors.New(errors.ErrCodeInvalidToken, "Invalid or expired reset token")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.Internal("Failed to hash password")
	}

	// Update password
	user.PasswordHash = string(hashedPassword)

	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// Revoke reset token and all existing session tokens
	if err := s.tokenStore.RevokePasswordResetToken(ctx, req.Token); err != nil {
		s.logger.WithContext(ctx).WithError(err).Warn("Failed to revoke password reset token")
	}
	if err := s.tokenStore.RevokeAllUserTokens(ctx, userID); err != nil {
		s.logger.WithContext(ctx).WithError(err).Warn("Failed to revoke tokens after password reset")
	}

	s.logger.WithContext(ctx).Infof("Password reset completed for user: %s", userID)
	return nil
}

// generateTokenPair generates access and refresh tokens
func (s *AuthService) generateTokenPair(user *domain.User) (*domain.TokenPair, error) {
	now := time.Now()
	accessExpiry := now.Add(s.accessTokenDuration)
	refreshExpiry := now.Add(s.refreshTokenDuration)

	// Access token claims
	accessClaims := jwt.MapClaims{
		"sub":         user.ID,
		"email":       user.Email,
		"role":        user.Role,
		"permissions": user.Permissions,
		"iat":         now.Unix(),
		"exp":         accessExpiry.Unix(),
		"iss":         s.issuer,
		"jti":         uuid.New().String(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, errors.Internal("Failed to generate access token")
	}

	// Refresh token claims
	refreshClaims := jwt.MapClaims{
		"sub":  user.ID,
		"type": "refresh",
		"iat":  now.Unix(),
		"exp":  refreshExpiry.Unix(),
		"iss":  s.issuer,
		"jti":  uuid.New().String(),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, errors.Internal("Failed to generate refresh token")
	}

	return &domain.TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    accessExpiry,
	}, nil
}

// validateRefreshToken validates a refresh token
func (s *AuthService) validateRefreshToken(tokenString string) (*domain.TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New(errors.ErrCodeInvalidToken, "Invalid token signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New(errors.ErrCodeInvalidToken, "Token is not valid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New(errors.ErrCodeInvalidToken, "Invalid token claims")
	}

	// Verify it's a refresh token
	tokenType, _ := claims["type"].(string)
	if tokenType != "refresh" {
		return nil, errors.New(errors.ErrCodeInvalidToken, "Not a refresh token")
	}

	userID, _ := claims["sub"].(string)

	return &domain.TokenClaims{
		UserID: userID,
	}, nil
}

// generateSecureToken generates a secure random token
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Ensure interface compliance
var _ domain.AuthService = (*AuthService)(nil)
