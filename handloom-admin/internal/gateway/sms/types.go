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

// otpFlowRequest is the MSG91 Flow API request body for OTP send.
// Maps to POST /api/v5/flow/.
type otpFlowRequest struct {
	TemplateID string            `json:"template_id"`
	ShortURL   string            `json:"short_url"`
	Recipients []otpFlowRecipient `json:"recipients"`
}

// otpFlowRecipient holds the per-mobile template variables.
// var1 = OTP code, var2 = validity minutes (template-defined).
type otpFlowRecipient struct {
	Mobiles string `json:"mobiles"`
	Var1    string `json:"var1"`
	Var2    string `json:"var2"`
}

// otpFlowResponse is MSG91's response envelope for OTP send.
type otpFlowResponse struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
