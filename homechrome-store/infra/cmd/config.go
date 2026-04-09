package main

// EnvConfig holds all environment-specific configuration.
// Add new environments here.
type EnvConfig struct {
	CertArn       string // ACM certificate ARN (us-east-1 for CloudFront)
	DomainName    string // Custom domain for the storefront
	BackendApiUrl string // Backend API URL for Next.js rewrites
}

var envConfigs = map[string]EnvConfig{
	"dev": {
		CertArn:       "arn:aws:acm:us-east-1:163053486005:certificate/4e15c02f-e7ef-48df-8eff-5097eefed2e8",
		DomainName:    "dev-store.homechrome.in",
		BackendApiUrl: "https://dev-api.homechrome.in",
	},
	"prod": {
		CertArn:       "arn:aws:acm:us-east-1:163053486005:certificate/4e15c02f-e7ef-48df-8eff-5097eefed2e8",
		DomainName:    "homechrome.in",
		BackendApiUrl: "https://api.homechrome.in",
	},
}
