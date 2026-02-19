package stacks

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// StorageStackProps holds properties for the storage stack
type StorageStackProps struct {
	awscdk.StackProps
	Environment string
}

// StorageStack contains the S3 buckets
type StorageStack struct {
	awscdk.Stack
	AssetsBucket  awss3.Bucket
	UploadsBucket awss3.Bucket
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
	// Public reads allowed via bucket policy for assets/* prefix; writes stay IAM-protected
	assetsBucket := awss3.NewBucket(stack, jsii.String("AssetsBucket"), &awss3.BucketProps{
		BucketName:        jsii.String("handloom-assets-" + props.Environment),
		RemovalPolicy:     removalPolicy,
		AutoDeleteObjects: autoDeleteObjects,
		Versioned:         jsii.Bool(false), // Disable versioning to save storage (free tier: 5GB)
		Encryption:        awss3.BucketEncryption_S3_MANAGED,
		BlockPublicAccess: awss3.NewBlockPublicAccess(&awss3.BlockPublicAccessOptions{
			BlockPublicAcls:       jsii.Bool(true),
			IgnorePublicAcls:      jsii.Bool(true),
			BlockPublicPolicy:     jsii.Bool(false), // Allow bucket policy for public read
			RestrictPublicBuckets: jsii.Bool(false), // Allow public access via policy
		}),
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
				AbortIncompleteMultipartUploadAfter: awscdk.Duration_Days(jsii.Number(1)),
			},
			{
				Id:         jsii.String("CleanupTmpUploads"),
				Prefix:     jsii.String("tmp/"),
				Expiration: awscdk.Duration_Days(jsii.Number(1)), // Auto-delete tmp/ objects after 24h
			},
		},
	})

	// Add bucket policy for public read on assets/* prefix
	assetsBucket.AddToResourcePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Sid:        jsii.String("PublicReadAssets"),
		Effect:     awsiam.Effect_ALLOW,
		Principals: &[]awsiam.IPrincipal{awsiam.NewAnyPrincipal()},
		Actions:    jsii.Strings("s3:GetObject"),
		Resources:  jsii.Strings(*assetsBucket.ArnForObjects(jsii.String("assets/*"))),
	}))

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

	return &StorageStack{
		Stack:         stack,
		AssetsBucket:  assetsBucket,
		UploadsBucket: uploadsBucket,
	}
}
