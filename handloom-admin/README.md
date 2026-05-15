# Handloom Admin API

A serverless backend for the Homechrome handloom e-commerce platform. Powers both the admin dashboard and the B2C customer storefront.

## Architecture

The application is built as up to **26 AWS Lambda services** (12 admin + 9 B2C store + 4 event workers + 1 migrator), designed for high availability, scalability, and cost efficiency. In dev, only 15 Lambdas are active (5 admin + 9 store + 1 migrator); the event stack and several admin services are disabled.

### Admin Microservices

| Service | Purpose | Key Endpoints |
|---------|---------|---------------|
| **Auth** | Authentication & password management | `/admin/auth/login`, `/admin/auth/refresh` |
| **User** | Admin user CRUD & role management | `/admin/users`, `/admin/users/{id}` |
| **Catalog** | Products, categories, designs | `/admin/categories`, `/admin/products` |
| **Order** | Order & customer management | `/admin/orders`, `/admin/customers` |
| **Pricing** | Dynamic pricing engine | `/api/v1/pricing/*`, `/admin/pricing/rules` |
| **Inventory** | Stock management | `/admin/inventory` |
| **Analytics** | Dashboard metrics | `/admin/analytics/*` |
| **Notification** | User notifications | `/admin/notifications` |
| **Coupon** | Coupon management | `/admin/coupons` |
| **Asset** | Media/image management | `/admin/assets` |
| **Report** | Report generation | `/admin/reports` |
| **Audit** | Audit log access (admin only) | `/admin/audit` |

### B2C Store Routes

| Route Group | Purpose | Key Endpoints |
|-------------|---------|---------------|
| **Store Auth** | Phone OTP login for customers | `/api/v1/store/auth/*` |
| **Store Catalog** | Public product/category browsing | `/api/v1/store/catalog/*` |
| **Store Cart** | Shopping cart management | `/api/v1/store/cart/*` |
| **Store Checkout** | Order placement + payment | `/api/v1/store/checkout/*` |
| **Store Orders** | Customer order history | `/api/v1/store/orders/*` |
| **Store Profile** | Customer account management | `/api/v1/store/me/*` |
| **Store Tracking** | Order tracking | `/api/v1/store/track/*` |
| **Store Events** | Storefront analytics events | `/api/v1/store/events/*` |
| **Webhooks** | Payment callback (PhonePe) | `/api/v1/store/webhooks/*` |

### External Integrations

| Gateway | Purpose | Config Prefix |
|---------|---------|---------------|
| **PhonePe** | Payment processing (Standard Checkout v2) | `PHONEPE_*` |
| **Delhivery** | Shipping & delivery tracking | `DELHIVERY_*` |
| **MSG91** | SMS OTP for customer auth | `MSG91_*` |

### Event-Driven Architecture

| Worker | Queue | Purpose |
|--------|-------|---------|
| **worker-analytics** | analytics queue | Process analytics events |
| **worker-audit** | audit queue | Record audit trail entries |
| **worker-notification** | notification queue | Send user notifications |
| **worker-report** | report queue | Generate business reports |

Domain events are published to an SNS topic and fanned out to SQS queues. Each worker Lambda processes its queue independently.

> **Note:** The event stack (SNS topic, 4 SQS queues, EventBridge rule, and 4 worker Lambdas) is **disabled in dev**. The `EventStack` creation is commented out in `infra/cmd/main.go`. When events are disabled, a `NoopPublisher` is used instead of the SNS publisher.

### Active Lambdas in Dev

In the dev environment, only **15 Lambdas** are deployed:
- **Admin (5):** auth, user, catalog, asset, order
- **Store (9):** store-auth, store-catalog, store-cart, store-checkout, store-orders, store-tracking, store-profile, store-events, store-webhooks
- **Utility (1):** migrator

**Disabled in dev:** pricing, inventory, analytics, notification, coupon, report, audit (admin), + 4 event workers

### Features

- **Hierarchical Categories** - Nested categories with inherited attributes
- **Dynamic Pricing Engine** - Area/length-based pricing with material multipliers and attribute surcharges
- **Custom Dimensions** - Allow customers to specify custom product dimensions within configurable ranges
- **Multi-Table DynamoDB Design** - Optimized for scale with separate tables for core data, orders, audit logs, and analytics
- **B2C Storefront** - Customer-facing cart, checkout, payment, and order tracking

## Tech Stack

- **Language**: Go 1.25+
- **Router**: Chi
- **Database**: DynamoDB (7 tables) + PostgreSQL (catalog data)
- **Infrastructure**: AWS CDK (Go)
- **DI**: Wire
- **Runtime**: AWS Lambda (ARM64, provided.al2023)
- **Testing**: testify, mockgen
- **Local Emulation**: LocalStack (DynamoDB, S3, Lambda, API Gateway, IAM)

## Project Structure

```
.
├── cmd/
│   ├── api/                    # Local development server entry point
│   └── lambda/                 # Lambda entry points (up to 26 services; 15 active in dev)
│       ├── auth/               # Admin services (12 total, 5 active in dev)
│       ├── user/
│       ├── catalog/
│       ├── order/
│       ├── pricing/
│       ├── inventory/
│       ├── analytics/
│       ├── notification/
│       ├── coupon/
│       ├── asset/
│       ├── report/
│       ├── audit/
│       ├── store-auth/         # B2C store services (9)
│       ├── store-catalog/
│       ├── store-cart/
│       ├── store-checkout/
│       ├── store-events/
│       ├── store-orders/
│       ├── store-profile/
│       ├── store-tracking/
│       ├── store-webhooks/
│       ├── worker-analytics/   # Event workers (4)
│       ├── worker-audit/
│       ├── worker-notification/
│       └── worker-report/
├── internal/
│   ├── config/                 # Configuration loading
│   ├── domain/                 # Domain entities and interfaces
│   ├── handler/                # HTTP handlers (admin)
│   │   └── store/              # B2C store handlers
│   ├── gateway/                # External service integrations
│   │   ├── phonepe/            # PhonePe payment gateway
│   │   ├── courier/            # Delhivery shipping (courier.Gateway)
│   │   └── sms/                # MSG91 SMS/OTP
│   ├── middleware/             # HTTP middleware (admin + customer auth)
│   ├── repository/             # Data access layer
│   │   ├── dynamodb/           # DynamoDB implementations
│   │   └── postgres/           # PostgreSQL implementations (catalog)
│   ├── router/                 # Service-specific routers & Lambda adapter
│   ├── s3client/               # S3 client wrapper
│   ├── event/                  # Event-driven architecture
│   │   ├── publisher.go        # SNS event publisher
│   │   ├── types.go            # Event type definitions
│   │   └── handlers/           # SQS worker handlers (analytics, audit, notification, report)
│   ├── service/                # Business logic layer
│   ├── validator/              # Request validation
│   └── wire/                   # Dependency injection (Wire)
├── infra/                      # AWS CDK Infrastructure (Go)
│   ├── cmd/                    # CDK entry point
│   └── stacks/                 # CDK stacks (database, storage, api, events)
├── pkg/
│   ├── errors/                 # Custom error types
│   ├── logger/                 # Structured logging
│   └── response/               # HTTP response helpers
├── scripts/                    # Development scripts
├── test/                       # Integration tests
└── docs/                       # Documentation
```

## Quick Start

### Prerequisites

- Go 1.25+
- Docker & Docker Compose
- AWS CLI v2 (for LocalStack commands)
- AWS CDK CLI (for deployment)
- jq (optional, for JSON formatting in test commands)

### One-Command Setup

```bash
make setup-local  # Starts LocalStack, creates DynamoDB tables, S3 buckets, seeds data
make run          # Start the API server (monolith mode)
```

**That's it!** The API will be available at `http://localhost:8081`.

### Default Credentials

After running `make setup-local`, you can log in with:

| Role | Email | Password |
|------|-------|----------|
| Admin | admin@handloom.com | Admin@123! |
| Manager | manager@handloom.com | Admin@123! |

> **Important**: Change these credentials in production!

## Local Development

All AWS services are emulated locally using [LocalStack](https://localstack.cloud/). There are two development modes:

### Mode 1: Monolith (Recommended for daily development)

Runs the entire backend as a single Go process with hot reload. Fastest iteration cycle.

```bash
# Terminal 1 — Backend
cd handloom-admin
make setup-local   # Only needed once (or after make teardown-local)
make run           # API on :8081
# Or: make run-watch  # Hot reload via air

# Terminal 2 — Frontend
cd handloom-admin-frontend
npm run dev:local  # Vite on :5173 → localhost:8081
```

### Mode 2: Lambda (Integration testing)

Deploys actual Lambda binaries to LocalStack with API Gateway, matching the production topology. Use this to verify Lambda-specific behavior before deploying to AWS.

```bash
# Terminal 1 — LocalStack must be running
cd handloom-admin
make setup-local      # Only needed once
make deploy-local     # Builds 4 Lambdas + deploys to LocalStack with API Gateway

# Terminal 2 — Frontend
cd handloom-admin-frontend
npm run dev:lambda    # Vite on :5173 → localhost:4566 (API Gateway)
```

After code changes, redeploy Lambda code without recreating the API Gateway:

```bash
make redeploy-local   # Faster — rebuilds and updates Lambda code only
```

### Frontend Dev Modes

The **admin frontend** (`handloom-admin-frontend/`) supports three targets via Vite modes:

| Script | Target | Env File | Use Case |
|--------|--------|----------|----------|
| `npm run dev:local` | `localhost:8081` | `.env.local-backend` | Daily dev against monolith |
| `npm run dev:lambda` | `localhost:4566` | `.env.local-lambda` | Test against local Lambdas |
| `npm run dev` | AWS dev API | `.env.development` | Test against deployed AWS |

The **B2C storefront** (`homechrome-store/`) supports two targets:

| Script | Target | Env File | Use Case |
|--------|--------|----------|----------|
| `npm run dev` | `localhost:8081` | `.env.local` | Daily dev against monolith |
| `npm run dev:lambda` | `localhost:4566` | `.env.local-lambda` | Test against local Lambdas |

### Step-by-Step Setup

1. **Clone and install dependencies**
   ```bash
   git clone <repo-url>
   cd handloom-admin
   go mod download
   ```

2. **Install development tools**
   ```bash
   make install-tools
   ```

3. **Start LocalStack**
   ```bash
   make docker-up
   ```

4. **Create tables, S3 buckets, and seed data**
   ```bash
   make create-tables
   make init-s3
   make seed-data
   ```
   Or all at once: `make setup-local` (includes `docker-up`)

5. **Run the API server**
   ```bash
   make run
   ```

### Verify Setup

```bash
# Check service health
make health

# Test login
make test-login

# Quick API test (login + list categories)
make test-api
```

### Local Services

| Service | URL | Description |
|---------|-----|-------------|
| API Server | http://localhost:8081 | Go monolith (when using `make run`) |
| LocalStack | http://localhost:4566 | DynamoDB, S3, Lambda, API Gateway |
| DynamoDB Admin UI | http://localhost:8001 | Browse DynamoDB tables in browser |
| pgAdmin | http://localhost:5050 | Browse PostgreSQL tables in browser |

### Environment Configuration

Copy `.env.example` to `.env` for custom configuration:

```bash
cp .env.example .env
```

Key environment variables:
- `APP_ENV` - Environment (development/production)
- `APP_DEBUG` - Enable debug logging
- `SERVER_PORT` - API server port (default: 8081)
- `AWS_ENDPOINT` - LocalStack endpoint (set for local, empty for AWS)
- `AWS_REGION` - AWS region (ap-southeast-1)
- `JWT_SECRET_KEY` - Admin JWT signing key (change in production!)
- `CUSTOMER_JWT_SECRET` - Customer JWT signing key (B2C store auth)
- `PHONEPE_CLIENT_ID` / `PHONEPE_CLIENT_SECRET` / `PHONEPE_CLIENT_VERSION` - PhonePe OAuth credentials (Standard Checkout v2). When CLIENT_ID or CLIENT_SECRET is empty, a DevClient with mock responses is used (based on credential presence, not `APP_ENV`)
- `PHONEPE_WEBHOOK_USERNAME` / `PHONEPE_WEBHOOK_PASSWORD` - PhonePe webhook verification credentials
- `DELHIVERY_*` - Delhivery shipping config (DevClient used when credentials empty)
- `MSG91_*` - MSG91 SMS/OTP config
- `SNS_TOPIC_ARN` - SNS event topic ARN
- `EVENT_PUBLISHING_ENABLED` - Enable/disable event publishing

## Deployment

### AWS Free Tier Optimization

The infrastructure is configured to stay within AWS Free Tier limits:

| Service | Free Tier | Configuration |
|---------|-----------|---------------|
| **Lambda** | 1M requests, 400K GB-sec/month | 128MB memory, ARM64 (cost-effective) |
| **DynamoDB** | 25GB storage, 25 RCU/WCU | On-demand billing, TTL for cleanup |
| **S3** | 5GB storage, 20K GET, 2K PUT | No versioning, lifecycle cleanup |
| **CloudFront** | 1TB transfer, 10M requests (12 mo) | PRICE_CLASS_100 (cheapest) |
| **API Gateway** | 1M calls/month (12 months) | Minimal logging, no X-Ray |
| **CloudWatch** | 5GB logs | 3-day retention (dev) |

**Cost-saving features:**
- X-Ray tracing disabled (not free)
- CloudWatch metrics disabled on API Gateway
- Point-in-Time Recovery disabled on DynamoDB (dev only)
- Aggressive lifecycle rules for S3 cleanup
- Minimal log retention

### Build Lambda Functions

```bash
# Build all Lambda functions
make build-lambdas

# Build a specific Lambda
make build-lambda-auth
make build-lambda-catalog
# ... etc
```

### Deploy with AWS CDK

```bash
# Synthesize CloudFormation templates
make cdk-synth

# Deploy to development
make cdk-deploy-dev

# Deploy to production
make cdk-deploy-prod

# Show changes before deploying
make cdk-diff

# Destroy infrastructure
make cdk-destroy
```

## API Endpoints

### Public APIs

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/api/v1/pricing/calculate` | POST | Calculate price for custom dimensions |
| `/api/v1/pricing/dimension-options/{categoryId}` | GET | Get dimension options for a category |
| `/api/v1/pricing/bulk-calculate` | POST | Calculate prices for multiple configurations |

### B2C Store APIs (`/api/v1/store/*`)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/store/auth/send-otp` | POST | Send OTP to phone number |
| `/api/v1/store/auth/verify-otp` | POST | Verify OTP and get tokens |
| `/api/v1/store/auth/refresh` | POST | Refresh customer access token |
| `/api/v1/store/catalog/*` | GET | Browse categories and products |
| `/api/v1/store/cart/*` | GET, POST, PATCH, DELETE | Cart management |
| `/api/v1/store/checkout/*` | POST | Place order and initiate payment |
| `/api/v1/store/orders/*` | GET | Customer order history |
| `/api/v1/store/me/*` | GET, PATCH | Customer profile |
| `/api/v1/store/track/*` | GET | Order tracking |
| `/api/v1/store/webhooks/phonepe` | POST | PhonePe payment callback |

### Admin APIs (`/admin/*`)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/admin/auth/login` | POST | Admin login |
| `/admin/auth/refresh` | POST | Refresh access token |
| `/admin/categories` | GET, POST | List/Create categories |
| `/admin/categories/{id}` | GET, PATCH, DELETE | Get/Update/Delete category |
| `/admin/products` | GET, POST | List/Create products |
| `/admin/products/{id}` | GET, PATCH, DELETE | Get/Update/Delete product |
| `/admin/orders` | GET | List orders |
| `/admin/orders/{id}` | GET, PATCH | Get/Update order |
| `/admin/pricing/rules` | GET, POST | List/Create pricing rules |
| `/admin/analytics/dashboard` | GET | Dashboard statistics |
| `/admin/audit` | GET | Audit logs |

## Example: Calculate Price

```bash
curl -X POST http://localhost:8081/api/v1/pricing/calculate \
  -H "Content-Type: application/json" \
  -d '{
    "category_id": "cat_bedsheets",
    "dimensions": {
      "length": 100,
      "width": 90,
      "unit": "inches"
    },
    "attributes": {
      "material": "silk",
      "thread_count": "600",
      "elastic_type": "fitted"
    },
    "quantity": 1
  }'
```

## Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run unit tests only
make test-unit

# Run integration tests (requires DynamoDB Local)
make test-integration
```

## DynamoDB Tables

| Table | Purpose | Entities |
|-------|---------|----------|
| `handloom-core` | Core business data | Users, Pricing Rules, Coupons |
| `handloom-orders` | Order management | Orders, OrderItems, StatusHistory, Customers, PriceQuotes |
| `handloom-sessions` | Auth sessions (TTL-based) | OTPs, Refresh Tokens |
| `handloom-audit` | Compliance logs (90-day TTL) | AuditLogs |
| `handloom-analytics` | Dashboard metrics (2-year TTL) | DailyMetrics, TopProducts, Alerts, Reports |
| `handloom-notifications` | User notifications | Notifications |

## Development Commands

Run `make help` to see all available commands. Key commands:

```bash
# Quick Start
make setup-local       # Full local setup (LocalStack + tables + S3 + seed)
make run               # Start API server (monolith mode)
make run-watch         # Start with hot reload (air)

# Local Lambda
make deploy-local      # Build + deploy Lambdas to LocalStack
make redeploy-local    # Redeploy Lambda code only (faster)
make teardown-local    # Stop all Docker services and remove volumes

# Testing & Debug
make test              # Run all tests
make test-unit         # Unit tests only (no Docker needed)
make test-integration  # Integration tests (needs LocalStack)
make test-coverage     # Run tests with coverage report
make health            # Check local service health
make test-login        # Test login with admin credentials
make test-api          # Quick API tests

# Docker & Database
make docker-up         # Start LocalStack + DynamoDB Admin
make docker-down       # Stop Docker services
make create-tables     # Create DynamoDB tables
make init-s3           # Create S3 buckets in LocalStack
make seed-data         # Seed test data
make reset-db          # Drop and recreate all tables
make list-tables       # List DynamoDB tables

# Code Generation
make wire              # Generate Wire DI code
make generate-mocks    # Generate mocks

# Building
make build             # Build local API binary
make build-lambdas     # Build all Lambda functions

# Deployment
make cdk-deploy-dev    # Deploy to development
make cdk-deploy-prod   # Deploy to production
make cdk-diff          # Show CDK diff
make cdk-destroy       # Destroy CDK stacks
```

## Troubleshooting

### LocalStack / DynamoDB Connection Issues

```bash
# Check if LocalStack is running
make health

# Restart LocalStack
make docker-down && make docker-up
```

### Port 4566 Already in Use

```bash
# Check what's using the port
lsof -i :4566

# Stop all Docker containers and restart
docker stop $(docker ps -q) 2>/dev/null
make docker-up
```

### Reset Everything

```bash
# Full reset: stop services, remove volumes, restart
make teardown-local
make setup-local
```

### Lambda Deployment Issues

```bash
# Check if Lambdas are deployed
aws --endpoint-url=http://localhost:4566 --region ap-southeast-1 lambda list-functions

# Check API Gateway
aws --endpoint-url=http://localhost:4566 --region ap-southeast-1 apigateway get-rest-apis

# View Lambda logs
aws --endpoint-url=http://localhost:4566 --region ap-southeast-1 logs describe-log-groups
```

### View Database Data

- **DynamoDB**: Visit http://localhost:8001 for the DynamoDB Admin UI.
- **PostgreSQL**: Visit http://localhost:5050 for pgAdmin. On first use, add server: host `postgres`, port `5432`, user/password `handloom`.

### Generate Password Hash

```bash
go run scripts/generate-password-hash.go 'YourNewPassword!'
```

## Documentation

- [Design Document](docs/DESIGN.md) - Detailed architecture, entity design, and API contracts

## License

Private - All rights reserved
