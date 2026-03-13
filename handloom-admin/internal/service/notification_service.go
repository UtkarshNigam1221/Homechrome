// Package service implements the business logic layer
package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/logger"
)

// NotificationService implements notification operations
type NotificationService struct {
	notificationRepo domain.NotificationRepository
	userRepo         domain.UserRepository
	logger           *logger.Logger
}

// NewNotificationService creates a new NotificationService
func NewNotificationService(
	notificationRepo domain.NotificationRepository,
	userRepo domain.UserRepository,
	logger *logger.Logger,
) *NotificationService {
	return &NotificationService{
		notificationRepo: notificationRepo,
		userRepo:         userRepo,
		logger:           logger,
	}
}

// Send sends a notification
func (s *NotificationService) Send(ctx context.Context, req domain.SendNotificationRequest, createdBy string) (*domain.Notification, error) {
	notification := &domain.Notification{
		ID:             "notif_" + uuid.New().String()[:8],
		Type:           req.Type,
		Status:         domain.NotificationStatusPending,
		RecipientID:    req.RecipientID,
		RecipientEmail: req.RecipientEmail,
		RecipientPhone: req.RecipientPhone,
		Subject:        req.Subject,
		Body:           req.Body,
		TemplateID:     req.TemplateID,
		TemplateData:   req.TemplateData,
		TriggerType:    req.TriggerType,
		ReferenceType:  req.ReferenceType,
		ReferenceID:    req.ReferenceID,
		CreatedAt:      time.Now(),
		CreatedBy:      createdBy,
	}

	if err := s.notificationRepo.Create(ctx, notification); err != nil {
		return nil, err
	}

	s.logger.WithContext(ctx).Infof("Created notification: %s", notification.ID)
	return notification, nil
}

// SendBulk sends notifications to multiple recipients
func (s *NotificationService) SendBulk(ctx context.Context, req domain.SendBulkNotificationRequest, createdBy string) (*domain.SendBulkNotificationResponse, error) {
	response := &domain.SendBulkNotificationResponse{
		Total:     len(req.RecipientIDs),
		Sent:      0,
		Failed:    0,
		FailedIDs: []string{},
	}

	for _, recipientID := range req.RecipientIDs {
		sendReq := domain.SendNotificationRequest{
			Type:         req.Type,
			RecipientID:  recipientID,
			Subject:      req.Subject,
			Body:         req.Body,
			TemplateID:   req.TemplateID,
			TemplateData: req.TemplateData,
			TriggerType:  req.TriggerType,
		}

		_, err := s.Send(ctx, sendReq, createdBy)
		if err != nil {
			response.Failed++
			response.FailedIDs = append(response.FailedIDs, recipientID)
			s.logger.WithContext(ctx).WithError(err).Errorf("Failed to send notification to: %s", recipientID)
		} else {
			response.Sent++
		}
	}

	return response, nil
}

// GetByID retrieves a notification by ID
func (s *NotificationService) GetByID(ctx context.Context, id string) (*domain.Notification, error) {
	return s.notificationRepo.GetByID(ctx, id)
}

// List retrieves notifications with filters
func (s *NotificationService) List(ctx context.Context, req domain.ListNotificationsRequest) (*domain.ListNotificationsResponse, error) {
	return s.notificationRepo.List(ctx, req)
}

// GetByUser retrieves notifications for a specific user
func (s *NotificationService) GetByUser(ctx context.Context, userID string, pagination domain.PaginationRequest) (*domain.ListNotificationsResponse, error) {
	return s.notificationRepo.GetByUser(ctx, userID, pagination)
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, id string) error {
	notification, err := s.notificationRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now()
	notification.Status = domain.NotificationStatusDelivered
	notification.DeliveredAt = &now

	return s.notificationRepo.Update(ctx, notification)
}

// MarkAllAsRead marks all notifications for a user as read
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID string) error {
	return s.notificationRepo.MarkAllAsRead(ctx, userID)
}

// SendOrderNotification sends an order-related notification
func (s *NotificationService) SendOrderNotification(ctx context.Context, order *domain.Order, trigger domain.NotificationTrigger, createdBy string) error {
	req := domain.SendNotificationRequest{
		Type:           domain.NotificationTypeEmail,
		RecipientID:    order.CustomerID,
		RecipientEmail: order.CustomerEmail,
		Subject:        s.getOrderSubject(trigger, order.OrderNumber),
		Body:           s.getOrderBody(trigger, order),
		TriggerType:    trigger,
		ReferenceType:  "ORDER",
		ReferenceID:    order.ID,
		TemplateData: map[string]interface{}{
			"order_number": order.OrderNumber,
			"order_status": order.Status,
			"total_amount": order.TotalAmount,
		},
	}

	_, err := s.Send(ctx, req, createdBy)
	return err
}

func (s *NotificationService) getOrderSubject(trigger domain.NotificationTrigger, orderNumber string) string {
	switch trigger {
	case domain.NotificationTriggerOrderCreated:
		return "Order Confirmation - " + orderNumber
	case domain.NotificationTriggerOrderStatus:
		return "Order Update - " + orderNumber
	case domain.NotificationTriggerShipment:
		return "Your Order Has Been Shipped - " + orderNumber
	case domain.NotificationTriggerPayment:
		return "Payment Received - " + orderNumber
	case domain.NotificationTriggerRefund:
		return "Refund Processed - " + orderNumber
	default:
		return "Order Update - " + orderNumber
	}
}

func (s *NotificationService) getOrderBody(trigger domain.NotificationTrigger, order *domain.Order) string {
	// In production, these would be templates rendered with proper HTML
	switch trigger {
	case domain.NotificationTriggerOrderCreated:
		return "Thank you for your order! Your order number is " + order.OrderNumber
	case domain.NotificationTriggerShipment:
		return "Great news! Your order " + order.OrderNumber + " has been shipped."
	default:
		return "Your order " + order.OrderNumber + " has been updated."
	}
}
