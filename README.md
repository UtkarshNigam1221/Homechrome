# Homechrome

Admin panel for a handloom e-commerce platform. Monorepo with two independently deployable projects.

| Directory | Stack | Purpose |
|-----------|-------|---------|
| `handloom-admin/` | Go 1.24, Chi, DynamoDB, AWS Lambda | Backend API (14 microservices) |
| `handloom-admin-frontend/` | React 19, TypeScript, Vite 7, Tailwind CSS 4 | Admin dashboard SPA |

Both deploy to AWS via CDK (written in Go). Domain: `*.homechrome.lldlab.com`.

## Prerequisites

- **Go 1.24+**
- **Node.js 20+** and npm
- **Docker** and Docker Compose
- **AWS CLI v2** (for LocalStack commands)
- **AWS CDK CLI** (for deployment: `npm install -g aws-cdk`)
- **jq** (optional, for JSON formatting)

## Quick Start

```bash
# 1. Start backend infrastructure + API
cd handloom-admin
make setup-local    # Starts LocalStack, creates DynamoDB tables + S3 buckets, seeds data
make run            # API on http://localhost:8080

# 2. In another terminal, start frontend
cd handloom-admin-frontend
npm install         # First time only
npm run dev:local   # Vite on http://localhost:5173 -> backend on :8080
```

Login with `admin@handloom.com` / `Admin@123!`

## Local Development

All AWS services (DynamoDB, S3, Lambda, API Gateway, IAM) are emulated locally via [LocalStack](https://localstack.cloud/). Two development modes are available:

### Mode 1: Monolith (recommended for daily dev)

Runs the entire backend as a single Go process. Fastest iteration cycle.

```bash
# Terminal 1 - Backend
cd handloom-admin
make setup-local        # One-time setup (or after teardown)
make run                # Start API on :8080
# Or: make run-watch    # Hot reload via air

# Terminal 2 - Frontend
cd handloom-admin-frontend
npm run dev:local       # Vite on :5173 -> localhost:8080
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

### Frontend Dev Modes

| Script | Target | Env File | Use Case |
|--------|--------|----------|----------|
| `npm run dev:local` | `localhost:8080` | `.env.local-backend` | Daily dev against monolith |
| `npm run dev:lambda` | `localhost:4566` | `.env.local-lambda` | Test against local Lambdas |
| `npm run dev` | AWS dev API | `.env.development` | Test against deployed AWS |

### Local Services

| Service | URL | Description |
|---------|-----|-------------|
| API Server | http://localhost:8080 | Go monolith (when using `make run`) |
| Frontend | http://localhost:5173 | Vite dev server |
| LocalStack | http://localhost:4566 | DynamoDB, S3, Lambda, API Gateway |
| DynamoDB Admin UI | http://localhost:8001 | Browse tables in browser |

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
make run                  # Start API on :8080 (monolith)
make run-watch            # Hot reload via air
make deploy-local         # Build + deploy Lambdas to LocalStack
make redeploy-local       # Redeploy Lambda code only (faster)
make teardown-local       # Stop all Docker services and remove volumes

# Database
make create-tables        # Create DynamoDB tables in LocalStack
make init-s3              # Create S3 buckets in LocalStack
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
make cdk-deploy-dev       # Build + CDK deploy to dev
make cdk-deploy-prod      # Build + CDK deploy to prod
```

### Frontend (`handloom-admin-frontend/`)

```bash
# Development
npm run dev               # Vite on :5173 -> AWS dev API
npm run dev:local         # Vite on :5173 -> localhost:8080
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

## Project Structure

```
.
├── handloom-admin/                 # Backend (Go)
│   ├── cmd/
│   │   ├── api/                    # Local monolith entry point
│   │   └── lambda/                 # Lambda entry points (14 services)
│   │       ├── auth/
│   │       ├── user/
│   │       ├── catalog/
│   │       └── asset/              # + 10 more services
│   ├── internal/
│   │   ├── config/                 # Configuration loading
│   │   ├── domain/                 # Entities + interfaces
│   │   ├── handler/                # HTTP handlers
│   │   ├── middleware/             # Auth, validation, logging
│   │   ├── service/                # Business logic
│   │   ├── repository/dynamodb/    # Data access layer
│   │   ├── router/                 # Route mounting + Lambda adapter
│   │   └── wire/                   # Compile-time DI (Google Wire)
│   ├── pkg/                        # Shared packages (errors, response, logger)
│   ├── infra/                      # AWS CDK (Database, Storage, API stacks)
│   ├── scripts/                    # Dev scripts (init-local-db, seed-data, etc.)
│   ├── docs/                       # API contracts, design docs, DB schema
│   ├── docker-compose.yml          # LocalStack + DynamoDB Admin UI
│   └── Makefile
│
├── handloom-admin-frontend/        # Frontend (React)
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
- **DynamoDB**: single-table design across 4 tables (core, orders, audit, analytics)
- **Auth**: JWT in HttpOnly cookies, roles: ADMIN / OPERATOR
- **Prices**: stored in paise (1 INR = 100 paise), cursor-based pagination

### Frontend

- **Routing**: React Router v6 with auth guards
- **State**: Zustand (auth, UI) + React Query (server state)
- **API**: Axios with response envelope unwrapping, silent 401 token refresh
- **Forms**: react-hook-form + zod validation
- **Dev proxy**: Vite proxies `/admin/*` and `/api/*` to backend (cookie-compatible same-origin)

### Infrastructure

- Backend CDK: 3 stacks (DatabaseStack, StorageStack, APIStack) -- 4 of 14 services active
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

### Frontend can't reach backend

Make sure you're using the right dev mode:
- `npm run dev:local` requires `make run` (backend on :8080)
- `npm run dev:lambda` requires `make deploy-local` (Lambdas on LocalStack :4566)
- `npm run dev` connects to deployed AWS dev API (no local backend needed)

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
