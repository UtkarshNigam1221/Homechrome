// Package stacks — LogsStack centralizes ownership of all CloudWatch log
// groups consumed by every other stack in this app (DatabaseStack migrator,
// APIStack service + image-resizer Lambdas, EmbedderStack). Owning the groups
// in one place enforces a single retention policy and removes ad-hoc inline
// log group creation that caused physical-name collisions during the
// SharedApiLogGroup → ApiLogGroup refactor.
package stacks

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// LogsStackProps configures the LogsStack.
type LogsStackProps struct {
	awscdk.StackProps
	Environment string
}

// LogsStack publishes the shared log groups. Both fields are exposed as
// awslogs.ILogGroup so consumers don't depend on the concrete LogGroup type
// (forward-compatible with FromLogGroupName if we ever need to import an
// externally-owned group here).
type LogsStack struct {
	awscdk.Stack
	// ApiLogGroup — synchronous API services: all admin + store handler
	// Lambdas, embedder, image-resizer, migrator.
	ApiLogGroup awslogs.ILogGroup
	// WorkerLogGroup — async workers (backfill, metrics consumer). Kept
	// separate so chatty per-message worker logs don't drown out request-trace
	// logs.
	WorkerLogGroup awslogs.ILogGroup
}

// NewLogsStack creates the log groups. 1-day retention everywhere
// (CloudWatch is a short-lived safety net — Grafana via OTel holds the real
// logs); RemovalPolicy DESTROY so `cdk destroy` cleans them up.
func NewLogsStack(scope constructs.Construct, id string, props *LogsStackProps) *LogsStack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	api := awslogs.NewLogGroup(stack, jsii.String("ApiLogGroup"), &awslogs.LogGroupProps{
		LogGroupName:  jsii.String(fmt.Sprintf("/aws/lambda/handloom-api-%s", props.Environment)),
		Retention:     awslogs.RetentionDays_ONE_DAY,
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})
	workers := awslogs.NewLogGroup(stack, jsii.String("WorkerLogGroup"), &awslogs.LogGroupProps{
		LogGroupName:  jsii.String(fmt.Sprintf("/aws/lambda/handloom-workers-%s", props.Environment)),
		Retention:     awslogs.RetentionDays_ONE_DAY,
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})

	return &LogsStack{
		Stack:          stack,
		ApiLogGroup:    api,
		WorkerLogGroup: workers,
	}
}
