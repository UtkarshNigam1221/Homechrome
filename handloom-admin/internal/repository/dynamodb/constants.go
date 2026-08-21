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

	// Expression placeholders for the status attribute, which is reserved in
	// DynamoDB and so always aliased.
	nameStatus = "#status"
	valStatus  = ":status"
	attrTTL    = "ttl"

	// nameTTL aliases attrTTL inside expressions.
	nameTTL = "#ttl"

	// attrSuccessorHash names the token that replaced this one. Its presence is
	// the condition that lets one concurrent refresh claim the rotation.
	attrSuccessorHash = "successor_hash"

	// nameSuccessor aliases attrSuccessorHash inside expressions.
	nameSuccessor = "#successor"

	// skMetadata aliases domain.SKMetadata so dynamodb-layer reads short.
	skMetadata = domain.SKMetadata

	// Expression-attribute-value placeholders shared across repositories.
	valTTL = ":ttl"
	valOne = ":one"
)
