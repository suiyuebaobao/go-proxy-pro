# 接口文档索引

> 最后更新：2025-12-29

## 文档列表

| 文档 | 说明 |
|------|------|
| [认证接口文档](认证接口文档.md) | 登录、注册接口 |
| [用户接口文档](用户接口文档.md) | 个人资料、使用统计、套餐查询 |
| [代理转发接口文档](代理转发接口文档.md) | OpenAI/Claude/Gemini 代理接口 |
| [OpenAI-Responses接口文档](OpenAI-Responses接口文档.md) | OpenAI Responses API (Codex) 接口 |
| [APIKey接口文档](APIKey接口文档.md) | API Key 管理接口 |
| [系统监控接口文档](系统监控接口文档.md) | 系统监控数据接口 |
| [系统日志接口文档](系统日志接口文档.md) | 系统日志查看接口 |
| [管理员接口-用户账户模型](管理员接口-用户账户模型.md) | 用户/账户/模型/OAuth 管理 |
| [管理员接口-日志缓存配置](管理员接口-日志缓存配置.md) | 日志/缓存/配置/代理配置 管理 |
| [管理员接口-套餐客户端过滤](管理员接口-套餐客户端过滤.md) | 套餐/客户端过滤 管理 |

---

## 认证方式

### 1. JWT Token（管理后台）

用于管理后台接口，登录后获取。

```bash
Authorization: Bearer <jwt_token>
```

### 2. API Key（代理接口）

用于 AI 代理转发接口。

```bash
Authorization: Bearer sk-xxxxxxxxxxxx
```

或

```bash
X-API-Key: sk-xxxxxxxxxxxx
```

或（Claude 客户端常用）

```bash
x-api-key: sk-xxxxxxxxxxxx
```

---

## Base URL 配置（推荐）

客户端配置时只需填写 Base URL，客户端会自动拼接后续路径。

| 平台 | Base URL | 完整端点 |
|------|----------|----------|
| Claude | `http://域名/claude/` | `/claude/v1/messages` |
| OpenAI | `http://域名/openai/` | `/openai/v1/chat/completions` |
| Gemini | `http://域名/gemini/` | `/gemini/v1/chat` |

## 接口分类

### 公开接口（无需认证）

| 接口 | 方法 | 说明 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/api/auth/captcha` | GET | 获取验证码（如启用） |
| `/api/auth/captcha/status` | GET | 获取验证码/登录限流开关状态 |
| `/api/auth/login` | POST | 用户登录 |
| `/api/auth/register` | POST | 用户注册 |

### 代理接口（API Key 认证）

| 接口 | 方法 | 说明 |
|------|------|------|
| `/claude/v1/messages` | POST | Claude 原生接口 |
| `/openai/v1/chat/completions` | POST | OpenAI Chat Completions |
| `/gemini/v1/chat` | POST | Gemini 接口 |
| `/responses` | POST | OpenAI Responses API（Codex/Claude Code 等） |
| `/v1/responses` | POST | OpenAI Responses API（v1 路径） |

### 用户接口（JWT 认证）

| 路径前缀 | 说明 |
|----------|------|
| `/api/profile` | 个人资料管理 |
| `/api/api-keys` | API Key 管理 |
| `/api/usage` | 使用统计 |
| `/api/models` | 模型查询 |
| `/api/packages` | 套餐查询 |
| `/api/my-packages` | 我的套餐 |

### 管理员接口（JWT + Admin）

| 路径前缀 | 说明 |
|----------|------|
| `/api/admin/users` | 用户管理 |
| `/api/admin/api-keys` | API Key 管理（全局） |
| `/api/admin/accounts` | 账户管理 |
| `/api/admin/account-groups` | 账户分组 |
| `/api/admin/oauth` | OAuth 授权 |
| `/api/admin/logs` | 请求日志 |
| `/api/admin/operation-logs` | 操作日志 |
| `/api/admin/models` | 模型管理 |
| `/api/admin/cache` | 缓存管理 |
| `/api/admin/configs` | 系统配置 |
| `/api/admin/packages` | 套餐管理 |
| `/api/admin/user-packages` | 用户套餐 |
| `/api/admin/proxy-configs` | 代理配置 |
| `/api/admin/monitor` | 系统监控 |
| `/api/admin/system-logs` | 系统日志 |
| `/api/admin/client-filter` | 客户端过滤 |

---

## 通用响应格式

以下“通用响应格式”主要用于后台/管理类接口（`/api/*`）。

代理转发接口（`/claude/*`、`/openai/*`、`/gemini/*`、`/responses`）为上游透传，响应体以各平台原始格式为准；`/health` 直接返回 `{ "status": "ok" }`。

### 成功响应（HTTP 200/201）

```json
{
  "code": 0,
  "message": "success",
  "data": { }
}
```

### 错误响应（HTTP 4xx/5xx）

```json
{
  "code": 400,
  "message": "错误信息"
}
```

### 分页响应

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [...],
    "total": 100,
    "page": 1
  }
}
```

---

## HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 401 | 未认证 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 429 | 请求限流 |
| 500 | 服务器错误 |
| 502 | 上游错误 |
| 503 | 服务不可用 |
