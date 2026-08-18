package phonepe

// Config holds PhonePe Standard Checkout configuration
type Config struct {
	ClientID      string
	ClientSecret  string
	ClientVersion string
	BaseURL       string
	CallbackURL   string
	RedirectURL   string
}

// --- OAuth Token ---

// TokenResponse is the response from the PhonePe OAuth token endpoint
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
	TokenType   string `json:"token_type"` // "O-Bearer"
}

// --- Create Payment (Standard Checkout v2) ---

// PayRequest is the payload for creating a payment order
type PayRequest struct {
	MerchantOrderID string      `json:"merchantOrderId"`
	Amount          int64       `json:"amount"` // in paise
	PaymentFlow     PaymentFlow `json:"paymentFlow"`
}

// PaymentFlow specifies the checkout type and redirect URL
type PaymentFlow struct {
	Type         string       `json:"type"` // "PG_CHECKOUT"
	MerchantURLs MerchantURLs `json:"merchantUrls"`
}

// MerchantURLs holds the redirect URL for post-payment
type MerchantURLs struct {
	RedirectURL string `json:"redirectUrl"`
}

// PayResponse is the response from the create payment API
type PayResponse struct {
	OrderID     string `json:"orderId"`
	State       string `json:"state"` // PENDING
	ExpireAt    int64  `json:"expireAt"`
	RedirectURL string `json:"redirectUrl"`
}

// PayErrorResponse is returned on API errors
type PayErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// --- Order Status ---

// StatusResponse is the response from the order status API
type StatusResponse struct {
	OrderID        string          `json:"orderId"`
	State          string          `json:"state"` // COMPLETED, PENDING, FAILED
	Amount         int64           `json:"amount"`
	PayableAmount  int64           `json:"payableAmount"`
	FeeAmount      int64           `json:"feeAmount"`
	ExpireAt       int64           `json:"expireAt"`
	PaymentDetails []PaymentDetail `json:"paymentDetails"`
	ErrorCode      string          `json:"errorCode,omitempty"`
}

// PaymentDetail holds details of a payment attempt
type PaymentDetail struct {
	TransactionID string `json:"transactionId"`
	PaymentMode   string `json:"paymentMode"` // UPI_INTENT, UPI_COLLECT, UPI_QR, CARD, NET_BANKING
	Timestamp     int64  `json:"timestamp"`
	Amount        int64  `json:"amount"`
	State         string `json:"state"` // COMPLETED, PENDING, FAILED
	ErrorCode     string `json:"errorCode,omitempty"`
}

// --- Webhook ---

// WebhookPayload is the callback payload from PhonePe Standard Checkout
type WebhookPayload struct {
	Event   string       `json:"event"` // checkout.order.completed, checkout.order.failed
	Payload WebhookOrder `json:"payload"`
}

// WebhookOrder holds the order details in a webhook callback
type WebhookOrder struct {
	OrderID         string          `json:"orderId"`
	MerchantID      string          `json:"merchantId"`
	MerchantOrderID string          `json:"merchantOrderId"`
	State           string          `json:"state"` // COMPLETED, FAILED
	Amount          int64           `json:"amount"`
	ExpireAt        int64           `json:"expireAt"`
	PaymentDetails  []PaymentDetail `json:"paymentDetails"`
}

// RefundResponse is what PhonePe returns when a refund is accepted for
// processing. State is PENDING: there is no "accepted" event, a refund goes
// PENDING then settles, so RefundID is the only handle on it until it does.
type RefundResponse struct {
	RefundID string `json:"refundId"`
	Amount   int64  `json:"amount"`
	State    string `json:"state"`
}

// RefundStatusResponse is the provider's current view of a refund, used when a
// webhook never arrived or its initiation response was lost.
type RefundStatusResponse struct {
	OriginalMerchantOrderID string `json:"originalMerchantOrderId"`
	RefundID                string `json:"refundId"`
	Amount                  int64  `json:"amount"`
	State                   string `json:"state"`
	ErrorCode               string `json:"errorCode,omitempty"`
	DetailedErrorCode       string `json:"detailedErrorCode,omitempty"`
}

// Refund states PhonePe reports.
const (
	RefundStatePending   = "PENDING"
	RefundStateCompleted = "COMPLETED"
	RefundStateConfirmed = "CONFIRMED"
	RefundStateFailed    = "FAILED"
)
