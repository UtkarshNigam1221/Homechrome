package sms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSMSClient_SendOTP_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v5/flow/", r.URL.Path)
		assert.Equal(t, "test-auth-key", r.Header.Get("authkey"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, "tmpl-123", payload["template_id"])

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"type": "success"})
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:       server.URL,
		AuthKey:       "test-auth-key",
		OTPTemplateID: "tmpl-123",
	})

	err := client.SendOTP(context.Background(), "+919876543210", "123456")
	assert.NoError(t, err)
}

func TestSMSClient_SendOTP_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:       server.URL,
		AuthKey:       "test-auth-key",
		OTPTemplateID: "tmpl-123",
	})

	err := client.SendOTP(context.Background(), "+919876543210", "123456")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestSMSClient_SendOTP_MSG91Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"type":    "error",
			"message": "Invalid template",
		})
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL:       server.URL,
		AuthKey:       "test-auth-key",
		OTPTemplateID: "invalid-tmpl",
	})

	err := client.SendOTP(context.Background(), "+919876543210", "123456")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid template")
}

func TestNewClient_DefaultBaseURL(t *testing.T) {
	client := NewClient(Config{AuthKey: "key", OTPTemplateID: "tmpl"})
	assert.Equal(t, "https://control.msg91.com", client.config.BaseURL)
}
