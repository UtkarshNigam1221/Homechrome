#!/usr/bin/env bash
# Bootstrap SSM parameters required by HandloomAPIStack before first deploy.
# Idempotent: --no-overwrite skips if already exists.
#
# Usage:
#   ./scripts/bootstrap-env.sh dev
#   ./scripts/bootstrap-env.sh prod

set -euo pipefail

ENV="${1:-}"
REGION="${AWS_REGION:-ap-south-1}"

if [ -z "$ENV" ]; then
  echo "Usage: $0 <env>  (e.g. dev, prod)" >&2
  exit 1
fi

if [[ "$ENV" != "dev" && "$ENV" != "prod" ]]; then
  echo "Warning: env '$ENV' is not dev or prod. Continuing anyway..." >&2
fi

put_secret() {
  local name="$1"
  local description="$2"
  if aws ssm get-parameter --name "$name" --region "$REGION" >/dev/null 2>&1; then
    echo "✓ $name already exists, skipping (use 'aws ssm put-parameter --overwrite' to rotate)"
    return 0
  fi
  local value
  value=$(openssl rand -hex 32)
  aws ssm put-parameter \
    --name "$name" \
    --value "$value" \
    --type SecureString \
    --description "$description" \
    --region "$REGION" \
    --no-overwrite >/dev/null
  echo "✓ Created $name"
}

put_secret "/handloom/$ENV/jwt-secret" "JWT secret for admin token signing"
put_secret "/handloom/$ENV/customer-jwt-secret" "JWT secret for B2C customer token signing"

echo ""
echo "Bootstrap complete for env=$ENV in region=$REGION"
echo "Proceed with CDK deploy."
