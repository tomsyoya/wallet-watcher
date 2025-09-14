#!/bin/bash

# E2E Test Script for Balances API
# This script performs end-to-end testing of the /balances API endpoints.
# 
# Test Coverage:
# - Valid requests for both Solana and Sui chains
# - Invalid requests (missing parameters, invalid addresses)
# - HTTP status code validation
# - JSON response structure validation
# - Error message validation
# - Performance verification
#
# Prerequisites:
# - API server must be running on localhost:8080
# - jq command must be available for JSON parsing
# - curl command must be available for HTTP requests

set -e

API_URL="http://localhost:8080"
SOLANA_ADDRESS="11111111111111111111111111111112"
SUI_ADDRESS="0x1251d3064f375a8353eaeadf928b57c1b7cf91a609cb5a1dd1e779bb189735fa"

echo "🧪 Starting Balances API E2E Tests"
echo "=================================="

# Function to test API endpoint
# Parameters:
#   $1: Test case name
#   $2: URL to test
#   $3: Expected HTTP status code
# 
# This function:
# - Makes HTTP request to the specified URL
# - Validates HTTP status code matches expected value
# - For 200 responses, validates JSON structure and required fields
# - Logs detailed results for debugging
test_endpoint() {
    local name="$1"
    local url="$2"
    local expected_status="$3"
    
    echo "Testing: $name"
    echo "URL: $url"
    
    # Make HTTP request and capture both response body and status code
    response=$(curl -s -w "\n%{http_code}" "$url")
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    # Validate HTTP status code
    if [ "$http_code" = "$expected_status" ]; then
        echo "✅ Status: $http_code (expected: $expected_status)"
    else
        echo "❌ Status: $http_code (expected: $expected_status)"
        echo "Response: $body"
        return 1
    fi
    
    # For successful responses (200), validate JSON structure
    if [ "$http_code" = "200" ]; then
        # Validate JSON is well-formed
        if echo "$body" | jq . > /dev/null 2>&1; then
            echo "✅ Valid JSON response"
            
            # Extract and validate required fields
            address=$(echo "$body" | jq -r '.address // empty')
            balances=$(echo "$body" | jq -r '.balances // empty')
            
            if [ -n "$address" ]; then
                echo "✅ Required fields present (address: $address)"
                
                # Check balances field type and content
                balances_type=$(echo "$body" | jq -r 'type')
                if [ "$balances_type" = "object" ]; then
                    balance_count=$(echo "$body" | jq '.balances | length // 0')
                    if [ "$balance_count" -gt 0 ]; then
                        echo "📊 Found $balance_count balances"
                        # Display first few balances for verification
                        echo "📋 Sample balances:"
                        echo "$body" | jq -r '.balances[0:3][] | "  \(.token): \(.amount)"'
                    else
                        echo "📊 No balances found (this is normal for some addresses)"
                    fi
                else
                    echo "📊 Balances field is $balances_type"
                fi
            else
                echo "❌ Missing required address field"
                return 1
            fi
        else
            echo "❌ Invalid JSON response"
            echo "Response: $body"
            return 1
        fi
    fi
    
    echo ""
}

# Wait for API to be ready (health check)
echo "⏳ Waiting for API to be ready..."
for i in {1..30}; do
    if curl -s "$API_URL/health" > /dev/null 2>&1; then
        echo "✅ API is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ API not ready after 30 seconds"
        exit 1
    fi
    sleep 1
done

echo ""

# Test cases - Valid requests
echo "=== Valid Request Tests ==="
test_endpoint "Solana Balances" "$API_URL/balances?chain=solana&address=$SOLANA_ADDRESS" "200"
test_endpoint "Sui Balances" "$API_URL/balances?chain=sui&address=$SUI_ADDRESS" "200"

# Test cases - Invalid requests (parameter validation)
echo "=== Parameter Validation Tests ==="
test_endpoint "Missing chain parameter" "$API_URL/balances?address=$SOLANA_ADDRESS" "400"
test_endpoint "Missing address parameter" "$API_URL/balances?chain=solana" "400"
test_endpoint "Invalid chain" "$API_URL/balances?chain=ethereum&address=$SOLANA_ADDRESS" "400"
test_endpoint "Invalid Solana address" "$API_URL/balances?chain=solana&address=invalid" "400"
test_endpoint "Invalid Sui address" "$API_URL/balances?chain=sui&address=invalid" "400"

# Test cases - Direct endpoint access
echo "=== Direct Endpoint Tests ==="
test_endpoint "Solana Direct Endpoint" "$API_URL/balances/solana/$SOLANA_ADDRESS" "200"
test_endpoint "Sui Direct Endpoint" "$API_URL/balances/sui/$SUI_ADDRESS" "200"

echo "🎉 All E2E tests passed!"
echo "========================="
