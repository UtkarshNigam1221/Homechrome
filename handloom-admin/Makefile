.PHONY: all build run test clean docker-up docker-down create-tables seed-data generate-mocks lint \
	build-lambdas build-lambda-auth build-lambda-user build-lambda-catalog build-lambda-order \
	build-lambda-pricing build-lambda-inventory build-lambda-analytics build-lambda-notification \
	build-lambda-coupon build-lambda-artisan build-lambda-asset \
	build-lambda-report build-lambda-audit cdk-synth cdk-deploy-dev cdk-deploy-prod cdk-destroy

# ACM certificate ARN for custom domain (*.lldlab.com) - must be in us-east-1
CERT_ARN ?= arn:aws:acm:us-east-1:163053486005:certificate/a8dca48d-5992-4e6e-94c9-fffc7f3ac447

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
LAMBDA_SERVICES=auth user catalog order pricing inventory analytics notification coupon artisan asset report audit

# Active services for deployment
LAMBDA_SERVICES_ACTIVE=auth user catalog asset

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
	@echo "  API Server:     http://localhost:8080"
	@echo "  DynamoDB Local: http://localhost:8000"
	@echo "  DynamoDB Admin: http://localhost:8001"
	@echo ""
	@echo "  Health Check:   curl http://localhost:8080/health"
	@echo ""
	@echo "  Default Admin:  admin@handloom.com / Admin@123!"
	@echo ""
	@echo "=============================================="
	APP_ENV=development \
	APP_DEBUG=true \
	AWS_ENDPOINT=http://localhost:8000 \
	AWS_REGION=ap-south-1 \
	AWS_ACCESS_KEY_ID=local \
	AWS_SECRET_ACCESS_KEY=local \
	JWT_SECRET_KEY=dev-secret-key-change-in-production \
	$(GOCMD) run $(MAIN_PKG)

## Run with hot reload using air (requires: go install github.com/cosmtrek/air@latest)
run-watch:
	@echo "Starting with hot reload (air)..."
	@which air > /dev/null 2>&1 || (echo "Installing air..." && $(GOCMD) install github.com/cosmtrek/air@latest)
	APP_ENV=development \
	APP_DEBUG=true \
	AWS_ENDPOINT=http://localhost:8000 \
	AWS_REGION=ap-south-1 \
	AWS_ACCESS_KEY_ID=local \
	AWS_SECRET_ACCESS_KEY=local \
	JWT_SECRET_KEY=dev-secret-key-change-in-production \
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

## Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

## Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

## Start Docker services (DynamoDB Local)
docker-up:
	@echo "Starting Docker services..."
	docker-compose up -d dynamodb-local dynamodb-admin
	@echo "Waiting for DynamoDB to be ready..."
	@sleep 3

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

## Run unit tests only (no DynamoDB required)
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -race -cover ./internal/service/...

## Run integration tests (requires DynamoDB Local)
test-integration:
	@echo "Running integration tests..."
	DYNAMODB_LOCAL_ENDPOINT=http://localhost:8000 $(GOTEST) -v -race ./internal/repository/...

## Seed test data
seed-data:
	@echo "Seeding test data..."
	chmod +x scripts/seed-data.sh
	./scripts/seed-data.sh

## Setup local development (start services, create tables, seed data)
setup-local: docker-up create-tables seed-data
	@echo "Local development environment ready!"
	@echo ""
	@echo "Services:"
	@echo "  - DynamoDB Local: http://localhost:8000"
	@echo "  - DynamoDB Admin: http://localhost:8001"
	@echo ""
	@echo "Run 'make run' to start the API server"

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
	@aws dynamodb delete-table --table-name handloom-core --endpoint-url http://localhost:8000 --region ap-south-1 2>/dev/null || true
	@aws dynamodb delete-table --table-name handloom-orders --endpoint-url http://localhost:8000 --region ap-south-1 2>/dev/null || true
	@aws dynamodb delete-table --table-name handloom-audit --endpoint-url http://localhost:8000 --region ap-south-1 2>/dev/null || true
	@aws dynamodb delete-table --table-name handloom-analytics --endpoint-url http://localhost:8000 --region ap-south-1 2>/dev/null || true
	@sleep 2
	@$(MAKE) create-tables
	@$(MAKE) seed-data
	@echo "Database reset complete!"

## View DynamoDB tables
list-tables:
	@aws dynamodb list-tables --endpoint-url http://localhost:8000 --region ap-south-1

## Generate password hash for development
gen-password:
	@echo "Usage: go run scripts/generate-password-hash.go <password>"
	@echo "Example: go run scripts/generate-password-hash.go 'MyNewPassword123!'"

## Login test - get JWT token
test-login:
	@echo "Testing login with default admin credentials..."
	@curl -s -X POST http://localhost:8080/admin/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"admin@handloom.com","password":"Admin@123!"}' | jq .

## Quick API test
test-api:
	@echo "Testing API endpoints..."
	@echo ""
	@echo "Health check:"
	@curl -s http://localhost:8080/health | jq .
	@echo ""
	@echo "Categories (requires auth):"
	@TOKEN=$$(curl -s -X POST http://localhost:8080/admin/auth/login \
		-H "Content-Type: application/json" \
		-d '{"email":"admin@handloom.com","password":"Admin@123!"}' | jq -r '.access_token'); \
	curl -s http://localhost:8080/admin/categories \
		-H "Authorization: Bearer $$TOKEN" | jq .

## Build all Lambda functions
build-lambdas:
	@echo "Building all Lambda functions..."
	@mkdir -p $(LAMBDA_OUT_DIR)
	@for service in $(LAMBDA_SERVICES); do \
		echo "Building $$service Lambda..."; \
		$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/$$service/bootstrap ./cmd/lambda/$$service; \
	done
	@echo "All Lambda functions built successfully!"

## Build only active Lambda functions (auth & user)
build-lambdas-active:
	@echo "Building active Lambda functions (auth & user)..."
	@mkdir -p $(LAMBDA_OUT_DIR)
	@for service in $(LAMBDA_SERVICES_ACTIVE); do \
		echo "Building $$service Lambda..."; \
		$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/$$service/bootstrap ./cmd/lambda/$$service; \
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

build-lambda-artisan:
	@echo "Building artisan Lambda..."
	@mkdir -p $(LAMBDA_OUT_DIR)/artisan
	$(LAMBDA_BUILD_FLAGS) $(GOBUILD) -ldflags="-s -w" -o $(LAMBDA_OUT_DIR)/artisan/bootstrap ./cmd/lambda/artisan

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

## CDK commands
cdk-synth:
	@echo "Synthesizing CDK stacks..."
	cd infra && cdk synth

cdk-synth-dev:
	@echo "Synthesizing CDK stacks for dev..."
	cd infra && cdk synth -c environment=dev -c certArn=$(CERT_ARN)

cdk-synth-prod:
	@echo "Synthesizing CDK stacks for prod..."
	cd infra && cdk synth -c environment=prod -c certArn=$(CERT_ARN)

cdk-synth-all:
	@echo "Synthesizing CDK stacks for all environments..."
	cd infra && cdk synth -c deployMode=all -c certArn=$(CERT_ARN)

cdk-deploy-dev: build-lambdas-active
	@echo "Deploying to development environment..."
	cd infra && cdk deploy --all --require-approval never -c environment=dev -c certArn=$(CERT_ARN)

cdk-deploy-prod: build-lambdas-active
	@echo "Deploying to production environment..."
	cd infra && cdk deploy --all --require-approval broadening -c environment=prod -c certArn=$(CERT_ARN)

cdk-deploy-all: build-lambdas-active
	@echo "Deploying to all environments..."
	cd infra && cdk deploy --all --require-approval broadening -c deployMode=all -c certArn=$(CERT_ARN)

cdk-destroy-dev:
	@echo "Destroying dev CDK stacks..."
	cd infra && cdk destroy --all -c environment=dev

cdk-destroy-prod:
	@echo "Destroying prod CDK stacks..."
	cd infra && cdk destroy --all -c environment=prod

cdk-destroy:
	@echo "Destroying CDK stacks..."
	cd infra && cdk destroy --all

cdk-diff:
	@echo "Showing CDK diff..."
	cd infra && cdk diff

cdk-diff-dev:
	@echo "Showing CDK diff for dev..."
	cd infra && cdk diff -c environment=dev -c certArn=$(CERT_ARN)

cdk-diff-prod:
	@echo "Showing CDK diff for prod..."
	cd infra && cdk diff -c environment=prod -c certArn=$(CERT_ARN)

cdk-bootstrap:
	@echo "Bootstrapping CDK (required once per account/region)..."
	cd infra && cdk bootstrap

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
	@echo "  make seed-data          - Seed test data"
	@echo "  make reset-db           - Drop and recreate all tables"
	@echo "  make list-tables        - List DynamoDB tables"
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
	@echo "  make build-lambdas-active - Build active Lambdas only (auth, user)"
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
