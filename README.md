# Go-AIProxy

一个 Go 语言实现的 AI 代理服务，支持多平台（Claude、OpenAI、Gemini）账户管理和统一 API 接口。

[![QQ群](https://img.shields.io/badge/QQ群-加入交流-orange?logo=tencentqq&style=flat-square)](https://qm.qq.com/q/iJ4bHLlMEa)

## 联系交流

- **QQ 交流群**：[点击加入群聊【go-proxy-pro】](https://qm.qq.com/q/iJ4bHLlMEa) （或访问网站主页扫描二维码）
- **GitHub Issues**：[提交问题](https://github.com/suiyuebaobao/go-proxy-pro/issues)
- **GitHub Discussions**：[参与讨论](https://github.com/suiyuebaobao/go-proxy-pro/discussions)

## 功能特性

- **多平台支持**：Claude (Official/Console/CCR/Bedrock)、OpenAI (API/Azure/Responses)、Gemini
- **平台专用路由**：按平台区分的 API 端点，清晰简洁
- **OpenAI Responses API**：支持 Codex CLI、Claude Code 等客户端的 `/responses` 接口
- **账户池管理**：支持多账户轮询、负载均衡、故障转移
- **用户 API Key**：用户可生成自己的 API Key 调用服务
- **权限控制**：平台/模型级别的访问权限控制
- **使用统计**：请求次数、Token 消耗、费用统计

## 快速开始

### 1. 编译运行

```bash
# 编译
go build -o aiproxy ./cmd/server

# 运行
./aiproxy
```

服务默认监听 `8080` 端口。

### Docker 部署 (推荐)

使用 Docker Compose 可以一键部署完整的应用环境（包括 MySQL 数据库）：

```bash
# 克隆项目
git clone https://github.com/suiyuebaobao/go-proxy-pro.git
cd go-proxy-pro/go-aiproxy

# 启动服务（自动拉取镜像、构建、启动）
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down

# 停止并删除数据卷（⚠️ 会删除数据库数据）
docker-compose down -v
```

服务启动后：
- **Web 管理界面**: http://localhost:8080
- **API 接口**: http://localhost:8080/claude/v1/messages 等
- **MySQL 数据库**: localhost:3306

#### Docker 环境变量

可在 `docker-compose.yml` 中修改以下配置：

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `MYSQL_ROOT_PASSWORD` | `go-aiproxy-root` | MySQL root 密码 |
| `MYSQL_DATABASE` | `aiproxy` | 数据库名 |
| `MYSQL_USER` | `aiproxy` | MySQL 用户名 |
| `MYSQL_PASSWORD` | `aiproxy-password` | MySQL 密码 |
| `PORT` | `8080` | 应用端口 |
| `JWT_SECRET` | `go-aiproxy-jwt-secret-change-in-production` | JWT 密钥（生产环境请修改） |

#### 生产环境部署建议

1. **修改默认密码**：修改 `docker-compose.yml` 中的所有默认密码
2. **持久化数据**：数据已通过 Docker volumes 持久化，升级不会丢失数据
3. **健康检查**：应用内置健康检查 `/health` 端点
4. **日志管理**：日志文件挂载到 `./logs` 目录

### 2. 默认管理员账号

- 用户名: `admin`
- 密码: `admin123`

首次登录后请及时修改密码。

### 3. 配置流程

1. **添加 AI 账户**：进入"账户管理"，添加 Claude/OpenAI/Gemini 等账户
2. **创建 API Key**：进入"我的 API Key"，创建用于调用服务的 Key
3. **开始使用**：使用 API Key 调用代理接口

## API 使用指南

### Base URL 配置

| 平台 | Base URL | 完整端点 |
|------|----------|----------|
| Claude | `http://域名/claude/` | `/claude/v1/messages` |
| OpenAI | `http://域名/openai/` | `/openai/v1/chat/completions` |
| Gemini | `http://域名/gemini/` | `/gemini/v1/chat` |

客户端配置时只需填写 Base URL，客户端会自动拼接后续路径。

### Claude 接口

```bash
curl http://localhost:8080/claude/v1/messages \
  -H "x-api-key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-20250514",
    "messages": [{"role": "user", "content": "Hello!"}],
    "max_tokens": 1024
  }'
```

### OpenAI 接口

```bash
curl http://localhost:8080/openai/v1/chat/completions \
  -H "x-api-key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": false
  }'
```

### Gemini 接口

```bash
curl http://localhost:8080/gemini/v1/chat \
  -H "x-api-key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-2.5-flash",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### OpenAI Responses API (Codex)

支持 OpenAI Responses API，兼容 Claude Code / Codex CLI 等客户端：

```bash
curl http://localhost:8080/responses \
  -H "x-api-key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.1-codex-max",
    "input": "Write a hello world function",
    "stream": true
  }'
```

也支持 `/v1/responses` 路径。

## 项目结构

```
go-aiproxy/
├── cmd/server/          # 程序入口
├── internal/
│   ├── handler/         # HTTP 处理器
│   ├── middleware/      # 中间件 (JWT、API Key 认证等)
│   ├── model/           # 数据模型
│   ├── repository/      # 数据访问层
│   ├── service/         # 业务逻辑层
│   └── proxy/           # 代理核心
│       ├── adapter/     # 各平台适配器
│       └── scheduler/   # 调度器
├── pkg/                 # 公共工具包
└── web/                 # 前端 (Vue 3 + Element Plus)
```

## 环境要求

- Go 1.21+
- MySQL 8.0+
- Node.js 18+ (前端开发)

## 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `PORT` | `8080` | 服务端口 |
| `DB_HOST` | `localhost` | MySQL 主机 |
| `DB_PORT` | `3306` | MySQL 端口 |
| `DB_USER` | `root` | MySQL 用户名 |
| `DB_PASSWORD` | - | MySQL 密码 |
| `DB_NAME` | `aiproxy` | 数据库名 |

## 前端开发

```bash
cd web
npm install
npm run dev
```

## 技术栈

**后端**
- Go 1.21+, Gin 1.10+
- MySQL 8.0+ (GORM)
- 内存缓存 (sync.Map)
- JWT + API Key 认证

**前端**
- Vue 3.4+ (Composition API)
- Vite 5.x
- Element Plus 2.6+
- Alova 3.x (HTTP 客户端)
- Font Awesome 6.x

## License

MIT
