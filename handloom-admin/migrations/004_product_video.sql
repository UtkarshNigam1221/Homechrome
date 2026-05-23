-- 004_product_video.sql
-- Add nullable video columns to products. Backwards compatible: existing rows
-- have NULL = no video.

ALTER TABLE products
  ADD COLUMN video_url        TEXT,
  ADD COLUMN video_poster_url TEXT;
