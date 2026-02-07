// Package service implements the business logic layer
package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

// UserService implements domain.UserService
type UserService struct {
	userRepo domain.UserRepository
	logger   *logger.Logger
}

// NewUserService creates a new UserService
func NewUserService(userRepo domain.UserRepository, logger *logger.Logger) *UserService {
	return &UserService{
		userRepo: userRepo,
		logger:   logger,
	}
}

// Create creates a new user
func (s *UserService) Create(ctx context.Context, req domain.CreateUserRequest, createdBy string) (*domain.User, error) {
	// Check if email already exists
	existing, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existing != nil {
		return nil, errors.New(errors.ErrCodeAlreadyExists, "User with this email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.Internal("Failed to hash password")
	}

	user := &domain.User{
		ID:           "user_" + uuid.New().String()[:8],
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Phone:        req.Phone,
		Role:         req.Role,
		Permissions:  req.Permissions,
		Status:       domain.UserStatusPending,
	}
	user.CreatedBy = createdBy

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	s.logger.WithContext(ctx).Infof("Created user: %s", user.ID)

	// Remove sensitive data before returning
	user.PasswordHash = ""
	return user, nil
}

// GetByID retrieves a user by ID
func (s *UserService) GetByID(ctx context.Context, id string) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Remove sensitive data
	user.PasswordHash = ""
	return user, nil
}

// Update updates an existing user
func (s *UserService) Update(ctx context.Context, id string, req domain.UpdateUserRequest, updatedBy string) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.Phone != nil {
		user.Phone = *req.Phone
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.Permissions != nil {
		user.Permissions = req.Permissions
	}

	user.UpdatedBy = updatedBy
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	s.logger.WithContext(ctx).Infof("Updated user: %s", id)

	// Remove sensitive data
	user.PasswordHash = ""
	return user, nil
}

// Delete deletes a user by ID
func (s *UserService) Delete(ctx context.Context, id string) error {
	if err := s.userRepo.Delete(ctx, id); err != nil {
		return err
	}

	s.logger.WithContext(ctx).Infof("Deleted user: %s", id)
	return nil
}

// List retrieves users with filters
func (s *UserService) List(ctx context.Context, req domain.ListUsersRequest) (*domain.ListUsersResponse, error) {
	response, err := s.userRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	// Remove sensitive data
	for _, user := range response.Users {
		user.PasswordHash = ""
	}

	return response, nil
}

// UpdateStatus updates user status
func (s *UserService) UpdateStatus(ctx context.Context, id string, status domain.UserStatus, updatedBy string) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	user.Status = status
	user.UpdatedBy = updatedBy
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	s.logger.WithContext(ctx).Infof("Updated user status: %s -> %s", id, status)
	return nil
}

// Ensure interface compliance
var _ domain.UserService = (*UserService)(nil)
