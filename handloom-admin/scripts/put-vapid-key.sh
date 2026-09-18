#!/bin/bash

# Store the Web Push (VAPID) private key in SSM for one environment.
#
# The push Lambda reads /handloom/{env}/vapid-private-key at runtime, so this
# parameter must exist before the stack is deployed. Generate the pair once per
# environment with `make vapid-keys`; the public key is not secret and goes in
# the BACKEND_ENV_{ENV} blob instead.
#
# Usage:
#   VAPID_PRIVATE_KEY=<key> ./scripts/put-vapid-key.sh dev
#   ./scripts/put-vapid-key.sh dev --stdin < key.txt
#   VAPID_PRIVATE_KEY=<key> ./scripts/put-vapid-key.sh prod --rotate
#
# The key is never taken from the command line: argv is visible to every other
# process on the host and lands in shell history and CI logs.

set -euo pipefail

ENVIRONMENT="${1:-}"
REGION="${AWS_REGION:-ap-south-1}"
ROTATE=false
FROM_STDIN=false

for arg in "${@:2}"; do
    case "$arg" in
        --rotate) ROTATE=true ;;
        --stdin)  FROM_STDIN=true ;;
        *) echo "Unknown option: $arg" >&2; exit 2 ;;
    esac
done

case "$ENVIRONMENT" in
    dev|prod) ;;
    *)
        echo "Usage: [VAPID_PRIVATE_KEY=<key>] $0 <dev|prod> [--stdin] [--rotate]" >&2
        exit 2
        ;;
esac

PARAM_NAME="/handloom/${ENVIRONMENT}/vapid-private-key"

# --- Read the key ---------------------------------------------------------
if [ "$FROM_STDIN" = true ]; then
    read -r VAPID_PRIVATE_KEY
fi

if [ -z "${VAPID_PRIVATE_KEY:-}" ]; then
    echo "::error::No key supplied. Set VAPID_PRIVATE_KEY, or pass --stdin." >&2
    exit 1
fi

# A P-256 private key is 32 bytes, base64url-encoded without padding.
if ! printf '%s' "$VAPID_PRIVATE_KEY" | grep -qE '^[A-Za-z0-9_-]{43}$'; then
    echo "::error::That does not look like a VAPID private key (expected 43 base64url characters)." >&2
    echo "         Did you pass the public key by mistake? It is 87 characters." >&2
    exit 1
fi

# Keep it out of any log this script or its caller produces.
echo "::add-mask::$VAPID_PRIVATE_KEY"

# --- Refuse to clobber a live key ----------------------------------------
# Browsers bind their subscription to the key they subscribed with, so replacing
# it silently kills every existing subscriber with no error surfaced anywhere.
# Fail closed: only a definite ParameterNotFound counts as "safe to create".
# Treating any failed call as absent would let a transient API error skip this
# guard and --overwrite a live key.
probe_status=0
probe_output=$(aws ssm get-parameter --name "$PARAM_NAME" --region "$REGION" 2>&1) || probe_status=$?

if [ "$probe_status" -ne 0 ] && ! grep -q "ParameterNotFound" <<<"$probe_output"; then
    echo "::error::Could not determine whether $PARAM_NAME already exists:" >&2
    echo "$probe_output" >&2
    echo "         Refusing to write blind — a live key could be overwritten." >&2
    exit 1
fi

if [ "$probe_status" -eq 0 ]; then
    if [ "$ROTATE" != true ]; then
        echo "::error::$PARAM_NAME already exists in ${ENVIRONMENT}." >&2
        echo "         Overwriting it invalidates EVERY existing push subscription:" >&2
        echo "         those browsers stop receiving notifications and are never told." >&2
        echo "         Re-run with --rotate only if that is genuinely what you want." >&2
        exit 1
    fi
    echo "WARNING: rotating ${ENVIRONMENT} — every existing subscription dies."
    ACTION="Rotated"
else
    ACTION="Created"
fi

# --- Write ----------------------------------------------------------------
aws ssm put-parameter \
    --name "$PARAM_NAME" \
    --type SecureString \
    --value "$VAPID_PRIVATE_KEY" \
    --description "Web Push VAPID private key (${ENVIRONMENT}) — read at runtime by the push Lambda" \
    --region "$REGION" \
    --overwrite >/dev/null

echo "${ACTION} ${PARAM_NAME} in ${REGION}."
echo
echo "Next:"
echo "  1. Put the matching VAPID_PUBLIC_KEY and VAPID_SUBJECT in the"
echo "     BACKEND_ENV_$(echo "$ENVIRONMENT" | tr '[:lower:]' '[:upper:]') secret (neither is secret)."
echo "  2. Deploy the backend so the push Lambda picks up the parameter."
