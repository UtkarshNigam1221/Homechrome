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
//
// Keys may appear as string literals OR as string constants (e.g. labelCountry
// = "country"); the guard resolves constant identifiers to their value before
// checking, so const-ifying a key for goconst does not bypass the check.
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

	// Drift guard: event_type on unmapped_store_event. Bounded by the
	// store-event allow-list, so cardinality is safe.
	"event_type": {},
}

// walkGoFiles invokes fn for every non-test .go file under root, skipping
// vendored / generated / pkg-metrics paths (pkg/metrics keys aren't governed).
func walkGoFiles(t *testing.T, root string, skipMetrics bool, fn func(path string, file *ast.File, fset *token.FileSet)) {
	t.Helper()
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
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if skipMetrics && strings.Contains(filepath.ToSlash(path), "/pkg/metrics/") {
			return nil
		}
		fset := token.NewFileSet()
		file, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return nil //nolint:nilerr // skip files that don't parse; not a walk error
		}
		fn(path, file, fset)
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
}

// collectStringConsts returns a map of const name -> string value for every
// string-valued constant declared in the scanned files. Used to resolve
// constant-identifier label keys back to their underlying string.
func collectStringConsts(t *testing.T, root string) map[string]string {
	consts := make(map[string]string)
	// skipMetrics=false: collect from pkg/metrics too, so the shared
	// metrics.Label* vocabulary is resolvable when used elsewhere.
	walkGoFiles(t, root, false, func(_ string, file *ast.File, _ *token.FileSet) {
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.CONST {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, name := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					if lit, ok := vs.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						consts[name.Name] = strings.Trim(lit.Value, "`\"")
					}
				}
			}
		}
	})
	return consts
}

func TestNoUnknownLabelsInMetricsRecord(t *testing.T) {
	root := "../.."
	consts := collectStringConsts(t, root)

	// resolveKey returns (value, resolvable). A key is resolvable if it is a
	// string literal or a (possibly package-qualified) string constant.
	resolveKey := func(expr ast.Expr) (string, bool) {
		switch k := expr.(type) {
		case *ast.BasicLit:
			if k.Kind == token.STRING {
				return strings.Trim(k.Value, "`\""), true
			}
		case *ast.Ident:
			if v, ok := consts[k.Name]; ok {
				return v, true
			}
		case *ast.SelectorExpr:
			if v, ok := consts[k.Sel.Name]; ok {
				return v, true
			}
		}
		return "", false
	}

	var violations []string
	// skipMetrics=true: pkg/metrics label keys aren't governed by the allowlist.
	walkGoFiles(t, root, true, func(_ string, file *ast.File, fset *token.FileSet) {
		ast.Inspect(file, func(n ast.Node) bool {
			comp, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			sel, ok := comp.Type.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "L" {
				return true
			}
			if pkgIdent, ok := sel.X.(*ast.Ident); !ok || pkgIdent.Name != "metrics" {
				return true
			}

			for _, elt := range comp.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				pos := fset.Position(kv.Pos()).String()
				val, resolvable := resolveKey(kv.Key)
				if !resolvable {
					violations = append(violations, pos+": label key is not a literal or string constant")
					continue
				}
				if _, allowed := AllowedLabels[val]; !allowed {
					violations = append(violations, pos+": forbidden label key: "+val)
				}
			}
			return true
		})
	})
	assert.Empty(t, violations, "Unknown label keys used in metrics.L{...}")
}
