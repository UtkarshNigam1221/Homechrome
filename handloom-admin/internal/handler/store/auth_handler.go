// Package store implements HTTP handlers for the B2C storefront
package store

import (
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/response"
)

// AuthHandler handles customer authentication requests
type AuthHandler struct {
	customerAuthService domain.CustomerAuthService
	validation          *middleware.Validation
}

// NewAuthHandler creates a new store AuthHandler
func NewAuthHandler(customerAuthService domain.CustomerAuthService, validation *middleware.Validation) *AuthHandler {
	return &AuthHandler{
		customerAuthService: customerAuthService,
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

// cookieSettings returns Secure, SameSite, and Domain values for store auth cookies.
//   - COOKIE_DOMAIN set (custom domain, same-site): Secure + Lax + Domain
//   - Lambda without custom domain (cross-origin): Secure + None (third-party cookies)
//   - Local dev: insecure + Lax (Vite proxy, same-origin)
func cookieSettings() (secure bool, sameSite http.SameSite, domain string) {
	if d := os.Getenv("COOKIE_DOMAIN"); d != "" {
		return true, http.SameSiteLaxMode, d
	}
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		return true, http.SameSiteNoneMode, ""
	}
	return false, http.SameSiteLaxMode, ""
}

func (h *AuthHandler) setStoreCookies(w http.ResponseWriter, tokens *domain.TokenPair) {
	secure, sameSite, domain := cookieSettings()

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

	http.SetCookie(w, &http.Cookie{
		Name:     "store_refresh",
		Value:    tokens.RefreshToken,
		Path:     "/api/v1/store/auth",
		Domain:   domain,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   int(7 * 24 * time.Hour / time.Second),
	})
}

func (h *AuthHandler) clearStoreCookies(w http.ResponseWriter) {
	secure, sameSite, domain := cookieSettings()

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

	http.SetCookie(w, &http.Cookie{
		Name:     "store_refresh",
		Value:    "",
		Path:     "/api/v1/store/auth",
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
		"message": "OTP sent successfully",
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
		h.clearStoreCookies(w)
		response.Error(w, err)
		return
	}

	h.setStoreCookies(w, tokens)

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"customer": customer,
		"message":  "Token refreshed",
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
		"message": "Logged out successfully",
	})
}
