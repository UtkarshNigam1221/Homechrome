package embedder

import (
	"context"
	"encoding/base64"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"

	"github.com/handloom/admin/internal/domain"
)

// Weights are the linear-combination coefficients used by the hybrid SQL.
type Weights struct {
	Semantic float64
	Keyword  float64
	Trigram  float64
}

// Searcher runs hybrid queries (semantic + tsvector + trigram) on Postgres.
type Searcher struct {
	pool    *pgxpool.Pool
	repo    domain.ProductRepository
	weights Weights
}

func NewSearcher(pool *pgxpool.Pool, repo domain.ProductRepository, w Weights) *Searcher {
	return &Searcher{pool: pool, repo: repo, weights: w}
}

// semanticThreshold is the minimum cosine similarity (1 - distance) required for a
// semantic match to count as a positive signal in the WHERE clause.
// Products that match only on keyword or trigram are still returned.
const semanticThreshold = 0.25

// SQL is built dynamically by buildSearchSQL so we can inject optional filter
// clauses without trying to express every combination as a static string. The
// hybrid scoring + threshold logic is identical to before; only the WHERE
// clause grows with the filters present in SearchRequest.

// buildSearchSQL constructs the dynamic hybrid-search SQL with optional filters.
// Returns the SQL string + the args slice in $1, $2, ... order.
//
// The base query selects p.id from products joined left with inventory (so
// in_stock filtering works) and scored by the hybrid expression. Filters are
// AND-ed into the WHERE clause. When qvec is nil the semantic term collapses
// to 0 in scoring and the cosine threshold branch is skipped.
func buildSearchSQL(qvec []float32, q string, req SearchRequest, w Weights, limit, offset int) (string, []any) {
	hasSemantic := qvec != nil

	var sb strings.Builder
	var args []any
	add := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	// CTE: bind q + (optional) qvec once for the planner.
	if hasSemantic {
		qvecPlaceholder := add(pgvector.NewVector(qvec))
		qPlaceholder := add(q)
		fmt.Fprintf(&sb, `WITH query_data AS (
  SELECT
    %s::vector(768) AS qvec,
    websearch_to_tsquery('english', %s) AS qts,
    %s::text AS qstr
)
`, qvecPlaceholder, qPlaceholder, qPlaceholder)
	} else {
		qPlaceholder := add(q)
		fmt.Fprintf(&sb, `WITH query_data AS (
  SELECT
    websearch_to_tsquery('english', %s) AS qts,
    %s::text AS qstr
)
`, qPlaceholder, qPlaceholder)
	}

	sb.WriteString(`SELECT p.id
FROM products p
LEFT JOIN inventory i ON i.product_id = p.id, query_data qd
WHERE p.status = 'ACTIVE'
`)

	// Text-match gate: at least one of semantic / keyword / trigram must hit.
	// When the query is empty (filter-only listing) the relevance gate is
	// skipped so we return all matching products regardless of search rank.
	if q != "" {
		if hasSemantic {
			semThresholdPlaceholder := add(semanticThreshold)
			fmt.Fprintf(&sb, `  AND (
    (p.embedding IS NOT NULL AND (1 - (p.embedding <=> qd.qvec)) > %s::float)
    OR p.search_vector @@ qd.qts
    OR p.name %% qd.qstr
    OR p.sku %% qd.qstr
  )
`, semThresholdPlaceholder)
		} else {
			sb.WriteString(`  AND (p.search_vector @@ qd.qts OR p.name % qd.qstr OR p.sku % qd.qstr)
`)
		}
	}

	// Filters.
	if req.CategoryID != "" {
		fmt.Fprintf(&sb, `  AND p.category_id = %s
`, add(req.CategoryID))
	}
	if req.MinPrice != nil {
		fmt.Fprintf(&sb, `  AND p.selling_price >= %s
`, add(*req.MinPrice))
	}
	if req.MaxPrice != nil {
		fmt.Fprintf(&sb, `  AND p.selling_price <= %s
`, add(*req.MaxPrice))
	}
	if req.InStockOnly {
		sb.WriteString(`  AND COALESCE(i.available_qty, 0) > 0
`)
	}
	// material/color are stored as attribute rows — treat them uniformly with
	// af_* filters. Sorted keys keep the SQL string deterministic so pgx's
	// prepared-statement cache doesn't thrash on clause reordering.
	attrs := make(map[string][]string, len(req.AttributeFilters)+2)
	for k, v := range req.AttributeFilters {
		attrs[k] = v
	}
	if req.Material != "" {
		attrs["material"] = append(attrs["material"], req.Material)
	}
	if req.Color != "" {
		attrs["color"] = append(attrs["color"], req.Color)
	}
	for _, attrName := range slices.Sorted(maps.Keys(attrs)) {
		values := attrs[attrName]
		if len(values) == 0 {
			continue
		}
		fmt.Fprintf(&sb, `  AND EXISTS (SELECT 1 FROM product_attribute_values v WHERE v.product_id = p.id AND v.attribute_name = %s AND v.attribute_value = ANY(%s))
`, add(attrName), add(values))
	}

	// Ordering. With a query: weighted hybrid score. Without: stable sort_order/id.
	if q != "" {
		terms := []string{
			fmt.Sprintf(`%s::float * LEAST(COALESCE(ts_rank(p.search_vector, qd.qts), 0) * 10, 1.0)`, add(w.Keyword)),
			fmt.Sprintf(`%s::float * GREATEST(similarity(p.name, qd.qstr), similarity(p.sku, qd.qstr))`, add(w.Trigram)),
		}
		if hasSemantic {
			terms = append([]string{
				fmt.Sprintf(`%s::float * COALESCE(1 - (p.embedding <=> qd.qvec), 0)`, add(w.Semantic)),
			}, terms...)
		}
		fmt.Fprintf(&sb, `ORDER BY (%s) DESC, p.id ASC
`, strings.Join(terms, " + "))
	} else {
		sb.WriteString(`ORDER BY p.sort_order, p.id
`)
	}

	fmt.Fprintf(&sb, `LIMIT %s OFFSET %s`, add(limit), add(offset))

	return sb.String(), args
}

// scanIDRows drains pgx.Rows into a slice of product IDs (score-ordered).
func scanIDRows(rows pgx.Rows) ([]string, error) {
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// encodeCursor encodes an integer offset as a base64 cursor string.
func encodeCursor(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

// DecodeCursor decodes a base64 cursor string back to an integer offset.
// Returns 0 if the cursor is empty or invalid.
func DecodeCursor(cursor string) int {
	if cursor == "" {
		return 0
	}
	b, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(string(b))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// toStoreProduct converts a domain.Product to a StoreProduct, matching the
// conversion logic in handler/store/catalog_handler.go:toStoreProduct.
func toStoreProduct(p *domain.Product) *StoreProduct {
	// Convert domain.ProductImage slice to embedder ProductImage slice.
	var images []ProductImage
	if len(p.Images) > 0 {
		images = make([]ProductImage, len(p.Images))
		for i, img := range p.Images {
			images[i] = ProductImage{
				URL:       img.URL,
				AltText:   img.AltText,
				IsPrimary: img.IsPrimary,
				SortOrder: img.SortOrder,
			}
		}
	}

	// Convert domain.Dimensions to embedder Dimensions.
	var dims *Dimensions
	if p.Dimensions != nil {
		dims = &Dimensions{
			Length: p.Dimensions.Length,
			Width:  p.Dimensions.Width,
			Height: p.Dimensions.Height,
			Unit:   p.Dimensions.Unit,
		}
	}

	return &StoreProduct{
		ID:                    p.ID,
		Name:                  p.Name,
		Slug:                  p.Slug,
		SKU:                   p.SKU,
		Description:           p.Description,
		CategoryID:            p.CategoryID,
		BasePrice:             p.BasePrice,
		SellingPrice:          p.SellingPrice,
		Currency:              p.Currency,
		Dimensions:            dims,
		Weight:                p.Weight,
		AllowCustomDimensions: p.AllowCustomDimensions,
		PricingRuleID:         p.PricingRuleID,
		Attributes:            p.Attributes,
		Material:              p.Material,
		Color:                 p.Color,
		WeaveType:             p.WeaveType,
		Origin:                p.Origin,
		CraftType:             p.CraftType,
		Images:                images,
		VideoURL:              p.VideoURL,
		VideoPosterURL:        p.VideoPosterURL,
		Tags:                  p.Tags,
		InStock:               p.AvailableQty > 0,
		CreatedAt:             p.CreatedAt,
		UpdatedAt:             p.UpdatedAt,
	}
}

// Search runs the hybrid query with optional filters. When qvec is nil (embedder
// call failed) the semantic term is omitted entirely to avoid a zero-vector
// producing NaN via undefined cosine distance.
//
// Step 1: SQL returns only (id) in score order, fetching limit+1 to detect HasMore.
// Step 2: BatchGetByIDs hydrates the full domain.Product structs.
// Step 3: Products are assembled in score order and converted to StoreProduct.
func (s *Searcher) Search(ctx context.Context, qvec []float32, req SearchRequest, offset int) (SearchResponse, error) {
	limit := req.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// Fetch one extra row to determine whether a next page exists.
	fetchLimit := limit + 1
	query, args := buildSearchSQL(qvec, req.Query, req, s.weights, fetchLimit, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("search query: %w", err)
	}
	ids, err := scanIDRows(rows)
	if err != nil {
		return SearchResponse{}, err
	}

	// Trim the sentinel row and derive hasMore.
	hasMore := len(ids) > limit
	if hasMore {
		ids = ids[:limit]
	}

	// Hydrate products via BatchGetByIDs.
	var storeProducts []*StoreProduct
	if len(ids) > 0 {
		products, batchErr := s.repo.BatchGetByIDs(ctx, ids)
		if batchErr != nil {
			return SearchResponse{}, fmt.Errorf("batch get products: %w", batchErr)
		}

		// Build a map for O(1) lookup.
		productMap := make(map[string]*domain.Product, len(products))
		for _, p := range products {
			productMap[p.ID] = p
		}

		// Assemble in score-ordered ID slice order.
		storeProducts = make([]*StoreProduct, 0, len(ids))
		for _, id := range ids {
			if p, ok := productMap[id]; ok {
				storeProducts = append(storeProducts, toStoreProduct(p))
			}
		}
	}

	if storeProducts == nil {
		storeProducts = []*StoreProduct{}
	}

	// Build pagination meta matching the legacy endpoint shape.
	var nextCursor string
	if hasMore {
		nextCursor = encodeCursor(offset + limit)
	}

	return SearchResponse{
		Success: true,
		Data:    storeProducts,
		Meta: paginationMeta{
			Limit:      limit,
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}
