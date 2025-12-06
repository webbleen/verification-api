# UnionHub

**UnionHub** - A unified verification and subscription service center providing email verification code functionality and subscription management for iOS and Android apps. Built with Gin framework, supporting multi-project deployment with centralized subscription verification and App Store Server Notifications handling.

## Features

### Email Verification
- 📧 **Code Sending** - Send 6-digit verification codes to specified email addresses
- ✅ **Code Verification** - Validate user-input verification codes
- 🚀 **Redis Caching** - Efficient verification code storage and management
- ⚡ **Rate Limiting** - Prevent verification code abuse (1-minute cooldown)
- 🔒 **Security** - 5-minute code expiration, one-time use
- 🌍 **Multi-Language Support** - Email content in 8 languages (EN, ZH-CN, ZH-TW, JA, KO, ES, FR, DE)

### Subscription Center
- 🍎 **iOS Subscription** - Verify App Store receipts using App Store Server API (JWT-based)
- 🤖 **Android Subscription** - Verify Google Play purchases using Google Play Developer API
- 🔄 **Auto-Renewal** - Automatic subscription status updates via webhooks
- 📱 **Multi-App Support** - Support multiple iOS/Android apps under one developer account
- 🔐 **Unified Status** - Single source of truth for subscription status
- 🌐 **Production & Sandbox** - Unified webhook endpoint automatically handles both environments
- 🔗 **Account Binding** - Bind user_id to subscriptions when webhook arrives first
- 📜 **History Tracking** - Complete subscription history for audit and analytics
- 🔁 **Restore Purchases** - Support for purchase restoration

### Infrastructure
- 🏢 **Multi-Project Support** - Support multiple projects with data isolation
- 🔐 **Project Authentication** - API key-based project identity verification
- 📊 **Project Management** - Full CRUD operations for project management
- 🗄️ **Database Support** - PostgreSQL + Redis dual storage architecture
- 📈 **Statistics & Monitoring** - Comprehensive analytics and monitoring
- 🐳 **Docker Support** - Containerized deployment

## Quick Start

### 1. Environment Setup

```bash
# Clone the project
git clone https://github.com/webbleen/verification-api.git
cd verification-api

# Install dependencies
go mod tidy
```

### 2. Configure Environment Variables

```bash
# Copy environment template
cp env.example .env

# Edit configuration file
vim .env
```

### 3. Start Database and Redis

```bash
# PostgreSQL
docker run -d --name postgres -p 5432:5432 -e POSTGRES_DB=verification_api -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=password postgres:15-alpine

# Redis
docker run -d --name redis -p 6379:6379 redis:7-alpine
```

### 4. Run Service

```bash
# Development mode
go run cmd/server/main.go

# Or use the start script
chmod +x start.sh
./start.sh
```

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | Server port | `8080` | No |
| `GIN_MODE` | Gin mode (debug/release) | `debug` | No |
| `DATABASE_URL` | PostgreSQL connection URL | - | Yes (production) |
| `REDIS_URL` | Redis connection URL | `redis://localhost:6379/0` | Yes |
| `BREVO_API_KEY` | Brevo API key | - | Yes |
| `BREVO_FROM_EMAIL` | Sender email address | - | Yes |
| `CODE_EXPIRE_MINUTES` | Code expiration time (minutes) | `5` | No |
| `RATE_LIMIT_MINUTES` | Rate limit cooldown (minutes) | `1` | No |
| `SERVICE_NAME` | Service name | `UnionHub` | No |
| `AUTO_MIGRATE` | Enable automatic database migration | `true` | No |
| `APPSTORE_KEY_ID` | App Store Connect API Key ID | - | No (for subscriptions) |
| `APPSTORE_ISSUER_ID` | App Store Connect Issuer ID | - | No (for subscriptions) |
| `APPSTORE_BUNDLE_ID` | App Store Bundle ID | - | No (for subscriptions) |
| `APPSTORE_ENVIRONMENT` | App Store environment (sandbox/production) | `sandbox` | No |
| `APPSTORE_PRIVATE_KEY_PATH` | Path to App Store private key file | - | No (for subscriptions) |
| `APPSTORE_PRIVATE_KEY` | App Store private key content (base64) | - | No (for subscriptions) |
| `APPSTORE_SHARED_SECRET` | App Store shared secret | - | No (for subscriptions) |

### Database Configuration

The service supports both PostgreSQL (production) and SQLite (development):

- **Production**: Set `DATABASE_URL` to your PostgreSQL connection string
- **Development**: Leave `DATABASE_URL` empty to use SQLite

### Multi-Language Support

The service supports 8 languages for email content:

- **English (en)** - Default language
- **Chinese Simplified (zh-CN)** - 简体中文
- **Chinese Traditional (zh-TW)** - 繁體中文
- **Japanese (ja)** - 日本語
- **Korean (ko)** - 한국어
- **Spanish (es)** - Español
- **French (fr)** - Français
- **German (de)** - Deutsch

Specify the language in the `language` field when sending verification codes.

### Brevo Configuration

The service uses Brevo (formerly Sendinblue) for email delivery:

- **Free Tier**: Supports single sender email address
- **From Email**: Configured globally via `BREVO_FROM_EMAIL` environment variable
- **From Name**: Customized per project via `from_name` field in project configuration
- **API Key**: Required for authentication

**Note**: Due to Brevo's free tier limitation, all projects must use the same sender email address, but each project can have its own sender name configured in the database.

### App Store Configuration

For subscription functionality, configure App Store Connect API credentials:

1. **Create App Store Connect API Key**:
   - Go to App Store Connect → Users and Access → Keys
   - Create a new key with "App Manager" or "Admin" role
   - Download the `.p8` private key file

2. **Configure Environment Variables**:
   ```bash
   APPSTORE_KEY_ID=ABC123XYZ        # Key ID from App Store Connect
   APPSTORE_ISSUER_ID=12345678-1234-1234-1234-123456789012  # Issuer ID
   APPSTORE_BUNDLE_ID=com.example.app  # Your app's bundle ID
   APPSTORE_ENVIRONMENT=sandbox      # or "production"
   APPSTORE_PRIVATE_KEY_PATH=/path/to/AuthKey_ABC123XYZ.p8  # Path to .p8 file
   # OR
   APPSTORE_PRIVATE_KEY=LS0tLS1CRUdJTi...  # Base64 encoded private key content
   APPSTORE_SHARED_SECRET=your-shared-secret  # Optional, for receipt validation
   ```

3. **Configure Webhook URLs in App Store Connect**:
   - **Recommended**: `https://your-domain.com/webhook/apple` (unified endpoint)
   - **Legacy**: 
     - Production: `https://your-domain.com/api/appstore/notifications/production`
     - Sandbox: `https://your-domain.com/api/appstore/notifications/sandbox`

## API Documentation

### Authentication

Most API endpoints require project authentication using headers:

```bash
X-Project-ID: your-project-id
X-API-Key: your-api-key
```

**Note**: Subscription endpoints (`/api/subscription/*`) can be called without authentication by clients, but app backends should use authentication headers when querying subscription status.

### Verification Endpoints

#### Send Verification Code

```http
POST /api/verification/send-code
Content-Type: application/json
X-Project-ID: your-project-id
X-API-Key: your-api-key

{
  "email": "user@example.com",
  "project_id": "your-project-id",
  "language": "en"
}
```

**Response:**

```json
{
  "success": true,
  "message": "Verification code sent successfully"
}
```

#### Verify Code

```http
POST /api/verification/verify-code
Content-Type: application/json
X-Project-ID: your-project-id
X-API-Key: your-api-key

{
  "email": "user@example.com",
  "code": "123456",
  "project_id": "your-project-id"
}
```

**Response:**

```json
{
  "success": true,
  "message": "Verification code verified successfully"
}
```

### Project Management Endpoints

#### Get All Projects

```http
GET /api/admin/projects
```

#### Create Project

```http
POST /api/admin/projects
Content-Type: application/json

{
  "project_id": "my-project",
  "project_name": "My Project",
  "api_key": "my-api-key",
  "from_name": "My Project Service",
  "description": "Project description",
  "rate_limit": 60,
  "max_requests": 1000,
  "bundle_id": "com.example.app",
  "package_name": "com.example.app"
}
```

**Note**: 
- `bundle_id` is required for iOS app identification
- `package_name` is required for Android app identification
- Both can be the same value if iOS and Android use the same package identifier

#### Update Project

```http
PUT /api/admin/projects/{project_id}
Content-Type: application/json

{
  "project_name": "Updated Project Name",
  "from_name": "Updated Project Service",
  "is_active": true
}
```

#### Delete Project

```http
DELETE /api/admin/projects/{project_id}
```

### Statistics Endpoints

#### Get Verification Statistics

```http
GET /api/stats/verification?days=7
X-Project-ID: your-project-id
X-API-Key: your-api-key
```

#### Get Project Statistics

```http
GET /api/stats/project
X-Project-ID: your-project-id
X-API-Key: your-api-key
```

### Subscription Endpoints

#### Verify Subscription (Client)

Verify a subscription receipt/token from iOS or Android app using standardized format:

**iOS Request (Recommended - App Store Server API):**

```http
POST /api/subscription/verify
Content-Type: application/json

{
  "platform": "ios",
  "user_id": "user_123",
  "product_id": "com.example.monthly",
  "signed_transaction": "eyJhbGciOiJFUzI1NiIsIng1YyI6WyJNSUlCU...",
  "transaction_id": "1000000999999",
  "app_id": "com.example.app"
}
```

**iOS Request (Legacy - Receipt Verification):**

```http
POST /api/subscription/verify
Content-Type: application/json

{
  "platform": "ios",
  "user_id": "user_123",
  "receipt_data": "base64_receipt_string",
  "app_id": "com.example.app"
}
```

**Android Request:**

```http
POST /api/subscription/verify
Content-Type: application/json

{
  "platform": "android",
  "user_id": "user_123",
  "product_id": "com.example.monthly",
  "purchase_token": "opaque-token-up-to-150-characters",
  "app_id": "com.example.app"
}
```

**Response:**

```json
{
  "success": true,
  "message": "Subscription verified successfully",
  "is_active": true,
  "platform": "ios",
  "expires_date": "2025-12-31T23:59:59Z",
  "plan": "monthly",
  "product_id": "com.example.monthly",
  "auto_renew": true
}
```

**Note**: 
- iOS: Use `signed_transaction` (JWT) and `transaction_id` for App Store Server API (recommended)
- Android: Use `purchase_token` for Google Play verification
- Legacy `receipt_data` format is still supported for backward compatibility

#### Get Subscription Status

Query subscription status (can be called by clients or app backends):

```http
GET /api/subscription/status?user_id=user_123&app_id=com.example.app&platform=ios
```

**For App Backend (with authentication):**

```http
GET /api/subscription/status?user_id=user_123&app_id=com.example.app&platform=ios
X-Project-ID: your-project-id
X-API-Key: your-api-key
```

**Response:**

```json
{
  "success": true,
  "is_active": true,
  "platform": "ios",
  "status": "active",
  "plan": "monthly",
  "expires_date": "2025-12-31T23:59:59Z",
  "product_id": "com.example.monthly",
  "auto_renew": true
}
```

#### Restore Subscription

Restore purchases for a user:

```http
POST /api/subscription/restore
Content-Type: application/json

{
  "user_id": "user_123",
  "app_id": "com.example.app"
}
```

**Response:**

```json
{
  "success": true,
  "message": "Subscription restored successfully",
  "is_active": true,
  "expires_date": "2025-12-31T23:59:59Z",
  "plan": "monthly",
  "product_id": "com.example.monthly"
}
```

#### Bind Account

Bind user_id to a subscription (useful when webhook arrives before user verification):

```http
POST /api/subscription/bind_account
Content-Type: application/json

{
  "user_id": "user_123",
  "original_transaction_id": "1000000999999"
}
```

**For Android:**

```http
POST /api/subscription/bind_account
Content-Type: application/json

{
  "user_id": "user_123",
  "purchase_token": "opaque-token-up-to-150-characters"
}
```

**Response:**

```json
{
  "success": true,
  "message": "Account bound successfully"
}
```

#### Get Subscription History

Get subscription history for a user:

```http
GET /api/subscription/history?user_id=user_123&app_id=com.example.app&platform=ios
```

**Response:**

```json
{
  "success": true,
  "subscriptions": [
    {
      "id": 1,
      "user_id": "user_123",
      "platform": "ios",
      "plan": "monthly",
      "status": "active",
      "product_id": "com.example.monthly",
      "transaction_id": "1000000999999",
      "original_transaction_id": "1000000999999",
      "purchase_date": "2025-01-01T00:00:00Z",
      "expires_date": "2025-12-31T23:59:59Z",
      "auto_renew": true,
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

### Webhook Endpoints

These endpoints are called by Apple and Google automatically:

#### Apple App Store Webhook (Unified)

```http
POST /webhook/apple
X-Apple-Notification-Signature: <JWT signature>
```

**Note**: This unified endpoint handles both production and sandbox environments automatically.

#### Apple App Store Webhook (Legacy - Deprecated)

For backward compatibility, these endpoints are still supported:

```http
POST /api/appstore/notifications/production
POST /api/appstore/notifications/sandbox
```

**Note**: It's recommended to use `/webhook/apple` for new integrations.

#### Google Play Webhook

```http
POST /webhook/google
```

**Note**: These endpoints are called automatically by Apple/Google. Configure the URLs in App Store Connect and Google Play Console.

## Project Structure

```text
verification-api/
├── cmd/
│   └── server/
│       └── main.go                    # Application entry point
├── internal/
│   ├── api/
│   │   ├── routes.go                  # API routes
│   │   ├── verification.go           # Verification handlers
│   │   ├── subscription_verify.go     # Subscription verification
│   │   ├── subscription_status.go     # Subscription status query
│   │   ├── subscription_restore.go    # Purchase restoration
│   │   ├── subscription_bind.go       # Bind account
│   │   ├── subscription_history.go    # Subscription history
│   │   ├── appstore_notification.go   # App Store webhook handlers
│   │   └── google_play_notification.go # Google Play webhook handlers
│   ├── config/
│   │   └── config.go                  # Configuration management
│   ├── database/
│   │   ├── database.go                # Database connection
│   │   └── subscription.go            # Subscription database operations
│   ├── middleware/
│   │   └── auth.go                    # Authentication middleware
│   ├── models/
│   │   ├── database.go                # Database models (Project, BaseModel)
│   │   ├── project.go                 # Project models
│   │   └── subscription.go            # Subscription models
│   └── services/
│       ├── brevo_service.go           # Email service
│       ├── project_service.go         # Project management
│       ├── redis_service.go           # Redis operations
│       ├── verification_service.go    # Verification logic
│       └── subscription_verification_service.go  # Subscription verification
├── pkg/
│   └── logging/
│       └── logger.go                  # Logging utilities
├── Dockerfile                         # Docker configuration
├── env.example                        # Environment variables template
├── go.mod                             # Go module dependencies
├── Makefile                           # Build and deployment commands
├── start.sh                           # Start script
└── test_api.sh                        # API testing script
```

## Database Schema

### Projects Table

- `id` - Primary key
- `project_id` - Unique project identifier
- `project_name` - Project display name
- `api_key` - Project API key
- `from_name` - Sender name
- `template_id` - Email template ID (optional)
- `description` - Project description
- `contact_email` - Contact email
- `rate_limit` - Rate limit per hour
- `max_requests` - Max requests per day
- `is_active` - Project status
- `bundle_id` - iOS bundle identifier (unique, for app identification)
- `package_name` - Android package name (unique, for app identification)
- `created_at` - Creation timestamp
- `updated_at` - Last update timestamp
- `deleted_at` - Soft delete timestamp

**Note**: `bundle_id` and `package_name` can be the same value if iOS and Android apps share the same package identifier.

### Subscriptions Table

- `id` - Primary key
- `user_id` - User identifier (string, defined by app)
- `project_id` - Project identifier (foreign key to projects)
- `platform` - Platform: "ios" or "android"
- `plan` - Subscription plan: "basic", "monthly", "yearly"
- `status` - Subscription status: "active", "inactive", "cancelled", "expired", "refunded", "failed"
- `start_date` - Subscription start date
- `end_date` - Subscription end date
- `product_id` - Product identifier from App Store/Google Play
- `transaction_id` - Transaction identifier (unique)
- `original_transaction_id` - Original transaction ID (for renewals)
- `environment` - Environment: "sandbox" or "production"
- `purchase_date` - Purchase date
- `expires_date` - Expiration date
- `auto_renew_status` - Auto-renewal status
- `latest_receipt` - Latest receipt data (base64 for iOS, token for Android)
- `latest_receipt_info` - Complete receipt information (JSON)
- `created_at` - Creation timestamp
- `updated_at` - Last update timestamp
- `deleted_at` - Soft delete timestamp

### Verification Codes

**Note**: Verification codes are now stored in Redis only (not in database) for better performance and automatic expiration. The following fields are stored in Redis:

- Key format: `verification:{project_id}:{email}`
- Value: JSON containing code, expires_at, is_used
- TTL: 5 minutes (configurable via `CODE_EXPIRE_MINUTES`)

## Subscription Center Architecture

The Subscription Center serves as a unified service for managing subscriptions across multiple apps:

### Architecture Overview

```
App Client (iOS/Android)
    ↓
    ├─→ POST /api/subscription/verify (upload receipt/token)
    ├─→ GET /api/subscription/status (query status)
    ├─→ POST /api/subscription/restore (restore purchases)
    ├─→ POST /api/subscription/bind_account (bind user_id)
    └─→ GET /api/subscription/history (get history)

App Backend
    ↓
    └─→ GET /api/subscription/status (with auth headers)

Subscription Center
    ↓
    ├─→ Apple App Store Server API (JWT-based verification)
    ├─→ Google Play Developer API (purchase verification)
    └─→ Database (store subscription state)

App Store Server Notifications V2
    ↓
    └─→ POST /webhook/apple (unified endpoint)

Google Play Real-Time Developer Notifications
    ↓
    └─→ POST /webhook/google
```

### Key Principles

1. **Single Source of Truth**: Subscription Center is the only place that stores and manages subscription state
2. **Data Isolation**: Each app's subscriptions are isolated by `project_id`, `bundle_id`, and `package_name`
3. **Platform Support**: Supports both iOS (App Store Server API) and Android (Google Play Developer API)
4. **Standardized API**: Uses industry-standard request/response formats
5. **Webhook Processing**: Automatic subscription status updates via App Store Server Notifications V2 and Google Play RTDN
6. **JWT Authentication**: Uses App Store Connect API Key for secure verification

### Multi-App Support

- **iOS Apps**: Identified by `bundle_id` (e.g., `com.example.app`)
- **Android Apps**: Identified by `package_name` (e.g., `com.example.app`)
- **Same Package Name**: iOS and Android can share the same identifier if needed
- **Multiple Apps**: One Subscription Center can manage subscriptions for multiple apps under the same developer account

## Testing

### Run API Tests

```bash
# Make test script executable
chmod +x test_api.sh

# Run tests
./test_api.sh
```

### Manual Testing

#### Health Check

```bash
curl http://localhost:8080/health
```

#### Email Verification

```bash
# Send verification code
curl -X POST http://localhost:8080/api/verification/send-code \
  -H "Content-Type: application/json" \
  -H "X-Project-ID: default" \
  -H "X-API-Key: default-api-key" \
  -d '{"email": "test@example.com", "project_id": "default", "language": "en"}'

# Verify code
curl -X POST http://localhost:8080/api/verification/verify-code \
  -H "Content-Type: application/json" \
  -H "X-Project-ID: default" \
  -H "X-API-Key: default-api-key" \
  -d '{"email": "test@example.com", "code": "123456", "project_id": "default"}'
```

#### Subscription Testing

**Verify Subscription (iOS - App Store Server API):**

```bash
curl -X POST http://localhost:8080/api/subscription/verify \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "ios",
    "user_id": "user_123",
    "product_id": "com.example.monthly",
    "signed_transaction": "eyJhbGciOiJFUzI1NiIsIng1YyI6WyJNSUlCU..."",
    "transaction_id": "1000000999999",
    "app_id": "com.example.app"
  }'
```

**Verify Subscription (Android):**

```bash
curl -X POST http://localhost:8080/api/subscription/verify \
  -H "Content-Type: application/json" \
  -d '{
    "platform": "android",
    "user_id": "user_123",
    "product_id": "com.example.monthly",
    "purchase_token": "opaque-token-up-to-150-characters",
    "app_id": "com.example.app"
  }'
```

**Query Subscription Status:**

```bash
curl "http://localhost:8080/api/subscription/status?user_id=user_123&app_id=com.example.app&platform=ios"
```

**Restore Subscription:**

```bash
curl -X POST http://localhost:8080/api/subscription/restore \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_123",
    "app_id": "com.example.app"
  }'
```

**Bind Account:**

```bash
curl -X POST http://localhost:8080/api/subscription/bind_account \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_123",
    "original_transaction_id": "1000000999999"
  }'
```

**Get Subscription History:**

```bash
curl "http://localhost:8080/api/subscription/history?user_id=user_123&app_id=com.example.app&platform=ios"
```

## Deployment

### Production Deployment

1. Set up PostgreSQL database
2. Set up Redis
3. Configure environment variables
4. Build and run the service:

```bash
# Build
make build
# or
go build -o unionhub cmd/server/main.go

# Run
./unionhub
```

### Railway Deployment

```bash
# Deploy to Railway
make deploy
# or
railway up
```

**Note**: 
- Set `AUTO_MIGRATE=false` in production to avoid running migrations on every deployment
- Ensure all required environment variables are configured in Railway
- Configure App Store webhook URLs in App Store Connect after deployment

## Security Considerations

- **API Key Security**: Store API keys securely, rotate regularly
- **Rate Limiting**: Configure appropriate rate limits per project
- **Database Security**: Use strong database credentials and SSL
- **Network Security**: Use HTTPS in production
- **Logging**: Monitor logs for suspicious activity
- **Code Expiration**: Keep verification codes short-lived
- **App Store Webhooks**: Verify `X-Apple-Notification-Signature` headers (implemented)
- **Receipt Validation**: Always validate receipts with Apple/Google servers
- **Subscription Data**: Encrypt sensitive subscription data at rest

## Monitoring

The service provides comprehensive monitoring capabilities:

- **Health Check**: `/health` endpoint for service status
- **Statistics**: Detailed usage statistics per project
- **Logging**: Complete audit trail of all operations
- **Rate Limiting**: Built-in abuse prevention
- **Subscription Status**: Real-time subscription status tracking
- **Webhook Processing**: Monitor App Store notification processing
- **Database Migration**: Control via `AUTO_MIGRATE` environment variable

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:
- Create an issue in the repository
- Check the documentation
- Review the API examples

## Subscription Workflow

### For App Developers

1. **Create Project**: Register your app in the Subscription Center with `bundle_id` (iOS) and/or `package_name` (Android)
2. **Configure Webhooks**: 
   - **Apple**: Set up App Store Server Notification URL in App Store Connect: `https://your-domain.com/webhook/apple`
   - **Google**: Set up Real-Time Developer Notification URL in Google Play Console: `https://your-domain.com/webhook/google`
3. **Client Integration**: 
   - After purchase, send receipt/token to `/api/subscription/verify` using standardized format
   - For iOS: Use `signed_transaction` (JWT) and `transaction_id` for App Store Server API
   - For Android: Use `purchase_token` for Google Play verification
   - Query subscription status via `/api/subscription/status`
   - Use `/api/subscription/bind_account` if webhook arrives before user verification
4. **Backend Integration**: 
   - Query subscription status with API key authentication
   - Use subscription status to control feature access
   - Query subscription history via `/api/subscription/history` for audit purposes

### For App Backends

1. **Authenticate**: Use `X-Project-ID` and `X-API-Key` headers
2. **Query Status**: Call `/api/subscription/status` with `user_id` and `app_id`
3. **Control Access**: Grant or deny access based on `is_active` and `expires_date`
4. **Monitor Subscriptions**: Use `/api/subscription/history` to track subscription changes

## Troubleshooting

### Common Issues

1. **502 Errors**: Check database and Redis connections, ensure service is binding to `0.0.0.0`
2. **Subscription Not Found**: Verify `bundle_id`/`package_name` matches App Store/Google Play configuration
3. **Webhook Not Received**: 
   - Check App Store Connect webhook URL configuration (use `/webhook/apple`)
   - Check Google Play Console RTDN URL configuration (use `/webhook/google`)
   - Verify webhook endpoints are publicly accessible
4. **Migration Errors**: Set `AUTO_MIGRATE=false` in production, run migrations manually
5. **JWT Authentication Failed**: Verify App Store Connect API Key credentials (Key ID, Issuer ID, Private Key)
6. **Transaction Verification Failed**: 
   - For iOS: Ensure `signed_transaction` is valid JWT and `transaction_id` is correct
   - For Android: Verify `purchase_token` is valid and not expired

---

**Note**: This service is designed for production use with proper database and email service configuration. Make sure to configure all required environment variables before deployment. For subscription features, ensure App Store Connect API credentials are properly configured.