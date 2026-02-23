// Package postgres provides PostgreSQL-backed repository implementations
// for catalog data (categories, products, inventory).
package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/jackc/pgx/v5/pgxpool"

	appconfig "github.com/handloom/admin/internal/config"
)

// NewPool creates a PostgreSQL connection pool.
// Uses POSTGRES_DSN directly if set (local dev), otherwise reads credentials
// from Secrets Manager (Lambda/production).
func NewPool(ctx context.Context, pgCfg *appconfig.PostgresConfig) (*pgxpool.Pool, error) {
	dsn := pgCfg.DSN

	// If no direct DSN, build one from Secrets Manager + RDS env vars
	if dsn == "" && pgCfg.SecretARN != "" {
		resolved, err := resolveDSNFromSecret(ctx, pgCfg)
		if err != nil {
			return nil, fmt.Errorf("resolve postgres DSN from secret: %w", err)
		}
		dsn = resolved
	}

	if dsn == "" {
		return nil, fmt.Errorf("no postgres DSN configured (set POSTGRES_DSN or RDS_SECRET_ARN)")
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres DSN: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

// resolveDSNFromSecret reads RDS credentials from Secrets Manager and builds a DSN.
func resolveDSNFromSecret(ctx context.Context, pgCfg *appconfig.PostgresConfig) (string, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("load AWS config: %w", err)
	}

	client := secretsmanager.NewFromConfig(awsCfg)
	result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &pgCfg.SecretARN,
	})
	if err != nil {
		return "", fmt.Errorf("get secret %s: %w", pgCfg.SecretARN, err)
	}

	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal([]byte(*result.SecretString), &creds); err != nil {
		return "", fmt.Errorf("parse secret JSON: %w", err)
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
		creds.Username, creds.Password, pgCfg.Endpoint, pgCfg.Port, pgCfg.DatabaseName), nil
}
