# User Lambda - Sequence Diagrams

## Overview
This document contains sequence diagrams for all user management operations.

---

## 1. Create User Sequence

```
┌────────┐          ┌─────────┐          ┌─────────────┐          ┌──────────────┐          ┌──────────┐
│ Client │          │ API GW  │          │ User Lambda │          │ User Repo    │          │ DynamoDB │
└───┬────┘          └────┬────┘          └──────┬──────┘          └──────┬───────┘          └────┬─────┘
    │                    │                      │                        │                       │
    │ POST /admin/users  │                      │                        │                       │
    │ {email, password,  │                      │                        │                       │
    │  firstName, role}  │                      │                        │                       │
    │───────────────────▶│                      │                        │                       │
    │                    │                      │                        │                       │
    │                    │  Forward request     │                        │                       │
    │                    │─────────────────────▶│                        │                       │
    │                    │                      │                        │                       │
    │                    │                      │  Validate input        │                       │
    │                    │                      │  - Email format        │                       │
    │                    │                      │  - Password strength   │                       │
    │                    │                      │  - Required fields     │                       │
    │                    │                      │──────────┐             │                       │
    │                    │                      │          │             │                       │
    │                    │                      │◀─────────┘             │                       │
    │                    │                      │                        │                       │
    │                    │                      │  GetByEmail(email)     │                       │
    │                    │                      │───────────────────────▶│                       │
    │                    │                      │                        │                       │
    │                    │                      │                        │  Query GSI1           │
    │                    │                      │                        │  GSI1PK: USER_EMAIL   │
    │                    │                      │                        │  GSI1SK: email        │
    │                    │                      │                        │──────────────────────▶│
    │                    │                      │                        │                       │
    │                    │                      │                        │  null (not found)     │
    │                    │                      │                        │◀──────────────────────│
    │                    │                      │                        │                       │
    │                    │                      │  null (OK to create)   │                       │
    │                    │                      │◀───────────────────────│                       │
    │                    │                      │                        │                       │
    │                    │                      │  Hash password         │                       │
    │                    │                      │  (bcrypt)              │                       │
    │                    │                      │──────────┐             │                       │
    │                    │                      │          │             │                       │
    │                    │                      │◀─────────┘             │                       │
    │                    │                      │                        │                       │
    │                    │                      │  Build User entity     │                       │
    │                    │                      │  - ID: user_<uuid>     │                       │
    │                    │                      │  - Status: PENDING     │                       │
    │                    │                      │──────────┐             │                       │
    │                    │                      │          │             │                       │
    │                    │                      │◀─────────┘             │                       │
    │                    │                      │                        │                       │
    │                    │                      │  Create(user)          │                       │
    │                    │                      │───────────────────────▶│                       │
    │                    │                      │                        │                       │
    │                    │                      │                        │  PutItem with         │
    │                    │                      │                        │  condition:           │
    │                    │                      │                        │  attr_not_exists(PK)  │
    │                    │                      │                        │──────────────────────▶│
    │                    │                      │                        │                       │
    │                    │                      │                        │  Success              │
    │                    │                      │                        │◀──────────────────────│
    │                    │                      │                        │                       │
    │                    │                      │  Success               │                       │
    │                    │                      │◀───────────────────────│                       │
    │                    │                      │                        │                       │
    │                    │  {user} (no password)│                        │                       │
    │                    │◀─────────────────────│                        │                       │
    │                    │                      │                        │                       │
    │ 201 Created        │                      │                        │                       │
    │ {user}             │                      │                        │                       │
    │◀───────────────────│                      │                        │                       │
    │                    │                      │                        │                       │
```

---

## 2. List Users Sequence

```
┌────────┐          ┌─────────┐          ┌─────────────┐          ┌──────────────┐          ┌──────────┐
│ Client │          │ API GW  │          │ User Lambda │          │ User Repo    │          │ DynamoDB │
└───┬────┘          └────┬────┘          └──────┬──────┘          └──────┬───────┘          └────┬─────┘
    │                    │                      │                        │                       │
    │ GET /admin/users   │                      │                        │                       │
    │ ?role=OPERATOR     │                      │                        │                       │
    │ &status=ACTIVE     │                      │                        │                       │
    │ &search=john       │                      │                        │                       │
    │───────────────────▶│                      │                        │                       │
    │                    │                      │                        │                       │
    │                    │  Forward request     │                        │                       │
    │                    │─────────────────────▶│                        │                       │
    │                    │                      │                        │                       │
    │                    │                      │  Parse query params    │                       │
    │                    │                      │  - role                │                       │
    │                    │                      │  - status              │                       │
    │                    │                      │  - search              │                       │
    │                    │                      │  - pagination          │                       │
    │                    │                      │──────────┐             │                       │
    │                    │                      │          │             │                       │
    │                    │                      │◀─────────┘             │                       │
    │                    │                      │                        │                       │
    │                    │                      │  List(request)         │                       │
    │                    │                      │───────────────────────▶│                       │
    │                    │                      │                        │                       │
    │                    │                      │                        │  Build filter expr    │
    │                    │                      │                        │  - entity_type=USER   │
    │                    │                      │                        │  - role=OPERATOR      │
    │                    │                      │                        │  - status=ACTIVE      │
    │                    │                      │                        │  - contains(search)   │
    │                    │                      │                        │──────────┐            │
    │                    │                      │                        │          │            │
    │                    │                      │                        │◀─────────┘            │
    │                    │                      │                        │                       │
    │                    │                      │                        │  Scan with filters    │
    │                    │                      │                        │──────────────────────▶│
    │                    │                      │                        │                       │
    │                    │                      │                        │  User items           │
    │                    │                      │                        │◀──────────────────────│
    │                    │                      │                        │                       │
    │                    │                      │                        │  Apply pagination     │
    │                    │                      │                        │──────────┐            │
    │                    │                      │                        │          │            │
    │                    │                      │                        │◀─────────┘            │
    │                    │                      │                        │                       │
    │                    │                      │  {users, pagination}   │                       │
    │                    │                      │◀───────────────────────│                       │
    │                    │                      │                        │                       │
    │                    │                      │  Remove password hashes│                       │
    │                    │                      │──────────┐             │                       │
    │                    │                      │          │             │                       │
    │                    │                      │◀─────────┘             │                       │
    │                    │                      │                        │                       │
    │                    │  {users, pagination} │                        │                       │
    │                    │◀─────────────────────│                        │                       │
    │                    │                      │                        │                       │
    │ 200 OK             │                      │                        │                       │
    │ {users, pagination}│                      │                        │                       │
    │◀───────────────────│                      │                        │                       │
    │                    │                      │                        │                       │
```

---

## 3. Get User by ID Sequence

```
┌────────┐          ┌─────────┐          ┌─────────────┐          ┌──────────────┐          ┌──────────┐
│ Client │          │ API GW  │          │ User Lambda │          │ User Repo    │          │ DynamoDB │
└───┬────┘          └────┬────┘          └──────┬──────┘          └──────┬───────┘          └────┬─────┘
    │                    │                      │                        │                       │
    │ GET /admin/users/  │                      │                        │                       │
    │     user_abc123    │                      │                        │                       │
    │───────────────────▶│                      │                        │                       │
    │                    │                      │                        │                       │
    │                    │  Forward request     │                        │                       │
    │                    │─────────────────────▶│                        │                       │
    │                    │                      │                        │                       │
    │                    │                      │  GetByID(user_abc123)  │                       │
    │                    │                      │───────────────────────▶│                       │
    │                    │                      │                        │                       │
    │                    │                      │                        │  GetItem              │
    │                    │                      │                        │  PK: USER#user_abc123 │
    │                    │                      │                        │  SK: METADATA         │
    │                    │                      │                        │──────────────────────▶│
    │                    │                      │                        │                       │
    │                    │                      │                        │  User item            │
    │                    │                      │                        │◀──────────────────────│
    │                    │                      │                        │                       │
    │                    │                      │  User record           │                       │
    │                    │                      │◀───────────────────────│                       │
    │                    │                      │                        │                       │
    │                    │                      │  Remove password hash  │                       │
    │                    │                      │──────────┐             │                       │
    │                    │                      │          │             │                       │
    │                    │                      │◀─────────┘             │                       │
    │                    │                      │                        │                       │
    │                    │  {user}              │                        │                       │
    │                    │◀─────────────────────│                        │                       │
    │                    │                      │                        │                       │
    │ 200 OK {user}      │                      │                        │                       │
    │◀───────────────────│                      │                        │                       │
    │                    │                      │                        │                       │
```

---

## 4. Update User Sequence

```
┌────────┐          ┌─────────┐          ┌─────────────┐          ┌──────────────┐          ┌──────────┐
│ Client │          │ API GW  │          │ User Lambda │          │ User Repo    │          │ DynamoDB │
└───┬────┘          └────┬────┘          └──────┬──────┘          └──────┬───────┘          └────┬─────┘
    │                    │                      │                        │                       │
    │ PATCH /admin/users/│                      │                        │                       │
    │      user_abc123   │                      │                        │                       │
    │ {firstName, role}  │                      │                        │                       │
    │───────────────────▶│                      │                        │                       │
    │                    │                      │                        │                       │
    │                    │  Forward request     │                        │                       │
    │                    │─────────────────────▶│                        │                       │
    │                    │                      │                        │                       │
    │                    │                      │  GetByID(user_abc123)  │                       │
    │                    │                      │───────────────────────▶│                       │
    │                    │                      │                        │                       │
    │                    │                      │                        │  GetItem              │
    │                    │                      │                        │──────────────────────▶│
    │                    │                      │                        │                       │
    │                    │                      │                        │  User item            │
    │                    │                      │                        │◀──────────────────────│
    │                    │                      │                        │                       │
    │                    │                      │  Existing user         │                       │
    │                    │                      │◀───────────────────────│                       │
    │                    │                      │                        │                       │
    │                    │                      │  Apply updates         │                       │
    │                    │                      │  - firstName           │                       │
    │                    │                      │  - lastName            │                       │
    │                    │                      │  - phone               │                       │
    │                    │                      │  - role                │                       │
    │                    │                      │  - permissions         │                       │
    │                    │                      │  - updatedAt           │                       │
    │                    │                      │  - updatedBy           │                       │
    │                    │                      │──────────┐             │                       │
    │                    │                      │          │             │                       │
    │                    │                      │◀─────────┘             │                       │
    │                    │                      │                        │                       │
    │                    │                      │  Update(user)          │                       │
    │                    │                      │───────────────────────▶│                       │
    │                    │                      │                        │                       │
    │                    │                      │                        │  PutItem with         │
    │                    │                      │                        │  condition:           │
    │                    │                      │                        │  attr_exists(PK)      │
    │                    │                      │                        │──────────────────────▶│
    │                    │                      │                        │                       │
    │                    │                      │                        │  Success              │
    │                    │                      │                        │◀──────────────────────│
    │                    │                      │                        │                       │
    │                    │                      │  Success               │                       │
    │                    │                      │◀───────────────────────│                       │
    │                    │                      │                        │                       │
    │                    │  {updated user}      │                        │                       │
    │                    │◀─────────────────────│                        │                       │
    │                    │                      │                        │                       │
    │ 200 OK {user}      │                      │                        │                       │
    │◀───────────────────│                      │                        │                       │
    │                    │                      │                        │                       │
```

---

## 5. Update User Status Sequence

```
┌────────┐          ┌─────────┐          ┌─────────────┐          ┌──────────────┐          ┌──────────┐
│ Client │          │ API GW  │          │ User Lambda │          │ User Repo    │          │ DynamoDB │
└───┬────┘          └────┬────┘          └──────┬──────┘          └──────┬───────┘          └────┬─────┘
    │                    │                      │                        │                       │
    │ PATCH /admin/users/│                      │                        │                       │
    │   user_abc123/     │                      │                        │                       │
    │   status           │                      │                        │                       │
    │ {status: "ACTIVE"} │                      │                        │                       │
    │───────────────────▶│                      │                        │                       │
    │                    │                      │                        │                       │
    │                    │  Forward request     │                        │                       │
    │                    │─────────────────────▶│                        │                       │
    │                    │                      │                        │                       │
    │                    │                      │  Validate status       │                       │
    │                    │                      │  (ACTIVE/INACTIVE/     │                       │
    │                    │                      │   PENDING)             │                       │
    │                    │                      │──────────┐             │                       │
    │                    │                      │          │             │                       │
    │                    │                      │◀─────────┘             │                       │
    │                    │                      │                        │                       │
    │                    │                      │  GetByID(user_abc123)  │                       │
    │                    │                      │───────────────────────▶│                       │
    │                    │                      │                        │                       │
    │                    │                      │                        │  GetItem              │
    │                    │                      │                        │──────────────────────▶│
    │                    │                      │                        │                       │
    │                    │                      │                        │  User item            │
    │                    │                      │                        │◀──────────────────────│
    │                    │                      │                        │                       │
    │                    │                      │  Existing user         │                       │
    │                    │                      │◀───────────────────────│                       │
    │                    │                      │                        │                       │
    │                    │                      │  Update status field   │                       │
    │                    │                      │  - status: ACTIVE      │                       │
    │                    │                      │  - updatedAt           │                       │
    │                    │                      │  - updatedBy           │                       │
    │                    │                      │──────────┐             │                       │
    │                    │                      │          │             │                       │
    │                    │                      │◀─────────┘             │                       │
    │                    │                      │                        │                       │
    │                    │                      │  Update(user)          │                       │
    │                    │                      │───────────────────────▶│                       │
    │                    │                      │                        │                       │
    │                    │                      │                        │  PutItem              │
    │                    │                      │                        │──────────────────────▶│
    │                    │                      │                        │                       │
    │                    │                      │                        │  Success              │
    │                    │                      │                        │◀──────────────────────│
    │                    │                      │                        │                       │
    │                    │                      │  Success               │                       │
    │                    │                      │◀───────────────────────│                       │
    │                    │                      │                        │                       │
    │                    │  {message}           │                        │                       │
    │                    │◀─────────────────────│                        │                       │
    │                    │                      │                        │                       │
    │ 200 OK             │                      │                        │                       │
    │ {message}          │                      │                        │                       │
    │◀───────────────────│                      │                        │                       │
    │                    │                      │                        │                       │
```

---

## 6. Delete User Sequence

```
┌────────┐          ┌─────────┐          ┌─────────────┐          ┌──────────────┐          ┌──────────┐
│ Client │          │ API GW  │          │ User Lambda │          │ User Repo    │          │ DynamoDB │
└───┬────┘          └────┬────┘          └──────┬──────┘          └──────┬───────┘          └────┬─────┘
    │                    │                      │                        │                       │
    │ DELETE /admin/     │                      │                        │                       │
    │   users/user_abc123│                      │                        │                       │
    │───────────────────▶│                      │                        │                       │
    │                    │                      │                        │                       │
    │                    │  Forward request     │                        │                       │
    │                    │─────────────────────▶│                        │                       │
    │                    │                      │                        │                       │
    │                    │                      │  Delete(user_abc123)   │                       │
    │                    │                      │───────────────────────▶│                       │
    │                    │                      │                        │                       │
    │                    │                      │                        │  DeleteItem with      │
    │                    │                      │                        │  condition:           │
    │                    │                      │                        │  attr_exists(PK)      │
    │                    │                      │                        │  PK: USER#user_abc123 │
    │                    │                      │                        │  SK: METADATA         │
    │                    │                      │                        │──────────────────────▶│
    │                    │                      │                        │                       │
    │                    │                      │                        │  Success              │
    │                    │                      │                        │◀──────────────────────│
    │                    │                      │                        │                       │
    │                    │                      │  Success               │                       │
    │                    │                      │◀───────────────────────│                       │
    │                    │                      │                        │                       │
    │                    │  {message}           │                        │                       │
    │                    │◀─────────────────────│                        │                       │
    │                    │                      │                        │                       │
    │ 200 OK             │                      │                        │                       │
    │ "User deleted"     │                      │                        │                       │
    │◀───────────────────│                      │                        │                       │
    │                    │                      │                        │                       │
```

---

## 7. Update Last Login Sequence

```
┌─────────────┐          ┌─────────────┐          ┌──────────────┐          ┌──────────┐
│Auth Lambda  │          │ User Repo   │          │  DynamoDB    │          │   Audit  │
└──────┬──────┘          └──────┬──────┘          └──────┬───────┘          └────┬─────┘
       │                        │                        │                       │
       │  (After successful     │                        │                       │
       │   login)               │                        │                       │
       │                        │                        │                       │
       │  UpdateLastLogin       │                        │                       │
       │  (user_abc123)         │                        │                       │
       │───────────────────────▶│                        │                       │
       │                        │                        │                       │
       │                        │  UpdateItem            │                       │
       │                        │  PK: USER#user_abc123  │                       │
       │                        │  SK: METADATA          │                       │
       │                        │  SET last_login_at,    │                       │
       │                        │      updated_at        │                       │
       │                        │───────────────────────▶│                       │
       │                        │                        │                       │
       │                        │  Success               │                       │
       │                        │◀───────────────────────│                       │
       │                        │                        │                       │
       │  Success               │                        │                       │
       │◀───────────────────────│                        │                       │
       │                        │                        │                       │
       │  Log login event       │                        │                       │
       │───────────────────────────────────────────────────────────────────────▶│
       │                        │                        │                       │
```

---

## 8. Get User by Email Sequence

```
┌─────────────┐          ┌──────────────┐          ┌──────────┐
│Auth Service │          │ User Repo    │          │ DynamoDB │
└──────┬──────┘          └──────┬───────┘          └────┬─────┘
       │                        │                       │
       │  (During login)        │                       │
       │                        │                       │
       │  GetByEmail            │                       │
       │  (john@example.com)    │                       │
       │───────────────────────▶│                       │
       │                        │                       │
       │                        │  Query GSI1           │
       │                        │  GSI1PK: USER_EMAIL   │
       │                        │  GSI1SK: john@        │
       │                        │          example.com  │
       │                        │──────────────────────▶│
       │                        │                       │
       │                        │  User item            │
       │                        │  (includes password   │
       │                        │   hash for auth)      │
       │                        │◀──────────────────────│
       │                        │                       │
       │  User with hash        │                       │
       │◀───────────────────────│                       │
       │                        │                       │
       │  Verify password       │                       │
       │  against hash          │                       │
       │──────────┐             │                       │
       │          │             │                       │
       │◀─────────┘             │                       │
       │                        │                       │
```

---

## 9. Error Handling - User Not Found Sequence

```
┌────────┐          ┌─────────┐          ┌─────────────┐          ┌──────────────┐          ┌──────────┐
│ Client │          │ API GW  │          │ User Lambda │          │ User Repo    │          │ DynamoDB │
└───┬────┘          └────┬────┘          └──────┬──────┘          └──────┬───────┘          └────┬─────┘
    │                    │                      │                        │                       │
    │ GET /admin/users/  │                      │                        │                       │
    │     invalid_id     │                      │                        │                       │
    │───────────────────▶│                      │                        │                       │
    │                    │                      │                        │                       │
    │                    │  Forward request     │                        │                       │
    │                    │─────────────────────▶│                        │                       │
    │                    │                      │                        │                       │
    │                    │                      │  GetByID(invalid_id)   │                       │
    │                    │                      │───────────────────────▶│                       │
    │                    │                      │                        │                       │
    │                    │                      │                        │  GetItem              │
    │                    │                      │                        │  PK: USER#invalid_id  │
    │                    │                      │                        │  SK: METADATA         │
    │                    │                      │                        │──────────────────────▶│
    │                    │                      │                        │                       │
    │                    │                      │                        │  null (not found)     │
    │                    │                      │                        │◀──────────────────────│
    │                    │                      │                        │                       │
    │                    │                      │  NotFoundError         │                       │
    │                    │                      │  "User not found"      │                       │
    │                    │                      │◀───────────────────────│                       │
    │                    │                      │                        │                       │
    │                    │  404 Not Found       │                        │                       │
    │                    │◀─────────────────────│                        │                       │
    │                    │                      │                        │                       │
    │ 404 Not Found      │                      │                        │                       │
    │ {error: "User      │                      │                        │                       │
    │  not found"}       │                      │                        │                       │
    │◀───────────────────│                      │                        │                       │
    │                    │                      │                        │                       │
```

---

## 10. Error Handling - Email Already Exists Sequence

```
┌────────┐          ┌─────────┐          ┌─────────────┐          ┌──────────────┐          ┌──────────┐
│ Client │          │ API GW  │          │ User Lambda │          │ User Repo    │          │ DynamoDB │
└───┬────┘          └────┬────┘          └──────┬──────┘          └──────┬───────┘          └────┬─────┘
    │                    │                      │                        │                       │
    │ POST /admin/users  │                      │                        │                       │
    │ {email: existing}  │                      │                        │                       │
    │───────────────────▶│                      │                        │                       │
    │                    │                      │                        │                       │
    │                    │  Forward request     │                        │                       │
    │                    │─────────────────────▶│                        │                       │
    │                    │                      │                        │                       │
    │                    │                      │  GetByEmail(email)     │                       │
    │                    │                      │───────────────────────▶│                       │
    │                    │                      │                        │                       │
    │                    │                      │                        │  Query GSI1           │
    │                    │                      │                        │──────────────────────▶│
    │                    │                      │                        │                       │
    │                    │                      │                        │  Existing user found  │
    │                    │                      │                        │◀──────────────────────│
    │                    │                      │                        │                       │
    │                    │                      │  User found            │                       │
    │                    │                      │◀───────────────────────│                       │
    │                    │                      │                        │                       │
    │                    │                      │  AlreadyExistsError    │                       │
    │                    │                      │  "User with this email │                       │
    │                    │                      │   already exists"      │                       │
    │                    │                      │──────────┐             │                       │
    │                    │                      │          │             │                       │
    │                    │                      │◀─────────┘             │                       │
    │                    │                      │                        │                       │
    │                    │  409 Conflict        │                        │                       │
    │                    │◀─────────────────────│                        │                       │
    │                    │                      │                        │                       │
    │ 409 Conflict       │                      │                        │                       │
    │ {error: "User with │                      │                        │                       │
    │  this email exists"}                      │                        │                       │
    │◀───────────────────│                      │                        │                       │
    │                    │                      │                        │                       │
```
