-- 007_city_centroids.sql
--
-- city_centroids stores a (city, country) -> (lat, lng) marker, learned from
-- CloudFront viewer headers on first sighting. No seed: the backend
-- lazy-upserts (internal/repository/postgres/centroids_repo.go) as traffic
-- arrives. Schema matches the repo's INSERT exactly, so centroid writes succeed
-- as soon as this migration has run — independent of any later migration.
--
-- Reads happen from the admin frontend via the Neon Data API. RLS grants for
-- the `authenticated` role are applied as a manual step (see 009_metrics_rls.sql).

CREATE TABLE IF NOT EXISTS city_centroids (
  city          TEXT NOT NULL,
  country       TEXT NOT NULL,
  lat           DOUBLE PRECISION NOT NULL,
  lng           DOUBLE PRECISION NOT NULL,
  first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (city, country)
);

CREATE INDEX IF NOT EXISTS city_centroids_country_idx ON city_centroids (country);
