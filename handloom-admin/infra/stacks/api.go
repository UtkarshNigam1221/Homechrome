package stacks

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigateway"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3assets"
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
	EventStack     *EventStack // Optional: event-driven async infrastructure
	DomainName     string      // Optional: custom domain for API Gateway (e.g. dev-api.lldlab.com)
	FrontendOrigin string      // Optional: frontend origin for CORS (e.g. https://dev.lldlab.com)
	CertArn        string      // Optional: ACM certificate ARN (us-east-1) for custom domain
}

// ServiceLambda represents a Lambda function for a service
type ServiceLambda struct {
	Function awslambda.Function
	Name     string
}

// APIStack contains the API Gateway and Lambda functions
type APIStack struct {
	awscdk.Stack
	API     awsapigateway.RestApi
	Lambdas map[string]*ServiceLambda
}

// NewAPIStack creates a new API stack
func NewAPIStack(scope constructs.Construct, id string, props *APIStackProps) *APIStack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}

	stack := awscdk.NewStack(scope, &id, &sprops)
	isProd := props.Environment == "prod"

	// JWT Secret parameter
	jwtSecret := awsssm.NewStringParameter(stack, jsii.String("JwtSecret"), &awsssm.StringParameterProps{
		ParameterName: jsii.String(fmt.Sprintf("/handloom/%s/jwt-secret", props.Environment)),
		StringValue:   jsii.String("CHANGE_ME_IN_PRODUCTION"),
		Description:   jsii.String("JWT Secret for token signing"),
		Tier:          awsssm.ParameterTier_STANDARD,
	})

	// S3 buckets from StorageStack
	assetsBucket := props.StorageStack.AssetsBucket

	// Common environment variables for all Lambdas
	commonEnv := map[string]*string{
		"APP_ENV":                    jsii.String(props.Environment),
		"APP_DEBUG":                  jsii.String(fmt.Sprintf("%t", !isProd)),
		"DYNAMODB_CORE_TABLE":          props.DatabaseStack.CoreTable.TableName(),
		"DYNAMODB_ORDERS_TABLE":        props.DatabaseStack.OrdersTable.TableName(),
		"DYNAMODB_SESSIONS_TABLE":      props.DatabaseStack.SessionsTable.TableName(),
		"DYNAMODB_AUDIT_TABLE":         props.DatabaseStack.AuditTable.TableName(),
		"DYNAMODB_ANALYTICS_TABLE":     props.DatabaseStack.AnalyticsTable.TableName(),
		"DYNAMODB_NOTIFICATIONS_TABLE": props.DatabaseStack.NotificationsTable.TableName(),
		"DYNAMODB_EVENTS_TABLE":        props.DatabaseStack.EventsTable.TableName(),
		"RDS_SECRET_ARN": props.DatabaseStack.CatalogDBSecret.SecretArn(),
		"RDS_ENDPOINT":   props.DatabaseStack.CatalogDB.DbInstanceEndpointAddress(),
		"RDS_PORT":       jsii.String("5432"),
		"RDS_DATABASE":   jsii.String("handloom"),
		"S3_ASSETS_BUCKET":           assetsBucket.BucketName(),
		"JWT_SECRET_PARAM":           jwtSecret.ParameterName(),
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
		commonEnv["COOKIE_DOMAIN"] = jsii.String(".homechrome.lldlab.com")
	}

	// Add event publishing env vars when EventStack is available
	if props.EventStack != nil {
		commonEnv["SNS_TOPIC_ARN"] = props.EventStack.TopicARN
		commonEnv["EVENT_PUBLISHING_ENABLED"] = jsii.String("true")
	}

	// Memory sizes - optimized for AWS Free Tier
	// Free tier: 1M requests/month, 400,000 GB-seconds compute time
	// 128MB = ~3.2M seconds/month free (plenty for development)
	memorySize := float64(128)
	if isProd {
		memorySize = 256 // Slightly higher for production, still cost-effective
	}

	// Log retention - minimize to reduce CloudWatch costs
	// Free tier: 5GB ingestion, 5GB storage, 5GB data scanned
	logRetention := awslogs.RetentionDays_THREE_DAYS
	if isProd {
		logRetention = awslogs.RetentionDays_ONE_WEEK
	}

	// Create Lambda functions for each service
	// TODO: Uncomment services as they are implemented
	services := []string{
		"auth",
		"user",
		"catalog",
		"asset",
		// "order",
		// "pricing",
		// "inventory",
		// "analytics",
		// "notification",
		// "coupon",
		// "artisan",
		// "report",
		// "audit",
	}

	lambdas := make(map[string]*ServiceLambda)
	for _, svc := range services {
		lambdaFn := createServiceLambda(stack, svc, props.Environment, commonEnv, memorySize, logRetention)
		lambdas[svc] = &ServiceLambda{
			Function: lambdaFn,
			Name:     svc,
		}

		// Grant permissions
		props.DatabaseStack.CoreTable.GrantReadWriteData(lambdaFn)
		props.DatabaseStack.OrdersTable.GrantReadWriteData(lambdaFn)
		props.DatabaseStack.SessionsTable.GrantReadWriteData(lambdaFn)
		props.DatabaseStack.AuditTable.GrantReadWriteData(lambdaFn)
		props.DatabaseStack.AnalyticsTable.GrantReadWriteData(lambdaFn)
		props.DatabaseStack.NotificationsTable.GrantReadWriteData(lambdaFn)
		props.DatabaseStack.EventsTable.GrantReadWriteData(lambdaFn)
		assetsBucket.GrantReadWrite(lambdaFn, nil)
		jwtSecret.GrantRead(lambdaFn)
		props.DatabaseStack.CatalogDBSecret.GrantRead(lambdaFn, nil)

		// Grant SNS publish permission when EventStack is available
		if props.EventStack != nil {
			props.EventStack.Topic.GrantPublish(lambdaFn)
		}
	}

	// Create API Gateway - optimized for AWS Free Tier
	// Free tier: 1M API calls/month for first 12 months
	api := awsapigateway.NewRestApi(stack, jsii.String("API"), &awsapigateway.RestApiProps{
		RestApiName: jsii.String("handloom-api-" + props.Environment),
		Description: jsii.String("Handloom Admin API - CORS via Lambda"),
		DeployOptions: &awsapigateway.StageOptions{
			StageName:            jsii.String(props.Environment),
			ThrottlingRateLimit:  jsii.Number(50),                      // Lower throttle for cost control
			ThrottlingBurstLimit: jsii.Number(100),                     // Lower burst for cost control
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
	setupAPIRoutes(api, lambdas)

	// Custom domain for API Gateway (edge-optimized)
	if props.CertArn != "" && props.DomainName != "" {
		cert := awscertificatemanager.Certificate_FromCertificateArn(stack, jsii.String("ApiCertificate"), jsii.String(props.CertArn))

		customDomain := awsapigateway.NewDomainName(stack, jsii.String("ApiCustomDomain"), &awsapigateway.DomainNameProps{
			DomainName:  jsii.String(props.DomainName),
			Certificate: cert,
			EndpointType: awsapigateway.EndpointType_EDGE,
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
			Description: jsii.String("CNAME target for API custom domain (add to Cloudflare DNS)"),
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

	return &APIStack{
		Stack:   stack,
		API:     api,
		Lambdas: lambdas,
	}
}

func createServiceLambda(
	stack awscdk.Stack,
	serviceName string,
	environment string,
	commonEnv map[string]*string,
	memorySize float64,
	logRetention awslogs.RetentionDays,
) awslambda.Function {
	// Add service-specific environment variable
	env := make(map[string]*string)
	for k, v := range commonEnv {
		env[k] = v
	}
	env["SERVICE_NAME"] = jsii.String(serviceName)

	// Explicit log group (replaces deprecated LogRetention on Lambda)
	logGroup := awslogs.NewLogGroup(stack, jsii.String(fmt.Sprintf("%sLogGroup", capitalize(serviceName))), &awslogs.LogGroupProps{
		LogGroupName: jsii.String(fmt.Sprintf("/aws/lambda/handloom-%s-%s", serviceName, environment)),
		Retention:    logRetention,
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})

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
		Timeout:      awscdk.Duration_Seconds(jsii.Number(15)), // Reduced timeout for cost efficiency
		Environment:  &env,
		LogGroup:     logGroup,
		Tracing:      awslambda.Tracing_DISABLED, // Disable X-Ray tracing (not free)
	})
}

func setupAPIRoutes(api awsapigateway.RestApi, lambdas map[string]*ServiceLambda) {
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
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}
