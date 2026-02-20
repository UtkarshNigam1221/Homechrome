# User Lambda - High-Level Design (HLD)

## 1. Overview

The User Lambda service handles user management operations for the Handloom Admin system. It provides functionality for creating, updating, deleting, and listing admin users, as well as managing user roles, permissions, and account status.

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    USER MANAGEMENT SYSTEM                                    │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

                                    ┌───────────────────┐
                                    │   React Frontend  │
                                    │   (Admin Portal)  │
                                    └─────────┬─────────┘
                                              │
                                              │ HTTPS
                                              ▼
                                    ┌───────────────────┐
                                    │   API Gateway     │
                                    │   (REST API)      │
                                    └─────────┬─────────┘
                                              │
                         ┌────────────────────┼────────────────────┐
                         │                    │                    │
                         ▼                    ▼                    ▼
              ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
              │  User Lambda    │  │  User Lambda    │  │  User Lambda    │
              │  (CRUD)         │  │  (List/Search)  │  │  (Status)       │
              └────────┬────────┘  └────────┬────────┘  └────────┬────────┘
                       │                    │                    │
                       └────────────────────┼────────────────────┘
                                            │
                         ┌──────────────────┼──────────────────┐
                         │                  │                  │
                         ▼                  ▼                  ▼
              ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
              │   DynamoDB      │  │   CloudWatch    │  │   Audit Lambda  │
              │   (Users)       │  │   (Logs)        │  │   (Events)      │
              └─────────────────┘  └─────────────────┘  └─────────────────┘
```

---

## 3. Component Design

### 3.1 User Handler

```
┌─────────────────────────────────────────────────────────────────────┐
│                         USER HANDLER                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐ │
│  │   Create    │  │  GetByID    │  │   Update    │  │   Delete   │ │
│  │   Handler   │  │  Handler    │  │   Handler   │  │   Handler  │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └─────┬──────┘ │
│         │                │                │               │         │
│  ┌──────┴──────┐  ┌──────┴──────┐                                   │
│  │   List      │  │ UpdateStatus│                                   │
│  │   Handler   │  │   Handler   │                                   │
│  └──────┬──────┘  └──────┬──────┘                                   │
│         │                │                                           │
│         └────────────────┼────────────────────────────────┘         │
│                          │                                           │
│                          ▼                                           │
│                   ┌─────────────────────────────┐                   │
│                   │       User Service          │                   │
│                   │  - Create()                 │                   │
│                   │  - GetByID()                │                   │
│                   │  - Update()                 │                   │
│                   │  - Delete()                 │                   │
│                   │  - List()                   │                   │
│                   │  - UpdateStatus()           │                   │
│                   └──────────────┬──────────────┘                   │
│                                  │                                   │
│                                  ▼                                   │
│                        ┌───────────────┐                            │
│                        │ User          │                            │
│                        │ Repository    │                            │
│                        └───────────────┘                            │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 User Entity Structure

```
┌─────────────────────────────────────────────────────────────────────┐
│                          USER ENTITY                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Identification:                                                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ id            │ user_abc12345                                 │  │
│  │ email         │ john@example.com (unique)                     │  │
│  │ entity_type   │ USER                                          │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Personal Information:                                               │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ first_name    │ John                                          │  │
│  │ last_name     │ Smith                                         │  │
│  │ phone         │ +1-555-123-4567                               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Security:                                                           │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ password_hash │ $2a$12$...  (bcrypt)                          │  │
│  │ role          │ ADMIN | OPERATOR                              │  │
│  │ permissions   │ ["manage_products", "view_reports"]           │  │
│  │ status        │ ACTIVE | INACTIVE | PENDING                   │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Activity:                                                           │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ last_login_at │ 2024-01-15T10:30:00Z                          │  │
│  │ created_at    │ 2024-01-01T00:00:00Z                          │  │
│  │ created_by    │ user_admin123                                 │  │
│  │ updated_at    │ 2024-01-15T08:00:00Z                          │  │
│  │ updated_by    │ user_admin123                                 │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Model

### 4.1 DynamoDB Table Design

```
┌─────────────────────────────────────────────────────────────────────┐
│                    TABLE: handloom-admin                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  USER RECORDS                                                        │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ PK: USER#<user_id>                                          │    │
│  │ SK: METADATA                                                 │    │
│  │                                                             │    │
│  │ Attributes:                                                 │    │
│  │   - id                                                      │    │
│  │   - email                                                   │    │
│  │   - password_hash (excluded from API responses)             │    │
│  │   - first_name                                              │    │
│  │   - last_name                                               │    │
│  │   - phone                                                   │    │
│  │   - role (ADMIN | OPERATOR)                                 │    │
│  │   - permissions (list of strings)                           │    │
│  │   - status (ACTIVE | INACTIVE | PENDING)                    │    │
│  │   - last_login_at                                           │    │
│  │   - created_at, created_by                                  │    │
│  │   - updated_at, updated_by                                  │    │
│  │   - entity_type: "USER"                                     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  GSI1: Email Index (Lookup by Email)                                 │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ GSI1PK: USER_EMAIL                                          │    │
│  │ GSI1SK: <email>                                             │    │
│  │         (e.g., john@example.com)                            │    │
│  │                                                             │    │
│  │ Use case: Email uniqueness check, login lookup              │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 4.2 User Roles & Permissions

```
┌─────────────────────────────────────────────────────────────────────┐
│                    ROLES & PERMISSIONS                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Roles:                                                              │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ ADMIN     │ Full system access                                │  │
│  │           │ - Can manage all users                            │  │
│  │           │ - Can access all features                         │  │
│  │           │ - Can view audit logs                             │  │
│  │           │ - Can modify system settings                      │  │
│  ├───────────┼───────────────────────────────────────────────────┤  │
│  │ OPERATOR  │ Limited access based on permissions               │  │
│  │           │ - Cannot manage admin users                       │  │
│  │           │ - Cannot access audit logs                        │  │
│  │           │ - Access controlled by permissions list           │  │
│  └───────────┴───────────────────────────────────────────────────┘  │
│                                                                      │
│  Available Permissions:                                              │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ manage_products    │ Create, update, delete products          │  │
│  │ manage_orders      │ Update order status, process orders      │  │
│  │ manage_inventory   │ Adjust stock levels                      │  │
│  │ view_reports       │ Access analytics and reports             │  │
│  │ manage_coupons     │ Create and manage discount coupons       │  │
│  │ manage_artisans    │ Manage artisan/weaver information        │  │
│  │ manage_categories  │ Create and organize categories           │  │
│  │ manage_designs     │ Manage design patterns                   │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 4.3 User Status Lifecycle

```
┌─────────────────────────────────────────────────────────────────────┐
│                    USER STATUS LIFECYCLE                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│                          ┌─────────┐                                │
│                          │ PENDING │                                │
│                          └────┬────┘                                │
│                               │                                      │
│                    Admin activates user                             │
│                               │                                      │
│                               ▼                                      │
│                          ┌─────────┐                                │
│              ┌──────────▶│ ACTIVE  │◀──────────┐                    │
│              │           └────┬────┘           │                    │
│              │                │                │                    │
│         Reactivate       Deactivate        Reactivate              │
│              │                │                │                    │
│              │                ▼                │                    │
│              │          ┌──────────┐           │                    │
│              └──────────│ INACTIVE │───────────┘                    │
│                         └──────────┘                                │
│                                                                      │
│  Status Descriptions:                                                │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ PENDING   │ Newly created, awaiting admin activation          │  │
│  │ ACTIVE    │ Can log in and use the system                     │  │
│  │ INACTIVE  │ Account disabled, cannot log in                   │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 5. API Endpoints

```
┌─────────────────────────────────────────────────────────────────────┐
│                      USER API ENDPOINTS                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  List Users:                                                         │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ GET /admin/users                                              │  │
│  │                                                               │  │
│  │ Query Parameters:                                             │  │
│  │   - role: Filter by role (ADMIN, OPERATOR)                    │  │
│  │   - status: Filter by status (ACTIVE, INACTIVE, PENDING)      │  │
│  │   - search: Search in email, first_name, last_name            │  │
│  │   - page: Page number (default: 1)                            │  │
│  │   - per_page: Items per page (default: 20)                    │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Create User:                                                        │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ POST /admin/users                                             │  │
│  │                                                               │  │
│  │ Request Body:                                                 │  │
│  │   - email (required): Unique email address                    │  │
│  │   - password (required): Min 8 characters                     │  │
│  │   - first_name (required): User's first name                  │  │
│  │   - last_name (required): User's last name                    │  │
│  │   - phone (optional): Contact number                          │  │
│  │   - role (required): ADMIN or OPERATOR                        │  │
│  │   - permissions (optional): List of permission strings        │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Get User:                                                           │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ GET /admin/users/{id}                                         │  │
│  │                                                               │  │
│  │ Returns complete user profile (excluding password_hash)       │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Update User:                                                        │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ PATCH /admin/users/{id}                                       │  │
│  │                                                               │  │
│  │ Request Body (all optional):                                  │  │
│  │   - first_name                                                │  │
│  │   - last_name                                                 │  │
│  │   - phone                                                     │  │
│  │   - role                                                      │  │
│  │   - permissions                                               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Update Status:                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ PATCH /admin/users/{id}/status                                │  │
│  │                                                               │  │
│  │ Request Body:                                                 │  │
│  │   - status (required): ACTIVE, INACTIVE, or PENDING           │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Delete User:                                                        │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ DELETE /admin/users/{id}                                      │  │
│  │                                                               │  │
│  │ Permanently removes the user from the system                  │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 6. Security Design

### 6.1 Password Security

```
┌─────────────────────────────────────────────────────────────────────┐
│                      PASSWORD SECURITY                               │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Password Requirements:                                              │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Minimum 8 characters                                        │  │
│  │ • Recommended: uppercase, lowercase, numbers, special chars   │  │
│  │ • Password strength validation on frontend                    │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Hashing:                                                            │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                                                               │  │
│  │  bcrypt with default cost (10)                               │  │
│  │                                                               │  │
│  │  password ──▶ bcrypt.GenerateFromPassword() ──▶ hash         │  │
│  │                                                               │  │
│  │  Notes:                                                       │  │
│  │  • Salt is automatically included in bcrypt                   │  │
│  │  • Hash is 60 characters                                      │  │
│  │  • Password never logged or stored in plaintext               │  │
│  │                                                               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Response Sanitization:                                              │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • password_hash ALWAYS removed before API response            │  │
│  │ • Applies to: GetByID, List, Create, Update                   │  │
│  │ • Internal calls (auth) receive full user with hash           │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 6.2 Access Control

```
┌─────────────────────────────────────────────────────────────────────┐
│                      ACCESS CONTROL                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  User Management Permissions:                                        │
│  ┌─────────────────┬───────────────────────────────────────────┐    │
│  │ Operation       │ Required Role                             │    │
│  ├─────────────────┼───────────────────────────────────────────┤    │
│  │ List users      │ ADMIN only                                │    │
│  │ Create user     │ ADMIN only                                │    │
│  │ View user       │ ADMIN only                                │    │
│  │ Update user     │ ADMIN only                                │    │
│  │ Delete user     │ ADMIN only                                │    │
│  │ Change status   │ ADMIN only                                │    │
│  └─────────────────┴───────────────────────────────────────────┘    │
│                                                                      │
│  Self-Service Restrictions:                                          │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Users cannot delete their own account                       │  │
│  │ • Users cannot change their own role                          │  │
│  │ • Users cannot deactivate themselves                          │  │
│  │ • Profile updates (name, phone) allowed via /auth/me          │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 7. Error Handling

```
┌─────────────────────────────────────────────────────────────────────┐
│                      ERROR CODES                                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Validation Errors (400):                                            │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ USR001 │ Email is required                                  │    │
│  │ USR002 │ Invalid email format                               │    │
│  │ USR003 │ Password is required                               │    │
│  │ USR004 │ Password must be at least 8 characters             │    │
│  │ USR005 │ First name is required                             │    │
│  │ USR006 │ Last name is required                              │    │
│  │ USR007 │ Role is required                                   │    │
│  │ USR008 │ Invalid role (must be ADMIN or OPERATOR)           │    │
│  │ USR009 │ Status is required                                 │    │
│  │ USR010 │ Invalid status                                     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  Not Found Errors (404):                                             │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ USR101 │ User not found                                     │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  Conflict Errors (409):                                              │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ USR201 │ User with this email already exists                │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  Internal Errors (500):                                              │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ USR301 │ Failed to hash password                            │    │
│  │ USR302 │ Failed to create user                              │    │
│  │ USR303 │ Failed to update user                              │    │
│  │ USR304 │ Failed to delete user                              │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 8. Monitoring & Observability

```
┌─────────────────────────────────────────────────────────────────────┐
│                    MONITORING DESIGN                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Metrics:                                                            │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • User creation rate                                          │  │
│  │ • User deletion rate                                          │  │
│  │ • Status change frequency                                     │  │
│  │ • Active vs inactive users count                              │  │
│  │ • API response times                                          │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Alerts:                                                             │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Bulk user creation (potential abuse)                        │  │
│  │ • Multiple failed user operations                             │  │
│  │ • Admin role assignment                                       │  │
│  │ • Mass status changes                                         │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Audit Events:                                                       │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • User created (with role and created_by)                     │  │
│  │ • User updated (field changes logged)                         │  │
│  │ • User deleted (by whom)                                      │  │
│  │ • Status changes (old -> new status)                          │  │
│  │ • Role changes (old -> new role)                              │  │
│  │ • Permission changes                                          │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 9. Scalability Considerations

```
┌─────────────────────────────────────────────────────────────────────┐
│                    SCALABILITY DESIGN                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Lambda Configuration:                                               │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Memory: 256 MB                                              │  │
│  │ • Timeout: 30 seconds                                         │  │
│  │ • Concurrent executions: 50 (reserved)                        │  │
│  │ • Cold start optimization via provisioned concurrency         │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  DynamoDB Configuration:                                             │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Shared table with other entities (handloom-admin)           │  │
│  │ • On-demand capacity mode                                     │  │
│  │ • GSI1 for email lookups (critical for auth)                  │  │
│  │ • Conditional writes for uniqueness                           │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Query Optimization:                                                 │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Email lookup via GSI1 (constant time)                       │  │
│  │ • User list uses Scan with filters                            │  │
│  │ • Future: Consider GSI for role/status queries at scale       │  │
│  │ • Pagination for large result sets                            │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 10. Dependencies

```
┌─────────────────────────────────────────────────────────────────────┐
│                      DEPENDENCIES                                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  External Services:                                                  │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • AWS DynamoDB - User data storage                            │  │
│  │ • AWS CloudWatch - Logging & monitoring                       │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Internal Services:                                                  │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • Auth Lambda - Uses UserRepository for authentication        │  │
│  │ • Audit Lambda - Logs user management events                  │  │
│  │ • Notification Lambda - Welcome emails (future)               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Libraries:                                                          │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │ • golang.org/x/crypto/bcrypt - Password hashing               │  │
│  │ • github.com/google/uuid - User ID generation                 │  │
│  │ • aws-sdk-go-v2 - AWS service clients                         │  │
│  │ • go-chi/chi - HTTP routing                                   │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 11. Data Flow Summary

```
┌─────────────────────────────────────────────────────────────────────┐
│                    USER DATA FLOW                                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Create User Flow:                                                   │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                                                               │  │
│  │  Request ──▶ Validate ──▶ Check Email ──▶ Hash Password      │  │
│  │     │                          │              │               │  │
│  │     │                          │              ▼               │  │
│  │     │                          │         Build User          │  │
│  │     │                          │              │               │  │
│  │     │                          │              ▼               │  │
│  │     │                          └─────▶ Save to DynamoDB       │  │
│  │     │                                        │               │  │
│  │     │                                        ▼               │  │
│  │     └───────────────────────────────── Remove Hash           │  │
│  │                                              │               │  │
│  │                                              ▼               │  │
│  │                                        Return User           │  │
│  │                                                               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
│  Authentication Flow (via Auth Lambda):                              │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                                                               │  │
│  │  Email ──▶ GetByEmail (GSI1) ──▶ User with Hash               │  │
│  │                                        │                      │  │
│  │                                        ▼                      │  │
│  │  Password ──▶ bcrypt.Compare() ──▶ Match?                    │  │
│  │                                        │                      │  │
│  │                    ┌───────────────────┴───────────────────┐  │  │
│  │                    │                                       │  │  │
│  │                    ▼                                       ▼  │  │
│  │              Generate JWT                            Reject   │  │
│  │                    │                                          │  │
│  │                    ▼                                          │  │
│  │              Update LastLogin                                 │  │
│  │                                                               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```
