package handlers

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/aws/aws-lambda-go/events"

	"github.com/handloom/admin/internal/event"
)

// AuditHandler processes audit-log events from SQS.
type AuditHandler struct{}

func NewAuditHandler() *AuditHandler {
	return &AuditHandler{}
}

// HandleSQSEvent is the Lambda entry point for SQS-triggered invocations.
func (h *AuditHandler) HandleSQSEvent(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	var failures []events.SQSBatchItemFailure
	for _, record := range sqsEvent.Records {
		if err := h.processRecord(ctx, record); err != nil {
			slog.ErrorContext(ctx, "failed to process audit event", "message_id", record.MessageId, "error", err)
			failures = append(failures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
		}
	}
	return events.SQSEventResponse{BatchItemFailures: failures}, nil
}

func (h *AuditHandler) processRecord(ctx context.Context, record events.SQSMessage) error {
	var evt event.Event
	if err := json.Unmarshal([]byte(record.Body), &evt); err != nil {
		return err
	}
	slog.InfoContext(ctx, "Processing audit for event", "event_type", evt.Type, "event_id", evt.ID)
	// TODO: implement audit log creation
	return nil
}

// CanHandle returns true for all events — audit captures everything.
func (h *AuditHandler) CanHandle(_ event.EventType) bool {
	return true
}

// Handle processes a single event (used by LocalPublisher).
func (h *AuditHandler) Handle(ctx context.Context, evt event.Event) error {
	slog.InfoContext(ctx, "[local] audit handler", "event_type", evt.Type, "event_id", evt.ID)
	return nil
}
