package slogx

import (
	"log/slog"
	"os"
)

// Setup configures the global slog default logger.
// Handler chain: Context -> Redacting -> JSON/Text -> stdout.
func Setup(debug bool) {
	var base slog.Handler
	// AddSource intentionally omitted everywhere — file:line is noise once
	// you have trace_id + span_id correlation, and it bloats log size.
	if debug {
		base = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		base = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	if svc := os.Getenv("SERVICE_NAME"); svc != "" {
		base = base.WithAttrs([]slog.Attr{slog.String("service", svc)})
	}

	handler := NewContextHandler(NewRedactingHandler(base))
	slog.SetDefault(slog.New(handler))
}
