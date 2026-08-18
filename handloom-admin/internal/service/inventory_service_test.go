package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
)

// created_by holds an opaque user id, which tells a reader nothing. The ledger
// read resolves it to a name so the history is legible.
func TestInventoryService_GetTransactions_ResolvesActorNames(t *testing.T) {
	ctx := context.Background()

	newService := func(t *testing.T) (*InventoryService, *mocks.MockInventoryRepository, *mocks.MockUserRepository) {
		t.Helper()
		ctrl := gomock.NewController(t)
		inventoryRepo := mocks.NewMockInventoryRepository(ctrl)
		userRepo := mocks.NewMockUserRepository(ctrl)
		return NewInventoryService(inventoryRepo, userRepo), inventoryRepo, userRepo
	}

	respond := func(txns ...*domain.InventoryTransaction) *domain.ListInventoryTransactionsResponse {
		return &domain.ListInventoryTransactionsResponse{Transactions: txns}
	}

	t.Run("looks a user up once however many movements they made", func(t *testing.T) {
		svc, inventoryRepo, userRepo := newService(t)

		inventoryRepo.EXPECT().GetTransactions(gomock.Any(), "prod_1", gomock.Any()).
			Return(respond(
				&domain.InventoryTransaction{ID: "t1", CreatedBy: "usr_1"},
				&domain.InventoryTransaction{ID: "t2", CreatedBy: "usr_1"},
			), nil)

		// Exactly once, not once per row.
		userRepo.EXPECT().GetByID(gomock.Any(), "usr_1").
			Return(&domain.User{ID: "usr_1", FirstName: "Asha", LastName: "Rao"}, nil).
			Times(1)

		result, err := svc.GetTransactions(ctx, "prod_1", domain.PaginationRequest{})
		require.NoError(t, err)
		require.Equal(t, "Asha Rao", result.Transactions[0].CreatedByName)
		require.Equal(t, "Asha Rao", result.Transactions[1].CreatedByName)
	})

	t.Run("leaves order-driven movements without an actor", func(t *testing.T) {
		svc, inventoryRepo, _ := newService(t)

		inventoryRepo.EXPECT().GetTransactions(gomock.Any(), "prod_1", gomock.Any()).
			Return(respond(&domain.InventoryTransaction{ID: "t1", CreatedBy: ""}), nil)

		result, err := svc.GetTransactions(ctx, "prod_1", domain.PaginationRequest{})
		require.NoError(t, err)
		require.Empty(t, result.Transactions[0].CreatedByName, "no admin stands behind a reservation")
	})

	t.Run("falls back to the email when the user has no name", func(t *testing.T) {
		svc, inventoryRepo, userRepo := newService(t)

		inventoryRepo.EXPECT().GetTransactions(gomock.Any(), "prod_1", gomock.Any()).
			Return(respond(&domain.InventoryTransaction{ID: "t1", CreatedBy: "usr_1"}), nil)
		userRepo.EXPECT().GetByID(gomock.Any(), "usr_1").
			Return(&domain.User{ID: "usr_1", Email: "ops@handloom.in"}, nil)

		result, err := svc.GetTransactions(ctx, "prod_1", domain.PaginationRequest{})
		require.NoError(t, err)
		require.Equal(t, "ops@handloom.in", result.Transactions[0].CreatedByName)
	})

	// The history is worth reading without the names; a directory failure must
	// not take the whole endpoint down with it.
	t.Run("still returns the history when a lookup fails", func(t *testing.T) {
		svc, inventoryRepo, userRepo := newService(t)

		inventoryRepo.EXPECT().GetTransactions(gomock.Any(), "prod_1", gomock.Any()).
			Return(respond(&domain.InventoryTransaction{ID: "t1", CreatedBy: "usr_gone"}), nil)
		userRepo.EXPECT().GetByID(gomock.Any(), "usr_gone").
			Return(nil, errors.NotFound("User not found"))

		result, err := svc.GetTransactions(ctx, "prod_1", domain.PaginationRequest{})
		require.NoError(t, err)
		require.Len(t, result.Transactions, 1)
		require.Empty(t, result.Transactions[0].CreatedByName)
	})

	// The storefront builds this service for stock levels and wires no directory.
	t.Run("works without a user directory at all", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		inventoryRepo := mocks.NewMockInventoryRepository(ctrl)
		inventoryRepo.EXPECT().GetTransactions(gomock.Any(), "prod_1", gomock.Any()).
			Return(respond(&domain.InventoryTransaction{ID: "t1", CreatedBy: "usr_1"}), nil)

		svc := NewInventoryService(inventoryRepo, nil)
		result, err := svc.GetTransactions(ctx, "prod_1", domain.PaginationRequest{})
		require.NoError(t, err)
		require.Empty(t, result.Transactions[0].CreatedByName)
	})
}
