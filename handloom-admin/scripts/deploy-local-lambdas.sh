#!/bin/bash

# Deploy Lambda functions to LocalStack
# Usage: ./scripts/deploy-local-lambdas.sh
# Set REDEPLOY_ONLY=true to skip API Gateway recreation

set -e

ENDPOINT="${AWS_ENDPOINT:-http://localhost:4566}"
REGION="${AWS_REGION:-ap-south-1}"
LAMBDA_DIR="./bin/lambda"
ACTIVE_SERVICES="auth user catalog asset store-auth store-catalog store-cart store-checkout store-orders store-tracking store-profile store-events store-webhooks"

# Env vars injected into each Lambda
LAMBDA_ENV='{
  "Variables": {
    "APP_ENV": "development",
    "APP_DEBUG": "true",
    "AWS_ENDPOINT": "http://host.docker.internal:4566",
    "AWS_REGION": "ap-south-1",
    "AWS_ACCESS_KEY_ID": "local",
    "AWS_SECRET_ACCESS_KEY": "local",
    "DYNAMODB_CORE_TABLE": "handloom-core",
    "DYNAMODB_ORDERS_TABLE": "handloom-orders",
    "DYNAMODB_AUDIT_TABLE": "handloom-audit",
    "DYNAMODB_NOTIFICATIONS_TABLE": "handloom-notifications",
    "DYNAMODB_SESSIONS_TABLE": "handloom-sessions",
    "POSTGRES_DSN": "postgres://handloom:handloom@host.docker.internal:5432/handloom?sslmode=disable",
    "CUSTOMER_JWT_SECRET": "customer-secret-change-in-production",
    "JWT_SECRET_KEY": "dev-secret-key-change-in-production",
    "S3_ASSETS_BUCKET": "handloom-assets",
    "QUOTE_VALIDITY_HRS": "24"
  }
}'

# Set dummy AWS credentials
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-local}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-local}"
export AWS_DEFAULT_REGION="$REGION"

echo "=============================================="
echo "  Deploying Lambda Functions to LocalStack"
echo "=============================================="
echo ""

# --- IAM Role (required even in LocalStack) ---
ROLE_ARN="arn:aws:iam::000000000000:role/handloom-lambda-role"
if [ "$REDEPLOY_ONLY" != "true" ]; then
    echo "Creating IAM execution role..."
    aws iam create-role \
        --endpoint-url "$ENDPOINT" \
        --region "$REGION" \
        --role-name handloom-lambda-role \
        --assume-role-policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"lambda.amazonaws.com"},"Action":"sts:AssumeRole"}]}' \
        2>/dev/null || echo "  Role already exists"
fi

# --- Deploy each Lambda ---
for svc in $ACTIVE_SERVICES; do
    BOOTSTRAP="$LAMBDA_DIR/$svc/bootstrap"
    ZIP_FILE="/tmp/handloom-lambda-$svc.zip"

    if [ ! -f "$BOOTSTRAP" ]; then
        echo "ERROR: $BOOTSTRAP not found. Run 'make build-lambdas-active' first."
        exit 1
    fi

    echo ""
    echo "Deploying $svc Lambda..."

    # Zip the bootstrap binary
    (cd "$LAMBDA_DIR/$svc" && zip -j "$ZIP_FILE" bootstrap > /dev/null)

    FUNC_NAME="handloom-$svc"

    if [ "$REDEPLOY_ONLY" = "true" ]; then
        # Update code only
        aws lambda update-function-code \
            --endpoint-url "$ENDPOINT" \
            --region "$REGION" \
            --function-name "$FUNC_NAME" \
            --zip-file "fileb://$ZIP_FILE" \
            > /dev/null
        echo "  Updated $FUNC_NAME code"
    else
        # Delete existing (ignore error if doesn't exist)
        aws lambda delete-function \
            --endpoint-url "$ENDPOINT" \
            --region "$REGION" \
            --function-name "$FUNC_NAME" \
            2>/dev/null || true

        # Create function
        aws lambda create-function \
            --endpoint-url "$ENDPOINT" \
            --region "$REGION" \
            --function-name "$FUNC_NAME" \
            --runtime "provided.al2023" \
            --architectures arm64 \
            --handler "bootstrap" \
            --role "$ROLE_ARN" \
            --zip-file "fileb://$ZIP_FILE" \
            --environment "$LAMBDA_ENV" \
            --timeout 30 \
            --memory-size 128 \
            > /dev/null
        echo "  Created $FUNC_NAME"
    fi

    rm -f "$ZIP_FILE"
done

# --- API Gateway ---
if [ "$REDEPLOY_ONLY" = "true" ]; then
    echo ""
    echo "Skipping API Gateway (REDEPLOY_ONLY=true)"
    echo ""
    echo "Lambda code updated successfully!"
    exit 0
fi

echo ""
echo "Setting up API Gateway..."

# Create REST API
API_ID=$(aws apigateway create-rest-api \
    --endpoint-url "$ENDPOINT" \
    --region "$REGION" \
    --name "handloom-admin-local" \
    --query 'id' --output text 2>/dev/null || true)

if [ -z "$API_ID" ]; then
    echo "ERROR: Failed to create API Gateway"
    exit 1
fi

echo "  Created API: $API_ID"

# Get root resource ID
ROOT_ID=$(aws apigateway get-resources \
    --endpoint-url "$ENDPOINT" \
    --region "$REGION" \
    --rest-api-id "$API_ID" \
    --query 'items[?path==`/`].id' --output text)

# Helper: create a catch-all proxy resource under a path and wire it to a Lambda
create_proxy_route() {
    local parent_path="$1"
    local lambda_name="$2"
    local parent_id="$ROOT_ID"

    # Create each path segment
    IFS='/' read -ra SEGMENTS <<< "${parent_path#/}"
    for seg in "${SEGMENTS[@]}"; do
        EXISTING=$(aws apigateway get-resources \
            --endpoint-url "$ENDPOINT" \
            --region "$REGION" \
            --rest-api-id "$API_ID" \
            --query "items[?parentId==\`$parent_id\` && pathPart==\`$seg\`].id" --output text 2>/dev/null)
        if [ -n "$EXISTING" ] && [ "$EXISTING" != "None" ]; then
            parent_id="$EXISTING"
        else
            parent_id=$(aws apigateway create-resource \
                --endpoint-url "$ENDPOINT" \
                --region "$REGION" \
                --rest-api-id "$API_ID" \
                --parent-id "$parent_id" \
                --path-part "$seg" \
                --query 'id' --output text)
        fi
    done

    # Create {proxy+} child
    PROXY_ID=$(aws apigateway create-resource \
        --endpoint-url "$ENDPOINT" \
        --region "$REGION" \
        --rest-api-id "$API_ID" \
        --parent-id "$parent_id" \
        --path-part "{proxy+}" \
        --query 'id' --output text 2>/dev/null || true)

    LAMBDA_ARN="arn:aws:lambda:$REGION:000000000000:function:$lambda_name"
    INTEGRATION_URI="arn:aws:apigateway:$REGION:lambda:path/2015-03-31/functions/$LAMBDA_ARN/invocations"

    # Wire methods on the parent resource itself
    for method in GET POST PUT PATCH DELETE OPTIONS; do
        aws apigateway put-method \
            --endpoint-url "$ENDPOINT" \
            --region "$REGION" \
            --rest-api-id "$API_ID" \
            --resource-id "$parent_id" \
            --http-method "$method" \
            --authorization-type "NONE" \
            > /dev/null 2>&1 || true

        aws apigateway put-integration \
            --endpoint-url "$ENDPOINT" \
            --region "$REGION" \
            --rest-api-id "$API_ID" \
            --resource-id "$parent_id" \
            --http-method "$method" \
            --type AWS_PROXY \
            --integration-http-method POST \
            --uri "$INTEGRATION_URI" \
            > /dev/null 2>&1 || true
    done

    # Wire methods on {proxy+}
    if [ -n "$PROXY_ID" ] && [ "$PROXY_ID" != "None" ]; then
        for method in GET POST PUT PATCH DELETE OPTIONS; do
            aws apigateway put-method \
                --endpoint-url "$ENDPOINT" \
                --region "$REGION" \
                --rest-api-id "$API_ID" \
                --resource-id "$PROXY_ID" \
                --http-method "$method" \
                --authorization-type "NONE" \
                > /dev/null 2>&1 || true

            aws apigateway put-integration \
                --endpoint-url "$ENDPOINT" \
                --region "$REGION" \
                --rest-api-id "$API_ID" \
                --resource-id "$PROXY_ID" \
                --http-method "$method" \
                --type AWS_PROXY \
                --integration-http-method POST \
                --uri "$INTEGRATION_URI" \
                > /dev/null 2>&1 || true
        done
    fi

    echo "  Routed $parent_path/* → $lambda_name"
}

# Route mappings (matching CDK/production)
create_proxy_route "/admin/auth" "handloom-auth"
create_proxy_route "/admin/users" "handloom-user"
create_proxy_route "/admin/categories" "handloom-catalog"
create_proxy_route "/admin/products" "handloom-catalog"
create_proxy_route "/admin/assets" "handloom-asset"

# Store routes (B2C storefront)
create_proxy_route "/api/v1/store/auth"     "handloom-store-auth"
create_proxy_route "/api/v1/store/catalog"  "handloom-store-catalog"
create_proxy_route "/api/v1/store/cart"     "handloom-store-cart"
create_proxy_route "/api/v1/store/checkout" "handloom-store-checkout"
create_proxy_route "/api/v1/store/orders"   "handloom-store-orders"
create_proxy_route "/api/v1/store/me"       "handloom-store-profile"
create_proxy_route "/api/v1/store/track"    "handloom-store-tracking"
create_proxy_route "/api/v1/store/events"   "handloom-store-events"
create_proxy_route "/api/v1/store/webhooks" "handloom-store-webhooks"

# Add health check on root
aws apigateway put-method \
    --endpoint-url "$ENDPOINT" \
    --region "$REGION" \
    --rest-api-id "$API_ID" \
    --resource-id "$ROOT_ID" \
    --http-method "GET" \
    --authorization-type "NONE" \
    > /dev/null 2>&1 || true

# Deploy the API
aws apigateway create-deployment \
    --endpoint-url "$ENDPOINT" \
    --region "$REGION" \
    --rest-api-id "$API_ID" \
    --stage-name "local" \
    > /dev/null

INVOKE_URL="http://localhost:4566/restapis/$API_ID/local/_user_request_"

echo ""
echo "=============================================="
echo "  Lambda Deployment Complete!"
echo "=============================================="
echo ""
echo "  API Gateway URL: $INVOKE_URL"
echo "  Health Check:    curl $INVOKE_URL/health"
echo ""
echo "  Functions deployed:"
for svc in $ACTIVE_SERVICES; do
    echo "    - handloom-$svc"
done
echo ""
echo "  Save this API Gateway URL for frontend .env.local-lambda:"
echo "  VITE_API_URL=$INVOKE_URL"
echo "  NEXT_PUBLIC_API_URL=$INVOKE_URL"
echo ""

# Save the API ID for redeploy reference
echo "$API_ID" > /tmp/handloom-local-api-id
echo "$INVOKE_URL" > /tmp/handloom-local-api-url

# Auto-update frontend .env.local-lambda if it exists
FRONTEND_ENV="../handloom-admin-frontend/.env.local-lambda"
if [ -f "$FRONTEND_ENV" ]; then
    sed -i.bak "s|^VITE_API_URL=.*|VITE_API_URL=$INVOKE_URL|" "$FRONTEND_ENV"
    rm -f "${FRONTEND_ENV}.bak"
    echo "  Updated $FRONTEND_ENV with API Gateway URL"
fi

# Auto-update storefront .env.local-lambda if it exists
STORE_ENV="../homechrome-store/.env.local-lambda"
if [ -f "$STORE_ENV" ]; then
    sed -i.bak "s|^NEXT_PUBLIC_API_URL=.*|NEXT_PUBLIC_API_URL=$INVOKE_URL|" "$STORE_ENV"
    rm -f "${STORE_ENV}.bak"
    echo "  Updated $STORE_ENV with API Gateway URL"
fi
echo ""
