package middleware

import (
	"context"
	"net/http"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/logger"
	"github.com/handloom/admin/pkg/response"
)

// CustomerAuth provides customer JWT authentication middleware
type CustomerAuth struct {
	customerAuthService domain.CustomerAuthService
	logger              *logger.Logger
}

// NewCustomerAuth creates a new CustomerAuth middleware
func NewCustomerAuth(
	customerAuthService domain.CustomerAuthService,
	logger *logger.Logger,
) *CustomerAuth {
	return &CustomerAuth{
		customerAuthService: customerAuthService,
		logger:              logger,
	}
}

// Authenticate validates customer JWT token and sets customer in context
func (a *CustomerAuth) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := extractBearerToken(r, "store_token")
		if err != nil {
			response.Unauthorized(w, err.Error())
			return
		}

		// Validate customer token
		claims, err := a.customerAuthService.ValidateCustomerToken(r.Context(), token)
		if err != nil {
			response.Unauthorized(w, "Invalid or expired token")
			return
		}

		if claims.CustomerID == "" {
			response.Unauthorized(w, "Invalid token claims")
			return
		}

		// Set customer ID in context
		ctx := context.WithValue(r.Context(), CustomerIDKey, claims.CustomerID)
		ctx = logger.SetUserID(ctx, claims.CustomerID)

		// Create a minimal customer object from claims
		customer := &domain.Customer{
			ID:    claims.CustomerID,
			Phone: claims.Phone,
			Email: claims.Email,
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
