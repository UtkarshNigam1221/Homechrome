// Package embedder hosts the search + inference service that runs inside the
// embedder Lambda container.
package embedder

// EmbedRequest is the JSON body for POST /embed.
type EmbedRequest struct {
	Texts []string `json:"texts"`
}

// EmbedResponse is the JSON body returned by POST /embed.
type EmbedResponse struct {
	Vectors [][]float32 `json:"vectors"`
	Model   string      `json:"model"`
}

// SearchRequest is the JSON body for POST /search.
type SearchRequest struct {
	Query  string `json:"q"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// SearchProduct is one row in the search response data array.
type SearchProduct struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	PricePaise      int64  `json:"price_paise"`
	PrimaryImageURL string `json:"primary_image_url"`
}

// SearchResponse mirrors the legacy keyword-search Lambda envelope so the
// storefront only swaps URL, not body shape.
type SearchResponse struct {
	Success bool            `json:"success"`
	Data    []SearchProduct `json:"data"`
	Meta    SearchMeta      `json:"meta"`
}

// SearchMeta carries pagination context back to the client.
// TotalEstimate is a lower-bound estimate: offset + len(data) + 1 when HasMore is
// true, or offset + len(data) when it is false. It is suitable for "page X of ?"
// UIs but should not be treated as an exact count.
// HasMore is true when there are more results beyond the current page.
type SearchMeta struct {
	Limit         int  `json:"limit"`
	Offset        int  `json:"offset"`
	TotalEstimate int  `json:"total_estimate"`
	HasMore       bool `json:"has_more"`
}

// PingResponse is the JSON body returned by GET /ping. No state, only warmth.
type PingResponse struct {
	OK             bool  `json:"ok"`
	Warm           bool  `json:"warm"`
	ContainerAgeMs int64 `json:"container_age_ms"`
}

// ModelVersion is stamped into rows + responses so a future model swap can be
// detected and back-filled.
const ModelVersion = "l3cube-indic-sbert-nli-v1"

// EmbeddingDim must equal the model output dim and the pg vector(N) column.
const EmbeddingDim = 768
