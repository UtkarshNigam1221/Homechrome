-- 010_world_geo.sql
--
-- Migrate geo from India-only (state-coded) to worldwide (country-coded).
-- city_centroids: drop + recreate with (city, country) composite PK and
-- columns auto-populated from CloudFront viewer headers on first sighting.
-- No initial seed — backend lazy-upserts as traffic arrives.
--
-- metric_counters: clean-slate delete of historical rows whose labels carry
-- the legacy "state" key. Retention policy would drop these in 90d anyway,
-- but dashboard rewrite is incompatible with the old shape so we cut now.
--
-- ─── REQUIRED MANUAL STEP after migrator runs ───
-- The recreated city_centroids loses its RLS grants. Re-run these once via
-- Neon SQL Editor (mirrors 009_metrics_rls.sql):
--
--   GRANT SELECT ON city_centroids TO authenticated;
--   ALTER TABLE city_centroids ENABLE ROW LEVEL SECURITY;
--   DROP POLICY IF EXISTS authenticated_select_centroids ON city_centroids;
--   CREATE POLICY authenticated_select_centroids
--     ON city_centroids FOR SELECT TO authenticated USING (true);
--
-- ─────────────────────────────────────────────────

DROP TABLE IF EXISTS city_centroids;

CREATE TABLE city_centroids (
  city          TEXT NOT NULL,
  country       TEXT NOT NULL,
  lat           DOUBLE PRECISION NOT NULL,
  lng           DOUBLE PRECISION NOT NULL,
  first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (city, country)
);

CREATE INDEX city_centroids_country_idx ON city_centroids (country);

-- Clean slate: delete metric rows whose labels reference the legacy `state`
-- key. Geography dashboard reads break otherwise. New emits use `country`.
DELETE FROM metric_counters WHERE labels ? 'state';
