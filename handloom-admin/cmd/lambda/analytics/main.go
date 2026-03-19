// Package main is the Lambda entry point for the Analytics service
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/router"
	"github.com/handloom/admin/internal/wire"
	"github.com/handloom/admin/pkg/slogx"
)

func main() {
	cfg := config.Load()
	slogx.Setup(cfg.App.Debug)
	slog.Info("Starting Analytics Lambda")

	ctx := context.Background()

	deps, err := wire.InitializeAnalyticsDeps(ctx, cfg)
	if err != nil {
		slog.Error("Failed to initialize dependencies", "error", err)
		os.Exit(1)
	}

	r := router.NewAuthenticatedRouter(
		router.Config{AllowedOrigins: getAllowedOrigins(), Debug: cfg.App.Debug},
		deps.AuthMiddleware,
	)
	router.NewAnalyticsRouter(r, deps.Handler)

	router.NewLambdaAdapter(r).Start()
}

func getAllowedOrigins() []string {
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		return []string{origins}
	}
	return []string{"*"}
}
