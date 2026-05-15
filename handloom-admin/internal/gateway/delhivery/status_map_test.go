package delhivery

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/handloom/admin/internal/gateway/courier"
)

func TestMapDelhiveryStatus(t *testing.T) {
	cases := []struct {
		in   string
		want courier.ShipmentEvent
	}{
		{"Manifested", courier.EventManifested},
		{"Not Picked", courier.EventManifested},
		{"In Transit", courier.EventInTransit},
		{"Dispatched", courier.EventInTransit},
		{"Out for delivery", courier.EventOutForDelivery},
		{"Delivered", courier.EventDelivered},
		{"DTO", courier.EventNDR},
		{"NDR", courier.EventNDR},
		{"RTO Initiated", courier.EventRTOInitiated},
		{"RTO Delivered", courier.EventRTODelivered},
		{"Reverse Picked Up", courier.EventReversePickedUp},
		{"Reverse Delivered", courier.EventReverseDelivered},
		{"banana", courier.EventUnknown},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := MapStatus(c.in)
			assert.Equal(t, c.want, got)
		})
	}
}
