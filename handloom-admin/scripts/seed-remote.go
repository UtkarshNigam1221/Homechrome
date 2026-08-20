package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func main() {
	table := os.Getenv("DYNAMODB_CORE_TABLE")
	if table == "" {
		table = "handloom-core-dev"
	}
	// Validated before it is used or logged: a typo here silently seeds the wrong
	// table, and an unchecked env value has no business reaching a log line.
	if !regexp.MustCompile(`^[A-Za-z0-9_.-]{3,255}$`).MatchString(table) {
		log.Fatal("DYNAMODB_CORE_TABLE is not a valid DynamoDB table name")
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	// The hash is supplied, never baked in. A hardcoded one turns every re-seed
	// into a silent password reset back to whatever the repo happened to contain
	// — which is exactly how a rotated credential gets restored months later by
	// someone following the table-recreation runbook.
	passwordHash := os.Getenv("SEED_ADMIN_PASSWORD_HASH")
	if passwordHash == "" {
		log.Fatal("SEED_ADMIN_PASSWORD_HASH is required.\n" +
			"  Generate one:  go run ./scripts/generate-password-hash.go\n" +
			"  Then:          SEED_ADMIN_PASSWORD_HASH='$2a$12$...' go run ./scripts/seed-remote.go")
	}
	// Catches passing the password itself, which would store an unusable hash.
	if !strings.HasPrefix(passwordHash, "$2a$") && !strings.HasPrefix(passwordHash, "$2b$") {
		log.Fatal("SEED_ADMIN_PASSWORD_HASH does not look like a bcrypt hash — pass the hash, not the password")
	}

	adminItem := map[string]types.AttributeValue{
		"PK":             &types.AttributeValueMemberS{Value: "USER#usr_admin_001"},
		"SK":             &types.AttributeValueMemberS{Value: "METADATA"},
		"GSI1PK":         &types.AttributeValueMemberS{Value: "USER_EMAIL"},
		"GSI1SK":         &types.AttributeValueMemberS{Value: "admin@handloom.com"},
		"GSI2PK":         &types.AttributeValueMemberS{Value: "USER_ROLE"},
		"GSI2SK":         &types.AttributeValueMemberS{Value: "ADMIN#usr_admin_001"},
		"entity_type":    &types.AttributeValueMemberS{Value: "USER"},
		"id":             &types.AttributeValueMemberS{Value: "usr_admin_001"},
		"email":          &types.AttributeValueMemberS{Value: "admin@handloom.com"},
		"name":           &types.AttributeValueMemberS{Value: "System Administrator"},
		"password_hash":  &types.AttributeValueMemberS{Value: passwordHash},
		"role":           &types.AttributeValueMemberS{Value: "ADMIN"},
		"status":         &types.AttributeValueMemberS{Value: "ACTIVE"},
		"email_verified": &types.AttributeValueMemberBOOL{Value: true},
		"created_at":     &types.AttributeValueMemberS{Value: "2024-01-15T10:00:00Z"},
		"updated_at":     &types.AttributeValueMemberS{Value: "2024-01-15T10:00:00Z"},
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      adminItem,
	}
	// Seeding means creating, not replacing. Without this an accidental re-run
	// against a live environment overwrites the current admin's password.
	if os.Getenv("SEED_OVERWRITE") != "true" {
		input.ConditionExpression = aws.String("attribute_not_exists(PK)")
	}

	if _, err = client.PutItem(ctx, input); err != nil {
		var failed *types.ConditionalCheckFailedException
		if errors.As(err, &failed) {
			// G706 flags the env-derived table name reaching a log. It is matched
			// against ^[A-Za-z0-9_.-]{3,255}$ above, so it can carry no newline to
			// forge a log line with — and naming the table is the whole point of
			// this message.
			//nolint:gosec // G706: table is validated against a strict charset above.
			log.Fatalf("admin user already exists in %s — refusing to overwrite its password.\n"+
				"  Set SEED_OVERWRITE=true only if replacing it is what you mean.", table)
		}
		log.Fatalf("failed to seed admin user: %v", err)
	}
	fmt.Printf("Admin user seeded in %s: admin@handloom.com\n", table)
}
