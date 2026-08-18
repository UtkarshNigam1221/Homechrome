package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgvector "github.com/pgvector/pgvector-go"

	"github.com/handloom/admin/internal/domain"
	"github.com/handloom/admin/internal/repository/postgres/querybuilder"
	"github.com/handloom/admin/pkg/errors"
)

// hardcodedAttrFields maps Product struct field names to the attribute_name
// used in the product_attribute_values table.
var hardcodedAttrFields = []struct {
	Name  string
	Field func(*domain.Product) string
	Set   func(*domain.Product, string)
}{
	{"material", func(p *domain.Product) string { return p.Material }, func(p *domain.Product, v string) { p.Material = v }},
	// color is deliberately NOT here: products can carry multiple colors, so it
	// flows through the Attributes map (string or []string) like any EAV attribute.
	{"weave_type", func(p *domain.Product) string { return p.WeaveType }, func(p *domain.Product, v string) { p.WeaveType = v }},
	{"origin", func(p *domain.Product) string { return p.Origin }, func(p *domain.Product, v string) { p.Origin = v }},
	{"craft_type", func(p *domain.Product) string { return p.CraftType }, func(p *domain.Product, v string) { p.CraftType = v }},
}

// ProductRepository implements product data access on PostgreSQL.
type ProductRepository struct {
	pool *pgxpool.Pool
}

// NewProductRepository creates a new ProductRepository.
func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{pool: pool}
}

// productRow is an intermediate scan target that handles the flattened
// dimension columns (dim_length, dim_width, dim_height, dim_unit) which
// don't map directly to the domain.Product.Dimensions struct, the
// nullable video columns which are stored as NULL when empty, and the
// pgvector embedding column which requires a pgvector.Vector scan target.
type productRow struct {
	domain.Product
	DimLength      *float64 `db:"dim_length"`
	DimWidth       *float64 `db:"dim_width"`
	DimHeight      *float64 `db:"dim_height"`
	DimUnit        string   `db:"dim_unit"`
	VideoURL       *string  `db:"video_url"`
	VideoPosterURL *string  `db:"video_poster_url"`
	// Pointer, not a value: a NULL embedding cannot be represented by
	// pgvector.Vector, and DecodeBinary panics on the zero-length buffer.
	EmbeddingVec       *pgvector.Vector `db:"embedding"`
	EmbeddingUpdatedAt *time.Time       `db:"embedding_updated_at"`
}

// toProduct converts a productRow into a *domain.Product, reconstructing
// the Dimensions struct from the flat columns, coercing NULL video columns
// back to empty strings, and populating Embedding + EmbeddingUpdatedAt from
// their pgvector scan targets.
func (r *productRow) toProduct() *domain.Product {
	p := r.Product
	if r.DimLength != nil || r.DimWidth != nil || r.DimHeight != nil {
		d := &domain.Dimensions{Unit: r.DimUnit}
		if r.DimLength != nil {
			d.Length = *r.DimLength
		}
		if r.DimWidth != nil {
			d.Width = *r.DimWidth
		}
		if r.DimHeight != nil {
			d.Height = *r.DimHeight
		}
		p.Dimensions = d
	}
	if r.VideoURL != nil {
		p.VideoURL = *r.VideoURL
	}
	if r.VideoPosterURL != nil {
		p.VideoPosterURL = *r.VideoPosterURL
	}
	if r.EmbeddingVec != nil {
		if slice := r.EmbeddingVec.Slice(); len(slice) > 0 {
			p.Embedding = slice
		}
	}
	if r.EmbeddingUpdatedAt != nil {
		p.EmbeddingUpdatedAt = r.EmbeddingUpdatedAt
	}
	return &p
}

// scanProductRows converts a slice of productRow into []*domain.Product.
func scanProductRows(rows []productRow) []*domain.Product {
	products := make([]*domain.Product, len(rows))
	for i := range rows {
		products[i] = rows[i].toProduct()
	}
	return products
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

// Create inserts a product and its related rows (attribute values, images)
// plus the corresponding inventory record, all inside a single transaction.
func (r *ProductRepository) Create(ctx context.Context, product *domain.Product, inventory *domain.Inventory) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return errors.Internal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := r.createProductInTx(ctx, tx, product, inventory); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return errors.Internal(err)
	}
	return nil
}

// createProductInTx executes the full product+inventory insert on an existing
// transaction. It mutates product.CreatedAt/UpdatedAt and inventory.ID/timestamps.
func (r *ProductRepository) createProductInTx(ctx context.Context, tx pgx.Tx, product *domain.Product, inventory *domain.Inventory) error {
	// --- product ---
	var dimLength, dimWidth, dimHeight *float64
	dimUnit := "cm"
	if product.Dimensions != nil {
		dimLength = &product.Dimensions.Length
		dimWidth = &product.Dimensions.Width
		dimHeight = &product.Dimensions.Height
		dimUnit = product.Dimensions.Unit
	}

	now := time.Now()
	product.CreatedAt = now
	product.UpdatedAt = now

	qb := querybuilder.Insert("products").
		Set(ColID, product.ID).
		Set(ColName, product.Name).
		Set(ColSlug, product.Slug).
		Set(ColSKU, product.SKU).
		Set(ColDescription, product.Description).
		Set(ColCategoryID, product.CategoryID).
		Set(ColBasePrice, product.BasePrice).
		Set(ColSellingPrice, product.SellingPrice).
		Set(ColCostPrice, product.CostPrice).
		Set(ColCurrency, product.Currency).
		Set(ColDimLength, dimLength).
		Set(ColDimWidth, dimWidth).
		Set(ColDimHeight, dimHeight).
		Set(ColDimUnit, dimUnit).
		Set(ColWeight, product.Weight).
		Set(ColAllowCustomDimensions, product.AllowCustomDimensions).
		Set(ColPricingRuleID, product.PricingRuleID).
		Set(ColTags, product.Tags).
		Set(ColStatus, string(product.Status)).
		Set(ColSortOrder, product.SortOrder).
		Set(ColVideoURL, nullableString(product.VideoURL)).
		Set(ColVideoPosterURL, nullableString(product.VideoPosterURL)).
		Set(ColCreatedAt, product.CreatedAt).
		Set(ColUpdatedAt, product.UpdatedAt).
		Set(ColCreatedBy, product.CreatedBy).
		Set(ColUpdatedBy, product.UpdatedBy)

	query, args := qb.Build()
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			if strings.Contains(err.Error(), "products_sku_key") {
				return errors.New(errors.ErrCodeProductSKUExists, "Product with this SKU already exists")
			}
			return errors.Conflict("Product already exists")
		}
		return errors.Internal(err)
	}

	// --- attribute values ---
	if attrErr := insertAttributeValues(ctx, tx, product); attrErr != nil {
		return attrErr
	}

	// --- images ---
	if imgErr := insertImages(ctx, tx, product); imgErr != nil {
		return imgErr
	}

	// --- inventory ---
	if inventory != nil {
		if inventory.ID == "" {
			inventory.ID = uuid.New().String()
		}
		inventory.CreatedAt = now
		inventory.UpdatedAt = now

		invQB := querybuilder.Insert("inventory").
			Set(ColID, inventory.ID).
			Set(ColProductID, inventory.ProductID).
			Set(ColQuantity, inventory.Quantity).
			Set(ColReservedQty, inventory.ReservedQty).
			Set(ColAvailableQty, inventory.AvailableQty).
			Set(ColLowStockThreshold, inventory.LowStockThreshold).
			Set(ColReorderPoint, inventory.ReorderPoint).
			Set(ColLastRestockAt, inventory.LastRestockAt).
			Set(ColCreatedAt, inventory.CreatedAt).
			Set(ColUpdatedAt, inventory.UpdatedAt).
			Set(ColCreatedBy, inventory.CreatedBy).
			Set(ColUpdatedBy, inventory.UpdatedBy)

		invSQL, invArgs := invQB.Build()
		if _, err := tx.Exec(ctx, invSQL, invArgs...); err != nil {
			return errors.Internal(err)
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// GetByID
// ---------------------------------------------------------------------------

// GetByID retrieves a product by its primary key.
func (r *ProductRepository) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	selectCols := append(prefixColumns("p", productColumns),
		"COALESCE(i.quantity, 0) AS inv_quantity",
		"COALESCE(i.reserved_qty, 0) AS inv_reserved_qty",
		"COALESCE(i.available_qty, 0) AS inv_available_qty",
		"COALESCE(i.low_stock_threshold, 0) AS inv_low_stock_threshold",
	)
	qb := querybuilder.Select(selectCols...).From("products p").
		LeftJoin("inventory i", "i.product_id = p.id").
		Where("p.id", id)
	query, args := qb.Build()

	var row productRow
	if err := pgxscan.Get(ctx, r.pool, &row, query, args...); err != nil {
		if pgxscan.NotFound(err) {
			return nil, errors.New(errors.ErrCodeProductNotFound, "Product not found")
		}
		return nil, errors.Internal(err)
	}

	product := row.toProduct()
	if err := loadProductRelations(ctx, r.pool, []*domain.Product{product}); err != nil {
		return nil, err
	}
	return product, nil
}

// ---------------------------------------------------------------------------
// GetBySKU
// ---------------------------------------------------------------------------

// GetBySKU retrieves a product by its unique SKU.
func (r *ProductRepository) GetBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	selectCols := append(prefixColumns("p", productColumns),
		"COALESCE(i.quantity, 0) AS inv_quantity",
		"COALESCE(i.reserved_qty, 0) AS inv_reserved_qty",
		"COALESCE(i.available_qty, 0) AS inv_available_qty",
		"COALESCE(i.low_stock_threshold, 0) AS inv_low_stock_threshold",
	)
	qb := querybuilder.Select(selectCols...).From("products p").
		LeftJoin("inventory i", "i.product_id = p.id").
		Where("p.sku", sku)
	query, args := qb.Build()

	var row productRow
	if err := pgxscan.Get(ctx, r.pool, &row, query, args...); err != nil {
		if pgxscan.NotFound(err) {
			return nil, errors.New(errors.ErrCodeProductNotFound, "Product not found")
		}
		return nil, errors.Internal(err)
	}

	product := row.toProduct()
	if err := loadProductRelations(ctx, r.pool, []*domain.Product{product}); err != nil {
		return nil, err
	}
	return product, nil
}

// ---------------------------------------------------------------------------
// MaxSlugSuffix
// ---------------------------------------------------------------------------

// MaxSlugSuffix returns the highest numeric suffix among slugs matching base:
// 0 if base is unused, 1 if only the bare base exists, else max N of "base-N".
// base comes from generateSlug ([a-z0-9-] only), so it is regex-safe here.
// excludeID (when non-empty) skips that product's own row.
func (r *ProductRepository) MaxSlugSuffix(ctx context.Context, base, excludeID string) (int, error) {
	const q = `
		SELECT COALESCE(MAX(
			CASE WHEN slug = $1 THEN 1
			     ELSE substring(slug from '^' || $1 || '-([0-9]+)$')::int
			END), 0)
		FROM products
		WHERE (slug = $1 OR slug ~ ('^' || $1 || '-[0-9]+$'))
		  AND ($2 = '' OR id <> $2)`
	var maxN int
	if err := r.pool.QueryRow(ctx, q, base, excludeID).Scan(&maxN); err != nil {
		return 0, errors.Internal(err)
	}
	return maxN, nil
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

// Update replaces a product's core columns, attribute values and images
// inside a single transaction.
func (r *ProductRepository) Update(ctx context.Context, product *domain.Product) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return errors.Internal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := r.updateProductInTx(ctx, tx, product); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return errors.Internal(err)
	}
	return nil
}

// updateProductInTx executes the full product update (core columns + attribute
// values + images) on an existing transaction. It mutates product.UpdatedAt.
func (r *ProductRepository) updateProductInTx(ctx context.Context, tx pgx.Tx, product *domain.Product) error {
	var dimLength, dimWidth, dimHeight *float64
	dimUnit := "cm"
	if product.Dimensions != nil {
		dimLength = &product.Dimensions.Length
		dimWidth = &product.Dimensions.Width
		dimHeight = &product.Dimensions.Height
		dimUnit = product.Dimensions.Unit
	}

	product.UpdatedAt = time.Now()

	qb := querybuilder.Update("products").
		Set(ColName, product.Name).
		Set(ColSlug, product.Slug).
		Set(ColSKU, product.SKU).
		Set(ColDescription, product.Description).
		Set(ColCategoryID, product.CategoryID).
		Set(ColBasePrice, product.BasePrice).
		Set(ColSellingPrice, product.SellingPrice).
		Set(ColCostPrice, product.CostPrice).
		Set(ColCurrency, product.Currency).
		Set(ColDimLength, dimLength).
		Set(ColDimWidth, dimWidth).
		Set(ColDimHeight, dimHeight).
		Set(ColDimUnit, dimUnit).
		Set(ColWeight, product.Weight).
		Set(ColAllowCustomDimensions, product.AllowCustomDimensions).
		Set(ColPricingRuleID, product.PricingRuleID).
		Set(ColTags, product.Tags).
		Set(ColStatus, string(product.Status)).
		Set(ColSortOrder, product.SortOrder).
		Set(ColVideoURL, nullableString(product.VideoURL)).
		Set(ColVideoPosterURL, nullableString(product.VideoPosterURL)).
		Set(ColUpdatedAt, product.UpdatedAt).
		Set(ColUpdatedBy, product.UpdatedBy).
		Where(ColID, product.ID)

	query, args := qb.Build()
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return errors.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeProductNotFound, "Product not found")
	}

	// Replace attribute values
	if _, err := tx.Exec(ctx, `DELETE FROM product_attribute_values WHERE product_id = $1`, product.ID); err != nil {
		return errors.Internal(err)
	}
	if err := insertAttributeValues(ctx, tx, product); err != nil {
		return err
	}

	// Replace images
	if _, err := tx.Exec(ctx, `DELETE FROM product_images WHERE product_id = $1`, product.ID); err != nil {
		return errors.Internal(err)
	}
	if err := insertImages(ctx, tx, product); err != nil {
		return err
	}

	return nil
}

// ---------------------------------------------------------------------------
// UpsertProductWithEmbedding
// ---------------------------------------------------------------------------

// UpsertProductWithEmbedding is the embedding-aware create path. It performs
// everything createProductInTx does (product row + attributes + images +
// inventory), then conditionally writes the embedding vector, all in one
// transaction. Passing a nil embedding leaves the embedding columns unset.
func (r *ProductRepository) UpsertProductWithEmbedding(ctx context.Context, product *domain.Product, inventory *domain.Inventory, embedding []float32) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return errors.Internal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := r.createProductInTx(ctx, tx, product, inventory); err != nil {
		return err
	}
	if embedding != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE products SET embedding = $1, embedding_updated_at = now() WHERE id = $2`,
			pgvector.NewVector(embedding), product.ID,
		); err != nil {
			return errors.Wrap(err, "write embedding")
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return errors.Internal(err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// UpdateProductWithOptionalEmbedding
// ---------------------------------------------------------------------------

// UpdateProductWithOptionalEmbedding is the embedding-aware update path. It
// performs everything updateProductInTx does, then conditionally writes the
// embedding vector according to the contract:
//   - writeEmbedding == false             → embedding columns untouched
//   - writeEmbedding == true,  nil vec    → embedding columns untouched
//   - writeEmbedding == true, !nil vec    → write embedding + embedding_updated_at = now()
func (r *ProductRepository) UpdateProductWithOptionalEmbedding(ctx context.Context, product *domain.Product, embedding []float32, writeEmbedding bool) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return errors.Internal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := r.updateProductInTx(ctx, tx, product); err != nil {
		return err
	}
	if writeEmbedding && embedding != nil {
		if _, err := tx.Exec(ctx,
			`UPDATE products SET embedding = $1, embedding_updated_at = now() WHERE id = $2`,
			pgvector.NewVector(embedding), product.ID,
		); err != nil {
			return errors.Wrap(err, "write embedding")
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return errors.Internal(err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

// Delete removes a product by ID. Related attribute_values and images are
// removed automatically via ON DELETE CASCADE.
func (r *ProductRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return errors.Internal(err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New(errors.ErrCodeProductNotFound, "Product not found")
	}
	return nil
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

// List retrieves products with dynamic filters, attribute-filter sub-queries,
// cursor-based pagination and sort order.
func (r *ProductRepository) List(ctx context.Context, req domain.ListProductsRequest) (*domain.ListProductsResponse, error) {
	limit, offset := pageParams(req.PaginationRequest)

	// Merge Material / Color into AttributeFilters so they go through the
	// same EXISTS-subquery path.
	attrFilters := make(map[string][]string)
	for k, v := range req.AttributeFilters {
		attrFilters[k] = v
	}
	if req.Material != nil && *req.Material != "" {
		attrFilters["material"] = []string{*req.Material}
	}
	if req.Color != nil && *req.Color != "" {
		attrFilters["color"] = []string{*req.Color}
	}

	searching := req.Search != ""

	// Always LEFT JOIN inventory so product rows include live stock data.
	selectCols := append(prefixColumns("p", productListColumns),
		"COALESCE(i.quantity, 0) AS inv_quantity",
		"COALESCE(i.reserved_qty, 0) AS inv_reserved_qty",
		"COALESCE(i.available_qty, 0) AS inv_available_qty",
		"COALESCE(i.low_stock_threshold, 0) AS inv_low_stock_threshold",
	)

	qb := querybuilder.Select(selectCols...).
		From("products p").
		LeftJoin("inventory i", "i.product_id = p.id").
		WithFilter(req.CategoryID != nil, "p.category_id", deref(req.CategoryID)).
		WithFilter(req.Status != nil, "p.status", string(deref(req.Status))).
		WithFilter(req.Slug != "", "p.slug", req.Slug).
		WithSearch(searching, "p.search_vector", "p.name", req.Search, "p.sku").
		WithRange("p.selling_price", req.MinPrice, req.MaxPrice).
		Limit(limit + 1).
		Offset(offset)

	if searching {
		qb.OrderByRaw("ts_rank(p.search_vector, websearch_to_tsquery('english', %s)) DESC, p.sort_order, p.id", req.Search)
	} else {
		qb.OrderBy("p.sort_order, p.id")
	}

	if req.InStock != nil && *req.InStock {
		qb.WithRaw(true, "i.available_qty > 0")
	}
	if req.LowStock != nil && *req.LowStock {
		qb.WithRaw(true, "i.available_qty <= i.low_stock_threshold")
	}

	for attrName, values := range attrFilters {
		qb.WithRaw(len(values) > 0,
			"EXISTS (SELECT 1 FROM product_attribute_values v WHERE v.product_id = p.id AND v.attribute_name = %s AND v.attribute_value = ANY(%s))",
			attrName, values,
		)
	}

	query, args := qb.Build()

	var rows []productRow
	if err := pgxscan.Select(ctx, r.pool, &rows, query, args...); err != nil {
		return nil, errors.Internal(err)
	}

	fetched := len(rows)
	if fetched > limit {
		rows = rows[:limit]
	}

	products := scanProductRows(rows)

	if err := loadProductRelations(ctx, r.pool, products); err != nil {
		return nil, err
	}

	return &domain.ListProductsResponse{
		Products:   products,
		Pagination: buildPaginationResponse(limit, offset, fetched),
	}, nil
}

// ---------------------------------------------------------------------------
// BatchGetByIDs
// ---------------------------------------------------------------------------

// BatchGetByIDs retrieves multiple products by their IDs.
func (r *ProductRepository) BatchGetByIDs(ctx context.Context, ids []string) ([]*domain.Product, error) {
	if len(ids) == 0 {
		return []*domain.Product{}, nil
	}

	selectCols := append(prefixColumns("p", productListColumns),
		"COALESCE(i.quantity, 0) AS inv_quantity",
		"COALESCE(i.reserved_qty, 0) AS inv_reserved_qty",
		"COALESCE(i.available_qty, 0) AS inv_available_qty",
		"COALESCE(i.low_stock_threshold, 0) AS inv_low_stock_threshold",
	)
	qb := querybuilder.Select(selectCols...).From("products p").
		LeftJoin("inventory i", "i.product_id = p.id").
		WhereIn("p.id", ids)
	query, args := qb.Build()

	var rows []productRow
	if err := pgxscan.Select(ctx, r.pool, &rows, query, args...); err != nil {
		return nil, errors.Internal(err)
	}

	products := scanProductRows(rows)

	if err := loadProductRelations(ctx, r.pool, products); err != nil {
		return nil, err
	}

	return products, nil
}

// ---------------------------------------------------------------------------
// BatchUpdateSortOrder
// ---------------------------------------------------------------------------

// BatchUpdateSortOrder updates the sort_order column for a batch of products
// inside a single transaction.
func (r *ProductRepository) BatchUpdateSortOrder(ctx context.Context, products []*domain.Product) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return errors.Internal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now()
	for _, p := range products {
		p.UpdatedAt = now
		_, err := tx.Exec(ctx,
			`UPDATE products SET sort_order = $1, updated_at = $2 WHERE id = $3`,
			p.SortOrder, p.UpdatedAt, p.ID,
		)
		if err != nil {
			return errors.Internal(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return errors.Internal(err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// GetByCategoryAll
// ---------------------------------------------------------------------------

// GetByCategoryAll retrieves every product in a category ordered by sort_order.
func (r *ProductRepository) GetByCategoryAll(ctx context.Context, categoryID string) ([]*domain.Product, error) {
	selectCols := append(prefixColumns("p", productListColumns),
		"COALESCE(i.quantity, 0) AS inv_quantity",
		"COALESCE(i.reserved_qty, 0) AS inv_reserved_qty",
		"COALESCE(i.available_qty, 0) AS inv_available_qty",
		"COALESCE(i.low_stock_threshold, 0) AS inv_low_stock_threshold",
	)
	qb := querybuilder.Select(selectCols...).From("products p").
		LeftJoin("inventory i", "i.product_id = p.id").
		Where("p.category_id", categoryID).OrderBy("p.sort_order, p.id")
	query, args := qb.Build()

	var rows []productRow
	if err := pgxscan.Select(ctx, r.pool, &rows, query, args...); err != nil {
		return nil, errors.Internal(err)
	}

	products := scanProductRows(rows)

	if err := loadProductRelations(ctx, r.pool, products); err != nil {
		return nil, err
	}

	return products, nil
}

// ---------------------------------------------------------------------------
// GetAttributeFilterOptions
// ---------------------------------------------------------------------------

// GetAttributeFilterOptions returns distinct attribute values for each of the
// requested attribute names, scoped to products in the given category.
func (r *ProductRepository) GetAttributeFilterOptions(ctx context.Context, categoryID string, attrNames []string) (map[string][]string, error) {
	result := make(map[string][]string, len(attrNames))
	for _, attrName := range attrNames {
		rows, err := r.pool.Query(ctx, `
			SELECT DISTINCT v.attribute_value
			FROM product_attribute_values v
			WHERE v.product_id IN (SELECT id FROM products WHERE category_id = $1)
			  AND v.attribute_name = $2
			ORDER BY v.attribute_value`, categoryID, attrName)
		if err != nil {
			return nil, errors.Internal(err)
		}

		var values []string
		for rows.Next() {
			var val string
			if err := rows.Scan(&val); err != nil {
				rows.Close()
				return nil, errors.Internal(err)
			}
			values = append(values, val)
		}
		rows.Close()
		if rows.Err() != nil {
			return nil, errors.Internal(rows.Err())
		}

		result[attrName] = values
	}
	return result, nil
}

// ===========================================================================
// Helpers
// ===========================================================================

// loadProductRelations batch-loads attribute values and images for a slice of
// products, populating the Attributes map, hardcoded attribute fields (Material,
// Color, etc.) and the Images slice on each product.
func loadProductRelations(ctx context.Context, pool *pgxpool.Pool, products []*domain.Product) error {
	if len(products) == 0 {
		return nil
	}

	ids := make([]string, len(products))
	productsByID := make(map[string]*domain.Product, len(products))
	for i, p := range products {
		ids[i] = p.ID
		productsByID[p.ID] = p
		// Initialize maps/slices so callers always get non-nil values.
		p.Attributes = make(map[string]interface{})
		p.Images = nil
	}

	if err := loadProductAttributes(ctx, pool, ids, productsByID); err != nil {
		return err
	}
	return loadProductImages(ctx, pool, ids, productsByID)
}

type attrPair struct{ name, value string }

// loadProductAttributes batch-loads attribute values for products and populates
// the Attributes map and hardcoded fields (Material, Color, etc.).
func loadProductAttributes(ctx context.Context, pool *pgxpool.Pool, ids []string, productsByID map[string]*domain.Product) error {
	attrRows, err := pool.Query(ctx, `
		SELECT product_id, attribute_name, attribute_value
		FROM product_attribute_values
		WHERE product_id = ANY($1)
		ORDER BY product_id, attribute_name, attribute_value`, ids)
	if err != nil {
		return errors.Internal(err)
	}
	defer attrRows.Close()

	// Collect all (name, value) pairs per product so we can build proper
	// multi-value entries in the Attributes map.
	productAttrs := make(map[string][]attrPair)

	for attrRows.Next() {
		var productID, attrName, attrValue string
		if scanErr := attrRows.Scan(&productID, &attrName, &attrValue); scanErr != nil {
			return errors.Internal(scanErr)
		}
		productAttrs[productID] = append(productAttrs[productID], attrPair{attrName, attrValue})
	}
	if attrRows.Err() != nil {
		return errors.Internal(attrRows.Err())
	}

	// Build the Attributes map and set hardcoded fields.
	for pid, pairs := range productAttrs {
		p := productsByID[pid]
		if p == nil {
			continue
		}
		applyAttributePairs(p, pairs)
	}
	return nil
}

// applyAttributePairs groups attribute pairs by name and populates the product's
// Attributes map and hardcoded fields.
func applyAttributePairs(p *domain.Product, pairs []attrPair) {
	hardcodedNames := make(map[string]bool, len(hardcodedAttrFields))
	for _, h := range hardcodedAttrFields {
		hardcodedNames[h.Name] = true
	}

	// Group values by attribute name.
	grouped := make(map[string][]string)
	for _, ap := range pairs {
		grouped[ap.name] = append(grouped[ap.name], ap.value)
	}

	for name, vals := range grouped {
		// Set hardcoded fields (take first value).
		for _, h := range hardcodedAttrFields {
			if h.Name == name && len(vals) > 0 {
				h.Set(p, vals[0])
			}
		}

		// Populate Attributes map: single value as string, multiple as []interface{}.
		if !hardcodedNames[name] {
			if len(vals) == 1 {
				p.Attributes[name] = vals[0]
			} else {
				iface := make([]interface{}, len(vals))
				for i, v := range vals {
					iface[i] = v
				}
				p.Attributes[name] = iface
			}
		}
	}
}

// loadProductImages batch-loads images for products and populates the Images slice.
func loadProductImages(ctx context.Context, pool *pgxpool.Pool, ids []string, productsByID map[string]*domain.Product) error {
	imgRows, err := pool.Query(ctx, `
		SELECT product_id, url, alt_text, sort_order
		FROM product_images
		WHERE product_id = ANY($1)
		ORDER BY product_id, sort_order`, ids)
	if err != nil {
		return errors.Internal(err)
	}
	defer imgRows.Close()

	for imgRows.Next() {
		var productID string
		var img domain.ProductImage
		if scanErr := imgRows.Scan(&productID, &img.URL, &img.AltText, &img.SortOrder); scanErr != nil {
			return errors.Internal(scanErr)
		}
		if p, ok := productsByID[productID]; ok {
			// The first image (lowest sort_order) is treated as primary.
			if len(p.Images) == 0 {
				img.IsPrimary = true
			}
			p.Images = append(p.Images, img)
		}
	}
	if imgRows.Err() != nil {
		return errors.Internal(imgRows.Err())
	}

	return nil
}

// insertAttributeValues writes rows into product_attribute_values for both the
// flexible Attributes map and the hardcoded fields (Material, Color, etc.).
// attributeRow holds a single name-value pair destined for the product_attribute_values table.
type attributeRow struct {
	name  string
	value string
}

func insertAttributeValues(ctx context.Context, tx pgx.Tx, product *domain.Product) error {
	var rows []attributeRow

	rows = collectDynamicAttributes(product.Attributes, rows)
	rows = collectHardcodedAttributes(product, rows)

	if len(rows) == 0 {
		return nil
	}

	query, args := buildAttributeBatchInsert(product.ID, rows)
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return errors.Internal(err)
	}
	return nil
}

// collectDynamicAttributes extracts attribute rows from the flexible Attributes map.
// Values can be a single string, []string, or []interface{} (from JSON unmarshalling).
func collectDynamicAttributes(attributes map[string]interface{}, rows []attributeRow) []attributeRow {
	for attrName, attrVal := range attributes {
		switch typedVal := attrVal.(type) {
		case string:
			if typedVal != "" {
				rows = append(rows, attributeRow{attrName, typedVal})
			}
		case []interface{}:
			for _, elem := range typedVal {
				if strVal, ok := elem.(string); ok && strVal != "" {
					rows = append(rows, attributeRow{attrName, strVal})
				}
			}
		case []string:
			for _, strVal := range typedVal {
				if strVal != "" {
					rows = append(rows, attributeRow{attrName, strVal})
				}
			}
		}
	}
	return rows
}

// collectHardcodedAttributes extracts attribute rows from fixed product fields
// like Material, Color, etc.
func collectHardcodedAttributes(product *domain.Product, rows []attributeRow) []attributeRow {
	for _, field := range hardcodedAttrFields {
		fieldVal := field.Field(product)
		if fieldVal != "" {
			rows = append(rows, attributeRow{field.Name, fieldVal})
		}
	}
	return rows
}

// buildAttributeBatchInsert constructs a multi-row INSERT statement for product_attribute_values.
func buildAttributeBatchInsert(productID string, rows []attributeRow) (string, []interface{}) {
	qb := querybuilder.InsertBatch("product_attribute_values", "product_id", "attribute_name", "attribute_value").
		OnConflictDoNothing()

	for _, row := range rows {
		qb.AddRow(productID, row.name, row.value)
	}

	return qb.Build()
}

// nullableString returns nil for empty strings so that empty values are written
// as NULL rather than empty TEXT. Avoids querying `WHERE col = ”` ambiguity.
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// insertImages writes rows into product_images for the product.
func insertImages(ctx context.Context, tx pgx.Tx, product *domain.Product) error {
	if len(product.Images) == 0 {
		return nil
	}

	qb := querybuilder.InsertBatch("product_images", "id", "product_id", "url", "alt_text", "sort_order")
	for _, img := range product.Images {
		qb.AddRow(uuid.New().String(), product.ID, img.URL, img.AltText, img.SortOrder)
	}

	query, args := qb.Build()
	_, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return errors.Internal(err)
	}
	return nil
}
