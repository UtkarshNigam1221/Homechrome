# Get the RDS endpoint from CDK output
RDS_ENDPOINT=$(aws cloudformation describe-stacks \
--stack-name HandloomDatabaseStack-dev \
--query "Stacks[0].Outputs[?OutputKey=='CatalogDBEndpoint'].OutputValue" \
--output text)

# Get the password from Secrets Manager
SECRET_ARN=$(aws cloudformation describe-stacks \
--stack-name HandloomDatabaseStack-dev \
--query "Stacks[0].Outputs[?OutputKey=='CatalogDBSecretARN'].OutputValue" \
--output text)
PASSWORD=$(aws secretsmanager get-secret-value \
--secret-id "$SECRET_ARN" \
--query "SecretString" --output text | jq -r '.password')

# Run migration
PGPASSWORD="$PASSWORD" psql -h "$RDS_ENDPOINT" -U handloom -d handloom \
-f migrations/001_catalog_schema.sql

This is a manual step after first deploy — not automated in CDK.
