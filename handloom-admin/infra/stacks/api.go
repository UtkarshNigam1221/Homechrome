package stacks

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigateway"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambdanodejs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3assets"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsssm"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// APIStackProps holds properties for the API stack
type APIStackProps struct {
	awscdk.StackProps
	Environment    string
	DatabaseStack  *DatabaseStack
	StorageStack   *StorageStack
	LogsStack      *LogsStack     // Shared CloudWatch log groups (ApiLogGroup, WorkerLogGroup)
	EmbedderStack  *EmbedderStack // Optional: embedder Lambda for hybrid semantic search
	MetricsQueue   awssqs.Queue   // Optional: PostgreSQL metrics pipeline SQS queue (publishers only)
	BaseDomain     string         // Base domain (e.g. homechrome.in) — used for cookie domain
	DomainName     string         // Optional: custom domain for API Gateway (e.g. dev-api.homechrome.in)
	FrontendOrigin string         // Optional: frontend origin for CORS (e.g. https://dev-admin.homechrome.in)
	CertArn        string         // Optional: ACM certificate ARN (us-east-1) for custom domain

	// Non-secret gateway config, baked per-env in infra/cmd/config.go and
	// injected into Lambda env. Secrets still come from the deploy shell.
	PhonePeBaseURL          string
	PhonePeCallbackURL      string
	PhonePeRedirectURL      string
	PhonePeClientVersion    string
	MSG91BaseURL            string
	MSG91OTPTemplateID      string
	ShiprocketBaseURL       string
	ShiprocketPickupPincode string

	// Telemetry — community-published OTel Collector layer ARN + SSM parameter
	// names for Grafana Cloud credentials. All three must be set together; if
	// CollectorLayerArn is empty the applyTelemetry helper is a no-op.
	// Use stacks.OtelCollectorLayerArn(region, "arm64") to compute the ARN.
	CollectorLayerArn       string // Community OTel Collector layer ARN (account 184161586896)
	GrafanaAuthSSMParam     string // Plain SSM String, e.g. /handloom/dev/grafana-otlp-auth
	GrafanaEndpointSSMParam string // Plain SSM String, e.g. /handloom/dev/grafana-otlp-endpoint
	// Both params are plain `String` type (not SecureString) because CFN
	// forbids {{resolve:ssm-secure:...}} in Lambda env vars.
}

// ServiceLambda represents a Lambda function for a service
type ServiceLambda struct {
	Function awslambda.Function
	Name     string
}

// APIStack contains the API Gateway and Lambda functions
type APIStack struct {
	awscdk.Stack
	API            awsapigateway.RestApi
	Lambdas        map[string]*ServiceLambda
	props          *APIStackProps
	collectorLayer awslambda.ILayerVersion // imported once; nil when CollectorLayerArn is empty
}

// NewAPIStack creates a new API stack
func NewAPIStack(scope constructs.Construct, id string, props *APIStackProps) *APIStack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}

	stack := awscdk.NewStack(scope, &id, &sprops)
	isProd := props.Environment == "prod"

	// Initialise the receiver early so applyTelemetry can be called during
	// construction (before API and Lambdas are fully populated).
	apiStackRef := &APIStack{Stack: stack, props: props}

	// Import the community OTel Collector layer once per stack. The otel.yaml
	// is bundled into each Lambda's zip (see Makefile build-lambdas targets) and
	// is read from /var/task/otel.yaml at runtime.
	if props.CollectorLayerArn != "" {
		apiStackRef.collectorLayer = awslambda.LayerVersion_FromLayerVersionArn(
			stack, jsii.String("OtelCollectorLayer"),
			jsii.String(props.CollectorLayerArn),
		)
	}

	// Shared log groups come from LogsStack — single source of truth for
	// retention + lifecycle.
	apiLogGroup := props.LogsStack.ApiLogGroup
	workerLogGroup := props.LogsStack.WorkerLogGroup

	// JWT secret parameter names. These SSM parameters must be created out-of-band
	// (see scripts/bootstrap-env.sh) before deploying this stack. The stack only
	// CONSUMES the values via dynamic reference — it does not create them. This is
	// the standard pattern for secret material: not in the CFN template, not in
	// git, rotation-safe (rotate via `aws ssm put-parameter` without redeploying CDK).
	jwtSecretParamName := fmt.Sprintf("/handloom/%s/jwt-secret", props.Environment)
	customerJwtSecretParamName := fmt.Sprintf("/handloom/%s/customer-jwt-secret", props.Environment)

	// S3 buckets from StorageStack
	assetsBucket := props.StorageStack.AssetsBucket

	// ImageResizer Lambda — generates responsive variants (320/640/1080/1920 × webp/avif/jpg|png).
	// Co-located with APIStack (asset Lambda is its only consumer) to avoid fragile cross-stack
	// exports. Sharp has Linux ARM64 native binaries, so ForceDockerBundling is required.
	imageResizer := awslambdanodejs.NewNodejsFunction(stack, jsii.String("ImageResizer"), &awslambdanodejs.NodejsFunctionProps{
		Entry:            jsii.String("../lambda/image-resizer/index.mjs"),
		DepsLockFilePath: jsii.String("../lambda/image-resizer/package-lock.json"),
		Runtime:          awslambda.Runtime_NODEJS_20_X(),
		Architecture:     awslambda.Architecture_ARM_64(),
		MemorySize:       jsii.Number(1024),
		Timeout:          awscdk.Duration_Seconds(jsii.Number(90)),
		LogGroup:         apiLogGroup,
		Bundling: &awslambdanodejs.BundlingOptions{
			// Sharp must NOT be esbuild-bundled — it has native binaries.
			// @aws-sdk/client-s3 is provided by the Node 20 Lambda runtime.
			ExternalModules:     jsii.Strings("sharp", "@aws-sdk/client-s3"),
			NodeModules:         jsii.Strings("sharp"),
			ForceDockerBundling: jsii.Bool(true),
		},
		FunctionName: jsii.String(fmt.Sprintf("homechrome-image-resizer-%s", props.Environment)),
	})
	assetsBucket.GrantReadWrite(imageResizer, jsii.String("assets/*"))

	// Common environment variables for all Lambdas
	commonEnv := map[string]*string{
		"IMAGE_RESIZER_FUNCTION_NAME":  imageResizer.FunctionName(),
		"APP_ENV":                      jsii.String(props.Environment),
		"APP_DEBUG":                    jsii.String(fmt.Sprintf("%t", !isProd)),
		"DYNAMODB_CORE_TABLE":          props.DatabaseStack.CoreTable.TableName(),
		"DYNAMODB_ORDERS_TABLE":        props.DatabaseStack.OrdersTable.TableName(),
		"DYNAMODB_SESSIONS_TABLE":      props.DatabaseStack.SessionsTable.TableName(),
		"DYNAMODB_AUDIT_TABLE":         props.DatabaseStack.AuditTable.TableName(),
		"DYNAMODB_NOTIFICATIONS_TABLE": props.DatabaseStack.NotificationsTable.TableName(),
		"POSTGRES_DSN":                 props.DatabaseStack.PostgresDSN,
		"S3_ASSETS_BUCKET":             assetsBucket.BucketName(),
		"CDN_DOMAIN":                   jsii.String(props.StorageStack.CDNDomain),
		// Resolve SSM values at CloudFormation deploy time, inject as plain env vars.
		// Eliminates the ~3s SSM GetParameter cold-start latency observed in Lambda init.
		// Updating the SSM parameter value requires re-deploying the stack to propagate.
		"JWT_SECRET_KEY":             awsssm.StringParameter_ValueForStringParameter(stack, jsii.String(jwtSecretParamName), nil),
		"CUSTOMER_JWT_SECRET":        awsssm.StringParameter_ValueForStringParameter(stack, jsii.String(customerJwtSecretParamName), nil),
		"JWT_ISSUER":                 jsii.String("handloom-admin"),
		"JWT_ACCESS_TOKEN_DURATION":  jsii.String("15m"),
		"JWT_REFRESH_TOKEN_DURATION": jsii.String("168h"),
		"QUOTE_VALIDITY_HRS":         jsii.String("24"),
	}

	// Add custom domain env vars when configured
	if props.FrontendOrigin != "" {
		commonEnv["ALLOWED_ORIGINS"] = jsii.String(props.FrontendOrigin)
	}
	if props.DomainName != "" {
		commonEnv["COOKIE_DOMAIN"] = jsii.String("." + props.BaseDomain)
	}

	// Non-secret gateway config — baked per-env in infra/cmd/config.go, no
	// deploy-time env var needed. Empty values fall through to each gateway's
	// DevClient at runtime.
	gatewayConfig := map[string]string{
		"PHONEPE_BASE_URL":          props.PhonePeBaseURL,
		"PHONEPE_CALLBACK_URL":      props.PhonePeCallbackURL,
		"PHONEPE_REDIRECT_URL":      props.PhonePeRedirectURL,
		"PHONEPE_CLIENT_VERSION":    props.PhonePeClientVersion,
		"MSG91_BASE_URL":            props.MSG91BaseURL,
		"MSG91_OTP_TEMPLATE_ID":     props.MSG91OTPTemplateID,
		"SHIPROCKET_BASE_URL":       props.ShiprocketBaseURL,
		"SHIPROCKET_PICKUP_PINCODE": props.ShiprocketPickupPincode,
	}
	for k, v := range gatewayConfig {
		if v != "" {
			commonEnv[k] = jsii.String(v)
		}
	}

	// Gateway SECRETS — still propagated from the deploy shell (BACKEND_ENV_*
	// secret + the MSG91_AUTH_KEY step secret). Empty → gateway DevClient.
	gatewaySecretKeys := []string{
		"PHONEPE_CLIENT_ID", "PHONEPE_CLIENT_SECRET",
		"PHONEPE_WEBHOOK_USERNAME", "PHONEPE_WEBHOOK_PASSWORD",
		"MSG91_AUTH_KEY",
		"SHIPROCKET_EMAIL", "SHIPROCKET_PASSWORD",
	}
	for _, key := range gatewaySecretKeys {
		if v := os.Getenv(key); v != "" {
			commonEnv[key] = jsii.String(v)
		}
	}

	// Metrics pipeline (PostgreSQL-backed). When the MetricsStack is wired in,
	// every Lambda gets the queue URL as env + IAM SendMessage permission
	// (granted below in the per-service loop). bootstrap.InitLambda flips the
	// default metrics publisher to SQSPublisher when METRICS_QUEUE_URL is set.
	if props.MetricsQueue != nil {
		commonEnv["METRICS_QUEUE_URL"] = props.MetricsQueue.QueueUrl()
	}

	// Memory sizes - optimized for AWS Free Tier
	// Free tier: 1M requests/month, 400,000 GB-seconds compute time
	// 128MB = ~3.2M seconds/month free (plenty for development)
	memorySize := float64(128)
	if isProd {
		memorySize = 256 // Slightly higher for production, still cost-effective
	}

	// apiLogGroup + workerLogGroup are declared at the top of this stack
	// (sourced from LogsStack).

	// Create Lambda functions for each service
	// TODO: Uncomment services as they are implemented
	services := []string{
		"auth",
		"user",
		"catalog",
		"asset",
		"store-auth",
		"store-catalog",
		"store-cart",
		"store-checkout",
		"store-orders",
		"store-tracking",
		"store-profile",
		"store-events",
		"store-webhooks",
		"order",
		// "pricing",
		// "inventory",
		// "analytics",
		// "notification",
		// "coupon",
		// "report",
		// "audit",
	}

	lambdas := make(map[string]*ServiceLambda)
	for _, svc := range services {
		// Asset Lambda keeps a longer timeout as headroom for batched finalize
		// operations (multiple S3 copy/delete RTs per request). ImageResizer is
		// now invoked async, so callers no longer block on the resize itself.
		timeout := float64(15)
		if svc == "asset" {
			timeout = 45
		}
		lambdaFn := createServiceLambda(stack, svc, props.Environment, commonEnv, memorySize, timeout, apiLogGroup)
		lambdas[svc] = &ServiceLambda{
			Function: lambdaFn,
			Name:     svc,
		}

		// Attach OTel Collector layer + telemetry env vars. The service name
		// matches the cmd/lambda/<svc> directory so traces group correctly in
		// Grafana. No-op when CollectorLayer is nil.
		apiStackRef.applyTelemetry(lambdaFn, "handloom-"+svc)

		// Grant permissions
		props.DatabaseStack.CoreTable.GrantReadWriteData(lambdaFn)
		props.DatabaseStack.OrdersTable.GrantReadWriteData(lambdaFn)
		props.DatabaseStack.SessionsTable.GrantReadWriteData(lambdaFn)
		props.DatabaseStack.AuditTable.GrantReadWriteData(lambdaFn)
		props.DatabaseStack.NotificationsTable.GrantReadWriteData(lambdaFn)
		assetsBucket.GrantReadWrite(lambdaFn, nil)
		// Every service Lambda may end up invoking the ImageResizer because
		// AssetService is wired into product/category services (catalog Lambda)
		// as well as the asset Lambda itself. Grant invoke to all.
		imageResizer.GrantInvoke(lambdaFn)
		// SSM JWT secrets resolved at deploy time via env vars above — no runtime read needed.
		// Grant SQS SendMessage to metrics queue when MetricsStack is wired in.
		// Consumer Lambda lives in MetricsStack and owns ConsumeMessages there.
		if props.MetricsQueue != nil {
			props.MetricsQueue.GrantSendMessages(lambdaFn)
		}
	}

	// Embedder integration — grant catalog Lambda invoke + SSM read for embedder auth key.
	// Only wired when EmbedderStack is provided (always the case in practice).
	if props.EmbedderStack != nil {
		catalogFn := lambdas["catalog"].Function

		// Inject embedder config into catalog Lambda
		catalogFn.AddEnvironment(jsii.String("EMBEDDER_FN_NAME"),
			props.EmbedderStack.Function.FunctionName(), nil)
		catalogFn.AddEnvironment(jsii.String("EMBEDDER_AUTH_KEY_PARAM"),
			jsii.String(fmt.Sprintf("/handloom/%s/embedder-auth-key", props.Environment)), nil)
		catalogFn.AddEnvironment(jsii.String("EMBEDDER_TIMEOUT_MS"),
			jsii.String("10000"), nil)
		catalogFn.AddEnvironment(jsii.String("EMBEDDING_MODEL_VERSION"),
			jsii.String("l3cube-indic-sbert-nli-v1"), nil)

		// Allow catalog Lambda to invoke the embedder
		props.EmbedderStack.Function.GrantInvoke(catalogFn)

		// Allow catalog Lambda to read the embedder auth key from SSM
		catalogFn.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
			Actions: jsii.Strings("ssm:GetParameter"),
			Resources: jsii.Strings(fmt.Sprintf(
				"arn:aws:ssm:*:*:parameter/handloom/%s/embedder-auth-key",
				props.Environment,
			)),
		}))

		// Backfill Lambda — one-shot job to embed all existing products.
		// High memory (1769 MB = 1 vCPU) and long timeout (15 min) to handle large catalogs.
		backfillFn := awslambda.NewFunction(stack, jsii.String("EmbeddingBackfill"), &awslambda.FunctionProps{
			FunctionName: jsii.String(fmt.Sprintf("handloom-worker-embedding-backfill-%s", props.Environment)),
			Runtime:      awslambda.Runtime_PROVIDED_AL2023(),
			Architecture: awslambda.Architecture_ARM_64(),
			Handler:      jsii.String("bootstrap"),
			Code:         awslambda.Code_FromAsset(jsii.String("../bin/lambda/worker-embedding-backfill"), &awss3assets.AssetOptions{}),
			MemorySize:   jsii.Number(1769),
			Timeout:      awscdk.Duration_Minutes(jsii.Number(15)),
			LogGroup:     workerLogGroup,
			Environment: &map[string]*string{
				"POSTGRES_DSN":            props.DatabaseStack.PostgresDSN,
				"EMBEDDER_FN_NAME":        props.EmbedderStack.Function.FunctionName(),
				"EMBEDDER_AUTH_KEY_PARAM": jsii.String(fmt.Sprintf("/handloom/%s/embedder-auth-key", props.Environment)),
				"EMBEDDING_MODEL_VERSION": jsii.String("l3cube-indic-sbert-nli-v1"),
			},
		})

		// Allow backfill Lambda to invoke the embedder
		props.EmbedderStack.Function.GrantInvoke(backfillFn)

		// Allow backfill Lambda to read both the embedder auth key and Postgres DSN from SSM
		backfillFn.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
			Actions: jsii.Strings("ssm:GetParameter"),
			Resources: jsii.Strings(
				fmt.Sprintf("arn:aws:ssm:*:*:parameter/handloom/%s/embedder-auth-key", props.Environment),
				fmt.Sprintf("arn:aws:ssm:*:*:parameter/handloom/%s/postgres-dsn", props.Environment),
			),
		}))

		// Attach OTel Collector layer + telemetry env to the backfill worker.
		apiStackRef.applyTelemetry(backfillFn, "handloom-worker-embedding-backfill")

		lambdas["embedding-backfill"] = &ServiceLambda{
			Function: backfillFn,
			Name:     "worker-embedding-backfill",
		}
	}

	// Create API Gateway - optimized for AWS Free Tier
	// Free tier: 1M API calls/month for first 12 months
	api := awsapigateway.NewRestApi(stack, jsii.String("API"), &awsapigateway.RestApiProps{
		RestApiName: jsii.String("handloom-api-" + props.Environment),
		Description: jsii.String("Handloom Admin API - CORS via Lambda"),
		DeployOptions: &awsapigateway.StageOptions{
			StageName:            jsii.String(props.Environment),
			ThrottlingRateLimit:  jsii.Number(500),                     // Accommodates SSR fan-out from storefront Server Lambda
			ThrottlingBurstLimit: jsii.Number(1000),                    // Burst headroom for parallel SSR fetches
			LoggingLevel:         awsapigateway.MethodLoggingLevel_OFF, // Disable logging (CloudWatchRole is false)
			MetricsEnabled:       jsii.Bool(false),                     // Disable detailed metrics to save costs
			TracingEnabled:       jsii.Bool(false),                     // Disable X-Ray tracing (not free)
		},
		// CORS preflight is handled by each Lambda's chi middleware (origin-reflecting),
		// so we do NOT set DefaultCorsPreflightOptions here. API Gateway mock OPTIONS
		// cannot reflect the request Origin, which breaks credentialed (withCredentials)
		// requests from cross-origin frontends.
		CloudWatchRole: jsii.Bool(false), // Disable CloudWatch role to reduce costs
	})

	// Create integrations and routes
	setupAPIRoutes(api, lambdas, props.EmbedderStack)

	// Custom domain for API Gateway (regional — lowest latency for ap-south-1 origin)
	if props.CertArn != "" && props.DomainName != "" {
		cert := awscertificatemanager.Certificate_FromCertificateArn(stack, jsii.String("ApiCertificate"), jsii.String(props.CertArn))

		customDomain := awsapigateway.NewDomainName(stack, jsii.String("ApiCustomDomain"), &awsapigateway.DomainNameProps{
			DomainName:     jsii.String(props.DomainName),
			Certificate:    cert,
			EndpointType:   awsapigateway.EndpointType_REGIONAL,
			SecurityPolicy: awsapigateway.SecurityPolicy_TLS_1_2,
		})

		// Map the custom domain root to the API deployment stage
		awsapigateway.NewBasePathMapping(stack, jsii.String("ApiBasePathMapping"), &awsapigateway.BasePathMappingProps{
			DomainName: customDomain,
			RestApi:    api,
			Stage:      api.DeploymentStage(),
		})

		awscdk.NewCfnOutput(stack, jsii.String("ApiCustomDomainTarget"), &awscdk.CfnOutputProps{
			Value:       customDomain.DomainNameAliasDomainName(),
			Description: jsii.String("CNAME target for API custom domain (add to GoDaddy DNS)"),
		})
	}

	// Outputs
	awscdk.NewCfnOutput(stack, jsii.String("APIEndpoint"), &awscdk.CfnOutputProps{
		Value:       api.Url(),
		Description: jsii.String("API Gateway endpoint URL"),
		ExportName:  jsii.String("handloom-api-url-" + props.Environment),
	})

	awscdk.NewCfnOutput(stack, jsii.String("APIId"), &awscdk.CfnOutputProps{
		Value:       api.RestApiId(),
		Description: jsii.String("API Gateway ID"),
		ExportName:  jsii.String("handloom-api-id-" + props.Environment),
	})

	apiStackRef.API = api
	apiStackRef.Lambdas = lambdas
	return apiStackRef
}

// applyTelemetry attaches the community OTel Collector layer + environment
// variables to a Lambda function. serviceName is used as both the OTel
// service.name attribute and the collector's service identification.
// This is a no-op when CollectorLayerArn is empty so the stack degrades
// gracefully when telemetry has not been configured.
//
// otel.yaml is expected to be bundled into each Lambda zip at /var/task/otel.yaml
// (copied by the Makefile build-lambdas targets from infra/configs/otel-collector.yaml).
func (s *APIStack) applyTelemetry(fn awslambda.Function, serviceName string) {
	if s.props.CollectorLayerArn == "" {
		return
	}
	fn.AddLayers(s.collectorLayer)
	fn.AddEnvironment(jsii.String("OTEL_SERVICE_NAME"), jsii.String(serviceName), nil)
	fn.AddEnvironment(jsii.String("OTEL_RESOURCE_ATTRIBUTES"),
		jsii.String("deployment.environment="+s.props.Environment+",service.namespace=handloom"), nil)
	fn.AddEnvironment(jsii.String("OTEL_EXPORTER_OTLP_PROTOCOL"), jsii.String("grpc"), nil)
	// OTel gRPC exporter wants bare host:port, NOT a URL with scheme.
	// "http://localhost:4317" gets parsed as host="http", port="//localhost:4317:443".
	fn.AddEnvironment(jsii.String("OTEL_EXPORTER_OTLP_ENDPOINT"), jsii.String("localhost:4317"), nil)
	fn.AddEnvironment(jsii.String("AWS_LAMBDA_EXEC_WRAPPER"), jsii.String("/opt/otel-handler"), nil)
	fn.AddEnvironment(jsii.String("OPENTELEMETRY_COLLECTOR_CONFIG_URI"),
		jsii.String("/var/task/otel.yaml"), nil) // yaml bundled into Lambda zip, not baked into layer
	fn.AddEnvironment(jsii.String("GRAFANA_OTLP_ENDPOINT"),
		jsii.String("{{resolve:ssm:"+s.props.GrafanaEndpointSSMParam+"}}"), nil)
	fn.AddEnvironment(jsii.String("GRAFANA_OTLP_AUTH"),
		jsii.String("{{resolve:ssm:"+s.props.GrafanaAuthSSMParam+"}}"), nil)
}

func createServiceLambda(
	stack awscdk.Stack,
	serviceName string,
	environment string,
	commonEnv map[string]*string,
	memorySize float64,
	timeout float64,
	logGroup awslogs.ILogGroup,
) awslambda.Function {
	// Add service-specific environment variable
	env := make(map[string]*string)
	for k, v := range commonEnv {
		env[k] = v
	}
	env["SERVICE_NAME"] = jsii.String(serviceName)

	// Lambda function optimized for AWS Free Tier
	// Free tier: 1M requests/month, 400,000 GB-seconds compute
	// ARM64 is more cost-effective than x86
	return awslambda.NewFunction(stack, jsii.String(fmt.Sprintf("%sFunction", capitalize(serviceName))), &awslambda.FunctionProps{
		FunctionName: jsii.String(fmt.Sprintf("handloom-%s-%s", serviceName, environment)),
		Runtime:      awslambda.Runtime_PROVIDED_AL2023(),
		Handler:      jsii.String("bootstrap"),
		Code:         awslambda.Code_FromAsset(jsii.String(fmt.Sprintf("../bin/lambda/%s", serviceName)), &awss3assets.AssetOptions{}),
		Architecture: awslambda.Architecture_ARM_64(), // ARM64 is ~20% cheaper than x86
		MemorySize:   jsii.Number(memorySize),
		Timeout:      awscdk.Duration_Seconds(jsii.Number(timeout)),
		Environment:  &env,
		LogGroup:     logGroup,
		Tracing:      awslambda.Tracing_DISABLED, // Disable X-Ray tracing (not free)
	})
}

func setupAPIRoutes(api awsapigateway.RestApi, lambdas map[string]*ServiceLambda, embedderStack *EmbedderStack) {
	// Health check - use proxy integration to pass full path
	health := api.Root().AddResource(jsii.String("health"), nil)
	health.AddMethod(jsii.String("ANY"), awsapigateway.NewLambdaIntegration(lambdas["auth"].Function, &awsapigateway.LambdaIntegrationOptions{
		Proxy: jsii.Bool(true),
	}), nil)

	// Admin routes
	admin := api.Root().AddResource(jsii.String("admin"), nil)

	// Auth routes — use ANY on each resource so OPTIONS preflight reaches Lambda for CORS.
	// Cannot use {proxy+} alongside named child resources in API Gateway.
	authLambda := lambdas["auth"].Function
	authLambda.AddPermission(jsii.String("AuthApiInvoke"), &awslambda.Permission{
		Principal: awsiam.NewServicePrincipal(jsii.String("apigateway.amazonaws.com"), nil),
		Action:    jsii.String("lambda:InvokeFunction"),
		SourceArn: jsii.String(fmt.Sprintf("arn:aws:execute-api:%s:%s:%s/*",
			*awscdk.Aws_REGION(),
			*awscdk.Aws_ACCOUNT_ID(),
			*api.RestApiId(),
		)),
	})
	authIntegration := awsapigateway.NewLambdaIntegration(authLambda, &awsapigateway.LambdaIntegrationOptions{
		Proxy: jsii.Bool(true),
	})
	auth := admin.AddResource(jsii.String("auth"), nil)
	auth.AddResource(jsii.String("login"), nil).AddMethod(jsii.String("ANY"), authIntegration, nil)
	auth.AddResource(jsii.String("refresh"), nil).AddMethod(jsii.String("ANY"), authIntegration, nil)
	auth.AddResource(jsii.String("logout"), nil).AddMethod(jsii.String("ANY"), authIntegration, nil)
	auth.AddResource(jsii.String("me"), nil).AddMethod(jsii.String("ANY"), authIntegration, nil)
	password := auth.AddResource(jsii.String("password"), nil)
	password.AddResource(jsii.String("change"), nil).AddMethod(jsii.String("ANY"), authIntegration, nil)
	password.AddResource(jsii.String("reset-request"), nil).AddMethod(jsii.String("ANY"), authIntegration, nil)
	password.AddResource(jsii.String("reset"), nil).AddMethod(jsii.String("ANY"), authIntegration, nil)

	// User routes — use ANY on each resource so OPTIONS preflight reaches Lambda for CORS.
	// Cannot use {proxy+} because {id} already occupies the variable path slot.
	userLambda := lambdas["user"].Function
	userLambda.AddPermission(jsii.String("UserApiInvoke"), &awslambda.Permission{
		Principal: awsiam.NewServicePrincipal(jsii.String("apigateway.amazonaws.com"), nil),
		Action:    jsii.String("lambda:InvokeFunction"),
		SourceArn: jsii.String(fmt.Sprintf("arn:aws:execute-api:%s:%s:%s/*",
			*awscdk.Aws_REGION(),
			*awscdk.Aws_ACCOUNT_ID(),
			*api.RestApiId(),
		)),
	})
	userIntegration := awsapigateway.NewLambdaIntegration(userLambda, &awsapigateway.LambdaIntegrationOptions{
		Proxy: jsii.Bool(true),
	})
	users := admin.AddResource(jsii.String("users"), nil)
	users.AddMethod(jsii.String("ANY"), userIntegration, nil)
	userId := users.AddResource(jsii.String("{id}"), nil)
	userId.AddMethod(jsii.String("ANY"), userIntegration, nil)
	userId.AddResource(jsii.String("status"), nil).AddMethod(jsii.String("ANY"), userIntegration, nil)

	// Catalog routes (categories, products) - use proxy integration with single wildcard permission
	// This avoids the Lambda resource policy size limit (20KB) by using a single permission
	catalogLambda := lambdas["catalog"].Function

	// Add a single permission for all catalog routes (wildcard)
	catalogLambda.AddPermission(jsii.String("CatalogApiInvoke"), &awslambda.Permission{
		Principal: awsiam.NewServicePrincipal(jsii.String("apigateway.amazonaws.com"), nil),
		Action:    jsii.String("lambda:InvokeFunction"),
		SourceArn: jsii.String(fmt.Sprintf("arn:aws:execute-api:%s:%s:%s/*",
			*awscdk.Aws_REGION(),
			*awscdk.Aws_ACCOUNT_ID(),
			*api.RestApiId(),
		)),
	})

	// Create integration without auto-creating permissions (we added one above)
	catalogIntegration := awsapigateway.NewLambdaIntegration(catalogLambda, &awsapigateway.LambdaIntegrationOptions{
		Proxy: jsii.Bool(true),
	})

	// Categories routes - using ANY method with proxy to reduce permissions
	categories := admin.AddResource(jsii.String("categories"), nil)
	categories.AddMethod(jsii.String("ANY"), catalogIntegration, &awsapigateway.MethodOptions{})
	categories.AddProxy(&awsapigateway.ProxyResourceOptions{
		AnyMethod:          jsii.Bool(true),
		DefaultIntegration: catalogIntegration,
	})

	// Products routes
	products := admin.AddResource(jsii.String("products"), nil)
	products.AddMethod(jsii.String("ANY"), catalogIntegration, &awsapigateway.MethodOptions{})
	products.AddProxy(&awsapigateway.ProxyResourceOptions{
		AnyMethod:          jsii.Bool(true),
		DefaultIntegration: catalogIntegration,
	})

	// Asset routes
	if assetLambda, ok := lambdas["asset"]; ok {
		assetLambda.Function.AddPermission(jsii.String("AssetApiInvoke"), &awslambda.Permission{
			Principal: awsiam.NewServicePrincipal(jsii.String("apigateway.amazonaws.com"), nil),
			Action:    jsii.String("lambda:InvokeFunction"),
			SourceArn: jsii.String(fmt.Sprintf("arn:aws:execute-api:%s:%s:%s/*",
				*awscdk.Aws_REGION(),
				*awscdk.Aws_ACCOUNT_ID(),
				*api.RestApiId(),
			)),
		})

		assetIntegration := awsapigateway.NewLambdaIntegration(assetLambda.Function, &awsapigateway.LambdaIntegrationOptions{
			Proxy: jsii.Bool(true),
		})

		assets := admin.AddResource(jsii.String("assets"), nil)
		assets.AddMethod(jsii.String("ANY"), assetIntegration, &awsapigateway.MethodOptions{})
		assets.AddProxy(&awsapigateway.ProxyResourceOptions{
			AnyMethod:          jsii.Bool(true),
			DefaultIntegration: assetIntegration,
		})
	}

	// Order routes
	if orderLambda, ok := lambdas["order"]; ok {
		orderLambda.Function.AddPermission(jsii.String("OrderApiInvoke"), &awslambda.Permission{
			Principal: awsiam.NewServicePrincipal(jsii.String("apigateway.amazonaws.com"), nil),
			Action:    jsii.String("lambda:InvokeFunction"),
			SourceArn: jsii.String(fmt.Sprintf("arn:aws:execute-api:%s:%s:%s/*",
				*awscdk.Aws_REGION(),
				*awscdk.Aws_ACCOUNT_ID(),
				*api.RestApiId(),
			)),
		})

		orderIntegration := awsapigateway.NewLambdaIntegration(orderLambda.Function, &awsapigateway.LambdaIntegrationOptions{
			Proxy: jsii.Bool(true),
		})

		orders := admin.AddResource(jsii.String("orders"), nil)
		orders.AddMethod(jsii.String("ANY"), orderIntegration, &awsapigateway.MethodOptions{})
		orders.AddProxy(&awsapigateway.ProxyResourceOptions{
			AnyMethod:          jsii.Bool(true),
			DefaultIntegration: orderIntegration,
		})

		customers := admin.AddResource(jsii.String("customers"), nil)
		customers.AddMethod(jsii.String("ANY"), orderIntegration, &awsapigateway.MethodOptions{})
		customers.AddProxy(&awsapigateway.ProxyResourceOptions{
			AnyMethod:          jsii.Bool(true),
			DefaultIntegration: orderIntegration,
		})
	}

	// Store routes (B2C storefront) — /api/v1/store/*
	apiV1Store := api.Root().
		AddResource(jsii.String("api"), nil).
		AddResource(jsii.String("v1"), nil).
		AddResource(jsii.String("store"), nil)

	storeRoutes := map[string]string{
		"auth":     "store-auth",
		"catalog":  "store-catalog",
		"cart":     "store-cart",
		"checkout": "store-checkout",
		"orders":   "store-orders",
		"me":       "store-profile",
		"track":    "store-tracking",
		"events":   "store-events",
		"webhooks": "store-webhooks",
	}

	// Sort to make CloudFormation output deterministic — Go map iteration is
	// randomized per process and would otherwise create spurious cdk-diff noise.
	storePaths := make([]string, 0, len(storeRoutes))
	for k := range storeRoutes {
		storePaths = append(storePaths, k)
	}
	sort.Strings(storePaths)

	for _, path := range storePaths {
		svcName := storeRoutes[path]
		if svcLambda, ok := lambdas[svcName]; ok {
			svcLambda.Function.AddPermission(jsii.String(fmt.Sprintf("%sApiInvoke", capitalize(svcName))), &awslambda.Permission{
				Principal: awsiam.NewServicePrincipal(jsii.String("apigateway.amazonaws.com"), nil),
				Action:    jsii.String("lambda:InvokeFunction"),
				SourceArn: jsii.String(fmt.Sprintf("arn:aws:execute-api:%s:%s:%s/*",
					*awscdk.Aws_REGION(),
					*awscdk.Aws_ACCOUNT_ID(),
					*api.RestApiId(),
				)),
			})

			integration := awsapigateway.NewLambdaIntegration(svcLambda.Function, &awsapigateway.LambdaIntegrationOptions{
				Proxy: jsii.Bool(true),
			})

			resource := apiV1Store.AddResource(jsii.String(path), nil)
			resource.AddMethod(jsii.String("ANY"), integration, nil)

			// Mount embedder Lambda under /api/v1/store/catalog/{search,embedder-ping}.
			// API Gateway matches exact static paths before the {proxy+} fallback,
			// so these override the store-catalog Lambda for those two routes.
			if path == "catalog" && embedderStack != nil {
				embFn := embedderStack.Function
				// Create the invoke permission in APIStack (not the embedder
				// Lambda's stack) to avoid a cross-stack dependency cycle:
				// APIStack already depends on EmbedderStack for the Function
				// ARN; if AddPermission ran on the Lambda it would also make
				// EmbedderStack depend on APIStack (for the API's RestApiId).
				awslambda.NewCfnPermission(awscdk.Stack_Of(api), jsii.String("EmbedderApiInvoke"), &awslambda.CfnPermissionProps{
					Action:       jsii.String("lambda:InvokeFunction"),
					FunctionName: embFn.FunctionArn(),
					Principal:    jsii.String("apigateway.amazonaws.com"),
					SourceArn: jsii.String(fmt.Sprintf("arn:aws:execute-api:%s:%s:%s/*",
						*awscdk.Aws_REGION(),
						*awscdk.Aws_ACCOUNT_ID(),
						*api.RestApiId(),
					)),
				})
				embIntegration := awsapigateway.NewLambdaIntegration(embFn, &awsapigateway.LambdaIntegrationOptions{
					Proxy: jsii.Bool(true),
				})

				search := resource.AddResource(jsii.String("search"), nil)
				search.AddMethod(jsii.String("GET"), embIntegration, nil)
				search.AddMethod(jsii.String("OPTIONS"), embIntegration, nil)

				ping := resource.AddResource(jsii.String("embedder-ping"), nil)
				ping.AddMethod(jsii.String("GET"), embIntegration, nil)
				ping.AddMethod(jsii.String("OPTIONS"), embIntegration, nil)
			}

			resource.AddProxy(&awsapigateway.ProxyResourceOptions{
				AnyMethod:          jsii.Bool(true),
				DefaultIntegration: integration,
			})
		}
	}

	// TODO: Uncomment routes as services are implemented
	/*
		// API v1 - public routes
		apiV1 := api.Root().AddResource(jsii.String("api"), nil).AddResource(jsii.String("v1"), nil)

		// Public pricing
		pricingPublic := apiV1.AddResource(jsii.String("pricing"), nil)
		pricingIntegration := awsapigateway.NewLambdaIntegration(lambdas["pricing"].Function, nil)
		pricingPublic.AddResource(jsii.String("calculate"), nil).AddMethod(jsii.String("POST"), pricingIntegration, nil)
		pricingPublic.AddResource(jsii.String("dimension-options"), nil).AddResource(jsii.String("{categoryId}"), nil).AddMethod(jsii.String("GET"), pricingIntegration, nil)
		pricingPublic.AddResource(jsii.String("bulk-calculate"), nil).AddMethod(jsii.String("POST"), pricingIntegration, nil)

		// Catalog routes (categories, designs, products)
		catalogIntegration := awsapigateway.NewLambdaIntegration(lambdas["catalog"].Function, nil)
		addResourceRoutes(admin.AddResource(jsii.String("categories"), nil), catalogIntegration)
		addResourceRoutes(admin.AddResource(jsii.String("designs"), nil), catalogIntegration)
		addResourceRoutes(admin.AddResource(jsii.String("products"), nil), catalogIntegration)

		// Order routes
		orderIntegration := awsapigateway.NewLambdaIntegration(lambdas["order"].Function, nil)
		addResourceRoutes(admin.AddResource(jsii.String("orders"), nil), orderIntegration)
		addResourceRoutes(admin.AddResource(jsii.String("customers"), nil), orderIntegration)

		// Pricing admin routes
		pricingAdmin := admin.AddResource(jsii.String("pricing"), nil)
		pricingRules := pricingAdmin.AddResource(jsii.String("rules"), nil)
		addResourceRoutes(pricingRules, pricingIntegration)

		// Inventory routes
		inventoryIntegration := awsapigateway.NewLambdaIntegration(lambdas["inventory"].Function, nil)
		addResourceRoutes(admin.AddResource(jsii.String("inventory"), nil), inventoryIntegration)

		// Analytics routes
		analyticsIntegration := awsapigateway.NewLambdaIntegration(lambdas["analytics"].Function, nil)
		analytics := admin.AddResource(jsii.String("analytics"), nil)
		analytics.AddResource(jsii.String("dashboard"), nil).AddMethod(jsii.String("GET"), analyticsIntegration, nil)
		analytics.AddResource(jsii.String("sales"), nil).AddMethod(jsii.String("GET"), analyticsIntegration, nil)
		analytics.AddResource(jsii.String("top-products"), nil).AddMethod(jsii.String("GET"), analyticsIntegration, nil)
		analytics.AddResource(jsii.String("top-categories"), nil).AddMethod(jsii.String("GET"), analyticsIntegration, nil)
		analytics.AddResource(jsii.String("customers"), nil).AddMethod(jsii.String("GET"), analyticsIntegration, nil)
		analytics.AddResource(jsii.String("inventory"), nil).AddMethod(jsii.String("GET"), analyticsIntegration, nil)

		// Notification routes
		notificationIntegration := awsapigateway.NewLambdaIntegration(lambdas["notification"].Function, nil)
		addResourceRoutes(admin.AddResource(jsii.String("notifications"), nil), notificationIntegration)

		// Coupon routes
		couponIntegration := awsapigateway.NewLambdaIntegration(lambdas["coupon"].Function, nil)
		addResourceRoutes(admin.AddResource(jsii.String("coupons"), nil), couponIntegration)

		// Artisan routes
		artisanIntegration := awsapigateway.NewLambdaIntegration(lambdas["artisan"].Function, nil)
		addResourceRoutes(admin.AddResource(jsii.String("artisans"), nil), artisanIntegration)

		// Asset routes
		assetIntegration := awsapigateway.NewLambdaIntegration(lambdas["asset"].Function, nil)
		addResourceRoutes(admin.AddResource(jsii.String("assets"), nil), assetIntegration)

		// Report routes
		reportIntegration := awsapigateway.NewLambdaIntegration(lambdas["report"].Function, nil)
		addResourceRoutes(admin.AddResource(jsii.String("reports"), nil), reportIntegration)

		// Audit routes
		auditIntegration := awsapigateway.NewLambdaIntegration(lambdas["audit"].Function, nil)
		audit := admin.AddResource(jsii.String("audit"), nil)
		audit.AddMethod(jsii.String("GET"), auditIntegration, nil)
		audit.AddResource(jsii.String("{id}"), nil).AddMethod(jsii.String("GET"), auditIntegration, nil)
		audit.AddResource(jsii.String("entity"), nil).AddResource(jsii.String("{type}"), nil).AddResource(jsii.String("{entityId}"), nil).AddMethod(jsii.String("GET"), auditIntegration, nil)
		audit.AddResource(jsii.String("user"), nil).AddResource(jsii.String("{userId}"), nil).AddMethod(jsii.String("GET"), auditIntegration, nil)
	*/
}

func addResourceRoutes(resource awsapigateway.Resource, integration awsapigateway.LambdaIntegration) {
	resource.AddMethod(jsii.String("GET"), integration, nil)
	resource.AddMethod(jsii.String("POST"), integration, nil)

	idResource := resource.AddResource(jsii.String("{id}"), nil)
	idResource.AddMethod(jsii.String("GET"), integration, nil)
	idResource.AddMethod(jsii.String("PUT"), integration, nil)
	idResource.AddMethod(jsii.String("PATCH"), integration, nil)
	idResource.AddMethod(jsii.String("DELETE"), integration, nil)
}

func capitalize(s string) string {
	var result string
	for _, part := range strings.Split(s, "-") {
		if len(part) > 0 {
			result += string(part[0]-32) + part[1:]
		}
	}
	return result
}
