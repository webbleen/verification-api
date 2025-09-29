#!/bin/bash

echo "🧪 Testing Verification Service API (Multi-project Support)..."

API_BASE_URL="http://localhost:8082"
TEST_EMAIL="test@example.com"
TEST_CODE="123456"
PROJECT_ID="default"
API_KEY="default-api-key"

echo "📧 Test Email: $TEST_EMAIL"
echo "🔢 Test Code: $TEST_CODE"
echo "🏢 Project ID: $PROJECT_ID"
echo "🔑 API Key: $API_KEY"
echo ""

# 1. Health check
echo "1️⃣ Health Check..."
curl -s -X GET "$API_BASE_URL/health" | jq .
echo ""

# 2. Get project list
echo "2️⃣ Get Project List..."
curl -s -X GET "$API_BASE_URL/api/admin/projects" | jq .
echo ""

# 2.1. Get project stats (admin)
echo "2.1️⃣ Get Project Stats (Admin)..."
curl -s -X GET "$API_BASE_URL/api/admin/projects/$PROJECT_ID/stats" | jq .
echo ""

# 3. Send verification code
echo "3️⃣ Send Verification Code..."
SEND_RESPONSE=$(curl -s -X POST "$API_BASE_URL/api/verification/send-code" \
  -H "Content-Type: application/json" \
  -H "X-Project-ID: $PROJECT_ID" \
  -H "X-API-Key: $API_KEY" \
  -d "{\"email\": \"$TEST_EMAIL\", \"project_id\": \"$PROJECT_ID\"}")

echo "$SEND_RESPONSE" | jq .

# Check if successful
SUCCESS=$(echo "$SEND_RESPONSE" | jq -r '.success')
if [ "$SUCCESS" = "true" ]; then
    echo "✅ Verification code sent successfully!"
    echo "📧 Please check email: $TEST_EMAIL"
    echo ""
    echo "💡 Note: Since this is a test environment, actual emails may not be sent"
    echo "   Please check Brevo configuration and API key"
    echo ""
    
    # 4. Verify verification code
    echo "4️⃣ Verify Verification Code..."
    VERIFY_RESPONSE=$(curl -s -X POST "$API_BASE_URL/api/verification/verify-code" \
      -H "Content-Type: application/json" \
      -H "X-Project-ID: $PROJECT_ID" \
      -H "X-API-Key: $API_KEY" \
      -d "{\"email\": \"$TEST_EMAIL\", \"code\": \"$TEST_CODE\", \"project_id\": \"$PROJECT_ID\"}")
    
    echo "$VERIFY_RESPONSE" | jq .
    
    VERIFY_SUCCESS=$(echo "$VERIFY_RESPONSE" | jq -r '.success')
    if [ "$VERIFY_SUCCESS" = "true" ]; then
        echo "✅ Verification code verified successfully!"
        
        # 5. Get verification stats
        echo "5️⃣ Get Verification Stats..."
        curl -s -X GET "$API_BASE_URL/api/stats/verification?days=7" \
          -H "X-Project-ID: $PROJECT_ID" \
          -H "X-API-Key: $API_KEY" | jq .
        echo ""
        
        # 6. Get project stats
        echo "6️⃣ Get Project Stats..."
        curl -s -X GET "$API_BASE_URL/api/stats/project" \
          -H "X-Project-ID: $PROJECT_ID" \
          -H "X-API-Key: $API_KEY" | jq .
        echo ""
    else
        echo "❌ Verification code verification failed"
        echo "💡 Tip: Please use the actual verification code from the email"
    fi
else
    echo "❌ Verification code sending failed"
    echo "Please check if the service is running properly"
fi

echo ""
echo "🔍 Testing completed!"
