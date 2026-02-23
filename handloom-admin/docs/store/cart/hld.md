# Store Cart - High-Level Design (HLD)

## 1. Overview

The Store Cart service manages shopping cart operations for the B2C customer storefront. It allows authenticated customers to add, update, remove, and clear items in their cart. The cart supports both standard products and custom-sized products (via price quotes). Guest carts can be merged into customer carts upon login. Cart data is stored in the `handloom-orders` DynamoDB table using a single-table design pattern.

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                      CART SYSTEM                                             │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

                                    ┌───────────────────┐
                                    │  Next.js Frontend │
                                    │  (B2C Storefront) │
                                    └─────────┬─────────┘
                                              │
                                              │ HTTPS (Customer JWT Cookie)
                                              ▼
                                    ┌───────────────────┐
                                    │   API Gateway /   │
                                    │   Lambda / Local  │
                                    └─────────┬─────────┘
                                              │
                                              │ CustomerAuth Middleware
                                              ▼
                                    ┌───────────────────┐
                                    │   Cart Handler    │
                                    │  (store/cart)     │
                                    │                   │
                                    │  GET /            │
                                    │  POST /items      │
                                    │  PATCH /items/{id}│
                                    │  DELETE /items/{id}│
                                    │  DELETE /          │
                                    │  POST /merge      │
                                    └─────────┬─────────┘
                                              │
                                              ▼
                                    ┌───────────────────┐
                                    │   Cart Service    │
                                    │  (Business Logic) │
                                    └─────────┬─────────┘
                                              │
                    ┌─────────────────────────┼─────────────────────────┐
                    │                         │                         │
                    ▼                         ▼                         ▼
         ┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
         │  Cart Repository│       │ Product Service  │       │ Inventory       │
         │  (DynamoDB)     │       │ (Catalog)        │       │ Service         │
         │                 │       │                  │       │                 │
         │ handloom-orders │       │ handloom-core    │       │ handloom-core   │
         └─────────────────┘       └─────────────────┘       └─────────────────┘
```

---

## 3. Component Design

### 3.1 Service Layer Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CART SERVICE LAYER                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         Handler Layer                                │   │
│  │                                                                      │   │
│  │  CartHandler (internal/handler/store/cart_handler.go)                │   │
│  │  Routes() chi.Router                                                │   │
│  │                                                                      │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │   │
│  │  │  GetCart     │  │  AddItem    │  │ UpdateQty   │                 │   │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘                 │   │
│  │         │                │                │                         │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │   │
│  │  │ RemoveItem  │  │  ClearCart  │  │ MergeCart   │                 │   │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘                 │   │
│  └─────────┼────────────────┼────────────────┼──────────────────────────┘   │
│            │                │                │                              │
│            └────────────────┼────────────────┘                              │
│                             ▼                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         Cart Service                                 │   │
│  │  implements domain.CartService                                      │   │
│  │                                                                      │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │  - GetCart(customerID)            → CartWithItems              │ │   │
│  │  │  - AddItem(customerID, req)       → CartWithItems              │ │   │
│  │  │  - UpdateItemQuantity(cID, pID, q)→ CartWithItems              │ │   │
│  │  │  - RemoveItem(customerID, pID)    → CartWithItems              │ │   │
│  │  │  - ClearCart(customerID)          → error                      │ │   │
│  │  │  - MergeGuestCart(cID, items)     → CartWithItems              │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                             │                                               │
│         ┌───────────────────┼───────────────────┐                          │
│         │                   │                   │                          │
│         ▼                   ▼                   ▼                          │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐              │
│  │ CartRepository   │ │ ProductRepo     │ │ InventoryRepo   │              │
│  │ (DynamoDB)       │ │ (Catalog lookup)│ │ (Stock checks)  │              │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘              │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Key Design Decisions

- **Cart auto-creation**: Cart is lazily created on the first `AddItem` call. `GetCart` returns an empty cart if none exists.
- **Denormalized subtotal**: Cart header stores `subtotal` and `item_count` to avoid recalculating on every read. Recalculated on every write operation.
- **Product snapshot**: Product name, SKU, image, and unit price are snapshotted into the cart item at add time. Prices are locked when added.
- **Custom size via quote**: Custom-dimension products reference a `PriceQuote` by ID. The quote's `calculated_price` becomes the `unit_price`.

---

## 4. Data Model

### 4.1 DynamoDB Table Design

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                       TABLE: handloom-orders                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  CART HEADER                                                                 │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: CART#<customer_id>                                            │      │
│  │ SK: METADATA                                                      │      │
│  │ entity_type: CART                                                 │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - id              (UUID)                                        │      │
│  │   - customer_id     (customer UUID)                               │      │
│  │   - session_id      (session identifier)                          │      │
│  │   - item_count      (denormalized count)                          │      │
│  │   - subtotal        (total in paise, denormalized)                │      │
│  │   - currency        ("INR")                                       │      │
│  │   - updated_at      (ISO 8601)                                    │      │
│  │   - ttl             (Unix timestamp, 30 days)                     │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  CART ITEM                                                                   │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: CART#<customer_id>                                            │      │
│  │ SK: ITEM#<product_id>                                             │      │
│  │ entity_type: CART_ITEM                                            │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - product_id      (product UUID)                                │      │
│  │   - product_name    (snapshot from catalog)                       │      │
│  │   - product_sku     (snapshot from catalog)                       │      │
│  │   - product_image   (snapshot from catalog)                       │      │
│  │   - quantity         (item quantity)                               │      │
│  │   - unit_price       (price per unit in paise)                    │      │
│  │   - total_price      (quantity * unit_price in paise)             │      │
│  │   - is_custom_size   (boolean)                                    │      │
│  │   - dimensions       (optional: {length, width, height, unit})    │      │
│  │   - quote_id         (optional: price quote reference)            │      │
│  │   - attributes       (optional: product attribute map)            │      │
│  │   - added_at         (ISO 8601)                                   │      │
│  │   - ttl              (Unix timestamp, 30 days)                    │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  KEY PATTERNS                                                                │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ Authenticated user:  PK = CART#<customer_id>                      │      │
│  │ Guest user:          PK = CART#<session_id>                       │      │
│  │ Cart header:         SK = METADATA                                │      │
│  │ Cart item:           SK = ITEM#<product_id>                       │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  ACCESS PATTERNS                                                             │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ Get cart + all items:  PK = CART#<customer_id> (query all SKs)    │      │
│  │ Get single item:       PK = CART#<cid>, SK = ITEM#<product_id>    │      │
│  │ Delete all items:      PK = CART#<cid>, BatchDelete all SKs       │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  TTL CONFIGURATION                                                           │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ TTL Attribute: ttl                                                │      │
│  │ Default Expiry: 30 days of inactivity                             │      │
│  │ Purpose: Auto-clean abandoned carts                               │      │
│  │ Updated: TTL refreshed on every cart write operation               │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 4.2 Cart Recalculation

On every write operation (add, update, remove), the service:
1. Queries all `ITEM#*` records for the cart PK
2. Sums `total_price` across all items to compute `subtotal`
3. Counts items to compute `item_count`
4. Writes the updated Cart header (METADATA record)

---

## 5. Security

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           SECURITY MODEL                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Authentication:                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ - Customer JWT via HttpOnly cookie (CUSTOMER_JWT_SECRET)            │   │
│  │ - CustomerAuth middleware extracts customer_id from token            │   │
│  │ - Cart PK derived from authenticated customer_id (not user input)   │   │
│  │ - Token refresh at /api/v1/store/auth/refresh                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Authorization:                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ - Customers can only access their own cart (PK = CART#<own_id>)     │   │
│  │ - No cross-customer cart access possible by design                  │   │
│  │ - Product IDs validated against catalog before adding               │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Input Validation:                                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ - middleware.ValidateJSONTyped[T] for request body validation       │   │
│  │ - Quantity: gt=0 for add, gte=0 for update (0 = remove)            │   │
│  │ - Product ID: validated as existing active product                  │   │
│  │ - Quote ID: validated as existing and not expired                   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Error Handling

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              ERROR CODES                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Cart Errors:                                                                │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ VALIDATION_ERROR     │ 400 │ Invalid request body or fields         │   │
│  │ UNAUTHORIZED         │ 401 │ Missing or invalid customer JWT        │   │
│  │ PRODUCT_NOT_FOUND    │ 404 │ Product does not exist or is inactive  │   │
│  │ ITEM_NOT_IN_CART     │ 404 │ Product not found in customer's cart   │   │
│  │ INSUFFICIENT_STOCK   │ 409 │ Quantity exceeds available inventory   │   │
│  │ QUOTE_EXPIRED        │ 400 │ Price quote has expired                │   │
│  │ QUOTE_NOT_FOUND      │ 404 │ Price quote does not exist             │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Error Response Format:                                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ {                                                                   │   │
│  │   "success": false,                                                 │   │
│  │   "error": {                                                        │   │
│  │     "code": "INSUFFICIENT_STOCK",                                   │   │
│  │     "message": "Only 2 units available for this product"            │   │
│  │   }                                                                 │   │
│  │ }                                                                   │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Integration Points

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          INTEGRATION POINTS                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                     Product Service (Catalog)                       │   │
│  │                                                                      │   │
│  │  CartService ──▶ ProductRepository                                  │   │
│  │    - GetByID(productID) → Product (validate exists + active)        │   │
│  │    - Snapshot name, SKU, image, selling_price into CartItem         │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                     Inventory Service                               │   │
│  │                                                                      │   │
│  │  CartService ──▶ InventoryRepository                                │   │
│  │    - GetByProductID(productID) → Inventory                          │   │
│  │    - Check available_qty >= requested quantity                       │   │
│  │    - Validated on add and update quantity                            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                     Price Quote Service                             │   │
│  │                                                                      │   │
│  │  CartService ──▶ PriceQuoteRepository                               │   │
│  │    - GetByID(quoteID) → PriceQuote                                  │   │
│  │    - Validate quote not expired (valid_until > now)                  │   │
│  │    - Use calculated_price as unit_price for custom-size items       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                     Checkout Service (Downstream)                    │   │
│  │                                                                      │   │
│  │  CheckoutService ──▶ CartService                                    │   │
│  │    - GetCart(customerID) → read cart for order placement             │   │
│  │    - ClearCart(customerID) → clear after successful order            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              DEPENDENCIES                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  AWS Services:                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ - DynamoDB (handloom-orders table) — Cart and CartItem storage      │   │
│  │ - CloudWatch — Logging and metrics                                  │   │
│  │ - API Gateway — HTTP endpoint (Lambda mode)                         │   │
│  │ - Lambda — Compute (production)                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Internal Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ - CustomerAuth Middleware — JWT authentication                      │   │
│  │ - Product Repository (handloom-core) — Product validation           │   │
│  │ - Inventory Repository (handloom-core) — Stock availability         │   │
│  │ - PriceQuote Repository (handloom-orders) — Custom pricing          │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Go Packages:                                                                │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ - github.com/go-chi/chi/v5 — HTTP router                           │   │
│  │ - github.com/aws/aws-sdk-go-v2/service/dynamodb — DynamoDB client  │   │
│  │ - github.com/google/wire — Compile-time dependency injection        │   │
│  │ - github.com/go-playground/validator/v10 — Request validation       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```
