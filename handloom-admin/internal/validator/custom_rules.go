package validator

import (
	"reflect"

	"github.com/go-playground/validator/v10"
)

// Order status constants matching domain.OrderStatus
var validOrderStatuses = map[string]bool{
	"PENDING":    true,
	"CONFIRMED":  true,
	"PROCESSING": true,
	"SHIPPED":    true,
	"DELIVERED":  true,
	"CANCELED":   true,
	"RETURNED":   true,
	"REFUNDED":   true,
}

// validateOrderStatus validates that the field contains a valid order status.
func validateOrderStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	return validOrderStatuses[status]
}

// User role constants matching domain.UserRole
var validUserRoles = map[string]bool{
	"ADMIN":    true,
	"OPERATOR": true,
}

// validateUserRole validates that the field contains a valid user role.
func validateUserRole(fl validator.FieldLevel) bool {
	role := fl.Field().String()
	return validUserRoles[role]
}

// User status constants matching domain.UserStatus
var validUserStatuses = map[string]bool{
	"ACTIVE":   true,
	"INACTIVE": true,
	"PENDING":  true,
}

// validateUserStatus validates that the field contains a valid user status.
func validateUserStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	return validUserStatuses[status]
}

// validatePositivePrice validates that a price is positive.
// Works with int, int64, float32, float64.
func validatePositivePrice(fl validator.FieldLevel) bool {
	field := fl.Field()

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() > 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return field.Uint() > 0
	case reflect.Float32, reflect.Float64:
		return field.Float() > 0
	default:
		return false
	}
}

// Coupon type constants matching domain.CouponType
var validCouponTypes = map[string]bool{
	"PERCENTAGE": true,
	"FIXED":      true,
}

// validateCouponType validates that the field contains a valid coupon type.
func validateCouponType(fl validator.FieldLevel) bool {
	couponType := fl.Field().String()
	return validCouponTypes[couponType]
}

// Bulk operation type constants
var validBulkOperationTypes = map[string]bool{
	"IMPORT": true,
	"EXPORT": true,
}

// validateBulkOperationType validates that the field contains a valid bulk operation type.
func validateBulkOperationType(fl validator.FieldLevel) bool {
	opType := fl.Field().String()
	return validBulkOperationTypes[opType]
}

// Entity type constants
var validEntityTypes = map[string]bool{
	"PRODUCT":   true,
	"ORDER":     true,
	"CUSTOMER":  true,
	"INVENTORY": true,
	"PRICING":   true,
}

// validateEntityType validates that the field contains a valid entity type.
func validateEntityType(fl validator.FieldLevel) bool {
	entityType := fl.Field().String()
	return validEntityTypes[entityType]
}

// Payment status constants
var validPaymentStatuses = map[string]bool{
	"PENDING":  true,
	"PAID":     true,
	"FAILED":   true,
	"REFUNDED": true,
	"CANCELED": true,
}

// ValidatePaymentStatus validates payment status.
func ValidatePaymentStatus(status string) bool {
	return validPaymentStatuses[status]
}

// Artisan status constants
var validArtisanStatuses = map[string]bool{
	"ACTIVE":   true,
	"INACTIVE": true,
	"PENDING":  true,
}

// ValidateArtisanStatus validates artisan status.
func ValidateArtisanStatus(status string) bool {
	return validArtisanStatuses[status]
}

// Product status constants
var validProductStatuses = map[string]bool{
	"ACTIVE":   true,
	"INACTIVE": true,
	"DRAFT":    true,
	"ARCHIVED": true,
}

// ValidateProductStatus validates product status.
func ValidateProductStatus(status string) bool {
	return validProductStatuses[status]
}
