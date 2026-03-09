# GitHub Actions CI/CD Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Set up GitHub Actions CI/CD with OIDC federation for all three Homechrome projects — auto-deploy dev on push, manual dispatch for prod.

**Architecture:** CloudFormation bootstraps OIDC provider + IAM role. Three independent workflow files, one per project. Each workflow uses `aws-actions/configure-aws-credentials@v4` to assume the OIDC role, then runs the same build/deploy commands used locally.

**Tech Stack:** GitHub Actions, AWS CloudFormation, AWS IAM OIDC, AWS CDK (Go)

---

### Task 1: Create CloudFormation OIDC Template

**Files:**
- Create: `infra/github-oidc.yml`

**Step 1: Create the CloudFormation template**

```yaml
AWSTemplateFormatVersion: '2010-09-09'
Description: GitHub Actions OIDC provider and IAM role for Homechrome deployments

Parameters:
  GitHubOrg:
    Type: String
    Default: UtkarshNigam1221
  GitHubRepo:
    Type: String
    Default: Homechrome

Resources:
  GitHubOIDCProvider:
    Type: AWS::IAM::OIDCProvider
    Properties:
      Url: https://token.actions.githubusercontent.com
      ClientIdList:
        - sts.amazonaws.com
      ThumbprintList:
        - 6938fd4d98bab03faadb97b34396831e3780aea1
        - 1c58a3a8518e8759bf075b76b750d4f2df264fcd

  GitHubActionsDeployRole:
    Type: AWS::IAM::Role
    Properties:
      RoleName: GitHubActionsDeployRole
      AssumeRolePolicyDocument:
        Version: '2012-10-17'
        Statement:
          - Effect: Allow
            Principal:
              Federated: !Ref GitHubOIDCProvider
            Action: sts:AssumeRoleWithWebIdentity
            Condition:
              StringLike:
                token.actions.githubusercontent.com:sub: !Sub repo:${GitHubOrg}/${GitHubRepo}:*
              StringEquals:
                token.actions.githubusercontent.com:aud: sts.amazonaws.com
      ManagedPolicyArns:
        - arn:aws:iam::aws:policy/AdministratorAccess

Outputs:
  RoleArn:
    Description: IAM Role ARN for GitHub Actions
    Value: !GetAtt GitHubActionsDeployRole.Arn
    Export:
      Name: GitHubActionsDeployRoleArn
```

**Step 2: Commit**

```bash
git add infra/github-oidc.yml
git commit -m "infra: add CloudFormation template for GitHub OIDC provider and IAM role"
```

---

### Task 2: Create Backend Deploy Workflow

**Files:**
- Create: `.github/workflows/deploy-backend.yml`

**Step 1: Create the workflow file**

```yaml
name: Deploy Backend

on:
  pull_request:
    branches: [main]
    paths: ['handloom-admin/**']

  push:
    branches: [main]
    paths: ['handloom-admin/**']

  workflow_dispatch:
    inputs:
      environment:
        description: 'Environment to deploy'
        required: true
        type: choice
        options: [dev, prod]
        default: dev

permissions:
  id-token: write
  contents: read

env:
  CERT_ARN: arn:aws:acm:us-east-1:163053486005:certificate/a8dca48d-5992-4e6e-94c9-fffc7f3ac447

jobs:
  check:
    name: Lint & Test
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: handloom-admin
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache-dependency-path: handloom-admin/go.sum

      - name: Lint
        uses: golangci/golangci-lint-action@v6
        with:
          working-directory: handloom-admin

      - name: Test
        run: make test

  deploy:
    name: Deploy
    if: github.event_name != 'pull_request'
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: handloom-admin
    steps:
      - uses: actions/checkout@v4

      - uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ vars.AWS_ROLE_ARN }}
          aws-region: ${{ vars.AWS_REGION }}

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache-dependency-path: handloom-admin/go.sum

      - uses: actions/setup-node@v4
        with:
          node-version: 20

      - name: Install CDK
        run: npm install -g aws-cdk

      - name: Determine environment
        id: env
        run: |
          if [ "${{ github.event_name }}" = "workflow_dispatch" ]; then
            echo "env=${{ inputs.environment }}" >> "$GITHUB_OUTPUT"
          else
            echo "env=dev" >> "$GITHUB_OUTPUT"
          fi

      - name: Write .env file
        run: |
          if [ "${{ steps.env.outputs.env }}" = "prod" ]; then
            echo "${{ secrets.BACKEND_ENV_PROD }}" > .env.deploy
          else
            echo "${{ secrets.BACKEND_ENV_DEV }}" > .env.deploy
          fi

      - name: Build Lambdas
        run: make build-lambdas-active

      - name: CDK Deploy
        run: |
          set -a && . ./.env.deploy && set +a
          cd infra && cdk deploy --all --require-approval never \
            -c environment=${{ steps.env.outputs.env }} \
            -c certArn=${{ env.CERT_ARN }}
```

**Step 2: Commit**

```bash
git add .github/workflows/deploy-backend.yml
git commit -m "ci: add GitHub Actions workflow for backend deployment"
```

---

### Task 3: Create Admin Frontend Deploy Workflow

**Files:**
- Create: `.github/workflows/deploy-admin-frontend.yml`

**Step 1: Create the workflow file**

```yaml
name: Deploy Admin Frontend

on:
  pull_request:
    branches: [main]
    paths: ['handloom-admin-frontend/**']

  push:
    branches: [main]
    paths: ['handloom-admin-frontend/**']

  workflow_dispatch:
    inputs:
      environment:
        description: 'Environment to deploy'
        required: true
        type: choice
        options: [dev, prod]
        default: dev

permissions:
  id-token: write
  contents: read

env:
  CERT_ARN: arn:aws:acm:us-east-1:163053486005:certificate/a8dca48d-5992-4e6e-94c9-fffc7f3ac447

jobs:
  check:
    name: Typecheck, Lint & Format
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: handloom-admin-frontend
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: npm
          cache-dependency-path: handloom-admin-frontend/package-lock.json

      - run: npm ci
      - run: npm run check

  deploy:
    name: Deploy
    if: github.event_name != 'pull_request'
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: handloom-admin-frontend
    steps:
      - uses: actions/checkout@v4

      - uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ vars.AWS_ROLE_ARN }}
          aws-region: ${{ vars.AWS_REGION }}

      - uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: npm
          cache-dependency-path: handloom-admin-frontend/package-lock.json

      - name: Install CDK
        run: npm install -g aws-cdk

      - name: Determine environment
        id: env
        run: |
          if [ "${{ github.event_name }}" = "workflow_dispatch" ]; then
            echo "env=${{ inputs.environment }}" >> "$GITHUB_OUTPUT"
          else
            echo "env=dev" >> "$GITHUB_OUTPUT"
          fi

      - run: npm ci

      - name: Build
        run: npm run build:${{ steps.env.outputs.env }}

      - name: CDK Deploy
        run: |
          cd infra && cdk deploy --all --require-approval never \
            -c environment=${{ steps.env.outputs.env }} \
            -c certArn=${{ env.CERT_ARN }}
```

**Step 2: Commit**

```bash
git add .github/workflows/deploy-admin-frontend.yml
git commit -m "ci: add GitHub Actions workflow for admin frontend deployment"
```

---

### Task 4: Create Storefront Deploy Workflow

**Files:**
- Create: `.github/workflows/deploy-store.yml`

**Step 1: Create the workflow file**

```yaml
name: Deploy Storefront

on:
  pull_request:
    branches: [main]
    paths: ['homechrome-store/**']

  push:
    branches: [main]
    paths: ['homechrome-store/**']

  workflow_dispatch:
    inputs:
      environment:
        description: 'Environment to deploy'
        required: true
        type: choice
        options: [dev, prod]
        default: dev

permissions:
  id-token: write
  contents: read

env:
  CERT_ARN: arn:aws:acm:us-east-1:163053486005:certificate/a8dca48d-5992-4e6e-94c9-fffc7f3ac447

jobs:
  check:
    name: Lint
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: homechrome-store
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: npm
          cache-dependency-path: homechrome-store/package-lock.json

      - run: npm ci
      - run: npm run lint

  deploy:
    name: Deploy
    if: github.event_name != 'pull_request'
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: homechrome-store
    steps:
      - uses: actions/checkout@v4

      - uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ vars.AWS_ROLE_ARN }}
          aws-region: ${{ vars.AWS_REGION }}

      - uses: actions/setup-node@v4
        with:
          node-version: 20
          cache: npm
          cache-dependency-path: homechrome-store/package-lock.json

      - name: Install CDK
        run: npm install -g aws-cdk

      - name: Determine environment
        id: env
        run: |
          if [ "${{ github.event_name }}" = "workflow_dispatch" ]; then
            echo "env=${{ inputs.environment }}" >> "$GITHUB_OUTPUT"
          else
            echo "env=dev" >> "$GITHUB_OUTPUT"
          fi

      - name: Set environment URLs
        id: urls
        run: |
          if [ "${{ steps.env.outputs.env }}" = "prod" ]; then
            echo "api_url=https://api.homechrome.lldlab.com" >> "$GITHUB_OUTPUT"
            echo "site_url=https://homechrome.lldlab.com" >> "$GITHUB_OUTPUT"
            echo "stack_name=HomechromeStoreStack-prod" >> "$GITHUB_OUTPUT"
          else
            echo "api_url=https://dev-api.homechrome.lldlab.com" >> "$GITHUB_OUTPUT"
            echo "site_url=https://dev-store.homechrome.lldlab.com" >> "$GITHUB_OUTPUT"
            echo "stack_name=HomechromeStoreStack-dev" >> "$GITHUB_OUTPUT"
          fi

      - run: npm ci

      - name: Build with OpenNext
        env:
          NEXT_PUBLIC_API_URL: ${{ steps.urls.outputs.api_url }}
          NEXT_PUBLIC_SITE_URL: ${{ steps.urls.outputs.site_url }}
        run: npx @opennextjs/aws build

      - name: CDK Deploy
        run: |
          cd infra && cdk deploy --all --require-approval never \
            -c environment=${{ steps.env.outputs.env }} \
            -c certArn=${{ env.CERT_ARN }}

      - name: Invalidate CloudFront Cache
        run: |
          DIST_ID=$(aws cloudformation describe-stacks \
            --stack-name ${{ steps.urls.outputs.stack_name }} \
            --query "Stacks[0].Outputs[?OutputKey=='DistributionId'].OutputValue" \
            --output text)
          aws cloudfront create-invalidation --distribution-id "$DIST_ID" --paths '/*'
```

**Step 2: Commit**

```bash
git add .github/workflows/deploy-store.yml
git commit -m "ci: add GitHub Actions workflow for storefront deployment"
```

---

### Task 5: Deploy OIDC Stack & Configure GitHub

**Step 1: Deploy the CloudFormation stack**

```bash
aws cloudformation deploy \
  --template-file infra/github-oidc.yml \
  --stack-name GitHubOIDCStack \
  --capabilities CAPABILITY_NAMED_IAM \
  --region ap-south-1
```

**Step 2: Get the role ARN**

```bash
aws cloudformation describe-stacks \
  --stack-name GitHubOIDCStack \
  --query "Stacks[0].Outputs[?OutputKey=='RoleArn'].OutputValue" \
  --output text
```

Expected: `arn:aws:iam::163053486005:role/GitHubActionsDeployRole`

**Step 3: Configure GitHub repo**

Go to `github.com/UtkarshNigam1221/Homechrome` → Settings → Secrets and variables → Actions:

Variables:
- `AWS_ROLE_ARN` = (the ARN from step 2)
- `AWS_REGION` = `ap-south-1`

Secrets:
- `BACKEND_ENV_DEV` = contents of `handloom-admin/.env.dev`
- `BACKEND_ENV_PROD` = contents of `handloom-admin/.env.prod`

**Step 4: Test with manual dispatch**

Go to Actions tab → select any workflow → "Run workflow" → choose branch and environment `dev` → run.

---

### Task 6: Verify All Workflows

**Step 1: Push all changes to trigger auto-deploy**

```bash
git push origin main
```

**Step 2: Verify in GitHub Actions tab**

- All three workflows should trigger (since files changed in all paths)
- Check jobs: each should assume the OIDC role and deploy to dev
- Backend: Lambdas built and CDK deployed
- Admin Frontend: Vite built and CDK deployed
- Storefront: OpenNext built, CDK deployed, CloudFront invalidated

**Step 3: Test manual dispatch for prod**

Run each workflow manually with `environment: prod` to verify prod deploys work.
