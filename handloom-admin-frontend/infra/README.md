# Handloom Admin Frontend Infrastructure

This directory contains AWS CDK infrastructure code for deploying the Handloom Admin Frontend as a static website.

## Deployment Options

### Option 1: S3 Only (FREE - Default for Dev)
- **Cost**: FREE (within AWS Free Tier: 5GB storage, 20K GET requests/month)
- **Protocol**: HTTP only (no HTTPS)
- **Best for**: Development, testing

### Option 2: S3 + CloudFront (Default for Prod)
- **Cost**: ~$1-5/month for low traffic (Free tier: 1TB transfer, 10M requests for first 12 months)
- **Protocol**: HTTPS with automatic redirect
- **Best for**: Production, when HTTPS is required

## Architecture

**S3 Only Mode (Dev Default):**
- S3 Bucket with static website hosting enabled
- Public read access
- HTTP only

**CloudFront Mode (Prod Default):**
- S3 Bucket (private, no public access)
- CloudFront Distribution with HTTPS
- Origin Access Control (OAC) for secure S3 access
- Security headers (HSTS, X-Frame-Options, etc.)

## Prerequisites

1. **AWS CLI** configured with appropriate credentials
2. **AWS CDK** installed globally: `npm install -g aws-cdk`
3. **Go 1.21+** installed
4. **Node.js 18+** for building the frontend

## Setup

### 1. Bootstrap CDK (first time only, once per AWS account/region)

```bash
cd infra
cdk bootstrap
```

### 2. Update Environment Variables

Edit the environment files with your actual API Gateway URLs:

- `.env.dev` - Development API URL
- `.env.prod` - Production API URL

After deploying the backend, update these with the actual API Gateway URLs.

## Deployment Commands

### From Project Root (using npm scripts)

```bash
# Build and deploy to dev (S3 only - FREE, HTTP)
npm run cdk:deploy:dev

# Build and deploy to dev with CloudFront (HTTPS, costs money)
npm run cdk:deploy:dev:cdn

# Build and deploy to prod (always uses CloudFront)
npm run cdk:deploy:prod

# View changes before deployment
npm run cdk:diff:dev
npm run cdk:diff:prod

# Destroy stacks
npm run cdk:destroy:dev
npm run cdk:destroy:prod
```

### From Infra Directory (using CDK directly)

```bash
cd infra

# Synthesize CloudFormation templates
cdk synth -c environment=dev
cdk synth -c environment=prod

# Deploy
cdk deploy --all -c environment=dev
cdk deploy --all -c environment=prod

# Destroy
cdk destroy --all -c environment=dev
cdk destroy --all -c environment=prod
```

## Stack Outputs

After deployment, the following outputs are available:

- **BucketName**: S3 bucket name
- **DistributionId**: CloudFront distribution ID (for cache invalidation)
- **DistributionDomainName**: CloudFront domain (e.g., `d1234567890.cloudfront.net`)
- **WebsiteURL**: Full HTTPS URL to access the website

## Environment Configuration

| Environment | Mode | S3 Bucket | CloudFront | Protocol | Cost |
|-------------|------|-----------|------------|----------|------|
| dev | S3 Only (default) | handloom-admin-frontend-dev | None | HTTP | FREE |
| dev | With CDN | handloom-admin-frontend-dev | PRICE_CLASS_100 | HTTPS | ~$1/mo |
| prod | CloudFront (always) | handloom-admin-frontend-prod | PRICE_CLASS_ALL | HTTPS | ~$1-5/mo |

### Force CloudFront for Dev

If you need HTTPS in dev:
```bash
# Via npm script
npm run cdk:deploy:dev:cdn

# Or via CDK context
cdk deploy -c environment=dev -c useCDN=true
```

## Custom Domain (Optional)

To use a custom domain:

1. Create an ACM certificate in `us-east-1` region (required for CloudFront)
2. Pass the domain and certificate ARN via CDK context:

```bash
cdk deploy -c environment=prod \
  -c domainName=admin.handloom.com \
  -c certArn=arn:aws:acm:us-east-1:123456789:certificate/xxx
```

## Cache Invalidation

After deploying new content, CloudFront cache is automatically invalidated. For manual invalidation:

```bash
aws cloudfront create-invalidation \
  --distribution-id YOUR_DISTRIBUTION_ID \
  --paths "/*"
```

## Security Features

- S3 bucket is not publicly accessible (uses OAC)
- HTTPS only (HTTP redirects to HTTPS)
- Security headers (HSTS, X-Frame-Options, etc.)
- Brotli/Gzip compression enabled

## Cost Optimization

- Dev uses PRICE_CLASS_100 (US, Canada, Europe only) - cheapest
- Prod uses PRICE_CLASS_ALL for global availability
- S3 versioning only enabled in prod
- Auto-delete enabled in dev for easy cleanup
