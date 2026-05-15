// Package errors provides custom error types for the application
package errors

import (
	"fmt"
	"net/http"
)

// ErrorCode represents an application error code
type ErrorCode string

const (
	// General errors
	ErrCodeInternal     ErrorCode = "INTERNAL_ERROR"
	ErrCodeValidation   ErrorCode = "VALIDATION_ERROR"
	ErrCodeNotFound     ErrorCode = "NOT_FOUND"
	ErrCodeConflict     ErrorCode = "CONFLICT"
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden    ErrorCode = "FORBIDDEN"
	ErrCodeBadRequest   ErrorCode = "BAD_REQUEST"

	// Auth errors
	ErrCodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	ErrCodeTokenExpired       ErrorCode = "TOKEN_EXPIRED"
	ErrCodeTokenInvalid       ErrorCode = "TOKEN_INVALID"

	// User errors
	ErrCodeUserExists      ErrorCode = "USER_EXISTS"
	ErrCodeUserNotFound    ErrorCode = "USER_NOT_FOUND"
	ErrCodeInvalidPassword ErrorCode = "INVALID_PASSWORD"
	ErrCodeUserInactive    ErrorCode = "USER_INACTIVE"
	ErrCodeInvalidToken    ErrorCode = "INVALID_TOKEN"
	ErrCodeAlreadyExists   ErrorCode = "ALREADY_EXISTS"
	ErrCodeHasDependencies ErrorCode = "HAS_DEPENDENCIES"

	// Category errors
	ErrCodeCategoryNotFound    ErrorCode = "CATEGORY_NOT_FOUND"
	ErrCodeCategoryHasChildren ErrorCode = "CATEGORY_HAS_CHILDREN"
	ErrCodeCategoryHasProducts ErrorCode = "CATEGORY_HAS_PRODUCTS"

	// Product errors
	ErrCodeProductNotFound  ErrorCode = "PRODUCT_NOT_FOUND"
	ErrCodeProductSKUExists ErrorCode = "PRODUCT_SKU_EXISTS"
	ErrCodeProductHasOrders ErrorCode = "PRODUCT_HAS_ORDERS"

	// Pricing errors
	ErrCodePricingRuleNotFound ErrorCode = "PRICING_RULE_NOT_FOUND"
	ErrCodeConflictingPriority ErrorCode = "CONFLICTING_PRIORITY"
	ErrCodeRuleIsDefault       ErrorCode = "RULE_IS_DEFAULT"
	ErrCodeDimensionOutOfRange ErrorCode = "DIMENSION_OUT_OF_RANGE"
	ErrCodeMinOrderValue       ErrorCode = "MIN_ORDER_VALUE"
	ErrCodeQuoteNotFound       ErrorCode = "QUOTE_NOT_FOUND"
	ErrCodeQuoteExpired        ErrorCode = "QUOTE_EXPIRED"

	// Inventory errors
	ErrCodeInsufficientStock ErrorCode = "INSUFFICIENT_STOCK"
	ErrCodeInventoryNotFound ErrorCode = "INVENTORY_NOT_FOUND"

	// Generic
	ErrCodeNotImplemented ErrorCode = "NOT_IMPLEMENTED"
)

// AppError represents an application error
type AppError struct {
	Code       ErrorCode   `json:"code"`
	Message    string      `json:"message"`
	Details    interface{} `json:"details,omitempty"`
	HTTPStatus int         `json:"-"`
	Err        error       `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// WithDetails adds details to the error
func (e *AppError) WithDetails(details interface{}) *AppError {
	e.Details = details
	return e
}

// WithError wraps an underlying error
func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	return e
}

// New creates a new AppError
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: codeToHTTPStatus(code),
	}
}

// Newf creates a new AppError with formatted message
func Newf(code ErrorCode, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       code,
		Message:    fmt.Sprintf(format, args...),
		HTTPStatus: codeToHTTPStatus(code),
	}
}

// Common errors
var (
	ErrInternal     = New(ErrCodeInternal, "An internal error occurred")
	ErrUnauthorized = New(ErrCodeUnauthorized, "Authentication required")
	ErrForbidden    = New(ErrCodeForbidden, "Access denied")
)

// NotFound creates a not found error for a resource
func NotFound(resource string) *AppError {
	return Newf(ErrCodeNotFound, "%s not found", resource)
}

// Conflict creates a conflict error
func Conflict(message string) *AppError {
	return New(ErrCodeConflict, message)
}

// Validation creates a validation error
func Validation(message string) *AppError {
	return New(ErrCodeValidation, message)
}

// ValidationWithDetails creates a validation error with details
func ValidationWithDetails(message string, details interface{}) *AppError {
	return New(ErrCodeValidation, message).WithDetails(details)
}

// BadRequest creates a bad request error
func BadRequest(message string) *AppError {
	return New(ErrCodeBadRequest, message)
}

// Unauthorized creates an unauthorized error
func Unauthorized(message string) *AppError {
	return New(ErrCodeUnauthorized, message)
}

// Forbidden creates a forbidden error
func Forbidden(message string) *AppError {
	return New(ErrCodeForbidden, message)
}

// NotImplemented creates a not implemented error
func NotImplemented(message string) *AppError {
	return New(ErrCodeNotImplemented, message)
}

// Internal creates an internal error wrapping another error or with a message
func Internal(v interface{}) *AppError {
	switch e := v.(type) {
	case error:
		return New(ErrCodeInternal, "An internal error occurred").WithError(e)
	case string:
		return New(ErrCodeInternal, e)
	default:
		return New(ErrCodeInternal, "An internal error occurred")
	}
}

// Wrap wraps an error with a message
func Wrap(err error, message string) *AppError {
	return New(ErrCodeInternal, message).WithError(err)
}

// IsNotFound checks if an error is a not found error
func IsNotFound(err error) bool {
	if appErr, ok := AsAppError(err); ok {
		return appErr.Code == ErrCodeNotFound ||
			appErr.Code == ErrCodeUserNotFound ||
			appErr.Code == ErrCodeCategoryNotFound ||
			appErr.Code == ErrCodeProductNotFound ||
			appErr.Code == ErrCodePricingRuleNotFound ||
			appErr.Code == ErrCodeQuoteNotFound ||
			appErr.Code == ErrCodeInventoryNotFound
	}
	return false
}

// codeToHTTPStatus maps error codes to HTTP status codes
func codeToHTTPStatus(code ErrorCode) int {
	switch code {
	case ErrCodeValidation, ErrCodeBadRequest:
		return http.StatusBadRequest
	case ErrCodeUnauthorized, ErrCodeInvalidCredentials, ErrCodeTokenExpired, ErrCodeTokenInvalid, ErrCodeInvalidToken, ErrCodeUserInactive:
		return http.StatusUnauthorized
	case ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeNotFound, ErrCodeUserNotFound, ErrCodeCategoryNotFound,
		ErrCodeProductNotFound, ErrCodePricingRuleNotFound, ErrCodeQuoteNotFound, ErrCodeInventoryNotFound:
		return http.StatusNotFound
	case ErrCodeConflict, ErrCodeUserExists, ErrCodeProductSKUExists, ErrCodeConflictingPriority, ErrCodeAlreadyExists:
		return http.StatusConflict
	case ErrCodeCategoryHasChildren, ErrCodeCategoryHasProducts,
		ErrCodeProductHasOrders, ErrCodeRuleIsDefault, ErrCodeDimensionOutOfRange,
		ErrCodeMinOrderValue, ErrCodeQuoteExpired, ErrCodeInsufficientStock, ErrCodeHasDependencies:
		return http.StatusBadRequest
	case ErrCodeNotImplemented:
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// AsAppError converts an error to AppError if possible
func AsAppError(err error) (*AppError, bool) {
	appErr, ok := err.(*AppError)
	return appErr, ok
}

// GetHTTPStatus returns the HTTP status for an error
func GetHTTPStatus(err error) int {
	if appErr, ok := AsAppError(err); ok {
		return appErr.HTTPStatus
	}
	return http.StatusInternalServerError
}
