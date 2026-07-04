-- ============================================================================
-- PRODUCT SLUG UNIQUENESS
-- ============================================================================
-- Product slugs were only indexed, not unique, and generateSlug() didn't dedupe.
-- So multiple products could share a slug (e.g. same name), and the storefront's
-- slug->product lookup (LIMIT 1) resolved every one of them to a single product.
-- Slug generation now appends -2/-3/... on collision; this migration backfills
-- existing duplicates and enforces uniqueness so the bug can't recur.

-- Backfill: keep the bare slug for the product that the storefront currently
-- serves for that slug, and suffix the rest by their ordinal.
-- findProductBySlug resolves a shared slug via ORDER BY sort_order, id LIMIT 1,
-- so partition in the SAME order — the currently-served (and SEO-indexed)
-- product keeps its URL; the others become ...-2, ...-3, ...
-- Note: if a suffixed value collides with a genuine existing slug the UNIQUE
-- constraint below aborts the migration (fails safe) — resolve that name by hand.
WITH ranked AS (
    SELECT id,
           row_number() OVER (PARTITION BY slug ORDER BY sort_order, id) AS rn
    FROM products
)
UPDATE products p
SET slug = p.slug || '-' || ranked.rn
FROM ranked
WHERE p.id = ranked.id AND ranked.rn > 1;

-- The plain index is superseded by the unique index the constraint creates.
DROP INDEX IF EXISTS idx_products_slug;

-- Guarded so the file is re-runnable by hand (the migrator also tracks it).
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'products_slug_key'
    ) THEN
        ALTER TABLE products ADD CONSTRAINT products_slug_key UNIQUE (slug);
    END IF;
END $$;
