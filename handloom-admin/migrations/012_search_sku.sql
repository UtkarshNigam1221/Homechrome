-- 012_search_sku.sql
-- Search must match SKUs (design codes): rebuild the generated search_vector
-- to include sku (weight A, 'simple' config so codes aren't stemmed), and add
-- a trigram index on sku for partial-code matches (e.g. "2221").
--
-- Generated columns can't be altered in place: drop + re-add (the dependent
-- GIN index drops with the column).

ALTER TABLE products DROP COLUMN search_vector;

ALTER TABLE products
  ADD COLUMN search_vector tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
    setweight(to_tsvector('simple',  coalesce(sku,  '')), 'A') ||
    setweight(to_tsvector('english', coalesce(description, '')), 'B')
  ) STORED;

CREATE INDEX idx_products_search_vector ON products USING GIN (search_vector);
CREATE INDEX idx_products_sku_trgm ON products USING GIN (sku gin_trgm_ops);
