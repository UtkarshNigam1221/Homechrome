package main

import (
	"fmt"
	"sort"
	"strings"
)

// EnvConfig holds all environment-specific configuration.
// Add new environments here.
type EnvConfig struct {
	CertArn        string // ACM certificate ARN (ap-south-1 for regional API Gateway)
	BaseDomain     string
	DomainName     string // Custom domain for API Gateway (e.g. dev-api.homechrome.in)
	FrontendOrigin string // Frontend origin for CORS (e.g. https://dev-admin.homechrome.in)
	StoreFrontHost string // B2C storefront origin for embedder CORS (e.g. https://store-dev.homechrome.in)

	// Stable cross-stack resource names. Single source of truth so the backend
	// app (which imports the embedder by name + the metrics queue by name) and
	// the standalone embedder app agree byte-for-byte. Drift here breaks invoke
	// at runtime with NO synth error.
	MetricsQueueName string // SQS metrics queue (MetricsStack owns; Embedder + API import)
	EmbedderFnName   string // Embedder Lambda (Embedder app owns; API imports for invoke + route)

	// GrafanaEndpoint is the non-secret Grafana Cloud OTLP base URL. Baked here
	// (like the cert ARNs / collector layer ARN) — only the AUTH token is secret
	// and stays in SSM (/handloom/<env>/grafana-otlp-auth).
	GrafanaEndpoint string

	// Non-secret gateway config — baked per-env here so it never needs a
	// deploy-time env var. SECRETS (client secret, passwords, auth keys,
	// POSTGRES_DSN, JWT secrets) stay in the BACKEND_ENV_* deploy secret and are
	// read from the shell in infra/stacks/api.go.
	PhonePeBaseURL          string
	PhonePeCallbackURL      string
	PhonePeRedirectURL      string
	PhonePeClientVersion    string
	MSG91BaseURL            string
	MSG91OTPTemplateID      string
	ShiprocketBaseURL       string
	ShiprocketPickupPincode string
}

var envConfigs = map[string]EnvConfig{
	"dev": {
		CertArn:        "arn:aws:acm:ap-south-1:163053486005:certificate/c20f97ff-ba58-4821-8f3d-6f50f772df89",
		BaseDomain:     "homechrome.in",
		DomainName:     "dev-api.homechrome.in",
		FrontendOrigin: "https://dev-admin.homechrome.in",
		StoreFrontHost: "https://store-dev.homechrome.in",

		MetricsQueueName: "handloom-metrics-events-dev",
		EmbedderFnName:   "handloom-embedder-dev",
		GrafanaEndpoint:  "https://otlp-gateway-prod-ap-south-1.grafana.net/otlp",

		PhonePeBaseURL:          "https://api-preprod.phonepe.com/apis/pg-sandbox",
		PhonePeCallbackURL:      "https://dev-api.homechrome.in/api/v1/store/webhooks/phonepe",
		PhonePeRedirectURL:      "https://dev-store.homechrome.in/checkout/confirmation",
		PhonePeClientVersion:    "1",
		MSG91BaseURL:            "https://control.msg91.com",
		MSG91OTPTemplateID:      "6a04664c95bc5e4fa90fb332",
		ShiprocketBaseURL:       "", // unset in dev — gateway falls back to DevClient
		ShiprocketPickupPincode: "",
	},
	"prod": {
		CertArn:        "arn:aws:acm:ap-south-1:163053486005:certificate/c20f97ff-ba58-4821-8f3d-6f50f772df89",
		BaseDomain:     "homechrome.in",
		DomainName:     "api.homechrome.in",
		FrontendOrigin: "https://admin.homechrome.in",
		StoreFrontHost: "https://www.homechrome.in",

		MetricsQueueName: "handloom-metrics-events-prod",
		EmbedderFnName:   "handloom-embedder-prod",
		GrafanaEndpoint:  "https://otlp-gateway-prod-ap-south-1.grafana.net/otlp",

		// when this was filled — confirm against BACKEND_ENV_PROD before the next
		// prod deploy. The URLs below follow the prod domain pattern.
		PhonePeBaseURL:          "https://api-preprod.phonepe.com/apis/pg-sandbox", // live PhonePe — VERIFY
		PhonePeCallbackURL:      "https://api.homechrome.in/api/v1/store/webhooks/phonepe",
		PhonePeRedirectURL:      "https://www.homechrome.in/checkout/confirmation",
		PhonePeClientVersion:    "1",
		MSG91BaseURL:            "https://control.msg91.com",
		MSG91OTPTemplateID:      "6a04664c95bc5e4fa90fb332", // VERIFY prod template
		ShiprocketBaseURL:       "",
		ShiprocketPickupPincode: "",
	},
}

// validate fails the CDK synth if a required config field is empty, so a
// misconfigured env never deploys and surfaces as a runtime failure instead.
// Shiprocket* are intentionally optional (empty → gateway DevClient).
func (c EnvConfig) validate(env string) error {
	required := map[string]string{
		"CertArn":              c.CertArn,
		"BaseDomain":           c.BaseDomain,
		"DomainName":           c.DomainName,
		"FrontendOrigin":       c.FrontendOrigin,
		"StoreFrontHost":       c.StoreFrontHost,
		"PhonePeBaseURL":       c.PhonePeBaseURL,
		"PhonePeCallbackURL":   c.PhonePeCallbackURL,
		"PhonePeRedirectURL":   c.PhonePeRedirectURL,
		"PhonePeClientVersion": c.PhonePeClientVersion,
		"MSG91BaseURL":         c.MSG91BaseURL,
		"MSG91OTPTemplateID":   c.MSG91OTPTemplateID,
		"MetricsQueueName":     c.MetricsQueueName,
		"EmbedderFnName":       c.EmbedderFnName,
		"GrafanaEndpoint":      c.GrafanaEndpoint,
	}
	var missing []string
	for name, val := range required {
		if strings.TrimSpace(val) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("env %q config has empty required field(s): %s", env, strings.Join(missing, ", "))
	}
	return nil
}
