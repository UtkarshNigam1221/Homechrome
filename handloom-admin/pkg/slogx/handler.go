package slogx

import (
	"context"
	"log/slog"
)

// ContextHandler wraps a slog.Handler and enriches each record with
// request_id and user_id from context.
type ContextHandler struct {
	inner slog.Handler
}

// NewContextHandler creates a handler that extracts context values into log attributes.
func NewContextHandler(inner slog.Handler) *ContextHandler {
	return &ContextHandler{inner: inner}
}

func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := GetRequestID(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	if id := GetUserID(ctx); id != "" {
		r.AddAttrs(slog.String("user_id", id))
	}
	return h.inner.Handle(ctx, r)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{inner: h.inner.WithGroup(name)}
}
