package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/repository/postgres"
)

// seedInventory inserts an inventory row directly, bypassing product-create
// semantics so the starting numbers are explicit.
func seedInventory(t *testing.T, pool *pgxpool.Pool, productID string, quantity, reserved int) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO inventory (id, product_id, quantity, reserved_qty, available_qty)
		 VALUES ($1, $2, $3, $4, $5)`,
		uuid.New().String(), productID, quantity, reserved, quantity-reserved,
	)
	require.NoError(t, err)
}

// readInventory returns the current quantity, reserved_qty and available_qty.
func readInventory(t *testing.T, pool *pgxpool.Pool, productID string) (qty, reserved, available int) {
	t.Helper()
	err := pool.QueryRow(context.Background(),
		`SELECT quantity, reserved_qty, available_qty FROM inventory WHERE product_id = $1`,
		productID,
	).Scan(&qty, &reserved, &available)
	require.NoError(t, err)
	return
}

func TestInventoryRepository_CommitStock(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewInventoryRepository(pool)
	ctx := context.Background()
	category := seedCategory(t, pool)

	newProduct := func(t *testing.T, quantity, reserved int) string {
		t.Helper()
		p := newTestProduct(category.ID)
		require.NoError(t, postgres.NewProductRepository(pool).Create(ctx, p, nil))
		t.Cleanup(func() {
			_, _ = pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, p.ID)
		})
		_, err := pool.Exec(ctx, `DELETE FROM inventory WHERE product_id = $1`, p.ID)
		require.NoError(t, err)
		seedInventory(t, pool, p.ID, quantity, reserved)
		return p.ID
	}

	t.Run("commit decrements quantity and reserved, leaving available unchanged", func(t *testing.T) {
		productID := newProduct(t, 100, 10)

		txn, err := repo.CommitStock(ctx, productID, 4, "order_abc")
		require.NoError(t, err)

		qty, reserved, available := readInventory(t, pool, productID)
		require.Equal(t, 96, qty)
		require.Equal(t, 6, reserved)
		require.Equal(t, 90, available, "available_qty must not move on commit")

		require.Equal(t, domain.InventoryTransactionTypeCommit, txn.Type)
		require.Equal(t, 4, txn.Quantity)
		require.Equal(t, 100, txn.PreviousQty)
		require.Equal(t, 96, txn.NewQty)
		require.Equal(t, "order_abc", txn.ReferenceID)
	})

	t.Run("commit writes exactly one ledger row", func(t *testing.T) {
		productID := newProduct(t, 50, 5)

		_, err := repo.CommitStock(ctx, productID, 2, "order_def")
		require.NoError(t, err)

		var count int
		require.NoError(t, pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM inventory_transactions WHERE product_id = $1 AND type = $2`,
			productID, string(domain.InventoryTransactionTypeCommit),
		).Scan(&count))
		require.Equal(t, 1, count)
	})

	t.Run("commit beyond reserved is rejected and changes nothing", func(t *testing.T) {
		productID := newProduct(t, 100, 3)

		_, err := repo.CommitStock(ctx, productID, 5, "order_ghi")
		require.Error(t, err)
		require.ErrorContains(t, err, "insufficient stock")

		qty, reserved, available := readInventory(t, pool, productID)
		require.Equal(t, 100, qty)
		require.Equal(t, 3, reserved)
		require.Equal(t, 97, available)
	})

	t.Run("commit on a missing product returns not found", func(t *testing.T) {
		_, err := repo.CommitStock(ctx, "does-not-exist", 1, "order_jkl")
		require.Error(t, err)
	})
}
