// Package awsmiddleware emits aws_sdk_call{} metrics for every AWS SDK v2 call.
package awsmiddleware

import (
	"context"

	smithymiddleware "github.com/aws/smithy-go/middleware"

	pkgmetrics "github.com/handloom/admin/pkg/metrics"
)

// With returns an APIOptions function that appends a metric-emitting
// middleware to the smithy stack. Usage example:
//
//	sqs.NewFromConfig(awsCfg, func(o *sqs.Options) {
//	    o.APIOptions = append(o.APIOptions, awsmiddleware.With(serviceName))
//	})
//
// service is the caller's logical service (e.g. "handloom-store-checkout");
// sdk_service + operation come from the smithy context.
func With(service string) func(*smithymiddleware.Stack) error {
	return func(stack *smithymiddleware.Stack) error {
		return stack.Initialize.Add(
			smithymiddleware.InitializeMiddlewareFunc("MetricsMiddleware",
				func(ctx context.Context, in smithymiddleware.InitializeInput, next smithymiddleware.InitializeHandler) (smithymiddleware.InitializeOutput, smithymiddleware.Metadata, error) {
					out, md, err := next.HandleInitialize(ctx, in)

					status := "ok"
					if err != nil {
						status = "err"
					}
					pkgmetrics.Record(ctx, "aws_sdk_call", pkgmetrics.L{
						"service":     service,
						"sdk_service": smithymiddleware.GetServiceID(ctx),
						"operation":   smithymiddleware.GetOperationName(ctx),
						"status":      status,
					})
					return out, md, err
				}),
			smithymiddleware.After,
		)
	}
}
