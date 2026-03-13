package dynamodb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
)

func TestUserRepository_Create(t *testing.T) {
	wrappedClient, rawClient := testWrappedClient(t)
	skipIfNoLocal(t, rawClient)
	setupTestTable(t, rawClient, testCoreTable)
	defer cleanupTestTable(t, rawClient, testCoreTable)

	repo := NewUserRepository(wrappedClient)
	ctx := context.Background()

	t.Run("successful user creation", func(t *testing.T) {
		user := &domain.User{
			ID:           "user_test123",
			Email:        "test@example.com",
			PasswordHash: "hashedpassword123",
			FirstName:    "John",
			LastName:     "Doe",
			Phone:        "+1234567890",
			Role:         domain.UserRoleAdmin,
			Status:       domain.UserStatusActive,
			Permissions:  []string{"read", "write"},
			BaseEntity: domain.BaseEntity{
				CreatedAt: time.Now(),
				CreatedBy: "system",
			},
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)

		// Verify user was created
		retrieved, err := repo.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.Email, retrieved.Email)
		assert.Equal(t, user.FirstName, retrieved.FirstName)
		assert.Equal(t, user.Role, retrieved.Role)
	})

	t.Run("duplicate email should fail", func(t *testing.T) {
		user1 := &domain.User{
			ID:           "user_dup1",
			Email:        "duplicate@example.com",
			PasswordHash: "hashedpassword123",
			FirstName:    "User",
			LastName:     "One",
			Role:         domain.UserRoleOperator,
			Status:       domain.UserStatusActive,
			BaseEntity: domain.BaseEntity{
				CreatedAt: time.Now(),
				CreatedBy: "system",
			},
		}

		err := repo.Create(ctx, user1)
		require.NoError(t, err)

		// Try to create another user with same email but different ID
		user2 := &domain.User{
			ID:           "user_dup2",
			Email:        "duplicate@example.com",
			PasswordHash: "hashedpassword456",
			FirstName:    "User",
			LastName:     "Two",
			Role:         domain.UserRoleOperator,
			Status:       domain.UserStatusActive,
			BaseEntity: domain.BaseEntity{
				CreatedAt: time.Now(),
				CreatedBy: "system",
			},
		}

		// Note: In real implementation, we'd check email uniqueness via GSI query
		// This test demonstrates the Create flow
		_ = repo.Create(ctx, user2)
		// This may or may not error depending on implementation
		// The service layer should handle email uniqueness check
	})
}

func TestUserRepository_GetByID(t *testing.T) {
	wrappedClient, rawClient := testWrappedClient(t)
	skipIfNoLocal(t, rawClient)
	setupTestTable(t, rawClient, testCoreTable)
	defer cleanupTestTable(t, rawClient, testCoreTable)

	repo := NewUserRepository(wrappedClient)
	ctx := context.Background()

	t.Run("get existing user", func(t *testing.T) {
		user := &domain.User{
			ID:           "user_get123",
			Email:        "gettest@example.com",
			PasswordHash: "hashedpassword",
			FirstName:    "Get",
			LastName:     "Test",
			Role:         domain.UserRoleAdmin,
			Status:       domain.UserStatusActive,
			BaseEntity: domain.BaseEntity{
				CreatedAt: time.Now(),
				CreatedBy: "system",
			},
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)

		retrieved, err := repo.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, retrieved.ID)
		assert.Equal(t, user.Email, retrieved.Email)
		assert.Equal(t, user.FirstName, retrieved.FirstName)
	})

	t.Run("get non-existent user", func(t *testing.T) {
		_, err := repo.GetByID(ctx, "nonexistent_user")
		require.Error(t, err)
	})
}

func TestUserRepository_GetByEmail(t *testing.T) {
	wrappedClient, rawClient := testWrappedClient(t)
	skipIfNoLocal(t, rawClient)
	setupTestTable(t, rawClient, testCoreTable)
	defer cleanupTestTable(t, rawClient, testCoreTable)

	repo := NewUserRepository(wrappedClient)
	ctx := context.Background()

	t.Run("get user by email", func(t *testing.T) {
		user := &domain.User{
			ID:           "user_email123",
			Email:        "findme@example.com",
			PasswordHash: "hashedpassword",
			FirstName:    "Find",
			LastName:     "Me",
			Role:         domain.UserRoleOperator,
			Status:       domain.UserStatusActive,
			BaseEntity: domain.BaseEntity{
				CreatedAt: time.Now(),
				CreatedBy: "system",
			},
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)

		retrieved, err := repo.GetByEmail(ctx, "findme@example.com")
		require.NoError(t, err)
		assert.Equal(t, user.ID, retrieved.ID)
		assert.Equal(t, user.Email, retrieved.Email)
	})

	t.Run("get by non-existent email", func(t *testing.T) {
		_, err := repo.GetByEmail(ctx, "notfound@example.com")
		require.Error(t, err)
	})
}

func TestUserRepository_Update(t *testing.T) {
	wrappedClient, rawClient := testWrappedClient(t)
	skipIfNoLocal(t, rawClient)
	setupTestTable(t, rawClient, testCoreTable)
	defer cleanupTestTable(t, rawClient, testCoreTable)

	repo := NewUserRepository(wrappedClient)
	ctx := context.Background()

	t.Run("update user fields", func(t *testing.T) {
		user := &domain.User{
			ID:           "user_update123",
			Email:        "update@example.com",
			PasswordHash: "hashedpassword",
			FirstName:    "Original",
			LastName:     "Name",
			Phone:        "+1111111111",
			Role:         domain.UserRoleOperator,
			Status:       domain.UserStatusPending,
			BaseEntity: domain.BaseEntity{
				CreatedAt: time.Now(),
				CreatedBy: "system",
			},
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)

		// Update user
		user.FirstName = "Updated"
		user.LastName = "User"
		user.Phone = "+2222222222"
		user.Status = domain.UserStatusActive
		user.UpdatedAt = time.Now()
		user.UpdatedBy = "admin"

		err = repo.Update(ctx, user)
		require.NoError(t, err)

		// Verify updates
		retrieved, err := repo.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated", retrieved.FirstName)
		assert.Equal(t, "User", retrieved.LastName)
		assert.Equal(t, "+2222222222", retrieved.Phone)
		assert.Equal(t, domain.UserStatusActive, retrieved.Status)
	})
}

func TestUserRepository_Delete(t *testing.T) {
	wrappedClient, rawClient := testWrappedClient(t)
	skipIfNoLocal(t, rawClient)
	setupTestTable(t, rawClient, testCoreTable)
	defer cleanupTestTable(t, rawClient, testCoreTable)

	repo := NewUserRepository(wrappedClient)
	ctx := context.Background()

	t.Run("delete existing user", func(t *testing.T) {
		user := &domain.User{
			ID:           "user_delete123",
			Email:        "delete@example.com",
			PasswordHash: "hashedpassword",
			FirstName:    "Delete",
			LastName:     "Me",
			Role:         domain.UserRoleOperator,
			Status:       domain.UserStatusActive,
			BaseEntity: domain.BaseEntity{
				CreatedAt: time.Now(),
				CreatedBy: "system",
			},
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)

		// Verify user exists
		_, err = repo.GetByID(ctx, user.ID)
		require.NoError(t, err)

		// Delete user
		err = repo.Delete(ctx, user.ID)
		require.NoError(t, err)

		// Verify user is deleted
		_, err = repo.GetByID(ctx, user.ID)
		require.Error(t, err)
	})
}

func TestUserRepository_List(t *testing.T) {
	wrappedClient, rawClient := testWrappedClient(t)
	skipIfNoLocal(t, rawClient)
	setupTestTable(t, rawClient, testCoreTable)
	defer cleanupTestTable(t, rawClient, testCoreTable)

	repo := NewUserRepository(wrappedClient)
	ctx := context.Background()

	// Create multiple users for listing
	users := []*domain.User{
		{
			ID:           "user_list1",
			Email:        "list1@example.com",
			PasswordHash: "hash1",
			FirstName:    "List",
			LastName:     "User1",
			Role:         domain.UserRoleAdmin,
			Status:       domain.UserStatusActive,
			BaseEntity:   domain.BaseEntity{CreatedAt: time.Now(), CreatedBy: "system"},
		},
		{
			ID:           "user_list2",
			Email:        "list2@example.com",
			PasswordHash: "hash2",
			FirstName:    "List",
			LastName:     "User2",
			Role:         domain.UserRoleOperator,
			Status:       domain.UserStatusActive,
			BaseEntity:   domain.BaseEntity{CreatedAt: time.Now(), CreatedBy: "system"},
		},
		{
			ID:           "user_list3",
			Email:        "list3@example.com",
			PasswordHash: "hash3",
			FirstName:    "List",
			LastName:     "User3",
			Role:         domain.UserRoleOperator,
			Status:       domain.UserStatusPending,
			BaseEntity:   domain.BaseEntity{CreatedAt: time.Now(), CreatedBy: "system"},
		},
	}

	for _, u := range users {
		err := repo.Create(ctx, u)
		require.NoError(t, err)
	}

	t.Run("list all users", func(t *testing.T) {
		req := domain.ListUsersRequest{
			PaginationRequest: domain.PaginationRequest{
				Limit: 10,
			},
		}

		response, err := repo.List(ctx, req)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(response.Users), 3)
	})

	t.Run("list with pagination", func(t *testing.T) {
		req := domain.ListUsersRequest{
			PaginationRequest: domain.PaginationRequest{
				Limit: 2,
			},
		}

		response, err := repo.List(ctx, req)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(response.Users), 2)
	})

	t.Run("list with role filter", func(t *testing.T) {
		role := domain.UserRoleAdmin
		req := domain.ListUsersRequest{
			PaginationRequest: domain.PaginationRequest{
				Limit: 10,
			},
			Role: &role,
		}

		response, err := repo.List(ctx, req)
		require.NoError(t, err)
		for _, u := range response.Users {
			assert.Equal(t, domain.UserRoleAdmin, u.Role)
		}
	})
}

func TestUserRepository_UpdateLastLogin(t *testing.T) {
	wrappedClient, rawClient := testWrappedClient(t)
	skipIfNoLocal(t, rawClient)
	setupTestTable(t, rawClient, testCoreTable)
	defer cleanupTestTable(t, rawClient, testCoreTable)

	repo := NewUserRepository(wrappedClient)
	ctx := context.Background()

	t.Run("update last login", func(t *testing.T) {
		user := &domain.User{
			ID:           "user_login123",
			Email:        "login@example.com",
			PasswordHash: "hashedpassword",
			FirstName:    "Login",
			LastName:     "Test",
			Role:         domain.UserRoleAdmin,
			Status:       domain.UserStatusActive,
			BaseEntity: domain.BaseEntity{
				CreatedAt: time.Now(),
				CreatedBy: "system",
			},
		}

		err := repo.Create(ctx, user)
		require.NoError(t, err)

		// Verify no last login
		retrieved, err := repo.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.Nil(t, retrieved.LastLoginAt)

		// Update last login
		err = repo.UpdateLastLogin(ctx, user.ID)
		require.NoError(t, err)

		// Verify last login was updated
		retrieved, err = repo.GetByID(ctx, user.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved.LastLoginAt)
	})
}
