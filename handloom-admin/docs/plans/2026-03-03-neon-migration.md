# Neon Postgres Migration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace AWS RDS PostgreSQL with Neon Postgres to eliminate ~$21/month infrastructure costs.

**Architecture:** Remove VPC, RDS instance, and Secrets Manager from CDK stacks. Replace with a single `POSTGRES_DSN` env var pointing to Neon's pooled endpoint. No changes to Go application code except removing the Secrets Manager credential resolution in the Postgres client.

**Tech Stack:** Go CDK (aws-cdk-go), pgx/v5, Neon Postgres (ap-southeast-1)

---

### Task 1: Simplify Postgres client — remove Secrets Manager resolution

**Files:**
- Modify: `internal/repository/postgres/client.go`

**Step 1: Replace client.go with simplified version**

Remove the `resolveDSNFromSecret` function and all Secrets Manager imports. `NewPool` should only use `pgCfg.DSN`.

```go
// Package postgres provides PostgreSQL-backed repository implementations
// for catalog data (categories, products, inventory).
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	appconfig "github.com/handloom/admin/internal/config"
)

// NewPool creates a PostgreSQL connection pool.
// Uses POSTGRES_DSN environment variable in all environments.
func NewPool(ctx context.Context, pgCfg *appconfig.PostgresConfig) (*pgxpool.Pool, error) {
	if pgCfg.DSN == "" {
		return nil, fmt.Errorf("no postgres DSN configured (set POSTGRES_DSN)")
	}

	cfg, err := pgxpool.ParseConfig(pgCfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse postgres DSN: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && go build ./internal/repository/postgres/...`
Expected: SUCCESS (no errors)

**Step 3: Commit**

```bash
git add internal/repository/postgres/client.go
git commit -m "refactor(postgres): remove Secrets Manager resolution, use DSN directly"
```

---

### Task 2: Clean up PostgresConfig in config.go

**Files:**
- Modify: `internal/config/config.go`

**Step 1: Remove unused PostgresConfig fields**

Change the `PostgresConfig` struct and its loading in `Load()`. Remove `SecretARN`, `Endpoint`, `Port`, `DatabaseName` since all environments now use `POSTGRES_DSN`.

In the struct (lines 27-33), replace with:

```go
// PostgresConfig holds PostgreSQL connection configuration
type PostgresConfig struct {
	DSN string // Connection string: env POSTGRES_DSN
}
```

In `Load()` (lines 149-155), replace the `Postgres:` block with:

```go
		Postgres: PostgresConfig{
			DSN: getEnv("POSTGRES_DSN", ""),
		},
```

**Step 2: Verify it compiles**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && go build ./...`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "refactor(config): simplify PostgresConfig to DSN-only"
```

---

### Task 3: Update CDK database stack — remove RDS, VPC, Secrets Manager

**Files:**
- Modify: `infra/stacks/database.go`

**Step 1: Remove RDS/VPC/Secrets Manager resources and add DSN parameter**

Remove these from the `DatabaseStack` struct:
- `CatalogVPC`, `CatalogDB`, `CatalogDBSecret` fields

Remove from `NewDatabaseStack`:
- VPC creation (lines 303-315)
- Security group (lines 317-329)
- Secrets Manager secret (lines 331-335)
- RDS instance (lines 337-359)
- `catalogDBSecret.GrantRead(migratorFn, nil)` (line 380)
- `ExecuteAfter: &[]constructs.Construct{catalogDB}` in trigger (line 384)
- CfnOutputs for `CatalogDBEndpoint` and `CatalogDBSecretARN`
- References to `catalogVPC`, `catalogDB`, `catalogDBSecret` in the return statement

Remove unused imports: `awsec2`, `awsrds`

Add a `PostgresDSN` field to `DatabaseStack` that reads from CDK context.

Updated migrator env vars — replace RDS vars with `POSTGRES_DSN`.

The full updated file:

```go
package stacks

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/triggers"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// DatabaseStackProps holds properties for the database stack
type DatabaseStackProps struct {
	awscdk.StackProps
	Environment string
	PostgresDSN string // Neon Postgres pooled connection string
}

// DatabaseStack contains the DynamoDB tables and Neon Postgres DSN
type DatabaseStack struct {
	awscdk.Stack
	CoreTable          awsdynamodb.Table
	OrdersTable        awsdynamodb.Table
	SessionsTable      awsdynamodb.Table
	AuditTable         awsdynamodb.Table
	AnalyticsTable     awsdynamodb.Table
	NotificationsTable awsdynamodb.Table
	EventsTable        awsdynamodb.Table
	// Neon Postgres DSN (passed to API/Event stacks for Lambda env vars)
	PostgresDSN *string
}

// NewDatabaseStack creates a new database stack
// AWS DynamoDB Free Tier (Always Free):
// - 25 GB of storage
// - 25 provisioned Write Capacity Units (WCU)
// - 25 provisioned Read Capacity Units (RCU)
// OR with on-demand: 25 WRUs and 25 RRUs per month
func NewDatabaseStack(scope constructs.Construct, id string, props *DatabaseStackProps) *DatabaseStack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}

	stack := awscdk.NewStack(scope, &id, &sprops)
	isProd := props.Environment == "prod"

	// Billing mode - PAY_PER_REQUEST (on-demand) is actually better for free tier
	// when traffic is unpredictable and low. For dev, this stays within free tier.
	billingMode := awsdynamodb.BillingMode_PAY_PER_REQUEST
	removalPolicy := awscdk.RemovalPolicy_DESTROY
	if isProd {
		removalPolicy = awscdk.RemovalPolicy_RETAIN
	}

	// Core Table - categories, designs, products, pricing rules, users, etc.
	// Free tier: 25GB storage across all tables
	coreTable := awsdynamodb.NewTable(stack, jsii.String("CoreTable"), &awsdynamodb.TableProps{
		TableName:     jsii.String("handloom-core-" + props.Environment),
		BillingMode:   billingMode,
		RemovalPolicy: removalPolicy,
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		PointInTimeRecoverySpecification: &awsdynamodb.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: jsii.Bool(false),
		},
		TimeToLiveAttribute: jsii.String("ttl"), // TTL for pricing/coupon expiry
	})

	// GSI1 - General purpose index
	coreTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
		IndexName: jsii.String("GSI1"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		ProjectionType: awsdynamodb.ProjectionType_ALL,
	})

	// GSI2 - Secondary index
	coreTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
		IndexName: jsii.String("GSI2"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI2PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI2SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		ProjectionType: awsdynamodb.ProjectionType_ALL,
	})

	// Notifications Table - notifications with recipient-based GSI
	// Separated for independent scaling during notification spikes
	notificationsTable := awsdynamodb.NewTable(stack, jsii.String("NotificationsTable"), &awsdynamodb.TableProps{
		TableName:     jsii.String("handloom-notifications-" + props.Environment),
		BillingMode:   billingMode,
		RemovalPolicy: removalPolicy,
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		PointInTimeRecoverySpecification: &awsdynamodb.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: jsii.Bool(false),
		},
		TimeToLiveAttribute: jsii.String("ttl"),
	})

	notificationsTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
		IndexName: jsii.String("GSI1"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		ProjectionType: awsdynamodb.ProjectionType_ALL,
	})

	// Sessions Table - OTP, admin refresh tokens, customer refresh tokens
	// Isolated from core table to separate TTL churn from catalog data
	sessionsTable := awsdynamodb.NewTable(stack, jsii.String("SessionsTable"), &awsdynamodb.TableProps{
		TableName:     jsii.String("handloom-sessions-" + props.Environment),
		BillingMode:   billingMode,
		RemovalPolicy: removalPolicy,
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		PointInTimeRecoverySpecification: &awsdynamodb.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: jsii.Bool(false),
		},
		TimeToLiveAttribute: jsii.String("ttl"),
	})

	// Orders Table - orders and customers
	ordersTable := awsdynamodb.NewTable(stack, jsii.String("OrdersTable"), &awsdynamodb.TableProps{
		TableName:     jsii.String("handloom-orders-" + props.Environment),
		BillingMode:   billingMode,
		RemovalPolicy: removalPolicy,
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		PointInTimeRecoverySpecification: &awsdynamodb.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: jsii.Bool(false),
		},
		TimeToLiveAttribute: jsii.String("ttl"), // TTL for PriceQuote and Cart expiry
	})

	ordersTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
		IndexName: jsii.String("GSI1"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		ProjectionType: awsdynamodb.ProjectionType_ALL,
	})

	// GSI2 - Secondary index for orders (e.g., customer lookups, price quote expiry)
	ordersTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
		IndexName: jsii.String("GSI2"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI2PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI2SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		ProjectionType: awsdynamodb.ProjectionType_ALL,
	})

	// Audit Table - audit logs with 30-day TTL to limit storage
	auditTable := awsdynamodb.NewTable(stack, jsii.String("AuditTable"), &awsdynamodb.TableProps{
		TableName:     jsii.String("handloom-audit-" + props.Environment),
		BillingMode:   billingMode,
		RemovalPolicy: removalPolicy,
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		PointInTimeRecoverySpecification: &awsdynamodb.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: jsii.Bool(false),
		},
		TimeToLiveAttribute: jsii.String("ttl"), // Use TTL to auto-delete old records and save storage
	})

	auditTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
		IndexName: jsii.String("GSI1"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		ProjectionType: awsdynamodb.ProjectionType_ALL,
	})

	// GSI2 - Secondary index for audit
	auditTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
		IndexName: jsii.String("GSI2"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI2PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI2SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		ProjectionType: awsdynamodb.ProjectionType_ALL,
	})

	// Analytics Table - analytics data
	analyticsTable := awsdynamodb.NewTable(stack, jsii.String("AnalyticsTable"), &awsdynamodb.TableProps{
		TableName:     jsii.String("handloom-analytics-" + props.Environment),
		BillingMode:   billingMode,
		RemovalPolicy: removalPolicy,
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		TimeToLiveAttribute: jsii.String("ttl"),
	})

	// GSI1 - General purpose index for analytics
	analyticsTable.AddGlobalSecondaryIndex(&awsdynamodb.GlobalSecondaryIndexProps{
		IndexName: jsii.String("GSI1"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("GSI1SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		ProjectionType: awsdynamodb.ProjectionType_ALL,
	})

	// Events Table - raw frontend tracking events with 30-day TTL
	// Simple PK/SK table, no GSIs needed
	eventsTable := awsdynamodb.NewTable(stack, jsii.String("EventsTable"), &awsdynamodb.TableProps{
		TableName:     jsii.String("handloom-events-" + props.Environment),
		BillingMode:   billingMode,
		RemovalPolicy: removalPolicy,
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		TimeToLiveAttribute: jsii.String("ttl"),
	})

	// --- Schema Migrator Lambda ---
	// Runs embedded SQL migrations against Neon Postgres during CDK deploy.
	// CDK Trigger invokes this on handler change, before API Lambdas need the schema.
	postgresDSN := jsii.String(props.PostgresDSN)

	migratorFn := awslambda.NewFunction(stack, jsii.String("MigratorFunction"), &awslambda.FunctionProps{
		FunctionName: jsii.String(fmt.Sprintf("handloom-migrator-%s", props.Environment)),
		Runtime:      awslambda.Runtime_PROVIDED_AL2023(),
		Handler:      jsii.String("bootstrap"),
		Code:         awslambda.Code_FromAsset(jsii.String("../bin/lambda/migrator"), nil),
		Architecture: awslambda.Architecture_ARM_64(),
		MemorySize:   jsii.Number(128),
		Timeout:      awscdk.Duration_Seconds(jsii.Number(60)),
		Environment: &map[string]*string{
			"POSTGRES_DSN": postgresDSN,
			"APP_ENV":      jsii.String(props.Environment),
		},
	})

	triggers.NewTrigger(stack, jsii.String("MigratorTrigger"), &triggers.TriggerProps{
		Handler:                migratorFn,
		ExecuteOnHandlerChange: jsii.Bool(true),
	})

	// Outputs
	awscdk.NewCfnOutput(stack, jsii.String("CoreTableName"), &awscdk.CfnOutputProps{
		Value:       coreTable.TableName(),
		Description: jsii.String("Core DynamoDB table name"),
		ExportName:  jsii.String("handloom-core-table-" + props.Environment),
	})

	awscdk.NewCfnOutput(stack, jsii.String("OrdersTableName"), &awscdk.CfnOutputProps{
		Value:       ordersTable.TableName(),
		Description: jsii.String("Orders DynamoDB table name"),
		ExportName:  jsii.String("handloom-orders-table-" + props.Environment),
	})

	awscdk.NewCfnOutput(stack, jsii.String("SessionsTableName"), &awscdk.CfnOutputProps{
		Value:       sessionsTable.TableName(),
		Description: jsii.String("Sessions DynamoDB table name"),
		ExportName:  jsii.String("handloom-sessions-table-" + props.Environment),
	})

	awscdk.NewCfnOutput(stack, jsii.String("NotificationsTableName"), &awscdk.CfnOutputProps{
		Value:       notificationsTable.TableName(),
		Description: jsii.String("Notifications DynamoDB table name"),
		ExportName:  jsii.String("handloom-notifications-table-" + props.Environment),
	})

	awscdk.NewCfnOutput(stack, jsii.String("EventsTableName"), &awscdk.CfnOutputProps{
		Value:       eventsTable.TableName(),
		Description: jsii.String("Events DynamoDB table name"),
		ExportName:  jsii.String("handloom-events-table-" + props.Environment),
	})

	return &DatabaseStack{
		Stack:              stack,
		CoreTable:          coreTable,
		OrdersTable:        ordersTable,
		SessionsTable:      sessionsTable,
		AuditTable:         auditTable,
		AnalyticsTable:     analyticsTable,
		NotificationsTable: notificationsTable,
		EventsTable:        eventsTable,
		PostgresDSN:        postgresDSN,
	}
}
```

**Step 2: Verify CDK compiles**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && go build ./infra/...`
Expected: FAIL — `api.go` and `events.go` still reference removed fields. That's expected, we fix those next.

**Step 3: Commit (partial — will fix compilation in subsequent tasks)**

```bash
git add infra/stacks/database.go
git commit -m "infra(database): remove RDS, VPC, Secrets Manager; use Neon Postgres DSN"
```

---

### Task 4: Update CDK API stack — replace RDS env vars with POSTGRES_DSN

**Files:**
- Modify: `infra/stacks/api.go`

**Step 1: Replace RDS env vars in commonEnv**

In `commonEnv` (lines 74-95), remove these 4 lines:
```go
"RDS_SECRET_ARN":               props.DatabaseStack.CatalogDBSecret.SecretArn(),
"RDS_ENDPOINT":                 props.DatabaseStack.CatalogDB.DbInstanceEndpointAddress(),
"RDS_PORT":                     jsii.String("5432"),
"RDS_DATABASE":                 jsii.String("handloom"),
```

Replace with:
```go
"POSTGRES_DSN": props.DatabaseStack.PostgresDSN,
```

**Step 2: Remove CatalogDBSecret.GrantRead call**

Remove line 171:
```go
props.DatabaseStack.CatalogDBSecret.GrantRead(lambdaFn, nil)
```

**Step 3: Verify CDK compiles**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && go build ./infra/...`
Expected: FAIL — `events.go` still references removed fields. Fix in next task.

**Step 4: Commit**

```bash
git add infra/stacks/api.go
git commit -m "infra(api): replace RDS env vars with POSTGRES_DSN for all Lambdas"
```

---

### Task 5: Update CDK event stack — replace RDS env vars with POSTGRES_DSN

**Files:**
- Modify: `infra/stacks/events.go`

**Step 1: Replace RDS env vars in worker env**

In the worker loop env map (lines 166-180), remove:
```go
"RDS_SECRET_ARN":               props.DatabaseStack.CatalogDBSecret.SecretArn(),
"RDS_ENDPOINT":                 props.DatabaseStack.CatalogDB.DbInstanceEndpointAddress(),
"RDS_PORT":                     jsii.String("5432"),
"RDS_DATABASE":                 jsii.String("handloom"),
```

Replace with:
```go
"POSTGRES_DSN": props.DatabaseStack.PostgresDSN,
```

**Step 2: Remove CatalogDBSecret.GrantRead call**

Remove line 216:
```go
props.DatabaseStack.CatalogDBSecret.GrantRead(lambdaFn, nil)
```

**Step 3: Verify CDK compiles**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && go build ./infra/...`
Expected: FAIL — `main.go` still needs update for `PostgresDSN` prop.

**Step 4: Commit**

```bash
git add infra/stacks/events.go
git commit -m "infra(events): replace RDS env vars with POSTGRES_DSN for workers"
```

---

### Task 6: Update CDK entry point — pass PostgresDSN from context

**Files:**
- Modify: `infra/cmd/main.go`

**Step 1: Add PostgresDSN resolution and pass to DatabaseStack**

Add a `getPostgresDSN` helper that reads from CDK context (`-c postgresDsn=...`) or `POSTGRES_DSN` env var.

Add to `createEnvironmentStacks`, passing to `DatabaseStackProps`:

```go
func getPostgresDSN(app constructs.Construct) string {
	if dsn := app.Node().TryGetContext(jsii.String("postgresDsn")); dsn != nil {
		return dsn.(string)
	}
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		return dsn
	}
	return ""
}
```

In `createEnvironmentStacks`, add `PostgresDSN` to `DatabaseStackProps`:

```go
postgresDSN := getPostgresDSN(app)

databaseStack := stacks.NewDatabaseStack(app, "HandloomDatabaseStack-"+environment, &stacks.DatabaseStackProps{
	StackProps: awscdk.StackProps{
		Env:         env,
		Description: jsii.String("Handloom Admin - Database resources (" + environment + ")"),
		Tags: &map[string]*string{
			"Environment": jsii.String(environment),
			"Project":     jsii.String("handloom-admin"),
			"ManagedBy":   jsii.String("cdk"),
		},
	},
	Environment: environment,
	PostgresDSN: postgresDSN,
})
```

**Step 2: Verify full CDK compiles**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && go build ./infra/...`
Expected: SUCCESS

**Step 3: Verify Go application compiles**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && go build ./...`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add infra/cmd/main.go
git commit -m "infra(main): pass Neon Postgres DSN from CDK context to stacks"
```

---

### Task 7: Run tests to verify no regressions

**Step 1: Run unit tests**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && make test-unit`
Expected: All tests pass. No test should reference `SecretARN`, `Endpoint`, `Port`, or `DatabaseName` fields on `PostgresConfig` since tests use `POSTGRES_DSN` (or mock the pool directly).

**Step 2: Run lint**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && golangci-lint run`
Expected: No new lint issues (removed unused imports should clear any warnings)

**Step 3: Commit any fixes if needed**

---

### Task 8: Update local dev docker-compose (if needed)

**Files:**
- Check: `docker-compose.yml` or `docker-compose.yaml` for local PostgreSQL container

**Step 1: Verify local dev still works**

The local dev setup uses `POSTGRES_DSN` env var already (per `config.go` line 150). Check that `make run` or docker-compose sets `POSTGRES_DSN` correctly for the local PostgreSQL container.

Run: `grep -r "POSTGRES_DSN" /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin/` to find all references.

**Step 2: Update any .env files if they reference RDS_* vars**

If `.env.local` or similar files reference `RDS_SECRET_ARN`, `RDS_ENDPOINT`, etc., replace with `POSTGRES_DSN`.

**Step 3: Commit if changes were needed**

---

### Task 9: Verify deployment readiness

**Step 1: Build all Lambda binaries**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && make build-lambdas-active`
Expected: SUCCESS — all active Lambda binaries build

**Step 2: Build migrator**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin && GOOS=linux GOARCH=arm64 go build -o bin/lambda/migrator/bootstrap ./cmd/lambda/migrator`
Expected: SUCCESS

**Step 3: Dry-run CDK diff (optional, requires AWS credentials)**

Run: `cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/handloom-admin/infra && POSTGRES_DSN="placeholder" cdk diff -c environment=dev -c postgresDsn="placeholder"`
Expected: Shows removal of RDS, VPC, Secrets Manager resources and addition of POSTGRES_DSN env var

---

## Deployment Checklist (Manual Steps After Code Changes)

1. Create Neon project at console.neon.tech (Singapore, Postgres 16)
2. Copy pooled connection string
3. Run migrations: `psql "<neon-dsn>" -f migrations/001_catalog_schema.sql` (repeat for each migration file)
4. Deploy: `POSTGRES_DSN="<neon-dsn>" make cdk-deploy-dev` (or pass via `-c postgresDsn=...`)
5. Verify: hit `https://dev-store.homechrome.in` — categories and products load
6. After 48h stable: delete RDS instance manually via AWS console
