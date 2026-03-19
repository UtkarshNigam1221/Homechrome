package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/aws/aws-lambda-go/events"

	"github.com/handloom/admin/internal/event"
)

// ReportHandler processes report-generation events from SQS.
type ReportHandler struct{}

func NewReportHandler() *ReportHandler {
	return &ReportHandler{}
}

// HandleSQSEvent is the Lambda entry point for SQS-triggered invocations.
func (h *ReportHandler) HandleSQSEvent(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	var failures []events.SQSBatchItemFailure
	for _, record := range sqsEvent.Records {
		if err := h.processRecord(ctx, record); err != nil {
			slog.ErrorContext(ctx, "failed to process report event", "message_id", record.MessageId, "error", err)
			failures = append(failures, events.SQSBatchItemFailure{ItemIdentifier: record.MessageId})
		}
	}
	return events.SQSEventResponse{BatchItemFailures: failures}, nil
}

func (h *ReportHandler) processRecord(ctx context.Context, record events.SQSMessage) error {
	var evt event.Event
	if err := json.Unmarshal([]byte(record.Body), &evt); err != nil {
		return err
	}
	slog.InfoContext(ctx, "Processing report for event", "event_type", evt.Type, "event_id", evt.ID)
	// TODO: implement report generation logic
	return nil
}

// CanHandle returns true for order and payment events.
func (h *ReportHandler) CanHandle(t event.EventType) bool {
	return strings.HasPrefix(string(t), "order.") ||
		strings.HasPrefix(string(t), "payment.")
}

// Handle processes a single event (used by LocalPublisher).
func (h *ReportHandler) Handle(ctx context.Context, evt event.Event) error {
	slog.InfoContext(ctx, "[local] report handler", "event_type", evt.Type, "event_id", evt.ID)
	return nil
}
