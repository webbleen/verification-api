# Verification API

A standalone verification code API service based on Gin framework and Brevo email service, supporting multi-project global deployment with email verification code sending and validation functionality.

## Features

- 📧 **Code Sending** - Send 6-digit verification codes to specified email addresses
- ✅ **Code Verification** - Validate user-input verification codes
- 🏢 **Multi-Project Support** - Support multiple projects with data isolation
- 🔐 **Project Authentication** - API key-based project identity verification
- 🚀 **Redis Caching** - Efficient verification code storage and management
- ⚡ **Rate Limiting** - Prevent verification code abuse (1-minute cooldown)
- 🐳 **Docker Support** - Containerized deployment
- 🔒 **Security** - 5-minute code expiration, one-time use
- 📊 **Project Management** - Full CRUD operations for project management
- 🗄️ **Database Support** - PostgreSQL + Redis dual storage architecture
- 📈 **Statistics & Monitoring** - Comprehensive analytics and monitoring
- 🌍 **Multi-Language Support** - Email content in 8 languages (EN, ZH-CN, ZH-TW, JA, KO, ES, FR, DE)

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
# Using Docker Compose (recommended)
docker-compose up -d

# Or start individually
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
| `SERVICE_NAME` | Service name | `Verification Service` | No |

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

## API Documentation

### Authentication

All API endpoints require project authentication using headers:

```bash
X-Project-ID: your-project-id
X-API-Key: your-api-key
```

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
  "max_requests": 1000
}
```

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

## Project Structure

```text
verification-api/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── api/
│   │   ├── routes.go            # API routes
│   │   └── verification.go      # Verification handlers
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── database/
│   │   └── database.go          # Database connection
│   ├── middleware/
│   │   └── auth.go              # Authentication middleware
│   ├── models/
│   │   ├── database.go          # Database models
│   │   └── project.go           # Project models
│   └── services/
│       ├── brevo_service.go     # Email service
│       ├── project_service.go   # Project management
│       ├── redis_service.go     # Redis operations
│       └── verification_service.go # Verification logic
├── pkg/
│   └── logging/
│       └── logger.go            # Logging utilities
├── docker-compose.yml           # Docker Compose configuration
├── Dockerfile                   # Docker configuration
├── env.example                  # Environment variables template
├── go.mod                       # Go module dependencies
├── start.sh                     # Start script
└── test_api.sh                  # API testing script
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
- `webhook_url` - Webhook URL (optional)
- `rate_limit` - Rate limit per hour
- `max_requests` - Max requests per day
- `is_active` - Project status
- `created_at` - Creation timestamp
- `updated_at` - Last update timestamp

### Verification Codes Table

- `id` - Primary key
- `project_id` - Project identifier
- `email` - User email address
- `code` - Verification code
- `is_used` - Usage status
- `expires_at` - Expiration timestamp
- `used_at` - Usage timestamp
- `ip_address` - Client IP address
- `user_agent` - Client user agent
- `created_at` - Creation timestamp

### Verification Logs Table

- `id` - Primary key
- `project_id` - Project identifier
- `email` - User email address
- `action` - Action type (send/verify)
- `success` - Success status
- `ip_address` - Client IP address
- `user_agent` - Client user agent
- `error_msg` - Error message (if failed)
- `request_time` - Request timestamp

## Testing

### Run API Tests

```bash
# Make test script executable
chmod +x test_api.sh

# Run tests
./test_api.sh
```

### Manual Testing

```bash
# Health check
curl http://localhost:8080/health

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

## Deployment

### Docker Compose

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f verification-api

# Stop services
docker-compose down
```

### Production Deployment

1. Set up PostgreSQL database
2. Configure environment variables
3. Build and run the service:

```bash
# Build
go build -o verification-api cmd/server/main.go

# Run
./verification-api
```

## Security Considerations

- **API Key Security**: Store API keys securely, rotate regularly
- **Rate Limiting**: Configure appropriate rate limits per project
- **Database Security**: Use strong database credentials and SSL
- **Network Security**: Use HTTPS in production
- **Logging**: Monitor logs for suspicious activity
- **Code Expiration**: Keep verification codes short-lived

## Monitoring

The service provides comprehensive monitoring capabilities:

- **Health Check**: `/health` endpoint for service status
- **Statistics**: Detailed usage statistics per project
- **Logging**: Complete audit trail of all operations
- **Rate Limiting**: Built-in abuse prevention

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

---

**Note**: This service is designed for production use with proper database and email service configuration. Make sure to configure all required environment variables before deployment.