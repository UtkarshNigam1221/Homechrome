# Handloom E-Commerce Admin Portal - Design Document

## Table of Contents
1. [High-Level Design (HLD)](#high-level-design)
   - [System Architecture Overview](#system-architecture-overview)
   - [Microservices Architecture](#microservices-architecture)
   - [Infrastructure Design (AWS CDK)](#infrastructure-design-aws-cdk)
   - [Database Design](#database-design-hybrid-multi-table-approach)
2. [Category & Attribute System](#category--attribute-system)
3. [Dynamic Pricing Engine](#dynamic-pricing-engine)
4. [Entity Design](#entity-design)
5. [Low-Level Design (LLD)](#low-level-design)
6. [API Contracts](#api-contracts)
7. [Additional Features](#additional-features)

---

## High-Level Design

### System Architecture Overview

The Handloom Admin Portal is built as a **serverless microservices architecture** on AWS, designed for high availability, scalability, and cost efficiency.

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                         HANDLOOM ADMIN SYSTEM ARCHITECTURE                           │
├─────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                      │
│                              ┌─────────────────────┐                                │
│                              │   Admin Frontend    │                                │
│                              │   (React/Next.js)   │                                │
│                              └──────────┬──────────┘                                │
│                                         │                                           │
│                                         ▼                                           │
│  ┌──────────────────────────────────────────────────────────────────────────────┐  │
│  │                           API Gateway (REST)                                  │  │
│  │  ┌─────────────┬─────────────┬─────────────┬─────────────┬─────────────┐     │  │
│  │  │  /admin/*   │  /api/v1/*  │   CORS      │   Logging   │  Throttle   │     │  │
│  │  │  (Admin)    │  (Public)   │   Config    │   Enabled   │  100/200    │     │  │
│  │  └─────────────┴─────────────┴─────────────┴─────────────┴─────────────┘     │  │
│  └─────────────────────────────────────┬────────────────────────────────────────┘  │
│                                        │                                           │
│    ┌───────────────────────────────────┼───────────────────────────────────┐       │
│    │                                   │                                   │       │
│    ▼                                   ▼                                   ▼       │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐    │
│  │  Auth   │  │  User   │  │ Catalog │  │  Order  │  │ Pricing │  │Inventory│    │
│  │ Lambda  │  │ Lambda  │  │ Lambda  │  │ Lambda  │  │ Lambda  │  │ Lambda  │    │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘    │
│       │            │            │            │            │            │          │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐               │
│  │Analytics│  │  Notif  │  │ Coupon  │  │  Asset  │  │  Bulk   │               │
│  │ Lambda  │  │ Lambda  │  │ Lambda  │  │ Lambda  │  │ Lambda  │               │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘               │
│       │            │            │            │            │                      │
│  ┌─────────┐  ┌─────────┐                                                        │
│  │ Report  │  │  Audit  │   All Lambdas: ARM64, provided.al2023                  │
│  │ Lambda  │  │ Lambda  │   Memory: 128MB (dev) / 256MB (prod)                   │
│  └────┬────┘  └────┬────┘   Timeout: 30s, X-Ray Tracing Disabled                 │
│       │            │                                                              │
│       └────────────┼────────────────────────┬───────────────────────────┘        │
│                    │                        │                                     │
│                    ▼                        ▼                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐ │
│  │                           DynamoDB (7 Tables)                                │ │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐        │ │
│  │  │handloom-core │ │handloom-     │ │handloom-     │ │handloom-     │        │ │
│  │  │              │ │orders        │ │audit         │ │analytics     │        │ │
│  │  │• Users       │ │• Orders      │ │• AuditLogs   │ │• Metrics     │        │ │
│  │  │• PricingRules│ │• OrderItems  │ │  (90d TTL)   │ │• TopProducts │        │ │
│  │  │• Coupons     │ │• Customers   │ │              │ │• Alerts      │        │ │
│  │  │              │ │• StatusHist  │ │              │ │  (2yr TTL)   │        │ │
│  │  │ (PITR on)    │ │ (PITR on)    │ │              │ │              │        │ │
│  │  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘        │ │
│  └─────────────────────────────────────────────────────────────────────────────┘ │
│                                        │                                          │
│                                        ▼                                          │
│  ┌─────────────────────────────────────────────────────────────────────────────┐ │
│  │                              Storage Layer                                   │ │
│  │  ┌────────────────────────┐      ┌──────────────────────────────────────┐   │ │
│  │  │   S3 Buckets           │      │         CloudFront CDN               │   │ │
│  │  │  ├── Assets (public)   │─────▶│   • HTTPS/TLS                        │   │ │
│  │  │  └── Uploads (private) │      │   • Edge caching                     │   │ │
│  │  └────────────────────────┘      │   • Image optimization               │   │ │
│  │                                   └──────────────────────────────────────┘   │ │
│  └─────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
└───────────────────────────────────────────────────────────────────────────────────┘
```

### Microservices Architecture

The application is decomposed into **25 Lambda services** (12 admin + 9 B2C store + 4 event workers), each responsible for a specific domain:

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                          LAMBDA MICROSERVICES BREAKDOWN                              │
├─────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                      │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                          AUTHENTICATION & USERS                              │   │
│  │  ┌─────────────────────────┐   ┌─────────────────────────────────────────┐  │   │
│  │  │       Auth Lambda       │   │           User Lambda                   │  │   │
│  │  │  ├── POST /auth/login   │   │  ├── GET/POST /users                   │  │   │
│  │  │  ├── POST /auth/refresh │   │  ├── GET/PATCH/DELETE /users/{id}      │  │   │
│  │  │  ├── POST /auth/logout  │   │  └── PATCH /users/{id}/status          │  │   │
│  │  │  └── POST /password/*   │   │                                        │  │   │
│  │  └─────────────────────────┘   └─────────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                      │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                           PRODUCT CATALOG                                    │   │
│  │  ┌───────────────────────────────────────────────────────────────────────┐  │   │
│  │  │                        Catalog Lambda                                  │  │   │
│  │  │  ├── Categories: GET/POST/PATCH/DELETE /categories                    │  │   │
│  │  │  ├── Designs:    GET/POST/PATCH/DELETE /designs                       │  │   │
│  │  │  └── Products:   GET/POST/PATCH/DELETE /products                      │  │   │
│  │  └───────────────────────────────────────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                      │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                           ORDER MANAGEMENT                                   │   │
│  │  ┌───────────────────────────────────────────────────────────────────────┐  │   │
│  │  │                         Order Lambda                                   │  │   │
│  │  │  ├── Orders:    GET/POST/PATCH /orders, /orders/{id}/status           │  │   │
│  │  │  ├── Customers: GET/POST/PATCH/DELETE /customers                      │  │   │
│  │  │  └── Items:     GET /orders/{id}/items                                │  │   │
│  │  └───────────────────────────────────────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                      │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                      PRICING & INVENTORY                                     │   │
│  │  ┌─────────────────────────────┐   ┌─────────────────────────────────────┐  │   │
│  │  │      Pricing Lambda         │   │        Inventory Lambda             │  │   │
│  │  │  PUBLIC:                    │   │  ├── GET /inventory                 │  │   │
│  │  │  ├── POST /api/v1/calculate │   │  ├── PATCH /inventory/{id}          │  │   │
│  │  │  ├── GET  /dimension-opts   │   │  ├── POST /inventory/adjust         │  │   │
│  │  │  └── POST /bulk-calculate   │   │  └── GET /inventory/alerts          │  │   │
│  │  │  ADMIN:                     │   │                                     │  │   │
│  │  │  └── CRUD /pricing/rules    │   │                                     │  │   │
│  │  └─────────────────────────────┘   └─────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                      │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                         ANALYTICS & REPORTING                                │   │
│  │  ┌─────────────────────────────┐   ┌─────────────────────────────────────┐  │   │
│  │  │     Analytics Lambda        │   │         Report Lambda               │  │   │
│  │  │  ├── GET /analytics/dash    │   │  ├── GET/POST /reports              │  │   │
│  │  │  ├── GET /analytics/sales   │   │  ├── GET /reports/{id}              │  │   │
│  │  │  ├── GET /top-products      │   │  └── GET /reports/{id}/download     │  │   │
│  │  │  ├── GET /top-categories    │   │                                     │  │   │
│  │  │  └── GET /analytics/inv     │   │                                     │  │   │
│  │  └─────────────────────────────┘   └─────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                      │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                        AUXILIARY SERVICES                                    │   │
│  │  ┌─────────────────┐ ┌─────────────────┐                                     │   │
│  │  │ Coupon Lambda   │ │ Notification    │                                     │   │
│  │  │ ├── CRUD coupon │ │ Lambda          │                                     │   │
│  │  │ ├── POST apply  │ │ ├── CRUD notifs │                                     │   │
│  │  │ └── GET /code   │ │ └── GET my-noti │                                     │   │
│  │  └─────────────────┘ └─────────────────┘                                     │   │
│  │                                                                              │   │
│  │  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐                │   │
│  │  │  Bulk Lambda    │ │  Asset Lambda   │ │  Audit Lambda   │                │   │
│  │  │ ├── POST import │ │ ├── POST upload │ │ ├── GET logs    │                │   │
│  │  │ ├── POST export │ │ ├── DELETE asset│ │ ├── GET by entity│               │   │
│  │  │ └── GET status  │ │ └── GET assets  │ │ └── GET by user │                │   │
│  │  └─────────────────┘ └─────────────────┘ └─────────────────┘                │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                      │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

### Lambda Service Details

| Service | Lambda Name | Purpose | Key Routes |
|---------|-------------|---------|------------|
| **Auth** | `handloom-auth-{env}` | Authentication & password management | `/auth/login`, `/auth/refresh`, `/password/*` |
| **User** | `handloom-user-{env}` | Admin user CRUD & role management | `/users`, `/users/{id}` |
| **Catalog** | `handloom-catalog-{env}` | Products, categories, designs | `/categories`, `/designs`, `/products` |
| **Order** | `handloom-order-{env}` | Order & customer management | `/orders`, `/customers` |
| **Pricing** | `handloom-pricing-{env}` | Dynamic pricing engine | `/api/v1/pricing/*`, `/pricing/rules` |
| **Inventory** | `handloom-inventory-{env}` | Stock management | `/inventory`, `/inventory/alerts` |
| **Analytics** | `handloom-analytics-{env}` | Dashboard metrics | `/analytics/*` |
| **Notification** | `handloom-notification-{env}` | User notifications | `/notifications` |
| **Coupon** | `handloom-coupon-{env}` | Coupon management | `/coupons`, `/coupons/apply` |
| **Bulk** | `handloom-bulk-{env}` | Bulk import/export | `/bulk/import`, `/bulk/export` |
| **Asset** | `handloom-asset-{env}` | Media/image management | `/assets`, `/assets/upload` |
| **Report** | `handloom-report-{env}` | Report generation | `/reports` |
| **Audit** | `handloom-audit-{env}` | Audit log access | `/audit`, `/audit/entity/*` |

### Infrastructure Design (AWS CDK)

Infrastructure is defined as code using **AWS CDK for Go**:

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                           AWS CDK STACK ARCHITECTURE                                 │
├─────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                      │
│  ┌───────────────────────────────────────────────────────────────────────────────┐  │
│  │                              CDK App (infra/)                                  │  │
│  │                                                                                │  │
│  │    main.go                                                                    │  │
│  │      │                                                                        │  │
│  │      ├── DatabaseStack (stacks/database.go)                                   │  │
│  │      │     ├── handloom-core-{env}          (DynamoDB, GSIs, PITR)           │  │
│  │      │     ├── PostgreSQL RDS              (Catalog: categories, products)  │  │
│  │      │     ├── handloom-orders-{env}        (DynamoDB, GSIs, PITR)          │  │
│  │      │     ├── handloom-sessions-{env}      (DynamoDB, TTL)                 │  │
│  │      │     ├── handloom-audit-{env}         (DynamoDB, GSIs, 90d TTL)       │  │
│  │      │     ├── handloom-analytics-{env}     (DynamoDB, GSI, 2yr TTL)        │  │
│  │      │     └── handloom-notifications-{env} (DynamoDB)                      │  │
│  │      │                                                                        │  │
│  │      ├── StorageStack (stacks/storage.go)                                     │  │
│  │      │     ├── handloom-assets-{env}   (S3, public, versioned)               │  │
│  │      │     ├── handloom-uploads-{env}  (S3, private, lifecycle)              │  │
│  │      │     └── CloudFront Distribution (OAI, HTTPS, caching)                 │  │
│  │      │                                                                        │  │
│  │      ├── APIStack (stacks/api.go)                                              │  │
│  │      │     ├── JWT Secret (SSM Parameter)                                     │  │
│  │      │     ├── 25 Lambda Functions (ARM64, Go runtime)                        │  │
│  │      │     │     └── IAM Roles with least-privilege access                    │  │
│  │      │     └── API Gateway (REST API)                                         │  │
│  │      │           ├── CORS configuration                                       │  │
│  │      │           ├── Throttling (100 rps, burst 200)                          │  │
│  │      │           └── Route integrations to Lambdas                            │  │
│  │      │                                                                        │  │
│  │      └── EventStack (stacks/events.go)                                        │  │
│  │            ├── SNS Topic (domain events)                                      │  │
│  │            ├── 4 SQS Queues (analytics, audit, notification, report)          │  │
│  │            └── 4 Worker Lambdas (SQS consumers)                               │  │
│  │                                                                                │  │
│  └───────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                      │
│  Deployment Commands:                                                                │
│    make cdk-synth          # Generate CloudFormation                                │
│    make cdk-deploy-dev     # Deploy to development                                  │
│    make cdk-deploy-prod    # Deploy to production                                   │
│    make cdk-destroy        # Tear down infrastructure                               │
│                                                                                      │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

### Application Architecture (Go)

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                        GO APPLICATION LAYER ARCHITECTURE                             │
├─────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                      │
│  ┌───────────────────────────────────────────────────────────────────────────────┐  │
│  │  cmd/lambda/{service}/main.go                                                 │  │
│  │  ┌─────────────────────────────────────────────────────────────────────────┐  │  │
│  │  │                      Lambda Entry Point                                  │  │  │
│  │  │                                                                          │  │  │
│  │  │   1. Load Configuration (config.Load())                                  │  │  │
│  │  │   2. Initialize Logger                                                   │  │  │
│  │  │   3. Initialize DynamoDB Client                                          │  │  │
│  │  │   4. Initialize Repositories                                             │  │  │
│  │  │   5. Initialize Services                                                 │  │  │
│  │  │   6. Initialize Handlers                                                 │  │  │
│  │  │   7. Create Router (chi) with Middleware                                 │  │  │
│  │  │   8. Start Lambda Adapter (aws-lambda-go-api-proxy)                      │  │  │
│  │  │                                                                          │  │  │
│  │  └─────────────────────────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────────────────────┘  │
│                                        │                                             │
│                                        ▼                                             │
│  ┌───────────────────────────────────────────────────────────────────────────────┐  │
│  │  internal/                                                                    │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │  │
│  │  │   config/   │  │  router/    │  │ middleware/ │  │     handler/        │  │  │
│  │  │  ├─config.go│  │ ├─common.go │  │ ├─auth.go   │  │ ├─auth_handler.go   │  │  │
│  │  │  └─Load()   │  │ ├─lambda.go │  │ ├─logging.go│  │ ├─user_handler.go   │  │  │
│  │  │             │  │ ├─health.go │  │ ├─recovery.go│ │ ├─catalog_handler.go│  │  │
│  │  │             │  │ ├─auth.go   │  │ └─cors.go   │  │ ├─order_handler.go  │  │  │
│  │  │             │  │ ├─catalog.go│  │             │  │ └─...               │  │  │
│  │  │             │  │ └─...       │  │             │  │                     │  │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────────────┘  │  │
│  │         │                │                │                    │              │  │
│  │         │                │                │                    ▼              │  │
│  │         │                │                │         ┌─────────────────────┐   │  │
│  │         │                │                │         │     service/        │   │  │
│  │         │                │                │         │ ├─auth_service.go   │   │  │
│  │         │                │                │         │ ├─user_service.go   │   │  │
│  │         │                │                │         │ ├─catalog_service.go│   │  │
│  │         │                │                │         │ ├─pricing_service.go│   │  │
│  │         │                │                │         │ └─...               │   │  │
│  │         │                │                │         └──────────┬──────────┘   │  │
│  │         │                │                │                    │              │  │
│  │         │                │                │                    ▼              │  │
│  │         │                │                │         ┌─────────────────────┐   │  │
│  │         │                │                │         │     domain/         │   │  │
│  │         │                │                │         │ ├─entities (User,   │   │  │
│  │         │                │                │         │ │ Product, Order...)│   │  │
│  │         │                │                │         │ └─interfaces        │   │  │
│  │         │                │                │         │   (repositories)    │   │  │
│  │         │                │                │         └──────────┬──────────┘   │  │
│  │         │                │                │                    │              │  │
│  │         │                │                │                    ▼              │  │
│  │         │                │                │    ┌────────────────────────────┐ │  │
│  │         │                │                │    │  repository/dynamodb/      │ │  │
│  │         │                │                │    │  ├─client.go               │ │  │
│  │         │                │                │    │  ├─user_repository.go      │ │  │
│  │         │                │                │    │  ├─product_repository.go   │ │  │
│  │         │                │                │    │  └─...                     │ │  │
│  │         │                │                │    └────────────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                      │
│  ┌───────────────────────────────────────────────────────────────────────────────┐  │
│  │  pkg/                                                                         │  │
│  │  ├── errors/    - Custom error types (BadRequest, NotFound, Unauthorized)    │  │
│  │  ├── logger/    - Structured logging (zap-based)                             │  │
│  │  └── response/  - HTTP response helpers (JSON, Error)                        │  │
│  └───────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                      │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

### Database Design: Hybrid Multi-Table Approach

We're using a **hybrid multi-table design** that groups related entities together while separating concerns based on access patterns, retention policies, and operational requirements.

#### Table Structure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         DynamoDB Tables Overview                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  handloom-core (DynamoDB)                                             │    │
│  │  ├── Users (Admin, Operator)                                         │    │
│  │  ├── Coupons + CouponUsage                                           │    │
│  │  └── PricingRules                                                    │    │
│  │                                                                       │    │
│  │  PostgreSQL (Catalog)                                                 │    │
│  │  ├── Categories + CategoryAttributes                                  │    │
│  │  ├── Products + ProductImages + ProductAttributeValues                │    │
│  │  └── Inventory + InventoryTransactions                                │    │
│  │                                                                       │    │
│  │  Retention: Permanent | Backup: Daily | GSIs: 3                      │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  handloom-orders                                                     │    │
│  │  ├── Orders (metadata)                                               │    │
│  │  ├── OrderItems                                                      │    │
│  │  ├── OrderStatusHistory                                              │    │
│  │  └── Customers (for B2C later)                                       │    │
│  │                                                                       │    │
│  │  Retention: Permanent | Backup: Daily | GSIs: 3                      │    │
│  │  Note: High write volume, separate for scaling                       │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  handloom-audit                                                      │    │
│  │  └── AuditLogs (all admin actions)                                   │    │
│  │                                                                       │    │
│  │  Retention: 90 days (TTL) | Backup: Weekly | GSIs: 2                 │    │
│  │  Note: Compliance, separate for cost & retention management          │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │  handloom-analytics                                                  │    │
│  │  ├── DailyMetrics                                                    │    │
│  │  ├── WeeklyMetrics                                                   │    │
│  │  ├── MonthlyMetrics                                                  │    │
│  │  ├── TopProducts                                                     │    │
│  │  └── InventoryAlerts                                                 │    │
│  │                                                                       │    │
│  │  Retention: 2 years (TTL) | Backup: Monthly | GSIs: 1                │    │
│  │  Note: Pre-aggregated for fast dashboard queries                     │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### Why This Design?

| Table | Rationale |
|-------|-----------|
| **handloom-core** | Core business entities that need transactional consistency (Product + Inventory updates) |
| **handloom-orders** | High-volume writes during peak sales; separate scaling; Order + Items fetched together |
| **handloom-audit** | Different retention (90 days); compliance requirement; high write, rare read |
| **handloom-analytics** | Pre-aggregated data; different access pattern (read-heavy); can be rebuilt from source |

#### Cross-Table Considerations

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      Cross-Table Operations                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. Order Creation (B2C - future)                                           │
│     ┌──────────────┐    ┌──────────────┐    ┌──────────────┐               │
│     │ handloom-core│───▶│handloom-orders│───▶│handloom-audit│               │
│     │ (Reserve Inv)│    │ (Create Order)│    │ (Log Action) │               │
│     └──────────────┘    └──────────────┘    └──────────────┘               │
│     → Use SQS for eventual consistency OR application-level saga            │
│                                                                              │
│  2. Order Status Update (Admin)                                             │
│     ┌──────────────┐    ┌──────────────┐    ┌──────────────┐               │
│     │handloom-orders│───▶│handloom-audit│───▶│handloom-analytics│           │
│     │(Update Status)│    │ (Log Change) │    │(Update Metrics)│             │
│     └──────────────┘    └──────────────┘    └──────────────┘               │
│     → DynamoDB Streams trigger Lambda for audit & analytics                 │
│                                                                              │
│  3. Product Update                                                          │
│     ┌──────────────┐    ┌──────────────┐                                   │
│     │ handloom-core│───▶│handloom-audit│                                   │
│     │(Update Product)│   │ (Log Change) │                                   │
│     └──────────────┘    └──────────────┘                                   │
│     → Single transaction in core + async audit via Stream                   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### Data Flow Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Data Flow with Streams                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│                              ┌─────────────────┐                            │
│                              │   API Gateway   │                            │
│                              └────────┬────────┘                            │
│                                       │                                      │
│                    ┌──────────────────┼──────────────────┐                  │
│                    ▼                  ▼                  ▼                  │
│           ┌──────────────┐   ┌──────────────┐   ┌──────────────┐           │
│           │ Core Lambda  │   │ Order Lambda │   │ Admin Lambda │           │
│           └──────┬───────┘   └──────┬───────┘   └──────┬───────┘           │
│                  │                  │                  │                    │
│                  ▼                  ▼                  ▼                    │
│           ┌──────────────┐   ┌──────────────┐   ┌──────────────┐           │
│           │handloom-core │   │handloom-orders│  │handloom-audit│           │
│           └──────┬───────┘   └──────┬───────┘   └──────────────┘           │
│                  │                  │                                       │
│                  │ Stream           │ Stream                                │
│                  ▼                  ▼                                       │
│           ┌────────────────────────────────────────────────────┐           │
│           │              Stream Processor Lambda                │           │
│           │  - Writes to handloom-audit (async)                 │           │
│           │  - Aggregates to handloom-analytics                 │           │
│           │  - Triggers notifications (inventory alerts)        │           │
│           └────────────────────────────────────────────────────┘           │
│                                       │                                     │
│                                       ▼                                     │
│                            ┌──────────────────┐                            │
│                            │handloom-analytics│                            │
│                            └──────────────────┘                            │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

### Data Flow Architecture (Detailed View)

> **Note:** This section provides additional detail on data flows. See the [System Architecture Overview](#system-architecture-overview) above for the complete microservices architecture with all 25 Lambda services.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Admin Portal (Frontend)                         │
└─────────────────────────────────────────────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            API Gateway (REST)                                │
│                    - Authentication (JWT validation)                         │
│                    - Rate Limiting                                           │
│                    - Request Routing                                         │
└─────────────────────────────────────────────────────────────────────────────┘
                                        │
          ┌─────────────────────────────┼─────────────────────────────┐
          ▼                             ▼                             ▼
┌───────────────────┐       ┌───────────────────┐       ┌───────────────────┐
│   Auth Lambda     │       │   Core Lambda     │       │   Order Lambda    │
│ - Login/Logout    │       │ - Products        │       │ - Get Orders      │
│ - Password Reset  │       │ - Categories      │       │ - Update Status   │
│ - User Mgmt       │       │ - Inventory       │       │ - Refunds         │
│ - RBAC            │       │ - Coupons         │       │ - Cancellations   │
└─────────┬─────────┘       │ - Assets          │       └─────────┬─────────┘
          │                 │                   │                 │
          │                 └─────────┬─────────┘                 │
          │                           │                           │
          ▼                           ▼                           ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              DynamoDB Tables                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐             │
│  │  handloom-core  │  │ handloom-orders │  │  handloom-audit │             │
│  │  ├─Users        │  │  ├─Orders       │  │  └─AuditLogs    │             │
│  │  ├─PricingRules │  │  ├─OrderItems   │  │    (90-day TTL) │             │
│  │  └─Coupons      │  │  ├─StatusHistory│  └─────────────────┘             │
│  │                  │  │  └─Customers    │                                  │
│  │  PostgreSQL      │  └─────────┬───────┘  ┌─────────────────┐             │
│  │  (Catalog)       │            │          │handloom-analytics│            │
│  │  ├─Categories    │            │          │  ├─DailyMetrics │             │
│  │  ├─Products      │            │          │  ├─TopProducts  │             │
│  └─────────┬───────┘            │          │  └─Alerts       │             │
│            │                    │          └─────────────────┘             │
└────────────┼────────────────────┼──────────────────┼────────────────────────┘
             │                    │                  ▲
             │   DynamoDB Streams │                  │
             └────────────────────┼──────────────────┘
                                  ▼
                    ┌───────────────────────────┐
                    │   Stream Processor Lambda │
                    │   - Audit logging         │
                    │   - Analytics aggregation │
                    │   - Inventory alerts      │
                    └─────────────┬─────────────┘
                                  │
          ┌───────────────────────┼───────────────────────┐
          ▼                       ▼                       ▼
┌───────────────────┐   ┌───────────────────┐   ┌───────────────────┐
│  SQS Queues       │   │  S3 Buckets       │   │  CloudFront CDN   │
│  - Notifications  │   │  - Assets         │   │  - Image delivery │
│  - Bulk Jobs      │   │  - Reports        │   └───────────────────┘
│  - Reports        │   │  - Bulk uploads   │
└─────────┬─────────┘   └───────────────────┘
          │
          ▼
┌───────────────────┐
│  Worker Lambdas   │
│  - Notification   │
│  - Bulk Processor │
│  - Report Gen     │
│  - Image Process  │
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐
│   SES / SNS       │
│   (Email/SMS)     │
└───────────────────┘
```

### Component Breakdown

#### 1. API Gateway
- **Purpose**: Single entry point for all admin API requests
- **Responsibilities**:
  - JWT token validation via Lambda Authorizer
  - Request throttling and rate limiting
  - Route requests to appropriate Lambda functions
  - CORS handling

#### 2. Lambda Functions

> See [Lambda Service Details](#lambda-service-details) above for the complete list of 25 Lambda services (12 admin + 9 store + 4 workers).

#### 3. Database Design (Hybrid)

The system uses a hybrid database architecture: **DynamoDB** for core/transactional data and **PostgreSQL (RDS)** for catalog data requiring relational queries and full-text search.

**DynamoDB Tables (7):**

| Table | Purpose | Entities | TTL |
|-------|---------|----------|-----|
| `handloom-core` | Core business data | Users, PricingRules, Coupons | None |
| `handloom-orders` | Order management | Orders, OrderItems, StatusHistory, Customers, Carts, PriceQuotes | None |
| `handloom-sessions` | Auth sessions | OTPs, Refresh Tokens, Password Resets | TTL-based |
| `handloom-audit` | Compliance logs | AuditLogs | 90 days |
| `handloom-analytics` | Dashboard metrics | DailyMetrics, TopProducts, Alerts, Reports | 2 years |
| `handloom-notifications` | User notifications | Notifications, Templates | None |
| `handloom-events` | Raw tracking events | Frontend tracking events | 30 days |

- On-demand capacity for cost optimization
- Point-in-time recovery on core & orders tables
- Composite PK/SK keys with GSIs for flexible access patterns
- Cursor-based pagination (base64-encoded `ExclusiveStartKey`)

**PostgreSQL (Catalog) — 8 tables:**

Catalog data (categories, products, inventory) lives in PostgreSQL for relational querying, full-text search, and transactional integrity. Schema: `migrations/001_catalog_schema.sql`.

| Table | Purpose | Key Columns |
|-------|---------|-------------|
| `categories` | Product categories | `id`, `name`, `slug` (UNIQUE), `status`, `product_count` |
| `category_attributes` | Dynamic attributes per category | `category_id` (FK), `name`, `type`, `required`, `searchable` |
| `category_attribute_options` | Allowed values per attribute | `attribute_id` (FK), `value`, `label` |
| `products` | Product catalog | `id`, `sku` (UNIQUE), `slug`, `category_id` (FK), `base_price`, `selling_price`, `status` |
| `product_attribute_values` | EAV attribute storage | `product_id` (FK), `attribute_name`, `attribute_value` (composite PK) |
| `product_images` | Product images | `product_id` (FK), `url`, `alt_text`, `sort_order` |
| `inventory` | Stock levels | `product_id` (FK, UNIQUE), `quantity`, `reserved_qty`, `available_qty`, `low_stock_threshold` |
| `inventory_transactions` | Audit trail for stock changes | `product_id` (FK), `type`, `quantity`, `previous_qty`, `new_qty`, `reason` |

**PostgreSQL Key Patterns:**

- **Connection pooling**: `pgxpool` (jackc/pgx v5). Local dev uses `POSTGRES_DSN` directly; Lambda resolves credentials from AWS Secrets Manager (`RDS_SECRET_ARN`) and builds the DSN at startup.
- **Full-text search**: GIN trigram index (`pg_trgm` extension) on `products.name` for fast `ILIKE '%term%'` queries without full table scans.
- **Dynamic attribute filtering**: `EXISTS` subqueries on `product_attribute_values` table (EAV pattern). Hardcoded fields (material, color) also stored as attribute rows for uniform filtering across any attribute combination.
- **Inventory row locking**: `SELECT ... FOR UPDATE` within transactions to prevent race conditions on concurrent stock changes. Every mutation creates an `inventory_transaction` audit record.
- **Transactional writes**: `pgx.BeginFunc` wraps product creation (product + attributes + images + inventory atomically) to ensure consistency.
- **Batch relation loading**: Product lists batch-load attributes and images via `WHERE product_id = ANY($1)` to avoid N+1 queries.
- **Pagination**: Base64-encoded integer offset cursors (different from DynamoDB's `ExclusiveStartKey`). Fetch `LIMIT+1` rows to detect HasMore.
- **In-process caching**: TTL-based go-cache (`internal/cache/`) wraps category (2–5 min) and product (2 min) repos. Prefix-based invalidation on writes. Product list queries are NOT cached (too many filter combinations).
- **Partial index for low stock**: `WHERE available_qty <= low_stock_threshold` for efficient low-stock detection without scanning all inventory rows.

#### 4. SQS Queues
- Decouple notification sending from main request flow
- Enable retry mechanisms for failed notifications
- Dead letter queues for failed messages

---

## Category & Attribute System

### Hierarchical Category Structure

Your product categories follow a hierarchy where attributes are **inherited** from parent to child:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Category Hierarchy Example                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Home Textiles (Root)                                                        │
│  └── Base Attributes: material, color, weave_type, wash_care                 │
│      │                                                                       │
│      ├── Bedding                                                             │
│      │   └── + thread_count, gsm                                             │
│      │       │                                                               │
│      │       ├── Bedsheets                                                   │
│      │       │   └── + bed_size (single/double/king/queen)                   │
│      │       │       + elastic_type (flat/fitted/elasticated)                │
│      │       │       + pillow_cover_included (yes/no)                        │
│      │       │                                                               │
│      │       ├── Pillow Covers                                               │
│      │       │   └── + pillow_size (standard/king/euro)                      │
│      │       │       + closure_type (envelope/zipper/button)                 │
│      │       │                                                               │
│      │       └── Mattress Protectors                                         │
│      │           └── + waterproof (yes/no)                                   │
│      │               + depth_fit                                             │
│      │                                                                       │
│      ├── Curtains & Drapes                                                   │
│      │   └── + opacity (sheer/semi-sheer/blackout)                           │
│      │       + hanging_style (rod_pocket/grommet/tab_top/pinch_pleat)        │
│      │       │                                                               │
│      │       ├── Window Curtains                                             │
│      │       │   └── + lining (lined/unlined)                                │
│      │       │       + tie_backs_included                                    │
│      │       │                                                               │
│      │       └── Door Curtains                                               │
│      │           └── + door_type (main/room/bathroom)                        │
│      │                                                                       │
│      └── Table Linen                                                         │
│          └── + shape (rectangular/square/round/oval)                         │
│              │                                                               │
│              ├── Table Cloths                                                │
│              │   └── + seating_capacity                                      │
│              │                                                               │
│              ├── Table Runners                                               │
│              │   └── + runner_width                                          │
│              │                                                               │
│              └── Placemats                                                   │
│                  └── + set_quantity                                          │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Attribute Inheritance Model

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Attribute Inheritance                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  When you create a product in "Bedsheets" category, it automatically gets:  │
│                                                                              │
│  FROM Root (Home Textiles):                                                  │
│    ├── material (Cotton, Silk, Linen, Polyester, Blend)                     │
│    ├── color                                                                 │
│    ├── weave_type (Plain, Twill, Satin, Jacquard)                           │
│    └── wash_care                                                             │
│                                                                              │
│  FROM Parent (Bedding):                                                      │
│    ├── thread_count (200, 300, 400, 600, 800, 1000)                         │
│    └── gsm (grams per square meter)                                          │
│                                                                              │
│  FROM Own Category (Bedsheets):                                              │
│    ├── bed_size (Single, Double, King, Queen, Custom)                       │
│    ├── elastic_type (Flat, Fitted, Elasticated)                             │
│    └── pillow_cover_included (Yes/No)                                        │
│                                                                              │
│  TOTAL: Product form shows all 10 attributes automatically                   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Attribute Types

| Type | Description | Example |
|------|-------------|---------|
| `SELECT` | Single choice from options | bed_size: Single/Double/King |
| `MULTI_SELECT` | Multiple choices allowed | colors: Red, Blue, Green |
| `TEXT` | Free text input | custom_note |
| `NUMBER` | Numeric value | thread_count: 400 |
| `BOOLEAN` | Yes/No toggle | pillow_cover_included |
| `DIMENSION` | Length/Width/Height with unit | **Custom sizing** |
| `DIMENSION_RANGE` | Min-Max dimension for customization | **Custom sizing** |

### Custom Dimension Attributes

This is your **USP** - allowing customers to specify custom dimensions:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Custom Dimension Configuration                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Attribute: "custom_length"                                                  │
│  Type: DIMENSION_RANGE                                                       │
│  Config:                                                                     │
│    min_value: 60 (inches)                                                    │
│    max_value: 120 (inches)                                                   │
│    step: 1 (increment by 1 inch)                                             │
│    unit: inches                                                              │
│    default_value: 90                                                         │
│                                                                              │
│  Attribute: "custom_width"                                                   │
│  Type: DIMENSION_RANGE                                                       │
│  Config:                                                                     │
│    min_value: 40 (inches)                                                    │
│    max_value: 108 (inches)                                                   │
│    step: 1                                                                   │
│    unit: inches                                                              │
│    default_value: 60                                                         │
│                                                                              │
│  UI shows a slider or input field bounded by min/max                         │
│  Price calculates dynamically based on area (length × width)                 │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Dynamic Pricing Engine

### Pricing Model Overview

The pricing engine calculates product price based on:
1. **Base Price** - Fixed starting price
2. **Dimension-Based Pricing** - Price per unit area/length
3. **Material Multiplier** - Premium materials cost more
4. **Attribute Surcharges** - Extra cost for specific options

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Pricing Calculation Flow                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Customer selects:                                                           │
│    Product: Bedsheet                                                         │
│    Material: Silk                                                            │
│    Length: 100 inches (custom)                                               │
│    Width: 90 inches (custom)                                                 │
│    Thread Count: 600                                                         │
│    Elastic Type: Fitted                                                      │
│                                                                              │
│  Pricing Calculation:                                                        │
│                                                                              │
│  1. Calculate Area                                                           │
│     Area = 100 × 90 = 9000 sq inches = 62.5 sq feet                         │
│                                                                              │
│  2. Base Price (per sq ft for this category)                                │
│     Base Rate = ₹50/sq ft                                                   │
│     Base Cost = 62.5 × 50 = ₹3,125                                          │
│                                                                              │
│  3. Material Multiplier                                                      │
│     Cotton = 1.0x, Silk = 2.5x, Linen = 1.8x                                │
│     Material Cost = 3,125 × 2.5 = ₹7,812.50                                 │
│                                                                              │
│  4. Attribute Surcharges                                                     │
│     Thread Count 600 = +₹500                                                │
│     Fitted Elastic = +₹200                                                  │
│     Subtotal Surcharges = ₹700                                              │
│                                                                              │
│  5. Final Price                                                              │
│     = Material Cost + Surcharges                                            │
│     = ₹7,812.50 + ₹700                                                      │
│     = ₹8,512.50 (rounded to ₹8,513)                                         │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Pricing Rule Hierarchy

Pricing rules can be defined at different levels (most specific wins):

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      Pricing Rule Inheritance                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  Level 1: GLOBAL (fallback)                                                  │
│  ├── Default price per sq ft: ₹30                                           │
│  └── Default material multipliers                                            │
│                                                                              │
│  Level 2: CATEGORY (e.g., Bedding)                                           │
│  ├── Price per sq ft: ₹50                                                   │
│  └── Override material multipliers for bedding                               │
│                                                                              │
│  Level 3: SUBCATEGORY (e.g., Bedsheets)                                      │
│  ├── Price per sq ft: ₹55 (higher quality bedsheets)                        │
│  └── Specific surcharges for bedsheet attributes                             │
│                                                                              │
│  Level 4: PRODUCT (specific product)                                         │
│  ├── Fixed base price: ₹2,000 (premium designer piece)                      │
│  └── Custom multipliers for this product only                                │
│                                                                              │
│  Level 5: MATERIAL (material-specific override)                              │
│  └── Banarasi Silk bedsheets: ₹150/sq ft                                    │
│                                                                              │
│  Resolution: Product > Material > Subcategory > Category > Global            │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Pricing Rule Types

| Rule Type | Description | Example |
|-----------|-------------|---------|
| `AREA_BASED` | Price × (Length × Width) | Bedsheets, Curtains |
| `LENGTH_BASED` | Price × Length | Table Runners |
| `FIXED` | Single fixed price | Pillow Cover (standard size) |
| `TIERED` | Different rates for size ranges | 0-50 sq ft: ₹60/sq ft, 50-100: ₹55/sq ft |
| `FORMULA` | Custom formula | `base + (area × rate) + (perimeter × hem_cost)` |

### Pricing Configuration Schema

```go
// PricingRule defines how to calculate price for a category/product
type PricingRule struct {
    ID              string            `json:"id"`
    PK              string            `json:"-" dynamodbav:"PK"`           // PRICING_RULE#<id>
    SK              string            `json:"-" dynamodbav:"SK"`           // METADATA
    GSI1PK          string            `json:"-" dynamodbav:"GSI1PK"`       // SCOPE#<type>#<id>

    Name            string            `json:"name"`
    Description     string            `json:"description"`

    // Scope - where this rule applies
    ScopeType       string            `json:"scope_type"`   // GLOBAL, CATEGORY, PRODUCT, MATERIAL
    ScopeID         string            `json:"scope_id"`     // category_id, product_id, or material name
    CategoryID      *string           `json:"category_id"`  // If scope is category/product
    MaterialName    *string           `json:"material_name"` // If scope is material-specific

    // Pricing Type
    PricingType     string            `json:"pricing_type"` // AREA_BASED, LENGTH_BASED, FIXED, TIERED, FORMULA

    // Base Pricing
    BasePrice       int64             `json:"base_price"`       // Fixed base price in paise
    PricePerUnit    int64             `json:"price_per_unit"`   // Price per sq inch/inch in paise
    Unit            string            `json:"unit"`             // SQ_INCH, SQ_FOOT, SQ_CM, INCH, CM

    // Material Multipliers
    MaterialMultipliers map[string]float64 `json:"material_multipliers"` // {"Cotton": 1.0, "Silk": 2.5}

    // Attribute Surcharges
    AttributeSurcharges []AttributeSurcharge `json:"attribute_surcharges"`

    // Tiered Pricing (if PricingType = TIERED)
    Tiers           []PricingTier     `json:"tiers"`

    // Formula (if PricingType = FORMULA)
    Formula         string            `json:"formula"` // "base + (area * rate * material_multiplier)"

    // Dimension Constraints
    MinArea         *float64          `json:"min_area"`         // Minimum order area
    MaxArea         *float64          `json:"max_area"`         // Maximum order area
    MinOrderValue   int64             `json:"min_order_value"`  // Minimum order value in paise

    // Status
    Priority        int               `json:"priority"`         // Higher priority wins
    IsActive        bool              `json:"is_active"`
    ValidFrom       *time.Time        `json:"valid_from"`
    ValidUntil      *time.Time        `json:"valid_until"`

    CreatedAt       time.Time         `json:"created_at"`
    UpdatedAt       time.Time         `json:"updated_at"`
    CreatedBy       string            `json:"created_by"`
}

type AttributeSurcharge struct {
    AttributeName   string  `json:"attribute_name"`
    AttributeValue  string  `json:"attribute_value"`
    SurchargeType   string  `json:"surcharge_type"`   // FIXED, PERCENTAGE
    SurchargeValue  int64   `json:"surcharge_value"`  // In paise or percentage × 100
}

type PricingTier struct {
    MinValue      float64 `json:"min_value"`      // Min area/length
    MaxValue      float64 `json:"max_value"`      // Max area/length
    PricePerUnit  int64   `json:"price_per_unit"` // Price for this tier
}
```

### Price Calculation API

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Price Calculation API Flow                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  POST /api/pricing/calculate                                                 │
│                                                                              │
│  Request:                                                                    │
│  {                                                                           │
│    "product_id": "prod_abc123",           // OR category_id for estimates   │
│    "category_id": "cat_bedsheets",        // Required if no product_id      │
│    "dimensions": {                                                           │
│      "length": 100,                       // Custom length                   │
│      "width": 90,                         // Custom width                    │
│      "unit": "inches"                                                        │
│    },                                                                        │
│    "attributes": {                                                           │
│      "material": "Silk",                                                     │
│      "thread_count": "600",                                                  │
│      "elastic_type": "Fitted"                                                │
│    },                                                                        │
│    "quantity": 1                                                             │
│  }                                                                           │
│                                                                              │
│  Response:                                                                   │
│  {                                                                           │
│    "success": true,                                                          │
│    "data": {                                                                 │
│      "breakdown": {                                                          │
│        "area": {                                                             │
│          "value": 62.5,                                                      │
│          "unit": "sq_feet"                                                   │
│        },                                                                    │
│        "base_cost": 312500,               // ₹3,125.00 in paise             │
│        "material_multiplier": 2.5,                                           │
│        "material_cost": 781250,           // ₹7,812.50                      │
│        "surcharges": [                                                       │
│          {"name": "Thread Count 600", "amount": 50000},                     │
│          {"name": "Fitted Elastic", "amount": 20000}                        │
│        ],                                                                    │
│        "total_surcharges": 70000,         // ₹700.00                        │
│        "subtotal": 851250,                // ₹8,512.50                      │
│        "quantity": 1,                                                        │
│        "total": 851250                    // ₹8,512.50                      │
│      },                                                                      │
│      "pricing_rule_applied": "rule_silk_bedsheets",                         │
│      "valid_until": "2024-01-15T23:59:59Z" // Price quote validity          │
│    }                                                                         │
│  }                                                                           │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Pricing Engine Service

```go
// PricingEngine calculates dynamic prices
type PricingEngine interface {
    // Calculate price for given dimensions and attributes
    CalculatePrice(ctx context.Context, req PriceCalculationRequest) (*PriceCalculationResult, error)

    // Get applicable pricing rule for a product/category
    GetApplicablePricingRule(ctx context.Context, productID, categoryID string, material *string) (*PricingRule, error)

    // Validate dimensions are within allowed range
    ValidateDimensions(ctx context.Context, categoryID string, dimensions Dimensions) error

    // Get dimension constraints for a category
    GetDimensionConstraints(ctx context.Context, categoryID string) (*DimensionConstraints, error)
}

// Pricing Rule Management (Admin)
type PricingRuleService interface {
    CreatePricingRule(ctx context.Context, req CreatePricingRuleRequest, userID string) (*PricingRule, error)
    GetPricingRule(ctx context.Context, ruleID string) (*PricingRule, error)
    ListPricingRules(ctx context.Context, req ListPricingRulesRequest) (*ListPricingRulesResponse, error)
    UpdatePricingRule(ctx context.Context, ruleID string, req UpdatePricingRuleRequest) (*PricingRule, error)
    DeletePricingRule(ctx context.Context, ruleID string) error

    // Test pricing rule with sample data
    TestPricingRule(ctx context.Context, ruleID string, testData PriceCalculationRequest) (*PriceCalculationResult, error)

    // Clone rule for different scope
    ClonePricingRule(ctx context.Context, ruleID string, newScope ScopeConfig) (*PricingRule, error)
}
```

---

## Entity Design

### DynamoDB Multi-Table Design

We use 7 tables with related entities grouped together for optimal performance and operational management.

---

### Table 1: `handloom-core`

**Purpose**: Core business entities that need transactional consistency

#### Key Schema
- **PK (Partition Key)**: Entity type + ID (e.g., `PRODUCT#prod_123`)
- **SK (Sort Key)**: Sub-entity or `METADATA`

#### Entity Layout

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           handloom-core Table                                │
├──────────────────┬──────────────────┬────────────────────────────────────────┤
│       PK         │       SK         │              Attributes                │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ USER#<id>        │ PROFILE          │ email, name, role, status, created_at  │
│ USER#<id>        │ SESSION#<token>  │ expires_at, device_info (TTL)          │
│ USER#<id>        │ RESET#<token>    │ expires_at, is_used (TTL)              │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ CATEGORY#<id>    │ METADATA         │ name, slug, parent_id, status          │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ DESIGN#<id>      │ METADATA         │ name, category_id, images, attributes  │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ PRODUCT#<id>     │ METADATA         │ name, sku, design_id, price, artisan_id│
│ PRODUCT#<id>     │ INVENTORY        │ quantity, reserved, threshold          │
│ PRODUCT#<id>     │ INVTXN#<ts>#<id> │ type, qty, reason, reference           │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ ARTISAN#<id>     │ METADATA         │ name, region, craft_types, commission  │
│ ARTISAN#<id>     │ PAYMENT#<ts>#<id>│ amount, status, period, transaction_ref│
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ COUPON#<id>      │ METADATA         │ code, type, value, limits, validity    │
│ COUPON#<id>      │ USAGE#<order_id> │ customer_id, discount_amount, used_at  │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ ASSET#<id>       │ METADATA         │ urls, entity_type, entity_id, status   │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ NOTIFICATION#<id>│ METADATA         │ type, recipient, status, sent_at       │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ BULK_JOB#<id>    │ METADATA         │ type, status, progress, file_url       │
│ BULK_JOB#<id>    │ ERROR#<row>      │ field, value, error_code, message      │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ REPORT#<id>      │ METADATA         │ type, status, parameters, file_url     │
└──────────────────┴──────────────────┴────────────────────────────────────────┘
```

#### Global Secondary Indexes (GSIs)

| GSI Name | PK | SK | Purpose |
|----------|----|----|---------|
| GSI1 | GSI1PK | GSI1SK | Products by category, Designs by category, Artisans by region |
| GSI2 | GSI2PK | GSI2SK | Users by email, Products by SKU, Coupons by code |
| GSI3 | entity_type | created_at | List all entities of a type with date sorting |

---

### Table 2: `handloom-orders`

**Purpose**: Order management with high write volume; separate for independent scaling

#### Key Schema
- **PK (Partition Key)**: Entity type + ID
- **SK (Sort Key)**: Sub-entity or `METADATA`

#### Entity Layout

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          handloom-orders Table                               │
├──────────────────┬──────────────────┬────────────────────────────────────────┤
│       PK         │       SK         │              Attributes                │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ ORDER#<id>       │ METADATA         │ order_number, customer_id, status,     │
│                  │                  │ amounts, addresses, payment_info       │
│ ORDER#<id>       │ ITEM#<product_id>│ quantity, unit_price, total, attributes│
│ ORDER#<id>       │ STATUS#<ts>      │ from_status, to_status, reason, by     │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ CUSTOMER#<id>    │ PROFILE          │ email, name, phone, addresses          │
│ CUSTOMER#<id>    │ ADDRESS#<id>     │ label, line1, line2, city, state, pin  │
└──────────────────┴──────────────────┴────────────────────────────────────────┘
```

#### Global Secondary Indexes (GSIs)

| GSI Name | PK | SK | Purpose |
|----------|----|----|---------|
| GSI1 | GSI1PK | GSI1SK | Orders by status (STATUS#PENDING → created_at) |
| GSI2 | GSI2PK | GSI2SK | Orders by customer (CUSTOMER#id → created_at) |
| GSI3 | entity_type | created_at | List orders by date, customers by date |

---

### Table 3: `handloom-audit`

**Purpose**: Compliance and debugging; separate for retention management

#### Key Schema
- **PK (Partition Key)**: Date-based partitioning (`AUDIT#2024-01-15`)
- **SK (Sort Key)**: Timestamp + ID for ordering

#### Entity Layout

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          handloom-audit Table                                │
├──────────────────┬──────────────────┬────────────────────────────────────────┤
│       PK         │       SK         │              Attributes                │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ AUDIT#<date>     │ <timestamp>#<id> │ user_id, user_email, user_role,        │
│                  │                  │ action, entity_type, entity_id,        │
│                  │                  │ changes, ip_address, user_agent        │
│                  │                  │ TTL (90 days from created_at)          │
└──────────────────┴──────────────────┴────────────────────────────────────────┘
```

#### Global Secondary Indexes (GSIs)

| GSI Name | PK | SK | Purpose |
|----------|----|----|---------|
| GSI1 | GSI1PK | GSI1SK | Audit logs by user (USER#id → timestamp) |
| GSI2 | GSI2PK | GSI2SK | Audit logs by entity (ENTITY#type#id → timestamp) |

#### TTL Configuration
- **TTL Attribute**: `ttl` (Unix timestamp, 90 days from creation)
- Automatically deletes old audit logs to manage storage costs

---

### Table 4: `handloom-analytics`

**Purpose**: Pre-aggregated metrics for fast dashboard queries

#### Key Schema
- **PK (Partition Key)**: Metric type
- **SK (Sort Key)**: Time period or identifier

#### Entity Layout

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        handloom-analytics Table                              │
├──────────────────┬──────────────────┬────────────────────────────────────────┤
│       PK         │       SK         │              Attributes                │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ METRICS#DAILY    │ <date>           │ orders, revenue, items_sold, refunds,  │
│                  │ (2024-01-15)     │ new_customers, avg_order_value         │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ METRICS#WEEKLY   │ <year-week>      │ (same as daily, aggregated weekly)     │
│                  │ (2024-W03)       │                                        │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ METRICS#MONTHLY  │ <year-month>     │ (same as daily, aggregated monthly)    │
│                  │ (2024-01)        │                                        │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ TOP_PRODUCTS     │ <period>#<rank>  │ product_id, name, sku, quantity,       │
│                  │ (DAILY#2024-01-15#1) │ revenue, order_count              │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ TOP_ARTISANS     │ <period>#<rank>  │ artisan_id, name, products_sold,       │
│                  │ (MONTHLY#2024-01#1)  │ revenue, orders                   │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ INVENTORY_ALERT  │ PRODUCT#<id>     │ product_name, sku, alert_type,         │
│                  │                  │ current_qty, threshold, created_at     │
├──────────────────┼──────────────────┼────────────────────────────────────────┤
│ ORDER_FUNNEL     │ <date>           │ pending, confirmed, processing,        │
│                  │                  │ shipped, delivered, cancelled          │
└──────────────────┴──────────────────┴────────────────────────────────────────┘
```

#### Global Secondary Indexes (GSIs)

| GSI Name | PK | SK | Purpose |
|----------|----|----|---------|
| GSI1 | alert_type | created_at | Get all LOW_STOCK or OUT_OF_STOCK alerts |

#### TTL Configuration
- **TTL Attribute**: `ttl` (2 years for metrics, can be rebuilt from source tables)

### Entity Definitions (Go Structs)

```go
// ==================== TABLE NAMES & CONFIGURATION ====================

// Table names - loaded from environment in production
const (
    TableCore      = "handloom-core"      // Users, Products, Categories, etc.
    TableOrders    = "handloom-orders"    // Orders, OrderItems, Customers
    TableAudit     = "handloom-audit"     // AuditLogs
    TableAnalytics = "handloom-analytics" // Metrics, TopProducts, Alerts
)

// TableConfig holds configuration for each table
type TableConfig struct {
    Name           string
    HasStreams     bool
    PointInTimeRecovery bool
    TTLAttribute   string
}

var TableConfigs = map[string]TableConfig{
    TableCore: {
        Name:                TableCore,
        HasStreams:          true,
        PointInTimeRecovery: true,
        TTLAttribute:        "ttl", // For sessions only
    },
    TableOrders: {
        Name:                TableOrders,
        HasStreams:          true,
        PointInTimeRecovery: true,
        TTLAttribute:        "",
    },
    TableAudit: {
        Name:                TableAudit,
        HasStreams:          false,
        PointInTimeRecovery: false,
        TTLAttribute:        "ttl", // 90-day retention
    },
    TableAnalytics: {
        Name:                TableAnalytics,
        HasStreams:          false,
        PointInTimeRecovery: false,
        TTLAttribute:        "ttl", // 2-year retention
    },
}

// ==================== USER ENTITIES (Table: handloom-core) ====================

type UserRole string

const (
    RoleAdmin    UserRole = "ADMIN"
    RoleOperator UserRole = "OPERATOR"
)

type UserStatus string

const (
    UserStatusActive   UserStatus = "ACTIVE"
    UserStatusInactive UserStatus = "INACTIVE"
    UserStatusSuspended UserStatus = "SUSPENDED"
)

// User stored in handloom-core table
type User struct {
    ID           string     `json:"id" dynamodbav:"id"`
    PK           string     `json:"-" dynamodbav:"PK"`           // USER#<id>
    SK           string     `json:"-" dynamodbav:"SK"`           // PROFILE
    GSI1PK       string     `json:"-" dynamodbav:"GSI1PK"`       // ROLE#<role>
    GSI2PK       string     `json:"-" dynamodbav:"GSI2PK"`       // EMAIL#<email>
    EntityType   string     `json:"-" dynamodbav:"entity_type"`  // USER

    Email        string     `json:"email" dynamodbav:"email"`
    Name         string     `json:"name" dynamodbav:"name"`
    PasswordHash string     `json:"-" dynamodbav:"password_hash"`
    Role         UserRole   `json:"role" dynamodbav:"role"`
    Status       UserStatus `json:"status" dynamodbav:"status"`
    Permissions  []string   `json:"permissions" dynamodbav:"permissions"`

    CreatedAt    time.Time  `json:"created_at" dynamodbav:"created_at"`
    UpdatedAt    time.Time  `json:"updated_at" dynamodbav:"updated_at"`
    CreatedBy    string     `json:"created_by" dynamodbav:"created_by"`
}

// TableName returns the DynamoDB table for User
func (u User) TableName() string { return TableCore }

// ==================== CATEGORY ENTITIES (Table: handloom-core) ====================

type CategoryStatus string

const (
    CategoryStatusActive   CategoryStatus = "ACTIVE"
    CategoryStatusInactive CategoryStatus = "INACTIVE"
)

// Category stored in handloom-core table - supports hierarchy and attribute inheritance
type Category struct {
    ID          string         `json:"id" dynamodbav:"id"`
    PK          string         `json:"-" dynamodbav:"PK"`           // CATEGORY#<id>
    SK          string         `json:"-" dynamodbav:"SK"`           // METADATA
    GSI1PK      string         `json:"-" dynamodbav:"GSI1PK"`       // PARENT#<parent_id> or PARENT#ROOT
    GSI2PK      string         `json:"-" dynamodbav:"GSI2PK"`       // SLUG#<slug>
    EntityType  string         `json:"-" dynamodbav:"entity_type"`  // CATEGORY

    Name        string         `json:"name" dynamodbav:"name"`
    Slug        string         `json:"slug" dynamodbav:"slug"`
    Description string         `json:"description" dynamodbav:"description"`
    ParentID    *string        `json:"parent_id" dynamodbav:"parent_id"`

    // Hierarchy info
    Level       int            `json:"level" dynamodbav:"level"`           // 0 = root, 1 = child, etc.
    Path        string         `json:"path" dynamodbav:"path"`             // /root_id/parent_id/this_id
    AncestorIDs []string       `json:"ancestor_ids" dynamodbav:"ancestor_ids"` // [root_id, parent_id]

    ImageURL    string         `json:"image_url" dynamodbav:"image_url"`
    Status      CategoryStatus `json:"status" dynamodbav:"status"`
    SortOrder   int            `json:"sort_order" dynamodbav:"sort_order"`

    // Category-specific attributes (inherited by products)
    // These are ADDED to parent's attributes - not replaced
    OwnAttributes []CategoryAttribute `json:"own_attributes" dynamodbav:"own_attributes"`

    // Customization settings
    AllowCustomDimensions bool               `json:"allow_custom_dimensions" dynamodbav:"allow_custom_dimensions"`
    DimensionConfig       *DimensionConfig   `json:"dimension_config" dynamodbav:"dimension_config"`

    // Pricing configuration
    DefaultPricingRuleID  *string            `json:"default_pricing_rule_id" dynamodbav:"default_pricing_rule_id"`

    CreatedAt   time.Time      `json:"created_at" dynamodbav:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at" dynamodbav:"updated_at"`
    CreatedBy   string         `json:"created_by" dynamodbav:"created_by"`
}

func (c Category) TableName() string { return TableCore }

// CategoryAttribute defines an attribute that products in this category must/can have
type CategoryAttribute struct {
    Name          string           `json:"name" dynamodbav:"name"`                     // e.g., "material", "thread_count"
    DisplayName   string           `json:"display_name" dynamodbav:"display_name"`     // e.g., "Material", "Thread Count"
    Type          AttributeType    `json:"type" dynamodbav:"type"`                     // SELECT, MULTI_SELECT, TEXT, NUMBER, etc.
    Required      bool             `json:"required" dynamodbav:"required"`
    Options       []AttributeOption `json:"options" dynamodbav:"options"`              // For SELECT/MULTI_SELECT
    DefaultValue  *string          `json:"default_value" dynamodbav:"default_value"`

    // For NUMBER type
    MinValue      *float64         `json:"min_value" dynamodbav:"min_value"`
    MaxValue      *float64         `json:"max_value" dynamodbav:"max_value"`
    Step          *float64         `json:"step" dynamodbav:"step"`
    Unit          string           `json:"unit" dynamodbav:"unit"`

    // Display settings
    ShowInFilters bool             `json:"show_in_filters" dynamodbav:"show_in_filters"`
    ShowInListing bool             `json:"show_in_listing" dynamodbav:"show_in_listing"`
    SortOrder     int              `json:"sort_order" dynamodbav:"sort_order"`

    // Pricing impact
    AffectsPricing bool            `json:"affects_pricing" dynamodbav:"affects_pricing"`
}

type AttributeType string

const (
    AttributeTypeSelect       AttributeType = "SELECT"
    AttributeTypeMultiSelect  AttributeType = "MULTI_SELECT"
    AttributeTypeText         AttributeType = "TEXT"
    AttributeTypeNumber       AttributeType = "NUMBER"
    AttributeTypeBoolean      AttributeType = "BOOLEAN"
    AttributeTypeDimension    AttributeType = "DIMENSION"
    AttributeTypeDimensionRange AttributeType = "DIMENSION_RANGE"
)

type AttributeOption struct {
    Value       string  `json:"value" dynamodbav:"value"`
    Label       string  `json:"label" dynamodbav:"label"`
    SortOrder   int     `json:"sort_order" dynamodbav:"sort_order"`
    IsDefault   bool    `json:"is_default" dynamodbav:"is_default"`
    // For pricing
    Surcharge   int64   `json:"surcharge" dynamodbav:"surcharge"`           // Additional cost in paise
    Multiplier  float64 `json:"multiplier" dynamodbav:"multiplier"`         // Price multiplier (1.0 = no change)
}

// DimensionConfig defines custom dimension rules for a category
type DimensionConfig struct {
    // Length configuration
    LengthEnabled   bool    `json:"length_enabled" dynamodbav:"length_enabled"`
    LengthMin       float64 `json:"length_min" dynamodbav:"length_min"`
    LengthMax       float64 `json:"length_max" dynamodbav:"length_max"`
    LengthStep      float64 `json:"length_step" dynamodbav:"length_step"`
    LengthDefault   float64 `json:"length_default" dynamodbav:"length_default"`
    LengthUnit      string  `json:"length_unit" dynamodbav:"length_unit"`     // inches, cm, feet

    // Width configuration
    WidthEnabled    bool    `json:"width_enabled" dynamodbav:"width_enabled"`
    WidthMin        float64 `json:"width_min" dynamodbav:"width_min"`
    WidthMax        float64 `json:"width_max" dynamodbav:"width_max"`
    WidthStep       float64 `json:"width_step" dynamodbav:"width_step"`
    WidthDefault    float64 `json:"width_default" dynamodbav:"width_default"`
    WidthUnit       string  `json:"width_unit" dynamodbav:"width_unit"`

    // Height configuration (for 3D products like cushion covers)
    HeightEnabled   bool    `json:"height_enabled" dynamodbav:"height_enabled"`
    HeightMin       float64 `json:"height_min" dynamodbav:"height_min"`
    HeightMax       float64 `json:"height_max" dynamodbav:"height_max"`
    HeightStep      float64 `json:"height_step" dynamodbav:"height_step"`
    HeightDefault   float64 `json:"height_default" dynamodbav:"height_default"`
    HeightUnit      string  `json:"height_unit" dynamodbav:"height_unit"`

    // Pricing model
    PricingModel    string  `json:"pricing_model" dynamodbav:"pricing_model"` // AREA_BASED, LENGTH_BASED, VOLUME_BASED
}

// ==================== PRICING RULE ENTITIES (Table: handloom-core) ====================

type PricingRuleScope string

const (
    PricingRuleScopeGlobal   PricingRuleScope = "GLOBAL"
    PricingRuleScopeCategory PricingRuleScope = "CATEGORY"
    PricingRuleScopeProduct  PricingRuleScope = "PRODUCT"
    PricingRuleScopeMaterial PricingRuleScope = "MATERIAL"
)

type PricingType string

const (
    PricingTypeAreaBased   PricingType = "AREA_BASED"
    PricingTypeLengthBased PricingType = "LENGTH_BASED"
    PricingTypeFixed       PricingType = "FIXED"
    PricingTypeTiered      PricingType = "TIERED"
    PricingTypeFormula     PricingType = "FORMULA"
)

// PricingRule defines how to calculate price for a category/product
type PricingRule struct {
    ID              string            `json:"id" dynamodbav:"id"`
    PK              string            `json:"-" dynamodbav:"PK"`           // PRICING_RULE#<id>
    SK              string            `json:"-" dynamodbav:"SK"`           // METADATA
    GSI1PK          string            `json:"-" dynamodbav:"GSI1PK"`       // SCOPE#<type>#<id>
    EntityType      string            `json:"-" dynamodbav:"entity_type"`  // PRICING_RULE

    Name            string            `json:"name" dynamodbav:"name"`
    Description     string            `json:"description" dynamodbav:"description"`

    // Scope - where this rule applies
    ScopeType       PricingRuleScope  `json:"scope_type" dynamodbav:"scope_type"`
    ScopeID         string            `json:"scope_id" dynamodbav:"scope_id"`
    CategoryID      *string           `json:"category_id" dynamodbav:"category_id"`
    ProductID       *string           `json:"product_id" dynamodbav:"product_id"`
    MaterialName    *string           `json:"material_name" dynamodbav:"material_name"`

    // Pricing Type
    PricingType     PricingType       `json:"pricing_type" dynamodbav:"pricing_type"`

    // Base Pricing
    BasePrice       int64             `json:"base_price" dynamodbav:"base_price"`
    PricePerUnit    int64             `json:"price_per_unit" dynamodbav:"price_per_unit"`
    Unit            string            `json:"unit" dynamodbav:"unit"`

    // Material Multipliers
    MaterialMultipliers map[string]float64 `json:"material_multipliers" dynamodbav:"material_multipliers"`

    // Attribute Surcharges
    AttributeSurcharges []PricingAttributeSurcharge `json:"attribute_surcharges" dynamodbav:"attribute_surcharges"`

    // Tiered Pricing
    Tiers           []PricingTier     `json:"tiers" dynamodbav:"tiers"`

    // Custom Formula
    Formula         string            `json:"formula" dynamodbav:"formula"`

    // Constraints
    MinArea         *float64          `json:"min_area" dynamodbav:"min_area"`
    MaxArea         *float64          `json:"max_area" dynamodbav:"max_area"`
    MinOrderValue   int64             `json:"min_order_value" dynamodbav:"min_order_value"`

    // Priority & Status
    Priority        int               `json:"priority" dynamodbav:"priority"`
    IsActive        bool              `json:"is_active" dynamodbav:"is_active"`
    ValidFrom       *time.Time        `json:"valid_from" dynamodbav:"valid_from"`
    ValidUntil      *time.Time        `json:"valid_until" dynamodbav:"valid_until"`

    CreatedAt       time.Time         `json:"created_at" dynamodbav:"created_at"`
    UpdatedAt       time.Time         `json:"updated_at" dynamodbav:"updated_at"`
    CreatedBy       string            `json:"created_by" dynamodbav:"created_by"`
}

func (pr PricingRule) TableName() string { return TableCore }

type PricingAttributeSurcharge struct {
    AttributeName   string  `json:"attribute_name" dynamodbav:"attribute_name"`
    AttributeValue  string  `json:"attribute_value" dynamodbav:"attribute_value"`
    SurchargeType   string  `json:"surcharge_type" dynamodbav:"surcharge_type"`     // FIXED, PERCENTAGE
    SurchargeValue  int64   `json:"surcharge_value" dynamodbav:"surcharge_value"`
}

type PricingTier struct {
    MinValue      float64 `json:"min_value" dynamodbav:"min_value"`
    MaxValue      float64 `json:"max_value" dynamodbav:"max_value"`
    PricePerUnit  int64   `json:"price_per_unit" dynamodbav:"price_per_unit"`
}

// ==================== DESIGN ENTITIES (Table: handloom-core) ====================

type DesignStatus string

const (
    DesignStatusActive   DesignStatus = "ACTIVE"
    DesignStatusInactive DesignStatus = "INACTIVE"
    DesignStatusDraft    DesignStatus = "DRAFT"
)

// Design stored in handloom-core table
type Design struct {
    ID          string       `json:"id" dynamodbav:"id"`
    PK          string       `json:"-" dynamodbav:"PK"`           // DESIGN#<id>
    SK          string       `json:"-" dynamodbav:"SK"`           // METADATA
    GSI1PK      string       `json:"-" dynamodbav:"GSI1PK"`       // CATEGORY#<category_id>
    EntityType  string       `json:"-" dynamodbav:"entity_type"`  // DESIGN

    Name        string       `json:"name" dynamodbav:"name"`
    Slug        string       `json:"slug" dynamodbav:"slug"`
    CategoryID  string       `json:"category_id" dynamodbav:"category_id"`
    Description string       `json:"description" dynamodbav:"description"`
    Images      []ImageInfo  `json:"images" dynamodbav:"images"`
    Attributes  []Attribute  `json:"attributes" dynamodbav:"attributes"` // Color, Pattern, etc.
    Status      DesignStatus `json:"status" dynamodbav:"status"`

    CreatedAt   time.Time    `json:"created_at" dynamodbav:"created_at"`
    UpdatedAt   time.Time    `json:"updated_at" dynamodbav:"updated_at"`
    CreatedBy   string       `json:"created_by" dynamodbav:"created_by"`
}

func (d Design) TableName() string { return TableCore }

type ImageInfo struct {
    URL       string `json:"url" dynamodbav:"url"`
    AltText   string `json:"alt_text" dynamodbav:"alt_text"`
    IsPrimary bool   `json:"is_primary" dynamodbav:"is_primary"`
    SortOrder int    `json:"sort_order" dynamodbav:"sort_order"`
}

type Attribute struct {
    Name   string   `json:"name" dynamodbav:"name"`
    Values []string `json:"values" dynamodbav:"values"`
}

// ==================== PRODUCT ENTITIES (Table: handloom-core) ====================

type ProductStatus string

const (
    ProductStatusActive       ProductStatus = "ACTIVE"
    ProductStatusInactive     ProductStatus = "INACTIVE"
    ProductStatusOutOfStock   ProductStatus = "OUT_OF_STOCK"
    ProductStatusDiscontinued ProductStatus = "DISCONTINUED"
)

type Product struct {
    ID           string            `json:"id" dynamodbav:"id"`
    PK           string            `json:"-" dynamodbav:"PK"`           // PRODUCT#<id>
    SK           string            `json:"-" dynamodbav:"SK"`           // METADATA
    GSI1PK       string            `json:"-" dynamodbav:"GSI1PK"`       // DESIGN#<design_id>
    GSI2PK       string            `json:"-" dynamodbav:"GSI2PK"`       // SKU#<sku>
    EntityType   string            `json:"-" dynamodbav:"entity_type"`  // PRODUCT

    SKU          string            `json:"sku" dynamodbav:"sku"`
    Name         string            `json:"name" dynamodbav:"name"`
    DesignID     string            `json:"design_id" dynamodbav:"design_id"`
    CategoryID   string            `json:"category_id" dynamodbav:"category_id"`

    // Pricing
    BasePrice    int64             `json:"base_price" dynamodbav:"base_price"`       // In paise
    SellingPrice int64             `json:"selling_price" dynamodbav:"selling_price"` // In paise
    CostPrice    int64             `json:"cost_price" dynamodbav:"cost_price"`       // In paise
    Currency     string            `json:"currency" dynamodbav:"currency"`           // INR

    // Product Attributes (specific variant details)
    Dimensions   Dimensions        `json:"dimensions" dynamodbav:"dimensions"`
    Weight       int               `json:"weight" dynamodbav:"weight"`       // In grams
    Material     string            `json:"material" dynamodbav:"material"`
    Color        string            `json:"color" dynamodbav:"color"`
    Size         string            `json:"size" dynamodbav:"size"`

    // Handloom Specific
    WeavingType  string            `json:"weaving_type" dynamodbav:"weaving_type"`
    Origin       string            `json:"origin" dynamodbav:"origin"`        // Region/State
    Artisan      string            `json:"artisan" dynamodbav:"artisan"`      // Artisan/Weaver name
    CraftType    string            `json:"craft_type" dynamodbav:"craft_type"` // Banarasi, Chanderi, etc.

    Images       []ImageInfo       `json:"images" dynamodbav:"images"`
    Tags         []string          `json:"tags" dynamodbav:"tags"`
    Status       ProductStatus     `json:"status" dynamodbav:"status"`

    CreatedAt    time.Time         `json:"created_at" dynamodbav:"created_at"`
    UpdatedAt    time.Time         `json:"updated_at" dynamodbav:"updated_at"`
    CreatedBy    string            `json:"created_by" dynamodbav:"created_by"`
}

type Dimensions struct {
    Length float64 `json:"length" dynamodbav:"length"` // In cm
    Width  float64 `json:"width" dynamodbav:"width"`   // In cm
    Height float64 `json:"height" dynamodbav:"height"` // In cm
}

// ==================== INVENTORY ENTITIES ====================

type Inventory struct {
    PK              string    `json:"-" dynamodbav:"PK"`           // PRODUCT#<product_id>
    SK              string    `json:"-" dynamodbav:"SK"`           // INVENTORY
    EntityType      string    `json:"-" dynamodbav:"entity_type"`  // INVENTORY

    ProductID       string    `json:"product_id" dynamodbav:"product_id"`
    Quantity        int       `json:"quantity" dynamodbav:"quantity"`
    ReservedQty     int       `json:"reserved_qty" dynamodbav:"reserved_qty"`
    AvailableQty    int       `json:"available_qty" dynamodbav:"available_qty"` // quantity - reserved

    LowStockThreshold int     `json:"low_stock_threshold" dynamodbav:"low_stock_threshold"`
    ReorderPoint      int     `json:"reorder_point" dynamodbav:"reorder_point"`

    LastRestockAt   *time.Time `json:"last_restock_at" dynamodbav:"last_restock_at"`
    UpdatedAt       time.Time  `json:"updated_at" dynamodbav:"updated_at"`
    UpdatedBy       string     `json:"updated_by" dynamodbav:"updated_by"`
}

type InventoryTransaction struct {
    ID            string    `json:"id" dynamodbav:"id"`
    PK            string    `json:"-" dynamodbav:"PK"`           // PRODUCT#<product_id>
    SK            string    `json:"-" dynamodbav:"SK"`           // INVTXN#<timestamp>#<id>
    EntityType    string    `json:"-" dynamodbav:"entity_type"`  // INVENTORY_TXN

    ProductID     string    `json:"product_id" dynamodbav:"product_id"`
    Type          string    `json:"type" dynamodbav:"type"`       // ADD, REMOVE, RESERVE, RELEASE
    Quantity      int       `json:"quantity" dynamodbav:"quantity"`
    PreviousQty   int       `json:"previous_qty" dynamodbav:"previous_qty"`
    NewQty        int       `json:"new_qty" dynamodbav:"new_qty"`
    Reason        string    `json:"reason" dynamodbav:"reason"`
    ReferenceType string    `json:"reference_type" dynamodbav:"reference_type"` // ORDER, MANUAL, ADJUSTMENT
    ReferenceID   string    `json:"reference_id" dynamodbav:"reference_id"`

    CreatedAt     time.Time `json:"created_at" dynamodbav:"created_at"`
    CreatedBy     string    `json:"created_by" dynamodbav:"created_by"`
}

// ==================== ORDER ENTITIES (Table: handloom-orders) ====================

type OrderStatus string

const (
    OrderStatusPending          OrderStatus = "PENDING"
    OrderStatusConfirmed        OrderStatus = "CONFIRMED"
    OrderStatusProcessing       OrderStatus = "PROCESSING"
    OrderStatusShipped          OrderStatus = "SHIPPED"
    OrderStatusOutForDelivery   OrderStatus = "OUT_FOR_DELIVERY"
    OrderStatusDelivered        OrderStatus = "DELIVERED"
    OrderStatusCancelled        OrderStatus = "CANCELLED"
    OrderStatusRefundInitiated  OrderStatus = "REFUND_INITIATED"
    OrderStatusRefundCompleted  OrderStatus = "REFUND_COMPLETED"
    OrderStatusReturnRequested  OrderStatus = "RETURN_REQUESTED"
    OrderStatusReturnApproved   OrderStatus = "RETURN_APPROVED"
    OrderStatusReturnReceived   OrderStatus = "RETURN_RECEIVED"
)

type PaymentStatus string

const (
    PaymentStatusPending   PaymentStatus = "PENDING"
    PaymentStatusPaid      PaymentStatus = "PAID"
    PaymentStatusFailed    PaymentStatus = "FAILED"
    PaymentStatusRefunded  PaymentStatus = "REFUNDED"
    PaymentStatusPartial   PaymentStatus = "PARTIAL_REFUND"
)

// Order stored in handloom-orders table
type Order struct {
    ID              string        `json:"id" dynamodbav:"id"`
    PK              string        `json:"-" dynamodbav:"PK"`           // ORDER#<id>
    SK              string        `json:"-" dynamodbav:"SK"`           // METADATA
    GSI1PK          string        `json:"-" dynamodbav:"GSI1PK"`       // STATUS#<status>
    GSI1SK          string        `json:"-" dynamodbav:"GSI1SK"`       // <created_at>
    GSI2PK          string        `json:"-" dynamodbav:"GSI2PK"`       // CUSTOMER#<customer_id>
    GSI2SK          string        `json:"-" dynamodbav:"GSI2SK"`       // <created_at>
    EntityType      string        `json:"-" dynamodbav:"entity_type"`  // ORDER

    OrderNumber     string        `json:"order_number" dynamodbav:"order_number"` // Human readable
    CustomerID      string        `json:"customer_id" dynamodbav:"customer_id"`
    CustomerEmail   string        `json:"customer_email" dynamodbav:"customer_email"`
    CustomerName    string        `json:"customer_name" dynamodbav:"customer_name"`
    CustomerPhone   string        `json:"customer_phone" dynamodbav:"customer_phone"`

    // Addresses
    ShippingAddress Address       `json:"shipping_address" dynamodbav:"shipping_address"`
    BillingAddress  Address       `json:"billing_address" dynamodbav:"billing_address"`

    // Amounts (all in paise)
    Subtotal        int64         `json:"subtotal" dynamodbav:"subtotal"`
    DiscountAmount  int64         `json:"discount_amount" dynamodbav:"discount_amount"`
    TaxAmount       int64         `json:"tax_amount" dynamodbav:"tax_amount"`
    ShippingAmount  int64         `json:"shipping_amount" dynamodbav:"shipping_amount"`
    TotalAmount     int64         `json:"total_amount" dynamodbav:"total_amount"`
    Currency        string        `json:"currency" dynamodbav:"currency"`

    // Status
    Status          OrderStatus   `json:"status" dynamodbav:"status"`
    PaymentStatus   PaymentStatus `json:"payment_status" dynamodbav:"payment_status"`
    PaymentMethod   string        `json:"payment_method" dynamodbav:"payment_method"`
    PaymentID       string        `json:"payment_id" dynamodbav:"payment_id"`

    // Shipping
    ShippingMethod  string        `json:"shipping_method" dynamodbav:"shipping_method"`
    TrackingNumber  string        `json:"tracking_number" dynamodbav:"tracking_number"`
    ShippingCarrier string        `json:"shipping_carrier" dynamodbav:"shipping_carrier"`

    // Coupon/Discount
    CouponCode      string        `json:"coupon_code" dynamodbav:"coupon_code"`

    // Notes
    CustomerNote    string        `json:"customer_note" dynamodbav:"customer_note"`
    AdminNote       string        `json:"admin_note" dynamodbav:"admin_note"`

    // Timestamps
    CreatedAt       time.Time     `json:"created_at" dynamodbav:"created_at"`
    UpdatedAt       time.Time     `json:"updated_at" dynamodbav:"updated_at"`
    ConfirmedAt     *time.Time    `json:"confirmed_at" dynamodbav:"confirmed_at"`
    ShippedAt       *time.Time    `json:"shipped_at" dynamodbav:"shipped_at"`
    DeliveredAt     *time.Time    `json:"delivered_at" dynamodbav:"delivered_at"`
    CancelledAt     *time.Time    `json:"cancelled_at" dynamodbav:"cancelled_at"`

    ItemCount       int           `json:"item_count" dynamodbav:"item_count"`
}

func (o Order) TableName() string { return TableOrders }

type Address struct {
    Name        string `json:"name" dynamodbav:"name"`
    Line1       string `json:"line1" dynamodbav:"line1"`
    Line2       string `json:"line2" dynamodbav:"line2"`
    City        string `json:"city" dynamodbav:"city"`
    State       string `json:"state" dynamodbav:"state"`
    PinCode     string `json:"pin_code" dynamodbav:"pin_code"`
    Country     string `json:"country" dynamodbav:"country"`
    Phone       string `json:"phone" dynamodbav:"phone"`
}

// OrderItem stored in handloom-orders table (same PK as Order for efficient queries)
type OrderItem struct {
    PK            string    `json:"-" dynamodbav:"PK"`           // ORDER#<order_id>
    SK            string    `json:"-" dynamodbav:"SK"`           // ITEM#<product_id>
    EntityType    string    `json:"-" dynamodbav:"entity_type"`  // ORDER_ITEM

    OrderID       string    `json:"order_id" dynamodbav:"order_id"`
    ProductID     string    `json:"product_id" dynamodbav:"product_id"`
    ProductName   string    `json:"product_name" dynamodbav:"product_name"`
    ProductSKU    string    `json:"product_sku" dynamodbav:"product_sku"`
    ProductImage  string    `json:"product_image" dynamodbav:"product_image"`

    Quantity      int       `json:"quantity" dynamodbav:"quantity"`
    UnitPrice     int64     `json:"unit_price" dynamodbav:"unit_price"`     // In paise
    TotalPrice    int64     `json:"total_price" dynamodbav:"total_price"`   // In paise

    // For variants
    Attributes    map[string]string `json:"attributes" dynamodbav:"attributes"`
}

func (oi OrderItem) TableName() string { return TableOrders }

type OrderStatusHistory struct {
    PK          string    `json:"-" dynamodbav:"PK"`           // ORDER#<order_id>
    SK          string    `json:"-" dynamodbav:"SK"`           // STATUS#<timestamp>
    EntityType  string    `json:"-" dynamodbav:"entity_type"`  // ORDER_STATUS

    OrderID     string    `json:"order_id" dynamodbav:"order_id"`
    FromStatus  string    `json:"from_status" dynamodbav:"from_status"`
    ToStatus    string    `json:"to_status" dynamodbav:"to_status"`
    Reason      string    `json:"reason" dynamodbav:"reason"`
    Note        string    `json:"note" dynamodbav:"note"`

    ChangedBy   string    `json:"changed_by" dynamodbav:"changed_by"`
    ChangedAt   time.Time `json:"changed_at" dynamodbav:"changed_at"`
}

// ==================== NOTIFICATION ENTITIES ====================

type NotificationType string

const (
    NotificationTypeEmail NotificationType = "EMAIL"
    NotificationTypeSMS   NotificationType = "SMS"
    NotificationTypePush  NotificationType = "PUSH"
)

type NotificationStatus string

const (
    NotificationStatusPending   NotificationStatus = "PENDING"
    NotificationStatusSent      NotificationStatus = "SENT"
    NotificationStatusFailed    NotificationStatus = "FAILED"
    NotificationStatusDelivered NotificationStatus = "DELIVERED"
)

type Notification struct {
    ID            string             `json:"id" dynamodbav:"id"`
    PK            string             `json:"-" dynamodbav:"PK"`           // NOTIFICATION#<id>
    SK            string             `json:"-" dynamodbav:"SK"`           // METADATA
    GSI1PK        string             `json:"-" dynamodbav:"GSI1PK"`       // RECIPIENT#<customer_id>
    GSI1SK        string             `json:"-" dynamodbav:"GSI1SK"`       // <created_at>
    EntityType    string             `json:"-" dynamodbav:"entity_type"`  // NOTIFICATION

    Type          NotificationType   `json:"type" dynamodbav:"type"`
    RecipientID   string             `json:"recipient_id" dynamodbav:"recipient_id"`
    RecipientEmail string            `json:"recipient_email" dynamodbav:"recipient_email"`
    RecipientPhone string            `json:"recipient_phone" dynamodbav:"recipient_phone"`

    Subject       string             `json:"subject" dynamodbav:"subject"`
    Body          string             `json:"body" dynamodbav:"body"`
    TemplateID    string             `json:"template_id" dynamodbav:"template_id"`
    TemplateData  map[string]string  `json:"template_data" dynamodbav:"template_data"`

    // Context
    TriggerType   string             `json:"trigger_type" dynamodbav:"trigger_type"` // ORDER_STATUS, REFUND, etc.
    ReferenceType string             `json:"reference_type" dynamodbav:"reference_type"` // ORDER, PRODUCT
    ReferenceID   string             `json:"reference_id" dynamodbav:"reference_id"`

    Status        NotificationStatus `json:"status" dynamodbav:"status"`
    SentAt        *time.Time         `json:"sent_at" dynamodbav:"sent_at"`
    FailureReason string             `json:"failure_reason" dynamodbav:"failure_reason"`
    RetryCount    int                `json:"retry_count" dynamodbav:"retry_count"`

    CreatedAt     time.Time          `json:"created_at" dynamodbav:"created_at"`
    CreatedBy     string             `json:"created_by" dynamodbav:"created_by"`
}

// ==================== AUDIT LOG ENTITIES (Table: handloom-audit) ====================

// AuditLog stored in handloom-audit table with 90-day TTL
type AuditLog struct {
    ID          string                 `json:"id" dynamodbav:"id"`
    PK          string                 `json:"-" dynamodbav:"PK"`           // AUDIT#<date>
    SK          string                 `json:"-" dynamodbav:"SK"`           // <timestamp>#<id>
    GSI1PK      string                 `json:"-" dynamodbav:"GSI1PK"`       // USER#<user_id>
    GSI1SK      string                 `json:"-" dynamodbav:"GSI1SK"`       // <timestamp>
    GSI2PK      string                 `json:"-" dynamodbav:"GSI2PK"`       // ENTITY#<type>#<id>
    GSI2SK      string                 `json:"-" dynamodbav:"GSI2SK"`       // <timestamp>
    EntityType  string                 `json:"-" dynamodbav:"entity_type"`  // AUDIT

    UserID      string                 `json:"user_id" dynamodbav:"user_id"`
    UserEmail   string                 `json:"user_email" dynamodbav:"user_email"`
    UserRole    string                 `json:"user_role" dynamodbav:"user_role"`

    Action      string                 `json:"action" dynamodbav:"action"`       // CREATE, UPDATE, DELETE
    TargetType  string                 `json:"target_type" dynamodbav:"target_type"` // ORDER, PRODUCT, etc.
    TargetID    string                 `json:"target_id" dynamodbav:"target_id"`

    Changes     map[string]ChangeDetail `json:"changes" dynamodbav:"changes"`

    IPAddress   string                 `json:"ip_address" dynamodbav:"ip_address"`
    UserAgent   string                 `json:"user_agent" dynamodbav:"user_agent"`

    CreatedAt   time.Time              `json:"created_at" dynamodbav:"created_at"`
    TTL         int64                  `json:"-" dynamodbav:"ttl"` // 90 days from CreatedAt
}

func (a AuditLog) TableName() string { return TableAudit }

// CalculateTTL returns Unix timestamp 90 days from now
func (a AuditLog) CalculateTTL() int64 {
    return a.CreatedAt.Add(90 * 24 * time.Hour).Unix()
}

type ChangeDetail struct {
    OldValue interface{} `json:"old_value" dynamodbav:"old_value"`
    NewValue interface{} `json:"new_value" dynamodbav:"new_value"`
}

// ==================== ANALYTICS ENTITIES (Table: handloom-analytics) ====================

// DailyMetrics stored in handloom-analytics table
type DailyMetrics struct {
    PK               string    `json:"-" dynamodbav:"PK"`           // METRICS#DAILY
    SK               string    `json:"-" dynamodbav:"SK"`           // 2024-01-15
    EntityType       string    `json:"-" dynamodbav:"entity_type"`  // DAILY_METRICS

    Date             string    `json:"date" dynamodbav:"date"`

    // Order Metrics
    TotalOrders      int       `json:"total_orders" dynamodbav:"total_orders"`
    PendingOrders    int       `json:"pending_orders" dynamodbav:"pending_orders"`
    ConfirmedOrders  int       `json:"confirmed_orders" dynamodbav:"confirmed_orders"`
    ShippedOrders    int       `json:"shipped_orders" dynamodbav:"shipped_orders"`
    DeliveredOrders  int       `json:"delivered_orders" dynamodbav:"delivered_orders"`
    CancelledOrders  int       `json:"cancelled_orders" dynamodbav:"cancelled_orders"`

    // Revenue Metrics (in paise)
    TotalRevenue     int64     `json:"total_revenue" dynamodbav:"total_revenue"`
    AvgOrderValue    int64     `json:"avg_order_value" dynamodbav:"avg_order_value"`
    TotalRefunds     int64     `json:"total_refunds" dynamodbav:"total_refunds"`
    NetRevenue       int64     `json:"net_revenue" dynamodbav:"net_revenue"`

    // Product Metrics
    TotalItemsSold   int       `json:"total_items_sold" dynamodbav:"total_items_sold"`
    UniqueProducts   int       `json:"unique_products" dynamodbav:"unique_products"`

    // Inventory Metrics
    LowStockCount    int       `json:"low_stock_count" dynamodbav:"low_stock_count"`
    OutOfStockCount  int       `json:"out_of_stock_count" dynamodbav:"out_of_stock_count"`

    UpdatedAt        time.Time `json:"updated_at" dynamodbav:"updated_at"`
    TTL              int64     `json:"-" dynamodbav:"ttl"` // 2 years
}

func (dm DailyMetrics) TableName() string { return TableAnalytics }

// TopProductMetric stored in handloom-analytics table
type TopProductMetric struct {
    PK            string    `json:"-" dynamodbav:"PK"`           // TOP_PRODUCTS
    SK            string    `json:"-" dynamodbav:"SK"`           // DAILY#2024-01-15#1 (period#date#rank)
    EntityType    string    `json:"-" dynamodbav:"entity_type"`  // TOP_PRODUCT

    Period        string    `json:"period" dynamodbav:"period"`   // DAILY, WEEKLY, MONTHLY
    Date          string    `json:"date" dynamodbav:"date"`
    Rank          int       `json:"rank" dynamodbav:"rank"`

    ProductID     string    `json:"product_id" dynamodbav:"product_id"`
    ProductName   string    `json:"product_name" dynamodbav:"product_name"`
    ProductSKU    string    `json:"product_sku" dynamodbav:"product_sku"`
    TotalQuantity int       `json:"total_quantity" dynamodbav:"total_quantity"`
    TotalRevenue  int64     `json:"total_revenue" dynamodbav:"total_revenue"`
    OrderCount    int       `json:"order_count" dynamodbav:"order_count"`

    TTL           int64     `json:"-" dynamodbav:"ttl"`
}

func (tp TopProductMetric) TableName() string { return TableAnalytics }

// InventoryAlert stored in handloom-analytics table
type InventoryAlert struct {
    PK            string    `json:"-" dynamodbav:"PK"`           // INVENTORY_ALERT
    SK            string    `json:"-" dynamodbav:"SK"`           // PRODUCT#<id>
    GSI1PK        string    `json:"-" dynamodbav:"GSI1PK"`       // ALERT_TYPE#LOW_STOCK or ALERT_TYPE#OUT_OF_STOCK
    EntityType    string    `json:"-" dynamodbav:"entity_type"`  // INVENTORY_ALERT

    ProductID     string    `json:"product_id" dynamodbav:"product_id"`
    ProductName   string    `json:"product_name" dynamodbav:"product_name"`
    ProductSKU    string    `json:"product_sku" dynamodbav:"product_sku"`
    AlertType     string    `json:"alert_type" dynamodbav:"alert_type"` // LOW_STOCK, OUT_OF_STOCK
    CurrentQty    int       `json:"current_qty" dynamodbav:"current_qty"`
    Threshold     int       `json:"threshold" dynamodbav:"threshold"`

    CreatedAt     time.Time `json:"created_at" dynamodbav:"created_at"`
    ResolvedAt    *time.Time `json:"resolved_at" dynamodbav:"resolved_at"`
}

func (ia InventoryAlert) TableName() string { return TableAnalytics }
```

---

## Low-Level Design

### Project Structure

```
handloom-admin/
├── cmd/
│   ├── api/
│   │   └── main.go              # Main API Lambda (admin routes)
│   ├── auth/
│   │   └── main.go              # Auth Lambda entry point
│   ├── stream-processor/
│   │   └── main.go              # DynamoDB Streams processor
│   ├── notification-worker/
│   │   └── main.go              # Notification queue consumer
│   ├── bulk-processor/
│   │   └── main.go              # Bulk import/export processor
│   ├── image-processor/
│   │   └── main.go              # S3 image resize/optimize
│   ├── report-generator/
│   │   └── main.go              # Report generation worker
│   └── analytics-aggregator/
│       └── main.go              # Scheduled analytics job
│
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration (table names from env)
│   │
│   ├── database/
│   │   ├── client.go            # Multi-table DynamoDB client
│   │   └── tables.go            # Table name constants and helpers
│   │
│   ├── models/
│   │   ├── tables.go            # Table name constants
│   │   ├── user.go              # → handloom-core
│   │   ├── category.go          # → handloom-core
│   │   ├── design.go            # → handloom-core
│   │   ├── product.go           # → handloom-core
│   │   ├── inventory.go         # → handloom-core
│   │   ├── artisan.go           # → handloom-core
│   │   ├── coupon.go            # → handloom-core
│   │   ├── asset.go             # → handloom-core
│   │   ├── notification.go      # → handloom-core
│   │   ├── bulk_job.go          # → handloom-core
│   │   ├── report.go            # → handloom-core
│   │   ├── order.go             # → handloom-orders
│   │   ├── customer.go          # → handloom-orders
│   │   ├── audit.go             # → handloom-audit
│   │   └── analytics.go         # → handloom-analytics
│   │
│   ├── repository/
│   │   ├── base.go              # Base repository with table selection
│   │   ├── core_repo.go         # Repository for handloom-core
│   │   ├── orders_repo.go       # Repository for handloom-orders
│   │   ├── audit_repo.go        # Repository for handloom-audit
│   │   └── analytics_repo.go    # Repository for handloom-analytics
│   │
│   ├── service/
│   │   ├── auth_service.go
│   │   ├── user_service.go
│   │   ├── category_service.go
│   │   ├── design_service.go
│   │   ├── product_service.go
│   │   ├── inventory_service.go
│   │   ├── artisan_service.go
│   │   ├── coupon_service.go
│   │   ├── asset_service.go
│   │   ├── order_service.go      # Uses both core_repo + orders_repo
│   │   ├── notification_service.go
│   │   ├── bulk_service.go
│   │   ├── report_service.go
│   │   ├── dashboard_service.go  # Uses analytics_repo
│   │   └── audit_service.go      # Uses audit_repo
│   │
│   ├── handler/
│   │   ├── auth_handler.go
│   │   ├── user_handler.go
│   │   ├── category_handler.go
│   │   ├── design_handler.go
│   │   ├── product_handler.go
│   │   ├── inventory_handler.go
│   │   ├── artisan_handler.go
│   │   ├── coupon_handler.go
│   │   ├── asset_handler.go
│   │   ├── order_handler.go
│   │   ├── notification_handler.go
│   │   ├── bulk_handler.go
│   │   ├── report_handler.go
│   │   └── dashboard_handler.go
│   │
│   ├── middleware/
│   │   ├── auth.go              # JWT validation
│   │   ├── rbac.go              # Role-based access control
│   │   ├── audit.go             # Auto-audit logging middleware
│   │   ├── logging.go
│   │   └── error.go
│   │
│   ├── queue/
│   │   └── sqs.go               # SQS publisher/consumer
│   │
│   ├── storage/
│   │   └── s3.go                # S3 operations (assets, reports)
│   │
│   └── utils/
│       ├── validator.go
│       ├── pagination.go
│       ├── response.go
│       └── errors.go
│
├── pkg/
│   └── common/
│       ├── types.go
│       └── constants.go
│
├── scripts/
│   └── create_tables.go         # DynamoDB table creation
│
├── go.mod
├── go.sum
├── Makefile
└── serverless.yml               # Or SAM template
```

### Multi-Table Repository Pattern

```go
// internal/database/client.go

package database

import (
    "context"
    "os"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// TableNames holds all table names loaded from environment
type TableNames struct {
    Core      string // handloom-core
    Orders    string // handloom-orders
    Audit     string // handloom-audit
    Analytics string // handloom-analytics
}

// Client wraps DynamoDB client with table name resolution
type Client struct {
    DDB    *dynamodb.Client
    Tables TableNames
}

// NewClient creates a new multi-table DynamoDB client
func NewClient(ctx context.Context) (*Client, error) {
    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        return nil, err
    }

    return &Client{
        DDB: dynamodb.NewFromConfig(cfg),
        Tables: TableNames{
            Core:      os.Getenv("CORE_TABLE"),
            Orders:    os.Getenv("ORDERS_TABLE"),
            Audit:     os.Getenv("AUDIT_TABLE"),
            Analytics: os.Getenv("ANALYTICS_TABLE"),
        },
    }, nil
}

// TableFor returns the correct table name for an entity type
func (c *Client) TableFor(entityType string) string {
    switch entityType {
    case "ORDER", "ORDER_ITEM", "ORDER_STATUS", "CUSTOMER":
        return c.Tables.Orders
    case "AUDIT":
        return c.Tables.Audit
    case "METRICS", "TOP_PRODUCTS", "INVENTORY_ALERT":
        return c.Tables.Analytics
    default:
        return c.Tables.Core
    }
}
```

```go
// internal/repository/base.go

package repository

import (
    "context"

    "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
    "handloom-admin/internal/database"
)

// BaseRepository provides common DynamoDB operations
type BaseRepository struct {
    client *database.Client
    table  string
}

// NewBaseRepository creates a repository for a specific table
func NewBaseRepository(client *database.Client, table string) *BaseRepository {
    return &BaseRepository{client: client, table: table}
}

// Put saves an item to the table
func (r *BaseRepository) Put(ctx context.Context, item interface{}) error {
    av, err := attributevalue.MarshalMap(item)
    if err != nil {
        return err
    }

    _, err = r.client.DDB.PutItem(ctx, &dynamodb.PutItemInput{
        TableName: &r.table,
        Item:      av,
    })
    return err
}

// Get retrieves an item by PK and SK
func (r *BaseRepository) Get(ctx context.Context, pk, sk string, out interface{}) error {
    result, err := r.client.DDB.GetItem(ctx, &dynamodb.GetItemInput{
        TableName: &r.table,
        Key: map[string]types.AttributeValue{
            "PK": &types.AttributeValueMemberS{Value: pk},
            "SK": &types.AttributeValueMemberS{Value: sk},
        },
    })
    if err != nil {
        return err
    }

    return attributevalue.UnmarshalMap(result.Item, out)
}

// Query executes a query against the table
func (r *BaseRepository) Query(ctx context.Context, input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
    input.TableName = &r.table
    return r.client.DDB.Query(ctx, input)
}
```

```go
// internal/repository/core_repo.go

package repository

import (
    "context"
    "fmt"

    "handloom-admin/internal/database"
    "handloom-admin/internal/models"
)

// CoreRepository handles all entities in handloom-core table
type CoreRepository struct {
    *BaseRepository
}

func NewCoreRepository(client *database.Client) *CoreRepository {
    return &CoreRepository{
        BaseRepository: NewBaseRepository(client, client.Tables.Core),
    }
}

// Users
func (r *CoreRepository) CreateUser(ctx context.Context, user *models.User) error {
    user.PK = fmt.Sprintf("USER#%s", user.ID)
    user.SK = "PROFILE"
    user.GSI1PK = fmt.Sprintf("ROLE#%s", user.Role)
    user.GSI2PK = fmt.Sprintf("EMAIL#%s", user.Email)
    user.EntityType = "USER"
    return r.Put(ctx, user)
}

func (r *CoreRepository) GetUser(ctx context.Context, userID string) (*models.User, error) {
    var user models.User
    err := r.Get(ctx, fmt.Sprintf("USER#%s", userID), "PROFILE", &user)
    return &user, err
}

// Products
func (r *CoreRepository) CreateProduct(ctx context.Context, product *models.Product) error {
    product.PK = fmt.Sprintf("PRODUCT#%s", product.ID)
    product.SK = "METADATA"
    product.GSI1PK = fmt.Sprintf("CATEGORY#%s", product.CategoryID)
    product.GSI2PK = fmt.Sprintf("SKU#%s", product.SKU)
    product.EntityType = "PRODUCT"
    return r.Put(ctx, product)
}

// ... similar methods for Category, Design, Inventory, Artisan, Coupon, etc.
```

```go
// internal/repository/orders_repo.go

package repository

import (
    "context"
    "fmt"

    "handloom-admin/internal/database"
    "handloom-admin/internal/models"
)

// OrdersRepository handles all entities in handloom-orders table
type OrdersRepository struct {
    *BaseRepository
}

func NewOrdersRepository(client *database.Client) *OrdersRepository {
    return &OrdersRepository{
        BaseRepository: NewBaseRepository(client, client.Tables.Orders),
    }
}

// Orders
func (r *OrdersRepository) CreateOrder(ctx context.Context, order *models.Order) error {
    order.PK = fmt.Sprintf("ORDER#%s", order.ID)
    order.SK = "METADATA"
    order.GSI1PK = fmt.Sprintf("STATUS#%s", order.Status)
    order.GSI2PK = fmt.Sprintf("CUSTOMER#%s", order.CustomerID)
    order.EntityType = "ORDER"
    return r.Put(ctx, order)
}

func (r *OrdersRepository) GetOrder(ctx context.Context, orderID string) (*models.Order, error) {
    var order models.Order
    err := r.Get(ctx, fmt.Sprintf("ORDER#%s", orderID), "METADATA", &order)
    return &order, err
}

// GetOrderWithItems returns order + all items in single query
func (r *OrdersRepository) GetOrderWithItems(ctx context.Context, orderID string) (*models.OrderDetail, error) {
    // Query with PK = ORDER#<id>, no SK filter to get all items
    // ... implementation
}

// Customers (for B2C)
func (r *OrdersRepository) CreateCustomer(ctx context.Context, customer *models.Customer) error {
    customer.PK = fmt.Sprintf("CUSTOMER#%s", customer.ID)
    customer.SK = "PROFILE"
    customer.EntityType = "CUSTOMER"
    return r.Put(ctx, customer)
}
```

```go
// internal/repository/audit_repo.go

package repository

import (
    "context"
    "fmt"
    "time"

    "handloom-admin/internal/database"
    "handloom-admin/internal/models"
)

// AuditRepository handles all entities in handloom-audit table
type AuditRepository struct {
    *BaseRepository
}

func NewAuditRepository(client *database.Client) *AuditRepository {
    return &AuditRepository{
        BaseRepository: NewBaseRepository(client, client.Tables.Audit),
    }
}

func (r *AuditRepository) LogAction(ctx context.Context, log *models.AuditLog) error {
    date := log.CreatedAt.Format("2006-01-02")
    log.PK = fmt.Sprintf("AUDIT#%s", date)
    log.SK = fmt.Sprintf("%s#%s", log.CreatedAt.Format(time.RFC3339Nano), log.ID)
    log.GSI1PK = fmt.Sprintf("USER#%s", log.UserID)
    log.GSI1SK = log.CreatedAt.Format(time.RFC3339Nano)
    log.GSI2PK = fmt.Sprintf("ENTITY#%s#%s", log.TargetType, log.TargetID)
    log.GSI2SK = log.CreatedAt.Format(time.RFC3339Nano)
    log.EntityType = "AUDIT"
    log.TTL = log.CalculateTTL() // 90 days from now
    return r.Put(ctx, log)
}

// GetAuditLogsByUser retrieves audit logs for a specific user
func (r *AuditRepository) GetAuditLogsByUser(ctx context.Context, userID string, limit int) ([]*models.AuditLog, error) {
    // Query GSI1 with GSI1PK = USER#<userID>
    // ... implementation
}
```

### Cross-Table Operations

For operations that span multiple tables (e.g., creating an order that reserves inventory):

```go
// internal/service/order_service.go

type OrderService struct {
    coreRepo   *repository.CoreRepository
    ordersRepo *repository.OrdersRepository
    auditRepo  *repository.AuditRepository
    sqsClient  *queue.SQSClient
}

// CreateOrder handles order creation with inventory reservation
func (s *OrderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error) {
    // 1. Validate products exist and have stock (from handloom-core)
    for _, item := range req.Items {
        inventory, err := s.coreRepo.GetInventory(ctx, item.ProductID)
        if err != nil {
            return nil, err
        }
        if inventory.AvailableQty < item.Quantity {
            return nil, ErrInsufficientStock
        }
    }

    // 2. Reserve inventory (update handloom-core)
    // Using DynamoDB conditional writes to prevent overselling
    for _, item := range req.Items {
        if err := s.coreRepo.ReserveInventory(ctx, item.ProductID, item.Quantity); err != nil {
            // Rollback previous reservations
            s.rollbackReservations(ctx, reservedItems)
            return nil, err
        }
    }

    // 3. Create order (in handloom-orders)
    order := buildOrder(req)
    if err := s.ordersRepo.CreateOrder(ctx, order); err != nil {
        s.rollbackReservations(ctx, req.Items)
        return nil, err
    }

    // 4. Create order items (in handloom-orders, same PK as order)
    for _, item := range req.Items {
        if err := s.ordersRepo.CreateOrderItem(ctx, order.ID, item); err != nil {
            // Order created but items failed - log for manual review
            log.Error("Failed to create order item", err)
        }
    }

    // 5. Audit log (async via SQS → handloom-audit)
    // Don't block order creation for audit
    s.sqsClient.SendAuditEvent(ctx, AuditEvent{
        Action:     "CREATE",
        EntityType: "ORDER",
        EntityID:   order.ID,
        UserID:     req.UserID,
    })

    return order, nil
}
```

### RBAC Permission Matrix

```go
// Permission constants
const (
    // User permissions
    PermUserCreate     = "user:create"
    PermUserRead       = "user:read"
    PermUserUpdate     = "user:update"
    PermUserDelete     = "user:delete"

    // Category permissions
    PermCategoryCreate = "category:create"
    PermCategoryRead   = "category:read"
    PermCategoryUpdate = "category:update"
    PermCategoryDelete = "category:delete"

    // Design permissions
    PermDesignCreate   = "design:create"
    PermDesignRead     = "design:read"
    PermDesignUpdate   = "design:update"
    PermDesignDelete   = "design:delete"

    // Product permissions
    PermProductCreate  = "product:create"
    PermProductRead    = "product:read"
    PermProductUpdate  = "product:update"
    PermProductDelete  = "product:delete"

    // Inventory permissions
    PermInventoryRead  = "inventory:read"
    PermInventoryAdd   = "inventory:add"
    PermInventoryRemove = "inventory:remove"
    PermInventoryAdjust = "inventory:adjust"

    // Order permissions
    PermOrderRead      = "order:read"
    PermOrderUpdate    = "order:update"
    PermOrderCancel    = "order:cancel"
    PermOrderRefund    = "order:refund"

    // Notification permissions
    PermNotificationSend = "notification:send"
    PermNotificationRead = "notification:read"

    // Audit permissions
    PermAuditRead      = "audit:read"
)

// Role permission mapping
var RolePermissions = map[UserRole][]string{
    RoleAdmin: {
        // Full access
        PermUserCreate, PermUserRead, PermUserUpdate, PermUserDelete,
        PermCategoryCreate, PermCategoryRead, PermCategoryUpdate, PermCategoryDelete,
        PermDesignCreate, PermDesignRead, PermDesignUpdate, PermDesignDelete,
        PermProductCreate, PermProductRead, PermProductUpdate, PermProductDelete,
        PermInventoryRead, PermInventoryAdd, PermInventoryRemove, PermInventoryAdjust,
        PermOrderRead, PermOrderUpdate, PermOrderCancel, PermOrderRefund,
        PermNotificationSend, PermNotificationRead,
        PermAuditRead,
    },
    RoleOperator: {
        // Limited access - no user management, no delete operations
        PermCategoryRead,
        PermDesignRead,
        PermProductRead, PermProductUpdate,
        PermInventoryRead, PermInventoryAdd, PermInventoryRemove,
        PermOrderRead, PermOrderUpdate,
        PermNotificationRead,
    },
}
```

### Service Layer Interfaces

```go
// ==================== AUTH SERVICE ====================

type AuthService interface {
    Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
    ValidateToken(ctx context.Context, token string) (*TokenClaims, error)
    RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)
    Logout(ctx context.Context, userID string) error
}

// ==================== USER SERVICE ====================

type UserService interface {
    CreateUser(ctx context.Context, req CreateUserRequest, createdBy string) (*User, error)
    GetUser(ctx context.Context, userID string) (*User, error)
    ListUsers(ctx context.Context, req ListUsersRequest) (*ListUsersResponse, error)
    UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) (*User, error)
    DeleteUser(ctx context.Context, userID string) error
    UpdateUserStatus(ctx context.Context, userID string, status UserStatus) error
}

// ==================== CATEGORY SERVICE ====================

type CategoryService interface {
    CreateCategory(ctx context.Context, req CreateCategoryRequest, createdBy string) (*Category, error)
    GetCategory(ctx context.Context, categoryID string) (*Category, error)
    ListCategories(ctx context.Context, req ListCategoriesRequest) (*ListCategoriesResponse, error)
    GetCategoryTree(ctx context.Context) ([]*CategoryNode, error)
    UpdateCategory(ctx context.Context, categoryID string, req UpdateCategoryRequest) (*Category, error)
    DeleteCategory(ctx context.Context, categoryID string) error
}

// ==================== DESIGN SERVICE ====================

type DesignService interface {
    CreateDesign(ctx context.Context, req CreateDesignRequest, createdBy string) (*Design, error)
    GetDesign(ctx context.Context, designID string) (*Design, error)
    ListDesigns(ctx context.Context, req ListDesignsRequest) (*ListDesignsResponse, error)
    ListDesignsByCategory(ctx context.Context, categoryID string, pagination PaginationRequest) (*ListDesignsResponse, error)
    UpdateDesign(ctx context.Context, designID string, req UpdateDesignRequest) (*Design, error)
    DeleteDesign(ctx context.Context, designID string) error
}

// ==================== PRODUCT SERVICE ====================

type ProductService interface {
    CreateProduct(ctx context.Context, req CreateProductRequest, createdBy string) (*Product, error)
    GetProduct(ctx context.Context, productID string) (*ProductWithInventory, error)
    GetProductBySKU(ctx context.Context, sku string) (*ProductWithInventory, error)
    ListProducts(ctx context.Context, req ListProductsRequest) (*ListProductsResponse, error)
    UpdateProduct(ctx context.Context, productID string, req UpdateProductRequest) (*Product, error)
    DeleteProduct(ctx context.Context, productID string) error
}

// ==================== INVENTORY SERVICE ====================

type InventoryService interface {
    GetInventory(ctx context.Context, productID string) (*Inventory, error)
    AddStock(ctx context.Context, req AddStockRequest, userID string) (*Inventory, error)
    RemoveStock(ctx context.Context, req RemoveStockRequest, userID string) (*Inventory, error)
    AdjustStock(ctx context.Context, req AdjustStockRequest, userID string) (*Inventory, error)
    ReserveStock(ctx context.Context, productID string, quantity int, orderID string) error
    ReleaseStock(ctx context.Context, productID string, quantity int, orderID string) error
    GetInventoryTransactions(ctx context.Context, productID string, pagination PaginationRequest) (*ListInventoryTransactionsResponse, error)
    GetLowStockProducts(ctx context.Context, pagination PaginationRequest) (*ListProductsResponse, error)
}

// ==================== ORDER SERVICE ====================

type OrderService interface {
    GetOrder(ctx context.Context, orderID string) (*OrderDetail, error)
    ListOrders(ctx context.Context, req ListOrdersRequest) (*ListOrdersResponse, error)
    UpdateOrderStatus(ctx context.Context, orderID string, req UpdateOrderStatusRequest, userID string) (*Order, error)
    AddOrderNote(ctx context.Context, orderID string, note string, userID string) error
    UpdateTrackingInfo(ctx context.Context, orderID string, req UpdateTrackingRequest) (*Order, error)
    InitiateRefund(ctx context.Context, orderID string, req RefundRequest, userID string) error
    CancelOrder(ctx context.Context, orderID string, reason string, userID string) error
    GetOrderStatusHistory(ctx context.Context, orderID string) ([]*OrderStatusHistory, error)
}

// ==================== NOTIFICATION SERVICE ====================

type NotificationService interface {
    SendNotification(ctx context.Context, req SendNotificationRequest, userID string) (*Notification, error)
    SendOrderStatusNotification(ctx context.Context, orderID string, status OrderStatus) error
    SendRefundNotification(ctx context.Context, orderID string, amount int64, reason string) error
    SendCancellationNotification(ctx context.Context, orderID string, reason string) error
    GetNotifications(ctx context.Context, req ListNotificationsRequest) (*ListNotificationsResponse, error)
    GetNotificationsByRecipient(ctx context.Context, recipientID string, pagination PaginationRequest) (*ListNotificationsResponse, error)
}

// ==================== AUDIT SERVICE ====================

type AuditService interface {
    LogAction(ctx context.Context, req AuditLogRequest) error
    GetAuditLogs(ctx context.Context, req ListAuditLogsRequest) (*ListAuditLogsResponse, error)
    GetAuditLogsByUser(ctx context.Context, userID string, pagination PaginationRequest) (*ListAuditLogsResponse, error)
    GetAuditLogsByEntity(ctx context.Context, entityType, entityID string, pagination PaginationRequest) (*ListAuditLogsResponse, error)
}
```

### Inventory Transaction Flow

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Handler   │────▶│   Service   │────▶│ Repository  │────▶│  DynamoDB   │
└─────────────┘     └─────────────┘     └─────────────┘     └─────────────┘
                           │
                           │ (Async)
                           ▼
                    ┌─────────────┐     ┌─────────────┐
                    │     SQS     │────▶│   Lambda    │────▶ Low Stock Alert
                    └─────────────┘     └─────────────┘
```

**Add Stock Transaction:**
```go
func (s *inventoryService) AddStock(ctx context.Context, req AddStockRequest, userID string) (*Inventory, error) {
    // 1. Get current inventory
    inventory, err := s.repo.GetInventory(ctx, req.ProductID)
    if err != nil {
        return nil, err
    }

    // 2. Calculate new quantity
    previousQty := inventory.Quantity
    newQty := previousQty + req.Quantity

    // 3. Update inventory with conditional write
    inventory.Quantity = newQty
    inventory.AvailableQty = newQty - inventory.ReservedQty
    inventory.LastRestockAt = timeNow()
    inventory.UpdatedAt = time.Now()
    inventory.UpdatedBy = userID

    if err := s.repo.UpdateInventory(ctx, inventory, previousQty); err != nil {
        return nil, err
    }

    // 4. Create transaction record
    txn := &InventoryTransaction{
        ID:            generateID(),
        ProductID:     req.ProductID,
        Type:          "ADD",
        Quantity:      req.Quantity,
        PreviousQty:   previousQty,
        NewQty:        newQty,
        Reason:        req.Reason,
        ReferenceType: "MANUAL",
        CreatedAt:     time.Now(),
        CreatedBy:     userID,
    }

    if err := s.repo.CreateInventoryTransaction(ctx, txn); err != nil {
        // Log but don't fail - transaction record is for audit
        log.Error("Failed to create inventory transaction", err)
    }

    // 5. Log audit
    s.auditService.LogAction(ctx, AuditLogRequest{
        UserID:     userID,
        Action:     "ADD_STOCK",
        EntityType: "INVENTORY",
        EntityID:   req.ProductID,
        Changes: map[string]ChangeDetail{
            "quantity": {OldValue: previousQty, NewValue: newQty},
        },
    })

    return inventory, nil
}
```

### Order Status State Machine

```
                            ┌─────────────────┐
                            │     PENDING     │
                            └────────┬────────┘
                                     │
                    ┌────────────────┼────────────────┐
                    ▼                                 ▼
           ┌─────────────────┐              ┌─────────────────┐
           │   CONFIRMED     │              │   CANCELLED     │
           └────────┬────────┘              └─────────────────┘
                    │
                    ▼
           ┌─────────────────┐
           │  PROCESSING     │
           └────────┬────────┘
                    │
                    ▼
           ┌─────────────────┐
           │    SHIPPED      │
           └────────┬────────┘
                    │
                    ▼
           ┌─────────────────┐
           │OUT_FOR_DELIVERY │
           └────────┬────────┘
                    │
                    ▼
           ┌─────────────────┐         ┌─────────────────┐
           │   DELIVERED     │────────▶│RETURN_REQUESTED │
           └─────────────────┘         └────────┬────────┘
                                                │
                              ┌─────────────────┼─────────────────┐
                              ▼                                   ▼
                     ┌─────────────────┐              ┌─────────────────┐
                     │ RETURN_APPROVED │              │   CANCELLED     │
                     └────────┬────────┘              └─────────────────┘
                              │
                              ▼
                     ┌─────────────────┐
                     │ RETURN_RECEIVED │
                     └────────┬────────┘
                              │
                              ▼
                     ┌─────────────────┐
                     │REFUND_INITIATED │
                     └────────┬────────┘
                              │
                              ▼
                     ┌─────────────────┐
                     │REFUND_COMPLETED │
                     └─────────────────┘
```

```go
// Valid state transitions
var ValidOrderTransitions = map[OrderStatus][]OrderStatus{
    OrderStatusPending:         {OrderStatusConfirmed, OrderStatusCancelled},
    OrderStatusConfirmed:       {OrderStatusProcessing, OrderStatusCancelled},
    OrderStatusProcessing:      {OrderStatusShipped, OrderStatusCancelled},
    OrderStatusShipped:         {OrderStatusOutForDelivery, OrderStatusDelivered},
    OrderStatusOutForDelivery:  {OrderStatusDelivered},
    OrderStatusDelivered:       {OrderStatusReturnRequested},
    OrderStatusReturnRequested: {OrderStatusReturnApproved, OrderStatusCancelled},
    OrderStatusReturnApproved:  {OrderStatusReturnReceived},
    OrderStatusReturnReceived:  {OrderStatusRefundInitiated},
    OrderStatusRefundInitiated: {OrderStatusRefundCompleted},
    // Terminal states: CANCELLED, REFUND_COMPLETED have no transitions
}

func (s *orderService) ValidateStatusTransition(current, next OrderStatus) error {
    validNext, ok := ValidOrderTransitions[current]
    if !ok {
        return ErrInvalidOrderStatus
    }

    for _, status := range validNext {
        if status == next {
            return nil
        }
    }

    return fmt.Errorf("cannot transition from %s to %s", current, next)
}
```

---

## API Contracts

### Base Response Structure

```go
// Success Response
type APIResponse[T any] struct {
    Success bool   `json:"success"`
    Data    T      `json:"data,omitempty"`
    Meta    *Meta  `json:"meta,omitempty"`
}

type Meta struct {
    Page       int    `json:"page,omitempty"`
    PerPage    int    `json:"per_page,omitempty"`
    Total      int64  `json:"total,omitempty"`
    TotalPages int    `json:"total_pages,omitempty"`
    NextCursor string `json:"next_cursor,omitempty"`
}

// Error Response
type ErrorResponse struct {
    Success bool         `json:"success"`
    Error   ErrorDetail  `json:"error"`
}

type ErrorDetail struct {
    Code    string            `json:"code"`
    Message string            `json:"message"`
    Details map[string]string `json:"details,omitempty"`
}
```

### Authentication APIs

#### POST /auth/login
```yaml
Request:
  Body:
    email: string (required, email format)
    password: string (required, min 8 chars)

Response 200:
  {
    "success": true,
    "data": {
      "access_token": "eyJhbGciOiJIUzI1NiIs...",
      "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2g...",
      "token_type": "Bearer",
      "expires_in": 3600,
      "user": {
        "id": "usr_abc123",
        "email": "admin@example.com",
        "name": "Admin User",
        "role": "ADMIN",
        "permissions": ["user:create", "user:read", ...]
      }
    }
  }

Response 401:
  {
    "success": false,
    "error": {
      "code": "INVALID_CREDENTIALS",
      "message": "Invalid email or password"
    }
  }
```

#### POST /auth/refresh
```yaml
Request:
  Body:
    refresh_token: string (required)

Response 200:
  {
    "success": true,
    "data": {
      "access_token": "eyJhbGciOiJIUzI1NiIs...",
      "expires_in": 3600
    }
  }
```

#### POST /auth/logout
```yaml
Headers:
  Authorization: Bearer <token>

Response 200:
  {
    "success": true,
    "data": {
      "message": "Logged out successfully"
    }
  }
```

---

### User Management APIs

#### POST /admin/users
```yaml
Permission: user:create
Role: ADMIN only

Request:
  Headers:
    Authorization: Bearer <token>
  Body:
    email: string (required, email format)
    name: string (required, min 2 chars)
    password: string (required, min 8 chars)
    role: string (required, enum: ADMIN, OPERATOR)

Response 201:
  {
    "success": true,
    "data": {
      "id": "usr_xyz789",
      "email": "operator@example.com",
      "name": "New Operator",
      "role": "OPERATOR",
      "status": "ACTIVE",
      "permissions": ["inventory:read", "inventory:add", ...],
      "created_at": "2024-01-15T10:30:00Z",
      "created_by": "usr_abc123"
    }
  }

Response 400:
  {
    "success": false,
    "error": {
      "code": "VALIDATION_ERROR",
      "message": "Validation failed",
      "details": {
        "email": "Email already exists"
      }
    }
  }
```

#### GET /admin/users
```yaml
Permission: user:read

Request:
  Headers:
    Authorization: Bearer <token>
  Query:
    page: int (default: 1)
    per_page: int (default: 20, max: 100)
    role: string (optional, filter by role)
    status: string (optional, filter by status)
    search: string (optional, search by name/email)

Response 200:
  {
    "success": true,
    "data": [
      {
        "id": "usr_abc123",
        "email": "admin@example.com",
        "name": "Admin User",
        "role": "ADMIN",
        "status": "ACTIVE",
        "created_at": "2024-01-01T00:00:00Z"
      }
    ],
    "meta": {
      "page": 1,
      "per_page": 20,
      "total": 5,
      "total_pages": 1
    }
  }
```

#### GET /admin/users/{id}
```yaml
Permission: user:read

Response 200:
  {
    "success": true,
    "data": {
      "id": "usr_abc123",
      "email": "admin@example.com",
      "name": "Admin User",
      "role": "ADMIN",
      "status": "ACTIVE",
      "permissions": ["user:create", ...],
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-10T00:00:00Z"
    }
  }
```

#### PATCH /admin/users/{id}
```yaml
Permission: user:update
Role: ADMIN only

Request:
  Body:
    name: string (optional)
    role: string (optional, enum: ADMIN, OPERATOR)
    status: string (optional, enum: ACTIVE, INACTIVE, SUSPENDED)

Response 200:
  {
    "success": true,
    "data": {
      "id": "usr_abc123",
      "email": "admin@example.com",
      "name": "Updated Name",
      "role": "ADMIN",
      "status": "ACTIVE",
      "updated_at": "2024-01-15T12:00:00Z"
    }
  }
```

#### DELETE /admin/users/{id}
```yaml
Permission: user:delete
Role: ADMIN only

Response 200:
  {
    "success": true,
    "data": {
      "message": "User deleted successfully"
    }
  }
```

---

### Category APIs

#### POST /admin/categories
```yaml
Permission: category:create

Request:
  Body:
    name: string (required)
    description: string (optional)
    parent_id: string (optional, for subcategory - will inherit parent's attributes)
    image_url: string (optional)
    sort_order: int (optional, default: 0)
    # Custom dimension configuration
    allow_custom_dimensions: bool (optional, default: false)
    dimension_config: object (optional, required if allow_custom_dimensions is true)
      length_enabled: bool
      length_min: float
      length_max: float
      length_step: float (increment)
      length_unit: string (inches/cm/feet)
      width_enabled: bool
      width_min: float
      width_max: float
      width_step: float
      width_unit: string
      height_enabled: bool (for 3D products)
      height_min: float
      height_max: float
      height_step: float
      height_unit: string
      pricing_model: string (AREA_BASED/LENGTH_BASED/VOLUME_BASED)
    default_pricing_rule_id: string (optional)
    # Category-specific attributes (own attributes, not inherited)
    own_attributes: array (optional)
      - name: string (e.g., "bed_size", "elastic_type")
        label: string (e.g., "Bed Size", "Elastic Type")
        type: string (SELECT/MULTI_SELECT/TEXT/NUMBER/BOOLEAN/DIMENSION_RANGE)
        required: bool
        options: array (for SELECT/MULTI_SELECT types)
          - value: string
            label: string
            surcharge: int (optional, in paise)
        dimension_config: object (for DIMENSION_RANGE type)
          min_value: float
          max_value: float
          step: float
          unit: string
          default_value: float
        affects_pricing: bool

Response 201:
  {
    "success": true,
    "data": {
      "id": "cat_bedsheets",
      "name": "Bedsheets",
      "slug": "bedsheets",
      "description": "High quality handloom bedsheets",
      "parent_id": "cat_bedding",
      "level": 2,
      "path": "cat_home_textiles/cat_bedding/cat_bedsheets",
      "ancestor_ids": ["cat_home_textiles", "cat_bedding"],
      "image_url": "https://...",
      "status": "ACTIVE",
      "sort_order": 1,
      "allow_custom_dimensions": true,
      "dimension_config": {
        "length_enabled": true,
        "length_min": 60,
        "length_max": 120,
        "length_step": 1,
        "length_unit": "inches",
        "width_enabled": true,
        "width_min": 40,
        "width_max": 108,
        "width_step": 1,
        "width_unit": "inches",
        "pricing_model": "AREA_BASED"
      },
      "own_attributes": [
        {
          "name": "bed_size",
          "label": "Bed Size",
          "type": "SELECT",
          "required": true,
          "options": [
            {"value": "single", "label": "Single (36x75)", "surcharge": 0},
            {"value": "double", "label": "Double (54x75)", "surcharge": 0},
            {"value": "queen", "label": "Queen (60x80)", "surcharge": 10000},
            {"value": "king", "label": "King (76x80)", "surcharge": 20000},
            {"value": "custom", "label": "Custom Size", "surcharge": 0}
          ],
          "affects_pricing": true
        },
        {
          "name": "elastic_type",
          "label": "Elastic Type",
          "type": "SELECT",
          "required": true,
          "options": [
            {"value": "flat", "label": "Flat Sheet", "surcharge": 0},
            {"value": "fitted", "label": "Fitted Sheet", "surcharge": 15000},
            {"value": "elasticated", "label": "Elasticated Corners", "surcharge": 10000}
          ],
          "affects_pricing": true
        }
      ],
      "inherited_attributes_count": 6,
      "total_attributes_count": 8,
      "created_at": "2024-01-15T10:00:00Z"
    }
  }
```

#### GET /admin/categories
```yaml
Permission: category:read

Request:
  Query:
    parent_id: string (optional, filter by parent)
    status: string (optional)
    tree: bool (optional, return hierarchical tree)

Response 200 (flat):
  {
    "success": true,
    "data": [
      {
        "id": "cat_abc123",
        "name": "Sarees",
        "slug": "sarees",
        "parent_id": null,
        "level": 0,
        "status": "ACTIVE"
      }
    ]
  }

Response 200 (tree=true):
  {
    "success": true,
    "data": [
      {
        "id": "cat_abc123",
        "name": "Sarees",
        "slug": "sarees",
        "level": 0,
        "children": [
          {
            "id": "cat_def456",
            "name": "Banarasi Sarees",
            "slug": "banarasi-sarees",
            "level": 1,
            "children": []
          }
        ]
      }
    ]
  }
```

#### GET /admin/categories/{id}
```yaml
Permission: category:read

Request:
  Query:
    include_attributes: bool (optional, default: true, includes inherited attributes)

Response 200:
  {
    "success": true,
    "data": {
      "id": "cat_bedsheets",
      "name": "Bedsheets",
      "slug": "bedsheets",
      "description": "High quality handloom bedsheets",
      "parent_id": "cat_bedding",
      "level": 2,
      "path": "cat_home_textiles/cat_bedding/cat_bedsheets",
      "ancestor_ids": ["cat_home_textiles", "cat_bedding"],
      "image_url": "https://...",
      "status": "ACTIVE",
      "sort_order": 1,
      "allow_custom_dimensions": true,
      "dimension_config": {
        "length_enabled": true,
        "length_min": 60,
        "length_max": 120,
        "length_step": 1,
        "length_unit": "inches",
        "width_enabled": true,
        "width_min": 40,
        "width_max": 108,
        "width_step": 1,
        "width_unit": "inches",
        "pricing_model": "AREA_BASED"
      },
      "default_pricing_rule_id": "rule_bedsheets_area",
      "own_attributes": [
        {
          "name": "bed_size",
          "label": "Bed Size",
          "type": "SELECT",
          "required": true,
          "options": [...],
          "affects_pricing": true
        }
      ],
      "inherited_attributes": [
        {
          "name": "material",
          "label": "Material",
          "type": "SELECT",
          "source_category_id": "cat_home_textiles",
          "source_category_name": "Home Textiles",
          "required": true,
          "options": [
            {"value": "cotton", "label": "Cotton"},
            {"value": "silk", "label": "Silk"},
            {"value": "linen", "label": "Linen"}
          ],
          "affects_pricing": true
        },
        {
          "name": "thread_count",
          "label": "Thread Count",
          "type": "SELECT",
          "source_category_id": "cat_bedding",
          "source_category_name": "Bedding",
          "required": false,
          "options": [
            {"value": "200", "label": "200 TC"},
            {"value": "400", "label": "400 TC", "surcharge": 30000},
            {"value": "600", "label": "600 TC", "surcharge": 50000}
          ],
          "affects_pricing": true
        }
      ],
      "all_attributes": [...], // Combined own + inherited, sorted by display_order
      "created_at": "2024-01-15T10:00:00Z",
      "updated_at": "2024-01-15T10:00:00Z",
      "product_count": 150,
      "design_count": 25,
      "subcategory_count": 0,
      "parent": {
        "id": "cat_bedding",
        "name": "Bedding",
        "slug": "bedding"
      },
      "breadcrumb": [
        {"id": "cat_home_textiles", "name": "Home Textiles", "slug": "home-textiles"},
        {"id": "cat_bedding", "name": "Bedding", "slug": "bedding"},
        {"id": "cat_bedsheets", "name": "Bedsheets", "slug": "bedsheets"}
      ]
    }
  }
```

#### PATCH /admin/categories/{id}
```yaml
Permission: category:update

Request:
  Body:
    name: string (optional)
    description: string (optional)
    image_url: string (optional)
    status: string (optional)
    sort_order: int (optional)
    allow_custom_dimensions: bool (optional)
    dimension_config: object (optional)
    default_pricing_rule_id: string (optional)
    own_attributes: array (optional, replaces existing own_attributes)

Response 200:
  {
    "success": true,
    "data": {
      "id": "cat_bedsheets",
      "name": "Updated Name",
      ... (full category object with all attributes)
    }
  }
```

#### POST /admin/categories/{id}/attributes
```yaml
Permission: category:update
Description: Add a new attribute to the category

Request:
  Body:
    name: string (required, e.g., "pillow_cover_included")
    label: string (required, e.g., "Pillow Cover Included")
    type: string (required, SELECT/MULTI_SELECT/TEXT/NUMBER/BOOLEAN/DIMENSION_RANGE)
    required: bool (optional, default: false)
    display_order: int (optional)
    options: array (required for SELECT/MULTI_SELECT)
      - value: string
        label: string
        surcharge: int (optional, in paise)
    dimension_config: object (required for DIMENSION_RANGE)
    affects_pricing: bool (optional, default: false)

Response 201:
  {
    "success": true,
    "data": {
      "attribute": {
        "name": "pillow_cover_included",
        "label": "Pillow Cover Included",
        "type": "BOOLEAN",
        "required": false,
        "affects_pricing": true
      },
      "category": {
        "id": "cat_bedsheets",
        "own_attributes_count": 3,
        "total_attributes_count": 9
      }
    }
  }
```

#### PATCH /admin/categories/{id}/attributes/{attributeName}
```yaml
Permission: category:update
Description: Update an existing category attribute

Request:
  Body:
    label: string (optional)
    required: bool (optional)
    display_order: int (optional)
    options: array (optional, for SELECT/MULTI_SELECT)
    dimension_config: object (optional, for DIMENSION_RANGE)
    affects_pricing: bool (optional)

Response 200:
  {
    "success": true,
    "data": {
      "attribute": {...updated attribute},
      "affected_products_count": 45
    }
  }
```

#### DELETE /admin/categories/{id}/attributes/{attributeName}
```yaml
Permission: category:update
Description: Remove an attribute from the category (products retain existing values)

Response 200:
  {
    "success": true,
    "data": {
      "message": "Attribute removed successfully",
      "affected_products_count": 45
    }
  }

Response 400:
  {
    "success": false,
    "error": {
      "code": "ATTRIBUTE_IN_USE",
      "message": "Cannot delete attribute used in active pricing rules"
    }
  }
```

#### GET /admin/categories/{id}/attributes
```yaml
Permission: category:read
Description: Get all attributes for a category (own + inherited)

Request:
  Query:
    include_inherited: bool (optional, default: true)
    affects_pricing_only: bool (optional, filter to pricing-related attributes)

Response 200:
  {
    "success": true,
    "data": {
      "own_attributes": [
        {
          "name": "bed_size",
          "label": "Bed Size",
          "type": "SELECT",
          "required": true,
          "display_order": 1,
          "options": [...],
          "affects_pricing": true,
          "usage_count": 120
        }
      ],
      "inherited_attributes": [
        {
          "name": "material",
          "label": "Material",
          "type": "SELECT",
          "source_category": {
            "id": "cat_home_textiles",
            "name": "Home Textiles",
            "level": 0
          },
          "required": true,
          "options": [...],
          "affects_pricing": true
        }
      ],
      "total_count": 10
    }
  }
```

#### DELETE /admin/categories/{id}
```yaml
Permission: category:delete

Response 200:
  {
    "success": true,
    "data": {
      "message": "Category deleted successfully"
    }
  }

Response 400:
  {
    "success": false,
    "error": {
      "code": "CATEGORY_HAS_PRODUCTS",
      "message": "Cannot delete category with existing products"
    }
  }
```

---

### Design APIs

#### POST /admin/designs
```yaml
Permission: design:create

Request:
  Body:
    name: string (required)
    category_id: string (required)
    description: string (optional)
    images: array (optional)
      - url: string
        alt_text: string
        is_primary: bool
    attributes: array (optional)
      - name: string
        values: array of strings

Response 201:
  {
    "success": true,
    "data": {
      "id": "dsgn_abc123",
      "name": "Floral Pattern",
      "slug": "floral-pattern",
      "category_id": "cat_xyz789",
      "description": "Beautiful floral design",
      "images": [...],
      "attributes": [
        {"name": "Pattern", "values": ["Floral", "Traditional"]}
      ],
      "status": "ACTIVE",
      "created_at": "2024-01-15T10:00:00Z"
    }
  }
```

#### GET /admin/designs
```yaml
Permission: design:read

Request:
  Query:
    page: int
    per_page: int
    category_id: string (optional)
    status: string (optional)
    search: string (optional)

Response 200:
  {
    "success": true,
    "data": [...],
    "meta": {...}
  }
```

#### GET /admin/designs/{id}
```yaml
Permission: design:read

Response 200:
  {
    "success": true,
    "data": {
      "id": "dsgn_abc123",
      "name": "Floral Pattern",
      ...
      "category": {
        "id": "cat_xyz789",
        "name": "Sarees"
      },
      "product_count": 10
    }
  }
```

#### PATCH /admin/designs/{id}
```yaml
Permission: design:update

Request:
  Body:
    name: string (optional)
    description: string (optional)
    images: array (optional)
    attributes: array (optional)
    status: string (optional)

Response 200:
  {
    "success": true,
    "data": {...}
  }
```

#### DELETE /admin/designs/{id}
```yaml
Permission: design:delete

Response 200:
  {
    "success": true,
    "data": {
      "message": "Design deleted successfully"
    }
  }
```

---

### Product APIs

#### POST /admin/products
```yaml
Permission: product:create

Request:
  Body:
    name: string (required)
    sku: string (required, unique)
    design_id: string (required)
    category_id: string (required)
    base_price: int (required, in paise)
    selling_price: int (required, in paise)
    cost_price: int (optional, in paise)
    dimensions:
      length: float
      width: float
      height: float
    weight: int (in grams)
    material: string
    color: string
    size: string
    weaving_type: string
    origin: string
    artisan: string
    craft_type: string
    images: array
    tags: array of strings
    initial_stock: int (optional, initial inventory)
    low_stock_threshold: int (optional, default: 5)

Response 201:
  {
    "success": true,
    "data": {
      "id": "prod_abc123",
      "sku": "SAR-BAN-001",
      "name": "Banarasi Silk Saree - Red",
      "design_id": "dsgn_xyz789",
      "category_id": "cat_def456",
      "base_price": 1500000,
      "selling_price": 1299900,
      "cost_price": 800000,
      "currency": "INR",
      "dimensions": {...},
      "weight": 500,
      "material": "Silk",
      "color": "Red",
      "weaving_type": "Handloom",
      "origin": "Varanasi, UP",
      "artisan": "Master Weaver Ram",
      "craft_type": "Banarasi",
      "images": [...],
      "tags": ["silk", "banarasi", "wedding"],
      "status": "ACTIVE",
      "inventory": {
        "quantity": 10,
        "reserved_qty": 0,
        "available_qty": 10
      },
      "created_at": "2024-01-15T10:00:00Z"
    }
  }
```

#### GET /admin/products
```yaml
Permission: product:read

Request:
  Query:
    page: int
    per_page: int
    category_id: string (optional)
    design_id: string (optional)
    status: string (optional)
    search: string (optional, search name/sku)
    min_price: int (optional)
    max_price: int (optional)
    in_stock: bool (optional)
    low_stock: bool (optional)
    sort_by: string (optional: created_at, price, name)
    sort_order: string (optional: asc, desc)

Response 200:
  {
    "success": true,
    "data": [
      {
        "id": "prod_abc123",
        "sku": "SAR-BAN-001",
        "name": "Banarasi Silk Saree - Red",
        "selling_price": 1299900,
        "status": "ACTIVE",
        "inventory": {
          "quantity": 10,
          "available_qty": 8
        },
        "primary_image": "https://..."
      }
    ],
    "meta": {...}
  }
```

#### GET /admin/products/{id}
```yaml
Permission: product:read

Response 200:
  {
    "success": true,
    "data": {
      "id": "prod_abc123",
      "sku": "SAR-BAN-001",
      "name": "Banarasi Silk Saree - Red",
      ... (full product details),
      "inventory": {
        "quantity": 10,
        "reserved_qty": 2,
        "available_qty": 8,
        "low_stock_threshold": 5,
        "last_restock_at": "2024-01-10T00:00:00Z"
      },
      "category": {
        "id": "cat_def456",
        "name": "Banarasi Sarees"
      },
      "design": {
        "id": "dsgn_xyz789",
        "name": "Floral Pattern"
      }
    }
  }
```

#### PATCH /admin/products/{id}
```yaml
Permission: product:update

Request:
  Body:
    name: string (optional)
    base_price: int (optional)
    selling_price: int (optional)
    cost_price: int (optional)
    ... (any updatable fields)
    status: string (optional)

Response 200:
  {
    "success": true,
    "data": {...}
  }
```

#### DELETE /admin/products/{id}
```yaml
Permission: product:delete

Response 200:
  {
    "success": true,
    "data": {
      "message": "Product deleted successfully"
    }
  }

Response 400:
  {
    "success": false,
    "error": {
      "code": "PRODUCT_HAS_ORDERS",
      "message": "Cannot delete product with pending orders"
    }
  }
```

---

### Pricing Rule APIs

#### POST /admin/pricing/rules
```yaml
Permission: pricing:create
Description: Create a new pricing rule for dynamic price calculation

Request:
  Body:
    name: string (required, e.g., "Silk Bedsheets Area Pricing")
    description: string (optional)
    # Scope - where this rule applies
    scope_type: string (required, GLOBAL/CATEGORY/SUBCATEGORY/PRODUCT/MATERIAL)
    scope_id: string (required for non-GLOBAL scopes, e.g., category_id or product_id)
    category_id: string (optional, for category/product scopes)
    material_name: string (optional, for material-specific rules)
    # Pricing Type
    pricing_type: string (required, AREA_BASED/LENGTH_BASED/FIXED/TIERED/FORMULA)
    # Base Pricing
    base_price: int (required, fixed base price in paise)
    price_per_unit: int (required for AREA/LENGTH based, price per unit in paise)
    unit: string (required, SQ_INCH/SQ_FOOT/SQ_CM/SQ_METER/INCH/CM/FOOT/METER)
    # Material Multipliers
    material_multipliers: object (optional)
      cotton: float (e.g., 1.0)
      silk: float (e.g., 2.5)
      linen: float (e.g., 1.8)
    # Attribute Surcharges
    attribute_surcharges: array (optional)
      - attribute_name: string (e.g., "thread_count")
        attribute_value: string (e.g., "600")
        surcharge_type: string (FIXED/PERCENTAGE)
        surcharge_value: int (in paise or percentage × 100)
    # Tiered Pricing (for TIERED type)
    tiers: array (optional)
      - min_value: float (min area/length)
        max_value: float (max area/length)
        price_per_unit: int (price for this tier)
    # Formula (for FORMULA type)
    formula: string (optional, e.g., "base + (area * rate * material_multiplier) + surcharges")
    # Constraints
    min_area: float (optional)
    max_area: float (optional)
    min_order_value: int (optional, in paise)
    # Status
    priority: int (required, higher wins, e.g., 100)
    is_active: bool (optional, default: true)
    valid_from: string (optional, ISO datetime)
    valid_until: string (optional, ISO datetime)

Response 201:
  {
    "success": true,
    "data": {
      "id": "rule_silk_bedsheets_area",
      "name": "Silk Bedsheets Area Pricing",
      "description": "Area-based pricing for silk bedsheets",
      "scope_type": "CATEGORY",
      "scope_id": "cat_bedsheets",
      "category_id": "cat_bedsheets",
      "pricing_type": "AREA_BASED",
      "base_price": 50000,
      "price_per_unit": 35,
      "unit": "SQ_INCH",
      "material_multipliers": {
        "cotton": 1.0,
        "silk": 2.5,
        "linen": 1.8,
        "blend": 1.3
      },
      "attribute_surcharges": [
        {
          "attribute_name": "thread_count",
          "attribute_value": "400",
          "surcharge_type": "FIXED",
          "surcharge_value": 30000
        },
        {
          "attribute_name": "thread_count",
          "attribute_value": "600",
          "surcharge_type": "FIXED",
          "surcharge_value": 50000
        },
        {
          "attribute_name": "elastic_type",
          "attribute_value": "fitted",
          "surcharge_type": "FIXED",
          "surcharge_value": 15000
        }
      ],
      "min_area": 1000,
      "max_area": 15000,
      "min_order_value": 100000,
      "priority": 100,
      "is_active": true,
      "valid_from": null,
      "valid_until": null,
      "created_at": "2024-01-15T10:00:00Z",
      "created_by": "usr_admin123"
    }
  }
```

#### GET /admin/pricing/rules
```yaml
Permission: pricing:read
Description: List pricing rules with filtering

Request:
  Query:
    page: int (optional, default: 1)
    per_page: int (optional, default: 20)
    scope_type: string (optional, filter by scope)
    category_id: string (optional, filter by category)
    pricing_type: string (optional, filter by type)
    is_active: bool (optional, filter by status)
    search: string (optional, search by name)
    sort_by: string (optional: priority, created_at, name)
    sort_order: string (optional: asc, desc)

Response 200:
  {
    "success": true,
    "data": [
      {
        "id": "rule_silk_bedsheets_area",
        "name": "Silk Bedsheets Area Pricing",
        "scope_type": "CATEGORY",
        "scope_id": "cat_bedsheets",
        "category_name": "Bedsheets",
        "pricing_type": "AREA_BASED",
        "base_price": 50000,
        "price_per_unit": 35,
        "unit": "SQ_INCH",
        "priority": 100,
        "is_active": true,
        "created_at": "2024-01-15T10:00:00Z"
      }
    ],
    "meta": {
      "current_page": 1,
      "per_page": 20,
      "total_count": 15,
      "total_pages": 1
    }
  }
```

#### GET /admin/pricing/rules/{id}
```yaml
Permission: pricing:read
Description: Get a single pricing rule with full details

Response 200:
  {
    "success": true,
    "data": {
      "id": "rule_silk_bedsheets_area",
      "name": "Silk Bedsheets Area Pricing",
      ... (full pricing rule object),
      "usage_stats": {
        "calculations_today": 145,
        "calculations_this_month": 3250,
        "revenue_generated": 4500000
      }
    }
  }
```

#### PATCH /admin/pricing/rules/{id}
```yaml
Permission: pricing:update
Description: Update an existing pricing rule

Request:
  Body:
    name: string (optional)
    description: string (optional)
    base_price: int (optional)
    price_per_unit: int (optional)
    material_multipliers: object (optional, replaces existing)
    attribute_surcharges: array (optional, replaces existing)
    tiers: array (optional, replaces existing)
    priority: int (optional)
    is_active: bool (optional)
    valid_from: string (optional)
    valid_until: string (optional)

Response 200:
  {
    "success": true,
    "data": {...updated pricing rule}
  }

Response 400:
  {
    "success": false,
    "error": {
      "code": "CONFLICTING_PRIORITY",
      "message": "Another active rule with same scope has this priority"
    }
  }
```

#### DELETE /admin/pricing/rules/{id}
```yaml
Permission: pricing:delete
Description: Delete a pricing rule (soft delete)

Response 200:
  {
    "success": true,
    "data": {
      "message": "Pricing rule deleted successfully"
    }
  }

Response 400:
  {
    "success": false,
    "error": {
      "code": "RULE_IS_DEFAULT",
      "message": "Cannot delete a default pricing rule for a category"
    }
  }
```

#### GET /admin/pricing/rules/category/{categoryId}
```yaml
Permission: pricing:read
Description: Get all applicable pricing rules for a category (including inherited)

Response 200:
  {
    "success": true,
    "data": {
      "category_rules": [
        {
          "id": "rule_silk_bedsheets_area",
          "name": "Bedsheets Area Pricing",
          "priority": 100,
          "is_active": true
        }
      ],
      "parent_rules": [
        {
          "id": "rule_bedding_default",
          "name": "Bedding Default Pricing",
          "source_category": "Bedding",
          "priority": 50
        }
      ],
      "global_rules": [
        {
          "id": "rule_global_fallback",
          "name": "Global Fallback",
          "priority": 1
        }
      ],
      "effective_rule": {
        "id": "rule_silk_bedsheets_area",
        "name": "Bedsheets Area Pricing",
        "reason": "Highest priority category-specific rule"
      }
    }
  }
```

---

### Price Calculation APIs (Public/B2C Ready)

#### POST /api/pricing/calculate
```yaml
Description: Calculate price for custom dimensions (customer-facing API)
Rate Limit: 100 requests/minute per IP

Request:
  Body:
    product_id: string (optional, for specific product pricing)
    category_id: string (required if no product_id)
    dimensions: object (required for custom sizing)
      length: float (required)
      width: float (optional, based on category config)
      height: float (optional, for 3D products)
      unit: string (required, inches/cm/feet/meters)
    attributes: object (required, product attributes)
      material: string (e.g., "silk")
      thread_count: string (e.g., "600")
      elastic_type: string (e.g., "fitted")
      ... (other category-specific attributes)
    quantity: int (optional, default: 1)

Response 200:
  {
    "success": true,
    "data": {
      "price_breakdown": {
        "dimensions": {
          "length": 100,
          "width": 90,
          "unit": "inches",
          "area": 9000,
          "area_unit": "sq_inches",
          "area_in_sq_feet": 62.5
        },
        "base_calculation": {
          "price_per_unit": 35,
          "unit": "SQ_INCH",
          "base_cost": 315000
        },
        "material_adjustment": {
          "material": "silk",
          "multiplier": 2.5,
          "adjusted_cost": 787500
        },
        "attribute_surcharges": [
          {
            "attribute": "thread_count",
            "value": "600",
            "surcharge_type": "FIXED",
            "amount": 50000
          },
          {
            "attribute": "elastic_type",
            "value": "fitted",
            "surcharge_type": "FIXED",
            "amount": 15000
          }
        ],
        "surcharges_total": 65000,
        "subtotal_per_unit": 852500,
        "quantity": 1,
        "total": 852500
      },
      "formatted_price": {
        "subtotal": "₹8,525.00",
        "total": "₹8,525.00",
        "currency": "INR"
      },
      "pricing_rule_id": "rule_silk_bedsheets_area",
      "quote_id": "quote_xyz789",
      "quote_valid_until": "2024-01-15T23:59:59Z",
      "dimension_constraints": {
        "min_length": 60,
        "max_length": 120,
        "min_width": 40,
        "max_width": 108,
        "unit": "inches"
      }
    }
  }

Response 400:
  {
    "success": false,
    "error": {
      "code": "DIMENSION_OUT_OF_RANGE",
      "message": "Length must be between 60 and 120 inches",
      "details": {
        "field": "dimensions.length",
        "provided": 150,
        "min": 60,
        "max": 120
      }
    }
  }

Response 400:
  {
    "success": false,
    "error": {
      "code": "MIN_ORDER_VALUE",
      "message": "Minimum order value is ₹1,000",
      "details": {
        "calculated_price": 85000,
        "min_required": 100000
      }
    }
  }
```

#### GET /api/pricing/dimension-options/{categoryId}
```yaml
Description: Get available dimension options and constraints for a category

Response 200:
  {
    "success": true,
    "data": {
      "category_id": "cat_bedsheets",
      "category_name": "Bedsheets",
      "allow_custom_dimensions": true,
      "dimension_config": {
        "length": {
          "enabled": true,
          "min": 60,
          "max": 120,
          "step": 1,
          "unit": "inches",
          "default": 90
        },
        "width": {
          "enabled": true,
          "min": 40,
          "max": 108,
          "step": 1,
          "unit": "inches",
          "default": 60
        },
        "height": {
          "enabled": false
        }
      },
      "pricing_model": "AREA_BASED",
      "standard_sizes": [
        {
          "name": "Single",
          "length": 75,
          "width": 36,
          "unit": "inches",
          "starting_price": 150000
        },
        {
          "name": "Double",
          "length": 75,
          "width": 54,
          "unit": "inches",
          "starting_price": 225000
        },
        {
          "name": "Queen",
          "length": 80,
          "width": 60,
          "unit": "inches",
          "starting_price": 285000
        },
        {
          "name": "King",
          "length": 80,
          "width": 76,
          "unit": "inches",
          "starting_price": 350000
        }
      ],
      "pricing_attributes": [
        {
          "name": "material",
          "label": "Material",
          "type": "SELECT",
          "affects_pricing": true,
          "options": [
            {"value": "cotton", "label": "Cotton", "price_multiplier": 1.0},
            {"value": "silk", "label": "Silk", "price_multiplier": 2.5},
            {"value": "linen", "label": "Linen", "price_multiplier": 1.8}
          ]
        },
        {
          "name": "thread_count",
          "label": "Thread Count",
          "type": "SELECT",
          "affects_pricing": true,
          "options": [
            {"value": "200", "label": "200 TC", "surcharge": 0},
            {"value": "400", "label": "400 TC", "surcharge": 30000},
            {"value": "600", "label": "600 TC", "surcharge": 50000}
          ]
        }
      ],
      "min_order_value": 100000,
      "price_range": {
        "min": 150000,
        "max": 2500000,
        "currency": "INR"
      }
    }
  }
```

#### POST /api/pricing/bulk-calculate
```yaml
Description: Calculate prices for multiple configurations (for comparison UI)
Rate Limit: 20 requests/minute per IP

Request:
  Body:
    category_id: string (required)
    configurations: array (required, max 10)
      - dimensions:
          length: float
          width: float
          unit: string
        attributes:
          material: string
          thread_count: string
        quantity: int

Response 200:
  {
    "success": true,
    "data": {
      "calculations": [
        {
          "configuration_index": 0,
          "dimensions": {"length": 75, "width": 36, "unit": "inches"},
          "attributes": {"material": "cotton", "thread_count": "200"},
          "quantity": 1,
          "price": 150000,
          "formatted_price": "₹1,500.00"
        },
        {
          "configuration_index": 1,
          "dimensions": {"length": 100, "width": 90, "unit": "inches"},
          "attributes": {"material": "silk", "thread_count": "600"},
          "quantity": 1,
          "price": 852500,
          "formatted_price": "₹8,525.00"
        }
      ],
      "quote_id": "quote_bulk_abc123",
      "quote_valid_until": "2024-01-15T23:59:59Z"
    }
  }
```

#### GET /admin/pricing/quotes/{quoteId}
```yaml
Permission: pricing:read
Description: Get details of a generated price quote (for order validation)

Response 200:
  {
    "success": true,
    "data": {
      "quote_id": "quote_xyz789",
      "category_id": "cat_bedsheets",
      "product_id": null,
      "dimensions": {...},
      "attributes": {...},
      "quantity": 1,
      "calculated_price": 852500,
      "pricing_rule_id": "rule_silk_bedsheets_area",
      "price_breakdown": {...},
      "created_at": "2024-01-15T10:00:00Z",
      "valid_until": "2024-01-15T23:59:59Z",
      "is_valid": true,
      "used_in_order": null
    }
  }
```

---

### Inventory APIs

#### GET /admin/inventory/{productId}
```yaml
Permission: inventory:read

Response 200:
  {
    "success": true,
    "data": {
      "product_id": "prod_abc123",
      "product_name": "Banarasi Silk Saree - Red",
      "product_sku": "SAR-BAN-001",
      "quantity": 10,
      "reserved_qty": 2,
      "available_qty": 8,
      "low_stock_threshold": 5,
      "reorder_point": 10,
      "is_low_stock": false,
      "last_restock_at": "2024-01-10T00:00:00Z",
      "updated_at": "2024-01-15T10:00:00Z"
    }
  }
```

#### POST /admin/inventory/{productId}/add
```yaml
Permission: inventory:add

Request:
  Body:
    quantity: int (required, positive)
    reason: string (required)
    reference_id: string (optional, e.g., purchase order)

Response 200:
  {
    "success": true,
    "data": {
      "product_id": "prod_abc123",
      "previous_quantity": 10,
      "added_quantity": 5,
      "new_quantity": 15,
      "available_qty": 13,
      "transaction_id": "invtxn_xyz789",
      "updated_at": "2024-01-15T11:00:00Z"
    }
  }
```

#### POST /admin/inventory/{productId}/remove
```yaml
Permission: inventory:remove

Request:
  Body:
    quantity: int (required, positive)
    reason: string (required)
    reference_id: string (optional)

Response 200:
  {
    "success": true,
    "data": {
      "product_id": "prod_abc123",
      "previous_quantity": 15,
      "removed_quantity": 2,
      "new_quantity": 13,
      "available_qty": 11,
      "transaction_id": "invtxn_abc123",
      "updated_at": "2024-01-15T11:30:00Z"
    }
  }

Response 400:
  {
    "success": false,
    "error": {
      "code": "INSUFFICIENT_STOCK",
      "message": "Cannot remove 20 items. Only 13 available (15 total - 2 reserved)"
    }
  }
```

#### POST /admin/inventory/{productId}/adjust
```yaml
Permission: inventory:adjust

Request:
  Body:
    new_quantity: int (required, non-negative)
    reason: string (required)

Response 200:
  {
    "success": true,
    "data": {
      "product_id": "prod_abc123",
      "previous_quantity": 13,
      "new_quantity": 10,
      "adjustment": -3,
      "available_qty": 8,
      "transaction_id": "invtxn_def456",
      "updated_at": "2024-01-15T12:00:00Z"
    }
  }
```

#### GET /admin/inventory/{productId}/transactions
```yaml
Permission: inventory:read

Request:
  Query:
    page: int
    per_page: int
    type: string (optional: ADD, REMOVE, RESERVE, RELEASE, ADJUST)
    start_date: string (optional, ISO date)
    end_date: string (optional, ISO date)

Response 200:
  {
    "success": true,
    "data": [
      {
        "id": "invtxn_xyz789",
        "type": "ADD",
        "quantity": 5,
        "previous_qty": 10,
        "new_qty": 15,
        "reason": "Restock from artisan",
        "reference_type": "MANUAL",
        "created_at": "2024-01-15T11:00:00Z",
        "created_by": {
          "id": "usr_abc123",
          "name": "Admin User"
        }
      }
    ],
    "meta": {...}
  }
```

#### GET /admin/inventory/low-stock
```yaml
Permission: inventory:read

Request:
  Query:
    page: int
    per_page: int

Response 200:
  {
    "success": true,
    "data": [
      {
        "product_id": "prod_def456",
        "product_name": "Chanderi Cotton Saree",
        "product_sku": "SAR-CHN-002",
        "quantity": 3,
        "available_qty": 2,
        "low_stock_threshold": 5,
        "reorder_point": 10
      }
    ],
    "meta": {...}
  }
```

---

### Order APIs

#### GET /admin/orders
```yaml
Permission: order:read

Request:
  Query:
    page: int
    per_page: int
    status: string (optional, comma-separated for multiple)
    payment_status: string (optional)
    customer_id: string (optional)
    search: string (optional, order number or customer email)
    start_date: string (optional)
    end_date: string (optional)
    sort_by: string (optional: created_at, total_amount)
    sort_order: string (optional: asc, desc)

Response 200:
  {
    "success": true,
    "data": [
      {
        "id": "ord_abc123",
        "order_number": "ORD-2024-00001",
        "customer_name": "John Doe",
        "customer_email": "john@example.com",
        "status": "CONFIRMED",
        "payment_status": "PAID",
        "total_amount": 2599800,
        "currency": "INR",
        "item_count": 2,
        "created_at": "2024-01-15T10:00:00Z"
      }
    ],
    "meta": {...}
  }
```

#### GET /admin/orders/{id}
```yaml
Permission: order:read

Response 200:
  {
    "success": true,
    "data": {
      "id": "ord_abc123",
      "order_number": "ORD-2024-00001",
      "customer": {
        "id": "cust_xyz789",
        "name": "John Doe",
        "email": "john@example.com",
        "phone": "+91-9876543210"
      },
      "shipping_address": {...},
      "billing_address": {...},
      "items": [
        {
          "product_id": "prod_abc123",
          "product_name": "Banarasi Silk Saree - Red",
          "product_sku": "SAR-BAN-001",
          "product_image": "https://...",
          "quantity": 1,
          "unit_price": 1299900,
          "total_price": 1299900,
          "attributes": {
            "color": "Red",
            "size": "Free Size"
          }
        }
      ],
      "subtotal": 2599800,
      "discount_amount": 0,
      "tax_amount": 467964,
      "shipping_amount": 0,
      "total_amount": 3067764,
      "currency": "INR",
      "status": "CONFIRMED",
      "payment_status": "PAID",
      "payment_method": "UPI",
      "payment_id": "pay_xyz789",
      "shipping_method": "Standard",
      "tracking_number": null,
      "coupon_code": null,
      "customer_note": "Please gift wrap",
      "admin_note": "VIP customer",
      "created_at": "2024-01-15T10:00:00Z",
      "confirmed_at": "2024-01-15T10:05:00Z",
      "status_history": [
        {
          "from_status": "PENDING",
          "to_status": "CONFIRMED",
          "changed_by": "system",
          "changed_at": "2024-01-15T10:05:00Z"
        }
      ]
    }
  }
```

#### PATCH /admin/orders/{id}/status
```yaml
Permission: order:update

Request:
  Body:
    status: string (required, valid next status)
    reason: string (optional, required for cancellation)
    note: string (optional)
    notify_customer: bool (optional, default: true)

Response 200:
  {
    "success": true,
    "data": {
      "id": "ord_abc123",
      "order_number": "ORD-2024-00001",
      "previous_status": "CONFIRMED",
      "new_status": "PROCESSING",
      "updated_at": "2024-01-15T14:00:00Z",
      "notification_sent": true
    }
  }

Response 400:
  {
    "success": false,
    "error": {
      "code": "INVALID_STATUS_TRANSITION",
      "message": "Cannot transition from CONFIRMED to DELIVERED"
    }
  }
```

#### PATCH /admin/orders/{id}/tracking
```yaml
Permission: order:update

Request:
  Body:
    tracking_number: string (required)
    shipping_carrier: string (required)

Response 200:
  {
    "success": true,
    "data": {
      "id": "ord_abc123",
      "tracking_number": "TRACK123456",
      "shipping_carrier": "Delhivery",
      "updated_at": "2024-01-15T14:00:00Z"
    }
  }
```

#### POST /admin/orders/{id}/note
```yaml
Permission: order:update

Request:
  Body:
    note: string (required)

Response 200:
  {
    "success": true,
    "data": {
      "id": "ord_abc123",
      "admin_note": "Customer requested delay in delivery",
      "updated_at": "2024-01-15T14:00:00Z"
    }
  }
```

#### POST /admin/orders/{id}/cancel
```yaml
Permission: order:cancel

Request:
  Body:
    reason: string (required)
    notify_customer: bool (optional, default: true)

Response 200:
  {
    "success": true,
    "data": {
      "id": "ord_abc123",
      "order_number": "ORD-2024-00001",
      "status": "CANCELLED",
      "cancelled_at": "2024-01-15T15:00:00Z",
      "cancellation_reason": "Customer requested",
      "inventory_released": true,
      "notification_sent": true
    }
  }
```

#### POST /admin/orders/{id}/refund
```yaml
Permission: order:refund

Request:
  Body:
    amount: int (required, in paise, max = order total)
    reason: string (required)
    notify_customer: bool (optional, default: true)

Response 200:
  {
    "success": true,
    "data": {
      "id": "ord_abc123",
      "order_number": "ORD-2024-00001",
      "refund_amount": 3067764,
      "status": "REFUND_INITIATED",
      "payment_status": "REFUNDED",
      "refund_id": "ref_xyz789",
      "notification_sent": true
    }
  }
```

#### GET /admin/orders/{id}/history
```yaml
Permission: order:read

Response 200:
  {
    "success": true,
    "data": [
      {
        "from_status": "PENDING",
        "to_status": "CONFIRMED",
        "reason": null,
        "note": null,
        "changed_by": {
          "id": "system",
          "name": "System"
        },
        "changed_at": "2024-01-15T10:05:00Z"
      },
      {
        "from_status": "CONFIRMED",
        "to_status": "PROCESSING",
        "reason": null,
        "note": "Starting fulfillment",
        "changed_by": {
          "id": "usr_abc123",
          "name": "Admin User"
        },
        "changed_at": "2024-01-15T14:00:00Z"
      }
    ]
  }
```

---

### Notification APIs

#### POST /admin/notifications/send
```yaml
Permission: notification:send

Request:
  Body:
    type: string (required, enum: EMAIL, SMS)
    recipient_id: string (required, customer ID)
    subject: string (required for EMAIL)
    body: string (required)
    template_id: string (optional)
    template_data: object (optional)
    reference_type: string (optional, ORDER, PRODUCT)
    reference_id: string (optional)

Response 200:
  {
    "success": true,
    "data": {
      "id": "notif_abc123",
      "type": "EMAIL",
      "recipient_id": "cust_xyz789",
      "recipient_email": "john@example.com",
      "subject": "Update on your order",
      "status": "PENDING",
      "created_at": "2024-01-15T16:00:00Z"
    }
  }
```

#### GET /admin/notifications
```yaml
Permission: notification:read

Request:
  Query:
    page: int
    per_page: int
    type: string (optional)
    status: string (optional)
    recipient_id: string (optional)
    reference_type: string (optional)
    reference_id: string (optional)
    start_date: string (optional)
    end_date: string (optional)

Response 200:
  {
    "success": true,
    "data": [
      {
        "id": "notif_abc123",
        "type": "EMAIL",
        "recipient_email": "john@example.com",
        "subject": "Your order has been shipped",
        "status": "SENT",
        "trigger_type": "ORDER_STATUS",
        "reference_type": "ORDER",
        "reference_id": "ord_xyz789",
        "sent_at": "2024-01-15T16:05:00Z",
        "created_at": "2024-01-15T16:00:00Z"
      }
    ],
    "meta": {...}
  }
```

#### GET /admin/notifications/{id}
```yaml
Permission: notification:read

Response 200:
  {
    "success": true,
    "data": {
      "id": "notif_abc123",
      "type": "EMAIL",
      "recipient_id": "cust_xyz789",
      "recipient_email": "john@example.com",
      "subject": "Your order has been shipped",
      "body": "Dear John, Your order ORD-2024-00001 has been shipped...",
      "template_id": "order_shipped",
      "template_data": {...},
      "trigger_type": "ORDER_STATUS",
      "reference_type": "ORDER",
      "reference_id": "ord_xyz789",
      "status": "SENT",
      "sent_at": "2024-01-15T16:05:00Z",
      "created_at": "2024-01-15T16:00:00Z",
      "created_by": "system"
    }
  }
```

---

### Audit Log APIs

#### GET /admin/audit-logs
```yaml
Permission: audit:read (ADMIN only)

Request:
  Query:
    page: int
    per_page: int
    user_id: string (optional)
    entity_type: string (optional)
    entity_id: string (optional)
    action: string (optional)
    start_date: string (optional)
    end_date: string (optional)

Response 200:
  {
    "success": true,
    "data": [
      {
        "id": "audit_abc123",
        "user": {
          "id": "usr_xyz789",
          "email": "admin@example.com",
          "role": "ADMIN"
        },
        "action": "UPDATE",
        "entity_type": "ORDER",
        "entity_id": "ord_def456",
        "changes": {
          "status": {
            "old_value": "CONFIRMED",
            "new_value": "PROCESSING"
          }
        },
        "ip_address": "192.168.1.1",
        "created_at": "2024-01-15T14:00:00Z"
      }
    ],
    "meta": {...}
  }
```

---

## Additional Features

### 1. Dashboard & Analytics

#### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Dashboard Architecture                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────────────────┐     │
│  │   DynamoDB   │────▶│  DynamoDB    │────▶│   Analytics Lambda       │     │
│  │   Streams    │     │  Streams     │     │   (Aggregates data)      │     │
│  └──────────────┘     └──────────────┘     └──────────────────────────┘     │
│                                                      │                       │
│                                                      ▼                       │
│  ┌──────────────────────────────────────────────────────────────────┐       │
│  │                    Analytics Table (DynamoDB)                     │       │
│  │  ┌─────────────────┬─────────────────┬─────────────────────────┐ │       │
│  │  │ METRICS#DAILY   │ 2024-01-15      │ orders, revenue, items  │ │       │
│  │  │ METRICS#WEEKLY  │ 2024-W03        │ aggregated weekly       │ │       │
│  │  │ METRICS#MONTHLY │ 2024-01         │ aggregated monthly      │ │       │
│  │  │ TOP_PRODUCTS    │ 2024-01-15      │ top 10 by sales         │ │       │
│  │  │ INVENTORY_ALERT │ LOW_STOCK       │ products below threshold│ │       │
│  │  └─────────────────┴─────────────────┴─────────────────────────┘ │       │
│  └──────────────────────────────────────────────────────────────────┘       │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### Entity Definitions

```go
// ==================== ANALYTICS ENTITIES ====================

type DailyMetrics struct {
    PK               string    `json:"-" dynamodbav:"PK"`           // METRICS#DAILY
    SK               string    `json:"-" dynamodbav:"SK"`           // 2024-01-15
    EntityType       string    `json:"-" dynamodbav:"entity_type"`  // DAILY_METRICS

    Date             string    `json:"date" dynamodbav:"date"`

    // Order Metrics
    TotalOrders      int       `json:"total_orders" dynamodbav:"total_orders"`
    PendingOrders    int       `json:"pending_orders" dynamodbav:"pending_orders"`
    ConfirmedOrders  int       `json:"confirmed_orders" dynamodbav:"confirmed_orders"`
    ShippedOrders    int       `json:"shipped_orders" dynamodbav:"shipped_orders"`
    DeliveredOrders  int       `json:"delivered_orders" dynamodbav:"delivered_orders"`
    CancelledOrders  int       `json:"cancelled_orders" dynamodbav:"cancelled_orders"`

    // Revenue Metrics (in paise)
    TotalRevenue     int64     `json:"total_revenue" dynamodbav:"total_revenue"`
    AvgOrderValue    int64     `json:"avg_order_value" dynamodbav:"avg_order_value"`
    TotalRefunds     int64     `json:"total_refunds" dynamodbav:"total_refunds"`
    NetRevenue       int64     `json:"net_revenue" dynamodbav:"net_revenue"`

    // Product Metrics
    TotalItemsSold   int       `json:"total_items_sold" dynamodbav:"total_items_sold"`
    UniqueProducts   int       `json:"unique_products" dynamodbav:"unique_products"`

    // Inventory Metrics
    LowStockCount    int       `json:"low_stock_count" dynamodbav:"low_stock_count"`
    OutOfStockCount  int       `json:"out_of_stock_count" dynamodbav:"out_of_stock_count"`

    UpdatedAt        time.Time `json:"updated_at" dynamodbav:"updated_at"`
}

type TopProduct struct {
    ProductID     string `json:"product_id" dynamodbav:"product_id"`
    ProductName   string `json:"product_name" dynamodbav:"product_name"`
    ProductSKU    string `json:"product_sku" dynamodbav:"product_sku"`
    TotalQuantity int    `json:"total_quantity" dynamodbav:"total_quantity"`
    TotalRevenue  int64  `json:"total_revenue" dynamodbav:"total_revenue"`
    OrderCount    int    `json:"order_count" dynamodbav:"order_count"`
}

type InventoryAlert struct {
    PK            string    `json:"-" dynamodbav:"PK"`           // INVENTORY_ALERT
    SK            string    `json:"-" dynamodbav:"SK"`           // PRODUCT#<id>
    EntityType    string    `json:"-" dynamodbav:"entity_type"`  // INVENTORY_ALERT

    ProductID     string    `json:"product_id" dynamodbav:"product_id"`
    ProductName   string    `json:"product_name" dynamodbav:"product_name"`
    ProductSKU    string    `json:"product_sku" dynamodbav:"product_sku"`
    AlertType     string    `json:"alert_type" dynamodbav:"alert_type"` // LOW_STOCK, OUT_OF_STOCK
    CurrentQty    int       `json:"current_qty" dynamodbav:"current_qty"`
    Threshold     int       `json:"threshold" dynamodbav:"threshold"`
    CreatedAt     time.Time `json:"created_at" dynamodbav:"created_at"`
}
```

#### Dashboard Service Interface

```go
type DashboardService interface {
    // Overview
    GetDashboardSummary(ctx context.Context) (*DashboardSummary, error)

    // Sales Metrics
    GetSalesMetrics(ctx context.Context, req SalesMetricsRequest) (*SalesMetricsResponse, error)
    GetRevenueChart(ctx context.Context, period string, startDate, endDate time.Time) (*RevenueChartData, error)

    // Order Metrics
    GetOrderStatusDistribution(ctx context.Context) (*OrderStatusDistribution, error)
    GetRecentOrders(ctx context.Context, limit int) ([]*OrderSummary, error)

    // Product Metrics
    GetTopSellingProducts(ctx context.Context, period string, limit int) ([]*TopProduct, error)
    GetLowPerformingProducts(ctx context.Context, limit int) ([]*ProductPerformance, error)

    // Inventory Alerts
    GetInventoryAlerts(ctx context.Context) (*InventoryAlerts, error)
    GetLowStockProducts(ctx context.Context) ([]*InventoryAlert, error)
    GetOutOfStockProducts(ctx context.Context) ([]*InventoryAlert, error)

    // Artisan Metrics
    GetTopArtisans(ctx context.Context, period string, limit int) ([]*ArtisanPerformance, error)
}
```

#### Dashboard APIs

##### GET /admin/dashboard/summary
```yaml
Permission: dashboard:read

Response 200:
  {
    "success": true,
    "data": {
      "today": {
        "orders": 45,
        "revenue": 1250000,
        "items_sold": 52,
        "new_customers": 12
      },
      "comparison": {
        "orders_change": 15.5,
        "revenue_change": 22.3,
        "items_change": 18.2
      },
      "inventory_alerts": {
        "low_stock": 8,
        "out_of_stock": 2
      },
      "pending_actions": {
        "orders_to_process": 12,
        "returns_pending": 3,
        "refunds_pending": 1
      }
    }
  }
```

##### GET /admin/dashboard/sales
```yaml
Permission: dashboard:read

Request:
  Query:
    period: string (required: daily, weekly, monthly)
    start_date: string (ISO date)
    end_date: string (ISO date)

Response 200:
  {
    "success": true,
    "data": {
      "period": "daily",
      "start_date": "2024-01-01",
      "end_date": "2024-01-15",
      "total_revenue": 45000000,
      "total_orders": 320,
      "avg_order_value": 140625,
      "chart_data": [
        {"date": "2024-01-01", "revenue": 3200000, "orders": 22},
        {"date": "2024-01-02", "revenue": 2800000, "orders": 18},
        ...
      ]
    }
  }
```

##### GET /admin/dashboard/top-products
```yaml
Permission: dashboard:read

Request:
  Query:
    period: string (optional: week, month, year. default: month)
    limit: int (optional, default: 10, max: 50)

Response 200:
  {
    "success": true,
    "data": [
      {
        "rank": 1,
        "product_id": "prod_abc123",
        "product_name": "Banarasi Silk Saree - Red",
        "product_sku": "SAR-BAN-001",
        "product_image": "https://...",
        "total_quantity": 45,
        "total_revenue": 5849550,
        "order_count": 42
      },
      ...
    ]
  }
```

##### GET /admin/dashboard/inventory-alerts
```yaml
Permission: dashboard:read

Response 200:
  {
    "success": true,
    "data": {
      "low_stock": [
        {
          "product_id": "prod_def456",
          "product_name": "Chanderi Cotton Saree",
          "product_sku": "SAR-CHN-002",
          "current_qty": 3,
          "threshold": 5,
          "days_of_stock": 2
        }
      ],
      "out_of_stock": [
        {
          "product_id": "prod_ghi789",
          "product_name": "Kanjivaram Silk Saree",
          "product_sku": "SAR-KAN-001",
          "last_sold_at": "2024-01-10T14:00:00Z",
          "pending_orders": 3
        }
      ]
    }
  }
```

---

### 2. Bulk Operations

#### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Bulk Operations Flow                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────────────────┐     │
│  │  Admin UI    │────▶│  S3 Upload   │────▶│   Bulk Processor Lambda  │     │
│  │  (CSV File)  │     │  (Pre-signed │     │   - Validates CSV        │     │
│  └──────────────┘     │   URL)       │     │   - Processes in batches │     │
│                       └──────────────┘     │   - Updates DynamoDB     │     │
│                                            └──────────────────────────┘     │
│                                                      │                       │
│                                                      ▼                       │
│  ┌──────────────────────────────────────────────────────────────────┐       │
│  │                    Bulk Job Status Table                          │       │
│  │  ┌─────────────────┬─────────────────┬─────────────────────────┐ │       │
│  │  │ BULK_JOB#123    │ METADATA        │ status, progress, errors│ │       │
│  │  │ BULK_JOB#123    │ ERROR#1         │ row, field, message     │ │       │
│  │  │ BULK_JOB#123    │ ERROR#2         │ row, field, message     │ │       │
│  │  └─────────────────┴─────────────────┴─────────────────────────┘ │       │
│  └──────────────────────────────────────────────────────────────────┘       │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### Entity Definitions

```go
// ==================== BULK OPERATION ENTITIES ====================

type BulkJobType string

const (
    BulkJobTypeProductImport    BulkJobType = "PRODUCT_IMPORT"
    BulkJobTypeProductExport    BulkJobType = "PRODUCT_EXPORT"
    BulkJobTypeInventoryUpdate  BulkJobType = "INVENTORY_UPDATE"
    BulkJobTypePriceUpdate      BulkJobType = "PRICE_UPDATE"
    BulkJobTypeOrderExport      BulkJobType = "ORDER_EXPORT"
)

type BulkJobStatus string

const (
    BulkJobStatusPending    BulkJobStatus = "PENDING"
    BulkJobStatusProcessing BulkJobStatus = "PROCESSING"
    BulkJobStatusCompleted  BulkJobStatus = "COMPLETED"
    BulkJobStatusFailed     BulkJobStatus = "FAILED"
    BulkJobStatusPartial    BulkJobStatus = "PARTIAL_SUCCESS"
)

type BulkJob struct {
    ID             string        `json:"id" dynamodbav:"id"`
    PK             string        `json:"-" dynamodbav:"PK"`           // BULK_JOB#<id>
    SK             string        `json:"-" dynamodbav:"SK"`           // METADATA
    GSI1PK         string        `json:"-" dynamodbav:"GSI1PK"`       // USER#<user_id>
    GSI1SK         string        `json:"-" dynamodbav:"GSI1SK"`       // <created_at>
    EntityType     string        `json:"-" dynamodbav:"entity_type"`  // BULK_JOB

    Type           BulkJobType   `json:"type" dynamodbav:"type"`
    Status         BulkJobStatus `json:"status" dynamodbav:"status"`

    // File Info
    FileName       string        `json:"file_name" dynamodbav:"file_name"`
    FileURL        string        `json:"file_url" dynamodbav:"file_url"`
    FileSize       int64         `json:"file_size" dynamodbav:"file_size"`

    // Progress
    TotalRows      int           `json:"total_rows" dynamodbav:"total_rows"`
    ProcessedRows  int           `json:"processed_rows" dynamodbav:"processed_rows"`
    SuccessCount   int           `json:"success_count" dynamodbav:"success_count"`
    ErrorCount     int           `json:"error_count" dynamodbav:"error_count"`

    // Result
    ResultFileURL  string        `json:"result_file_url" dynamodbav:"result_file_url"`

    CreatedAt      time.Time     `json:"created_at" dynamodbav:"created_at"`
    StartedAt      *time.Time    `json:"started_at" dynamodbav:"started_at"`
    CompletedAt    *time.Time    `json:"completed_at" dynamodbav:"completed_at"`
    CreatedBy      string        `json:"created_by" dynamodbav:"created_by"`
}

type BulkJobError struct {
    PK         string `json:"-" dynamodbav:"PK"`           // BULK_JOB#<job_id>
    SK         string `json:"-" dynamodbav:"SK"`           // ERROR#<row_number>

    JobID      string `json:"job_id" dynamodbav:"job_id"`
    RowNumber  int    `json:"row_number" dynamodbav:"row_number"`
    Field      string `json:"field" dynamodbav:"field"`
    Value      string `json:"value" dynamodbav:"value"`
    ErrorCode  string `json:"error_code" dynamodbav:"error_code"`
    Message    string `json:"message" dynamodbav:"message"`
}
```

#### Bulk Operations Service Interface

```go
type BulkOperationService interface {
    // Import Operations
    InitiateProductImport(ctx context.Context, req InitiateImportRequest, userID string) (*BulkJob, error)
    InitiateInventoryUpdate(ctx context.Context, req InitiateImportRequest, userID string) (*BulkJob, error)
    InitiatePriceUpdate(ctx context.Context, req InitiateImportRequest, userID string) (*BulkJob, error)

    // Export Operations
    InitiateProductExport(ctx context.Context, req ExportRequest, userID string) (*BulkJob, error)
    InitiateOrderExport(ctx context.Context, req OrderExportRequest, userID string) (*BulkJob, error)
    InitiateInventoryExport(ctx context.Context, userID string) (*BulkJob, error)

    // Job Management
    GetBulkJob(ctx context.Context, jobID string) (*BulkJobDetail, error)
    ListBulkJobs(ctx context.Context, req ListBulkJobsRequest) (*ListBulkJobsResponse, error)
    GetBulkJobErrors(ctx context.Context, jobID string, pagination PaginationRequest) (*ListBulkJobErrorsResponse, error)
    CancelBulkJob(ctx context.Context, jobID string) error

    // Templates
    GetImportTemplate(ctx context.Context, templateType string) (*ImportTemplate, error)
}
```

#### Bulk Operations APIs

##### POST /admin/bulk/upload-url
```yaml
Permission: bulk:write

Request:
  Body:
    type: string (required: PRODUCT_IMPORT, INVENTORY_UPDATE, PRICE_UPDATE)
    file_name: string (required)
    content_type: string (required, must be text/csv)

Response 200:
  {
    "success": true,
    "data": {
      "upload_url": "https://s3.amazonaws.com/bucket/uploads/...",
      "job_id": "bulk_abc123",
      "expires_in": 3600
    }
  }
```

##### POST /admin/bulk/products/import
```yaml
Permission: bulk:write

Request:
  Body:
    job_id: string (required, from upload-url)
    options:
      update_existing: bool (default: false)
      skip_invalid: bool (default: true)

Response 202:
  {
    "success": true,
    "data": {
      "job_id": "bulk_abc123",
      "status": "PENDING",
      "message": "Import job queued for processing"
    }
  }
```

##### POST /admin/bulk/inventory/update
```yaml
Permission: bulk:write

Request:
  Body:
    job_id: string (required)
    options:
      operation: string (required: SET, ADD, SUBTRACT)

Response 202:
  {
    "success": true,
    "data": {
      "job_id": "bulk_def456",
      "status": "PENDING",
      "message": "Inventory update job queued"
    }
  }
```

##### POST /admin/bulk/products/export
```yaml
Permission: bulk:read

Request:
  Body:
    filters:
      category_ids: array of strings (optional)
      status: array of strings (optional)
      in_stock: bool (optional)
    fields: array of strings (optional, specific fields to export)
    format: string (optional: csv, xlsx. default: csv)

Response 202:
  {
    "success": true,
    "data": {
      "job_id": "bulk_ghi789",
      "status": "PENDING",
      "message": "Export job queued"
    }
  }
```

##### GET /admin/bulk/jobs/{id}
```yaml
Permission: bulk:read

Response 200:
  {
    "success": true,
    "data": {
      "id": "bulk_abc123",
      "type": "PRODUCT_IMPORT",
      "status": "PROCESSING",
      "file_name": "products_jan_2024.csv",
      "progress": {
        "total_rows": 500,
        "processed_rows": 234,
        "success_count": 230,
        "error_count": 4,
        "percentage": 46.8
      },
      "created_at": "2024-01-15T10:00:00Z",
      "started_at": "2024-01-15T10:00:05Z"
    }
  }
```

##### GET /admin/bulk/jobs/{id}/errors
```yaml
Permission: bulk:read

Response 200:
  {
    "success": true,
    "data": [
      {
        "row_number": 45,
        "field": "sku",
        "value": "SAR-BAN-001",
        "error_code": "DUPLICATE_SKU",
        "message": "SKU already exists in database"
      },
      {
        "row_number": 78,
        "field": "selling_price",
        "value": "abc",
        "error_code": "INVALID_NUMBER",
        "message": "selling_price must be a valid number"
      }
    ],
    "meta": {...}
  }
```

##### GET /admin/bulk/templates/{type}
```yaml
Permission: bulk:read

Request:
  Path:
    type: string (PRODUCT_IMPORT, INVENTORY_UPDATE, PRICE_UPDATE)

Response 200:
  {
    "success": true,
    "data": {
      "type": "PRODUCT_IMPORT",
      "download_url": "https://...",
      "fields": [
        {"name": "sku", "required": true, "type": "string", "description": "Unique product identifier"},
        {"name": "name", "required": true, "type": "string", "description": "Product name"},
        {"name": "selling_price", "required": true, "type": "number", "description": "Price in paise"},
        ...
      ],
      "sample_rows": [
        {"sku": "SAR-BAN-001", "name": "Banarasi Silk Saree", "selling_price": "1299900", ...}
      ]
    }
  }
```

---

### 3. Image/Asset Management

#### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Image Upload Flow                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────────────────┐     │
│  │  Admin UI    │────▶│  Lambda      │────▶│   S3 Pre-signed URL      │     │
│  │  (Image)     │     │  (Get URL)   │     │   (Direct Upload)        │     │
│  └──────────────┘     └──────────────┘     └──────────────────────────┘     │
│                                                      │                       │
│                                                      ▼                       │
│  ┌──────────────────────────────────────────────────────────────────┐       │
│  │              S3 Event → Image Processor Lambda                    │       │
│  │  ┌─────────────────────────────────────────────────────────────┐ │       │
│  │  │ 1. Validate image (size, type, dimensions)                  │ │       │
│  │  │ 2. Generate thumbnails (150x150, 300x300, 600x600)          │ │       │
│  │  │ 3. Optimize original (compress, strip metadata)             │ │       │
│  │  │ 4. Store in CDN bucket                                      │ │       │
│  │  │ 5. Update DynamoDB with URLs                                │ │       │
│  │  └─────────────────────────────────────────────────────────────┘ │       │
│  └──────────────────────────────────────────────────────────────────┘       │
│                              │                                               │
│                              ▼                                               │
│  ┌──────────────────────────────────────────────────────────────────┐       │
│  │                     CloudFront CDN                                │       │
│  │        https://cdn.yourstore.com/images/products/...              │       │
│  └──────────────────────────────────────────────────────────────────┘       │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### Entity Definitions

```go
// ==================== ASSET ENTITIES ====================

type AssetType string

const (
    AssetTypeProductImage  AssetType = "PRODUCT_IMAGE"
    AssetTypeCategoryImage AssetType = "CATEGORY_IMAGE"
    AssetTypeDesignImage   AssetType = "DESIGN_IMAGE"
    AssetTypeArtisanImage  AssetType = "ARTISAN_IMAGE"
)

type AssetStatus string

const (
    AssetStatusPending    AssetStatus = "PENDING"
    AssetStatusProcessing AssetStatus = "PROCESSING"
    AssetStatusReady      AssetStatus = "READY"
    AssetStatusFailed     AssetStatus = "FAILED"
)

type Asset struct {
    ID             string      `json:"id" dynamodbav:"id"`
    PK             string      `json:"-" dynamodbav:"PK"`           // ASSET#<id>
    SK             string      `json:"-" dynamodbav:"SK"`           // METADATA
    GSI1PK         string      `json:"-" dynamodbav:"GSI1PK"`       // ENTITY#<entity_type>#<entity_id>
    EntityType     string      `json:"-" dynamodbav:"entity_type"`  // ASSET

    Type           AssetType   `json:"type" dynamodbav:"type"`
    Status         AssetStatus `json:"status" dynamodbav:"status"`

    // Original File
    OriginalName   string      `json:"original_name" dynamodbav:"original_name"`
    MimeType       string      `json:"mime_type" dynamodbav:"mime_type"`
    FileSize       int64       `json:"file_size" dynamodbav:"file_size"`

    // Dimensions
    Width          int         `json:"width" dynamodbav:"width"`
    Height         int         `json:"height" dynamodbav:"height"`

    // URLs (after processing)
    URLs           AssetURLs   `json:"urls" dynamodbav:"urls"`

    // Association
    EntityType2    string      `json:"entity_type" dynamodbav:"entity_type_2"` // PRODUCT, CATEGORY, etc.
    EntityID       string      `json:"entity_id" dynamodbav:"entity_id"`
    IsPrimary      bool        `json:"is_primary" dynamodbav:"is_primary"`
    SortOrder      int         `json:"sort_order" dynamodbav:"sort_order"`
    AltText        string      `json:"alt_text" dynamodbav:"alt_text"`

    CreatedAt      time.Time   `json:"created_at" dynamodbav:"created_at"`
    ProcessedAt    *time.Time  `json:"processed_at" dynamodbav:"processed_at"`
    CreatedBy      string      `json:"created_by" dynamodbav:"created_by"`
}

type AssetURLs struct {
    Original  string `json:"original" dynamodbav:"original"`
    Large     string `json:"large" dynamodbav:"large"`       // 1200px
    Medium    string `json:"medium" dynamodbav:"medium"`     // 600px
    Small     string `json:"small" dynamodbav:"small"`       // 300px
    Thumbnail string `json:"thumbnail" dynamodbav:"thumbnail"` // 150px
}
```

#### Asset Service Interface

```go
type AssetService interface {
    // Upload
    GetUploadURL(ctx context.Context, req GetUploadURLRequest, userID string) (*UploadURLResponse, error)
    ConfirmUpload(ctx context.Context, assetID string, entityType, entityID string) (*Asset, error)

    // Management
    GetAsset(ctx context.Context, assetID string) (*Asset, error)
    GetAssetsByEntity(ctx context.Context, entityType, entityID string) ([]*Asset, error)
    UpdateAsset(ctx context.Context, assetID string, req UpdateAssetRequest) (*Asset, error)
    DeleteAsset(ctx context.Context, assetID string) error

    // Ordering
    ReorderAssets(ctx context.Context, entityType, entityID string, assetIDs []string) error
    SetPrimaryAsset(ctx context.Context, entityType, entityID, assetID string) error
}
```

#### Asset APIs

##### POST /admin/assets/upload-url
```yaml
Permission: asset:write

Request:
  Body:
    file_name: string (required)
    content_type: string (required, image/jpeg, image/png, image/webp)
    entity_type: string (required: PRODUCT, CATEGORY, DESIGN, ARTISAN)
    entity_id: string (required)

Response 200:
  {
    "success": true,
    "data": {
      "asset_id": "asset_abc123",
      "upload_url": "https://s3.amazonaws.com/bucket/uploads/...",
      "expires_in": 3600,
      "max_size_bytes": 10485760
    }
  }
```

##### POST /admin/assets/{id}/confirm
```yaml
Permission: asset:write

Request:
  Body:
    alt_text: string (optional)
    is_primary: bool (optional, default: false)

Response 200:
  {
    "success": true,
    "data": {
      "id": "asset_abc123",
      "status": "PROCESSING",
      "message": "Image uploaded, processing will complete shortly"
    }
  }
```

##### GET /admin/assets/entity/{entityType}/{entityId}
```yaml
Permission: asset:read

Response 200:
  {
    "success": true,
    "data": [
      {
        "id": "asset_abc123",
        "status": "READY",
        "urls": {
          "original": "https://cdn.../original/abc123.jpg",
          "large": "https://cdn.../large/abc123.jpg",
          "medium": "https://cdn.../medium/abc123.jpg",
          "small": "https://cdn.../small/abc123.jpg",
          "thumbnail": "https://cdn.../thumb/abc123.jpg"
        },
        "width": 2000,
        "height": 3000,
        "file_size": 524288,
        "is_primary": true,
        "sort_order": 0,
        "alt_text": "Red Banarasi silk saree with gold zari work"
      },
      ...
    ]
  }
```

##### PUT /admin/assets/entity/{entityType}/{entityId}/reorder
```yaml
Permission: asset:write

Request:
  Body:
    asset_ids: array of strings (required, in desired order)

Response 200:
  {
    "success": true,
    "data": {
      "message": "Assets reordered successfully"
    }
  }
```

##### DELETE /admin/assets/{id}
```yaml
Permission: asset:delete

Response 200:
  {
    "success": true,
    "data": {
      "message": "Asset deleted successfully"
    }
  }
```

---

### 4. Coupon/Discount Management

#### Entity Definitions

```go
// ==================== COUPON ENTITIES ====================

type CouponType string

const (
    CouponTypePercentage CouponType = "PERCENTAGE"
    CouponTypeFlat       CouponType = "FLAT"
    CouponTypeFreeShipping CouponType = "FREE_SHIPPING"
    CouponTypeBuyXGetY   CouponType = "BUY_X_GET_Y"
)

type CouponStatus string

const (
    CouponStatusActive    CouponStatus = "ACTIVE"
    CouponStatusInactive  CouponStatus = "INACTIVE"
    CouponStatusExpired   CouponStatus = "EXPIRED"
    CouponStatusExhausted CouponStatus = "EXHAUSTED"
)

type Coupon struct {
    ID              string       `json:"id" dynamodbav:"id"`
    PK              string       `json:"-" dynamodbav:"PK"`           // COUPON#<id>
    SK              string       `json:"-" dynamodbav:"SK"`           // METADATA
    GSI1PK          string       `json:"-" dynamodbav:"GSI1PK"`       // CODE#<code>
    GSI2PK          string       `json:"-" dynamodbav:"GSI2PK"`       // STATUS#<status>
    EntityType      string       `json:"-" dynamodbav:"entity_type"`  // COUPON

    Code            string       `json:"code" dynamodbav:"code"`
    Name            string       `json:"name" dynamodbav:"name"`
    Description     string       `json:"description" dynamodbav:"description"`

    Type            CouponType   `json:"type" dynamodbav:"type"`
    Status          CouponStatus `json:"status" dynamodbav:"status"`

    // Discount Value
    DiscountValue   int64        `json:"discount_value" dynamodbav:"discount_value"` // Percentage (1-100) or Amount in paise
    MaxDiscount     int64        `json:"max_discount" dynamodbav:"max_discount"`     // Cap for percentage discounts

    // Conditions
    MinOrderValue   int64        `json:"min_order_value" dynamodbav:"min_order_value"`
    MinQuantity     int          `json:"min_quantity" dynamodbav:"min_quantity"`

    // Applicability
    ApplicableTo    string       `json:"applicable_to" dynamodbav:"applicable_to"` // ALL, CATEGORIES, PRODUCTS
    CategoryIDs     []string     `json:"category_ids" dynamodbav:"category_ids"`
    ProductIDs      []string     `json:"product_ids" dynamodbav:"product_ids"`
    ExcludedProductIDs []string  `json:"excluded_product_ids" dynamodbav:"excluded_product_ids"`

    // Usage Limits
    TotalUsageLimit int          `json:"total_usage_limit" dynamodbav:"total_usage_limit"`
    PerUserLimit    int          `json:"per_user_limit" dynamodbav:"per_user_limit"`
    CurrentUsage    int          `json:"current_usage" dynamodbav:"current_usage"`

    // Validity
    StartsAt        time.Time    `json:"starts_at" dynamodbav:"starts_at"`
    ExpiresAt       time.Time    `json:"expires_at" dynamodbav:"expires_at"`

    // Flags
    IsFirstOrderOnly bool        `json:"is_first_order_only" dynamodbav:"is_first_order_only"`
    IsStackable      bool        `json:"is_stackable" dynamodbav:"is_stackable"`
    IsPublic         bool        `json:"is_public" dynamodbav:"is_public"` // Show on website

    CreatedAt       time.Time    `json:"created_at" dynamodbav:"created_at"`
    UpdatedAt       time.Time    `json:"updated_at" dynamodbav:"updated_at"`
    CreatedBy       string       `json:"created_by" dynamodbav:"created_by"`
}

type CouponUsage struct {
    PK          string    `json:"-" dynamodbav:"PK"`           // COUPON#<coupon_id>
    SK          string    `json:"-" dynamodbav:"SK"`           // USAGE#<order_id>
    EntityType  string    `json:"-" dynamodbav:"entity_type"`  // COUPON_USAGE

    CouponID    string    `json:"coupon_id" dynamodbav:"coupon_id"`
    CouponCode  string    `json:"coupon_code" dynamodbav:"coupon_code"`
    OrderID     string    `json:"order_id" dynamodbav:"order_id"`
    CustomerID  string    `json:"customer_id" dynamodbav:"customer_id"`

    OrderValue  int64     `json:"order_value" dynamodbav:"order_value"`
    DiscountAmount int64  `json:"discount_amount" dynamodbav:"discount_amount"`

    UsedAt      time.Time `json:"used_at" dynamodbav:"used_at"`
}
```

#### Coupon Service Interface

```go
type CouponService interface {
    // CRUD
    CreateCoupon(ctx context.Context, req CreateCouponRequest, userID string) (*Coupon, error)
    GetCoupon(ctx context.Context, couponID string) (*Coupon, error)
    GetCouponByCode(ctx context.Context, code string) (*Coupon, error)
    ListCoupons(ctx context.Context, req ListCouponsRequest) (*ListCouponsResponse, error)
    UpdateCoupon(ctx context.Context, couponID string, req UpdateCouponRequest) (*Coupon, error)
    DeleteCoupon(ctx context.Context, couponID string) error

    // Status Management
    ActivateCoupon(ctx context.Context, couponID string) (*Coupon, error)
    DeactivateCoupon(ctx context.Context, couponID string) (*Coupon, error)

    // Validation (for B2C, but admin might test)
    ValidateCoupon(ctx context.Context, code string, orderValue int64, customerID string) (*CouponValidationResult, error)

    // Usage Tracking
    GetCouponUsage(ctx context.Context, couponID string, pagination PaginationRequest) (*ListCouponUsageResponse, error)
    GetCouponStats(ctx context.Context, couponID string) (*CouponStats, error)
}
```

#### Coupon APIs

##### POST /admin/coupons
```yaml
Permission: coupon:create

Request:
  Body:
    code: string (required, unique, uppercase)
    name: string (required)
    description: string (optional)
    type: string (required: PERCENTAGE, FLAT, FREE_SHIPPING)
    discount_value: int (required, percentage 1-100 or amount in paise)
    max_discount: int (optional, for percentage coupons)
    min_order_value: int (optional, in paise)
    applicable_to: string (optional: ALL, CATEGORIES, PRODUCTS. default: ALL)
    category_ids: array (optional, if applicable_to = CATEGORIES)
    product_ids: array (optional, if applicable_to = PRODUCTS)
    total_usage_limit: int (optional, 0 = unlimited)
    per_user_limit: int (optional, default: 1)
    starts_at: string (required, ISO datetime)
    expires_at: string (required, ISO datetime)
    is_first_order_only: bool (optional, default: false)
    is_public: bool (optional, default: false)

Response 201:
  {
    "success": true,
    "data": {
      "id": "cpn_abc123",
      "code": "WELCOME20",
      "name": "Welcome Discount",
      "type": "PERCENTAGE",
      "discount_value": 20,
      "max_discount": 50000,
      "min_order_value": 100000,
      "status": "ACTIVE",
      "starts_at": "2024-01-01T00:00:00Z",
      "expires_at": "2024-12-31T23:59:59Z",
      "created_at": "2024-01-15T10:00:00Z"
    }
  }
```

##### GET /admin/coupons
```yaml
Permission: coupon:read

Request:
  Query:
    page: int
    per_page: int
    status: string (optional)
    type: string (optional)
    search: string (optional, search code/name)
    is_expired: bool (optional)

Response 200:
  {
    "success": true,
    "data": [
      {
        "id": "cpn_abc123",
        "code": "WELCOME20",
        "name": "Welcome Discount",
        "type": "PERCENTAGE",
        "discount_value": 20,
        "status": "ACTIVE",
        "usage": {
          "current": 45,
          "limit": 100
        },
        "expires_at": "2024-12-31T23:59:59Z"
      }
    ],
    "meta": {...}
  }
```

##### GET /admin/coupons/{id}
```yaml
Permission: coupon:read

Response 200:
  {
    "success": true,
    "data": {
      "id": "cpn_abc123",
      "code": "WELCOME20",
      "name": "Welcome Discount",
      "description": "20% off for new customers",
      "type": "PERCENTAGE",
      "discount_value": 20,
      "max_discount": 50000,
      "min_order_value": 100000,
      "applicable_to": "ALL",
      "total_usage_limit": 100,
      "per_user_limit": 1,
      "current_usage": 45,
      "status": "ACTIVE",
      "is_first_order_only": true,
      "is_public": true,
      "starts_at": "2024-01-01T00:00:00Z",
      "expires_at": "2024-12-31T23:59:59Z",
      "stats": {
        "total_orders": 45,
        "total_discount_given": 1850000,
        "avg_order_value": 285000
      }
    }
  }
```

##### GET /admin/coupons/{id}/usage
```yaml
Permission: coupon:read

Response 200:
  {
    "success": true,
    "data": [
      {
        "order_id": "ord_xyz789",
        "order_number": "ORD-2024-00045",
        "customer_email": "customer@example.com",
        "order_value": 350000,
        "discount_amount": 50000,
        "used_at": "2024-01-15T14:30:00Z"
      }
    ],
    "meta": {...}
  }
```

---

### 5. Password Reset Flow

#### Entity Definitions

```go
// ==================== PASSWORD RESET ENTITIES ====================

type PasswordResetToken struct {
    PK          string    `json:"-" dynamodbav:"PK"`           // RESET_TOKEN#<token_hash>
    SK          string    `json:"-" dynamodbav:"SK"`           // METADATA
    GSI1PK      string    `json:"-" dynamodbav:"GSI1PK"`       // USER#<user_id>
    EntityType  string    `json:"-" dynamodbav:"entity_type"`  // PASSWORD_RESET

    TokenHash   string    `json:"-" dynamodbav:"token_hash"`
    UserID      string    `json:"user_id" dynamodbav:"user_id"`
    UserEmail   string    `json:"user_email" dynamodbav:"user_email"`

    IsUsed      bool      `json:"is_used" dynamodbav:"is_used"`
    UsedAt      *time.Time `json:"used_at" dynamodbav:"used_at"`

    CreatedAt   time.Time `json:"created_at" dynamodbav:"created_at"`
    ExpiresAt   time.Time `json:"expires_at" dynamodbav:"expires_at"`

    // TTL for automatic cleanup
    TTL         int64     `json:"-" dynamodbav:"ttl"`
}
```

#### Password Reset APIs

##### POST /auth/forgot-password
```yaml
Request:
  Body:
    email: string (required, email format)

Response 200:
  {
    "success": true,
    "data": {
      "message": "If an account exists with this email, a password reset link has been sent"
    }
  }

Note: Always return success to prevent email enumeration
```

##### POST /auth/reset-password
```yaml
Request:
  Body:
    token: string (required, from email link)
    new_password: string (required, min 8 chars, must contain uppercase, lowercase, number)

Response 200:
  {
    "success": true,
    "data": {
      "message": "Password reset successfully"
    }
  }

Response 400:
  {
    "success": false,
    "error": {
      "code": "INVALID_TOKEN",
      "message": "Reset token is invalid or expired"
    }
  }
```

##### POST /auth/change-password
```yaml
Headers:
  Authorization: Bearer <token>

Request:
  Body:
    current_password: string (required)
    new_password: string (required)

Response 200:
  {
    "success": true,
    "data": {
      "message": "Password changed successfully"
    }
  }
```

---

### 7. Report Generation

#### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Report Generation Flow                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────────────────┐     │
│  │  Admin UI    │────▶│  Report API  │────▶│   Report Generator       │     │
│  │  (Request)   │     │  (Lambda)    │     │   Lambda                 │     │
│  └──────────────┘     └──────────────┘     └──────────────────────────┘     │
│                                                      │                       │
│                                        ┌─────────────┴─────────────┐        │
│                                        ▼                           ▼        │
│                              ┌──────────────┐            ┌──────────────┐   │
│                              │  DynamoDB    │            │  S3 Storage  │   │
│                              │  (Query)     │            │  (Reports)   │   │
│                              └──────────────┘            └──────────────┘   │
│                                                                   │          │
│                                                                   ▼          │
│                                                          ┌──────────────┐   │
│                                                          │  CloudFront  │   │
│                                                          │  (Download)  │   │
│                                                          └──────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

#### Entity Definitions

```go
// ==================== REPORT ENTITIES ====================

type ReportType string

const (
    ReportTypeSales       ReportType = "SALES"
    ReportTypeOrders      ReportType = "ORDERS"
    ReportTypeInventory   ReportType = "INVENTORY"
    ReportTypeProducts    ReportType = "PRODUCTS"
    ReportTypeArtisans    ReportType = "ARTISANS"
    ReportTypeCustomers   ReportType = "CUSTOMERS"
    ReportTypeCoupons     ReportType = "COUPONS"
)

type ReportStatus string

const (
    ReportStatusPending    ReportStatus = "PENDING"
    ReportStatusGenerating ReportStatus = "GENERATING"
    ReportStatusCompleted  ReportStatus = "COMPLETED"
    ReportStatusFailed     ReportStatus = "FAILED"
)

type Report struct {
    ID           string        `json:"id" dynamodbav:"id"`
    PK           string        `json:"-" dynamodbav:"PK"`           // REPORT#<id>
    SK           string        `json:"-" dynamodbav:"SK"`           // METADATA
    GSI1PK       string        `json:"-" dynamodbav:"GSI1PK"`       // USER#<user_id>
    GSI1SK       string        `json:"-" dynamodbav:"GSI1SK"`       // <created_at>
    EntityType   string        `json:"-" dynamodbav:"entity_type"`  // REPORT

    Type         ReportType    `json:"type" dynamodbav:"type"`
    Name         string        `json:"name" dynamodbav:"name"`
    Status       ReportStatus  `json:"status" dynamodbav:"status"`

    // Parameters
    Parameters   ReportParams  `json:"parameters" dynamodbav:"parameters"`

    // Output
    Format       string        `json:"format" dynamodbav:"format"` // CSV, XLSX, PDF
    FileURL      string        `json:"file_url" dynamodbav:"file_url"`
    FileSize     int64         `json:"file_size" dynamodbav:"file_size"`
    RowCount     int           `json:"row_count" dynamodbav:"row_count"`

    // Error (if failed)
    ErrorMessage string        `json:"error_message" dynamodbav:"error_message"`

    CreatedAt    time.Time     `json:"created_at" dynamodbav:"created_at"`
    CompletedAt  *time.Time    `json:"completed_at" dynamodbav:"completed_at"`
    ExpiresAt    time.Time     `json:"expires_at" dynamodbav:"expires_at"`
    CreatedBy    string        `json:"created_by" dynamodbav:"created_by"`

    TTL          int64         `json:"-" dynamodbav:"ttl"` // Auto-delete after expiry
}

type ReportParams struct {
    StartDate    string            `json:"start_date" dynamodbav:"start_date"`
    EndDate      string            `json:"end_date" dynamodbav:"end_date"`
    Filters      map[string]string `json:"filters" dynamodbav:"filters"`
    GroupBy      string            `json:"group_by" dynamodbav:"group_by"`
    SortBy       string            `json:"sort_by" dynamodbav:"sort_by"`
    IncludeFields []string         `json:"include_fields" dynamodbav:"include_fields"`
}
```

#### Report Service Interface

```go
type ReportService interface {
    // Generate Reports
    GenerateSalesReport(ctx context.Context, req SalesReportRequest, userID string) (*Report, error)
    GenerateOrdersReport(ctx context.Context, req OrdersReportRequest, userID string) (*Report, error)
    GenerateInventoryReport(ctx context.Context, req InventoryReportRequest, userID string) (*Report, error)
    GenerateProductsReport(ctx context.Context, req ProductsReportRequest, userID string) (*Report, error)
    GenerateArtisansReport(ctx context.Context, req ArtisansReportRequest, userID string) (*Report, error)

    // Management
    GetReport(ctx context.Context, reportID string) (*Report, error)
    ListReports(ctx context.Context, req ListReportsRequest) (*ListReportsResponse, error)
    GetReportDownloadURL(ctx context.Context, reportID string) (*DownloadURLResponse, error)
    DeleteReport(ctx context.Context, reportID string) error

    // Scheduled Reports
    CreateScheduledReport(ctx context.Context, req CreateScheduledReportRequest, userID string) (*ScheduledReport, error)
    ListScheduledReports(ctx context.Context) ([]*ScheduledReport, error)
    DeleteScheduledReport(ctx context.Context, scheduleID string) error
}
```

#### Report APIs

##### POST /admin/reports/sales
```yaml
Permission: report:generate

Request:
  Body:
    start_date: string (required, ISO date)
    end_date: string (required, ISO date)
    group_by: string (optional: day, week, month)
    filters:
      category_ids: array (optional)
      artisan_ids: array (optional)
    format: string (optional: CSV, XLSX, PDF. default: CSV)

Response 202:
  {
    "success": true,
    "data": {
      "report_id": "rpt_abc123",
      "type": "SALES",
      "status": "PENDING",
      "message": "Report generation started"
    }
  }
```

##### POST /admin/reports/orders
```yaml
Permission: report:generate

Request:
  Body:
    start_date: string (required)
    end_date: string (required)
    filters:
      status: array (optional)
      payment_status: array (optional)
    include_items: bool (optional, default: false)
    format: string (optional)

Response 202:
  {
    "success": true,
    "data": {
      "report_id": "rpt_def456",
      "type": "ORDERS",
      "status": "PENDING"
    }
  }
```

##### POST /admin/reports/inventory
```yaml
Permission: report:generate

Request:
  Body:
    filters:
      category_ids: array (optional)
      low_stock_only: bool (optional)
      out_of_stock_only: bool (optional)
    include_transactions: bool (optional)
    format: string (optional)

Response 202:
  {
    "success": true,
    "data": {
      "report_id": "rpt_ghi789",
      "type": "INVENTORY",
      "status": "PENDING"
    }
  }
```

##### GET /admin/reports/{id}
```yaml
Permission: report:read

Response 200 (pending):
  {
    "success": true,
    "data": {
      "id": "rpt_abc123",
      "type": "SALES",
      "status": "GENERATING",
      "progress": 45,
      "created_at": "2024-01-15T10:00:00Z"
    }
  }

Response 200 (completed):
  {
    "success": true,
    "data": {
      "id": "rpt_abc123",
      "type": "SALES",
      "status": "COMPLETED",
      "name": "Sales Report - Jan 2024",
      "format": "CSV",
      "file_size": 125432,
      "row_count": 1520,
      "download_url": "https://...",
      "expires_at": "2024-01-22T10:00:00Z",
      "created_at": "2024-01-15T10:00:00Z",
      "completed_at": "2024-01-15T10:02:30Z"
    }
  }
```

##### GET /admin/reports
```yaml
Permission: report:read

Request:
  Query:
    page: int
    per_page: int
    type: string (optional)
    status: string (optional)

Response 200:
  {
    "success": true,
    "data": [
      {
        "id": "rpt_abc123",
        "type": "SALES",
        "name": "Sales Report - Jan 2024",
        "status": "COMPLETED",
        "format": "CSV",
        "created_at": "2024-01-15T10:00:00Z",
        "expires_at": "2024-01-22T10:00:00Z"
      }
    ],
    "meta": {...}
  }
```

---

## Updated Project Structure

```
handloom-admin/
├── cmd/
│   ├── auth/
│   │   └── main.go
│   ├── admin/
│   │   └── main.go
│   ├── notification-worker/
│   │   └── main.go
│   ├── bulk-processor/
│   │   └── main.go
│   ├── image-processor/
│   │   └── main.go
│   ├── report-generator/
│   │   └── main.go
│   └── analytics-aggregator/
│       └── main.go
│
├── internal/
│   ├── config/
│   ├── models/
│   │   ├── user.go
│   │   ├── category.go
│   │   ├── design.go
│   │   ├── product.go
│   │   ├── inventory.go
│   │   ├── order.go
│   │   ├── notification.go
│   │   ├── audit.go
│   │   ├── coupon.go          # NEW
│   │   ├── asset.go           # NEW
│   │   ├── bulk_job.go        # NEW
│   │   ├── report.go          # NEW
│   │   └── analytics.go       # NEW
│   │
│   ├── repository/
│   │   ├── ... (existing)
│   │   ├── coupon_repo.go     # NEW
│   │   ├── asset_repo.go      # NEW
│   │   ├── bulk_repo.go       # NEW
│   │   ├── report_repo.go     # NEW
│   │   └── analytics_repo.go  # NEW
│   │
│   ├── service/
│   │   ├── ... (existing)
│   │   ├── dashboard_service.go  # NEW
│   │   ├── coupon_service.go     # NEW
│   │   ├── asset_service.go      # NEW
│   │   ├── bulk_service.go       # NEW
│   │   └── report_service.go     # NEW
│   │
│   ├── handler/
│   │   ├── ... (existing)
│   │   ├── dashboard_handler.go  # NEW
│   │   ├── coupon_handler.go     # NEW
│   │   ├── asset_handler.go      # NEW
│   │   ├── bulk_handler.go       # NEW
│   │   └── report_handler.go     # NEW
│   │
│   └── ...
│
└── ...
```

---

## Updated RBAC Permissions

```go
// Additional permission constants
const (
    // Dashboard permissions
    PermDashboardRead = "dashboard:read"

    // Coupon permissions
    PermCouponCreate  = "coupon:create"
    PermCouponRead    = "coupon:read"
    PermCouponUpdate  = "coupon:update"
    PermCouponDelete  = "coupon:delete"

    // Asset permissions
    PermAssetRead   = "asset:read"
    PermAssetWrite  = "asset:write"
    PermAssetDelete = "asset:delete"

    // Bulk operations permissions
    PermBulkRead  = "bulk:read"
    PermBulkWrite = "bulk:write"

    // Report permissions
    PermReportGenerate = "report:generate"
    PermReportRead     = "report:read"
)

// Updated role permissions
var RolePermissions = map[UserRole][]string{
    RoleAdmin: {
        // ... existing permissions ...
        PermDashboardRead,
        PermCouponCreate, PermCouponRead, PermCouponUpdate, PermCouponDelete,
        PermAssetRead, PermAssetWrite, PermAssetDelete,
        PermBulkRead, PermBulkWrite,
        PermReportGenerate, PermReportRead,
    },
    RoleOperator: {
        // ... existing permissions ...
        PermDashboardRead,
        PermCouponRead,
        PermAssetRead, PermAssetWrite,
        PermBulkRead,
        PermReportRead,
    },
}
```

---

## AWS Infrastructure (Serverless Framework Example)

```yaml
# serverless.yml
service: handloom

provider:
  name: aws
  runtime: provided.al2
  architecture: arm64
  region: ap-south-1
  stage: ${opt:stage, 'dev'}

  environment:
    # Multi-table configuration
    CORE_TABLE: handloom-core-${self:provider.stage}
    ORDERS_TABLE: handloom-orders-${self:provider.stage}
    AUDIT_TABLE: handloom-audit-${self:provider.stage}
    ANALYTICS_TABLE: handloom-analytics-${self:provider.stage}
    # S3 Buckets
    ASSET_BUCKET: ${self:service}-${self:provider.stage}-assets
    REPORT_BUCKET: ${self:service}-${self:provider.stage}-reports
    CDN_DOMAIN: !GetAtt AssetDistribution.DomainName
    # Secrets
    JWT_SECRET: ${ssm:/handloom/${self:provider.stage}/jwt-secret}

  iam:
    role:
      statements:
        # DynamoDB - All 7 tables
        - Effect: Allow
          Action:
            - dynamodb:GetItem
            - dynamodb:PutItem
            - dynamodb:UpdateItem
            - dynamodb:DeleteItem
            - dynamodb:Query
            - dynamodb:Scan
            - dynamodb:BatchGetItem
            - dynamodb:BatchWriteItem
            - dynamodb:TransactWriteItems
            - dynamodb:TransactGetItems
          Resource:
            # Core table
            - !GetAtt CoreTable.Arn
            - !Sub "${CoreTable.Arn}/index/*"
            # Orders table
            - !GetAtt OrdersTable.Arn
            - !Sub "${OrdersTable.Arn}/index/*"
            # Audit table
            - !GetAtt AuditTable.Arn
            - !Sub "${AuditTable.Arn}/index/*"
            # Analytics table
            - !GetAtt AnalyticsTable.Arn
            - !Sub "${AnalyticsTable.Arn}/index/*"
        # DynamoDB Streams
        - Effect: Allow
          Action:
            - dynamodb:DescribeStream
            - dynamodb:GetRecords
            - dynamodb:GetShardIterator
            - dynamodb:ListStreams
          Resource:
            - !GetAtt CoreTable.StreamArn
            - !GetAtt OrdersTable.StreamArn
        # SQS
        - Effect: Allow
          Action:
            - sqs:*
          Resource:
            - !GetAtt NotificationQueue.Arn
            - !GetAtt BulkProcessorQueue.Arn
            - !GetAtt ReportQueue.Arn
        # S3
        - Effect: Allow
          Action:
            - s3:*
          Resource:
            - !GetAtt AssetBucket.Arn
            - !Sub "${AssetBucket.Arn}/*"
            - !GetAtt ReportBucket.Arn
            - !Sub "${ReportBucket.Arn}/*"
        # SES
        - Effect: Allow
          Action:
            - ses:SendEmail
            - ses:SendTemplatedEmail
          Resource: "*"

functions:
  # API Functions
  authorizer:
    handler: bootstrap
    package:
      artifact: bin/authorizer.zip

  auth:
    handler: bootstrap
    package:
      artifact: bin/auth.zip
    events:
      - http:
          path: /auth/{proxy+}
          method: ANY
          cors: true

  admin:
    handler: bootstrap
    package:
      artifact: bin/admin.zip
    timeout: 30
    events:
      - http:
          path: /admin/{proxy+}
          method: ANY
          cors: true
          authorizer:
            name: authorizer
            resultTtlInSeconds: 300

  # Worker Functions
  notification-worker:
    handler: bootstrap
    package:
      artifact: bin/notification-worker.zip
    timeout: 60
    events:
      - sqs:
          arn: !GetAtt NotificationQueue.Arn
          batchSize: 10

  bulk-processor:
    handler: bootstrap
    package:
      artifact: bin/bulk-processor.zip
    timeout: 900  # 15 minutes for large files
    memorySize: 1024
    events:
      - sqs:
          arn: !GetAtt BulkProcessorQueue.Arn
          batchSize: 1

  image-processor:
    handler: bootstrap
    package:
      artifact: bin/image-processor.zip
    timeout: 60
    memorySize: 1024
    events:
      - s3:
          bucket: !Ref AssetBucket
          event: s3:ObjectCreated:*
          rules:
            - prefix: uploads/

  report-generator:
    handler: bootstrap
    package:
      artifact: bin/report-generator.zip
    timeout: 900
    memorySize: 2048
    events:
      - sqs:
          arn: !GetAtt ReportQueue.Arn
          batchSize: 1

  # Stream processor for Core table (Products, Inventory changes)
  core-stream-processor:
    handler: bootstrap
    package:
      artifact: bin/stream-processor.zip
    timeout: 300
    events:
      - stream:
          type: dynamodb
          arn: !GetAtt CoreTable.StreamArn
          batchSize: 100
          startingPosition: LATEST

  # Stream processor for Orders table (Order status changes, analytics)
  orders-stream-processor:
    handler: bootstrap
    package:
      artifact: bin/stream-processor.zip
    timeout: 300
    events:
      - stream:
          type: dynamodb
          arn: !GetAtt OrdersTable.StreamArn
          batchSize: 100
          startingPosition: LATEST

  # Scheduled analytics aggregation
  analytics-aggregator:
    handler: bootstrap
    package:
      artifact: bin/analytics-aggregator.zip
    timeout: 300
    events:
      - schedule:
          rate: rate(1 hour)
          description: Hourly analytics aggregation

resources:
  Resources:
    # ============== DynamoDB Tables (Multi-Table Design) ==============

    # Table 1: handloom-core
    # Stores: Users, PricingRules, Coupons
    CoreTable:
      Type: AWS::DynamoDB::Table
      Properties:
        TableName: handloom-core-${self:provider.stage}
        BillingMode: PAY_PER_REQUEST
        StreamSpecification:
          StreamViewType: NEW_AND_OLD_IMAGES
        AttributeDefinitions:
          - AttributeName: PK
            AttributeType: S
          - AttributeName: SK
            AttributeType: S
          - AttributeName: GSI1PK
            AttributeType: S
          - AttributeName: GSI1SK
            AttributeType: S
          - AttributeName: GSI2PK
            AttributeType: S
          - AttributeName: GSI2SK
            AttributeType: S
          - AttributeName: entity_type
            AttributeType: S
          - AttributeName: created_at
            AttributeType: S
        KeySchema:
          - AttributeName: PK
            KeyType: HASH
          - AttributeName: SK
            KeyType: RANGE
        GlobalSecondaryIndexes:
          - IndexName: GSI1
            KeySchema:
              - AttributeName: GSI1PK
                KeyType: HASH
              - AttributeName: GSI1SK
                KeyType: RANGE
            Projection:
              ProjectionType: ALL
          - IndexName: GSI2
            KeySchema:
              - AttributeName: GSI2PK
                KeyType: HASH
              - AttributeName: GSI2SK
                KeyType: RANGE
            Projection:
              ProjectionType: ALL
          - IndexName: GSI3
            KeySchema:
              - AttributeName: entity_type
                KeyType: HASH
              - AttributeName: created_at
                KeyType: RANGE
            Projection:
              ProjectionType: ALL
        PointInTimeRecoverySpecification:
          PointInTimeRecoveryEnabled: true
        TimeToLiveSpecification:
          AttributeName: ttl
          Enabled: true
        Tags:
          - Key: Purpose
            Value: Core business entities

    # Table 2: handloom-orders
    # Stores: Orders, OrderItems, OrderStatusHistory, Customers
    OrdersTable:
      Type: AWS::DynamoDB::Table
      Properties:
        TableName: handloom-orders-${self:provider.stage}
        BillingMode: PAY_PER_REQUEST
        StreamSpecification:
          StreamViewType: NEW_AND_OLD_IMAGES
        AttributeDefinitions:
          - AttributeName: PK
            AttributeType: S
          - AttributeName: SK
            AttributeType: S
          - AttributeName: GSI1PK
            AttributeType: S
          - AttributeName: GSI1SK
            AttributeType: S
          - AttributeName: GSI2PK
            AttributeType: S
          - AttributeName: GSI2SK
            AttributeType: S
          - AttributeName: entity_type
            AttributeType: S
          - AttributeName: created_at
            AttributeType: S
        KeySchema:
          - AttributeName: PK
            KeyType: HASH
          - AttributeName: SK
            KeyType: RANGE
        GlobalSecondaryIndexes:
          - IndexName: GSI1
            KeySchema:
              - AttributeName: GSI1PK
                KeyType: HASH
              - AttributeName: GSI1SK
                KeyType: RANGE
            Projection:
              ProjectionType: ALL
          - IndexName: GSI2
            KeySchema:
              - AttributeName: GSI2PK
                KeyType: HASH
              - AttributeName: GSI2SK
                KeyType: RANGE
            Projection:
              ProjectionType: ALL
          - IndexName: GSI3
            KeySchema:
              - AttributeName: entity_type
                KeyType: HASH
              - AttributeName: created_at
                KeyType: RANGE
            Projection:
              ProjectionType: ALL
        PointInTimeRecoverySpecification:
          PointInTimeRecoveryEnabled: true
        Tags:
          - Key: Purpose
            Value: Order management

    # Table 3: handloom-audit
    # Stores: AuditLogs (90-day retention via TTL)
    AuditTable:
      Type: AWS::DynamoDB::Table
      Properties:
        TableName: handloom-audit-${self:provider.stage}
        BillingMode: PAY_PER_REQUEST
        AttributeDefinitions:
          - AttributeName: PK
            AttributeType: S
          - AttributeName: SK
            AttributeType: S
          - AttributeName: GSI1PK
            AttributeType: S
          - AttributeName: GSI1SK
            AttributeType: S
          - AttributeName: GSI2PK
            AttributeType: S
          - AttributeName: GSI2SK
            AttributeType: S
        KeySchema:
          - AttributeName: PK
            KeyType: HASH
          - AttributeName: SK
            KeyType: RANGE
        GlobalSecondaryIndexes:
          - IndexName: GSI1
            KeySchema:
              - AttributeName: GSI1PK
                KeyType: HASH
              - AttributeName: GSI1SK
                KeyType: RANGE
            Projection:
              ProjectionType: ALL
          - IndexName: GSI2
            KeySchema:
              - AttributeName: GSI2PK
                KeyType: HASH
              - AttributeName: GSI2SK
                KeyType: RANGE
            Projection:
              ProjectionType: ALL
        TimeToLiveSpecification:
          AttributeName: ttl
          Enabled: true
        Tags:
          - Key: Purpose
            Value: Audit logs
          - Key: Retention
            Value: 90 days

    # Table 4: handloom-analytics
    # Stores: DailyMetrics, TopProducts, TopArtisans, InventoryAlerts
    AnalyticsTable:
      Type: AWS::DynamoDB::Table
      Properties:
        TableName: handloom-analytics-${self:provider.stage}
        BillingMode: PAY_PER_REQUEST
        AttributeDefinitions:
          - AttributeName: PK
            AttributeType: S
          - AttributeName: SK
            AttributeType: S
          - AttributeName: GSI1PK
            AttributeType: S
          - AttributeName: created_at
            AttributeType: S
        KeySchema:
          - AttributeName: PK
            KeyType: HASH
          - AttributeName: SK
            KeyType: RANGE
        GlobalSecondaryIndexes:
          - IndexName: GSI1
            KeySchema:
              - AttributeName: GSI1PK
                KeyType: HASH
              - AttributeName: created_at
                KeyType: RANGE
            Projection:
              ProjectionType: ALL
        TimeToLiveSpecification:
          AttributeName: ttl
          Enabled: true
        Tags:
          - Key: Purpose
            Value: Analytics and dashboard metrics

    # ============== SQS Queues ==============
    NotificationQueue:
      Type: AWS::SQS::Queue
      Properties:
        QueueName: ${self:service}-${self:provider.stage}-notifications
        VisibilityTimeout: 300
        RedrivePolicy:
          deadLetterTargetArn: !GetAtt NotificationDLQ.Arn
          maxReceiveCount: 3

    NotificationDLQ:
      Type: AWS::SQS::Queue
      Properties:
        QueueName: ${self:service}-${self:provider.stage}-notifications-dlq
        MessageRetentionPeriod: 1209600  # 14 days

    BulkProcessorQueue:
      Type: AWS::SQS::Queue
      Properties:
        QueueName: ${self:service}-${self:provider.stage}-bulk
        VisibilityTimeout: 900
        RedrivePolicy:
          deadLetterTargetArn: !GetAtt BulkDLQ.Arn
          maxReceiveCount: 2

    BulkDLQ:
      Type: AWS::SQS::Queue
      Properties:
        QueueName: ${self:service}-${self:provider.stage}-bulk-dlq

    ReportQueue:
      Type: AWS::SQS::Queue
      Properties:
        QueueName: ${self:service}-${self:provider.stage}-reports
        VisibilityTimeout: 900
        RedrivePolicy:
          deadLetterTargetArn: !GetAtt ReportDLQ.Arn
          maxReceiveCount: 2

    ReportDLQ:
      Type: AWS::SQS::Queue
      Properties:
        QueueName: ${self:service}-${self:provider.stage}-reports-dlq

    # ============== S3 Buckets ==============
    AssetBucket:
      Type: AWS::S3::Bucket
      Properties:
        BucketName: ${self:service}-${self:provider.stage}-assets
        CorsConfiguration:
          CorsRules:
            - AllowedHeaders: ['*']
              AllowedMethods: [GET, PUT, POST]
              AllowedOrigins: ['*']
              MaxAge: 3600
        PublicAccessBlockConfiguration:
          BlockPublicAcls: true
          BlockPublicPolicy: true
          IgnorePublicAcls: true
          RestrictPublicBuckets: true

    ReportBucket:
      Type: AWS::S3::Bucket
      Properties:
        BucketName: ${self:service}-${self:provider.stage}-reports
        LifecycleConfiguration:
          Rules:
            - Id: DeleteOldReports
              Status: Enabled
              ExpirationInDays: 30
        PublicAccessBlockConfiguration:
          BlockPublicAcls: true
          BlockPublicPolicy: true
          IgnorePublicAcls: true
          RestrictPublicBuckets: true

    # ============== CloudFront ==============
    AssetDistribution:
      Type: AWS::CloudFront::Distribution
      Properties:
        DistributionConfig:
          Enabled: true
          Origins:
            - Id: AssetOrigin
              DomainName: !GetAtt AssetBucket.RegionalDomainName
              S3OriginConfig:
                OriginAccessIdentity: !Sub "origin-access-identity/cloudfront/${AssetOAI}"
          DefaultCacheBehavior:
            TargetOriginId: AssetOrigin
            ViewerProtocolPolicy: redirect-to-https
            CachePolicyId: 658327ea-f89d-4fab-a63d-7e88639e58f6  # CachingOptimized
            AllowedMethods: [GET, HEAD]
            CachedMethods: [GET, HEAD]
          PriceClass: PriceClass_200

    AssetOAI:
      Type: AWS::CloudFront::CloudFrontOriginAccessIdentity
      Properties:
        CloudFrontOriginAccessIdentityConfig:
          Comment: OAI for Handloom Assets

    AssetBucketPolicy:
      Type: AWS::S3::BucketPolicy
      Properties:
        Bucket: !Ref AssetBucket
        PolicyDocument:
          Statement:
            - Effect: Allow
              Principal:
                CanonicalUser: !GetAtt AssetOAI.S3CanonicalUserId
              Action: s3:GetObject
              Resource: !Sub "${AssetBucket.Arn}/processed/*"

  Outputs:
    ApiEndpoint:
      Value: !Sub "https://${ApiGatewayRestApi}.execute-api.${AWS::Region}.amazonaws.com/${self:provider.stage}"
    AssetCDN:
      Value: !Sub "https://${AssetDistribution.DomainName}"
    MainTableArn:
      Value: !GetAtt MainTable.Arn
```

---

## Summary

This design provides a **comprehensive e-commerce platform** for handloom products, with both admin dashboard and B2C customer storefront:

### Core Features
1. **Complete RBAC** with Admin and Operator roles
2. **Full inventory management** with add/remove/check/adjust operations
3. **Category and Design hierarchy** for organizing handloom products
4. **Comprehensive order management** with status tracking and history
5. **Notification system** for order updates, refunds, and cancellations
6. **Audit logging** for compliance and debugging

### Additional Features (Now Included)
7. **Dashboard & Analytics** - Real-time metrics, sales charts, inventory alerts
8. **Bulk Operations** - CSV import/export for products, inventory, prices
9. **Image Management** - S3 upload, automatic resizing, CDN delivery
10. **Coupon/Discount Management** - Percentage/flat discounts, usage limits, validity periods
11. **Password Reset Flow** - Secure token-based password recovery
13. **Report Generation** - Sales, orders, inventory reports in CSV/XLSX/PDF
14. **Hierarchical Categories** - Nested categories with inherited attributes (Bedding → Bedsheets)
15. **Dynamic Pricing Engine** - Area/length-based pricing, material multipliers, attribute surcharges
16. **Custom Dimensions** - Customer-configurable product sizes within admin-defined ranges

### Architecture Highlights

| Component | Technology | Purpose |
|-----------|------------|---------|
| API | API Gateway + Lambda | Serverless, auto-scaling |
| Database | DynamoDB + PostgreSQL | Hybrid: DynamoDB for core/transactional, PostgreSQL for catalog |
| File Storage | S3 + CloudFront | Images, reports, bulk files |
| Async Processing | SQS + Lambda | Notifications, bulk ops, reports |
| Event Bus | SNS + SQS | Domain event fan-out to worker Lambdas |

### Hybrid Database Design

**DynamoDB Tables (7):**

| Table | Purpose | Entities | TTL |
|-------|---------|----------|-----|
| `handloom-core` | Core business data | Users, PricingRules, Coupons | None |
| `handloom-orders` | Order management | Orders, OrderItems, StatusHistory, Customers, Carts, PriceQuotes | None |
| `handloom-sessions` | Auth sessions | OTPs, Refresh Tokens, Password Resets | TTL-based |
| `handloom-audit` | Compliance logs | AuditLogs | 90 days |
| `handloom-analytics` | Dashboard metrics | DailyMetrics, TopProducts, InventoryAlerts, Reports | 2 years |
| `handloom-notifications` | User notifications | Notifications, Templates | None |
| `handloom-events` | Raw tracking events | Frontend tracking events | 30 days |

**PostgreSQL (Catalog) — 8 tables:**

| Table | Purpose |
|-------|---------|
| `categories` | Product categories with slug-based routing |
| `category_attributes` | Dynamic attributes defined per category |
| `category_attribute_options` | Allowed values for each attribute |
| `products` | Product catalog (prices in paise, UNIQUE sku) |
| `product_attribute_values` | EAV storage for product attributes |
| `product_images` | Product images with sort ordering |
| `inventory` | Stock levels with low-stock threshold detection |
| `inventory_transactions` | Full audit trail for every stock mutation |

Key PostgreSQL patterns: GIN trigram index for full-text search, `EXISTS` subqueries for dynamic attribute filtering (EAV), `SELECT ... FOR UPDATE` for inventory row locking, `pgxpool` connection pooling, in-process TTL cache (go-cache) for categories and products.

**Benefits of this design:**
1. **Logical Separation** - Orders scale independently during peak sales
2. **Different Retention** - Audit logs auto-expire, analytics retained longer
3. **Operational Clarity** - Easy to backup, monitor, and troubleshoot per domain
4. **Cost Optimization** - Audit table uses minimal provisioning
5. **Efficient Queries** - Related entities co-located within each table
6. **Relational Integrity** - Foreign keys and UNIQUE constraints for catalog data
7. **Full-Text Search** - GIN trigram index enables fast product name search

### B2C Storefront (Implemented)

The B2C storefront is fully integrated with 9 dedicated Lambda services:
- **Store Auth**: Phone OTP login via MSG91, customer JWT in HttpOnly cookies
- **Store Catalog**: Public product/category browsing (backed by PostgreSQL)
- **Store Cart**: Shopping cart management (customer-authenticated)
- **Store Checkout**: Order placement + PhonePe payment initiation
- **Store Orders**: Customer order history
- **Store Profile**: Customer account and address management
- **Store Tracking**: Public order tracking via Shiprocket
- **Store Events**: Storefront analytics event ingestion (rate-limited)
- **Store Webhooks**: Payment callback handling (signature-verified)

### Event-Driven Architecture (Implemented)

Domain events flow through SNS → SQS → Worker Lambdas:
- **SNS Topic**: Publishes order.created, payment.confirmed, shipment.updated, etc.
- **4 Worker Lambdas**: analytics, audit, notification, report — each consumes its own SQS queue
- **CDK EventStack**: Defines SNS topic, SQS queues with DLQs, and worker Lambda functions

### API Count Summary

| Module | Endpoints |
|--------|-----------|
| Auth | 5 |
| Users | 5 |
| Categories | 9 | (includes attribute management)
| Designs | 5 |
| Products | 5 |
| Pricing Rules (Admin) | 6 |
| Price Calculation (Public) | 4 |
| Inventory | 6 |
| Orders | 8 |
| Notifications | 3 |
| Dashboard | 5 |
| Bulk Operations | 7 |
| Assets | 5 |
| Coupons | 6 |
| Reports | 5 |
| **Admin Total** | **84 endpoints** |
| | |
| Store Auth | 4 |
| Store Catalog | 6 |
| Store Cart | 4 |
| Store Checkout | 2 |
| Store Orders | 2 |
| Store Profile | 4 |
| Store Tracking | 2 |
| Store Webhooks | 1 |
| **B2C Store Total** | **25 endpoints** |
| | |
| **Grand Total** | **117 endpoints** |
