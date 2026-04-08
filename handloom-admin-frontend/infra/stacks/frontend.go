package stacks

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscertificatemanager"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfront"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscloudfrontorigins"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3deployment"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// FrontendStackProps holds properties for the frontend stack
type FrontendStackProps struct {
	awscdk.StackProps
	Environment string
	APIURL      string
	DomainName  string // Optional: custom domain name
	CertArn     string // Optional: ACM certificate ARN for custom domain
	UseCDN      bool   // If false, use S3 static website hosting (cheaper, HTTP only)
}

// FrontendStack contains the S3 bucket and CloudFront distribution for hosting
type FrontendStack struct {
	awscdk.Stack
	Bucket       awss3.Bucket
	Distribution awscloudfront.Distribution // nil if UseCDN is false
	WebsiteURL   string
}

// NewFrontendStack creates a new frontend hosting stack
func NewFrontendStack(scope constructs.Construct, id string, props *FrontendStackProps) *FrontendStack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}

	stack := awscdk.NewStack(scope, &id, &sprops)
	isProd := props.Environment == "prod"

	// Determine if we should use CDN
	// Default: CDN for prod, S3 direct for dev (to save costs)
	useCDN := props.UseCDN
	if !useCDN && isProd {
		useCDN = true // Always use CDN for production
	}

	if useCDN {
		return createCloudFrontStack(stack, props, isProd)
	}
	return createS3OnlyStack(stack, props, isProd)
}

// createS3OnlyStack creates a stack with S3 static website hosting (no CloudFront)
// This is FREE (within S3 free tier limits) but HTTP only
func createS3OnlyStack(stack awscdk.Stack, props *FrontendStackProps, isProd bool) *FrontendStack {
	// S3 bucket configured for static website hosting
	bucket := awss3.NewBucket(stack, jsii.String("WebsiteBucket"), &awss3.BucketProps{
		BucketName:        jsii.String(fmt.Sprintf("handloom-admin-frontend-%s", props.Environment)),
		RemovalPolicy:     getRemovalPolicy(isProd),
		AutoDeleteObjects: jsii.Bool(!isProd),
		// For S3 static website hosting, we need public access
		BlockPublicAccess: awss3.NewBlockPublicAccess(&awss3.BlockPublicAccessOptions{
			BlockPublicAcls:       jsii.Bool(false),
			BlockPublicPolicy:     jsii.Bool(false),
			IgnorePublicAcls:      jsii.Bool(false),
			RestrictPublicBuckets: jsii.Bool(false),
		}),
		WebsiteIndexDocument: jsii.String("index.html"),
		WebsiteErrorDocument: jsii.String("index.html"), // SPA routing
		PublicReadAccess:     jsii.Bool(true),
	})

	// Deploy website content
	awss3deployment.NewBucketDeployment(stack, jsii.String("DeployWebsite"), &awss3deployment.BucketDeploymentProps{
		Sources: &[]awss3deployment.ISource{
			awss3deployment.Source_Asset(jsii.String("../dist"), nil),
		},
		DestinationBucket: bucket,
		Prune:             jsii.Bool(true),
	})

	// Website URL (HTTP only)
	websiteURL := bucket.BucketWebsiteUrl()

	// Outputs
	awscdk.NewCfnOutput(stack, jsii.String("BucketName"), &awscdk.CfnOutputProps{
		Value:       bucket.BucketName(),
		Description: jsii.String("S3 bucket name for website hosting"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("WebsiteURL"), &awscdk.CfnOutputProps{
		Value:       websiteURL,
		Description: jsii.String("Website URL (HTTP only - use CloudFront for HTTPS)"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("Note"), &awscdk.CfnOutputProps{
		Value:       jsii.String("Using S3 static hosting (FREE). For HTTPS, deploy with useCDN=true"),
		Description: jsii.String("Deployment mode"),
	})

	return &FrontendStack{
		Stack:        stack,
		Bucket:       bucket,
		Distribution: nil,
		WebsiteURL:   *websiteURL,
	}
}

// createCloudFrontStack creates a stack with S3 + CloudFront (HTTPS, CDN)
// This costs money but provides HTTPS and better performance
func createCloudFrontStack(stack awscdk.Stack, props *FrontendStackProps, isProd bool) *FrontendStack {
	// S3 bucket - private, accessed only via CloudFront OAC
	bucket := awss3.NewBucket(stack, jsii.String("WebsiteBucket"), &awss3.BucketProps{
		BucketName:        jsii.String(fmt.Sprintf("handloom-admin-frontend-%s-mumbai", props.Environment)),
		RemovalPolicy:     getRemovalPolicy(isProd),
		AutoDeleteObjects: jsii.Bool(!isProd),
		BlockPublicAccess: awss3.BlockPublicAccess_BLOCK_ALL(),
		Encryption:        awss3.BucketEncryption_S3_MANAGED,
		EnforceSSL:        jsii.Bool(true),
		Versioned:         jsii.Bool(isProd),
	})

	// CloudFront Origin Access Control
	oac := awscloudfront.NewS3OriginAccessControl(stack, jsii.String("OAC"), &awscloudfront.S3OriginAccessControlProps{
		OriginAccessControlName: jsii.String(fmt.Sprintf("handloom-frontend-oac-%s", props.Environment)),
		Signing:                 awscloudfront.Signing_SIGV4_ALWAYS(),
	})

	// S3 origin with OAC
	s3Origin := awscloudfrontorigins.S3BucketOrigin_WithOriginAccessControl(bucket, &awscloudfrontorigins.S3BucketOriginWithOACProps{
		OriginAccessControl: oac,
	})

	// Response headers policy for security
	responseHeadersPolicy := awscloudfront.NewResponseHeadersPolicy(stack, jsii.String("SecurityHeadersPolicy"), &awscloudfront.ResponseHeadersPolicyProps{
		ResponseHeadersPolicyName: jsii.String(fmt.Sprintf("handloom-security-headers-%s", props.Environment)),
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

	// Cache policy optimized for SPA
	cachePolicy := awscloudfront.NewCachePolicy(stack, jsii.String("SPACachePolicy"), &awscloudfront.CachePolicyProps{
		CachePolicyName:            jsii.String(fmt.Sprintf("handloom-spa-cache-%s", props.Environment)),
		DefaultTtl:                 awscdk.Duration_Hours(jsii.Number(24)),
		MaxTtl:                     awscdk.Duration_Days(jsii.Number(365)),
		MinTtl:                     awscdk.Duration_Seconds(jsii.Number(0)),
		EnableAcceptEncodingGzip:   jsii.Bool(true),
		EnableAcceptEncodingBrotli: jsii.Bool(true),
	})

	// CloudFront distribution
	distributionProps := &awscloudfront.DistributionProps{
		DefaultBehavior: &awscloudfront.BehaviorOptions{
			Origin:                s3Origin,
			ViewerProtocolPolicy:  awscloudfront.ViewerProtocolPolicy_REDIRECT_TO_HTTPS,
			AllowedMethods:        awscloudfront.AllowedMethods_ALLOW_GET_HEAD_OPTIONS(),
			CachedMethods:         awscloudfront.CachedMethods_CACHE_GET_HEAD_OPTIONS(),
			CachePolicy:           cachePolicy,
			ResponseHeadersPolicy: responseHeadersPolicy,
			Compress:              jsii.Bool(true),
		},
		DefaultRootObject: jsii.String("index.html"),
		HttpVersion:       awscloudfront.HttpVersion_HTTP2_AND_3,
		PriceClass:        awscloudfront.PriceClass_PRICE_CLASS_200,
		Comment:           jsii.String(fmt.Sprintf("Handloom Admin Frontend - %s", props.Environment)),
		// SPA routing - return index.html for 404s
		ErrorResponses: &[]*awscloudfront.ErrorResponse{
			{
				HttpStatus:         jsii.Number(404),
				ResponseHttpStatus: jsii.Number(200),
				ResponsePagePath:   jsii.String("/index.html"),
				Ttl:                awscdk.Duration_Minutes(jsii.Number(5)),
			},
			{
				HttpStatus:         jsii.Number(403),
				ResponseHttpStatus: jsii.Number(200),
				ResponsePagePath:   jsii.String("/index.html"),
				Ttl:                awscdk.Duration_Minutes(jsii.Number(5)),
			},
		},
	}

	// Add custom domain if provided
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

	// Deploy website content
	awss3deployment.NewBucketDeployment(stack, jsii.String("DeployWebsite"), &awss3deployment.BucketDeploymentProps{
		Sources: &[]awss3deployment.ISource{
			awss3deployment.Source_Asset(jsii.String("../dist"), nil),
		},
		DestinationBucket: bucket,
		Distribution:      distribution,
		DistributionPaths: jsii.Strings("/*"),
		Prune:             jsii.Bool(true),
	})

	// Outputs
	awscdk.NewCfnOutput(stack, jsii.String("BucketName"), &awscdk.CfnOutputProps{
		Value:       bucket.BucketName(),
		Description: jsii.String("S3 bucket name"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("DistributionId"), &awscdk.CfnOutputProps{
		Value:       distribution.DistributionId(),
		Description: jsii.String("CloudFront distribution ID"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("DistributionDomainName"), &awscdk.CfnOutputProps{
		Value:       distribution.DistributionDomainName(),
		Description: jsii.String("CloudFront domain name"),
	})

	websiteURL := fmt.Sprintf("https://%s", *distribution.DistributionDomainName())
	awscdk.NewCfnOutput(stack, jsii.String("WebsiteURL"), &awscdk.CfnOutputProps{
		Value:       jsii.String(websiteURL),
		Description: jsii.String("Website URL (HTTPS)"),
	})

	return &FrontendStack{
		Stack:        stack,
		Bucket:       bucket,
		Distribution: distribution,
		WebsiteURL:   websiteURL,
	}
}

func getRemovalPolicy(isProd bool) awscdk.RemovalPolicy {
	if isProd {
		return awscdk.RemovalPolicy_RETAIN
	}
	return awscdk.RemovalPolicy_DESTROY
}
