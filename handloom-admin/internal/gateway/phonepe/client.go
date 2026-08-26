package phonepe

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.AuthBaseURL+"/v1/oauth/token", strings.NewReader(data.Encode()))
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

// maxSendAttempts bounds the 429 retry. Three waits (0.5s, 1s, 2s) keep a
// throttled checkout slow rather than failed, inside the 30s client timeout.
const maxSendAttempts = 4

// sendWithRetry performs an authenticated request, retrying only a 429. PhonePe
// throttles bursts, and every id in these payloads (merchantOrderId,
// merchantRefundId) is stable across attempts, so a retry re-addresses the same
// order rather than creating a second one.
func (c *Client) sendWithRetry(ctx context.Context, method, url string, body []byte, token string) (int, []byte, error) {
	var status int
	var respBody []byte

	for attempt := range maxSendAttempts {
		if attempt > 0 {
			wait := time.Duration(1<<(attempt-1)) * 500 * time.Millisecond
			select {
			case <-ctx.Done():
				return 0, nil, ctx.Err()
			case <-time.After(wait):
			}
		}

		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reader)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", contentTypeJSON)
		req.Header.Set("Authorization", "O-Bearer "+token)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to call PhonePe: %w", err)
		}
		respBody, err = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return 0, nil, fmt.Errorf("failed to read response: %w", err)
		}
		status = resp.StatusCode

		if status != http.StatusTooManyRequests {
			return status, respBody, nil
		}
		slog.WarnContext(ctx, "PhonePe throttled the request, retrying",
			"method", method, "attempt", attempt+1, "of", maxSendAttempts)
	}

	return status, respBody, nil
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

	status, body, err := c.sendWithRetry(ctx, http.MethodPost, c.config.BaseURL+"/checkout/v2/pay", reqBody, token)
	if err != nil {
		return "", err
	}

	if status != http.StatusOK {
		var errResp PayErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Code != "" {
			return "", fmt.Errorf("PhonePe payment initiation failed: %s - %s", errResp.Code, errResp.Message)
		}
		return "", fmt.Errorf("PhonePe payment initiation failed (status %d): %s", status, string(body))
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

// VerifyWebhookSignature checks the PhonePe webhook Authorization header, which is
// SHA256(username:password). Computes the same and compares.
func (c *Client) VerifyWebhookSignature(username, password, authHeader string) bool {
	expected := fmt.Sprintf("%x", sha256.Sum256([]byte(username+":"+password)))
	return subtle.ConstantTimeCompare([]byte(expected), []byte(authHeader)) == 1
}

// InitiateRefund asks PhonePe to send money back. originalMerchantOrderID is checkout's
// MerchantTransactionID; merchantRefundID is ours, and the only handle if this is lost.
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

	status, body, err := c.sendWithRetry(ctx, http.MethodPost, c.config.BaseURL+"/payments/v2/refund", payload, token)
	if err != nil {
		return nil, err
	}

	// A 4xx is the provider answering no. A 5xx, a 429 that outlived its retries,
	// a timeout or an unreadable body is the provider not answering, which is not
	// the same thing and must not be recorded as a refusal — see ErrRejected.
	if status >= 400 && status < 500 && status != http.StatusTooManyRequests {
		return nil, fmt.Errorf("%w (status %d): %s", ErrRejected, status, string(body))
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return nil, fmt.Errorf("PhonePe refund failed (status %d): %s", status, string(body))
	}

	var refundResp RefundResponse
	if err := json.Unmarshal(body, &refundResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &refundResp, nil
}

// CheckRefundStatus reads the provider's view of a refund, keyed on our merchantRefundID
// rather than PhonePe's — which is what makes it the recovery path.
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

	status, body, err := c.sendWithRetry(ctx, http.MethodGet, c.config.BaseURL+endpoint, nil, token)
	if err != nil {
		return nil, err
	}

	if status != http.StatusOK {
		return nil, fmt.Errorf("%s (status %d): %s", failure, status, string(body))
	}

	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &out, nil
}
