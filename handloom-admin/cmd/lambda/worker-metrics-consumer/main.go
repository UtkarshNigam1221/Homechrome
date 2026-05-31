package main

import (
	"context"
	"log/slog"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/handloom/admin/internal/bootstrap"
	"github.com/handloom/admin/internal/wire"
)

func main() {
	bc := bootstrap.InitLambda("handloom-metrics-consumer")
	defer bc.Shutdown()

	ctx := context.Background()
	deps, err := wire.InitializeMetricsConsumerDeps(ctx, bc.Cfg)
	if err != nil {
		slog.Error("init deps failed", "error", err)
		panic(err)
	}

	handler := func(ctx context.Context, evt events.SQSEvent) (events.SQSEventResponse, error) {
		return deps.Handler.HandleSQSEvent(ctx, evt)
	}
	lambda.Start(handler)
}
