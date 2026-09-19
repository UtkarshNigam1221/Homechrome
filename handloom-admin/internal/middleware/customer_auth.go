package middleware

import (
	"context"
	"errors"
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
		ctx, err := a.authenticate(r)
		if err != nil {
			response.Unauthorized(w, err.Error())
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalCustomer sets the customer context when a valid token is present
// and no-ops otherwise, so a public route still serves anonymous visitors.
func (m *CustomerAuth) OptionalCustomer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ctx, err := m.authenticate(r); err == nil {
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// authenticate parses and validates the store_token. Shared by Authenticate
// (required) and OptionalCustomer (best-effort) so parsing lives in one place.
func (a *CustomerAuth) authenticate(r *http.Request) (context.Context, error) {
	token, err := extractBearerToken(r, "store_token")
	if err != nil {
		return nil, err
	}

	claims, err := a.customerAuthService.ValidateCustomerToken(r.Context(), token)
	if err != nil {
		return nil, errors.New("invalid or expired token")
	}

	if claims.CustomerID == "" {
		return nil, errors.New("invalid token claims")
	}

	return setCustomerContext(r.Context(), claims), nil
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
