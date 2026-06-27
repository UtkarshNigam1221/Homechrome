package slogx

import (
	"context"
	"log/slog"
	"strings"
)

const redactedPlaceholder = "[REDACTED]"

// dropKeys names attribute keys whose values are replaced with [REDACTED].
// Matching is case-insensitive substring on the attribute key.
var dropKeys = []string{
	"password", "passwd", "secret", "token", "jwt", "cookie",
	"authorization", "auth_key", "api_key",
	"otp", "otp_code", "verification_code",
	"card_number", "pan", "cvv",
	"first_name", "last_name", "name",
	"address", "street", "line1", "line2", "pincode",
}

// RedactingHandler wraps another slog.Handler and scrubs PII attribute values
// before forwarding the record. Keys are matched case-insensitively at word
// boundaries (start, end, or non-alphanumeric neighbor), so "user_password"
// IS redacted (token match around `_`) while "span_id" is NOT (the `pan`
// token is surrounded by alphanumerics on both sides).
type RedactingHandler struct {
	inner slog.Handler
}

func NewRedactingHandler(inner slog.Handler) *RedactingHandler {
	return &RedactingHandler{inner: inner}
}

func (h *RedactingHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return h.inner.Enabled(ctx, lvl)
}

func (h *RedactingHandler) Handle(ctx context.Context, r slog.Record) error {
	clone := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		clone.AddAttrs(redactAttr(a))
		return true
	})
	return h.inner.Handle(ctx, clone)
}

func (h *RedactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	redacted := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		redacted[i] = redactAttr(a)
	}
	return &RedactingHandler{inner: h.inner.WithAttrs(redacted)}
}

func (h *RedactingHandler) WithGroup(name string) slog.Handler {
	return &RedactingHandler{inner: h.inner.WithGroup(name)}
}

func redactAttr(a slog.Attr) slog.Attr {
	key := strings.ToLower(a.Key)
	for _, drop := range dropKeys {
		if matchesToken(key, drop) {
			return slog.String(a.Key, redactedPlaceholder)
		}
	}
	if matchesToken(key, "email") {
		return slog.String(a.Key, redactEmail(a.Value.String()))
	}
	if matchesToken(key, "phone") || matchesToken(key, "mobile") || matchesToken(key, "msisdn") {
		return slog.String(a.Key, redactPhone(a.Value.String()))
	}
	return a
}

// matchesToken returns true when `token` appears in `key` bounded by start,
// end, or a non-alphanumeric character. Prevents accidental matches like
// `pan` inside `span_id`.
func matchesToken(key, token string) bool {
	idx := 0
	for {
		i := strings.Index(key[idx:], token)
		if i < 0 {
			return false
		}
		start := idx + i
		end := start + len(token)
		leftOK := start == 0 || !isWordChar(key[start-1])
		rightOK := end == len(key) || !isWordChar(key[end])
		if leftOK && rightOK {
			return true
		}
		idx = start + 1
	}
}

func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func redactEmail(s string) string {
	at := strings.LastIndex(s, "@")
	if at <= 0 {
		return redactedPlaceholder
	}
	return "***" + s[at:]
}

func redactPhone(s string) string {
	if len(s) <= 4 {
		return redactedPlaceholder
	}
	return "***" + s[len(s)-4:]
}
