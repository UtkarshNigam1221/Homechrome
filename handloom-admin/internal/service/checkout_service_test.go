package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
)

func TestFindShippingAddress(t *testing.T) {
	customer := &domain.Customer{
		Addresses: []domain.Address{
			{ID: "addr-1", City: "Bengaluru"},
			{ID: "addr-2", City: "Mumbai"},
		},
	}

	t.Run("found", func(t *testing.T) {
		got, err := findShippingAddress(customer, "addr-2")
		require.NoError(t, err)
		assert.Equal(t, "Mumbai", got.City)
	})

	t.Run("not found returns NotFound", func(t *testing.T) {
		_, err := findShippingAddress(customer, "missing")
		require.Error(t, err)
	})

	t.Run("empty address list", func(t *testing.T) {
		_, err := findShippingAddress(&domain.Customer{}, "addr-1")
		require.Error(t, err)
	})
}

func TestCartItemsToOrderItems(t *testing.T) {
	t.Run("maps fields and assigns unique ids", func(t *testing.T) {
		items := []domain.CartItem{
			{ProductID: "p1", ProductName: "Saree", Quantity: 2, UnitPrice: 100, TotalPrice: 200},
			{ProductID: "p2", ProductName: "Dupatta", Quantity: 1, UnitPrice: 50, TotalPrice: 50},
		}

		got := cartItemsToOrderItems(items)

		require.Len(t, got, 2)
		assert.Equal(t, "p1", got[0].ProductID)
		assert.Equal(t, "Saree", got[0].ProductName)
		assert.Equal(t, 2, got[0].Quantity)
		assert.Equal(t, int64(200), got[0].TotalPrice)

		// Each order item gets a fresh, non-empty, unique ID.
		assert.NotEmpty(t, got[0].ID)
		assert.NotEmpty(t, got[1].ID)
		assert.NotEqual(t, got[0].ID, got[1].ID)
	})

	t.Run("empty input yields empty non-nil slice", func(t *testing.T) {
		got := cartItemsToOrderItems(nil)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
}
