# CI/CD Pipeline Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Set up GitHub Actions CI/CD with OIDC auth for all 3 projects — PR checks gate merges, push to main auto-deploys to dev, manual dispatch deploys to any environment.

**Architecture:** 3 separate workflow files, one per project (backend, admin frontend, B2C store). Each has two jobs: `check` (runs on PRs) and `deploy` (runs on push to main or manual dispatch). AWS auth via OIDC — no stored credentials.

**Tech Stack:** GitHub Actions, AWS CDK (Go), AWS IAM OIDC, Go 1.24, Node.js 20, npm

---

### Task 1: AWS OIDC Identity Provider + IAM Role

This is a one-time manual AWS Console / CLI setup. Cannot be automated in a workflow file.

**Step 1: Create the OIDC identity provider**

Run in your terminal (or do via AWS Console > IAM > Identity providers > Add provider):

```bash
aws iam create-open-id-connect-provider \
  --url https://token.actions.githubusercontent.com \
  --client-id-list sts.amazonaws.com \
  --thumbprint-list 6938fd4d98bab03faadb97b34396831e3780aea1
```

**Step 2: Create the IAM role trust policy**

Create a file `github-actions-trust-policy.json` (do NOT commit this — it's a one-time setup artifact):

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::oidc-provider/token.actions.githubusercontent.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
        },
        "StringLike": {
          "token.actions.githubusercontent.com:sub": "repo:<your-github-org>/<your-repo-name>:*"
        }
      }
    }
  ]
}
```

Replace `<your-github-org>/<your-repo-name>` with the actual GitHub repo path (e.g., `utkarsh-nigam/Homechrome`).

**Step 3: Create the IAM role**

```bash
aws iam create-role \
  --role-name github-actions-deploy \
  --assume-role-policy-document file://github-actions-trust-policy.json
```

**Step 4: Attach permissions**

Attach `AdministratorAccess` for now (scope down later once stable):

```bash
aws iam attach-role-policy \
  --role-name github-actions-deploy \
  --policy-arn arn:aws:iam::aws:policy/AdministratorAccess
```

Note the role ARN from the output (format: `arn:aws:iam::<account-id>:role/github-actions-deploy`).

**Step 5: Configure GitHub repo settings**

Go to GitHub repo > Settings > Secrets and variables > Actions:

**Variables** (not secrets):
- `AWS_ROLE_ARN` = the role ARN from step 4
- `AWS_REGION` = `ap-south-1`
- `AWS_ACCOUNT_ID` = your AWS account ID

**Secrets** (encrypted):
- `BACKEND_ENV_DEV` = full contents of `handloom-admin/.env.dev`
- `BACKEND_ENV_PROD` = full contents of `handloom-admin/.env.prod`

**Step 6: Verify setup**

No automated test — verified when the first workflow runs successfully in Task 4.

**Step 7: Commit (nothing to commit — this is AWS/GitHub config only)**

---

### Task 2: Backend Workflow (`deploy-backend.yml`)

**Files:**
- Create: `.github/workflows/deploy-backend.yml`

**Step 1: Create the workflow file**

```yaml
name: Backend

on:
  pull_request:
    branches: [main]
    paths:
      - 'handloom-admin/**'
      - '.github/workflows/deploy-backend.yml'

  push:
    branches: [main]
    paths:
      - 'handloom-admin/**'
      - '.github/workflows/deploy-backend.yml'

  workflow_dispatch:
    inputs:
      environment:
        description: 'Environment to deploy'
        required: true
        type: choice
        options:
          - dev
          - prod

permissions:
  id-token: write
  contents: read

env:
  GO_VERSION: '1.24'
  NODE_VERSION: '20'

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
          go-version: ${{ env.GO_VERSION }}
          cache-dependency-path: handloom-admin/go.sum

      - name: Install golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest
          working-directory: handloom-admin

      - name: Run tests
        run: make test

  deploy:
    name: Deploy (${{ github.event.inputs.environment || 'dev' }})
    if: github.event_name == 'push' || github.event_name == 'workflow_dispatch'
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: handloom-admin
    environment: ${{ github.event.inputs.environment || 'dev' }}

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache-dependency-path: handloom-admin/go.sum

      - uses: actions/setup-node@v4
        with:
          node-version: ${{ env.NODE_VERSION }}

      - name: Install AWS CDK
        run: npm install -g aws-cdk

      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ vars.AWS_ROLE_ARN }}
          aws-region: ${{ vars.AWS_REGION }}

      - name: Write env file
        run: |
          ENV=${{ github.event.inputs.environment || 'dev' }}
          if [ "$ENV" = "prod" ]; then
            echo "${{ secrets.BACKEND_ENV_PROD }}" > .env.deploy
          else
            echo "${{ secrets.BACKEND_ENV_DEV }}" > .env.deploy
          fi

      - name: Build Lambda functions
        run: make build-lambdas-active

      - name: CDK Deploy
        run: |
          ENV=${{ github.event.inputs.environment || 'dev' }}
          CERT_ARN="arn:aws:acm:us-east-1:${{ vars.AWS_ACCOUNT_ID }}:certificate/a8dca48d-5992-4e6e-94c9-fffc7f3ac447"
          set -a && . ./.env.deploy && set +a
          cd infra && cdk deploy --all --require-approval never \
            -c environment=$ENV \
            -c certArn=$CERT_ARN
```

**Step 2: Verify the file is valid YAML**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/deploy-backend.yml'))"`
Expected: No output (valid YAML)

**Step 3: Commit**

```bash
git add .github/workflows/deploy-backend.yml
git commit -m "ci: add backend deploy workflow with OIDC auth"
```

---

### Task 3: Admin Frontend Workflow (`deploy-admin-frontend.yml`)

**Files:**
- Create: `.github/workflows/deploy-admin-frontend.yml`

**Step 1: Create the workflow file**

```yaml
name: Admin Frontend

on:
  pull_request:
    branches: [main]
    paths:
      - 'handloom-admin-frontend/**'
      - '.github/workflows/deploy-admin-frontend.yml'

  push:
    branches: [main]
    paths:
      - 'handloom-admin-frontend/**'
      - '.github/workflows/deploy-admin-frontend.yml'

  workflow_dispatch:
    inputs:
      environment:
        description: 'Environment to deploy'
        required: true
        type: choice
        options:
          - dev
          - prod

permissions:
  id-token: write
  contents: read

env:
  NODE_VERSION: '20'

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
          node-version: ${{ env.NODE_VERSION }}
          cache: npm
          cache-dependency-path: handloom-admin-frontend/package-lock.json

      - run: npm ci
      - run: npm run check

  deploy:
    name: Deploy (${{ github.event.inputs.environment || 'dev' }})
    if: github.event_name == 'push' || github.event_name == 'workflow_dispatch'
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: handloom-admin-frontend
    environment: ${{ github.event.inputs.environment || 'dev' }}

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: npm
          cache-dependency-path: handloom-admin-frontend/package-lock.json

      - run: npm ci

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache-dependency-path: handloom-admin-frontend/infra/go.sum

      - name: Install AWS CDK
        run: npm install -g aws-cdk

      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ vars.AWS_ROLE_ARN }}
          aws-region: ${{ vars.AWS_REGION }}

      - name: Build & Deploy
        run: |
          ENV=${{ github.event.inputs.environment || 'dev' }}
          CERT_ARN="arn:aws:acm:us-east-1:${{ vars.AWS_ACCOUNT_ID }}:certificate/a8dca48d-5992-4e6e-94c9-fffc7f3ac447"
          npm run build:$ENV
          cd infra && cdk deploy --all --require-approval never \
            -c environment=$ENV \
            -c certArn=$CERT_ARN
```

**Step 2: Verify the file is valid YAML**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/deploy-admin-frontend.yml'))"`
Expected: No output (valid YAML)

**Step 3: Commit**

```bash
git add .github/workflows/deploy-admin-frontend.yml
git commit -m "ci: add admin frontend deploy workflow"
```

---

### Task 4: B2C Store Workflow (`deploy-store.yml`)

**Files:**
- Create: `.github/workflows/deploy-store.yml`

**Step 1: Create the workflow file**

```yaml
name: Store

on:
  pull_request:
    branches: [main]
    paths:
      - 'homechrome-store/**'
      - '.github/workflows/deploy-store.yml'

  push:
    branches: [main]
    paths:
      - 'homechrome-store/**'
      - '.github/workflows/deploy-store.yml'

  workflow_dispatch:
    inputs:
      environment:
        description: 'Environment to deploy'
        required: true
        type: choice
        options:
          - dev
          - prod

permissions:
  id-token: write
  contents: read

env:
  NODE_VERSION: '20'

jobs:
  check:
    name: Lint & Build
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: homechrome-store

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: npm
          cache-dependency-path: homechrome-store/package-lock.json

      - run: npm ci
      - run: npm run lint
      - run: npm run build

  deploy:
    name: Deploy (${{ github.event.inputs.environment || 'dev' }})
    if: github.event_name == 'push' || github.event_name == 'workflow_dispatch'
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: homechrome-store
    environment: ${{ github.event.inputs.environment || 'dev' }}

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: npm
          cache-dependency-path: homechrome-store/package-lock.json

      - run: npm ci

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache-dependency-path: homechrome-store/infra/go.sum

      - name: Install AWS CDK
        run: npm install -g aws-cdk

      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ vars.AWS_ROLE_ARN }}
          aws-region: ${{ vars.AWS_REGION }}

      - name: Set environment URLs
        id: urls
        run: |
          ENV=${{ github.event.inputs.environment || 'dev' }}
          if [ "$ENV" = "prod" ]; then
            echo "api_url=https://api.homechrome.in" >> $GITHUB_OUTPUT
            echo "site_url=https://homechrome.in" >> $GITHUB_OUTPUT
            echo "stack_name=HomechromeStoreStack-prod" >> $GITHUB_OUTPUT
          else
            echo "api_url=https://dev-api.homechrome.in" >> $GITHUB_OUTPUT
            echo "site_url=https://dev-store.homechrome.in" >> $GITHUB_OUTPUT
            echo "stack_name=HomechromeStoreStack-dev" >> $GITHUB_OUTPUT
          fi

      - name: Build with OpenNext
        env:
          NEXT_PUBLIC_API_URL: ${{ steps.urls.outputs.api_url }}
          NEXT_PUBLIC_SITE_URL: ${{ steps.urls.outputs.site_url }}
        run: npx @opennextjs/aws build

      - name: CDK Deploy
        run: |
          ENV=${{ github.event.inputs.environment || 'dev' }}
          CERT_ARN="arn:aws:acm:us-east-1:${{ vars.AWS_ACCOUNT_ID }}:certificate/a8dca48d-5992-4e6e-94c9-fffc7f3ac447"
          cd infra && cdk deploy --all --require-approval never \
            -c environment=$ENV \
            -c certArn=$CERT_ARN

      - name: Invalidate CloudFront cache
        run: |
          DIST_ID=$(aws cloudformation describe-stacks \
            --stack-name ${{ steps.urls.outputs.stack_name }} \
            --query "Stacks[0].Outputs[?OutputKey=='DistributionId'].OutputValue" \
            --output text)
          aws cloudfront create-invalidation \
            --distribution-id $DIST_ID \
            --paths '/*'
```

**Step 2: Verify the file is valid YAML**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/deploy-store.yml'))"`
Expected: No output (valid YAML)

**Step 3: Commit**

```bash
git add .github/workflows/deploy-store.yml
git commit -m "ci: add B2C store deploy workflow"
```

---

### Task 5: Test the Pipeline

**Step 1: Push and verify PR checks**

Create a test branch and PR to verify the check jobs work:

```bash
git checkout -b ci/test-pipeline
git push -u origin ci/test-pipeline
gh pr create --title "ci: test pipeline" --body "Testing CI/CD workflows" --base main
```

Go to GitHub > Actions tab and verify the check jobs trigger and pass for each project that has changes.

**Step 2: Merge and verify dev deploy**

Once PR checks pass, merge the PR:

```bash
gh pr merge --squash
```

Go to GitHub > Actions tab and verify the deploy jobs trigger for changed projects and deploy to dev.

**Step 3: Test manual dispatch**

Go to GitHub > Actions > select a workflow > "Run workflow" > choose `dev` > Run.

Verify it deploys successfully.

**Step 4: Commit (nothing to commit — this is verification)**
