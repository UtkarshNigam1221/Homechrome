// Package lambdainvoker provides a thin AWS Lambda async-invocation client.
//
// Admin handlers that need to delegate long-running work to a scheduled
// Lambda (e.g. the rate-refresh button → cron-rate-refresh Lambda) call
// Invoke to fire-and-forget an Event-type invocation. Errors surface to
// the caller so the HTTP handler can choose a sync fallback.
package lambdainvoker

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awslambda "github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

// LambdaInvoker invokes a single configured Lambda function asynchronously.
type LambdaInvoker struct {
	client       *awslambda.Client
	functionName string
}

// NewLambdaInvoker returns an invoker bound to the given function name.
func NewLambdaInvoker(client *awslambda.Client, functionName string) *LambdaInvoker {
	return &LambdaInvoker{client: client, functionName: functionName}
}

// Invoke triggers an asynchronous Lambda execution.
//
// Despite the "fire-and-forget" execution semantics on the receiving end
// (InvocationType=Event), this method still performs a synchronous HTTPS
// call to the Lambda invoke API and blocks until that call returns a 202.
// Slow Lambda invoke responses (cold dependencies, throttling) will delay
// the caller. Use ctx.WithTimeout for hard upper bounds.
func (i *LambdaInvoker) Invoke(ctx context.Context) error {
	if i.functionName == "" {
		return fmt.Errorf("lambda invoker: function name not set")
	}
	_, err := i.client.Invoke(ctx, &awslambda.InvokeInput{
		FunctionName:   aws.String(i.functionName),
		InvocationType: types.InvocationTypeEvent,
	})
	if err != nil {
		return fmt.Errorf("lambda invoke %s: %w", i.functionName, err)
	}
	return nil
}
