package main

import (
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/handloom/admin-frontend/infra/stacks"
)

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	// Get deployment mode: "single" deploys one env, "all" deploys both
	deployMode := getDeployMode(app)

	// Get AWS environment configuration
	env := getAWSEnv()

	if deployMode == "all" {
		// Deploy both dev and prod environments
		createEnvironmentStack(app, "dev", env)
		createEnvironmentStack(app, "prod", env)
	} else {
		// Deploy single environment (default: dev)
		environment := getEnvironment(app)
		createEnvironmentStack(app, environment, env)
	}

	app.Synth(nil)
}

// createEnvironmentStack creates the frontend hosting stack for a given environment
func createEnvironmentStack(app awscdk.App, environment string, env *awscdk.Environment) {
	// Get the backend API URL from context or use default
	apiURL := getAPIURL(app, environment)

	// Check if CDN should be used (default: false for dev, true for prod)
	useCDN := getUseCDN(app, environment)

	// Custom domain config from CDK context
	certArn := getCertArn(app)
	baseDomain := getBaseDomain(app)
	var domainName string
	if certArn != "" {
		// Custom domain requires CloudFront
		useCDN = true
		switch environment {
		case "prod":
			domainName = "admin." + baseDomain
		default:
			domainName = "dev-admin." + baseDomain
		}
	}

	stacks.NewFrontendStack(app, "HandloomFrontendStack-"+environment, &stacks.FrontendStackProps{
		StackProps: awscdk.StackProps{
			Env:         env,
			Description: jsii.String("Handloom Admin Frontend - Static website hosting (" + environment + ")"),
			Tags: &map[string]*string{
				"Environment": jsii.String(environment),
				"Project":     jsii.String("handloom-admin-frontend"),
				"ManagedBy":   jsii.String("cdk"),
			},
		},
		Environment: environment,
		APIURL:      apiURL,
		UseCDN:      useCDN,
		DomainName:  domainName,
		CertArn:     certArn,
	})
}

func getDeployMode(app constructs.Construct) string {
	if mode := app.Node().TryGetContext(jsii.String("deployMode")); mode != nil {
		return mode.(string)
	}
	if mode := os.Getenv("CDK_DEPLOY_MODE"); mode != "" {
		return mode
	}
	return "single"
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

func getAPIURL(app constructs.Construct, environment string) string {
	// Try to get from CDK context
	contextKey := "apiUrl-" + environment
	if url := app.Node().TryGetContext(jsii.String(contextKey)); url != nil {
		return url.(string)
	}

	// Try generic apiUrl context
	if url := app.Node().TryGetContext(jsii.String("apiUrl")); url != nil {
		return url.(string)
	}

	// Try environment variable
	if url := os.Getenv("API_URL"); url != "" {
		return url
	}

	// Default placeholder - will be replaced after backend deployment
	if environment == "prod" {
		return "https://api.handloom.com"
	}
	return "https://api-dev.handloom.com"
}

func getUseCDN(app constructs.Construct, environment string) bool {
	// Check CDK context for useCDN flag
	if useCDN := app.Node().TryGetContext(jsii.String("useCDN")); useCDN != nil {
		return useCDN.(bool)
	}

	// Check environment variable
	if useCDN := os.Getenv("USE_CDN"); useCDN == "true" {
		return true
	}

	// Default: CDN for prod only (to save costs in dev)
	// Dev uses S3 static hosting (FREE, but HTTP only)
	// Prod always uses CloudFront (HTTPS required)
	return environment == "prod"
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

func getBaseDomain(app constructs.Construct) string {
	if d := app.Node().TryGetContext(jsii.String("baseDomain")); d != nil {
		return d.(string)
	}
	if d := os.Getenv("BASE_DOMAIN"); d != "" {
		return d
	}
	return "homechrome.in"
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
	if region == "" {
		region = "ap-south-1"
	}

	if account == "" {
		return nil
	}

	return &awscdk.Environment{
		Account: jsii.String(account),
		Region:  jsii.String(region),
	}
}
