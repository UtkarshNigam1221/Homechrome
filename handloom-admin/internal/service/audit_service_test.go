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
)

func TestAuditService_Log(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuditRepo := mocks.NewMockAuditRepository(ctrl)
	service := NewAuditService(mockAuditRepo)
	ctx := context.Background()

	t.Run("successful audit log creation", func(t *testing.T) {
		changes := []domain.FieldChange{
			{OldValue: "pending", NewValue: "active"},
		}
		metadata := map[string]interface{}{
			"ip_address": "192.168.1.1",
		}

		mockAuditRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, auditLog *domain.AuditLog) error {
				assert.Equal(t, domain.AuditAction("UPDATE"), auditLog.Action)
				assert.Equal(t, "USER", auditLog.EntityTypeAudit)
				assert.Equal(t, "user_123", auditLog.EntityID)
				assert.Equal(t, "admin_456", auditLog.UserID)
				assert.Len(t, auditLog.Changes, 1)
				change, ok := auditLog.Changes["change_0"]
				assert.True(t, ok)
				assert.Equal(t, "pending", change.OldValue)
				// Verify TTL is approximately 90 days from now
				expectedTTL := time.Now().Add(90 * 24 * time.Hour).Unix()
				assert.InDelta(t, expectedTTL, auditLog.TTL, 60) // Allow 60 seconds variance
				return nil
			})

		err := service.Log(ctx, "UPDATE", "USER", "user_123", "admin_456", changes, metadata)

		require.NoError(t, err)
	})

	t.Run("audit log creation failure", func(t *testing.T) {
		mockAuditRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(errors.Internal("Database error"))

		err := service.Log(ctx, "CREATE", "PRODUCT", "prod_123", "admin_123", nil, nil)

		require.Error(t, err)
	})
}

func TestAuditService_GetByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuditRepo := mocks.NewMockAuditRepository(ctrl)
	service := NewAuditService(mockAuditRepo)
	ctx := context.Background()

	t.Run("successful get by ID", func(t *testing.T) {
		expectedLog := &domain.AuditLog{
			ID:              "audit_123",
			Action:          "UPDATE",
			EntityTypeAudit: "USER",
			EntityID:        "user_123",
			UserID:          "admin_456",
		}

		mockAuditRepo.EXPECT().
			GetByID(ctx, "audit_123").
			Return(expectedLog, nil)

		result, err := service.GetByID(ctx, "audit_123")

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "audit_123", result.ID)
		assert.Equal(t, domain.AuditAction("UPDATE"), result.Action)
	})

	t.Run("audit log not found", func(t *testing.T) {
		mockAuditRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Audit log not found"))

		result, err := service.GetByID(ctx, "nonexistent")

		assert.Nil(t, result)
		require.Error(t, err)
	})
}

func TestAuditService_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuditRepo := mocks.NewMockAuditRepository(ctrl)
	service := NewAuditService(mockAuditRepo)
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		req := domain.ListAuditLogsRequest{
			PaginationRequest: domain.PaginationRequest{
				Limit: 20,
			},
		}

		expectedResponse := &domain.ListAuditLogsResponse{
			Logs: []*domain.AuditLog{
				{ID: "audit_1", Action: "CREATE"},
				{ID: "audit_2", Action: "UPDATE"},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockAuditRepo.EXPECT().
			List(ctx, req).
			Return(expectedResponse, nil)

		response, err := service.List(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Logs, 2)
	})

	t.Run("list with action filter", func(t *testing.T) {
		action := "UPDATE"
		req := domain.ListAuditLogsRequest{
			PaginationRequest: domain.PaginationRequest{
				Limit: 10,
			},
			Action: &action,
		}

		expectedResponse := &domain.ListAuditLogsResponse{
			Logs: []*domain.AuditLog{
				{ID: "audit_1", Action: "UPDATE"},
			},
			Pagination: domain.PaginationResponse{
				Limit:   10,
				HasMore: false,
			},
		}

		mockAuditRepo.EXPECT().
			List(ctx, req).
			Return(expectedResponse, nil)

		response, err := service.List(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Logs, 1)
	})
}

func TestAuditService_GetByEntity(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuditRepo := mocks.NewMockAuditRepository(ctrl)
	service := NewAuditService(mockAuditRepo)
	ctx := context.Background()

	t.Run("successful get by entity", func(t *testing.T) {
		pagination := domain.PaginationRequest{
			Limit: 20,
		}

		expectedResponse := &domain.ListAuditLogsResponse{
			Logs: []*domain.AuditLog{
				{ID: "audit_1", EntityTypeAudit: "USER", EntityID: "user_123"},
				{ID: "audit_2", EntityTypeAudit: "USER", EntityID: "user_123"},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockAuditRepo.EXPECT().
			GetByEntity(ctx, "USER", "user_123", pagination).
			Return(expectedResponse, nil)

		response, err := service.GetByEntity(ctx, "USER", "user_123", pagination)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Logs, 2)
	})
}

func TestAuditService_GetByUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAuditRepo := mocks.NewMockAuditRepository(ctrl)
	service := NewAuditService(mockAuditRepo)
	ctx := context.Background()

	t.Run("successful get by user", func(t *testing.T) {
		pagination := domain.PaginationRequest{
			Limit: 20,
		}

		expectedResponse := &domain.ListAuditLogsResponse{
			Logs: []*domain.AuditLog{
				{ID: "audit_1", UserID: "admin_123"},
				{ID: "audit_2", UserID: "admin_123"},
				{ID: "audit_3", UserID: "admin_123"},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockAuditRepo.EXPECT().
			GetByUser(ctx, "admin_123", pagination).
			Return(expectedResponse, nil)

		response, err := service.GetByUser(ctx, "admin_123", pagination)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Logs, 3)
	})

	t.Run("get by user with no results", func(t *testing.T) {
		pagination := domain.PaginationRequest{
			Limit: 20,
		}

		expectedResponse := &domain.ListAuditLogsResponse{
			Logs: []*domain.AuditLog{},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockAuditRepo.EXPECT().
			GetByUser(ctx, "user_with_no_logs", pagination).
			Return(expectedResponse, nil)

		response, err := service.GetByUser(ctx, "user_with_no_logs", pagination)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Logs, 0)
	})
}
