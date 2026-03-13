package domain

import (
	"context"
	"time"
)

// ==================== NOTIFICATION ENTITIES ====================

// NotificationType defines the type of notification
type NotificationType string

const (
	NotificationTypeEmail NotificationType = "EMAIL"
	NotificationTypeSMS   NotificationType = "SMS"
	NotificationTypePush  NotificationType = "PUSH"
)

// NotificationStatus defines the status of a notification
type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "PENDING"
	NotificationStatusSent      NotificationStatus = "SENT"
	NotificationStatusFailed    NotificationStatus = "FAILED"
	NotificationStatusDelivered NotificationStatus = "DELIVERED"
)

// NotificationTrigger defines what triggered the notification
type NotificationTrigger string

const (
	NotificationTriggerOrderStatus   NotificationTrigger = "ORDER_STATUS"
	NotificationTriggerOrderCreated  NotificationTrigger = "ORDER_CREATED"
	NotificationTriggerShipment      NotificationTrigger = "SHIPMENT"
	NotificationTriggerPayment       NotificationTrigger = "PAYMENT"
	NotificationTriggerRefund        NotificationTrigger = "REFUND"
	NotificationTriggerPasswordReset NotificationTrigger = "PASSWORD_RESET"
	NotificationTriggerManual        NotificationTrigger = "MANUAL"
)

// Notification represents a notification sent to a user/customer
type Notification struct {
	ID         string `json:"id" dynamodbav:"id"`
	PK         string `json:"-" dynamodbav:"PK"`
	SK         string `json:"-" dynamodbav:"SK"`
	GSI1PK     string `json:"-" dynamodbav:"GSI1PK"`
	GSI1SK     string `json:"-" dynamodbav:"GSI1SK"`
	EntityType string `json:"-" dynamodbav:"entity_type"`

	Type   NotificationType   `json:"type" dynamodbav:"type"`
	Status NotificationStatus `json:"status" dynamodbav:"status"`

	// Recipient
	RecipientID    string `json:"recipient_id" dynamodbav:"recipient_id"`
	RecipientEmail string `json:"recipient_email,omitempty" dynamodbav:"recipient_email,omitempty"`
	RecipientPhone string `json:"recipient_phone,omitempty" dynamodbav:"recipient_phone,omitempty"`

	// Content
	Subject      string                 `json:"subject,omitempty" dynamodbav:"subject,omitempty"`
	Body         string                 `json:"body" dynamodbav:"body"`
	TemplateID   string                 `json:"template_id,omitempty" dynamodbav:"template_id,omitempty"`
	TemplateData map[string]interface{} `json:"template_data,omitempty" dynamodbav:"template_data,omitempty"`

	// Trigger
	TriggerType   NotificationTrigger `json:"trigger_type" dynamodbav:"trigger_type"`
	ReferenceType string              `json:"reference_type,omitempty" dynamodbav:"reference_type,omitempty"`
	ReferenceID   string              `json:"reference_id,omitempty" dynamodbav:"reference_id,omitempty"`

	// Status tracking
	SentAt        *time.Time `json:"sent_at,omitempty" dynamodbav:"sent_at,omitempty"`
	DeliveredAt   *time.Time `json:"delivered_at,omitempty" dynamodbav:"delivered_at,omitempty"`
	FailedAt      *time.Time `json:"failed_at,omitempty" dynamodbav:"failed_at,omitempty"`
	FailureReason string     `json:"failure_reason,omitempty" dynamodbav:"failure_reason,omitempty"`

	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
	CreatedBy string    `json:"created_by" dynamodbav:"created_by"`
}

// TableName returns the DynamoDB table name for Notification
func (n *Notification) TableName() string {
	return TableCore
}

// SetKeys sets the DynamoDB keys for Notification
func (n *Notification) SetKeys() {
	n.PK = "NOTIFICATION#" + n.ID
	n.SK = SKMetadata
	n.GSI1PK = "RECIPIENT#" + n.RecipientID
	n.GSI1SK = n.CreatedAt.Format("2006-01-02T15:04:05Z")
	n.EntityType = "NOTIFICATION"
}

// NotificationTemplate represents a notification template
type NotificationTemplate struct {
	ID         string `json:"id" dynamodbav:"id"`
	PK         string `json:"-" dynamodbav:"PK"`
	SK         string `json:"-" dynamodbav:"SK"`
	EntityType string `json:"-" dynamodbav:"entity_type"`

	Name      string           `json:"name" dynamodbav:"name"`
	Type      NotificationType `json:"type" dynamodbav:"type"`
	Subject   string           `json:"subject,omitempty" dynamodbav:"subject,omitempty"`
	Body      string           `json:"body" dynamodbav:"body"`
	Variables []string         `json:"variables" dynamodbav:"variables"`
	IsActive  bool             `json:"is_active" dynamodbav:"is_active"`

	BaseEntity
}

// TableName returns the DynamoDB table name for NotificationTemplate
func (t *NotificationTemplate) TableName() string {
	return TableCore
}

// SetKeys sets the DynamoDB keys for NotificationTemplate
func (t *NotificationTemplate) SetKeys() {
	t.PK = "NOTIFICATION_TEMPLATE#" + t.ID
	t.SK = SKMetadata
	t.EntityType = "NOTIFICATION_TEMPLATE"
}

// ==================== NOTIFICATION REPOSITORY ====================

// NotificationRepository defines the interface for notification data access
type NotificationRepository interface {
	// Create creates a new notification
	Create(ctx context.Context, notification *Notification) error

	// GetByID retrieves a notification by ID
	GetByID(ctx context.Context, id string) (*Notification, error)

	// Update updates an existing notification
	Update(ctx context.Context, notification *Notification) error

	// List retrieves notifications with filters
	List(ctx context.Context, req ListNotificationsRequest) (*ListNotificationsResponse, error)

	// GetByUser retrieves notifications for a user
	GetByUser(ctx context.Context, userID string, pagination PaginationRequest) (*ListNotificationsResponse, error)

	// MarkAllAsRead marks all notifications for a user as read
	MarkAllAsRead(ctx context.Context, userID string) error
}

// ListNotificationsRequest contains parameters for listing notifications
type ListNotificationsRequest struct {
	PaginationRequest
	UserID      *string              `json:"user_id,omitempty"`
	Type        *NotificationType    `json:"type,omitempty"`
	Status      *NotificationStatus  `json:"status,omitempty"`
	TriggerType *NotificationTrigger `json:"trigger_type,omitempty"`
}

// ListNotificationsResponse contains the list of notifications
type ListNotificationsResponse struct {
	Notifications []*Notification    `json:"notifications"`
	Pagination    PaginationResponse `json:"pagination"`
	UnreadCount   int                `json:"unread_count"`
}

// ==================== NOTIFICATION SERVICE REQUESTS ====================

// SendNotificationRequest contains data for sending a notification
type SendNotificationRequest struct {
	Type           NotificationType       `json:"type" validate:"required"`
	RecipientID    string                 `json:"recipient_id" validate:"required"`
	RecipientEmail string                 `json:"recipient_email,omitempty"`
	RecipientPhone string                 `json:"recipient_phone,omitempty"`
	Subject        string                 `json:"subject,omitempty"`
	Body           string                 `json:"body,omitempty"`
	TemplateID     string                 `json:"template_id,omitempty"`
	TemplateData   map[string]interface{} `json:"template_data,omitempty"`
	TriggerType    NotificationTrigger    `json:"trigger_type,omitempty"`
	ReferenceType  string                 `json:"reference_type,omitempty"`
	ReferenceID    string                 `json:"reference_id,omitempty"`
}

// SendBulkNotificationRequest contains data for sending bulk notifications
type SendBulkNotificationRequest struct {
	Type         NotificationType       `json:"type" validate:"required"`
	RecipientIDs []string               `json:"recipient_ids" validate:"required,min=1"`
	Subject      string                 `json:"subject,omitempty"`
	Body         string                 `json:"body,omitempty"`
	TemplateID   string                 `json:"template_id,omitempty"`
	TemplateData map[string]interface{} `json:"template_data,omitempty"`
	TriggerType  NotificationTrigger    `json:"trigger_type,omitempty"`
}

// SendBulkNotificationResponse contains the result of bulk notification send
type SendBulkNotificationResponse struct {
	Total     int      `json:"total"`
	Sent      int      `json:"sent"`
	Failed    int      `json:"failed"`
	FailedIDs []string `json:"failed_ids,omitempty"`
}
