# OpenAI Responses 接口账户类型选择修复

> 修复日期：2025-12-19
> 类型：BUG修复
> 严重程度：高（导致 /responses 接口 503 错误）

## 问题描述

用户使用第三方 OpenAI API（账户类型为 `openai`）调用 `/responses` 接口时，返回 503 Service Unavailable 错误：

```json
{
  "error": {
    "type": "no_available_account",
    "message": "no available account"
  }
}
```

用户配置了支持 Codex /responses 端点的第三方 OpenAI API 账户，但系统无法选择该账户。

## 原因分析

1. **原有逻辑**：`openai_responses.go` 的 `HandleResponses` 方法只选择 `openai-responses` 类型的账户
2. **问题场景**：第三方 OpenAI API 提供商也支持 `/responses` 接口，但其账户类型配置为 `openai`
3. **导致结果**：当系统中只有 `openai` 类型账户（无 `openai-responses` 类型）时，调度器找不到可用账户

**原有代码** (`internal/handler/openai_responses.go`):
```go
// 只选择 openai-responses 类型
account, err := h.scheduler.SelectAccountByTypeWithSession(ctx, model.AccountTypeOpenAIResponses, sessionID, userID, apiKeyID)
```

## 解决方案

### 1. 新增多类型账户选择方法

**文件**: `internal/proxy/scheduler/scheduler.go`

新增 `SelectAccountByTypesWithSession` 方法，支持从多个账户类型中选择：

```go
// SelectAccountByTypesWithSession 根据多个账户类型选择（支持会话粘性）
func (s *Scheduler) SelectAccountByTypesWithSession(ctx context.Context, accountTypes []string, sessionID string, userID uint, apiKeyID uint) (*model.Account, error) {
    log := logger.GetLogger("scheduler")

    // 获取所有类型的账户
    var allAccounts []model.Account
    for _, accountType := range accountTypes {
        accounts, err := s.repo.GetEnabledByType(accountType)
        if err == nil {
            allAccounts = append(allAccounts, accounts...)
        }
    }

    if len(allAccounts) == 0 {
        return nil, ErrNoAvailableAccount
    }

    // 检查会话粘性（从 Redis）
    if sessionID != "" && s.sessionCache != nil {
        binding, err := s.sessionCache.GetSessionBinding(ctx, sessionID)
        if err == nil && binding != nil {
            for _, acc := range allAccounts {
                if acc.ID == binding.AccountID && acc.Enabled && acc.Status == model.AccountStatusValid {
                    log.Info("会话粘性命中 - SessionID: %s, 账户ID: %d, 名称: %s", sessionID, acc.ID, acc.Name)
                    s.sessionCache.UpdateSessionLastUsed(ctx, sessionID)
                    return &acc, nil
                }
            }
            s.sessionCache.RemoveSessionBinding(ctx, sessionID)
        }
    }

    // 转换为指针切片
    accountPtrs := make([]*model.Account, len(allAccounts))
    for i := range allAccounts {
        accountPtrs[i] = &allAccounts[i]
    }

    // 根据权重选择
    account := s.selectByWeight(accountPtrs)

    // 绑定会话到 Redis
    if sessionID != "" && s.sessionCache != nil && account != nil {
        binding := &cache.SessionBinding{
            SessionID: sessionID,
            AccountID: account.ID,
            Platform:  account.Platform,
            Model:     account.Type,
            UserID:    userID,
            APIKeyID:  apiKeyID,
        }
        s.sessionCache.SetSessionBinding(ctx, binding)
        log.Info("会话粘性绑定 - SessionID: %s, 账户ID: %d, 名称: %s, UserID: %d", sessionID, account.ID, account.Name, userID)
    }

    return account, nil
}
```

### 2. 修改 OpenAI Responses Handler

**文件**: `internal/handler/openai_responses.go`

修改账户选择逻辑，同时支持 `openai-responses` 和 `openai` 两种类型：

```go
// 选择账户（支持 openai-responses 和 openai 两种类型，支持会话粘性）
ctx := context.Background()
accountTypes := []string{model.AccountTypeOpenAIResponses, model.AccountTypeOpenAI}
account, err := h.scheduler.SelectAccountByTypesWithSession(ctx, accountTypes, sessionID, userID, apiKeyID)
```

## 修改文件清单

| 文件 | 修改内容 |
|------|----------|
| `internal/proxy/scheduler/scheduler.go` | 新增 `SelectAccountByTypesWithSession()` 方法 |
| `internal/handler/openai_responses.go` | 使用多类型账户选择，同时支持 `openai-responses` 和 `openai` |

## 账户类型说明

| 账户类型 | 说明 | 适用场景 |
|----------|------|----------|
| `openai-responses` | ChatGPT 官方 Codex 账户 | 使用 chatgpt.com 的 Session Key |
| `openai` | OpenAI API 或第三方兼容 API | 支持 /responses 端点的第三方提供商 |

## 验证方法

1. 配置一个 `openai` 类型的第三方账户（支持 /responses 端点）
2. 发送 `/responses` 请求
3. 确认请求成功转发到该账户

**测试日志**:
```
[INFO] [openai-responses] 选中账户 - ID: 27, Name: 其他家的api测试, BaseURL: https://coordcode.com/openai
[INFO] [openai-responses] 转发目标 - TargetURL: https://coordcode.com/openai/responses
[INFO] [openai-responses] Stream 完成 - Model: gpt-5.1-codex-max, InputTokens: 3610, OutputTokens: 15
```

## 相关功能

本次修复还包含了会话粘性（Sticky Session）功能的实现：

1. **会话哈希生成**：基于请求内容生成会话标识
   - 优先级1：`Session_id` 请求头
   - 优先级2：`instructions` 字段
   - 优先级3：第一条 input 消息内容

2. **会话绑定**：将会话与账户绑定到 Redis，后续相同会话的请求会路由到同一账户

3. **TTL 管理**：会话绑定默认 30 分钟过期，每次请求会更新最后使用时间
