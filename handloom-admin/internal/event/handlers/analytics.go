package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/event"
	"github.com/handloom/admin/internal/service"
)

// AnalyticsHandler processes analytics-aggregation events from SQS and
// scheduled daily aggregation from EventBridge.
// It writes raw events to the events table, updates live dashboard counters,
// and runs daily aggregation on a schedule.
type AnalyticsHandler struct {
	eventsRepo    domain.EventsRepository
	analyticsRepo domain.AnalyticsRepository
	aggregator    *service.AnalyticsAggregator
}

func NewAnalyticsHandler(
	eventsRepo domain.EventsRepository,
	analyticsRepo domain.AnalyticsRepository,
	aggregator *service.AnalyticsAggregator,
) *AnalyticsHandler {
	return &AnalyticsHandler{
		eventsRepo:    eventsRepo,
		analyticsRepo: analyticsRepo,
		aggregator:    aggregator,
	}
}

// HandleScheduledEvent runs daily aggregation for the previous day.
// It is invoked by EventBridge on a cron schedule (00:30 UTC daily).
func (h *AnalyticsHandler) HandleScheduledEvent(ctx context.Context) error {
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	slog.Info("Running daily aggregation", "date", yesterday)
	return h.aggregator.AggregateDate(ctx, yesterday)
}

// HandleSQSEvent is the Lambda entry point for SQS-triggered invocations.
func (h *AnalyticsHandler) HandleSQSEvent(ctx context.Context, sqsEvent events.SQSEvent) (events.SQSEventResponse, error) {
	var failures []events.SQSBatchItemFailure
	for _, record := range sqsEvent.Records {
		if err := h.processRecord(ctx, record); err != nil {
			slog.ErrorContext(ctx, "failed to process analytics event", "message_id", record.MessageId, "error", err)
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
	slog.InfoContext(ctx, "Processing analytics for event", "event_type", evt.Type, "event_id", evt.ID)

	h.storeRawEvent(ctx, evt)
	h.updateCounters(ctx, evt)

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
	slog.InfoContext(ctx, "[local] analytics handler", "event_type", evt.Type, "event_id", evt.ID)

	h.storeRawEvent(ctx, evt)
	h.updateCounters(ctx, evt)

	return nil
}

// storeRawEvent converts the backend event into a StoreEvent and writes it to the events table.
// This is best-effort — errors are logged but not propagated.
func (h *AnalyticsHandler) storeRawEvent(ctx context.Context, evt event.Event) {
	// Convert event data to properties map
	var props map[string]interface{}
	if len(evt.Data) > 0 {
		if err := json.Unmarshal(evt.Data, &props); err != nil {
			// If we can't unmarshal as a map, store the raw JSON string
			props = map[string]interface{}{"raw": string(evt.Data)}
		}
	}

	storeEvt := domain.StoreEvent{
		EventType:  string(evt.Type),
		Timestamp:  evt.Timestamp,
		SessionID:  "backend",
		VisitorID:  "backend",
		DeviceType: "server",
		PagePath:   "",
		Properties: props,
	}

	if err := h.eventsRepo.BatchWriteEvents(ctx, []domain.StoreEvent{storeEvt}); err != nil {
		slog.ErrorContext(ctx, "failed to write raw event", "event_type", evt.Type, "error", err)
	}
}

// orderData is a minimal struct to extract TotalAmount from order event payloads.
type orderData struct {
	TotalAmount int64 `json:"total_amount"`
}

// updateCounters updates live dashboard counters based on event type.
// All operations are best-effort — errors are logged but not propagated.
func (h *AnalyticsHandler) updateCounters(ctx context.Context, evt event.Event) {
	switch evt.Type {
	case event.OrderCreated:
		// Increment today_orders by 1
		if err := h.analyticsRepo.IncrementDashboardCounter(ctx, "today_orders", 1); err != nil {
			slog.ErrorContext(ctx, "failed to increment today_orders", "error", err)
		}
		// Increment today_revenue by order total amount
		var od orderData
		if err := json.Unmarshal(evt.Data, &od); err == nil && od.TotalAmount > 0 {
			if err := h.analyticsRepo.IncrementDashboardCounter(ctx, "today_revenue", od.TotalAmount); err != nil {
				slog.ErrorContext(ctx, "failed to increment today_revenue", "error", err)
			}
		}

	case event.PaymentReceived:
		if err := h.analyticsRepo.IncrementDashboardCounter(ctx, "today_payments_success", 1); err != nil {
			slog.ErrorContext(ctx, "failed to increment today_payments_success", "error", err)
		}

	case event.PaymentFailed:
		if err := h.analyticsRepo.IncrementDashboardCounter(ctx, "today_payments_failed", 1); err != nil {
			slog.ErrorContext(ctx, "failed to increment today_payments_failed", "error", err)
		}

	case event.CustomerRegistered:
		if err := h.analyticsRepo.IncrementDashboardCounter(ctx, "today_new_customers", 1); err != nil {
			slog.ErrorContext(ctx, "failed to increment today_new_customers", "error", err)
		}
	}
}
