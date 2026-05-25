-- Enable pgvector extension (idempotent).
CREATE EXTENSION IF NOT EXISTS vector;

-- Add embedding column + last-write timestamp. NULL values mean
-- the row has not been embedded yet (backfill will populate).
ALTER TABLE products
  ADD COLUMN embedding vector(768),
  ADD COLUMN embedding_updated_at timestamptz;

-- HNSW index on cosine distance for fast top-K retrieval.
-- m=16, ef_construction=64 are pgvector defaults — sufficient at <10k rows.
-- ef_search is set per-connection in the Go service.
CREATE INDEX idx_products_embedding_hnsw
  ON products
  USING hnsw (embedding vector_cosine_ops)
  WITH (m = 16, ef_construction = 64);
