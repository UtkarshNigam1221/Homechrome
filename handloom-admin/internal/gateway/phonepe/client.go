package phonepe

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client implements the PhonePe payment gateway
type Client struct {
	config     Config
	httpClient *http.Client
}

// NewClient creates a new PhonePe client
func NewClient(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://api-preprod.phonepe.com/apis/pg-sandbox"
	}
	if config.SaltIndex == "" {
		config.SaltIndex = "1"
	}
	return &Client{
		config:     config,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// InitiatePayment initiates a payment and returns the redirect URL
func (c *Client) InitiatePayment(ctx context.Context, merchantTxnID, customerID string, amount int64) (string, error) {
	payReq := PayRequest{
		MerchantID:            c.config.MerchantID,
		MerchantTransactionID: merchantTxnID,
		MerchantUserID:        customerID,
		Amount:                amount,
		CallbackURL:           c.config.CallbackURL,
		RedirectURL:           c.config.RedirectURL,
		RedirectMode:          "REDIRECT",
	}
	payReq.PaymentInstrument.Type = "PAY_PAGE"

	payloadBytes, err := json.Marshal(payReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payment request: %w", err)
	}

	base64Payload := base64.StdEncoding.EncodeToString(payloadBytes)
	xVerify := c.generateChecksum(base64Payload, "/pg/v1/pay")

	reqBody, _ := json.Marshal(map[string]string{"request": base64Payload})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/pg/v1/pay", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-VERIFY", xVerify)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call PhonePe: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var payResp PayResponse
	if err := json.Unmarshal(body, &payResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if !payResp.Success {
		return "", fmt.Errorf("PhonePe payment initiation failed: %s - %s", payResp.Code, payResp.Message)
	}

	return payResp.Data.InstrumentResponse.RedirectInfo.URL, nil
}

// CheckPaymentStatus checks the status of a payment
func (c *Client) CheckPaymentStatus(ctx context.Context, merchantTxnID string) (*StatusResponse, error) {
	endpoint := fmt.Sprintf("/pg/v1/status/%s/%s", c.config.MerchantID, merchantTxnID)
	xVerify := c.generateChecksum("", endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.config.BaseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-VERIFY", xVerify)
	req.Header.Set("X-MERCHANT-ID", c.config.MerchantID)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call PhonePe: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var statusResp StatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &statusResp, nil
}

// VerifyWebhookSignature verifies the X-VERIFY header from a PhonePe callback
func (c *Client) VerifyWebhookSignature(responseBase64, xVerifyHeader string) bool {
	checksum := sha256Sum(responseBase64+c.config.SaltKey) + "###" + c.config.SaltIndex
	return checksum == xVerifyHeader
}

// DecodeWebhookResponse decodes the base64-encoded webhook response
func (c *Client) DecodeWebhookResponse(responseBase64 string) (*StatusResponse, error) {
	decoded, err := base64.StdEncoding.DecodeString(responseBase64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode webhook response: %w", err)
	}

	var statusResp StatusResponse
	if err := json.Unmarshal(decoded, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to parse webhook response: %w", err)
	}

	return &statusResp, nil
}

// generateChecksum generates the X-VERIFY checksum for PhonePe API
func (c *Client) generateChecksum(base64Payload, endpoint string) string {
	data := base64Payload + endpoint + c.config.SaltKey
	return sha256Sum(data) + "###" + c.config.SaltIndex
}

func sha256Sum(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return fmt.Sprintf("%x", h.Sum(nil))
}
