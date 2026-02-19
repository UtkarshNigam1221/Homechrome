#!/bin/bash

# Initialize local S3 buckets in LocalStack
# Usage: ./scripts/init-local-s3.sh

set -e

ENDPOINT="${AWS_ENDPOINT:-http://localhost:4566}"
REGION="${AWS_REGION:-ap-south-1}"

# Set dummy AWS credentials for LocalStack
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-local}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-local}"

echo "Creating S3 buckets at $ENDPOINT..."

# Assets bucket (product images, category images, artisan photos)
aws s3 mb "s3://handloom-assets" \
    --endpoint-url "$ENDPOINT" \
    --region "$REGION" \
    2>/dev/null || echo "Bucket handloom-assets already exists"

echo "Created handloom-assets bucket"

# Uploads bucket (temporary private uploads)
aws s3 mb "s3://handloom-uploads" \
    --endpoint-url "$ENDPOINT" \
    --region "$REGION" \
    2>/dev/null || echo "Bucket handloom-uploads already exists"

echo "Created handloom-uploads bucket"

echo ""
echo "S3 buckets created successfully!"
echo ""
echo "Buckets:"
aws s3 ls --endpoint-url "$ENDPOINT" --region "$REGION"
