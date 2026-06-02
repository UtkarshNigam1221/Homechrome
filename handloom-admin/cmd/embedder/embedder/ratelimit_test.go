package embedder

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_AllowsUnderCap(t *testing.T) {
	rl := NewIPRateLimiter(5)
	h := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d should pass", i+1)
	}
}

func TestRateLimiter_BlocksOverCap(t *testing.T) {
	rl := NewIPRateLimiter(2)
	h := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "9.9.9.9:1"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "9.9.9.9:1"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}
