# Storefront Infrastructure Design: OpenNext + Go CDK

**Date:** 2026-02-28
**Status:** Approved

## Goal

Deploy homechrome-store (Next.js 16 SSR storefront) to AWS using OpenNext + Go CDK, matching the infrastructure pattern of handloom-admin-frontend.

## Decisions

- **Framework:** Keep Next.js (SSR needed for SEO, performance, ISR)
- **Deployment method:** OpenNext v3 compiles Next.js into Lambda-compatible artifacts
- **CDK language:** Go (consistent with handloom-admin-frontend and handloom-admin)
- **Certificate:** Reuse existing ACM wildcard cert (`arn:aws:acm:us-east-1:163053486005:certificate/a8dca48d-...`)

## Domains

| Environment | Domain |
|-------------|--------|
| dev | `dev-store.homechrome.lldlab.com` |
| prod | `homechrome.lldlab.com` |

## API URLs

| Environment | Backend API |
|-------------|-------------|
| dev | `https://dev-api.homechrome.lldlab.com` |
| prod | `https://api.homechrome.lldlab.com` |

## Architecture

```
CloudFront Distribution (PRICE_CLASS_100)
├── Behavior: /_next/static/*  → S3 Origin (immutable hashed assets, 1yr cache)
├── Behavior: /_next/image     → Image Optimization Lambda (via Function URL)
├── Behavior: /favicon.ico, /robots.txt, /sitemap*.xml → S3 Origin
└── Default Behavior: *        → Server Lambda (via Function URL)

S3 Bucket (private, OAC, DESTROY policy, no versioning)
├── _next/static/    (JS, CSS, media — content-hashed)
├── public assets    (favicon, robots, etc.)
└── cache/           (ISR cache seed)

Server Lambda (ARM64, 128MB, 15s timeout)
├── Handler: .open-next/server-function/
├── Env: CACHE_BUCKET_NAME, CACHE_BUCKET_REGION
└── Function URL (CloudFront origin)

Image Optimization Lambda (ARM64, 128MB, 15s timeout)
├── Handler: .open-next/image-optimization-function/
├── Env: BUCKET_NAME
└── Function URL (CloudFront origin)
```

## Cost Optimization

All settings optimized for cheapest deployment:

| Setting | Value |
|---------|-------|
| Lambda memory | 128MB (both functions) |
| Lambda timeout | 15s |
| CloudFront price class | PRICE_CLASS_100 |
| S3 versioning | Off |
| S3 removal policy | DESTROY (both environments) |
| Provisioned concurrency | None |

Estimated: ~$1-3/mo at low traffic.

## Build Pipeline

```
npm run cdk:deploy:dev
  ↓
(1) next build (with .env.dev)      → .next/
(2) npx open-next build             → .open-next/
(3) cd infra && cdk deploy
    ├── Upload .open-next/assets/ to S3
    ├── Deploy server Lambda
    ├── Deploy image optimization Lambda
    ├── Create/update CloudFront distribution
    └── Invalidate CloudFront cache
```

## New Files

```
homechrome-store/
├── infra/
│   ├── cmd/main.go           # CDK entry point
│   ├── stacks/storefront.go  # CloudFront + Lambda + S3 stack
│   ├── cdk.json
│   ├── go.mod
│   └── go.sum
├── .env.dev                   # NEXT_PUBLIC_API_URL=https://dev-api.homechrome.lldlab.com
├── .env.prod                  # NEXT_PUBLIC_API_URL=https://api.homechrome.lldlab.com
└── open-next.config.ts        # OpenNext configuration (if needed)
```

## Package.json Scripts (new)

```
build:dev       → next build with .env.dev
build:prod      → next build with .env.prod
open-next:build → npx open-next build
cdk:deploy:dev  → build:dev + open-next:build + cd infra && cdk deploy -c environment=dev -c certArn=...
cdk:deploy:prod → build:prod + open-next:build + cd infra && cdk deploy -c environment=prod -c certArn=...
cdk:destroy:dev → cd infra && cdk destroy -c environment=dev
cdk:destroy:prod→ cd infra && cdk destroy -c environment=prod
```

## Security

Same as admin frontend:
- S3 private with OAC (CloudFront-only access)
- HTTPS redirect
- HSTS (1 year)
- X-Frame-Options: DENY
- X-Content-Type-Options: nosniff
- Referrer-Policy: strict-origin-when-cross-origin

## DNS (manual, one-time)

After first deploy, create Route 53 records:
- `dev-store.homechrome.lldlab.com` → CloudFront distribution (CNAME/alias)
- `homechrome.lldlab.com` → CloudFront distribution (CNAME/alias)

## Key Differences from Admin Frontend

| Aspect | Admin Frontend | Store |
|--------|---------------|-------|
| Type | Static SPA (Vite) | SSR (Next.js + OpenNext) |
| CloudFront origins | 1 (S3) | 3 (S3 + Server Lambda + Image Lambda) |
| Lambda functions | 0 | 2 |
| Cache behaviors | 1 default | 4 |
| Error handling | 404→index.html (SPA) | Server handles all routes |
| Build tool | Vite | Next.js + OpenNext |
