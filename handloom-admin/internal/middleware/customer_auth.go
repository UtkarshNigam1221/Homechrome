package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/service"
	"github.com/handloom/admin/pkg/logger"
	"github.com/handloom/admin/pkg/response"
)

// CustomerAuth provides customer JWT authentication middleware
type CustomerAuth struct {
	customerAuthService *service.CustomerAuthService
	customerRepo        domain.CustomerRepository
	logger              *logger.Logger
}

// NewCustomerAuth creates a new CustomerAuth middleware
func NewCustomerAuth(
	customerAuthService *service.CustomerAuthService,
	customerRepo domain.CustomerRepository,
	logger *logger.Logger,
) *CustomerAuth {
	return &CustomerAuth{
		customerAuthService: customerAuthService,
		customerRepo:        customerRepo,
		logger:              logger,
	}
}

// Authenticate validates customer JWT token and sets customer in context
func (a *CustomerAuth) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try store_token cookie first, then fall back to Authorization header
		var token string
		if cookie, err := r.Cookie("store_token"); err == nil && cookie.Value != "" {
			token = cookie.Value
		} else {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Unauthorized(w, "Authentication required")
				return
			}
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				response.Unauthorized(w, "Invalid authorization header format")
				return
			}
			token = parts[1]
		}

		// Validate customer token
		claims, err := a.customerAuthService.ValidateCustomerToken(r.Context(), token)
		if err != nil {
			response.Unauthorized(w, "Invalid or expired token")
			return
		}

		customerID, _ := claims["sub"].(string)
		if customerID == "" {
			response.Unauthorized(w, "Invalid token claims")
			return
		}

		// Set customer ID in context
		ctx := context.WithValue(r.Context(), CustomerIDKey, customerID)
		ctx = logger.SetUserID(ctx, customerID)

		// Create a minimal customer object from claims
		phone, _ := claims["phone"].(string)
		email, _ := claims["email"].(string)
		customer := &domain.Customer{
			ID:    customerID,
			Phone: phone,
			Email: email,
		}
		ctx = context.WithValue(ctx, CustomerKey, customer)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetCustomerIDFromContext retrieves the customer ID from context
func GetCustomerIDFromContext(ctx context.Context) string {
	if customerID, ok := ctx.Value(CustomerIDKey).(string); ok {
		return customerID
	}
	return ""
}

// GetCustomerFromContext retrieves the customer from context
func GetCustomerFromContext(ctx context.Context) *domain.Customer {
	if customer, ok := ctx.Value(CustomerKey).(*domain.Customer); ok {
		return customer
	}
	return nil
}
