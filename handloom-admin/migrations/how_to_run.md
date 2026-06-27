# Get the RDS endpoint from CDK output
RDS_ENDPOINT=$(aws cloudformation describe-stacks \
--stack-name HandloomDatabaseStack-dev \
--query "Stacks[0].Outputs[?OutputKey=='CatalogDBEndpoint'].OutputValue" \
--output text)

# Get the password from Secrets Manager
SECRET_ARN=$(aws cloudformation describe-stacks \
--stack-name HandloomDatabaseStack-dev \
--query "Stacks[0].Outputs[?OutputKey=='CatalogDBSecretARN'].OutputValue" \
--output text)
PASSWORD=$(aws secretsmanager get-secret-value \
--secret-id "$SECRET_ARN" \
--query "SecretString" --output text | jq -r '.password')

# Run migration
PGPASSWORD="$PASSWORD" psql -h "$RDS_ENDPOINT" -U handloom -d handloom \
-f migrations/001_catalog_schema.sql

This is a manual step after first deploy — not automated in CDK.

## Migration 008 prerequisite — enable pg_cron from Neon Console

Migration `008_metrics_retention_cron.sql` installs the `pg_cron` extension and schedules a daily 03:00 UTC retention sweep of `metric_counters`. Neon requires pg_cron to be enabled at the project level via the Console UI **before** the migration is applied; `neondb_owner` cannot install it via SQL because the `postgres` database (where pg_cron must live) is owned by Neon's internal `cloud_admin`.

**Steps (one-time per Neon project/environment):**

1. Open the Neon Console → select the project (e.g. "Catalog" for dev, "Catalog-Prod" for prod).
2. Go to **Settings → Extensions** (or **Extensions** tab on the branch detail page).
3. Enable **pg_cron**. Neon installs it in the `postgres` database and sets `cron.database_name` to allow cross-database job scheduling.
4. Then run the migration normally (migrator Lambda on `cdk-deploy`, or `psql` manually):
   ```bash
   NEON_DSN=<your-dev-dsn>
   psql "$NEON_DSN" -f migrations/008_metrics_retention_cron.sql
   ```
5. Verify the schedule was registered:
   ```sql
   SELECT jobid, jobname, schedule, command
   FROM cron.job
   WHERE jobname = 'metric-counters-retention';
   ```
   Expected: 1 row, schedule `0 3 * * *`.

**Why can't the migration self-install pg_cron?**
`CREATE EXTENSION pg_cron` must run in the `postgres` database (pg_cron's background worker reads job metadata from there). Neon's `postgres` DB is owned by `cloud_admin`; `neondb_owner` (and even `neon_superuser`) lack `CREATE` privilege on it. The Neon Console path is the only supported enablement mechanism.

**Local Docker / pgvector:**
Local dev uses a pgvector image that does not ship pg_cron. Migration 008 will fail on `make reset-db` with `ERROR: extension "pg_cron" not available`. This is expected and acceptable — migration 008 is Neon-only. If it blocks local resets, comment out migration 008 locally and re-apply only against Neon.
