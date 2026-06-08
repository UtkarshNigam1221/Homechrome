-- 010_world_geo.sql
--
-- Country migration cleanup. city_centroids is already created in its final
-- (city, country) shape by 007, so there is nothing to restructure here — this
-- migration only purges legacy metric rows.
--
-- metric_counters: clean-slate delete of historical rows whose labels carry the
-- legacy "state" key (the geo pipeline is now country/city only). Retention
-- would drop these in 90d anyway, but the dashboard rewrite is incompatible
-- with the old shape, so we cut now. Safe no-op on a fresh database.

DELETE FROM metric_counters WHERE labels ? 'state';
