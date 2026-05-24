package stacks

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfront"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfrontorigins"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambdanodejs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// StorageStackProps holds properties for the storage stack
type StorageStackProps struct {
	awscdk.StackProps
	Environment string
}

// StorageStack contains the S3 buckets and CloudFront distribution
type StorageStack struct {
	awscdk.Stack
	AssetsBucket  awss3.Bucket
	UploadsBucket awss3.Bucket
	CDNDomain     string // CloudFront distribution domain name
	ImageResizer  awslambdanodejs.NodejsFunction
}

// NewStorageStack creates a new storage stack
// AWS Free Tier: 5GB S3 storage, 20,000 GET requests, 2,000 PUT requests
// CloudFront Free Tier: 1TB data transfer out, 10M requests/month (first 12 months)
func NewStorageStack(scope constructs.Construct, id string, props *StorageStackProps) *StorageStack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}

	stack := awscdk.NewStack(scope, &id, &sprops)
	isProd := props.Environment == "prod"

	removalPolicy := awscdk.RemovalPolicy_DESTROY
	autoDeleteObjects := jsii.Bool(true)
	if isProd {
		removalPolicy = awscdk.RemovalPolicy_RETAIN
		autoDeleteObjects = jsii.Bool(false)
	}

	// Assets bucket — fully private, accessed only via CloudFront OAC for GET,
	// and presigned PUT URLs from the browser for uploads (which need CORS).
	assetsBucket := awss3.NewBucket(stack, jsii.String("AssetsBucket"), &awss3.BucketProps{
		BucketName:        jsii.String("handloom-assets-" + props.Environment),
		RemovalPolicy:     removalPolicy,
		AutoDeleteObjects: autoDeleteObjects,
		Versioned:         jsii.Bool(false),
		Encryption:        awss3.BucketEncryption_S3_MANAGED,
		BlockPublicAccess: awss3.BlockPublicAccess_BLOCK_ALL(),
		Cors: &[]*awss3.CorsRule{
			{
				AllowedMethods: &[]awss3.HttpMethods{
					awss3.HttpMethods_GET,
					awss3.HttpMethods_PUT,
					awss3.HttpMethods_POST,
					awss3.HttpMethods_HEAD,
				},
				AllowedOrigins: jsii.Strings(
					"https://dev-admin.homechrome.in",
					"https://admin.homechrome.in",
					"http://localhost:5173",
				),
				AllowedHeaders: jsii.Strings("*"),
				ExposedHeaders: jsii.Strings("ETag"),
				MaxAge:         jsii.Number(3600),
			},
		},
		LifecycleRules: &[]*awss3.LifecycleRule{
			{
				Id:                                  jsii.String("CleanupIncompleteUploads"),
				AbortIncompleteMultipartUploadAfter: awscdk.Duration_Days(jsii.Number(1)),
			},
			{
				Id:         jsii.String("CleanupTmpUploads"),
				Prefix:     jsii.String("tmp/"),
				Expiration: awscdk.Duration_Days(jsii.Number(1)),
			},
		},
	})

	// ImageResizer Lambda — generates responsive variants (320/640/1080/1920 × webp/avif/jpg|png).
	// Invoked synchronously by the asset service after finalize (see Task 6/11 in the image-variants
	// plan). Sharp has Linux ARM64 native binaries, so ForceDockerBundling is required to produce
	// the correct binary for Lambda.
	imageResizer := awslambdanodejs.NewNodejsFunction(stack, jsii.String("ImageResizer"), &awslambdanodejs.NodejsFunctionProps{
		Entry:            jsii.String("../lambda/image-resizer/index.mjs"),
		DepsLockFilePath: jsii.String("../lambda/image-resizer/package-lock.json"),
		Runtime:          awslambda.Runtime_NODEJS_20_X(),
		Architecture:     awslambda.Architecture_ARM_64(),
		MemorySize:       jsii.Number(1024),
		Timeout:          awscdk.Duration_Seconds(jsii.Number(90)),
		Bundling: &awslambdanodejs.BundlingOptions{
			// Sharp must NOT be esbuild-bundled — it has native binaries.
			// ExternalModules skips bundling; NodeModules installs the resolved
			// native binary via npm inside the Docker container (Linux ARM64).
			// @aws-sdk/client-s3 is provided by the Node 20 Lambda runtime, so
			// mark it external too — esbuild otherwise fails to bundle its
			// transitive @smithy/core + @aws-sdk/core ESM paths.
			ExternalModules:     jsii.Strings("sharp", "@aws-sdk/client-s3"),
			NodeModules:         jsii.Strings("sharp"),
			ForceDockerBundling: jsii.Bool(true),
		},
		FunctionName: jsii.String(fmt.Sprintf("homechrome-image-resizer-%s", props.Environment)),
	})

	assetsBucket.GrantReadWrite(imageResizer, jsii.String("assets/*"))

	// CloudFront distribution for serving assets from S3
	s3Origin := awscloudfrontorigins.S3BucketOrigin_WithOriginAccessControl(assetsBucket, &awscloudfrontorigins.S3BucketOriginWithOACProps{})

	distribution := awscloudfront.NewDistribution(stack, jsii.String("AssetsCDN"), &awscloudfront.DistributionProps{
		DefaultBehavior: &awscloudfront.BehaviorOptions{
			Origin:               s3Origin,
			ViewerProtocolPolicy: awscloudfront.ViewerProtocolPolicy_REDIRECT_TO_HTTPS,
			AllowedMethods:       awscloudfront.AllowedMethods_ALLOW_GET_HEAD(),
			CachePolicy:          awscloudfront.CachePolicy_CACHING_OPTIMIZED(),
			Compress:             jsii.Bool(true),
		},
		HttpVersion: awscloudfront.HttpVersion_HTTP2_AND_3,
		PriceClass:  awscloudfront.PriceClass_PRICE_CLASS_200,
		Comment:     jsii.String(fmt.Sprintf("Handloom Assets CDN - %s", props.Environment)),
	})

	cdnDomain := *distribution.DistributionDomainName()

	// Uploads bucket — fully private, presigned URLs only
	uploadsBucket := awss3.NewBucket(stack, jsii.String("UploadsBucket"), &awss3.BucketProps{
		BucketName:        jsii.String("handloom-uploads-" + props.Environment),
		RemovalPolicy:     removalPolicy,
		AutoDeleteObjects: autoDeleteObjects,
		Encryption:        awss3.BucketEncryption_S3_MANAGED,
		BlockPublicAccess: awss3.BlockPublicAccess_BLOCK_ALL(),
		Cors: &[]*awss3.CorsRule{
			{
				AllowedMethods: &[]awss3.HttpMethods{
					awss3.HttpMethods_GET,
					awss3.HttpMethods_PUT,
					awss3.HttpMethods_POST,
				},
				AllowedOrigins: jsii.Strings(
					"https://dev-admin.homechrome.in",
					"https://admin.homechrome.in",
					"http://localhost:5173",
				),
				AllowedHeaders: jsii.Strings("*"),
				MaxAge:         jsii.Number(3600),
			},
		},
		LifecycleRules: &[]*awss3.LifecycleRule{
			{
				Id:         jsii.String("CleanupTempFiles"),
				Expiration: awscdk.Duration_Days(jsii.Number(1)),
			},
		},
	})

	// Outputs
	awscdk.NewCfnOutput(stack, jsii.String("AssetsBucketName"), &awscdk.CfnOutputProps{
		Value:       assetsBucket.BucketName(),
		Description: jsii.String("Assets S3 bucket name"),
		ExportName:  jsii.String("handloom-assets-bucket-" + props.Environment),
	})

	awscdk.NewCfnOutput(stack, jsii.String("UploadsBucketName"), &awscdk.CfnOutputProps{
		Value:       uploadsBucket.BucketName(),
		Description: jsii.String("Uploads S3 bucket name"),
		ExportName:  jsii.String("handloom-uploads-bucket-" + props.Environment),
	})

	awscdk.NewCfnOutput(stack, jsii.String("AssetsCDNDomain"), &awscdk.CfnOutputProps{
		Value:       jsii.String(cdnDomain),
		Description: jsii.String("CloudFront domain for assets"),
		ExportName:  jsii.String("handloom-assets-cdn-" + props.Environment),
	})

	return &StorageStack{
		Stack:         stack,
		AssetsBucket:  assetsBucket,
		UploadsBucket: uploadsBucket,
		CDNDomain:     cdnDomain,
		ImageResizer:  imageResizer,
	}
}
