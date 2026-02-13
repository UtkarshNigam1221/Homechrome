// Package handler implements HTTP handlers
package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
	"github.com/handloom/admin/pkg/response"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	authService domain.AuthService
	userService domain.UserService
	logger      *logger.Logger
	validation  *middleware.Validation
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(authService domain.AuthService, userService domain.UserService, logger *logger.Logger, validation *middleware.Validation) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userService: userService,
		logger:      logger,
		validation:  validation,
	}
}

// Routes returns all auth routes. The authenticate parameter is the auth middleware
// to apply to protected routes (logout, me, change-password).
func (h *AuthHandler) Routes(authenticate func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Public routes (no auth required)
	r.With(middleware.ValidateJSONTyped[domain.LoginRequest](h.validation)).Post("/login", h.Login)
	r.With(middleware.ValidateJSONTyped[RefreshTokenRequest](h.validation)).Post("/refresh", h.RefreshToken)
	r.With(middleware.ValidateJSONTyped[PasswordResetEmailRequest](h.validation)).Post("/password/reset-request", h.RequestPasswordReset)
	r.With(middleware.ValidateJSONTyped[domain.ResetPasswordRequest](h.validation)).Post("/password/reset", h.ResetPassword)

	// Protected routes (require authentication)
	r.Group(func(r chi.Router) {
		r.Use(authenticate)
		r.Post("/logout", h.Logout)
		r.Get("/me", h.GetCurrentUser)
		r.With(middleware.ValidateJSONTyped[domain.ChangePasswordRequest](h.validation)).Post("/password/change", h.ChangePassword)
	})

	return r
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := middleware.MustGetValidatedBody[domain.LoginRequest](ctx)

	result, err := h.authService.Login(ctx, *req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := middleware.MustGetValidatedBody[RefreshTokenRequest](ctx)

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

	userID := middleware.GetUserIDFromContext(ctx)
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

// GetCurrentUser returns the authenticated user's profile
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := middleware.GetUserIDFromContext(ctx)
	if userID == "" {
		response.Error(w, errors.Unauthorized("User not authenticated"))
		return
	}

	user, err := h.userService.GetByID(ctx, userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, user)
}

// ChangePassword handles password change
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID := middleware.GetUserIDFromContext(ctx)
	if userID == "" {
		response.Error(w, errors.Unauthorized("User not authenticated"))
		return
	}

	req := middleware.MustGetValidatedBody[domain.ChangePasswordRequest](ctx)

	if err := h.authService.ChangePassword(ctx, userID, *req); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Password changed successfully"})
}

// RequestPasswordReset handles password reset request
func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := middleware.MustGetValidatedBody[PasswordResetEmailRequest](ctx)

	// Don't reveal if email exists or not
	_ = h.authService.RequestPasswordReset(ctx, req.Email)

	response.JSON(w, http.StatusOK, map[string]string{
		"message": "If the email exists, a password reset link has been sent",
	})
}

// ResetPassword handles password reset
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := middleware.MustGetValidatedBody[domain.ResetPasswordRequest](ctx)

	if err := h.authService.ResetPassword(ctx, *req); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "Password reset successfully"})
}
