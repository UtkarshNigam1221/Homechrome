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
	"github.com/go-chi/chi/v5"
	chiadapter "github.com/awslabs/aws-lambda-go-api-proxy/chi"

	emb "github.com/handloom/admin/cmd/embedder/embedder"
)

func main() {
	ctx := context.Background()
	cfg := loadConfig(ctx)

	pool, err := emb.NewPGPool(ctx, cfg.PostgresDSN, 10)
	must(err, "pg pool")

	onnx, err := emb.NewONNXSession(cfg.ModelPath, cfg.ORTLibPath, cfg.TokenizerPath, 128)
	must(err, "onnx session")
	// Warmup pass — first inference is slow due to lazy graph init.
	_, _ = onnx.Embed([]string{"warmup"})

	d := &deps{
		onnx:        onnx,
		searcher:    emb.NewSearcher(pool, cfg.Weights),
		hmac:        emb.NewHMACVerifier([]byte(cfg.AuthKey)),
		rl:          emb.NewIPRateLimiter(cfg.RatePerMin),
		allowOrigin: cfg.AllowedOrigin,
		startedAt:   time.Now(),
	}

	// newRouter returns http.Handler; the underlying value is always *chi.Mux.
	mux := newRouter(d).(*chi.Mux)
	adapter := chiadapter.NewV2(mux)

	lambda.Start(func(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		return adapter.ProxyWithContextV2(ctx, req)
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
	ssmc := ssm.NewFromConfig(awsCfg)

	authKey := ssmGetSecure(ctx, ssmc, getEnv("EMBEDDER_AUTH_KEY_PARAM"))

	// POSTGRES_DSN is passed as a plain env var by CDK (sourced from
	// .env.{env} locally or secrets.BACKEND_ENV_{ENV} in GH Actions). Only
	// the HMAC auth key lives in SSM SecureString.
	pgDSN := getEnv("POSTGRES_DSN")

	return appConfig{
		PostgresDSN:   pgDSN,
		ModelPath:     getEnv("MODEL_PATH"),
		TokenizerPath: getEnv("TOKENIZER_PATH"),
		ORTLibPath:    getEnv("ONNXRUNTIME_SHARED_LIB_PATH"),
		AllowedOrigin: getEnv("ALLOWED_ORIGIN"),
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

func getEnv(k string) string { return os.Getenv(k) }

func parseFloatEnv(key string, def float64) float64 {
	if v := getEnv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

// parseIntEnv reads an integer env var with a fallback.
func parseIntEnv(key string, def int) int {
	if v := getEnv(key); v != "" {
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
