package stacks

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfront"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfrontorigins"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3deployment"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type StorefrontStackProps struct {
	awscdk.StackProps
	Environment   string
	DomainNames   []string   // Primary domain first, then aliases (e.g. ["homechrome.in", "www.homechrome.in"])
	CertArn       string
	BackendApiUrl string     // e.g. https://dev-api.homechrome.in
	LogsStack     *LogsStack // shared CloudWatch log groups (server Lambda → ServerLogGroup)

	// Telemetry — two community-published OTel Lambda layers attach to the
	// Server Lambda when CollectorLayerArn is non-empty:
	//   1. Collector layer (OTLP receiver + Grafana exporter)
	//   2. Node auto-instrumentation layer (wraps Node runtime, no app code needed)
	// We removed instrumentation.ts because Next.js 16 + OpenNext fails to
	// include it in the standalone bundle; auto-instrumentation layer covers it.
	CollectorLayerArn       string // ADOT Collector layer
	NodeAutoInstrLayerArn   string // Node auto-instrumentation layer
	GrafanaEndpointSSMParam string // Plain SSM String, e.g. /handloom/dev/grafana-otlp-endpoint
	GrafanaAuthSSMParam     string // Plain SSM String, e.g. /handloom/dev/grafana-otlp-auth
}

type StorefrontStack struct {
	awscdk.Stack
	Bucket       awss3.Bucket
	Distribution awscloudfront.Distribution
	WebsiteURL   string
}

func NewStorefrontStack(scope constructs.Construct, id string, props *StorefrontStackProps) *StorefrontStack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}

	stack := awscdk.NewStack(scope, &id, &sprops)
	env := props.Environment

	// ─── S3 Bucket (static assets + ISR cache) ───
	bucket := awss3.NewBucket(stack, jsii.String("AssetsBucket"), &awss3.BucketProps{
		BucketName:        jsii.String(fmt.Sprintf("homechrome-store-%s-mumbai", env)),
		RemovalPolicy:     awscdk.RemovalPolicy_DESTROY,
		AutoDeleteObjects: jsii.Bool(true),
		BlockPublicAccess: awss3.BlockPublicAccess_BLOCK_ALL(),
		Encryption:        awss3.BucketEncryption_S3_MANAGED,
		EnforceSSL:        jsii.Bool(true),
	})

	// ─── Server Lambda ───
	// 1024 MB gives ~0.5 vCPU on ARM64. Cuts init from ~3s (on 128) to <1s and
	// halves SSR render time, freeing concurrency slots faster under cold-cache bursts.
	//
	// Log group comes from LogsStack — single source of truth for retention.
	serverFn := awslambda.NewFunction(stack, jsii.String("ServerFunction"), &awslambda.FunctionProps{
		FunctionName: jsii.String(fmt.Sprintf("homechrome-store-server-%s", env)),
		Runtime:      awslambda.Runtime_NODEJS_22_X(),
		Handler:      jsii.String("index.handler"),
		Code:         awslambda.Code_FromAsset(jsii.String("../.open-next/server-functions/default"), nil),
		Architecture: awslambda.Architecture_ARM_64(),
		// 1769MB = 1 full vCPU (per AWS Lambda pricing tiers). Memory used by
		// the workload is only ~270MB; we bump for CPU not RAM. Next.js SSR +
		// OTel auto-instr layer is CPU-bound during cold start and SSR render.
		MemorySize: jsii.Number(1769),
		Timeout:      awscdk.Duration_Seconds(jsii.Number(15)),
		LogGroup:     props.LogsStack.ServerLogGroup,
		Environment: &map[string]*string{
			"CACHE_BUCKET_NAME":       bucket.BucketName(),
			"CACHE_BUCKET_KEY_PREFIX": jsii.String("_cache"),
			"CACHE_BUCKET_REGION":     stack.Region(),
			"NEXT_PUBLIC_API_URL":     jsii.String(props.BackendApiUrl),
		},
	})
	bucket.GrantReadWrite(serverFn, nil)

	// ─── OTel Collector layer (no-op when CollectorLayerArn is empty) ───
	// Uses the community-published OpenTelemetry Collector layer (account 184161586896).
	// Compute the ARN with stacks.OtelCollectorLayerArn from handloom-admin/infra/stacks/telemetry.go,
	// then pass via OTEL_COLLECTOR_LAYER_ARN at deploy time (see infra/cmd/main.go).
	//
	// otel.yaml is bundled into the OpenNext server-function zip via the open-next:build
	// script in package.json (copied from handloom-admin/infra/configs/otel-collector.yaml
	// into .open-next/server-functions/default/otel.yaml after open-next build).
	// Lambda extracts the zip to /var/task/, so the collector reads it from there.
	if props.CollectorLayerArn != "" {
		collectorLayer := awslambda.LayerVersion_FromLayerVersionArn(
			stack, jsii.String("ImportedOtelCollectorLayer"), jsii.String(props.CollectorLayerArn),
		)
		serverFn.AddLayers(collectorLayer)
		if props.NodeAutoInstrLayerArn != "" {
			nodeAutoInstrLayer := awslambda.LayerVersion_FromLayerVersionArn(
				stack, jsii.String("ImportedOtelNodeAutoInstrLayer"), jsii.String(props.NodeAutoInstrLayerArn),
			)
			serverFn.AddLayers(nodeAutoInstrLayer)
		}
		serverFn.AddEnvironment(jsii.String("OTEL_SERVICE_NAME"), jsii.String("homechrome-store"), nil)
		serverFn.AddEnvironment(jsii.String("OTEL_RESOURCE_ATTRIBUTES"),
			jsii.String(fmt.Sprintf("deployment.environment=%s,service.namespace=handloom", env)), nil)
		// Node OTel SDK wants a full URL with scheme. Use HTTP/protobuf to
		// localhost:4318 (collector's HTTP port). Without scheme the Node SDK
		// parses 'localhost:4317' wrong and ends up dialing localhost:443.
		serverFn.AddEnvironment(jsii.String("OTEL_EXPORTER_OTLP_PROTOCOL"), jsii.String("http/protobuf"), nil)
		serverFn.AddEnvironment(jsii.String("OTEL_EXPORTER_OTLP_ENDPOINT"), jsii.String("http://localhost:4318"), nil)
		// Disable OTel metrics export — backend collector dropped the metrics pipeline
		// (M30) and Grafana Cloud returns 404 on /v1/metrics. Traces + logs continue.
		serverFn.AddEnvironment(jsii.String("OTEL_METRICS_EXPORTER"), jsii.String("none"), nil)
		// /opt/otel-handler is provided by both layers; the Node auto-instr layer's
		// version overrides the collector's and wraps the Node runtime so app code
		// is traced without any instrumentation.ts in source.
		serverFn.AddEnvironment(jsii.String("AWS_LAMBDA_EXEC_WRAPPER"), jsii.String("/opt/otel-handler"), nil)
		serverFn.AddEnvironment(jsii.String("OPENTELEMETRY_COLLECTOR_CONFIG_URI"),
			jsii.String("/var/task/otel.yaml"), nil) // yaml bundled into Lambda zip, not baked into layer
		if props.GrafanaEndpointSSMParam != "" {
			serverFn.AddEnvironment(jsii.String("GRAFANA_OTLP_ENDPOINT"),
				jsii.String("{{resolve:ssm:"+props.GrafanaEndpointSSMParam+"}}"), nil)
		}
		if props.GrafanaAuthSSMParam != "" {
			serverFn.AddEnvironment(jsii.String("GRAFANA_OTLP_AUTH"),
				jsii.String("{{resolve:ssm:"+props.GrafanaAuthSSMParam+"}}"), nil)
		}
	}

	serverUrl := serverFn.AddFunctionUrl(&awslambda.FunctionUrlOptions{
		AuthType:   awslambda.FunctionUrlAuthType_NONE,
		InvokeMode: awslambda.InvokeMode_BUFFERED,
	})
	// Since Oct 2025, new Function URLs require lambda:InvokeFunction in addition to lambda:InvokeFunctionUrl
	serverFn.AddPermission(jsii.String("PublicInvoke"), &awslambda.Permission{
		Principal: awsiam.NewAnyPrincipal(),
		Action:    jsii.String("lambda:InvokeFunction"),
	})

	// ─── CloudFront Origins ───

	// S3 origin with OAC (serves _assets/ prefix)
	oac := awscloudfront.NewS3OriginAccessControl(stack, jsii.String("OAC"), &awscloudfront.S3OriginAccessControlProps{
		OriginAccessControlName: jsii.String(fmt.Sprintf("homechrome-store-oac-%s", env)),
		Signing:                 awscloudfront.Signing_SIGV4_ALWAYS(),
	})

	s3Origin := awscloudfrontorigins.S3BucketOrigin_WithOriginAccessControl(bucket, &awscloudfrontorigins.S3BucketOriginWithOACProps{
		OriginAccessControl: oac,
		OriginPath:          jsii.String("/_assets"),
	})

	// Server Lambda origin (via Function URL)
	// Function URL format: https://xxxxx.lambda-url.region.on.aws/
	// Split by "/" → ["https:", "", "xxxxx.lambda-url.region.on.aws"] → select index 2
	serverDomain := awscdk.Fn_Select(jsii.Number(2), awscdk.Fn_Split(jsii.String("/"), serverUrl.Url(), nil))
	serverOrigin := awscloudfrontorigins.NewHttpOrigin(serverDomain, &awscloudfrontorigins.HttpOriginProps{
		ProtocolPolicy: awscloudfront.OriginProtocolPolicy_HTTPS_ONLY,
	})

	// ─── CloudFront Function (inject x-forwarded-host) ───
	// CloudFront replaces the Host header when forwarding to origins.
	// Next.js needs the original host for canonical URLs and redirects.
	cfFunction := awscloudfront.NewFunction(stack, jsii.String("ForwardHostFunction"), &awscloudfront.FunctionProps{
		FunctionName: jsii.String(fmt.Sprintf("homechrome-store-fwd-host-%s", env)),
		Code: awscloudfront.FunctionCode_FromInline(jsii.String(
			`function handler(event) { var request = event.request; request.headers["x-forwarded-host"] = request.headers.host; return request; }`,
		)),
		Runtime: awscloudfront.FunctionRuntime_JS_2_0(),
	})

	// ─── Server Cache Policy ───
	serverCachePolicy := awscloudfront.NewCachePolicy(stack, jsii.String("ServerCachePolicy"), &awscloudfront.CachePolicyProps{
		CachePolicyName:            jsii.String(fmt.Sprintf("homechrome-store-server-cache-%s", env)),
		DefaultTtl:                 awscdk.Duration_Seconds(jsii.Number(0)),
		MaxTtl:                     awscdk.Duration_Days(jsii.Number(365)),
		MinTtl:                     awscdk.Duration_Seconds(jsii.Number(0)),
		EnableAcceptEncodingGzip:   jsii.Bool(true),
		EnableAcceptEncodingBrotli: jsii.Bool(true),
		QueryStringBehavior:        awscloudfront.CacheQueryStringBehavior_All(),
		HeaderBehavior: awscloudfront.CacheHeaderBehavior_AllowList(
			jsii.String("accept"),
			jsii.String("rsc"),
			jsii.String("next-router-prefetch"),
			jsii.String("next-router-state-tree"),
			jsii.String("next-url"),
			jsii.String("x-prerender-revalidate"),
		),
		CookieBehavior: awscloudfront.CacheCookieBehavior_None(),
	})

	// ─── Security Headers ───
	responseHeadersPolicy := awscloudfront.NewResponseHeadersPolicy(stack, jsii.String("SecurityHeadersPolicy"), &awscloudfront.ResponseHeadersPolicyProps{
		ResponseHeadersPolicyName: jsii.String(fmt.Sprintf("homechrome-store-security-%s", env)),
		SecurityHeadersBehavior: &awscloudfront.ResponseSecurityHeadersBehavior{
			ContentTypeOptions: &awscloudfront.ResponseHeadersContentTypeOptions{
				Override: jsii.Bool(true),
			},
			FrameOptions: &awscloudfront.ResponseHeadersFrameOptions{
				FrameOption: awscloudfront.HeadersFrameOption_DENY,
				Override:    jsii.Bool(true),
			},
			ReferrerPolicy: &awscloudfront.ResponseHeadersReferrerPolicy{
				ReferrerPolicy: awscloudfront.HeadersReferrerPolicy_STRICT_ORIGIN_WHEN_CROSS_ORIGIN,
				Override:       jsii.Bool(true),
			},
			StrictTransportSecurity: &awscloudfront.ResponseHeadersStrictTransportSecurity{
				AccessControlMaxAge: awscdk.Duration_Seconds(jsii.Number(31536000)),
				IncludeSubdomains:   jsii.Bool(true),
				Override:            jsii.Bool(true),
			},
			XssProtection: &awscloudfront.ResponseHeadersXSSProtection{
				Protection: jsii.Bool(true),
				ModeBlock:  jsii.Bool(true),
				Override:   jsii.Bool(true),
			},
		},
	})

	// ─── Lambda behavior helpers ───
	lambdaFunctionAssociations := &[]*awscloudfront.FunctionAssociation{
		{
			Function:  cfFunction,
			EventType: awscloudfront.FunctionEventType_VIEWER_REQUEST,
		},
	}

	// ─── Origin Request Policy ───
	// ALL_VIEWER_EXCEPT_HOST_HEADER is a managed policy that forwards all viewer
	// headers but does NOT include CloudFront-injected geo headers. We replace it
	// with a custom allowlist so the Next.js middleware can read viewer geo data.
	storefrontOriginPolicy := awscloudfront.NewOriginRequestPolicy(stack, jsii.String("StorefrontOriginPolicy"), &awscloudfront.OriginRequestPolicyProps{
		OriginRequestPolicyName: jsii.String(fmt.Sprintf("homechrome-store-origin-policy-%s", env)),
		Comment:                 jsii.String("Forward viewer headers + CloudFront geo headers to origin"),
		// AllowList capped at 10 headers (default CloudFront quota). Visitor
		// attribution is packed into a single X-Hc-Visitor header (assembled
		// browser-side by axios, geo-augmented server-side by middleware.ts)
		// so we spend just one slot on the entire attribution tuple instead
		// of one per label. Leaves room for future expansion.
		//
		// IMPORTANT: do NOT switch to `All` mode. OpenNext's Server Lambda
		// is fronted by a Function URL with OAC; OAC signs requests with
		// SigV4 including the Host header. `All` forwards viewer Host →
		// signature mismatch → 403 AccessDeniedException at the edge.
		//
		// Accept-Encoding + Authorization cannot be forwarded via
		// OriginRequestPolicy per CloudFront constraint; OpenNext's default
		// cache policy already covers Accept-Encoding for compression.
		HeaderBehavior: awscloudfront.OriginRequestHeaderBehavior_AllowList(
			jsii.String("Accept"),
			jsii.String("x-forwarded-host"),
			jsii.String("CloudFront-Viewer-Country"),
			jsii.String("CloudFront-Viewer-City"),
			jsii.String("CloudFront-Viewer-Latitude"),
			jsii.String("CloudFront-Viewer-Longitude"),
			jsii.String("X-Hc-Visitor"),
		),
		QueryStringBehavior: awscloudfront.OriginRequestQueryStringBehavior_All(),
		CookieBehavior:      awscloudfront.OriginRequestCookieBehavior_All(),
	})

	// ─── CloudFront Distribution ───
	distributionProps := &awscloudfront.DistributionProps{
		DefaultBehavior: &awscloudfront.BehaviorOptions{
			Origin:                serverOrigin,
			ViewerProtocolPolicy:  awscloudfront.ViewerProtocolPolicy_REDIRECT_TO_HTTPS,
			AllowedMethods:        awscloudfront.AllowedMethods_ALLOW_ALL(),
			CachedMethods:         awscloudfront.CachedMethods_CACHE_GET_HEAD_OPTIONS(),
			CachePolicy:           serverCachePolicy,
			OriginRequestPolicy:   storefrontOriginPolicy,
			ResponseHeadersPolicy: responseHeadersPolicy,
			Compress:              jsii.Bool(true),
			FunctionAssociations:  lambdaFunctionAssociations,
		},
		AdditionalBehaviors: &map[string]*awscloudfront.BehaviorOptions{
			"_next/static/*": {
				Origin:               s3Origin,
				ViewerProtocolPolicy: awscloudfront.ViewerProtocolPolicy_REDIRECT_TO_HTTPS,
				AllowedMethods:       awscloudfront.AllowedMethods_ALLOW_GET_HEAD(),
				CachePolicy:          awscloudfront.CachePolicy_CACHING_OPTIMIZED(),
				Compress:             jsii.Bool(true),
			},
		},
		HttpVersion: awscloudfront.HttpVersion_HTTP2_AND_3,
		PriceClass:  awscloudfront.PriceClass_PRICE_CLASS_200,
		Comment:     jsii.String(fmt.Sprintf("Homechrome Store - %s", env)),
	}

	// Custom domains (primary + aliases)
	if len(props.DomainNames) > 0 && props.CertArn != "" {
		cert := awscertificatemanager.Certificate_FromCertificateArn(stack, jsii.String("Certificate"), jsii.String(props.CertArn))
		distributionProps.DomainNames = jsii.Strings(props.DomainNames...)
		distributionProps.Certificate = cert
	}

	distribution := awscloudfront.NewDistribution(stack, jsii.String("Distribution"), distributionProps)

	// Grant CloudFront access to S3
	bucket.AddToResourcePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   jsii.Strings("s3:GetObject"),
		Resources: jsii.Strings(*bucket.ArnForObjects(jsii.String("*"))),
		Principals: &[]awsiam.IPrincipal{
			awsiam.NewServicePrincipal(jsii.String("cloudfront.amazonaws.com"), nil),
		},
		Conditions: &map[string]interface{}{
			"StringEquals": map[string]*string{
				"AWS:SourceArn": jsii.String(fmt.Sprintf("arn:aws:cloudfront::%s:distribution/%s",
					*awscdk.Aws_ACCOUNT_ID(), *distribution.DistributionId())),
			},
		},
	}))

	// ─── Deploy static assets to S3 ───
	// NOTE: CloudFront invalidation is handled by the deploy script after `cdk deploy`.
	// Including it here caused the custom resource Lambda's invalidation waiter to
	// exceed max attempts on slow distributions, failing the entire deployment.
	awss3deployment.NewBucketDeployment(stack, jsii.String("DeployAssets"), &awss3deployment.BucketDeploymentProps{
		Sources: &[]awss3deployment.ISource{
			awss3deployment.Source_Asset(jsii.String("../.open-next/assets"), nil),
		},
		DestinationBucket:    bucket,
		DestinationKeyPrefix: jsii.String("_assets"),
		Prune:                jsii.Bool(true),
		CacheControl: &[]awss3deployment.CacheControl{
			awss3deployment.CacheControl_MaxAge(awscdk.Duration_Days(jsii.Number(365))),
			awss3deployment.CacheControl_SetPublic(),
			awss3deployment.CacheControl_Immutable(),
		},
	})

	// Deploy ISR cache seed to S3 (prune=false to preserve runtime cache)
	awss3deployment.NewBucketDeployment(stack, jsii.String("DeployCache"), &awss3deployment.BucketDeploymentProps{
		Sources: &[]awss3deployment.ISource{
			awss3deployment.Source_Asset(jsii.String("../.open-next/cache"), nil),
		},
		DestinationBucket:    bucket,
		DestinationKeyPrefix: jsii.String("_cache"),
		Prune:                jsii.Bool(false),
	})

	// ─── Outputs ───
	awscdk.NewCfnOutput(stack, jsii.String("BucketName"), &awscdk.CfnOutputProps{
		Value:       bucket.BucketName(),
		Description: jsii.String("S3 bucket for assets and cache"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("DistributionId"), &awscdk.CfnOutputProps{
		Value:       distribution.DistributionId(),
		Description: jsii.String("CloudFront distribution ID"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("DistributionDomainName"), &awscdk.CfnOutputProps{
		Value:       distribution.DistributionDomainName(),
		Description: jsii.String("CloudFront domain name"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("ServerFunctionName"), &awscdk.CfnOutputProps{
		Value:       serverFn.FunctionName(),
		Description: jsii.String("Server Lambda function name"),
	})

	websiteURL := fmt.Sprintf("https://%s", *distribution.DistributionDomainName())
	if len(props.DomainNames) > 0 {
		websiteURL = fmt.Sprintf("https://%s", props.DomainNames[0])
	}
	awscdk.NewCfnOutput(stack, jsii.String("WebsiteURL"), &awscdk.CfnOutputProps{
		Value:       jsii.String(websiteURL),
		Description: jsii.String("Website URL"),
	})

	return &StorefrontStack{
		Stack:        stack,
		Bucket:       bucket,
		Distribution: distribution,
		WebsiteURL:   websiteURL,
	}
}

