package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
)

func TestUserService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	log := logger.NewNoop()
	mockTokenStore := mocks.NewMockTokenStore(ctrl)
	service := NewUserService(mockUserRepo, mockTokenStore, log)
	ctx := context.Background()

	t.Run("successful user creation", func(t *testing.T) {
		req := domain.CreateUserRequest{
			Email:     "test@example.com",
			Password:  "password123",
			FirstName: "John",
			LastName:  "Doe",
			Phone:     "+1234567890",
			Role:      domain.UserRoleAdmin,
		}

		// Expect email check (no existing user)
		mockUserRepo.EXPECT().
			GetByEmail(ctx, req.Email).
			Return(nil, errors.NotFound("User not found"))

		// Expect user creation
		mockUserRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, user *domain.User) error {
				assert.Equal(t, req.Email, user.Email)
				assert.Equal(t, req.FirstName, user.FirstName)
				assert.Equal(t, req.LastName, user.LastName)
				assert.Equal(t, req.Role, user.Role)
				assert.Equal(t, domain.UserStatusPending, user.Status)
				// Verify password is hashed
				err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
				assert.NoError(t, err)
				return nil
			})

		user, err := service.Create(ctx, req, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, req.Email, user.Email)
		assert.Empty(t, user.PasswordHash) // Password should be cleared
	})

	t.Run("user with email already exists", func(t *testing.T) {
		req := domain.CreateUserRequest{
			Email:    "existing@example.com",
			Password: "password123",
		}

		existingUser := &domain.User{
			ID:    "user_existing",
			Email: req.Email,
		}

		mockUserRepo.EXPECT().
			GetByEmail(ctx, req.Email).
			Return(existingUser, nil)

		user, err := service.Create(ctx, req, "admin_123")

		assert.Nil(t, user)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestUserService_GetByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	log := logger.NewNoop()
	mockTokenStore := mocks.NewMockTokenStore(ctrl)
	service := NewUserService(mockUserRepo, mockTokenStore, log)
	ctx := context.Background()

	t.Run("successful get by ID", func(t *testing.T) {
		expectedUser := &domain.User{
			ID:           "user_123",
			Email:        "test@example.com",
			PasswordHash: "hashedpassword",
			FirstName:    "John",
			LastName:     "Doe",
		}

		mockUserRepo.EXPECT().
			GetByID(ctx, "user_123").
			Return(expectedUser, nil)

		user, err := service.GetByID(ctx, "user_123")

		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "user_123", user.ID)
		assert.Empty(t, user.PasswordHash) // Password should be cleared
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("User not found"))

		user, err := service.GetByID(ctx, "nonexistent")

		assert.Nil(t, user)
		require.Error(t, err)
	})
}

func TestUserService_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	log := logger.NewNoop()
	mockTokenStore := mocks.NewMockTokenStore(ctrl)
	service := NewUserService(mockUserRepo, mockTokenStore, log)
	ctx := context.Background()

	t.Run("successful update", func(t *testing.T) {
		existingUser := &domain.User{
			ID:        "user_123",
			Email:     "test@example.com",
			FirstName: "John",
			LastName:  "Doe",
			Role:      domain.UserRoleAdmin,
		}

		newFirstName := "Jane"
		newPhone := "+9876543210"
		req := domain.UpdateUserRequest{
			FirstName: &newFirstName,
			Phone:     &newPhone,
		}

		mockUserRepo.EXPECT().
			GetByID(ctx, "user_123").
			Return(existingUser, nil)

		mockUserRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, user *domain.User) error {
				assert.Equal(t, newFirstName, user.FirstName)
				assert.Equal(t, newPhone, user.Phone)
				assert.Equal(t, "admin_456", user.UpdatedBy)
				return nil
			})

		user, err := service.Update(ctx, "user_123", req, "admin_456")

		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, newFirstName, user.FirstName)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("User not found"))

		user, err := service.Update(ctx, "nonexistent", domain.UpdateUserRequest{}, "admin_456")

		assert.Nil(t, user)
		require.Error(t, err)
	})
}

func TestUserService_Delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	log := logger.NewNoop()
	mockTokenStore := mocks.NewMockTokenStore(ctrl)
	service := NewUserService(mockUserRepo, mockTokenStore, log)
	ctx := context.Background()

	t.Run("successful delete", func(t *testing.T) {
		mockUserRepo.EXPECT().
			Delete(ctx, "user_123").
			Return(nil)
		mockTokenStore.EXPECT().
			RevokeAllUserTokens(ctx, "user_123").
			Return(nil)

		err := service.Delete(ctx, "user_123")

		require.NoError(t, err)
	})

	t.Run("delete non-existent user", func(t *testing.T) {
		mockUserRepo.EXPECT().
			Delete(ctx, "nonexistent").
			Return(errors.NotFound("User not found"))

		err := service.Delete(ctx, "nonexistent")

		require.Error(t, err)
	})

	t.Run("delete - token revoke failure is non-fatal", func(t *testing.T) {
		mockUserRepo.EXPECT().
			Delete(ctx, "user_456").
			Return(nil)
		mockTokenStore.EXPECT().
			RevokeAllUserTokens(ctx, "user_456").
			Return(errors.Internal("redis down"))

		err := service.Delete(ctx, "user_456")
		require.NoError(t, err) // should still succeed
	})
}

func TestUserService_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	log := logger.NewNoop()
	mockTokenStore := mocks.NewMockTokenStore(ctrl)
	service := NewUserService(mockUserRepo, mockTokenStore, log)
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		req := domain.ListUsersRequest{
			PaginationRequest: domain.PaginationRequest{
				Limit: 20,
			},
		}

		expectedResponse := &domain.ListUsersResponse{
			Users: []*domain.User{
				{ID: "user_1", Email: "user1@example.com", PasswordHash: "hash1"},
				{ID: "user_2", Email: "user2@example.com", PasswordHash: "hash2"},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockUserRepo.EXPECT().
			List(ctx, req).
			Return(expectedResponse, nil)

		response, err := service.List(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Users, 2)
		// Verify passwords are cleared
		for _, user := range response.Users {
			assert.Empty(t, user.PasswordHash)
		}
	})

	t.Run("list with filters", func(t *testing.T) {
		role := domain.UserRoleAdmin
		req := domain.ListUsersRequest{
			PaginationRequest: domain.PaginationRequest{
				Limit: 10,
			},
			Role: &role,
		}

		expectedResponse := &domain.ListUsersResponse{
			Users: []*domain.User{
				{ID: "user_1", Email: "admin@example.com", Role: domain.UserRoleAdmin, PasswordHash: "hash1"},
			},
			Pagination: domain.PaginationResponse{
				Limit:   10,
				HasMore: false,
			},
		}

		mockUserRepo.EXPECT().
			List(ctx, req).
			Return(expectedResponse, nil)

		response, err := service.List(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Users, 1)
	})
}

func TestUserService_UpdateStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	log := logger.NewNoop()
	mockTokenStore := mocks.NewMockTokenStore(ctrl)
	service := NewUserService(mockUserRepo, mockTokenStore, log)
	ctx := context.Background()

	t.Run("successful status update", func(t *testing.T) {
		existingUser := &domain.User{
			ID:     "user_123",
			Email:  "test@example.com",
			Status: domain.UserStatusPending,
		}

		mockUserRepo.EXPECT().
			GetByID(ctx, "user_123").
			Return(existingUser, nil)

		mockUserRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, user *domain.User) error {
				assert.Equal(t, domain.UserStatusActive, user.Status)
				assert.Equal(t, "admin_789", user.UpdatedBy)
				return nil
			})

		err := service.UpdateStatus(ctx, "user_123", domain.UserStatusActive, "admin_789")

		require.NoError(t, err)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("User not found"))

		err := service.UpdateStatus(ctx, "nonexistent", domain.UserStatusActive, "admin_789")

		require.Error(t, err)
	})
}

func TestUserService_Create_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockTokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.NewNoop()
	svc := NewUserService(mockUserRepo, mockTokenStore, log)
	ctx := context.Background()

	t.Run("new user starts with PENDING status", func(t *testing.T) {
		mockUserRepo.EXPECT().GetByEmail(ctx, "new@example.com").Return(nil, errors.NotFound("User"))
		mockUserRepo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, user *domain.User) error {
			assert.Equal(t, domain.UserStatusPending, user.Status)
			return nil
		})

		user, err := svc.Create(ctx, domain.CreateUserRequest{
			Email:    "new@example.com",
			Password: "password123",
			Role:     domain.UserRoleOperator,
		}, "admin_1")

		require.NoError(t, err)
		assert.Equal(t, domain.UserStatusPending, user.Status)
	})

	t.Run("password hash not in returned user", func(t *testing.T) {
		mockUserRepo.EXPECT().GetByEmail(ctx, "safe@example.com").Return(nil, errors.NotFound("User"))
		mockUserRepo.EXPECT().Create(ctx, gomock.Any()).Return(nil)

		user, err := svc.Create(ctx, domain.CreateUserRequest{
			Email:    "safe@example.com",
			Password: "password123",
		}, "admin_1")

		require.NoError(t, err)
		assert.Empty(t, user.PasswordHash) // security: never expose hash
	})

	t.Run("duplicate email returns ALREADY_EXISTS error code", func(t *testing.T) {
		mockUserRepo.EXPECT().GetByEmail(ctx, "dup@example.com").Return(&domain.User{
			ID: "user_existing", Email: "dup@example.com",
		}, nil)

		_, err := svc.Create(ctx, domain.CreateUserRequest{
			Email:    "dup@example.com",
			Password: "password123",
		}, "admin_1")

		require.Error(t, err)
		var appErr *errors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, errors.ErrCodeAlreadyExists, appErr.Code)
	})

	t.Run("repo create failure returns error", func(t *testing.T) {
		mockUserRepo.EXPECT().GetByEmail(ctx, "fail@example.com").Return(nil, errors.NotFound("User"))
		mockUserRepo.EXPECT().Create(ctx, gomock.Any()).Return(errors.Internal("db error"))

		user, err := svc.Create(ctx, domain.CreateUserRequest{
			Email:    "fail@example.com",
			Password: "password123",
		}, "admin_1")

		assert.Nil(t, user)
		require.Error(t, err)
	})
}

func TestUserService_GetByID_Security(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockTokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.NewNoop()
	svc := NewUserService(mockUserRepo, mockTokenStore, log)
	ctx := context.Background()

	t.Run("password hash cleared even when repo returns one", func(t *testing.T) {
		mockUserRepo.EXPECT().GetByID(ctx, "user_123").Return(&domain.User{
			ID:           "user_123",
			Email:        "test@example.com",
			PasswordHash: "$2a$10$somebcrypthash", // repo returns hash
		}, nil)

		user, err := svc.GetByID(ctx, "user_123")

		require.NoError(t, err)
		assert.Equal(t, "user_123", user.ID)
		assert.Empty(t, user.PasswordHash) // must be cleared
	})
}

func TestUserService_Update_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockTokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.NewNoop()
	svc := NewUserService(mockUserRepo, mockTokenStore, log)
	ctx := context.Background()

	t.Run("empty password skips hashing", func(t *testing.T) {
		existing := &domain.User{
			ID:           "user_123",
			PasswordHash: "original_hash",
		}

		emptyPass := ""
		req := domain.UpdateUserRequest{Password: &emptyPass}

		mockUserRepo.EXPECT().GetByID(ctx, "user_123").Return(existing, nil)
		mockUserRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, user *domain.User) error {
			// Password hash should remain unchanged since empty password was provided
			assert.Equal(t, "original_hash", user.PasswordHash)
			return nil
		})

		_, err := svc.Update(ctx, "user_123", req, "admin_1")
		require.NoError(t, err)
	})

	t.Run("permission update replaces entire list", func(t *testing.T) {
		existing := &domain.User{
			ID:          "user_123",
			Permissions: []string{"read", "write", "delete"},
		}

		req := domain.UpdateUserRequest{
			Permissions: []string{"read"}, // replace with single permission
		}

		mockUserRepo.EXPECT().GetByID(ctx, "user_123").Return(existing, nil)
		mockUserRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, user *domain.User) error {
			assert.Equal(t, []string{"read"}, user.Permissions) // old permissions gone
			return nil
		})

		_, err := svc.Update(ctx, "user_123", req, "admin_1")
		require.NoError(t, err)
	})

	t.Run("sets updated_at timestamp", func(t *testing.T) {
		existing := &domain.User{ID: "user_123"}

		newName := "Updated"
		req := domain.UpdateUserRequest{FirstName: &newName}

		mockUserRepo.EXPECT().GetByID(ctx, "user_123").Return(existing, nil)
		mockUserRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, user *domain.User) error {
			assert.False(t, user.UpdatedAt.IsZero()) // timestamp should be set
			return nil
		})

		_, err := svc.Update(ctx, "user_123", req, "admin_1")
		require.NoError(t, err)
	})

	t.Run("returned user has cleared password", func(t *testing.T) {
		existing := &domain.User{
			ID:           "user_123",
			PasswordHash: "some_hash",
		}

		newName := "Test"
		req := domain.UpdateUserRequest{FirstName: &newName}

		mockUserRepo.EXPECT().GetByID(ctx, "user_123").Return(existing, nil)
		mockUserRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		user, err := svc.Update(ctx, "user_123", req, "admin_1")

		require.NoError(t, err)
		assert.Empty(t, user.PasswordHash)
	})
}

func TestUserService_List_Security(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockTokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.NewNoop()
	svc := NewUserService(mockUserRepo, mockTokenStore, log)
	ctx := context.Background()

	t.Run("all users have password cleared in batch", func(t *testing.T) {
		mockUserRepo.EXPECT().List(ctx, gomock.Any()).Return(&domain.ListUsersResponse{
			Users: []*domain.User{
				{ID: "u1", PasswordHash: "hash1"},
				{ID: "u2", PasswordHash: "hash2"},
				{ID: "u3", PasswordHash: "hash3"},
			},
			Pagination: domain.PaginationResponse{Limit: 20},
		}, nil)

		resp, err := svc.List(ctx, domain.ListUsersRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 20},
		})

		require.NoError(t, err)
		assert.Len(t, resp.Users, 3)
		for _, u := range resp.Users {
			assert.Empty(t, u.PasswordHash, "user %s should have password cleared", u.ID)
		}
	})
}

func TestUserService_UpdateStatus_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	mockTokenStore := mocks.NewMockTokenStore(ctrl)
	log := logger.NewNoop()
	svc := NewUserService(mockUserRepo, mockTokenStore, log)
	ctx := context.Background()

	t.Run("sets updated_at timestamp", func(t *testing.T) {
		mockUserRepo.EXPECT().GetByID(ctx, "user_123").Return(&domain.User{
			ID:     "user_123",
			Status: domain.UserStatusPending,
		}, nil)

		mockUserRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(ctx context.Context, user *domain.User) error {
			assert.False(t, user.UpdatedAt.IsZero())
			assert.Equal(t, "admin_1", user.UpdatedBy)
			return nil
		})

		err := svc.UpdateStatus(ctx, "user_123", domain.UserStatusActive, "admin_1")
		require.NoError(t, err)
	})

	t.Run("repo update failure returns error", func(t *testing.T) {
		mockUserRepo.EXPECT().GetByID(ctx, "user_123").Return(&domain.User{
			ID: "user_123", Status: domain.UserStatusPending,
		}, nil)

		mockUserRepo.EXPECT().Update(ctx, gomock.Any()).Return(errors.Internal("db error"))

		err := svc.UpdateStatus(ctx, "user_123", domain.UserStatusActive, "admin_1")
		require.Error(t, err)
	})
}
