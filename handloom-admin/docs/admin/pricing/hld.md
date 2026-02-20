# Pricing Lambda - High-Level Design (HLD)

## 1. Overview

The Pricing Lambda service manages pricing rules and price calculations for the Handloom Admin system. It supports multiple pricing models including fixed pricing, area-based pricing, length-based pricing, and tiered pricing with material multipliers and attribute surcharges.

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                                    PRICING SYSTEM                                            │
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
              │  Pricing Rules  │  │  Price          │  │  Dimension      │
              │  (Admin)        │  │  Calculator     │  │  Options        │
              │  - CRUD         │  │  - Calculate    │  │  - Get config   │
              │  - List         │  │  - Bulk calc    │  │                 │
              └────────┬────────┘  └────────┬────────┘  └────────┬────────┘
                       │                    │                    │
                       └────────────────────┼────────────────────┘
                                            │
                                            ▼
                                  ┌─────────────────────┐
                                  │   Pricing Service   │
                                  │   (Business Logic)  │
                                  └──────────┬──────────┘
                                             │
                         ┌───────────────────┼───────────────────┐
                         │                   │                   │
                         ▼                   ▼                   ▼
              ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
              │   DynamoDB      │  │   Category      │  │   CloudWatch    │
              │   (Rules,       │  │   Service       │  │   (Logs &       │
              │   Quotes)       │  │   (Dimensions)  │  │   Metrics)      │
              └─────────────────┘  └─────────────────┘  └─────────────────┘
```

---

## 3. Component Design

### 3.1 Pricing Handler

```
┌─────────────────────────────────────────────────────────────────────┐
│                         PRICING HANDLER                              │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Admin Routes (/admin/pricing):                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ POST   /rules              - Create pricing rule             │    │
│  │ GET    /rules              - List pricing rules              │    │
│  │ GET    /rules/{id}         - Get rule details               │    │
│  │ PATCH  /rules/{id}         - Update pricing rule            │    │
│  │ DELETE /rules/{id}         - Delete pricing rule            │    │
│  │ GET    /rules/category/{id}- Get rules for category         │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
│  Public Routes (/api/pricing):                                       │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ POST   /calculate          - Calculate price                 │    │
│  │ POST   /bulk-calculate     - Bulk price calculation         │    │
│  │ GET    /dimension-options/{id} - Get dimension options      │    │
│  └─────────────────────────────────────────────────────────────┘    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.2 Service Layer Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           PRICING SERVICE LAYER                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        PricingService                                │   │
│  │                                                                      │   │
│  │  Rule Management:                                                    │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │  - CreateRule()         - GetRule()                           │ │   │
│  │  │  - UpdateRule()         - DeleteRule()                        │ │   │
│  │  │  - ListRules()          - GetRulesForCategory()               │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  │                                                                      │   │
│  │  Price Calculation:                                                  │   │
│  │  ┌───────────────────────────────────────────────────────────────┐ │   │
│  │  │  - CalculatePrice()     - BulkCalculatePrice()                │ │   │
│  │  │  - GetDimensionOptions()- GetQuote()                          │ │   │
│  │  │  - validateDimensions() - calculatePriceBreakdown()           │ │   │
│  │  │  - findEffectiveRule()  - convertArea()                       │ │   │
│  │  └───────────────────────────────────────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                             │                                               │
│         ┌───────────────────┼───────────────────┐                          │
│         │                   │                   │                          │
│         ▼                   ▼                   ▼                          │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐              │
│  │ PricingRule     │ │ PriceQuote      │ │ Category        │              │
│  │ Repository      │ │ Repository      │ │ Repository      │              │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘              │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Model

### 4.1 DynamoDB Table Design

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          TABLE: handloom-admin                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  PRICING RULE RECORDS                                                        │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: PRICING_RULE#<rule_id>                                        │      │
│  │ SK: METADATA                                                      │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - name                    - description                         │      │
│  │   - scope_type              - scope_id                            │      │
│  │   - category_id             - material_name                       │      │
│  │   - pricing_type            - base_price                          │      │
│  │   - price_per_unit          - unit                                │      │
│  │   - material_multipliers{}  - attribute_surcharges[]              │      │
│  │   - tiers[]                 - formula                             │      │
│  │   - min_area                - max_area                            │      │
│  │   - min_order_value         - priority                            │      │
│  │   - is_active               - valid_from                          │      │
│  │   - valid_until             - created_by                          │      │
│  │   - created_at              - updated_at                          │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  PRICE QUOTE RECORDS                                                         │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ PK: QUOTE#<quote_id>                                              │      │
│  │ SK: METADATA                                                      │      │
│  │                                                                   │      │
│  │ Attributes:                                                       │      │
│  │   - category_id             - product_id                          │      │
│  │   - dimensions{}            - attributes{}                        │      │
│  │   - quantity                - pricing_rule_id                     │      │
│  │   - calculated_price        - price_breakdown{}                   │      │
│  │   - valid_until             - created_at                          │      │
│  │   - used_in_order           - ttl                                 │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  GLOBAL SECONDARY INDEXES                                                    │
│  ┌───────────────────────────────────────────────────────────────────┐      │
│  │ GSI1: scope-index         (rules by scope_type + scope_id)        │      │
│  │       GSI1PK: SCOPE#<scope_type>                                  │      │
│  │       GSI1SK: <scope_id>                                          │      │
│  └───────────────────────────────────────────────────────────────────┘      │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 4.2 Pricing Rule Scope Hierarchy

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        PRICING RULE SCOPE HIERARCHY                          │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Priority Order (Highest to Lowest):                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  1. PRODUCT Scope    (Most specific - applies to single product)    │   │
│  │         │                                                            │   │
│  │         ▼                                                            │   │
│  │  2. MATERIAL Scope   (Applies to specific material type)            │   │
│  │         │                                                            │   │
│  │         ▼                                                            │   │
│  │  3. CATEGORY Scope   (Applies to all products in category)          │   │
│  │         │                                                            │   │
│  │         ▼                                                            │   │
│  │  4. GLOBAL Scope     (Fallback rule for all products)               │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Rule Selection Algorithm:                                                   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  1. Collect all applicable rules (product, material, category,      │   │
│  │     parent categories, global)                                       │   │
│  │  2. Filter by validity period (valid_from <= now <= valid_until)    │   │
│  │  3. Filter by is_active = true                                       │   │
│  │  4. Sort by priority (descending)                                    │   │
│  │  5. Select highest priority rule                                     │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Pricing Types

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            PRICING TYPES                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  FIXED Pricing:                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Simple flat rate pricing                                           │   │
│  │ • Price = base_price                                                 │   │
│  │ • Use case: Standard-sized products with fixed prices               │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  AREA_BASED Pricing:                                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Price based on product dimensions (length x width)                 │   │
│  │ • Price = base_price + (area × price_per_unit)                      │   │
│  │ • Supports unit conversion (sq inch, sq foot, sq cm, sq meter)      │   │
│  │ • Use case: Sarees, fabrics, rugs with custom dimensions            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  LENGTH_BASED Pricing:                                                       │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Price based on single dimension (length)                           │   │
│  │ • Price = base_price + (length × price_per_unit)                    │   │
│  │ • Use case: Fabric rolls, borders, trims                            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  TIERED Pricing:                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Different price per unit based on area range                       │   │
│  │ • Tiers: [{min: 0, max: 10, price: 100},                            │   │
│  │          {min: 10, max: 20, price: 90},                             │   │
│  │          {min: 20, max: 50, price: 80}]                             │   │
│  │ • Use case: Volume discounts based on size                          │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Price Breakdown Structure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         PRICE BREAKDOWN                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  {                                                                           │
│    "base_cost": 500000,           // Base price in paise (₹5,000)           │
│    "area": 24.0,                  // Calculated area                        │
│    "area_unit": "sq_feet",        // Unit of measurement                    │
│    "material_multiplier": 1.5,    // Silk multiplier                        │
│    "material_cost": 750000,       // After material multiplier              │
│    "surcharges": [                                                          │
│      {                                                                       │
│        "attribute": "border",                                                │
│        "value": "zari",                                                      │
│        "amount": 50000            // ₹500 surcharge                         │
│      }                                                                       │
│    ],                                                                        │
│    "surcharges_total": 50000,     // Total surcharges                       │
│    "subtotal_per_unit": 800000,   // Per unit price                         │
│    "quantity": 2,                                                            │
│    "total": 1600000               // Final total (₹16,000)                  │
│  }                                                                           │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Error Handling

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          ERROR CODES                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Pricing Rule Errors:                                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ PRC001 │ Pricing rule not found                                     │   │
│  │ PRC002 │ No applicable pricing rule found                           │   │
│  │ PRC003 │ Invalid pricing type                                       │   │
│  │ PRC004 │ Rule already exists                                        │   │
│  │ PRC005 │ Invalid validity period                                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Dimension Errors:                                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ DIM001 │ Dimensions required                                        │   │
│  │ DIM002 │ Length out of range                                        │   │
│  │ DIM003 │ Width out of range                                         │   │
│  │ DIM004 │ Height out of range                                        │   │
│  │ DIM005 │ Area out of allowed range                                  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Quote Errors:                                                               │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ QOT001 │ Quote not found                                            │   │
│  │ QOT002 │ Quote expired                                              │   │
│  │ QOT003 │ Quote already used                                         │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Order Value Errors:                                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ MOV001 │ Minimum order value not met                                │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Quote Management

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          QUOTE MANAGEMENT                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Quote Lifecycle:                                                            │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                      │   │
│  │  Calculate Price ──▶ Quote Created ──▶ Quote Valid ──▶ Used/Expired │   │
│  │                           │                │                         │   │
│  │                           │                │                         │   │
│  │                    (24hr validity)    (Order placed)                │   │
│  │                                                                      │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Quote Properties:                                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Quote ID: Unique identifier for price quote                       │   │
│  │ • Valid Until: 24 hours from creation (configurable)               │   │
│  │ • TTL: Automatic DynamoDB cleanup after expiry                      │   │
│  │ • Used In Order: Tracks if quote was used in an order              │   │
│  │ • Immutable: Price locked at calculation time                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 9. Scalability & Performance

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     SCALABILITY CONSIDERATIONS                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Lambda Configuration:                                                       │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Memory: 256 MB                                                    │   │
│  │ • Timeout: 30 seconds                                               │   │
│  │ • Concurrent executions: 100 (reserved)                             │   │
│  │ • Provisioned concurrency: 5 (for cold start mitigation)           │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  DynamoDB Configuration:                                                     │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Capacity: On-demand                                               │   │
│  │ • GSI: Scope index for efficient rule lookups                       │   │
│  │ • TTL: Enabled for quote auto-cleanup                               │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Caching Strategy:                                                           │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Pricing rules cached in Lambda memory (5 min TTL)                 │   │
│  │ • Category dimension configs cached (10 min TTL)                    │   │
│  │ • Quote validity configurable via environment variable              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Performance Optimizations:                                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Bulk calculation limited to 10 configurations per request         │   │
│  │ • GSI for efficient scope-based rule queries                        │   │
│  │ • Area conversion formulas optimized for common units              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 10. Dependencies

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          DEPENDENCIES                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  External Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • AWS DynamoDB - Rule and quote storage                             │   │
│  │ • AWS CloudWatch - Logging and metrics                              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Internal Services:                                                          │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • Auth Lambda - Authentication                                      │   │
│  │ • Catalog Lambda - Category and product information                 │   │
│  │ • Order Lambda - Quote validation during checkout                   │   │
│  │ • Audit Lambda - Rule change logging                                │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Libraries:                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ • aws-sdk-go-v2/dynamodb - DynamoDB client                          │   │
│  │ • go-chi/chi - HTTP routing                                         │   │
│  │ • google/uuid - ID generation                                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```
