#!/bin/bash

# Initialize local DynamoDB tables for development and testing
# Usage: ./scripts/init-local-db.sh

set -e

ENDPOINT="${DYNAMODB_LOCAL_ENDPOINT:-http://localhost:4566}"
REGION="${AWS_REGION:-ap-south-1}"

# Set dummy AWS credentials for LocalStack (bypasses SSO)
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-local}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-local}"

echo "Creating DynamoDB tables at $ENDPOINT..."

# Core Table (Users, Pricing Rules, Coupons)
aws dynamodb create-table \
    --endpoint-url $ENDPOINT \
    --region $REGION \
    --table-name handloom-core \
    --attribute-definitions \
        AttributeName=PK,AttributeType=S \
        AttributeName=SK,AttributeType=S \
        AttributeName=GSI1PK,AttributeType=S \
        AttributeName=GSI1SK,AttributeType=S \
        AttributeName=GSI2PK,AttributeType=S \
        AttributeName=GSI2SK,AttributeType=S \
    --key-schema \
        AttributeName=PK,KeyType=HASH \
        AttributeName=SK,KeyType=RANGE \
    --global-secondary-indexes \
        "[
            {
                \"IndexName\": \"GSI1\",
                \"KeySchema\": [{\"AttributeName\":\"GSI1PK\",\"KeyType\":\"HASH\"},{\"AttributeName\":\"GSI1SK\",\"KeyType\":\"RANGE\"}],
                \"Projection\": {\"ProjectionType\":\"ALL\"},
                \"ProvisionedThroughput\": {\"ReadCapacityUnits\": 5, \"WriteCapacityUnits\": 5}
            },
            {
                \"IndexName\": \"GSI2\",
                \"KeySchema\": [{\"AttributeName\":\"GSI2PK\",\"KeyType\":\"HASH\"},{\"AttributeName\":\"GSI2SK\",\"KeyType\":\"RANGE\"}],
                \"Projection\": {\"ProjectionType\":\"ALL\"},
                \"ProvisionedThroughput\": {\"ReadCapacityUnits\": 5, \"WriteCapacityUnits\": 5}
            }
        ]" \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
    2>/dev/null || echo "Table handloom-core already exists"

echo "Created handloom-core table"

# Catalog Table (Categories, Products, Inventory, Artisans)
aws dynamodb create-table \
    --endpoint-url $ENDPOINT \
    --region $REGION \
    --table-name handloom-catalog \
    --attribute-definitions \
        AttributeName=PK,AttributeType=S \
        AttributeName=SK,AttributeType=S \
        AttributeName=GSI1PK,AttributeType=S \
        AttributeName=GSI1SK,AttributeType=S \
        AttributeName=GSI2PK,AttributeType=S \
        AttributeName=GSI2SK,AttributeType=S \
    --key-schema \
        AttributeName=PK,KeyType=HASH \
        AttributeName=SK,KeyType=RANGE \
    --global-secondary-indexes \
        "[
            {
                \"IndexName\": \"GSI1\",
                \"KeySchema\": [{\"AttributeName\":\"GSI1PK\",\"KeyType\":\"HASH\"},{\"AttributeName\":\"GSI1SK\",\"KeyType\":\"RANGE\"}],
                \"Projection\": {\"ProjectionType\":\"ALL\"},
                \"ProvisionedThroughput\": {\"ReadCapacityUnits\": 5, \"WriteCapacityUnits\": 5}
            },
            {
                \"IndexName\": \"GSI2\",
                \"KeySchema\": [{\"AttributeName\":\"GSI2PK\",\"KeyType\":\"HASH\"},{\"AttributeName\":\"GSI2SK\",\"KeyType\":\"RANGE\"}],
                \"Projection\": {\"ProjectionType\":\"ALL\"},
                \"ProvisionedThroughput\": {\"ReadCapacityUnits\": 5, \"WriteCapacityUnits\": 5}
            }
        ]" \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
    2>/dev/null || echo "Table handloom-catalog already exists"

echo "Created handloom-catalog table"

# Notifications Table (Notifications)
aws dynamodb create-table \
    --endpoint-url $ENDPOINT \
    --region $REGION \
    --table-name handloom-notifications \
    --attribute-definitions \
        AttributeName=PK,AttributeType=S \
        AttributeName=SK,AttributeType=S \
        AttributeName=GSI1PK,AttributeType=S \
        AttributeName=GSI1SK,AttributeType=S \
    --key-schema \
        AttributeName=PK,KeyType=HASH \
        AttributeName=SK,KeyType=RANGE \
    --global-secondary-indexes \
        "[
            {
                \"IndexName\": \"GSI1\",
                \"KeySchema\": [{\"AttributeName\":\"GSI1PK\",\"KeyType\":\"HASH\"},{\"AttributeName\":\"GSI1SK\",\"KeyType\":\"RANGE\"}],
                \"Projection\": {\"ProjectionType\":\"ALL\"},
                \"ProvisionedThroughput\": {\"ReadCapacityUnits\": 5, \"WriteCapacityUnits\": 5}
            }
        ]" \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
    2>/dev/null || echo "Table handloom-notifications already exists"

echo "Created handloom-notifications table"

# Orders Table (Orders, Customers)
aws dynamodb create-table \
    --endpoint-url $ENDPOINT \
    --region $REGION \
    --table-name handloom-orders \
    --attribute-definitions \
        AttributeName=PK,AttributeType=S \
        AttributeName=SK,AttributeType=S \
        AttributeName=GSI1PK,AttributeType=S \
        AttributeName=GSI1SK,AttributeType=S \
        AttributeName=GSI2PK,AttributeType=S \
        AttributeName=GSI2SK,AttributeType=S \
    --key-schema \
        AttributeName=PK,KeyType=HASH \
        AttributeName=SK,KeyType=RANGE \
    --global-secondary-indexes \
        "[
            {
                \"IndexName\": \"GSI1\",
                \"KeySchema\": [{\"AttributeName\":\"GSI1PK\",\"KeyType\":\"HASH\"},{\"AttributeName\":\"GSI1SK\",\"KeyType\":\"RANGE\"}],
                \"Projection\": {\"ProjectionType\":\"ALL\"},
                \"ProvisionedThroughput\": {\"ReadCapacityUnits\": 5, \"WriteCapacityUnits\": 5}
            },
            {
                \"IndexName\": \"GSI2\",
                \"KeySchema\": [{\"AttributeName\":\"GSI2PK\",\"KeyType\":\"HASH\"},{\"AttributeName\":\"GSI2SK\",\"KeyType\":\"RANGE\"}],
                \"Projection\": {\"ProjectionType\":\"ALL\"},
                \"ProvisionedThroughput\": {\"ReadCapacityUnits\": 5, \"WriteCapacityUnits\": 5}
            }
        ]" \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
    2>/dev/null || echo "Table handloom-orders already exists"

echo "Created handloom-orders table"

# Audit Table (Audit Logs)
aws dynamodb create-table \
    --endpoint-url $ENDPOINT \
    --region $REGION \
    --table-name handloom-audit \
    --attribute-definitions \
        AttributeName=PK,AttributeType=S \
        AttributeName=SK,AttributeType=S \
        AttributeName=GSI1PK,AttributeType=S \
        AttributeName=GSI1SK,AttributeType=S \
        AttributeName=GSI2PK,AttributeType=S \
        AttributeName=GSI2SK,AttributeType=S \
    --key-schema \
        AttributeName=PK,KeyType=HASH \
        AttributeName=SK,KeyType=RANGE \
    --global-secondary-indexes \
        "[
            {
                \"IndexName\": \"GSI1\",
                \"KeySchema\": [{\"AttributeName\":\"GSI1PK\",\"KeyType\":\"HASH\"},{\"AttributeName\":\"GSI1SK\",\"KeyType\":\"RANGE\"}],
                \"Projection\": {\"ProjectionType\":\"ALL\"},
                \"ProvisionedThroughput\": {\"ReadCapacityUnits\": 5, \"WriteCapacityUnits\": 5}
            },
            {
                \"IndexName\": \"GSI2\",
                \"KeySchema\": [{\"AttributeName\":\"GSI2PK\",\"KeyType\":\"HASH\"},{\"AttributeName\":\"GSI2SK\",\"KeyType\":\"RANGE\"}],
                \"Projection\": {\"ProjectionType\":\"ALL\"},
                \"ProvisionedThroughput\": {\"ReadCapacityUnits\": 5, \"WriteCapacityUnits\": 5}
            }
        ]" \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
    2>/dev/null || echo "Table handloom-audit already exists"

echo "Created handloom-audit table"

# Analytics Table (Analytics data)
aws dynamodb create-table \
    --endpoint-url $ENDPOINT \
    --region $REGION \
    --table-name handloom-analytics \
    --attribute-definitions \
        AttributeName=PK,AttributeType=S \
        AttributeName=SK,AttributeType=S \
        AttributeName=GSI1PK,AttributeType=S \
        AttributeName=GSI1SK,AttributeType=S \
    --key-schema \
        AttributeName=PK,KeyType=HASH \
        AttributeName=SK,KeyType=RANGE \
    --global-secondary-indexes \
        "[
            {
                \"IndexName\": \"GSI1\",
                \"KeySchema\": [{\"AttributeName\":\"GSI1PK\",\"KeyType\":\"HASH\"},{\"AttributeName\":\"GSI1SK\",\"KeyType\":\"RANGE\"}],
                \"Projection\": {\"ProjectionType\":\"ALL\"},
                \"ProvisionedThroughput\": {\"ReadCapacityUnits\": 5, \"WriteCapacityUnits\": 5}
            }
        ]" \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
    2>/dev/null || echo "Table handloom-analytics already exists"

echo "Created handloom-analytics table"

# Sessions Table (OTP, admin refresh tokens, customer refresh tokens)
aws dynamodb create-table \
    --endpoint-url $ENDPOINT \
    --region $REGION \
    --table-name handloom-sessions \
    --attribute-definitions \
        AttributeName=PK,AttributeType=S \
        AttributeName=SK,AttributeType=S \
    --key-schema \
        AttributeName=PK,KeyType=HASH \
        AttributeName=SK,KeyType=RANGE \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
    2>/dev/null || echo "Table handloom-sessions already exists"

echo "Created handloom-sessions table"

# Events table (raw event store)
echo "Creating events table..."
aws dynamodb create-table \
    --endpoint-url $ENDPOINT \
    --region $REGION \
    --table-name handloom-events \
    --attribute-definitions \
        AttributeName=PK,AttributeType=S \
        AttributeName=SK,AttributeType=S \
    --key-schema \
        AttributeName=PK,KeyType=HASH \
        AttributeName=SK,KeyType=RANGE \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 \
    2>/dev/null || echo "Table handloom-events already exists"

echo "Created handloom-events table"

# Enable TTL on sessions table (for OTP and refresh token expiry)
aws dynamodb update-time-to-live \
    --table-name handloom-sessions \
    --time-to-live-specification "Enabled=true, AttributeName=ttl" \
    --endpoint-url $ENDPOINT \
    --region $REGION \
    2>/dev/null || echo "TTL already enabled on handloom-sessions"

echo "Enabled TTL on handloom-sessions table"

# Enable TTL on core table (for pricing/coupon expiry)
aws dynamodb update-time-to-live \
    --table-name handloom-core \
    --time-to-live-specification "Enabled=true, AttributeName=ttl" \
    --endpoint-url $ENDPOINT \
    --region $REGION \
    2>/dev/null || echo "TTL already enabled on handloom-core"

echo "Enabled TTL on handloom-core table"

# Enable TTL on catalog table
aws dynamodb update-time-to-live \
    --table-name handloom-catalog \
    --time-to-live-specification "Enabled=true, AttributeName=ttl" \
    --endpoint-url $ENDPOINT \
    --region $REGION \
    2>/dev/null || echo "TTL already enabled on handloom-catalog"

echo "Enabled TTL on handloom-catalog table"

# Enable TTL on notifications table
aws dynamodb update-time-to-live \
    --table-name handloom-notifications \
    --time-to-live-specification "Enabled=true, AttributeName=ttl" \
    --endpoint-url $ENDPOINT \
    --region $REGION \
    2>/dev/null || echo "TTL already enabled on handloom-notifications"

echo "Enabled TTL on handloom-notifications table"

# Enable TTL on orders table (for PriceQuote and Cart expiry)
aws dynamodb update-time-to-live \
    --table-name handloom-orders \
    --time-to-live-specification "Enabled=true, AttributeName=ttl" \
    --endpoint-url $ENDPOINT \
    --region $REGION \
    2>/dev/null || echo "TTL already enabled on handloom-orders"

echo "Enabled TTL on handloom-orders table"

# Enable TTL on events table (for raw event expiry)
aws dynamodb update-time-to-live \
    --table-name handloom-events \
    --time-to-live-specification "Enabled=true, AttributeName=ttl" \
    --endpoint-url $ENDPOINT \
    --region $REGION \
    2>/dev/null || echo "TTL already enabled on handloom-events"

echo "Enabled TTL on handloom-events table"

echo ""
echo "All DynamoDB tables created successfully!"
echo ""
echo "Tables:"
aws dynamodb list-tables --endpoint-url $ENDPOINT --region $REGION
