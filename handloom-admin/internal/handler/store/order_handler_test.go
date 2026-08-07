package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/handloom/admin/internal/domain"
)

// Internal notes are admin-only. domain.Order serializes them, and every
// customer-facing route hands back a *domain.Order, so the strip is the only
// thing keeping staff notes out of the customer's API response.
func TestStripInternal_RemovesAdminNotes(t *testing.T) {
	order := &domain.Order{
		OrderNumber: "HC-123",
		InternalNotes: []domain.OrderNote{
			{ID: "n1", Note: "customer haggled, watch this one", IsInternal: true, CreatedAt: time.Now()},
		},
	}

	stripInternal(order)

	assert.Nil(t, order.InternalNotes)
	assert.Equal(t, "HC-123", order.OrderNumber, "unrelated fields must survive")
}

func TestStripInternal_NilSafe(t *testing.T) {
	assert.NotPanics(t, func() { stripInternal(nil) })
}
