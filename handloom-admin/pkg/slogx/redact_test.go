package slogx

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestRedactingHandler_StripsPasswordsAndTokens(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, nil)
	h := NewRedactingHandler(inner)
	logger := slog.New(h)

	logger.InfoContext(context.Background(), "auth attempt",
		"password", "hunter2",
		"jwt", "eyJhbGc...",
		"authorization", "Bearer abc",
		"user_id", "u_123",
	)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, k := range []string{"password", "jwt", "authorization"} {
		v, ok := entry[k]
		if !ok {
			continue
		}
		if s, _ := v.(string); s != "[REDACTED]" {
			t.Errorf("expected %s redacted, got %v", k, v)
		}
	}
	if entry["user_id"] != "u_123" {
		t.Errorf("user_id should pass through, got %v", entry["user_id"])
	}
}

func TestRedactingHandler_EmailKeepsDomain(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, nil)
	h := NewRedactingHandler(inner)
	slog.New(h).Info("login", "email", "alice@example.com")

	if !strings.Contains(buf.String(), `"email":"***@example.com"`) {
		t.Errorf("email not partially redacted: %s", buf.String())
	}
}

func TestRedactingHandler_PhoneKeepsLast4(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, nil)
	h := NewRedactingHandler(inner)
	slog.New(h).Info("otp", "phone", "+919876543210")

	if !strings.Contains(buf.String(), `"phone":"***3210"`) {
		t.Errorf("phone not partially redacted: %s", buf.String())
	}
}
