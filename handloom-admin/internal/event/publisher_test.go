package event

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/pkg/logger"
)

// ---------------------------------------------------------------------------
// TestNew
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	type payload struct {
		OrderID string `json:"order_id"`
		Total   int    `json:"total"`
	}

	data := payload{OrderID: "ORD-123", Total: 5000}
	evt := New(OrderCreated, data)

	assert.NotEmpty(t, evt.ID, "ID should be a non-empty UUID")
	assert.Equal(t, OrderCreated, evt.Type)
	assert.Equal(t, "handloom-api", evt.Source)
	assert.False(t, evt.Timestamp.IsZero(), "Timestamp should be set")

	var decoded payload
	require.NoError(t, json.Unmarshal(evt.Data, &decoded))
	assert.Equal(t, data, decoded)
}

// ---------------------------------------------------------------------------
// spyHandler — test double that records calls
// ---------------------------------------------------------------------------

type spyHandler struct {
	types  []EventType
	events []Event
}

func (s *spyHandler) CanHandle(et EventType) bool {
	for _, t := range s.types {
		if t == et {
			return true
		}
	}
	return false
}

func (s *spyHandler) Handle(_ context.Context, evt Event) error {
	s.events = append(s.events, evt)
	return nil
}

// ---------------------------------------------------------------------------
// TestLocalPublisher_DispatchesToMatchingHandlers
// ---------------------------------------------------------------------------

func TestLocalPublisher_DispatchesToMatchingHandlers(t *testing.T) {
	orderHandler := &spyHandler{types: []EventType{OrderCreated, OrderCancelled}}
	paymentHandler := &spyHandler{types: []EventType{PaymentReceived}}

	pub := NewLocalPublisher(logger.NewNoop(), orderHandler, paymentHandler)

	ctx := context.Background()

	// Publish an order event — only orderHandler should fire.
	orderEvt := New(OrderCreated, map[string]string{"id": "ORD-1"})
	err := pub.Publish(ctx, orderEvt)
	require.NoError(t, err)

	assert.Len(t, orderHandler.events, 1)
	assert.Equal(t, orderEvt.ID, orderHandler.events[0].ID)
	assert.Empty(t, paymentHandler.events, "paymentHandler should not be called for order events")

	// Publish a payment event — only paymentHandler should fire.
	payEvt := New(PaymentReceived, map[string]string{"id": "PAY-1"})
	err = pub.Publish(ctx, payEvt)
	require.NoError(t, err)

	assert.Len(t, paymentHandler.events, 1)
	assert.Equal(t, payEvt.ID, paymentHandler.events[0].ID)
	assert.Len(t, orderHandler.events, 1, "orderHandler should still have only 1 event")

	// Publish an event that no handler cares about.
	err = pub.Publish(ctx, New(ProductCreated, nil))
	require.NoError(t, err)

	assert.Len(t, orderHandler.events, 1)
	assert.Len(t, paymentHandler.events, 1)
}

// ---------------------------------------------------------------------------
// TestNoopPublisher
// ---------------------------------------------------------------------------

func TestNoopPublisher(t *testing.T) {
	pub := NewNoopPublisher()
	err := pub.Publish(context.Background(), New(OrderCreated, nil))
	assert.NoError(t, err)
}
