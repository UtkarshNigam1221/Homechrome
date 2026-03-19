package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
)

func TestAuthService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)

	service := NewAuthService(
		userRepo,
		tokenStore,
		"test-secret-key",
		15*time.Minute,
		7*24*time.Hour,
		"test-issuer",
	)

	ctx := context.Background()

	// Hash password for test user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	tests := []struct {
		name    string
		req     domain.LoginRequest
		setup   func()
		wantErr bool
		errCode string
	}{
		{
			name: "successful login",
			req: domain.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			setup: func() {
				userRepo.EXPECT().GetByEmail(ctx, "test@example.com").Return(&domain.User{
					ID:           "user_123",
					Email:        "test@example.com",
					PasswordHash: string(hashedPassword),
					Status:       domain.UserStatusActive,
					Role:         domain.UserRoleAdmin,
					Permissions:  []string{"*"},
				}, nil)
				tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(nil)
				tokenStore.EXPECT().StoreRefreshToken(ctx, "user_123", gomock.Any(), gomock.Any()).Return(nil)
				userRepo.EXPECT().UpdateLastLogin(ctx, "user_123").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "invalid email",
			req: domain.LoginRequest{
				Email:    "nonexistent@example.com",
				Password: "password123",
			},
			setup: func() {
				userRepo.EXPECT().GetByEmail(ctx, "nonexistent@example.com").Return(nil, errors.NotFound("User"))
			},
			wantErr: true,
			errCode: "INVALID_CREDENTIALS",
		},
		{
			name: "wrong password",
			req: domain.LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			setup: func() {
				userRepo.EXPECT().GetByEmail(ctx, "test@example.com").Return(&domain.User{
					ID:           "user_123",
					Email:        "test@example.com",
					PasswordHash: string(hashedPassword),
					Status:       domain.UserStatusActive,
				}, nil)
			},
			wantErr: true,
			errCode: "INVALID_CREDENTIALS",
		},
		{
			name: "inactive user",
			req: domain.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			setup: func() {
				userRepo.EXPECT().GetByEmail(ctx, "test@example.com").Return(&domain.User{
					ID:           "user_123",
					Email:        "test@example.com",
					PasswordHash: string(hashedPassword),
					Status:       domain.UserStatusInactive,
				}, nil)
			},
			wantErr: true,
			errCode: "USER_INACTIVE",
		},
		{
			name: "store refresh token fails - returns error",
			req:  domain.LoginRequest{Email: "test@example.com", Password: "password123"},
			setup: func() {
				userRepo.EXPECT().GetByEmail(ctx, "test@example.com").Return(&domain.User{
					ID: "user_123", Email: "test@example.com",
					PasswordHash: string(hashedPassword), Status: domain.UserStatusActive,
					Role: domain.UserRoleAdmin, Permissions: []string{"*"},
				}, nil)
				tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(nil)
				tokenStore.EXPECT().StoreRefreshToken(ctx, "user_123", gomock.Any(), gomock.Any()).Return(errors.Internal("redis down"))
			},
			wantErr: true,
			errCode: "INTERNAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			result, err := service.Login(ctx, tt.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.Tokens.AccessToken)
				assert.NotEmpty(t, result.Tokens.RefreshToken)
				assert.Equal(t, "test@example.com", result.User.Email)
				assert.Empty(t, result.User.PasswordHash) // Should be removed
			}
		})
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)

	service := NewAuthService(
		userRepo,
		tokenStore,
		"test-secret-key",
		15*time.Minute,
		7*24*time.Hour,
		"test-issuer",
	)

	ctx := context.Background()

	// Generate a valid token
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &domain.User{
		ID:           "user_123",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		Status:       domain.UserStatusActive,
		Role:         domain.UserRoleAdmin,
		Permissions:  []string{"read", "write"},
	}

	tokens, err := service.generateTokenPair(user)
	require.NoError(t, err)

	t.Run("valid token", func(t *testing.T) {
		claims, err := service.ValidateToken(ctx, tokens.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, "user_123", claims.UserID)
		assert.Equal(t, "test@example.com", claims.Email)
		assert.Equal(t, domain.UserRoleAdmin, claims.Role)
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := service.ValidateToken(ctx, "invalid-token")
		require.Error(t, err)
	})

	t.Run("tampered token", func(t *testing.T) {
		_, err := service.ValidateToken(ctx, tokens.AccessToken+"tampered")
		require.Error(t, err)
	})
}

func TestAuthService_RefreshToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)

	service := NewAuthService(
		userRepo,
		tokenStore,
		"test-secret-key",
		15*time.Minute,
		7*24*time.Hour,
		"test-issuer",
	)

	ctx := context.Background()

	// Generate tokens for testing
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &domain.User{
		ID:           "user_123",
		Email:        "test@example.com",
		PasswordHash: string(hashedPassword),
		Status:       domain.UserStatusActive,
		Role:         domain.UserRoleAdmin,
		Permissions:  []string{"*"},
	}
	tokens, _ := service.generateTokenPair(user)

	t.Run("successful refresh", func(t *testing.T) {
		tokenStore.EXPECT().ValidateRefreshToken(ctx, "user_123", tokens.RefreshToken).Return(true, nil)
		userRepo.EXPECT().GetByID(ctx, "user_123").Return(user, nil)
		tokenStore.EXPECT().RevokeRefreshToken(ctx, "user_123", tokens.RefreshToken).Return(nil)
		tokenStore.EXPECT().StoreRefreshToken(ctx, "user_123", gomock.Any(), gomock.Any()).Return(nil)

		newTokens, err := service.RefreshToken(ctx, tokens.RefreshToken)
		require.NoError(t, err)
		assert.NotNil(t, newTokens)
		assert.NotEmpty(t, newTokens.AccessToken)
		assert.NotEmpty(t, newTokens.RefreshToken)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		_, err := service.RefreshToken(ctx, "invalid-token")
		require.Error(t, err)
	})
}

func TestAuthService_Logout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)

	service := NewAuthService(
		userRepo,
		tokenStore,
		"test-secret-key",
		15*time.Minute,
		7*24*time.Hour,
		"test-issuer",
	)

	ctx := context.Background()

	t.Run("successful logout", func(t *testing.T) {
		tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(nil)

		err := service.Logout(ctx, "user_123")
		require.NoError(t, err)
	})
}

func TestAuthService_ChangePassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)

	service := NewAuthService(
		userRepo,
		tokenStore,
		"test-secret-key",
		15*time.Minute,
		7*24*time.Hour,
		"test-issuer",
	)

	ctx := context.Background()
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("oldpassword"), bcrypt.DefaultCost)

	t.Run("successful password change", func(t *testing.T) {
		user := &domain.User{
			ID:           "user_123",
			Email:        "test@example.com",
			PasswordHash: string(hashedPassword),
		}

		userRepo.EXPECT().GetByID(ctx, "user_123").Return(user, nil)
		userRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
		tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(nil)

		err := service.ChangePassword(ctx, "user_123", domain.ChangePasswordRequest{
			CurrentPassword: "oldpassword",
			NewPassword:     "newpassword123",
		})
		require.NoError(t, err)
	})

	t.Run("wrong current password", func(t *testing.T) {
		user := &domain.User{
			ID:           "user_123",
			Email:        "test@example.com",
			PasswordHash: string(hashedPassword),
		}

		userRepo.EXPECT().GetByID(ctx, "user_123").Return(user, nil)

		err := service.ChangePassword(ctx, "user_123", domain.ChangePasswordRequest{
			CurrentPassword: "wrongpassword",
			NewPassword:     "newpassword123",
		})
		require.Error(t, err)
	})
}

func TestAuthService_RequestPasswordReset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)

	svc := NewAuthService(userRepo, tokenStore, "test-secret", 15*time.Minute, 7*24*time.Hour, "test-issuer")
	ctx := context.Background()

	tests := []struct {
		name    string
		email   string
		setup   func()
		wantErr bool
	}{
		{
			name:  "existing user - stores reset token",
			email: "test@example.com",
			setup: func() {
				userRepo.EXPECT().GetByEmail(ctx, "test@example.com").Return(&domain.User{
					ID:    "user_123",
					Email: "test@example.com",
				}, nil)
				tokenStore.EXPECT().StorePasswordResetToken(ctx, "user_123", gomock.Any(), time.Hour).Return(nil)
			},
			wantErr: false,
		},
		{
			name:  "non-existent email - succeeds silently (no leak)",
			email: "noone@example.com",
			setup: func() {
				userRepo.EXPECT().GetByEmail(ctx, "noone@example.com").Return(nil, errors.NotFound("User"))
			},
			wantErr: false,
		},
		{
			name:  "repo error - returns error",
			email: "test@example.com",
			setup: func() {
				userRepo.EXPECT().GetByEmail(ctx, "test@example.com").Return(nil, errors.Internal("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := svc.RequestPasswordReset(ctx, tt.email)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAuthService_ResetPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)

	svc := NewAuthService(userRepo, tokenStore, "test-secret", 15*time.Minute, 7*24*time.Hour, "test-issuer")
	ctx := context.Background()

	tests := []struct {
		name    string
		req     domain.ResetPasswordRequest
		setup   func()
		wantErr bool
	}{
		{
			name: "valid reset token",
			req:  domain.ResetPasswordRequest{Token: "valid-token", NewPassword: "newpass123"},
			setup: func() {
				tokenStore.EXPECT().ValidatePasswordResetToken(ctx, "valid-token").Return("user_123", nil)
				userRepo.EXPECT().GetByID(ctx, "user_123").Return(&domain.User{ID: "user_123"}, nil)
				userRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
				tokenStore.EXPECT().RevokePasswordResetToken(ctx, "valid-token").Return(nil)
				tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "invalid reset token",
			req:  domain.ResetPasswordRequest{Token: "bad-token", NewPassword: "newpass123"},
			setup: func() {
				tokenStore.EXPECT().ValidatePasswordResetToken(ctx, "bad-token").Return("", errors.New(errors.ErrCodeInvalidToken, "expired"))
			},
			wantErr: true,
		},
		{
			name: "user not found after token validation",
			req:  domain.ResetPasswordRequest{Token: "valid-token", NewPassword: "newpass123"},
			setup: func() {
				tokenStore.EXPECT().ValidatePasswordResetToken(ctx, "valid-token").Return("user_999", nil)
				userRepo.EXPECT().GetByID(ctx, "user_999").Return(nil, errors.NotFound("User"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := svc.ResetPassword(ctx, tt.req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAuthService_Login_ErrorCodes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)

	svc := NewAuthService(userRepo, tokenStore, "test-secret", 15*time.Minute, 7*24*time.Hour, "test-issuer")
	ctx := context.Background()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	t.Run("inactive user returns USER_INACTIVE error code", func(t *testing.T) {
		userRepo.EXPECT().GetByEmail(ctx, "inactive@example.com").Return(&domain.User{
			ID:           "user_123",
			Email:        "inactive@example.com",
			PasswordHash: string(hashedPassword),
			Status:       domain.UserStatusInactive,
		}, nil)

		_, err := svc.Login(ctx, domain.LoginRequest{Email: "inactive@example.com", Password: "password123"})

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeUserInactive, appErr.Code)
	})

	t.Run("pending user returns USER_INACTIVE error code", func(t *testing.T) {
		userRepo.EXPECT().GetByEmail(ctx, "pending@example.com").Return(&domain.User{
			ID:           "user_456",
			Email:        "pending@example.com",
			PasswordHash: string(hashedPassword),
			Status:       domain.UserStatusPending,
		}, nil)

		_, err := svc.Login(ctx, domain.LoginRequest{Email: "pending@example.com", Password: "password123"})

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeUserInactive, appErr.Code)
	})

	t.Run("update last login failure is non-fatal", func(t *testing.T) {
		userRepo.EXPECT().GetByEmail(ctx, "test@example.com").Return(&domain.User{
			ID: "user_123", Email: "test@example.com",
			PasswordHash: string(hashedPassword), Status: domain.UserStatusActive,
			Role: domain.UserRoleAdmin, Permissions: []string{"*"},
		}, nil)
		tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(nil)
		tokenStore.EXPECT().StoreRefreshToken(ctx, "user_123", gomock.Any(), gomock.Any()).Return(nil)
		userRepo.EXPECT().UpdateLastLogin(ctx, "user_123").Return(errors.Internal("audit db down"))

		result, err := svc.Login(ctx, domain.LoginRequest{Email: "test@example.com", Password: "password123"})

		require.NoError(t, err) // non-fatal
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Tokens.AccessToken)
	})

	t.Run("revoke previous tokens failure is non-fatal", func(t *testing.T) {
		userRepo.EXPECT().GetByEmail(ctx, "test@example.com").Return(&domain.User{
			ID: "user_123", Email: "test@example.com",
			PasswordHash: string(hashedPassword), Status: domain.UserStatusActive,
			Role: domain.UserRoleAdmin, Permissions: []string{"*"},
		}, nil)
		tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(errors.Internal("redis down"))
		tokenStore.EXPECT().StoreRefreshToken(ctx, "user_123", gomock.Any(), gomock.Any()).Return(nil)
		userRepo.EXPECT().UpdateLastLogin(ctx, "user_123").Return(nil)

		result, err := svc.Login(ctx, domain.LoginRequest{Email: "test@example.com", Password: "password123"})

		require.NoError(t, err) // revoke failure is non-fatal
		assert.NotNil(t, result)
	})
}

func TestAuthService_RefreshToken_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)

	svc := NewAuthService(userRepo, tokenStore, "test-secret", 15*time.Minute, 7*24*time.Hour, "test-issuer")
	ctx := context.Background()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &domain.User{
		ID: "user_123", Email: "test@example.com",
		PasswordHash: string(hashedPassword), Status: domain.UserStatusActive,
		Role: domain.UserRoleAdmin, Permissions: []string{"*"},
	}
	tokens, _ := svc.generateTokenPair(user)

	t.Run("inactive user returns USER_INACTIVE error code", func(t *testing.T) {
		inactiveUser := &domain.User{
			ID: "user_123", Email: "test@example.com",
			Status: domain.UserStatusInactive,
		}
		tokenStore.EXPECT().ValidateRefreshToken(ctx, "user_123", tokens.RefreshToken).Return(true, nil)
		userRepo.EXPECT().GetByID(ctx, "user_123").Return(inactiveUser, nil)

		_, err := svc.RefreshToken(ctx, tokens.RefreshToken)

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeUserInactive, appErr.Code)
	})

	t.Run("old token revoke failure is non-fatal", func(t *testing.T) {
		tokenStore.EXPECT().ValidateRefreshToken(ctx, "user_123", tokens.RefreshToken).Return(true, nil)
		userRepo.EXPECT().GetByID(ctx, "user_123").Return(user, nil)
		tokenStore.EXPECT().RevokeRefreshToken(ctx, "user_123", tokens.RefreshToken).Return(errors.Internal("redis error"))
		tokenStore.EXPECT().StoreRefreshToken(ctx, "user_123", gomock.Any(), gomock.Any()).Return(nil)

		newTokens, err := svc.RefreshToken(ctx, tokens.RefreshToken)

		require.NoError(t, err) // revoke failure is non-fatal
		assert.NotEmpty(t, newTokens.AccessToken)
	})
}

func TestAuthService_ChangePassword_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)

	svc := NewAuthService(userRepo, tokenStore, "test-secret", 15*time.Minute, 7*24*time.Hour, "test-issuer")
	ctx := context.Background()

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("oldpassword"), bcrypt.DefaultCost)

	t.Run("wrong password returns BAD_REQUEST error code", func(t *testing.T) {
		user := &domain.User{
			ID: "user_123", PasswordHash: string(hashedPassword),
		}
		userRepo.EXPECT().GetByID(ctx, "user_123").Return(user, nil)

		err := svc.ChangePassword(ctx, "user_123", domain.ChangePasswordRequest{
			CurrentPassword: "wrongpassword",
			NewPassword:     "newpassword123",
		})

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeBadRequest, appErr.Code)
	})

	t.Run("user not found returns error", func(t *testing.T) {
		userRepo.EXPECT().GetByID(ctx, "user_999").Return(nil, errors.NotFound("User"))

		err := svc.ChangePassword(ctx, "user_999", domain.ChangePasswordRequest{
			CurrentPassword: "old",
			NewPassword:     "new",
		})

		require.Error(t, err)
	})

	t.Run("token revocation failure is non-fatal", func(t *testing.T) {
		user := &domain.User{
			ID: "user_123", PasswordHash: string(hashedPassword),
		}
		userRepo.EXPECT().GetByID(ctx, "user_123").Return(user, nil)
		userRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
		tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(errors.Internal("redis down"))

		err := svc.ChangePassword(ctx, "user_123", domain.ChangePasswordRequest{
			CurrentPassword: "oldpassword",
			NewPassword:     "newpassword123",
		})

		require.NoError(t, err) // token revoke failure is non-fatal
	})

	t.Run("revokes all tokens on success", func(t *testing.T) {
		user := &domain.User{
			ID: "user_123", PasswordHash: string(hashedPassword),
		}
		userRepo.EXPECT().GetByID(ctx, "user_123").Return(user, nil)
		userRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
		tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(nil)

		err := svc.ChangePassword(ctx, "user_123", domain.ChangePasswordRequest{
			CurrentPassword: "oldpassword",
			NewPassword:     "newpassword123",
		})

		require.NoError(t, err)
		// If we got here without gomock error, RevokeAllUserTokens was called
	})
}

func TestAuthService_RequestPasswordReset_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)

	svc := NewAuthService(userRepo, tokenStore, "test-secret", 15*time.Minute, 7*24*time.Hour, "test-issuer")
	ctx := context.Background()

	t.Run("token store failure returns error", func(t *testing.T) {
		userRepo.EXPECT().GetByEmail(ctx, "test@example.com").Return(&domain.User{
			ID:    "user_123",
			Email: "test@example.com",
		}, nil)
		tokenStore.EXPECT().StorePasswordResetToken(ctx, "user_123", gomock.Any(), time.Hour).Return(errors.Internal("store failed"))

		err := svc.RequestPasswordReset(ctx, "test@example.com")

		require.Error(t, err)
	})
}

func TestAuthService_ResetPassword_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)

	svc := NewAuthService(userRepo, tokenStore, "test-secret", 15*time.Minute, 7*24*time.Hour, "test-issuer")
	ctx := context.Background()

	t.Run("revocation failures are non-fatal", func(t *testing.T) {
		tokenStore.EXPECT().ValidatePasswordResetToken(ctx, "valid-token").Return("user_123", nil)
		userRepo.EXPECT().GetByID(ctx, "user_123").Return(&domain.User{ID: "user_123"}, nil)
		userRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)
		tokenStore.EXPECT().RevokePasswordResetToken(ctx, "valid-token").Return(errors.Internal("revoke failed"))
		tokenStore.EXPECT().RevokeAllUserTokens(ctx, "user_123").Return(errors.Internal("revoke failed"))

		err := svc.ResetPassword(ctx, domain.ResetPasswordRequest{Token: "valid-token", NewPassword: "newpass"})

		require.NoError(t, err) // both revocations are non-fatal
	})
}

func TestAuthService_ValidateToken_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)

	svc := NewAuthService(userRepo, tokenStore, "test-secret", 15*time.Minute, 7*24*time.Hour, "test-issuer")
	ctx := context.Background()

	t.Run("permissions defaults to empty slice when missing", func(t *testing.T) {
		// Generate a token without permissions claim
		user := &domain.User{
			ID:           "user_123",
			Email:        "test@example.com",
			PasswordHash: "hash",
			Status:       domain.UserStatusActive,
			Role:         domain.UserRoleAdmin,
			Permissions:  nil, // no permissions
		}

		tokens, err := svc.generateTokenPair(user)
		require.NoError(t, err)

		claims, err := svc.ValidateToken(ctx, tokens.AccessToken)
		require.NoError(t, err)

		assert.NotNil(t, claims.Permissions) // should be empty slice, not nil
		assert.Empty(t, claims.Permissions)
	})

	t.Run("expired token returns INVALID_TOKEN error code", func(t *testing.T) {
		// Create a service with negative duration to generate already-expired tokens
		expiredSvc := NewAuthService(userRepo, tokenStore, "test-secret", -1*time.Second, 7*24*time.Hour, "test-issuer")

		user := &domain.User{
			ID: "user_123", Email: "test@example.com",
			PasswordHash: "hash", Status: domain.UserStatusActive,
			Role: domain.UserRoleAdmin, Permissions: []string{"*"},
		}

		tokens, err := expiredSvc.generateTokenPair(user)
		require.NoError(t, err)

		_, err = svc.ValidateToken(ctx, tokens.AccessToken)

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeInvalidToken, appErr.Code)
	})
}
