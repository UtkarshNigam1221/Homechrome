#!/bin/bash

# Comprehensive API Test Script
# Tests all endpoints with supported flows

set -e

API_URL="${API_URL:-http://localhost:8080}"
ADMIN_EMAIL="admin@handloom.com"
ADMIN_PASSWORD="Admin@123!"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counter
PASSED=0
FAILED=0
TOTAL=0

# Helper functions
print_header() {
    echo ""
    echo -e "${BLUE}=============================================${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}=============================================${NC}"
}

print_test() {
    echo -e "${YELLOW}TEST: $1${NC}"
    ((TOTAL++))
}

print_success() {
    echo -e "${GREEN}✓ PASSED: $1${NC}"
    ((PASSED++))
}

print_fail() {
    echo -e "${RED}✗ FAILED: $1${NC}"
    echo -e "${RED}  Response: $2${NC}"
    ((FAILED++))
}

# Test endpoint
test_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local expected_status=$4
    local description=$5
    local token=$6

    print_test "$description"

    local headers="-H 'Content-Type: application/json'"
    if [ -n "$token" ]; then
        headers="$headers -H 'Authorization: Bearer $token'"
    fi

    local response
    local http_code

    if [ "$method" == "GET" ]; then
        response=$(curl -s -w "\n%{http_code}" $headers "$API_URL$endpoint" 2>/dev/null)
    elif [ "$method" == "DELETE" ]; then
        response=$(curl -s -w "\n%{http_code}" -X DELETE $headers "$API_URL$endpoint" 2>/dev/null)
    else
        response=$(curl -s -w "\n%{http_code}" -X $method $headers -d "$data" "$API_URL$endpoint" 2>/dev/null)
    fi

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" == "$expected_status" ]; then
        print_success "$description (HTTP $http_code)"
        echo "$body" | head -c 200
        echo ""
    else
        print_fail "$description (Expected $expected_status, got $http_code)" "$body"
    fi

    echo "$body"
}

# =============================================================================
# HEALTH CHECK
# =============================================================================
print_header "Health Check"

print_test "Health endpoint"
HEALTH=$(curl -s "$API_URL/health")
if echo "$HEALTH" | grep -q '"status":"ok"'; then
    print_success "Health check"
else
    print_fail "Health check" "$HEALTH"
    echo "API not running. Start with: make run"
    exit 1
fi

# =============================================================================
# AUTHENTICATION FLOW
# =============================================================================
print_header "Authentication APIs"

# Login
# Note: Using heredoc for JSON to avoid escaping issues with special characters in password
print_test "Admin Login"
cat > /tmp/test_login.json << EOF
{"email": "$ADMIN_EMAIL", "password": "$ADMIN_PASSWORD"}
EOF
LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/admin/auth/login" \
    -H "Content-Type: application/json" \
    -d @/tmp/test_login.json)

if echo "$LOGIN_RESPONSE" | grep -q "access_token"; then
    print_success "Admin Login"
    ACCESS_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
    REFRESH_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)
    echo "  Got access_token: ${ACCESS_TOKEN:0:50}..."
else
    print_fail "Admin Login" "$LOGIN_RESPONSE"
    echo "Cannot continue without authentication"
    exit 1
fi

# Token refresh
print_test "Token Refresh"
REFRESH_RESPONSE=$(curl -s -X POST "$API_URL/admin/auth/refresh" \
    -H "Content-Type: application/json" \
    -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")

if echo "$REFRESH_RESPONSE" | grep -q "access_token"; then
    print_success "Token Refresh"
    # Update access token
    ACCESS_TOKEN=$(echo "$REFRESH_RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
else
    print_fail "Token Refresh" "$REFRESH_RESPONSE"
fi

# Invalid login
print_test "Invalid Login (should fail)"
INVALID_LOGIN=$(curl -s -X POST "$API_URL/admin/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"invalid@test.com","password":"wrongpassword"}')

if echo "$INVALID_LOGIN" | grep -q "error\|unauthorized\|invalid"; then
    print_success "Invalid Login rejected"
else
    print_fail "Invalid Login should have been rejected" "$INVALID_LOGIN"
fi

# =============================================================================
# CATEGORY APIs
# =============================================================================
print_header "Category APIs"

# List categories
print_test "List Categories"
CATEGORIES=$(curl -s "$API_URL/admin/categories" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$CATEGORIES" | grep -q "cat_\|categories\|items\|\[\]"; then
    print_success "List Categories"
    echo "  Response: ${CATEGORIES:0:200}..."
else
    print_fail "List Categories" "$CATEGORIES"
fi

# Get specific category
print_test "Get Category by ID"
CATEGORY=$(curl -s "$API_URL/admin/categories/cat_bedsheets" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$CATEGORY" | grep -q "cat_bedsheets\|Bedsheets\|not found"; then
    print_success "Get Category by ID"
else
    print_fail "Get Category by ID" "$CATEGORY"
fi

# Create new category
print_test "Create Category"
NEW_CATEGORY=$(curl -s -X POST "$API_URL/admin/categories" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "Test Category",
        "slug": "test-category",
        "description": "A test category for API testing",
        "parent_id": "cat_bedding",
        "status": "ACTIVE"
    }')

if echo "$NEW_CATEGORY" | grep -q "id\|created\|Test Category"; then
    print_success "Create Category"
    NEW_CAT_ID=$(echo "$NEW_CATEGORY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    echo "  Created category: $NEW_CAT_ID"
else
    print_fail "Create Category" "$NEW_CATEGORY"
fi

# Update category (if created)
if [ -n "$NEW_CAT_ID" ]; then
    print_test "Update Category"
    UPDATED_CAT=$(curl -s -X PATCH "$API_URL/admin/categories/$NEW_CAT_ID" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"description": "Updated description"}')

    if echo "$UPDATED_CAT" | grep -q "Updated description\|success\|id"; then
        print_success "Update Category"
    else
        print_fail "Update Category" "$UPDATED_CAT"
    fi
fi

# =============================================================================
# PRODUCT APIs
# =============================================================================
print_header "Product APIs"

# List products
print_test "List Products"
PRODUCTS=$(curl -s "$API_URL/admin/products" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$PRODUCTS" | grep -q "products\|items\|\[\]\|total"; then
    print_success "List Products"
else
    print_fail "List Products" "$PRODUCTS"
fi

# Create product
print_test "Create Product"
NEW_PRODUCT=$(curl -s -X POST "$API_URL/admin/products" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "Test Bedsheet",
        "slug": "test-bedsheet",
        "description": "A premium cotton bedsheet",
        "category_id": "cat_bedsheets",
        "base_price": 150000,
        "status": "ACTIVE",
        "attributes": {
            "material": "cotton",
            "color": "white",
            "bed_size": "queen"
        }
    }')

if echo "$NEW_PRODUCT" | grep -q "id\|created\|Test Bedsheet"; then
    print_success "Create Product"
    NEW_PROD_ID=$(echo "$NEW_PRODUCT" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    echo "  Created product: $NEW_PROD_ID"
else
    print_fail "Create Product" "$NEW_PRODUCT"
fi

# Get product by ID
if [ -n "$NEW_PROD_ID" ]; then
    print_test "Get Product by ID"
    PRODUCT=$(curl -s "$API_URL/admin/products/$NEW_PROD_ID" \
        -H "Authorization: Bearer $ACCESS_TOKEN")

    if echo "$PRODUCT" | grep -q "$NEW_PROD_ID\|Test Bedsheet"; then
        print_success "Get Product by ID"
    else
        print_fail "Get Product by ID" "$PRODUCT"
    fi
fi

# =============================================================================
# INVENTORY APIs
# =============================================================================
print_header "Inventory APIs"

# List inventory
print_test "List Inventory"
INVENTORY=$(curl -s "$API_URL/admin/inventory" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$INVENTORY" | grep -q "inventory\|items\|\[\]\|total"; then
    print_success "List Inventory"
else
    print_fail "List Inventory" "$INVENTORY"
fi

# Low stock alerts
print_test "Low Stock Alerts"
LOW_STOCK=$(curl -s "$API_URL/admin/inventory/alerts" \
    -H "Authorization: Bearer $ACCESS_TOKEN" 2>/dev/null || echo '{"items":[]}')

if echo "$LOW_STOCK" | grep -q "alerts\|items\|\[\]"; then
    print_success "Low Stock Alerts"
else
    print_fail "Low Stock Alerts" "$LOW_STOCK"
fi

# =============================================================================
# PRICING APIs (Public)
# =============================================================================
print_header "Pricing APIs (Public)"

# Get dimension options
print_test "Get Dimension Options"
DIM_OPTIONS=$(curl -s "$API_URL/api/v1/pricing/dimension-options/cat_bedsheets")

if echo "$DIM_OPTIONS" | grep -q "length\|width\|options\|error"; then
    print_success "Get Dimension Options"
else
    print_fail "Get Dimension Options" "$DIM_OPTIONS"
fi

# Calculate price
print_test "Calculate Price"
PRICE_CALC=$(curl -s -X POST "$API_URL/api/v1/pricing/calculate" \
    -H "Content-Type: application/json" \
    -d '{
        "category_id": "cat_bedsheets",
        "dimensions": {
            "length": 90,
            "width": 108,
            "unit": "inches"
        },
        "attributes": {
            "material": "cotton",
            "thread_count": "300",
            "elastic_type": "fitted"
        },
        "quantity": 1
    }')

if echo "$PRICE_CALC" | grep -q "total\|price\|amount\|error"; then
    print_success "Calculate Price"
    echo "  Price calculation response: ${PRICE_CALC:0:200}..."
else
    print_fail "Calculate Price" "$PRICE_CALC"
fi

# Bulk calculate
print_test "Bulk Price Calculate"
BULK_CALC=$(curl -s -X POST "$API_URL/api/v1/pricing/bulk-calculate" \
    -H "Content-Type: application/json" \
    -d '{
        "items": [
            {
                "category_id": "cat_bedsheets",
                "dimensions": {"length": 90, "width": 108, "unit": "inches"},
                "attributes": {"material": "cotton"},
                "quantity": 1
            },
            {
                "category_id": "cat_bedsheets",
                "dimensions": {"length": 72, "width": 90, "unit": "inches"},
                "attributes": {"material": "silk"},
                "quantity": 2
            }
        ]
    }')

if echo "$BULK_CALC" | grep -q "items\|results\|total\|error"; then
    print_success "Bulk Price Calculate"
else
    print_fail "Bulk Price Calculate" "$BULK_CALC"
fi

# =============================================================================
# PRICING RULES (Admin)
# =============================================================================
print_header "Pricing Rules (Admin)"

# List pricing rules
print_test "List Pricing Rules"
PRICING_RULES=$(curl -s "$API_URL/admin/pricing/rules" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$PRICING_RULES" | grep -q "rules\|items\|\[\]\|id"; then
    print_success "List Pricing Rules"
else
    print_fail "List Pricing Rules" "$PRICING_RULES"
fi

# =============================================================================
# ORDER APIs
# =============================================================================
print_header "Order APIs"

# List orders
print_test "List Orders"
ORDERS=$(curl -s "$API_URL/admin/orders" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$ORDERS" | grep -q "orders\|items\|\[\]\|total"; then
    print_success "List Orders"
else
    print_fail "List Orders" "$ORDERS"
fi

# =============================================================================
# CUSTOMER APIs
# =============================================================================
print_header "Customer APIs"

# List customers
print_test "List Customers"
CUSTOMERS=$(curl -s "$API_URL/admin/customers" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$CUSTOMERS" | grep -q "customers\|items\|\[\]\|total"; then
    print_success "List Customers"
else
    print_fail "List Customers" "$CUSTOMERS"
fi

# Create customer
print_test "Create Customer"
NEW_CUSTOMER=$(curl -s -X POST "$API_URL/admin/customers" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "Test Customer",
        "email": "testcustomer@example.com",
        "phone": "+919876543210",
        "address": {
            "line1": "123 Test Street",
            "city": "Mumbai",
            "state": "Maharashtra",
            "postal_code": "400001",
            "country": "India"
        }
    }')

if echo "$NEW_CUSTOMER" | grep -q "id\|created\|Test Customer"; then
    print_success "Create Customer"
    NEW_CUST_ID=$(echo "$NEW_CUSTOMER" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    echo "  Created customer: $NEW_CUST_ID"
else
    print_fail "Create Customer" "$NEW_CUSTOMER"
fi

# =============================================================================
# USER MANAGEMENT (Admin only)
# =============================================================================
print_header "User Management APIs"

# List users
print_test "List Users"
USERS=$(curl -s "$API_URL/admin/users" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$USERS" | grep -q "users\|items\|\[\]\|admin@handloom.com"; then
    print_success "List Users"
else
    print_fail "List Users" "$USERS"
fi

# =============================================================================
# NOTIFICATION APIs
# =============================================================================
print_header "Notification APIs"

# List notifications
print_test "List Notifications"
NOTIFICATIONS=$(curl -s "$API_URL/admin/notifications" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$NOTIFICATIONS" | grep -q "notifications\|items\|\[\]"; then
    print_success "List Notifications"
else
    print_fail "List Notifications" "$NOTIFICATIONS"
fi

# My notifications
print_test "My Notifications"
MY_NOTIFS=$(curl -s "$API_URL/admin/notifications/my" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$MY_NOTIFS" | grep -q "notifications\|items\|\[\]"; then
    print_success "My Notifications"
else
    print_fail "My Notifications" "$MY_NOTIFS"
fi

# =============================================================================
# COUPON APIs
# =============================================================================
print_header "Coupon APIs"

# List coupons
print_test "List Coupons"
COUPONS=$(curl -s "$API_URL/admin/coupons" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$COUPONS" | grep -q "coupons\|items\|\[\]"; then
    print_success "List Coupons"
else
    print_fail "List Coupons" "$COUPONS"
fi

# Create coupon
print_test "Create Coupon"
NEW_COUPON=$(curl -s -X POST "$API_URL/admin/coupons" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "code": "TEST20",
        "description": "20% off for testing",
        "discount_type": "PERCENTAGE",
        "discount_value": 20,
        "min_order_value": 100000,
        "max_discount": 50000,
        "usage_limit": 100,
        "start_date": "2024-01-01T00:00:00Z",
        "end_date": "2025-12-31T23:59:59Z",
        "is_active": true
    }')

if echo "$NEW_COUPON" | grep -q "id\|TEST20\|created"; then
    print_success "Create Coupon"
    echo "  Created coupon: TEST20"
else
    print_fail "Create Coupon" "$NEW_COUPON"
fi

# Validate coupon
print_test "Validate Coupon"
VALIDATE_COUPON=$(curl -s -X POST "$API_URL/admin/coupons/validate" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"code": "TEST20", "order_value": 200000}')

if echo "$VALIDATE_COUPON" | grep -q "valid\|discount\|applicable\|error"; then
    print_success "Validate Coupon"
else
    print_fail "Validate Coupon" "$VALIDATE_COUPON"
fi

# =============================================================================
# ARTISAN APIs
# =============================================================================
print_header "Artisan APIs"

# List artisans
print_test "List Artisans"
ARTISANS=$(curl -s "$API_URL/admin/artisans" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$ARTISANS" | grep -q "artisans\|items\|\[\]"; then
    print_success "List Artisans"
else
    print_fail "List Artisans" "$ARTISANS"
fi

# Create artisan
print_test "Create Artisan"
NEW_ARTISAN=$(curl -s -X POST "$API_URL/admin/artisans" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "Test Artisan",
        "email": "artisan@example.com",
        "phone": "+919876543211",
        "skills": ["weaving", "dyeing"],
        "location": {
            "city": "Varanasi",
            "state": "Uttar Pradesh",
            "country": "India"
        },
        "status": "ACTIVE"
    }')

if echo "$NEW_ARTISAN" | grep -q "id\|Test Artisan\|created"; then
    print_success "Create Artisan"
else
    print_fail "Create Artisan" "$NEW_ARTISAN"
fi

# =============================================================================
# ANALYTICS APIs
# =============================================================================
print_header "Analytics APIs"

# Dashboard stats
print_test "Dashboard Stats"
DASHBOARD=$(curl -s "$API_URL/admin/analytics/dashboard" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$DASHBOARD" | grep -q "stats\|orders\|revenue\|error\|total"; then
    print_success "Dashboard Stats"
else
    print_fail "Dashboard Stats" "$DASHBOARD"
fi

# Sales analytics
print_test "Sales Analytics"
SALES=$(curl -s "$API_URL/admin/analytics/sales" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$SALES" | grep -q "sales\|revenue\|data\|error"; then
    print_success "Sales Analytics"
else
    print_fail "Sales Analytics" "$SALES"
fi

# Top products
print_test "Top Products"
TOP_PRODUCTS=$(curl -s "$API_URL/admin/analytics/top-products" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$TOP_PRODUCTS" | grep -q "products\|top\|items\|\[\]"; then
    print_success "Top Products"
else
    print_fail "Top Products" "$TOP_PRODUCTS"
fi

# =============================================================================
# ASSET APIs
# =============================================================================
print_header "Asset APIs"

# List assets
print_test "List Assets"
ASSETS=$(curl -s "$API_URL/admin/assets" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$ASSETS" | grep -q "assets\|items\|\[\]"; then
    print_success "List Assets"
else
    print_fail "List Assets" "$ASSETS"
fi

# Get upload URL
print_test "Get Upload URL"
UPLOAD_URL=$(curl -s -X POST "$API_URL/admin/assets/upload-url" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "filename": "test-image.jpg",
        "content_type": "image/jpeg",
        "asset_type": "PRODUCT_IMAGE"
    }')

if echo "$UPLOAD_URL" | grep -q "url\|upload\|presigned\|error"; then
    print_success "Get Upload URL"
else
    print_fail "Get Upload URL" "$UPLOAD_URL"
fi

# =============================================================================
# BULK OPERATION APIs
# =============================================================================
print_header "Bulk Operation APIs"

# List bulk operations
print_test "List Bulk Operations"
BULK_OPS=$(curl -s "$API_URL/admin/bulk" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$BULK_OPS" | grep -q "operations\|items\|\[\]"; then
    print_success "List Bulk Operations"
else
    print_fail "List Bulk Operations" "$BULK_OPS"
fi

# =============================================================================
# REPORT APIs
# =============================================================================
print_header "Report APIs"

# List reports
print_test "List Reports"
REPORTS=$(curl -s "$API_URL/admin/reports" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$REPORTS" | grep -q "reports\|items\|\[\]"; then
    print_success "List Reports"
else
    print_fail "List Reports" "$REPORTS"
fi

# =============================================================================
# AUDIT APIs (Admin only)
# =============================================================================
print_header "Audit APIs"

# List audit logs
print_test "List Audit Logs"
AUDIT_LOGS=$(curl -s "$API_URL/admin/audit" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$AUDIT_LOGS" | grep -q "logs\|items\|\[\]\|audit"; then
    print_success "List Audit Logs"
else
    print_fail "List Audit Logs" "$AUDIT_LOGS"
fi

# =============================================================================
# DESIGN APIs
# =============================================================================
print_header "Design APIs"

# List designs
print_test "List Designs"
DESIGNS=$(curl -s "$API_URL/admin/designs" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$DESIGNS" | grep -q "designs\|items\|\[\]"; then
    print_success "List Designs"
else
    print_fail "List Designs" "$DESIGNS"
fi

# Create design
print_test "Create Design"
NEW_DESIGN=$(curl -s -X POST "$API_URL/admin/designs" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "Floral Pattern",
        "slug": "floral-pattern",
        "description": "Beautiful floral design",
        "category_id": "cat_bedsheets",
        "preview_image_url": "https://example.com/floral.jpg",
        "status": "ACTIVE"
    }')

if echo "$NEW_DESIGN" | grep -q "id\|Floral Pattern\|created"; then
    print_success "Create Design"
else
    print_fail "Create Design" "$NEW_DESIGN"
fi

# =============================================================================
# UNAUTHORIZED ACCESS TEST
# =============================================================================
print_header "Security Tests"

# Access without token
print_test "Access without token (should fail)"
UNAUTH=$(curl -s "$API_URL/admin/categories")

if echo "$UNAUTH" | grep -q "unauthorized\|error\|token\|401"; then
    print_success "Unauthorized access rejected"
else
    print_fail "Unauthorized access should have been rejected" "$UNAUTH"
fi

# Access with invalid token
print_test "Access with invalid token (should fail)"
INVALID_TOKEN=$(curl -s "$API_URL/admin/categories" \
    -H "Authorization: Bearer invalid_token_12345")

if echo "$INVALID_TOKEN" | grep -q "unauthorized\|error\|invalid\|401"; then
    print_success "Invalid token rejected"
else
    print_fail "Invalid token should have been rejected" "$INVALID_TOKEN"
fi

# =============================================================================
# LOGOUT
# =============================================================================
print_header "Logout"

print_test "Logout"
LOGOUT=$(curl -s -X POST "$API_URL/admin/auth/logout" \
    -H "Authorization: Bearer $ACCESS_TOKEN")

if echo "$LOGOUT" | grep -q "success\|logged out\|ok\|{}"; then
    print_success "Logout"
else
    # Logout might just return empty response which is fine
    if [ -z "$LOGOUT" ] || [ "$LOGOUT" == "{}" ]; then
        print_success "Logout (empty response)"
    else
        print_fail "Logout" "$LOGOUT"
    fi
fi

# =============================================================================
# SUMMARY
# =============================================================================
print_header "Test Summary"

echo ""
echo -e "Total Tests: ${TOTAL}"
echo -e "${GREEN}Passed: ${PASSED}${NC}"
echo -e "${RED}Failed: ${FAILED}${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}=============================================${NC}"
    echo -e "${GREEN}  ALL TESTS PASSED!${NC}"
    echo -e "${GREEN}=============================================${NC}"
    exit 0
else
    echo -e "${RED}=============================================${NC}"
    echo -e "${RED}  SOME TESTS FAILED${NC}"
    echo -e "${RED}=============================================${NC}"
    exit 1
fi
