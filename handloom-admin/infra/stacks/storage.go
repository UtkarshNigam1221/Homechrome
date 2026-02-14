package stacks

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfront"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfrontorigins"
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
	AssetsBucket    awss3.Bucket
	UploadsBucket   awss3.Bucket
	CDNDistribution awscloudfront.Distribution
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

	// Assets bucket - for product images, etc.
	// Optimized for Free Tier: 5GB storage limit
	assetsBucket := awss3.NewBucket(stack, jsii.String("AssetsBucket"), &awss3.BucketProps{
		BucketName:        jsii.String("handloom-assets-" + props.Environment),
		RemovalPolicy:     removalPolicy,
		AutoDeleteObjects: autoDeleteObjects,
		Versioned:         jsii.Bool(false), // Disable versioning to save storage (free tier: 5GB)
		Encryption:        awss3.BucketEncryption_S3_MANAGED,
		BlockPublicAccess: awss3.BlockPublicAccess_BLOCK_ALL(),
		Cors: &[]*awss3.CorsRule{
			{
				AllowedMethods: &[]awss3.HttpMethods{
					awss3.HttpMethods_GET,
					awss3.HttpMethods_PUT,
					awss3.HttpMethods_POST,
				},
				AllowedOrigins: jsii.Strings("*"),
				AllowedHeaders: jsii.Strings("*"),
				MaxAge:         jsii.Number(3600),
			},
		},
		LifecycleRules: &[]*awss3.LifecycleRule{
			{
				Id:                          jsii.String("CleanupIncompleteUploads"),
				AbortIncompleteMultipartUploadAfter: awscdk.Duration_Days(jsii.Number(1)), // Cleanup faster to save storage
			},
		},
	})

	// Uploads bucket - for temporary uploads
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
				AllowedOrigins: jsii.Strings("*"),
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

	// CloudFront distribution for assets
	oai := awscloudfront.NewOriginAccessIdentity(stack, jsii.String("OAI"), &awscloudfront.OriginAccessIdentityProps{
		Comment: jsii.String("OAI for handloom assets"),
	})
	assetsBucket.GrantRead(oai, nil)

	// CloudFront distribution - optimized for Free Tier
	// Free tier (first 12 months): 1TB data transfer out, 10M HTTP/HTTPS requests
	cdn := awscloudfront.NewDistribution(stack, jsii.String("CDN"), &awscloudfront.DistributionProps{
		Comment: jsii.String("Handloom Assets CDN - " + props.Environment),
		DefaultBehavior: &awscloudfront.BehaviorOptions{
			Origin: awscloudfrontorigins.NewS3Origin(assetsBucket, &awscloudfrontorigins.S3OriginProps{
				OriginAccessIdentity: oai,
			}),
			ViewerProtocolPolicy: awscloudfront.ViewerProtocolPolicy_REDIRECT_TO_HTTPS,
			CachePolicy:          awscloudfront.CachePolicy_CACHING_OPTIMIZED(), // Maximize caching to reduce origin requests
			AllowedMethods:       awscloudfront.AllowedMethods_ALLOW_GET_HEAD(),  // Minimal methods to reduce complexity
		},
		PriceClass:        awscloudfront.PriceClass_PRICE_CLASS_100, // Cheapest price class (US, Canada, Europe only)
		HttpVersion:       awscloudfront.HttpVersion_HTTP2,          // HTTP/2 only (HTTP/3 adds cost)
		EnableIpv6:        jsii.Bool(true),
		MinimumProtocolVersion: awscloudfront.SecurityPolicyProtocol_TLS_V1_2_2021,
		Enabled:           jsii.Bool(true),
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

	awscdk.NewCfnOutput(stack, jsii.String("CDNDomain"), &awscdk.CfnOutputProps{
		Value:       cdn.DistributionDomainName(),
		Description: jsii.String("CloudFront distribution domain"),
		ExportName:  jsii.String("handloom-cdn-domain-" + props.Environment),
	})

	return &StorageStack{
		Stack:           stack,
		AssetsBucket:    assetsBucket,
		UploadsBucket:   uploadsBucket,
		CDNDistribution: cdn,
	}
}
