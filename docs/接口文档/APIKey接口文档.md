# API Key 管理接口文档

## 概述

API Key 管理接口用于用户创建和管理自己的 API Key，以便调用 AI 代理服务。

**认证方式**：JWT Token（管理接口）/ API Key（代理接口）

---

## 用户 API Key 管理接口

### 1. 获取 API Key 列表

**路径**: `GET /api/api-keys`

**认证**: JWT Token

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": 1,
      "name": "测试Key",
      "key_prefix": "sk-a1b2c3d4...",
      "status": "active",
      "allowed_platforms": "all",
      "allowed_models": "",
      "rate_limit": 60,
      "daily_limit": 0,
      "monthly_quota": 0,
      "used_tokens": 1500,
      "used_cost": 0.05,
      "request_count": 10,
      "last_used_at": "2025-12-09T10:30:00Z",
      "expires_at": null,
      "created_at": "2025-12-09T08:00:00Z"
    }
  ]
}
```

---

### 2. 创建 API Key

**路径**: `POST /api/api-keys`

**认证**: JWT Token

**请求参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | Key 名称 |
| allowed_platforms | string | 否 | 允许的平台，逗号分隔，默认 "all" |
| allowed_models | string | 否 | 允许的模型，逗号分隔，空=全部 |
| rate_limit | int | 否 | 每分钟请求数限制，默认 60 |
| daily_limit | int | 否 | 每日请求限制，0=无限 |
| monthly_quota | float | 否 | 月度费用配额，0=无限 |
| expires_at | string | 否 | 过期时间，ISO 8601 格式 |

**请求示例**:
```json
{
  "name": "生产环境Key",
  "allowed_platforms": "claude-official,openai",
  "rate_limit": 100,
  "expires_at": "2025-12-31T23:59:59Z"
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "created",
  "data": {
    "id": 2,
    "name": "生产环境Key",
    "key": "sk-a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6",
    "key_prefix": "sk-a1b2c3d4..."
  }
}
```

> **注意**: `key` 字段只在创建时返回一次，请妥善保存！

---

### 3. 获取单个 API Key

**路径**: `GET /api/api-keys/:id`

**认证**: JWT Token

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "测试Key",
    "key_prefix": "sk-a1b2c3d4...",
    "status": "active",
    "allowed_platforms": "all",
    "allowed_models": "",
    "rate_limit": 60,
    "daily_limit": 0,
    "monthly_quota": 0,
    "used_tokens": 1500,
    "used_cost": 0.05,
    "request_count": 10,
    "last_used_at": "2025-12-09T10:30:00Z",
    "expires_at": null,
    "created_at": "2025-12-09T08:00:00Z"
  }
}
```

---

### 4. 更新 API Key

**路径**: `PUT /api/api-keys/:id`

**认证**: JWT Token

**请求参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | Key 名称 |
| allowed_platforms | string | 否 | 允许的平台 |
| allowed_models | string | 否 | 允许的模型 |
| rate_limit | int | 否 | 每分钟请求数限制 |
| daily_limit | int | 否 | 每日请求限制 |
| monthly_quota | float | 否 | 月度费用配额 |
| expires_at | string | 否 | 过期时间 |
| status | string | 否 | 状态：active/disabled |

**请求示例**:
```json
{
  "name": "更新后的名称",
  "rate_limit": 120
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "name": "更新后的名称",
    "rate_limit": 120
  }
}
```

---

### 5. 删除 API Key

**路径**: `DELETE /api/api-keys/:id`

**认证**: JWT Token

**响应示例**:
```json
{
  "code": 0,
  "message": "success"
}
```

---

### 6. 切换 API Key 状态

**路径**: `PUT /api/api-keys/:id/toggle`

**认证**: JWT Token

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "status": "disabled"
  }
}
```

---

## 代理接口（使用 API Key 认证）

### 认证方式

所有代理接口需要在请求头中携带 API Key：

```bash
# 方式1：Authorization Header（推荐）
Authorization: Bearer sk-xxxxxxxxxxxx

# 方式2：X-API-Key Header
X-API-Key: sk-xxxxxxxxxxxx
```

---

### 1. OpenAI 兼容接口

**路径**: `POST /openai/v1/chat/completions`

**请求示例**:
```bash
curl http://localhost:8080/openai/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-20250514",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ],
    "stream": false
  }'
```

**模型格式**:
```json
// 直接使用模型名（自动选择可用平台）
{"model": "claude-sonnet-4-20250514"}

// 指定平台类型
{"model": "claude-official,claude-sonnet-4-20250514"}
```

**响应示例**:
```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1702123456,
  "model": "claude-sonnet-4-20250514",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! How can I help you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 20,
    "total_tokens": 30
  }
}
```

---

### 2. Claude 原生接口

**路径**: `POST /claude/v1/messages`

**请求示例**:
```bash
curl http://localhost:8080/claude/v1/messages \
  -H "Authorization: Bearer sk-xxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-20250514",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ],
    "max_tokens": 1024
  }'
```

**响应示例**:
```json
{
  "id": "msg_xxx",
  "type": "message",
  "role": "assistant",
  "content": [
    {
      "type": "text",
      "text": "Hello! How can I help you today?"
    }
  ],
  "model": "claude-sonnet-4-20250514",
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 10,
    "output_tokens": 20
  }
}
```

---

### 3. Gemini 接口

**路径**: `POST /gemini/v1/chat`

**请求示例**:
```bash
curl http://localhost:8080/gemini/v1/chat \
  -H "Authorization: Bearer sk-xxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-2.5-flash",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

---

## 错误响应

### 认证错误

```json
{
  "code": 401,
  "message": "缺少 API Key，请在 Authorization 或 x-api-key header 中提供"
}
```

```json
{
  "code": 401,
  "message": "无效的 API Key"
}
```

```json
{
  "code": 401,
  "message": "API Key 已被禁用"
}
```

```json
{
  "code": 401,
  "message": "API Key 已过期"
}
```

### 权限错误

```json
{
  "code": 403,
  "message": "无权访问此平台"
}
```

```json
{
  "code": 403,
  "message": "无权使用此模型"
}
```

### 限额错误

```json
{
  "code": 429,
  "message": "请求过于频繁，请稍后再试"
}
```

---

## 支持的平台

| 平台标识 | 说明 |
|---------|------|
| claude-official | Claude 官方 API |
| claude-console | Claude Console |
| claude-ccr | Claude CCR |
| bedrock | AWS Bedrock |
| openai | OpenAI API |
| azure-openai | Azure OpenAI |
| gemini | Google Gemini |

---

## 限制说明

- 每个用户最多创建 **10** 个 API Key
- 默认每分钟请求限制：**60** 次
- API Key 只在创建时显示一次，请妥善保存
