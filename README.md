# Homechrome

Handloom e-commerce platform. Monorepo with three independently deployable projects.

| Directory | Stack | Purpose |
|-----------|-------|---------|
| `handloom-admin/` | Go 1.24, Chi, DynamoDB + PostgreSQL, AWS Lambda | Backend API (12 admin + 9 store + 4 worker Lambdas) |
| `handloom-admin-frontend/` | React 19, TypeScript, Vite 7, Tailwind CSS 4 | Admin dashboard SPA |
| `homechrome-store/` | Next.js 16, React 19, Tailwind CSS 4 | B2C customer storefront |

All projects deploy to AWS via CDK (written in Go). Domain: `*.homechrome.in`.

## Prerequisites

- **Go 1.24+**
- **Node.js 20+** and npm
- **Docker** and Docker Compose
- **AWS CLI v2** (for LocalStack commands)
- **awscli-local** (for LocalStack shortcuts: `pip install awscli-local`)
- **AWS CDK CLI** (for deployment: `npm install -g aws-cdk`)
- **jq** (optional, for JSON formatting)

## Quick Start

```bash
# 1. Start backend infrastructure + API
cd handloom-admin
make setup-local    # Starts LocalStack, creates DynamoDB tables + S3 buckets, seeds data
make run            # API on http://localhost:8081

# 2. In another terminal, start admin frontend
cd handloom-admin-frontend
npm install         # First time only
npm run dev:local   # Vite on http://localhost:5173 -> backend on :8081

# 3. (Optional) In another terminal, start B2C storefront
cd homechrome-store
npm install         # First time only
npm run dev         # Next.js on http://localhost:3000 -> backend on :8081
```

Admin login: `admin@handloom.com` / `Admin@123!`

## Local Development

All AWS services (DynamoDB, S3, Lambda, API Gateway, IAM, SNS, SQS) are emulated locally via [LocalStack](https://localstack.cloud/). Two backend modes are available:

### Mode 1: Monolith (recommended for daily dev)

Runs the entire backend as a single Go process. Fastest iteration cycle.

```bash
# Terminal 1 - Backend
cd handloom-admin
make setup-local        # One-time setup (or after teardown)
make run                # Start API on :8081
# Or: make run-watch    # Hot reload via air

# Terminal 2 - Admin Frontend
cd handloom-admin-frontend
npm run dev:local       # Vite on :5173 -> localhost:8081

# Terminal 3 - B2C Storefront (optional)
cd homechrome-store
npm run dev             # Next.js on :3000 -> localhost:8081
```

### Mode 2: Lambda (integration testing)

Deploys actual Lambda binaries to LocalStack with API Gateway, matching production topology.

```bash
# Terminal 1 - LocalStack + Lambdas
cd handloom-admin
make setup-local        # One-time setup
make deploy-local       # Build 4 Lambdas + deploy to LocalStack with API Gateway

# Terminal 2 - Frontend
cd handloom-admin-frontend
npm run dev:lambda      # Vite on :5173 -> localhost:4566 (API Gateway)
```

After code changes, redeploy without recreating API Gateway:

```bash
make redeploy-local     # Rebuild + update Lambda code only (faster)
```

### Viewing Lambda Logs

```bash
cd handloom-admin

# Tail all LocalStack logs (Lambda invocations, API Gateway, errors)
make logs

# Tail logs for a specific Lambda function
make logs-lambda SVC=catalog        # Admin catalog service
make logs-lambda SVC=store-catalog  # B2C store catalog
make logs-lambda SVC=store-cart     # B2C cart service
make logs-lambda SVC=auth           # Admin auth service
```

Requires `awscli-local` (`pip install awscli-local`) for per-service logs.

### Admin Frontend Dev Modes

| Script | Target | Env File | Use Case |
|--------|--------|----------|----------|
| `npm run dev:local` | `localhost:8081` | `.env.local-backend` | Daily dev against monolith |
| `npm run dev:lambda` | `localhost:4566` | `.env.local-lambda` | Test against local Lambdas |
| `npm run dev` | AWS dev API | `.env.development` | Test against deployed AWS |

### B2C Storefront Dev Modes

| Script | Target | Env File | Use Case |
|--------|--------|----------|----------|
| `npm run dev` | `localhost:8081` | `.env.local` | Daily dev against monolith |
| `npm run dev:lambda` | `localhost:4566` | `.env.local-lambda` | Test against local Lambdas |

### Local Services

| Service | URL | Description |
|---------|-----|-------------|
| API Server | http://localhost:8081 | Go monolith (when using `make run`) |
| Admin Frontend | http://localhost:5173 | Vite dev server |
| B2C Storefront | http://localhost:3000 | Next.js dev server |
| LocalStack | http://localhost:4566 | DynamoDB, S3, Lambda, API Gateway, SNS, SQS |
| DynamoDB Admin UI | http://localhost:8001 | Browse DynamoDB tables in browser |
| pgAdmin | http://localhost:5050 | Browse PostgreSQL tables in browser |

### Default Credentials

| Role | Email | Password |
|------|-------|----------|
| Admin | admin@handloom.com | Admin@123! |
| Manager | manager@handloom.com | Admin@123! |

## Common Commands

### Backend (`handloom-admin/`)

```bash
# Local development
make setup-local          # Full setup: LocalStack + tables + S3 + seed data
make run                  # Start API on :8081 (monolith)
make run-watch            # Hot reload via air
make deploy-local         # Build + deploy Lambdas to LocalStack
make redeploy-local       # Redeploy Lambda code only (faster)
make teardown-local       # Stop all Docker services and remove volumes
make logs                 # Tail all LocalStack logs
make logs-lambda SVC=x    # Tail logs for a specific Lambda (e.g., SVC=catalog)

# Database & Infrastructure
make create-tables        # Create DynamoDB tables in LocalStack
make init-s3              # Create S3 buckets in LocalStack
make create-events        # Create SNS topic + SQS queues in LocalStack
make seed-data            # Seed test data
make reset-db             # Drop and recreate all tables + reseed
make list-tables          # List DynamoDB tables

# Testing
make test                 # All tests (go test -v -race -cover)
make test-unit            # Unit tests only (no Docker needed)
make test-integration     # Integration tests (needs LocalStack)
make health               # Check local service health
make test-login           # Test login endpoint
make test-api             # Quick API smoke test

# Code generation
make wire                 # Regenerate DI (after wiring changes)
make generate-mocks       # Regenerate mocks (after interface changes)

# Lint
golangci-lint run

# Build + Deploy
make build-lambdas-active # Build active lambdas (auth, user, catalog, asset)
make build-workers        # Build all 4 worker Lambdas
make cdk-deploy-dev       # Build + CDK deploy to dev
make cdk-deploy-prod      # Build + CDK deploy to prod
```

### Admin Frontend (`handloom-admin-frontend/`)

```bash
# Development
npm run dev               # Vite on :5173 -> AWS dev API
npm run dev:local         # Vite on :5173 -> localhost:8081
npm run dev:lambda        # Vite on :5173 -> localhost:4566

# Quality checks
npm run check             # typecheck + lint + format:check (full CI check)
npm run typecheck         # tsc --noEmit
npm run lint              # ESLint
npm run lint:fix          # ESLint with auto-fix
npm run format            # Prettier write
npm run format:check      # Prettier check

# Testing
npm run test              # Vitest run
npm run test:watch        # Vitest watch mode

# Build + Deploy
npm run build             # TypeScript check + Vite production build
npm run build:dev         # Build for dev environment
npm run build:prod        # Build for prod environment
npm run cdk:deploy:dev    # Build + CDK deploy to dev
npm run cdk:deploy:prod   # Build + CDK deploy to prod
```

### B2C Storefront (`homechrome-store/`)

```bash
# Development
npm run dev               # Next.js on :3000 -> localhost:8081
npm run dev:lambda        # Next.js on :3000 -> localhost:4566

# Quality
npm run lint              # ESLint

# Build + Run
npm run build             # Next.js production build
npm run start             # Start production server
```

## Project Structure

```
.
├── handloom-admin/                 # Backend (Go)
│   ├── cmd/
│   │   ├── api/                    # Local monolith entry point
│   │   └── lambda/                 # Lambda entry points (25 services)
│   │       ├── auth/               # Admin services (12)
│   │       ├── user/
│   │       ├── catalog/
│   │       ├── asset/              # + 8 more admin services
│   │       ├── store-auth/         # B2C store services (9)
│   │       ├── store-catalog/
│   │       ├── store-cart/
│   │       ├── store-checkout/     # + 5 more store services
│   │       ├── worker-analytics/   # Event workers (4)
│   │       ├── worker-audit/
│   │       ├── worker-notification/
│   │       └── worker-report/
│   ├── internal/
│   │   ├── config/                 # Configuration loading
│   │   ├── domain/                 # Entities + interfaces
│   │   ├── handler/                # HTTP handlers (admin)
│   │   │   └── store/              # B2C store handlers
│   │   ├── middleware/             # Auth, validation, logging
│   │   ├── service/                # Business logic
│   │   ├── repository/             # Data access layer
│   │   │   ├── dynamodb/          # DynamoDB implementations
│   │   │   └── postgres/          # PostgreSQL implementations (catalog)
│   │   ├── gateway/                # External integrations
│   │   │   ├── phonepe/            # PhonePe payment gateway
│   │   │   ├── shiprocket/         # Shiprocket shipping
│   │   │   └── sms/                # MSG91 SMS/OTP
│   │   ├── event/                  # Event-driven architecture
│   │   │   ├── publisher.go        # SNS event publisher
│   │   │   ├── types.go            # Event type definitions
│   │   │   └── handlers/           # SQS worker handlers
│   │   ├── router/                 # Route mounting + Lambda adapter
│   │   └── wire/                   # Compile-time DI (Google Wire)
│   ├── pkg/                        # Shared packages (errors, response, logger)
│   ├── infra/                      # AWS CDK (Database, Storage, API, Event stacks)
│   ├── scripts/                    # Dev scripts (init-local-db, seed-data, etc.)
│   ├── docs/                       # API contracts, design docs, DB schema
│   ├── docker-compose.yml          # LocalStack + DynamoDB Admin + PostgreSQL + pgAdmin
│   └── Makefile
│
├── handloom-admin-frontend/        # Admin Frontend (React)
│   ├── src/
│   │   ├── app/                    # App.tsx (routes), providers
│   │   ├── features/               # Feature modules (settings, etc.)
│   │   ├── pages/                  # Page components by domain
│   │   ├── shared/
│   │   │   ├── components/         # Common UI (Button, Modal, Table, etc.)
│   │   │   ├── stores/             # Zustand stores (auth, ui)
│   │   │   ├── hooks/              # Custom hooks
│   │   │   └── utils/              # Utilities
│   │   └── api/                    # Axios client + API functions
│   ├── infra/                      # AWS CDK (S3 / CloudFront hosting)
│   └── package.json
│
├── homechrome-store/               # B2C Storefront (Next.js)
│   ├── src/
│   │   ├── app/                    # Next.js App Router pages
│   │   │   ├── c/[slug]/           # Category pages
│   │   │   ├── p/[slug]/           # Product pages
│   │   │   ├── cart/               # Shopping cart
│   │   │   ├── checkout/           # Checkout flow
│   │   │   ├── login/              # Phone OTP login
│   │   │   ├── account/            # Customer account
│   │   │   ├── track/              # Order tracking
│   │   │   └── categories/         # Category listing
│   │   ├── components/             # UI components
│   │   ├── lib/                    # API client + utilities
│   │   ├── stores/                 # Zustand stores
│   │   ├── hooks/                  # Custom hooks
│   │   └── types/                  # TypeScript types
│   └── package.json
│
├── scripts/                        # Monorepo-level scripts
│   └── setup-hooks.sh              # Git hooks setup
└── CLAUDE.md                       # AI coding assistant guidance
```

## Architecture

### Backend

Clean architecture with domain-driven layers:

```
domain/ (entities + interfaces) <- handler/ -> service/ -> repository/dynamodb/
```

- **Dual entry points**: monolith locally (`cmd/api/main.go`), separate Lambda binaries in prod (`cmd/lambda/<service>/main.go`)
- **Hybrid database**: DynamoDB (7 tables: core, orders, sessions, audit, analytics, notifications, events) + PostgreSQL (catalog data)
- **Admin auth**: JWT in HttpOnly cookies, roles: ADMIN / OPERATOR
- **B2C store auth**: Phone OTP via MSG91, customer JWT in cookies
- **Payments**: PhonePe payment gateway integration
- **Shipping**: Shiprocket integration for delivery and tracking
- **Prices**: stored in paise (1 INR = 100 paise), cursor-based pagination
- **Event-driven**: SNS fan-out → 4 SQS queues (with DLQs + filter policies) → worker Lambdas (notification, report, analytics, audit). `LocalPublisher` for monolith mode, `SNSPublisher` for Lambda mode. 20 event types across 7 categories. Fire-and-forget publishing.

### Admin Frontend

- **Routing**: React Router v6 with auth guards
- **State**: Zustand (auth, UI) + React Query (server state)
- **API**: Axios with response envelope unwrapping, silent 401 token refresh
- **Forms**: react-hook-form + zod validation
- **Dev proxy**: Vite proxies `/admin/*` and `/api/*` to backend (cookie-compatible same-origin)

### B2C Storefront

- **Framework**: Next.js 16 with App Router (server + client components)
- **State**: Zustand + React Query
- **API**: Axios client with `/api/*` rewrites to backend, auto token refresh
- **Auth**: Phone number + OTP login, customer JWT in cookies
- **Pages**: Homepage, category browsing, product detail, cart, checkout, account, order tracking
- **SEO**: JSON-LD structured data, dynamic sitemap, robots.txt

### Infrastructure

- Backend CDK: 4 stacks (DatabaseStack, StorageStack, APIStack, EventStack)
- Frontend CDK: S3 static hosting (dev) or CloudFront + S3 (prod)
- All Lambdas: ARM64, `provided.al2023`, 128MB (dev) / 256MB (prod)
- Region: `ap-south-1`

## Troubleshooting

### LocalStack not starting

```bash
# Check if port 4566 is in use
lsof -i :4566

# Full reset
cd handloom-admin
make teardown-local
make setup-local
```

### Inspecting local database data

**DynamoDB** — Browse tables in the **DynamoDB Admin UI** at http://localhost:8001.

From the CLI:

```bash
# List all tables
aws --endpoint-url=http://localhost:4566 --region ap-south-1 dynamodb list-tables

# Scan a table
aws --endpoint-url=http://localhost:4566 --region ap-south-1 dynamodb scan --table-name handloom-core --max-items 10

# Get a specific item
aws --endpoint-url=http://localhost:4566 --region ap-south-1 dynamodb get-item \
  --table-name handloom-core \
  --key '{"PK": {"S": "USER#<id>"}, "SK": {"S": "METADATA"}}'

# Count items in a table
aws --endpoint-url=http://localhost:4566 --region ap-south-1 dynamodb scan --table-name handloom-orders --select COUNT
```

**PostgreSQL (catalog data)** — Browse tables in **pgAdmin** at http://localhost:5050. On first use, register a server: host `postgres`, port `5432`, user/password `handloom`. Then navigate to Servers > handloom > Databases > handloom > Schemas > public > Tables, right-click a table > View/Edit Data > All Rows.

From the CLI:

```bash
# Interactive psql shell
docker exec -it handloom-postgres psql -U handloom -d handloom

# List tables
\dt

# Query data
SELECT * FROM categories;
SELECT * FROM products;
SELECT * FROM inventory;

# Exit
\q
```

> **Note:** PostgreSQL schema is auto-created on first `docker-compose up` via `migrations/001_catalog_schema.sql`. Tables are populated when you run `make seed-data` with the API server running (`make run`).

### Frontend can't reach backend

Make sure you're using the right dev mode:
- **Admin frontend** `npm run dev:local` requires `make run` (backend on :8081)
- **Admin frontend** `npm run dev:lambda` requires `make deploy-local` (Lambdas on LocalStack :4566)
- **Admin frontend** `npm run dev` connects to deployed AWS dev API (no local backend needed)
- **B2C storefront** `npm run dev` requires `make run` (backend on :8081)
- **B2C storefront** `npm run dev:lambda` requires `make deploy-local` (Lambdas on :4566)

### Lambda deployment issues

```bash
# Check deployed Lambdas
aws --endpoint-url=http://localhost:4566 --region ap-south-1 lambda list-functions

# Check API Gateway
aws --endpoint-url=http://localhost:4566 --region ap-south-1 apigateway get-rest-apis
```

### Reset everything

```bash
cd handloom-admin
make teardown-local    # Stop Docker, remove volumes
make setup-local       # Recreate everything
```

## License

Private - All rights reserved
