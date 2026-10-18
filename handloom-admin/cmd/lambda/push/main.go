// Package main is the Web Push Lambda: public device routes under
// /api/v1/store/push/* and the admin-only console under /admin/push/*.
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

	router.NewStorePushRouter(r, deps.StoreHandler)
	router.NewPushRouter(r, deps.AdminHandler, deps.AuthMiddleware)

	router.NewLambdaAdapter(r).Start()
}

func getAllowedOrigins() []string {
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		return []string{origins}
	}
	return []string{"*"}
}
