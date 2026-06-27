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
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("/checkout/v2/order/%s/status?details=true", merchantTxnID)
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
		return nil, fmt.Errorf("PhonePe status check failed (status %d): %s", resp.StatusCode, string(body))
	}

	var statusResp StatusResponse
	if err := json.Unmarshal(body, &statusResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &statusResp, nil
}

// VerifyWebhookSignature verifies the Authorization header from a PhonePe webhook.
// PhonePe sends: Authorization: SHA256(username:password)
// We compute the same and compare.
func (c *Client) VerifyWebhookSignature(username, password, authHeader string) bool {
	expected := fmt.Sprintf("%x", sha256.Sum256([]byte(username+":"+password)))
	return subtle.ConstantTimeCompare([]byte(expected), []byte(authHeader)) == 1
}
