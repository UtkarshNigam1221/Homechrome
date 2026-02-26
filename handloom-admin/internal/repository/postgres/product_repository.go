package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/handloom/admin/internal/domain"
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
	{"color", func(p *domain.Product) string { return p.Color }, func(p *domain.Product, v string) { p.Color = v }},
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
	defer tx.Rollback(ctx) //nolint:errcheck

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

	_, err = tx.Exec(ctx, `
		INSERT INTO products (
			id, name, slug, sku, description, category_id,
			base_price, selling_price, cost_price, currency,
			dim_length, dim_width, dim_height, dim_unit,
			weight, allow_custom_dimensions, pricing_rule_id,
			tags, status, sort_order,
			created_at, updated_at, created_by, updated_by
		) VALUES (
			$1,$2,$3,$4,$5,$6,
			$7,$8,$9,$10,
			$11,$12,$13,$14,
			$15,$16,$17,
			$18,$19,$20,
			$21,$22,$23,$24
		)`,
		product.ID, product.Name, product.Slug, product.SKU, product.Description, product.CategoryID,
		product.BasePrice, product.SellingPrice, product.CostPrice, product.Currency,
		dimLength, dimWidth, dimHeight, dimUnit,
		product.Weight, product.AllowCustomDimensions, product.PricingRuleID,
		product.Tags, string(product.Status), product.SortOrder,
		product.CreatedAt, product.UpdatedAt, product.CreatedBy, product.UpdatedBy,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			if strings.Contains(err.Error(), "products_sku_key") {
				return errors.New(errors.ErrCodeProductSKUExists, "Product with this SKU already exists")
			}
			return errors.Conflict("Product already exists")
		}
		return errors.Internal(err)
	}

	// --- attribute values ---
	if err := insertAttributeValues(ctx, tx, product); err != nil {
		return err
	}

	// --- images ---
	if err := insertImages(ctx, tx, product); err != nil {
		return err
	}

	// --- inventory ---
	if inventory != nil {
		if inventory.ID == "" {
			inventory.ID = uuid.New().String()
		}
		inventory.CreatedAt = now
		inventory.UpdatedAt = now

		_, err = tx.Exec(ctx, `
			INSERT INTO inventory (
				id, product_id,
				quantity, reserved_qty, available_qty,
				low_stock_threshold, reorder_point, last_restock_at,
				created_at, updated_at, created_by, updated_by
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			inventory.ID, inventory.ProductID,
			inventory.Quantity, inventory.ReservedQty, inventory.AvailableQty,
			inventory.LowStockThreshold, inventory.ReorderPoint, inventory.LastRestockAt,
			inventory.CreatedAt, inventory.UpdatedAt, inventory.CreatedBy, inventory.UpdatedBy,
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
// GetByID
// ---------------------------------------------------------------------------

// GetByID retrieves a product by its primary key.
func (r *ProductRepository) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, sku, description, category_id,
			base_price, selling_price, cost_price, currency,
			dim_length, dim_width, dim_height, dim_unit,
			weight, allow_custom_dimensions, pricing_rule_id,
			tags, status, sort_order,
			created_at, updated_at, created_by, updated_by
		FROM products WHERE id = $1`, id)

	product, err := scanProduct(row)
	if err != nil {
		return nil, err
	}

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
	row := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, sku, description, category_id,
			base_price, selling_price, cost_price, currency,
			dim_length, dim_width, dim_height, dim_unit,
			weight, allow_custom_dimensions, pricing_rule_id,
			tags, status, sort_order,
			created_at, updated_at, created_by, updated_by
		FROM products WHERE sku = $1`, sku)

	product, err := scanProduct(row)
	if err != nil {
		return nil, err
	}

	if err := loadProductRelations(ctx, r.pool, []*domain.Product{product}); err != nil {
		return nil, err
	}
	return product, nil
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
	defer tx.Rollback(ctx) //nolint:errcheck

	var dimLength, dimWidth, dimHeight *float64
	dimUnit := "cm"
	if product.Dimensions != nil {
		dimLength = &product.Dimensions.Length
		dimWidth = &product.Dimensions.Width
		dimHeight = &product.Dimensions.Height
		dimUnit = product.Dimensions.Unit
	}

	product.UpdatedAt = time.Now()

	tag, err := tx.Exec(ctx, `
		UPDATE products SET
			name=$1, slug=$2, sku=$3, description=$4, category_id=$5,
			base_price=$6, selling_price=$7, cost_price=$8, currency=$9,
			dim_length=$10, dim_width=$11, dim_height=$12, dim_unit=$13,
			weight=$14, allow_custom_dimensions=$15, pricing_rule_id=$16,
			tags=$17, status=$18, sort_order=$19,
			updated_at=$20, updated_by=$21
		WHERE id = $22`,
		product.Name, product.Slug, product.SKU, product.Description, product.CategoryID,
		product.BasePrice, product.SellingPrice, product.CostPrice, product.Currency,
		dimLength, dimWidth, dimHeight, dimUnit,
		product.Weight, product.AllowCustomDimensions, product.PricingRuleID,
		product.Tags, string(product.Status), product.SortOrder,
		product.UpdatedAt, product.UpdatedBy,
		product.ID,
	)
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

	// ---- build query ----
	var (
		where []string
		args  []interface{}
		argN  int // running $N counter
	)

	nextArg := func(val interface{}) string {
		argN++
		args = append(args, val)
		return fmt.Sprintf("$%d", argN)
	}

	needInventoryJoin := false

	if req.CategoryID != nil {
		where = append(where, "p.category_id = "+nextArg(*req.CategoryID))
	}
	if req.Status != nil {
		where = append(where, "p.status = "+nextArg(string(*req.Status)))
	}
	if req.Search != "" {
		where = append(where, "p.name ILIKE "+nextArg("%"+req.Search+"%"))
	}
	if req.MinPrice != nil {
		where = append(where, "p.selling_price >= "+nextArg(*req.MinPrice))
	}
	if req.MaxPrice != nil {
		where = append(where, "p.selling_price <= "+nextArg(*req.MaxPrice))
	}
	if req.InStock != nil && *req.InStock {
		needInventoryJoin = true
		where = append(where, "i.available_qty > 0")
	}
	if req.LowStock != nil && *req.LowStock {
		needInventoryJoin = true
		where = append(where, "i.available_qty <= i.low_stock_threshold")
	}

	// Attribute filters: each key gets an EXISTS subquery.
	for attrName, values := range attrFilters {
		if len(values) == 0 {
			continue
		}
		nameArg := nextArg(attrName)
		valuesArg := nextArg(values)
		where = append(where, fmt.Sprintf(
			`EXISTS (SELECT 1 FROM product_attribute_values v WHERE v.product_id = p.id AND v.attribute_name = %s AND v.attribute_value = ANY(%s))`,
			nameArg, valuesArg,
		))
	}

	// ---- assemble SQL ----
	var sb strings.Builder
	sb.WriteString("SELECT p.id, p.name, p.slug, p.sku, p.description, p.category_id, ")
	sb.WriteString("p.base_price, p.selling_price, p.cost_price, p.currency, ")
	sb.WriteString("p.dim_length, p.dim_width, p.dim_height, p.dim_unit, ")
	sb.WriteString("p.weight, p.allow_custom_dimensions, p.pricing_rule_id, ")
	sb.WriteString("p.tags, p.status, p.sort_order, ")
	sb.WriteString("p.created_at, p.updated_at, p.created_by, p.updated_by ")
	sb.WriteString("FROM products p ")

	if needInventoryJoin {
		sb.WriteString("LEFT JOIN inventory i ON i.product_id = p.id ")
	}

	if len(where) > 0 {
		sb.WriteString("WHERE ")
		sb.WriteString(strings.Join(where, " AND "))
		sb.WriteString(" ")
	}

	sb.WriteString("ORDER BY p.sort_order, p.id ")
	sb.WriteString(fmt.Sprintf("LIMIT %s OFFSET %s", nextArg(limit+1), nextArg(offset)))

	rows, err := r.pool.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, errors.Internal(err)
	}
	defer rows.Close()

	var products []*domain.Product
	for rows.Next() {
		p, scanErr := scanProductFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		products = append(products, p)
	}
	if rows.Err() != nil {
		return nil, errors.Internal(rows.Err())
	}

	fetched := len(products)
	if fetched > limit {
		products = products[:limit]
	}

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

	rows, err := r.pool.Query(ctx, `
		SELECT id, name, slug, sku, description, category_id,
			base_price, selling_price, cost_price, currency,
			dim_length, dim_width, dim_height, dim_unit,
			weight, allow_custom_dimensions, pricing_rule_id,
			tags, status, sort_order,
			created_at, updated_at, created_by, updated_by
		FROM products WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, errors.Internal(err)
	}
	defer rows.Close()

	var products []*domain.Product
	for rows.Next() {
		p, scanErr := scanProductFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		products = append(products, p)
	}
	if rows.Err() != nil {
		return nil, errors.Internal(rows.Err())
	}

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
	defer tx.Rollback(ctx) //nolint:errcheck

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
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, slug, sku, description, category_id,
			base_price, selling_price, cost_price, currency,
			dim_length, dim_width, dim_height, dim_unit,
			weight, allow_custom_dimensions, pricing_rule_id,
			tags, status, sort_order,
			created_at, updated_at, created_by, updated_by
		FROM products WHERE category_id = $1
		ORDER BY sort_order, id`, categoryID)
	if err != nil {
		return nil, errors.Internal(err)
	}
	defer rows.Close()

	var products []*domain.Product
	for rows.Next() {
		p, scanErr := scanProductFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		products = append(products, p)
	}
	if rows.Err() != nil {
		return nil, errors.Internal(rows.Err())
	}

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

// scanProduct scans a single product row (from QueryRow) into a *domain.Product,
// reconstructing the Dimensions struct from the flat dim_* columns.
func scanProduct(row pgx.Row) (*domain.Product, error) {
	var (
		p         domain.Product
		dimLength *float64
		dimWidth  *float64
		dimHeight *float64
		dimUnit   string
		status    string
		tags      []string
	)

	err := row.Scan(
		&p.ID, &p.Name, &p.Slug, &p.SKU, &p.Description, &p.CategoryID,
		&p.BasePrice, &p.SellingPrice, &p.CostPrice, &p.Currency,
		&dimLength, &dimWidth, &dimHeight, &dimUnit,
		&p.Weight, &p.AllowCustomDimensions, &p.PricingRuleID,
		&tags, &status, &p.SortOrder,
		&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New(errors.ErrCodeProductNotFound, "Product not found")
		}
		return nil, errors.Internal(err)
	}

	p.Status = domain.ProductStatus(status)
	if tags != nil {
		p.Tags = tags
	}

	if dimLength != nil || dimWidth != nil || dimHeight != nil {
		d := &domain.Dimensions{Unit: dimUnit}
		if dimLength != nil {
			d.Length = *dimLength
		}
		if dimWidth != nil {
			d.Width = *dimWidth
		}
		if dimHeight != nil {
			d.Height = *dimHeight
		}
		p.Dimensions = d
	}

	return &p, nil
}

// scanProductFromRows scans a product from an active pgx.Rows iterator.
func scanProductFromRows(rows pgx.Rows) (*domain.Product, error) {
	var (
		p         domain.Product
		dimLength *float64
		dimWidth  *float64
		dimHeight *float64
		dimUnit   string
		status    string
		tags      []string
	)

	err := rows.Scan(
		&p.ID, &p.Name, &p.Slug, &p.SKU, &p.Description, &p.CategoryID,
		&p.BasePrice, &p.SellingPrice, &p.CostPrice, &p.Currency,
		&dimLength, &dimWidth, &dimHeight, &dimUnit,
		&p.Weight, &p.AllowCustomDimensions, &p.PricingRuleID,
		&tags, &status, &p.SortOrder,
		&p.CreatedAt, &p.UpdatedAt, &p.CreatedBy, &p.UpdatedBy,
	)
	if err != nil {
		return nil, errors.Internal(err)
	}

	p.Status = domain.ProductStatus(status)
	if tags != nil {
		p.Tags = tags
	}

	if dimLength != nil || dimWidth != nil || dimHeight != nil {
		d := &domain.Dimensions{Unit: dimUnit}
		if dimLength != nil {
			d.Length = *dimLength
		}
		if dimWidth != nil {
			d.Width = *dimWidth
		}
		if dimHeight != nil {
			d.Height = *dimHeight
		}
		p.Dimensions = d
	}

	return &p, nil
}

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
		// Initialise maps/slices so callers always get non-nil values.
		p.Attributes = make(map[string]interface{})
		p.Images = nil
	}

	// --- attribute values ---
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
	type attrPair struct{ name, value string }
	productAttrs := make(map[string][]attrPair)

	for attrRows.Next() {
		var productID, attrName, attrValue string
		if err := attrRows.Scan(&productID, &attrName, &attrValue); err != nil {
			return errors.Internal(err)
		}
		productAttrs[productID] = append(productAttrs[productID], attrPair{attrName, attrValue})
	}
	if attrRows.Err() != nil {
		return errors.Internal(attrRows.Err())
	}

	// Build the Attributes map and set hardcoded fields.
	hardcodedNames := make(map[string]bool, len(hardcodedAttrFields))
	for _, h := range hardcodedAttrFields {
		hardcodedNames[h.Name] = true
	}

	for pid, pairs := range productAttrs {
		p := productsByID[pid]
		if p == nil {
			continue
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

	// --- images ---
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
		if err := imgRows.Scan(&productID, &img.URL, &img.AltText, &img.SortOrder); err != nil {
			return errors.Internal(err)
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
	var query strings.Builder
	query.WriteString("INSERT INTO product_attribute_values (product_id, attribute_name, attribute_value) VALUES ")

	args := make([]interface{}, 0, len(rows)*3)
	for i, row := range rows {
		if i > 0 {
			query.WriteString(",")
		}
		paramOffset := i*3 + 1
		query.WriteString(fmt.Sprintf("($%d,$%d,$%d)", paramOffset, paramOffset+1, paramOffset+2))
		args = append(args, productID, row.name, row.value)
	}
	query.WriteString(" ON CONFLICT DO NOTHING")

	return query.String(), args
}

// insertImages writes rows into product_images for the product.
func insertImages(ctx context.Context, tx pgx.Tx, product *domain.Product) error {
	if len(product.Images) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("INSERT INTO product_images (id, product_id, url, alt_text, sort_order) VALUES ")
	args := make([]interface{}, 0, len(product.Images)*5)
	for i, img := range product.Images {
		if i > 0 {
			sb.WriteString(",")
		}
		base := i*5 + 1
		sb.WriteString(fmt.Sprintf("($%d,$%d,$%d,$%d,$%d)", base, base+1, base+2, base+3, base+4))
		args = append(args, uuid.New().String(), product.ID, img.URL, img.AltText, img.SortOrder)
	}

	_, err := tx.Exec(ctx, sb.String(), args...)
	if err != nil {
		return errors.Internal(err)
	}
	return nil
}
