-- 008_metrics_retention_cron.sql
-- Daily retention sweep for metric_counters. Runs inside DB via pg_cron.
-- Business metrics: 90 days. Service metrics: 14 days.
--
-- NEON PREREQUISITE: pg_cron must be enabled via Neon Console first
-- (Console → Settings → Extensions → Enable pg_cron). Neon enforces that
-- CREATE EXTENSION pg_cron runs in the `postgres` system database, which
-- neondb_owner cannot access — so this migration cannot install the
-- extension itself. It only registers the retention job IF the extension
-- is already installed in the current database.
--
-- Until you enable pg_cron:
--   - This migration silently no-ops on every deploy. Retention runs are
--     not scheduled. Manual table cleanup OR re-run this migration after
--     enabling pg_cron.

DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_cron') THEN
    -- pg_cron already installed in this database → register the job.
    PERFORM cron.unschedule(jobid)
    FROM cron.job
    WHERE jobname = 'metric-counters-retention';

    PERFORM cron.schedule(
      'metric-counters-retention',
      '0 3 * * *',
      $sql$
        DELETE FROM metric_counters
        WHERE (retention_class = 'business' AND bucket_start < now() - interval '90 days')
           OR (retention_class = 'service'  AND bucket_start < now() - interval '14 days');
      $sql$
    );
  ELSE
    RAISE NOTICE 'pg_cron extension not installed in current database — retention schedule skipped. '
                 'Enable pg_cron via Neon Console, then re-deploy or run this migration manually.';
  END IF;
END$$;
