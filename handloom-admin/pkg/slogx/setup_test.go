package slogx

import (
	"context"
	"log/slog"
	"testing"
)

func TestSetup_Debug(t *testing.T) {
	Setup(true)
	if !slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("debug mode should enable debug level")
	}
}

func TestSetup_Production(t *testing.T) {
	Setup(false)
	if slog.Default().Enabled(context.Background(), slog.LevelDebug) {
		t.Error("production mode should not enable debug level")
	}
}
