#!/usr/bin/env bash
# Re-invoke ImageResizer for every original image in the bucket.
# Use after enabling sync-invoke contract to populate variants for legacy uploads.
# Idempotent: re-running skips originals that already have variants (HEAD probe
# of the -640.webp companion). Override with SKIP_EXISTING=0 to force re-process.
set -euo pipefail

BUCKET="${1:?Usage: $0 <bucket> <region> <function-name> [parallel]}"
REGION="${2:?Usage: $0 <bucket> <region> <function-name> [parallel]}"
FN="${3:?Usage: $0 <bucket> <region> <function-name> [parallel]}"
PARALLEL="${4:-1}"
SKIP_EXISTING="${SKIP_EXISTING:-1}"

variant_key() {
  local key="$1"
  local stem="${key%.*}"
  printf '%s-640.webp' "$stem"
}

resize_one() {
  local key="$1"

  if [ "$SKIP_EXISTING" = "1" ]; then
    local probe
    probe="$(variant_key "$key")"
    if aws s3api head-object \
      --bucket "$BUCKET" \
      --key "$probe" \
      --region "$REGION" \
      >/dev/null 2>&1; then
      echo "Skip (variants exist): $key"
      return 0
    fi
  fi

  echo "Resizing: $key"
  local payload
  payload=$(printf '{"Records":[{"s3":{"bucket":{"name":"%s"},"object":{"key":"%s"}}}]}' "$BUCKET" "$key")
  local outfile
  outfile=$(mktemp)
  trap 'rm -f "$outfile"' RETURN
  aws lambda invoke \
    --function-name "$FN" \
    --region "$REGION" \
    --payload "$payload" \
    --cli-binary-format raw-in-base64-out \
    --cli-read-timeout 0 \
    --cli-connect-timeout 60 \
    "$outfile" >/dev/null
}
export -f resize_one variant_key
export BUCKET REGION FN SKIP_EXISTING

aws s3api list-objects-v2 \
  --bucket "$BUCKET" \
  --prefix "assets/IMAGE/" \
  --region "$REGION" \
  --query "Contents[*].Key" \
  --output text \
  | tr '\t' '\n' \
  | grep -iE '\.(jpe?g|png|webp)$' \
  | grep -vE '\-(320|640|1080|1920)\.(jpg|jpeg|png|webp|avif)$' \
  | xargs -I {} -P "$PARALLEL" bash -c 'resize_one "$@"' _ {}

echo "Done."
