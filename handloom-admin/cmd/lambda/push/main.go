// Package main is the Lambda entry point for the Web Push service.
//
// It serves both push surfaces: the public storefront routes under
// /api/v1/store/push/* that let a visitor manage their own device's
// subscription, and the operator-only console under /admin/push/* that reads
// subscriber records and broadcasts to everyone. The router starts
// unauthenticated and the admin routes add their own auth group, so the two
// surfaces cannot be confused for one another.
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
	bc := bootstrap.InitLambda("handloom-push")
	defer bc.Shutdown()

	ctx := context.Background()

	deps, err := wire.InitializePushDeps(ctx, bc.Cfg)
	if err != nil {
		slog.Error("Failed to initialize dependencies", "error", err)
		os.Exit(1)
	}

	r := router.NewBaseRouter(router.Config{
		AllowedOrigins: getAllowedOrigins(),
		Debug:          bc.Cfg.App.Debug,
	}, true)

	router.NewStorePushRouter(r, deps.StoreHandler, deps.CustomerAuthMiddleware)
	router.NewPushRouter(r, deps.AdminHandler, deps.AuthMiddleware)

	router.NewLambdaAdapter(r).Start()
}

func getAllowedOrigins() []string {
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		return []string{origins}
	}
	return []string{"*"}
}
