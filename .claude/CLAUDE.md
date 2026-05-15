# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Homechrome is a monorepo for a handloom e-commerce platform. It contains three independently deployable projects:

| Directory | Stack | Purpose |
|-----------|-------|---------|
| `handloom-admin/` | Go 1.25, Chi, DynamoDB + PostgreSQL, AWS Lambda | Backend API (25 Lambdas in dev; 29 with event stack enabled) |
| `handloom-admin-frontend/` | React 19, TypeScript, Vite 7, Tailwind CSS 4 | Admin dashboard SPA |
| `homechrome-store/` | Next.js 16, React 19, Tailwind CSS 4 | B2C customer storefront |

All deploy to AWS via CDK (written in Go). The brand domain is `*.homechrome.in`.

**For detailed backend guidance, see `handloom-admin/CLAUDE.md`.**

## Common Commands

### Backend (`handloom-admin/`)
```bash
cd handloom-admin
make setup-local          # LocalStack + DynamoDB tables + S3 buckets + SNS/SQS + seed data
make run                  # Local API on :8081 (monolith mode)
make run-watch            # Hot reload via air
make deploy-local         # Build + deploy Lambdas to LocalStack (Lambda mode)
make redeploy-local       # Redeploy Lambda code only (faster)
make teardown-local       # Stop all Docker services and remove volumes
make test                 # go test -v -race -cover ./...
make test-unit            # Unit tests only (internal/service/...)
make test-integration     # Integration tests (needs LocalStack)
go test -v -run TestName ./internal/service/...   # Single test
make test-coverage        # Tests with HTML coverage report → coverage.html
make wire                 # Regenerate DI after wiring changes
make generate-mocks       # Regenerate mocks after interface changes
make install-tools        # Install Wire, mockgen, golangci-lint, air
golangci-lint run         # Lint (uses .golangci.yml)
make build-lambdas-active # Build active lambdas (auth, user, catalog, asset)
make cdk-deploy-dev       # Build + deploy to dev
make cdk-diff-dev         # CDK diff against dev (preview changes)
make reset-db             # Drop and recreate DynamoDB + PostgreSQL tables
make health               # Check local service health
```

### Admin Frontend (`handloom-admin-frontend/`)
```bash
cd handloom-admin-frontend
npm run dev               # Vite dev server on :5173 → deployed AWS dev API
npm run dev:local         # Vite on :5173 → localhost:8081 (monolith backend)
npm run dev:lambda        # Vite on :5173 → localhost:4566 (LocalStack Lambda)
npm run build             # TypeScript check + Vite production build
npm run build:dev         # Build for dev environment
npm run build:prod        # Build for prod environment
npm run lint              # ESLint (.ts, .tsx)
npm run lint:fix          # ESLint with auto-fix
npm run format            # Prettier write
npm run format:check      # Prettier check
npm run typecheck         # tsc --noEmit
npm run check             # typecheck + lint + format:check (full CI check)
npm run test              # Vitest (Testing Library + JSDOM)
npm run test:watch        # Vitest in watch mode
npm run cdk:deploy:dev    # Build + CDK deploy to dev
npm run cdk:deploy:prod   # Build + CDK deploy to prod
```

### B2C Storefront (`homechrome-store/`)
```bash
cd homechrome-store
npm run dev               # Next.js on :3000 → localhost:8081 (monolith backend)
npm run dev:lambda        # Next.js on :3000 → localhost:4566 (LocalStack Lambda)
npm run build             # Next.js production build
npm run build:dev         # Build for dev environment
npm run build:prod        # Build for prod environment
npm run open-next:build   # OpenNext build (.next/ → .open-next/ for Lambda deployment)
npm run start             # Start production server
npm run lint              # ESLint
npm run cdk:deploy:dev    # Build + OpenNext + CDK deploy to dev
npm run cdk:deploy:prod   # Build + OpenNext + CDK deploy to prod
```

## Architecture

### Monorepo Structure
- **No shared package manager or workspace** — each project is independent
- Backend uses Go modules (`github.com/handloom/admin`), frontends use npm
- Backend and admin frontend each have their own `infra/` directory with AWS CDK stacks (Go)
- Husky pre-commit hooks run lint-staged on admin frontend files (configured in `handloom-admin-frontend/package.json`)

### Backend Architecture (see `handloom-admin/CLAUDE.md` for details)
Clean architecture with domain-driven layers:
```
domain/ (entities + interfaces) ← handler/ → service/ → repository/{dynamodb,postgres}/
```
- Dual entry points: monolith locally (`cmd/api/main.go`), separate Lambda binaries in prod (`cmd/lambda/<service>/main.go`)
- Hybrid database: DynamoDB (7 tables: core, orders, sessions, audit, analytics, notifications, events) + PostgreSQL (catalog data)
- `internal/repository/postgres/` — PostgreSQL implementations for catalog (categories, products, inventory)
- Google Wire for compile-time DI, Chi router, JWT auth via HttpOnly cookies
- Prices in **paise** (1 INR = 100 paise), cursor-based pagination

### Admin Frontend Architecture
- **Routing**: React Router v6 in `App.tsx` — all routes defined in one file with auth guards
- **State**: Zustand stores (`src/stores/`) for auth and UI state; React Query for server state
- **API layer**: Axios client (`src/api/client.ts`) with automatic response envelope unwrapping (`{success, data}` → `data`), 401 interceptor with silent token refresh queue
- **Dev proxy**: Vite proxies `/admin/*` to `http://localhost:8081` so backend cookies work same-origin
- **Path alias**: `@/` maps to `src/` (configured in `vite.config.ts`)
- **Pages pattern**: Each page domain in `src/pages/<domain>/` has a list page, form modal, and barrel `index.ts`
- **Shared components**: `src/components/common/` (Button, Modal, Table, Input, etc.) and `src/components/layout/` (MainLayout, Sidebar, Header)
- **Forms**: react-hook-form + zod for validation
- **Build**: Vite with manual chunk splitting (vendor-react, vendor-state, vendor-ui, vendor-charts, vendor-forms, vendor-utils)

### B2C Storefront Architecture (`homechrome-store/`)
- **Framework**: Next.js 16 with App Router (server + client components)
- **State**: Zustand + React Query (same pattern as admin frontend)
- **API layer**: Axios client (`src/lib/api.ts`) with response envelope unwrapping, auto 401 token refresh
- **Routing**: Next.js App Router — `/c/[slug]` (categories), `/p/[slug]` (products), `/cart`, `/checkout`, `/login`, `/account`, `/track`
- **Auth**: Phone OTP login (MSG91), customer JWT in HttpOnly cookies
- **SEO**: JSON-LD structured data, dynamic sitemap.ts, robots.ts
- **API rewrites**: Next.js rewrites `/api/*` to backend (`NEXT_PUBLIC_API_URL`)
- **Backend routes**: Consumes `/api/v1/store/*` (auth, catalog, cart, checkout, orders, profile, tracking, events, webhooks)
- **Deployment**: OpenNext (`@opennextjs/aws`) compiles `.next/` → `.open-next/` for Lambda. CDK deploys CloudFront + Server Lambda (Node.js 20, ARM64, 128MB) + Image Lambda (256MB for Sharp). 3 cache behaviors: default, `_next/static/*`, `_next/image`. S3 for static assets (`_assets/`, `_cache/`) and ISR cache seeding.

### Event-Driven Architecture
- **Three publishers**: `SNSPublisher` (Lambda mode with events enabled), `LocalPublisher` (monolith mode, calls handlers in-process), `NoopPublisher` (Lambda mode with events disabled — logs and discards)
- **Dev: event stack disabled** — EventStack (SNS + SQS + 4 worker Lambdas + EventBridge rule) is commented out in `infra/cmd/main.go` for cost savings. `nil` is passed to APIStack, which handles it gracefully (no `SNS_TOPIC_ARN`, no `EVENT_PUBLISHING_ENABLED`). To re-enable: uncomment the EventStack block and pass it to APIStack.
- SNS fan-out (when enabled): single `handloom-events-{env}` topic → 4 SQS queues (with DLQs) → 4 worker Lambdas: `worker-notification`, `worker-report`, `worker-analytics`, `worker-audit`
- 20 event types across 7 categories: `order.*`, `payment.*`, `shipment.*`, `product.*`, `inventory.*`, `customer.*`, `admin.*`
- `internal/event/` — publisher + event types; `internal/event/handlers/` — SQS consumers (also implement `EventHandler` for `LocalPublisher`)
- Fire-and-forget: publish errors are logged, never propagate to callers
- Config: `SNS_TOPIC_ARN`, `EVENT_PUBLISHING_ENABLED`

### Infrastructure
- **All CDK stacks are Go** — each project has its own `infra/` directory with `cmd/main.go` entry point
- Backend CDK: 4 stacks (DatabaseStack, StorageStack, APIStack, EventStack) — EventStack is currently disabled in dev (commented out in `infra/cmd/main.go`)
- Admin Frontend CDK: S3 static hosting (dev) or CloudFront + S3 (prod), custom domain via ACM cert
- Storefront CDK: CloudFront + S3 + Server Lambda + Image Lambda (OpenNext architecture), custom domain via ACM cert (requires `certArn` context param in us-east-1)
- All backend Lambdas: ARM64, `provided.al2023`, 128MB (dev) / 256MB (prod)
- Lambda count: 25 in dev (12 admin + 9 store + 1 migrator + 3 cron), 29 with event stack (+ 4 workers)
- Region: `ap-south-1` (Mumbai)

## Key Conventions

- **Backend error handling**: Services return `*errors.AppError`; handlers call `response.Error(w, err)` — standard JSON envelope `{success, error: {code, message}}`
- **Backend validation**: `middleware.ValidateJSONTyped[T]` as Chi middleware, then `middleware.MustGetValidatedBody[T]` in handler
- **Admin frontend env vars**: Prefixed with `VITE_` — files are `.env.local-backend` (monolith), `.env.local-lambda` (LocalStack Lambda), `.env.development` (AWS dev), `.env.dev` (AWS dev build), `.env.prod` (AWS prod build)
- **Storefront env vars**: Prefixed with `NEXT_PUBLIC_` — files are `.env.local` (monolith), `.env.local-lambda` (LocalStack Lambda)
- **Admin frontend imports**: `eslint-plugin-simple-import-sort` enforces import ordering; use `@/` path alias
- **Lint-staged**: On commit, `.ts/.tsx` files get eslint --fix + prettier; `.css/.json/.md` get prettier
