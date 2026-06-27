package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/handloom/admin-frontend/infra/stacks"
)

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)
	env := getAWSEnv()
	environment := getEnvironment(app)

	cfg, ok := envConfigs[environment]
	if !ok {
		panic(fmt.Sprintf("unknown environment: %s (valid: dev, prod)", environment))
	}
	if err := cfg.validate(environment); err != nil {
		panic(err)
	}
	// CERT_ARN override lets a fresh account inject its own us-east-1 ACM cert
	// without a code change (the baked ARN belongs to one account and passes
	// validate() but fails at CloudFront on any other account).
	if v := os.Getenv("CERT_ARN"); v != "" {
		cfg.CertArn = v
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
		APIURL:      cfg.APIURL,
		UseCDN:      cfg.UseCDN,
		DomainName:  cfg.DomainName,
		CertArn:     cfg.CertArn,
	})

	app.Synth(nil)
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
