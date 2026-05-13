package dynamodb

import "github.com/handloom/admin/internal/domain"

// Centralized so :pk / :sk / :now placeholders, status/ttl attribute names
// stay identical across every Query/UpdateItem call. Drift here would silently
// break expressions that share placeholder maps.
const (
	exprPK  = ":pk"
	exprSK  = ":sk"
	exprNow = ":now"

	attrStatus = "status"
	attrTTL    = "ttl"

	// skMetadata aliases domain.SKMetadata so dynamodb-layer reads short.
	skMetadata = domain.SKMetadata
)
