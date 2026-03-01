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
			domainName = "homechrome.lldlab.com"
		default:
			domainName = "dev-store.homechrome.lldlab.com"
		}
	}

	var backendApiUrl string
	switch environment {
	case "prod":
		backendApiUrl = "https://api.homechrome.lldlab.com"
	default:
		backendApiUrl = "https://dev-api.homechrome.lldlab.com"
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
		Environment:   environment,
		DomainName:    domainName,
		CertArn:       certArn,
		BackendApiUrl: backendApiUrl,
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
