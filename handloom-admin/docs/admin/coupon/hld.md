# Coupon Lambda - High Level Design

## 1. Overview

The Coupon Lambda provides comprehensive discount coupon management for the Handloom Admin platform. It handles coupon creation, validation, application, and usage tracking to enable flexible promotional campaigns.

### Key Features
- Multiple discount types (percentage, fixed, free shipping)
- Flexible validity rules (date range, usage limits)
- Product/category applicability
- Per-user usage limits
- Usage history and analytics
- Real-time validation

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          COUPON LAMBDA ARCHITECTURE                          │
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
│  │  /coupons               GET     - List coupons                       │    │
│  │  /coupons               POST    - Create coupon                      │    │
│  │  /coupons/{id}          GET     - Get coupon details                 │    │
│  │  /coupons/{id}          PUT     - Update coupon                      │    │
│  │  /coupons/{id}          DELETE  - Delete coupon                      │    │
│  │  /coupons/validate      POST    - Validate coupon                    │    │
│  │  /coupons/{id}/usage    GET     - Get usage history                  │    │
│  │  /coupons/code/{code}   GET     - Get coupon by code                 │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            Coupon Lambda                                     │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │                         Handler Layer                               │     │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                │     │
│  │  │   Create     │ │    List      │ │   Validate   │                │     │
│  │  │   Handler    │ │   Handler    │ │   Handler    │                │     │
│  │  └──────────────┘ └──────────────┘ └──────────────┘                │     │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                │     │
│  │  │   Update     │ │   Delete     │ │    Usage     │                │     │
│  │  │   Handler    │ │   Handler    │ │   Handler    │                │     │
│  │  └──────────────┘ └──────────────┘ └──────────────┘                │     │
│  └────────────────────────────────────────────────────────────────────┘     │
│                                    │                                         │
│                                    ▼                                         │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │                        Service Layer                                │     │
│  │  ┌──────────────────────────────────────────────────────────────┐  │     │
│  │  │                     Coupon Service                            │  │     │
│  │  │  - CreateCoupon()         - GetCoupon()                      │  │     │
│  │  │  - UpdateCoupon()         - DeleteCoupon()                   │  │     │
│  │  │  - ListCoupons()          - ValidateCoupon()                 │  │     │
│  │  │  - ApplyCoupon()          - ReleaseCoupon()                  │  │     │
│  │  │  - GetUsageHistory()      - CalculateDiscount()              │  │     │
│  │  └──────────────────────────────────────────────────────────────┘  │     │
│  └────────────────────────────────────────────────────────────────────┘     │
│                                    │                                         │
│                                    ▼                                         │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │                      Repository Layer                               │     │
│  │  ┌──────────────────────────────────────────────────────────────┐  │     │
│  │  │                   Coupon Repository                           │  │     │
│  │  │  - Create()         - GetByID()       - GetByCode()          │  │     │
│  │  │  - Update()         - Delete()        - List()               │  │     │
│  │  │  - RecordUsage()    - GetUsage()      - GetUserUsage()       │  │     │
│  │  └──────────────────────────────────────────────────────────────┘  │     │
│  └────────────────────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
                              ┌──────────────┐
                              │  DynamoDB    │
                              │   Tables     │
                              └──────────────┘
```

---

## 3. Component Design

### 3.1 Coupon Handler

```go
type CouponHandler struct {
    couponService domain.CouponService
    logger        *logger.Logger
}

// Handler Methods
- CreateCoupon(c *gin.Context)
- GetCoupon(c *gin.Context)
- GetCouponByCode(c *gin.Context)
- UpdateCoupon(c *gin.Context)
- DeleteCoupon(c *gin.Context)
- ListCoupons(c *gin.Context)
- ValidateCoupon(c *gin.Context)
- GetUsageHistory(c *gin.Context)
```

### 3.2 Coupon Service

```go
type CouponService interface {
    // CRUD Operations
    CreateCoupon(ctx context.Context, req *CreateCouponRequest) (*Coupon, error)
    GetCoupon(ctx context.Context, id string) (*Coupon, error)
    GetCouponByCode(ctx context.Context, code string) (*Coupon, error)
    UpdateCoupon(ctx context.Context, id string, req *UpdateCouponRequest) (*Coupon, error)
    DeleteCoupon(ctx context.Context, id string) error
    ListCoupons(ctx context.Context, filter *CouponFilter) (*CouponList, error)

    // Validation and Application
    ValidateCoupon(ctx context.Context, req *ValidateCouponRequest) (*ValidationResult, error)
    ApplyCoupon(ctx context.Context, couponID, orderID, userID string, amount float64) error
    ReleaseCoupon(ctx context.Context, couponID, orderID string) error

    // Usage
    GetUsageHistory(ctx context.Context, couponID string, pagination *Pagination) (*UsageHistory, error)
    GetUserUsage(ctx context.Context, couponID, userID string) (int, error)
}
```

### 3.3 Coupon Repository

```go
type CouponRepository interface {
    Create(ctx context.Context, coupon *Coupon) error
    GetByID(ctx context.Context, id string) (*Coupon, error)
    GetByCode(ctx context.Context, code string) (*Coupon, error)
    Update(ctx context.Context, coupon *Coupon) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, filter *CouponFilter) ([]*Coupon, error)
    IncrementUsage(ctx context.Context, id string) error
    DecrementUsage(ctx context.Context, id string) error
    RecordUsage(ctx context.Context, usage *CouponUsage) error
    GetUsage(ctx context.Context, couponID string) ([]*CouponUsage, error)
    GetUserUsageCount(ctx context.Context, couponID, userID string) (int, error)
}
```

---

## 4. Data Model

### 4.1 Coupon Entity

```go
type Coupon struct {
    ID              string           `json:"id" dynamodbav:"id"`
    Code            string           `json:"code" dynamodbav:"code"`
    Description     string           `json:"description" dynamodbav:"description"`
    DiscountType    DiscountType     `json:"discount_type" dynamodbav:"discount_type"`
    DiscountValue   float64          `json:"discount_value" dynamodbav:"discount_value"`
    MaxDiscount     *float64         `json:"max_discount,omitempty" dynamodbav:"max_discount,omitempty"`
    MinOrderValue   float64          `json:"min_order_value" dynamodbav:"min_order_value"`
    MaxUses         *int             `json:"max_uses,omitempty" dynamodbav:"max_uses,omitempty"`
    MaxUsesPerUser  int              `json:"max_uses_per_user" dynamodbav:"max_uses_per_user"`
    CurrentUses     int              `json:"current_uses" dynamodbav:"current_uses"`
    StartDate       time.Time        `json:"start_date" dynamodbav:"start_date"`
    EndDate         time.Time        `json:"end_date" dynamodbav:"end_date"`
    Status          CouponStatus     `json:"status" dynamodbav:"status"`
    Applicability   *Applicability   `json:"applicability,omitempty" dynamodbav:"applicability,omitempty"`
    Exclusions      *Exclusions      `json:"exclusions,omitempty" dynamodbav:"exclusions,omitempty"`
    CreatedAt       time.Time        `json:"created_at" dynamodbav:"created_at"`
    UpdatedAt       time.Time        `json:"updated_at" dynamodbav:"updated_at"`
    CreatedBy       string           `json:"created_by" dynamodbav:"created_by"`
}
```

### 4.2 Discount Types

```go
type DiscountType string

const (
    DiscountTypePercentage  DiscountType = "PERCENTAGE"
    DiscountTypeFixed       DiscountType = "FIXED"
    DiscountTypeFreeShipping DiscountType = "FREE_SHIPPING"
)
```

### 4.3 Coupon Status

```go
type CouponStatus string

const (
    CouponStatusDraft     CouponStatus = "DRAFT"
    CouponStatusActive    CouponStatus = "ACTIVE"
    CouponStatusPaused    CouponStatus = "PAUSED"
    CouponStatusExpired   CouponStatus = "EXPIRED"
    CouponStatusExhausted CouponStatus = "EXHAUSTED"
    CouponStatusDeleted   CouponStatus = "DELETED"
)
```

### 4.4 Applicability Rules

```go
type Applicability struct {
    ApplyTo        ApplyToType `json:"apply_to" dynamodbav:"apply_to"`
    CategoryIDs    []string    `json:"category_ids,omitempty" dynamodbav:"category_ids,omitempty"`
    ProductIDs     []string    `json:"product_ids,omitempty" dynamodbav:"product_ids,omitempty"`
    UserSegment    string      `json:"user_segment,omitempty" dynamodbav:"user_segment,omitempty"`
    FirstTimeOnly  bool        `json:"first_time_only" dynamodbav:"first_time_only"`
}

type ApplyToType string

const (
    ApplyToAll        ApplyToType = "ALL"
    ApplyToCategories ApplyToType = "CATEGORIES"
    ApplyToProducts   ApplyToType = "PRODUCTS"
)
```

### 4.5 Exclusions

```go
type Exclusions struct {
    ExcludeSaleItems     bool     `json:"exclude_sale_items" dynamodbav:"exclude_sale_items"`
    ExcludeDiscounted    bool     `json:"exclude_discounted" dynamodbav:"exclude_discounted"`
    ExcludedCategoryIDs  []string `json:"excluded_category_ids,omitempty" dynamodbav:"excluded_category_ids,omitempty"`
    ExcludedProductIDs   []string `json:"excluded_product_ids,omitempty" dynamodbav:"excluded_product_ids,omitempty"`
}
```

### 4.6 Coupon Usage

```go
type CouponUsage struct {
    ID            string    `json:"id" dynamodbav:"id"`
    CouponID      string    `json:"coupon_id" dynamodbav:"coupon_id"`
    CouponCode    string    `json:"coupon_code" dynamodbav:"coupon_code"`
    OrderID       string    `json:"order_id" dynamodbav:"order_id"`
    UserID        string    `json:"user_id" dynamodbav:"user_id"`
    DiscountAmount float64  `json:"discount_amount" dynamodbav:"discount_amount"`
    OrderValue    float64   `json:"order_value" dynamodbav:"order_value"`
    UsedAt        time.Time `json:"used_at" dynamodbav:"used_at"`
}
```

### 4.7 Validation Request/Result

```go
type ValidateCouponRequest struct {
    Code       string      `json:"code" binding:"required"`
    UserID     string      `json:"user_id" binding:"required"`
    CartItems  []CartItem  `json:"cart_items" binding:"required"`
    CartTotal  float64     `json:"cart_total" binding:"required"`
}

type ValidationResult struct {
    Valid           bool     `json:"valid"`
    CouponID        string   `json:"coupon_id,omitempty"`
    DiscountAmount  float64  `json:"discount_amount,omitempty"`
    ApplicableItems []string `json:"applicable_items,omitempty"`
    ErrorCode       string   `json:"error_code,omitempty"`
    ErrorMessage    string   `json:"error_message,omitempty"`
}
```

---

## 5. DynamoDB Schema

### 5.1 Coupon Table

```
Table: handloom-coupons

Primary Key:
- PK: COUPON#<coupon_id>
- SK: COUPON#<coupon_id>

Attributes:
- id: string
- code: string (unique)
- description: string
- discount_type: string
- discount_value: number
- max_discount: number
- min_order_value: number
- max_uses: number
- max_uses_per_user: number
- current_uses: number
- start_date: string (ISO8601)
- end_date: string (ISO8601)
- status: string
- applicability: map
- exclusions: map
- created_at: string
- updated_at: string
- created_by: string

GSI1: code-index
- PK: code
- SK: COUPON

GSI2: status-index
- PK: status
- SK: end_date
```

### 5.2 Coupon Usage Table

```
Table: handloom-coupon-usage

Primary Key:
- PK: COUPON#<coupon_id>
- SK: USAGE#<usage_id>

Attributes:
- id: string
- coupon_id: string
- coupon_code: string
- order_id: string
- user_id: string
- discount_amount: number
- order_value: number
- used_at: string

GSI1: user-coupon-index
- PK: USER#<user_id>
- SK: COUPON#<coupon_id>

GSI2: order-index
- PK: ORDER#<order_id>
- SK: COUPON#<coupon_id>
```

### 5.3 Access Patterns

| Access Pattern | Key Condition | Index |
|----------------|---------------|-------|
| Get coupon by ID | PK = COUPON#{id} | Main |
| Get coupon by code | PK = {code} | GSI1 |
| List active coupons | PK = ACTIVE | GSI2 |
| Get usage by coupon | PK = COUPON#{id} | Usage Table |
| Get user usage | PK = USER#{user_id} | GSI1 |
| Get order coupon | PK = ORDER#{order_id} | GSI2 |

---

## 6. API Endpoints

### 6.1 Create Coupon

```
POST /coupons

Request:
{
    "code": "SUMMER20",
    "description": "20% off summer sale",
    "discount_type": "PERCENTAGE",
    "discount_value": 20,
    "max_discount": 500,
    "min_order_value": 500,
    "max_uses": 1000,
    "max_uses_per_user": 1,
    "start_date": "2024-01-15T00:00:00Z",
    "end_date": "2024-02-15T23:59:59Z",
    "applicability": {
        "apply_to": "ALL"
    },
    "exclusions": {
        "exclude_sale_items": false,
        "exclude_discounted": true
    }
}

Response:
{
    "success": true,
    "data": {
        "id": "coupon_123",
        "code": "SUMMER20",
        "status": "ACTIVE",
        ...
    }
}
```

### 6.2 Validate Coupon

```
POST /coupons/validate

Request:
{
    "code": "SUMMER20",
    "user_id": "user_456",
    "cart_items": [
        {"product_id": "prod_1", "price": 1500, "quantity": 1},
        {"product_id": "prod_2", "price": 1000, "quantity": 1}
    ],
    "cart_total": 2500
}

Response (Valid):
{
    "success": true,
    "data": {
        "valid": true,
        "coupon_id": "coupon_123",
        "discount_amount": 500,
        "applicable_items": ["prod_1", "prod_2"]
    }
}

Response (Invalid):
{
    "success": true,
    "data": {
        "valid": false,
        "error_code": "COUPON_EXPIRED",
        "error_message": "This coupon has expired"
    }
}
```

### 6.3 List Coupons

```
GET /coupons?status=active&type=percentage&page=1&limit=20

Response:
{
    "success": true,
    "data": {
        "coupons": [
            {
                "id": "coupon_123",
                "code": "SUMMER20",
                "discount_type": "PERCENTAGE",
                "discount_value": 20,
                "current_uses": 234,
                "max_uses": 1000,
                "status": "ACTIVE",
                "end_date": "2024-02-15T23:59:59Z"
            }
        ],
        "total": 25,
        "page": 1,
        "limit": 20
    }
}
```

### 6.4 Get Coupon Usage

```
GET /coupons/{id}/usage?page=1&limit=20

Response:
{
    "success": true,
    "data": {
        "coupon_id": "coupon_123",
        "code": "SUMMER20",
        "total_uses": 234,
        "total_discount": 117000,
        "usage": [
            {
                "order_id": "ORD-1234",
                "user_id": "user_456",
                "customer_name": "Priya Sharma",
                "order_value": 3000,
                "discount_amount": 500,
                "used_at": "2024-01-20T10:30:00Z"
            }
        ]
    }
}
```

---

## 7. Validation Rules

### 7.1 Validation Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          COUPON VALIDATION FLOW                              │
└─────────────────────────────────────────────────────────────────────────────┘

┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│ Coupon Exists │────>│ Status Active │────>│ Within Dates  │
└───────────────┘     └───────────────┘     └───────┬───────┘
       │ No                  │ No                   │
       ▼                     ▼                      ▼
┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│ INVALID_CODE  │     │COUPON_INACTIVE│     │ COUPON_EXPIRED│
└───────────────┘     └───────────────┘     └───────────────┘

       ▼ Yes                 ▼ Yes                  ▼ Yes
┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│ Usage Limit   │────>│ Per-User Limit│────>│ Min Order Val │
│ Available     │     │ Not Exceeded  │     │ Met           │
└───────────────┘     └───────────────┘     └───────┬───────┘
       │ No                  │ No                   │
       ▼                     ▼                      ▼
┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│COUPON_EXHAUSTED│    │ ALREADY_USED  │     │ MIN_NOT_MET   │
└───────────────┘     └───────────────┘     └───────────────┘

       ▼ Yes                 ▼ Yes                  ▼ Yes
┌───────────────┐     ┌───────────────┐
│ Applicable to │────>│   VALID       │
│ Cart Items    │     │ (Calculate $) │
└───────────────┘     └───────────────┘
       │ No
       ▼
┌───────────────┐
│NOT_APPLICABLE │
└───────────────┘
```

### 7.2 Validation Error Codes

| Error Code | Description |
|------------|-------------|
| INVALID_CODE | Coupon code does not exist |
| COUPON_INACTIVE | Coupon is not active |
| COUPON_EXPIRED | Coupon validity period has passed |
| COUPON_NOT_STARTED | Coupon validity has not started |
| COUPON_EXHAUSTED | Usage limit reached |
| ALREADY_USED | User has reached per-user limit |
| MIN_ORDER_NOT_MET | Cart total below minimum |
| NOT_APPLICABLE | No items eligible for discount |
| EXCLUDED_ITEMS | All items are in exclusion list |

---

## 8. Discount Calculation

### 8.1 Calculation Logic

```go
func CalculateDiscount(coupon *Coupon, applicableAmount float64) float64 {
    var discount float64

    switch coupon.DiscountType {
    case DiscountTypePercentage:
        discount = applicableAmount * (coupon.DiscountValue / 100)
        if coupon.MaxDiscount != nil && discount > *coupon.MaxDiscount {
            discount = *coupon.MaxDiscount
        }
    case DiscountTypeFixed:
        discount = coupon.DiscountValue
        if discount > applicableAmount {
            discount = applicableAmount
        }
    case DiscountTypeFreeShipping:
        discount = shippingCost // Applied at order level
    }

    return math.Round(discount*100) / 100
}
```

### 8.2 Applicable Amount Calculation

```go
func CalculateApplicableAmount(coupon *Coupon, items []CartItem) float64 {
    var total float64

    for _, item := range items {
        if isItemApplicable(coupon, item) && !isItemExcluded(coupon, item) {
            total += item.Price * float64(item.Quantity)
        }
    }

    return total
}
```

---

## 9. Error Handling

### 9.1 Error Types

| Error Code | Description | HTTP Status |
|------------|-------------|-------------|
| COUPON_NOT_FOUND | Coupon does not exist | 404 |
| CODE_EXISTS | Coupon code already in use | 409 |
| INVALID_DISCOUNT | Invalid discount configuration | 400 |
| INVALID_DATES | End date before start date | 400 |
| VALIDATION_FAILED | Coupon validation failed | 400 |
| USAGE_CONFLICT | Concurrent usage conflict | 409 |

### 9.2 Error Response Format

```json
{
    "success": false,
    "error": {
        "code": "COUPON_NOT_FOUND",
        "message": "Coupon with code 'INVALID' not found"
    }
}
```

---

## 10. Security

### 10.1 Access Control

| Role | Create | Read | Update | Delete | Validate |
|------|--------|------|--------|--------|----------|
| Admin | Yes | All | Yes | Yes | Yes |
| Manager | Yes | All | Yes | No | Yes |
| Staff | No | All | No | No | Yes |
| Customer | No | Own Used | No | No | Yes |

### 10.2 Code Generation

- Codes are case-insensitive (stored uppercase)
- Random generation uses cryptographically secure random
- Collision detection before creation
- Rate limiting on validation attempts

---

## 11. Concurrency Handling

### 11.1 Atomic Usage Increment

```go
// DynamoDB conditional update for usage tracking
updateExpression := "SET current_uses = current_uses + :inc"
conditionExpression := "current_uses < :max"

// Handles concurrent coupon applications
err := repo.UpdateWithCondition(ctx, couponID, updateExpression, conditionExpression)
if err == conditionalCheckFailed {
    return ErrCouponExhausted
}
```

### 11.2 Race Condition Prevention

- Use DynamoDB conditional writes
- Optimistic locking on usage updates
- Transaction support for apply/release

---

## 12. Performance Optimization

### 12.1 Caching Strategy

- Cache active coupons by code (5-minute TTL)
- Cache user usage counts (1-minute TTL)
- Invalidate on usage/update

### 12.2 Query Optimization

- GSI for code lookup (most frequent)
- Sparse index for active coupons
- Batch operations for usage queries

---

## 13. Monitoring

### 13.1 Key Metrics

| Metric | Description | Threshold |
|--------|-------------|-----------|
| Validation Rate | Validations per minute | Monitor for abuse |
| Success Rate | % of successful validations | Track trends |
| Usage Rate | Coupon redemptions | Alert near limit |
| Revenue Impact | Total discount given | Track ROI |

### 13.2 Alerts

- Coupon approaching usage limit (90%)
- High validation failure rate
- Unusual usage patterns (abuse detection)
- Expired coupons still being accessed

---

## 14. Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              DEPENDENCIES                                    │
└─────────────────────────────────────────────────────────────────────────────┘

                          Coupon Lambda
                               │
           ┌───────────────────┼───────────────────┐
           │                   │                   │
           ▼                   ▼                   ▼
    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
    │  DynamoDB   │    │Order Service│    │Product Svc  │
    │  (Storage)  │    │ (Apply)     │    │ (Validate)  │
    └─────────────┘    └─────────────┘    └─────────────┘
```

### Internal Dependencies
- Order Service: Applies coupon during checkout
- Product Service: Validates product applicability
- User Service: Validates user eligibility

### External Dependencies
- AWS DynamoDB: Coupon storage
- AWS CloudWatch: Logging and monitoring

