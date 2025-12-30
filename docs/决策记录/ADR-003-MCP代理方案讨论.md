# ADR-003: MCP代理方案讨论

## 状态
待讨论

## 日期
2025-12-26

## 背景

用户希望通过go-aiproxy服务器转发智谱GLM的MCP功能（联网搜索、视觉理解等），让终端用户无需单独配置MCP服务器。

### 当前架构

```
Claude Code 发请求
    │
    ├── LLM API (/v1/messages)
    │   └── 用户代理服务器 (go-aiproxy) → GLM API
    │
    └── MCP 工具调用
        └── 用户需要自行配置 → 智谱 MCP 服务器
```

### 智谱MCP配置命令（用户自行配置方式）
```bash
claude mcp add -s user -t http web-search-prime https://open.bigmodel.cn/api/mcp/web_search_prime/mcp --header "Authorization: Bearer 用户的智谱API-KEY"
```

## 问题

1. **用户需要额外配置**：除了配置代理服务器，还需要单独配置MCP
2. **用户需要智谱Key**：每个用户需要自己申请智谱API Key
3. **管理分散**：LLM和MCP分开管理，不便于统一控制

## 可选方案

### 方案A: MCP代理（推荐）

在go-aiproxy中添加MCP转发功能，用户只需配置一个地址。

**架构**：
```
Claude Code
    │
    ├── /v1/messages → go-aiproxy → GLM API
    │
    └── /mcp/*       → go-aiproxy → 智谱 MCP 服务器
                         (服务器加上智谱Key)
```

**用户配置**：
```bash
# LLM (已有)
export ANTHROPIC_BASE_URL="https://你的服务器"
export ANTHROPIC_API_KEY="用户API Key"

# MCP (新增，一行命令)
claude mcp add -s user -t http web-search https://你的服务器/mcp/web_search
```

**优点**：
- 用户无需智谱Key，服务器统一管理
- 代码量小（约100行）
- 可统计MCP调用量和费用

**缺点**：
- 用户仍需配置一行MCP命令
- MCP费用由服务器承担

**代码量估算**：
| 组件 | 代码量 | 说明 |
|------|--------|------|
| MCP Handler | ~80行 | HTTP反向代理 + 加Key |
| 路由配置 | ~5行 | 添加 `/mcp/*` 路由 |
| 系统配置 | ~10行 | 存储智谱MCP的API Key |
| **总计** | **~100行** | |

**核心代码逻辑**：
```go
func MCPProxy(c *gin.Context) {
    // 1. 读取请求体
    body := c.Request.Body

    // 2. 转发到智谱MCP（加上服务器的Key）
    req := http.NewRequest("POST", "https://open.bigmodel.cn/api/mcp/web_search_prime/mcp", body)
    req.Header.Set("Authorization", "Bearer 服务器的智谱Key")
    req.Header.Set("MCP-Protocol-Version", c.GetHeader("MCP-Protocol-Version"))

    // 3. 透传所有其他请求头
    for key, values := range c.Request.Header {
        if key != "Authorization" && key != "Host" {
            req.Header[key] = values
        }
    }

    // 4. 返回响应（支持SSE流式）
    resp := client.Do(req)
    io.Copy(c.Writer, resp.Body)
}
```

### 方案B: 服务器端Agent Loop（用户零配置）

在响应中检测tool_use，服务器自动执行工具并返回最终结果。

**架构**：
```
Claude Code → go-aiproxy → GLM API
                  │
                  ▼ 检测到 tool_use
                  │
                  ▼ 服务器调用 MCP 执行工具
                  │
                  ▼ 把结果再发给 GLM
                  │
                  ▼ 返回最终结果给用户
```

**优点**：
- 用户完全零配置
- 对用户完全透明

**缺点**：
- 代码量大（约500-800行）
- 流式响应处理复杂
- 需要处理多轮工具调用
- 增加延迟

**代码量估算**：
| 组件 | 代码量 | 说明 |
|------|--------|------|
| 响应拦截中间件 | ~150行 | 检测tool_use |
| Agent Loop逻辑 | ~200行 | 多轮调用处理 |
| MCP执行器 | ~100行 | 工具执行 |
| 流式处理 | ~200行 | 流式响应缓冲和处理 |
| 错误处理 | ~100行 | 异常情况处理 |
| **总计** | **~500-800行** | |

### 方案C: 混合模式

用户可选择：
1. 用自己的Key直连智谱MCP（零费用）
2. 用服务器代理MCP（可能收费或限制调用量）

## 安全考虑

### Key安全性（方案A）

**智谱Key不会泄露给用户**：

```
用户 ──────────────────► 服务器 ──────────────────► 智谱MCP
     用户的API Key          服务器加上智谱Key
     (用于认证用户)          (用户看不到)
```

| 环节 | 用户能看到什么 |
|------|---------------|
| 请求发送 | 只有自己的API Key，发到服务器 |
| 服务器转发 | 服务器在后端加上智谱Key，用户无感知 |
| 响应返回 | 只有MCP结果数据，不含任何Key |

智谱Key存在服务器的数据库/配置文件里，用户完全接触不到。

## MCP协议说明

MCP (Model Context Protocol) 基于 JSON-RPC 2.0，使用 HTTP 传输：

**请求格式**：
- 方法：POST
- Content-Type: application/json
- 必需Header: `MCP-Protocol-Version: 2025-06-18`
- Accept: `application/json, text/event-stream`

**响应格式**：
- 非流式：application/json
- 流式：text/event-stream (SSE)

## 智谱MCP服务列表

| 服务 | 端点 | 说明 |
|------|------|------|
| 联网搜索 | `/api/mcp/web_search_prime/mcp` | Pro套餐包含 |
| 视觉理解 | `/api/mcp/vision/mcp` | 图像理解 |

## 参考资料

- [智谱AI开放文档 - 接入 Claude Code](https://docs.bigmodel.cn/cn/guide/develop/claude)
- [MCP协议规范](https://modelcontextprotocol.io/specification/2025-06-18/basic/transports)
- [claude code + GLM 配置教程](https://zhuanlan.zhihu.com/p/1957370685748938170)

## 决策

待定。建议采用**方案A（MCP代理）**，因为：
1. 代码量小，实现简单
2. Key安全，用户无法获取
3. 便于统一管理和计费
4. 用户只需多配置一行命令

## 后续步骤

如果采用方案A：
1. 在系统配置中添加智谱MCP API Key配置项
2. 创建MCP Handler（`/mcp/web_search`, `/mcp/vision`）
3. 添加路由
4. 测试MCP转发功能
5. 更新用户文档
