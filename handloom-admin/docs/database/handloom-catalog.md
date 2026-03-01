# Catalog Data (PostgreSQL)

Catalog data (categories, products, inventory) lives in **PostgreSQL** (RDS in production, Docker locally). This provides relational querying, ACID transactions with row-level locking, full-text search via tsvector with ts_rank ordering, and normalized attribute filtering.

Schema: `migrations/*.sql` (auto-applied by migrator Lambda on `cdk deploy`; locally via Docker init scripts). See [migrations.md](./migrations.md).
Repository: `internal/repository/postgres/`
Cache: `internal/cache/` (in-process TTL-based via go-cache)

---

## Configuration

```bash
# Local development (direct DSN)
POSTGRES_DSN=postgres://handloom:handloom@localhost:5432/handloom?sslmode=disable

# Production / Lambda (credentials from Secrets Manager)
RDS_SECRET_ARN=arn:aws:secretsmanager:ap-south-1:...   # JSON: {username, password}
RDS_ENDPOINT=handloom-db.xxxxx.ap-south-1.rds.amazonaws.com
RDS_PORT=5432
RDS_DATABASE=handloom
```

Connection pool: `pgxpool` (jackc/pgx v5). Local uses DSN directly; Lambda resolves credentials from Secrets Manager and builds `postgres://user:pass@endpoint:port/db?sslmode=require`.

---

## Schema

### Tables Overview

| Table | Purpose | Key Relations |
|-------|---------|---------------|
| `categories` | Product categories | Parent of `category_attributes`, referenced by `products` |
| `category_attributes` | Attribute definitions per category | FK → `categories`, parent of `category_attribute_options` |
| `category_attribute_options` | Allowed values per attribute | FK → `category_attributes` |
| `products` | Product records with pricing/dimensions | FK → `categories`, parent of images/attributes/inventory |
| `product_attribute_values` | Product attribute values (filtering index) | FK → `products` |
| `product_images` | Product image URLs with ordering | FK → `products` |
| `inventory` | Stock levels per product | FK → `products` (1:1) |
| `inventory_transactions` | Stock change audit trail | FK → `products` |

### 1. categories

```sql
CREATE TABLE categories (
    id              TEXT PRIMARY KEY,              -- UUID generated in Go
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,           -- URL-friendly, uniqueness enforced
    description     TEXT NOT NULL DEFAULT '',
    image_url       TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'ACTIVE', -- ACTIVE | INACTIVE
    product_count   INT NOT NULL DEFAULT 0,         -- Denormalized, updated via atomic increment
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      TEXT NOT NULL DEFAULT '',
    updated_by      TEXT NOT NULL DEFAULT ''
);
```

### 2. category_attributes

Defines the attribute schema for each category (e.g., Bedsheets have "material", "thread_count").

```sql
CREATE TABLE category_attributes (
    id              TEXT PRIMARY KEY,
    category_id     TEXT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,                  -- e.g., "material"
    label           TEXT NOT NULL,                  -- e.g., "Material"
    type            TEXT NOT NULL,                  -- e.g., "select", "text"
    required        BOOLEAN NOT NULL DEFAULT FALSE,
    searchable      BOOLEAN NOT NULL DEFAULT FALSE,
    display_order   INT NOT NULL DEFAULT 0,
    UNIQUE(category_id, name)
);
```

### 3. category_attribute_options

Allowed values for select-type attributes (e.g., material: ["Silk", "Cotton", "Linen"]).

```sql
CREATE TABLE category_attribute_options (
    id              TEXT PRIMARY KEY,
    attribute_id    TEXT NOT NULL REFERENCES category_attributes(id) ON DELETE CASCADE,
    value           TEXT NOT NULL,
    label           TEXT NOT NULL,
    sort_order      INT NOT NULL DEFAULT 0,
    UNIQUE(attribute_id, value)
);
```

### 4. products

```sql
CREATE TABLE products (
    id                      TEXT PRIMARY KEY,
    name                    TEXT NOT NULL,
    slug                    TEXT NOT NULL,
    sku                     TEXT NOT NULL UNIQUE,    -- Uniqueness via DB constraint (no guard items needed)
    description             TEXT NOT NULL DEFAULT '',
    category_id             TEXT NOT NULL REFERENCES categories(id),
    artisan_id              TEXT,                     -- Optional artisan reference
    base_price              BIGINT NOT NULL,          -- Paise (1 INR = 100 paise)
    selling_price           BIGINT NOT NULL,
    cost_price              BIGINT NOT NULL DEFAULT 0,
    currency                TEXT NOT NULL DEFAULT 'INR',
    dim_length              NUMERIC,                  -- Nullable dimensions (flattened, not nested)
    dim_width               NUMERIC,
    dim_height              NUMERIC,
    dim_unit                TEXT NOT NULL DEFAULT 'cm',
    weight                  INT NOT NULL DEFAULT 0,   -- Grams
    allow_custom_dimensions BOOLEAN NOT NULL DEFAULT FALSE,
    pricing_rule_id         TEXT,
    tags                    TEXT[] NOT NULL DEFAULT '{}',  -- PostgreSQL array
    status                  TEXT NOT NULL DEFAULT 'DRAFT', -- DRAFT | ACTIVE | INACTIVE
    sort_order              INT NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              TEXT NOT NULL DEFAULT '',
    updated_by              TEXT NOT NULL DEFAULT ''
);
```

**Indexes:**

| Index | Type | Columns | Purpose |
|-------|------|---------|---------|
| `idx_products_category_sort` | B-tree | `(category_id, sort_order, id)` | List products in category by display order |
| `idx_products_category_price` | B-tree | `(category_id, selling_price)` | Price-sorted category browsing |
| `idx_products_status` | B-tree | `(status)` | Filter by status |
| `idx_products_slug` | B-tree | `(slug)` | Lookup by URL slug |
| `idx_products_name_trgm` | GIN (trigram) | `name gin_trgm_ops` | ILIKE fallback for partial/substring matches |
| `idx_products_search_vector` | GIN | `search_vector` | Full-text search via tsvector (ts_rank ordering) |

### 5. product_attribute_values

Normalized attribute storage for filtering. Each product-attribute-value combination is a separate row.

```sql
CREATE TABLE product_attribute_values (
    product_id      TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    attribute_name  TEXT NOT NULL,     -- e.g., "material"
    attribute_value TEXT NOT NULL,     -- e.g., "Silk"
    PRIMARY KEY (product_id, attribute_name, attribute_value)
);
```

**Indexes:**

| Index | Columns | Purpose |
|-------|---------|---------|
| `idx_pav_filter` | `(attribute_name, attribute_value)` | Filter products by attribute |
| `idx_pav_category_filter` | `(attribute_name, attribute_value, product_id)` | Covering index for category-scoped attribute filters |

### 6. product_images

```sql
CREATE TABLE product_images (
    id          TEXT PRIMARY KEY,
    product_id  TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    url         TEXT NOT NULL,
    alt_text    TEXT NOT NULL DEFAULT '',
    sort_order  INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_product_images_product ON product_images (product_id, sort_order);
```

### 7. inventory

One row per product. Stock levels updated atomically with row-level locking.

```sql
CREATE TABLE inventory (
    id                  TEXT PRIMARY KEY,
    product_id          TEXT NOT NULL UNIQUE REFERENCES products(id) ON DELETE CASCADE,
    quantity            INT NOT NULL DEFAULT 0,
    reserved_qty        INT NOT NULL DEFAULT 0,
    available_qty       INT NOT NULL DEFAULT 0, -- = quantity - reserved_qty
    low_stock_threshold INT NOT NULL DEFAULT 0,
    reorder_point       INT NOT NULL DEFAULT 0,
    last_restock_at     TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          TEXT NOT NULL DEFAULT '',
    updated_by          TEXT NOT NULL DEFAULT ''
);

-- Partial index: only indexes products that are actually low-stock
CREATE INDEX idx_inventory_low_stock ON inventory (product_id)
    WHERE available_qty <= low_stock_threshold;
```

### 8. inventory_transactions

Immutable audit trail of all stock changes.

```sql
CREATE TABLE inventory_transactions (
    id              TEXT PRIMARY KEY,
    product_id      TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    type            TEXT NOT NULL,   -- ADD | REMOVE | RESERVE | RELEASE | ADJUST
    quantity        INT NOT NULL,    -- Delta (positive)
    previous_qty    INT NOT NULL,
    new_qty         INT NOT NULL,
    reason          TEXT NOT NULL DEFAULT '',
    reference_type  TEXT NOT NULL DEFAULT '',  -- e.g., "ORDER"
    reference_id    TEXT NOT NULL DEFAULT '',  -- e.g., order ID
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_inv_txn_product_time ON inventory_transactions (product_id, created_at DESC);
```

---

## Access Patterns

### Category Access Patterns

| Pattern | Query | Implementation |
|---------|-------|----------------|
| Get by ID | `SELECT ... WHERE id = $1` + LEFT JOIN attributes/options | `CategoryRepository.GetByID` |
| List all | `SELECT ... ORDER BY created_at DESC LIMIT $1+1` | `CategoryRepository.List` (cursor-paginated) |
| Filter by status | `WHERE status = $1` | Dynamic WHERE clause |
| Search by name | `WHERE name ILIKE '%' \|\| $1 \|\| '%'` | Case-insensitive substring match |
| Create | Transaction: INSERT category + batch INSERT attributes + batch INSERT options | `CategoryRepository.Create` |
| Update | Transaction: UPDATE category + DELETE/re-INSERT attributes and options | `CategoryRepository.Update` |
| Delete | `DELETE FROM categories WHERE id = $1` (CASCADE removes attributes) | `CategoryRepository.Delete` |
| Increment product count | `UPDATE categories SET product_count = product_count + $1` | `CategoryRepository.IncrementProductCount` |

### Product Access Patterns

| Pattern | Query | Implementation |
|---------|-------|----------------|
| Get by ID | `SELECT ... WHERE id = $1` + batch load attributes + images | `ProductRepository.GetByID` |
| Get by SKU | `SELECT ... WHERE sku = $1` | `ProductRepository.GetBySKU` |
| List (paginated) | Dynamic WHERE + ORDER BY sort_order, id + LIMIT | `ProductRepository.List` |
| Filter by category | `WHERE p.category_id = $1` | Dynamic WHERE clause |
| Filter by status | `WHERE p.status = $1` | Dynamic WHERE clause |
| Filter by price range | `WHERE p.selling_price BETWEEN $1 AND $2` | Dynamic WHERE clause |
| Filter by attributes | `EXISTS (SELECT 1 FROM product_attribute_values v WHERE v.product_id = p.id AND v.attribute_name = $N AND v.attribute_value = ANY($M))` | Dynamic EXISTS per attribute |
| Filter by stock level | `LEFT JOIN inventory i ... WHERE i.available_qty > 0` or `<= i.low_stock_threshold` | Optional JOIN |
| Full-text search | `WHERE (p.search_vector @@ websearch_to_tsquery('english', $1) OR p.name ILIKE $2)` ordered by `ts_rank()` | tsvector + ILIKE fallback |
| Get filter options | `SELECT DISTINCT attribute_value FROM product_attribute_values WHERE attribute_name = $1` scoped to category | `ProductRepository.GetAttributeFilterOptions` |
| Create | Transaction: INSERT product + batch INSERT attributes + batch INSERT images + INSERT inventory | `ProductRepository.Create` |
| Update | Transaction: UPDATE product + DELETE/re-INSERT attributes + DELETE/re-INSERT images | `ProductRepository.Update` |
| Delete | `DELETE FROM products WHERE id = $1` (CASCADE removes attributes, images, inventory) | `ProductRepository.Delete` |
| Batch get by IDs | `SELECT ... WHERE id = ANY($1)` | `ProductRepository.BatchGetByIDs` |
| Reorder products | Transaction: multiple `UPDATE products SET sort_order = $1 WHERE id = $2` | `ProductRepository.BatchUpdateSortOrder` |
| Get all in category | `SELECT ... WHERE category_id = $1 ORDER BY sort_order, id` (unpaginated) | `ProductRepository.GetByCategoryAll` |

### Inventory Access Patterns

| Pattern | Query | Implementation |
|---------|-------|----------------|
| Get by product | `SELECT ... WHERE product_id = $1` | `InventoryRepository.GetByProductID` |
| Add stock | `SELECT ... FOR UPDATE` → `UPDATE quantity, available_qty` → `INSERT transaction` | `InventoryRepository.AddStock` |
| Remove stock | `SELECT ... FOR UPDATE` → validate → `UPDATE quantity, available_qty` → `INSERT transaction` | `InventoryRepository.RemoveStock` |
| Reserve stock | `SELECT ... FOR UPDATE` → validate → `UPDATE reserved_qty, available_qty` → `INSERT transaction` | `InventoryRepository.ReserveStock` |
| Release stock | `SELECT ... FOR UPDATE` → validate → `UPDATE reserved_qty, available_qty` → `INSERT transaction` | `InventoryRepository.ReleaseStock` |
| Adjust stock | `SELECT ... FOR UPDATE` → `UPDATE quantity, available_qty` (absolute) → `INSERT transaction` | `InventoryRepository.AdjustStock` |
| List transactions | `SELECT ... WHERE product_id = $1 ORDER BY created_at DESC LIMIT $2+1` | `InventoryRepository.GetTransactions` |
| Low-stock products | `SELECT ... WHERE available_qty <= low_stock_threshold AND low_stock_threshold > 0 ORDER BY available_qty ASC` | `InventoryRepository.GetLowStockProducts` |

---

## Key Patterns

### Pagination

Cursor-based using base64-encoded integer offsets (`internal/repository/postgres/pagination.go`):

- Fetch `LIMIT + 1` rows to detect if more exist
- If `fetched > limit`: set `HasMore = true`, trim result, encode `offset + limit` as next cursor
- Empty cursor = offset 0
- Default limit: 20, max: 100

### Attribute Filtering

Products support dynamic attribute filtering via `EXISTS` subqueries:

```sql
-- For each attribute filter (e.g., material=["Silk","Cotton"], color=["Red"]):
SELECT p.* FROM products p
WHERE p.category_id = $1
  AND EXISTS (
    SELECT 1 FROM product_attribute_values v
    WHERE v.product_id = p.id AND v.attribute_name = 'material'
    AND v.attribute_value = ANY(ARRAY['Silk','Cotton'])
  )
  AND EXISTS (
    SELECT 1 FROM product_attribute_values v
    WHERE v.product_id = p.id AND v.attribute_name = 'color'
    AND v.attribute_value = ANY(ARRAY['Red'])
  )
ORDER BY p.sort_order, p.id
```

Hardcoded fields (material, color, weave_type) are stored in `product_attribute_values` alongside dynamic attributes, enabling uniform filtering.

### Inventory Row Locking

All inventory mutations use `SELECT ... FOR UPDATE` to prevent race conditions:

```sql
BEGIN;
  SELECT quantity, reserved_qty, available_qty FROM inventory WHERE product_id = $1 FOR UPDATE;
  -- Validate (e.g., available_qty >= requested quantity)
  UPDATE inventory SET quantity = $2, available_qty = $3, updated_at = NOW() WHERE product_id = $1;
  INSERT INTO inventory_transactions (...) VALUES (...);
COMMIT;
```

### Batch Relation Loading

Product list queries batch-load related data to avoid N+1:

1. Fetch core product rows
2. Collect all product IDs
3. Single query: `SELECT ... FROM product_attribute_values WHERE product_id = ANY($1)`
4. Single query: `SELECT ... FROM product_images WHERE product_id = ANY($1) ORDER BY sort_order`
5. Map results back to products by ID

### Transactional Writes

Create/update operations wrap multiple inserts in a single `pgx.BeginFunc` transaction:

- Product create: INSERT product + batch INSERT attribute values + batch INSERT images + INSERT inventory
- Product update: UPDATE product + DELETE old attributes + INSERT new attributes + DELETE old images + INSERT new images
- Category create: INSERT category + batch INSERT attributes + batch INSERT attribute options

---

## Caching Strategy

In-process TTL-based cache (`internal/cache/`, using `patrickmn/go-cache`):

| Entity | Cache Key | TTL | When Cached | Invalidated On |
|--------|-----------|-----|-------------|----------------|
| Category (by ID) | `cat:{id}` | 2 min | GetByID | Update, Delete |
| Category list | `cat:list` | 5 min | Unfiltered first-page List only | Create, Update, Delete |
| Product (by ID) | `prod:{id}` | 2 min | GetByID | Update, Delete |
| Attribute filter options | `prod:attr:{categoryID}` | 5 min | GetAttributeFilterOptions | Product Create, Update |

**Not cached:** Product list queries (too many filter combinations), inventory data (needs real-time accuracy).

Cache invalidation uses prefix-based deletion (e.g., `DeletePrefix("prod:cat:{catID}")` clears all cached products in a category).

---

## Local Development

### Docker Setup

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:16
    ports: ["5432:5432"]
    environment:
      POSTGRES_DB: handloom
      POSTGRES_USER: handloom
      POSTGRES_PASSWORD: handloom
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d   # Auto-runs SQL on first start
    healthcheck:
      test: pg_isready -U handloom

  pgadmin:
    image: dpage/pgadmin4:latest
    ports: ["5050:80"]
    environment:
      PGADMIN_DEFAULT_EMAIL: admin@homechrome.dev
      PGADMIN_DEFAULT_PASSWORD: admin
```

Schema auto-created on first `docker-compose up` via `migrations/001_catalog_schema.sql` mounted into `/docker-entrypoint-initdb.d/`.

### Seed Data

`scripts/seed-data.sh` seeds PostgreSQL with sample categories, attributes, and products using `psql`:

```bash
psql "$POSTGRES_DSN" <<'EOSQL'
INSERT INTO categories (...) VALUES (...) ON CONFLICT (id) DO NOTHING;
INSERT INTO category_attributes (...) VALUES (...) ON CONFLICT (id) DO NOTHING;
INSERT INTO category_attribute_options (...) VALUES (...) ON CONFLICT (id) DO NOTHING;
EOSQL
```

### Inspecting Data

```bash
# pgAdmin UI
open http://localhost:5050
# Register server: host=postgres, port=5432, user/password=handloom

# CLI
docker exec -it handloom-postgres psql -U handloom -d handloom
\dt                    # List tables
SELECT * FROM categories;
SELECT * FROM products WHERE category_id = 'cat-001';
SELECT * FROM inventory WHERE available_qty <= low_stock_threshold;
\q
```

---

## Dependency Injection (Wire)

```go
// internal/wire/providers.go
func ProvidePostgresPool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error)
func ProvideCatalogCache() *cache.Cache
func ProvideCategoryRepository(pool *pgxpool.Pool, c *cache.Cache) domain.CategoryRepository
func ProvideProductRepository(pool *pgxpool.Pool, c *cache.Cache) domain.ProductRepository
func ProvideInventoryRepository(pool *pgxpool.Pool) domain.InventoryRepository
```

Wire injectors that use PostgreSQL:
- `InitializeApiDeps` — monolith (all repos)
- `InitializeCatalogDeps` — Catalog Lambda
- `InitializeInventoryDeps` — Inventory Lambda
- `InitializeOrderDeps` — Order Lambda (needs product/inventory repos for checkout)
