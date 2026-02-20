package phonepe

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_InitiatePayment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/pg/v1/pay", r.URL.Path)
		assert.Contains(t, r.Header.Get("X-VERIFY"), "###1")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(PayResponse{
			Success: true,
			Code:    "PAYMENT_INITIATED",
			Data: struct {
				MerchantID            string `json:"merchantId"`
				MerchantTransactionID string `json:"merchantTransactionId"`
				InstrumentResponse    struct {
					Type         string `json:"type"`
					RedirectInfo struct {
						URL    string `json:"url"`
						Method string `json:"method"`
					} `json:"redirectInfo"`
				} `json:"instrumentResponse"`
			}{
				InstrumentResponse: struct {
					Type         string `json:"type"`
					RedirectInfo struct {
						URL    string `json:"url"`
						Method string `json:"method"`
					} `json:"redirectInfo"`
				}{
					RedirectInfo: struct {
						URL    string `json:"url"`
						Method string `json:"method"`
					}{
						URL:    "https://phonepe.com/pay/redirect",
						Method: "GET",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		MerchantID:  "TEST_MERCHANT",
		SaltKey:     "test-salt-key",
		SaltIndex:   "1",
		BaseURL:     server.URL,
		CallbackURL: "https://example.com/callback",
		RedirectURL: "https://example.com/redirect",
	})

	redirectURL, err := client.InitiatePayment(context.Background(), "txn_123", "cust_456", 10000)
	require.NoError(t, err)
	assert.Equal(t, "https://phonepe.com/pay/redirect", redirectURL)
}

func TestClient_InitiatePayment_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(PayResponse{
			Success: false,
			Code:    "PAYMENT_ERROR",
			Message: "Invalid merchant",
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		MerchantID: "BAD_MERCHANT",
		SaltKey:    "test-salt-key",
		BaseURL:    server.URL,
	})

	_, err := client.InitiatePayment(context.Background(), "txn_123", "cust_456", 10000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid merchant")
}

func TestClient_VerifyWebhookSignature(t *testing.T) {
	client := NewClient(Config{
		SaltKey:   "test-salt-key",
		SaltIndex: "1",
	})

	responseBase64 := base64.StdEncoding.EncodeToString([]byte(`{"success":true}`))
	expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(responseBase64+"test-salt-key")))
	validSignature := expectedHash + "###1"

	assert.True(t, client.VerifyWebhookSignature(responseBase64, validSignature))
	assert.False(t, client.VerifyWebhookSignature(responseBase64, "invalid-signature"))
}

func TestClient_CheckPaymentStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/pg/v1/status/TEST_MERCHANT/txn_123")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(StatusResponse{
			Success: true,
			Code:    "PAYMENT_SUCCESS",
			Data: struct {
				MerchantID            string `json:"merchantId"`
				MerchantTransactionID string `json:"merchantTransactionId"`
				TransactionID         string `json:"transactionId"`
				Amount                int64  `json:"amount"`
				State                 string `json:"state"`
				ResponseCode          string `json:"responseCode"`
				PaymentInstrument     struct {
					Type string `json:"type"`
					UTR  string `json:"utr,omitempty"`
				} `json:"paymentInstrument"`
			}{
				State:                 "COMPLETED",
				MerchantTransactionID: "txn_123",
				Amount:                10000,
			},
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		MerchantID: "TEST_MERCHANT",
		SaltKey:    "test-salt-key",
		BaseURL:    server.URL,
	})

	status, err := client.CheckPaymentStatus(context.Background(), "txn_123")
	require.NoError(t, err)
	assert.True(t, status.Success)
	assert.Equal(t, "COMPLETED", status.Data.State)
}

func TestNewClient_Defaults(t *testing.T) {
	client := NewClient(Config{MerchantID: "M", SaltKey: "K"})
	assert.Equal(t, "https://api-preprod.phonepe.com/apis/pg-sandbox", client.config.BaseURL)
	assert.Equal(t, "1", client.config.SaltIndex)
}
