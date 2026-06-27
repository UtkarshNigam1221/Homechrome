// Package main is the Lambda entry point for the Coupon service
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
	bc := bootstrap.InitLambda("handloom-coupon")
	defer bc.Shutdown()

	ctx := context.Background()

	deps, err := wire.InitializeCouponDeps(ctx, bc.Cfg)
	if err != nil {
		slog.Error("Failed to initialize dependencies", "error", err)
		os.Exit(1)
	}

	r := router.NewAuthenticatedRouter(
		router.Config{AllowedOrigins: getAllowedOrigins(), Debug: bc.Cfg.App.Debug},
		deps.AuthMiddleware,
	)
	router.NewCouponRouter(r, deps.Handler)

	router.NewLambdaAdapter(r).Start()
}

func getAllowedOrigins() []string {
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		return []string{origins}
	}
	return []string{"*"}
}
