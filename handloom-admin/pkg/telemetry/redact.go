package telemetry

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
)

const piiRedactedPlaceholder = "[REDACTED]"

// spanDropKeys lists attribute key substrings whose values are replaced with
// piiRedactedPlaceholder by redactSpanAttr.
var spanDropKeys = []string{
	"password", "passwd", "secret", "token", "jwt", "cookie",
	"authorization", "auth_key", "api_key",
	"otp", "otp_code", "verification_code",
	"card_number", "pan", "cvv",
	"first_name", "last_name",
	"address", "street", "line1", "line2", "pincode",
}

// RedactingSpanProcessor is a no-op SpanProcessor placeholder. Span attribute
// redaction in the Go SDK can only happen via callers invoking redactSpanAttr
// before SetAttributes, or via the collector-side attributes/redact processor.
// This type exists so the tracer pipeline can be assembled symmetrically with
// other processors when needed.
type RedactingSpanProcessor struct{}

func NewRedactingSpanProcessor() *RedactingSpanProcessor { return &RedactingSpanProcessor{} }

func (p *RedactingSpanProcessor) OnStart(_ context.Context, _ trace.ReadWriteSpan) {}
func (p *RedactingSpanProcessor) OnEnd(_ trace.ReadOnlySpan)                       {}
func (p *RedactingSpanProcessor) Shutdown(_ context.Context) error                 { return nil }
func (p *RedactingSpanProcessor) ForceFlush(_ context.Context) error               { return nil }

// RedactAttr is the exported helper callers can use when setting span
// attributes that might contain PII.
func RedactAttr(a attribute.KeyValue) attribute.KeyValue { return redactSpanAttr(a) }

func redactSpanAttr(a attribute.KeyValue) attribute.KeyValue {
	key := strings.ToLower(string(a.Key))
	for _, drop := range spanDropKeys {
		if strings.Contains(key, drop) {
			return attribute.String(string(a.Key), piiRedactedPlaceholder)
		}
	}
	if strings.Contains(key, "email") {
		return attribute.String(string(a.Key), redactSpanEmail(a.Value.AsString()))
	}
	if strings.Contains(key, "phone") || strings.Contains(key, "mobile") || strings.Contains(key, "msisdn") {
		return attribute.String(string(a.Key), redactSpanPhone(a.Value.AsString()))
	}
	return a
}

func redactSpanEmail(s string) string {
	at := strings.LastIndex(s, "@")
	if at <= 0 {
		return piiRedactedPlaceholder
	}
	return "***" + s[at:]
}

func redactSpanPhone(s string) string {
	if len(s) <= 4 {
		return piiRedactedPlaceholder
	}
	return "***" + s[len(s)-4:]
}
