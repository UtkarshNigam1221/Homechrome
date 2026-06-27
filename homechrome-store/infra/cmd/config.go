package main

import (
	"fmt"
	"sort"
	"strings"
)

// EnvConfig holds all environment-specific configuration.
// Add new environments here.
type EnvConfig struct {
	CertArn       string   // ACM certificate ARN (us-east-1 for CloudFront)
	DomainNames   []string // Custom domains for the storefront (primary first, then aliases)
	BackendApiUrl string   // Backend API URL for Next.js rewrites

	// Telemetry (non-secret). Baked per-env so deploys don't rely on env vars.
	// Leave the layer ARNs empty to skip OTel — StorefrontStack treats an empty
	// CollectorLayerArn as a no-op. To enable, set the community OTel Collector
	// layer ARN (account 184161586896, see handloom-admin telemetry.go) here.
	CollectorLayerArn       string
	NodeAutoInstrLayerArn   string
	GrafanaEndpointSSMParam string
	GrafanaAuthSSMParam     string
}

var envConfigs = map[string]EnvConfig{
	"dev": {
		CertArn:       "arn:aws:acm:us-east-1:163053486005:certificate/4e15c02f-e7ef-48df-8eff-5097eefed2e8",
		DomainNames:   []string{"dev-store.homechrome.in"},
		BackendApiUrl: "https://dev-api.homechrome.in",

		CollectorLayerArn:       "", // set to enable storefront OTel
		NodeAutoInstrLayerArn:   "",
		GrafanaEndpointSSMParam: "/handloom/dev/grafana-otlp-endpoint",
		GrafanaAuthSSMParam:     "/handloom/dev/grafana-otlp-auth",
	},
	"prod": {
		CertArn:       "arn:aws:acm:us-east-1:163053486005:certificate/4e15c02f-e7ef-48df-8eff-5097eefed2e8",
		DomainNames:   []string{"homechrome.in", "www.homechrome.in"},
		BackendApiUrl: "https://api.homechrome.in",

		CollectorLayerArn:       "", // set to enable storefront OTel
		NodeAutoInstrLayerArn:   "",
		GrafanaEndpointSSMParam: "/handloom/prod/grafana-otlp-endpoint",
		GrafanaAuthSSMParam:     "/handloom/prod/grafana-otlp-auth",
	},
}

// validate fails the CDK synth if a required config field is empty. Telemetry
// fields are intentionally optional (empty CollectorLayerArn → OTel skipped).
func (c EnvConfig) validate(env string) error {
	required := map[string]string{
		"CertArn":       c.CertArn,
		"BackendApiUrl": c.BackendApiUrl,
	}
	var missing []string
	for name, val := range required {
		if strings.TrimSpace(val) == "" {
			missing = append(missing, name)
		}
	}
	if len(c.DomainNames) == 0 {
		missing = append(missing, "DomainNames")
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("env %q config has empty required field(s): %s", env, strings.Join(missing, ", "))
	}
	return nil
}
