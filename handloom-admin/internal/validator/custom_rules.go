package validator

import (
	"reflect"

	"github.com/go-playground/validator/v10"
)

// Status string literals used across multiple validators.
const statusPending = "PENDING"

// Order status constants matching domain.OrderStatus
var validOrderStatuses = map[string]bool{
	statusPending: true,
	"CONFIRMED":   true,
	"PROCESSING":  true,
	"SHIPPED":     true,
	"DELIVERED":   true,
	"CANCELED":    true,
	"RETURNED":    true,
	"REFUNDED":    true,
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
	"ACTIVE":      true,
	"INACTIVE":    true,
	statusPending: true,
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

// couponPercentageMax is 100.00% in the units Coupon.Value stores a percentage in:
// percentage x 100.
const couponPercentageMax = 10000

// validateCouponValue caps a percentage coupon at 100%. Above that the discount exceeds
// the cart, which used to zero the payable total and then fail the payment outright.
// A fixed-amount coupon's value is paise and has no such ceiling, so this has to read
// the sibling Type field — go-playground has no "lte only when another field says so".
func validateCouponValue(fl validator.FieldLevel) bool {
	parent := fl.Parent()
	for parent.Kind() == reflect.Pointer {
		parent = parent.Elem()
	}
	// Fails OPEN on an unreadable Type, which is only safe because
	// computeCouponDiscount's non-PERCENTAGE branch prices such a coupon as FIXED too.
	typeField := parent.FieldByName("Type")
	if !typeField.IsValid() || typeField.Kind() != reflect.String {
		return true
	}
	if typeField.String() != "PERCENTAGE" {
		return true
	}

	switch fl.Field().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fl.Field().Int() <= couponPercentageMax
	default:
		return false // fail closed rather than wave through a type we cannot read
	}
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
	statusPending: true,
	"PAID":        true,
	"FAILED":      true,
	"REFUNDED":    true,
	"CANCELED":    true,
}

// ValidatePaymentStatus validates payment status.
func ValidatePaymentStatus(status string) bool {
	return validPaymentStatuses[status]
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
