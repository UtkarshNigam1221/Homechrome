CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ============================================================================
-- CATEGORIES
-- ============================================================================
CREATE TABLE categories (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    description     TEXT NOT NULL DEFAULT '',
    image_url       TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'ACTIVE',
    product_count   INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      TEXT NOT NULL DEFAULT '',
    updated_by      TEXT NOT NULL DEFAULT ''
);

CREATE TABLE category_attributes (
    id              TEXT PRIMARY KEY,
    category_id     TEXT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    label           TEXT NOT NULL,
    type            TEXT NOT NULL,
    required        BOOLEAN NOT NULL DEFAULT FALSE,
    searchable      BOOLEAN NOT NULL DEFAULT FALSE,
    display_order   INT NOT NULL DEFAULT 0,
    UNIQUE(category_id, name)
);

CREATE TABLE category_attribute_options (
    id              TEXT PRIMARY KEY,
    attribute_id    TEXT NOT NULL REFERENCES category_attributes(id) ON DELETE CASCADE,
    value           TEXT NOT NULL,
    label           TEXT NOT NULL,
    sort_order      INT NOT NULL DEFAULT 0,
    UNIQUE(attribute_id, value)
);

-- ============================================================================
-- PRODUCTS
-- ============================================================================
CREATE TABLE products (
    id                      TEXT PRIMARY KEY,
    name                    TEXT NOT NULL,
    slug                    TEXT NOT NULL,
    sku                     TEXT NOT NULL UNIQUE,
    description             TEXT NOT NULL DEFAULT '',
    category_id             TEXT NOT NULL REFERENCES categories(id),
    artisan_id              TEXT,
    base_price              BIGINT NOT NULL,
    selling_price           BIGINT NOT NULL,
    cost_price              BIGINT NOT NULL DEFAULT 0,
    currency                TEXT NOT NULL DEFAULT 'INR',
    dim_length              NUMERIC,
    dim_width               NUMERIC,
    dim_height              NUMERIC,
    dim_unit                TEXT NOT NULL DEFAULT 'cm',
    weight                  INT NOT NULL DEFAULT 0,
    allow_custom_dimensions BOOLEAN NOT NULL DEFAULT FALSE,
    pricing_rule_id         TEXT,
    tags                    TEXT[] NOT NULL DEFAULT '{}',
    status                  TEXT NOT NULL DEFAULT 'DRAFT',
    sort_order              INT NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by              TEXT NOT NULL DEFAULT '',
    updated_by              TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_products_category_sort  ON products (category_id, sort_order, id);
CREATE INDEX idx_products_category_price ON products (category_id, selling_price);
CREATE INDEX idx_products_status         ON products (status);
CREATE INDEX idx_products_slug           ON products (slug);
CREATE INDEX idx_products_name_trgm      ON products USING GIN (name gin_trgm_ops);

CREATE TABLE product_attribute_values (
    product_id      TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    attribute_name  TEXT NOT NULL,
    attribute_value TEXT NOT NULL,
    PRIMARY KEY (product_id, attribute_name, attribute_value)
);

CREATE INDEX idx_pav_filter ON product_attribute_values (attribute_name, attribute_value);
CREATE INDEX idx_pav_category_filter ON product_attribute_values (attribute_name, attribute_value, product_id);

CREATE TABLE product_images (
    id          TEXT PRIMARY KEY,
    product_id  TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    url         TEXT NOT NULL,
    alt_text    TEXT NOT NULL DEFAULT '',
    sort_order  INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_product_images_product ON product_images (product_id, sort_order);

-- ============================================================================
-- INVENTORY
-- ============================================================================
CREATE TABLE inventory (
    id                  TEXT PRIMARY KEY,
    product_id          TEXT NOT NULL UNIQUE REFERENCES products(id) ON DELETE CASCADE,
    product_sku         TEXT NOT NULL,
    product_name        TEXT NOT NULL,
    quantity            INT NOT NULL DEFAULT 0,
    reserved_qty        INT NOT NULL DEFAULT 0,
    available_qty       INT NOT NULL DEFAULT 0,
    low_stock_threshold INT NOT NULL DEFAULT 0,
    reorder_point       INT NOT NULL DEFAULT 0,
    last_restock_at     TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          TEXT NOT NULL DEFAULT '',
    updated_by          TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_inventory_low_stock ON inventory (product_id)
    WHERE available_qty <= low_stock_threshold;

CREATE TABLE inventory_transactions (
    id              TEXT PRIMARY KEY,
    product_id      TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    type            TEXT NOT NULL,
    quantity        INT NOT NULL,
    previous_qty    INT NOT NULL,
    new_qty         INT NOT NULL,
    reason          TEXT NOT NULL DEFAULT '',
    reference_type  TEXT NOT NULL DEFAULT '',
    reference_id    TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_inv_txn_product_time ON inventory_transactions (product_id, created_at DESC);
