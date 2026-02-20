# Artisan Lambda - High Level Design

## 1. Overview

The Artisan Lambda provides comprehensive management of artisans/weavers for the Handloom Admin platform. It handles artisan registration, profile management, product associations, and payment tracking to support the handloom ecosystem.

### Key Features
- Artisan registration and profile management
- Craft specialization tracking
- Product-artisan associations
- Payment history and pending payouts
- Geographic and craft-based filtering
- Award and certification tracking

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         ARTISAN LAMBDA ARCHITECTURE                          │
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
│  │  /artisans              GET     - List artisans                      │    │
│  │  /artisans              POST    - Create artisan                     │    │
│  │  /artisans/{id}         GET     - Get artisan details                │    │
│  │  /artisans/{id}         PUT     - Update artisan                     │    │
│  │  /artisans/{id}         DELETE  - Delete artisan                     │    │
│  │  /artisans/{id}/products GET/POST - Manage products                  │    │
│  │  /artisans/{id}/payments GET    - Get payment history                │    │
│  │  /artisans/search       GET     - Search artisans                    │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            Artisan Lambda                                    │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │                         Handler Layer                               │     │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                │     │
│  │  │   Create     │ │    List      │ │     Get      │                │     │
│  │  │   Handler    │ │   Handler    │ │   Handler    │                │     │
│  │  └──────────────┘ └──────────────┘ └──────────────┘                │     │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐                │     │
│  │  │   Update     │ │   Delete     │ │  Products    │                │     │
│  │  │   Handler    │ │   Handler    │ │   Handler    │                │     │
│  │  └──────────────┘ └──────────────┘ └──────────────┘                │     │
│  │  ┌──────────────┐ ┌──────────────┐                                 │     │
│  │  │  Payments    │ │   Search     │                                 │     │
│  │  │   Handler    │ │   Handler    │                                 │     │
│  │  └──────────────┘ └──────────────┘                                 │     │
│  └────────────────────────────────────────────────────────────────────┘     │
│                                    │                                         │
│                                    ▼                                         │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │                        Service Layer                                │     │
│  │  ┌──────────────────────────────────────────────────────────────┐  │     │
│  │  │                     Artisan Service                           │  │     │
│  │  │  - CreateArtisan()        - GetArtisan()                     │  │     │
│  │  │  - UpdateArtisan()        - DeleteArtisan()                  │  │     │
│  │  │  - ListArtisans()         - SearchArtisans()                 │  │     │
│  │  │  - AssociateProducts()    - GetArtisanProducts()             │  │     │
│  │  │  - GetPaymentHistory()                                        │  │     │
│  │  └──────────────────────────────────────────────────────────────┘  │     │
│  └────────────────────────────────────────────────────────────────────┘     │
│                                    │                                         │
│                                    ▼                                         │
│  ┌────────────────────────────────────────────────────────────────────┐     │
│  │                      Repository Layer                               │     │
│  │  ┌──────────────────────────────────────────────────────────────┐  │     │
│  │  │                   Artisan Repository                          │  │     │
│  │  │  - Create()         - GetByID()       - Update()             │  │     │
│  │  │  - Delete()         - List()          - Search()             │  │     │
│  │  │  - GetByPhone()     - GetProducts()   - GetPayments()        │  │     │
│  │  └──────────────────────────────────────────────────────────────┘  │     │
│  └────────────────────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────────────────┘
                                     │
                     ┌───────────────┼───────────────┐
                     │               │               │
                     ▼               ▼               ▼
              ┌──────────┐    ┌──────────┐    ┌──────────┐
              │ DynamoDB │    │    S3    │    │CloudWatch│
              │  Tables  │    │ (Photos) │    │  (Logs)  │
              └──────────┘    └──────────┘    └──────────┘
```

---

## 3. Component Design

### 3.1 Artisan Handler

```go
type ArtisanHandler struct {
    artisanService domain.ArtisanService
    logger         *logger.Logger
}

// Handler Methods
- CreateArtisan(c *gin.Context)
- GetArtisan(c *gin.Context)
- UpdateArtisan(c *gin.Context)
- DeleteArtisan(c *gin.Context)
- ListArtisans(c *gin.Context)
- SearchArtisans(c *gin.Context)
- GetArtisanProducts(c *gin.Context)
- AssociateProducts(c *gin.Context)
- GetPaymentHistory(c *gin.Context)
```

### 3.2 Artisan Service

```go
type ArtisanService interface {
    // CRUD Operations
    CreateArtisan(ctx context.Context, req *CreateArtisanRequest) (*Artisan, error)
    GetArtisan(ctx context.Context, id string) (*Artisan, error)
    UpdateArtisan(ctx context.Context, id string, req *UpdateArtisanRequest) (*Artisan, error)
    DeleteArtisan(ctx context.Context, id string) error
    ListArtisans(ctx context.Context, filter *ArtisanFilter) (*ArtisanList, error)
    SearchArtisans(ctx context.Context, query string) ([]*Artisan, error)

    // Product Association
    AssociateProducts(ctx context.Context, artisanID string, productIDs []string) error
    RemoveProductAssociation(ctx context.Context, artisanID, productID string) error
    GetArtisanProducts(ctx context.Context, artisanID string) ([]*Product, error)

    // Payment
    GetPaymentHistory(ctx context.Context, artisanID string) (*PaymentHistory, error)
}
```

### 3.3 Artisan Repository

```go
type ArtisanRepository interface {
    Create(ctx context.Context, artisan *Artisan) error
    GetByID(ctx context.Context, id string) (*Artisan, error)
    GetByPhone(ctx context.Context, phone string) (*Artisan, error)
    Update(ctx context.Context, artisan *Artisan) error
    Delete(ctx context.Context, id string) error
    List(ctx context.Context, filter *ArtisanFilter) ([]*Artisan, error)
    Search(ctx context.Context, query string) ([]*Artisan, error)
    GetProductsByArtisan(ctx context.Context, artisanID string) ([]*Product, error)
    GetPayments(ctx context.Context, artisanID string) ([]*Payment, error)
}
```

---

## 4. Data Model

### 4.1 Artisan Entity

```go
type Artisan struct {
    ID               string         `json:"id" dynamodbav:"id"`
    ArtisanCode      string         `json:"artisan_code" dynamodbav:"artisan_code"`
    Name             string         `json:"name" dynamodbav:"name"`
    Phone            string         `json:"phone" dynamodbav:"phone"`
    Email            string         `json:"email,omitempty" dynamodbav:"email,omitempty"`
    DateOfBirth      *time.Time     `json:"date_of_birth,omitempty" dynamodbav:"date_of_birth,omitempty"`
    Gender           string         `json:"gender,omitempty" dynamodbav:"gender,omitempty"`
    PhotoURL         string         `json:"photo_url,omitempty" dynamodbav:"photo_url,omitempty"`
    Address          *Address       `json:"address" dynamodbav:"address"`
    Craft            *CraftDetails  `json:"craft" dynamodbav:"craft"`
    BankDetails      *BankDetails   `json:"bank_details,omitempty" dynamodbav:"bank_details,omitempty"`
    Status           ArtisanStatus  `json:"status" dynamodbav:"status"`
    ProductCount     int            `json:"product_count" dynamodbav:"product_count"`
    TotalEarnings    float64        `json:"total_earnings" dynamodbav:"total_earnings"`
    PendingPayout    float64        `json:"pending_payout" dynamodbav:"pending_payout"`
    CreatedAt        time.Time      `json:"created_at" dynamodbav:"created_at"`
    UpdatedAt        time.Time      `json:"updated_at" dynamodbav:"updated_at"`
    CreatedBy        string         `json:"created_by" dynamodbav:"created_by"`
}
```

### 4.2 Address

```go
type Address struct {
    Line1     string   `json:"line1" dynamodbav:"line1"`
    Line2     string   `json:"line2,omitempty" dynamodbav:"line2,omitempty"`
    City      string   `json:"city" dynamodbav:"city"`
    District  string   `json:"district" dynamodbav:"district"`
    State     string   `json:"state" dynamodbav:"state"`
    PinCode   string   `json:"pin_code" dynamodbav:"pin_code"`
    Latitude  *float64 `json:"latitude,omitempty" dynamodbav:"latitude,omitempty"`
    Longitude *float64 `json:"longitude,omitempty" dynamodbav:"longitude,omitempty"`
}
```

### 4.3 Craft Details

```go
type CraftDetails struct {
    PrimaryCraft     CraftType `json:"primary_craft" dynamodbav:"primary_craft"`
    Specializations  []string  `json:"specializations" dynamodbav:"specializations"`
    ExperienceYears  int       `json:"experience_years" dynamodbav:"experience_years"`
    Techniques       []string  `json:"techniques,omitempty" dynamodbav:"techniques,omitempty"`
    Awards           []Award   `json:"awards,omitempty" dynamodbav:"awards,omitempty"`
}

type CraftType string

const (
    CraftTypeWeaving    CraftType = "WEAVING"
    CraftTypePrinting   CraftType = "PRINTING"
    CraftTypeEmbroidery CraftType = "EMBROIDERY"
    CraftTypeDyeing     CraftType = "DYEING"
    CraftTypeKnitting   CraftType = "KNITTING"
    CraftTypeOther      CraftType = "OTHER"
)

type Award struct {
    Name     string `json:"name" dynamodbav:"name"`
    Year     int    `json:"year" dynamodbav:"year"`
    IssuedBy string `json:"issued_by,omitempty" dynamodbav:"issued_by,omitempty"`
}
```

### 4.4 Bank Details

```go
type BankDetails struct {
    AccountHolder string `json:"account_holder" dynamodbav:"account_holder"`
    AccountNumber string `json:"account_number" dynamodbav:"account_number"`
    IFSCCode      string `json:"ifsc_code" dynamodbav:"ifsc_code"`
    BankName      string `json:"bank_name" dynamodbav:"bank_name"`
    Branch        string `json:"branch,omitempty" dynamodbav:"branch,omitempty"`
}
```

### 4.5 Artisan Status

```go
type ArtisanStatus string

const (
    ArtisanStatusPending   ArtisanStatus = "PENDING"
    ArtisanStatusActive    ArtisanStatus = "ACTIVE"
    ArtisanStatusInactive  ArtisanStatus = "INACTIVE"
    ArtisanStatusOnLeave   ArtisanStatus = "ON_LEAVE"
    ArtisanStatusSuspended ArtisanStatus = "SUSPENDED"
)
```

### 4.6 Artisan Payment

```go
type ArtisanPayment struct {
    ID           string    `json:"id" dynamodbav:"id"`
    ArtisanID    string    `json:"artisan_id" dynamodbav:"artisan_id"`
    Amount       float64   `json:"amount" dynamodbav:"amount"`
    OrderIDs     []string  `json:"order_ids" dynamodbav:"order_ids"`
    Status       string    `json:"status" dynamodbav:"status"` // PENDING, COMPLETED, FAILED
    PaymentDate  time.Time `json:"payment_date" dynamodbav:"payment_date"`
    Reference    string    `json:"reference,omitempty" dynamodbav:"reference,omitempty"`
    CreatedAt    time.Time `json:"created_at" dynamodbav:"created_at"`
}
```

---

## 5. DynamoDB Schema

### 5.1 Artisan Table

```
Table: handloom-artisans

Primary Key:
- PK: ARTISAN#<artisan_id>
- SK: ARTISAN#<artisan_id>

Attributes:
- id: string
- artisan_code: string (ART-YYYY-NNN format)
- name: string
- phone: string (unique)
- email: string
- photo_url: string
- address: map
- craft: map
- bank_details: map (encrypted)
- status: string
- product_count: number
- total_earnings: number
- pending_payout: number
- created_at: string
- updated_at: string
- created_by: string

GSI1: phone-index
- PK: phone
- SK: ARTISAN

GSI2: craft-state-index
- PK: craft.primary_craft
- SK: address.state

GSI3: status-index
- PK: status
- SK: created_at
```

### 5.2 Artisan-Product Association Table

```
Table: handloom-artisan-products

Primary Key:
- PK: ARTISAN#<artisan_id>
- SK: PRODUCT#<product_id>

GSI1: product-artisan-index
- PK: PRODUCT#<product_id>
- SK: ARTISAN#<artisan_id>
```

### 5.3 Artisan Payment Table

```
Table: handloom-artisan-payments

Primary Key:
- PK: ARTISAN#<artisan_id>
- SK: PAYMENT#<payment_id>

Attributes:
- id: string
- artisan_id: string
- amount: number
- order_ids: list
- status: string
- payment_date: string
- reference: string
- created_at: string

GSI1: status-date-index
- PK: status
- SK: payment_date
```

### 5.4 Access Patterns

| Access Pattern | Key Condition | Index |
|----------------|---------------|-------|
| Get artisan by ID | PK = ARTISAN#{id} | Main |
| Get artisan by phone | PK = {phone} | GSI1 |
| List by craft & state | PK = {craft}, SK begins_with {state} | GSI2 |
| List by status | PK = {status} | GSI3 |
| Get artisan products | PK = ARTISAN#{id} | Association Table |
| Get product artisan | PK = PRODUCT#{id} | Association GSI1 |

---

## 6. API Endpoints

### 6.1 Create Artisan

```
POST /artisans

Request:
{
    "name": "Ramesh Kumar",
    "phone": "+919876543210",
    "email": "ramesh@email.com",
    "date_of_birth": "1975-05-15",
    "gender": "male",
    "address": {
        "line1": "Village Kothara",
        "line2": "Near Temple",
        "city": "Kothara",
        "district": "Kutch",
        "state": "Gujarat",
        "pin_code": "370430",
        "latitude": 23.4567,
        "longitude": 69.1234
    },
    "craft": {
        "primary_craft": "WEAVING",
        "specializations": ["silk", "cotton"],
        "experience_years": 25,
        "techniques": ["patola", "double ikat"],
        "awards": [
            {"name": "National Award", "year": 2015, "issued_by": "Ministry of Textiles"}
        ]
    },
    "bank_details": {
        "account_holder": "Ramesh Kumar",
        "account_number": "1234567890",
        "ifsc_code": "SBIN0001234",
        "bank_name": "State Bank of India",
        "branch": "Bhuj Main"
    }
}

Response:
{
    "success": true,
    "data": {
        "id": "art_123",
        "artisan_code": "ART-2024-001",
        "name": "Ramesh Kumar",
        "status": "ACTIVE",
        ...
    }
}
```

### 6.2 List Artisans

```
GET /artisans?craft=WEAVING&state=Gujarat&status=ACTIVE&page=1&limit=20

Response:
{
    "success": true,
    "data": {
        "artisans": [
            {
                "id": "art_123",
                "artisan_code": "ART-2024-001",
                "name": "Ramesh Kumar",
                "craft": {
                    "primary_craft": "WEAVING",
                    "experience_years": 25
                },
                "address": {
                    "district": "Kutch",
                    "state": "Gujarat"
                },
                "product_count": 45,
                "status": "ACTIVE"
            }
        ],
        "total": 150,
        "page": 1,
        "limit": 20
    }
}
```

### 6.3 Get Artisan Products

```
GET /artisans/{id}/products

Response:
{
    "success": true,
    "data": {
        "artisan_id": "art_123",
        "products": [
            {
                "id": "prod_456",
                "name": "Patola Silk Saree",
                "sku": "SKU-001",
                "price": 15000,
                "stock": 5,
                "status": "ACTIVE"
            }
        ],
        "total": 45
    }
}
```

### 6.4 Get Payment History

```
GET /artisans/{id}/payments

Response:
{
    "success": true,
    "data": {
        "artisan_id": "art_123",
        "total_earnings": 456000,
        "pending_payout": 45000,
        "payments": [
            {
                "id": "pay_789",
                "amount": 32000,
                "order_ids": ["ord_1", "ord_2", "ord_3"],
                "status": "COMPLETED",
                "payment_date": "2024-01-15T10:00:00Z",
                "reference": "UTR123456"
            }
        ]
    }
}
```

---

## 7. Error Handling

### 7.1 Error Types

| Error Code | Description | HTTP Status |
|------------|-------------|-------------|
| ARTISAN_NOT_FOUND | Artisan does not exist | 404 |
| PHONE_ALREADY_EXISTS | Phone number already registered | 409 |
| INVALID_CRAFT_TYPE | Unknown craft type | 400 |
| PENDING_PAYMENTS | Cannot delete with pending payments | 400 |
| INVALID_BANK_DETAILS | Bank details validation failed | 400 |

### 7.2 Error Response Format

```json
{
    "success": false,
    "error": {
        "code": "PHONE_ALREADY_EXISTS",
        "message": "An artisan with this phone number already exists"
    }
}
```

---

## 8. Security

### 8.1 Access Control

| Role | Create | Read | Update | Delete | Payments |
|------|--------|------|--------|--------|----------|
| Admin | Yes | All | Yes | Yes | View/Process |
| Manager | Yes | All | Yes | No | View |
| Staff | No | All | Limited | No | View |

### 8.2 Data Privacy

- Bank details encrypted at rest
- Phone numbers masked in list views
- PII access logged for audit
- Payment details restricted access

---

## 9. Performance Optimization

### 9.1 Caching Strategy

- Cache active artisan list (5-minute TTL)
- Cache artisan details (1-minute TTL)
- Invalidate on updates

### 9.2 Query Optimization

- GSI for craft + state filtering
- Sparse index for active artisans
- Batch operations for product associations

---

## 10. Monitoring

### 10.1 Key Metrics

| Metric | Description | Threshold |
|--------|-------------|-----------|
| Registration Rate | New artisans per day | Monitor trends |
| Active Ratio | Active vs total artisans | > 80% |
| Payout Processing | Time to process payouts | < 24h |
| Product Association | Products per artisan | Track average |

### 10.2 Alerts

- Failed payment processing
- Unusual registration patterns
- Data validation failures
- High inactive artisan count

---

## 11. Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              DEPENDENCIES                                    │
└─────────────────────────────────────────────────────────────────────────────┘

                          Artisan Lambda
                               │
           ┌───────────────────┼───────────────────┐
           │                   │                   │
           ▼                   ▼                   ▼
    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
    │  DynamoDB   │    │     S3      │    │Product Svc  │
    │  (Storage)  │    │  (Photos)   │    │ (Associate) │
    └─────────────┘    └─────────────┘    └─────────────┘
```

### Internal Dependencies
- Product Service: Product-artisan associations
- Order Service: Payment calculation
- Asset Service: Photo upload

### External Dependencies
- AWS DynamoDB: Artisan data storage
- AWS S3: Photo storage
- AWS CloudWatch: Logging and monitoring

