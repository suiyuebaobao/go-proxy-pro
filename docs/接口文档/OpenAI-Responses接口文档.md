# OpenAI Responses API 接口文档

## 概述

OpenAI Responses API（又称 Codex API）是用于代码生成和对话的专用接口，支持 Claude Code、Codex CLI 等客户端。

**认证方式**：API Key

```bash
Authorization: Bearer sk-xxxxxxxxxxxx
```

---

## 支持的路由

| 路由 | 方法 | 说明 |
|------|------|------|
| `/responses` | POST | OpenAI Responses API |
| `/v1/responses` | POST | OpenAI Responses API（v1 路径） |
| `/openai/responses` | POST | OpenAI Responses API（平台路由） |
| `/openai/v1/responses` | POST | OpenAI Responses API（平台路由 v1） |

---

## 一、Responses 接口

### 请求

**路径**: `POST /responses` 或 `POST /v1/responses`

**请求头**:
| 头部 | 必填 | 说明 |
|------|------|------|
| `Authorization` | 是 | Bearer Token，格式：`Bearer sk-xxx` |
| `Content-Type` | 是 | 必须为 `application/json` |
| `Session_id` | 否 | 会话 ID，用于会话粘性 |
| `Version` | 否 | 客户端版本号 |
| `Openai-Beta` | 否 | Beta 特性标识 |

**请求体**:
```json
{
  "model": "gpt-5.1-codex-max",
  "input": "Write a hello world function in Python",
  "stream": true
}
```

**请求参数**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `model` | string | 否 | 模型名称，默认 `gpt-4` |
| `input` | string/array | 是 | 输入内容，可以是字符串或消息数组 |
| `stream` | boolean | 否 | 是否使用流式响应，默认 `true` |
| `max_tokens` | number | 否 | 最大输出 Token 数 |
| `temperature` | number | 否 | 采样温度 |

### 响应

#### 流式响应（stream: true）

**Content-Type**: `text/event-stream`

流式响应以 SSE（Server-Sent Events）格式返回，每个事件包含 `event` 和 `data` 字段：

```
event: response.created
data: {"id":"resp_xxx","type":"response.created","response":{...}}

event: response.output_item.added
data: {"id":"resp_xxx","type":"response.output_item.added",...}

event: response.content_part.added
data: {"id":"resp_xxx","type":"response.content_part.added",...}

event: response.output_text.delta
data: {"id":"resp_xxx","type":"response.output_text.delta","delta":"Hello",...}

event: response.completed
data: {"id":"resp_xxx","type":"response.completed","response":{"model":"gpt-5.1-codex-max","usage":{"input_tokens":10,"output_tokens":50}}}

data: [DONE]
```

**关键事件类型**:
| 事件类型 | 说明 |
|----------|------|
| `response.created` | 响应创建 |
| `response.output_item.added` | 输出项添加 |
| `response.content_part.added` | 内容块添加 |
| `response.output_text.delta` | 文本增量输出 |
| `response.completed` | 响应完成，包含 usage 统计 |

#### 非流式响应（stream: false）

```json
{
  "id": "resp_xxx",
  "type": "response",
  "model": "gpt-5.1-codex-max",
  "output": [
    {
      "type": "message",
      "content": [
        {
          "type": "text",
          "text": "def hello_world():\n    print('Hello, World!')"
        }
      ]
    }
  ],
  "usage": {
    "input_tokens": 10,
    "output_tokens": 50
  }
}
```

---

## 二、请求示例

### cURL 示例

```bash
# 流式请求
curl http://localhost:8080/responses \
  -H "Authorization: Bearer sk-xxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.1-codex-max",
    "input": "Write a hello world function in Python",
    "stream": true
  }'

# 非流式请求
curl http://localhost:8080/v1/responses \
  -H "Authorization: Bearer sk-xxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.1-codex-max",
    "input": "Write a hello world function in Python",
    "stream": false
  }'
```

### Claude Code 配置

在 Claude Code 客户端中配置：

```
Base URL: http://your-server:8080
API Key: sk-xxxxxxxxxxxx
```

注意：Base URL 不要包含 `/responses` 路径，客户端会自动追加。

如果你的客户端支持按平台配置 Base URL，也可以使用：

```
Base URL: http://your-server:8080/openai/
API Key: sk-xxxxxxxxxxxx
```

---

## 三、账户配置

### 添加 OpenAI Responses 账户

在账户管理页面添加类型为 `openai-responses` 的账户：

| 字段 | 说明 | 示例 |
|------|------|------|
| `name` | 账户名称 | `MyCodexAccount` |
| `type` | 账户类型 | `openai-responses` |
| `api_key` | API 密钥 | `cr_xxxx` |
| `base_url` | 上游 API 地址 | `https://api.example.com/openai` |
| `organization_id` | 组织 ID（可选） | 用于 chatgpt.com 请求 |

### 默认上游地址

如果未配置 `base_url`，将使用默认地址：
```
https://chatgpt.com/backend-api/codex
```

---

## 四、特殊处理

### chatgpt.com 请求

当请求目标为 `chatgpt.com` 时，系统自动添加以下请求头：

| 头部 | 值 |
|------|-----|
| `openai-beta` | `responses=experimental` |
| `chatgpt-account-id` | 账户的 `organization_id` 值 |

### 请求头透传

以下客户端请求头会透传到上游：

| 客户端头 | 上游头 |
|----------|--------|
| `User-Agent` | `User-Agent` |
| `Session_id` | `session_id` |
| `Version` | `version` |
| `Openai-Beta` | `openai-beta` |

---

## 五、错误码

| 状态码 | 错误码 | 说明 |
|--------|--------|------|
| 400 | `invalid_request` | 请求体无效 |
| 401 | `unauthorized` | API Key 无效/禁用/过期 |
| 403 | `forbidden` | 客户端被过滤 |
| 429 | `rate_limited` | 用户并发超限 |
| 502 | `upstream_error` | 上游 API 网络错误 |
| 503 | `no_available_account` | 无可用的 openai-responses 账户 |

---

## 六、使用统计

系统会自动从流式响应的 `response.completed` 事件中提取 usage 数据：

- `input_tokens`: 输入 Token 数
- `output_tokens`: 输出 Token 数

这些数据会记录到用户和账户的使用统计中。

---

## 七、技术说明

### 流式转发机制

本服务采用直接字节流转发策略，参考 claude-relay 项目实现：

1. 接收上游响应后，直接将原始字节写入客户端响应
2. 不解析或修改 SSE 格式，保持流的完整性
3. 同时异步解析 `response.completed` 事件获取 usage 统计

这种方式确保了：
- SSE 事件格式不被破坏
- 流式响应延迟最小化
- 兼容各种客户端实现

### 中间件链

请求经过以下中间件处理：

```
APIKeyAuth → ClientFilter → CheckAllowedClients → UserConcurrencyControl → Handler
```

---

## 八、会话粘性

OpenAI Responses API 支持会话粘性功能，确保同一 API Key 的请求始终路由到同一后端账户。

### 实现机制

| 项目 | 说明 |
|------|------|
| Session ID 格式 | `apikey:{apiKeyID}` |
| 存储方式 | 内存缓存 (sync.Map) |
| TTL | 约 1 小时（可配置） |

### 工作流程

1. **首次请求**：按权重选择账户，创建会话绑定到内存缓存
2. **后续请求**：从缓存获取绑定，直接使用已绑定账户
3. **账户不可用**：自动移除绑定，重新选择账户

### 查看会话绑定

通过管理界面「缓存管理」页面查看：
- 访问「缓存管理」→「会话列表」Tab
- 可查看所有会话绑定及其 TTL
- 支持手动清除会话绑定
