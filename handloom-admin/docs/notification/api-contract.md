# Notification Lambda API Documentation

User notification management service.

## Base Path
`/admin/notifications`

## Authentication
All endpoints require authentication.

---

### List Notifications

Get paginated list of all notifications.

**Endpoint:** `GET /admin/notifications`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| type | string | - | Filter by type |
| status | string | - | Filter by status (read, unread) |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "notif_abc123",
      "type": "order",
      "title": "New Order Received",
      "message": "Order #ORD-2024-001 has been placed",
      "data": {
        "order_id": "ord_abc123",
        "order_total": 15000.00
      },
      "recipient_id": "user_xyz789",
      "read": false,
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 50,
    "total_pages": 5
  }
}
```

---

### Send Notification

Send a notification to a user.

**Endpoint:** `POST /admin/notifications`
**Authentication:** Required

**Request Body:**
```json
{
  "recipient_id": "user_xyz789",
  "type": "system",
  "title": "System Maintenance",
  "message": "Scheduled maintenance on Jan 20, 2024 from 2:00 AM to 4:00 AM IST",
  "data": {
    "maintenance_start": "2024-01-20T02:00:00+05:30",
    "maintenance_end": "2024-01-20T04:00:00+05:30"
  }
}
```

**Response (201 Created):**
```json
{
  "id": "notif_xyz789",
  "type": "system",
  "title": "System Maintenance",
  "recipient_id": "user_xyz789",
  "created_at": "2024-01-15T10:30:00Z"
}
```

---

### Send Bulk Notifications

Send notifications to multiple users.

**Endpoint:** `POST /admin/notifications/bulk`
**Authentication:** Required

**Request Body:**
```json
{
  "recipient_ids": ["user_abc123", "user_xyz789", "user_def456"],
  "type": "promotion",
  "title": "New Year Sale!",
  "message": "Get up to 30% off on all silk sarees. Valid till Jan 31.",
  "data": {
    "promo_code": "NEWYEAR30",
    "valid_until": "2024-01-31T23:59:59Z"
  }
}
```

**Response (200 OK):**
```json
{
  "success_count": 3,
  "failure_count": 0,
  "notification_ids": ["notif_001", "notif_002", "notif_003"]
}
```

---

### Get Notification by ID

**Endpoint:** `GET /admin/notifications/{id}`
**Authentication:** Required

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | string | Notification ID |

**Response (200 OK):**
```json
{
  "id": "notif_abc123",
  "type": "order",
  "title": "New Order Received",
  "message": "Order #ORD-2024-001 has been placed",
  "data": {
    "order_id": "ord_abc123"
  },
  "recipient_id": "user_xyz789",
  "read": true,
  "read_at": "2024-01-15T11:00:00Z",
  "created_at": "2024-01-15T10:30:00Z"
}
```

---

### Mark Notification as Read

**Endpoint:** `POST /admin/notifications/{id}/read`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "id": "notif_abc123",
  "read": true,
  "read_at": "2024-01-15T11:00:00Z"
}
```

---

### Mark All Notifications as Read

**Endpoint:** `POST /admin/notifications/read-all`
**Authentication:** Required

**Response (200 OK):**
```json
{
  "marked_count": 15,
  "message": "All notifications marked as read"
}
```

---

### Get My Notifications

Get notifications for the authenticated user.

**Endpoint:** `GET /admin/notifications/my`
**Authentication:** Required

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| unread_only | bool | false | Show only unread |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "notif_abc123",
      "type": "order",
      "title": "New Order Received",
      "message": "Order #ORD-2024-001 has been placed",
      "read": false,
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "unread_count": 5,
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 25,
    "total_pages": 3
  }
}
```

---

## Notification Types

| Type | Description |
|------|-------------|
| `order` | Order-related notifications |
| `inventory` | Low stock / out of stock alerts |
| `system` | System announcements |
| `promotion` | Marketing promotions |
| `customer` | Customer-related updates |

---

## TODO

No pending TODO items identified.
