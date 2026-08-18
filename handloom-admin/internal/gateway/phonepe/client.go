package phonepe

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	metricsmw "github.com/handloom/admin/pkg/metrics/middleware"
)

const (
	contentTypeJSON = "application/json"
	// StateCompleted is the PhonePe payment-state value indicating success.
	StateCompleted = "COMPLETED"
)

// Client implements the PhonePe Standard Checkout v2 gateway
type Client struct {
	config     Config
	httpClient *http.Client

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

// NewClient creates a new PhonePe Standard Checkout client
func NewClient(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://api-preprod.phonepe.com/apis/pg-sandbox"
	}
	return &Client{
		config:     config,
		httpClient: metricsmw.NewInstrumentedClient(30*time.Second, "phonepe"),
	}
}

// getToken returns a valid OAuth access token, refreshing if expired.
func (c *Client) getToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Return cached token if still valid (with 60s buffer)
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry.Add(-60*time.Second)) {
		return c.accessToken, nil
	}

	data := url.Values{
		"client_id":      {c.config.ClientID},
		"client_version": {c.config.ClientVersion},
		"client_secret":  {c.config.ClientSecret},
		"grant_type":     {"client_credentials"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/v1/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request auth token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("PhonePe auth failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("PhonePe returned empty access token")
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Unix(tokenResp.ExpiresAt, 0)
	return c.accessToken, nil
}

// InitiatePayment creates a payment order and returns the redirect URL
func (c *Client) InitiatePayment(ctx context.Context, merchantTxnID, _ string, amount int64, orderID string) (string, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return "", err
	}

	payReq := PayRequest{
		MerchantOrderID: merchantTxnID,
		Amount:          amount,
		PaymentFlow: PaymentFlow{
			Type: "PG_CHECKOUT",
			MerchantURLs: MerchantURLs{
				RedirectURL: c.config.RedirectURL + "?order_id=" + orderID,
			},
		},
	}

	reqBody, err := json.Marshal(payReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payment request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/checkout/v2/pay", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("Authorization", "O-Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call PhonePe: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp PayErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Code != "" {
			return "", fmt.Errorf("PhonePe payment initiation failed: %s - %s", errResp.Code, errResp.Message)
		}
		return "", fmt.Errorf("PhonePe payment initiation failed (status %d): %s", resp.StatusCode, string(body))
	}

	var payResp PayResponse
	if err := json.Unmarshal(body, &payResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return payResp.RedirectURL, nil
}

// CheckPaymentStatus checks the status of a payment order
func (c *Client) CheckPaymentStatus(ctx context.Context, merchantTxnID string) (*StatusResponse, error) {
	return getAuthedJSON[StatusResponse](ctx, c,
		fmt.Sprintf("/checkout/v2/order/%s/status?details=true", merchantTxnID),
		"PhonePe status check failed")
}

// VerifyWebhookSignature verifies the Authorization header from a PhonePe webhook.
// PhonePe sends: Authorization: SHA256(username:password)
// We compute the same and compare.
func (c *Client) VerifyWebhookSignature(username, password, authHeader string) bool {
	expected := fmt.Sprintf("%x", sha256.Sum256([]byte(username+":"+password)))
	return subtle.ConstantTimeCompare([]byte(expected), []byte(authHeader)) == 1
}

// InitiateRefund asks PhonePe to send money back for part or all of an order.
//
// originalMerchantOrderID is the Payment.MerchantTransactionID recorded at
// checkout — the same value passed as merchantOrderId when the payment was
// created. merchantRefundID is ours and unique per attempt; it is the key the
// status endpoint accepts, and the only handle we keep if this call's response
// is lost.
func (c *Client) InitiateRefund(ctx context.Context, merchantRefundID, originalMerchantOrderID string, amount int64) (*RefundResponse, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(map[string]any{
		"merchantRefundId":        merchantRefundID,
		"originalMerchantOrderId": originalMerchantOrderID,
		"amount":                  amount,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build refund request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.config.BaseURL+"/payments/v2/refund", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("Authorization", "O-Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call PhonePe: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("PhonePe refund failed (status %d): %s", resp.StatusCode, string(body))
	}

	var refundResp RefundResponse
	if err := json.Unmarshal(body, &refundResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &refundResp, nil
}

// CheckRefundStatus reads the provider's current view of a refund, keyed on our
// merchantRefundID rather than PhonePe's id — which is what makes it the
// recovery path when the initiation response never came back.
func (c *Client) CheckRefundStatus(ctx context.Context, merchantRefundID string) (*RefundStatusResponse, error) {
	return getAuthedJSON[RefundStatusResponse](ctx, c,
		fmt.Sprintf("/payments/v2/refund/%s/status", merchantRefundID),
		"PhonePe refund status check failed")
}

// getAuthedJSON performs an authenticated GET and decodes the body. The payment
// and refund status calls differ only in path and response type.
func getAuthedJSON[T any](ctx context.Context, c *Client, endpoint, failure string) (*T, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.config.BaseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("Authorization", "O-Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call PhonePe: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s (status %d): %s", failure, resp.StatusCode, string(body))
	}

	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &out, nil
}
