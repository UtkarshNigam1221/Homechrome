# Storefront Infrastructure (OpenNext + Go CDK) Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Deploy homechrome-store (Next.js 16 SSR) to AWS using OpenNext + Go CDK with S3, Lambda, and CloudFront.

**Architecture:** OpenNext compiles Next.js into Lambda-compatible artifacts. A Go CDK stack creates an S3 bucket (static assets + ISR cache), Server Lambda (SSR/API), Image Optimization Lambda, and CloudFront distribution with multiple cache behaviors. DynamoDB tag cache and SQS revalidation queue are disabled for cost savings (`disableTagCache: true`, `queue: "direct"`).

**Tech Stack:** Next.js 16, OpenNext v3 (`@opennextjs/aws`), AWS CDK v2 (Go), Lambda (Node.js 20, ARM64), CloudFront, S3

**Design doc:** `docs/plans/2026-02-28-storefront-infra-design.md`

---

### Task 1: Install OpenNext and configure Next.js

**Files:**
- Modify: `homechrome-store/package.json`
- Modify: `homechrome-store/next.config.ts`
- Create: `homechrome-store/open-next.config.ts`

**Step 1: Install OpenNext as a dev dependency**

Run:
```bash
cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store
npm install --save-dev @opennextjs/aws
```

**Step 2: Add `output: "standalone"` to next.config.ts**

OpenNext requires standalone output mode. Update `homechrome-store/next.config.ts`:

```typescript
import type { NextConfig } from 'next';

const apiBase = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8081';

const isDev = process.env.NODE_ENV === 'development';

const nextConfig: NextConfig = {
  output: 'standalone',
  images: {
    remotePatterns: [
      { protocol: 'https', hostname: '*.s3.amazonaws.com' },
      { protocol: 'http', hostname: 'localhost', port: '4566' },
    ],
    unoptimized: isDev,
  },
  async rewrites() {
    return [
      { source: '/api/:path*', destination: `${apiBase}/api/:path*` },
    ];
  },
};

export default nextConfig;
```

Only change: added `output: 'standalone'`. This does NOT affect `next dev` (local development unchanged).

**Step 3: Create open-next.config.ts**

Create `homechrome-store/open-next.config.ts`:

```typescript
import type { OpenNextConfig } from "@opennextjs/aws/types/open-next.js";

const config = {
  default: {
    override: {
      queue: "direct",
    },
  },
  dangerous: {
    disableTagCache: true,
  },
} satisfies OpenNextConfig;

export default config;
```

- `queue: "direct"` — ISR revalidation happens in-process (no SQS queue needed)
- `disableTagCache: true` — no DynamoDB table needed (time-based ISR still works, tag-based revalidation does not)

**Step 4: Verify build works**

Run:
```bash
cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store
npx next build
npx @opennextjs/aws build
ls .open-next/
```

Expected output — `.open-next/` directory containing:
- `server-functions/default/` (server Lambda handler)
- `image-optimization-function/` (image Lambda handler)
- `assets/` (static files for S3)
- `cache/` (ISR cache seed, may be empty)
- `open-next.output.json` (deployment manifest)

**Step 5: Add .open-next to .gitignore**

Append to `homechrome-store/.gitignore`:
```
# OpenNext build output
.open-next/
```

**Step 6: Commit**

```bash
git add homechrome-store/package.json homechrome-store/package-lock.json homechrome-store/next.config.ts homechrome-store/open-next.config.ts homechrome-store/.gitignore
git commit -m "feat(store): configure OpenNext for AWS Lambda deployment"
```

---

### Task 2: Create Go CDK infrastructure module

**Files:**
- Create: `homechrome-store/infra/cdk.json`
- Create: `homechrome-store/infra/go.mod` (via `go mod init`)

**Step 1: Create directory structure**

Run:
```bash
mkdir -p /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store/infra/cmd
mkdir -p /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store/infra/stacks
```

**Step 2: Create cdk.json**

Create `homechrome-store/infra/cdk.json`:

```json
{
  "app": "go run cmd/main.go",
  "watch": {
    "include": ["**"],
    "exclude": [
      "README.md",
      "cdk*.json",
      "go.mod",
      "go.sum",
      "**/*_test.go"
    ]
  },
  "context": {
    "@aws-cdk/aws-lambda:recognizeLayerVersion": true,
    "@aws-cdk/core:checkSecretUsage": true,
    "@aws-cdk/core:target-partitions": ["aws", "aws-cn"],
    "@aws-cdk/aws-iam:minimizePolicies": true,
    "@aws-cdk/core:validateSnapshotRemovalPolicy": true,
    "@aws-cdk/aws-s3:createDefaultLoggingPolicy": true,
    "@aws-cdk/core:enablePartitionLiterals": true,
    "@aws-cdk/customresources:installLatestAwsSdkDefault": false
  }
}
```

**Step 3: Initialize Go module and install dependencies**

Run:
```bash
cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store/infra
go mod init github.com/homechrome/store/infra
go get github.com/aws/aws-cdk-go/awscdk/v2@v2.170.0
go get github.com/aws/constructs-go/constructs/v10@v10.4.2
go get github.com/aws/jsii-runtime-go@v1.104.0
```

**Step 4: Commit**

```bash
git add homechrome-store/infra/cdk.json homechrome-store/infra/go.mod homechrome-store/infra/go.sum
git commit -m "feat(store/infra): initialize Go CDK module"
```

---

### Task 3: Write CDK entry point

**Files:**
- Create: `homechrome-store/infra/cmd/main.go`

**Step 1: Write main.go**

Create `homechrome-store/infra/cmd/main.go`:

```go
package main

import (
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/homechrome/store/infra/stacks"
)

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	environment := getEnvironment(app)
	env := getAWSEnv()

	createEnvironmentStack(app, environment, env)

	app.Synth(nil)
}

func createEnvironmentStack(app awscdk.App, environment string, env *awscdk.Environment) {
	certArn := getCertArn(app)
	var domainName string
	if certArn != "" {
		switch environment {
		case "prod":
			domainName = "homechrome.in"
		default:
			domainName = "dev-store.homechrome.in"
		}
	}

	stacks.NewStorefrontStack(app, "HomechromeStoreStack-"+environment, &stacks.StorefrontStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Homechrome Store - Next.js SSR hosting (" + environment + ")"),
			Tags: &map[string]*string{
				"Environment": jsii.String(environment),
				"Project":     jsii.String("homechrome-store"),
				"ManagedBy":   jsii.String("cdk"),
			},
		},
		Environment: environment,
		DomainName:  domainName,
		CertArn:     certArn,
	})
}

func getEnvironment(app constructs.Construct) string {
	if env := app.Node().TryGetContext(jsii.String("environment")); env != nil {
		return env.(string)
	}
	if env := os.Getenv("CDK_ENVIRONMENT"); env != "" {
		return env
	}
	return "dev"
}

func getCertArn(app constructs.Construct) string {
	if arn := app.Node().TryGetContext(jsii.String("certArn")); arn != nil {
		return arn.(string)
	}
	if arn := os.Getenv("ACM_CERT_ARN"); arn != "" {
		return arn
	}
	return ""
}

func getAWSEnv() *awscdk.Environment {
	account := os.Getenv("CDK_DEFAULT_ACCOUNT")
	region := os.Getenv("CDK_DEFAULT_REGION")

	if account == "" {
		account = os.Getenv("AWS_ACCOUNT_ID")
	}
	if region == "" {
		region = os.Getenv("AWS_REGION")
	}

	if account == "" || region == "" {
		return nil
	}

	return &awscdk.Environment{
		Account: jsii.String(account),
		Region:  jsii.String(region),
	}
}
```

Same pattern as `handloom-admin-frontend/infra/cmd/main.go` but simpler (no deploy mode, no API URL, no useCDN toggle — always CloudFront).

**Step 2: Commit** (will compile after Task 4)

```bash
git add homechrome-store/infra/cmd/main.go
git commit -m "feat(store/infra): add CDK entry point"
```

---

### Task 4: Write CDK storefront stack

**Files:**
- Create: `homechrome-store/infra/stacks/storefront.go`

**Step 1: Write storefront.go**

Create `homechrome-store/infra/stacks/storefront.go`:

```go
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
	Environment string
	DomainName  string // Optional: custom domain
	CertArn     string // Optional: ACM certificate ARN
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
		BucketName:        jsii.String(fmt.Sprintf("homechrome-store-%s", env)),
		RemovalPolicy:     awscdk.RemovalPolicy_DESTROY,
		AutoDeleteObjects: jsii.Bool(true),
		BlockPublicAccess: awss3.BlockPublicAccess_BLOCK_ALL(),
		Encryption:        awss3.BucketEncryption_S3_MANAGED,
		EnforceSSL:        jsii.Bool(true),
	})

	// ─── Server Lambda ───
	serverFn := awslambda.NewFunction(stack, jsii.String("ServerFunction"), &awslambda.FunctionProps{
		FunctionName: jsii.String(fmt.Sprintf("homechrome-store-server-%s", env)),
		Runtime:      awslambda.Runtime_NODEJS_20_X(),
		Handler:      jsii.String("index.handler"),
		Code:         awslambda.Code_FromAsset(jsii.String("../.open-next/server-functions/default"), nil),
		Architecture: awslambda.Architecture_ARM_64(),
		MemorySize:   jsii.Number(128),
		Timeout:      awscdk.Duration_Seconds(jsii.Number(15)),
		Environment: &map[string]*string{
			"CACHE_BUCKET_NAME":       bucket.BucketName(),
			"CACHE_BUCKET_KEY_PREFIX": jsii.String("_cache"),
			"CACHE_BUCKET_REGION":     stack.Region(),
		},
	})
	bucket.GrantReadWrite(serverFn, nil)

	serverUrl := serverFn.AddFunctionUrl(&awslambda.FunctionUrlOptions{
		AuthType:   awslambda.FunctionUrlAuthType_NONE,
		InvokeMode: awslambda.InvokeMode_RESPONSE_STREAM,
	})

	// ─── Image Optimization Lambda ───
	imageFn := awslambda.NewFunction(stack, jsii.String("ImageFunction"), &awslambda.FunctionProps{
		FunctionName: jsii.String(fmt.Sprintf("homechrome-store-image-%s", env)),
		Runtime:      awslambda.Runtime_NODEJS_20_X(),
		Handler:      jsii.String("index.handler"),
		Code:         awslambda.Code_FromAsset(jsii.String("../.open-next/image-optimization-function"), nil),
		Architecture: awslambda.Architecture_ARM_64(),
		MemorySize:   jsii.Number(256),
		Timeout:      awscdk.Duration_Seconds(jsii.Number(15)),
		Environment: &map[string]*string{
			"BUCKET_NAME":       bucket.BucketName(),
			"BUCKET_KEY_PREFIX": jsii.String("_assets"),
		},
	})
	bucket.GrantRead(imageFn, nil)

	imageUrl := imageFn.AddFunctionUrl(&awslambda.FunctionUrlOptions{
		AuthType: awslambda.FunctionUrlAuthType_NONE,
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
	serverDomain := awscdk.Fn_Select(jsii.Number(2), awscdk.Fn_Split(jsii.String("/"), serverUrl.Url(), nil))
	serverOrigin := awscloudfrontorigins.NewHttpOrigin(serverDomain, &awscloudfrontorigins.HttpOriginProps{
		ProtocolPolicy: awscloudfront.OriginProtocolPolicy_HTTPS_ONLY,
	})

	// Image Lambda origin (via Function URL)
	imageDomain := awscdk.Fn_Select(jsii.Number(2), awscdk.Fn_Split(jsii.String("/"), imageUrl.Url(), nil))
	imageOrigin := awscloudfrontorigins.NewHttpOrigin(imageDomain, &awscloudfrontorigins.HttpOriginProps{
		ProtocolPolicy: awscloudfront.OriginProtocolPolicy_HTTPS_ONLY,
	})

	// ─── CloudFront Function (inject x-forwarded-host) ───
	// CloudFront replaces Host header when forwarding to origins.
	// Next.js needs the original host to generate correct URLs.
	cfFunction := awscloudfront.NewFunction(stack, jsii.String("ForwardHostFunction"), &awscloudfront.FunctionProps{
		FunctionName: jsii.String(fmt.Sprintf("homechrome-store-fwd-host-%s", env)),
		Code: awscloudfront.FunctionCode_FromInline(jsii.String(
			`function handler(event) { var request = event.request; request.headers["x-forwarded-host"] = request.headers.host; return request; }`,
		)),
		Runtime: awscloudfront.FunctionRuntime_JS_2_0(),
	})

	// ─── Server Cache Policy ───
	// DefaultTTL=0: no caching by default.
	// CloudFront respects Cache-Control from Lambda (ISR pages get cached).
	// Cookies forwarded via OriginRequestPolicy, NOT included in cache key.
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

	// ─── CloudFront Distribution ───
	distributionProps := &awscloudfront.DistributionProps{
		// Default: all requests go to Server Lambda
		DefaultBehavior: &awscloudfront.BehaviorOptions{
			Origin:                serverOrigin,
			ViewerProtocolPolicy:  awscloudfront.ViewerProtocolPolicy_REDIRECT_TO_HTTPS,
			AllowedMethods:        awscloudfront.AllowedMethods_ALLOW_ALL(),
			CachedMethods:         awscloudfront.CachedMethods_CACHE_GET_HEAD_OPTIONS(),
			CachePolicy:           serverCachePolicy,
			OriginRequestPolicy:   awscloudfront.OriginRequestPolicy_ALL_VIEWER_EXCEPT_HOST_HEADER(),
			ResponseHeadersPolicy: responseHeadersPolicy,
			Compress:              jsii.Bool(true),
			FunctionAssociations:  lambdaFunctionAssociations,
		},
		AdditionalBehaviors: &map[string]*awscloudfront.BehaviorOptions{
			// Hashed JS/CSS/media — immutable, long cache
			"_next/static/*": {
				Origin:               s3Origin,
				ViewerProtocolPolicy: awscloudfront.ViewerProtocolPolicy_REDIRECT_TO_HTTPS,
				AllowedMethods:       awscloudfront.AllowedMethods_ALLOW_GET_HEAD(),
				CachePolicy:          awscloudfront.CachePolicy_CACHING_OPTIMIZED(),
				Compress:             jsii.Bool(true),
			},
			// Image optimization
			"_next/image": {
				Origin:               imageOrigin,
				ViewerProtocolPolicy: awscloudfront.ViewerProtocolPolicy_REDIRECT_TO_HTTPS,
				AllowedMethods:       awscloudfront.AllowedMethods_ALLOW_GET_HEAD_OPTIONS(),
				CachedMethods:        awscloudfront.CachedMethods_CACHE_GET_HEAD_OPTIONS(),
				CachePolicy:          serverCachePolicy,
				OriginRequestPolicy:  awscloudfront.OriginRequestPolicy_ALL_VIEWER_EXCEPT_HOST_HEADER(),
				Compress:             jsii.Bool(true),
				FunctionAssociations: lambdaFunctionAssociations,
			},
		},
		HttpVersion: awscloudfront.HttpVersion_HTTP2_AND_3,
		PriceClass:  awscloudfront.PriceClass_PRICE_CLASS_100,
		Comment:     jsii.String(fmt.Sprintf("Homechrome Store - %s", env)),
	}

	// Custom domain
	if props.DomainName != "" && props.CertArn != "" {
		cert := awscertificatemanager.Certificate_FromCertificateArn(stack, jsii.String("Certificate"), jsii.String(props.CertArn))
		distributionProps.DomainNames = jsii.Strings(props.DomainName)
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
	awss3deployment.NewBucketDeployment(stack, jsii.String("DeployAssets"), &awss3deployment.BucketDeploymentProps{
		Sources: &[]awss3deployment.ISource{
			awss3deployment.Source_Asset(jsii.String("../.open-next/assets"), nil),
		},
		DestinationBucket:    bucket,
		DestinationKeyPrefix: jsii.String("_assets"),
		Distribution:         distribution,
		DistributionPaths:    jsii.Strings("/*"),
		Prune:                jsii.Bool(true),
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
	if props.DomainName != "" {
		websiteURL = fmt.Sprintf("https://%s", props.DomainName)
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
```

**Key design decisions in this stack:**
- S3 origin uses `OriginPath: /_assets` — requests for `/_next/static/foo.js` resolve to S3 key `_assets/_next/static/foo.js`
- Server Lambda uses `InvokeMode: RESPONSE_STREAM` for lower time-to-first-byte
- Image Lambda gets 256MB (Sharp needs more memory than basic SSR)
- Server cache policy: `DefaultTTL=0` so CloudFront respects Cache-Control from Lambda. ISR pages return `s-maxage` and get cached; dynamic pages don't
- Cookies forwarded via `OriginRequestPolicy` (for customer auth) but NOT used as cache keys
- CloudFront Function injects `x-forwarded-host` header so Next.js knows the original domain
- `public/` directory files (SVGs) are served by the Server Lambda since the app only has 5 SVGs. If many static files are added later, create dedicated S3 behaviors for them.

**Step 2: Verify Go compilation**

Run:
```bash
cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store/infra
go build ./...
```

Expected: compiles without errors.

**Step 3: Commit**

```bash
git add homechrome-store/infra/stacks/storefront.go
git commit -m "feat(store/infra): add CloudFront + Lambda + S3 storefront stack"
```

---

### Task 5: Add deploy scripts to package.json

**Files:**
- Modify: `homechrome-store/package.json`

**Step 1: Add build and deploy scripts**

Add these scripts to `homechrome-store/package.json` (keep existing scripts, add new ones):

```json
{
  "scripts": {
    "dev": "next dev",
    "dev:local": "next dev --port 3000",
    "dev:lambda": "cp .env.local-lambda .env.local && next dev --port 3000",
    "build": "next build",
    "build:dev": "NEXT_PUBLIC_API_URL=https://dev-api.homechrome.in NEXT_PUBLIC_SITE_URL=https://dev-store.homechrome.in next build",
    "build:prod": "NEXT_PUBLIC_API_URL=https://api.homechrome.in NEXT_PUBLIC_SITE_URL=https://homechrome.in next build",
    "open-next:build": "npx @opennextjs/aws build",
    "start": "next start",
    "lint": "eslint",
    "cdk:synth:dev": "cd infra && cdk synth -c environment=dev",
    "cdk:synth:prod": "cd infra && cdk synth -c environment=prod",
    "cdk:deploy:dev": "npm run build:dev && npm run open-next:build && cd infra && cdk deploy --all --require-approval never -c environment=dev -c certArn=arn:aws:acm:us-east-1:163053486005:certificate/a8dca48d-5992-4e6e-94c9-fffc7f3ac447",
    "cdk:deploy:prod": "npm run build:prod && npm run open-next:build && cd infra && cdk deploy --all --require-approval never -c environment=prod -c certArn=arn:aws:acm:us-east-1:163053486005:certificate/a8dca48d-5992-4e6e-94c9-fffc7f3ac447",
    "cdk:destroy:dev": "cd infra && cdk destroy --all -c environment=dev",
    "cdk:destroy:prod": "cd infra && cdk destroy --all -c environment=prod"
  }
}
```

- `build:dev`/`build:prod` — inline env vars override `.env.local`, so local dev config is not affected
- `open-next:build` — transforms `.next/` into `.open-next/` Lambda artifacts
- `cdk:deploy:*` — full pipeline: build → open-next → cdk deploy

**Step 2: Commit**

```bash
git add homechrome-store/package.json
git commit -m "feat(store): add build and CDK deploy scripts"
```

---

### Task 6: Verify full build pipeline

**Step 1: Run the full build pipeline (without deploying)**

```bash
cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store
npm run build:dev
npm run open-next:build
```

Expected: `.open-next/` directory with all artifacts.

**Step 2: Run CDK synth to validate CloudFormation template**

```bash
cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store
npm run cdk:synth:dev
```

Expected: CloudFormation template JSON printed to stdout. Verify it contains:
- `AWS::S3::Bucket` (homechrome-store-dev)
- `AWS::Lambda::Function` (server + image, 2 functions)
- `AWS::Lambda::Url` (2 function URLs)
- `AWS::CloudFront::Distribution` (with 3 behaviors)
- `AWS::CloudFront::Function` (x-forwarded-host)

**Step 3: Fix any compilation or synth errors**

If errors occur, fix and re-run until synth succeeds.

**Step 4: Commit any fixes**

```bash
git add -A homechrome-store/
git commit -m "fix(store/infra): fix build pipeline issues"
```

---

### Task 7: Deploy to dev (optional — requires AWS credentials)

**Prerequisites:**
- AWS CLI configured with credentials for account `163053486005`
- CDK bootstrapped in `ap-south-1` region
- `CDK_DEFAULT_ACCOUNT` and `CDK_DEFAULT_REGION` env vars set

**Step 1: Deploy**

```bash
cd /Users/utkarsh.nigam/Desktop/CP/Homechrome/homechrome-store
npm run cdk:deploy:dev
```

Expected: CloudFormation creates all resources. Stack outputs include:
- `BucketName`: `homechrome-store-dev`
- `DistributionId`: CloudFront distribution ID
- `DistributionDomainName`: `dXXXXXXXXXX.cloudfront.net`
- `ServerFunctionName`: `homechrome-store-server-dev`
- `WebsiteURL`: `https://dev-store.homechrome.in`

**Step 2: Create DNS record (one-time)**

In Route 53 (or DNS provider), create:
- `dev-store.homechrome.in` → CNAME to `dXXXXXXXXXX.cloudfront.net`

**Step 3: Test the deployment**

Visit `https://dev-store.homechrome.in` (or the CloudFront domain directly).

Verify:
- Homepage loads with SSR content (view source shows rendered HTML, not empty div)
- `/_next/static/*` assets load (JS, CSS)
- Images load (if any use `next/image`)
- Navigation between pages works
- API calls work (cart, auth)

**Step 4: Commit**

```bash
git add -A homechrome-store/
git commit -m "feat(store): complete OpenNext + CDK infrastructure setup"
```

---

## Notes

### Public files in CloudFront
The store's `public/` directory contains only 5 SVG files (default Next.js template files). These are currently served by the Server Lambda via the default CloudFront behavior. If you add many static files to `public/` later, create dedicated CloudFront S3 behaviors for them (e.g., `*.png`, `*.jpg`) to avoid unnecessary Lambda invocations.

### Troubleshooting
- **Server Lambda OOM**: If pages fail to render, increase `MemorySize` from 128 to 256 or 512
- **Image optimization fails**: Sharp needs memory — increase Image Lambda from 256 to 512 if images fail to optimize
- **ISR not working**: Check that `CACHE_BUCKET_NAME` and `CACHE_BUCKET_REGION` env vars are correct on the server Lambda
- **Cookies not forwarded**: `OriginRequestPolicy_ALL_VIEWER_EXCEPT_HOST_HEADER` forwards all cookies. If auth breaks, check CloudFront behavior settings
- **OpenNext version compatibility**: If OpenNext doesn't support Next.js 16, try `@opennextjs/aws@latest` or check their GitHub for compatibility

### Future improvements (not in scope)
- Add DynamoDB tag cache for on-demand ISR revalidation
- Add SQS queue for async revalidation
- Add Lambda warmer to reduce cold starts
- Add CloudFront behaviors for public static files
- Add CI/CD pipeline (GitHub Actions)
