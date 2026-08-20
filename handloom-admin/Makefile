.PHONY: all build run test clean docker-up docker-down create-tables seed-data generate-mocks lint \
	build-lambdas build-lambda-auth build-lambda-user build-lambda-catalog build-lambda-order \
	build-lambda-pricing build-lambda-inventory build-lambda-analytics build-lambda-notification \
	build-lambda-coupon build-lambda-asset \
	build-lambda-report build-lambda-audit \
	build-lambda-store-auth build-lambda-store-catalog build-lambda-store-cart \
	build-lambda-store-checkout build-lambda-store-orders build-lambda-store-tracking \
	build-lambda-store-profile build-lambda-store-events build-lambda-store-webhooks \
	build-lambda-worker-metrics-consumer \
	build-lambda-backfill \
	cdk-synth cdk-deploy-dev cdk-deploy-prod cdk-destroy \
	init-s3 deploy-local redeploy-local teardown-local \
	logs logs-lambda \
	install-libtokenizers

# AWS region for deployment
AWS_REGION ?= ap-south-1

# Environment (dev | prod)
ENV ?= dev

# Config (certs, domains) lives in infra/cmd/config.go

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=handloom-api
BINARY_PATH=./bin/$(BINARY_NAME)

# Lambda build parameters
LAMBDA_BUILD_FLAGS=CGO_ENABLED=0 GOOS=linux GOARCH=arm64
LAMBDA_OUT_DIR=./bin/lambda
LAMBDA_SERVICES=auth user catalog order pricing inventory analytics notification coupon asset report audit \
	store-auth store-catalog store-cart store-checkout store-orders store-tracking store-profile store-events store-webhooks \
	worker-metrics-consumer \
	migrator

# Active services for deployment
LAMBDA_SERVICES_ACTIVE=auth user catalog asset order store-auth store-catalog store-cart store-checkout store-orders store-tracking store-profile store-events store-webhooks migrator worker-metrics-consumer worker-embedding-backfill

# Main package path
MAIN_PKG=./cmd/api

all: build

## Build the application
build:
	@echo "Building..."
	$(GOBUILD) -o $(BINARY_PATH) $(MAIN_PKG)

## Run the application (requires DynamoDB local)
run:
	@echo "=============================================="
	@echo "  Starting Handloom Admin API"
	@echo "=============================================="
	@echo ""
	@echo "  API Server:     http://localhost:8081"
	@echo "  LocalStack:     http://localhost:4566"
	@echo "  DynamoDB Admin: http://localhost:8001"
	@echo "  pgAdmin:        http://localhost:5050"
	@echo ""
	@echo "  Health Check:   curl http://localhost:8081/health"
	@echo ""
	@echo "  Default Admin:  admin@handloom.com / Admin@123!"
	@echo ""
	@echo "=============================================="
	APP_ENV=development \
	APP_DEBUG=true \
	SERVER_PORT=8081 \
	AWS_ENDPOINT=http://localhost:4566 \
	AWS_REGION=$(AWS_REGION) \
	AWS_ACCESS_KEY_ID=local \
	AWS_SECRET_ACCESS_KEY=local \
	JWT_SECRET_KEY=dev-secret-key-change-in-production \
	POSTGRES_DSN=postgres://handloom:handloom@localhost:5432/handloom?sslmode=disable \
	$(GOCMD) run $(MAIN_PKG)

## Run with hot reload using air (requires: go install github.com/cosmtrek/air@latest)
run-watch:
	@echo "Starting with hot reload (air)..."
	@which air > /dev/null 2>&1 || (echo "Installing air..." && $(GOCMD) install github.com/cosmtrek/air@latest)
	APP_ENV=development \
	APP_DEBUG=true \
	SERVER_PORT=8081 \
	AWS_ENDPOINT=http://localhost:4566 \
	AWS_REGION=$(AWS_REGION) \
	AWS_ACCESS_KEY_ID=local \
	AWS_SECRET_ACCESS_KEY=local \
	JWT_SECRET_KEY=dev-secret-key-change-in-production \
	POSTGRES_DSN=postgres://handloom:handloom@localhost:5432/handloom?sslmode=disable \
	air

## Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

## Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf ./bin
	rm -f coverage.out coverage.html
	rm -rf ./infra/cdk.out

## Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## Generate Wire dependency injection code
wire:
	@echo "Generating Wire code..."
	cd internal/wire && wire

## Generate mocks using mockgen
generate-mocks:
	@echo "Generating mocks..."
	mockgen -source=internal/domain/repository.go -destination=internal/mocks/repository_mock.go -package=mocks
	mockgen -source=internal/domain/service.go -destination=internal/mocks/service_mock.go -package=mocks
	mockgen -source=internal/domain/asset.go -destination=internal/mocks/asset_domain_mock.go -package=mocks
	mockgen -source=internal/domain/audit_repository.go -destination=internal/mocks/audit_repository_mock.go -package=mocks
	mockgen -source=internal/domain/coupon.go -destination=internal/mocks/coupon_mock.go -package=mocks
	mockgen -source=internal/domain/notification.go -destination=internal/mocks/notification_mock.go -package=mocks
	mockgen -source=internal/domain/order_repository.go -destination=internal/mocks/order_repository_mock.go -package=mocks
	mockgen -source=internal/domain/token_store.go -destination=internal/mocks/token_store_mock.go -package=mocks
	mockgen -source=internal/domain/store_service.go -destination=internal/mocks/store_service_mock.go -package=mocks
	mockgen -source=internal/domain/store_repository.go -destination=internal/mocks/store_repository_mock.go -package=mocks
	mockgen -source=internal/domain/refund.go -destination=internal/mocks/refund_mock.go -package=mocks

## Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

## Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

## Start Docker services (LocalStack + PostgreSQL + Admin UIs)
docker-up:
	@echo "Starting LocalStack + PostgreSQL + Admin UIs..."
	docker-compose up -d localstack dynamodb-admin postgres pgadmin
	@echo "Waiting for LocalStack to be ready..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		curl -sf http://localhost:4566/_localstack/health > /dev/null 2>&1 && break; \
		echo "  waiting... ($$i)"; \
		sleep 3; \
	done
	@echo "LocalStack is ready!"
	@echo "Waiting for PostgreSQL to be ready..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		docker exec handloom-postgres pg_isready -U handloom > /dev/null 2>&1 && break; \
		echo "  waiting... ($$i)"; \
		sleep 2; \
	done
	@echo "PostgreSQL is ready!"

## Stop Docker services
docker-down:
	@echo "Stopping Docker services..."
	docker-compose down

## Start all services including API
docker-up-all:
	@echo "Starting all services..."
	docker-compose up -d --build

## Create DynamoDB tables
create-tables:
	@echo "Creating DynamoDB tables..."
	chmod +x scripts/init-local-db.sh
	./scripts/init-local-db.sh

## Create S3 buckets in LocalStack
init-s3:
	@echo "Creating S3 buckets..."
	chmod +x scripts/init-local-s3.sh
	./scripts/init-local-s3.sh

## Run unit tests only (no DynamoDB required)
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -race -cover ./internal/service/...

## Run integration tests (requires `make docker-up`: Postgres + DynamoDB Local)
#
# Fixes the four ways this target used to not do what its name said (#191):
#   1. no `-tags integration`, so every tagged test was excluded from the build
#   2. no POSTGRES_DSN, so the Postgres tests fell back to postgres:postgres and
#      skipped — docker-compose provisions handloom:handloom
#   3. the DynamoDB endpoint pointed at LocalStack (:4566) while testClient
#      defaults to amazon/dynamodb-local (:8000)
#   4. ./internal/repository/... excludes cmd/embedder/embedder, where the
#      hybrid-search integration test lives, so it was unreachable through its
#      own documented runner
#
# CI overrides both endpoints; these defaults match docker-compose.yml.
INTEGRATION_POSTGRES_DSN ?= postgres://handloom:handloom@localhost:5432/handloom?sslmode=disable
INTEGRATION_DYNAMODB_ENDPOINT ?= http://localhost:8000

test-integration:
	@echo "Running integration tests..."
	POSTGRES_DSN="$(INTEGRATION_POSTGRES_DSN)" \
	DYNAMODB_LOCAL_ENDPOINT="$(INTEGRATION_DYNAMODB_ENDPOINT)" \
	$(GOTEST) -v -race -tags integration \
		./internal/repository/... \
		./cmd/embedder/embedder/...

## Seed test data
seed-data:
	@echo "Seeding test data..."
	chmod +x scripts/seed-data.sh
	./scripts/seed-data.sh

## Setup local development (start services, create tables, init S3, seed data)
setup-local: docker-up create-tables init-s3 seed-data
	@echo "Local development environment ready!"
	@echo ""
	@echo "Services:"
	@echo "  - LocalStack:     http://localhost:4566"
	@echo "  - DynamoDB Admin: http://localhost:8001"
	@echo "  - pgAdmin:        http://localhost:5050"
	@echo "  - PostgreSQL:     localhost:5432 (handloom/handloom)"
	@echo ""
	@echo "Run 'make run' to start the API server (monolith mode)"
	@echo "Run 'make deploy-local' to deploy Lambda functions"

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

## Tail all LocalStack logs (Lambda invocations, API Gateway, etc.)
logs:
	docker logs -f handloom-localstack 2>&1

## Tail logs for a specific Lambda function (requires SVC=<name>)
## Usage: make logs-lambda SVC=catalog
##        make logs-lambda SVC=store-catalog
logs-lambda:
ifndef SVC
	$(error Usage: make logs-lambda SVC=<service-name>  (e.g., SVC=catalog, SVC=store-cart))
endif
	@echo "Tailing logs for handloom-$(SVC)..."
	@echo "(Press Ctrl+C to stop)"
	awslocal --region ap-south-1 logs tail /aws/lambda/handloom-$(SVC) --follow

## Install development tools
install-tools:
	@echo "Installing development tools..."
	$(GOCMD) install github.com/google/wire/cmd/wire@latest
	$(GOCMD) install go.uber.org/mock/mockgen@latest
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOCMD) install github.com/cosmtrek/air@latest
	@echo "Development tools installed!"

## Check health of local services
health:
	@chmod +x scripts/health-check.sh
	@./scripts/health-check.sh

## Reset local database (drop all tables and recreate)
reset-db: docker-up
	@echo "Resetting local database..."
	@echo "Resetting DynamoDB tables..."
	@AWS_ACCESS_KEY_ID=local AWS_SECRET_ACCESS_KEY=local aws dynamodb delete-table --table-name handloom-core --endpoint-url http://localhost:4566 --region ap-south-1 2>/dev/null || true
	@AWS_ACCESS_KEY_ID=local AWS_SECRET_ACCESS_KEY=local aws dynamodb delete-table --table-name handloom-orders --endpoint-url http://localhost:4566 --region ap-south-1 2>/dev/null || true
	@AWS_ACCESS_KEY_ID=local AWS_SECRET_ACCESS_KEY=local aws dynamodb delete-table --table-name handloom-audit --endpoint-url http://localhost:4566 --region ap-south-1 2>/dev/null || true
	@AWS_ACCESS_KEY_ID=local AWS_SECRET_ACCESS_KEY=local aws dynamodb delete-table --table-name handloom-sessions --endpoint-url http://localhost:4566 --region ap-south-1 2>/dev/null || true
	@echo "Resetting PostgreSQL catalog tables..."
	@docker exec handloom-postgres psql -U handloom -d handloom -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;" 2>/dev/null || true
	@docker exec handloom-postgres psql -U handloom -d handloom -f /docker-entrypoint-initdb.d/001_catalog_schema.sql 2>/dev/null || true
	@sleep 2
	@$(MAKE) create-tables
	@$(MAKE) seed-data
	@echo "Database reset complete!"

## View DynamoDB tables
list-tables:
	@AWS_ACCESS_KEY_ID=local AWS_SECRET_ACCESS_KEY=local aws dynamodb list-tables --endpoint-url http://localhost:4566 --region ap-south-1

## Generate password hash for development
gen-password:
	@echo "Usage: go run scripts/generate-password-hash.go <password>"
	@echo "Example: go run scripts/generate-password-hash.go 'MyNewPassword123!'"

## Login test - get JWT token
test-login:
	@echo "Testing login with default admin credentials..."
	@curl -s -X POST http://localhost:8081/admin/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"admin@handloom.com","password":"Admin@123!"}' | jq .

## Quick API test
test-api:
	@echo "Testing API endpoints..."
	@echo ""
	@echo "Health check:"
	@curl -s http://localhost:8081/health | jq .
	@echo ""
	@echo "Categories (requires auth):"
	@TOKEN=$$(curl -s -X POST http://localhost:8081/admin/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"admin@handloom.com","password":"Admin@123!"}' | jq -r '.access_token'); \
	curl -s http://localhost:8081/admin/categories \
		-H "Authorization: Bearer $$TOKEN" | jq .

## Build all Lambda functions
build-lambdas:
	@echo "Building all Lambda functions..."
	@mkdir -p $(LAMBDA_OUT_DIR)
	@for service in $(LAMBDA_SERVICES); do \
		echo "Building $$service Lambda..."; \
		$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/$$service/bootstrap ./cmd/lambda/$$service; \
		cp infra/configs/otel-collector.yaml $(LAMBDA_OUT_DIR)/$$service/otel.yaml; \
	done
	@echo "All Lambda functions built successfully!"

## Build only active Lambda functions (auth, user, catalog, asset)
build-lambdas-active:
	@echo "Building active Lambda functions (admin + store)..."
	@mkdir -p $(LAMBDA_OUT_DIR)
	@for service in $(LAMBDA_SERVICES_ACTIVE); do \
		echo "Building $$service Lambda..."; \
		$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/$$service/bootstrap ./cmd/lambda/$$service; \
		cp infra/configs/otel-collector.yaml $(LAMBDA_OUT_DIR)/$$service/otel.yaml; \
	done
	@echo "Active Lambda functions built successfully!"

## Build individual Lambda functions
build-lambda-auth:
	@echo "Building auth Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/auth
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/auth/bootstrap ./cmd/lambda/auth

build-lambda-user:
	@echo "Building user Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/user
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/user/bootstrap ./cmd/lambda/user

build-lambda-catalog:
	@echo "Building catalog Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/catalog
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/catalog/bootstrap ./cmd/lambda/catalog

build-lambda-order:
	@echo "Building order Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/order
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/order/bootstrap ./cmd/lambda/order

build-lambda-pricing:
	@echo "Building pricing Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/pricing
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/pricing/bootstrap ./cmd/lambda/pricing

build-lambda-inventory:
	@echo "Building inventory Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/inventory
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/inventory/bootstrap ./cmd/lambda/inventory

build-lambda-analytics:
	@echo "Building analytics Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/analytics
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/analytics/bootstrap ./cmd/lambda/analytics

build-lambda-notification:
	@echo "Building notification Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/notification
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/notification/bootstrap ./cmd/lambda/notification

build-lambda-coupon:
	@echo "Building coupon Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/coupon
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/coupon/bootstrap ./cmd/lambda/coupon

build-lambda-asset:
	@echo "Building asset Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/asset
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/asset/bootstrap ./cmd/lambda/asset

build-lambda-report:
	@echo "Building report Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/report
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/report/bootstrap ./cmd/lambda/report

build-lambda-audit:
	@echo "Building audit Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/audit
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/audit/bootstrap ./cmd/lambda/audit

## Build individual Store Lambda functions
build-lambda-store-auth:
	@echo "Building store-auth Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/store-auth
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/store-auth/bootstrap ./cmd/lambda/store-auth

build-lambda-store-catalog:
	@echo "Building store-catalog Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/store-catalog
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/store-catalog/bootstrap ./cmd/lambda/store-catalog

build-lambda-store-cart:
	@echo "Building store-cart Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/store-cart
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/store-cart/bootstrap ./cmd/lambda/store-cart

build-lambda-store-checkout:
	@echo "Building store-checkout Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/store-checkout
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/store-checkout/bootstrap ./cmd/lambda/store-checkout

build-lambda-store-orders:
	@echo "Building store-orders Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/store-orders
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/store-orders/bootstrap ./cmd/lambda/store-orders

build-lambda-store-tracking:
	@echo "Building store-tracking Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/store-tracking
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/store-tracking/bootstrap ./cmd/lambda/store-tracking

build-lambda-store-profile:
	@echo "Building store-profile Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/store-profile
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/store-profile/bootstrap ./cmd/lambda/store-profile

build-lambda-store-events:
	@echo "Building store-events Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/store-events
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/store-events/bootstrap ./cmd/lambda/store-events

build-lambda-store-webhooks:
	@echo "Building store-webhooks Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/store-webhooks
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/store-webhooks/bootstrap ./cmd/lambda/store-webhooks

build-lambda-worker-metrics-consumer:
	@echo "Building worker-metrics-consumer Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/worker-metrics-consumer
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/worker-metrics-consumer/bootstrap ./cmd/lambda/worker-metrics-consumer

## Build only the embedding backfill Lambda. Also built by build-lambdas-active
## (APIStack creates the Lambda on every deploy, so its binary is always needed).
.PHONY: prepare-embedder-assets build-lambda-backfill
build-lambda-backfill:
	@echo "Building worker-embedding-backfill Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/worker-embedding-backfill
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/worker-embedding-backfill/bootstrap ./cmd/lambda/worker-embedding-backfill
	@cp infra/configs/otel-collector.yaml $(LAMBDA_OUT_DIR)/worker-embedding-backfill/otel.yaml

## CDK commands
cdk-synth:
	@echo "Synthesizing CDK stacks..."
	export AWS_REGION=$(AWS_REGION) && cd infra && cdk synth

cdk-synth-dev:
	@echo "Synthesizing CDK stacks for dev..."
	export AWS_REGION=$(AWS_REGION) && cd infra && cdk synth -c environment=dev

cdk-synth-prod:
	@echo "Synthesizing CDK stacks for prod..."
	export AWS_REGION=$(AWS_REGION) && cd infra && cdk synth -c environment=prod

cdk-deploy-dev: build-lambdas-active prepare-embedder-assets
	@echo "Deploying to development environment..."
	set -a && . ./.env.dev && set +a && export AWS_REGION=$(AWS_REGION) && cd infra && cdk deploy --all --require-approval never -c environment=dev

cdk-deploy-prod: build-lambdas-active prepare-embedder-assets
	@echo "Deploying to production environment..."
	set -a && . ./.env.prod && set +a && export AWS_REGION=$(AWS_REGION) && cd infra && cdk deploy --all --require-approval broadening -c environment=prod

cdk-destroy-dev:
	@echo "Destroying dev CDK stacks..."
	export AWS_REGION=$(AWS_REGION) && cd infra && cdk destroy --all -c environment=dev

cdk-destroy-prod:
	@echo "Destroying prod CDK stacks..."
	export AWS_REGION=$(AWS_REGION) && cd infra && cdk destroy --all -c environment=prod

cdk-destroy:
	@echo "Destroying CDK stacks..."
	export AWS_REGION=$(AWS_REGION) && cd infra && cdk destroy --all

cdk-diff:
	@echo "Showing CDK diff..."
	export AWS_REGION=$(AWS_REGION) && cd infra && cdk diff

cdk-diff-dev:
	@echo "Showing CDK diff for dev..."
	export AWS_REGION=$(AWS_REGION) && cd infra && cdk diff -c environment=dev

cdk-diff-prod:
	@echo "Showing CDK diff for prod..."
	export AWS_REGION=$(AWS_REGION) && cd infra && cdk diff -c environment=prod

cdk-bootstrap:
	@echo "Bootstrapping CDK (required once per account/region)..."
	export AWS_REGION=$(AWS_REGION) && cd infra && cdk bootstrap

## Show available commands
help:
	@echo "=============================================="
	@echo "  Handloom Admin API - Available Commands"
	@echo "=============================================="
	@echo ""
	@echo "Quick Start:"
	@echo "  make setup-local        - Full local setup (docker + tables + seed)"
	@echo "  make run                - Start API server locally"
	@echo "  make run-watch          - Start with hot reload (air)"
	@echo ""
	@echo "Development:"
	@echo "  make build              - Build the application"
	@echo "  make test               - Run tests"
	@echo "  make test-coverage      - Run tests with coverage report"
	@echo "  make test-unit          - Run unit tests only"
	@echo "  make test-integration   - Run integration tests"
	@echo "  make lint               - Run linter"
	@echo "  make fmt                - Format code"
	@echo ""
	@echo "Docker & Database:"
	@echo "  make docker-up          - Start DynamoDB Local"
	@echo "  make docker-down        - Stop Docker services"
	@echo "  make create-tables      - Create DynamoDB tables"
	@echo "  make init-s3            - Create S3 buckets in LocalStack"
	@echo "  make seed-data          - Seed test data"
	@echo "  make reset-db           - Drop and recreate all tables"
	@echo "  make list-tables        - List DynamoDB tables"
	@echo ""
	@echo "Local Lambda:"
	@echo "  make deploy-local       - Deploy Lambdas to LocalStack"
	@echo "  make redeploy-local     - Redeploy Lambda code only"
	@echo "  make teardown-local     - Tear down all Docker services"
	@echo "  make logs               - Tail all LocalStack logs"
	@echo "  make logs-lambda SVC=x  - Tail logs for a specific Lambda (e.g., SVC=catalog)"
	@echo ""
	@echo "Testing & Debug:"
	@echo "  make health             - Check health of local services"
	@echo "  make test-login         - Test login with admin credentials"
	@echo "  make test-api           - Run quick API tests"
	@echo ""
	@echo "Code Generation:"
	@echo "  make wire               - Generate Wire dependency injection"
	@echo "  make generate-mocks     - Generate mock implementations"
	@echo ""
	@echo "Tools:"
	@echo "  make install-tools      - Install development tools"
	@echo "  make deps               - Download dependencies"
	@echo "  make clean              - Clean build artifacts"
	@echo ""
	@echo "Lambda & Deployment:"
	@echo "  make build-lambdas        - Build all Lambda functions"
	@echo "  make build-lambdas-active - Build active Lambdas (admin + store)"
	@echo "  make build-lambda-<svc>   - Build specific Lambda (auth, user, etc.)"
	@echo ""
	@echo "  make cdk-bootstrap        - Bootstrap CDK (once per account/region)"
	@echo "  make cdk-synth            - Synthesize CDK stacks"
	@echo "  make cdk-synth-dev        - Synthesize for dev environment"
	@echo "  make cdk-synth-prod       - Synthesize for prod environment"
	@echo "  make cdk-synth-all        - Synthesize for all environments"
	@echo ""
	@echo "  make cdk-deploy-dev       - Deploy to development"
	@echo "  make cdk-deploy-prod      - Deploy to production"
	@echo "  make cdk-deploy-all       - Deploy to all environments"
	@echo ""
	@echo "  make cdk-diff             - Show CDK diff"
	@echo "  make cdk-diff-dev         - Show CDK diff for dev"
	@echo "  make cdk-diff-prod        - Show CDK diff for prod"
	@echo ""
	@echo "  make cdk-destroy          - Destroy CDK stacks"
	@echo "  make cdk-destroy-dev      - Destroy dev stacks"
	@echo "  make cdk-destroy-prod     - Destroy prod stacks"
	@echo ""
	@echo "Default Admin Credentials:"
	@echo "  Email:    admin@handloom.com"
	@echo "  Password: Admin@123!"
	@echo ""

# ---- Embedder Lambda (container image built by CDK) ----
TOKENIZERS_VERSION := v1.27.0

# Populates cmd/embedder/assets/ from ~/.cache/handloom-embedder (which is
# populated by scripts/bootstrap-embedder-assets.sh — slow first run, fast
# subsequent runs). Required before `cdk synth` builds the embedder image.
.PHONY: prepare-embedder-assets
prepare-embedder-assets:
	@../scripts/bootstrap-embedder-assets.sh

# Install libtokenizers.a so local `go build` of the embedder works on macOS/Linux dev hosts.
.PHONY: install-libtokenizers
install-libtokenizers:
	@OS=$$(uname -s | tr A-Z a-z); ARCH=$$(uname -m); \
	case "$$OS-$$ARCH" in \
		darwin-arm64) ASSET=libtokenizers.darwin-arm64.tar.gz ;; \
		darwin-x86_64) ASSET=libtokenizers.darwin-x86_64.tar.gz ;; \
		linux-aarch64|linux-arm64) ASSET=libtokenizers.linux-arm64.tar.gz ;; \
		linux-x86_64) ASSET=libtokenizers.linux-x86_64.tar.gz ;; \
		*) echo "unsupported platform $$OS-$$ARCH"; exit 1 ;; \
	esac; \
	URL="https://github.com/daulet/tokenizers/releases/download/$(TOKENIZERS_VERSION)/$$ASSET"; \
	echo "Fetching $$URL"; \
	tmp=$$(mktemp -d); \
	curl -fsSL "$$URL" -o "$$tmp/libtok.tar.gz" && tar xzf "$$tmp/libtok.tar.gz" -C "$$tmp" && \
	sudo install -m 0644 "$$tmp/libtokenizers.a" /usr/local/lib/libtokenizers.a; \
	rm -rf "$$tmp"

# NOTE: the previous build-embedder-image / push-embedder-image targets were
# removed when the embedder switched to CDK Code_FromAssetImage. CDK now builds
# the Dockerfile inline during `cdk synth` and pushes the image to its
# bootstrap ECR repo automatically. To deploy a new embedder version, run
# `make cdk-deploy-{env}` — CDK detects the source-content hash change and
# rebuilds + pushes + updates the Lambda.

