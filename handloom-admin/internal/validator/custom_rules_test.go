package validator

import "testing"

func TestStorePath(t *testing.T) {
	s := New()
	for _, tc := range []struct {
		in string
		ok bool
	}{
		{"", true},
		{"/products", true},
		{"/c/bedsheets?sort=new", true},
		{"//evil.example/x", false},
		{"/\\evil.example/x", false},
		{"https://evil.example", false},
		{"products", false},
	} {
		if err := s.ValidateVar(tc.in, "omitempty,storepath"); (err == nil) != tc.ok {
			t.Errorf("storepath(%q) err=%v, want ok=%v", tc.in, err, tc.ok)
		}
	}
}
