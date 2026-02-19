package dynamodb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/pkg/errors"
)

// NotificationRepository implements domain.NotificationRepository
type NotificationRepository struct {
	client *Client
}

// NewNotificationRepository creates a new NotificationRepository
func NewNotificationRepository(client *Client) *NotificationRepository {
	return &NotificationRepository{
		client: client,
	}
}

// Create creates a new notification
func (r *NotificationRepository) Create(ctx context.Context, notification *domain.Notification) error {
	now := time.Now()
	notification.CreatedAt = now
	notification.SetKeys()

	av, err := attributevalue.MarshalMap(notification)
	if err != nil {
		return errors.Internal("Failed to marshal notification")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.coreTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.New(errors.ErrCodeAlreadyExists, "Notification already exists")
		}
		return errors.Wrap(err, "Failed to create notification")
	}

	return nil
}

// GetByID retrieves a notification by ID
func (r *NotificationRepository) GetByID(ctx context.Context, id string) (*domain.Notification, error) {
	result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.client.coreTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: "NOTIFICATION#" + id},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get notification")
	}

	if result.Item == nil {
		return nil, errors.NotFound("Notification")
	}

	var notification domain.Notification
	if err := attributevalue.UnmarshalMap(result.Item, &notification); err != nil {
		return nil, errors.Internal("Failed to unmarshal notification")
	}

	return &notification, nil
}

// Update updates a notification
func (r *NotificationRepository) Update(ctx context.Context, notification *domain.Notification) error {
	notification.SetKeys()

	av, err := attributevalue.MarshalMap(notification)
	if err != nil {
		return errors.Internal("Failed to marshal notification")
	}

	_, err = r.client.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.client.coreTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		if isConditionalCheckFailed(err) {
			return errors.NotFound("Notification")
		}
		return errors.Wrap(err, "Failed to update notification")
	}

	return nil
}

// List lists notifications with filters
func (r *NotificationRepository) List(ctx context.Context, req domain.ListNotificationsRequest) (*domain.ListNotificationsResponse, error) {
	// TODO: Implement with DynamoDB scan/query
	return &domain.ListNotificationsResponse{
		Notifications: []*domain.Notification{},
		Pagination:    domain.PaginationResponse{},
	}, nil
}

// GetByUser retrieves notifications for a user
func (r *NotificationRepository) GetByUser(ctx context.Context, userID string, pagination domain.PaginationRequest) (*domain.ListNotificationsResponse, error) {
	// TODO: Implement with GSI query
	return &domain.ListNotificationsResponse{
		Notifications: []*domain.Notification{},
		Pagination:    domain.PaginationResponse{},
	}, nil
}

// MarkAllAsRead marks all notifications as read for a user
func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID string) error {
	// TODO: Implement batch update
	return nil
}

// Ensure interface compliance
var _ domain.NotificationRepository = (*NotificationRepository)(nil)
