package querybuilder_test

import (
	"testing"

	"github.com/handloom/admin/internal/repository/postgres/querybuilder"
)

func TestBatchInsertSingleRow(t *testing.T) {
	sql, args := querybuilder.InsertBatch("attrs", "product_id", "name", "value").
		AddRow("p1", "color", "red").
		Build()

	wantSQL := "INSERT INTO attrs (product_id, name, value) VALUES ($1, $2, $3)"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 3 || args[0] != "p1" || args[1] != "color" || args[2] != "red" {
		t.Errorf("args = %v, want [p1 color red]", args)
	}
}

func TestBatchInsertMultipleRows(t *testing.T) {
	sql, args := querybuilder.InsertBatch("attrs", "pid", "name", "val").
		AddRow("p1", "color", "red").
		AddRow("p1", "size", "large").
		Build()

	wantSQL := "INSERT INTO attrs (pid, name, val) VALUES ($1, $2, $3), ($4, $5, $6)"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 6 {
		t.Errorf("args count = %d, want 6", len(args))
	}
}

func TestBatchInsertOnConflict(t *testing.T) {
	sql, _ := querybuilder.InsertBatch("attrs", "pid", "name").
		AddRow("p1", "color").
		OnConflictDoNothing().
		Build()

	wantSQL := "INSERT INTO attrs (pid, name) VALUES ($1, $2) ON CONFLICT DO NOTHING"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
}

func TestBatchInsertNoRows(t *testing.T) {
	sql, args := querybuilder.InsertBatch("attrs", "pid", "name").Build()

	if sql != "" {
		t.Errorf("sql = %q, want empty", sql)
	}
	if args != nil {
		t.Errorf("args = %v, want nil", args)
	}
}
