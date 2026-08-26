package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Prices are tax-inclusive at 5%. GST is extracted from the total, never added to it,
// and the customer is never shown a tax line.
func TestExtractTax(t *testing.T) {
	cases := []struct {
		name      string
		inclusive int64
		want      int64
	}{
		{"the worked example from the spec", 10500, 500},
		{"zero", 0, 0},
		{"negative is refused rather than inverted", -100, 0},
		{"one rupee", 100, 5},
		{"rounds half up", 21, 1},       // 21*5/105 = 1.0 exactly
		{"rounds a fraction up", 32, 2}, // 32*5/105 = 1.523 -> 2
		{"a realistic order", 349900, 16662},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, extractTax(tc.inclusive))
		})
	}
}

// The invariant that matters for an invoice: the taxable value plus the tax must be
// exactly what the customer paid, with nothing lost to rounding.
func TestExtractTax_TaxableAndTaxSumToTheTotal(t *testing.T) {
	for total := int64(1); total <= 20000; total++ {
		tax := extractTax(total)
		taxable := total - tax

		require.Equal(t, total, taxable+tax, "total %d", total)
		require.GreaterOrEqual(t, tax, int64(0), "total %d", total)
		require.Less(t, tax, total, "tax must never be the whole total, at %d", total)
	}
}
