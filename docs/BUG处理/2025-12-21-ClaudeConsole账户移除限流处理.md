# ClaudeConsole (API Key) 账户移除限流处理

> 修复日期：2025-12-21
> 类型：功能调整
> 严重程度：低（优化用户体验）

## 问题描述

用户反馈：对于使用 API Key 模式（`ClaudeConsole` 类型）的账户，不应该进行限流处理。

**原因**：
- API Key 模式的限流由上游 Anthropic API 直接控制
- 我们的代理不需要额外进行限流状态标记
- 限流标记可能导致账户被错误地暂停使用

## 账户类型说明

| 账户类型 | 认证方式 | 限流处理 |
|----------|----------|----------|
| `claude-official` | OAuth Token / SessionKey | 需要（我们管理 Token 刷新和限流恢复） |
| `claude-console` | Anthropic API Key | 不需要（由上游直接控制） |

## 解决方案

### 1. 不提取限流响应头

**文件**: `internal/proxy/adapter/claude.go`

在 `Send` 和 `SendStream` 方法中，只对非 ClaudeConsole 类型账户提取限流头：

```go
// Send 方法（非流式）
if account.Type != model.AccountTypeClaudeConsole {
    response.Headers = extractRateLimitHeaders(resp.Header)
}

// SendStream 方法（流式）
if account.Type != model.AccountTypeClaudeConsole {
    result.Headers = extractRateLimitHeaders(resp.Header)
}
```

### 2. 不标记限流状态

**文件**: `internal/proxy/scheduler/scheduler.go`

在 `MarkAccountErrorWithReset` 方法中，跳过 ClaudeConsole 类型账户的限流处理：

```go
func (s *Scheduler) MarkAccountErrorWithReset(accountID uint, accountType string, err error, rateLimitResetAt *time.Time) {
    // API Key 模式（ClaudeConsole）不做限流处理，直接透传上游错误
    if accountType == model.AccountTypeClaudeConsole {
        // API Key 模式只记录错误，不改变状态
        s.repo.IncrementErrorCount(accountID)
        return
    }

    // 其他类型账户正常处理限流逻辑...
}
```

### 3. 清理历史数据

对于已存在的 ClaudeConsole 账户，可能有历史的限流标记，需要清理：

```sql
UPDATE accounts
SET rate_limit_reset_at = NULL, status = 'valid'
WHERE type = 'claude-console' AND rate_limit_reset_at IS NOT NULL;
```

## 修改文件清单

| 文件 | 修改内容 |
|------|----------|
| `internal/proxy/adapter/claude.go` | 第98-101行、第154-157行：跳过 API Key 模式的限流头提取 |
| `internal/proxy/scheduler/scheduler.go` | 第351-356行：跳过 API Key 模式的限流状态标记 |

## 验证方法

1. 使用 ClaudeConsole 类型账户发送请求
2. 即使上游返回 429 错误，账户也不应被标记为限流状态
3. Web 界面不应显示该账户的限流倒计时
4. 账户应始终保持 `valid` 状态（除非被手动禁用）

## 相关文档

- [代码索引](../代码索引.md) - 变更记录
- [开发日志/2025-12-21](../开发日志/2025-12-21.md) - 完整开发记录
