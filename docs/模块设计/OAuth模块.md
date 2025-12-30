# OAuth 模块设计文档

## 概述

OAuth 模块负责处理多平台（Claude、OpenAI、Gemini）的授权认证流程，支持 OAuth 2.0 + PKCE 标准流程和 SessionKey Cookie 认证两种方式。

## 功能跟踪

### 后端功能 (`internal/handler/oauth.go`, 474 行)

| 功能 | 函数名 | 文件位置 | 行号 | 说明 |
|------|--------|----------|------|------|
| 创建处理器 | `NewOAuthHandler` | oauth.go | 53-55 | 初始化 OAuth 处理器实例 |
| 生成授权URL | `GenerateURL` | oauth.go | 58-124 | 生成 PKCE 授权 URL，支持 Claude/OpenAI/Gemini |
| 交换授权码 | `Exchange` | oauth.go | 127-175 | 使用授权码换取 Access Token |
| Cookie授权 | `CookieAuth` | oauth.go | 178-207 | 使用 SessionKey 自动完成 OAuth 流程 |
| SessionKey认证 | `authenticateWithSessionKey` | oauth.go | 210-238 | 完整的 SessionKey → Token 流程 |
| 获取组织信息 | `getOrganizationInfo` | oauth.go | 241-290 | 从 Claude 获取用户组织 UUID 和能力 |
| Cookie授权请求 | `authorizeWithCookie` | oauth.go | 293-364 | 使用 Cookie 向 Claude 发起授权请求 |
| Token交换 | `exchangeClaudeToken` | oauth.go | 367-416 | 向 Claude OAuth 服务器交换 Token |
| HTTP客户端 | `createHTTPClient` | oauth.go | 419-453 | 创建支持代理的 HTTP 客户端 |
| PKCE生成 | `generatePKCE` | oauth.go | 456-467 | 生成 PKCE verifier 和 challenge |
| 随机字符串 | `generateRandomString` | oauth.go | 470-474 | 生成指定长度随机字符串 |

### 路由注册 (`internal/handler/routes.go`)

| API | 方法 | 路径 | 处理函数 | 行号 |
|-----|------|------|----------|------|
| 生成授权URL | POST | `/api/admin/oauth/generate-url` | `GenerateURL` | 91 |
| 交换授权码 | POST | `/api/admin/oauth/exchange` | `Exchange` | 92 |
| Cookie认证 | POST | `/api/admin/oauth/cookie-auth` | `CookieAuth` | 93 |

路由组初始化: routes.go:20 (`oauthHandler := NewOAuthHandler()`)
路由组定义: routes.go:89-94

### 前端功能

#### OAuthFlow.vue (827 行)

| 功能 | 函数名 | 行号 | 说明 |
|------|--------|------|------|
| 生成授权URL | `generateAuthUrl` | 428-455 | 调用后端 API 生成 OAuth URL |
| 重新生成URL | `regenerateAuthUrl` | 457-463 | 重新生成新的授权 URL |
| 复制授权URL | `copyAuthUrl` | 465-489 | 复制 URL 到剪贴板 |
| 交换授权码 | `exchangeCode` | 491-514 | 使用授权码交换 Token |
| Cookie认证 | `handleCookieAuth` | 516-572 | SessionKey 批量认证 |
| 认证方式切换 | `onAuthMethodChange` | 574-581 | 切换 OAuth/Cookie 模式 |

#### 组件模板区域

| 区域 | 行号范围 | 说明 |
|------|----------|------|
| Claude OAuth | 3-176 | Claude 平台认证界面 |
| Gemini OAuth | 177-242 | Gemini 平台认证界面 |
| OpenAI OAuth | 243-322 | OpenAI 平台认证界面 |
| 操作按钮 | 323-334 | 通用操作按钮 |

#### AccountForm.vue (1167 行)

| 功能 | 位置 | 行号 | 说明 |
|------|------|------|------|
| Claude添加类型选择 | template | 90-131 | OAuth/SessionKey/SetupToken 选择卡片 |
| OpenAI添加类型选择 | template | 132-166 | OAuth/APIKey 选择卡片 |
| Gemini添加类型选择 | template | 167-201 | OAuth/APIKey 选择卡片 |
| OpenAI APIKey配置 | template | 203-226 | API Key 输入表单 |
| Gemini APIKey配置 | template | 228-251 | API Key 输入表单 |
| OAuth组件 | template | 358-366 | OAuthFlow 组件引用 |
| needsOAuth计算属性 | script | 582-585 | 判断是否需要 OAuth 流程 |
| handleOAuthSuccess | script | 657-680 | OAuth 成功回调处理 |

#### API 定义 (`web/src/api/index.js`)

| 函数名 | 行号 | API 路径 | 说明 |
|--------|------|----------|------|
| `generateOAuthUrl` | 80 | `/admin/oauth/generate-url` | 生成授权 URL |
| `exchangeOAuthCode` | 81 | `/admin/oauth/exchange` | 交换授权码 |
| `oauthByCookie` | 82 | `/admin/oauth/cookie-auth` | Cookie 认证 |

## 数据结构

### OAuthSession (oauth.go:35-41)

```go
type OAuthSession struct {
    Verifier  string    `json:"verifier"`   // PKCE verifier
    Challenge string    `json:"challenge"`  // PKCE challenge
    State     string    `json:"state"`      // 防 CSRF 状态码
    Platform  string    `json:"platform"`   // 平台标识
    CreatedAt time.Time `json:"created_at"` // 创建时间
}
```

### ProxyConfig (oauth.go:44-50)

```go
type ProxyConfig struct {
    Type     string `json:"type"`     // socks5, http, https
    Host     string `json:"host"`
    Port     int    `json:"port"`
    Username string `json:"username,omitempty"`
    Password string `json:"password,omitempty"`
}
```

## 认证流程

### OAuth 2.0 + PKCE 流程

```
1. 前端调用 generateOAuthUrl(platform)
   ↓
2. 后端生成 PKCE (verifier, challenge) 和 state
   ↓
3. 后端返回授权 URL + session_id
   ↓
4. 用户在新窗口完成授权，获得 code
   ↓
5. 前端调用 exchangeOAuthCode(platform, code, session_id)
   ↓
6. 后端使用 code + verifier 换取 token
   ↓
7. 返回 access_token, refresh_token
```

### SessionKey Cookie 流程

```
1. 用户提供 Claude sessionKey
   ↓
2. 前端调用 oauthByCookie(platform, session_key)
   ↓
3. 后端 getOrganizationInfo() - 获取组织 UUID
   ↓
4. 后端 generatePKCE() - 生成 PKCE
   ↓
5. 后端 authorizeWithCookie() - 自动获取授权码
   ↓
6. 后端 exchangeClaudeToken() - 交换 Token
   ↓
7. 返回 access_token + organization_uuid + capabilities
```

## 常量配置 (oauth.go:21-27)

| 常量 | 值 | 说明 |
|------|-----|------|
| `ClaudeAIURL` | `https://claude.ai` | Claude 网站地址 |
| `OAuthClientID` | `9cb5d115-7b98-4147-a0a2-f0ff7631afc7` | OAuth 客户端 ID |
| `OAuthRedirectURI` | `https://claude.ai/oauth/callback` | 回调地址 |
| `OAuthTokenURL` | `https://claude.ai/oauth/token` | Token 端点 |
| `OAuthAuthorizeURL` | `https://claude.ai/oauth/%s/authorize` | 授权端点模板 |

## 代理支持

支持三种代理类型 (createHTTPClient:419-453):

1. **SOCKS5** - 使用 `golang.org/x/net/proxy` 包
2. **HTTP** - 使用 `http.ProxyURL`
3. **HTTPS** - 使用 `http.ProxyURL`

代理配置通过请求参数传递，支持用户名密码认证。

## 错误处理

| 错误场景 | 处理方式 | 位置 |
|----------|----------|------|
| 无效请求参数 | 返回 400 | GenerateURL:64-67, Exchange:135-138, CookieAuth:185-188 |
| 无效会话ID | 返回 400 | Exchange:141-145 |
| 会话过期(10分钟) | 返回 400 | Exchange:151-155 |
| 不支持的平台 | 返回 400 | GenerateURL:113-116, Exchange:164-167, CookieAuth:204-206 |
| Token交换失败 | 返回 500 | Exchange:169-172, CookieAuth:199-202 |
| 组织信息获取失败 | 返回错误 | authenticateWithSessionKey:213-215 |
| HTTP请求失败 | 返回错误 | getOrganizationInfo:261-263, authorizeWithCookie:327-330 |
