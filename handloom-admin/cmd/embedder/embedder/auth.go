package embedder

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	hdrTimestamp = "X-Embedder-Timestamp"
	hdrNonce     = "X-Embedder-Nonce"
	hdrSignature = "X-Embedder-Signature"
	maxClockSkew = 5 * time.Minute
	nonceCacheN  = 4096
)

// HMACVerifier authenticates internal POST /embed callers.
type HMACVerifier struct {
	key       []byte
	mu        sync.Mutex
	seenOrder []string
	seen      map[string]struct{}
}

// NewHMACVerifier creates an HMACVerifier with the given shared secret.
func NewHMACVerifier(key []byte) *HMACVerifier {
	return &HMACVerifier{
		key:  key,
		seen: make(map[string]struct{}, nonceCacheN),
	}
}

// Middleware wraps the next handler with HMAC verification.
func (v *HMACVerifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts := r.Header.Get(hdrTimestamp)
		nonce := r.Header.Get(hdrNonce)
		sig := r.Header.Get(hdrSignature)
		if ts == "" || nonce == "" || sig == "" {
			http.Error(w, "missing auth headers", http.StatusUnauthorized)
			return
		}

		tsInt, err := strconv.ParseInt(ts, 10, 64)
		if err != nil {
			http.Error(w, "bad timestamp", http.StatusUnauthorized)
			return
		}
		if abs64(time.Now().Unix()-tsInt) > int64(maxClockSkew.Seconds()) {
			http.Error(w, "stale request", http.StatusUnauthorized)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		// Replace body so downstream handler can read it.
		r.Body = io.NopCloser(bytes.NewReader(body))

		expected := computeHMAC(v.key, ts, nonce, body)
		if !hmac.Equal([]byte(sig), []byte(expected)) {
			http.Error(w, "bad signature", http.StatusUnauthorized)
			return
		}

		if !v.acceptNonce(nonce) {
			http.Error(w, "replayed nonce", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// computeHMAC formats the canonical message (ts \n nonce \n body) and signs it.
func computeHMAC(key []byte, ts, nonce string, body []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(ts))
	h.Write([]byte("\n"))
	h.Write([]byte(nonce))
	h.Write([]byte("\n"))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

// acceptNonce returns true if the nonce hasn't been seen recently. Memory is
// bounded by an LRU cap of nonceCacheN; a replay beyond the skew window is
// already rejected by the timestamp check before this runs.
func (v *HMACVerifier) acceptNonce(nonce string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	if _, dup := v.seen[nonce]; dup {
		return false
	}
	v.seen[nonce] = struct{}{}
	v.seenOrder = append(v.seenOrder, nonce)
	if len(v.seenOrder) > nonceCacheN {
		old := v.seenOrder[0]
		v.seenOrder = v.seenOrder[1:]
		delete(v.seen, old)
	}
	return true
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
