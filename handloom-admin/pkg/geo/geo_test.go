package geo

import (
	"strings"
	"testing"
)

func TestNormalizeCity(t *testing.T) {
	long := strings.Repeat("a", 100)
	cases := []struct{ in, want string }{
		{"Mumbai", "mumbai"},
		{"  MUMBAI  ", "mumbai"},
		{"São Paulo", "são paulo"},
		{"", "unknown"},
		{"   ", "unknown"},
		{long, strings.Repeat("a", MaxCityNameLen)},
	}
	for _, c := range cases {
		got := NormalizeCity(c.in)
		if got != c.want {
			t.Errorf("NormalizeCity(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeCountry(t *testing.T) {
	cases := []struct{ in, want string }{
		{"IN", "IN"},
		{"in", "IN"},
		{" us ", "US"},
		{"", "unknown"},
		{"INX", "unknown"},
		{"I", "unknown"},
	}
	for _, c := range cases {
		got := NormalizeCountry(c.in)
		if got != c.want {
			t.Errorf("NormalizeCountry(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
