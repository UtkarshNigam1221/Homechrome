package querybuilder_test

import (
	"testing"

	"github.com/handloom/admin/internal/repository/postgres/querybuilder"
)

func TestUpdateBasic(t *testing.T) {
	sql, args := querybuilder.Update("products").
		Set("name", "New Name").
		Set("status", "ACTIVE").
		Where("id", "p1").
		Build()

	wantSQL := "UPDATE products SET name = $1, status = $2 WHERE id = $3"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 3 || args[0] != "New Name" || args[1] != "ACTIVE" || args[2] != "p1" {
		t.Errorf("args = %v, want [New Name ACTIVE p1]", args)
	}
}

func TestUpdateSetRawWithArg(t *testing.T) {
	sql, args := querybuilder.Update("categories").
		SetRaw("product_count", "product_count + %s", 1).
		SetRaw("updated_at", "NOW()").
		Where("id", "c1").
		Build()

	wantSQL := "UPDATE categories SET product_count = product_count + $1, updated_at = NOW() WHERE id = $2"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 2 || args[0] != 1 || args[1] != "c1" {
		t.Errorf("args = %v, want [1 c1]", args)
	}
}

func TestUpdateManyColumns(t *testing.T) {
	sql, args := querybuilder.Update("inventory").
		Set("quantity", 10).
		Set("reserved_qty", 2).
		Set("available_qty", 8).
		Set("updated_at", "now").
		Where("product_id", "p1").
		Build()

	wantSQL := "UPDATE inventory SET quantity = $1, reserved_qty = $2, available_qty = $3, updated_at = $4 WHERE product_id = $5"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 5 {
		t.Errorf("args count = %d, want 5", len(args))
	}
}

func TestUpdatePlaceholderSequence(t *testing.T) {
	_, args := querybuilder.Update("t").
		Set("a", 1).
		Set("b", 2).
		Set("c", 3).
		SetRaw("d", "d + %s", 4).
		Where("id", 5).
		Build()

	if len(args) != 5 {
		t.Errorf("args count = %d, want 5", len(args))
	}
	for i, want := range []interface{}{1, 2, 3, 4, 5} {
		if args[i] != want {
			t.Errorf("args[%d] = %v, want %v", i, args[i], want)
		}
	}
}
