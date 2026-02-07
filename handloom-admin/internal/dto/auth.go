package dto

import "github.com/handloom/admin/internal/domain"

// LoginRequest represents the login request body.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// ToDomain converts DTO to domain request.
func (r *LoginRequest) ToDomain() domain.LoginRequest {
	return domain.LoginRequest{
		Email:    r.Email,
		Password: r.Password,
	}
}

// RefreshTokenRequest represents the token refresh request.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ChangePasswordRequest represents the password change request.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

// ToDomain converts DTO to domain request.
func (r *ChangePasswordRequest) ToDomain() domain.ChangePasswordRequest {
	return domain.ChangePasswordRequest{
		CurrentPassword: r.CurrentPassword,
		NewPassword:     r.NewPassword,
	}
}

// ResetPasswordRequest represents the password reset request.
type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// ToDomain converts DTO to domain request.
func (r *ResetPasswordRequest) ToDomain() domain.ResetPasswordRequest {
	return domain.ResetPasswordRequest{
		Token:       r.Token,
		NewPassword: r.NewPassword,
	}
}

// RequestPasswordResetRequest represents the password reset initiation.
type RequestPasswordResetRequest struct {
	Email string `json:"email" validate:"required,email"`
}
