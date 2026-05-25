package embedder

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
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
	weights Weights
}

func NewSearcher(pool *pgxpool.Pool, w Weights) *Searcher {
	return &Searcher{pool: pool, weights: w}
}

// semanticThreshold is the minimum cosine similarity (1 - distance) required for a
// semantic match to count as a positive signal in the WHERE clause.
// Products that match only on keyword or trigram are still returned.
const semanticThreshold = 0.25

// hybridSQLWithSemantic is used when a query vector is available.
// Parameters: $1 qvec, $2 q, $3 αsem, $4 αkw, $5 αtri, $6 limit, $7 offset, $8 semThreshold.
// product_images uses sort_order (not position) per migration 001_catalog_schema.sql.
// Secondary ORDER BY p.id ensures deterministic pagination at equal scores.
const hybridSQLWithSemantic = `
WITH query_data AS (
  SELECT
    $1::vector(768) AS qvec,
    websearch_to_tsquery('english', $2) AS qts,
    $2::text AS qstr
)
SELECT
  p.id,
  p.name,
  p.slug,
  COALESCE(p.selling_price, p.base_price, 0) AS price_paise,
  COALESCE(
    (SELECT pi.url FROM product_images pi WHERE pi.product_id = p.id ORDER BY pi.sort_order LIMIT 1),
    ''
  ) AS primary_image_url
FROM products p, query_data q
WHERE p.status = 'active'
  AND (
    (p.embedding IS NOT NULL AND (1 - (p.embedding <=> q.qvec)) > $8::float)
    OR p.search_vector @@ q.qts
    OR p.name % q.qstr
  )
ORDER BY (
    $3::float * COALESCE(1 - (p.embedding <=> q.qvec), 0) +
    $4::float * LEAST(COALESCE(ts_rank(p.search_vector, q.qts), 0) * 10, 1.0) +
    $5::float * similarity(p.name, q.qstr)
) DESC, p.id ASC
LIMIT $6 OFFSET $7;
`

// hybridSQLKeywordOnly is used when no query vector is available (embedder call failed).
// Omitting the cosine term entirely avoids passing a zero-vector which would produce
// NaN from the undefined cosine distance and cause nondeterministic ORDER BY.
// Parameters: $1 q, $2 αkw, $3 αtri, $4 limit, $5 offset.
// Secondary ORDER BY p.id ensures deterministic pagination at equal scores.
const hybridSQLKeywordOnly = `
WITH query_data AS (
  SELECT
    websearch_to_tsquery('english', $1) AS qts,
    $1::text AS qstr
)
SELECT
  p.id,
  p.name,
  p.slug,
  COALESCE(p.selling_price, p.base_price, 0) AS price_paise,
  COALESCE(
    (SELECT pi.url FROM product_images pi WHERE pi.product_id = p.id ORDER BY pi.sort_order LIMIT 1),
    ''
  ) AS primary_image_url
FROM products p, query_data q
WHERE p.status = 'active'
  AND (p.search_vector @@ q.qts OR p.name % q.qstr)
ORDER BY (
    $2::float * LEAST(COALESCE(ts_rank(p.search_vector, q.qts), 0) * 10, 1.0) +
    $3::float * similarity(p.name, q.qstr)
) DESC, p.id ASC
LIMIT $4 OFFSET $5;
`

// scanRows drains pgx.Rows into a slice of SearchProduct.
func scanRows(rows pgx.Rows) ([]SearchProduct, error) {
	defer rows.Close()
	var data []SearchProduct
	for rows.Next() {
		var p SearchProduct
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.PricePaise, &p.PrimaryImageURL); err != nil {
			return nil, err
		}
		data = append(data, p)
	}
	return data, rows.Err()
}

// Search runs the hybrid query. When qvec is nil (embedder call failed) the semantic
// term is omitted entirely to avoid passing a zero-vector which would produce NaN via
// the undefined cosine distance and cause nondeterministic ORDER BY results.
//
// To detect whether a next page exists without an expensive COUNT(*), Search fetches
// limit+1 rows. If more than limit rows are returned, HasMore is set to true and the
// extra row is trimmed before returning. TotalEstimate is a lower-bound: offset +
// len(data) + 1 when HasMore, or offset + len(data) otherwise.
func (s *Searcher) Search(ctx context.Context, qvec []float32, q string, limit, offset int) (SearchResponse, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// Fetch one extra row to determine whether a next page exists.
	fetchLimit := limit + 1

	var (
		data []SearchProduct
		err  error
	)

	if qvec == nil {
		var rows pgx.Rows
		rows, err = s.pool.Query(ctx, hybridSQLKeywordOnly,
			q, s.weights.Keyword, s.weights.Trigram, fetchLimit, offset,
		)
		if err != nil {
			return SearchResponse{}, fmt.Errorf("search query (keyword-only): %w", err)
		}
		data, err = scanRows(rows)
	} else {
		var rows pgx.Rows
		rows, err = s.pool.Query(ctx, hybridSQLWithSemantic,
			pgvector.NewVector(qvec), q,
			s.weights.Semantic, s.weights.Keyword, s.weights.Trigram,
			fetchLimit, offset, semanticThreshold,
		)
		if err != nil {
			return SearchResponse{}, fmt.Errorf("search query (hybrid): %w", err)
		}
		data, err = scanRows(rows)
	}

	if err != nil {
		return SearchResponse{}, err
	}

	// Trim the sentinel row and derive HasMore / TotalEstimate.
	hasMore := len(data) > limit
	if hasMore {
		data = data[:limit]
	}
	totalEstimate := offset + len(data)
	if hasMore {
		totalEstimate++
	}

	return SearchResponse{
		Success: true,
		Data:    data,
		Meta: SearchMeta{
			Limit:         limit,
			Offset:        offset,
			TotalEstimate: totalEstimate,
			HasMore:       hasMore,
		},
	}, nil
}
