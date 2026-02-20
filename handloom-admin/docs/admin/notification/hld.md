# Notification Lambda - High Level Design

## 1. Overview

The Notification Lambda provides a comprehensive notification system for the Handloom Admin platform. It handles creation, delivery, and management of notifications across multiple channels including in-app, email, and push notifications.

### Key Features
- Multi-channel notification delivery (in-app, email, push)
- System-triggered and manual notifications
- User preference management
- Notification templates
- Read/unread status tracking
- Notification history and analytics

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        NOTIFICATION LAMBDA ARCHITECTURE                      │
└─────────────────────────────────────────────────────────────────────────────┘

                              ┌──────────────┐
                              │   Client     │
                              │  (Browser)   │
                              └──────┬───────┘
                                     │
                                     ▼
                              ┌──────────────┐
                              │  CloudFront  │
                              │     CDN      │
                              └──────┬───────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              API Gateway                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  /notifications             GET     - Get user notifications         │    │
│  │  /notifications             POST    - Create notification (admin)    │    │
│  │  /notifications/{id}        GET     - Get notification details       │    │
│  │  /notifications/{id}/read   PATCH   - Mark as read                   │    │
│  │  /notifications/mark-all    POST    - Mark all as read               │    │
│  │  /notifications/preferences GET/PUT - User preferences               │    │
│  │  /notifications/unread      GET     - Get unread count               │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Notification Lambda                                 │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │                         Handler Layer                               │     │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                │     │
│  │  │    List      │ │   Create     │ │   Read/      │                │     │
│  │  │   Handler    │ │   Handler    │ │   Unread     │                │     │
│  │  └──────────────┘ └──────────────┘ └──────────────┘                │     │
│  │  ┌──────────────┐ ┌──────────────┐                                 │     │
│  │  │  Preference  │ │   Delete     │                                 │     │
│  │  │   Handler    │ │   Handler    │                                 │     │
│  │  └──────────────┘ └──────────────┘                                 │     │
│  └────────────────────────────────────────────────────────────────────┘     │
│                                    │                                         │
│                                    ▼                                         │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │                        Service Layer                                │     │
│  │  ┌──────────────────────────────────────────────────────────────┐  │     │
│  │  │                  Notification Service                         │  │     │
│  │  │  - CreateNotification()      - GetUserNotifications()        │  │     │
│  │  │  - MarkAsRead()              - MarkAllAsRead()               │  │     │
│  │  │  - GetUnreadCount()          - DeleteNotification()          │  │     │
│  │  │  - UpdatePreferences()       - SendNotification()            │  │     │
│  │  └──────────────────────────────────────────────────────────────┘  │     │
│  └────────────────────────────────────────────────────────────────────┘     │
│                                    │                                         │
│                                    ▼                                         │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │                      Repository Layer                               │     │
│  │  ┌──────────────────────────────────────────────────────────────┐  │     │
│  │  │                Notification Repository                        │  │     │
│  │  │  - Create()          - GetByUserID()      - GetByID()        │  │     │
│  │  │  - Update()          - Delete()           - BatchCreate()    │  │     │
│  │  │  - GetPreferences()  - UpdatePreferences()                   │  │     │
│  │  └──────────────────────────────────────────────────────────────┘  │     │
│  └────────────────────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                     ┌───────────────┼───────────────┐
                     │               │               │
                     ▼               ▼               ▼
              ┌──────────┐    ┌──────────┐    ┌──────────┐
              │ DynamoDB │    │   SES    │    │   SNS    │
              │ (Storage)│    │ (Email)  │    │  (Push)  │
              └──────────┘    └──────────┘    └──────────┘

                              ┌──────────────┐
                              │ EventBridge  │
                              │ (Triggers)   │
                              └──────────────┘
```

---

## 3. Component Design

### 3.1 Notification Handler

```go
type NotificationHandler struct {
    notificationService domain.NotificationService
    logger              *logger.Logger
}

// Handler Methods
- GetNotifications(c *gin.Context)
- GetNotification(c *gin.Context)
- CreateNotification(c *gin.Context)
- MarkAsRead(c *gin.Context)
- MarkAllAsRead(c *gin.Context)
- DeleteNotification(c *gin.Context)
- GetUnreadCount(c *gin.Context)
- GetPreferences(c *gin.Context)
- UpdatePreferences(c *gin.Context)
```

### 3.2 Notification Service

```go
type NotificationService interface {
    // Notification Management
    CreateNotification(ctx context.Context, req *CreateNotificationRequest) (*Notification, error)
    GetNotifications(ctx context.Context, userID string, filter *NotificationFilter) (*NotificationList, error)
    GetNotification(ctx context.Context, id string) (*Notification, error)
    DeleteNotification(ctx context.Context, id string, userID string) error

    // Read Status
    MarkAsRead(ctx context.Context, id string, userID string) error
    MarkAllAsRead(ctx context.Context, userID string) (int, error)
    GetUnreadCount(ctx context.Context, userID string) (int, error)

    // Preferences
    GetPreferences(ctx context.Context, userID string) (*NotificationPreferences, error)
    UpdatePreferences(ctx context.Context, userID string, prefs *NotificationPreferences) error

    // Delivery
    SendNotification(ctx context.Context, notification *Notification) error
}
```

### 3.3 Notification Repository

```go
type NotificationRepository interface {
    Create(ctx context.Context, notification *Notification) error
    BatchCreate(ctx context.Context, notifications []*Notification) error
    GetByID(ctx context.Context, id string) (*Notification, error)
    GetByUserID(ctx context.Context, userID string, filter *NotificationFilter) ([]*Notification, error)
    Update(ctx context.Context, notification *Notification) error
    Delete(ctx context.Context, id string) error
    CountUnread(ctx context.Context, userID string) (int, error)
    GetPreferences(ctx context.Context, userID string) (*NotificationPreferences, error)
    UpdatePreferences(ctx context.Context, userID string, prefs *NotificationPreferences) error
}
```

---

## 4. Data Model

### 4.1 Notification Entity

```go
type Notification struct {
    ID           string            `json:"id" dynamodbav:"id"`
    UserID       string            `json:"user_id" dynamodbav:"user_id"`
    Type         NotificationType  `json:"type" dynamodbav:"type"`
    Title        string            `json:"title" dynamodbav:"title"`
    Message      string            `json:"message" dynamodbav:"message"`
    Data         map[string]string `json:"data,omitempty" dynamodbav:"data,omitempty"`
    Status       NotificationStatus`json:"status" dynamodbav:"status"`
    Priority     Priority          `json:"priority" dynamodbav:"priority"`
    ReadAt       *time.Time        `json:"read_at,omitempty" dynamodbav:"read_at,omitempty"`
    DeliveredAt  *time.Time        `json:"delivered_at,omitempty" dynamodbav:"delivered_at,omitempty"`
    CreatedAt    time.Time         `json:"created_at" dynamodbav:"created_at"`
    ExpiresAt    *time.Time        `json:"expires_at,omitempty" dynamodbav:"expires_at,omitempty"`
}
```

### 4.2 Notification Types

```go
type NotificationType string

const (
    NotificationTypeOrderCreated    NotificationType = "ORDER_CREATED"
    NotificationTypeOrderShipped    NotificationType = "ORDER_SHIPPED"
    NotificationTypeOrderDelivered  NotificationType = "ORDER_DELIVERED"
    NotificationTypeOrderCancelled  NotificationType = "ORDER_CANCELLED"
    NotificationTypePaymentReceived NotificationType = "PAYMENT_RECEIVED"
    NotificationTypeLowStock        NotificationType = "LOW_STOCK"
    NotificationTypeOutOfStock      NotificationType = "OUT_OF_STOCK"
    NotificationTypeNewReview       NotificationType = "NEW_REVIEW"
    NotificationTypeSystemAlert     NotificationType = "SYSTEM_ALERT"
    NotificationTypePromotion       NotificationType = "PROMOTION"
    NotificationTypeAnnouncement    NotificationType = "ANNOUNCEMENT"
)
```

### 4.3 Notification Status

```go
type NotificationStatus string

const (
    StatusCreated     NotificationStatus = "CREATED"
    StatusQueued      NotificationStatus = "QUEUED"
    StatusDelivered   NotificationStatus = "DELIVERED"
    StatusRead        NotificationStatus = "READ"
    StatusFailed      NotificationStatus = "FAILED"
    StatusArchived    NotificationStatus = "ARCHIVED"
)
```

### 4.4 Notification Preferences

```go
type NotificationPreferences struct {
    UserID       string                   `json:"user_id" dynamodbav:"user_id"`
    Email        ChannelPreferences       `json:"email" dynamodbav:"email"`
    Push         ChannelPreferences       `json:"push" dynamodbav:"push"`
    InApp        ChannelPreferences       `json:"in_app" dynamodbav:"in_app"`
    QuietHours   *QuietHours              `json:"quiet_hours,omitempty" dynamodbav:"quiet_hours,omitempty"`
    UpdatedAt    time.Time                `json:"updated_at" dynamodbav:"updated_at"`
}

type ChannelPreferences struct {
    Enabled      bool     `json:"enabled" dynamodbav:"enabled"`
    Types        []string `json:"types" dynamodbav:"types"`
}

type QuietHours struct {
    Enabled   bool   `json:"enabled" dynamodbav:"enabled"`
    StartTime string `json:"start_time" dynamodbav:"start_time"` // "22:00"
    EndTime   string `json:"end_time" dynamodbav:"end_time"`     // "08:00"
    Timezone  string `json:"timezone" dynamodbav:"timezone"`
}
```

### 4.5 Create Notification Request

```go
type CreateNotificationRequest struct {
    Type        NotificationType  `json:"type" binding:"required"`
    Title       string            `json:"title" binding:"required,max=200"`
    Message     string            `json:"message" binding:"required,max=1000"`
    Recipients  []string          `json:"recipients,omitempty"`
    AllUsers    bool              `json:"all_users,omitempty"`
    UserSegment string            `json:"user_segment,omitempty"`
    Priority    Priority          `json:"priority,omitempty"`
    Data        map[string]string `json:"data,omitempty"`
    ScheduleAt  *time.Time        `json:"schedule_at,omitempty"`
    ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
    Channels    []string          `json:"channels,omitempty"` // email, push, in_app
}
```

---

## 5. DynamoDB Schema

### 5.1 Notification Table

```
Table: handloom-notifications

Primary Key:
- PK: NOTIF#<notification_id>
- SK: USER#<user_id>

Attributes:
- id: string
- user_id: string
- type: string
- title: string
- message: string
- data: map
- status: string
- priority: string
- read_at: string (ISO8601)
- delivered_at: string (ISO8601)
- created_at: string (ISO8601)
- expires_at: string (ISO8601)
- ttl: number (for auto-expiry)

GSI1: user-notifications-index
- PK: USER#<user_id>
- SK: created_at

GSI2: type-status-index
- PK: type
- SK: status#created_at
```

### 5.2 Notification Preferences Table

```
Table: handloom-notification-preferences

Primary Key:
- PK: USER#<user_id>
- SK: PREFERENCES

Attributes:
- user_id: string
- email: map
- push: map
- in_app: map
- quiet_hours: map
- updated_at: string
```

### 5.3 Access Patterns

| Access Pattern | Key Condition | Index |
|----------------|---------------|-------|
| Get notification by ID | PK = NOTIF#{id} | Main |
| Get user notifications | PK = USER#{user_id} | GSI1 |
| Get unread notifications | GSI1 + filter status | GSI1 |
| Get notifications by type | PK = {type} | GSI2 |
| Get user preferences | PK = USER#{user_id}, SK = PREFERENCES | Main |

---

## 6. API Endpoints

### 6.1 Get User Notifications

```
GET /notifications?status=unread&limit=20&offset=0

Response:
{
    "success": true,
    "data": {
        "notifications": [
            {
                "id": "notif_123",
                "type": "ORDER_CREATED",
                "title": "New Order #ORD-1234",
                "message": "Order placed by customer Priya Sharma",
                "status": "DELIVERED",
                "priority": "HIGH",
                "data": {
                    "order_id": "ORD-1234",
                    "customer_name": "Priya Sharma"
                },
                "created_at": "2024-01-15T10:30:00Z"
            }
        ],
        "unread_count": 5,
        "total": 50,
        "has_more": true
    }
}
```

### 6.2 Create Notification (Admin)

```
POST /notifications

Request:
{
    "type": "ANNOUNCEMENT",
    "title": "Summer Sale Announcement",
    "message": "Get 20% off on all silk sarees this weekend!",
    "all_users": true,
    "priority": "NORMAL",
    "channels": ["in_app", "email"]
}

Response:
{
    "success": true,
    "data": {
        "id": "notif_456",
        "recipients_count": 789,
        "status": "QUEUED"
    }
}
```

### 6.3 Mark as Read

```
PATCH /notifications/{id}/read

Response:
{
    "success": true,
    "data": {
        "id": "notif_123",
        "read_at": "2024-01-15T11:00:00Z"
    }
}
```

### 6.4 Mark All as Read

```
POST /notifications/mark-all-read

Response:
{
    "success": true,
    "data": {
        "updated_count": 5
    }
}
```

### 6.5 Get Unread Count

```
GET /notifications/unread-count

Response:
{
    "success": true,
    "data": {
        "unread_count": 5
    }
}
```

### 6.6 Get/Update Preferences

```
GET /notifications/preferences

Response:
{
    "success": true,
    "data": {
        "email": {
            "enabled": true,
            "types": ["ORDER_CREATED", "PAYMENT_RECEIVED", "LOW_STOCK"]
        },
        "push": {
            "enabled": true,
            "types": ["ORDER_CREATED", "PAYMENT_RECEIVED"]
        },
        "in_app": {
            "enabled": true,
            "types": ["*"]
        },
        "quiet_hours": {
            "enabled": true,
            "start_time": "22:00",
            "end_time": "08:00",
            "timezone": "Asia/Kolkata"
        }
    }
}
```

```
PUT /notifications/preferences

Request:
{
    "email": {
        "enabled": true,
        "types": ["ORDER_CREATED", "PAYMENT_RECEIVED"]
    },
    "push": {
        "enabled": false
    },
    "quiet_hours": {
        "enabled": true,
        "start_time": "22:00",
        "end_time": "08:00"
    }
}
```

---

## 7. Event-Driven Notifications

### 7.1 Event Types

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          EVENT TRIGGERS                                      │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ Order Events    │     │ Inventory Events│     │ Payment Events  │
│                 │     │                 │     │                 │
│ • OrderCreated  │     │ • LowStock      │     │ • PaymentReceived│
│ • OrderShipped  │     │ • OutOfStock    │     │ • PaymentFailed │
│ • OrderDelivered│     │ • StockRestored │     │ • RefundProcessed│
│ • OrderCancelled│     │                 │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘

┌─────────────────┐     ┌─────────────────┐
│ Customer Events │     │ System Events   │
│                 │     │                 │
│ • NewReview     │     │ • Maintenance   │
│ • NewCustomer   │     │ • SystemAlert   │
│                 │     │ • Announcement  │
└─────────────────┘     └─────────────────┘
```

### 7.2 Event Processing Flow

```
Event Source ──> EventBridge ──> Lambda ──> Notification Service
                                    │
                                    ├──> Check user preferences
                                    │
                                    ├──> Apply quiet hours
                                    │
                                    ├──> Create notification records
                                    │
                                    └──> Queue for delivery
```

---

## 8. Error Handling

### 8.1 Error Types

| Error Code | Description | HTTP Status |
|------------|-------------|-------------|
| NOTIFICATION_NOT_FOUND | Notification does not exist | 404 |
| UNAUTHORIZED_ACCESS | User cannot access this notification | 403 |
| INVALID_TYPE | Unknown notification type | 400 |
| DELIVERY_FAILED | Failed to deliver notification | 500 |
| PREFERENCE_NOT_FOUND | User preferences not set | 404 |
| INVALID_RECIPIENTS | No valid recipients found | 400 |

### 8.2 Error Response Format

```json
{
    "success": false,
    "error": {
        "code": "NOTIFICATION_NOT_FOUND",
        "message": "Notification with ID 'notif_123' not found"
    }
}
```

---

## 9. Delivery Channels

### 9.1 Channel Configuration

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         DELIVERY CHANNELS                                    │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                              IN-APP                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ • Real-time via WebSocket                                            │    │
│  │ • Persistent in database                                             │    │
│  │ • Badge count updates                                                │    │
│  │ • Priority: All notification types                                   │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                               EMAIL                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ • AWS SES integration                                                │    │
│  │ • HTML templates with branding                                       │    │
│  │ • Unsubscribe link included                                          │    │
│  │ • Priority: Order updates, alerts, announcements                     │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                               PUSH                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ • AWS SNS integration                                                │    │
│  │ • Mobile app notifications                                           │    │
│  │ • Respects quiet hours                                               │    │
│  │ • Priority: Urgent alerts only                                       │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 10. Security

### 10.1 Access Control

| Role | View Own | View All | Create | Delete Own | Delete All |
|------|----------|----------|--------|------------|------------|
| Admin | Yes | Yes | Yes | Yes | Yes |
| Manager | Yes | Yes | Yes | Yes | No |
| Staff | Yes | No | No | Yes | No |

### 10.2 Data Privacy

- Users can only view their own notifications
- Admin can send to all users
- Notification content logged without PII
- Preferences are user-specific

---

## 11. Performance Optimization

### 11.1 Batch Operations

```go
// Batch create notifications for multiple users
func (s *notificationService) BatchCreate(ctx context.Context, notifications []*Notification) error {
    // Process in batches of 25 (DynamoDB limit)
    batchSize := 25
    for i := 0; i < len(notifications); i += batchSize {
        end := i + batchSize
        if end > len(notifications) {
            end = len(notifications)
        }
        batch := notifications[i:end]
        if err := s.repo.BatchCreate(ctx, batch); err != nil {
            return err
        }
    }
    return nil
}
```

### 11.2 Caching Strategy

- Cache unread count with 1-minute TTL
- Cache user preferences with 5-minute TTL
- Invalidate on updates

---

## 12. Monitoring

### 12.1 Key Metrics

| Metric | Description | Threshold |
|--------|-------------|-----------|
| Delivery Rate | % of notifications delivered | > 99% |
| Delivery Latency | Time from creation to delivery | < 5s |
| Email Bounce Rate | % of bounced emails | < 1% |
| Unread Accumulation | Avg unread per user | < 50 |

### 12.2 Alerts

- High email bounce rate
- Delivery queue backlog
- Failed batch operations
- SES/SNS service errors

---

## 13. Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              DEPENDENCIES                                    │
└─────────────────────────────────────────────────────────────────────────────┘

                       Notification Lambda
                              │
          ┌───────────────────┼───────────────────┐
          │                   │                   │
          ▼                   ▼                   ▼
   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
   │  DynamoDB   │    │    SES      │    │    SNS      │
   │  (Storage)  │    │  (Email)    │    │   (Push)    │
   └─────────────┘    └─────────────┘    └─────────────┘
          │
          ▼
   ┌─────────────┐
   │ EventBridge │
   │ (Triggers)  │
   └─────────────┘
```

### Internal Dependencies
- User Service: User details and preferences
- Order Service: Order event triggers
- Inventory Service: Stock alert triggers

### External Dependencies
- AWS DynamoDB: Notification storage
- AWS SES: Email delivery
- AWS SNS: Push notifications
- AWS EventBridge: Event triggers
- AWS CloudWatch: Logging and monitoring

