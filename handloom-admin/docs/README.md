# Handloom Admin API Documentation

This directory contains comprehensive API documentation for all Lambda functions in the Handloom Admin system.

## Documentation Structure

Each Lambda service has its own folder containing:
- **api-contract.md** - API endpoints, request/response formats, and sample payloads
- **user-flows.md** - User flow diagrams showing main user interactions
- **sequence-diagrams.md** - Component interaction sequence diagrams
- **hld.md** - High-Level Design document with architecture, data model, and error handling

## Lambda Services

| Lambda | Description | Documentation |
|--------|-------------|---------------|
| [Auth](./auth/) | Authentication & authorization | [API](./auth/api-contract.md) · [Flows](./auth/user-flows.md) · [Sequences](./auth/sequence-diagrams.md) · [HLD](./auth/hld.md) |
| [User](./user/) | User management (admin only) | [API](./user/api-contract.md) · [Flows](./user/user-flows.md) · [Sequences](./user/sequence-diagrams.md) · [HLD](./user/hld.md) |
| [Catalog](./catalog/) | Categories, Designs, Products | [API](./catalog/api-contract.md) · [Flows](./catalog/user-flows.md) · [Sequences](./catalog/sequence-diagrams.md) · [HLD](./catalog/hld.md) |
| [Order](./order/) | Orders & Customers | [API](./order/api-contract.md) · [Flows](./order/user-flows.md) · [Sequences](./order/sequence-diagrams.md) · [HLD](./order/hld.md) |
| [Pricing](./pricing/) | Pricing rules & quotes | [API](./pricing/api-contract.md) · [Flows](./pricing/user-flows.md) · [Sequences](./pricing/sequence-diagrams.md) · [HLD](./pricing/hld.md) |
| [Inventory](./inventory/) | Inventory management | [API](./inventory/api-contract.md) · [Flows](./inventory/user-flows.md) · [Sequences](./inventory/sequence-diagrams.md) · [HLD](./inventory/hld.md) |
| [Analytics](./analytics/) | Dashboard & reports | [API](./analytics/api-contract.md) · [Flows](./analytics/user-flows.md) · [Sequences](./analytics/sequence-diagrams.md) · [HLD](./analytics/hld.md) |
| [Notification](./notification/) | User notifications | [API](./notification/api-contract.md) · [Flows](./notification/user-flows.md) · [Sequences](./notification/sequence-diagrams.md) · [HLD](./notification/hld.md) |
| [Coupon](./coupon/) | Discount coupons | [API](./coupon/api-contract.md) · [Flows](./coupon/user-flows.md) · [Sequences](./coupon/sequence-diagrams.md) · [HLD](./coupon/hld.md) |
| [Artisan](./artisan/) | Artisan management | [API](./artisan/api-contract.md) · [Flows](./artisan/user-flows.md) · [Sequences](./artisan/sequence-diagrams.md) · [HLD](./artisan/hld.md) |
| [Bulk](./bulk/) | Bulk import/export operations | [API](./bulk/api-contract.md) · [Flows](./bulk/user-flows.md) · [Sequences](./bulk/sequence-diagrams.md) · [HLD](./bulk/hld.md) |
| [Asset](./asset/) | Media/file management | [API](./asset/api-contract.md) · [Flows](./asset/user-flows.md) · [Sequences](./asset/sequence-diagrams.md) · [HLD](./asset/hld.md) |
| [Report](./report/) | Report generation | [API](./report/api-contract.md) · [Flows](./report/user-flows.md) · [Sequences](./report/sequence-diagrams.md) · [HLD](./report/hld.md) |
| [Audit](./audit/) | Audit logging | [API](./audit/api-contract.md) · [Flows](./audit/user-flows.md) · [Sequences](./audit/sequence-diagrams.md) · [HLD](./audit/hld.md) |

## System Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                              HANDLOOM ADMIN SYSTEM ARCHITECTURE                              │
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
    ┌─────────┬─────────┬─────────┬──────────┼──────────┬─────────┬─────────┬─────────┐
    │         │         │         │          │          │         │         │         │
    ▼         ▼         ▼         ▼          ▼          ▼         ▼         ▼         ▼
┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐ ┌───────┐
│ Auth  │ │Catalog│ │ Order │ │Pricing│ │Invent │ │Notif  │ │Coupon │ │ Bulk  │ │Report │
│Lambda │ │Lambda │ │Lambda │ │Lambda │ │Lambda │ │Lambda │ │Lambda │ │Lambda │ │Lambda │
└───┬───┘ └───┬───┘ └───┬───┘ └───┬───┘ └───┬───┘ └───┬───┘ └───┬───┘ └───┬───┘ └───┬───┘
    │         │         │         │         │         │         │         │         │
    └─────────┴─────────┴─────────┴────┬────┴─────────┴─────────┴─────────┴─────────┘
                                       │
                         ┌─────────────┼─────────────┐
                         │             │             │
                         ▼             ▼             ▼
              ┌─────────────────┐ ┌─────────┐ ┌─────────────────┐
              │    DynamoDB     │ │   S3    │ │   CloudWatch    │
              │ (Single Table)  │ │ (Assets)│ │ (Logs/Metrics)  │
              └─────────────────┘ └─────────┘ └─────────────────┘
```

## Authentication

All protected endpoints require a valid JWT token in the Authorization header:

```
Authorization: Bearer <jwt_token>
```

## Common Response Formats

### Success Response
```json
{
  "data": { ... },
  "message": "Operation successful"
}
```

### Error Response
```json
{
  "error": "Error message",
  "code": "ERROR_CODE"
}
```

### Paginated Response
```json
{
  "data": [...],
  "pagination": {
    "current_page": 1,
    "per_page": 10,
    "total_count": 100,
    "total_pages": 10
  }
}
```

## Quick Links

### Core Operations
- [Login](./auth/api-contract.md#login) - Authenticate users
- [Create Product](./catalog/api-contract.md#create-product) - Add new products
- [Create Order](./order/api-contract.md#create-order) - Place orders
- [Update Inventory](./inventory/api-contract.md#update-inventory) - Manage stock

### Reporting
- [Dashboard Stats](./analytics/api-contract.md#get-dashboard-stats) - Overview metrics
- [Sales Analytics](./analytics/api-contract.md#get-sales-analytics) - Sales data
- [Generate Reports](./report/api-contract.md#generate-report) - Custom reports

### Bulk Operations
- [Export Products](./bulk/api-contract.md#export-data) - Bulk export
- [Import Products](./bulk/api-contract.md#import-products) - Bulk import

## DynamoDB Single-Table Design

All entities are stored in a single DynamoDB table using composite keys:

| Entity | PK Pattern | SK Pattern |
|--------|-----------|------------|
| User | `USER#<id>` | `PROFILE` |
| Product | `PRODUCT#<id>` | `METADATA` |
| Category | `CATEGORY#<id>` | `METADATA` |
| Order | `ORDER#<id>` | `METADATA` |
| Customer | `CUSTOMER#<id>` | `PROFILE` |
| Inventory | `INVENTORY#<product_id>` | `STOCK` |
| Coupon | `COUPON#<id>` | `METADATA` |
| Artisan | `ARTISAN#<id>` | `PROFILE` |
| Bulk Operation | `BULK#<id>` | `METADATA` |
| Asset | `ASSET#<id>` | `METADATA` |
| Audit Log | `AUDIT#<id>` | `<timestamp>` |

## TODO Items

No pending TODO items were identified in the handler implementations.

---

## Related Documentation

- [DESIGN.md](./DESIGN.md) - Detailed system design document
