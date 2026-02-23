// Package main is the Lambda entry point for the Store Webhooks service
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
	log.Info("Starting Store Webhooks Lambda")

	ctx := context.Background()

	deps, err := wire.InitializeStoreWebhooksDeps(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize dependencies: %v", err)
	}

	routerCfg := router.Config{
		AllowedOrigins: getAllowedOrigins(),
		Debug:          cfg.App.Debug,
	}
	r := router.NewBaseRouter(routerCfg, deps.Logger, true)

	router.NewStoreWebhooksRouter(r, deps.Handler)

	router.NewLambdaAdapter(r).Start()
}

func getAllowedOrigins() []string {
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		return []string{origins}
	}
	return []string{"*"}
}
