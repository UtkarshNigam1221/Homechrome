package validator

import (
	"context"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/handloom/admin/pkg/errors"
)

// Service implements the Validator interface.
// Strategy pattern - can be swapped with different validation implementations.
type Service struct {
	validate *validator.Validate
}

// New creates a new validator service.
func New() *Service {
	v := validator.New()

	// Register custom tag name function to use JSON field names in errors
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return fld.Name
		}
		if name == "" {
			return fld.Name
		}
		return name
	})

	s := &Service{validate: v}
	s.registerCustomValidations()

	return s
}

// Validate validates a struct using validation tags.
// Implements the middleware.Validator interface.
func (s *Service) Validate(ctx context.Context, v interface{}) error {
	err := s.validate.StructCtx(ctx, v)
	if err == nil {
		return nil
	}

	// Convert validation errors to AppError
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return errors.Validation(err.Error())
	}

	return s.formatErrors(validationErrors)
}

// ValidateVar validates a single variable against a tag.
func (s *Service) ValidateVar(v interface{}, tag string) error {
	return s.validate.Var(v, tag)
}

// formatErrors converts validator errors to structured error response.
func (s *Service) formatErrors(errs validator.ValidationErrors) *errors.AppError {
	details := make([]ErrorDetail, 0, len(errs))
	messages := make([]string, 0, len(errs))

	for _, e := range errs {
		detail := ErrorDetail{
			Field:   e.Field(),
			Tag:     e.Tag(),
			Value:   e.Value(),
			Message: s.errorMessage(e),
		}
		details = append(details, detail)
		messages = append(messages, detail.Message)
	}

	return errors.ValidationWithDetails(
		strings.Join(messages, "; "),
		map[string]interface{}{"errors": details},
	)
}

// ErrorDetail holds details about a single validation error.
type ErrorDetail struct {
	Field   string      `json:"field"`
	Tag     string      `json:"tag"`
	Value   interface{} `json:"value,omitempty"`
	Message string      `json:"message"`
}

// errorMessage generates a human-readable error message.
func (s *Service) errorMessage(e validator.FieldError) string {
	field := e.Field()

	switch e.Tag() {
	case "required":
		return field + " is required"
	case "email":
		return field + " must be a valid email address"
	case "min":
		if e.Kind() == reflect.String {
			return field + " must be at least " + e.Param() + " characters"
		}
		return field + " must be at least " + e.Param()
	case "max":
		if e.Kind() == reflect.String {
			return field + " must be at most " + e.Param() + " characters"
		}
		return field + " must be at most " + e.Param()
	case "gt":
		return field + " must be greater than " + e.Param()
	case "gte":
		return field + " must be greater than or equal to " + e.Param()
	case "lt":
		return field + " must be less than " + e.Param()
	case "lte":
		return field + " must be less than or equal to " + e.Param()
	case "oneof":
		return field + " must be one of: " + e.Param()
	case "len":
		return field + " must have exactly " + e.Param() + " characters"
	case "uuid":
		return field + " must be a valid UUID"
	case "url":
		return field + " must be a valid URL"
	case "alphanum":
		return field + " must contain only alphanumeric characters"
	case "numeric":
		return field + " must be a number"
	case "positive_price":
		return field + " must be a positive price"
	case "valid_order_status":
		return field + " must be a valid order status"
	case "valid_user_role":
		return field + " must be a valid user role"
	case "valid_user_status":
		return field + " must be a valid user status"
	case "coupon_value":
		return field + " must not exceed 100% on a percentage coupon"
	case "dive":
		return field + " contains invalid items"
	default:
		return field + " failed validation: " + e.Tag()
	}
}

// registerCustomValidations registers custom validation rules.
func (s *Service) registerCustomValidations() {
	// Register custom validation for order status
	_ = s.validate.RegisterValidation("valid_order_status", validateOrderStatus)

	// Register custom validation for user roles
	_ = s.validate.RegisterValidation("valid_user_role", validateUserRole)

	// Register custom validation for user status
	_ = s.validate.RegisterValidation("valid_user_status", validateUserStatus)

	// Register custom validation for positive prices
	_ = s.validate.RegisterValidation("positive_price", validatePositivePrice)

	// Register custom validation for coupon type
	_ = s.validate.RegisterValidation("valid_coupon_type", validateCouponType)

	// Register the percentage ceiling on a coupon's value
	_ = s.validate.RegisterValidation("coupon_value", validateCouponValue)

	// Register custom validation for entity type
	_ = s.validate.RegisterValidation("valid_entity_type", validateEntityType)
}

// GetValidator returns the underlying validator instance.
// Use this for advanced use cases.
func (s *Service) GetValidator() *validator.Validate {
	return s.validate
}
