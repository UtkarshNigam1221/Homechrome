package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/aws/aws-lambda-go/events"

	"github.com/handloom/admin/internal/event"
)

// NotificationHandler processes notification events from SQS.
type NotificationHandler struct{}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{}
}

// HandleSQSEvent is the Lambda entry point for SQS-triggered invocations.
func (h *NotificationHandler) HandleSQSEvent(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	var failures []events.SQSBatchItemFailure
	for _, record := range sqsEvent.Records {
		if err := h.processRecord(ctx, record); err != nil {
			slog.ErrorContext(ctx, "failed to process notification event", "message_id", record.MessageId, "error", err)
			failures = append(failures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
		}
	}
	return events.SQSEventResponse{BatchItemFailures: failures}, nil
}

func (h *NotificationHandler) processRecord(ctx context.Context, record events.SQSMessage) error {
	var evt event.Event
	if err := json.Unmarshal([]byte(record.Body), &evt); err != nil {
		return err
	}
	slog.InfoContext(ctx, "Processing notification for event", "event_type", evt.Type, "event_id", evt.ID)
	// TODO: implement actual notification sending (SMS/email) based on event type
	return nil
}

// CanHandle returns true for events the notification worker cares about.
func (h *NotificationHandler) CanHandle(t event.EventType) bool {
	return strings.HasPrefix(string(t), "order.") ||
		strings.HasPrefix(string(t), "payment.") ||
		strings.HasPrefix(string(t), "shipment.") ||
		t == event.CustomerRegistered
}

// Handle processes a single event (used by LocalPublisher).
func (h *NotificationHandler) Handle(ctx context.Context, evt event.Event) error {
	slog.InfoContext(ctx, "[local] notification handler", "event_type", evt.Type, "event_id", evt.ID)
	// TODO: implement actual notification sending for local dev
	return nil
}
