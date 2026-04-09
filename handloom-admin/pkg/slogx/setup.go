package slogx

import (
	"log/slog"
	"os"
)

// Setup configures the global slog default logger.
// Debug mode: text handler with source info, debug level.
// Production: JSON handler, info level.
// If SERVICE_NAME is set, it is attached to every log record.
func Setup(debug bool) {
	var inner slog.Handler

	if debug {
		inner = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		})
	} else {
		inner = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	if svc := os.Getenv("SERVICE_NAME"); svc != "" {
		inner = inner.WithAttrs([]slog.Attr{slog.String("service", svc)})
	}

	handler := NewContextHandler(inner)
	slog.SetDefault(slog.New(handler))
}
