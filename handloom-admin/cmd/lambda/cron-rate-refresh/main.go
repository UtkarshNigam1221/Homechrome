// Package main is the Lambda entry point for the carrier rate matrix refresh.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/wire"
	"github.com/handloom/admin/pkg/slogx"
)

func main() {
	cfg := config.Load()
	slogx.Setup(cfg.App.Debug)
	slog.Info("Starting cron-rate-refresh Lambda")

	ctx := context.Background()
	deps, err := wire.InitializeCronRateRefreshDeps(ctx, cfg)
	if err != nil {
		slog.Error("Failed to initialize dependencies", "error", err)
		os.Exit(1)
	}

	lambda.Start(deps.Handler.Handle)
}
