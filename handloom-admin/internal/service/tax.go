package service

// GST is 5% on every SKU, and every price is tax-inclusive: ₹105 charged is ₹100 taxable
// plus ₹5 tax. So tax is extracted from a total rather than added to it, and the customer
// is never shown a tax line. Order.TaxAmount exists for the invoice and the admin view.
const gstRatePercent = 5

// extractTax returns the GST contained within a tax-inclusive amount, rounded half up.
//
// Integer arithmetic throughout — no float touches money. The doubling is the half-up
// rounding trick: (2n + d) / 2d rounds n/d to the nearest integer, ties going up.
func extractTax(inclusive int64) int64 {
	if inclusive <= 0 {
		return 0
	}
	den := int64(100 + gstRatePercent)
	return (inclusive*gstRatePercent*2 + den) / (den * 2)
}
