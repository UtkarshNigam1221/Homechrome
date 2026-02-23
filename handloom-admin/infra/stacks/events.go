package stacks

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsevents"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseventstargets"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambdaeventsources"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssns"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssnssubscriptions"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3assets"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// EventStackProps holds properties for the event-driven async stack.
type EventStackProps struct {
	awscdk.StackProps
	Environment   string
	DatabaseStack *DatabaseStack
}

// EventStack contains the SNS topic, SQS queues, and worker Lambdas.
type EventStack struct {
	awscdk.Stack
	Topic    awssns.Topic
	TopicARN *string
}

// NewEventStack creates a new event stack with SNS topic, SQS queues, and worker Lambdas.
func NewEventStack(scope constructs.Construct, id string, props *EventStackProps) *EventStack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}

	stack := awscdk.NewStack(scope, &id, &sprops)
	isProd := props.Environment == "prod"

	// Memory sizes — optimized for AWS Free Tier
	memorySize := float64(128)
	if isProd {
		memorySize = 256
	}

	// Log retention — minimize CloudWatch costs
	logRetention := awslogs.RetentionDays_THREE_DAYS
	if isProd {
		logRetention = awslogs.RetentionDays_ONE_WEEK
	}

	// --- SNS Topic ---
	topic := awssns.NewTopic(stack, jsii.String("EventsTopic"), &awssns.TopicProps{
		TopicName:   jsii.String(fmt.Sprintf("handloom-events-%s", props.Environment)),
		DisplayName: jsii.String("Handloom System Events"),
	})

	// --- Worker definitions ---
	type workerDef struct {
		name           string
		batchSize      float64
		visibilityTmot float64
		dlqMaxReceive  float64
		concurrency    float64
		filterPrefixes []string // prefix-based SNS filter on event_type attribute
		filterExact    []string // exact-match SNS filter on event_type attribute
		filterAll      bool     // true = no filter (receives all events)
	}

	workers := []workerDef{
		{
			name:           "notification",
			batchSize:      10,
			visibilityTmot: 60,
			dlqMaxReceive:  3,
			concurrency:    5,
			filterPrefixes: []string{"order.", "payment.", "shipment."},
			filterExact:    []string{"customer.registered"},
		},
		{
			name:           "report",
			batchSize:      1,
			visibilityTmot: 120,
			dlqMaxReceive:  3,
			concurrency:    2,
			filterPrefixes: []string{"order.", "payment."},
		},
		{
			name:           "analytics",
			batchSize:      10,
			visibilityTmot: 60,
			dlqMaxReceive:  3,
			concurrency:    5,
			filterPrefixes: []string{"order.", "payment.", "product.", "inventory.", "customer."},
		},
		{
			name:           "audit",
			batchSize:      10,
			visibilityTmot: 60,
			dlqMaxReceive:  5,
			concurrency:    10,
			filterAll:      true,
		},
	}

	// Track analytics Lambda for EventBridge schedule
	var analyticsLambda awslambda.Function

	for _, w := range workers {
		// DLQ
		dlq := awssqs.NewQueue(stack, jsii.String(fmt.Sprintf("%sDLQ", capitalize(w.name))), &awssqs.QueueProps{
			QueueName:         jsii.String(fmt.Sprintf("handloom-%s-dlq-%s", w.name, props.Environment)),
			RetentionPeriod:   awscdk.Duration_Days(jsii.Number(14)),
			VisibilityTimeout: awscdk.Duration_Seconds(jsii.Number(w.visibilityTmot)),
		})

		// Main queue
		queue := awssqs.NewQueue(stack, jsii.String(fmt.Sprintf("%sQueue", capitalize(w.name))), &awssqs.QueueProps{
			QueueName:         jsii.String(fmt.Sprintf("handloom-%s-%s", w.name, props.Environment)),
			VisibilityTimeout: awscdk.Duration_Seconds(jsii.Number(w.visibilityTmot)),
			DeadLetterQueue: &awssqs.DeadLetterQueue{
				MaxReceiveCount: jsii.Number(w.dlqMaxReceive),
				Queue:           dlq,
			},
		})

		// SNS subscription with filter policy on message attributes
		if w.filterAll {
			// No filter — receives all events
			topic.AddSubscription(awssnssubscriptions.NewSqsSubscription(queue, &awssnssubscriptions.SqsSubscriptionProps{
				RawMessageDelivery: jsii.Bool(true),
			}))
		} else {
			// Build a single StringConditions combining prefixes and exact matches.
			// SNS evaluates conditions within a single attribute filter with OR logic.
			stringConditions := &awssns.StringConditions{}

			if len(w.filterPrefixes) > 0 {
				prefixes := make([]*string, len(w.filterPrefixes))
				for i, p := range w.filterPrefixes {
					prefixes[i] = jsii.String(p)
				}
				stringConditions.MatchPrefixes = &prefixes
			}

			if len(w.filterExact) > 0 {
				exact := make([]*string, len(w.filterExact))
				for i, e := range w.filterExact {
					exact[i] = jsii.String(e)
				}
				stringConditions.Allowlist = &exact
			}

			filterPolicy := map[string]awssns.SubscriptionFilter{
				"event_type": awssns.SubscriptionFilter_StringFilter(stringConditions),
			}

			topic.AddSubscription(awssnssubscriptions.NewSqsSubscription(queue, &awssnssubscriptions.SqsSubscriptionProps{
				RawMessageDelivery: jsii.Bool(true),
				FilterPolicy:       &filterPolicy,
			}))
		}

		// Worker Lambda
		workerName := fmt.Sprintf("worker-%s", w.name)
		env := map[string]*string{
			"APP_ENV":                  jsii.String(props.Environment),
			"APP_DEBUG":                jsii.String(fmt.Sprintf("%t", !isProd)),
			"DYNAMODB_CORE_TABLE":          props.DatabaseStack.CoreTable.TableName(),
			"DYNAMODB_ORDERS_TABLE":        props.DatabaseStack.OrdersTable.TableName(),
			"DYNAMODB_AUDIT_TABLE":         props.DatabaseStack.AuditTable.TableName(),
			"DYNAMODB_ANALYTICS_TABLE":     props.DatabaseStack.AnalyticsTable.TableName(),
			"DYNAMODB_NOTIFICATIONS_TABLE": props.DatabaseStack.NotificationsTable.TableName(),
			"DYNAMODB_EVENTS_TABLE":        props.DatabaseStack.EventsTable.TableName(),
			"RDS_SECRET_ARN": props.DatabaseStack.CatalogDBSecret.SecretArn(),
			"RDS_ENDPOINT":   props.DatabaseStack.CatalogDB.DbInstanceEndpointAddress(),
			"RDS_PORT":       jsii.String("5432"),
			"RDS_DATABASE":   jsii.String("handloom"),
			"SERVICE_NAME":             jsii.String(workerName),
		}

		// Explicit log group (replaces deprecated LogRetention on Lambda)
		workerLogGroup := awslogs.NewLogGroup(stack, jsii.String(fmt.Sprintf("%sLogGroup", capitalize(workerName))), &awslogs.LogGroupProps{
			LogGroupName:  jsii.String(fmt.Sprintf("/aws/lambda/handloom-%s-%s", workerName, props.Environment)),
			Retention:     logRetention,
			RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
		})

		lambdaFn := awslambda.NewFunction(stack, jsii.String(fmt.Sprintf("%sFunction", capitalize(workerName))), &awslambda.FunctionProps{
			FunctionName:                 jsii.String(fmt.Sprintf("handloom-%s-%s", workerName, props.Environment)),
			Runtime:                      awslambda.Runtime_PROVIDED_AL2023(),
			Handler:                      jsii.String("bootstrap"),
			Code:                         awslambda.Code_FromAsset(jsii.String(fmt.Sprintf("../bin/lambda/%s", workerName)), &awss3assets.AssetOptions{}),
			Architecture:                 awslambda.Architecture_ARM_64(),
			MemorySize:                   jsii.Number(memorySize),
			Timeout:                      awscdk.Duration_Seconds(jsii.Number(60)),
			Environment:                  &env,
			LogGroup:                     workerLogGroup,
			Tracing:                      awslambda.Tracing_DISABLED,
			ReservedConcurrentExecutions: jsii.Number(w.concurrency),
		})

		// SQS event source
		lambdaFn.AddEventSource(awslambdaeventsources.NewSqsEventSource(queue, &awslambdaeventsources.SqsEventSourceProps{
			BatchSize:               jsii.Number(w.batchSize),
			ReportBatchItemFailures: jsii.Bool(true),
			MaxBatchingWindow:       awscdk.Duration_Seconds(jsii.Number(5)),
		}))

		// Grant DynamoDB read/write access to all tables
		props.DatabaseStack.CoreTable.GrantReadWriteData(lambdaFn)
		props.DatabaseStack.OrdersTable.GrantReadWriteData(lambdaFn)
		props.DatabaseStack.AuditTable.GrantReadWriteData(lambdaFn)
		props.DatabaseStack.AnalyticsTable.GrantReadWriteData(lambdaFn)
		props.DatabaseStack.NotificationsTable.GrantReadWriteData(lambdaFn)
		props.DatabaseStack.EventsTable.GrantReadWriteData(lambdaFn)
		props.DatabaseStack.CatalogDBSecret.GrantRead(lambdaFn, nil)

		// Capture analytics Lambda for EventBridge schedule
		if w.name == "analytics" {
			analyticsLambda = lambdaFn
		}
	}

	// --- EventBridge Rule: Daily Analytics Aggregation ---
	// Triggers the analytics worker Lambda at 00:30 UTC daily to aggregate
	// the previous day's raw events into daily metrics.
	if analyticsLambda != nil {
		awsevents.NewRule(stack, jsii.String("DailyAggregationRule"), &awsevents.RuleProps{
			RuleName:    jsii.String(fmt.Sprintf("handloom-daily-aggregation-%s", props.Environment)),
			Description: jsii.String("Trigger daily analytics aggregation at 00:30 UTC"),
			Schedule:    awsevents.Schedule_Cron(&awsevents.CronOptions{
				Minute: jsii.String("30"),
				Hour:   jsii.String("0"),
			}),
			Targets: &[]awsevents.IRuleTarget{
				awseventstargets.NewLambdaFunction(analyticsLambda, &awseventstargets.LambdaFunctionProps{
					RetryAttempts: jsii.Number(2),
				}),
			},
		})
	}

	// Outputs
	awscdk.NewCfnOutput(stack, jsii.String("EventsTopicARN"), &awscdk.CfnOutputProps{
		Value:       topic.TopicArn(),
		Description: jsii.String("SNS Events Topic ARN"),
		ExportName:  jsii.String(fmt.Sprintf("handloom-events-topic-%s", props.Environment)),
	})

	return &EventStack{
		Stack:    stack,
		Topic:    topic,
		TopicARN: topic.TopicArn(),
	}
}
