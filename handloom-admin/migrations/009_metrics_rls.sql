-- 009_metrics_rls.sql
-- DEFERRED: RLS setup for Neon Data API.
--
-- Neon's `authenticated` role is managed by the Data API control plane,
-- not by neondb_owner. The migrator Lambda (running as neondb_owner)
-- cannot GRANT to it or enable RLS on tables that Data API surfaces.
--
-- Apply these commands ONCE via the Neon SQL Editor (Console → SQL Editor)
-- after Data API is enabled. They are idempotent — safe to re-run.
--
-- ─── COPY-PASTE INTO NEON SQL EDITOR ───
--
--   GRANT USAGE  ON SCHEMA public TO authenticated;
--   GRANT SELECT ON metric_counters TO authenticated;
--   GRANT SELECT ON city_centroids  TO authenticated;
--
--   ALTER TABLE metric_counters ENABLE ROW LEVEL SECURITY;
--   ALTER TABLE city_centroids  ENABLE ROW LEVEL SECURITY;
--
--   DROP POLICY IF EXISTS authenticated_select_counters ON metric_counters;
--   CREATE POLICY authenticated_select_counters
--     ON metric_counters FOR SELECT TO authenticated USING (true);
--
--   DROP POLICY IF EXISTS authenticated_select_centroids ON city_centroids;
--   CREATE POLICY authenticated_select_centroids
--     ON city_centroids FOR SELECT TO authenticated USING (true);
--
-- ────────────────────────────────────────
--
-- This file itself is a no-op so the migrator Lambda doesn't fail. The
-- migration row is recorded in schema_migrations so it won't be retried.

DO $$
BEGIN
  RAISE NOTICE 'Migration 009 is a no-op. Apply RLS setup via Neon SQL Editor manually. See header comment.';
END$$;
