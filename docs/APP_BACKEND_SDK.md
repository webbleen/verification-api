# App Backend SDK - UnionHub Subscription Center

## 📋 概述

本文档说明 App Backend（应用后端服务）如何对接 UnionHub Subscription Center（统一订阅中心）。

**核心原则**：App Backend **只需要查询订阅状态**，不需要处理订阅验证、Webhook 或直接与 Apple/Google 通信。

## 🎯 App Backend 需要实现的功能

### 核心功能

1. **查询订阅状态** - 判断用户是否有有效的订阅
2. **权限控制** - 根据订阅状态控制功能访问
3. **缓存优化**（可选）- 缓存订阅状态以提高性能

### 不需要实现的功能

❌ **不需要**验证订阅收据/令牌（由客户端和 Subscription Center 处理）  
❌ **不需要**处理 Webhook（由 Subscription Center 处理）  
❌ **不需要**直接与 Apple App Store 或 Google Play 通信  
❌ **不需要**存储订阅数据（Subscription Center 是唯一数据源）

## 🔑 认证方式

App Backend 调用 Subscription Center API 需要使用项目认证：

```http
X-Project-ID: your-project-id
X-API-Key: your-api-key
```

**获取凭证**：
- `project_id` 和 `api_key` 在创建项目时生成
- 可以通过 UnionHub 管理界面或 API 获取

## 📡 API 端点

### 1. 查询订阅状态（核心 API）

**端点**：`GET /api/subscription/status`

**用途**：查询用户是否有有效的订阅

**请求参数**：

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `user_id` | string | 是 | 用户 ID（由 App Backend 定义） |
| `app_id` | string | 是 | 应用标识符（iOS: bundle_id, Android: package_name） |
| `platform` | string | 否 | 平台类型：`ios` 或 `android`（默认：`ios`） |

**请求示例**：

```http
GET https://verify.flaretion.com/api/subscription/status?user_id=user_123&app_id=com.example.app&platform=ios
X-Project-ID: your-project-id
X-API-Key: your-api-key
```

**响应示例（有订阅）**：

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

**响应示例（无订阅）**：

```json
{
  "success": true,
  "is_active": false,
  "status": "inactive"
}
```

**响应字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `success` | boolean | 请求是否成功 |
| `is_active` | boolean | **关键字段**：订阅是否有效（状态为 active 且未过期） |
| `platform` | string | 平台：`ios` 或 `android` |
| `status` | string | 订阅状态：`active`, `inactive`, `cancelled`, `expired`, `refunded`, `failed` |
| `plan` | string | 订阅计划：`basic`, `monthly`, `yearly` |
| `expires_date` | string | 过期时间（ISO 8601 格式） |
| `product_id` | string | 产品 ID |
| `auto_renew` | boolean | 是否开启自动续订 |

### 2. 查询订阅历史（可选）

**端点**：`GET /api/subscription/history`

**用途**：获取用户的订阅历史记录（用于审计和分析）

**请求参数**：

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `user_id` | string | 是 | 用户 ID |
| `app_id` | string | 是 | 应用标识符 |
| `platform` | string | 否 | 平台类型（默认：`ios`） |

**请求示例**：

```http
GET https://verify.flaretion.com/api/subscription/history?user_id=user_123&app_id=com.example.app&platform=ios
X-Project-ID: your-project-id
X-API-Key: your-api-key
```

**响应示例**：

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

## 💻 代码示例

### Python SDK 示例

```python
import requests
from typing import Optional, Dict, Any
from datetime import datetime

class UnionHubClient:
    """UnionHub Subscription Center Client for App Backend"""
    
    def __init__(self, base_url: str, project_id: str, api_key: str):
        self.base_url = base_url.rstrip('/')
        self.project_id = project_id
        self.api_key = api_key
        self.headers = {
            'X-Project-ID': project_id,
            'X-API-Key': api_key,
            'Content-Type': 'application/json'
        }
    
    def get_subscription_status(
        self, 
        user_id: str, 
        app_id: str, 
        platform: str = 'ios'
    ) -> Dict[str, Any]:
        """
        查询用户订阅状态
        
        Args:
            user_id: 用户 ID
            app_id: 应用标识符（bundle_id 或 package_name）
            platform: 平台类型（ios 或 android）
        
        Returns:
            订阅状态信息
        """
        url = f"{self.base_url}/api/subscription/status"
        params = {
            'user_id': user_id,
            'app_id': app_id,
            'platform': platform
        }
        
        try:
            response = requests.get(url, headers=self.headers, params=params, timeout=5)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            return {
                'success': False,
                'is_active': False,
                'error': str(e)
            }
    
    def is_user_pro(self, user_id: str, app_id: str, platform: str = 'ios') -> bool:
        """
        判断用户是否是 Pro 用户（简化方法）
        
        Args:
            user_id: 用户 ID
            app_id: 应用标识符
            platform: 平台类型
        
        Returns:
            True 如果用户有有效订阅，False 否则
        """
        status = self.get_subscription_status(user_id, app_id, platform)
        return status.get('is_active', False)
    
    def get_subscription_history(
        self, 
        user_id: str, 
        app_id: str, 
        platform: str = 'ios'
    ) -> Dict[str, Any]:
        """
        获取用户订阅历史
        
        Args:
            user_id: 用户 ID
            app_id: 应用标识符
            platform: 平台类型
        
        Returns:
            订阅历史记录
        """
        url = f"{self.base_url}/api/subscription/history"
        params = {
            'user_id': user_id,
            'app_id': app_id,
            'platform': platform
        }
        
        try:
            response = requests.get(url, headers=self.headers, params=params, timeout=5)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            return {
                'success': False,
                'subscriptions': [],
                'error': str(e)
            }


# 使用示例
client = UnionHubClient(
    base_url='https://verify.flaretion.com',
    project_id='your-project-id',
    api_key='your-api-key'
)

# 检查用户是否是 Pro
if client.is_user_pro('user_123', 'com.example.app', 'ios'):
    # 允许访问 Pro 功能
    print("User has active subscription")
else:
    # 拒绝访问或显示升级提示
    print("User does not have active subscription")
```

### Node.js/TypeScript SDK 示例

```typescript
interface SubscriptionStatus {
  success: boolean;
  is_active: boolean;
  platform?: string;
  status?: string;
  plan?: string;
  expires_date?: string;
  product_id?: string;
  auto_renew?: boolean;
  message?: string;
}

interface SubscriptionHistory {
  success: boolean;
  subscriptions: Array<{
    id: number;
    user_id: string;
    platform: string;
    plan: string;
    status: string;
    product_id: string;
    expires_date: string;
    auto_renew: boolean;
  }>;
}

class UnionHubClient {
  private baseUrl: string;
  private projectId: string;
  private apiKey: string;

  constructor(baseUrl: string, projectId: string, apiKey: string) {
    this.baseUrl = baseUrl.replace(/\/$/, '');
    this.projectId = projectId;
    this.apiKey = apiKey;
  }

  private getHeaders() {
    return {
      'X-Project-ID': this.projectId,
      'X-API-Key': this.apiKey,
      'Content-Type': 'application/json',
    };
  }

  /**
   * 查询用户订阅状态
   */
  async getSubscriptionStatus(
    userId: string,
    appId: string,
    platform: 'ios' | 'android' = 'ios'
  ): Promise<SubscriptionStatus> {
    const url = `${this.baseUrl}/api/subscription/status`;
    const params = new URLSearchParams({
      user_id: userId,
      app_id: appId,
      platform,
    });

    try {
      const response = await fetch(`${url}?${params}`, {
        method: 'GET',
        headers: this.getHeaders(),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      return {
        success: false,
        is_active: false,
        message: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  /**
   * 判断用户是否是 Pro 用户
   */
  async isUserPro(
    userId: string,
    appId: string,
    platform: 'ios' | 'android' = 'ios'
  ): Promise<boolean> {
    const status = await this.getSubscriptionStatus(userId, appId, platform);
    return status.is_active === true;
  }

  /**
   * 获取用户订阅历史
   */
  async getSubscriptionHistory(
    userId: string,
    appId: string,
    platform: 'ios' | 'android' = 'ios'
  ): Promise<SubscriptionHistory> {
    const url = `${this.baseUrl}/api/subscription/history`;
    const params = new URLSearchParams({
      user_id: userId,
      app_id: appId,
      platform,
    });

    try {
      const response = await fetch(`${url}?${params}`, {
        method: 'GET',
        headers: this.getHeaders(),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      return {
        success: false,
        subscriptions: [],
      };
    }
  }
}

// 使用示例
const client = new UnionHubClient(
  'https://verify.flaretion.com',
  'your-project-id',
  'your-api-key'
);

// 在 Express.js 中间件中使用
app.get('/api/pro-feature', async (req, res) => {
  const userId = req.user.id; // 从认证中间件获取
  const appId = 'com.example.app';
  
  const isPro = await client.isUserPro(userId, appId, 'ios');
  
  if (!isPro) {
    return res.status(403).json({
      error: 'Pro subscription required',
      message: 'Please upgrade to Pro to access this feature',
    });
  }
  
  // 允许访问 Pro 功能
  res.json({ data: 'Pro feature data' });
});
```

### Go SDK 示例

```go
package unionhub

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "time"
)

// Client represents UnionHub Subscription Center client
type Client struct {
    BaseURL   string
    ProjectID string
    APIKey    string
    HTTPClient *http.Client
}

// NewClient creates a new UnionHub client
func NewClient(baseURL, projectID, apiKey string) *Client {
    return &Client{
        BaseURL:   baseURL,
        ProjectID: projectID,
        APIKey:    apiKey,
        HTTPClient: &http.Client{
            Timeout: 5 * time.Second,
        },
    }
}

// SubscriptionStatus represents subscription status response
type SubscriptionStatus struct {
    Success     bool   `json:"success"`
    IsActive    bool   `json:"is_active"`
    Platform    string `json:"platform,omitempty"`
    Status      string `json:"status,omitempty"`
    Plan        string `json:"plan,omitempty"`
    ExpiresDate string `json:"expires_date,omitempty"`
    ProductID   string `json:"product_id,omitempty"`
    AutoRenew   bool   `json:"auto_renew,omitempty"`
    Message     string `json:"message,omitempty"`
}

// GetSubscriptionStatus queries subscription status
func (c *Client) GetSubscriptionStatus(userID, appID, platform string) (*SubscriptionStatus, error) {
    if platform == "" {
        platform = "ios"
    }
    
    u, err := url.Parse(c.BaseURL + "/api/subscription/status")
    if err != nil {
        return nil, err
    }
    
    q := u.Query()
    q.Set("user_id", userID)
    q.Set("app_id", appID)
    q.Set("platform", platform)
    u.RawQuery = q.Encode()
    
    req, err := http.NewRequest("GET", u.String(), nil)
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("X-Project-ID", c.ProjectID)
    req.Header.Set("X-API-Key", c.APIKey)
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return &SubscriptionStatus{
            Success:  false,
            IsActive: false,
            Message:  err.Error(),
        }, nil
    }
    defer resp.Body.Close()
    
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return &SubscriptionStatus{
            Success:  false,
            IsActive: false,
            Message:  err.Error(),
        }, nil
    }
    
    var status SubscriptionStatus
    if err := json.Unmarshal(body, &status); err != nil {
        return &SubscriptionStatus{
            Success:  false,
            IsActive: false,
            Message:  err.Error(),
        }, nil
    }
    
    return &status, nil
}

// IsUserPro checks if user has active subscription
func (c *Client) IsUserPro(userID, appID, platform string) (bool, error) {
    status, err := c.GetSubscriptionStatus(userID, appID, platform)
    if err != nil {
        return false, err
    }
    return status.IsActive, nil
}

// 使用示例（Gin 框架）
func ProFeatureMiddleware(client *unionhub.Client, appID string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.GetString("user_id") // 从认证中间件获取
        
        isPro, err := client.IsUserPro(userID, appID, "ios")
        if err != nil || !isPro {
            c.JSON(http.StatusForbidden, gin.H{
                "error": "Pro subscription required",
                "message": "Please upgrade to Pro to access this feature",
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

### PHP SDK 示例

```php
<?php

class UnionHubClient {
    private $baseUrl;
    private $projectId;
    private $apiKey;
    private $timeout = 5;

    public function __construct($baseUrl, $projectId, $apiKey) {
        $this->baseUrl = rtrim($baseUrl, '/');
        $this->projectId = $projectId;
        $this->apiKey = $apiKey;
    }

    /**
     * 查询用户订阅状态
     */
    public function getSubscriptionStatus($userId, $appId, $platform = 'ios') {
        $url = $this->baseUrl . '/api/subscription/status';
        $params = http_build_query([
            'user_id' => $userId,
            'app_id' => $appId,
            'platform' => $platform,
        ]);

        $ch = curl_init($url . '?' . $params);
        curl_setopt_array($ch, [
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_HTTPHEADER => [
                'X-Project-ID: ' . $this->projectId,
                'X-API-Key: ' . $this->apiKey,
                'Content-Type: application/json',
            ],
            CURLOPT_TIMEOUT => $this->timeout,
        ]);

        $response = curl_exec($ch);
        $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        curl_close($ch);

        if ($httpCode !== 200) {
            return [
                'success' => false,
                'is_active' => false,
                'error' => 'HTTP ' . $httpCode,
            ];
        }

        return json_decode($response, true) ?: [
            'success' => false,
            'is_active' => false,
        ];
    }

    /**
     * 判断用户是否是 Pro 用户
     */
    public function isUserPro($userId, $appId, $platform = 'ios') {
        $status = $this->getSubscriptionStatus($userId, $appId, $platform);
        return isset($status['is_active']) && $status['is_active'] === true;
    }
}

// 使用示例（Laravel）
$client = new UnionHubClient(
    'https://verify.flaretion.com',
    'your-project-id',
    'your-api-key'
);

// 在控制器中使用
Route::middleware('auth')->get('/api/pro-feature', function (Request $request) use ($client) {
    $userId = $request->user()->id;
    $appId = 'com.example.app';
    
    if (!$client->isUserPro($userId, $appId, 'ios')) {
        return response()->json([
            'error' => 'Pro subscription required',
            'message' => 'Please upgrade to Pro to access this feature',
        ], 403);
    }
    
    return response()->json(['data' => 'Pro feature data']);
});
```

## 🔒 权限控制实现模式

### 模式 1：中间件模式（推荐）

在 API 路由中使用中间件检查订阅状态：

```python
# Python (Flask)
@app.route('/api/pro-feature', methods=['GET'])
@require_pro_subscription  # 自定义装饰器
def pro_feature():
    return jsonify({'data': 'Pro feature data'})

def require_pro_subscription(f):
    @wraps(f)
    def decorated_function(*args, **kwargs):
        user_id = get_current_user_id()
        if not unionhub_client.is_user_pro(user_id, APP_ID, 'ios'):
            return jsonify({
                'error': 'Pro subscription required'
            }), 403
        return f(*args, **kwargs)
    return decorated_function
```

### 模式 2：服务层模式

在业务逻辑层检查订阅状态：

```python
class FeatureService:
    def __init__(self, unionhub_client):
        self.unionhub_client = unionhub_client
    
    def access_pro_feature(self, user_id: str):
        if not self.unionhub_client.is_user_pro(user_id, APP_ID, 'ios'):
            raise PermissionError('Pro subscription required')
        
        # 执行业务逻辑
        return self._do_pro_feature()
```

### 模式 3：缓存模式（性能优化）

使用 Redis 缓存订阅状态，减少 API 调用：

```python
import redis
from datetime import timedelta

class CachedUnionHubClient:
    def __init__(self, unionhub_client, redis_client, cache_ttl=300):
        self.client = unionhub_client
        self.redis = redis_client
        self.cache_ttl = cache_ttl  # 5 分钟缓存
    
    def is_user_pro(self, user_id: str, app_id: str, platform: str = 'ios') -> bool:
        cache_key = f"subscription:{user_id}:{app_id}:{platform}"
        
        # 检查缓存
        cached = self.redis.get(cache_key)
        if cached is not None:
            return cached == b'true'
        
        # 查询 Subscription Center
        is_pro = self.client.is_user_pro(user_id, app_id, platform)
        
        # 写入缓存
        self.redis.setex(
            cache_key, 
            self.cache_ttl, 
            'true' if is_pro else 'false'
        )
        
        return is_pro
```

## ⚠️ 错误处理

### 常见错误情况

1. **网络错误**：Subscription Center 不可用
   - **处理**：返回 `is_active: false`，记录错误日志
   - **建议**：实现重试机制和降级策略

2. **认证失败**：`project_id` 或 `api_key` 错误
   - **处理**：检查凭证配置
   - **HTTP 状态码**：401 或 403

3. **应用未找到**：`app_id` 不存在
   - **处理**：检查 `bundle_id` 或 `package_name` 配置
   - **HTTP 状态码**：400

4. **用户无订阅**：正常情况，不是错误
   - **处理**：返回 `is_active: false`，拒绝访问 Pro 功能

### 错误处理示例

```python
def safe_check_subscription(user_id: str, app_id: str) -> bool:
    """
    安全地检查订阅状态，包含错误处理
    """
    try:
        status = unionhub_client.get_subscription_status(user_id, app_id, 'ios')
        
        # 检查请求是否成功
        if not status.get('success', False):
            # 请求失败，记录日志但允许访问（降级策略）
            logger.warning(f"Failed to check subscription for user {user_id}: {status.get('message')}")
            return False  # 或根据业务需求返回 True（允许访问）
        
        return status.get('is_active', False)
        
    except Exception as e:
        # 网络错误或其他异常
        logger.error(f"Error checking subscription: {e}")
        # 根据业务需求决定：返回 False（拒绝访问）或 True（允许访问，降级策略）
        return False
```

## 📊 最佳实践

### 1. 缓存策略

- **缓存时间**：建议 5-10 分钟
- **缓存键**：`subscription:{user_id}:{app_id}:{platform}`
- **缓存失效**：当用户购买/续订时，清除相关缓存

### 2. 超时设置

- **建议超时**：3-5 秒
- **重试机制**：最多重试 2 次
- **降级策略**：超时或失败时，根据业务需求决定允许/拒绝访问

### 3. 日志记录

记录以下信息：
- 订阅状态查询请求
- 查询结果（成功/失败）
- 错误详情
- 性能指标（响应时间）

### 4. 监控告警

监控以下指标：
- API 调用成功率
- API 响应时间
- 错误率
- 缓存命中率

## 🔄 完整集成流程

### 步骤 1：获取凭证

1. 在 UnionHub 创建项目
2. 获取 `project_id` 和 `api_key`
3. 配置 `bundle_id`（iOS）或 `package_name`（Android）

### 步骤 2：初始化 SDK

```python
from unionhub import UnionHubClient

client = UnionHubClient(
    base_url='https://verify.flaretion.com',
    project_id='your-project-id',
    api_key='your-api-key'
)
```

### 步骤 3：实现权限检查

```python
# 在需要 Pro 功能的 API 中
@app.route('/api/pro-feature', methods=['GET'])
@require_auth
def pro_feature():
    user_id = get_current_user_id()
    
    if not client.is_user_pro(user_id, 'com.example.app', 'ios'):
        return jsonify({
            'error': 'Pro subscription required',
            'upgrade_url': 'https://example.com/upgrade'
        }), 403
    
    # 执行业务逻辑
    return jsonify({'data': 'Pro feature data'})
```

### 步骤 4：测试

```bash
# 测试订阅状态查询
curl -X GET "https://verify.flaretion.com/api/subscription/status?user_id=test_user&app_id=com.example.app&platform=ios" \
  -H "X-Project-ID: your-project-id" \
  -H "X-API-Key: your-api-key"
```

## 📝 总结

App Backend 只需要实现：

1. ✅ **查询订阅状态** - 调用 `GET /api/subscription/status`
2. ✅ **权限控制** - 根据 `is_active` 字段控制功能访问
3. ✅ **错误处理** - 处理网络错误和异常情况
4. ✅ **缓存优化**（可选）- 提高性能

**不需要实现**：
- ❌ 订阅验证
- ❌ Webhook 处理
- ❌ 直接与 Apple/Google 通信
- ❌ 存储订阅数据

**记住**：Subscription Center 是订阅状态的**唯一数据源**，App Backend 只需要**查询**，不需要**验证**或**存储**。

