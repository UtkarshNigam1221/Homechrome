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
	cols := []column{{"id", "text"}, {colURL, "text"}}
	rows := [][]any{
		{"p1", "https://dev.cdn/assets/a.jpg"},
		{"p2", nil}, // NULL url stays NULL
	}
	rewriteURLs(rows, cols, []string{colURL}, [][2]string{{"https://dev.cdn/", "https://prod.cdn/"}})
	if rows[0][1] != "https://prod.cdn/assets/a.jpg" {
		t.Errorf("url not rewritten: %v", rows[0][1])
	}
	if rows[1][1] != nil {
		t.Errorf("nil url mutated: %v", rows[1][1])
	}
}

func TestProductFilter(t *testing.T) {
	where, args, ids := productFilter("active")
	if where != whereActive || args != nil || ids != nil {
		t.Errorf("active: got %q %v %v", where, args, ids)
	}
	// Keyword matching is case-insensitive so "Active" can't fall through to
	// the id-list path and silently match nothing.
	where, _, ids = productFilter("Active")
	if where != whereActive || ids != nil {
		t.Errorf("Active: got %q %v", where, ids)
	}
	where, args, ids = productFilter("id-1,id_2")
	if where != "id = ANY($1)" || len(args) != 1 || len(ids) != 2 {
		t.Errorf("ids: got %q %v %v", where, args, ids)
	}
}

func TestAssetKeys(t *testing.T) {
	keys := assetKeys("dev.cdn", "ap-south-1",
		[]string{"https://dev.cdn/assets/IMAGE/a.jpg", "https://dev.cdn/assets/IMAGE/a.jpg", ""},
		[]string{"https://handloom-assets-dev.s3.ap-south-1.amazonaws.com/assets/IMAGE/b.png"})
	want := []string{"IMAGE/a.jpg", "IMAGE/b.png"}
	if len(keys) != 2 || keys[0] != want[0] || keys[1] != want[1] {
		t.Errorf("got %v, want %v", keys, want)
	}
}
