# M4 - API Key 管理模块

## 模块概述

API Key 管理模块为用户提供创建和管理 API Key 的功能，用于调用代理服务的各类 AI 接口。

## 功能列表

### 已完成
- [x] API Key 生成（带前缀 `sk-`，32字节随机）
- [x] API Key 安全存储（SHA256 哈希）
- [x] API Key CRUD 操作
- [x] API Key 认证中间件
- [x] 前端管理页面

### 待完成
- [ ] 速率限制实现（当前仅存储配置）
- [ ] 每日/月度配额限制
- [ ] 平台级别权限控制
- [ ] 模型级别权限控制

## 数据结构

### API Key 模型

```go
type APIKey struct {
    ID               uint       `gorm:"primaryKey"`
    UserID           uint       `gorm:"index;not null"`
    Name             string     `gorm:"size:100;not null"`
    KeyHash          string     `gorm:"size:64;uniqueIndex;not null"` // SHA256 哈希
    KeyPrefix        string     `gorm:"size:12;not null"`             // 显示用前缀 sk-xxx...
    Status           string     `gorm:"size:20;default:active"`       // active/disabled
    AllowedPlatforms string     `gorm:"size:500;default:all"`         // 允许的平台，逗号分隔
    AllowedModels    string     `gorm:"size:1000"`                    // 允许的模型，逗号分隔
    RateLimit        int        `gorm:"default:60"`                   // 每分钟请求数
    DailyLimit       int        `gorm:"default:0"`                    // 每日请求限制，0=无限
    MonthlyQuota     float64    `gorm:"default:0"`                    // 月度费用配额，0=无限
    UsedTokens       int64      `gorm:"default:0"`                    // 已使用 Token
    UsedCost         float64    `gorm:"default:0"`                    // 已使用费用
    RequestCount     int64      `gorm:"default:0"`                    // 请求次数
    LastUsedAt       *time.Time                                       // 最后使用时间
    ExpiresAt        *time.Time                                       // 过期时间
    CreatedAt        time.Time
    UpdatedAt        time.Time
}
```

### API Key 生成

```go
// 生成格式：sk-{32字节随机hex} = sk- + 64字符
func GenerateAPIKey() (key, hash, prefix string, err error) {
    bytes := make([]byte, 32)
    rand.Read(bytes)
    key = "sk-" + hex.EncodeToString(bytes)
    hash = HashAPIKey(key)
    prefix = key[:12] + "..."
    return
}

func HashAPIKey(key string) string {
    h := sha256.Sum256([]byte(key))
    return hex.EncodeToString(h[:])
}
```

## 接口设计

### 用户接口（需 JWT 认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/api-keys` | 获取用户的所有 API Key |
| POST | `/api/api-keys` | 创建新的 API Key |
| GET | `/api/api-keys/:id` | 获取单个 API Key 详情 |
| PUT | `/api/api-keys/:id` | 更新 API Key 配置 |
| DELETE | `/api/api-keys/:id` | 删除 API Key |
| PUT | `/api/api-keys/:id/toggle` | 切换 API Key 状态 |

### 代理接口（需 API Key 认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/v1/chat/completions` | OpenAI 兼容接口 |
| POST | `/api/v1/messages` | Claude 原生接口 |
| POST | `/gemini/v1/chat` | Gemini 接口 |

### 认证方式

```bash
# 方式1：Authorization Header
Authorization: Bearer sk-xxxxxxxxxxxx

# 方式2：X-API-Key Header
X-API-Key: sk-xxxxxxxxxxxx
```

## 依赖关系

```
┌─────────────────┐
│   Handler       │  ← HTTP 层
├─────────────────┤
│   Service       │  ← 业务逻辑
├─────────────────┤
│   Repository    │  ← 数据访问
├─────────────────┤
│   Model         │  ← 数据模型
└─────────────────┘

┌─────────────────┐
│   Middleware    │  ← API Key 认证
│   (APIKeyAuth)  │
└─────────────────┘
        │
        ▼
┌─────────────────┐
│  Proxy Handler  │  ← 代理接口
└─────────────────┘
```

## 文件清单

```
internal/
├── model/
│   └── api_key.go          # 数据模型 + Key 生成函数
├── repository/
│   └── api_key.go          # 数据访问层
├── service/
│   └── api_key.go          # 业务逻辑层
├── handler/
│   └── api_key.go          # HTTP 处理器
└── middleware/
    └── api_key.go          # API Key 认证中间件

web/src/
├── views/
│   └── APIKeys.vue         # 管理页面
└── api/
    └── index.js            # API 接口定义
```

## 安全设计

1. **Key 存储安全**
   - 只存储 SHA256 哈希，不存储原始 Key
   - 完整 Key 只在创建时返回一次

2. **认证流程**
   ```
   请求 → 提取 API Key → 计算哈希 → 查询数据库 → 验证状态 → 放行/拒绝
   ```

3. **权限控制**
   - 平台级别：限制可访问的 AI 平台
   - 模型级别：限制可使用的模型
   - 所有权验证：用户只能操作自己的 Key

## 使用示例

### 创建 API Key

```bash
curl -X POST http://localhost:8080/api/api-keys \
  -H "Authorization: Bearer JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "我的测试Key",
    "allowed_platforms": "claude-official,openai",
    "rate_limit": 60
  }'
```

### 使用 API Key 调用代理

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-20250514",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

## 后续规划

1. **速率限制**：基于 Redis 实现滑动窗口限流
2. **配额管理**：每日/月度请求和费用限制
3. **使用统计**：详细的 Token 消耗和费用统计
4. **权限增强**：更细粒度的模型权限控制
