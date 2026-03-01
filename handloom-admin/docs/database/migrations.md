# PostgreSQL Schema Migrations

Automated schema migration system for the PostgreSQL catalog database. Migrations run automatically during `cdk deploy` via a dedicated Lambda triggered by CDK's `Trigger` construct.

---

## How It Works

```
cdk deploy
  │
  ├─ 1. RDS instance created/updated
  │
  ├─ 2. CDK Trigger invokes migrator Lambda (after RDS is ready)
  │     │
  │     ├─ Connect to RDS (credentials from Secrets Manager)
  │     ├─ Acquire pg_advisory_lock (prevents concurrent runs)
  │     ├─ Create schema_migrations table (if not exists)
  │     ├─ Read embedded SQL files, skip already-applied
  │     ├─ Execute each unapplied migration in a transaction
  │     └─ Record filename + timestamp in schema_migrations
  │
  └─ 3. API Lambdas deployed (schema is ready)
```

If a migration fails, the CDK deployment rolls back — API Lambdas are not updated with code that expects a schema that doesn't exist.

---

## Migration Files

All migration SQL files live in `migrations/`:

| File | Purpose |
|------|---------|
| `001_catalog_schema.sql` | Initial catalog schema (categories, products, inventory, images, attributes) |
| `002_normalize_inventory.sql` | Inventory normalization |
| `003_product_search_vector.sql` | Full-text search vector + GIN index |

Files are embedded into the migrator Lambda binary via `go:embed` (`migrations/embed.go`).

### Naming Convention

```
NNN_description.sql
```

- **NNN**: Zero-padded sequence number (001, 002, ...). Migrations run in filename sort order.
- **description**: Snake_case description of the change.

### Writing a New Migration

1. Create a new file in `migrations/`:
   ```bash
   touch migrations/004_add_product_tags.sql
   ```

2. Write idempotent-safe SQL. Each file runs in a single transaction.

3. Build and deploy:
   ```bash
   make cdk-deploy-dev
   ```
   The migrator Lambda automatically picks up the new file (embedded at build time) and applies it.

---

## Tracking Table

Applied migrations are tracked in `schema_migrations`:

```sql
CREATE TABLE schema_migrations (
    filename   TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

To check which migrations have been applied:
```sql
SELECT * FROM schema_migrations ORDER BY filename;
```

---

## Architecture

### Components

| Component | Path | Purpose |
|-----------|------|---------|
| SQL files | `migrations/*.sql` | Raw migration SQL |
| Embed | `migrations/embed.go` | `go:embed *.sql` exposes `migrations.FS` |
| Lambda | `cmd/lambda/migrator/main.go` | Connects to RDS, runs unapplied migrations |
| CDK Trigger | `infra/stacks/database.go` | Invokes migrator after RDS is ready |

### Concurrency Safety

The migrator acquires a PostgreSQL advisory lock (`pg_advisory_lock(7248301945)`) before running. If two migrator invocations overlap (e.g., concurrent deploys), the second blocks until the first completes, preventing duplicate runs.

### CDK Trigger Behavior

- **`ExecuteAfter: catalogDB`** — waits for the RDS instance to be available
- **`ExecuteOnHandlerChange: true`** — re-invokes when the Lambda code changes (new migration files produce a new binary hash, triggering a re-run)

---

## Lambda Configuration

```bash
# Environment variables (set by CDK)
RDS_SECRET_ARN   # Secrets Manager ARN for DB credentials
RDS_ENDPOINT     # RDS instance endpoint
RDS_PORT=5432
RDS_DATABASE=handloom
APP_ENV          # dev | prod
```

- Runtime: `provided.al2023` (ARM64)
- Memory: 128 MB
- Timeout: 60 seconds
- IAM: Read access to `CatalogDBSecret`

---

## Commands

```bash
# Build migrator Lambda (included in active builds)
make build-lambdas-active

# Deploy (migrator runs automatically)
make cdk-deploy-dev

# Check migrator logs
aws logs tail /aws/lambda/handloom-migrator-dev --follow
```

---

## Local Development

Locally, migrations are applied automatically by Docker Compose — the `postgres` service mounts `migrations/001_catalog_schema.sql` via `/docker-entrypoint-initdb.d/`. Subsequent migrations (`002_*`, `003_*`) are applied by the Go monolith at startup or manually:

```bash
# Apply all migrations via psql
docker exec handloom-postgres psql -U handloom -d handloom \
  -f /docker-entrypoint-initdb.d/001_catalog_schema.sql \
  -f /path/to/002_normalize_inventory.sql \
  -f /path/to/003_product_search_vector.sql
```

---

## Troubleshooting

### Migration failed during deploy

CDK rolls back the entire deployment. Check the CloudWatch logs:
```bash
aws logs tail /aws/lambda/handloom-migrator-dev --since 1h
```

Fix the SQL, rebuild, and redeploy.

### Need to see applied migrations

```bash
# Via psql or any PostgreSQL client
SELECT filename, applied_at FROM schema_migrations ORDER BY filename;
```

### Stuck advisory lock

If the migrator Lambda times out mid-migration, the advisory lock releases automatically when the connection closes. No manual intervention needed — advisory locks are session-scoped.
