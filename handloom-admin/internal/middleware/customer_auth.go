package middleware

import (
	"context"
	"net/http"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/response"
	"github.com/handloom/admin/pkg/slogx"
)

// CustomerAuth provides customer JWT authentication middleware
type CustomerAuth struct {
	customerAuthService domain.CustomerAuthService
}

// NewCustomerAuth creates a new CustomerAuth middleware
func NewCustomerAuth(
	customerAuthService domain.CustomerAuthService,
) *CustomerAuth {
	return &CustomerAuth{
		customerAuthService: customerAuthService,
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

		next.ServeHTTP(w, r.WithContext(setCustomerContext(r.Context(), claims)))
	})
}

// setCustomerContext sets the customer ID, slog correlation ID, and a minimal
// customer object on the context from validated token claims. Shared by
// CustomerAuth.Authenticate and OptionalCartAuth.Resolve.
func setCustomerContext(ctx context.Context, claims *domain.CustomerTokenClaims) context.Context {
	ctx = context.WithValue(ctx, CustomerIDKey, claims.CustomerID)
	ctx = slogx.SetUserID(ctx, claims.CustomerID)
	return context.WithValue(ctx, CustomerKey, &domain.Customer{
		ID:    claims.CustomerID,
		Phone: claims.Phone,
		Email: claims.Email,
	})
}

// GetCustomerIDFromContext retrieves the customer ID from context
func GetCustomerIDFromContext(ctx context.Context) string {
	if customerID, ok := ctx.Value(CustomerIDKey).(string); ok {
		return customerID
	}
	return ""
}
