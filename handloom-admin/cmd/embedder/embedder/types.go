// Package embedder hosts the search + inference service that runs inside the
// embedder Lambda container.
package embedder

import "time"

// EmbedRequest is the JSON body for POST /embed.
type EmbedRequest struct {
	Texts []string `json:"texts"`
}

// EmbedResponse is the JSON body returned by POST /embed.
type EmbedResponse struct {
	Vectors [][]float32 `json:"vectors"`
	Model   string      `json:"model"`
}

// SearchRequest holds the parsed query-string parameters for GET /search.
// The handler builds this from r.URL.Query(); not deserialized from JSON body.
type SearchRequest struct {
	Query            string              // q
	Limit            int                 // limit
	Cursor           string              // cursor (base64-encoded offset)
	CategoryID       string              // category_id
	MinPrice         *int64              // min_price (paise)
	MaxPrice         *int64              // max_price (paise)
	InStockOnly      bool                // in_stock=true
	Material         string              // material
	Color            string              // color
	AttributeFilters map[string][]string // af_<name>=v1,v2
}

// ProductImage mirrors domain.ProductImage JSON tags.
type ProductImage struct {
	URL       string `json:"url"`
	AltText   string `json:"alt_text"`
	IsPrimary bool   `json:"is_primary"`
	SortOrder int    `json:"sort_order"`
}

// Dimensions mirrors domain.Dimensions JSON tags.
type Dimensions struct {
	Length float64 `json:"length"`
	Width  float64 `json:"width"`
	Height float64 `json:"height,omitempty"`
	Unit   string  `json:"unit"`
}

// StoreProduct is the public-facing product response that matches the legacy
// /api/v1/store/catalog/products response shape exactly.
type StoreProduct struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	SKU         string `json:"sku"`
	Description string `json:"description,omitempty"`

	// Relations
	CategoryID string `json:"category_id"`

	// Pricing (in paise) — CostPrice intentionally excluded
	BasePrice    int64  `json:"base_price"`
	SellingPrice int64  `json:"selling_price"`
	Currency     string `json:"currency"`

	// Dimensions
	Dimensions *Dimensions `json:"dimensions,omitempty"`
	Weight     int         `json:"weight,omitempty"`

	// Custom Dimension Support
	AllowCustomDimensions bool    `json:"allow_custom_dimensions"`
	PricingRuleID         *string `json:"pricing_rule_id,omitempty"`

	// Attributes
	Attributes map[string]interface{} `json:"attributes,omitempty"`

	// Common Attributes
	Material  string `json:"material,omitempty"`
	Color     string `json:"color,omitempty"`
	WeaveType string `json:"weave_type,omitempty"`

	// Provenance
	Origin    string `json:"origin,omitempty"`
	CraftType string `json:"craft_type,omitempty"`

	// Media
	Images         []ProductImage `json:"images,omitempty"`
	VideoURL       string         `json:"video_url,omitempty"`
	VideoPosterURL string         `json:"video_poster_url,omitempty"`

	// Tags & SEO
	Tags []string `json:"tags,omitempty"`

	// Stock — public boolean instead of raw counts
	InStock bool `json:"in_stock"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// paginationMeta carries pagination context matching the legacy endpoint envelope.
type paginationMeta struct {
	Limit      int    `json:"limit"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// SearchResponse mirrors the legacy /api/v1/store/catalog/products envelope so
// storefront consumers only swap URL, not body shape.
type SearchResponse struct {
	Success bool            `json:"success"`
	Data    []*StoreProduct `json:"data"`
	Meta    paginationMeta  `json:"meta"`
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
