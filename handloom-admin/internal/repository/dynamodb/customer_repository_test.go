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

	// Customer and pointer are one transaction. A refused address must leave no customer
	// behind, or the guard has a hole exactly where it exists to have none.
	t.Run("creates neither the customer nor the pointer when the address is taken", func(t *testing.T) {
		require.NoError(t, repo.Create(ctx, newCustomer("cust_x", "shared@example.com", "+9120")))

		err := repo.Create(ctx, newCustomer("cust_y", "shared@example.com", "+9121"))
		require.Error(t, err)

		_, err = repo.GetByID(ctx, "cust_y")
		require.Error(t, err, "the rejected customer must not exist at all")

		// And the address still belongs to whoever held it.
		found, err := repo.GetByEmail(ctx, "shared@example.com")
		require.NoError(t, err)
		require.Equal(t, "cust_x", found.ID)
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

// The email pointer is a uniqueness guard, so leaving it behind claims a deleted
// customer's address forever — nobody could ever sign up with it again.
func TestCustomerRepository_DeleteFreesTheAddress(t *testing.T) {
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testOrdersTable)

	repo := NewCustomerRepository(wrapped)
	ctx := context.Background()

	t.Run("releases the email so it can be used again", func(t *testing.T) {
		first := &domain.Customer{ID: "cust_del_a", Email: "reuse@example.com", Phone: "+919800001111"}
		require.NoError(t, repo.Create(ctx, first))
		require.NoError(t, repo.Delete(ctx, first.ID))

		found, err := repo.GetByEmail(ctx, "reuse@example.com")
		require.Error(t, err, "the pointer must go with the customer")
		require.Nil(t, found)

		second := &domain.Customer{ID: "cust_del_b", Email: "reuse@example.com", Phone: "+919800002222"}
		require.NoError(t, repo.Create(ctx, second), "the address is free again")
	})

	t.Run("still refuses to delete a customer that is not there", func(t *testing.T) {
		require.Error(t, repo.Delete(ctx, "cust_missing"))
	})
}

// OTP signup creates a customer from a phone number alone — no email, ever,
// unless someone adds one later. Every pointer write has to tolerate that.
func TestCustomerRepository_PhoneOnlyCustomer(t *testing.T) {
	wrapped, raw := testWrappedClient(t)
	skipIfNoLocal(t, raw)
	setupTestTable(t, raw, testOrdersTable)

	repo := NewCustomerRepository(wrapped)
	ctx := context.Background()

	t.Run("creates, updates and deletes with no email at all", func(t *testing.T) {
		c := &domain.Customer{ID: "cust_otp", Phone: "+919800009001", FirstName: "OTP"}
		require.NoError(t, repo.Create(ctx, c))

		// The OTP login path flips phone_verified on an existing customer.
		c.PhoneVerified = true
		require.NoError(t, repo.Update(ctx, c), "an email-less update must not touch a pointer")

		found, err := repo.GetByPhone(ctx, "+919800009001")
		require.NoError(t, err)
		require.True(t, found.PhoneVerified)

		require.NoError(t, repo.Delete(ctx, "cust_otp"), "delete must not require an email pointer")
	})

	// The admin later fills in an address for a customer who signed up by phone.
	t.Run("claims an address added after signup", func(t *testing.T) {
		c := &domain.Customer{ID: "cust_otp2", Phone: "+919800009002", FirstName: "OTP"}
		require.NoError(t, repo.Create(ctx, c))

		c.Email = "added@example.com"
		require.NoError(t, repo.Update(ctx, c))

		found, err := repo.GetByEmail(ctx, "added@example.com")
		require.NoError(t, err)
		require.Equal(t, "cust_otp2", found.ID)
	})

	// Read-modify-write is what keeps this safe, so pin it: an update that
	// carries no email for a customer who has one would orphan the pointer.
	t.Run("keeps the pointer when an update carries the same address", func(t *testing.T) {
		c := &domain.Customer{ID: "cust_otp3", Phone: "+919800009003", Email: "keep@example.com"}
		require.NoError(t, repo.Create(ctx, c))

		c.FirstName = "Renamed"
		require.NoError(t, repo.Update(ctx, c))

		found, err := repo.GetByEmail(ctx, "keep@example.com")
		require.NoError(t, err)
		require.Equal(t, "cust_otp3", found.ID)
	})
}
