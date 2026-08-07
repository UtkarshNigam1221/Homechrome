package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
)

// domain.Order serializes the note slice, and every customer-facing route hands
// back a *domain.Order, so this strip is the only thing keeping staff notes out
// of the customer's API response.
func TestStripInternal_KeepsOnlyCustomerFacingNotes(t *testing.T) {
	order := &domain.Order{
		OrderNumber: "HC-123",
		InternalNotes: []domain.OrderNote{
			{ID: "n1", Note: "customer haggled, watch this one", IsInternal: true, CreatedAt: time.Now()},
			{ID: "n2", Note: "Dispatching Tuesday", IsInternal: false, CreatedAt: time.Now()},
			{ID: "n3", Note: "do not refund", IsInternal: true, CreatedAt: time.Now()},
			{ID: "n4", Note: "Gift wrapped as requested", IsInternal: false, CreatedAt: time.Now()},
		},
	}

	stripInternal(order)

	// Order preserved, and nothing marked internal survives — including the one
	// sandwiched between two shareable notes.
	require.Len(t, order.InternalNotes, 2)
	assert.Equal(t, "Dispatching Tuesday", order.InternalNotes[0].Note)
	assert.Equal(t, "Gift wrapped as requested", order.InternalNotes[1].Note)
	for _, n := range order.InternalNotes {
		assert.False(t, n.IsInternal)
	}
	assert.Equal(t, "HC-123", order.OrderNumber, "unrelated fields must survive")
}

func TestStripInternal_AllInternalLeavesNothing(t *testing.T) {
	order := &domain.Order{
		InternalNotes: []domain.OrderNote{{ID: "n1", Note: "private", IsInternal: true}},
	}

	stripInternal(order)

	assert.Empty(t, order.InternalNotes)
}

func TestStripInternal_NilSafe(t *testing.T) {
	assert.NotPanics(t, func() { stripInternal(nil) })
}
