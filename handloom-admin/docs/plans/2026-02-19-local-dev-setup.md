# Local Dev Setup Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the full backend testable locally with two modes — monolith (fast dev) and Lambda via LocalStack (integration testing) — with the frontend able to target either.

**Architecture:** Replace DynamoDB Local with LocalStack (unified AWS emulator for DynamoDB, S3, Lambda, API Gateway). Modify the S3 client to accept an endpoint override so presigned URLs work against LocalStack. Add Lambda deployment script and three frontend dev modes.

**Tech Stack:** LocalStack (Docker), Go AWS SDK v2 (endpoint override), bash scripts, Vite env modes

---

### Task 1: Replace DynamoDB Local with LocalStack in docker-compose.yml

**Files:**
- Modify: `handloom-admin/docker-compose.yml`

**Step 1: Rewrite docker-compose.yml**

Replace the entire file with LocalStack as the unified AWS emulator:

```yaml
version: '3.8'

services:
  # LocalStack — unified AWS emulator (DynamoDB, S3, Lambda, API Gateway, IAM)
  localstack:
    image: localstack/localstack:latest
    container_name: handloom-localstack
    ports:
      - "4566:4566"           # LocalStack gateway
    environment:
      SERVICES: dynamodb,s3,lambda,apigateway,iam
      DEFAULT_REGION: ap-south-1
      LAMBDA_EXECUTOR: docker
      DOCKER_HOST: unix:///var/run/docker.sock
      DEBUG: 0
    volumes:
      - localstack-data:/var/lib/localstack
      - /var/run/docker.sock:/var/run/docker.sock
    healthcheck:
      test: ["CMD-SHELL", "curl -sf http://localhost:4566/_localstack/health | grep -q '\"dynamodb\": \"running\"' || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 10

  # DynamoDB Admin UI (optional — browse tables in browser)
  dynamodb-admin:
    image: aaronshaf/dynamodb-admin
    container_name: handloom-dynamodb-admin
    ports:
      - "8001:8001"
    environment:
      DYNAMO_ENDPOINT: http://localstack:4566
      AWS_REGION: ap-south-1
      AWS_ACCESS_KEY_ID: local
      AWS_SECRET_ACCESS_KEY: local
    depends_on:
      localstack:
        condition: service_healthy

volumes:
  localstack-data:
```

**Step 2: Verify Docker services start**

Run: `cd handloom-admin && docker-compose up -d`
Expected: Both containers start. `curl http://localhost:4566/_localstack/health` returns JSON with `"dynamodb": "running"` and `"s3": "running"`.

**Step 3: Commit**

```bash
git add handloom-admin/docker-compose.yml
git commit -m "infra: replace DynamoDB Local with LocalStack"
```

---

### Task 2: Update init-local-db.sh endpoint

**Files:**
- Modify: `handloom-admin/scripts/init-local-db.sh`

**Step 1: Change the default endpoint**

Change line 8 from:
```bash
ENDPOINT="${DYNAMODB_LOCAL_ENDPOINT:-http://localhost:8000}"
```
to:
```bash
ENDPOINT="${DYNAMODB_LOCAL_ENDPOINT:-http://localhost:4566}"
```

**Step 2: Fix the default region to match the app**

Change line 9 from:
```bash
REGION="${AWS_REGION:-us-east-1}"
```
to:
```bash
REGION="${AWS_REGION:-ap-south-1}"
```

**Step 3: Test table creation**

Run: `cd handloom-admin && docker-compose up -d && sleep 5 && ./scripts/init-local-db.sh`
Expected: All 4 tables created. `aws dynamodb list-tables --endpoint-url http://localhost:4566 --region ap-south-1` shows `handloom-core`, `handloom-orders`, `handloom-audit`, `handloom-analytics`.

**Step 4: Commit**

```bash
git add handloom-admin/scripts/init-local-db.sh
git commit -m "fix: point init-local-db.sh at LocalStack endpoint"
```

---

### Task 3: Update seed-data.sh endpoint

**Files:**
- Modify: `handloom-admin/scripts/seed-data.sh`

**Step 1: Change the default endpoint**

Change line 6 from:
```bash
ENDPOINT="${DYNAMODB_LOCAL_ENDPOINT:-http://localhost:8000}"
```
to:
```bash
ENDPOINT="${DYNAMODB_LOCAL_ENDPOINT:-http://localhost:4566}"
```

**Step 2: Test seeding**

Run: `cd handloom-admin && ./scripts/seed-data.sh`
Expected: Output shows "Test data seeded successfully!" with admin user and categories created.

**Step 3: Commit**

```bash
git add handloom-admin/scripts/seed-data.sh
git commit -m "fix: point seed-data.sh at LocalStack endpoint"
```

---

### Task 4: Create init-local-s3.sh

**Files:**
- Create: `handloom-admin/scripts/init-local-s3.sh`

**Step 1: Write the S3 bucket initialization script**

```bash
#!/bin/bash

# Initialize local S3 buckets in LocalStack
# Usage: ./scripts/init-local-s3.sh

set -e

ENDPOINT="${AWS_ENDPOINT:-http://localhost:4566}"
REGION="${AWS_REGION:-ap-south-1}"

# Set dummy AWS credentials for LocalStack
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-local}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-local}"

echo "Creating S3 buckets at $ENDPOINT..."

# Assets bucket (product images, category images, artisan photos)
aws s3 mb "s3://handloom-assets" \
    --endpoint-url "$ENDPOINT" \
    --region "$REGION" \
    2>/dev/null || echo "Bucket handloom-assets already exists"

echo "Created handloom-assets bucket"

# Uploads bucket (temporary private uploads)
aws s3 mb "s3://handloom-uploads" \
    --endpoint-url "$ENDPOINT" \
    --region "$REGION" \
    2>/dev/null || echo "Bucket handloom-uploads already exists"

echo "Created handloom-uploads bucket"

echo ""
echo "S3 buckets created successfully!"
echo ""
echo "Buckets:"
aws s3 ls --endpoint-url "$ENDPOINT" --region "$REGION"
```

**Step 2: Make executable and test**

Run: `chmod +x handloom-admin/scripts/init-local-s3.sh && cd handloom-admin && ./scripts/init-local-s3.sh`
Expected: Output shows both buckets created. `aws s3 ls --endpoint-url http://localhost:4566 --region ap-south-1` shows `handloom-assets` and `handloom-uploads`.

**Step 3: Commit**

```bash
git add handloom-admin/scripts/init-local-s3.sh
git commit -m "feat: add init-local-s3.sh for LocalStack S3 buckets"
```

---

### Task 5: Add endpoint override to S3 client

**Files:**
- Modify: `handloom-admin/internal/s3client/client.go`

**Step 1: Write a test for the new endpoint behavior**

This is a wiring-level change. The key testable behavior is that `New()` with an endpoint produces a client that can be used (no nil pointer). Existing service tests mock `s3Ops` so they're unaffected.

We verify this works end-to-end in Task 12 (integration test against LocalStack).

**Step 2: Update s3client.New() to accept an endpoint**

Replace the full file:

```go
// Package s3client provides S3 presigned URL generation and object management
package s3client

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client wraps the AWS S3 client with presigned URL support
type S3Client struct {
	client        *s3.Client
	presignClient *s3.PresignClient
}

// New creates a new S3Client.
// When endpoint is non-empty (local dev), it targets that endpoint with
// path-style addressing and static dummy credentials.
// When endpoint is empty (production), it uses the default AWS credential chain.
func New(ctx context.Context, region string, endpoint string) (*S3Client, error) {
	var cfg aws.Config
	var err error

	if endpoint != "" {
		cfg, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(region),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				"local", "local", "",
			)),
		)
	} else {
		cfg, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(region),
		)
	}
	if err != nil {
		return nil, err
	}

	var client *s3.Client
	if endpoint != "" {
		client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		})
	} else {
		client = s3.NewFromConfig(cfg)
	}

	return &S3Client{
		client:        client,
		presignClient: s3.NewPresignClient(client),
	}, nil
}

// GeneratePresignedPutURL creates a presigned PUT URL for uploading an object to S3
func (c *S3Client) GeneratePresignedPutURL(ctx context.Context, bucket, key, contentType string, expiry time.Duration) (string, error) {
	req, err := c.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

// CopyObject copies an object within the same bucket
func (c *S3Client) CopyObject(ctx context.Context, bucket, srcKey, dstKey string) error {
	_, err := c.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		CopySource: aws.String(bucket + "/" + srcKey),
		Key:        aws.String(dstKey),
	})
	return err
}

// DeleteObject deletes an object from S3
func (c *S3Client) DeleteObject(ctx context.Context, bucket, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}
```

**Step 3: Verify it compiles**

Run: `cd handloom-admin && go build ./internal/s3client/...`
Expected: Build succeeds (compilation errors in callers are expected — fixed in next tasks).

**Step 4: Commit**

```bash
git add handloom-admin/internal/s3client/client.go
git commit -m "feat: add endpoint override to S3 client for LocalStack"
```

---

### Task 6: Update Wire provider and cmd/api/main.go

**Files:**
- Modify: `handloom-admin/internal/wire/providers.go` (line 34-36)
- Modify: `handloom-admin/cmd/api/main.go` (line 48)

**Step 1: Update ProvideS3Client in providers.go**

Change `ProvideS3Client` from:
```go
func ProvideS3Client(ctx context.Context, cfg *config.Config) (*s3client.S3Client, error) {
	return s3client.New(ctx, cfg.AWS.Region)
}
```
to:
```go
func ProvideS3Client(ctx context.Context, cfg *config.Config) (*s3client.S3Client, error) {
	return s3client.New(ctx, cfg.AWS.Region, cfg.AWS.Endpoint)
}
```

**Step 2: Update cmd/api/main.go**

Change line 48 from:
```go
s3c, err := s3client.New(ctx, cfg.AWS.Region)
```
to:
```go
s3c, err := s3client.New(ctx, cfg.AWS.Region, cfg.AWS.Endpoint)
```

**Step 3: Regenerate Wire**

Run: `cd handloom-admin && make wire`
Expected: `wire_gen.go` regenerated successfully.

**Step 4: Verify full build**

Run: `cd handloom-admin && go build ./...`
Expected: All packages compile.

**Step 5: Commit**

```bash
git add handloom-admin/internal/wire/providers.go handloom-admin/internal/wire/wire_gen.go handloom-admin/cmd/api/main.go
git commit -m "feat: wire S3 endpoint override through DI and monolith entry point"
```

---

### Task 7: Make AssetService URL generation endpoint-aware

**Files:**
- Modify: `handloom-admin/internal/service/asset_service.go`

**Step 1: Add endpoint field to AssetService**

Change the struct and constructor to accept and store the endpoint:

Replace the `AssetService` struct (lines 46-50):
```go
type AssetService struct {
	s3Client s3Ops
	logger   *logger.Logger
	bucket   string
	endpoint string // non-empty for local dev (e.g. "http://localhost:4566")
}
```

Replace `NewAssetService` (lines 53-63):
```go
func NewAssetService(
	logger *logger.Logger,
	s3Client *s3client.S3Client,
	bucket string,
	endpoint string,
) *AssetService {
	return &AssetService{
		s3Client: s3Client,
		logger:   logger,
		bucket:   bucket,
		endpoint: endpoint,
	}
}
```

**Step 2: Make s3URL() endpoint-aware**

Replace the `s3URL` method (lines 66-68):
```go
func (s *AssetService) s3URL(key string) string {
	if s.endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", s.endpoint, s.bucket, key)
	}
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucket, key)
}
```

**Step 3: Update DeleteAsset URL parsing for local**

The `DeleteAsset` method (line 148-166) parses URLs by checking `s.s3URL("")` as prefix. This works automatically because `s3URL` now returns the correct prefix for both local and production. No change needed.

**Step 4: Update callers of NewAssetService**

In `handloom-admin/cmd/api/main.go`, change line 83 from:
```go
assetService := service.NewAssetService(log, s3c, cfg.AWS.S3Bucket)
```
to:
```go
assetService := service.NewAssetService(log, s3c, cfg.AWS.S3Bucket, cfg.AWS.Endpoint)
```

In `handloom-admin/internal/wire/providers.go`, change `ProvideAssetService` from:
```go
func ProvideAssetService(
	log *logger.Logger,
	s3Client *s3client.S3Client,
	cfg *config.Config,
) *service.AssetService {
	return service.NewAssetService(log, s3Client, cfg.AWS.S3Bucket)
}
```
to:
```go
func ProvideAssetService(
	log *logger.Logger,
	s3Client *s3client.S3Client,
	cfg *config.Config,
) *service.AssetService {
	return service.NewAssetService(log, s3Client, cfg.AWS.S3Bucket, cfg.AWS.Endpoint)
}
```

**Step 5: Regenerate Wire and verify build**

Run: `cd handloom-admin && make wire && go build ./...`
Expected: Compiles cleanly.

**Step 6: Run existing unit tests**

Run: `cd handloom-admin && make test-unit`
Expected: All tests pass. Existing tests mock `s3Ops` and don't call `NewAssetService` directly — they construct `AssetService` via the test helper or inline. If any tests break due to the new `endpoint` field, fix them by adding `""` as the endpoint arg.

**Step 7: Commit**

```bash
git add handloom-admin/internal/service/asset_service.go handloom-admin/cmd/api/main.go handloom-admin/internal/wire/providers.go handloom-admin/internal/wire/wire_gen.go
git commit -m "feat: make AssetService URL generation endpoint-aware for local dev"
```

---

### Task 8: Update Makefile

**Files:**
- Modify: `handloom-admin/Makefile`

**Step 1: Update all localhost:8000 references to localhost:4566**

Globally replace `http://localhost:8000` with `http://localhost:4566` in the Makefile. This affects:
- `make run` (line 54): `AWS_ENDPOINT=http://localhost:4566`
- `make run-watch` (line 68): `AWS_ENDPOINT=http://localhost:4566`
- `make test-integration` (line 151): `DYNAMODB_LOCAL_ENDPOINT=http://localhost:4566`
- `make reset-db` (lines 186-189): `--endpoint-url http://localhost:4566`
- `make list-tables` (line 197): `--endpoint-url http://localhost:4566`

**Step 2: Update docker-up to start LocalStack**

Replace the `docker-up` target:
```makefile
## Start Docker services (LocalStack)
docker-up:
	@echo "Starting LocalStack..."
	docker-compose up -d localstack dynamodb-admin
	@echo "Waiting for LocalStack to be ready..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		curl -sf http://localhost:4566/_localstack/health > /dev/null 2>&1 && break; \
		echo "  waiting... ($$i)"; \
		sleep 3; \
	done
	@echo "LocalStack is ready!"
```

**Step 3: Add init-s3 target**

Add after the `create-tables` target:
```makefile
## Create S3 buckets in LocalStack
init-s3:
	@echo "Creating S3 buckets..."
	chmod +x scripts/init-local-s3.sh
	./scripts/init-local-s3.sh
```

**Step 4: Update setup-local to include S3 init**

Change:
```makefile
setup-local: docker-up create-tables seed-data
```
to:
```makefile
setup-local: docker-up create-tables init-s3 seed-data
```

Update the echo output to include LocalStack info:
```makefile
setup-local: docker-up create-tables init-s3 seed-data
	@echo "Local development environment ready!"
	@echo ""
	@echo "Services:"
	@echo "  - LocalStack:    http://localhost:4566"
	@echo "  - DynamoDB Admin: http://localhost:8001"
	@echo ""
	@echo "Run 'make run' to start the API server (monolith mode)"
	@echo "Run 'make deploy-local' to deploy Lambda functions"
```

**Step 5: Add deploy-local target**

Add after `setup-local`:
```makefile
## Deploy Lambda functions to LocalStack
deploy-local: build-lambdas-active
	@echo "Deploying Lambda functions to LocalStack..."
	chmod +x scripts/deploy-local-lambdas.sh
	./scripts/deploy-local-lambdas.sh

## Redeploy Lambda code only (skip API Gateway recreation)
redeploy-local: build-lambdas-active
	@echo "Redeploying Lambda code to LocalStack..."
	REDEPLOY_ONLY=true ./scripts/deploy-local-lambdas.sh

## Tear down all local services
teardown-local:
	@echo "Tearing down local environment..."
	docker-compose down -v
	@echo "Local environment torn down."
```

**Step 6: Update the banner in `make run`**

Update the echo block in the `run` target to show LocalStack instead of DynamoDB Local:
```
@echo "  API Server:     http://localhost:8080"
@echo "  LocalStack:     http://localhost:4566"
@echo "  DynamoDB Admin: http://localhost:8001"
```

**Step 7: Update the help target**

Add the new targets to the help output under the "Docker & Database" section:
```
@echo "  make init-s3            - Create S3 buckets in LocalStack"
```
And add a new "Local Lambda" section:
```
@echo "Local Lambda:"
@echo "  make deploy-local       - Deploy Lambdas to LocalStack"
@echo "  make redeploy-local     - Redeploy Lambda code only"
@echo "  make teardown-local     - Tear down all Docker services"
```

**Step 8: Add .PHONY entries**

Add `init-s3 deploy-local redeploy-local teardown-local` to the `.PHONY` line.

**Step 9: Verify Makefile syntax**

Run: `cd handloom-admin && make help`
Expected: Help output shows all targets including new ones.

**Step 10: Commit**

```bash
git add handloom-admin/Makefile
git commit -m "feat: update Makefile for LocalStack — add deploy-local, teardown-local"
```

---

### Task 9: Create Lambda deployment script

**Files:**
- Create: `handloom-admin/scripts/deploy-local-lambdas.sh`

**Step 1: Write the deployment script**

```bash
#!/bin/bash

# Deploy Lambda functions to LocalStack
# Usage: ./scripts/deploy-local-lambdas.sh
# Set REDEPLOY_ONLY=true to skip API Gateway recreation

set -e

ENDPOINT="${AWS_ENDPOINT:-http://localhost:4566}"
REGION="${AWS_REGION:-ap-south-1}"
LAMBDA_DIR="./bin/lambda"
ACTIVE_SERVICES="auth user catalog asset"

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
    "DYNAMODB_ANALYTICS_TABLE": "handloom-analytics",
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
echo ""

# Save the API ID for redeploy reference
echo "$API_ID" > /tmp/handloom-local-api-id
echo "$INVOKE_URL" > /tmp/handloom-local-api-url
```

**Step 2: Make executable**

Run: `chmod +x handloom-admin/scripts/deploy-local-lambdas.sh`

**Step 3: Commit**

```bash
git add handloom-admin/scripts/deploy-local-lambdas.sh
git commit -m "feat: add Lambda deployment script for LocalStack"
```

---

### Task 10: Frontend env files and scripts

**Files:**
- Create: `handloom-admin-frontend/.env.local-backend`
- Create: `handloom-admin-frontend/.env.local-lambda`
- Modify: `handloom-admin-frontend/package.json`

**Step 1: Create .env.local-backend**

```
# Local Backend (Monolith) Configuration
# Used by: npm run dev:local
VITE_API_URL=http://localhost:8080
VITE_APP_ENV=local
VITE_APP_NAME=Handloom Admin (Local)
```

**Step 2: Create .env.local-lambda**

```
# Local Lambda (LocalStack) Configuration
# Used by: npm run dev:lambda
# NOTE: Update the URL after running 'make deploy-local' — it prints the API Gateway URL
VITE_API_URL=http://localhost:4566
VITE_APP_ENV=local-lambda
VITE_APP_NAME=Handloom Admin (Lambda)
```

**Step 3: Add dev:local and dev:lambda scripts to package.json**

Add two new scripts to the `"scripts"` section:
```json
"dev:local": "vite --mode local-backend",
"dev:lambda": "vite --mode local-lambda",
```

**Step 4: Verify Vite picks up the new modes**

Run: `cd handloom-admin-frontend && npx vite --mode local-backend --help`
Expected: No errors. Vite recognizes the mode and would load `.env.local-backend`.

**Step 5: Commit**

```bash
git add handloom-admin-frontend/.env.local-backend handloom-admin-frontend/.env.local-lambda handloom-admin-frontend/package.json
git commit -m "feat: add frontend dev:local and dev:lambda modes"
```

---

### Task 11: Update frontend ImageUpload S3 URL check

**Files:**
- Modify: `handloom-admin-frontend/src/shared/components/ui/ImageUpload.tsx` (line 242)

**Step 1: Update the permanent URL detection for local dev**

The `handleRemove` function checks `val.includes('.s3.amazonaws.com/assets/')` to decide whether to call the delete API. For local dev, finalized URLs look like `http://localhost:4566/handloom-assets/assets/...`. Update the check to handle both:

Change line 242 from:
```ts
if (val && isPermanentUrl(val) && val.includes('.s3.amazonaws.com/assets/')) {
```
to:
```ts
if (val && isPermanentUrl(val) && (val.includes('.s3.amazonaws.com/assets/') || val.includes('/assets/'))) {
```

Actually, this is too broad — `/assets/` could match other URLs. Better approach: check if it contains the bucket name + `/assets/`:

```ts
if (val && isPermanentUrl(val) && val.includes('/assets/')) {
```

This is safe because `isPermanentUrl` already checks `val.startsWith('http')` and the only URLs that contain `/assets/` are finalized S3 URLs (both `https://bucket.s3.amazonaws.com/assets/...` and `http://localhost:4566/bucket/assets/...`).

**Step 2: Commit**

```bash
git add handloom-admin-frontend/src/shared/components/ui/ImageUpload.tsx
git commit -m "fix: handle local S3 URLs in image delete check"
```

---

### Task 12: End-to-end verification — Monolith mode

**Step 1: Full setup**

Run:
```bash
cd handloom-admin
make teardown-local   # clean slate
make setup-local      # LocalStack + tables + S3 buckets + seed data
```
Expected: All services start, tables and buckets created, data seeded.

**Step 2: Start backend monolith**

Run: `cd handloom-admin && make run`
Expected: Server starts on `:8080`, logs show "S3 client initialized" (no crash).

**Step 3: Test health and auth**

Run:
```bash
curl http://localhost:8080/health
curl -s -X POST http://localhost:8080/admin/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@handloom.com","password":"Admin@123!"}'
```
Expected: Health returns `{"status":"ok"}`. Login returns JWT tokens.

**Step 4: Test S3 presigned URL**

Run (using the access_token from login):
```bash
curl -s -X POST http://localhost:8080/admin/assets/upload-url \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{"file_name":"test.jpg","content_type":"image/jpeg","size":1024,"type":"IMAGE"}'
```
Expected: Returns JSON with `upload_url` containing `localhost:4566` and `tmp_key` starting with `tmp/IMAGE/`.

**Step 5: Test presigned URL upload**

Use the `upload_url` from the previous step:
```bash
curl -X PUT "<upload_url>" \
  -H "Content-Type: image/jpeg" \
  --data-binary @/dev/urandom | head -c 1024
```
Expected: HTTP 200 OK.

**Step 6: Start frontend and verify**

Run: `cd handloom-admin-frontend && npm run dev:local`
Expected: Vite starts on `:5173`, proxying to `localhost:8080`. Login works in browser.

---

### Task 13: End-to-end verification — Lambda mode

**Step 1: Deploy Lambdas**

Run: `cd handloom-admin && make deploy-local`
Expected: Builds 4 Lambda binaries, deploys them to LocalStack, creates API Gateway. Outputs invoke URL.

**Step 2: Test Lambda health**

Run: `curl <INVOKE_URL>/health`
Expected: Returns health check response (may be from auth Lambda).

**Step 3: Test Lambda auth**

Run:
```bash
curl -s -X POST <INVOKE_URL>/admin/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@handloom.com","password":"Admin@123!"}'
```
Expected: Returns JWT tokens, same as monolith mode.

**Step 4: Start frontend in Lambda mode**

Update `handloom-admin-frontend/.env.local-lambda` with the actual `INVOKE_URL` from deploy output, then:
Run: `cd handloom-admin-frontend && npm run dev:lambda`
Expected: Frontend proxies to LocalStack API Gateway. Login works.

**Step 5: Test code update cycle**

Make a trivial change to a handler, then:
Run: `cd handloom-admin && make redeploy-local`
Expected: Rebuilds Lambda binaries and updates function code. API responds with updated behavior.
