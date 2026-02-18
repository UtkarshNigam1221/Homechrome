package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newBulkServiceTest(t *testing.T) (*BulkService, *mocks.MockBulkOperationRepository) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	bulkRepo := mocks.NewMockBulkOperationRepository(ctrl)
	log := logger.NewNoop()
	svc := NewBulkService(bulkRepo, nil, nil, log)
	return svc, bulkRepo
}

// ---------------------------------------------------------------------------
// TestBulkService_CreateOperation
// ---------------------------------------------------------------------------

func TestBulkService_CreateOperation(t *testing.T) {
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)

		bulkRepo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, op *domain.BulkOperation) error {
			assert.Contains(t, op.ID, "bulk_")
			assert.Equal(t, domain.BulkOperationTypeImport, op.Type)
			assert.Equal(t, domain.BulkOperationEntityProduct, op.EntityType)
			assert.Equal(t, domain.BulkOperationStatusPending, op.Status)
			assert.Equal(t, "admin_1", op.CreatedBy)
			assert.Equal(t, "s3://bucket/file.csv", op.InputFileURL)
			return nil
		})

		op, err := svc.CreateOperation(ctx, domain.CreateBulkOperationRequest{
			Type:         domain.BulkOperationTypeImport,
			EntityType:   domain.BulkOperationEntityProduct,
			InputFileURL: "s3://bucket/file.csv",
		}, "admin_1")

		require.NoError(t, err)
		require.NotNil(t, op)
		assert.Contains(t, op.ID, "bulk_")
		assert.Equal(t, domain.BulkOperationStatusPending, op.Status)
	})

	t.Run("repo error on create", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)
		bulkRepo.EXPECT().Create(ctx, gomock.Any()).Return(errors.Internal("db error"))

		op, err := svc.CreateOperation(ctx, domain.CreateBulkOperationRequest{
			Type:       domain.BulkOperationTypeImport,
			EntityType: domain.BulkOperationEntityProduct,
		}, "admin_1")

		require.Error(t, err)
		assert.Nil(t, op)
	})
}

// ---------------------------------------------------------------------------
// TestBulkService_GetByID
// ---------------------------------------------------------------------------

func TestBulkService_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("successful get", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)
		expected := &domain.BulkOperation{
			ID:     "bulk_abc123",
			Type:   domain.BulkOperationTypeImport,
			Status: domain.BulkOperationStatusCompleted,
		}
		bulkRepo.EXPECT().GetByID(ctx, "bulk_abc123").Return(expected, nil)

		op, err := svc.GetByID(ctx, "bulk_abc123")
		require.NoError(t, err)
		assert.Equal(t, "bulk_abc123", op.ID)
	})

	t.Run("not found", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)
		bulkRepo.EXPECT().GetByID(ctx, "bulk_notfound").Return(nil, errors.NotFound("BulkOperation"))

		op, err := svc.GetByID(ctx, "bulk_notfound")
		require.Error(t, err)
		assert.Nil(t, op)
	})
}

// ---------------------------------------------------------------------------
// TestBulkService_List
// ---------------------------------------------------------------------------

func TestBulkService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)
		expected := &domain.ListBulkOperationsResponse{
			Operations: []*domain.BulkOperation{
				{ID: "bulk_1", Status: domain.BulkOperationStatusCompleted},
				{ID: "bulk_2", Status: domain.BulkOperationStatusPending},
			},
			Pagination: domain.PaginationResponse{Limit: 20},
		}
		bulkRepo.EXPECT().List(ctx, gomock.Any()).Return(expected, nil)

		resp, err := svc.List(ctx, domain.ListBulkOperationsRequest{})
		require.NoError(t, err)
		assert.Len(t, resp.Operations, 2)
	})
}

// ---------------------------------------------------------------------------
// TestBulkService_GetByUser
// ---------------------------------------------------------------------------

func TestBulkService_GetByUser(t *testing.T) {
	ctx := context.Background()

	t.Run("successful get by user", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)
		expected := &domain.ListBulkOperationsResponse{
			Operations: []*domain.BulkOperation{
				{ID: "bulk_1", CreatedBy: "admin_1"},
			},
		}
		bulkRepo.EXPECT().GetByUser(ctx, "admin_1", gomock.Any()).Return(expected, nil)

		resp, err := svc.GetByUser(ctx, "admin_1", domain.PaginationRequest{Limit: 20})
		require.NoError(t, err)
		assert.Len(t, resp.Operations, 1)
	})
}

// ---------------------------------------------------------------------------
// TestBulkService_ImportProducts
// ---------------------------------------------------------------------------

func TestBulkService_ImportProducts(t *testing.T) {
	ctx := context.Background()

	t.Run("successful import creation", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)

		bulkRepo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, op *domain.BulkOperation) error {
			assert.Equal(t, domain.BulkOperationTypeImport, op.Type)
			assert.Equal(t, domain.BulkOperationEntityProduct, op.EntityType)
			assert.Equal(t, domain.BulkOperationStatusPending, op.Status)
			assert.Equal(t, "s3://bucket/products.csv", op.InputFileURL)
			return nil
		})

		op, err := svc.ImportProducts(ctx, domain.BulkProductImportRequest{
			FileURL:  "s3://bucket/products.csv",
			Metadata: map[string]interface{}{},
		}, "admin_1")

		require.NoError(t, err)
		assert.Contains(t, op.ID, "bulk_")
	})

	t.Run("dry run sets metadata flag", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)

		bulkRepo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, op *domain.BulkOperation) error {
			assert.Equal(t, true, op.Metadata["dry_run"])
			return nil
		})

		op, err := svc.ImportProducts(ctx, domain.BulkProductImportRequest{
			FileURL:  "s3://bucket/products.csv",
			DryRun:   true,
			Metadata: map[string]interface{}{},
		}, "admin_1")

		require.NoError(t, err)
		assert.NotNil(t, op)
	})

	t.Run("repo error", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)
		bulkRepo.EXPECT().Create(ctx, gomock.Any()).Return(errors.Internal("db error"))

		op, err := svc.ImportProducts(ctx, domain.BulkProductImportRequest{
			FileURL:  "s3://bucket/products.csv",
			Metadata: map[string]interface{}{},
		}, "admin_1")

		require.Error(t, err)
		assert.Nil(t, op)
	})
}

// ---------------------------------------------------------------------------
// TestBulkService_UpdateInventory
// ---------------------------------------------------------------------------

func TestBulkService_UpdateInventory(t *testing.T) {
	ctx := context.Background()

	t.Run("successful inventory update creation", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)

		bulkRepo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, op *domain.BulkOperation) error {
			assert.Equal(t, domain.BulkOperationTypeUpdate, op.Type)
			assert.Equal(t, domain.BulkOperationEntityInventory, op.EntityType)
			assert.Equal(t, domain.BulkOperationStatusPending, op.Status)
			return nil
		})

		op, err := svc.UpdateInventory(ctx, domain.BulkInventoryUpdateRequest{
			FileURL:  "s3://bucket/inventory.csv",
			Metadata: map[string]interface{}{},
		}, "admin_1")

		require.NoError(t, err)
		assert.Contains(t, op.ID, "bulk_")
	})

	t.Run("dry run sets metadata flag", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)

		bulkRepo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, op *domain.BulkOperation) error {
			assert.Equal(t, true, op.Metadata["dry_run"])
			return nil
		})

		_, err := svc.UpdateInventory(ctx, domain.BulkInventoryUpdateRequest{
			FileURL:  "s3://bucket/inventory.csv",
			DryRun:   true,
			Metadata: map[string]interface{}{},
		}, "admin_1")
		require.NoError(t, err)
	})
}

// ---------------------------------------------------------------------------
// TestBulkService_UpdatePrices
// ---------------------------------------------------------------------------

func TestBulkService_UpdatePrices(t *testing.T) {
	ctx := context.Background()

	t.Run("successful price update creation", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)

		bulkRepo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, op *domain.BulkOperation) error {
			assert.Equal(t, domain.BulkOperationTypeUpdate, op.Type)
			assert.Equal(t, domain.BulkOperationEntityProduct, op.EntityType)
			assert.Equal(t, "price", op.Metadata["update_type"])
			return nil
		})

		op, err := svc.UpdatePrices(ctx, domain.BulkPriceUpdateRequest{
			FileURL: "s3://bucket/prices.csv",
		}, "admin_1")

		require.NoError(t, err)
		assert.Contains(t, op.ID, "bulk_")
	})

	t.Run("dry run with extra metadata", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)

		bulkRepo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, op *domain.BulkOperation) error {
			assert.Equal(t, true, op.Metadata["dry_run"])
			assert.Equal(t, "price", op.Metadata["update_type"])
			assert.Equal(t, "10%", op.Metadata["increase"])
			return nil
		})

		_, err := svc.UpdatePrices(ctx, domain.BulkPriceUpdateRequest{
			FileURL:  "s3://bucket/prices.csv",
			DryRun:   true,
			Metadata: map[string]interface{}{"increase": "10%"},
		}, "admin_1")
		require.NoError(t, err)
	})
}

// ---------------------------------------------------------------------------
// TestBulkService_ExportData
// ---------------------------------------------------------------------------

func TestBulkService_ExportData(t *testing.T) {
	ctx := context.Background()

	t.Run("successful export creation with format", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)

		bulkRepo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, op *domain.BulkOperation) error {
			assert.Equal(t, domain.BulkOperationTypeExport, op.Type)
			assert.Equal(t, domain.BulkOperationEntityProduct, op.EntityType)
			assert.Equal(t, "JSON", op.Metadata["format"])
			return nil
		})

		op, err := svc.ExportData(ctx, domain.BulkExportRequest{
			EntityType: domain.BulkOperationEntityProduct,
			Format:     "JSON",
		}, "admin_1")

		require.NoError(t, err)
		assert.Contains(t, op.ID, "bulk_")
	})

	t.Run("default format is CSV when empty", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)

		bulkRepo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, op *domain.BulkOperation) error {
			assert.Equal(t, "CSV", op.Metadata["format"])
			return nil
		})

		_, err := svc.ExportData(ctx, domain.BulkExportRequest{
			EntityType: domain.BulkOperationEntityProduct,
			Format:     "",
		}, "admin_1")
		require.NoError(t, err)
	})

	t.Run("repo error", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)
		bulkRepo.EXPECT().Create(ctx, gomock.Any()).Return(errors.Internal("db error"))

		op, err := svc.ExportData(ctx, domain.BulkExportRequest{
			EntityType: domain.BulkOperationEntityProduct,
		}, "admin_1")

		require.Error(t, err)
		assert.Nil(t, op)
	})
}

// ---------------------------------------------------------------------------
// TestBulkService_CancelOperation
// ---------------------------------------------------------------------------

func TestBulkService_CancelOperation(t *testing.T) {
	ctx := context.Background()

	t.Run("cancel pending operation", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)

		bulkRepo.EXPECT().GetByID(ctx, "bulk_123").Return(&domain.BulkOperation{
			ID:       "bulk_123",
			Status:   domain.BulkOperationStatusPending,
			Metadata: map[string]interface{}{},
		}, nil)
		bulkRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, op *domain.BulkOperation) error {
			assert.Equal(t, domain.BulkOperationStatusCancelled, op.Status)
			assert.Equal(t, "admin_1", op.Metadata["cancelled_by"])
			assert.NotNil(t, op.CompletedAt)
			return nil
		})

		err := svc.CancelOperation(ctx, "bulk_123", "admin_1")
		require.NoError(t, err)
	})

	t.Run("cancel processing operation", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)

		bulkRepo.EXPECT().GetByID(ctx, "bulk_123").Return(&domain.BulkOperation{
			ID:       "bulk_123",
			Status:   domain.BulkOperationStatusProcessing,
			Metadata: map[string]interface{}{},
		}, nil)
		bulkRepo.EXPECT().Update(ctx, gomock.Any()).Return(nil)

		err := svc.CancelOperation(ctx, "bulk_123", "admin_1")
		require.NoError(t, err)
	})

	t.Run("cannot cancel completed operation", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)

		bulkRepo.EXPECT().GetByID(ctx, "bulk_123").Return(&domain.BulkOperation{
			ID:       "bulk_123",
			Status:   domain.BulkOperationStatusCompleted,
			Metadata: map[string]interface{}{},
		}, nil)

		err := svc.CancelOperation(ctx, "bulk_123", "admin_1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot cancel operation in status")
	})

	t.Run("cannot cancel failed operation", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)

		bulkRepo.EXPECT().GetByID(ctx, "bulk_123").Return(&domain.BulkOperation{
			ID:       "bulk_123",
			Status:   domain.BulkOperationStatusFailed,
			Metadata: map[string]interface{}{},
		}, nil)

		err := svc.CancelOperation(ctx, "bulk_123", "admin_1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot cancel operation in status")
	})

	t.Run("operation not found", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)
		bulkRepo.EXPECT().GetByID(ctx, "bulk_notfound").Return(nil, errors.NotFound("BulkOperation"))

		err := svc.CancelOperation(ctx, "bulk_notfound", "admin_1")
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// TestBulkService_UpdateOperationStatus
// ---------------------------------------------------------------------------

func TestBulkService_UpdateOperationStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("update to processing sets started_at", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)

		bulkRepo.EXPECT().GetByID(ctx, "bulk_123").Return(&domain.BulkOperation{
			ID:     "bulk_123",
			Status: domain.BulkOperationStatusPending,
		}, nil)
		bulkRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, op *domain.BulkOperation) error {
			assert.Equal(t, domain.BulkOperationStatusProcessing, op.Status)
			assert.NotNil(t, op.StartedAt)
			assert.Nil(t, op.CompletedAt)
			return nil
		})

		err := svc.UpdateOperationStatus(ctx, "bulk_123", domain.BulkOperationStatusProcessing, nil)
		require.NoError(t, err)
	})

	t.Run("update to completed sets completed_at and result", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)
		now := time.Now()

		bulkRepo.EXPECT().GetByID(ctx, "bulk_123").Return(&domain.BulkOperation{
			ID:        "bulk_123",
			Status:    domain.BulkOperationStatusProcessing,
			StartedAt: &now,
		}, nil)
		bulkRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, op *domain.BulkOperation) error {
			assert.Equal(t, domain.BulkOperationStatusCompleted, op.Status)
			assert.NotNil(t, op.CompletedAt)
			assert.Equal(t, 100, op.TotalRecords)
			assert.Equal(t, 95, op.SuccessCount)
			assert.Equal(t, 5, op.FailureCount)
			assert.Equal(t, "s3://output.csv", op.OutputFileURL)
			assert.Equal(t, "s3://errors.csv", op.ErrorFileURL)
			return nil
		})

		err := svc.UpdateOperationStatus(ctx, "bulk_123", domain.BulkOperationStatusCompleted, &domain.BulkOperationResult{
			TotalRecords:  100,
			SuccessCount:  95,
			FailureCount:  5,
			OutputFileURL: "s3://output.csv",
			ErrorFileURL:  "s3://errors.csv",
		})
		require.NoError(t, err)
	})

	t.Run("update to failed sets completed_at", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)

		bulkRepo.EXPECT().GetByID(ctx, "bulk_123").Return(&domain.BulkOperation{
			ID:     "bulk_123",
			Status: domain.BulkOperationStatusProcessing,
		}, nil)
		bulkRepo.EXPECT().Update(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, op *domain.BulkOperation) error {
			assert.Equal(t, domain.BulkOperationStatusFailed, op.Status)
			assert.NotNil(t, op.CompletedAt)
			return nil
		})

		err := svc.UpdateOperationStatus(ctx, "bulk_123", domain.BulkOperationStatusFailed, nil)
		require.NoError(t, err)
	})

	t.Run("operation not found", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)
		bulkRepo.EXPECT().GetByID(ctx, "bulk_notfound").Return(nil, errors.NotFound("BulkOperation"))

		err := svc.UpdateOperationStatus(ctx, "bulk_notfound", domain.BulkOperationStatusProcessing, nil)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// TestBulkService_GetDownloadURL
// ---------------------------------------------------------------------------

func TestBulkService_GetDownloadURL(t *testing.T) {
	ctx := context.Background()

	t.Run("successful download URL", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)
		bulkRepo.EXPECT().GetByID(ctx, "bulk_123").Return(&domain.BulkOperation{
			ID:            "bulk_123",
			OutputFileURL: "s3://bucket/output.csv",
		}, nil)

		url, err := svc.GetDownloadURL(ctx, "bulk_123")
		require.NoError(t, err)
		assert.Equal(t, "s3://bucket/output.csv", url)
	})

	t.Run("output file not available", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)
		bulkRepo.EXPECT().GetByID(ctx, "bulk_123").Return(&domain.BulkOperation{
			ID:            "bulk_123",
			OutputFileURL: "",
		}, nil)

		url, err := svc.GetDownloadURL(ctx, "bulk_123")
		require.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "not available")
	})

	t.Run("operation not found", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)
		bulkRepo.EXPECT().GetByID(ctx, "bulk_notfound").Return(nil, errors.NotFound("BulkOperation"))

		url, err := svc.GetDownloadURL(ctx, "bulk_notfound")
		require.Error(t, err)
		assert.Empty(t, url)
	})
}

// ---------------------------------------------------------------------------
// TestBulkService_GetErrorFileURL
// ---------------------------------------------------------------------------

func TestBulkService_GetErrorFileURL(t *testing.T) {
	ctx := context.Background()

	t.Run("successful error file URL", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)
		bulkRepo.EXPECT().GetByID(ctx, "bulk_123").Return(&domain.BulkOperation{
			ID:           "bulk_123",
			ErrorFileURL: "s3://bucket/errors.csv",
		}, nil)

		url, err := svc.GetErrorFileURL(ctx, "bulk_123")
		require.NoError(t, err)
		assert.Equal(t, "s3://bucket/errors.csv", url)
	})

	t.Run("error file not available", func(t *testing.T) {
		svc, bulkRepo := newBulkServiceTest(t)
		bulkRepo.EXPECT().GetByID(ctx, "bulk_123").Return(&domain.BulkOperation{
			ID:           "bulk_123",
			ErrorFileURL: "",
		}, nil)

		url, err := svc.GetErrorFileURL(ctx, "bulk_123")
		require.Error(t, err)
		assert.Empty(t, url)
		assert.Contains(t, err.Error(), "not available")
	})
}

// ---------------------------------------------------------------------------
// TestBulkService_GetUploadURL
// ---------------------------------------------------------------------------

func TestBulkService_GetUploadURL(t *testing.T) {
	ctx := context.Background()

	t.Run("returns placeholder URL", func(t *testing.T) {
		svc, _ := newBulkServiceTest(t)

		url, err := svc.GetUploadURL(ctx, domain.BulkOperationEntityProduct, "products.csv")
		require.NoError(t, err)
		assert.Contains(t, url, "products.csv")
	})
}
