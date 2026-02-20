package main

import (
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/event/handlers"
	"github.com/handloom/admin/pkg/logger"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.App.Debug)
	log.Info("Starting Worker Audit Lambda")

	handler := handlers.NewAuditHandler(log)
	lambda.Start(handler.HandleSQSEvent)
}
