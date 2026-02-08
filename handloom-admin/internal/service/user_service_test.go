package service

import (
	"context"
	"testing"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func TestUserService_Create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	log := logger.NewNoop()
	service := NewUserService(mockUserRepo, log)
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
	service := NewUserService(mockUserRepo, log)
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
	service := NewUserService(mockUserRepo, log)
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
	service := NewUserService(mockUserRepo, log)
	ctx := context.Background()

	t.Run("successful delete", func(t *testing.T) {
		mockUserRepo.EXPECT().
			Delete(ctx, "user_123").
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
}

func TestUserService_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	log := logger.NewNoop()
	service := NewUserService(mockUserRepo, log)
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		req := domain.ListUsersRequest{
			PaginationRequest: domain.PaginationRequest{
				Page:    1,
				PerPage: 20,
			},
		}

		expectedResponse := &domain.ListUsersResponse{
			Users: []*domain.User{
				{ID: "user_1", Email: "user1@example.com", PasswordHash: "hash1"},
				{ID: "user_2", Email: "user2@example.com", PasswordHash: "hash2"},
			},
			Pagination: domain.PaginationResponse{
				CurrentPage:1,
				PerPage:    20,
				TotalCount: 2,
				TotalPages: 1,
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
				Page:    1,
				PerPage: 10,
			},
			Role: &role,
		}

		expectedResponse := &domain.ListUsersResponse{
			Users: []*domain.User{
				{ID: "user_1", Email: "admin@example.com", Role: domain.UserRoleAdmin, PasswordHash: "hash1"},
			},
			Pagination: domain.PaginationResponse{
				CurrentPage:1,
				PerPage:    10,
				TotalCount: 1,
				TotalPages: 1,
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
	service := NewUserService(mockUserRepo, log)
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
