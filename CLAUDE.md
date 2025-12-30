# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 提供在此代码库中工作的指导。

## 项目概述

Go-AIProxy 是一个企业级 AI API 代理服务，提供统一的多平台 AI API（Claude、OpenAI、Gemini）访问接口，支持账户池管理、负载均衡、计费和监控功能。

**技术架构**：Go 后端 (Gin) + Vue 3 前端 (Element Plus) + MySQL 8.0

## 开发规范（必须遵守）

### 代码文件注释规范（必须）

**所有项目代码文件**（`.go`、`.vue`、`.js`、`.ts` 等）的开头必须添加文件说明注释。

#### Go 文件模板

```go
/*
 * 文件作用：[一句话说明这个文件干什么]
 * 负责功能：
 *   - 功能1
 *   - 功能2
 *   - 功能3
 * 重要程度：⭐⭐⭐⭐⭐ 核心 / ⭐⭐⭐⭐ 重要 / ⭐⭐⭐ 一般 / ⭐⭐ 辅助 / ⭐ 工具
 * 依赖模块：[列出主要依赖的内部模块]
 */
package xxx
```

#### Go 文件示例

```go
/*
 * 文件作用：账户调度器，负责从多个AI平台账户中选择合适的账户处理请求
 * 负责功能：
 *   - 账户选择（按模型、按类型、按权重）
 *   - 会话粘性（同一会话路由到同一账户）
 *   - AllowedModels 过滤
 *   - ModelMapping 映射处理
 *   - 账户状态管理（错误标记、限流恢复）
 * 重要程度：⭐⭐⭐⭐⭐ 核心（代理转发的核心调度逻辑）
 * 依赖模块：cache, model, repository
 */
package scheduler
```

#### Vue/JS 文件模板

```vue
<!--
 * 文件作用：[一句话说明这个文件干什么]
 * 负责功能：
 *   - 功能1
 *   - 功能2
 * 重要程度：⭐⭐⭐⭐⭐ 核心 / ⭐⭐⭐⭐ 重要 / ⭐⭐⭐ 一般
 * 依赖组件：[列出主要依赖的组件]
-->
```

#### 重要程度标准

| 等级 | 符号 | 说明 | 示例 |
|------|------|------|------|
| 核心 | ⭐⭐⭐⭐⭐ | 系统无法运行的核心模块 | scheduler.go, proxy.go |
| 重要 | ⭐⭐⭐⭐ | 主要业务功能 | account.go, api_key.go |
| 一般 | ⭐⭐⭐ | 辅助业务功能 | usage.go, cache.go |
| 辅助 | ⭐⭐ | 配置、工具类 | config.go, logger.go |
| 工具 | ⭐ | 纯工具函数 | utils.go, helpers.go |

### 开发日志规范

**目录**: `docs/开发日志/`

**命名**: `YYYY-MM-DD.md`（按实际工作日期）

**更新时机**: 每日工作结束时更新，**每天创建新文件**

**模板**:
```markdown
# 开发日志 YYYY-MM-DD

## 今日工作

### 1. 功能/任务名称

**需求**: 简述需求背景
**实现方案**: 简述技术方案
**修改文件**:
- `path/to/file.go` - 修改说明

---

## 明日计划

### 1. 任务名称（优先级高）
- [ ] 具体子任务1
- [ ] 具体子任务2
```

### BUG 处理规范

**目录**: `docs/BUG处理/`

**命名**: `YYYY-MM-DD-问题简述.md`

**类型分类**: BUG修复 / 功能优化

**模板**:
```markdown
# BUG/优化: 问题简述

> 日期: YYYY-MM-DD
> 类型: BUG修复 / 功能优化

## 问题描述
**预期行为**: xxx
**实际行为**: xxx

## 修复方案
**文件**: `path/to/file.go`

## 验证方法
1. 验证步骤
2. 预期结果
```

### 决策记录规范（ADR）

**目录**: `docs/决策记录/`

**命名**: `ADR-XXX-决策标题.md`

**更新时机**: 做出重要技术决策时

**模板**:
```markdown
# ADR-XXX: 决策标题

## 状态
已采纳 / 已废弃 / 待讨论

## 背景
为什么需要做这个决策？

## 可选方案

### 方案A: xxx
- 优点：
- 缺点：

## 决策
选择方案X

## 原因
为什么选择这个方案？
```

### 文档更新检查清单

**每次完成功能开发/修复后，必须逐项检查以下文档是否需要更新：**

| 序号 | 文档 | 必须/可选 |
|------|------|-----------|
| 0 | **代码文件注释** | **必须** |
| 1 | `docs/开发日志/YYYY-MM-DD.md` | **必须** |
| 2 | `docs/代码索引.md` | **必须** |
| 3 | `docs/功能模块索引.md` | **必须** |
| 4 | `docs/接口文档/相关接口.md` | API变更时必须 |
| 5 | `docs/模块设计/相关模块.md` | 模块变更时必须 |
| 6 | `docs/故障排查手册.md` | 可选 |
| 7 | `docs/决策记录/ADR-XXX.md` | 重大决策时必须 |
| 8 | `docs/BUG处理/YYYY-MM-DD-xxx.md` | BUG修复时必须 |

## 开发命令

### 后端 (Go)

```bash
# 构建后端二进制文件
make build
# 输出: bin/server

# 直接运行后端（开发模式）
make run
# 服务启动在 8080 端口

# 完整生产构建（前端 + 后端）
make all

# 运行测试
make test

# 格式化代码
make fmt

# 代码检查
make lint

# 热重载开发（需要安装 air）
make dev
```

### 前端 (Vue 3)

```bash
# 安装依赖
make web-deps

# 开发服务器（代理 API 到后端）
make web-dev
# 前端: http://localhost:3000
# API 代理: → http://localhost:8080

# 生产构建
make web-build
# 输出: web/dist/* → internal/handler/dist/
```

### 配置文件

**后端配置**: `configs/config.yaml`
- 服务端口（默认: 8080）
- MySQL 连接
- JWT 密钥
- 缓存 TTL 设置
- 日志目录和级别

**前端配置**: `web/vite.config.js`
- 开发服务器端口: 3000
- API 代理: `/api` → `http://localhost:8080`

**默认管理员账号**: `admin` / `mowen3078697@`

## 核心架构

### 请求流程

```
客户端请求（带 x-api-key）
  ↓
API Key 中间件（验证身份，注入用户上下文）
  ↓
平台处理器（如 ClaudeMessages）
  ↓
调度器.SelectAccount()
  - 检查会话缓存（通过 x-session-id 绑定）
  - 按 AllowedModels 过滤
  - 按 ModelMapping 过滤
  - 加权选择（priority × weight）
  - 将会话绑定到账户
  ↓
适配器.Send/SendStream()
  - 构建上游请求
  - 应用账户的 ModelMapping
  - 调用上游 API
  - 处理流式响应（SSE）
  ↓
响应处理
  - 提取 token 使用量
  - 应用定价倍数
  - 更新统计数据（异步）
  - 返回给客户端
```

### 平台路由

**设计决策**：使用平台专用路由，而非统一的 `/v1/*`

```
/claude/v1/messages         → Claude 适配器
/openai/v1/chat/completions → OpenAI 适配器
/responses                  → OpenAI Responses API (Codex CLI)
/gemini/v1/chat             → Gemini 适配器
```

**原因**：平台分离清晰，调试更容易，可观测性更好

### 适配器模式

每个 AI 平台在 `internal/proxy/adapter/` 中都有对应的适配器：

- `claude.go`: Claude Official + Console
- `openai.go`: OpenAI API
- `azure.go`: Azure OpenAI
- `gemini.go`: Google Gemini
- `bedrock.go`: AWS Bedrock
- `openai_responses.go`: Codex CLI 支持

**接口定义**：
```go
type Adapter interface {
    Name() string
    Platform() string
    SupportedTypes() []string
    Send(ctx, account, req) (*Response, error)
    SendStream(ctx, account, req, writer) (*StreamResult, error)
}
```

### 调度器与账户选择

位于 `internal/proxy/scheduler/`：

**核心功能**：
1. **会话粘性**：通过 `x-session-id` header 将会话绑定到账户（对 Claude Code 至关重要）
2. **加权选择**：优先级 × 权重算法
3. **模型过滤**：账户级别的 `AllowedModels` 和 `ModelMapping`
4. **健康管理**：从限流中自动恢复
5. **平台检测**：从模型名称自动检测平台

**账户状态**：`valid`（正常）、`rate_limited`（限流）、`invalid`（无效）、`overloaded`（过载）、`token_expired`（令牌过期）、`suspended`（暂停）、`banned`（封禁）、`disabled`（禁用）

### 认证系统（两套独立系统）

1. **管理后台**：JWT 认证，通过 `/api/login`
   - 受保护的路由需要 `Authorization: Bearer <token>`
   - 中间件：`middleware.JWTAuth()`

2. **代理 API**：API Key 认证
   - Header: `x-api-key` 或 `Authorization: Bearer <key>`
   - 验证 `api_keys` 表
   - 中间件：`middleware.APIKeyAuth()`

### 计费系统

**多级定价**：
```
最终价格 = 基础价格 × 全局倍率 × 用户倍率 × 套餐倍率
```

**统计方式**：
- 内存缓存（`sync.Map`）：实时计数器
- MySQL：持久化每日汇总（`daily_usages` 表）
- 请求日志：包含请求/响应正文的详细日志

### 错误处理与重试

**智能错误匹配**（`internal/errormatch/`）：
- HTTP 状态码 → 账户状态
- 错误关键词 → 错误类型
- 可配置的规则（存储在数据库中）

**重试逻辑**：可配置的最大重试次数（默认 3 次），失败后排除账户

## 目录结构

```
go-aiproxy/
├── cmd/server/              # 程序入口
├── internal/
│   ├── handler/             # HTTP 处理器（平台路由）
│   ├── middleware/          # JWT、API Key、并发控制等中间件
│   ├── model/               # 数据模型（User、Account、APIKey 等）
│   ├── repository/          # 数据访问层（GORM）
│   ├── service/             # 业务逻辑（用户、账户等）
│   ├── proxy/               # 代理核心系统
│   │   ├── adapter/         # 平台适配器
│   │   └── scheduler/       # 账户调度器
│   ├── cache/               # 内存缓存（会话、数据）
│   └── errormatch/          # 错误匹配系统
├── pkg/
│   ├── logger/              # 结构化日志（zap）
│   ├── response/            # HTTP 响应助手
│   └── utils/               # 通用工具
├── web/                     # Vue 3 前端
│   ├── src/
│   │   ├── api/index.js     # 集中式 API 客户端（Alova）
│   │   ├── stores/          # Pinia 状态存储
│   │   ├── router/          # Vue Router 配置
│   │   ├── views/           # 页面组件
│   │   ├── layouts/         # 布局模板
│   │   └── components/      # 可复用组件
│   └── dist/                # 构建后的前端
├── configs/                 # YAML 配置
└── docs/                    # 完整文档
    ├── 开发日志/            # 每日开发记录
    ├── BUG处理/             # BUG修复和优化记录
    ├── 决策记录/            # 重要技术决策（ADR）
    ├── 模块设计/            # 各模块详细设计
    ├── 接口文档/            # API 接口文档
    ├── 代码索引.md          # 代码结构完整索引
    ├── 功能模块索引.md       # 功能模块索引
    └── 故障排查手册.md       # 问题速查手册
```

## 重要设计决策

1. **平台路由而非统一路由**：分离清晰，调试方便
2. **账户级模型映射**：每个账户可以有独立的模型名称转换
3. **会话粘性**：通过 `x-session-id` 绑定会话，保持对话连续性
4. **内存缓存而非 Redis**：简化部署，单实例足够
5. **两套认证系统**：管理后台用 JWT，代理用 API Key（不同使用场景）

## 支持的平台

### Claude
- `claude-official`：OAuth（access token + session key）
- `claude-console`：API Key 模式
- `bedrock`：AWS Bedrock
- `CCR`：第三方 Claude

### OpenAI
- `openai`：官方 API
- `azure-openai`：Azure OpenAI 服务
- `openai-responses`：Responses API（Codex CLI、Claude Code）

### Gemini
- `gemini`：基于 OAuth
- `gemini-api`：API Key 模式

## 核心数据表

- `users`：用户账户及角色
- `api_keys`：用户生成的 API Key
- `accounts`：AI 平台账户
- `ai_models`：模型配置与定价
- `request_logs`：请求/响应日志
- `daily_usages`：每日使用汇总
- `system_configs`：键值配置

## 特色功能

1. **健康检查系统**：后台账户验证，自动恢复
2. **客户端过滤**：阻止/限制特定客户端类型
3. **并发控制**：每用户和每账户的并发限制
4. **OpenAI Responses API**：支持 Codex CLI 和 Claude Code
5. **代理支持**：每账户或全局的 HTTP/SOCKS5 代理
6. **审计日志**：完整的管理员操作记录

## 前端架构 (Vue 3)

**技术栈**：Vue 3.4+（Composition API）、Vite 5.x、Element Plus 2.6+、Pinia、Alova 3.x

**主要页面**：
- 管理员：系统监控、账户管理、模型管理、用户管理、API Key、请求日志、缓存管理
- 用户：个人仪表板、我的 API Key、我的套餐、使用记录

**API 客户端**：集中在 `web/src/api/index.js`（Alova 实例，自动注入 JWT）

## 环境变量（可选）

可覆盖 `config.yaml`：
- `PORT`：服务端口（默认 8080）
- `DB_HOST`、`DB_PORT`、`DB_USER`、`DB_PASSWORD`、`DB_NAME`

## 开发流程

1. 启动 MySQL：确保数据库 `aiproxy` 存在
2. 如需要，配置 `configs/config.yaml`
3. 运行后端：`make run`（端口 8080）
4. 运行前端：`make web-dev`（端口 3000，代理到 8080）
5. 访问管理后台：http://localhost:3000
6. 生产构建：`make all`

## 热重载开发

先安装 air：`go install github.com/air-verse/air@latest`
然后使用：`make dev`

## 快速参考

| 文档 | 说明 | 适用场景 |
|------|------|----------|
| [故障排查手册](docs/故障排查手册.md) | 问题现象→原因→解决方法 | **遇到问题先看这个** |
| [功能模块索引](docs/功能模块索引.md) | 功能列表、关键代码、配置项 | 修改功能时查看 |
| [代码索引](docs/代码索引.md) | 完整代码结构、变更记录 | 查找代码位置 |
