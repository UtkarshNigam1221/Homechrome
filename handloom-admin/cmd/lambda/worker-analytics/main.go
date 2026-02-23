package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/handloom/admin/internal/config"
	apphandlers "github.com/handloom/admin/internal/event/handlers"
	"github.com/handloom/admin/internal/repository/dynamodb"
	"github.com/handloom/admin/internal/service"
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
	aggregator := service.NewAnalyticsAggregator(eventsRepo, analyticsRepo, log)

	handler := apphandlers.NewAnalyticsHandler(log, eventsRepo, analyticsRepo, aggregator)

	// The Lambda handles two event sources:
	// 1. SQS messages (real-time events from SNS fan-out)
	// 2. EventBridge scheduled events (daily aggregation trigger)
	//
	// We use (interface{}, error) return so SQS batch item failures are
	// properly reported back to the SQS event source mapping.
	lambda.Start(func(ctx context.Context, event json.RawMessage) (interface{}, error) {
		// Try SQS first — if Records array is present and non-empty, it's an SQS event
		var sqsEvent events.SQSEvent
		if err := json.Unmarshal(event, &sqsEvent); err == nil && len(sqsEvent.Records) > 0 {
			return handler.HandleSQSEvent(ctx, sqsEvent)
		}

		// Fall back to scheduled event (EventBridge)
		log.Info("Received scheduled event, running daily aggregation")
		return nil, handler.HandleScheduledEvent(ctx)
	})
}
