package embedder

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// IPRateLimiter is a token-bucket rate limiter keyed by source IP.
// Buckets refill at `perMinute` tokens per minute and start full.
type IPRateLimiter struct {
	perMinute float64
	mu        sync.Mutex
	buckets   map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewIPRateLimiter constructs a limiter with the given per-IP request/minute cap.
func NewIPRateLimiter(perMinute int) *IPRateLimiter {
	return &IPRateLimiter{
		perMinute: float64(perMinute),
		buckets:   make(map[string]*bucket),
	}
}

// Middleware enforces the per-IP limit and returns 429 when exceeded.
func (l *IPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !l.allow(ip) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *IPRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	b, ok := l.buckets[ip]
	if !ok {
		b = &bucket{tokens: l.perMinute, last: now}
		l.buckets[ip] = b
	}
	// Refill: tokens accrue at perMinute/60 per second; cap at perMinute.
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * (l.perMinute / 60.0)
	if b.tokens > l.perMinute {
		b.tokens = l.perMinute
	}
	b.last = now
	if b.tokens < 1.0 {
		return false
	}
	b.tokens--
	return true
}

// clientIP returns the request's source IP, honoring X-Forwarded-For (first hop)
// for Lambda Function URL / API GW deployments.
func clientIP(r *http.Request) string {
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		if comma := strings.IndexByte(xf, ','); comma > 0 {
			return strings.TrimSpace(xf[:comma])
		}
		return strings.TrimSpace(xf)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
