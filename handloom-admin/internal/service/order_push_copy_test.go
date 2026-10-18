package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
)

// Named for the status it carries: `order` alone would shadow the local
// variable of that name used throughout package service.
func orderWithStatus(status domain.OrderStatus) *domain.Order {
	return &domain.Order{ID: "order_1", OrderNumber: "HL-1042", Status: status}
}

func TestOrderStatusPushCopy(t *testing.T) {
	t.Run("shipped tells the shopper their parcel is moving", func(t *testing.T) {
		payload := orderStatusPush(orderWithStatus(domain.OrderStatusShipped))
		require.NotNil(t, payload)
		require.Contains(t, payload.Title, "HL-1042")
		require.NotEmpty(t, payload.Body)
		require.Equal(t, "/account/orders/order_1", payload.URL)
	})

	t.Run("delivered and cancelled are both worth a notification", func(t *testing.T) {
		require.NotNil(t, orderStatusPush(orderWithStatus(domain.OrderStatusDelivered)))
		require.NotNil(t, orderStatusPush(orderWithStatus(domain.OrderStatusCancelled)))
	})

	// Interrupting someone for a state they never see is how an opt-in gets
	// revoked. Only the states a shopper is actually waiting on send.
	t.Run("internal states send nothing", func(t *testing.T) {
		require.Nil(t, orderStatusPush(orderWithStatus(domain.OrderStatusPending)))
		require.Nil(t, orderStatusPush(orderWithStatus(domain.OrderStatusConfirmed)))
		require.Nil(t, orderStatusPush(orderWithStatus(domain.OrderStatusProcessing)))
		require.Nil(t, orderStatusPush(orderWithStatus(domain.OrderStatusReturned)))
	})

	t.Run("every notification is tagged per order so a second one replaces the first", func(t *testing.T) {
		shipped := orderStatusPush(orderWithStatus(domain.OrderStatusShipped))
		delivered := orderStatusPush(orderWithStatus(domain.OrderStatusDelivered))
		require.Equal(t, shipped.Tag, delivered.Tag)
		require.Contains(t, shipped.Tag, "order_1")
	})
}
