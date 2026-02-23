# Store Profile - High-Level Design (HLD)

## 1. Overview

The Store Profile module manages customer profile data and address book for the B2C storefront. Customers can view and update their personal information, and manage multiple shipping addresses. The profile is automatically created during the first OTP login and is stored in the handloom-orders DynamoDB table. Order statistics (total_orders, total_spent) are denormalized onto the customer record and updated asynchronously when orders are completed.

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                  STORE PROFILE SYSTEM                                        │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

                                    ┌───────────────────┐
                                    │  Next.js Frontend │
                                    │  (B2C Storefront) │
                                    └─────────┬─────────┘
                                              │
                                              │ HTTPS (Customer JWT)
                                              ▼
                                    ┌───────────────────┐
                                    │   API Gateway /   │
                                    │   Chi Router      │
                                    └─────────┬─────────┘
                                              │
                                              │ CustomerAuth Middleware
                                              ▼
                                    ┌───────────────────┐
                                    │  Store Profile    │
                                    │  Handler          │
                                    │  - GetProfile     │
                                    │  - UpdateProfile  │
                                    │  - AddAddress     │
                                    │  - UpdateAddress  │
                                    │  - DeleteAddress  │
                                    └─────────┬─────────┘
                                              │
                                              ▼
                                    ┌───────────────────┐
                                    │  Customer         │
                                    │  Repository       │
                                    └─────────┬─────────┘
                                              │
                                              ▼
                                    ┌───────────────────┐
                                    │   DynamoDB        │
                                    │  (handloom-       │
                                    │   orders)         │
                                    └───────────────────┘
```

---

## 3. Component Design

### 3.1 Handler Layer

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      STORE PROFILE HANDLER                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                     StoreProfileHandler                              │   │
│  │                                                                      │   │
│  │  Dependencies:                                                       │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │  - CustomerRepository   (domain.CustomerRepository)           │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  │  Routes (all require CustomerAuth middleware):                       │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │  GET    /                  → GetProfile(w, r)                 │ │   │
│  │  │  PATCH  /                  → UpdateProfile(w, r)              │ │   │
│  │  │  POST   /addresses         → AddAddress(w, r)                │ │   │
│  │  │  PUT    /addresses/{id}    → UpdateAddress(w, r)              │ │   │
│  │  │  DELETE /addresses/{id}    → DeleteAddress(w, r)              │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Address Management Logic

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     ADDRESS MANAGEMENT RULES                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Default Address Invariant:                                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • At most one address can be marked as is_default=true              │   │
│  │ • When a new address is added with is_default=true:                 │   │
│  │   → All existing addresses have is_default set to false             │   │
│  │   → The new address is saved with is_default=true                   │   │
│  │ • When an address is updated with is_default=true:                  │   │
│  │   → All other addresses have is_default set to false                │   │
│  │ • When the default address is deleted:                              │   │
│  │   → No automatic promotion of another address to default            │   │
│  │   → Customer must explicitly set a new default                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Storage:                                                                    │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Addresses are stored as an embedded array in the Customer doc    │   │
│  │ • Each address gets a generated UUID (addr-xxx)                    │   │
│  │ • Entire addresses array is updated atomically per operation        │   │
│  │ • No separate DynamoDB items for addresses (embedded model)         │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Model

### 4.1 DynamoDB Schema

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    TABLE: handloom-orders                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  CUSTOMER RECORD                                                             │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: CUSTOMER#<customer_id>                                        │      │
│  │ SK: METADATA                                                      │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - ID                    - Email                                 │      │
│  │   - FirstName             - LastName                              │      │
│  │   - Phone                 - PhoneVerified (bool)                  │      │
│  │   - Status (ACTIVE/INACTIVE/BLOCKED)                              │      │
│  │   - TotalOrders (denorm)  - TotalSpent (paise, denorm)            │      │
│  │   - Addresses[]           - CreatedAt                             │      │
│  │   - UpdatedAt                                                     │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  EMBEDDED ADDRESSES ARRAY                                                    │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ Addresses: [                                                      │      │
│  │   {                                                               │      │
│  │     ID, FirstName, LastName, Phone,                               │      │
│  │     AddressLine1, AddressLine2, City, State,                      │      │
│  │     PostalCode, Country, IsDefault                                │      │
│  │   }                                                               │      │
│  │ ]                                                                 │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  GSI1: Customer Email Lookup                                                 │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ GSI1PK: CUSTOMER_EMAIL                                            │      │
│  │ GSI1SK: <email>                                                   │      │
│  │                                                                   │      │
│  │ Used for:                                                         │      │
│  │   • Email uniqueness check during profile update                  │      │
│  │   • Admin customer lookup by email                                │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  DENORMALIZED STATS UPDATE (async)                                           │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ TotalOrders: Incremented when order status → DELIVERED            │      │
│  │ TotalSpent:  order.TotalAmount added when status → DELIVERED      │      │
│  │                                                                   │      │
│  │ Updated via: OrderService.CompleteOrder() → CustomerRepo.Update() │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Security

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           SECURITY MODEL                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Authentication:                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • All endpoints require Customer JWT (store_token cookie)          │   │
│  │ • CustomerAuth middleware extracts customer_id from claims          │   │
│  │ • Phone is immutable (set during OTP verification, cannot change)  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Authorization:                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Customers can only access their own profile (enforced by PK)     │   │
│  │ • Address operations scope to the customer's embedded array        │   │
│  │ • No cross-customer access is possible                             │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Input Validation:                                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Email format validated via regex                                  │   │
│  │ • Phone format validated as E.164 (not editable by customer)       │   │
│  │ • Postal code validated as 6-digit Indian format                    │   │
│  │ • All string fields length-limited to prevent abuse                 │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Error Handling

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            ERROR CODES                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Profile Errors:                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ Code                  │ HTTP │ Description                          │   │
│  ├───────────────────────┼──────┼────────────────────────────────────┤   │
│  │ UNAUTHORIZED          │ 401  │ Missing or invalid customer JWT     │   │
│  │ VALIDATION_ERROR      │ 400  │ Invalid field values                │   │
│  │ EMAIL_ALREADY_EXISTS  │ 409  │ Email in use by another customer    │   │
│  │ ADDRESS_NOT_FOUND     │ 404  │ Address ID not in address book      │   │
│  │ CUSTOMER_NOT_FOUND    │ 404  │ Customer record not found           │   │
│  │ INTERNAL_ERROR        │ 500  │ Unexpected server error             │   │
│  └───────────────────────┴──────┴────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Integration Points

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         INTEGRATION POINTS                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                   Customer Repository                                │   │
│  │                                                                      │   │
│  │  StoreProfileHandler ──▶ CustomerRepository                         │   │
│  │    • GetByID(customerID) — fetch profile                            │   │
│  │    • Update(customer) — save profile changes                        │   │
│  │    • GetByEmail(email) — uniqueness check                           │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                   Store Auth (upstream)                               │   │
│  │                                                                      │   │
│  │  StoreAuth ──▶ Creates Customer record on first OTP verification    │   │
│  │    • Customer created with phone, phone_verified=true                │   │
│  │    • Empty first_name, last_name, email                              │   │
│  │    • Empty addresses array                                           │   │
│  │    • total_orders=0, total_spent=0                                   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                   Checkout (downstream)                               │   │
│  │                                                                      │   │
│  │  Checkout uses customer addresses for shipping selection:            │   │
│  │    • Default address is pre-selected at checkout                     │   │
│  │    • Customer can pick any saved address or enter new                │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                             DEPENDENCIES                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  External Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • AWS DynamoDB (handloom-orders table) — customer storage          │   │
│  │ • AWS CloudWatch — logging & monitoring                            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Internal Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • CustomerAuth Middleware — JWT validation, customer_id extraction │   │
│  │ • CustomerRepository — DynamoDB CRUD for customer records          │   │
│  │ • Validation Middleware — request body validation (generics-based) │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Shared Domain Entities:                                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • domain.Customer — customer entity with profile and stats         │   │
│  │ • domain.Address — address structure (embedded in customer)        │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```
