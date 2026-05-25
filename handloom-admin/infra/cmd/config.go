package main

// EnvConfig holds all environment-specific configuration.
// Add new environments here.
type EnvConfig struct {
	CertArn        string // ACM certificate ARN (ap-south-1 for regional API Gateway)
	BaseDomain     string
	DomainName     string // Custom domain for API Gateway (e.g. dev-api.homechrome.in)
	FrontendOrigin string // Frontend origin for CORS (e.g. https://dev-admin.homechrome.in)
	StoreFrontHost string // B2C storefront origin for embedder CORS (e.g. https://store-dev.homechrome.in)
}

var envConfigs = map[string]EnvConfig{
	"dev": {
		CertArn:        "arn:aws:acm:ap-south-1:163053486005:certificate/c20f97ff-ba58-4821-8f3d-6f50f772df89",
		BaseDomain:     "homechrome.in",
		DomainName:     "dev-api.homechrome.in",
		FrontendOrigin: "https://dev-admin.homechrome.in",
		StoreFrontHost: "https://store-dev.homechrome.in",
	},
	"prod": {
		CertArn:        "arn:aws:acm:ap-south-1:163053486005:certificate/c20f97ff-ba58-4821-8f3d-6f50f772df89",
		BaseDomain:     "homechrome.in",
		DomainName:     "api.homechrome.in",
		FrontendOrigin: "https://admin.homechrome.in",
		StoreFrontHost: "https://www.homechrome.in",
	},
}
