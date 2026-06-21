package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"

	emb "github.com/handloom/admin/cmd/embedder/embedder"
	"github.com/handloom/admin/internal/repository/postgres"
	"github.com/handloom/admin/pkg/metrics/awsmiddleware"
)

func main() {
	ctx := context.Background()
	cfg := loadConfig(ctx)

	// SQS metrics publisher (shared wiring with bootstrap.InitLambda). No-op
	// when METRICS_QUEUE_URL is unset — metrics.Record() then falls through to
	// the global Noop publisher.
	awsmiddleware.InitSQSMetricsPublisher(ctx, embedderServiceName())

	pool, err := emb.NewPGPool(ctx, cfg.PostgresDSN, 10)
	must(err, "pg pool")

	onnx, err := emb.NewONNXSession(cfg.ModelPath, cfg.ORTLibPath, cfg.TokenizerPath, 128)
	must(err, "onnx session")
	// Warmup pass — first inference is slow due to lazy graph init.
	_, _ = onnx.Embed([]string{"warmup"})

	// Intent classifier — embeds prototype strings once at cold start. The
	// 0.45 threshold is the minimum cosine similarity below which a
	// prototype isn't considered a hit; tune from the Products dashboard
	// once real queries surface the distribution.
	classifier, err := emb.NewClassifier(onnx, 0.45)
	must(err, "intent classifier")

	productRepo := postgres.NewProductRepository(pool)

	d := &deps{
		onnx:        onnx,
		searcher:    emb.NewSearcher(pool, productRepo, cfg.Weights),
		classifier:  classifier,
		hmac:        emb.NewHMACVerifier([]byte(cfg.AuthKey)),
		rl:          emb.NewIPRateLimiter(cfg.RatePerMin),
		allowOrigin: cfg.AllowedOrigin,
		startedAt:   time.Now(),
	}

	// API Gateway REST API (v1) — same adapter the other Lambdas use.
	adapter := httpadapter.New(newRouter(d))

	lambda.Start(func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		return adapter.ProxyWithContext(ctx, req)
	})
}

type appConfig struct {
	PostgresDSN   string
	ModelPath     string
	TokenizerPath string
	ORTLibPath    string
	AllowedOrigin string
	AuthKey       string
	RatePerMin    int
	Weights       emb.Weights
}

func loadConfig(ctx context.Context) appConfig {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	must(err, "load aws config")
	otelaws.AppendMiddlewares(&awsCfg.APIOptions)
	awsCfg.APIOptions = append(awsCfg.APIOptions, awsmiddleware.With(embedderServiceName()))
	ssmc := ssm.NewFromConfig(awsCfg)

	authKey := ssmGetSecure(ctx, ssmc, os.Getenv("EMBEDDER_AUTH_KEY_PARAM"))

	// POSTGRES_DSN is passed as a plain env var by CDK (sourced from
	// .env.{env} locally or secrets.BACKEND_ENV_{ENV} in GH Actions). Only
	// the HMAC auth key lives in SSM SecureString.
	pgDSN := os.Getenv("POSTGRES_DSN")

	return appConfig{
		PostgresDSN:   pgDSN,
		ModelPath:     os.Getenv("MODEL_PATH"),
		TokenizerPath: os.Getenv("TOKENIZER_PATH"),
		ORTLibPath:    os.Getenv("ONNXRUNTIME_SHARED_LIB_PATH"),
		AllowedOrigin: os.Getenv("ALLOWED_ORIGIN"),
		AuthKey:       authKey,
		RatePerMin:    parseIntEnv("RATE_LIMIT_PER_IP_PER_MIN", 60),
		Weights: emb.Weights{
			Semantic: parseFloatEnv("SEARCH_WEIGHT_SEMANTIC", 0.60),
			Keyword:  parseFloatEnv("SEARCH_WEIGHT_KEYWORD", 0.30),
			Trigram:  parseFloatEnv("SEARCH_WEIGHT_TRIGRAM", 0.10),
		},
	}
}

func ssmGetSecure(ctx context.Context, c *ssm.Client, name string) string {
	if name == "" {
		return ""
	}
	out, err := c.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(true),
	})
	must(err, "ssm get "+name)
	return aws.ToString(out.Parameter.Value)
}

// embedderServiceName resolves the embedder's OTEL service name, used to label
// both the SSM/SQS aws_sdk_call metrics and the metrics publisher.
func embedderServiceName() string {
	if s := os.Getenv("OTEL_SERVICE_NAME"); s != "" {
		return s
	}
	return "handloom-embedder"
}

func parseFloatEnv(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

// parseIntEnv reads an integer env var with a fallback.
func parseIntEnv(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func must(err error, what string) {
	if err != nil {
		slog.Error("startup failure", "what", what, "err", err)
		os.Exit(1)
	}
}
