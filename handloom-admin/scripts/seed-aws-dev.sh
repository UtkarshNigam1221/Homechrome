#!/bin/bash

# Script to seed initial data to AWS DynamoDB (dev environment)
# Usage: ./scripts/seed-aws-dev.sh
#
# Prerequisites:
# - AWS CLI configured with credentials
# - Run: aws sso login (if using SSO)

REGION="${AWS_REGION:-ap-southeast-1}"
TABLE="handloom-core-dev"

echo "=============================================="
echo "  Seeding data to AWS DynamoDB (dev)"
echo "=============================================="
echo ""
echo "Region: $REGION"
echo "Table: $TABLE"
echo ""

# Find AWS CLI
if command -v aws &> /dev/null; then
    AWS_CMD="aws"
elif [ -x "/usr/local/bin/aws" ]; then
    AWS_CMD="/usr/local/bin/aws"
elif [ -x "/opt/homebrew/bin/aws" ]; then
    AWS_CMD="/opt/homebrew/bin/aws"
else
    echo "ERROR: AWS CLI not found"
    exit 1
fi

echo "Using AWS CLI: $AWS_CMD"
echo ""

# =============================================================================
# Admin User
# =============================================================================
echo "Creating admin user..."
# Password: Admin@123! (bcrypt hash, cost 12)
$AWS_CMD dynamodb put-item \
    --table-name $TABLE \
    --item '{
        "PK": {"S": "USER#usr_admin_001"},
        "SK": {"S": "METADATA"},
        "GSI1PK": {"S": "USER_EMAIL"},
        "GSI1SK": {"S": "admin@handloom.com"},
        "GSI2PK": {"S": "USER_ROLE"},
        "GSI2SK": {"S": "ADMIN#usr_admin_001"},
        "entity_type": {"S": "USER"},
        "id": {"S": "usr_admin_001"},
        "email": {"S": "admin@handloom.com"},
        "first_name": {"S": "System"},
        "last_name": {"S": "Administrator"},
        "password_hash": {"S": "$2a$12$QA9/lw32rWAx6606bYrj4e1qkl9cqzp.MiG7lZS/K7giv4kD5/InC"},
        "role": {"S": "ADMIN"},
        "status": {"S": "ACTIVE"},
        "email_verified": {"BOOL": true},
        "created_at": {"S": "2024-01-15T10:00:00Z"},
        "updated_at": {"S": "2024-01-15T10:00:00Z"}
    }' \
    --region $REGION && echo "✓ Admin user created" || echo "✗ Failed to create admin user"

# Create operator user
echo "Creating operator user..."
$AWS_CMD dynamodb put-item \
    --table-name $TABLE \
    --item '{
        "PK": {"S": "USER#usr_operator_001"},
        "SK": {"S": "METADATA"},
        "GSI1PK": {"S": "USER_EMAIL"},
        "GSI1SK": {"S": "operator@handloom.com"},
        "GSI2PK": {"S": "USER_ROLE"},
        "GSI2SK": {"S": "OPERATOR#usr_operator_001"},
        "entity_type": {"S": "USER"},
        "id": {"S": "usr_operator_001"},
        "email": {"S": "operator@handloom.com"},
        "first_name": {"S": "Store"},
        "last_name": {"S": "Operator"},
        "password_hash": {"S": "$2a$12$QA9/lw32rWAx6606bYrj4e1qkl9cqzp.MiG7lZS/K7giv4kD5/InC"},
        "role": {"S": "OPERATOR"},
        "status": {"S": "ACTIVE"},
        "email_verified": {"BOOL": true},
        "created_at": {"S": "2024-01-15T10:00:00Z"},
        "updated_at": {"S": "2024-01-15T10:00:00Z"}
    }' \
    --region $REGION && echo "✓ Operator user created" || echo "✗ Failed to create operator user"

echo ""
echo "=============================================="
echo "  Seed data complete!"
echo "=============================================="
echo ""
echo "Default Users:"
echo "  Admin:"
echo "    Email: admin@handloom.com"
echo "    Password: Admin@123!"
echo "    Role: ADMIN"
echo ""
echo "  Operator:"
echo "    Email: operator@handloom.com"
echo "    Password: Admin@123!"
echo "    Role: OPERATOR"
echo ""
echo "=============================================="
echo "IMPORTANT: Change default passwords in production!"
echo "=============================================="
