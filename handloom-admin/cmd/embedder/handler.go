package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	emb "github.com/handloom/admin/cmd/embedder/embedder"
	"github.com/handloom/admin/internal/middleware"
	"github.com/handloom/admin/internal/router"
	"github.com/handloom/admin/pkg/metrics"
)

type deps struct {
	onnx        *emb.ONNXSession
	searcher    *emb.Searcher
	classifier  *emb.Classifier
	hmac        *emb.HMACVerifier
	rl          *emb.IPRateLimiter
	allowOrigin string
	startedAt   time.Time
}

func newRouter(d *deps) http.Handler {
	r := chi.NewRouter()

	// Shared observability stack: server tracing, request ID, metrics buffer
	// (the metrics.Record() calls in handleSearch silently drop without it),
	// geo extraction, request logs, panic recovery, real-IP. Same stack the
	// other Lambdas get via router.NewBaseRouter.
	router.ApplyObservability(r)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{d.allowOrigin},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		// Accept is added by browsers automatically; without listing it the
		// preflight rejects the actual POST. Authorization left out — the
		// embedder uses HMAC headers via lambda.Invoke, not browser auth.
		AllowedHeaders:   []string{"Content-Type", "Accept"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Public endpoints — mounted at the absolute paths that API Gateway forwards
	// (REST API proxy integration does not strip the prefix). /search is GET so
	// the URL is bookmarkable, browser back/forward preserves state, and CDN
	// caching can apply if ever enabled.
	r.Route("/api/v1/store/catalog", func(r chi.Router) {
		r.With(d.rl.Middleware).Get("/embedder-ping", d.handlePing)
		r.With(d.rl.Middleware).Get("/search", d.handleSearch)
	})

	// Internal endpoint — invoked via lambda.Invoke SDK (catalog + backfill
	// Lambdas) with a hand-built APIGW v1 payload pointing at "/embed".
	r.With(d.hmac.Middleware).Post("/embed", d.handleEmbed)

	return r
}

func (d *deps) handlePing(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, emb.PingResponse{
		OK:             true,
		Warm:           true,
		ContainerAgeMs: time.Since(d.startedAt).Milliseconds(),
	})
}

func (d *deps) handleEmbed(w http.ResponseWriter, r *http.Request) {
	var req emb.EmbedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if len(req.Texts) == 0 || len(req.Texts) > 32 {
		http.Error(w, "texts: 1..32 entries required", http.StatusBadRequest)
		return
	}
	vecs, err := d.onnx.Embed(req.Texts)
	if err != nil {
		http.Error(w, "embed failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, emb.EmbedResponse{Vectors: vecs, Model: emb.ModelVersion})
}

func (d *deps) handleSearch(w http.ResponseWriter, r *http.Request) {
	req := parseSearchRequest(r)

	// Validate: allow empty q (filter-only listing); cap length to keep payloads sane.
	if len(req.Query) > 256 {
		http.Error(w, "q: max 256 chars", http.StatusBadRequest)
		return
	}

	// Query embedding (failure is non-fatal — semantic term collapses to 0 in SQL).
	// Skip when q is empty since semantic ranking has no meaning for a pure listing.
	var qvec []float32
	if req.Query != "" {
		vecs, err := d.onnx.Embed([]string{req.Query})
		if err != nil {
			slog.ErrorContext(r.Context(), "embed query failed; falling back to keyword-only",
				"q", req.Query, "err", err)
		} else if len(vecs) != 1 {
			slog.ErrorContext(r.Context(), "embed returned unexpected vector count",
				"q", req.Query, "got", len(vecs))
		} else {
			qvec = vecs[0]
		}
	}
	slog.InfoContext(r.Context(), "search request",
		"q", req.Query, "limit", req.Limit, "qvec_present", qvec != nil,
		"category_id", req.CategoryID, "in_stock_only", req.InStockOnly)

	offset := emb.DecodeCursor(req.Cursor)

	resp, err := d.searcher.Search(r.Context(), qvec, req, offset)
	if err != nil {
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}

	if strings.TrimSpace(req.Query) != "" {
		hasResults := len(resp.Data) > 0
		// Zero-shot intent classification reuses the embedding we just
		// computed for semantic search — no extra ONNX invocation. Empty
		// label space stays bounded: ~7 intents × ~6 categories + a few
		// "*_other" / "direct_*" + "unknown" → ~50 distinct values total.
		intent := "unknown"
		if qvec != nil {
			intent = d.classifier.Classify(qvec)
		}
		metrics.Record(r.Context(), "search_query", metrics.L{
			metrics.LabelHasResults: fmt.Sprintf("%t", hasResults),
			metrics.LabelCountry:    middleware.GetCountry(r.Context()),
			metrics.LabelIntent:     intent,
		})
		slog.InfoContext(r.Context(), "store.search",
			slog.String("query", req.Query),
			slog.Bool("has_results", hasResults),
			slog.Int("result_count", len(resp.Data)),
			slog.String("country", middleware.GetCountry(r.Context())),
			slog.String("city", middleware.GetCity(r.Context())),
		)
	}

	writeJSON(w, http.StatusOK, resp)
}

// parseSearchRequest builds a SearchRequest from query-string params:
//
//	q, limit, cursor, category_id, min_price, max_price, in_stock,
//	material, color, af_<name>=val1,val2
func parseSearchRequest(r *http.Request) emb.SearchRequest {
	q := r.URL.Query()
	req := emb.SearchRequest{
		Query:      q.Get("q"),
		Cursor:     q.Get("cursor"),
		CategoryID: q.Get("category_id"),
		Material:   q.Get("material"),
		Color:      q.Get("color"),
	}
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 {
		req.Limit = n
	}
	if v := q.Get("min_price"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			req.MinPrice = &n
		}
	}
	if v := q.Get("max_price"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			req.MaxPrice = &n
		}
	}
	req.InStockOnly = q.Get("in_stock") == "true"

	// af_<name>=v1,v2 — same convention as the legacy /products handler.
	attrs := make(map[string][]string)
	for key, values := range q {
		if !strings.HasPrefix(key, "af_") {
			continue
		}
		name := strings.TrimPrefix(key, "af_")
		for _, v := range values {
			for _, sv := range strings.Split(v, ",") {
				if sv = strings.TrimSpace(sv); sv != "" {
					attrs[name] = append(attrs[name], sv)
				}
			}
		}
	}
	if len(attrs) > 0 {
		req.AttributeFilters = attrs
	}
	return req
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
