package slogx

import (
	"log/slog"
	"os"
)

// Setup configures the global slog default logger.
// Debug mode: text handler with source info, debug level.
// Production: JSON handler, info level.
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

	handler := NewContextHandler(inner)
	slog.SetDefault(slog.New(handler))
}
