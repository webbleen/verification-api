#!/bin/bash

echo "🚀 Starting Verification Service..."

# Check if Redis is running
if ! redis-cli ping > /dev/null 2>&1; then
    echo "❌ Redis is not running, please start Redis first"
    echo "💡 You can use: docker run -d --name redis -p 6379:6379 redis:7-alpine"
    exit 1
fi

echo "✅ Redis connection is healthy"

# Check environment variables
if [ -z "$BREVO_API_KEY" ]; then
    echo "⚠️  Warning: BREVO_API_KEY is not set, email sending will be unavailable"
fi

if [ -z "$BREVO_FROM_EMAIL" ]; then
    echo "⚠️  Warning: BREVO_FROM_EMAIL is not set, email sending will be unavailable"
fi

# Start service
echo "🎯 Starting service on port 8080..."
go run cmd/server/main.go
