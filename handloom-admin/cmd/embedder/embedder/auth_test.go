package embedder

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHMACAuth_AcceptsValidSignature(t *testing.T) {
	key := []byte("test-key")
	v := NewHMACVerifier(key)

	body := []byte(`{"texts":["hi"]}`)
	ts, nonce := nowSecForTest(), "nonce-1"
	sig := signForTest(key, ts, nonce, body)

	req := httptest.NewRequest(http.MethodPost, "/embed", bytes.NewReader(body))
	req.Header.Set("X-Embedder-Timestamp", ts)
	req.Header.Set("X-Embedder-Nonce", nonce)
	req.Header.Set("X-Embedder-Signature", sig)

	rec := httptest.NewRecorder()
	v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestHMACAuth_RejectsBadSignature(t *testing.T) {
	v := NewHMACVerifier([]byte("test-key"))
	body := []byte(`{"texts":["hi"]}`)
	req := httptest.NewRequest(http.MethodPost, "/embed", bytes.NewReader(body))
	req.Header.Set("X-Embedder-Timestamp", nowSecForTest())
	req.Header.Set("X-Embedder-Nonce", "n2")
	req.Header.Set("X-Embedder-Signature", "deadbeef")

	rec := httptest.NewRecorder()
	v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {})).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHMACAuth_RejectsExpiredTimestamp(t *testing.T) {
	key := []byte("test-key")
	v := NewHMACVerifier(key)

	body := []byte(`{"texts":["hi"]}`)
	staleTs := timeToSec(time.Now().Add(-10 * time.Minute))
	nonce := "n3"
	sig := signForTest(key, staleTs, nonce, body)

	req := httptest.NewRequest(http.MethodPost, "/embed", bytes.NewReader(body))
	req.Header.Set("X-Embedder-Timestamp", staleTs)
	req.Header.Set("X-Embedder-Nonce", nonce)
	req.Header.Set("X-Embedder-Signature", sig)

	rec := httptest.NewRecorder()
	v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {})).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHMACAuth_RejectsReplayedNonce(t *testing.T) {
	key := []byte("test-key")
	v := NewHMACVerifier(key)
	body := []byte(`{"texts":["hi"]}`)
	ts, nonce := nowSecForTest(), "replay"
	sig := signForTest(key, ts, nonce, body)

	build := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/embed", bytes.NewReader(body))
		req.Header.Set("X-Embedder-Timestamp", ts)
		req.Header.Set("X-Embedder-Nonce", nonce)
		req.Header.Set("X-Embedder-Signature", sig)
		return req
	}

	rec1 := httptest.NewRecorder()
	v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rec1, build())
	require.Equal(t, http.StatusOK, rec1.Code)

	rec2 := httptest.NewRecorder()
	v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {})).ServeHTTP(rec2, build())
	assert.Equal(t, http.StatusUnauthorized, rec2.Code, "replayed nonce must be rejected")
}
