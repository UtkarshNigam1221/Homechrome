package phonepe

// Config holds PhonePe configuration
type Config struct {
	MerchantID  string
	SaltKey     string
	SaltIndex   string
	BaseURL     string
	CallbackURL string
	RedirectURL string
}

// PayRequest is the payload for initiating a payment
type PayRequest struct {
	MerchantID            string `json:"merchantId"`
	MerchantTransactionID string `json:"merchantTransactionId"`
	MerchantUserID        string `json:"merchantUserId"`
	Amount                int64  `json:"amount"` // in paise
	CallbackURL           string `json:"callbackUrl"`
	RedirectURL           string `json:"redirectUrl"`
	RedirectMode          string `json:"redirectMode"`
	PaymentInstrument     struct {
		Type string `json:"type"`
	} `json:"paymentInstrument"`
}

// PayResponse is the response from PhonePe pay API
type PayResponse struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		MerchantID            string `json:"merchantId"`
		MerchantTransactionID string `json:"merchantTransactionId"`
		InstrumentResponse    struct {
			Type         string `json:"type"`
			RedirectInfo struct {
				URL    string `json:"url"`
				Method string `json:"method"`
			} `json:"redirectInfo"`
		} `json:"instrumentResponse"`
	} `json:"data"`
}

// StatusResponse is the response from PhonePe status check API
type StatusResponse struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		MerchantID            string `json:"merchantId"`
		MerchantTransactionID string `json:"merchantTransactionId"`
		TransactionID         string `json:"transactionId"`
		Amount                int64  `json:"amount"`
		State                 string `json:"state"` // COMPLETED, PENDING, FAILED
		ResponseCode          string `json:"responseCode"`
		PaymentInstrument     struct {
			Type string `json:"type"`
			UTR  string `json:"utr,omitempty"`
		} `json:"paymentInstrument"`
	} `json:"data"`
}

// WebhookPayload is the callback payload from PhonePe
type WebhookPayload struct {
	Response string `json:"response"` // base64 encoded
}
