# Homechrome - Local Setup & Development Guide

This guide walks you through setting up all three projects locally so they can communicate with each other.

## Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| **Go** | 1.24+ | [go.dev/dl](https://go.dev/dl/) |
| **Node.js** | 18+ (recommended 22.x) | [nodejs.org](https://nodejs.org/) or `nvm install 22` |
| **npm** | 10+ | Comes with Node.js |
| **Docker & Docker Compose** | Latest | [docker.com](https://www.docker.com/) |
| **AWS CLI v2** | Latest | `brew install awscli` |
| **Git** | Latest | `brew install git` |

## Architecture Overview (Local)

When running locally, all three projects talk to each other like this:

```
Browser :5173 (Admin Frontend)  ──proxy /admin, /api──>  Backend :8081
Browser :3000 (B2C Storefront)  ──rewrite /api──>        Backend :8081
                                                             |
                                                     ┌──────┴──────┐
                                                     |             |
                                               LocalStack     PostgreSQL
                                               :4566          :5432
                                          (DynamoDB, S3,   (Categories,
                                           SNS, SQS)       Products)
```

- **Backend** runs as a monolith on `localhost:8081`
- **Admin Frontend** (Vite) runs on `localhost:5173` and proxies API calls to `:8081`
- **B2C Storefront** (Next.js) runs on `localhost:3000` and rewrites `/api/*` to `:8081`
- **LocalStack** emulates DynamoDB, S3, SNS, SQS on `localhost:4566`
- **PostgreSQL** runs on `localhost:5432` for catalog data

---

## Step 1: Clone the Repository

```bash
git clone <repo-url> Homechrome
cd Homechrome
```

---

## Step 2: Backend Setup (`handloom-admin/`)

The backend must be running first since both frontends depend on it.

### 2.1 Install Go Dependencies & Dev Tools

```bash
cd handloom-admin
go mod download
make install-tools   # Installs wire, mockgen, golangci-lint, air
```

### 2.2 Configure Environment

```bash
cp .env.example .env
```

The defaults in `.env.example` are pre-configured for local development. Key values:

| Variable | Default | Notes |
|----------|---------|-------|
| `SERVER_PORT` | `8081` | Backend API port |
| `AWS_ENDPOINT` | `http://localhost:4566` | LocalStack endpoint |
| `POSTGRES_DSN` | `postgres://handloom:handloom@localhost:5432/handloom?sslmode=disable` | Local PostgreSQL |
| `JWT_SECRET_KEY` | `dev-secret-key-change-in-production` | Dev JWT secret |
| `EVENT_PUBLISHING_ENABLED` | `false` | Uses in-process LocalPublisher |

### 2.3 Start Infrastructure & Seed Data

```bash
make setup-local
```

This single command:
1. Starts Docker containers (LocalStack, PostgreSQL, DynamoDB Admin UI, pgAdmin)
2. Creates all 7 DynamoDB tables
3. Initializes S3 buckets
4. Sets up SNS topics and SQS queues
5. Runs PostgreSQL migrations (via `migrations/` directory)
6. Seeds sample data (categories, products, admin user)

### 2.4 Start the Backend

```bash
make run          # Standard start on :8081
# OR
make run-watch    # Hot reload via air (recommended for development)
```

### 2.5 Verify It Works

```bash
curl http://localhost:8081/health
# Should return: {"status":"ok"}
```

### 2.6 Default Admin Credentials

| Field | Value |
|-------|-------|
| Email | `admin@handloom.com` |
| Password | `Admin@123!` |

### 2.7 Useful Local Services

| Service | URL | Purpose |
|---------|-----|---------|
| Backend API | http://localhost:8081 | Main API |
| DynamoDB Admin UI | http://localhost:8001 | Browse DynamoDB tables |
| pgAdmin | http://localhost:5050 | PostgreSQL management |

---

## Step 3: Admin Frontend Setup (`handloom-admin-frontend/`)

### 3.1 Install Dependencies

```bash
cd handloom-admin-frontend
npm install
```

### 3.2 Start Dev Server (Local Backend Mode)

```bash
npm run dev:local
```

This starts Vite on `http://localhost:5173` and proxies all `/admin/*` and `/api/*` requests to the backend at `localhost:8081`.

The environment file `.env.local-backend` is used automatically:

```
VITE_API_URL=http://localhost:8081
VITE_APP_ENV=local
VITE_APP_NAME=Handloom Admin (Local)
```

### 3.3 Verify It Works

Open http://localhost:5173 in your browser. You should see the login page. Sign in with the default admin credentials above.

### 3.4 How the Proxy Works

Vite's dev server proxies requests so cookies work same-origin:
- `/admin/*` -> `http://localhost:8081/admin/*`
- `/api/*` -> `http://localhost:8081/api/*`
- Secure flag is stripped from cookies (HTTP, not HTTPS locally)
- SameSite is adjusted from `None` to `Lax` for localhost

---

## Step 4: B2C Storefront Setup (`homechrome-store/`)

### 4.1 Install Dependencies

```bash
cd homechrome-store
npm install
```

### 4.2 Configure Environment

The `.env.local` file should already exist with the correct defaults:

```
NEXT_PUBLIC_API_URL=http://localhost:8081
NEXT_PUBLIC_SITE_URL=http://localhost:3000
```

If it doesn't exist, copy from the example:

```bash
cp .env.example .env.local
```

### 4.3 Start Dev Server

```bash
npm run dev
```

This starts Next.js on `http://localhost:3000`. All `/api/*` requests are rewritten to the backend at `localhost:8081`.

### 4.4 Verify It Works

Open http://localhost:3000 in your browser. You should see the storefront with seeded categories and products.

### 4.5 How the API Rewrite Works

Next.js rewrites in `next.config.ts` handle API routing:
- `/api/*` -> `http://localhost:8081/api/*`
- The Axios client uses relative URLs with `withCredentials: true` for cookie-based auth
- Customer auth uses phone OTP login (MSG91 integration - may need real credentials for full testing)

---

## Running All Three Together

Open three terminal windows/tabs:

```bash
# Terminal 1: Backend
cd handloom-admin
make run-watch

# Terminal 2: Admin Frontend
cd handloom-admin-frontend
npm run dev:local

# Terminal 3: B2C Storefront
cd homechrome-store
npm run dev
```

| Service | URL | Purpose |
|---------|-----|---------|
| Backend API | http://localhost:8081 | Go API (monolith mode) |
| Admin Dashboard | http://localhost:5173 | React admin SPA |
| B2C Storefront | http://localhost:3000 | Next.js customer store |
| DynamoDB Admin | http://localhost:8001 | DynamoDB table browser |
| pgAdmin | http://localhost:5050 | PostgreSQL management |

---

## Alternative: Lambda Mode

Instead of running the backend as a monolith, you can deploy Lambdas to LocalStack for a production-like setup.

### Start Lambda Mode

```bash
cd handloom-admin
make deploy-local    # Build and deploy all Lambdas to LocalStack
```

After deployment, note the API Gateway URL printed in the output (it will look like `http://localhost:4566/restapis/<id>/local/_user_request_`).

### Connect Frontends to Lambda Mode

**Admin Frontend:**
1. Update the REST API ID in `.env.local-lambda` if needed
2. Run `npm run dev:lambda`

**B2C Storefront:**
1. Update the REST API ID in `.env.local-lambda` if needed
2. Run `npm run dev:lambda`

> **Note:** Lambda mode is slower for iteration. Use monolith mode (`make run-watch`) for day-to-day development.

---

## Common Development Tasks

### Backend

```bash
cd handloom-admin

# Testing
make test                 # All tests (unit + integration)
make test-unit            # Unit tests only (no Docker needed)
make test-integration     # Integration tests (needs LocalStack running)
go test -v -run TestName ./internal/service/...  # Single test

# Code generation
make wire                 # Regenerate dependency injection (after wiring changes)
make generate-mocks       # Regenerate mocks (after interface changes)

# Linting
golangci-lint run

# Rebuild Lambdas (Lambda mode)
make redeploy-local       # Faster - reuses existing infrastructure
```

### Admin Frontend

```bash
cd handloom-admin-frontend

# Quality checks
npm run check             # Full CI check: typecheck + lint + format
npm run typecheck         # TypeScript only
npm run lint:fix          # Auto-fix lint issues
npm run format            # Auto-format with Prettier

# Testing
npm run test              # Run Vitest
npm run test:watch        # Watch mode

# Build
npm run build             # Production build
```

### B2C Storefront

```bash
cd homechrome-store

npm run lint              # ESLint
npm run build             # Production build
npm run start             # Start production server (after build)
```

---

## Troubleshooting

### Backend won't start

- **Port 8081 in use**: Kill the existing process: `lsof -ti:8081 | xargs kill`
- **Docker not running**: Start Docker Desktop, then `make setup-local`
- **LocalStack issues**: `make teardown-local` then `make setup-local` for a clean start

### Frontend can't reach backend

- **Backend not running**: Ensure `make run` or `make run-watch` is active on `:8081`
- **CORS issues**: The Vite proxy (admin) and Next.js rewrites (store) handle CORS by making requests same-origin. Ensure you're accessing via `localhost`, not `127.0.0.1`
- **Cookie issues**: Admin frontend Vite config strips Secure flag and adjusts SameSite for localhost. If auth isn't working, check browser dev tools for cookie warnings

### Database issues

- **DynamoDB tables missing**: `make create-tables`
- **PostgreSQL not reachable**: Check Docker: `docker ps | grep handloom-postgres`
- **Stale data**: `make teardown-local && make setup-local` for a full reset
- **View DynamoDB data**: Open http://localhost:8001
- **View PostgreSQL data**: Open http://localhost:5050 (pgAdmin)

### Lambda mode issues

- **API Gateway URL changed**: After `make deploy-local`, update the REST API ID in `.env.local-lambda` files for both frontends
- **Lambda not updating**: Use `make redeploy-local` to push new code without recreating infrastructure

### Storefront auth (OTP)

- Phone OTP login requires MSG91 credentials (`MSG91_AUTH_KEY`, `MSG91_OTP_TEMPLATE_ID` in backend `.env`)
- For local testing without real SMS, check backend logs for the OTP value or configure test credentials

---

## Teardown

To stop all local services and clean up:

```bash
cd handloom-admin
make teardown-local    # Stops Docker containers and removes volumes
```

This removes all LocalStack data, PostgreSQL data, and Docker volumes. Run `make setup-local` to start fresh.
