package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/event"
	"github.com/handloom/admin/pkg/logger"
)

func TestNotificationHandler_CanHandle(t *testing.T) {
	h := &NotificationHandler{}
	assert.True(t, h.CanHandle(event.OrderCreated))
	assert.True(t, h.CanHandle(event.PaymentReceived))
	assert.True(t, h.CanHandle(event.ShipmentCreated))
	assert.True(t, h.CanHandle(event.CustomerRegistered))
	assert.False(t, h.CanHandle(event.ProductCreated))
	assert.False(t, h.CanHandle(event.AdminEntityModified))
}

func TestReportHandler_CanHandle(t *testing.T) {
	h := &ReportHandler{}
	assert.True(t, h.CanHandle(event.OrderCreated))
	assert.True(t, h.CanHandle(event.PaymentReceived))
	assert.False(t, h.CanHandle(event.ShipmentCreated))
	assert.False(t, h.CanHandle(event.ProductCreated))
}

func TestAnalyticsHandler_CanHandle(t *testing.T) {
	h := &AnalyticsHandler{}
	assert.True(t, h.CanHandle(event.OrderCreated))
	assert.True(t, h.CanHandle(event.PaymentReceived))
	assert.True(t, h.CanHandle(event.ProductCreated))
	assert.True(t, h.CanHandle(event.InventoryLowStock))
	assert.True(t, h.CanHandle(event.CustomerRegistered))
	assert.False(t, h.CanHandle(event.AdminEntityModified))
}

func TestAuditHandler_CanHandle_AllEvents(t *testing.T) {
	h := &AuditHandler{}
	assert.True(t, h.CanHandle(event.OrderCreated))
	assert.True(t, h.CanHandle(event.ProductDeleted))
	assert.True(t, h.CanHandle(event.AdminEntityModified))
}

func TestNotificationHandler_HandleSQSEvent(t *testing.T) {
	log := logger.NewNoop()
	h := NewNotificationHandler(log)

	evt := event.New(event.OrderCreated, map[string]string{"order_id": "ord_123"})
	body, _ := json.Marshal(evt)

	sqsEvent := events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "msg-1", Body: string(body)},
		},
	}

	resp, err := h.HandleSQSEvent(context.Background(), sqsEvent)
	require.NoError(t, err)
	assert.Empty(t, resp.BatchItemFailures)
}

func TestNotificationHandler_HandleSQSEvent_InvalidJSON(t *testing.T) {
	log := logger.NewNoop()
	h := NewNotificationHandler(log)

	sqsEvent := events.SQSEvent{
		Records: []events.SQSMessage{
			{MessageId: "msg-bad", Body: "not json"},
		},
	}

	resp, err := h.HandleSQSEvent(context.Background(), sqsEvent)
	require.NoError(t, err)
	assert.Len(t, resp.BatchItemFailures, 1)
	assert.Equal(t, "msg-bad", resp.BatchItemFailures[0].ItemIdentifier)
}
