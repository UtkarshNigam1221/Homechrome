package handlers

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/aws/aws-lambda-go/events"

	"github.com/handloom/admin/internal/event"
	"github.com/handloom/admin/pkg/logger"
)

// NotificationHandler processes notification events from SQS.
type NotificationHandler struct {
	logger *logger.Logger
}

func NewNotificationHandler(log *logger.Logger) *NotificationHandler {
	return &NotificationHandler{logger: log}
}

// HandleSQSEvent is the Lambda entry point for SQS-triggered invocations.
func (h *NotificationHandler) HandleSQSEvent(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	var failures []events.SQSBatchItemFailure
	for _, record := range sqsEvent.Records {
		if err := h.processRecord(ctx, record); err != nil {
			h.logger.WithContext(ctx).WithError(err).Errorf("failed to process notification event: %s", record.MessageId)
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
	h.logger.WithContext(ctx).Infof("Processing notification for event %s: %s", evt.Type, evt.ID)
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
	h.logger.WithContext(ctx).Infof("[local] notification handler: %s %s", evt.Type, evt.ID)
	// TODO: implement actual notification sending for local dev
	return nil
}
