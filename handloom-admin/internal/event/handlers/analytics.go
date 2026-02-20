package handlers

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/aws/aws-lambda-go/events"

	"github.com/handloom/admin/internal/event"
	"github.com/handloom/admin/pkg/logger"
)

// AnalyticsHandler processes analytics-aggregation events from SQS.
type AnalyticsHandler struct {
	logger *logger.Logger
}

func NewAnalyticsHandler(log *logger.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{logger: log}
}

// HandleSQSEvent is the Lambda entry point for SQS-triggered invocations.
func (h *AnalyticsHandler) HandleSQSEvent(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	var failures []events.SQSBatchItemFailure
	for _, record := range sqsEvent.Records {
		if err := h.processRecord(ctx, record); err != nil {
			h.logger.WithContext(ctx).WithError(err).Errorf("failed to process analytics event: %s", record.MessageId)
			failures = append(failures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
		}
	}
	return events.SQSEventResponse{BatchItemFailures: failures}, nil
}

func (h *AnalyticsHandler) processRecord(ctx context.Context, record events.SQSMessage) error {
	var evt event.Event
	if err := json.Unmarshal([]byte(record.Body), &evt); err != nil {
		return err
	}
	h.logger.WithContext(ctx).Infof("Processing analytics for event %s: %s", evt.Type, evt.ID)
	// TODO: implement analytics aggregation
	return nil
}

// CanHandle returns true for order, payment, product, inventory, and customer events.
func (h *AnalyticsHandler) CanHandle(t event.EventType) bool {
	return strings.HasPrefix(string(t), "order.") ||
		strings.HasPrefix(string(t), "payment.") ||
		strings.HasPrefix(string(t), "product.") ||
		strings.HasPrefix(string(t), "inventory.") ||
		strings.HasPrefix(string(t), "customer.")
}

// Handle processes a single event (used by LocalPublisher).
func (h *AnalyticsHandler) Handle(ctx context.Context, evt event.Event) error {
	h.logger.WithContext(ctx).Infof("[local] analytics handler: %s %s", evt.Type, evt.ID)
	return nil
}
