// Package stacks — LogsStack centralizes ownership of the storefront's
// CloudWatch log groups. Mirrors the LogsStack pattern in handloom-admin
// (single source of truth for retention + lifecycle, no inline log group
// creation in the resource stacks).
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

// LogsStack publishes the shared log groups. Exposed as awslogs.ILogGroup
// so consumers depend on the interface, not the concrete LogGroup type.
type LogsStack struct {
	awscdk.Stack
	// ServerLogGroup — the Next.js SSR server Lambda's log group.
	ServerLogGroup awslogs.ILogGroup
}

// NewLogsStack creates the log groups. 3-day retention; RemovalPolicy
// DESTROY so `cdk destroy` cleans them up.
func NewLogsStack(scope constructs.Construct, id string, props *LogsStackProps) *LogsStack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	server := awslogs.NewLogGroup(stack, jsii.String("ServerLogGroup"), &awslogs.LogGroupProps{
		LogGroupName:  jsii.String(fmt.Sprintf("/aws/lambda/homechrome-store-server-%s", props.Environment)),
		Retention:     awslogs.RetentionDays_THREE_DAYS,
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})

	return &LogsStack{
		Stack:          stack,
		ServerLogGroup: server,
	}
}
