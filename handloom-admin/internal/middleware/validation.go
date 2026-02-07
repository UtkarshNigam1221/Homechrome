package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/response"
)

// ValidationConfig holds configuration for validation middleware
type ValidationConfig struct {
	// StopOnFirstError stops validation on first error
	StopOnFirstError bool
	// CustomMessages allows custom error messages per field/tag
	CustomMessages map[string]string
}

// Validation provides validation middleware.
// Implements Chain of Responsibility pattern.
type Validation struct {
	validator Validator
	config    ValidationConfig
}

// NewValidation creates a new Validation middleware.
// Factory pattern for creating middleware with dependencies.
func NewValidation(v Validator, cfg ValidationConfig) *Validation {
	return &Validation{
		validator: v,
		config:    cfg,
	}
}

// ValidateJSON creates middleware that validates JSON request body.
// Decorator pattern - wraps handler with validation.
//
// Usage:
//
//	r.With(validation.ValidateJSON(func() interface{} {
//	    return new(dto.LoginRequest)
//	})).Post("/login", h.Login)
func (v *Validation) ValidateJSON(factory func() interface{}) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create new instance of the target type
			target := factory()

			// Decode JSON body
			if err := json.NewDecoder(r.Body).Decode(target); err != nil {
				response.Error(w, errors.Validation("Invalid JSON: "+err.Error()))
				return
			}

			// Validate the decoded struct
			if err := v.validator.Validate(r.Context(), target); err != nil {
				response.Error(w, err)
				return
			}

			// Store validated request in context
			ctx := context.WithValue(r.Context(), RequestBodyKey, target)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ValidateJSONTyped is a type-safe version using generics.
// Creates a validation middleware for a specific request type.
//
// Usage:
//
//	r.With(middleware.ValidateJSONTyped[dto.LoginRequest](validation)).Post("/login", h.Login)
func ValidateJSONTyped[T any](v *Validation) MiddlewareFunc {
	return v.ValidateJSON(func() interface{} {
		return new(T)
	})
}

// RequireValidatedBody ensures a validated body exists in context.
// Chain of Responsibility - passes through if validation exists.
func RequireValidatedBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(RequestBodyKey) == nil {
			response.Error(w, errors.BadRequest("Request body not validated"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetValidatedBody retrieves the validated request body from context.
// Generic function for type-safe retrieval.
//
// Usage:
//
//	req, ok := middleware.GetValidatedBody[dto.LoginRequest](r.Context())
//	if !ok {
//	    // Handle error
//	}
func GetValidatedBody[T any](ctx context.Context) (*T, bool) {
	v := ctx.Value(RequestBodyKey)
	if v == nil {
		return nil, false
	}
	typed, ok := v.(*T)
	return typed, ok
}

// MustGetValidatedBody retrieves the validated request body or panics.
// Use only when you're certain validation middleware ran before handler.
func MustGetValidatedBody[T any](ctx context.Context) *T {
	req, ok := GetValidatedBody[T](ctx)
	if !ok {
		panic("validated body not found in context - ensure validation middleware is applied")
	}
	return req
}

// OptionalValidateJSON creates middleware that validates JSON request body if present.
// If the body is empty, it passes through without error.
func (v *Validation) OptionalValidateJSON(factory func() interface{}) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if body is empty
			if r.ContentLength == 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Create new instance of the target type
			target := factory()

			// Decode JSON body
			if err := json.NewDecoder(r.Body).Decode(target); err != nil {
				response.Error(w, errors.Validation("Invalid JSON: "+err.Error()))
				return
			}

			// Validate the decoded struct
			if err := v.validator.Validate(r.Context(), target); err != nil {
				response.Error(w, err)
				return
			}

			// Store validated request in context
			ctx := context.WithValue(r.Context(), RequestBodyKey, target)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ValidateQuery creates middleware that validates query parameters.
// The target struct should use `schema` tags for query parameter binding.
func (v *Validation) ValidateQuery(factory func() interface{}) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			target := factory()

			// Parse query parameters into struct
			// Note: This requires the target to implement a method or use reflection
			// For now, we'll skip binding and just validate
			// In production, use github.com/gorilla/schema or similar

			// Validate the struct
			if err := v.validator.Validate(r.Context(), target); err != nil {
				response.Error(w, err)
				return
			}

			ctx := context.WithValue(r.Context(), RequestBodyKey, target)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
