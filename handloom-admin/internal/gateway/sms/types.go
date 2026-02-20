package sms

// Config holds MSG91 configuration
type Config struct {
	BaseURL       string
	AuthKey       string
	OTPTemplateID string
}

// SMSGateway defines the interface for sending SMS
type SMSGateway interface {
	SendOTP(phone, code string) error
}
