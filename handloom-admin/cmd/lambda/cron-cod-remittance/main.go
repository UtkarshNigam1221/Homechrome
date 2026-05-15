// Package main is the Lambda entry point for the daily COD remittance pull.
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
	slog.Info("Starting cron-cod-remittance Lambda")

	ctx := context.Background()
	deps, err := wire.InitializeCronCODRemittanceDeps(ctx, cfg)
	if err != nil {
		slog.Error("Failed to initialize dependencies", "error", err)
		os.Exit(1)
	}

	lambda.Start(deps.Handler.Handle)
}
