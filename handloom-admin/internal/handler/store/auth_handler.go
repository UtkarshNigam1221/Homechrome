// Package store implements HTTP handlers for the B2C storefront
package store

import (
	stderrors "errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/response"
)

// AuthHandler handles customer authentication requests
type AuthHandler struct {
	customerAuthService domain.CustomerAuthService
	cartService         domain.CartService
	validation          *middleware.Validation
}

// NewAuthHandler creates a new store AuthHandler
func NewAuthHandler(customerAuthService domain.CustomerAuthService, cartService domain.CartService, validation *middleware.Validation) *AuthHandler {
	return &AuthHandler{
		customerAuthService: customerAuthService,
		cartService:         cartService,
		validation:          validation,
	}
}

// Routes returns all store auth routes. The authenticate parameter is the customer auth
// middleware to apply to protected routes (logout).
func (h *AuthHandler) Routes(authenticate func(http.Handler) http.Handler) chi.Router {
	r := chi.NewRouter()

	// Public routes (no auth required)
	r.With(middleware.ValidateJSONTyped[domain.SendOTPRequest](h.validation)).Post("/otp/send", h.SendOTP)
	r.With(middleware.ValidateJSONTyped[domain.VerifyOTPRequest](h.validation)).Post("/otp/verify", h.VerifyOTP)
	r.Post("/refresh", h.RefreshToken)

	// Protected routes (require customer authentication)
	r.Group(func(r chi.Router) {
		r.Use(authenticate)
		r.Post("/logout", h.Logout)
	})

	return r
}

func (h *AuthHandler) setStoreCookies(w http.ResponseWriter, tokens *domain.TokenPair) {
	secure, sameSite, domain := middleware.AuthCookieSettings()

	// Secure is dynamic by design: false on plain-HTTP local dev, true behind
	// the custom domain / Lambda URL. HttpOnly + SameSite are always set.
	//nolint:gosec // G124: Secure flag is environment-conditional, not omitted.
	http.SetCookie(w, &http.Cookie{
		Name:     "store_token",
		Value:    tokens.AccessToken,
		Path:     "/",
		Domain:   domain,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   int(15 * time.Minute / time.Second),
	})

	//nolint:gosec // G124: Secure flag is environment-conditional, not omitted.
	http.SetCookie(w, &http.Cookie{
		Name:     "store_refresh",
		Value:    tokens.RefreshToken,
		Path:     "/",
		Domain:   domain,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   int(7 * 24 * time.Hour / time.Second),
	})
}

// isTerminalAuthError reports whether an error means the customer's credential
// is spent — a malformed or revoked token, or a deactivated account — as
// opposed to an infrastructure failure that leaves the token's status unknown.
// Only the former justifies clearing the auth cookies.
func isTerminalAuthError(err error) bool {
	var appErr *errors.AppError
	if !stderrors.As(err, &appErr) {
		return false
	}
	switch appErr.Code {
	case errors.ErrCodeInvalidToken, errors.ErrCodeTokenExpired, errors.ErrCodeTokenInvalid,
		errors.ErrCodeInvalidCredentials, errors.ErrCodeUnauthorized, errors.ErrCodeUserInactive:
		return true
	default:
		return false
	}
}

func (h *AuthHandler) clearStoreCookies(w http.ResponseWriter) {
	secure, sameSite, domain := middleware.AuthCookieSettings()

	//nolint:gosec // G124: Secure flag is environment-conditional, not omitted.
	http.SetCookie(w, &http.Cookie{
		Name:     "store_token",
		Value:    "",
		Path:     "/",
		Domain:   domain,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   -1,
	})

	//nolint:gosec // G124: Secure flag is environment-conditional, not omitted.
	http.SetCookie(w, &http.Cookie{
		Name:     "store_refresh",
		Value:    "",
		Path:     "/",
		Domain:   domain,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   -1,
	})
}

// SendOTP handles sending OTP to a phone number
func (h *AuthHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := middleware.MustGetValidatedBody[domain.SendOTPRequest](ctx)

	if err := h.customerAuthService.SendOTP(ctx, req.Phone); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		response.KeyMessage: "OTP sent successfully",
	})
}

// VerifyOTP handles OTP verification and returns customer data
func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := middleware.MustGetValidatedBody[domain.VerifyOTPRequest](ctx)

	customer, tokens, isNewCustomer, err := h.customerAuthService.VerifyOTP(ctx, req.Phone, req.Code)
	if err != nil {
		response.Error(w, err)
		return
	}

	h.setStoreCookies(w, tokens)

	// Merge guest cart if guest_session cookie is present and valid UUID
	if cookie, err := r.Cookie("guest_session"); err == nil && cookie.Value != "" {
		if _, parseErr := uuid.Parse(cookie.Value); parseErr == nil {
			if mergeErr := h.cartService.MergeGuestSession(ctx, customer.ID, cookie.Value); mergeErr != nil {
				slog.WarnContext(ctx, "failed to merge guest cart", "customer_id", customer.ID, "error", mergeErr)
			}
		}
		// Clear the guest_session cookie regardless of validity
		secure, sameSite, domain := middleware.AuthCookieSettings()
		//nolint:gosec // G124: Secure flag is environment-conditional, not omitted.
		http.SetCookie(w, &http.Cookie{
			Name:     "guest_session",
			Value:    "",
			Path:     "/",
			Domain:   domain,
			HttpOnly: true,
			Secure:   secure,
			SameSite: sameSite,
			MaxAge:   -1,
		})
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"customer":        customer,
		"is_new_customer": isNewCustomer,
	})
}

// RefreshToken handles customer token refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Read refresh token from cookie
	cookie, err := r.Cookie("store_refresh")
	if err != nil || cookie.Value == "" {
		response.Unauthorized(w, "Refresh token required")
		return
	}

	customer, tokens, err := h.customerAuthService.RefreshToken(ctx, cookie.Value)
	if err != nil {
		// Only drop the session when the credential itself is finished. A
		// throttled DynamoDB read or any other 5xx says nothing about the
		// token, and clearing cookies on those turns a transient blip into a
		// logout the customer has to recover from by signing in again.
		if isTerminalAuthError(err) {
			h.clearStoreCookies(w)
		}
		response.Error(w, err)
		return
	}

	h.setStoreCookies(w, tokens)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"customer":          customer,
		response.KeyMessage: "Token refreshed",
	})
}

// Logout handles customer logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	customerID := middleware.GetCustomerIDFromContext(ctx)
	if customerID == "" {
		response.Error(w, errors.Unauthorized("Customer not authenticated"))
		return
	}

	// Get refresh token from cookie to revoke it
	var refreshToken string
	if cookie, err := r.Cookie("store_refresh"); err == nil && cookie.Value != "" {
		refreshToken = cookie.Value
	}

	if refreshToken != "" {
		if err := h.customerAuthService.Logout(ctx, customerID, refreshToken); err != nil {
			response.Error(w, err)
			return
		}
	}

	h.clearStoreCookies(w)

	response.JSON(w, http.StatusOK, map[string]string{
		response.KeyMessage: "Logged out successfully",
	})
}
