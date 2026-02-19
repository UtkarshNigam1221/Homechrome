#!/bin/bash

# Script to seed initial data for testing
# Usage: ./scripts/seed-data.sh

ENDPOINT="${DYNAMODB_LOCAL_ENDPOINT:-http://localhost:4566}"
REGION="${AWS_REGION:-ap-south-1}"
TABLE="handloom-core"

# Set dummy AWS credentials for local DynamoDB (bypasses SSO)
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-fakekey}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-fakesecret}"

# Find AWS CLI (check common locations)
if command -v aws &> /dev/null; then
    AWS_CMD="aws"
elif [ -x "/usr/local/bin/aws" ]; then
    AWS_CMD="/usr/local/bin/aws"
elif [ -x "/opt/homebrew/bin/aws" ]; then
    AWS_CMD="/opt/homebrew/bin/aws"
else
    echo "ERROR: AWS CLI not found. Please install it or add it to PATH."
    exit 1
fi

echo "Using AWS CLI: $AWS_CMD"
echo "Seeding test data at $ENDPOINT..."

# =============================================================================
# Admin User
# =============================================================================
# Default credentials:
#   Email: admin@handloom.com
#   Password: Admin@123!
#
# Password hash generated using bcrypt (cost 12)
# To generate a new hash: go run scripts/generate-password-hash.go "YourPassword"
# Or use: htpasswd -nbBC 12 "" "YourPassword" | tr -d ':\n' | sed 's/$2y/$2a/'
#
# DynamoDB Key Schema for Users:
#   PK: USER#<user_id>
#   SK: METADATA
#   GSI1PK: USER_EMAIL
#   GSI1SK: <email>
#   GSI2PK: USER_ROLE
#   GSI2SK: <role>#<user_id>

echo "Creating admin user..."
# Password: Admin@123! (bcrypt hash, cost 12)
# This hash was generated using: go run scripts/generate-password-hash.go "Admin@123!"
ADMIN_PASSWORD_HASH='$2a$12$QA9/lw32rWAx6606bYrj4e1qkl9cqzp.MiG7lZS/K7giv4kD5/InC'

$AWS_CMD dynamodb put-item \
    --table-name $TABLE \
    --item '{
        "PK": {"S": "USER#usr_admin_001"},
        "SK": {"S": "METADATA"},
        "GSI1PK": {"S": "USER_EMAIL"},
        "GSI1SK": {"S": "admin@handloom.com"},
        "GSI2PK": {"S": "USER_ROLE"},
        "GSI2SK": {"S": "admin#usr_admin_001"},
        "entity_type": {"S": "USER"},
        "id": {"S": "usr_admin_001"},
        "email": {"S": "admin@handloom.com"},
        "name": {"S": "System Administrator"},
        "password_hash": {"S": "$2a$12$QA9/lw32rWAx6606bYrj4e1qkl9cqzp.MiG7lZS/K7giv4kD5/InC"},
        "role": {"S": "admin"},
        "status": {"S": "ACTIVE"},
        "email_verified": {"BOOL": true},
        "created_at": {"S": "2024-01-15T10:00:00Z"},
        "updated_at": {"S": "2024-01-15T10:00:00Z"}
    }' \
    --endpoint-url $ENDPOINT \
    --region $REGION 2>/dev/null || echo "Admin user already exists"

# Create manager user
echo "Creating manager user..."
$AWS_CMD dynamodb put-item \
    --table-name $TABLE \
    --item '{
        "PK": {"S": "USER#usr_manager_001"},
        "SK": {"S": "METADATA"},
        "GSI1PK": {"S": "USER_EMAIL"},
        "GSI1SK": {"S": "manager@handloom.com"},
        "GSI2PK": {"S": "USER_ROLE"},
        "GSI2SK": {"S": "manager#usr_manager_001"},
        "entity_type": {"S": "USER"},
        "id": {"S": "usr_manager_001"},
        "email": {"S": "manager@handloom.com"},
        "name": {"S": "Store Manager"},
        "password_hash": {"S": "$2a$12$QA9/lw32rWAx6606bYrj4e1qkl9cqzp.MiG7lZS/K7giv4kD5/InC"},
        "role": {"S": "manager"},
        "status": {"S": "ACTIVE"},
        "email_verified": {"BOOL": true},
        "created_at": {"S": "2024-01-15T10:00:00Z"},
        "updated_at": {"S": "2024-01-15T10:00:00Z"}
    }' \
    --endpoint-url $ENDPOINT \
    --region $REGION 2>/dev/null || echo "Manager user already exists"

# =============================================================================
# Categories and Pricing Rules
# =============================================================================

echo ""
echo "Creating categories and pricing rules..."

# Create root category: Home Textiles
echo "Creating root category: Home Textiles..."
$AWS_CMD dynamodb put-item \
    --table-name $TABLE \
    --item '{
        "PK": {"S": "CATEGORY#cat_home_textiles"},
        "SK": {"S": "METADATA"},
        "GSI1PK": {"S": "CATEGORY#ROOT"},
        "GSI1SK": {"S": "CATEGORY#cat_home_textiles"},
        "entity_type": {"S": "CATEGORY"},
        "id": {"S": "cat_home_textiles"},
        "name": {"S": "Home Textiles"},
        "slug": {"S": "home-textiles"},
        "description": {"S": "Quality handloom home textiles"},
        "level": {"N": "0"},
        "path": {"S": "cat_home_textiles"},
        "ancestor_ids": {"L": []},
        "status": {"S": "ACTIVE"},
        "allow_custom_dimensions": {"BOOL": false},
        "product_count": {"N": "0"},
        "design_count": {"N": "0"},
        "own_attributes": {"L": [
            {"M": {
                "name": {"S": "material"},
                "label": {"S": "Material"},
                "type": {"S": "SELECT"},
                "required": {"BOOL": true},
                "affects_pricing": {"BOOL": true},
                "options": {"L": [
                    {"M": {"value": {"S": "cotton"}, "label": {"S": "Cotton"}, "surcharge": {"N": "0"}}},
                    {"M": {"value": {"S": "silk"}, "label": {"S": "Silk"}, "surcharge": {"N": "50000"}}},
                    {"M": {"value": {"S": "linen"}, "label": {"S": "Linen"}, "surcharge": {"N": "30000"}}},
                    {"M": {"value": {"S": "blend"}, "label": {"S": "Cotton-Silk Blend"}, "surcharge": {"N": "25000"}}}
                ]}
            }},
            {"M": {
                "name": {"S": "color"},
                "label": {"S": "Color"},
                "type": {"S": "TEXT"},
                "required": {"BOOL": true},
                "affects_pricing": {"BOOL": false}
            }},
            {"M": {
                "name": {"S": "weave_type"},
                "label": {"S": "Weave Type"},
                "type": {"S": "SELECT"},
                "required": {"BOOL": false},
                "affects_pricing": {"BOOL": true},
                "options": {"L": [
                    {"M": {"value": {"S": "plain"}, "label": {"S": "Plain Weave"}, "surcharge": {"N": "0"}}},
                    {"M": {"value": {"S": "twill"}, "label": {"S": "Twill Weave"}, "surcharge": {"N": "10000"}}},
                    {"M": {"value": {"S": "satin"}, "label": {"S": "Satin Weave"}, "surcharge": {"N": "20000"}}},
                    {"M": {"value": {"S": "jacquard"}, "label": {"S": "Jacquard"}, "surcharge": {"N": "35000"}}}
                ]}
            }}
        ]},
        "created_at": {"S": "2024-01-15T10:00:00Z"},
        "updated_at": {"S": "2024-01-15T10:00:00Z"}
    }' \
    --endpoint-url $ENDPOINT \
    --region $REGION

# Create Bedding category
echo "Creating category: Bedding..."
$AWS_CMD dynamodb put-item \
    --table-name $TABLE \
    --item '{
        "PK": {"S": "CATEGORY#cat_bedding"},
        "SK": {"S": "METADATA"},
        "GSI1PK": {"S": "CATEGORY#cat_home_textiles"},
        "GSI1SK": {"S": "CATEGORY#cat_bedding"},
        "entity_type": {"S": "CATEGORY"},
        "id": {"S": "cat_bedding"},
        "name": {"S": "Bedding"},
        "slug": {"S": "bedding"},
        "description": {"S": "Quality bedding products"},
        "parent_id": {"S": "cat_home_textiles"},
        "level": {"N": "1"},
        "path": {"S": "cat_home_textiles/cat_bedding"},
        "ancestor_ids": {"L": [{"S": "cat_home_textiles"}]},
        "status": {"S": "ACTIVE"},
        "allow_custom_dimensions": {"BOOL": false},
        "product_count": {"N": "0"},
        "design_count": {"N": "0"},
        "own_attributes": {"L": [
            {"M": {
                "name": {"S": "thread_count"},
                "label": {"S": "Thread Count"},
                "type": {"S": "SELECT"},
                "required": {"BOOL": false},
                "affects_pricing": {"BOOL": true},
                "options": {"L": [
                    {"M": {"value": {"S": "200"}, "label": {"S": "200 TC"}, "surcharge": {"N": "0"}}},
                    {"M": {"value": {"S": "300"}, "label": {"S": "300 TC"}, "surcharge": {"N": "15000"}}},
                    {"M": {"value": {"S": "400"}, "label": {"S": "400 TC"}, "surcharge": {"N": "30000"}}},
                    {"M": {"value": {"S": "600"}, "label": {"S": "600 TC"}, "surcharge": {"N": "50000"}}}
                ]}
            }}
        ]},
        "created_at": {"S": "2024-01-15T10:00:00Z"},
        "updated_at": {"S": "2024-01-15T10:00:00Z"}
    }' \
    --endpoint-url $ENDPOINT \
    --region $REGION

# Create Bedsheets category (with custom dimensions)
echo "Creating category: Bedsheets..."
$AWS_CMD dynamodb put-item \
    --table-name $TABLE \
    --item '{
        "PK": {"S": "CATEGORY#cat_bedsheets"},
        "SK": {"S": "METADATA"},
        "GSI1PK": {"S": "CATEGORY#cat_bedding"},
        "GSI1SK": {"S": "CATEGORY#cat_bedsheets"},
        "entity_type": {"S": "CATEGORY"},
        "id": {"S": "cat_bedsheets"},
        "name": {"S": "Bedsheets"},
        "slug": {"S": "bedsheets"},
        "description": {"S": "Premium handloom bedsheets with custom sizing"},
        "parent_id": {"S": "cat_bedding"},
        "level": {"N": "2"},
        "path": {"S": "cat_home_textiles/cat_bedding/cat_bedsheets"},
        "ancestor_ids": {"L": [{"S": "cat_home_textiles"}, {"S": "cat_bedding"}]},
        "status": {"S": "ACTIVE"},
        "allow_custom_dimensions": {"BOOL": true},
        "dimension_config": {"M": {
            "length_enabled": {"BOOL": true},
            "length_min": {"N": "60"},
            "length_max": {"N": "120"},
            "length_step": {"N": "1"},
            "length_unit": {"S": "inches"},
            "width_enabled": {"BOOL": true},
            "width_min": {"N": "40"},
            "width_max": {"N": "108"},
            "width_step": {"N": "1"},
            "width_unit": {"S": "inches"},
            "height_enabled": {"BOOL": false},
            "pricing_model": {"S": "AREA_BASED"}
        }},
        "default_pricing_rule_id": {"S": "rule_bedsheets_area"},
        "product_count": {"N": "0"},
        "design_count": {"N": "0"},
        "own_attributes": {"L": [
            {"M": {
                "name": {"S": "bed_size"},
                "label": {"S": "Bed Size"},
                "type": {"S": "SELECT"},
                "required": {"BOOL": true},
                "affects_pricing": {"BOOL": true},
                "options": {"L": [
                    {"M": {"value": {"S": "single"}, "label": {"S": "Single (36x75)"}, "surcharge": {"N": "0"}}},
                    {"M": {"value": {"S": "double"}, "label": {"S": "Double (54x75)"}, "surcharge": {"N": "0"}}},
                    {"M": {"value": {"S": "queen"}, "label": {"S": "Queen (60x80)"}, "surcharge": {"N": "10000"}}},
                    {"M": {"value": {"S": "king"}, "label": {"S": "King (76x80)"}, "surcharge": {"N": "20000"}}},
                    {"M": {"value": {"S": "custom"}, "label": {"S": "Custom Size"}, "surcharge": {"N": "0"}}}
                ]}
            }},
            {"M": {
                "name": {"S": "elastic_type"},
                "label": {"S": "Elastic Type"},
                "type": {"S": "SELECT"},
                "required": {"BOOL": true},
                "affects_pricing": {"BOOL": true},
                "options": {"L": [
                    {"M": {"value": {"S": "flat"}, "label": {"S": "Flat Sheet"}, "surcharge": {"N": "0"}}},
                    {"M": {"value": {"S": "fitted"}, "label": {"S": "Fitted Sheet"}, "surcharge": {"N": "15000"}}},
                    {"M": {"value": {"S": "elasticated"}, "label": {"S": "Elasticated Corners"}, "surcharge": {"N": "10000"}}}
                ]}
            }}
        ]},
        "created_at": {"S": "2024-01-15T10:00:00Z"},
        "updated_at": {"S": "2024-01-15T10:00:00Z"}
    }' \
    --endpoint-url $ENDPOINT \
    --region $REGION

# Create global pricing rule
echo "Creating global pricing rule..."
$AWS_CMD dynamodb put-item \
    --table-name $TABLE \
    --item '{
        "PK": {"S": "PRICING_RULE#rule_global_default"},
        "SK": {"S": "METADATA"},
        "GSI1PK": {"S": "SCOPE#GLOBAL"},
        "GSI1SK": {"S": "GLOBAL"},
        "entity_type": {"S": "PRICING_RULE"},
        "id": {"S": "rule_global_default"},
        "name": {"S": "Global Default Pricing"},
        "description": {"S": "Default pricing rule for all products"},
        "scope_type": {"S": "GLOBAL"},
        "pricing_type": {"S": "AREA_BASED"},
        "base_price": {"N": "50000"},
        "price_per_unit": {"N": "30"},
        "unit": {"S": "SQ_INCH"},
        "material_multipliers": {"M": {
            "cotton": {"N": "1.0"},
            "silk": {"N": "2.5"},
            "linen": {"N": "1.8"},
            "blend": {"N": "1.5"}
        }},
        "priority": {"N": "1"},
        "is_active": {"BOOL": true},
        "min_order_value": {"N": "100000"},
        "created_at": {"S": "2024-01-15T10:00:00Z"},
        "updated_at": {"S": "2024-01-15T10:00:00Z"}
    }' \
    --endpoint-url $ENDPOINT \
    --region $REGION

# Create bedsheets pricing rule
echo "Creating bedsheets pricing rule..."
$AWS_CMD dynamodb put-item \
    --table-name $TABLE \
    --item '{
        "PK": {"S": "PRICING_RULE#rule_bedsheets_area"},
        "SK": {"S": "METADATA"},
        "GSI1PK": {"S": "SCOPE#CATEGORY"},
        "GSI1SK": {"S": "cat_bedsheets"},
        "entity_type": {"S": "PRICING_RULE"},
        "id": {"S": "rule_bedsheets_area"},
        "name": {"S": "Bedsheets Area Pricing"},
        "description": {"S": "Area-based pricing for bedsheets with material multipliers"},
        "scope_type": {"S": "CATEGORY"},
        "scope_id": {"S": "cat_bedsheets"},
        "category_id": {"S": "cat_bedsheets"},
        "pricing_type": {"S": "AREA_BASED"},
        "base_price": {"N": "50000"},
        "price_per_unit": {"N": "35"},
        "unit": {"S": "SQ_INCH"},
        "material_multipliers": {"M": {
            "cotton": {"N": "1.0"},
            "silk": {"N": "2.5"},
            "linen": {"N": "1.8"},
            "blend": {"N": "1.5"}
        }},
        "attribute_surcharges": {"L": [
            {"M": {
                "attribute_name": {"S": "thread_count"},
                "attribute_value": {"S": "400"},
                "surcharge_type": {"S": "FIXED"},
                "surcharge_value": {"N": "30000"}
            }},
            {"M": {
                "attribute_name": {"S": "thread_count"},
                "attribute_value": {"S": "600"},
                "surcharge_type": {"S": "FIXED"},
                "surcharge_value": {"N": "50000"}
            }},
            {"M": {
                "attribute_name": {"S": "elastic_type"},
                "attribute_value": {"S": "fitted"},
                "surcharge_type": {"S": "FIXED"},
                "surcharge_value": {"N": "15000"}
            }},
            {"M": {
                "attribute_name": {"S": "elastic_type"},
                "attribute_value": {"S": "elasticated"},
                "surcharge_type": {"S": "FIXED"},
                "surcharge_value": {"N": "10000"}
            }}
        ]},
        "min_area": {"N": "1000"},
        "max_area": {"N": "15000"},
        "min_order_value": {"N": "100000"},
        "priority": {"N": "100"},
        "is_active": {"BOOL": true},
        "created_at": {"S": "2024-01-15T10:00:00Z"},
        "updated_at": {"S": "2024-01-15T10:00:00Z"}
    }' \
    --endpoint-url $ENDPOINT \
    --region $REGION

# Create Pillow Covers category
echo "Creating category: Pillow Covers..."
$AWS_CMD dynamodb put-item \
    --table-name $TABLE \
    --item '{
        "PK": {"S": "CATEGORY#cat_pillow_covers"},
        "SK": {"S": "METADATA"},
        "GSI1PK": {"S": "CATEGORY#cat_bedding"},
        "GSI1SK": {"S": "CATEGORY#cat_pillow_covers"},
        "entity_type": {"S": "CATEGORY"},
        "id": {"S": "cat_pillow_covers"},
        "name": {"S": "Pillow Covers"},
        "slug": {"S": "pillow-covers"},
        "description": {"S": "Premium handloom pillow covers"},
        "parent_id": {"S": "cat_bedding"},
        "level": {"N": "2"},
        "path": {"S": "cat_home_textiles/cat_bedding/cat_pillow_covers"},
        "ancestor_ids": {"L": [{"S": "cat_home_textiles"}, {"S": "cat_bedding"}]},
        "status": {"S": "ACTIVE"},
        "allow_custom_dimensions": {"BOOL": true},
        "dimension_config": {"M": {
            "length_enabled": {"BOOL": true},
            "length_min": {"N": "16"},
            "length_max": {"N": "36"},
            "length_step": {"N": "1"},
            "length_unit": {"S": "inches"},
            "width_enabled": {"BOOL": true},
            "width_min": {"N": "16"},
            "width_max": {"N": "36"},
            "width_step": {"N": "1"},
            "width_unit": {"S": "inches"},
            "height_enabled": {"BOOL": false},
            "pricing_model": {"S": "AREA_BASED"}
        }},
        "product_count": {"N": "0"},
        "design_count": {"N": "0"},
        "own_attributes": {"L": [
            {"M": {
                "name": {"S": "pillow_size"},
                "label": {"S": "Pillow Size"},
                "type": {"S": "SELECT"},
                "required": {"BOOL": true},
                "affects_pricing": {"BOOL": true},
                "options": {"L": [
                    {"M": {"value": {"S": "standard"}, "label": {"S": "Standard (20x26)"}, "surcharge": {"N": "0"}}},
                    {"M": {"value": {"S": "queen"}, "label": {"S": "Queen (20x30)"}, "surcharge": {"N": "5000"}}},
                    {"M": {"value": {"S": "king"}, "label": {"S": "King (20x36)"}, "surcharge": {"N": "10000"}}},
                    {"M": {"value": {"S": "euro"}, "label": {"S": "Euro Square (26x26)"}, "surcharge": {"N": "8000"}}},
                    {"M": {"value": {"S": "custom"}, "label": {"S": "Custom Size"}, "surcharge": {"N": "0"}}}
                ]}
            }},
            {"M": {
                "name": {"S": "closure_type"},
                "label": {"S": "Closure Type"},
                "type": {"S": "SELECT"},
                "required": {"BOOL": true},
                "affects_pricing": {"BOOL": true},
                "options": {"L": [
                    {"M": {"value": {"S": "envelope"}, "label": {"S": "Envelope Back"}, "surcharge": {"N": "0"}}},
                    {"M": {"value": {"S": "zipper"}, "label": {"S": "Hidden Zipper"}, "surcharge": {"N": "5000"}}},
                    {"M": {"value": {"S": "button"}, "label": {"S": "Button Closure"}, "surcharge": {"N": "7000"}}}
                ]}
            }}
        ]},
        "created_at": {"S": "2024-01-15T10:00:00Z"},
        "updated_at": {"S": "2024-01-15T10:00:00Z"}
    }' \
    --endpoint-url $ENDPOINT \
    --region $REGION

echo ""
echo "=============================================="
echo "  Test data seeded successfully!"
echo "=============================================="
echo ""
echo "Default Users:"
echo "  Admin:"
echo "    Email: admin@handloom.com"
echo "    Password: Admin@123!"
echo "    Role: admin"
echo ""
echo "  Manager:"
echo "    Email: manager@handloom.com"
echo "    Password: Admin@123!"
echo "    Role: manager"
echo ""
echo "Categories:"
echo "  - Home Textiles (root)"
echo "    - Bedding"
echo "      - Bedsheets (with custom dimensions)"
echo "      - Pillow Covers (with custom dimensions)"
echo ""
echo "Pricing Rules:"
echo "  - Global Default Pricing"
echo "  - Bedsheets Area Pricing"
echo ""
echo "=============================================="
echo "IMPORTANT: Change default passwords in production!"
echo "=============================================="
