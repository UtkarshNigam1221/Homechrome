package phonepe

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTokenThenHandler returns a handler that serves an OAuth token on /v1/oauth/token
// and delegates all other requests to the provided handler.
func fakeTokenThenHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/oauth/token" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(TokenResponse{
				AccessToken: "test-token",
				ExpiresAt:   9999999999,
				TokenType:   "O-Bearer",
			})
			return
		}
		next(w, r)
	}
}

func TestClient_InitiatePayment_Success(t *testing.T) {
	server := httptest.NewServer(fakeTokenThenHandler(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/checkout/v2/pay", r.URL.Path)
		assert.Equal(t, "O-Bearer test-token", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(PayResponse{
			OrderID:     "OMO123",
			State:       "PENDING",
			RedirectURL: "https://phonepe.com/pay/redirect",
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		ClientID:      "TEST_CLIENT",
		ClientSecret:  "test-secret",
		ClientVersion: "1",
		BaseURL:       server.URL,
		AuthBaseURL:   server.URL,
		RedirectURL:   "https://example.com/confirmation",
	})

	redirectURL, err := client.InitiatePayment(context.Background(), "txn_123", "cust_456", 10000, "order_789")
	require.NoError(t, err)
	assert.Equal(t, "https://phonepe.com/pay/redirect", redirectURL)
}

func TestClient_InitiatePayment_Failure(t *testing.T) {
	server := httptest.NewServer(fakeTokenThenHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(PayErrorResponse{
			Code:    "BAD_REQUEST",
			Message: "Invalid merchant",
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		ClientID:      "BAD_CLIENT",
		ClientSecret:  "test-secret",
		ClientVersion: "1",
		BaseURL:       server.URL,
		AuthBaseURL:   server.URL,
	})

	_, err := client.InitiatePayment(context.Background(), "txn_123", "cust_456", 10000, "order_789")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid merchant")
}

func TestClient_VerifyWebhookSignature(t *testing.T) {
	client := NewClient(Config{
		ClientID:     "C",
		ClientSecret: "S",
	})

	username := "homechrome"
	password := "secret123"
	expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(username+":"+password)))

	assert.True(t, client.VerifyWebhookSignature(username, password, expectedHash))
	assert.False(t, client.VerifyWebhookSignature(username, password, "invalid-signature"))
}

func TestClient_CheckPaymentStatus(t *testing.T) {
	server := httptest.NewServer(fakeTokenThenHandler(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/checkout/v2/order/txn_123/status")
		assert.Equal(t, "O-Bearer test-token", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(StatusResponse{
			OrderID: "OMO123",
			State:   StateCompleted,
			Amount:  10000,
			PaymentDetails: []PaymentDetail{
				{
					TransactionID: "TXN456",
					PaymentMode:   "UPI_QR",
					State:         StateCompleted,
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		ClientID:      "TEST_CLIENT",
		ClientSecret:  "test-secret",
		ClientVersion: "1",
		BaseURL:       server.URL,
		AuthBaseURL:   server.URL,
	})

	status, err := client.CheckPaymentStatus(context.Background(), "txn_123")
	require.NoError(t, err)
	assert.Equal(t, StateCompleted, status.State)
	assert.Equal(t, "OMO123", status.OrderID)
	assert.Len(t, status.PaymentDetails, 1)
	assert.Equal(t, "UPI_QR", status.PaymentDetails[0].PaymentMode)
}

func TestClient_CheckPaymentStatus_Error(t *testing.T) {
	server := httptest.NewServer(fakeTokenThenHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"code":"INVALID_MERCHANT_ORDER_ID","message":"No entry found"}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		ClientID:      "TEST_CLIENT",
		ClientSecret:  "test-secret",
		ClientVersion: "1",
		BaseURL:       server.URL,
		AuthBaseURL:   server.URL,
	})

	_, err := client.CheckPaymentStatus(context.Background(), "bad_txn")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 404")
}

func testRefundClient(baseURL string) *Client {
	return NewClient(Config{
		ClientID:      "TEST_CLIENT",
		ClientSecret:  "test-secret",
		ClientVersion: "1",
		BaseURL:       baseURL,
		AuthBaseURL:   baseURL,
	})
}

func TestClient_InitiateRefund_Success(t *testing.T) {
	var body map[string]any

	server := httptest.NewServer(fakeTokenThenHandler(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/payments/v2/refund", r.URL.Path)
		assert.Equal(t, "O-Bearer test-token", r.Header.Get("Authorization"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(RefundResponse{
			RefundID: "OMR456",
			Amount:   2500,
			State:    RefundStatePending,
		})
	}))
	defer server.Close()

	resp, err := testRefundClient(server.URL).InitiateRefund(
		context.Background(), "refund_abc", "txn_123", 2500)

	require.NoError(t, err)
	assert.Equal(t, "OMR456", resp.RefundID)
	assert.Equal(t, RefundStatePending, resp.State)

	// The body has to carry our id and the original order's, or the provider
	// cannot tie the refund to the payment it reverses.
	assert.Equal(t, "refund_abc", body["merchantRefundId"])
	assert.Equal(t, "txn_123", body["originalMerchantOrderId"])
	assert.EqualValues(t, 2500, body["amount"])
}

func TestClient_InitiateRefund_ProviderRejects(t *testing.T) {
	server := httptest.NewServer(fakeTokenThenHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"AMOUNT_EXCEEDS"}`))
	}))
	defer server.Close()

	_, err := testRefundClient(server.URL).InitiateRefund(
		context.Background(), "refund_abc", "txn_123", 999999)

	require.Error(t, err, "a rejected refund must not read as accepted")
}

func TestClient_CheckRefundStatus_Success(t *testing.T) {
	server := httptest.NewServer(fakeTokenThenHandler(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		// Keyed on our merchant refund id, which is the whole point of this
		// endpoint: it works even when the initiation response was lost.
		assert.Equal(t, "/payments/v2/refund/refund_abc/status", r.URL.Path)
		assert.Equal(t, "O-Bearer test-token", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(RefundStatusResponse{
			OriginalMerchantOrderID: "txn_123",
			RefundID:                "OMR456",
			Amount:                  2500,
			State:                   RefundStateCompleted,
		})
	}))
	defer server.Close()

	resp, err := testRefundClient(server.URL).CheckRefundStatus(context.Background(), "refund_abc")

	require.NoError(t, err)
	assert.Equal(t, RefundStateCompleted, resp.State)
	assert.Equal(t, "OMR456", resp.RefundID)
}

func TestClient_CheckRefundStatus_CarriesFailureCodes(t *testing.T) {
	server := httptest.NewServer(fakeTokenThenHandler(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"refundId":"OMR456","state":"FAILED",
			"errorCode":"REFUND_FAILED","detailedErrorCode":"INSUFFICIENT_BALANCE"}`))
	}))
	defer server.Close()

	resp, err := testRefundClient(server.URL).CheckRefundStatus(context.Background(), "refund_abc")

	require.NoError(t, err, "a failed refund is a valid answer, not a transport error")
	assert.Equal(t, RefundStateFailed, resp.State)
	assert.Equal(t, "REFUND_FAILED", resp.ErrorCode)
	assert.Equal(t, "INSUFFICIENT_BALANCE", resp.DetailedErrorCode)
}

// Development settles immediately: there is no provider to wait on, so a refund
// that stayed PENDING locally would never move.
func TestDevClient_RefundSettlesImmediately(t *testing.T) {
	dev := &DevClient{}

	initiated, err := dev.InitiateRefund(context.Background(), "refund_abc", "txn_123", 2500)
	require.NoError(t, err)
	assert.Equal(t, RefundStateCompleted, initiated.State)
	assert.NotEmpty(t, initiated.RefundID)

	status, err := dev.CheckRefundStatus(context.Background(), "refund_abc")
	require.NoError(t, err)
	assert.Equal(t, RefundStateCompleted, status.State)
	assert.Equal(t, initiated.RefundID, status.RefundID, "the same refund must keep one id")
}

func TestClient_TokenAndAPIHostsAreSeparate(t *testing.T) {
	var authPath, payPath string

	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"t","expires_at":9999999999}`))
	}))
	defer auth.Close()

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"refundId":"OMR_1","amount":100,"state":"PENDING"}`))
	}))
	defer api.Close()

	client := NewClient(Config{
		ClientID: "id", ClientSecret: "secret", ClientVersion: "1",
		BaseURL:     api.URL + "/apis/pg",
		AuthBaseURL: auth.URL + "/apis/identity-manager",
	})

	_, err := client.InitiateRefund(context.Background(), "mref_1", "txn_1", 100)
	require.NoError(t, err)

	require.Equal(t, "/apis/identity-manager/v1/oauth/token", authPath,
		"the token must come from the auth host")
	require.Equal(t, "/apis/pg/payments/v2/refund", payPath,
		"and everything else from the API host")
}

func TestClient_UATServesBothFromOneHost(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/apis/pg-sandbox/v1/oauth/token" {
			_, _ = w.Write([]byte(`{"access_token":"t","expires_at":9999999999}`))
			return
		}
		_, _ = w.Write([]byte(`{"refundId":"OMR_1","amount":100,"state":"PENDING"}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		ClientID: "id", ClientSecret: "secret", ClientVersion: "1",
		BaseURL:     server.URL + "/apis/pg-sandbox",
		AuthBaseURL: server.URL + "/apis/pg-sandbox",
	})

	_, err := client.InitiateRefund(context.Background(), "mref_1", "txn_1", 100)
	require.NoError(t, err)
	require.Equal(t, []string{"/apis/pg-sandbox/v1/oauth/token", "/apis/pg-sandbox/payments/v2/refund"}, paths)
}
