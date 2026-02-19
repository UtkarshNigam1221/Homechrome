# B2C Storefront Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a customer-facing B2C storefront to Homechrome — OTP auth, cart, checkout with PhonePe, Shiprocket shipping, order tracking, and a Next.js frontend.

**Architecture:** Extend the existing Go monolith with new `/api/v1/store/*` routes. New services (CustomerAuth, Cart, Checkout, Payment, Shipping) sit alongside existing admin services, sharing repositories and DynamoDB tables. A new Next.js 15 project (`homechrome-store/`) provides the SSR/SSG storefront.

**Tech Stack:** Go 1.24, Chi router, DynamoDB, Next.js 15, TypeScript, Tailwind CSS, PhonePe, Shiprocket, MSG91.

**Design Doc:** `docs/plans/2026-02-20-b2c-storefront-design.md`

---

## Phase 1: Infrastructure Fixes

### Task 1: Fix CDK — Add Missing GSIs and TTL

**Files:**
- Modify: `infra/stacks/database.go`
- Modify: `scripts/init-local-db.sh`

**Step 1: Add GSI2 to orders table in CDK**

In `infra/stacks/database.go`, the orders table definition is missing GSI2. Add it to match the local init script:

```go
// Inside the orders table definition, after the GSI1 block:
GlobalSecondaryIndexes: []awsdynamodb.GlobalSecondaryIndexSpecification{
    // ... existing GSI1 ...
    {
        IndexName: jsii.String("GSI2"),
        KeySchema: []awsdynamodb.KeySchemaElement{
            {AttributeName: jsii.String("GSI2PK"), KeyType: awsdynamodb.KeyType_HASH},
            {AttributeName: jsii.String("GSI2SK"), KeyType: awsdynamodb.KeyType_RANGE},
        },
        Projection: &awsdynamodb.Projection{ProjectionType: awsdynamodb.ProjectionType_ALL},
    },
},
```

Also add `GSI2PK` and `GSI2SK` to `AttributeDefinitions` for the orders table.

**Step 2: Enable TTL on orders table in CDK**

```go
TimeToLiveSpecification: &awsdynamodb.TimeToLiveSpecification{
    AttributeName: jsii.String("ttl"),
    Enabled:       jsii.Bool(true),
},
```

**Step 3: Enable TTL on core table in CDK**

Same TTL block added to the core table definition.

**Step 4: Add GSI2 to audit table in CDK**

Add GSI2 with `GSI2PK`/`GSI2SK` attribute definitions to the audit table.

**Step 5: Add GSI1 to analytics table in CDK**

Add GSI1 with `GSI1PK`/`GSI1SK` attribute definitions to the analytics table (matches local init script).

**Step 6: Update local init script to add TTL**

In `scripts/init-local-db.sh`, add TTL updates after table creation:

```bash
# Enable TTL on core table
aws dynamodb update-time-to-live \
    --table-name handloom-core \
    --time-to-live-specification "Enabled=true, AttributeName=ttl" \
    --endpoint-url http://localhost:4566

# Enable TTL on orders table
aws dynamodb update-time-to-live \
    --table-name handloom-orders \
    --time-to-live-specification "Enabled=true, AttributeName=ttl" \
    --endpoint-url http://localhost:4566
```

**Step 7: Verify locally**

Run: `make teardown-local && make setup-local`
Expected: All tables created with correct GSIs and TTL enabled.

**Step 8: Commit**

```bash
git add infra/stacks/database.go scripts/init-local-db.sh
git commit -m "fix: add missing GSIs and TTL to CDK and local init"
```

---

## Phase 2: Domain Entities & Interfaces

### Task 2: Add B2C Domain Entities

**Files:**
- Create: `internal/domain/cart.go`
- Create: `internal/domain/payment.go`
- Create: `internal/domain/shipment.go`
- Modify: `internal/domain/order.go` (add GSI2 keys to Order, add Customer phone index, fix OrderStatusHistory)
- Modify: `internal/domain/entity.go` (add GSI2 keys to PricingRule, sparse GSI2 to Inventory)

**Step 1: Create `internal/domain/cart.go`**

```go
package domain

import "time"

type Cart struct {
    ID         string `json:"id" dynamodbav:"id"`
    PK         string `json:"-" dynamodbav:"PK"`
    SK         string `json:"-" dynamodbav:"SK"`
    EntityType string `json:"-" dynamodbav:"entity_type"`
    CustomerID string `json:"customer_id,omitempty" dynamodbav:"customer_id,omitempty"`
    SessionID  string `json:"session_id" dynamodbav:"session_id"`
    ItemCount  int    `json:"item_count" dynamodbav:"item_count"`
    Subtotal   int64  `json:"subtotal" dynamodbav:"subtotal"`
    Currency   string `json:"currency" dynamodbav:"currency"`
    UpdatedAt  time.Time `json:"updated_at" dynamodbav:"updated_at"`
    TTL        int64     `json:"-" dynamodbav:"ttl"`
}

func (c *Cart) SetKeys() {
    if c.CustomerID != "" {
        c.PK = "CART#" + c.CustomerID
    } else {
        c.PK = "CART#" + c.SessionID
    }
    c.SK = "METADATA"
    c.EntityType = "CART"
}

type CartItem struct {
    PK             string `json:"-" dynamodbav:"PK"`
    SK             string `json:"-" dynamodbav:"SK"`
    EntityType     string `json:"-" dynamodbav:"entity_type"`
    ProductID      string `json:"product_id" dynamodbav:"product_id"`
    ProductName    string `json:"product_name" dynamodbav:"product_name"`
    ProductSKU     string `json:"product_sku" dynamodbav:"product_sku"`
    ProductImage   string `json:"product_image" dynamodbav:"product_image"`
    Quantity       int    `json:"quantity" dynamodbav:"quantity"`
    UnitPrice      int64  `json:"unit_price" dynamodbav:"unit_price"`
    TotalPrice     int64  `json:"total_price" dynamodbav:"total_price"`
    IsCustomSize   bool   `json:"is_custom_size" dynamodbav:"is_custom_size"`
    Dimensions     *Dimensions `json:"dimensions,omitempty" dynamodbav:"dimensions,omitempty"`
    QuoteID        *string     `json:"quote_id,omitempty" dynamodbav:"quote_id,omitempty"`
    Attributes     map[string]string `json:"attributes,omitempty" dynamodbav:"attributes,omitempty"`
    AddedAt        time.Time `json:"added_at" dynamodbav:"added_at"`
    TTL            int64     `json:"-" dynamodbav:"ttl"`
}

func (ci *CartItem) SetKeys(cartPK string) {
    ci.PK = cartPK
    ci.SK = "ITEM#" + ci.ProductID
    ci.EntityType = "CART_ITEM"
}

type AddCartItemRequest struct {
    ProductID  string      `json:"product_id" validate:"required"`
    Quantity   int         `json:"quantity" validate:"required,gt=0"`
    Dimensions *Dimensions `json:"dimensions,omitempty"`
    QuoteID    *string     `json:"quote_id,omitempty"`
}

type UpdateCartItemRequest struct {
    Quantity int `json:"quantity" validate:"required,gte=0"`
}

type MergeCartRequest struct {
    Items []AddCartItemRequest `json:"items" validate:"required"`
}

type CartWithItems struct {
    Cart  *Cart      `json:"cart"`
    Items []CartItem `json:"items"`
}
```

**Step 2: Create `internal/domain/payment.go`**

```go
package domain

import "time"

type PaymentStatus string

const (
    PaymentStatusInitiated PaymentStatus = "INITIATED"
    PaymentStatusSuccess   PaymentStatus = "SUCCESS"
    PaymentStatusFailed    PaymentStatus = "FAILED"
    PaymentStatusRefunded  PaymentStatus = "REFUNDED"
)

type PaymentProvider string

const (
    PaymentProviderPhonePe PaymentProvider = "PHONEPE"
)

type PaymentMethod string

const (
    PaymentMethodUPI        PaymentMethod = "UPI"
    PaymentMethodCard       PaymentMethod = "CARD"
    PaymentMethodNetBanking PaymentMethod = "NET_BANKING"
    PaymentMethodWallet     PaymentMethod = "WALLET"
)

type Payment struct {
    ID                     string          `json:"id" dynamodbav:"id"`
    PK                     string          `json:"-" dynamodbav:"PK"`
    SK                     string          `json:"-" dynamodbav:"SK"`
    GSI1PK                 string          `json:"-" dynamodbav:"GSI1PK"`
    GSI1SK                 string          `json:"-" dynamodbav:"GSI1SK"`
    GSI2PK                 string          `json:"-" dynamodbav:"GSI2PK"`
    GSI2SK                 string          `json:"-" dynamodbav:"GSI2SK"`
    EntityType             string          `json:"-" dynamodbav:"entity_type"`
    OrderID                string          `json:"order_id" dynamodbav:"order_id"`
    CustomerID             string          `json:"customer_id" dynamodbav:"customer_id"`
    Amount                 int64           `json:"amount" dynamodbav:"amount"`
    Currency               string          `json:"currency" dynamodbav:"currency"`
    Status                 PaymentStatus   `json:"status" dynamodbav:"status"`
    Provider               PaymentProvider `json:"provider" dynamodbav:"provider"`
    MerchantTransactionID  string          `json:"merchant_transaction_id" dynamodbav:"merchant_transaction_id"`
    ProviderTransactionID  string          `json:"provider_transaction_id,omitempty" dynamodbav:"provider_transaction_id,omitempty"`
    PaymentMethod          PaymentMethod   `json:"payment_method,omitempty" dynamodbav:"payment_method,omitempty"`
    ProviderResponse       string          `json:"provider_response,omitempty" dynamodbav:"provider_response,omitempty"`
    InitiatedAt            time.Time       `json:"initiated_at" dynamodbav:"initiated_at"`
    CompletedAt            *time.Time      `json:"completed_at,omitempty" dynamodbav:"completed_at,omitempty"`
    RefundAmount           int64           `json:"refund_amount,omitempty" dynamodbav:"refund_amount,omitempty"`
    RefundedAt             *time.Time      `json:"refunded_at,omitempty" dynamodbav:"refunded_at,omitempty"`
    BaseEntity
}

func (p *Payment) SetKeys() {
    p.PK = "PAYMENT#" + p.ID
    p.SK = "METADATA"
    p.GSI1PK = "ORDER#" + p.OrderID
    p.GSI1SK = "PAYMENT#" + p.InitiatedAt.Format("2006-01-02T15:04:05Z")
    p.GSI2PK = "PAYMENT_TXN"
    p.GSI2SK = p.MerchantTransactionID
    p.EntityType = "PAYMENT"
}

type InitiatePaymentRequest struct {
    OrderID    string `json:"order_id" validate:"required"`
    CustomerID string `json:"customer_id" validate:"required"`
    Amount     int64  `json:"amount" validate:"required,gt=0"`
    Phone      string `json:"phone" validate:"required"`
}
```

**Step 3: Create `internal/domain/shipment.go`**

```go
package domain

import "time"

type ShipmentStatus string

const (
    ShipmentStatusCreated        ShipmentStatus = "CREATED"
    ShipmentStatusPickedUp       ShipmentStatus = "PICKED_UP"
    ShipmentStatusInTransit      ShipmentStatus = "IN_TRANSIT"
    ShipmentStatusOutForDelivery ShipmentStatus = "OUT_FOR_DELIVERY"
    ShipmentStatusDelivered      ShipmentStatus = "DELIVERED"
    ShipmentStatusRTO            ShipmentStatus = "RTO"
)

type Shipment struct {
    ID                  string         `json:"id" dynamodbav:"id"`
    PK                  string         `json:"-" dynamodbav:"PK"`
    SK                  string         `json:"-" dynamodbav:"SK"`
    EntityType          string         `json:"-" dynamodbav:"entity_type"`
    OrderID             string         `json:"order_id" dynamodbav:"order_id"`
    Provider            string         `json:"provider" dynamodbav:"provider"`
    ProviderOrderID     string         `json:"provider_order_id,omitempty" dynamodbav:"provider_order_id,omitempty"`
    ProviderShipmentID  string         `json:"provider_shipment_id,omitempty" dynamodbav:"provider_shipment_id,omitempty"`
    AWBNumber           string         `json:"awb_number,omitempty" dynamodbav:"awb_number,omitempty"`
    CourierName         string         `json:"courier_name,omitempty" dynamodbav:"courier_name,omitempty"`
    Status              ShipmentStatus `json:"status" dynamodbav:"status"`
    LabelURL            string         `json:"label_url,omitempty" dynamodbav:"label_url,omitempty"`
    EstimatedDelivery   string         `json:"estimated_delivery,omitempty" dynamodbav:"estimated_delivery,omitempty"`
    WeightGrams         int            `json:"weight_grams" dynamodbav:"weight_grams"`
    ShippedAt           *time.Time     `json:"shipped_at,omitempty" dynamodbav:"shipped_at,omitempty"`
    DeliveredAt         *time.Time     `json:"delivered_at,omitempty" dynamodbav:"delivered_at,omitempty"`
    BaseEntity
}

func (s *Shipment) SetKeys() {
    s.PK = "ORDER#" + s.OrderID
    s.SK = "SHIPMENT#" + s.ID
    s.EntityType = "SHIPMENT"
}

type ServiceabilityRequest struct {
    Pincode string `json:"pincode" validate:"required,len=6"`
}

type ServiceabilityResult struct {
    Serviceable bool              `json:"serviceable"`
    Couriers    []CourierOption   `json:"couriers,omitempty"`
}

type CourierOption struct {
    ID            int     `json:"id"`
    Name          string  `json:"name"`
    Rate          int64   `json:"rate"`
    EstimatedDays int     `json:"estimated_days"`
}
```

**Step 4: Add OTP entity to `internal/domain/entity.go`**

At the end of entity.go, add:

```go
// OTP for customer authentication
type OTP struct {
    PK         string    `json:"-" dynamodbav:"PK"`
    SK         string    `json:"-" dynamodbav:"SK"`
    EntityType string    `json:"-" dynamodbav:"entity_type"`
    Phone      string    `json:"phone" dynamodbav:"phone"`
    CodeHash   string    `json:"-" dynamodbav:"code_hash"`
    Attempts   int       `json:"attempts" dynamodbav:"attempts"`
    CreatedAt  time.Time `json:"created_at" dynamodbav:"created_at"`
    TTL        int64     `json:"-" dynamodbav:"ttl"`
}

func (o *OTP) SetKeys() {
    o.PK = "OTP#" + o.Phone
    o.SK = "METADATA"
    o.EntityType = "OTP"
}

type SendOTPRequest struct {
    Phone string `json:"phone" validate:"required,e164"`
}

type VerifyOTPRequest struct {
    Phone string `json:"phone" validate:"required,e164"`
    Code  string `json:"code" validate:"required,len=6"`
}
```

**Step 5: Fix Order entity — add GSI2 keys**

In `internal/domain/order.go`, modify `Order.SetKeys()`:

```go
func (o *Order) SetKeys() {
    o.PK = "ORDER#" + o.ID
    o.SK = "METADATA"
    o.GSI1PK = "CUSTOMER#" + o.CustomerID
    o.GSI1SK = o.CreatedAt.Format("2006-01-02T15:04:05Z")
    o.GSI2PK = "ORDER#ALL"
    o.GSI2SK = o.CreatedAt.Format("2006-01-02T15:04:05Z")
    o.EntityType = "ORDER"
}
```

Add `GSI2PK` and `GSI2SK` fields to the Order struct if not already present.

**Step 6: Fix Customer entity — add GSI2 keys**

In `internal/domain/order.go`, modify `Customer.SetKeys()`:

```go
func (c *Customer) SetKeys() {
    c.PK = "CUSTOMER#" + c.ID
    c.SK = "METADATA"
    c.GSI1PK = "CUSTOMER_EMAIL"
    c.GSI1SK = c.Email
    c.GSI2PK = "CUSTOMER#ALL"
    c.GSI2SK = c.CreatedAt.Format("2006-01-02T15:04:05Z")
    c.EntityType = "CUSTOMER"
}
```

Add `PhoneVerified bool` field and `GSI2PK`/`GSI2SK` fields to Customer struct.

**Step 7: Add OrderNumberIndex and CustomerPhoneIndex structs**

In `internal/domain/order.go`:

```go
type OrderNumberIndex struct {
    PK         string `json:"-" dynamodbav:"PK"`
    SK         string `json:"-" dynamodbav:"SK"`
    EntityType string `json:"-" dynamodbav:"entity_type"`
    OrderID    string `json:"order_id" dynamodbav:"order_id"`
}

func (o *OrderNumberIndex) SetKeys(orderNumber string) {
    o.PK = "ORDER_NUMBER#" + orderNumber
    o.SK = "METADATA"
    o.EntityType = "ORDER_NUMBER_INDEX"
}

type CustomerPhoneIndex struct {
    PK         string `json:"-" dynamodbav:"PK"`
    SK         string `json:"-" dynamodbav:"SK"`
    EntityType string `json:"-" dynamodbav:"entity_type"`
    CustomerID string `json:"customer_id" dynamodbav:"customer_id"`
}

func (c *CustomerPhoneIndex) SetKeys(phone string) {
    c.PK = "CUSTOMER_PHONE#" + phone
    c.SK = "METADATA"
    c.EntityType = "CUSTOMER_PHONE_INDEX"
}
```

**Step 8: Fix PricingRule — add GSI2 keys**

In `internal/domain/entity.go`, modify `PricingRule.SetKeys()` to add:

```go
p.GSI2PK = "PRICING_RULE#ALL"
p.GSI2SK = "PRICING_RULE#" + p.ID
```

Add `GSI2PK`/`GSI2SK` fields to PricingRule struct.

**Step 9: Fix Inventory — add sparse GSI2 for low stock**

Add `GSI2PK`/`GSI2SK` fields to Inventory struct. Do NOT set them in SetKeys() — they are set conditionally in the repository when stock changes.

**Step 10: Commit**

```bash
git add internal/domain/
git commit -m "feat: add B2C domain entities — cart, payment, shipment, OTP, fix GSI keys"
```

---

### Task 3: Add B2C Repository Interfaces

**Files:**
- Modify: `internal/domain/repository.go`
- Create: `internal/domain/store_repository.go`

**Step 1: Create `internal/domain/store_repository.go`**

```go
package domain

import "context"

type CartRepository interface {
    GetCart(ctx context.Context, cartPK string) (*CartWithItems, error)
    PutCartItem(ctx context.Context, item *CartItem) error
    UpdateCartItem(ctx context.Context, cartPK, productID string, quantity int, totalPrice int64) error
    DeleteCartItem(ctx context.Context, cartPK, productID string) error
    UpdateCartHeader(ctx context.Context, cart *Cart) error
    ClearCart(ctx context.Context, cartPK string) error
}

type PaymentRepository interface {
    Create(ctx context.Context, payment *Payment) error
    GetByID(ctx context.Context, id string) (*Payment, error)
    GetByOrderID(ctx context.Context, orderID string) (*Payment, error)
    GetByMerchantTxnID(ctx context.Context, merchantTxnID string) (*Payment, error)
    UpdateStatus(ctx context.Context, id string, status PaymentStatus, updates map[string]interface{}) error
}

type ShipmentRepository interface {
    Create(ctx context.Context, shipment *Shipment) error
    GetByOrderID(ctx context.Context, orderID string) (*Shipment, error)
    UpdateStatus(ctx context.Context, orderID, shipmentID string, status ShipmentStatus, updates map[string]interface{}) error
}

type OTPRepository interface {
    Store(ctx context.Context, otp *OTP) error
    Get(ctx context.Context, phone string) (*OTP, error)
    IncrementAttempts(ctx context.Context, phone string) error
    Delete(ctx context.Context, phone string) error
}

type CustomerTokenStore interface {
    StoreToken(ctx context.Context, customerID, tokenHash string, ttl int64) error
    ValidateToken(ctx context.Context, customerID, tokenHash string) (bool, error)
    RevokeToken(ctx context.Context, customerID, tokenHash string) error
    RevokeAllTokens(ctx context.Context, customerID string) error
}
```

**Step 2: Commit**

```bash
git add internal/domain/store_repository.go
git commit -m "feat: add B2C repository interfaces"
```

---

### Task 4: Add B2C Service Interfaces

**Files:**
- Create: `internal/domain/store_service.go`

**Step 1: Create `internal/domain/store_service.go`**

```go
package domain

import "context"

type CustomerAuthService interface {
    SendOTP(ctx context.Context, phone string) error
    VerifyOTP(ctx context.Context, phone, code string) (*Customer, *TokenPair, bool, error)
    RefreshToken(ctx context.Context, refreshToken string) (*Customer, *TokenPair, error)
    Logout(ctx context.Context, customerID, refreshToken string) error
}

type TokenPair struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
}

type CartService interface {
    GetCart(ctx context.Context, customerID string) (*CartWithItems, error)
    AddItem(ctx context.Context, customerID string, req AddCartItemRequest) (*CartWithItems, error)
    UpdateItemQuantity(ctx context.Context, customerID, productID string, quantity int) (*CartWithItems, error)
    RemoveItem(ctx context.Context, customerID, productID string) (*CartWithItems, error)
    ClearCart(ctx context.Context, customerID string) error
    MergeGuestCart(ctx context.Context, customerID string, items []AddCartItemRequest) (*CartWithItems, error)
}

type CheckoutService interface {
    CheckServiceability(ctx context.Context, customerID, pincode string) (*ServiceabilityResult, error)
    Initiate(ctx context.Context, customerID string, req CheckoutRequest) (*CheckoutResult, error)
    GetPaymentStatus(ctx context.Context, customerID, orderID string) (*PaymentStatusResult, error)
}

type CheckoutRequest struct {
    ShippingAddressID string `json:"shipping_address_id" validate:"required"`
    CourierID         *int   `json:"courier_id,omitempty"`
}

type CheckoutResult struct {
    Order       *Order `json:"order"`
    RedirectURL string `json:"redirect_url"`
    MerchantTxnID string `json:"merchant_txn_id"`
}

type PaymentStatusResult struct {
    PaymentStatus PaymentStatus `json:"payment_status"`
    Order         *Order        `json:"order"`
}

type PaymentService interface {
    InitiatePayment(ctx context.Context, req InitiatePaymentRequest) (*PaymentResponse, error)
    HandleWebhook(ctx context.Context, payload []byte, signature string) error
    GetByOrderID(ctx context.Context, orderID string) (*Payment, error)
    GetByMerchantTxnID(ctx context.Context, merchantTxnID string) (*Payment, error)
    RefundPayment(ctx context.Context, paymentID string, amount int64, reason string) error
}

type PaymentResponse struct {
    PaymentID     string `json:"payment_id"`
    RedirectURL   string `json:"redirect_url"`
    MerchantTxnID string `json:"merchant_txn_id"`
}

type ShippingService interface {
    CheckServiceability(ctx context.Context, pickupPincode, deliveryPincode string, weightGrams int) (*ServiceabilityResult, error)
    CreateShipment(ctx context.Context, order *Order) (*Shipment, error)
    TrackShipment(ctx context.Context, orderID string) (*Shipment, error)
    HandleWebhook(ctx context.Context, payload []byte, token string) error
}
```

**Step 2: Commit**

```bash
git add internal/domain/store_service.go
git commit -m "feat: add B2C service interfaces"
```

---

## Phase 3: Fix Existing Scan Operations

### Task 5: Fix OrderRepository — Replace Scan with GSI2 Query

**Files:**
- Modify: `internal/repository/dynamodb/order_repository.go`
- Test: `go test -v -run TestOrderList ./internal/repository/...` (integration test — if it exists)

**Step 1: Replace `List()` method**

Replace the Scan-based `List()` in `order_repository.go` with a Query on GSI2:

```go
func (r *OrderRepository) List(ctx context.Context, req domain.ListOrdersRequest) (*domain.PaginatedResponse, error) {
    input := &dynamodb.QueryInput{
        TableName:              aws.String(r.client.ordersTable),
        IndexName:              aws.String("GSI2"),
        KeyConditionExpression: aws.String("GSI2PK = :pk"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":pk": &types.AttributeValueMemberS{Value: "ORDER#ALL"},
        },
        ScanIndexForward: aws.Bool(false), // newest first
        Limit:            aws.Int32(int32(req.Limit)),
    }

    // Add cursor for pagination
    if req.Cursor != "" {
        exclusiveStartKey, err := decodeCursor(req.Cursor)
        if err != nil {
            return nil, errors.New(errors.ErrCodeValidation, "Invalid cursor")
        }
        input.ExclusiveStartKey = exclusiveStartKey
    }

    // Build filter expressions for status, payment_status, date range
    filters := []string{}
    if req.Status != nil {
        filters = append(filters, "#status = :status")
        input.ExpressionAttributeValues[":status"] = &types.AttributeValueMemberS{Value: string(*req.Status)}
        if input.ExpressionAttributeNames == nil {
            input.ExpressionAttributeNames = map[string]string{}
        }
        input.ExpressionAttributeNames["#status"] = "status"
    }
    if req.PaymentStatus != nil {
        filters = append(filters, "payment_status = :paymentStatus")
        input.ExpressionAttributeValues[":paymentStatus"] = &types.AttributeValueMemberS{Value: string(*req.PaymentStatus)}
    }
    if len(filters) > 0 {
        filterExpr := strings.Join(filters, " AND ")
        input.FilterExpression = aws.String(filterExpr)
    }

    result, err := r.client.db.Query(ctx, input)
    if err != nil {
        return nil, errors.Wrap(err, "Failed to list orders")
    }

    var orders []domain.Order
    if err := attributevalue.UnmarshalListOfMaps(result.Items, &orders); err != nil {
        return nil, errors.Internal("Failed to unmarshal orders")
    }

    var nextCursor string
    if result.LastEvaluatedKey != nil {
        nextCursor = encodeCursor(result.LastEvaluatedKey)
    }

    return &domain.PaginatedResponse{
        Items:      orders,
        NextCursor: nextCursor,
        HasMore:    result.LastEvaluatedKey != nil,
    }, nil
}
```

**Step 2: Add `GetByOrderNumber()` using lookup item**

```go
func (r *OrderRepository) GetByOrderNumber(ctx context.Context, orderNumber string) (*domain.Order, error) {
    // Step 1: lookup order ID from index item
    result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
        TableName: aws.String(r.client.ordersTable),
        Key: map[string]types.AttributeValue{
            "PK": &types.AttributeValueMemberS{Value: "ORDER_NUMBER#" + orderNumber},
            "SK": &types.AttributeValueMemberS{Value: "METADATA"},
        },
    })
    if err != nil {
        return nil, errors.Wrap(err, "Failed to lookup order number")
    }
    if result.Item == nil {
        return nil, errors.NotFound("Order")
    }

    var index domain.OrderNumberIndex
    if err := attributevalue.UnmarshalMap(result.Item, &index); err != nil {
        return nil, errors.Internal("Failed to unmarshal order number index")
    }

    // Step 2: get full order
    return r.GetByID(ctx, index.OrderID)
}
```

**Step 3: Update `Create()` to write order number index**

Add the order number index item to the TransactWriteItems in the Create method:

```go
// Add to transact items:
{
    Put: &types.Put{
        TableName: aws.String(r.client.ordersTable),
        Item: orderNumberIndexAV,
        ConditionExpression: aws.String("attribute_not_exists(PK)"),
    },
},
```

**Step 4: Verify**

Run: `make test`
Expected: All existing tests pass.

**Step 5: Commit**

```bash
git add internal/repository/dynamodb/order_repository.go
git commit -m "fix: replace order list scan with GSI2 query, add order number index"
```

---

### Task 6: Fix CustomerRepository — Replace Scan with GSI2 Query

**Files:**
- Modify: `internal/repository/dynamodb/order_repository.go` (customer methods are in the order repo file)

**Step 1: Replace `List()` with GSI2 Query**

Same pattern as Task 5 — query GSI2PK=`CUSTOMER#ALL` with cursor pagination and FilterExpression for search.

**Step 2: Add phone index write to `Create()`**

Add `CUSTOMER_PHONE#<phone>` item write to customer creation TransactWriteItems.

**Step 3: Add `GetByPhone()` method**

```go
func (r *OrderRepository) GetCustomerByPhone(ctx context.Context, phone string) (*domain.Customer, error) {
    result, err := r.client.db.GetItem(ctx, &dynamodb.GetItemInput{
        TableName: aws.String(r.client.ordersTable),
        Key: map[string]types.AttributeValue{
            "PK": &types.AttributeValueMemberS{Value: "CUSTOMER_PHONE#" + phone},
            "SK": &types.AttributeValueMemberS{Value: "METADATA"},
        },
    })
    if err != nil {
        return nil, errors.Wrap(err, "Failed to lookup customer by phone")
    }
    if result.Item == nil {
        return nil, errors.NotFound("Customer")
    }
    var index domain.CustomerPhoneIndex
    if err := attributevalue.UnmarshalMap(result.Item, &index); err != nil {
        return nil, errors.Internal("Failed to unmarshal phone index")
    }
    return r.GetCustomerByID(ctx, index.CustomerID)
}
```

**Step 4: Commit**

```bash
git add internal/repository/dynamodb/order_repository.go
git commit -m "fix: replace customer list scan with GSI2 query, add phone index"
```

---

### Task 7: Fix PricingRuleRepository & InventoryRepository Scans

**Files:**
- Modify: `internal/repository/dynamodb/pricing_repository.go`
- Modify: `internal/repository/dynamodb/inventory_repository.go`

**Step 1: Fix PricingRule List — GSI2 Query**

Replace Scan with Query on GSI2PK=`PRICING_RULE#ALL`.

**Step 2: Fix Inventory GetLowStockProducts — sparse GSI2 Query**

Replace Scan with Query on GSI2PK=`LOW_STOCK`.

**Step 3: Update Inventory stock operations to maintain sparse GSI2**

In `AddStock`, `RemoveStock`, `AdjustStock` — after updating quantities, check if `available_qty <= low_stock_threshold`. If yes, set GSI2PK/GSI2SK. If no, remove them (set to empty string or use REMOVE).

**Step 4: Commit**

```bash
git add internal/repository/dynamodb/pricing_repository.go internal/repository/dynamodb/inventory_repository.go
git commit -m "fix: replace pricing rule and inventory scans with GSI2 queries"
```

---

### Task 8: Fix AuditRepository Scans & Broken GSI Queries

**Files:**
- Modify: `internal/domain/audit.go` (fix SetKeys)
- Modify: `internal/repository/dynamodb/audit_repository.go`

**Step 1: Fix AuditLog SetKeys**

```go
func (a *AuditLog) SetKeys() {
    a.PK = "AUDIT#" + a.CreatedAt.Format("2006-01-02")
    a.SK = a.CreatedAt.Format("15:04:05.000Z") + "#" + a.ID
    a.GSI1PK = string(a.EntityType) + "#" + a.EntityID  // entity-based lookup
    a.GSI1SK = a.CreatedAt.Format("2006-01-02T15:04:05Z")
    a.GSI2PK = "USER#" + a.UserID                        // user-based lookup
    a.GSI2SK = a.CreatedAt.Format("2006-01-02T15:04:05Z")
    a.EntityType_DDB = "AUDIT_LOG"
}
```

**Step 2: Fix List() — query by date partitions**

Replace Scan with multiple Queries, one per day in the requested date range:

```go
// For each date in range: Query PK=AUDIT#<YYYY-MM-DD>
// Merge results, apply sort/limit
```

**Step 3: Fix GetByEntity — use GSI1**

Query GSI1PK=`<entityType>#<entityID>`.

**Step 4: Fix GetByUser — use GSI2**

Query GSI2PK=`USER#<userID>`.

**Step 5: Commit**

```bash
git add internal/domain/audit.go internal/repository/dynamodb/audit_repository.go
git commit -m "fix: replace audit scans with date-partitioned queries, fix GSI patterns"
```

---

## Phase 4: External Gateway Clients

### Task 9: MSG91 SMS Gateway Client

**Files:**
- Create: `internal/gateway/sms/client.go`
- Create: `internal/gateway/sms/types.go`
- Test: `internal/gateway/sms/client_test.go`

**Step 1: Write test for SMS client**

```go
func TestSMSClient_SendSMS(t *testing.T) {
    // Use httptest server to mock MSG91 API
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "/api/v5/flow/", r.URL.Path)
        assert.Equal(t, "test-auth-key", r.Header.Get("authkey"))
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"type": "success"})
    }))
    defer server.Close()

    client := sms.NewClient(sms.Config{BaseURL: server.URL, AuthKey: "test-auth-key", OTPTemplateID: "tmpl-123"})
    err := client.SendOTP(context.Background(), "+919876543210", "123456")
    assert.NoError(t, err)
}
```

**Step 2: Run test, verify it fails**

Run: `go test -v -run TestSMSClient ./internal/gateway/sms/...`
Expected: FAIL (package doesn't exist)

**Step 3: Implement SMS client**

```go
package sms

type Config struct {
    BaseURL       string
    AuthKey       string
    OTPTemplateID string
}

type Client struct {
    config     Config
    httpClient *http.Client
}

func NewClient(config Config) *Client {
    return &Client{config: config, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) SendOTP(ctx context.Context, phone, code string) error {
    payload := map[string]interface{}{
        "template_id": c.config.OTPTemplateID,
        "short_url":   "0",
        "recipients":  []map[string]string{{"mobiles": phone, "otp": code}},
    }
    body, _ := json.Marshal(payload)
    req, _ := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/api/v5/flow/", bytes.NewReader(body))
    req.Header.Set("authkey", c.config.AuthKey)
    req.Header.Set("Content-Type", "application/json")
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to send OTP SMS: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("MSG91 returned status %d", resp.StatusCode)
    }
    return nil
}
```

**Step 4: Run test, verify it passes**

Run: `go test -v -run TestSMSClient ./internal/gateway/sms/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/gateway/sms/
git commit -m "feat: add MSG91 SMS gateway client"
```

---

### Task 10: PhonePe Payment Gateway Client

**Files:**
- Create: `internal/gateway/phonepe/client.go`
- Create: `internal/gateway/phonepe/types.go`
- Test: `internal/gateway/phonepe/client_test.go`

**Step 1: Write tests**

Test `InitiatePayment` (mock PhonePe API, verify payload encoding + X-VERIFY signature generation).
Test `VerifyWebhookSignature` (verify callback signature validation logic).

**Step 2: Run tests, verify they fail**

**Step 3: Implement PhonePe client**

Key functions:
- `InitiatePayment(req) -> (redirectURL, error)` — base64 encode payload, generate SHA256 X-VERIFY, POST to `/pg/v1/pay`
- `VerifyWebhookSignature(payload, xVerify) -> bool` — SHA256 verify callback
- `CheckPaymentStatus(merchantTxnID) -> (PaymentStatusResponse, error)` — GET `/pg/v1/status/<merchantId>/<txnId>`

**Step 4: Run tests, verify they pass**

**Step 5: Commit**

```bash
git add internal/gateway/phonepe/
git commit -m "feat: add PhonePe payment gateway client"
```

---

### Task 11: Shiprocket Shipping Gateway Client

**Files:**
- Create: `internal/gateway/shiprocket/client.go`
- Create: `internal/gateway/shiprocket/types.go`
- Test: `internal/gateway/shiprocket/client_test.go`

**Step 1: Write tests**

Test `CheckServiceability`, `CreateOrder`, `AssignAWB`, `TrackShipment` against mock HTTP server.

**Step 2: Run tests, verify they fail**

**Step 3: Implement Shiprocket client**

Key functions:
- `Authenticate() -> (token, error)` — POST `/auth/login`, cache token
- `CheckServiceability(pickup, delivery, weight) -> ([]CourierOption, error)` — GET `/courier/serviceability`
- `CreateOrder(order) -> (shiprocketOrderID, error)` — POST `/orders/create/adhoc`
- `AssignAWB(shipmentID, courierID) -> (awb, error)` — POST `/courier/assign/awb`
- `GenerateLabel(shipmentID) -> (labelURL, error)` — POST `/courier/generate/label`
- `TrackByAWB(awb) -> (TrackingResult, error)` — GET `/courier/track/awb/<awb>`

**Step 4: Run tests, verify they pass**

**Step 5: Commit**

```bash
git add internal/gateway/shiprocket/
git commit -m "feat: add Shiprocket shipping gateway client"
```

---

## Phase 5: Customer Auth

### Task 12: OTP Repository

**Files:**
- Create: `internal/repository/dynamodb/otp_repository.go`
- Test: `internal/repository/dynamodb/otp_repository_test.go` (unit test with mocked DynamoDB client)

**Step 1: Write tests for Store, Get, IncrementAttempts, Delete**

**Step 2: Implement OTP repository**

Following the exact patterns from coupon_repository.go:
- `Store()` → PutItem with TTL (5 min from now)
- `Get()` → GetItem
- `IncrementAttempts()` → UpdateItem `ADD attempts :one`
- `Delete()` → DeleteItem

**Step 3: Commit**

```bash
git add internal/repository/dynamodb/otp_repository.go
git commit -m "feat: add OTP repository"
```

---

### Task 13: Customer Token Store

**Files:**
- Create: `internal/repository/dynamodb/customer_token_store.go`

**Step 1: Implement customer token store**

Same pattern as existing `token_store.go` but with `CUST_TOKEN#<customerID>` prefix instead of `USER#<userID>`.

**Step 2: Commit**

```bash
git add internal/repository/dynamodb/customer_token_store.go
git commit -m "feat: add customer token store"
```

---

### Task 14: CustomerAuthService

**Files:**
- Create: `internal/service/customer_auth_service.go`
- Test: `internal/service/customer_auth_service_test.go`

**Step 1: Write tests**

Test `SendOTP` (generates code, hashes, stores, calls SMS gateway).
Test `VerifyOTP` — success (valid code, creates/finds customer, returns tokens).
Test `VerifyOTP` — failure (wrong code, increments attempts).
Test `VerifyOTP` — max attempts exceeded.
Test `VerifyOTP` — auto-register new customer.
Test `RefreshToken` — valid token returns new pair.
Test `Logout` — revokes token.

**Step 2: Implement service**

```go
type CustomerAuthService struct {
    otpRepo        domain.OTPRepository
    customerRepo   domain.CustomerRepository
    tokenStore     domain.CustomerTokenStore
    smsGateway     *sms.Client
    jwtSecret      string
    logger         *logger.Logger
}
```

Key logic:
- `SendOTP`: generate 6-digit code → SHA256 hash → store with TTL → call smsGateway.SendOTP
- `VerifyOTP`: get OTP → check attempts < 3 → verify hash → delete OTP → find/create customer → generate JWT pair → store refresh token → return
- JWT claims: `{CustomerID, Phone, Email, ExpiresAt}`

**Step 3: Run tests, verify they pass**

**Step 4: Commit**

```bash
git add internal/service/customer_auth_service.go internal/service/customer_auth_service_test.go
git commit -m "feat: add customer auth service with OTP"
```

---

### Task 15: Customer Auth Middleware

**Files:**
- Create: `internal/middleware/customer_auth.go`

**Step 1: Implement customer auth middleware**

Same pattern as `middleware.Auth` but:
- Reads `store_token` cookie (not `access_token`)
- Validates customer JWT claims (CustomerID, Phone)
- Sets `CustomerIDKey` and `CustomerKey` in context

```go
type CustomerAuth struct {
    customerAuthService domain.CustomerAuthService
    jwtSecret           string
    logger              *logger.Logger
}

func (a *CustomerAuth) Authenticate(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token, err := r.Cookie("store_token")
        if err != nil {
            // try Authorization header
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
                response.Unauthorized(w, "Authentication required")
                return
            }
            tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
        } else {
            tokenStr = token.Value
        }
        claims, err := validateCustomerJWT(tokenStr, a.jwtSecret)
        if err != nil {
            response.Unauthorized(w, "Invalid or expired token")
            return
        }
        ctx := context.WithValue(r.Context(), CustomerIDKey, claims.CustomerID)
        ctx = context.WithValue(ctx, CustomerKey, &domain.Customer{ID: claims.CustomerID, Phone: claims.Phone, Email: claims.Email})
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

**Step 2: Commit**

```bash
git add internal/middleware/customer_auth.go
git commit -m "feat: add customer auth middleware"
```

---

### Task 16: Store Auth Handlers

**Files:**
- Create: `internal/handler/store/auth_handler.go`

**Step 1: Implement store auth handler**

```go
type AuthHandler struct {
    customerAuthService *service.CustomerAuthService
    validation          *middleware.Validation
}

func (h *AuthHandler) Routes(customerAuth *middleware.CustomerAuth) chi.Router {
    r := chi.NewRouter()
    r.With(middleware.ValidateJSONTyped[domain.SendOTPRequest](h.validation)).Post("/otp/send", h.SendOTP)
    r.With(middleware.ValidateJSONTyped[domain.VerifyOTPRequest](h.validation)).Post("/otp/verify", h.VerifyOTP)
    r.Post("/refresh", h.RefreshToken)
    r.With(customerAuth.Authenticate).Post("/logout", h.Logout)
    return r
}
```

Handlers set HttpOnly cookies (`store_token`, `store_refresh`) on successful verify/refresh.

**Step 2: Commit**

```bash
git add internal/handler/store/
git commit -m "feat: add store auth handlers"
```

---

## Phase 6: Cart

### Task 17: Cart Repository

**Files:**
- Create: `internal/repository/dynamodb/cart_repository.go`
- Test: `internal/repository/dynamodb/cart_repository_test.go`

**Step 1: Write tests for all cart operations**

**Step 2: Implement cart repository**

- `GetCart()` → Query PK=`CART#<id>` (returns header + items in one query)
- `PutCartItem()` → PutItem with TTL
- `UpdateCartItem()` → UpdateItem on SK=`ITEM#<productID>`
- `DeleteCartItem()` → DeleteItem
- `UpdateCartHeader()` → UpdateItem on SK=`METADATA`
- `ClearCart()` → Query + BatchWriteItem (delete all)

**Step 3: Commit**

```bash
git add internal/repository/dynamodb/cart_repository.go
git commit -m "feat: add cart repository"
```

---

### Task 18: Cart Service

**Files:**
- Create: `internal/service/cart_service.go`
- Test: `internal/service/cart_service_test.go`

**Step 1: Write tests**

Test `AddItem` — validates product exists + ACTIVE, validates stock, validates quote (if custom size), denormalizes product info, writes item + updates header.
Test `AddItem` — existing item updates quantity.
Test `UpdateItemQuantity` — validates stock, updates item + header.
Test `UpdateItemQuantity` — quantity 0 removes item.
Test `RemoveItem` — removes item, updates header.
Test `ClearCart` — clears all items.
Test `MergeGuestCart` — merges items, keeps higher quantity.

**Step 2: Implement cart service**

```go
type CartService struct {
    cartRepo      domain.CartRepository
    productRepo   domain.ProductRepository
    inventoryRepo domain.InventoryRepository
    pricingRepo   domain.PricingRepository
    logger        *logger.Logger
}
```

30-day TTL constant: `const cartTTLDays = 30`

**Step 3: Run tests, verify they pass**

**Step 4: Commit**

```bash
git add internal/service/cart_service.go internal/service/cart_service_test.go
git commit -m "feat: add cart service"
```

---

### Task 19: Cart Handlers

**Files:**
- Create: `internal/handler/store/cart_handler.go`

**Step 1: Implement cart handler**

All routes require customer auth. Extract `customerID` from context.

```go
func (h *CartHandler) Routes() chi.Router {
    r := chi.NewRouter()
    r.Get("/", h.GetCart)
    r.With(middleware.ValidateJSONTyped[domain.AddCartItemRequest](h.validation)).Post("/items", h.AddItem)
    r.With(middleware.ValidateJSONTyped[domain.UpdateCartItemRequest](h.validation)).Patch("/items/{productID}", h.UpdateQuantity)
    r.Delete("/items/{productID}", h.RemoveItem)
    r.Delete("/", h.ClearCart)
    r.With(middleware.ValidateJSONTyped[domain.MergeCartRequest](h.validation)).Post("/merge", h.MergeGuestCart)
    return r
}
```

**Step 2: Commit**

```bash
git add internal/handler/store/cart_handler.go
git commit -m "feat: add cart handlers"
```

---

## Phase 7: Store Catalog API

### Task 20: Store Catalog Handlers

**Files:**
- Create: `internal/handler/store/catalog_handler.go`

**Step 1: Implement catalog handler**

Public routes (no auth). Reuses existing `ProductService` and `CategoryService` but:
- Filters to `status=ACTIVE` only
- Excludes `cost_price` from product responses
- Returns `in_stock` boolean instead of raw inventory numbers

```go
type CatalogHandler struct {
    productService  *service.ProductService
    categoryService *service.CategoryService
}

func (h *CatalogHandler) Routes() chi.Router {
    r := chi.NewRouter()
    r.Get("/categories", h.ListCategories)
    r.Get("/categories/{idOrSlug}", h.GetCategory)
    r.Get("/products", h.ListProducts)
    r.Get("/products/search", h.SearchProducts)
    r.Get("/products/{idOrSlug}", h.GetProduct)
    r.Get("/products/{id}/availability", h.CheckAvailability)
    return r
}
```

For response sanitization, create a helper that strips `cost_price` and converts inventory to boolean `in_stock`:

```go
type StoreProduct struct {
    // all Product fields EXCEPT CostPrice
    InStock bool `json:"in_stock"`
}

func toStoreProduct(p *domain.Product) *StoreProduct { ... }
```

**Step 2: Commit**

```bash
git add internal/handler/store/catalog_handler.go
git commit -m "feat: add store catalog handlers"
```

---

## Phase 8: Payment Service

### Task 21: Payment Repository

**Files:**
- Create: `internal/repository/dynamodb/payment_repository.go`

**Step 1: Implement payment repository**

- `Create()` → PutItem
- `GetByID()` → GetItem PK=`PAYMENT#<id>`
- `GetByOrderID()` → Query GSI1 GSI1PK=`ORDER#<orderID>`, Limit=1
- `GetByMerchantTxnID()` → Query GSI2 GSI2PK=`PAYMENT_TXN`, GSI2SK=`<txnID>`, Limit=1
- `UpdateStatus()` → UpdateItem with dynamic SET expression

**Step 2: Commit**

```bash
git add internal/repository/dynamodb/payment_repository.go
git commit -m "feat: add payment repository"
```

---

### Task 22: Payment Service

**Files:**
- Create: `internal/service/payment_service.go`
- Test: `internal/service/payment_service_test.go`

**Step 1: Write tests**

Test `InitiatePayment` — creates payment record, calls PhonePe, returns redirect URL.
Test `HandleWebhook` — SUCCESS: updates payment, confirms order, triggers notification.
Test `HandleWebhook` — FAILED: updates payment, releases inventory.
Test `HandleWebhook` — idempotent (already processed returns OK).
Test `HandleWebhook` — invalid signature returns error.

**Step 2: Implement payment service**

```go
type PaymentService struct {
    paymentRepo   domain.PaymentRepository
    orderRepo     domain.OrderRepository
    inventoryRepo domain.InventoryRepository
    phonePe       *phonepe.Client
    logger        *logger.Logger
}
```

Key: `HandleWebhook` must be idempotent — check if payment status is already SUCCESS/FAILED before processing.

**Step 3: Run tests, verify they pass**

**Step 4: Commit**

```bash
git add internal/service/payment_service.go internal/service/payment_service_test.go
git commit -m "feat: add payment service with PhonePe integration"
```

---

### Task 23: Webhook Handlers

**Files:**
- Create: `internal/handler/store/webhook_handler.go`

**Step 1: Implement webhook handler**

```go
type WebhookHandler struct {
    paymentService  *service.PaymentService
    shippingService *service.ShippingService
}

func (h *WebhookHandler) Routes() chi.Router {
    r := chi.NewRouter()
    r.Post("/phonepe", h.PhonePeWebhook)
    r.Post("/shiprocket", h.ShiprocketWebhook)
    return r
}
```

PhonePe webhook: read raw body, pass to `paymentService.HandleWebhook(body, xVerifyHeader)`.
Shiprocket webhook: verify token, pass to `shippingService.HandleWebhook(body, token)`.

**Step 2: Commit**

```bash
git add internal/handler/store/webhook_handler.go
git commit -m "feat: add payment and shipping webhook handlers"
```

---

## Phase 9: Checkout

### Task 24: Checkout Service

**Files:**
- Create: `internal/service/checkout_service.go`
- Test: `internal/service/checkout_service_test.go`

**Step 1: Write tests**

Test `CheckServiceability` — calls Shiprocket, returns courier options.
Test `Initiate` — full happy path: get cart → validate stock → calculate totals → create order (TransactWriteItems) → initiate payment → clear cart → return redirect URL.
Test `Initiate` — out of stock fails with error.
Test `Initiate` — empty cart fails.
Test `GetPaymentStatus` — returns current payment status + order.

**Step 2: Implement checkout service**

```go
type CheckoutService struct {
    cartService     domain.CartService
    orderService    domain.OrderService
    paymentService  domain.PaymentService
    shippingService domain.ShippingService
    inventoryRepo   domain.InventoryRepository
    customerRepo    domain.CustomerRepository
    logger          *logger.Logger
}
```

The `Initiate` method is the core orchestrator (see design doc Section 6.3 for the full sequence).

**Step 3: Run tests, verify they pass**

**Step 4: Commit**

```bash
git add internal/service/checkout_service.go internal/service/checkout_service_test.go
git commit -m "feat: add checkout service — orchestrates cart, order, payment, shipping"
```

---

### Task 25: Checkout Handlers

**Files:**
- Create: `internal/handler/store/checkout_handler.go`

**Step 1: Implement checkout handler**

```go
func (h *CheckoutHandler) Routes() chi.Router {
    r := chi.NewRouter()
    r.With(middleware.ValidateJSONTyped[domain.ServiceabilityRequest](h.validation)).Post("/serviceability", h.CheckServiceability)
    r.With(middleware.ValidateJSONTyped[domain.CheckoutRequest](h.validation)).Post("/initiate", h.Initiate)
    r.Get("/payment-status/{orderID}", h.GetPaymentStatus)
    return r
}
```

**Step 2: Commit**

```bash
git add internal/handler/store/checkout_handler.go
git commit -m "feat: add checkout handlers"
```

---

## Phase 10: Shipping

### Task 26: Shipping Service

**Files:**
- Create: `internal/service/shipping_service.go`
- Test: `internal/service/shipping_service_test.go`
- Create: `internal/repository/dynamodb/shipment_repository.go`

**Step 1: Implement shipment repository**

- `Create()` → PutItem PK=`ORDER#<orderID>`, SK=`SHIPMENT#<id>`
- `GetByOrderID()` → Query PK=`ORDER#<orderID>`, SK begins_with `SHIPMENT#`
- `UpdateStatus()` → UpdateItem

**Step 2: Write tests for shipping service**

Test `CheckServiceability` — calls Shiprocket.
Test `CreateShipment` — calls Shiprocket create + assign AWB + generate label, stores shipment.
Test `HandleWebhook` — updates shipment status, updates order status (SHIPPED/DELIVERED).

**Step 3: Implement shipping service**

```go
type ShippingService struct {
    shipmentRepo domain.ShipmentRepository
    orderRepo    domain.OrderRepository
    shiprocket   *shiprocket.Client
    logger       *logger.Logger
}
```

**Step 4: Run tests, verify they pass**

**Step 5: Commit**

```bash
git add internal/repository/dynamodb/shipment_repository.go internal/service/shipping_service.go internal/service/shipping_service_test.go
git commit -m "feat: add shipping service with Shiprocket integration"
```

---

## Phase 11: B2C Order & Tracking Endpoints

### Task 27: Store Order Handlers

**Files:**
- Create: `internal/handler/store/order_handler.go`

**Step 1: Implement store order handler**

Customer-facing order endpoints. All require customer auth. Filter by customer_id from JWT.

```go
func (h *OrderHandler) Routes() chi.Router {
    r := chi.NewRouter()
    r.Get("/", h.ListMyOrders)
    r.Get("/{id}", h.GetOrder)
    r.Post("/{id}/cancel", h.CancelOrder)
    return r
}
```

`ListMyOrders`: queries GSI1 with `CUSTOMER#<customerID>` from context.
`GetOrder`: validates `order.CustomerID == contextCustomerID` before returning.
`CancelOrder`: validates ownership + status is PENDING or CONFIRMED.

**Step 2: Commit**

```bash
git add internal/handler/store/order_handler.go
git commit -m "feat: add store order handlers for customer-facing order management"
```

---

### Task 28: Store Tracking Handler & Profile Handler

**Files:**
- Create: `internal/handler/store/tracking_handler.go`
- Create: `internal/handler/store/profile_handler.go`

**Step 1: Implement tracking handler (public)**

```go
func (h *TrackingHandler) Routes() chi.Router {
    r := chi.NewRouter()
    r.Get("/{orderNumber}", h.TrackOrder)
    return r
}
```

`TrackOrder`: lookup by order number → get status history → get shipment → return timeline.

**Step 2: Implement profile handler (customer auth)**

```go
func (h *ProfileHandler) Routes() chi.Router {
    r := chi.NewRouter()
    r.Get("/", h.GetProfile)
    r.Patch("/", h.UpdateProfile)
    r.Post("/addresses", h.AddAddress)
    r.Put("/addresses/{id}", h.UpdateAddress)
    r.Delete("/addresses/{id}", h.RemoveAddress)
    return r
}
```

**Step 3: Commit**

```bash
git add internal/handler/store/tracking_handler.go internal/handler/store/profile_handler.go
git commit -m "feat: add tracking and profile handlers"
```

---

## Phase 12: Wire It All Together

### Task 29: DI Wiring & Route Mounting

**Files:**
- Modify: `internal/wire/providers.go` (add new providers)
- Modify: `internal/wire/wire.go` (add store deps struct + initialize function)
- Modify: `cmd/api/main.go` (mount store routes)
- Create: `internal/router/store.go` (store route group)

**Step 1: Add providers**

In `internal/wire/providers.go`, add provider functions for all new repositories, services, handlers, gateways:

```go
func ProvideOTPRepository(client *dynamodb.Client) domain.OTPRepository { ... }
func ProvideCustomerTokenStore(client *dynamodb.Client) domain.CustomerTokenStore { ... }
func ProvideCartRepository(client *dynamodb.Client) domain.CartRepository { ... }
func ProvidePaymentRepository(client *dynamodb.Client) domain.PaymentRepository { ... }
func ProvideShipmentRepository(client *dynamodb.Client) domain.ShipmentRepository { ... }

func ProvideSMSGateway(cfg *config.Config) *sms.Client { ... }
func ProvidePhonePeGateway(cfg *config.Config) *phonepe.Client { ... }
func ProvideShiprocketGateway(cfg *config.Config) *shiprocket.Client { ... }

func ProvideCustomerAuthService(...) *service.CustomerAuthService { ... }
func ProvideCartService(...) *service.CartService { ... }
func ProvideCheckoutService(...) *service.CheckoutService { ... }
func ProvidePaymentService(...) *service.PaymentService { ... }
func ProvideShippingService(...) *service.ShippingService { ... }

func ProvideCustomerAuthMiddleware(...) *middleware.CustomerAuth { ... }

func ProvideStoreAuthHandler(...) *store.AuthHandler { ... }
func ProvideStoreCatalogHandler(...) *store.CatalogHandler { ... }
func ProvideStoreCartHandler(...) *store.CartHandler { ... }
func ProvideStoreCheckoutHandler(...) *store.CheckoutHandler { ... }
func ProvideStoreOrderHandler(...) *store.OrderHandler { ... }
func ProvideStoreTrackingHandler(...) *store.TrackingHandler { ... }
func ProvideStoreProfileHandler(...) *store.ProfileHandler { ... }
func ProvideStoreWebhookHandler(...) *store.WebhookHandler { ... }
```

**Step 2: Create store route group**

In `internal/router/store.go`:

```go
func NewStoreRouter(r *chi.Mux, cfg Config, customerAuth *middleware.CustomerAuth, handlers StoreHandlers) {
    r.Route("/api/v1/store", func(r chi.Router) {
        // Public routes
        r.Mount("/auth", handlers.Auth.Routes(customerAuth))
        r.Mount("/categories", handlers.Catalog.CategoryRoutes())
        r.Mount("/products", handlers.Catalog.ProductRoutes())
        r.Mount("/track", handlers.Tracking.Routes())

        // Webhook routes (signature-verified, not customer auth)
        r.Mount("/webhooks", handlers.Webhook.Routes())

        // Customer-authenticated routes
        r.Group(func(r chi.Router) {
            r.Use(customerAuth.Authenticate)
            r.Mount("/me", handlers.Profile.Routes())
            r.Mount("/cart", handlers.Cart.Routes())
            r.Mount("/checkout", handlers.Checkout.Routes())
            r.Mount("/orders", handlers.Order.Routes())
        })
    })
}
```

**Step 3: Update `cmd/api/main.go`**

Add new service/handler initialization and mount `NewStoreRouter`.

**Step 4: Run `make wire` to regenerate**

Run: `make wire`
Expected: `wire_gen.go` regenerated successfully.

**Step 5: Run `make run` and test manually**

Run: `make setup-local && make run`
Test: `curl http://localhost:8080/api/v1/store/categories` should return active categories.

**Step 6: Commit**

```bash
git add internal/wire/ internal/router/store.go cmd/api/main.go
git commit -m "feat: wire all B2C services and mount store routes"
```

---

### Task 30: Add Config for External Services

**Files:**
- Modify: `internal/config/config.go`

**Step 1: Add new config fields**

```go
// PhonePe
PhonePeMerchantID  string `env:"PHONEPE_MERCHANT_ID"`
PhonePeSaltKey     string `env:"PHONEPE_SALT_KEY"`
PhonePeSaltIndex   string `env:"PHONEPE_SALT_INDEX" envDefault:"1"`
PhonePeBaseURL     string `env:"PHONEPE_BASE_URL" envDefault:"https://api-preprod.phonepe.com/apis/pg-sandbox"`
PhonePeCallbackURL string `env:"PHONEPE_CALLBACK_URL"`
PhonePeRedirectURL string `env:"PHONEPE_REDIRECT_URL"`

// Shiprocket
ShiprocketEmail    string `env:"SHIPROCKET_EMAIL"`
ShiprocketPassword string `env:"SHIPROCKET_PASSWORD"`
ShiprocketBaseURL  string `env:"SHIPROCKET_BASE_URL" envDefault:"https://apiv2.shiprocket.in/v1/external"`
ShiprocketPickupPincode string `env:"SHIPROCKET_PICKUP_PINCODE"`

// MSG91
MSG91AuthKey       string `env:"MSG91_AUTH_KEY"`
MSG91OTPTemplateID string `env:"MSG91_OTP_TEMPLATE_ID"`
MSG91BaseURL       string `env:"MSG91_BASE_URL" envDefault:"https://control.msg91.com"`

// Customer Auth
CustomerJWTSecret       string `env:"CUSTOMER_JWT_SECRET"`
CustomerAccessTokenTTL  int    `env:"CUSTOMER_ACCESS_TOKEN_TTL" envDefault:"3600"`
CustomerRefreshTokenTTL int    `env:"CUSTOMER_REFRESH_TOKEN_TTL" envDefault:"2592000"`
```

**Step 2: Add to `.env.example`**

**Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add config for PhonePe, Shiprocket, MSG91, customer auth"
```

---

## Phase 13: Next.js Storefront Setup

### Task 31: Initialize Next.js Project

**Step 1: Create project**

```bash
cd /Users/utkarsh.nigam/Desktop/CP/Homechrome
npx create-next-app@latest homechrome-store \
  --typescript --tailwind --eslint --app \
  --src-dir --import-alias "@/*" --no-turbopack
```

**Step 2: Install dependencies**

```bash
cd homechrome-store
npm install zustand @tanstack/react-query axios
npm install -D @types/node
```

**Step 3: Configure `next.config.ts`**

Add API proxy for local development:

```ts
const nextConfig = {
  images: {
    remotePatterns: [
      { protocol: 'https', hostname: '*.s3.amazonaws.com' },
      { protocol: 'http', hostname: 'localhost', port: '4566' },
    ],
  },
  async rewrites() {
    return [
      { source: '/api/:path*', destination: 'http://localhost:8080/api/:path*' },
    ];
  },
};
```

**Step 4: Set up directory structure**

Create the skeleton directory structure as defined in design doc Section 8.1.

**Step 5: Commit**

```bash
git add homechrome-store/
git commit -m "feat: initialize Next.js storefront project"
```

---

### Task 32: API Client & Auth Hooks

**Files:**
- Create: `homechrome-store/src/lib/api.ts`
- Create: `homechrome-store/src/lib/auth.ts`
- Create: `homechrome-store/src/hooks/useAuth.ts`
- Create: `homechrome-store/src/types/index.ts`

**Step 1: Implement API client**

Fetch wrapper that:
- Prepends base URL
- Unwraps `{success, data}` envelope
- Handles 401 → silent token refresh
- Passes `credentials: 'include'` for cookies

**Step 2: Implement auth hooks**

- `useAuth()` — Zustand store with `customer`, `isAuthenticated`, `login()`, `logout()`
- `sendOTP(phone)`, `verifyOTP(phone, code)` functions

**Step 3: Implement shared types**

TypeScript types matching backend JSON responses (Product, Category, Cart, Order, etc.).

**Step 4: Commit**

```bash
git add homechrome-store/src/
git commit -m "feat: add API client, auth hooks, and shared types"
```

---

### Task 33: Layout & Common Components

**Files:**
- Create: `homechrome-store/src/components/layout/Header.tsx`
- Create: `homechrome-store/src/components/layout/Footer.tsx`
- Create: `homechrome-store/src/components/layout/MobileNav.tsx`
- Create: `homechrome-store/src/components/common/Button.tsx`
- Create: `homechrome-store/src/components/common/Input.tsx`
- Modify: `homechrome-store/src/app/layout.tsx`

**Step 1: Build root layout with header/footer**

Mobile-responsive header with logo, search, cart icon (with count badge), account link.

**Step 2: Commit**

```bash
git add homechrome-store/src/
git commit -m "feat: add storefront layout and common components"
```

---

## Phase 14: Next.js Pages

### Task 34: Home Page

**Files:**
- Modify: `homechrome-store/src/app/page.tsx`
- Create: `homechrome-store/src/components/catalog/CategoryCard.tsx`

SSG with ISR (5 min). Fetches categories and featured/new-arrival products from the Go API.

**Commit:** `feat: add storefront home page with categories and featured products`

---

### Task 35: Category & Product Listing Pages

**Files:**
- Create: `homechrome-store/src/app/categories/page.tsx`
- Create: `homechrome-store/src/app/c/[slug]/page.tsx`
- Create: `homechrome-store/src/components/catalog/ProductCard.tsx`
- Create: `homechrome-store/src/components/catalog/ProductGrid.tsx`
- Create: `homechrome-store/src/components/catalog/FilterSidebar.tsx`

SSG with ISR (2 min). Product listing with client-side attribute filtering (material, color, weave type, price range).

**Commit:** `feat: add category and product listing pages with filters`

---

### Task 36: Product Detail Page

**Files:**
- Create: `homechrome-store/src/app/p/[slug]/page.tsx`

SSG with ISR (2 min). Image gallery, product info, add-to-cart button. Client-side stock check on mount. Custom dimension selector + pricing engine integration for `AllowCustomDimensions` products.

**Commit:** `feat: add product detail page with gallery and add-to-cart`

---

### Task 37: Cart Page

**Files:**
- Create: `homechrome-store/src/app/cart/page.tsx`
- Create: `homechrome-store/src/components/cart/CartItem.tsx`
- Create: `homechrome-store/src/components/cart/CartSummary.tsx`
- Create: `homechrome-store/src/hooks/useCart.ts`

CSR (client-side rendered). Cart state managed via `useCart` hook backed by server API. Update quantities, remove items, show stock warnings.

**Commit:** `feat: add cart page and cart hooks`

---

### Task 38: Login Page

**Files:**
- Create: `homechrome-store/src/app/login/page.tsx`

Phone number input → OTP input (with countdown timer for resend). Redirects to previous page or home after success.

**Commit:** `feat: add phone OTP login page`

---

### Task 39: Checkout Page

**Files:**
- Create: `homechrome-store/src/app/checkout/page.tsx`
- Create: `homechrome-store/src/app/checkout/confirmation/page.tsx`
- Create: `homechrome-store/src/components/checkout/AddressForm.tsx`
- Create: `homechrome-store/src/components/checkout/ShippingOptions.tsx`
- Create: `homechrome-store/src/components/checkout/OrderSummary.tsx`

SSR (auth required). Steps: select/add address → check serviceability → select shipping → review order → initiate checkout → redirect to PhonePe → return to confirmation page.

Confirmation page polls `GET /checkout/payment-status/<orderID>` until payment resolves.

**Commit:** `feat: add checkout page with address, shipping, and payment flow`

---

### Task 40: Account Pages

**Files:**
- Create: `homechrome-store/src/app/account/page.tsx`
- Create: `homechrome-store/src/app/account/orders/page.tsx`
- Create: `homechrome-store/src/app/account/orders/[id]/page.tsx`
- Create: `homechrome-store/src/app/account/addresses/page.tsx`
- Create: `homechrome-store/src/components/order/OrderCard.tsx`
- Create: `homechrome-store/src/components/order/OrderTimeline.tsx`

SSR (auth required). Profile page, order list with status badges, order detail with timeline and tracking info, address management.

**Commit:** `feat: add account pages — profile, orders, addresses`

---

### Task 41: Public Tracking Page

**Files:**
- Create: `homechrome-store/src/app/track/[orderNumber]/page.tsx`
- Create: `homechrome-store/src/components/order/TrackingStatus.tsx`

SSR (no auth). Shows order status timeline, shipment tracking info, courier details.

**Commit:** `feat: add public order tracking page`

---

## Phase 15: SEO & Final Polish

### Task 42: SEO — Meta Tags, Structured Data, Sitemap

**Files:**
- Modify: `homechrome-store/src/app/p/[slug]/page.tsx` (JSON-LD Product schema)
- Modify: `homechrome-store/src/app/c/[slug]/page.tsx` (JSON-LD BreadcrumbList)
- Create: `homechrome-store/src/app/sitemap.ts` (dynamic sitemap)

**Step 1: Add `generateMetadata` to product and category pages**

```tsx
export async function generateMetadata({ params }): Promise<Metadata> {
    const product = await getProduct(params.slug);
    return {
        title: product.name + ' | Homechrome',
        description: product.description.slice(0, 160),
        openGraph: { images: [product.images[0]?.url] },
    };
}
```

**Step 2: Add JSON-LD structured data**

**Step 3: Add sitemap.ts**

Generates sitemap from all active categories and products.

**Step 4: Add `meta_title` and `meta_description` fields to Product and Category entities in backend**

Modify `internal/domain/entity.go` — add optional SEO fields.

**Step 5: Commit**

```bash
git add homechrome-store/ internal/domain/entity.go
git commit -m "feat: add SEO — meta tags, structured data, sitemap"
```

---

### Task 43: End-to-End Integration Test

**Step 1: Start local backend**

```bash
make setup-local && make run
```

**Step 2: Start storefront**

```bash
cd homechrome-store && npm run dev
```

**Step 3: Manual test flow**

1. Browse categories → products → product detail
2. Add to cart → view cart
3. Login with phone OTP (use test phone in local mode)
4. Checkout → select address → select shipping → initiate payment
5. PhonePe sandbox payment → confirmation page
6. View order in account → check tracking

**Step 4: Fix any issues found**

**Step 5: Final commit**

```bash
git add -A
git commit -m "feat: complete B2C storefront MVP"
```

---

## Summary

| Phase | Tasks | Description |
|-------|-------|-------------|
| 1 | 1 | Infrastructure — CDK GSIs, TTL |
| 2 | 2-4 | Domain entities, repository & service interfaces |
| 3 | 5-8 | Fix all existing scan operations |
| 4 | 9-11 | External gateway clients (MSG91, PhonePe, Shiprocket) |
| 5 | 12-16 | Customer auth (OTP, middleware, handlers) |
| 6 | 17-19 | Cart (repository, service, handlers) |
| 7 | 20 | Store catalog API |
| 8 | 21-23 | Payment (repository, service, webhook) |
| 9 | 24-25 | Checkout (service, handlers) |
| 10 | 26 | Shipping (service, repository) |
| 11 | 27-28 | B2C order & tracking endpoints |
| 12 | 29-30 | Wire DI, route mounting, config |
| 13 | 31-33 | Next.js project setup, layout, components |
| 14 | 34-41 | Next.js pages (home, catalog, cart, checkout, account, tracking) |
| 15 | 42-43 | SEO, end-to-end integration test |

**Total: 43 tasks across 15 phases.**
