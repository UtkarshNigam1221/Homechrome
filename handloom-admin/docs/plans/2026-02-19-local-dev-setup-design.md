# Local Development Setup Design

**Date:** 2026-02-19
**Status:** Approved

## Goal

Make the full backend testable locally — both as a monolith (fast dev) and as Lambda functions (integration testing). Frontend should be able to target either mode or the deployed AWS dev backend.

## Current State

- Frontend works locally via Vite dev server with proxy to `:8080`
- DynamoDB Local runs in Docker on port 8000 — all 14 services' CRUD works
- S3 has no local substitute — `s3client.New()` requires real AWS credentials, crashes on startup without them
- Asset uploads (presigned URL flow) are completely broken locally
- `.env.development` points Vite proxy at the deployed AWS dev API, not localhost

## Design

### 1. LocalStack as Unified AWS Emulator

Replace DynamoDB Local with LocalStack. One container, one endpoint (`localhost:4566`), all AWS services.

**docker-compose.yml changes:**
- Remove: `dynamodb-local` container
- Add: LocalStack container with `SERVICES=dynamodb,s3,lambda,apigateway,iam`
- Keep: DynamoDB Admin UI, re-pointed at `http://localstack:4566`

### 2. Two Local Backend Modes

| Mode | Command | What runs | Speed | Fidelity |
|------|---------|-----------|-------|----------|
| Monolith | `make run` / `make run-watch` | Single Go process, all 14 services, hot reload | Fast | Lower |
| Lambda | `make deploy-local` | 4 Lambda functions + API Gateway in LocalStack | Slower | High (matches prod) |

### 3. S3 Client Endpoint Override

`s3client.New()` gains an `endpoint` parameter:
- When empty (production): unchanged behavior
- When set (local): static credentials, `BaseEndpoint` → LocalStack, `UsePathStyle: true`

Signature: `New(ctx context.Context, region string, endpoint string) (*S3Client, error)`

`asset_service.go`'s `s3URL()` becomes endpoint-aware — returns `http://localhost:4566/{bucket}/{key}` locally instead of `https://{bucket}.s3.amazonaws.com/{key}`.

### 4. Frontend Three-Mode Configuration

| Script | Env file | Target |
|--------|----------|--------|
| `npm run dev` | `.env.development` | AWS dev backend |
| `npm run dev:local` | `.env.local-backend` | Monolith on `:8080` |
| `npm run dev:lambda` | `.env.local-lambda` | LocalStack API Gateway |

### 5. Makefile Targets

| Target | Action |
|--------|--------|
| `make setup-local` (updated) | docker-compose up → init DynamoDB tables → init S3 buckets → seed data |
| `make run` (updated) | Monolith with `AWS_ENDPOINT=http://localhost:4566` |
| `make deploy-local` (new) | Build Lambda binaries → zip → deploy to LocalStack + create API Gateway |
| `make redeploy-local` (new) | Rebuild + update Lambda code (skip API Gateway recreation) |
| `make teardown-local` (new) | docker-compose down -v |

### 6. Lambda Deployment Script

`scripts/deploy-local-lambdas.sh`:
1. Builds 4 active Lambda binaries (ARM64 linux)
2. Zips each `bootstrap` binary
3. Creates IAM execution role in LocalStack
4. Creates Lambda functions with env vars (table names, JWT secret, S3 bucket, endpoint)
5. Creates REST API Gateway with route mappings for auth, user, catalog, asset
6. Outputs the invoke URL

### 7. Setup Scripts

| Script | Purpose |
|--------|---------|
| `scripts/init-local-db.sh` (updated) | Endpoint changes from `localhost:8000` → `localhost:4566` |
| `scripts/init-local-s3.sh` (new) | Creates `handloom-assets` and `handloom-uploads` buckets |
| `scripts/deploy-local-lambdas.sh` (new) | Full Lambda + API Gateway deployment |

## Files Modified

- `handloom-admin/docker-compose.yml`
- `handloom-admin/internal/s3client/client.go`
- `handloom-admin/internal/service/asset_service.go`
- `handloom-admin/cmd/api/main.go`
- `handloom-admin/Makefile`
- `handloom-admin/scripts/init-local-db.sh`
- `handloom-admin-frontend/package.json`
- `handloom-admin-frontend/vite.config.ts`

## Files Created

- `handloom-admin/scripts/init-local-s3.sh`
- `handloom-admin/scripts/deploy-local-lambdas.sh`
- `handloom-admin-frontend/.env.local-backend`
- `handloom-admin-frontend/.env.local-lambda`

## Not Changing

- Production/AWS dev deployment paths
- Config structure (`IsLocal()` logic)
- Auth flow, cookie handling, JWT
- Lambda entry point code (`cmd/lambda/*/main.go`)

## Developer Workflow

```bash
# One-time setup
cd handloom-admin && make setup-local

# Mode A: Fast dev (monolith + hot reload)
make run-watch
cd ../handloom-admin-frontend && npm run dev:local

# Mode B: Lambda integration testing
make deploy-local
cd ../handloom-admin-frontend && npm run dev:lambda

# After code changes in Lambda mode
make redeploy-local

# Clean up
make teardown-local
```
