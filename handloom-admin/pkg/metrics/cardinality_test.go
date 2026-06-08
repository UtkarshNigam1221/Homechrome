package metrics_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// AllowedLabels is the set of label keys permitted in metrics.L{...} literals.
// Adding a label here is a deliberate decision — it goes into permanent PG
// storage and contributes to row cardinality. Anything not on this list will
// fail TestNoUnknownLabelsInMetricsRecord.
var AllowedLabels = map[string]struct{}{
	// Geo
	"city": {}, "country": {},

	// Device / session
	"device_type": {},
	"is_new_user": {},

	// Marketing attribution (storefront sends from URL ?utm_*)
	"utm_source":   {},
	"utm_medium":   {},
	"utm_campaign": {},

	// Funnel / business
	"has_results": {},
	"product_id":  {}, "category_id": {},
	"coupon_code": {},
	"outcome":     {},
	"gateway":     {},

	// HTTP / network
	"method": {}, "route": {}, "status_class": {},
	"service":     {},
	"target_host": {},

	// DB / AWS
	"operation":   {},
	"sdk_service": {},
	"status":      {},

	// RUM / page analytics
	"page_type":  {},
	"error_type": {},

	// Search intent classification (zero-shot via embedder model). Values
	// are combined "<sorted-intents>_<category>" like "color+material_saree";
	// label NAME is just "intent".
	"intent": {},

	// Catalog filter beacons
	"filter_key": {}, "filter_value": {},

	// Generic
	"reason": {}, "bucket": {},
}

func TestNoUnknownLabelsInMetricsRecord(t *testing.T) {
	var violations []string

	root := "../.."
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == "vendor" || base == "cdk.out" || base == ".git" || base == "node_modules" || base == "bin" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		// Skip the metrics package itself (its tests use arbitrary keys).
		if strings.Contains(filepath.ToSlash(path), "/pkg/metrics/") {
			return nil
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil //nolint:nilerr // skip files that don't parse; not a walk error
		}

		ast.Inspect(node, func(n ast.Node) bool {
			comp, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			sel, ok := comp.Type.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if sel.Sel.Name != "L" {
				return true
			}
			pkgIdent, _ := sel.X.(*ast.Ident)
			if pkgIdent == nil || pkgIdent.Name != "metrics" {
				return true
			}

			for _, elt := range comp.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.BasicLit)
				if !ok {
					continue
				}
				keyStr := strings.Trim(key.Value, `"`)
				if _, allowed := AllowedLabels[keyStr]; !allowed {
					pos := fset.Position(kv.Pos())
					violations = append(violations,
						pos.String()+": forbidden label key: "+keyStr)
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	assert.Empty(t, violations, "Unknown label keys used in metrics.L{...}")
}
