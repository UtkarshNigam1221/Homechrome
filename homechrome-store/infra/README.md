# Homechrome Store Infrastructure

AWS CDK (Go) infrastructure for deploying the Next.js SSR storefront using OpenNext.

## Architecture

```
CloudFront Distribution
├── /_next/static/*  → S3 Origin (immutable hashed JS/CSS, 1yr cache)
├── /_next/image      → Image Optimization Lambda (via Function URL)
└── * (default)       → Server Lambda (via Function URL, streaming)

S3 Bucket (private, OAC-only access)
├── _assets/         → Static assets (JS, CSS, media)
└── _cache/          → ISR cache seed

Server Lambda (Node.js 20, ARM64, 128MB, 15s timeout)
├── Handles SSR, API routes, ISR revalidation
├── Reads/writes ISR cache to S3 (_cache/ prefix)
└── Function URL with response streaming

Image Lambda (Node.js 20, ARM64, 256MB, 15s timeout)
├── On-demand image optimization via Sharp
└── Reads original images from S3 (_assets/ prefix)
```

## How It Works

[OpenNext](https://opennext.js.org/) compiles the Next.js build output (`.next/`) into Lambda-compatible artifacts (`.open-next/`):

1. **`next build`** produces `.next/` (standard Next.js output)
2. **`npx @opennextjs/aws build`** transforms `.next/` into `.open-next/`:
   - `server-functions/default/` → Server Lambda code
   - `image-optimization-function/` → Image Lambda code
   - `assets/` → Static files for S3
   - `cache/` → ISR cache seed for S3
3. **`cdk deploy`** provisions AWS resources and uploads artifacts

## Deploy Commands

```bash
# Build + deploy to dev
npm run cdk:deploy:dev

# Build + deploy to prod
npm run cdk:deploy:prod

# Synth only (validate CloudFormation without deploying)
npm run cdk:synth:dev
npm run cdk:synth:prod

# Tear down
npm run cdk:destroy:dev
npm run cdk:destroy:prod
```

Each deploy command runs the full pipeline: `next build` → `open-next build` → `cdk deploy`.

## Environments

| Setting | Dev | Prod |
|---------|-----|------|
| Domain | `dev-store.homechrome.in` | `homechrome.in` |
| Backend API | `https://dev-api.homechrome.in` | `https://api.homechrome.in` |
| Stack name | `HomechromeStoreStack-dev` | `HomechromeStoreStack-prod` |
| S3 bucket | `homechrome-store-dev` | `homechrome-store-prod` |

## CDK Context Parameters

Passed via `-c key=value` on the CLI:

| Key | Required | Description |
|-----|----------|-------------|
| `environment` | No | `dev` (default) or `prod` |
| `certArn` | No | ACM certificate ARN for custom domain (us-east-1) |

Falls back to `CDK_ENVIRONMENT` and `ACM_CERT_ARN` env vars respectively.

## CloudFront Behaviors

| Pattern | Origin | Cache | Methods | Notes |
|---------|--------|-------|---------|-------|
| `*` (default) | Server Lambda | Server cache policy (TTL=0, respects Cache-Control) | All (GET/POST/PUT/DELETE) | SSR pages, API routes, ISR |
| `_next/static/*` | S3 via OAC | CACHING_OPTIMIZED (immutable) | GET, HEAD | Content-hashed, long-lived |
| `_next/image` | Image Lambda | Server cache policy | GET, HEAD, OPTIONS | On-demand image resizing |

### Server Cache Policy

- **Default TTL**: 0 (CloudFront respects `Cache-Control` from the Lambda)
- **Max TTL**: 365 days
- **Compression**: gzip + brotli
- **Query strings**: All forwarded and used as cache key
- **Headers in cache key**: `accept`, `rsc`, `next-router-prefetch`, `next-router-state-tree`, `next-url`, `x-prerender-revalidate`
- **Cookies**: Forwarded to origin (via origin request policy) but NOT in cache key

### CloudFront Function

A viewer-request function injects `x-forwarded-host` from the original `Host` header. CloudFront replaces `Host` when forwarding to origins, but Next.js needs the original host for canonical URLs, redirects, and sitemap generation.

## Security Headers

Applied to all responses via a CloudFront response headers policy:

- `Strict-Transport-Security`: max-age=31536000; includeSubDomains
- `X-Frame-Options`: DENY
- `X-Content-Type-Options`: nosniff
- `Referrer-Policy`: strict-origin-when-cross-origin
- `X-XSS-Protection`: 1; mode=block

## OpenNext Configuration

`open-next.config.ts` at the project root:

- **`queue: "direct"`** — ISR revalidation runs in-process instead of via SQS (no extra cost)
- **`disableTagCache: true`** — Disables DynamoDB-backed tag cache for on-demand revalidation (saves DynamoDB cost; time-based ISR still works via S3 cache)

## Cost

Optimized for minimal spend at low traffic (~$1-3/month):

| Resource | Setting |
|----------|---------|
| Server Lambda | 128MB, ARM64 |
| Image Lambda | 256MB, ARM64 (minimum for Sharp) |
| CloudFront | PRICE_CLASS_100 (NA + Europe only) |
| S3 | No versioning, DESTROY removal policy |
| DynamoDB | None (tag cache disabled) |
| SQS | None (direct revalidation) |
| Provisioned concurrency | None |

## File Structure

```
infra/
├── cmd/
│   └── main.go                # CDK app entry point (environment + domain resolution)
├── stacks/
│   └── storefront.go          # CloudFront + S3 + Lambda stack
├── cdk.json                   # CDK config
├── go.mod                     # Go module (aws-cdk-go v2)
└── go.sum
```

## DNS Setup (One-Time, Manual)

After first deploy, create CNAME or alias records in Route 53:

```
dev-store.homechrome.in → <CloudFront distribution domain>
homechrome.in           → <CloudFront distribution domain>
```

The CloudFront distribution domain is printed in the `DistributionDomainName` stack output after deploy.

## Differences from Admin Frontend Infra

| Aspect | Admin Frontend | Store |
|--------|---------------|-------|
| Type | Static SPA (Vite) | SSR (Next.js + OpenNext) |
| CloudFront origins | 1 (S3) | 3 (S3 + Server Lambda + Image Lambda) |
| Lambda functions | 0 | 2 |
| Cache behaviors | 1 default | 3 |
| Error handling | 404 → index.html (SPA routing) | Server Lambda handles all routes |
| Build tool | Vite | Next.js + OpenNext |
