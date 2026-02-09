# handloom-core Table

The core table contains all primary business entities for the handloom admin system.

## Table Configuration

```
Table Name: handloom-core
Partition Key: PK (String)
Sort Key: SK (String)
Billing Mode: PAY_PER_REQUEST
```

### Global Secondary Indexes

| Index | Partition Key | Sort Key | Projection |
|-------|--------------|----------|------------|
| GSI1 | GSI1PK | GSI1SK | ALL |
| GSI2 | GSI2PK | GSI2SK | ALL |

---

## Entities

### 1. User

Admin portal users with role-based access control.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `USER#<id>` | `USER#550e8400-e29b-41d4-a716-446655440000` |
| SK | `METADATA` | `METADATA` |
| GSI1PK | `USER_EMAIL` | `USER_EMAIL` |
| GSI1SK | `<email>` | `admin@handloom.com` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| Email | String | Yes | Unique email address |
| PasswordHash | String | Yes | Bcrypt hashed password |
| FirstName | String | Yes | User's first name |
| LastName | String | Yes | User's last name |
| Phone | String | No | Phone number |
| Role | String | Yes | `ADMIN` or `OPERATOR` |
| Permissions | List[String] | No | Granular permissions |
| Status | String | Yes | `ACTIVE`, `INACTIVE`, `PENDING` |
| LastLoginAt | String | No | ISO 8601 timestamp |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| UpdatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID who created |
| UpdatedBy | String | Yes | User ID who last updated |

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get user by ID | PK = `USER#<id>`, SK = `METADATA` |
| Get user by email | GSI1: GSI1PK = `USER_EMAIL`, GSI1SK = `<email>` |
| List all users | GSI1: GSI1PK = `USER_EMAIL` (cursor pagination) |

---

### 2. Category

Hierarchical product categories with custom attributes.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `CATEGORY#<id>` | `CATEGORY#cat-001` |
| SK | `METADATA` | `METADATA` |
| GSI1PK | `CATEGORY#<parent_id>` or `CATEGORY#ROOT` | `CATEGORY#ROOT` |
| GSI1SK | `CATEGORY#<id>` | `CATEGORY#cat-001` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| Name | String | Yes | Category name |
| Slug | String | Yes | URL-friendly identifier |
| Description | String | No | Category description |
| ImageURL | String | No | Category image |
| Status | String | Yes | `ACTIVE`, `INACTIVE` |
| ParentID | String | No | Parent category ID (null for root) |
| Level | Number | Yes | Hierarchy depth (0 = root) |
| Path | String | Yes | Full path (e.g., `/sarees/silk`) |
| AncestorIDs | List[String] | No | All ancestor category IDs |
| OwnAttributes | List[Object] | No | Category-specific attributes |
| DimensionConfig | Object | No | Custom dimension constraints |
| DefaultPricingRuleID | String | No | Default pricing rule |
| AllowCustomDimensions | Boolean | No | Enable custom sizing |
| ProductCount | Number | No | Denormalized count |
| DesignCount | Number | No | Denormalized count |
| SortOrder | Number | No | Display order |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| UpdatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID |
| UpdatedBy | String | Yes | User ID |

#### OwnAttributes Structure

```json
{
  "Name": "material",
  "Type": "SELECT",
  "Label": "Material",
  "Required": true,
  "Options": ["Silk", "Cotton", "Linen"]
}
```

**AttributeType Enum**: `SELECT`, `MULTI_SELECT`, `TEXT`, `NUMBER`, `BOOLEAN`, `DIMENSION`, `DIMENSION_RANGE`

#### DimensionConfig Structure

```json
{
  "LengthMin": 100,
  "LengthMax": 600,
  "WidthMin": 44,
  "WidthMax": 48,
  "Unit": "CM"
}
```

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get category by ID | PK = `CATEGORY#<id>`, SK = `METADATA` |
| List all categories | GSI1: GSI1PK = `CATEGORY#ALL` (cursor pagination) |

---

### 3. Design

Design templates that can be applied to multiple products.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `DESIGN#<id>` | `DESIGN#des-001` |
| SK | `METADATA` | `METADATA` |
| GSI1PK | `CATEGORY#<category_id>` | `CATEGORY#cat-001` |
| GSI1SK | `DESIGN#<id>` | `DESIGN#des-001` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| Name | String | Yes | Design name |
| Slug | String | Yes | URL-friendly identifier |
| CategoryID | String | Yes | Parent category |
| Description | String | No | Design description |
| Status | String | Yes | `ACTIVE`, `INACTIVE` |
| Images | List[Object] | No | Design images |
| Attributes | List[Object] | No | Design-specific attributes |
| ProductCount | Number | No | Denormalized count |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| UpdatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID |
| UpdatedBy | String | Yes | User ID |

#### Images Structure

```json
{
  "URL": "https://cdn.example.com/image.jpg",
  "AltText": "Red silk saree design",
  "IsPrimary": true,
  "SortOrder": 1
}
```

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get design by ID | PK = `DESIGN#<id>`, SK = `METADATA` |
| Get designs by category | GSI1: GSI1PK = `CATEGORY#<category_id>` |

---

### 4. Product

Individual products with inventory and pricing.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `PRODUCT#<id>` | `PRODUCT#prod-001` |
| SK | `METADATA` | `METADATA` |
| GSI1PK | `CATEGORY#<category_id>` | `CATEGORY#cat-001` |
| GSI1SK | `PRODUCT#<id>` | `PRODUCT#prod-001` |
| GSI2PK | `PRODUCT#ALL` | `PRODUCT#ALL` |
| GSI2SK | `PRODUCT#<id>` | `PRODUCT#prod-001` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| Name | String | Yes | Product name |
| Slug | String | Yes | URL-friendly identifier |
| SKU | String | Yes | Stock keeping unit (unique) |
| Description | String | No | Product description |
| Status | String | Yes | `ACTIVE`, `INACTIVE`, `DRAFT` |
| DesignID | String | No | Associated design |
| CategoryID | String | Yes | Product category |
| ArtisanID | String | No | Associated artisan |
| BasePrice | Number | Yes | Base price in paise |
| SellingPrice | Number | Yes | Selling price in paise |
| CostPrice | Number | No | Cost price in paise |
| Currency | String | Yes | `INR` |
| Length | Number | No | Length dimension |
| Width | Number | No | Width dimension |
| Height | Number | No | Height dimension |
| Unit | String | No | Dimension unit (CM, INCH) |
| Weight | Number | No | Weight in grams |
| AllowCustomDimensions | Boolean | No | Enable custom sizing |
| PricingRuleID | String | No | Applied pricing rule |
| Attributes | Map | No | Category-specific attributes |
| Material | String | No | Primary material |
| Color | String | No | Primary color |
| WeaveType | String | No | Weave technique |
| Origin | String | No | Place of origin |
| CraftType | String | No | Craft type |
| Images | List[Object] | No | Product images |
| Tags | List[String] | No | Search tags |
| Quantity | Number | Yes | Current stock (denormalized) |
| ReservedQty | Number | Yes | Reserved quantity |
| AvailableQty | Number | Yes | Available = Quantity - Reserved |
| LowStockThreshold | Number | No | Alert threshold |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| UpdatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID |
| UpdatedBy | String | Yes | User ID |

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get product by ID | PK = `PRODUCT#<id>`, SK = `METADATA` |
| Get products by category | GSI1: GSI1PK = `CATEGORY#<category_id>`, GSI1SK begins_with `PRODUCT#` (cursor pagination) |
| List all products | GSI2: GSI2PK = `PRODUCT#ALL` (cursor pagination) |
| Get product by SKU | PK = `SKU#<sku>`, SK = `METADATA` → GetItem returns `product_id` → GetByID |

---

### 5. Product SKU Index

Uniqueness enforcement item for product SKUs. Created atomically with the product via TransactWriteItems with `attribute_not_exists(PK)` to guarantee SKU uniqueness without a Scan.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `SKU#<sku>` | `SKU#prod_a1b2c3d4` |
| SK | `METADATA` | `METADATA` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| product_id | String | Yes | The product ID this SKU belongs to |
| entity_type | String | Yes | Always `PRODUCT_SKU` |

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get product by SKU | PK = `SKU#<sku>`, SK = `METADATA` → returns `product_id` |

---

### 6. Inventory

Detailed inventory tracking with transaction history.

#### Key Structure (Main Record)

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `INVENTORY#<product_id>` | `INVENTORY#prod-001` |
| SK | `METADATA` | `METADATA` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| ProductID | String | Yes | Associated product |
| ProductSKU | String | Yes | Product SKU (denormalized) |
| ProductName | String | Yes | Product name (denormalized) |
| Quantity | Number | Yes | Total quantity |
| ReservedQty | Number | Yes | Reserved for orders |
| AvailableQty | Number | Yes | Available for sale |
| LowStockThreshold | Number | No | Alert threshold |
| ReorderPoint | Number | No | Reorder trigger level |
| LastRestockAt | String | No | Last restock timestamp |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| UpdatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID |
| UpdatedBy | String | Yes | User ID |

---

### 7. Inventory Transaction

Time-ordered transaction history for inventory changes.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `INVENTORY#<product_id>` | `INVENTORY#prod-001` |
| SK | `TXN#<timestamp>#<id>` | `TXN#2024-01-15T10:30:00Z#txn-001` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| ProductID | String | Yes | Associated product |
| Type | String | Yes | `ADD`, `REMOVE`, `RESERVE`, `RELEASE`, `ADJUST` |
| Quantity | Number | Yes | Transaction quantity |
| PreviousQty | Number | Yes | Quantity before transaction |
| NewQty | Number | Yes | Quantity after transaction |
| Reason | String | No | Transaction reason |
| ReferenceType | String | No | Related entity type (ORDER, QUOTE) |
| ReferenceID | String | No | Related entity ID |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID |

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get all transactions | PK = `INVENTORY#<product_id>`, SK begins_with `TXN#` |
| Get transactions in date range | PK = `INVENTORY#<product_id>`, SK between `TXN#<start>` and `TXN#<end>` |

---

### 8. Pricing Rule

Dynamic pricing rules for products and categories.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `PRICING_RULE#<id>` | `PRICING_RULE#rule-001` |
| SK | `METADATA` | `METADATA` |
| GSI1PK | `SCOPE#<scope_type>` | `SCOPE#CATEGORY` |
| GSI1SK | `<scope_id>` | `cat-001` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| Name | String | Yes | Rule name |
| Description | String | No | Rule description |
| Priority | Number | Yes | Rule priority (higher = first) |
| IsActive | Boolean | Yes | Rule status |
| ScopeType | String | Yes | `GLOBAL`, `CATEGORY`, `SUBCATEGORY`, `PRODUCT`, `MATERIAL` |
| ScopeID | String | No | Target entity ID |
| CategoryID | String | No | Category for CATEGORY scope |
| MaterialName | String | No | Material for MATERIAL scope |
| PricingType | String | Yes | `AREA_BASED`, `LENGTH_BASED`, `FIXED`, `TIERED`, `FORMULA` |
| BasePrice | Number | No | Base price in paise |
| PricePerUnit | Number | No | Price per unit in paise |
| Unit | String | No | `SQ_INCH`, `SQ_FOOT`, `SQ_CM`, `INCH`, `CM`, `METER` |
| MaterialMultipliers | Map | No | Material-based multipliers |
| AttributeSurcharges | List[Object] | No | Attribute-based surcharges |
| Tiers | List[Object] | No | Tiered pricing thresholds |
| Formula | String | No | Custom pricing formula |
| MinArea | Number | No | Minimum area constraint |
| MaxArea | Number | No | Maximum area constraint |
| MinOrderValue | Number | No | Minimum order value |
| ValidFrom | String | No | Rule start date |
| ValidUntil | String | No | Rule end date |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| UpdatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID |
| UpdatedBy | String | Yes | User ID |

#### MaterialMultipliers Example

```json
{
  "Silk": 1.5,
  "Cotton": 1.0,
  "Linen": 1.2
}
```

#### AttributeSurcharges Example

```json
{
  "Name": "zari_work",
  "Value": "heavy",
  "Type": "PERCENTAGE",
  "Amount": 20
}
```

#### Tiers Example

```json
{
  "MinValue": 0,
  "MaxValue": 100,
  "PricePerUnit": 500
}
```

---

### 9. Coupon

Discount coupons with usage tracking.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `COUPON#<id>` | `COUPON#coup-001` |
| SK | `METADATA` | `METADATA` |
| GSI1PK | `COUPON_CODE` | `COUPON_CODE` |
| GSI1SK | `<code>` | `SUMMER2024` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| Code | String | Yes | Unique coupon code |
| Name | String | Yes | Coupon name |
| Description | String | No | Coupon description |
| Status | String | Yes | `ACTIVE`, `INACTIVE`, `EXPIRED` |
| Type | String | Yes | `PERCENTAGE`, `FIXED` |
| Value | Number | Yes | Discount value (% * 100 or paise) |
| MinOrderValue | Number | No | Minimum order to apply |
| MaxDiscount | Number | No | Maximum discount amount (paise) |
| UsageLimit | Number | No | Total usage limit (0 = unlimited) |
| UsagePerUser | Number | No | Per-user limit (0 = unlimited) |
| UsageCount | Number | Yes | Current usage count |
| ApplicableCategories | List[String] | No | Allowed categories |
| ApplicableProducts | List[String] | No | Allowed products |
| ExcludedCategories | List[String] | No | Excluded categories |
| ExcludedProducts | List[String] | No | Excluded products |
| ValidFrom | String | Yes | Start date |
| ValidUntil | String | Yes | End date |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| UpdatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID |
| UpdatedBy | String | Yes | User ID |

---

### 10. Coupon Usage

Track individual coupon redemptions.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `COUPON#<coupon_id>` | `COUPON#coup-001` |
| SK | `USAGE#<timestamp>#<order_id>` | `USAGE#2024-01-15T10:30:00Z#ord-001` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| CouponID | String | Yes | Coupon ID |
| CouponCode | String | Yes | Coupon code |
| OrderID | String | Yes | Order ID |
| CustomerID | String | Yes | Customer ID |
| Discount | Number | Yes | Discount amount in paise |
| CreatedAt | String | Yes | ISO 8601 timestamp |

---

### 11. Artisan

Craftspeople who create products.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `ARTISAN#<id>` | `ARTISAN#art-001` |
| SK | `METADATA` | `METADATA` |
| GSI1PK | `ARTISAN_STATUS` | `ARTISAN_STATUS` |
| GSI1SK | `<status>#<id>` | `ACTIVE#art-001` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| Name | String | Yes | Artisan name |
| Email | String | No | Email address |
| Phone | String | Yes | Phone number |
| ProfileImage | String | No | Profile image URL |
| Bio | String | No | Biography |
| Location | String | No | City/Town |
| Address | String | No | Full address |
| CraftTypes | List[String] | Yes | Craft specializations |
| Specializations | List[String] | No | Specific techniques |
| Experience | Number | No | Years of experience |
| BankDetails | Object | No | Payment information |
| CommissionRate | Number | No | Commission percentage |
| Status | String | Yes | `ACTIVE`, `INACTIVE`, `PENDING` |
| ProductCount | Number | No | Total products (denormalized) |
| TotalSales | Number | No | Total sales (denormalized) |
| TotalEarnings | Number | No | Total earnings in paise |
| PendingPayout | Number | No | Pending payout in paise |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| UpdatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID |
| UpdatedBy | String | Yes | User ID |

#### BankDetails Structure

```json
{
  "AccountHolderName": "Artisan Name",
  "AccountNumber": "1234567890",
  "IFSCCode": "SBIN0001234",
  "BankName": "State Bank of India",
  "BranchName": "Main Branch"
}
```

---

### 12. Artisan Payout

Payment records for artisans.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `ARTISAN#<artisan_id>` | `ARTISAN#art-001` |
| SK | `PAYOUT#<timestamp>` | `PAYOUT#2024-01-15T10:30:00Z` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| ArtisanID | String | Yes | Artisan ID |
| Amount | Number | Yes | Payout amount in paise |
| Status | String | Yes | `PENDING`, `PROCESSING`, `COMPLETED`, `FAILED` |
| PaymentMethod | String | No | Payment method used |
| TransactionID | String | No | External transaction ID |
| PeriodStart | String | Yes | Payout period start |
| PeriodEnd | String | Yes | Payout period end |
| OrderIDs | List[String] | Yes | Included order IDs |
| OrderCount | Number | Yes | Number of orders |
| ProcessedAt | String | No | Processing timestamp |
| FailureReason | String | No | Failure reason if failed |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID |

---

### 13. Asset

Media and document storage metadata.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `ASSET#<id>` | `ASSET#asset-001` |
| SK | `METADATA` | `METADATA` |
| GSI1PK | `ASSET_TYPE#<type>` | `ASSET_TYPE#IMAGE` |
| GSI1SK | `ASSET#<id>` | `ASSET#asset-001` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| FileName | String | Yes | Stored file name |
| OriginalName | String | Yes | Original upload name |
| ContentType | String | Yes | MIME type |
| Size | Number | Yes | File size in bytes |
| Type | String | Yes | `IMAGE`, `DOCUMENT`, `VIDEO` |
| URL | String | Yes | Direct URL |
| ThumbnailURL | String | No | Thumbnail URL |
| CDNUrl | String | No | CDN URL |
| Bucket | String | Yes | S3 bucket name |
| Key | String | Yes | S3 object key |
| Width | Number | No | Image width (pixels) |
| Height | Number | No | Image height (pixels) |
| UsedBy | List[Object] | No | Entities using this asset |
| AltText | String | No | Alt text for accessibility |
| Tags | List[String] | No | Searchable tags |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| UpdatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID |
| UpdatedBy | String | Yes | User ID |

#### UsedBy Structure

```json
{
  "EntityType": "PRODUCT",
  "EntityID": "prod-001",
  "Field": "Images"
}
```

---

### 14. Bulk Job

Batch operation tracking.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `BULK_JOB#<id>` | `BULK_JOB#job-001` |
| SK | `METADATA` | `METADATA` |
| GSI1PK | `USER#<created_by>` | `USER#user-001` |
| GSI1SK | `BULK_JOB#<id>` | `BULK_JOB#job-001` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| Type | String | Yes | `PRODUCT_IMPORT`, `PRODUCT_EXPORT`, `INVENTORY_UPDATE`, `PRICE_UPDATE`, `ORDER_EXPORT` |
| Status | String | Yes | `PENDING`, `PROCESSING`, `COMPLETED`, `FAILED`, `PARTIAL_SUCCESS` |
| FileName | String | No | Input file name |
| FileURL | String | No | Input file URL |
| FileSize | Number | No | Input file size |
| TotalRows | Number | No | Total rows to process |
| ProcessedRows | Number | No | Rows processed |
| SuccessCount | Number | No | Successful operations |
| ErrorCount | Number | No | Failed operations |
| Errors | List[Object] | No | Error details |
| ResultFileURL | String | No | Output file URL |
| StartedAt | String | No | Processing start |
| CompletedAt | String | No | Processing end |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| UpdatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID |
| UpdatedBy | String | Yes | User ID |

#### Errors Structure

```json
{
  "Row": 15,
  "Field": "Price",
  "Message": "Invalid price format",
  "Value": "abc"
}
```

---

### 15. Notification

System notifications and alerts.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `NOTIFICATION#<id>` | `NOTIFICATION#notif-001` |
| SK | `METADATA` | `METADATA` |
| GSI1PK | `RECIPIENT#<recipient_id>` | `RECIPIENT#user-001` |
| GSI1SK | `<timestamp>` | `2024-01-15T10:30:00Z` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| Type | String | Yes | `EMAIL`, `SMS`, `PUSH` |
| Status | String | Yes | `PENDING`, `SENT`, `FAILED`, `DELIVERED` |
| RecipientID | String | Yes | Recipient user/customer ID |
| RecipientEmail | String | No | Recipient email |
| RecipientPhone | String | No | Recipient phone |
| Subject | String | No | Notification subject |
| Body | String | Yes | Notification content |
| TemplateID | String | No | Template used |
| TemplateData | Map | No | Template variables |
| TriggerType | String | Yes | `ORDER_STATUS`, `ORDER_CREATED`, `SHIPMENT`, `PAYMENT`, `REFUND`, `PASSWORD_RESET`, `MANUAL` |
| ReferenceType | String | No | Related entity type |
| ReferenceID | String | No | Related entity ID |
| SentAt | String | No | Send timestamp |
| DeliveredAt | String | No | Delivery timestamp |
| FailedAt | String | No | Failure timestamp |
| FailureReason | String | No | Failure reason |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID |

---

### 16. Notification Template

Reusable notification templates.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `NOTIFICATION_TEMPLATE#<id>` | `NOTIFICATION_TEMPLATE#tpl-001` |
| SK | `METADATA` | `METADATA` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| Name | String | Yes | Template name |
| Type | String | Yes | `EMAIL`, `SMS`, `PUSH` |
| Subject | String | No | Subject template |
| Body | String | Yes | Body template |
| Variables | List[String] | No | Available variables |
| IsActive | Boolean | Yes | Template status |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| UpdatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID |
| UpdatedBy | String | Yes | User ID |

---

### 17. Report

Generated report tracking.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `REPORT#<id>` | `REPORT#rep-001` |
| SK | `METADATA` | `METADATA` |
| GSI1PK | `USER#<created_by>` | `USER#user-001` |
| GSI1SK | `REPORT#<id>` | `REPORT#rep-001` |

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| ID | String | Yes | UUID |
| Name | String | Yes | Report name |
| Type | String | Yes | `SALES`, `ORDERS`, `INVENTORY`, `CUSTOMERS`, `PRODUCTS`, `ARTISANS` |
| Format | String | Yes | `CSV`, `XLSX`, `PDF` |
| Status | String | Yes | `PENDING`, `PROCESSING`, `COMPLETED`, `FAILED` |
| Parameters | Map | No | Report parameters |
| StartDate | String | No | Report period start |
| EndDate | String | No | Report period end |
| FileURL | String | No | Generated file URL |
| FileSize | Number | No | File size in bytes |
| RowCount | Number | No | Number of rows |
| ErrorMessage | String | No | Error if failed |
| StartedAt | String | No | Processing start |
| CompletedAt | String | No | Processing end |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| UpdatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID |
| UpdatedBy | String | Yes | User ID |
