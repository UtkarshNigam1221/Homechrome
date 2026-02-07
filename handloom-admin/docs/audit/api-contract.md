# Audit Lambda API Documentation

Audit logging service for tracking system changes (Admin only).

## Base Path
`/admin/audit`

## Authentication
All endpoints require admin role authentication.

---

### List Audit Logs

Get paginated list of all audit logs.

**Endpoint:** `GET /admin/audit`
**Authentication:** Required (Admin only)

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| action | string | - | Filter by action type |
| entity_type | string | - | Filter by entity type |
| user_id | string | - | Filter by user who made the change |
| start_date | string | - | Filter from date (ISO 8601) |
| end_date | string | - | Filter until date (ISO 8601) |

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "audit_abc123",
      "action": "create",
      "entity_type": "product",
      "entity_id": "prod_xyz789",
      "user_id": "user_abc123",
      "user_name": "Admin User",
      "user_email": "admin@handloom.com",
      "changes": {
        "name": {
          "old": null,
          "new": "Kanchipuram Silk Saree"
        },
        "base_price": {
          "old": null,
          "new": 15000.00
        }
      },
      "metadata": {
        "ip_address": "192.168.1.1",
        "user_agent": "Mozilla/5.0..."
      },
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 1000,
    "total_pages": 100
  }
}
```

---

### Get Audit Log by ID

**Endpoint:** `GET /admin/audit/{id}`
**Authentication:** Required (Admin only)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | string | Audit log ID |

**Response (200 OK):**
```json
{
  "id": "audit_abc123",
  "action": "update",
  "entity_type": "product",
  "entity_id": "prod_xyz789",
  "user_id": "user_abc123",
  "user_name": "Admin User",
  "user_email": "admin@handloom.com",
  "changes": {
    "base_price": {
      "old": 15000.00,
      "new": 16500.00
    },
    "status": {
      "old": "draft",
      "new": "active"
    }
  },
  "previous_state": {
    "id": "prod_xyz789",
    "name": "Kanchipuram Silk Saree",
    "base_price": 15000.00,
    "status": "draft"
  },
  "new_state": {
    "id": "prod_xyz789",
    "name": "Kanchipuram Silk Saree",
    "base_price": 16500.00,
    "status": "active"
  },
  "metadata": {
    "ip_address": "192.168.1.1",
    "user_agent": "Mozilla/5.0...",
    "request_id": "req_abc123"
  },
  "created_at": "2024-01-15T10:30:00Z"
}
```

---

### Get Audit Logs by Entity

Get all audit logs for a specific entity.

**Endpoint:** `GET /admin/audit/entity/{type}/{id}`
**Authentication:** Required (Admin only)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| type | string | Entity type (product, order, user, etc.) |
| id | string | Entity ID |

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |

**Request:**
```
GET /admin/audit/entity/product/prod_abc123
```

**Response (200 OK):**
```json
{
  "entity_type": "product",
  "entity_id": "prod_abc123",
  "data": [
    {
      "id": "audit_001",
      "action": "create",
      "user_name": "Admin User",
      "changes": {...},
      "created_at": "2024-01-01T10:00:00Z"
    },
    {
      "id": "audit_002",
      "action": "update",
      "user_name": "Staff User",
      "changes": {
        "base_price": {
          "old": 15000.00,
          "new": 16500.00
        }
      },
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 5,
    "total_pages": 1
  }
}
```

---

### Get Audit Logs by User

Get all audit logs for actions performed by a specific user.

**Endpoint:** `GET /admin/audit/user/{id}`
**Authentication:** Required (Admin only)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | string | User ID |

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| action | string | - | Filter by action type |
| entity_type | string | - | Filter by entity type |

**Request:**
```
GET /admin/audit/user/user_abc123?action=delete
```

**Response (200 OK):**
```json
{
  "user_id": "user_abc123",
  "user_name": "Admin User",
  "data": [
    {
      "id": "audit_del_001",
      "action": "delete",
      "entity_type": "product",
      "entity_id": "prod_old001",
      "changes": null,
      "previous_state": {
        "id": "prod_old001",
        "name": "Discontinued Saree",
        "status": "inactive"
      },
      "created_at": "2024-01-10T15:00:00Z"
    }
  ],
  "summary": {
    "total_actions": 150,
    "creates": 45,
    "updates": 95,
    "deletes": 10
  },
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 10,
    "total_pages": 1
  }
}
```

---

## Action Types

| Action | Description |
|--------|-------------|
| `create` | New entity created |
| `update` | Entity modified |
| `delete` | Entity deleted |
| `login` | User logged in |
| `logout` | User logged out |
| `status_change` | Entity status changed |
| `bulk_import` | Bulk import operation |
| `bulk_export` | Bulk export operation |

## Entity Types

| Entity Type | Description |
|-------------|-------------|
| `user` | Admin/staff users |
| `product` | Products |
| `category` | Categories |
| `design` | Designs |
| `order` | Orders |
| `customer` | Customers |
| `inventory` | Inventory records |
| `pricing_rule` | Pricing rules |
| `coupon` | Coupons |
| `artisan` | Artisans |
| `asset` | Media assets |
| `notification` | Notifications |

---

## Audit Log Retention

- Audit logs are retained for 2 years
- Logs older than 2 years are automatically archived
- Archived logs can be requested through support

## Security Notes

- Only admin users can access audit logs
- Audit logs themselves are immutable and cannot be modified
- All access to audit logs is also logged

---

## TODO

No pending TODO items identified.
