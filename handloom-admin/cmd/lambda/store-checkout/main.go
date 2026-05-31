// Package main is the Lambda entry point for the Store Checkout service
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/handloom/admin/internal/bootstrap"
	"github.com/handloom/admin/internal/router"
	"github.com/handloom/admin/internal/wire"
)

func main() {
	bc := bootstrap.InitLambda("handloom-store-checkout")
	defer bc.Shutdown()

	ctx := context.Background()

	deps, err := wire.InitializeStoreCheckoutDeps(ctx, bc.Cfg)
	if err != nil {
		slog.Error("Failed to initialize dependencies", "error", err)
		os.Exit(1)
	}

	routerCfg := router.Config{
		AllowedOrigins: getAllowedOrigins(),
		Debug:          bc.Cfg.App.Debug,
	}
	r := router.NewBaseRouter(routerCfg, true)

	router.NewStoreCheckoutRouter(r, deps.Handler, deps.CustomerAuthMiddleware)

	router.NewLambdaAdapter(r).Start()
}

func getAllowedOrigins() []string {
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		return []string{origins}
	}
	return []string{"*"}
}
