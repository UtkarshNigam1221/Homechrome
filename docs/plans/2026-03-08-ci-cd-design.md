# CI/CD Pipeline Design

**Goal:** Automated deployments for all 3 projects (backend, admin frontend, B2C store) with dev/prod stages using GitHub Actions and AWS OIDC.

## Architecture

3 separate GitHub Actions workflows, one per project. Each independently triggered by path-filtered changes.

### Trigger Model

| Trigger | Action |
|---------|--------|
| PR to main | Lint/typecheck/test checks (required status checks, blocks merge) |
| Push to main | Auto-deploy to dev (path-filtered per project) |
| Manual dispatch | Deploy any project to dev or prod (environment input) |

### Workflows

| Workflow | Path Filter | PR Checks | Deploy Command (dev) | Deploy Command (prod) |
|----------|-------------|-----------|---------------------|-----------------------|
| `deploy-backend.yml` | `handloom-admin/**` | `make test`, `golangci-lint run` | `make cdk-deploy-dev` | `make cdk-deploy-prod` |
| `deploy-admin-frontend.yml` | `handloom-admin-frontend/**` | `npm run check` | `npm run cdk:deploy:dev` | `npm run cdk:deploy:prod` |
| `deploy-store.yml` | `homechrome-store/**` | `npm run lint`, `npm run build` | `npm run cdk:deploy:dev` | `npm run cdk:deploy:prod` |

### AWS Authentication — OIDC

No long-lived AWS credentials stored in GitHub. GitHub assumes an IAM role via OpenID Connect.

**One-time setup required:**
1. IAM OIDC identity provider for `token.actions.githubusercontent.com`
2. IAM role `github-actions-deploy` with trust policy scoped to the repo
3. Role permissions: CloudFormation, S3, Lambda, API Gateway, DynamoDB, CloudFront, SNS/SQS, ACM, Route53, IAM:PassRole

### Workflow Structure

```
PR:
  [check] → lint, typecheck, test → pass/fail status check

Push to main:
  [deploy-dev] → configure AWS OIDC → build → cdk deploy (dev)

Manual dispatch (input: environment):
  [deploy] → configure AWS OIDC → build → cdk deploy (selected env)
```

### Secrets & Variables

**GitHub Actions Variables:**
- `AWS_ROLE_ARN` — OIDC role ARN
- `AWS_REGION` — `ap-south-1`

**GitHub Secrets:**
- `POSTGRES_DSN_DEV` / `POSTGRES_DSN_PROD` — passed as CDK context for backend

Runtime secrets (JWT keys, SMS/payment credentials) should live in AWS Secrets Manager or SSM Parameter Store, not GitHub.

### Excluded (YAGNI)

- No staging environment (2 stages only: dev + prod)
- No Slack/Discord notifications
- No rollback automation (CDK handles natively)
- No monorepo-level orchestration (each project independent)

## Domain Mapping

| Project | Dev | Prod |
|---------|-----|------|
| Backend API | `dev-api.homechrome.in` | `api.homechrome.in` |
| Admin Frontend | `dev-admin.homechrome.in` | `admin.homechrome.in` |
| B2C Store | `dev-store.homechrome.in` | `homechrome.in` |
