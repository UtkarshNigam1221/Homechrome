package dynamodb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// Email lookup is a pointer item rather than a GSI, which is what lets it also
// be the uniqueness guard — a GSI cannot refuse a duplicate.
func TestCustomerRepository_EmailIndex(t *testing.T) {
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testOrdersTable)

	repo := NewCustomerRepository(wrapped)
	ctx := context.Background()

	newCustomer := func(id, email, phone string) *domain.Customer {
		return &domain.Customer{ID: id, Email: email, Phone: phone, FirstName: "Test"}
	}

	t.Run("finds a customer through the pointer", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newCustomer("cust_a", "a@example.com", "+911")))

		found, err := repo.GetByEmail(ctx, "a@example.com")
		require.NoError(t, err)
		require.Equal(t, "cust_a", found.ID)
	})

	t.Run("refuses an address another customer holds", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newCustomer("cust_b", "b@example.com", "+912")))

		err := repo.Create(ctx, newCustomer("cust_c", "b@example.com", "+913"))
		require.Error(t, err, "two customers must not share an address")

		appErr, ok := errors.AsAppError(err)
		require.True(t, ok)
		require.Equal(t, errors.ErrCodeAlreadyExists, appErr.Code)
	})

	// Storefront signup is phone-OTP, so most customers have no email. They must
	// not get a pointer item, and must not collide with each other.
	t.Run("writes no pointer for a customer without an email", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newCustomer("cust_d", "", "+914")))
		require.NoError(t, repo.Create(ctx, newCustomer("cust_e", "", "+915")))

		found, err := repo.GetByID(ctx, "cust_e")
		require.NoError(t, err)
		require.Empty(t, found.Email)
	})

	t.Run("reports no customer for an unknown address", func(t *testing.T) {
		_, err := repo.GetByEmail(ctx, "nobody@example.com")
		require.Error(t, err)
	})

	t.Run("moves the pointer when the address changes", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newCustomer("cust_f", "old@example.com", "+916")))

		updated := newCustomer("cust_f", "new@example.com", "+916")
		require.NoError(t, repo.Update(ctx, updated))

		found, err := repo.GetByEmail(ctx, "new@example.com")
		require.NoError(t, err)
		require.Equal(t, "cust_f", found.ID)

		_, err = repo.GetByEmail(ctx, "old@example.com")
		require.Error(t, err, "the old address must be released, not left pointing at the customer")
	})

	t.Run("refuses an update onto an address someone else holds", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newCustomer("cust_g", "g@example.com", "+917")))
		require.NoError(t, repo.Create(ctx, newCustomer("cust_h", "h@example.com", "+918")))

		err := repo.Update(ctx, newCustomer("cust_h", "g@example.com", "+918"))
		require.Error(t, err)

		// The record itself must be untouched by the refusal.
		found, getErr := repo.GetByID(ctx, "cust_h")
		require.NoError(t, getErr)
		require.Equal(t, "h@example.com", found.Email)
	})

	// Re-saving without changing the address must not trip the guard.
	t.Run("allows a customer to keep its own address", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newCustomer("cust_i", "i@example.com", "+919")))

		same := newCustomer("cust_i", "i@example.com", "+919")
		same.FirstName = "Renamed"
		require.NoError(t, repo.Update(ctx, same))

		found, err := repo.GetByEmail(ctx, "i@example.com")
		require.NoError(t, err)
		require.Equal(t, "Renamed", found.FirstName)
	})
}
