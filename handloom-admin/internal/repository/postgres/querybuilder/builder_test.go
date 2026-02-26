package querybuilder_test

import (
	"testing"

	"github.com/handloom/admin/internal/repository/postgres/querybuilder"
)

func TestSelectFrom(t *testing.T) {
	sql, args := querybuilder.Select("id", "name").From("users").Build()
	wantSQL := "SELECT id, name FROM users"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 0 {
		t.Errorf("args = %v, want empty", args)
	}
}

func TestWithFilter(t *testing.T) {
	sql, args := querybuilder.Select("id").From("products p").
		WithFilter(true, "p.status", "ACTIVE").
		WithFilter(false, "p.category_id", "skip-me").
		Build()

	wantSQL := "SELECT id FROM products p WHERE p.status = $1"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 1 || args[0] != "ACTIVE" {
		t.Errorf("args = %v, want [ACTIVE]", args)
	}
}

func TestWithLike(t *testing.T) {
	sql, args := querybuilder.Select("id").From("products p").
		WithLike(true, "p.name", "%silk%").
		Build()

	wantSQL := "SELECT id FROM products p WHERE p.name ILIKE $1"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 1 || args[0] != "%silk%" {
		t.Errorf("args = %v, want [%%silk%%]", args)
	}
}

func TestWithRange(t *testing.T) {
	min := int64(100)
	max := int64(500)

	sql, args := querybuilder.Select("id").From("products p").
		WithRange("p.price", &min, &max).
		Build()

	wantSQL := "SELECT id FROM products p WHERE p.price >= $1 AND p.price <= $2"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 2 || args[0] != int64(100) || args[1] != int64(500) {
		t.Errorf("args = %v, want [100 500]", args)
	}
}

func TestWithRangeMinOnly(t *testing.T) {
	min := int64(100)
	sql, args := querybuilder.Select("id").From("t").
		WithRange("t.price", &min, nil).
		Build()

	wantSQL := "SELECT id FROM t WHERE t.price >= $1"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 1 {
		t.Errorf("args = %v, want [100]", args)
	}
}

func TestWithRaw(t *testing.T) {
	sql, args := querybuilder.Select("id").From("products p").
		WithRaw(true, "EXISTS (SELECT 1 FROM attrs v WHERE v.pid = p.id AND v.name = %s AND v.val = ANY(%s))", "color", []string{"red", "blue"}).
		Build()

	wantSQL := "SELECT id FROM products p WHERE EXISTS (SELECT 1 FROM attrs v WHERE v.pid = p.id AND v.name = $1 AND v.val = ANY($2))"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 2 || args[0] != "color" {
		t.Errorf("args = %v, want [color [red blue]]", args)
	}
}

func TestWithRawSkipped(t *testing.T) {
	sql, args := querybuilder.Select("id").From("t").
		WithRaw(false, "x = %s", "skip").
		Build()

	if sql != "SELECT id FROM t" {
		t.Errorf("sql = %q, want no WHERE", sql)
	}
	if len(args) != 0 {
		t.Errorf("args should be empty, got %v", args)
	}
}

func TestLeftJoin(t *testing.T) {
	sql, _ := querybuilder.Select("p.id").From("products p").
		LeftJoin("inventory i", "i.product_id = p.id").
		Build()

	want := "SELECT p.id FROM products p LEFT JOIN inventory i ON i.product_id = p.id"
	if sql != want {
		t.Errorf("sql = %q, want %q", sql, want)
	}
}

func TestOrderByLimitOffset(t *testing.T) {
	sql, args := querybuilder.Select("id").From("t").
		OrderBy("id DESC").
		Limit(20).
		Offset(40).
		Build()

	want := "SELECT id FROM t ORDER BY id DESC LIMIT $1 OFFSET $2"
	if sql != want {
		t.Errorf("sql = %q, want %q", sql, want)
	}
	if len(args) != 2 || args[0] != 20 || args[1] != 40 {
		t.Errorf("args = %v, want [20 40]", args)
	}
}

func TestPlaceholderNumbering(t *testing.T) {
	min := int64(100)
	sql, args := querybuilder.Select("id").From("products p").
		WithFilter(true, "p.status", "ACTIVE").
		WithLike(true, "p.name", "%silk%").
		WithRange("p.price", &min, nil).
		WithRaw(true, "p.category_id = ANY(%s)", []string{"a", "b"}).
		OrderBy("p.id").
		Limit(21).
		Offset(0).
		Build()

	wantSQL := "SELECT id FROM products p WHERE p.status = $1 AND p.name ILIKE $2 AND p.price >= $3 AND p.category_id = ANY($4) ORDER BY p.id LIMIT $5"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 5 {
		t.Errorf("args count = %d, want 5", len(args))
	}
}

func TestNoFilters(t *testing.T) {
	sql, args := querybuilder.Select("id", "name").From("t").
		WithFilter(false, "col", "val").
		WithLike(false, "col", "val").
		WithRange("col", nil, nil).
		WithRaw(false, "col = %s", "val").
		OrderBy("id").
		Limit(10).
		Build()

	want := "SELECT id, name FROM t ORDER BY id LIMIT $1"
	if sql != want {
		t.Errorf("sql = %q, want %q", sql, want)
	}
	if len(args) != 1 {
		t.Errorf("args = %v, want [10]", args)
	}
}

func TestWhere(t *testing.T) {
	sql, args := querybuilder.Select("id", "name").From("products").
		Where("id", "abc-123").
		Build()

	wantSQL := "SELECT id, name FROM products WHERE id = $1"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 1 || args[0] != "abc-123" {
		t.Errorf("args = %v, want [abc-123]", args)
	}
}

func TestWhereIn(t *testing.T) {
	ids := []string{"a", "b", "c"}
	sql, args := querybuilder.Select("id").From("products").
		WhereIn("id", ids).
		Build()

	wantSQL := "SELECT id FROM products WHERE id = ANY($1)"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 1 {
		t.Errorf("args = %v, want [ids slice]", args)
	}
}
