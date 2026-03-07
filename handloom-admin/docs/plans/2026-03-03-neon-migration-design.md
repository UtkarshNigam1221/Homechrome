# Neon Postgres Migration Design

**Date:** 2026-03-03
**Status:** Approved
**Goal:** Replace RDS PostgreSQL with Neon Postgres to eliminate ~$21/month RDS costs.

## Context

Current setup: RDS db.t4g.micro (PostgreSQL 16) in ap-south-1 with a dedicated VPC, Secrets Manager secret, and migrator Lambda. All 18 Lambdas (13 API + 4 workers + 1 migrator) fetch credentials from Secrets Manager at cold start. No Lambdas are placed inside the VPC — the VPC exists solely for RDS.

Business context: early-stage, 1K-10K products, 50-500 visits/day. Full-text search and attribute filtering are essential.

## Decision

Migrate to **Neon Postgres** (free tier: 0.5 GB storage, 100 compute-hours/month). Region: **ap-southeast-1 (Singapore)** — closest Neon AWS region to ap-south-1 (Mumbai) Lambdas.

## What Does NOT Change

- All repository code (`internal/repository/postgres/`)
- All service code (`internal/service/`)
- All handler code (`internal/handler/`)
- SQL queries, full-text search, attribute filtering
- Migration SQL files (`migrations/*.sql`)
- pgxpool connection pool (pgx/v5)
- `sslmode=require` (Neon requires SSL)

## What Changes

### 1. CDK Database Stack (`infra/stacks/database.go`)

**Remove:**
- VPC (`handloom-catalog-vpc-{env}`) + public subnets
- Security group (`handloom-catalog-db-{env}`)
- RDS instance (`handloom-catalog-{env}`)
- Secrets Manager secret (`handloom/{env}/catalog-db`)

**Keep (modified):**
- Migrator Lambda — change env vars from `RDS_SECRET_ARN`/`RDS_ENDPOINT`/`RDS_PORT`/`RDS_DATABASE` to single `POSTGRES_DSN`

**Exported fields change:**
- Remove: `CatalogDB`, `CatalogDBSecret`, `CatalogVpc`
- Add: `PostgresDSN` (SSM parameter or CDK context value)

### 2. CDK API Stack (`infra/stacks/api.go`)

**Replace** on all 13 service Lambdas:
```
RDS_SECRET_ARN → remove
RDS_ENDPOINT   → remove
RDS_PORT       → remove
RDS_DATABASE   → remove
```
**With:** `POSTGRES_DSN` env var (Neon pooled connection string)

**Remove:** `CatalogDBSecret.GrantRead()` calls on all 13 Lambdas.

### 3. CDK Event Stack (`infra/stacks/event.go`)

Same changes as API stack for all 4 worker Lambdas.

### 4. CDK Entry Point (`infra/cmd/main.go`)

Remove `DatabaseStack` dependency from `APIStackProps` and `EventStackProps` (no more `CatalogDB`/`CatalogDBSecret` references). The DSN comes from CDK context or SSM.

### 5. Application Config (`internal/config/config.go`)

`PostgresConfig` simplifies — `DSN` field is now used in all environments (local + Lambda). `SecretARN`/`Endpoint`/`Port`/`DatabaseName` fields become unused.

### 6. Postgres Client (`internal/repository/postgres/client.go`)

Remove `resolveDSNFromSecret()` and Secrets Manager import. `NewPool()` simply uses `pgCfg.DSN` directly.

### 7. Migrator Lambda (`cmd/lambda/migrator/main.go`)

No logic change — `config.Load()` → `postgres.NewPool()` still works, but now `DSN` is populated from `POSTGRES_DSN` env var instead of resolving from Secrets Manager.

## Connection Architecture

- Use Neon **pooled connection string** (built-in pgBouncer on port 5432 with `-pooler` suffix in hostname)
- DSN format: `postgres://user:pass@ep-xxx-pooler.ap-southeast-1.aws.neon.tech:5432/neondb?sslmode=require`
- Store as `POSTGRES_DSN` env var on all Lambdas
- No Secrets Manager, no VPC, no security groups

## Migration Steps

1. Create Neon project (Singapore region, Postgres 16)
2. Run `migrations/*.sql` files against Neon manually (`psql -f`)
3. Update CDK with Neon DSN
4. Deploy — migrator finds all migrations applied, no-ops
5. Verify storefront
6. Remove RDS resources from CDK in follow-up deploy

## Rollback

Keep RDS running 48h after cutover. Revert CDK to RDS config if needed (one deploy).
