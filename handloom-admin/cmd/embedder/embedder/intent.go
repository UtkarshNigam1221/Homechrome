package embedder

import (
	"fmt"
	"math"
)

// intentPrototypes describe what the visitor is searching FOR — the
// modifier on a product type. Each entry is a small bag of synonyms;
// the sentence-transformer model maps semantically similar queries
// near the prototype vector.
var intentPrototypes = map[string]string{
	"material": "silk cotton linen wool jute fabric weave",
	"color":    "red blue green yellow pink black white maroon ivory color shade",
	"style":    "jaipuri south patoola printed panipat gujrati embroidery patchwork handloom traditional",
	"occasion": "wedding festival diwali pooja casual everyday gift ceremony heavy",
	"price":    "cheap expensive budget affordable under below over premium",
}

// intentPrecedence picks ONE intent per query when multiple axes match.
// Listed highest-priority first. Material wins because silk-vs-cotton is
// the biggest price + quality + SKU-narrowing lever for a handloom shop.
// Reorder + redeploy when business priorities shift.
var intentPrecedence = []string{
	"material",
	"style",
	"occasion",
	"color",
	"price",
}

// categoryPrototypes describe the PRODUCT TYPE the visitor is browsing.
// Hand-curated rather than autoloaded from PG so cardinality of the
// combined label stays predictable across catalog changes.
var categoryPrototypes = map[string]string{
	"bedsheet":     "bedsheet bedspread bedding bed-cover quilt",
	"pillow-cover": "pillow-cover pillowcase cushion-cover pillow-sham bolster-cover",
	"curtain":      "curtain drape window-curtain door-curtain sheer panel valance",
	"dohar":        "dohar dohar-blanket summer-quilt light-blanket razai comforter throw",
	"dhurrie":      "dhurrie rug floor-rug carpet kilim runner-rug durrie",
	"table-linen":  "table-cloth table-runner placemat napkin table-mat dining-linen",
	"towel":        "towel bath-towel hand-towel hamam gamcha kitchen-towel",
	"blanket":      "blanket throw woolen-blanket ac-blanket fleece",
	"diwan-set":    "diwan-set diwan-cover bolster-set divan single-bedcover",
}

// Classifier maps a query embedding to a combined "<intent>_<category>"
// bucket. Both prototype sets are embedded once at construction; per-
// query classification is then O(N) cosine products on float32 vectors
// (microseconds for tens of prototypes).
type Classifier struct {
	intentVecs    map[string][]float32
	categoryNames []string
	categoryVecs  [][]float32
	threshold     float32
}

// NewClassifier constructs a Classifier and warms the prototype vectors
// from the ONNX session. Threshold is the minimum cosine similarity
// below which a prototype isn't considered a hit. 0.45 is a reasonable
// starting point for all-MiniLM-style embeddings; tune via the dashboard
// as real queries arrive.
func NewClassifier(onnx *ONNXSession, threshold float32) (*Classifier, error) {
	c := &Classifier{threshold: threshold, intentVecs: make(map[string][]float32, len(intentPrototypes))}

	for name, proto := range intentPrototypes {
		vec, err := embedOne(onnx, proto)
		if err != nil {
			return nil, fmt.Errorf("embed intent %q: %w", name, err)
		}
		c.intentVecs[name] = vec
	}
	for name, proto := range categoryPrototypes {
		vec, err := embedOne(onnx, proto)
		if err != nil {
			return nil, fmt.Errorf("embed category %q: %w", name, err)
		}
		c.categoryNames = append(c.categoryNames, name)
		c.categoryVecs = append(c.categoryVecs, vec)
	}
	return c, nil
}

// Classify takes a query vector (the same vector the semantic search
// already uses) and returns a single combined label like "material_saree".
// Precedence-wins: walk intentPrecedence in order, pick the first axis
// whose cosine similarity crosses the threshold. Keeps the label space
// dense (~N × M buckets, no combinatorial blow-up) so low-traffic
// dashboards stay readable.
//
// Forms returned:
//   - "material_saree"   highest-precedence intent that hits + category
//   - "direct_saree"     category match only (no intent qualifier)
//   - "material_other"   intent hits but no strong category match
//   - "unknown"          nothing crosses the threshold
//
// Multi-axis queries: a "red silk saree" lights up both material and
// color, but precedence picks material → "material_saree". Multi-axis
// drill-down is available via the raw query string in Loki
// (slog.InfoContext("store.search", query=...)), not surfaced on PG.
func (c *Classifier) Classify(qvec []float32) string {
	if c == nil || len(qvec) == 0 {
		return "unknown"
	}

	intent := c.precedenceHit(qvec)
	category, categorySim := argmaxCosine(qvec, c.categoryNames, c.categoryVecs)
	categoryHit := categorySim >= c.threshold

	switch {
	case intent != "" && categoryHit:
		return intent + "_" + category
	case categoryHit:
		return "direct_" + category
	case intent != "":
		return intent + "_other"
	default:
		return "unknown"
	}
}

// precedenceHit walks intentPrecedence in declared order and returns
// the first intent name whose cosine similarity with qvec meets the
// threshold. Returns "" if no intent crosses. Order is the lever — edit
// intentPrecedence and redeploy to change which axis "wins" multi-
// signal queries.
func (c *Classifier) precedenceHit(qvec []float32) string {
	for _, name := range intentPrecedence {
		v, ok := c.intentVecs[name]
		if !ok {
			continue // prototype missing — shouldn't happen, but stay safe
		}
		if cosine(qvec, v) >= c.threshold {
			return name
		}
	}
	return ""
}

func argmaxCosine(qvec []float32, names []string, vecs [][]float32) (string, float32) {
	var bestName string
	var bestSim float32 = -1
	for i, v := range vecs {
		sim := cosine(qvec, v)
		if sim > bestSim {
			bestSim = sim
			bestName = names[i]
		}
	}
	return bestName, bestSim
}

func cosine(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		af, bf := float64(a[i]), float64(b[i])
		dot += af * bf
		na += af * af
		nb += bf * bf
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(na) * math.Sqrt(nb)))
}

func embedOne(onnx *ONNXSession, text string) ([]float32, error) {
	vecs, err := onnx.Embed([]string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) != 1 {
		return nil, fmt.Errorf("embed returned %d vectors, want 1", len(vecs))
	}
	return vecs[0], nil
}
