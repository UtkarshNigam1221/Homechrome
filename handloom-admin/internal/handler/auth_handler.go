// Package handler implements HTTP handlers
package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
	"github.com/handloom/admin/pkg/response"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	authService domain.AuthService
	logger      *logger.Logger
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(authService domain.AuthService, logger *logger.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		logger:      logger,
	}
}

// Routes returns the auth routes
func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/login", h.Login)
	r.Post("/refresh", h.RefreshToken)
	r.Post("/logout", h.Logout)
	r.Post("/password/change", h.ChangePassword)
	r.Post("/password/reset-request", h.RequestPasswordReset)
	r.Post("/password/reset", h.ResetPassword)

	return r
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req domain.LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	// Validate request
	if req.Email == "" || req.Password == "" {
		response.Error(w, errors.Validation("Email and password are required"))
		return
	}

	result, err := h.authService.Login(ctx, req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if req.RefreshToken == "" {
		response.Error(w, errors.Validation("Refresh token is required"))
		return
	}

	tokens, err := h.authService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, tokens)
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context (set by auth middleware)
	userID := getUserIDFromContext(ctx)
	if userID == "" {
		response.Error(w, errors.Unauthorized("User not authenticated"))
		return
	}

	if err := h.authService.Logout(ctx, userID); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// ChangePassword handles password change
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := getUserIDFromContext(ctx)
	if userID == "" {
		response.Error(w, errors.Unauthorized("User not authenticated"))
		return
	}

	var req domain.ChangePasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		response.Error(w, errors.Validation("Current password and new password are required"))
		return
	}

	if len(req.NewPassword) < 8 {
		response.Error(w, errors.Validation("New password must be at least 8 characters"))
		return
	}

	if err := h.authService.ChangePassword(ctx, userID, req); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Password changed successfully"})
}

// RequestPasswordReset handles password reset request
func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if req.Email == "" {
		response.Error(w, errors.Validation("Email is required"))
		return
	}

	// Don't reveal if email exists or not
	_ = h.authService.RequestPasswordReset(ctx, req.Email)

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "If the email exists, a password reset link has been sent",
	})
}

// ResetPassword handles password reset
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req domain.ResetPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	if req.Token == "" || req.NewPassword == "" {
		response.Error(w, errors.Validation("Token and new password are required"))
		return
	}

	if len(req.NewPassword) < 8 {
		response.Error(w, errors.Validation("New password must be at least 8 characters"))
		return
	}

	if err := h.authService.ResetPassword(ctx, req); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Password reset successfully"})
}

// ExtractToken extracts the Bearer token from the Authorization header
func ExtractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}
