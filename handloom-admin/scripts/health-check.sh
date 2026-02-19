#!/bin/bash

# Health check script for local development
# Usage: ./scripts/health-check.sh

set -e

API_URL="${API_URL:-http://localhost:8080}"
LOCALSTACK_URL="${AWS_ENDPOINT:-http://localhost:4566}"

# Set dummy AWS credentials for LocalStack (bypasses SSO)
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-local}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-local}"

echo "Checking service health..."
echo ""

# Check LocalStack
echo -n "LocalStack ($LOCALSTACK_URL): "
if curl -s "$LOCALSTACK_URL/_localstack/health" > /dev/null 2>&1; then
    echo "Running"
else
    echo "Not running"
    echo "   Start with: make docker-up"
fi

# Check API Server
echo -n "API Server ($API_URL): "
HEALTH_RESPONSE=$(curl -s "$API_URL/health" 2>/dev/null || echo "")
if [ "$HEALTH_RESPONSE" = '{"status":"ok"}' ]; then
    echo "Running"
else
    echo "Not running"
    echo "   Start with: make run"
fi

# Check DynamoDB tables
echo ""
echo "DynamoDB Tables:"
TABLES=$(aws dynamodb list-tables --endpoint-url $LOCALSTACK_URL --region ap-south-1 2>/dev/null | grep -o '"handloom-[^"]*"' | tr -d '"' || echo "")
if [ -n "$TABLES" ]; then
    echo "$TABLES" | while read table; do
        echo "  $table"
    done
else
    echo "  No tables found"
    echo "     Create with: make create-tables"
fi

echo ""
echo "Quick commands:"
echo "  make setup-local  # Full local setup (docker + tables + seed)"
echo "  make run          # Start API server"
echo "  make docker-up    # Start LocalStack"
echo "  make seed-data    # Seed test data"
