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
	})

	_, err := client.CheckPaymentStatus(context.Background(), "bad_txn")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 404")
}

func TestNewClient_Defaults(t *testing.T) {
	client := NewClient(Config{ClientID: "C", ClientSecret: "S"})
	assert.Equal(t, "https://api-preprod.phonepe.com/apis/pg-sandbox", client.config.BaseURL)
}
