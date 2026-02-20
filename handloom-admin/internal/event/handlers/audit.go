package handlers

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"

	"github.com/handloom/admin/internal/event"
	"github.com/handloom/admin/pkg/logger"
)

// AuditHandler processes audit-log events from SQS.
type AuditHandler struct {
	logger *logger.Logger
}

func NewAuditHandler(log *logger.Logger) *AuditHandler {
	return &AuditHandler{logger: log}
}

// HandleSQSEvent is the Lambda entry point for SQS-triggered invocations.
func (h *AuditHandler) HandleSQSEvent(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	var failures []events.SQSBatchItemFailure
	for _, record := range sqsEvent.Records {
		if err := h.processRecord(ctx, record); err != nil {
			h.logger.WithContext(ctx).WithError(err).Errorf("failed to process audit event: %s", record.MessageId)
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
	h.logger.WithContext(ctx).Infof("Processing audit for event %s: %s", evt.Type, evt.ID)
	// TODO: implement audit log creation
	return nil
}

// CanHandle returns true for all events — audit captures everything.
func (h *AuditHandler) CanHandle(_ event.EventType) bool {
	return true
}

// Handle processes a single event (used by LocalPublisher).
func (h *AuditHandler) Handle(ctx context.Context, evt event.Event) error {
	h.logger.WithContext(ctx).Infof("[local] audit handler: %s %s", evt.Type, evt.ID)
	return nil
}
