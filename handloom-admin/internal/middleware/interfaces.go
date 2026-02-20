package middleware

import (
	"context"
	"net/http"
)

// Validator defines the interface for request validation.
// Following ISP - single method interface for validation.
type Validator interface {
	// Validate validates a struct and returns validation errors
	Validate(ctx context.Context, v interface{}) error
}

// RequestDecoder defines how to decode HTTP requests.
// Follows SRP - only decodes, doesn't validate.
type RequestDecoder interface {
	// Decode decodes the request body into the target struct
	Decode(r *http.Request, target interface{}) error
}

// MiddlewareFunc is a function that wraps an http.Handler
type MiddlewareFunc func(http.Handler) http.Handler

// ValidationMiddlewareInterface creates middleware that validates requests.
// Following DIP - depends on abstractions.
type ValidationMiddlewareInterface interface {
	// ValidateJSON creates middleware that validates JSON request body
	ValidateJSON(factory func() interface{}) MiddlewareFunc

	// ValidateQuery creates middleware that validates query parameters
	ValidateQuery(target interface{}) MiddlewareFunc
}

// ContextKey type for context values to prevent collisions
type ContextKey string

const (
	// RequestBodyKey stores the decoded and validated request body
	RequestBodyKey ContextKey = "validated_request_body"

	// ValidationErrorsKey stores any validation errors
	ValidationErrorsKey ContextKey = "validation_errors"

	// RequestIDKey stores the request ID
	RequestIDKey ContextKey = "request_id"

	// UserIDKey stores the authenticated user ID
	UserIDKey ContextKey = "user_id"

	// UserKey stores the authenticated user object
	UserKey ContextKey = "user"

	// PaginationKey stores parsed pagination parameters
	PaginationKey ContextKey = "pagination"

	// CustomerIDKey stores the authenticated customer ID
	CustomerIDKey ContextKey = "customer_id"

	// CustomerKey stores the authenticated customer object
	CustomerKey ContextKey = "customer"
)

// ValidationError represents a validation error with field details
type ValidationError struct {
	Field   string      `json:"field"`
	Tag     string      `json:"tag"`
	Value   interface{} `json:"value,omitempty"`
	Message string      `json:"message"`
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

// Error implements the error interface
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "validation failed"
	}
	return ve[0].Message
}

// HasErrors returns true if there are any validation errors
func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}
