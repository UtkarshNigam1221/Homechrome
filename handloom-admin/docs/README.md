# Handloom API Documentation

This directory contains comprehensive API documentation for all services in the Handloom system, organized by audience.

## Documentation Structure

```
docs/
├── admin/          ← Admin dashboard APIs (12 admin services)
├── store/          ← B2C storefront APIs (9 route groups)
├── database/       ← DynamoDB table designs (8 tables)
└── DESIGN.md       ← System-wide design document
```

Each service folder contains:
- **api-contract.md** — API endpoints, request/response formats, and sample payloads
- **user-flows.md** — User flow diagrams showing main user interactions
- **sequence-diagrams.md** — Component interaction sequence diagrams
- **hld.md** — High-Level Design document with architecture, data model, and error handling

---

## Admin APIs (`/admin/*`)

Admin dashboard services, each deployed as a separate AWS Lambda. See [admin/](./admin/) for full docs.

| Service | Description | Documentation |
|---------|-------------|---------------|
| [Auth](./admin/auth/) | Authentication & authorization | [API](./admin/auth/api-contract.md) · [Flows](./admin/auth/user-flows.md) · [Sequences](./admin/auth/sequence-diagrams.md) · [HLD](./admin/auth/hld.md) |
| [User](./admin/user/) | User management | [API](./admin/user/api-contract.md) · [Flows](./admin/user/user-flows.md) · [Sequences](./admin/user/sequence-diagrams.md) · [HLD](./admin/user/hld.md) |
| [Catalog](./admin/catalog/) | Categories, Designs, Products | [API](./admin/catalog/api-contract.md) · [Flows](./admin/catalog/user-flows.md) · [Sequences](./admin/catalog/sequence-diagrams.md) · [HLD](./admin/catalog/hld.md) |
| [Order](./admin/order/) | Orders & Customers | [API](./admin/order/api-contract.md) · [Flows](./admin/order/user-flows.md) · [Sequences](./admin/order/sequence-diagrams.md) · [HLD](./admin/order/hld.md) |
| [Pricing](./admin/pricing/) | Pricing rules & quotes | [API](./admin/pricing/api-contract.md) · [Flows](./admin/pricing/user-flows.md) · [Sequences](./admin/pricing/sequence-diagrams.md) · [HLD](./admin/pricing/hld.md) |
| [Inventory](./admin/inventory/) | Inventory management | [API](./admin/inventory/api-contract.md) · [Flows](./admin/inventory/user-flows.md) · [Sequences](./admin/inventory/sequence-diagrams.md) · [HLD](./admin/inventory/hld.md) |
| [Analytics](./admin/analytics/) | Dashboard & reports | [API](./admin/analytics/api-contract.md) · [Flows](./admin/analytics/user-flows.md) · [Sequences](./admin/analytics/sequence-diagrams.md) · [HLD](./admin/analytics/hld.md) |
| [Notification](./admin/notification/) | User notifications | [API](./admin/notification/api-contract.md) · [Flows](./admin/notification/user-flows.md) · [Sequences](./admin/notification/sequence-diagrams.md) · [HLD](./admin/notification/hld.md) |
| [Coupon](./admin/coupon/) | Discount coupons | [API](./admin/coupon/api-contract.md) · [Flows](./admin/coupon/user-flows.md) · [Sequences](./admin/coupon/sequence-diagrams.md) · [HLD](./admin/coupon/hld.md) |
| [Asset](./admin/asset/) | Media/file management | [API](./admin/asset/api-contract.md) · [Flows](./admin/asset/user-flows.md) · [Sequences](./admin/asset/sequence-diagrams.md) · [HLD](./admin/asset/hld.md) |
| [Report](./admin/report/) | Report generation | [API](./admin/report/api-contract.md) · [Flows](./admin/report/user-flows.md) · [Sequences](./admin/report/sequence-diagrams.md) · [HLD](./admin/report/hld.md) |
| [Audit](./admin/audit/) | Audit logging | [API](./admin/audit/api-contract.md) · [Flows](./admin/audit/user-flows.md) · [Sequences](./admin/audit/sequence-diagrams.md) · [HLD](./admin/audit/hld.md) |
| [Bulk](./admin/bulk/) | Bulk import/export | [API](./admin/bulk/api-contract.md) · [Flows](./admin/bulk/user-flows.md) · [Sequences](./admin/bulk/sequence-diagrams.md) · [HLD](./admin/bulk/hld.md) |

---

## B2C Store APIs (`/api/v1/store/*`)

Customer-facing storefront APIs consumed by the Next.js frontend. See [store/](./store/) for full docs.

| Service | Base Path | Auth | Documentation |
|---------|-----------|------|---------------|
| [Auth](./store/auth/) | `/api/v1/store/auth` | Public (rate-limited) | [API](./store/auth/api-contract.md) · [Flows](./store/auth/user-flows.md) · [Sequences](./store/auth/sequence-diagrams.md) · [HLD](./store/auth/hld.md) |
| [Catalog](./store/catalog/) | `/api/v1/store/catalog` | Public | [API](./store/catalog/api-contract.md) · [Flows](./store/catalog/user-flows.md) · [Sequences](./store/catalog/sequence-diagrams.md) · [HLD](./store/catalog/hld.md) |
| [Cart](./store/cart/) | `/api/v1/store/cart` | Customer JWT | [API](./store/cart/api-contract.md) · [Flows](./store/cart/user-flows.md) · [Sequences](./store/cart/sequence-diagrams.md) · [HLD](./store/cart/hld.md) |
| [Checkout](./store/checkout/) | `/api/v1/store/checkout` | Customer JWT | [API](./store/checkout/api-contract.md) · [Flows](./store/checkout/user-flows.md) · [Sequences](./store/checkout/sequence-diagrams.md) · [HLD](./store/checkout/hld.md) |
| [Orders](./store/orders/) | `/api/v1/store/orders` | Customer JWT | [API](./store/orders/api-contract.md) · [Flows](./store/orders/user-flows.md) · [Sequences](./store/orders/sequence-diagrams.md) · [HLD](./store/orders/hld.md) |
| [Profile](./store/profile/) | `/api/v1/store/me` | Customer JWT | [API](./store/profile/api-contract.md) · [Flows](./store/profile/user-flows.md) · [Sequences](./store/profile/sequence-diagrams.md) · [HLD](./store/profile/hld.md) |
| [Tracking](./store/tracking/) | `/api/v1/store/track` | Public | [API](./store/tracking/api-contract.md) · [Flows](./store/tracking/user-flows.md) · [Sequences](./store/tracking/sequence-diagrams.md) · [HLD](./store/tracking/hld.md) |
| [Events](./store/events/) | `/api/v1/store/events` | Public (rate-limited) | [API](./store/events/api-contract.md) · [Flows](./store/events/user-flows.md) · [Sequences](./store/events/sequence-diagrams.md) · [HLD](./store/events/hld.md) |
| [Webhooks](./store/webhooks/) | `/api/v1/store/webhooks` | Signature-verified | [API](./store/webhooks/api-contract.md) · [Flows](./store/webhooks/user-flows.md) · [Sequences](./store/webhooks/sequence-diagrams.md) · [HLD](./store/webhooks/hld.md) |

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────────────────┐
│                              HANDLOOM SYSTEM ARCHITECTURE                                    │
└─────────────────────────────────────────────────────────────────────────────────────────────┘

   ┌───────────────────┐                              ┌───────────────────┐
   │   React Frontend  │                              │  Next.js Store    │
   │   (Admin Portal)  │                              │  (B2C Storefront) │
   └─────────┬─────────┘                              └─────────┬─────────┘
             │                                                  │
             │ HTTPS /admin/*                                   │ HTTPS /api/v1/store/*
             ▼                                                  ▼
   ┌───────────────────────────────────────────────────────────────────────┐
   │                         API Gateway (REST API)                        │
   └───────────────────────────────┬───────────────────────────────────────┘
                                   │
                    ┌──────────────┼──────────────┐
                    │              │              │
                    ▼              ▼              ▼
            ┌─────────────┐ ┌──────────┐ ┌─────────────┐
            │ Admin APIs  │ │ B2C APIs │ │ Webhooks    │
            │ /admin/*    │ │ /store/* │ │ PhonePe     │
            │ JWT auth    │ │ OTP auth │ │ Delhivery   │
            └──────┬──────┘ └────┬─────┘ └──────┬──────┘
                   │             │               │
    ┌──────────────┴─────────────┴───────────────┴──────────────┐
    │                     Lambda Functions                        │
    │  admin: auth · user · catalog · order · pricing ·          │
    │         inventory · analytics · notification · coupon ·    │
    │         asset · report · audit                             │
    │  store: auth · catalog · cart · checkout · orders ·        │
    │         profile · tracking · events · webhooks             │
    └────────────────────────────┬───────────────────────────────┘
                                 │
                    ┌────────────┴────────────┐
                    │     SNS Event Topic     │
                    └────────────┬────────────┘
                                 │
              ┌──────────┬───────┴───────┬──────────┐
              ▼          ▼               ▼          ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
        │ Worker:  │ │ Worker:  │ │ Worker:  │ │ Worker:  │
        │Analytics │ │  Audit   │ │  Notif   │ │ Report   │
        └──────────┘ └──────────┘ └──────────┘ └──────────┘
                                 │
              ┌──────────────────┼──────────────────┐
              │                  │                  │
              ▼                  ▼                  ▼
   ┌─────────────────┐  ┌─────────────┐  ┌─────────────────┐
   │    DynamoDB      │  │  PostgreSQL │  │     S3      │  │   External      │
   │  (7 tables)      │  │  (Catalog)  │  │  (Assets)   │  │   Gateways      │
   │  core · orders    │  │ categories  │  │             │  │  PhonePe        │
   │  sessions · audit │  │ products    │  │             │  │  Delhivery      │
   │  analytics · notif│  │ inventory   │  │             │  │  MSG91 SMS      │
   │  events           │  │             │  │             │  │                 │
   └─────────────────┘  └─────────────┘  └─────────────┘  └─────────────────┘
```

## Shared Resources

- [Database Design](./database/) — DynamoDB table schemas, key patterns, GSI definitions
- [System Design](./DESIGN.md) — Overall architecture and design decisions

## Common Conventions

- **Prices** in paise (1 INR = 100 paise)
- **Pagination** is cursor-based (base64-encoded DynamoDB ExclusiveStartKey)
- **Response envelope**: `{success: true, data: T, meta?: {...}}` or `{success: false, error: {code, message}}`
- **Admin auth**: JWT in `access_token` HttpOnly cookie or `Authorization: Bearer` header
- **Customer auth**: JWT in `store_token` HttpOnly cookie, refresh via `store_refresh` cookie
