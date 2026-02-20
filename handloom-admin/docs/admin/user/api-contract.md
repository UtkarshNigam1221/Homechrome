# User Lambda API Documentation

User management service for admin users only.

## Base Path
`/admin/users`

## Authentication
All endpoints require admin role authentication.

## Endpoints

### List Users

Get a paginated list of all users.

**Endpoint:** `GET /admin/users`
**Authentication:** Required (Admin only)

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number |
| per_page | int | 10 | Items per page |
| role | string | - | Filter by role (admin, staff) |
| status | string | - | Filter by status (active, inactive) |

**Request:**
```
GET /admin/users?page=1&per_page=10&role=staff
```

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "user_abc123",
      "email": "staff@handloom.com",
      "name": "Staff User",
      "role": "staff",
      "status": "active",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 25,
    "total_pages": 3
  }
}
```

---

### Create User

Create a new user account.

**Endpoint:** `POST /admin/users`
**Authentication:** Required (Admin only)

**Request Body:**
```json
{
  "email": "newuser@handloom.com",
  "password": "securePassword123",
  "name": "New User",
  "role": "staff"
}
```

**Response (201 Created):**
```json
{
  "id": "user_xyz789",
  "email": "newuser@handloom.com",
  "name": "New User",
  "role": "staff",
  "status": "active",
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Error Responses:**
- `400 Bad Request` - Email already exists
- `422 Unprocessable Entity` - Validation failed

---

### Get User by ID

Retrieve a specific user's details.

**Endpoint:** `GET /admin/users/{id}`
**Authentication:** Required (Admin only)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | string | User ID |

**Response (200 OK):**
```json
{
  "id": "user_abc123",
  "email": "staff@handloom.com",
  "name": "Staff User",
  "role": "staff",
  "status": "active",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Error Responses:**
- `404 Not Found` - User not found

---

### Update User

Update user information.

**Endpoint:** `PUT /admin/users/{id}`
**Authentication:** Required (Admin only)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | string | User ID |

**Request Body:**
```json
{
  "name": "Updated Name",
  "role": "admin",
  "status": "active"
}
```

**Response (200 OK):**
```json
{
  "id": "user_abc123",
  "email": "staff@handloom.com",
  "name": "Updated Name",
  "role": "admin",
  "status": "active",
  "updated_at": "2024-01-15T11:00:00Z"
}
```

---

### Delete User

Delete a user account.

**Endpoint:** `DELETE /admin/users/{id}`
**Authentication:** Required (Admin only)

**Path Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| id | string | User ID |

**Response (204 No Content)**

**Error Responses:**
- `404 Not Found` - User not found
- `400 Bad Request` - Cannot delete own account

---

## TODO

No pending TODO items identified.
