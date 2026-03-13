package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/mocks"
	"github.com/handloom/admin/pkg/errors"
	"github.com/handloom/admin/pkg/logger"
)

func TestNotificationService_Send(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNotifRepo := mocks.NewMockNotificationRepository(ctrl)
	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	log := logger.NewNoop()
	service := NewNotificationService(mockNotifRepo, mockUserRepo, log)
	ctx := context.Background()

	t.Run("successful send", func(t *testing.T) {
		req := domain.SendNotificationRequest{
			Type:           domain.NotificationTypeEmail,
			RecipientID:    "user_123",
			RecipientEmail: "user@example.com",
			Subject:        "Test Subject",
			Body:           "Test Body",
			TriggerType:    domain.NotificationTriggerManual,
		}

		mockNotifRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, notif *domain.Notification) error {
				assert.Contains(t, notif.ID, "notif_")
				assert.Equal(t, domain.NotificationTypeEmail, notif.Type)
				assert.Equal(t, domain.NotificationStatusPending, notif.Status)
				assert.Equal(t, "user_123", notif.RecipientID)
				assert.Equal(t, "user@example.com", notif.RecipientEmail)
				assert.Equal(t, "Test Subject", notif.Subject)
				assert.Equal(t, "Test Body", notif.Body)
				assert.Equal(t, "admin_123", notif.CreatedBy)
				return nil
			})

		notif, err := service.Send(ctx, req, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, notif)
		assert.Contains(t, notif.ID, "notif_")
		assert.Equal(t, domain.NotificationStatusPending, notif.Status)
	})

	t.Run("repo error on create", func(t *testing.T) {
		req := domain.SendNotificationRequest{
			Type:        domain.NotificationTypeSMS,
			RecipientID: "user_123",
			Body:        "Test",
			TriggerType: domain.NotificationTriggerManual,
		}

		mockNotifRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(errors.Internal("Database error"))

		notif, err := service.Send(ctx, req, "admin_123")

		assert.Nil(t, notif)
		require.Error(t, err)
	})
}

func TestNotificationService_SendBulk(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNotifRepo := mocks.NewMockNotificationRepository(ctrl)
	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	log := logger.NewNoop()
	service := NewNotificationService(mockNotifRepo, mockUserRepo, log)
	ctx := context.Background()

	t.Run("bulk send with mixed results", func(t *testing.T) {
		req := domain.SendBulkNotificationRequest{
			Type:         domain.NotificationTypeEmail,
			RecipientIDs: []string{"user_1", "user_2"},
			Subject:      "Bulk Subject",
			Body:         "Bulk Body",
			TriggerType:  domain.NotificationTriggerManual,
		}

		// First recipient succeeds
		mockNotifRepo.EXPECT().
			Create(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, notif *domain.Notification) error {
				if notif.RecipientID == "user_1" {
					return nil
				}
				return errors.Internal("Failed for user_2")
			}).
			Times(2)

		response, err := service.SendBulk(ctx, req, "admin_123")

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, 2, response.Total)
		assert.Equal(t, 1, response.Sent)
		assert.Equal(t, 1, response.Failed)
		assert.Contains(t, response.FailedIDs, "user_2")
	})

	t.Run("all recipients succeed", func(t *testing.T) {
		req := domain.SendBulkNotificationRequest{
			Type:         domain.NotificationTypeEmail,
			RecipientIDs: []string{"user_1", "user_2"},
			Subject:      "Bulk Subject",
			Body:         "Bulk Body",
			TriggerType:  domain.NotificationTriggerManual,
		}

		mockNotifRepo.EXPECT().
			Create(ctx, gomock.Any()).
			Return(nil).
			Times(2)

		response, err := service.SendBulk(ctx, req, "admin_123")

		require.NoError(t, err)
		assert.Equal(t, 2, response.Total)
		assert.Equal(t, 2, response.Sent)
		assert.Equal(t, 0, response.Failed)
		assert.Empty(t, response.FailedIDs)
	})
}

func TestNotificationService_GetByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNotifRepo := mocks.NewMockNotificationRepository(ctrl)
	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	log := logger.NewNoop()
	service := NewNotificationService(mockNotifRepo, mockUserRepo, log)
	ctx := context.Background()

	t.Run("successful get", func(t *testing.T) {
		expected := &domain.Notification{
			ID:          "notif_abc123",
			Type:        domain.NotificationTypeEmail,
			Status:      domain.NotificationStatusSent,
			RecipientID: "user_123",
			Subject:     "Test",
		}

		mockNotifRepo.EXPECT().
			GetByID(ctx, "notif_abc123").
			Return(expected, nil)

		notif, err := service.GetByID(ctx, "notif_abc123")

		require.NoError(t, err)
		assert.Equal(t, "notif_abc123", notif.ID)
	})

	t.Run("not found", func(t *testing.T) {
		mockNotifRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Notification"))

		notif, err := service.GetByID(ctx, "nonexistent")

		assert.Nil(t, notif)
		require.Error(t, err)
	})
}

func TestNotificationService_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNotifRepo := mocks.NewMockNotificationRepository(ctrl)
	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	log := logger.NewNoop()
	service := NewNotificationService(mockNotifRepo, mockUserRepo, log)
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		req := domain.ListNotificationsRequest{
			PaginationRequest: domain.PaginationRequest{Limit: 20},
		}

		expectedResponse := &domain.ListNotificationsResponse{
			Notifications: []*domain.Notification{
				{ID: "notif_1", Subject: "Order Confirmation"},
				{ID: "notif_2", Subject: "Shipment Update"},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
			UnreadCount: 1,
		}

		mockNotifRepo.EXPECT().
			List(ctx, req).
			Return(expectedResponse, nil)

		response, err := service.List(ctx, req)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Notifications, 2)
		assert.Equal(t, 1, response.UnreadCount)
	})
}

func TestNotificationService_GetByUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNotifRepo := mocks.NewMockNotificationRepository(ctrl)
	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	log := logger.NewNoop()
	service := NewNotificationService(mockNotifRepo, mockUserRepo, log)
	ctx := context.Background()

	t.Run("successful get by user", func(t *testing.T) {
		pagination := domain.PaginationRequest{Limit: 20}

		expectedResponse := &domain.ListNotificationsResponse{
			Notifications: []*domain.Notification{
				{ID: "notif_1", RecipientID: "user_123"},
				{ID: "notif_2", RecipientID: "user_123"},
			},
			Pagination: domain.PaginationResponse{
				Limit:   20,
				HasMore: false,
			},
		}

		mockNotifRepo.EXPECT().
			GetByUser(ctx, "user_123", pagination).
			Return(expectedResponse, nil)

		response, err := service.GetByUser(ctx, "user_123", pagination)

		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.Notifications, 2)
	})
}

func TestNotificationService_MarkAsRead(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNotifRepo := mocks.NewMockNotificationRepository(ctrl)
	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	log := logger.NewNoop()
	service := NewNotificationService(mockNotifRepo, mockUserRepo, log)
	ctx := context.Background()

	t.Run("successful mark as read", func(t *testing.T) {
		existing := &domain.Notification{
			ID:          "notif_abc123",
			Type:        domain.NotificationTypeEmail,
			Status:      domain.NotificationStatusSent,
			RecipientID: "user_123",
		}

		mockNotifRepo.EXPECT().
			GetByID(ctx, "notif_abc123").
			Return(existing, nil)

		mockNotifRepo.EXPECT().
			Update(ctx, gomock.Any()).
			DoAndReturn(func(ctx context.Context, notif *domain.Notification) error {
				assert.Equal(t, domain.NotificationStatusDelivered, notif.Status)
				assert.NotNil(t, notif.DeliveredAt)
				return nil
			})

		err := service.MarkAsRead(ctx, "notif_abc123")

		require.NoError(t, err)
	})

	t.Run("notification not found", func(t *testing.T) {
		mockNotifRepo.EXPECT().
			GetByID(ctx, "nonexistent").
			Return(nil, errors.NotFound("Notification"))

		err := service.MarkAsRead(ctx, "nonexistent")

		require.Error(t, err)
	})
}

func TestNotificationService_MarkAllAsRead(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNotifRepo := mocks.NewMockNotificationRepository(ctrl)
	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	log := logger.NewNoop()
	service := NewNotificationService(mockNotifRepo, mockUserRepo, log)
	ctx := context.Background()

	t.Run("successful mark all as read", func(t *testing.T) {
		mockNotifRepo.EXPECT().
			MarkAllAsRead(ctx, "user_123").
			Return(nil)

		err := service.MarkAllAsRead(ctx, "user_123")

		require.NoError(t, err)
	})

	t.Run("repo error", func(t *testing.T) {
		mockNotifRepo.EXPECT().
			MarkAllAsRead(ctx, "user_123").
			Return(errors.Internal("Database error"))

		err := service.MarkAllAsRead(ctx, "user_123")

		require.Error(t, err)
	})
}

func TestNotificationService_SendOrderNotification(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNotifRepo := mocks.NewMockNotificationRepository(ctrl)
	mockUserRepo := mocks.NewMockUserRepository(ctrl)
	log := logger.NewNoop()
	service := NewNotificationService(mockNotifRepo, mockUserRepo, log)
	ctx := context.Background()

	tests := []struct {
		name            string
		trigger         domain.NotificationTrigger
		expectedSubject string
	}{
		{
			name:            "ORDER_CREATED trigger",
			trigger:         domain.NotificationTriggerOrderCreated,
			expectedSubject: "Order Confirmation - HL20240101001",
		},
		{
			name:            "SHIPMENT trigger",
			trigger:         domain.NotificationTriggerShipment,
			expectedSubject: "Your Order Has Been Shipped - HL20240101001",
		},
		{
			name:            "PAYMENT trigger",
			trigger:         domain.NotificationTriggerPayment,
			expectedSubject: "Payment Received - HL20240101001",
		},
		{
			name:            "REFUND trigger",
			trigger:         domain.NotificationTriggerRefund,
			expectedSubject: "Refund Processed - HL20240101001",
		},
		{
			name:            "ORDER_STATUS trigger",
			trigger:         domain.NotificationTriggerOrderStatus,
			expectedSubject: "Order Update - HL20240101001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := &domain.Order{
				ID:            "order_123",
				OrderNumber:   "HL20240101001",
				CustomerID:    "cust_123",
				CustomerEmail: "customer@example.com",
				Status:        domain.OrderStatusConfirmed,
				TotalAmount:   500000,
			}

			mockNotifRepo.EXPECT().
				Create(ctx, gomock.Any()).
				DoAndReturn(func(ctx context.Context, notif *domain.Notification) error {
					assert.Equal(t, domain.NotificationTypeEmail, notif.Type)
					assert.Equal(t, "cust_123", notif.RecipientID)
					assert.Equal(t, "customer@example.com", notif.RecipientEmail)
					assert.Equal(t, tt.expectedSubject, notif.Subject)
					assert.Equal(t, tt.trigger, notif.TriggerType)
					assert.Equal(t, "ORDER", notif.ReferenceType)
					assert.Equal(t, "order_123", notif.ReferenceID)
					assert.NotNil(t, notif.TemplateData)
					return nil
				})

			err := service.SendOrderNotification(ctx, order, tt.trigger, "admin_123")

			require.NoError(t, err)
		})
	}
}
