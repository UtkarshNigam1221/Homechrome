package main

import (
	"fmt"
	"sort"
	"strings"
)

// EnvConfig holds all environment-specific configuration.
// Add new environments here.
type EnvConfig struct {
	CertArn    string // ACM certificate ARN (us-east-1 for CloudFront)
	DomainName string // Custom domain for the admin frontend
	APIURL     string // Backend API URL baked into the frontend build
	UseCDN     bool
}

var envConfigs = map[string]EnvConfig{
	"dev": {
		CertArn:    "arn:aws:acm:us-east-1:163053486005:certificate/4e15c02f-e7ef-48df-8eff-5097eefed2e8",
		DomainName: "dev-admin.homechrome.in",
		APIURL:     "https://dev-api.homechrome.in",
		UseCDN:     true,
	},
	"prod": {
		CertArn:    "arn:aws:acm:us-east-1:163053486005:certificate/4e15c02f-e7ef-48df-8eff-5097eefed2e8",
		DomainName: "admin.homechrome.in",
		APIURL:     "https://api.homechrome.in",
		UseCDN:     true,
	},
}

// validate fails the CDK synth if a required config field is empty.
func (c EnvConfig) validate(env string) error {
	required := map[string]string{
		"CertArn":    c.CertArn,
		"DomainName": c.DomainName,
		"APIURL":     c.APIURL,
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
