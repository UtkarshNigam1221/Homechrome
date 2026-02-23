# handloom-core Table

The core table contains admin users, pricing rules, and coupons.

## Table Configuration

```
Table Name: handloom-core
Partition Key: PK (String)
Sort Key: SK (String)
Billing Mode: PAY_PER_REQUEST
TTL Attribute: ttl
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

#### Email Uniqueness Guard

Created atomically with the user via `TransactWriteItems`. Both items use `attribute_not_exists(PK)` to guarantee uniqueness.

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `USER_EMAIL#<email>` | `USER_EMAIL#admin@handloom.com` |
| SK | `UNIQUENESS` | `UNIQUENESS` |

Additional attributes: `user_id`, `entity_type` (`EMAIL_GUARD`).

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
| List all users | GSI1: GSI1PK = `USER_EMAIL` (paginated, in-memory filter/sort) |

#### Write Patterns

| Operation | Method | Condition |
|-----------|--------|-----------|
| Create | `TransactWriteItems` (user + email guard) | `attribute_not_exists(PK)` on both |
| Update | `PutItem` | `attribute_exists(PK)` |
| Delete | `TransactWriteItems` (user + email guard) | `attribute_exists(PK)` on user |

---

### 2. Pricing Rule

Dynamic pricing rules for products and categories.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `PRICING_RULE#<id>` | `PRICING_RULE#rule-001` |
| SK | `METADATA` | `METADATA` |
| GSI1PK | `SCOPE#<scope_type>` | `SCOPE#CATEGORY` |
| GSI1SK | `<scope_id>` | `cat-001` |
| GSI2PK | `PRICING_RULE#ALL` | `PRICING_RULE#ALL` |
| GSI2SK | `<created_at>` | `2024-01-15T10:30:00Z` |

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
| ValidFrom | String | No | Rule start date |
| ValidUntil | String | No | Rule end date |
| CreatedAt | String | Yes | ISO 8601 timestamp |
| UpdatedAt | String | Yes | ISO 8601 timestamp |
| CreatedBy | String | Yes | User ID |
| UpdatedBy | String | Yes | User ID |

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get rule by ID | PK = `PRICING_RULE#<id>`, SK = `METADATA` |
| Get rules by scope | GSI1: GSI1PK = `SCOPE#<type>`, GSI1SK = `<scope_id>` |
| List all rules | GSI2: GSI2PK = `PRICING_RULE#ALL` (with filter expressions) |

---

### 3. Coupon

Discount coupons with usage tracking.

#### Key Structure

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `COUPON#<id>` | `COUPON#coup-001` |
| SK | `METADATA` | `METADATA` |

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

#### Coupon Usage (sub-item)

| Key | Pattern | Example |
|-----|---------|---------|
| PK | `COUPON#<coupon_id>` | `COUPON#coup-001` |
| SK | `USAGE#<timestamp>#<user_id>` | `USAGE#2024-01-15T10:30:00Z#user-001` |

Usage records track individual redemptions. Count queries use `Select: SelectCount` with optional `customer_id` filter.

#### Access Patterns

| Pattern | Key Condition |
|---------|---------------|
| Get coupon by ID | PK = `COUPON#<id>`, SK = `METADATA` |
| Get coupon usage | PK = `COUPON#<id>`, SK begins_with `USAGE#` |
| Count usage by customer | PK = `COUPON#<id>`, SK begins_with `USAGE#`, filter `customer_id` |
