# Multi-Project Deployment Guide

## 🎯 **Overview**

Verification Service supports multi-project global deployment, allowing multiple projects to share the same verification code service while maintaining data isolation and independent configurations.

## 🏗️ **Architecture Design**

### **Service Architecture**
```
┌─────────────────────────────────────────────────────────────┐
│                    Verification Service                     │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   Project A │  │   Project B │  │   Project C │  ...   │
│  │   API Key   │  │   API Key   │  │   API Key   │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
├─────────────────────────────────────────────────────────────┤
│                    Project Manager                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   Project   │  │   Project   │  │   Project   │        │
│  │   Config A  │  │   Config B  │  │   Config C  │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
├─────────────────────────────────────────────────────────────┤
│                    Database Storage                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │verification_│  │verification_│  │verification_│        │
│  │code:proj_a: │  │code:proj_b: │  │code:proj_c: │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
└─────────────────────────────────────────────────────────────┘
```

### **Key Components**

1. **Project Manager**: Manages project configurations and authentication
2. **Database Layer**: PostgreSQL for persistent storage, Redis for caching
3. **API Gateway**: Handles project-specific routing and authentication
4. **Email Service**: Brevo integration for sending verification codes
5. **Statistics Engine**: Collects and analyzes usage data per project

## 🔧 **Configuration Management**

### **Project Configuration Structure**

```json
{
  "project_id": "my-project",
  "project_name": "My Project",
  "api_key": "secure-api-key-here",
  "from_email": "noreply@myproject.com",
  "from_name": "My Project Service",
  "template_id": "custom-template-id",
  "description": "Project description",
  "contact_email": "admin@myproject.com",
  "webhook_url": "https://myproject.com/webhook",
  "rate_limit": 60,
  "max_requests": 1000,
  "is_active": true,
  "custom_config": {
    "theme": "dark",
    "language": "en"
  }
}
```

### **Environment Variables**

```bash
# Database Configuration
DATABASE_URL=postgres://username:password@localhost:5432/auth_mail?sslmode=disable
REDIS_URL=redis://localhost:6379/0

# Service Configuration
PORT=8080
GIN_MODE=production
SERVICE_NAME=Verification Service

# Brevo Configuration
BREVO_API_KEY=your-brevo-api-key
BREVO_FROM_EMAIL=noreply@yourdomain.com
BREVO_FROM_NAME=Verification Service

# Verification Settings
CODE_EXPIRE_MINUTES=5
RATE_LIMIT_MINUTES=1
```

## 🔐 **Authentication & Security**

### **Project Authentication**

Each project requires:
- **Project ID**: Unique identifier
- **API Key**: Secure authentication token
- **Active Status**: Must be enabled

### **Request Headers**

```http
X-Project-ID: your-project-id
X-API-Key: your-secure-api-key
```

### **Security Features**

- **API Key Validation**: Database-backed authentication
- **Rate Limiting**: Per-project and per-IP limits
- **Data Isolation**: Project-specific data storage
- **Audit Logging**: Complete operation tracking
- **IP Tracking**: Client IP and User-Agent logging

## 📊 **Data Isolation**

### **Database Schema**

```sql
-- Projects table
CREATE TABLE project (
    id SERIAL PRIMARY KEY,
    project_id VARCHAR(255) UNIQUE NOT NULL,
    project_name VARCHAR(255) NOT NULL,
    api_key VARCHAR(255) UNIQUE NOT NULL,
    from_email VARCHAR(255) NOT NULL,
    from_name VARCHAR(255) NOT NULL,
    template_id VARCHAR(255),
    custom_config TEXT,
    is_active BOOLEAN DEFAULT true,
    max_requests INTEGER DEFAULT 1000,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Verification codes table
CREATE TABLE verification_code (
    id SERIAL PRIMARY KEY,
    project_id VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    code VARCHAR(6) NOT NULL,
    is_used BOOLEAN DEFAULT false,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Verification logs table
CREATE TABLE verification_log (
    id SERIAL PRIMARY KEY,
    project_id VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    action VARCHAR(50) NOT NULL,
    success BOOLEAN NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    error_msg TEXT,
    request_time TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### **Redis Key Structure**

```
verification_code:{project_id}:{email}  # Verification codes
rate_limit:{project_id}:{email}         # Rate limiting
rate_limit:{project_id}:{ip_address}    # IP rate limiting
```

## 🚀 **Deployment Strategies**

### **Option 1: Single Service Instance**

**Pros:**
- Simple deployment
- Easy maintenance
- Cost-effective

**Cons:**
- Single point of failure
- Limited scalability

**Use Case:** Small to medium projects

### **Option 2: Load Balanced Multiple Instances**

**Pros:**
- High availability
- Better performance
- Horizontal scaling

**Cons:**
- More complex setup
- Higher costs

**Use Case:** Large-scale production

### **Option 3: Microservice Architecture**

**Pros:**
- Independent scaling
- Technology flexibility
- Team autonomy

**Cons:**
- Complex orchestration
- Network overhead

**Use Case:** Enterprise applications

## 📈 **Monitoring & Analytics**

### **Project-Level Metrics**

- **Verification Codes Sent**: Daily/monthly counts
- **Success Rate**: Verification success percentage
- **Error Rate**: Failed verification attempts
- **Response Time**: API response times
- **Rate Limit Hits**: Frequency limit violations

### **Global Metrics**

- **Total Projects**: Active project count
- **Service Uptime**: Overall service availability
- **Database Performance**: Query response times
- **Redis Performance**: Cache hit rates

### **Monitoring Endpoints**

```http
# Project statistics
GET /api/stats/project
X-Project-ID: your-project-id
X-API-Key: your-api-key

# Verification statistics
GET /api/stats/verification?days=7
X-Project-ID: your-project-id
X-API-Key: your-api-key

# Admin project statistics
GET /api/admin/projects/{project_id}/stats
```

## 🔧 **Project Management**

### **Creating a New Project**

```bash
curl -X POST http://localhost:8080/api/admin/projects \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "new-project",
    "project_name": "New Project",
    "api_key": "secure-api-key-123",
    "from_email": "noreply@newproject.com",
    "from_name": "New Project Service",
    "description": "A new project for testing",
    "max_requests": 1000
  }'
```

### **Updating Project Configuration**

```bash
curl -X PUT http://localhost:8080/api/admin/projects/new-project \
  -H "Content-Type: application/json" \
  -d '{
    "project_name": "Updated Project Name",
    "from_email": "new@newproject.com",
    "is_active": true
  }'
```

### **Deactivating a Project**

```bash
curl -X PUT http://localhost:8080/api/admin/projects/new-project \
  -H "Content-Type: application/json" \
  -d '{
    "is_active": false
  }'
```

## 🛠️ **Development Setup**

### **Railway Deployment**

```bash
# Clone repository
git clone <repository-url>
cd verification-api

# Install dependencies
go mod tidy

# Deploy to Railway
make deploy
# or
railway up
```

**Note**: Database and Redis services are automatically provided by Railway. Configure environment variables in Railway dashboard.

### **Testing**

```bash
# Run API tests
./script/test_api.sh

# Test specific project
curl -X POST http://localhost:8080/api/verification/send-code \
  -H "Content-Type: application/json" \
  -H "X-Project-ID: test-project" \
  -H "X-API-Key: test-api-key" \
  -d '{"email": "test@example.com", "project_id": "test-project"}'
```

## 🔒 **Security Best Practices**

### **API Key Management**

- Use strong, random API keys
- Rotate keys regularly
- Store keys securely
- Monitor key usage

### **Rate Limiting**

- Set appropriate limits per project
- Monitor for abuse patterns
- Implement progressive penalties
- Log suspicious activity

### **Data Protection**

- Encrypt sensitive data
- Use HTTPS in production
- Implement proper access controls
- Regular security audits

## 📋 **Troubleshooting**

### **Common Issues**

1. **Authentication Failures**
   - Check API key validity
   - Verify project is active
   - Confirm header format

2. **Rate Limit Exceeded**
   - Check rate limit settings
   - Review usage patterns
   - Adjust limits if needed

3. **Database Connection Issues**
   - Verify database URL
   - Check network connectivity
   - Review database logs

4. **Email Delivery Problems**
   - Verify Brevo configuration
   - Check email templates
   - Review delivery logs

### **Debug Commands**

```bash
# Check service health
curl http://localhost:8080/health

# View project list
curl http://localhost:8080/api/admin/projects

# Check project stats
curl http://localhost:8080/api/admin/projects/{project_id}/stats

# Test verification flow
./script/test_api.sh
```

## 📚 **API Reference**

### **Project Management APIs**

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/admin/projects` | List all projects |
| POST | `/api/admin/projects` | Create new project |
| PUT | `/api/admin/projects/{id}` | Update project |
| DELETE | `/api/admin/projects/{id}` | Delete project |
| GET | `/api/admin/projects/{id}/stats` | Get project statistics |

### **Verification APIs**

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/verification/send-code` | Send verification code |
| POST | `/api/verification/verify-code` | Verify code |
| GET | `/api/stats/verification` | Get verification stats |
| GET | `/api/stats/project` | Get project stats |

## 🎯 **Best Practices**

### **Project Design**

- Use descriptive project IDs
- Set appropriate rate limits
- Configure proper email templates
- Monitor usage patterns

### **Security**

- Implement proper authentication
- Use HTTPS in production
- Regular security updates
- Monitor for anomalies

### **Performance**

- Optimize database queries
- Use Redis caching effectively
- Monitor response times
- Scale horizontally when needed

### **Monitoring**

- Set up comprehensive logging
- Monitor key metrics
- Implement alerting
- Regular health checks

---

This guide provides comprehensive information for deploying and managing the Verification Service in a multi-project environment. For additional support, please refer to the main README or create an issue in the repository.