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

	// nameTTL aliases attrTTL inside expressions. ttl is not reserved, but the
	// alias keeps every TTL expression written the same way.
	nameTTL = "#ttl"

	// attrSuccessorHash marks a refresh-token row as rotated and names the
	// token that replaced it. Its presence is the condition that lets exactly
	// one of several concurrent refreshes claim the rotation.
	attrSuccessorHash = "successor_hash"

	// nameSuccessor aliases attrSuccessorHash inside expressions.
	nameSuccessor = "#successor"

	// skMetadata aliases domain.SKMetadata so dynamodb-layer reads short.
	skMetadata = domain.SKMetadata
)
