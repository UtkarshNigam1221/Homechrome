# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Homechrome is a monorepo for a handloom e-commerce admin panel. It contains two independently deployable projects:

| Directory | Stack | Purpose |
|-----------|-------|---------|
| `handloom-admin/` | Go 1.24, Chi, DynamoDB, AWS Lambda | Backend API (14 microservices) |
| `handloom-admin-frontend/` | React 19, TypeScript, Vite 7, Tailwind CSS 4 | Admin dashboard SPA |

Both deploy to AWS via CDK (written in Go). The brand domain is `*.homechrome.lldlab.com`.

**For detailed backend guidance, see `handloom-admin/CLAUDE.md`.**

## Common Commands

### Backend (`handloom-admin/`)
```bash
cd handloom-admin
make setup-local          # LocalStack + DynamoDB tables + S3 buckets + seed data
make run                  # Local API on :8080 (monolith mode)
make run-watch            # Hot reload via air
make deploy-local         # Build + deploy Lambdas to LocalStack (Lambda mode)
make redeploy-local       # Redeploy Lambda code only (faster)
make teardown-local       # Stop all Docker services and remove volumes
make test                 # go test -v -race -cover ./...
make test-unit            # Unit tests only (internal/service/...)
make test-integration     # Integration tests (needs LocalStack)
go test -v -run TestName ./internal/service/...   # Single test
make wire                 # Regenerate DI after wiring changes
make generate-mocks       # Regenerate mocks after interface changes
golangci-lint run         # Lint (uses .golangci.yml)
make build-lambdas-active # Build active lambdas (auth, user, catalog, asset)
make cdk-deploy-dev       # Build + deploy to dev
```

### Frontend (`handloom-admin-frontend/`)
```bash
cd handloom-admin-frontend
npm run dev               # Vite dev server on :5173 → deployed AWS dev API
npm run dev:local         # Vite on :5173 → localhost:8080 (monolith backend)
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
npm run cdk:deploy:dev    # Build + CDK deploy to dev
npm run cdk:deploy:prod   # Build + CDK deploy to prod
```

## Architecture

### Monorepo Structure
- **No shared package manager or workspace** — each project is independent
- Backend uses Go modules (`github.com/handloom/admin`), frontend uses npm
- Each has its own `infra/` directory with AWS CDK stacks (Go)
- Husky pre-commit hooks run lint-staged on frontend files (configured in `handloom-admin-frontend/package.json`)

### Backend Architecture (see `handloom-admin/CLAUDE.md` for details)
Clean architecture with domain-driven layers:
```
domain/ (entities + interfaces) ← handler/ → service/ → repository/dynamodb/
```
- Dual entry points: monolith locally (`cmd/api/main.go`), separate Lambda binaries in prod (`cmd/lambda/<service>/main.go`)
- DynamoDB single-table design across 4 tables (core, orders, audit, analytics)
- Google Wire for compile-time DI, Chi router, JWT auth via HttpOnly cookies
- Prices in **paise** (1 INR = 100 paise), cursor-based pagination

### Frontend Architecture
- **Routing**: React Router v6 in `App.tsx` — all routes defined in one file with auth guards
- **State**: Zustand stores (`src/stores/`) for auth and UI state; React Query for server state
- **API layer**: Axios client (`src/api/client.ts`) with automatic response envelope unwrapping (`{success, data}` → `data`), 401 interceptor with silent token refresh queue
- **Dev proxy**: Vite proxies `/admin/*` to `http://localhost:8080` so backend cookies work same-origin
- **Path alias**: `@/` maps to `src/` (configured in `vite.config.ts`)
- **Pages pattern**: Each page domain in `src/pages/<domain>/` has a list page, form modal, and barrel `index.ts`
- **Shared components**: `src/components/common/` (Button, Modal, Table, Input, etc.) and `src/components/layout/` (MainLayout, Sidebar, Header)
- **Forms**: react-hook-form + zod for validation
- **Build**: Vite with manual chunk splitting (vendor-react, vendor-state, vendor-ui, vendor-charts, vendor-forms, vendor-utils)

### Infrastructure
- Backend CDK: 3 stacks (DatabaseStack, StorageStack, APIStack) — only 4 of 14 services active
- Frontend CDK: S3 static hosting (dev) or CloudFront + S3 (prod), custom domain via ACM cert
- All Lambdas: ARM64, `provided.al2023`, 128MB (dev) / 256MB (prod)
- Region: `ap-south-1`

## Key Conventions

- **Backend error handling**: Services return `*errors.AppError`; handlers call `response.Error(w, err)` — standard JSON envelope `{success, error: {code, message}}`
- **Backend validation**: `middleware.ValidateJSONTyped[T]` as Chi middleware, then `middleware.MustGetValidatedBody[T]` in handler
- **Frontend env vars**: Prefixed with `VITE_` — files are `.env.local-backend` (monolith), `.env.local-lambda` (LocalStack Lambda), `.env.development` (AWS dev), `.env.dev` (AWS dev build), `.env.prod` (AWS prod build)
- **Frontend imports**: `eslint-plugin-simple-import-sort` enforces import ordering; use `@/` path alias
- **Lint-staged**: On commit, `.ts/.tsx` files get eslint --fix + prettier; `.css/.json/.md` get prettier
