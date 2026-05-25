#!/usr/bin/env bash
# Loops `aws lambda invoke` on the worker-embedding-backfill Lambda until
# `remaining_estimate == 0`. Same loop the GH Actions workflow runs, but
# locally — useful when GH Actions quota is exhausted.
#
# Usage:
#   scripts/backfill-embeddings.sh <env> [batch_size] [force_reembed]
# Examples:
#   scripts/backfill-embeddings.sh dev
#   scripts/backfill-embeddings.sh dev 200 false
#   scripts/backfill-embeddings.sh prod 100 true     # force re-embed all rows
#
# Requires:
#   - AWS creds (aws sso login --profile <profile>)
#   - jq
#   - Lambda handloom-worker-embedding-backfill-<env> deployed
set -euo pipefail

ENV=${1:?usage: $0 <env> [batch_size] [force_reembed]}
BATCH_SIZE=${2:-100}
FORCE_REEMBED=${3:-false}
REGION=${AWS_REGION:-ap-south-1}
FN="handloom-worker-embedding-backfill-${ENV}"
MAX_ITER=${MAX_ITER:-50}

# Validate args
if ! [[ "$BATCH_SIZE" =~ ^[0-9]+$ ]]; then
  echo "ERROR: batch_size must be a positive integer, got: $BATCH_SIZE" >&2
  exit 1
fi
if [[ "$FORCE_REEMBED" != "true" && "$FORCE_REEMBED" != "false" ]]; then
  echo "ERROR: force_reembed must be 'true' or 'false', got: $FORCE_REEMBED" >&2
  exit 1
fi

# Sanity check Lambda exists + AWS creds work
if ! aws lambda get-function --function-name "$FN" --region "$REGION" >/dev/null 2>&1; then
  echo "ERROR: Lambda $FN not found in $REGION (or AWS creds expired). Run: aws sso login" >&2
  exit 1
fi

PAYLOAD=$(jq -c -n \
  --argjson b "$BATCH_SIZE" \
  --argjson f "$FORCE_REEMBED" \
  '{batch_size:$b, force_reembed:$f, max_duration_seconds:840}')

RESPONSE_FILE=$(mktemp)
trap 'rm -f "$RESPONSE_FILE"' EXIT

TOTAL_PROCESSED=0
ITER=0
while [ $ITER -lt $MAX_ITER ]; do
  ITER=$((ITER+1))
  echo "==== iteration $ITER ===="

  if ! aws lambda invoke \
      --function-name "$FN" \
      --payload "$PAYLOAD" \
      --cli-binary-format raw-in-base64-out \
      --region "$REGION" \
      "$RESPONSE_FILE" > /dev/null 2>&1; then
    echo "ERROR: aws lambda invoke failed" >&2
    exit 1
  fi

  if [ ! -s "$RESPONSE_FILE" ]; then
    echo "ERROR: empty response.json from Lambda" >&2
    exit 1
  fi

  ERR=$(jq -r '.error // ""' "$RESPONSE_FILE")
  if [ -n "$ERR" ]; then
    echo "ERROR: Lambda reported: $ERR" >&2
    cat "$RESPONSE_FILE" | jq . >&2
    exit 1
  fi

  PROCESSED=$(jq -r '.processed // 0' "$RESPONSE_FILE")
  REMAINING=$(jq -r '.remaining_estimate // -1' "$RESPONSE_FILE")
  DURATION=$(jq -r '.duration_ms // 0' "$RESPONSE_FILE")

  if [ "$REMAINING" = "-1" ]; then
    echo "ERROR: response missing remaining_estimate field" >&2
    cat "$RESPONSE_FILE" | jq . >&2
    exit 1
  fi

  TOTAL_PROCESSED=$((TOTAL_PROCESSED + PROCESSED))
  echo "  processed=$PROCESSED  remaining=$REMAINING  duration_ms=$DURATION"

  if [ "$REMAINING" = "0" ]; then
    echo ""
    echo "✓ Backfill complete after $ITER iteration(s)"
    echo "  total rows processed: $TOTAL_PROCESSED"
    exit 0
  fi
done

echo "ERROR: hit MAX_ITER=$MAX_ITER without finishing (remaining=$REMAINING). Re-run or raise MAX_ITER." >&2
exit 1
