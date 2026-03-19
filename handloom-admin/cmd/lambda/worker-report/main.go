package main

import (
	"log/slog"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/event/handlers"
	"github.com/handloom/admin/pkg/slogx"
)

func main() {
	cfg := config.Load()
	slogx.Setup(cfg.App.Debug)
	slog.Info("Starting Worker Report Lambda")

	handler := handlers.NewReportHandler()
	lambda.Start(handler.HandleSQSEvent)
}
