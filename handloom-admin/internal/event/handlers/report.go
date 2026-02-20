package handlers

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/aws/aws-lambda-go/events"

	"github.com/handloom/admin/internal/event"
	"github.com/handloom/admin/pkg/logger"
)

// ReportHandler processes report-generation events from SQS.
type ReportHandler struct {
	logger *logger.Logger
}

func NewReportHandler(log *logger.Logger) *ReportHandler {
	return &ReportHandler{logger: log}
}

// HandleSQSEvent is the Lambda entry point for SQS-triggered invocations.
func (h *ReportHandler) HandleSQSEvent(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	var failures []events.SQSBatchItemFailure
	for _, record := range sqsEvent.Records {
		if err := h.processRecord(ctx, record); err != nil {
			h.logger.WithContext(ctx).WithError(err).Errorf("failed to process report event: %s", record.MessageId)
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
	h.logger.WithContext(ctx).Infof("Processing report for event %s: %s", evt.Type, evt.ID)
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
	h.logger.WithContext(ctx).Infof("[local] report handler: %s %s", evt.Type, evt.ID)
	return nil
}
