package querybuilder_test

import (
	"testing"

	"github.com/handloom/admin/internal/repository/postgres/querybuilder"
)

func TestInsertSingle(t *testing.T) {
	sql, args := querybuilder.Insert("users").
		Set("id", "u1").
		Set("name", "Alice").
		Set("email", "alice@example.com").
		Build()

	wantSQL := "INSERT INTO users (id, name, email) VALUES ($1, $2, $3)"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 3 || args[0] != "u1" || args[1] != "Alice" || args[2] != "alice@example.com" {
		t.Errorf("args = %v, want [u1 Alice alice@example.com]", args)
	}
}

func TestInsertManyColumns(t *testing.T) {
	sql, args := querybuilder.Insert("products").
		Set("id", "p1").
		Set("name", "Silk Saree").
		Set("price", int64(5000)).
		Set("status", "ACTIVE").
		Build()

	wantSQL := "INSERT INTO products (id, name, price, status) VALUES ($1, $2, $3, $4)"
	if sql != wantSQL {
		t.Errorf("sql = %q, want %q", sql, wantSQL)
	}
	if len(args) != 4 {
		t.Errorf("args count = %d, want 4", len(args))
	}
}
