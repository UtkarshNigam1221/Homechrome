-- 006_metric_counters.sql
-- Central PG table for all metric counters (business + service tiers).
-- Updated by worker-metrics-consumer Lambda via batch UPSERT.

CREATE TABLE IF NOT EXISTS metric_counters (
  metric           TEXT NOT NULL,
  labels           JSONB NOT NULL,
  label_hash       BYTEA NOT NULL,
  bucket_start     TIMESTAMPTZ NOT NULL,
  count            BIGINT NOT NULL DEFAULT 0 CHECK (count >= 0),
  sum_value        BIGINT NOT NULL DEFAULT 0 CHECK (sum_value >= 0),
  retention_class  TEXT NOT NULL CHECK (retention_class IN ('business', 'service')),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (metric, label_hash, bucket_start)
);

CREATE INDEX IF NOT EXISTS idx_metric_counters_metric_time
  ON metric_counters (metric, bucket_start DESC);

CREATE INDEX IF NOT EXISTS idx_metric_counters_retention
  ON metric_counters (retention_class, bucket_start);

CREATE INDEX IF NOT EXISTS idx_metric_counters_labels
  ON metric_counters USING GIN (labels jsonb_path_ops);

COMMENT ON TABLE metric_counters IS
  'Bucketed counter values for product + service analytics. 5-min buckets. Updated by worker-metrics-consumer Lambda.';

COMMENT ON COLUMN metric_counters.label_hash IS
  'sha256 of canonical-ordered label JSON. Required for stable PK uniqueness since JSONB ordering varies.';
