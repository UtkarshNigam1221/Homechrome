// Package stacks contains the CDK stack definitions.
package stacks

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudwatch"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsevents"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseventstargets"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssns"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// CronStackProps holds dependencies for the cron stack.
type CronStackProps struct {
	awscdk.StackProps
	Environment   string
	OrdersTable   awsdynamodb.Table
	ShippingTable awsdynamodb.Table
	CoreTable     awsdynamodb.Table
	EventTopic    awssns.Topic // optional, nil when event stack disabled
	// EnvVars is a slim environment variable set built by the caller (see
	// main.go). Cron Lambdas do not need most of APIStack's commonEnv (no JWT
	// secrets, no MSG91, no PhonePe, no ALLOWED_ORIGINS, no COOKIE_DOMAIN, no
	// irrelevant DynamoDB table names), so this map is constructed
	// independently to break the circular dependency between APIStack and
	// CronStack.
	EnvVars *map[string]*string
}

// CronStack exposes the cron Lambdas.
type CronStack struct {
	awscdk.Stack
	PickupBatchFn   awslambda.Function
	CODRemittanceFn awslambda.Function
	RateRefreshFn   awslambda.Function
}

// NewCronStack provisions cron Lambdas + EventBridge schedules + CloudWatch alarms.
//
// Schedules (UTC, IST offset +5:30):
//   - pickup-batch:    11:30 UTC Mon-Fri        = 17:00 IST Mon-Fri (avoids scheduling Sunday pickups Delhivery may reject)
//   - cod-remittance:  02:30 UTC daily          = 08:00 IST daily
//   - rate-refresh:    21:30 UTC every Saturday = 03:00 IST every Sunday
func NewCronStack(scope constructs.Construct, id string, props *CronStackProps) *CronStack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)
	env := props.Environment

	memorySize := jsii.Number(256)
	defaultTimeout := awscdk.Duration_Minutes(jsii.Number(5))

	makeFn := func(fnName, codePath string, timeout awscdk.Duration) awslambda.Function {
		t := defaultTimeout
		if timeout != nil {
			t = timeout
		}
		fn := awslambda.NewFunction(stack, jsii.String(capitalize(fnName)+"Function"), &awslambda.FunctionProps{
			FunctionName: jsii.String("handloom-" + fnName + "-" + env),
			Runtime:      awslambda.Runtime_PROVIDED_AL2023(),
			Architecture: awslambda.Architecture_ARM_64(),
			Handler:      jsii.String("bootstrap"),
			Code:         awslambda.Code_FromAsset(jsii.String(codePath), nil),
			MemorySize:   memorySize,
			Timeout:      t,
			Environment:  props.EnvVars,
			// Reserved concurrency = 1 prevents at-least-once EventBridge delivery
			// races (a missed run replayed concurrently with the next scheduled run).
			// Cron handlers are non-idempotent: CreateManifest creates carrier-side
			// resources; COD pull may produce duplicate event publishes.
			ReservedConcurrentExecutions: jsii.Number(1),
		})
		props.OrdersTable.GrantReadWriteData(fn)
		props.ShippingTable.GrantReadWriteData(fn)
		props.CoreTable.GrantReadWriteData(fn)
		if props.EventTopic != nil {
			props.EventTopic.GrantPublish(fn)
		}
		return fn
	}

	pickupBatchFn := makeFn("cron-pickup-batch", "../bin/lambda/cron-pickup-batch", awscdk.Duration_Minutes(jsii.Number(10)))
	codRemittanceFn := makeFn("cron-cod-remittance", "../bin/lambda/cron-cod-remittance", nil)
	rateRefreshFn := makeFn("cron-rate-refresh", "../bin/lambda/cron-rate-refresh", awscdk.Duration_Minutes(jsii.Number(10)))

	awsevents.NewRule(stack, jsii.String("PickupBatchSchedule"), &awsevents.RuleProps{
		RuleName: jsii.String(fmt.Sprintf("handloom-pickup-batch-%s", env)),
		Schedule: awsevents.Schedule_Cron(&awsevents.CronOptions{
			Minute:  jsii.String("30"),
			Hour:    jsii.String("11"),
			WeekDay: jsii.String("MON-FRI"),
		}),
		Targets: &[]awsevents.IRuleTarget{
			awseventstargets.NewLambdaFunction(pickupBatchFn, nil),
		},
	})

	awsevents.NewRule(stack, jsii.String("CODRemittanceSchedule"), &awsevents.RuleProps{
		RuleName: jsii.String(fmt.Sprintf("handloom-cod-remittance-%s", env)),
		Schedule: awsevents.Schedule_Cron(&awsevents.CronOptions{
			Minute: jsii.String("30"),
			Hour:   jsii.String("2"),
		}),
		Targets: &[]awsevents.IRuleTarget{
			awseventstargets.NewLambdaFunction(codRemittanceFn, nil),
		},
	})

	awsevents.NewRule(stack, jsii.String("RateRefreshSchedule"), &awsevents.RuleProps{
		RuleName: jsii.String(fmt.Sprintf("handloom-rate-refresh-%s", env)),
		Schedule: awsevents.Schedule_Cron(&awsevents.CronOptions{
			Minute:  jsii.String("30"),
			Hour:    jsii.String("21"),
			WeekDay: jsii.String("SAT"),
		}),
		Targets: &[]awsevents.IRuleTarget{
			awseventstargets.NewLambdaFunction(rateRefreshFn, nil),
		},
	})

	makeErrorAlarm := func(alarmID string, fn awslambda.Function) {
		awscloudwatch.NewAlarm(stack, jsii.String(alarmID), &awscloudwatch.AlarmProps{
			AlarmName: jsii.String(fmt.Sprintf("%s-errors", *fn.FunctionName())),
			Metric: fn.MetricErrors(&awscloudwatch.MetricOptions{
				Period:    awscdk.Duration_Hours(jsii.Number(1)),
				Statistic: jsii.String("Sum"),
			}),
			Threshold:          jsii.Number(0),
			EvaluationPeriods:  jsii.Number(1),
			ComparisonOperator: awscloudwatch.ComparisonOperator_GREATER_THAN_THRESHOLD,
			TreatMissingData:   awscloudwatch.TreatMissingData_NOT_BREACHING,
		})
	}
	makeErrorAlarm("PickupBatchErrors", pickupBatchFn)
	makeErrorAlarm("CODRemittanceErrors", codRemittanceFn)
	makeErrorAlarm("RateRefreshErrors", rateRefreshFn)

	return &CronStack{
		Stack:           stack,
		PickupBatchFn:   pickupBatchFn,
		CODRemittanceFn: codRemittanceFn,
		RateRefreshFn:   rateRefreshFn,
	}
}
