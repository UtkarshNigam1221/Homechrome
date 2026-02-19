package router

import (
	"context"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/go-chi/chi/v5"
)

// LambdaAdapter wraps a chi router for Lambda execution
// Uses V1 adapter for API Gateway REST API
type LambdaAdapter struct {
	router  *chi.Mux
	adapter *httpadapter.HandlerAdapter
}

// NewLambdaAdapter creates a new Lambda adapter for a chi router
// Uses V1 format for API Gateway REST API proxy integration
func NewLambdaAdapter(router *chi.Mux) *LambdaAdapter {
	return &LambdaAdapter{
		router:  router,
		adapter: httpadapter.New(router),
	}
}

// Handler is the Lambda entry point for API Gateway REST API (V1 format)
func (l *LambdaAdapter) Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return l.adapter.ProxyWithContext(ctx, req)
}

// Start starts the Lambda handler
func (l *LambdaAdapter) Start() {
	lambda.Start(l.Handler)
}

// Router returns the underlying chi router (for testing)
func (l *LambdaAdapter) Router() *chi.Mux {
	return l.router
}

// ServeHTTP implements http.Handler for local testing
func (l *LambdaAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l.router.ServeHTTP(w, r)
}
