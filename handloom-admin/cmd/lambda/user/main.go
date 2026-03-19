// Package main is the Lambda entry point for the User service
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
	slog.Info("Starting User Lambda")

	ctx := context.Background()

	deps, err := wire.InitializeUserDeps(ctx, cfg)
	if err != nil {
		slog.Error("Failed to initialize dependencies", "error", err)
		os.Exit(1)
	}

	// Create authenticated router
	routerCfg := router.Config{
		AllowedOrigins: getAllowedOrigins(),
		Debug:          cfg.App.Debug,
	}
	r := router.NewAuthenticatedRouter(routerCfg, deps.AuthMiddleware)

	// Register routes
	router.NewUserRouter(r, deps.Handler, deps.AuthMiddleware)

	// Start Lambda
	adapter := router.NewLambdaAdapter(r)
	adapter.Start()
}

func getAllowedOrigins() []string {
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		return []string{origins}
	}
	return []string{"*"}
}
