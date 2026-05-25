package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	emb "github.com/handloom/admin/cmd/embedder/embedder"
)

type deps struct {
	onnx        *emb.ONNXSession
	searcher    *emb.Searcher
	hmac        *emb.HMACVerifier
	rl          *emb.IPRateLimiter
	allowOrigin string
	startedAt   time.Time
}

func newRouter(d *deps) http.Handler {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{d.allowOrigin},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.With(d.rl.Middleware).Get("/ping", d.handlePing)
	r.With(d.rl.Middleware).Post("/search", d.handleSearch)
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
	var req emb.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if l := len(req.Query); l == 0 || l > 256 {
		http.Error(w, "q: 1..256 chars required", http.StatusBadRequest)
		return
	}

	// Query embedding (failure is non-fatal — semantic term zeros out in SQL).
	var qvec []float32
	if vecs, err := d.onnx.Embed([]string{req.Query}); err == nil && len(vecs) == 1 {
		qvec = vecs[0]
	}

	resp, err := d.searcher.Search(r.Context(), qvec, req.Query, req.Limit, req.Offset)
	if err != nil {
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
