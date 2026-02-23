#!/bin/bash

# Script to seed initial data for testing
# Usage: ./scripts/seed-data.sh

ENDPOINT="${DYNAMODB_LOCAL_ENDPOINT:-http://localhost:4566}"
REGION="${AWS_REGION:-ap-south-1}"
TABLE="handloom-core"
POSTGRES_DSN="${POSTGRES_DSN:-postgres://handloom:handloom@localhost:5432/handloom?sslmode=disable}"

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
# Admin User (DynamoDB)
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
        "GSI2SK": {"S": "ADMIN#usr_admin_001"},
        "entity_type": {"S": "USER"},
        "id": {"S": "usr_admin_001"},
        "email": {"S": "admin@handloom.com"},
        "name": {"S": "System Administrator"},
        "password_hash": {"S": "$2a$12$QA9/lw32rWAx6606bYrj4e1qkl9cqzp.MiG7lZS/K7giv4kD5/InC"},
        "role": {"S": "ADMIN"},
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
        "GSI2SK": {"S": "OPERATOR#usr_manager_001"},
        "entity_type": {"S": "USER"},
        "id": {"S": "usr_manager_001"},
        "email": {"S": "manager@handloom.com"},
        "name": {"S": "Store Manager"},
        "password_hash": {"S": "$2a$12$QA9/lw32rWAx6606bYrj4e1qkl9cqzp.MiG7lZS/K7giv4kD5/InC"},
        "role": {"S": "OPERATOR"},
        "status": {"S": "ACTIVE"},
        "email_verified": {"BOOL": true},
        "created_at": {"S": "2024-01-15T10:00:00Z"},
        "updated_at": {"S": "2024-01-15T10:00:00Z"}
    }' \
    --endpoint-url $ENDPOINT \
    --region $REGION 2>/dev/null || echo "Manager user already exists"

# =============================================================================
# Categories (PostgreSQL)
# =============================================================================

echo ""
echo "Seeding categories into PostgreSQL..."

psql "$POSTGRES_DSN" <<'EOSQL'
-- Idempotent: use ON CONFLICT DO NOTHING

-- Home Textiles (root category)
INSERT INTO categories (id, name, slug, description, image_url, status, product_count, created_at, updated_at, created_by)
VALUES ('cat_home_textiles', 'Home Textiles', 'home-textiles', 'Quality handloom home textiles', '', 'ACTIVE', 0, '2024-01-15T10:00:00Z', '2024-01-15T10:00:00Z', 'seed')
ON CONFLICT (id) DO NOTHING;

-- Home Textiles attributes: material, color, weave_type
INSERT INTO category_attributes (id, category_id, name, label, type, required, searchable, display_order)
VALUES
    ('attr_ht_material', 'cat_home_textiles', 'material', 'Material', 'SELECT', true, true, 0),
    ('attr_ht_color', 'cat_home_textiles', 'color', 'Color', 'TEXT', true, true, 1),
    ('attr_ht_weave_type', 'cat_home_textiles', 'weave_type', 'Weave Type', 'SELECT', false, true, 2)
ON CONFLICT (id) DO NOTHING;

INSERT INTO category_attribute_options (id, attribute_id, value, label, sort_order)
VALUES
    ('opt_ht_mat_cotton', 'attr_ht_material', 'cotton', 'Cotton', 0),
    ('opt_ht_mat_silk', 'attr_ht_material', 'silk', 'Silk', 1),
    ('opt_ht_mat_linen', 'attr_ht_material', 'linen', 'Linen', 2),
    ('opt_ht_mat_blend', 'attr_ht_material', 'blend', 'Cotton-Silk Blend', 3),
    ('opt_ht_wt_plain', 'attr_ht_weave_type', 'plain', 'Plain Weave', 0),
    ('opt_ht_wt_twill', 'attr_ht_weave_type', 'twill', 'Twill Weave', 1),
    ('opt_ht_wt_satin', 'attr_ht_weave_type', 'satin', 'Satin Weave', 2),
    ('opt_ht_wt_jacquard', 'attr_ht_weave_type', 'jacquard', 'Jacquard', 3)
ON CONFLICT (id) DO NOTHING;

-- Bedding category
INSERT INTO categories (id, name, slug, description, image_url, status, product_count, created_at, updated_at, created_by)
VALUES ('cat_bedding', 'Bedding', 'bedding', 'Quality bedding products', '', 'ACTIVE', 0, '2024-01-15T10:00:00Z', '2024-01-15T10:00:00Z', 'seed')
ON CONFLICT (id) DO NOTHING;

-- Bedding attributes: thread_count
INSERT INTO category_attributes (id, category_id, name, label, type, required, searchable, display_order)
VALUES
    ('attr_bed_thread_count', 'cat_bedding', 'thread_count', 'Thread Count', 'SELECT', false, true, 0)
ON CONFLICT (id) DO NOTHING;

INSERT INTO category_attribute_options (id, attribute_id, value, label, sort_order)
VALUES
    ('opt_bed_tc_200', 'attr_bed_thread_count', '200', '200 TC', 0),
    ('opt_bed_tc_300', 'attr_bed_thread_count', '300', '300 TC', 1),
    ('opt_bed_tc_400', 'attr_bed_thread_count', '400', '400 TC', 2),
    ('opt_bed_tc_600', 'attr_bed_thread_count', '600', '600 TC', 3)
ON CONFLICT (id) DO NOTHING;

-- Bedsheets category
INSERT INTO categories (id, name, slug, description, image_url, status, product_count, created_at, updated_at, created_by)
VALUES ('cat_bedsheets', 'Bedsheets', 'bedsheets', 'Premium handloom bedsheets with custom sizing', '', 'ACTIVE', 0, '2024-01-15T10:00:00Z', '2024-01-15T10:00:00Z', 'seed')
ON CONFLICT (id) DO NOTHING;

-- Bedsheets attributes: bed_size, elastic_type
INSERT INTO category_attributes (id, category_id, name, label, type, required, searchable, display_order)
VALUES
    ('attr_bs_bed_size', 'cat_bedsheets', 'bed_size', 'Bed Size', 'SELECT', true, true, 0),
    ('attr_bs_elastic_type', 'cat_bedsheets', 'elastic_type', 'Elastic Type', 'SELECT', true, true, 1)
ON CONFLICT (id) DO NOTHING;

INSERT INTO category_attribute_options (id, attribute_id, value, label, sort_order)
VALUES
    ('opt_bs_bs_single', 'attr_bs_bed_size', 'single', 'Single (36x75)', 0),
    ('opt_bs_bs_double', 'attr_bs_bed_size', 'double', 'Double (54x75)', 1),
    ('opt_bs_bs_queen', 'attr_bs_bed_size', 'queen', 'Queen (60x80)', 2),
    ('opt_bs_bs_king', 'attr_bs_bed_size', 'king', 'King (76x80)', 3),
    ('opt_bs_bs_custom', 'attr_bs_bed_size', 'custom', 'Custom Size', 4),
    ('opt_bs_et_flat', 'attr_bs_elastic_type', 'flat', 'Flat Sheet', 0),
    ('opt_bs_et_fitted', 'attr_bs_elastic_type', 'fitted', 'Fitted Sheet', 1),
    ('opt_bs_et_elasticated', 'attr_bs_elastic_type', 'elasticated', 'Elasticated Corners', 2)
ON CONFLICT (id) DO NOTHING;

-- Pillow Covers category
INSERT INTO categories (id, name, slug, description, image_url, status, product_count, created_at, updated_at, created_by)
VALUES ('cat_pillow_covers', 'Pillow Covers', 'pillow-covers', 'Premium handloom pillow covers', '', 'ACTIVE', 0, '2024-01-15T10:00:00Z', '2024-01-15T10:00:00Z', 'seed')
ON CONFLICT (id) DO NOTHING;

-- Pillow Covers attributes: pillow_size, closure_type
INSERT INTO category_attributes (id, category_id, name, label, type, required, searchable, display_order)
VALUES
    ('attr_pc_pillow_size', 'cat_pillow_covers', 'pillow_size', 'Pillow Size', 'SELECT', true, true, 0),
    ('attr_pc_closure_type', 'cat_pillow_covers', 'closure_type', 'Closure Type', 'SELECT', true, true, 1)
ON CONFLICT (id) DO NOTHING;

INSERT INTO category_attribute_options (id, attribute_id, value, label, sort_order)
VALUES
    ('opt_pc_ps_standard', 'attr_pc_pillow_size', 'standard', 'Standard (20x26)', 0),
    ('opt_pc_ps_queen', 'attr_pc_pillow_size', 'queen', 'Queen (20x30)', 1),
    ('opt_pc_ps_king', 'attr_pc_pillow_size', 'king', 'King (20x36)', 2),
    ('opt_pc_ps_euro', 'attr_pc_pillow_size', 'euro', 'Euro Square (26x26)', 3),
    ('opt_pc_ps_custom', 'attr_pc_pillow_size', 'custom', 'Custom Size', 4),
    ('opt_pc_ct_envelope', 'attr_pc_closure_type', 'envelope', 'Envelope Back', 0),
    ('opt_pc_ct_zipper', 'attr_pc_closure_type', 'zipper', 'Hidden Zipper', 1),
    ('opt_pc_ct_button', 'attr_pc_closure_type', 'button', 'Button Closure', 2)
ON CONFLICT (id) DO NOTHING;

EOSQL

echo "Categories seeded into PostgreSQL."

# =============================================================================
# Pricing Rules (DynamoDB — stays in handloom-core)
# =============================================================================

echo ""
echo "Creating pricing rules..."

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
echo "Categories (PostgreSQL):"
echo "  - Home Textiles (root)"
echo "  - Bedding"
echo "  - Bedsheets (with attributes: bed_size, elastic_type)"
echo "  - Pillow Covers (with attributes: pillow_size, closure_type)"
echo ""
echo "Pricing Rules (DynamoDB):"
echo "  - Global Default Pricing"
echo "  - Bedsheets Area Pricing"
echo ""
echo "=============================================="
echo "IMPORTANT: Change default passwords in production!"
echo "=============================================="
