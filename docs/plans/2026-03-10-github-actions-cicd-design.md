# GitHub Actions CI/CD with OIDC Federation

**Date**: 2026-03-10
**Status**: Approved

## Overview

Set up GitHub Actions CI/CD for the Homechrome monorepo with AWS OIDC federation (no static access keys). Three separate workflows for three projects, with auto-deploy to dev on push and manual dispatch for prod.

## OIDC Bootstrap

Standalone CloudFormation template (`infra/github-oidc.yml`):
- GitHub OIDC identity provider (`token.actions.githubusercontent.com`)
- IAM role `GitHubActionsDeployRole` with `AdministratorAccess`
- Trust policy scoped to repo `UtkarshNigam1221/Homechrome` (any branch)
- Deploy once via `aws cloudformation deploy`

## Workflows

### deploy-backend.yml
- **Path filter**: `handloom-admin/**`
- **PR**: `golangci-lint` + `make test`
- **Push to main**: auto-deploy dev
- **Manual dispatch**: choose branch + environment (dev/prod)
- **Steps**: OIDC assume role → Go 1.24 + Node 20 → source env secret → `make build-lambdas-active` → `cd infra && cdk deploy --all`
- **Secrets**: `BACKEND_ENV_DEV`, `BACKEND_ENV_PROD` (full .env file contents)

### deploy-admin-frontend.yml
- **Path filter**: `handloom-admin-frontend/**`
- **PR**: `npm run check` (typecheck + lint + format)
- **Push to main**: auto-deploy dev
- **Manual dispatch**: choose branch + environment (dev/prod)
- **Steps**: OIDC assume role → Node 20 → `npm ci` → `npm run build:{env}` → `cd infra && cdk deploy --all`

### deploy-store.yml
- **Path filter**: `homechrome-store/**`
- **PR**: `npm run lint`
- **Push to main**: auto-deploy dev
- **Manual dispatch**: choose branch + environment (dev/prod)
- **Steps**: OIDC assume role → Node 20 → `npm ci` → set `NEXT_PUBLIC_*` vars → `npx @opennextjs/aws build` → `cd infra && cdk deploy --all` → CloudFront invalidation

## Trigger Matrix

| Event | Backend | Admin FE | Store |
|-------|---------|----------|-------|
| PR (path-filtered) | lint + test | typecheck + lint + format | lint |
| Push to main (path-filtered) | deploy dev | deploy dev | deploy dev |
| Manual dispatch (any branch) | deploy dev or prod | deploy dev or prod | deploy dev or prod |

## GitHub Secrets & Variables

| Type | Name | Value |
|------|------|-------|
| Variable | `AWS_ROLE_ARN` | Output from OIDC CloudFormation stack |
| Variable | `AWS_REGION` | `ap-south-1` |
| Secret | `BACKEND_ENV_DEV` | Contents of `handloom-admin/.env.dev` |
| Secret | `BACKEND_ENV_PROD` | Contents of `handloom-admin/.env.prod` |

## One-Time Setup

1. Deploy `infra/github-oidc.yml` via AWS CLI
2. Copy role ARN from stack output
3. Add secrets/variables in GitHub repo settings

## Key Decisions

- **OIDC over static keys**: No long-lived credentials, temporary creds per run
- **AdministratorAccess**: Simple to start, tighten later
- **Three separate workflows**: Independent triggers, parallel runs, cleaner UI
- **CloudFormation for OIDC**: Avoids chicken-and-egg with CDK
- **Backend env via secrets**: POSTGRES_DSN and other sensitive config stored as GitHub secrets
- **Frontend env hardcoded**: NEXT_PUBLIC_* and VITE_* vars are not secret, set directly in workflows
