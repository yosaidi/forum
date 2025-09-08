#!/bin/bash

BASE_URL="http://localhost:8080"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🚀 Rate Limiter Stress Test"
echo "==========================="

# Function to make requests and count responses
test_endpoint() {
    local endpoint=$1
    local method=$2
    local requests=$3
    local data=$4
    local test_name=$5
    
    echo ""
    echo "🔥 Testing: $test_name"
    echo "Endpoint: $method $endpoint"
    echo "Sending $requests requests..."
    
    success=0
    rate_limited=0
    errors=0
    
    for i in $(seq 1 $requests); do
        if [ "$method" = "POST" ]; then
            response=$(curl -s -w "%{http_code}" -X POST \
                -H "Content-Type: application/json" \
                -d "$data" \
                "$BASE_URL$endpoint" -o /dev/null)
        else
            response=$(curl -s -w "%{http_code}" \
                "$BASE_URL$endpoint" -o /dev/null)
        fi
        
        case $response in
            200|201|401|404)
                ((success++))
                ;;
            429)
                ((rate_limited++))
                printf "${RED}.${NC}"
                ;;
            *)
                ((errors++))
                printf "${YELLOW}?${NC}"
                ;;
        esac
        
        # Small delay to avoid overwhelming
        sleep 0.01
    done
    
    echo ""
    echo "Results:"
    printf "  ${GREEN}✅ Success: $success${NC}\n"
    printf "  ${RED}🚫 Rate Limited (429): $rate_limited${NC}\n"
    printf "  ${YELLOW}❌ Errors: $errors${NC}\n"
    
    if [ $rate_limited -gt 0 ]; then
        printf "  ${GREEN}✅ Rate limiting is working!${NC}\n"
    else
        printf "  ${YELLOW}⚠️  No rate limiting detected${NC}\n"
    fi
}

# Test 1: Auth endpoints (should be rate limited quickly)
test_endpoint "/api/auth/login" "POST" 20 '{"username":"test","password":"wrong"}' "Auth Brute Force"

echo ""
echo "Waiting 3 seconds..."
sleep 3

# Test 2: Posts (should allow more requests)
test_endpoint "/api/posts" "GET" 50 "" "Posts Flood"

echo ""
echo "Waiting 3 seconds..."
sleep 3

# Test 3: Comments
test_endpoint "/api/posts/1/comments" "GET" 80 "" "Comments Flood"

echo ""
echo "Waiting 3 seconds..."
sleep 3

# Test 4: Users
test_endpoint "/api/users/1" "GET" 120 "" "User Profile Spam"

echo ""
echo "Waiting 3 seconds..."
sleep 3

# Test 5: Categories (should be most lenient)
test_endpoint "/api/categories" "GET" 250 "" "Categories Flood"

echo ""
echo "=========================================="
echo "🎯 RAPID FIRE TEST (Concurrent)"
echo "=========================================="

# Rapid fire test with background processes
echo "Sending 100 rapid auth requests in background..."

for i in {1..100}; do
    curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{"username":"rapid'$i'","password":"test"}' \
        "$BASE_URL/api/auth/login" \
        -w "%{http_code}\n" -o /dev/null &
done

# Wait for all background jobs
wait

echo ""
echo "✅ Stress test completed!"
echo "Check your server logs to see rate limiting in action."

# Quick verification
echo ""
echo "🔍 Quick verification - sending 5 more auth requests:"
for i in {1..5}; do
    response=$(curl -s -w "%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d '{"username":"verify","password":"test"}' \
        "$BASE_URL/api/auth/login" -o /dev/null)
    
    if [ $response -eq 429 ]; then
        printf "${RED}$response ${NC}"
    else
        printf "${GREEN}$response ${NC}"
    fi
done
echo ""