// Package main is the Lambda entry point for the Inventory service
package main

import (
	"context"
	"os"

	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/router"
	"github.com/handloom/admin/internal/wire"
	"github.com/handloom/admin/pkg/logger"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.App.Debug)
	log.Info("Starting Inventory Lambda")

	ctx := context.Background()

	deps, err := wire.InitializeInventoryDeps(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize dependencies: %v", err)
	}

	r := router.NewAuthenticatedRouter(
		router.Config{AllowedOrigins: getAllowedOrigins(), Debug: cfg.App.Debug},
		deps.Logger,
		deps.AuthMiddleware,
	)
	router.NewInventoryRouter(r, deps.Handler)

	router.NewLambdaAdapter(r).Start()
}

func getAllowedOrigins() []string {
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		return []string{origins}
	}
	return []string{"*"}
}
