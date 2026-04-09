package main

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
