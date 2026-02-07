package dto

import "github.com/handloom/admin/internal/domain"

// CreateUserRequest represents the user creation request.
type CreateUserRequest struct {
	Email       string          `json:"email" validate:"required,email"`
	Password    string          `json:"password" validate:"required,min=8"`
	FirstName   string          `json:"first_name" validate:"required"`
	LastName    string          `json:"last_name" validate:"required"`
	Phone       string          `json:"phone,omitempty"`
	Role        domain.UserRole `json:"role" validate:"required,oneof=ADMIN OPERATOR"`
	Permissions []string        `json:"permissions,omitempty"`
}

// ToDomain converts DTO to domain request.
func (r *CreateUserRequest) ToDomain() domain.CreateUserRequest {
	return domain.CreateUserRequest{
		Email:       r.Email,
		Password:    r.Password,
		FirstName:   r.FirstName,
		LastName:    r.LastName,
		Phone:       r.Phone,
		Role:        r.Role,
		Permissions: r.Permissions,
	}
}

// UpdateUserRequest represents the user update request.
type UpdateUserRequest struct {
	FirstName   *string          `json:"first_name,omitempty"`
	LastName    *string          `json:"last_name,omitempty"`
	Phone       *string          `json:"phone,omitempty"`
	Role        *domain.UserRole `json:"role,omitempty" validate:"omitempty,oneof=ADMIN OPERATOR"`
	Permissions []string         `json:"permissions,omitempty"`
}

// ToDomain converts DTO to domain request.
func (r *UpdateUserRequest) ToDomain() domain.UpdateUserRequest {
	return domain.UpdateUserRequest{
		FirstName:   r.FirstName,
		LastName:    r.LastName,
		Phone:       r.Phone,
		Role:        r.Role,
		Permissions: r.Permissions,
	}
}

// UpdateUserStatusRequest represents the user status update request.
type UpdateUserStatusRequest struct {
	Status domain.UserStatus `json:"status" validate:"required,oneof=ACTIVE INACTIVE PENDING"`
}
