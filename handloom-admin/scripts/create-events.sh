#!/bin/bash
set -euo pipefail

ENDPOINT="${AWS_ENDPOINT:-http://localhost:4566}"
REGION="${AWS_REGION:-ap-south-1}"
ACCOUNT="000000000000"
ENV="local"

# Set dummy AWS credentials for LocalStack (prevents SSO/profile auth)
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-local}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-local}"

awslocal() {
    aws --endpoint-url="$ENDPOINT" --region "$REGION" "$@"
}

echo "=== Creating SNS topic ==="
TOPIC_ARN=$(awslocal sns create-topic --name "handloom-events-${ENV}" --query 'TopicArn' --output text)
echo "Topic: $TOPIC_ARN"

create_queue_pair() {
    local name=$1
    local max_receive=$2

    echo "--- Creating $name queue pair ---"

    # DLQ
    DLQ_URL=$(awslocal sqs create-queue --queue-name "handloom-${name}-dlq-${ENV}" --query 'QueueUrl' --output text)
    DLQ_ARN="arn:aws:sqs:${REGION}:${ACCOUNT}:handloom-${name}-dlq-${ENV}"

    # Main queue with redrive policy
    QUEUE_URL=$(awslocal sqs create-queue \
        --queue-name "handloom-${name}-${ENV}" \
        --attributes "{\"RedrivePolicy\":\"{\\\"deadLetterTargetArn\\\":\\\"${DLQ_ARN}\\\",\\\"maxReceiveCount\\\":\\\"${max_receive}\\\"}\"}" \
        --query 'QueueUrl' --output text)
    QUEUE_ARN="arn:aws:sqs:${REGION}:${ACCOUNT}:handloom-${name}-${ENV}"

    echo "  Queue: $QUEUE_URL"
    echo "  DLQ:   $DLQ_URL"
    echo "$QUEUE_ARN"
}

NOTIF_ARN=$(create_queue_pair "notification" 3)
REPORT_ARN=$(create_queue_pair "report" 3)
ANALYTICS_ARN=$(create_queue_pair "analytics" 3)
AUDIT_ARN=$(create_queue_pair "audit" 5)

echo ""
echo "=== Subscribing queues to SNS topic ==="

subscribe_with_filter() {
    local queue_arn=$1
    local filter=$2
    local name=$3

    awslocal sns subscribe \
        --topic-arn "$TOPIC_ARN" \
        --protocol sqs \
        --notification-endpoint "$queue_arn" \
        --attributes "{\"FilterPolicy\":\"$filter\"}" \
        > /dev/null

    echo "  Subscribed $name"
}

subscribe_with_filter "$NOTIF_ARN" \
    '{\"event_type\":[{\"prefix\":\"order.\"},{\"prefix\":\"payment.\"},{\"prefix\":\"shipment.\"},\"customer.registered\"]}' \
    "notification"

subscribe_with_filter "$REPORT_ARN" \
    '{\"event_type\":[{\"prefix\":\"order.\"},{\"prefix\":\"payment.\"}]}' \
    "report"

subscribe_with_filter "$ANALYTICS_ARN" \
    '{\"event_type\":[{\"prefix\":\"order.\"},{\"prefix\":\"payment.\"},{\"prefix\":\"product.\"},{\"prefix\":\"inventory.\"},{\"prefix\":\"customer.\"}]}' \
    "analytics"

# Audit gets ALL events — no filter policy
awslocal sns subscribe \
    --topic-arn "$TOPIC_ARN" \
    --protocol sqs \
    --notification-endpoint "$AUDIT_ARN" \
    > /dev/null
echo "  Subscribed audit (all events)"

echo ""
echo "=== Event infrastructure ready ==="
echo "SNS Topic ARN: $TOPIC_ARN"
