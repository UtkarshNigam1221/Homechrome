// Package stacks contains all AWS CDK stack constructors for the Handloom
// backend. EmbedderStack provisions the embedder container Lambda + Function URL.
//
// The Docker image is built inline by CDK during `cdk synth` (via
// Code_FromAssetImage) and pushed to the CDK bootstrap ECR repo. No
// custom-managed ECR repo or external `docker push` step needed.
//
// Build context: the entire handloom-admin/ directory. Dockerfile:
// cmd/embedder/Dockerfile. Model + tokenizer + libonnxruntime.so must be
// present at cmd/embedder/assets/ BEFORE running cdk deploy (CDK runs
// docker build from the working directory, not from S3). Use
// `make download-embedder-assets` to fetch them from the build-assets S3
// bucket which is populated by scripts/bootstrap-embedder-assets.sh.
package stacks

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecrassets"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// EmbedderStackProps configures the embedder Lambda stack.
//
// The embedder is a STANDALONE stack: it owns its own log group and references
// the (backend-owned) metrics queue and Postgres BY NAME/STRING, not by stack
// object. So it carries ZERO CloudFormation cross-stack imports and can be
// synthesized + deployed with no other stack present. See infra/cmd/main.go's
// buildEmbedderApp.
type EmbedderStackProps struct {
	awscdk.StackProps
	Environment    string
	FnName         string // stable function name (cfg.EmbedderFnName); APIStack imports by this exact name
	StoreFrontHost string // e.g. https://store-dev.homechrome.in — single allowed CORS origin
	PostgresDSN    string // plain env var, sourced from .env.{env} / GH secrets at deploy time
	// MetricsQueueName imports the (backend MetricsStack-owned) SQS queue by name
	// for METRICS_QUEUE_URL + SendMessage grant. Imported by ARN → identity-only
	// IAM grant, no foreign QueuePolicy, no cross-stack export. The queue must
	// merely EXIST at runtime; its absence does not block embedder deploy.
	MetricsQueueName string
}

// EmbedderStack publishes:
//   - A container-image Lambda whose image is built from cmd/embedder/Dockerfile
//     at synth time and stored in CDK's bootstrap ECR repo.
//
// The Lambda is mounted onto the existing REST API by APIStack at
// /api/v1/store/catalog/search and /api/v1/store/catalog/embedder-ping —
// no separate Function URL or domain.
type EmbedderStack struct {
	awscdk.Stack
	Function awslambda.Function
}

// NewEmbedderStack builds the EmbedderStack. The Lambda is created with
// memory=1769MB (1 vCPU) ARM64 on-demand. Cold start is ~5-7s; mitigated by
// storefront `/ping` on page mount.
//
// PREREQUISITE: `make download-embedder-assets ENV={env}` must have run so
// cmd/embedder/assets/ contains libonnxruntime.so, model-int8.onnx, and
// tokenizer.json. Without these, `cdk synth` runs `docker build` which
// fails when the Dockerfile's COPY commands reference missing files.
func NewEmbedderStack(scope constructs.Construct, id string, props *EmbedderStackProps) *EmbedderStack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}

	stack := awscdk.NewStack(scope, &id, &sprops)

	// Own log group — keeps the embedder a standalone stack with no dependency
	// on the backend LogsStack. Matches the Lambda's implicit name so AWS does
	// not auto-create a duplicate. 1-day retention + DESTROY mirrors LogsStack.
	logGroup := awslogs.NewLogGroup(stack, jsii.String("EmbedderLogGroup"), &awslogs.LogGroupProps{
		LogGroupName:  jsii.String("/aws/lambda/" + props.FnName),
		Retention:     awslogs.RetentionDays_ONE_DAY,
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})

	// Import the metrics queue by ARN (built from this stack's own account/region
	// tokens + the stable name). fromQueueAttributes sets autoCreatePolicy=false,
	// so GrantSendMessages adds only an identity statement to the embedder role —
	// no foreign QueuePolicy, no Fn::ImportValue.
	var metricsQueue awssqs.IQueue
	if props.MetricsQueueName != "" {
		metricsQueue = awssqs.Queue_FromQueueArn(stack, jsii.String("ImportedMetricsQueue"),
			jsii.String(fmt.Sprintf("arn:aws:sqs:%s:%s:%s",
				*stack.Region(), *stack.Account(), props.MetricsQueueName)))
	}

	fn := awslambda.NewFunction(stack, jsii.String("Embedder"), &awslambda.FunctionProps{
		FunctionName: jsii.String(props.FnName),
		Code: awslambda.Code_FromAssetImage(jsii.String(".."), &awslambda.AssetImageCodeProps{
			File:     jsii.String("cmd/embedder/Dockerfile"),
			Platform: awsecrassets.Platform_LINUX_ARM64(),
		}),
		Handler:      awslambda.Handler_FROM_IMAGE(),
		Runtime:      awslambda.Runtime_FROM_IMAGE(),
		Architecture: awslambda.Architecture_ARM_64(),
		MemorySize:   jsii.Number(1769),
		Timeout:      awscdk.Duration_Seconds(jsii.Number(60)),
		LogGroup:     logGroup,
		// Cap concurrency to prevent burst abuse via the public Function URL.
		// At 1769MB (1 vCPU) each cold-start costs ~6s × 1.769GB-sec; without
		// this cap the account-default ~1000 concurrent containers can be
		// triggered by anyone who discovers the URL. 20 matches expected peak
		// and is tunable via CDK without a code change.
		ReservedConcurrentExecutions: jsii.Number(20),
		Environment: func() *map[string]*string {
			envMap := map[string]*string{
				"EMBEDDER_AUTH_KEY_PARAM":   jsii.String(fmt.Sprintf("/handloom/%s/embedder-auth-key", props.Environment)),
				"POSTGRES_DSN":              jsii.String(props.PostgresDSN),
				"ALLOWED_ORIGIN":            jsii.String(props.StoreFrontHost),
				"SEARCH_WEIGHT_SEMANTIC":    jsii.String("0.60"),
				"SEARCH_WEIGHT_KEYWORD":     jsii.String("0.30"),
				"SEARCH_WEIGHT_TRIGRAM":     jsii.String("0.10"),
				"RATE_LIMIT_PER_IP_PER_MIN": jsii.String("60"),
				"OTEL_SERVICE_NAME":         jsii.String("handloom-embedder"),
			}
			// METRICS_QUEUE_URL triggers SQSPublisher init in main.go; without
			// it search_query metrics fall through to the noop publisher.
			if metricsQueue != nil {
				envMap["METRICS_QUEUE_URL"] = metricsQueue.QueueUrl()
			}
			return &envMap
		}(),
	})

	// SendMessage on the metrics queue so search_query events actually leave
	// this Lambda. Identity-only grant on an imported queue (no foreign policy).
	if metricsQueue != nil {
		metricsQueue.GrantSendMessages(fn)
	}

	// SSM read perm for the embedder-auth-key SecureString only. POSTGRES_DSN
	// is passed as a plain Lambda env var (same pattern as the other
	// Lambdas — sourced from .env.{env} locally or secrets.BACKEND_ENV_{ENV}
	// in GH Actions, exported as POSTGRES_DSN before `cdk deploy`).
	fn.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: jsii.Strings("ssm:GetParameter"),
		Resources: jsii.Strings(
			fmt.Sprintf("arn:aws:ssm:*:*:parameter/handloom/%s/embedder-auth-key", props.Environment),
		),
	}))

	// No Function URL — the embedder is mounted on the existing REST API
	// (APIStack) under /api/v1/store/catalog/search and /catalog/embedder-ping.
	// CORS is handled by chi middleware in cmd/embedder/handler.go.

	return &EmbedderStack{
		Stack:    stack,
		Function: fn,
	}
}
