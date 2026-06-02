// Package lambdaclient provides a thin wrapper around aws-sdk-go-v2 Lambda
// for in-process Lambda invocations — both synchronous (RequestResponse) and
// asynchronous (Event) modes.
package lambdaclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

// LambdaClient wraps the AWS Lambda client.
type LambdaClient struct {
	client *lambda.Client
}

// New creates a LambdaClient. endpoint != "" → local dev (LocalStack).
func New(ctx context.Context, region string, endpoint string) (*LambdaClient, error) {
	var cfg aws.Config
	var err error

	if endpoint != "" {
		cfg, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(region),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				"local", "local", "",
			)),
		)
	} else {
		cfg, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(region),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	var optFns []func(*lambda.Options)
	if endpoint != "" {
		optFns = append(optFns, func(o *lambda.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}
	return &LambdaClient{client: lambda.NewFromConfig(cfg, optFns...)}, nil
}

// InvokeSync calls a Lambda function with RequestResponse and returns an
// error if the invocation, the function, or the payload signal failure.
func (c *LambdaClient) InvokeSync(ctx context.Context, functionName string, payload []byte) ([]byte, error) {
	out, err := c.client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(functionName),
		InvocationType: types.InvocationTypeRequestResponse,
		Payload:        payload,
	})
	if err != nil {
		return nil, fmt.Errorf("invoke %s: %w", functionName, err)
	}
	if out.FunctionError != nil && *out.FunctionError != "" {
		var body struct {
			ErrorMessage string `json:"errorMessage"`
			ErrorType    string `json:"errorType"`
		}
		_ = json.Unmarshal(out.Payload, &body)
		return out.Payload, fmt.Errorf("lambda %s returned function error: %s (%s)",
			functionName, body.ErrorMessage, body.ErrorType)
	}
	if out.StatusCode < 200 || out.StatusCode >= 300 {
		return out.Payload, errors.New("lambda returned non-2xx status")
	}
	return out.Payload, nil
}

// InvokeAsync calls a Lambda function with InvocationType=Event (fire-and-forget).
// Returns once Lambda accepts the request for queued execution; the callee runs
// independently against its own configured timeout.
//
// Note: AWS retries failed async invocations up to MaximumRetryAttempts (default
// 2, up to MaximumEventAgeInSeconds — 6h by default) then drops the event.
// Targets MUST be idempotent, and callers should configure a DLQ or OnFailure
// destination on the callee Lambda for failure visibility.
func (c *LambdaClient) InvokeAsync(ctx context.Context, functionName string, payload []byte) error {
	out, err := c.client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName:   aws.String(functionName),
		InvocationType: types.InvocationTypeEvent,
		Payload:        payload,
	})
	if err != nil {
		return fmt.Errorf("invoke %s: %w", functionName, err)
	}
	if out == nil || out.StatusCode < 200 || out.StatusCode >= 300 {
		return fmt.Errorf("lambda %s async invoke returned non-2xx status", functionName)
	}
	return nil
}
