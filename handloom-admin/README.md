# Handloom Admin API

A serverless microservices backend for handloom product management with dynamic pricing and custom dimension support.

## Architecture

The application is built as **14 independent AWS Lambda microservices**, designed for high availability, scalability, and cost efficiency.

### Microservices

| Service | Purpose | Key Endpoints |
|---------|---------|---------------|
| **Auth** | Authentication & password management | `/auth/login`, `/auth/refresh`, `/password/*` |
| **User** | Admin user CRUD & role management | `/users`, `/users/{id}` |
| **Catalog** | Products, categories, designs | `/categories`, `/designs`, `/products` |
| **Order** | Order & customer management | `/orders`, `/customers` |
| **Pricing** | Dynamic pricing engine | `/api/v1/pricing/*`, `/pricing/rules` |
| **Inventory** | Stock management | `/inventory`, `/inventory/alerts` |
| **Analytics** | Dashboard metrics | `/analytics/*` |
| **Notification** | User notifications | `/notifications` |
| **Coupon** | Coupon management | `/coupons`, `/coupons/apply` |
| **Artisan** | Artisan & payout management | `/artisans`, `/artisans/{id}/payouts` |
| **Bulk** | Bulk import/export operations | `/bulk/import`, `/bulk/export` |
| **Asset** | Media/image management | `/assets`, `/assets/upload` |
| **Report** | Report generation | `/reports` |
| **Audit** | Audit log access (admin only) | `/audit`, `/audit/entity/*` |

### Features

- **Hierarchical Categories** - Nested categories with inherited attributes
- **Dynamic Pricing Engine** - Area/length-based pricing with material multipliers and attribute surcharges
- **Custom Dimensions** - Allow customers to specify custom product dimensions within configurable ranges
- **Multi-Table DynamoDB Design** - Optimized for scale with separate tables for core data, orders, audit logs, and analytics

## Tech Stack

- **Language**: Go 1.24+
- **Router**: Chi
- **Database**: DynamoDB (4 tables)
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
│   └── lambda/                 # Lambda entry points (14 services)
│       ├── auth/
│       ├── user/
│       ├── catalog/
│       ├── order/
│       ├── pricing/
│       ├── inventory/
│       ├── analytics/
│       ├── notification/
│       ├── coupon/
│       ├── artisan/
│       ├── bulk/
│       ├── asset/
│       ├── report/
│       └── audit/
├── internal/
│   ├── config/                 # Configuration loading
│   ├── domain/                 # Domain entities and interfaces
│   ├── handler/                # HTTP handlers
│   ├── middleware/             # HTTP middleware
│   ├── repository/             # Data access layer
│   │   └── dynamodb/           # DynamoDB implementations
│   ├── router/                 # Service-specific routers & Lambda adapter
│   ├── service/                # Business logic layer
│   └── wire/                   # Dependency injection (Wire)
├── infra/                      # AWS CDK Infrastructure (Go)
│   ├── cmd/                    # CDK entry point
│   └── stacks/                 # CDK stacks (database, storage, api)
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

- Go 1.24+
- Docker & Docker Compose
- AWS CLI v2 (for LocalStack commands)
- AWS CDK CLI (for deployment)
- jq (optional, for JSON formatting in test commands)

### One-Command Setup

```bash
make setup-local  # Starts LocalStack, creates DynamoDB tables, S3 buckets, seeds data
make run          # Start the API server (monolith mode)
```

**That's it!** The API will be available at `http://localhost:8080`.

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
make run           # API on :8080
# Or: make run-watch  # Hot reload via air

# Terminal 2 — Frontend
cd handloom-admin-frontend
npm run dev:local  # Vite on :5173 → localhost:8080
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

The frontend supports three targets via Vite modes:

| Script | Target | Env File | Use Case |
|--------|--------|----------|----------|
| `npm run dev:local` | `localhost:8080` | `.env.local-backend` | Daily dev against monolith |
| `npm run dev:lambda` | `localhost:4566` | `.env.local-lambda` | Test against local Lambdas |
| `npm run dev` | AWS dev API | `.env.development` | Test against deployed AWS |

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
| API Server | http://localhost:8080 | Go monolith (when using `make run`) |
| LocalStack | http://localhost:4566 | DynamoDB, S3, Lambda, API Gateway |
| DynamoDB Admin UI | http://localhost:8001 | Browse DynamoDB tables in browser |

### Environment Configuration

Copy `.env.example` to `.env` for custom configuration:

```bash
cp .env.example .env
```

Key environment variables:
- `APP_ENV` - Environment (development/production)
- `APP_DEBUG` - Enable debug logging
- `AWS_ENDPOINT` - LocalStack endpoint (set for local, empty for AWS)
- `AWS_REGION` - AWS region (ap-south-1)
- `JWT_SECRET_KEY` - JWT signing key (change in production!)

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

### Public APIs (B2C)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/pricing/calculate` | POST | Calculate price for custom dimensions |
| `/api/v1/pricing/dimension-options/{categoryId}` | GET | Get dimension options for a category |
| `/api/v1/pricing/bulk-calculate` | POST | Calculate prices for multiple configurations |

### Admin APIs

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
curl -X POST http://localhost:8080/api/v1/pricing/calculate \
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
| `handloom-core` | Core business data | Users, Products, Categories, Designs, Inventory, Artisans, Coupons, Assets |
| `handloom-orders` | Order management | Orders, OrderItems, StatusHistory, Customers |
| `handloom-audit` | Compliance logs (90-day TTL) | AuditLogs |
| `handloom-analytics` | Dashboard metrics (2-year TTL) | DailyMetrics, TopProducts, Alerts |

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
aws --endpoint-url=http://localhost:4566 --region ap-south-1 lambda list-functions

# Check API Gateway
aws --endpoint-url=http://localhost:4566 --region ap-south-1 apigateway get-rest-apis

# View Lambda logs
aws --endpoint-url=http://localhost:4566 --region ap-south-1 logs describe-log-groups
```

### View DynamoDB Data

Visit http://localhost:8001 for the DynamoDB Admin UI.

### Generate Password Hash

```bash
go run scripts/generate-password-hash.go 'YourNewPassword!'
```

## Documentation

- [Design Document](docs/DESIGN.md) - Detailed architecture, entity design, and API contracts

## License

Private - All rights reserved
