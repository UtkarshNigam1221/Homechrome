package main

import (
	"context"
	"fmt"
	"log"
	"os"

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

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	// Admin user — password: Admin@123!
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
		"password_hash":  &types.AttributeValueMemberS{Value: "$2a$12$QA9/lw32rWAx6606bYrj4e1qkl9cqzp.MiG7lZS/K7giv4kD5/InC"},
		"role":           &types.AttributeValueMemberS{Value: "ADMIN"},
		"status":         &types.AttributeValueMemberS{Value: "ACTIVE"},
		"email_verified": &types.AttributeValueMemberBOOL{Value: true},
		"created_at":     &types.AttributeValueMemberS{Value: "2024-01-15T10:00:00Z"},
		"updated_at":     &types.AttributeValueMemberS{Value: "2024-01-15T10:00:00Z"},
	}

	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      adminItem,
	})
	if err != nil {
		log.Fatalf("failed to seed admin user: %v", err)
	}
	fmt.Println("Admin user seeded: admin@handloom.com / Admin@123!")
}
