package slogx

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestContextHandler_AddsRequestID(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, nil)
	h := NewContextHandler(inner)
	logger := slog.New(h)

	ctx := SetRequestID(context.Background(), "req-123")
	logger.InfoContext(ctx, "test message")

	var record map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record["request_id"] != "req-123" {
		t.Errorf("expected request_id=req-123, got %v", record["request_id"])
	}
}

func TestContextHandler_AddsUserID(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, nil)
	h := NewContextHandler(inner)
	logger := slog.New(h)

	ctx := SetUserID(context.Background(), "user-456")
	logger.InfoContext(ctx, "test message")

	var record map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record["user_id"] != "user-456" {
		t.Errorf("expected user_id=user-456, got %v", record["user_id"])
	}
}

func TestContextHandler_NoContextValues(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, nil)
	h := NewContextHandler(inner)
	logger := slog.New(h)

	logger.InfoContext(context.Background(), "test message")

	var record map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if _, exists := record["request_id"]; exists {
		t.Error("request_id should not be present when not in context")
	}
}
