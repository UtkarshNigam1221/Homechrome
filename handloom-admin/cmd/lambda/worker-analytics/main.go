package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/handloom/admin/internal/config"
	"github.com/handloom/admin/internal/event/handlers"
	"github.com/handloom/admin/internal/repository/dynamodb"
	"github.com/handloom/admin/pkg/logger"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.App.Debug)
	log.Info("Starting Worker Analytics Lambda")

	ctx := context.Background()
	dbClient, err := dynamodb.NewClient(ctx, cfg)
	if err != nil {
		log.WithError(err).Fatal("failed to create DynamoDB client")
	}

	eventsRepo := dynamodb.NewEventsRepository(dbClient)
	analyticsRepo := dynamodb.NewAnalyticsRepository(dbClient)

	handler := handlers.NewAnalyticsHandler(log, eventsRepo, analyticsRepo)
	lambda.Start(handler.HandleSQSEvent)
}
