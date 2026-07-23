package main

import "testing"

func TestUpdateSetClause(t *testing.T) {
	cols := []column{{"id", "text"}, {"name", "text"}, {"tags", "text[]"}}
	got := updateSetClause(cols, "id")
	want := `"name" = EXCLUDED."name", "tags" = EXCLUDED."tags"`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRewriteURLs(t *testing.T) {
	cols := []column{{"id", "text"}, {"url", "text"}}
	rows := [][]any{
		{"p1", "https://dev.cdn/assets/a.jpg"},
		{"p2", nil}, // NULL url stays NULL
	}
	rewriteURLs(rows, cols, []string{"url"}, [][2]string{{"https://dev.cdn/", "https://prod.cdn/"}})
	if rows[0][1] != "https://prod.cdn/assets/a.jpg" {
		t.Errorf("url not rewritten: %v", rows[0][1])
	}
	if rows[1][1] != nil {
		t.Errorf("nil url mutated: %v", rows[1][1])
	}
}

func TestProductFilter(t *testing.T) {
	where, args := productFilter("active")
	if where != "status = 'ACTIVE'" || args != nil {
		t.Errorf("active: got %q %v", where, args)
	}
	where, args = productFilter("id-1,id_2")
	if where != "id = ANY($1)" || len(args) != 1 {
		t.Errorf("ids: got %q %v", where, args)
	}
}
