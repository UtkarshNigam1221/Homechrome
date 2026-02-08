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
	"github.com/handloom/admin/pkg/logger"
)

func TestAuthService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.New(true)

	service := NewAuthService(
		userRepo,
		tokenStore,
		log,
		"test-secret-key",
		15*time.Minute,
		7*24*time.Hour,
		"test-issuer",
	)

	ctx := context.Background()

	// Hash password for test user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	tests := []struct {
		name      string
		req       domain.LoginRequest
		setup     func()
		wantErr   bool
		errCode   string
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
	log := logger.New(true)

	service := NewAuthService(
		userRepo,
		tokenStore,
		log,
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
	log := logger.New(true)

	service := NewAuthService(
		userRepo,
		tokenStore,
		log,
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
	log := logger.New(true)

	service := NewAuthService(
		userRepo,
		tokenStore,
		log,
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
	log := logger.New(true)

	service := NewAuthService(
		userRepo,
		tokenStore,
		log,
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
