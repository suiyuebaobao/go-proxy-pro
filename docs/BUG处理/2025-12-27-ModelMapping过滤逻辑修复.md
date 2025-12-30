# 账户 ModelMapping 过滤逻辑修复

> 修复日期：2025-12-27
> 类型：BUG修复
> 严重程度：高（导致配置了 ModelMapping 的账户无法使用）

## 问题描述

当账户配置了 ModelMapping（如 `{"claude-opus-4-5-20251101":"MiniMax-M2.1"}`）且 AllowedModels 只允许目标模型（如 `MiniMax-M2.1`）时，该账户被错误过滤掉无法使用。

**预期行为**：
- 请求模型：`claude-opus-4-5-20251101`
- 账户 ModelMapping：`{"claude-opus-4-5-20251101":"MiniMax-M2.1"}`
- 账户 AllowedModels：`MiniMax-M2.1`
- **预期**：账户应该可用（因为映射后的目标模型在 AllowedModels 中）

**实际行为**：
- 账户被过滤掉，返回 `no available account`

## 原因分析

问题出在 `filterByAllowedModelsWithOriginal()` 函数的过滤逻辑：

```go
// 原逻辑（错误）
checkModel := mappedModel  // 使用原始模型名 claude-opus-4-5-20251101
// 直接用原始模型名去匹配 AllowedModels
```

**根本原因**：
1. `filterByAllowedModelsWithOriginal()` 在检查 AllowedModels 时，使用的是**原始请求模型名**（如 `claude-opus-4-5-20251101`）
2. 而不是使用账户 ModelMapping 映射后的**目标模型名**（如 `MiniMax-M2.1`）
3. 导致 AllowedModels 匹配失败，账户被错误过滤

## 解决方案

### 1. 新增 `getAccountMappedModel()` 函数

获取账户 ModelMapping 中原始模型对应的目标模型：

**文件**: `internal/proxy/scheduler/scheduler.go`

```go
// getAccountMappedModel 获取账户 ModelMapping 中原始模型对应的目标模型
func getAccountMappedModel(acc *model.Account, originalModel string) string {
    if acc.ModelMapping == "" {
        return ""
    }
    mapping := parseAccountModelMapping(acc)
    if mapping == nil {
        return ""
    }
    originalLower := strings.ToLower(originalModel)
    for sourceModel, targetModel := range mapping {
        sourceLower := strings.ToLower(sourceModel)
        // 支持前缀匹配
        if strings.HasPrefix(originalLower, sourceLower) || sourceLower == originalLower {
            return targetModel
        }
    }
    return ""
}
```

### 2. 修改 `filterByAllowedModelsWithOriginal()` 函数

添加账户 ModelMapping 检查逻辑：

```go
func (s *Scheduler) filterByAllowedModelsWithOriginal(accounts []*model.Account, mappedModel string, originalModel string) []*model.Account {
    for _, acc := range accounts {
        checkModel := mappedModel

        // 【新增】如果账户配置了 ModelMapping，检查原始模型是否在映射中
        if originalModel != "" && acc.ModelMapping != "" {
            targetModel := getAccountMappedModel(acc, originalModel)
            if targetModel != "" {
                // 原始模型在 ModelMapping 中，使用映射后的模型检查 AllowedModels
                checkModel = targetModel
            } else {
                // 账户配置了 ModelMapping 但不包含原始模型，跳过
                continue
            }
        }

        // 检查 AllowedModels（使用 checkModel）
        // ...
    }
}
```

### 3. 过滤逻辑流程

修复后的过滤逻辑：

```
请求模型: claude-opus-4-5-20251101
    ↓
检查账户 ModelMapping
    ↓ (找到映射)
目标模型: MiniMax-M2.1
    ↓
匹配 AllowedModels: MiniMax-M2.1
    ↓ (匹配成功)
账户可用 ✓
```

## 修改文件清单

| 文件 | 修改内容 |
|------|----------|
| `internal/proxy/scheduler/scheduler.go` | 第344-413行：重写 `filterByAllowedModelsWithOriginal()` 函数 |
| `internal/proxy/scheduler/scheduler.go` | 第478-501行：新增 `getAccountMappedModel()` 函数 |

## 关键代码位置

| 功能 | 方法 | 行号 |
|------|------|------|
| 过滤主函数 | `filterByAllowedModelsWithOriginal()` | 352-413 |
| 映射获取 | `getAccountMappedModel()` | 480-501 |
| 映射解析 | `parseAccountModelMapping()` | 439-449 |
| 映射存在检查 | `hasAccountModelMapping()` | 453-476 |

## 测试验证

### 场景 1：ModelMapping + AllowedModels

**配置**：
- 账户 ModelMapping：`{"claude-opus-4-5-20251101":"MiniMax-M2.1"}`
- 账户 AllowedModels：`MiniMax-M2.1`

**测试**：
```bash
curl http://localhost:8080/claude/v1/messages \
  -H "Authorization: Bearer sk-xxx" \
  -d '{"model": "claude-opus-4-5-20251101", ...}'
```

**预期**：请求成功，使用目标模型 MiniMax-M2.1

### 场景 2：ModelMapping 不包含请求模型

**配置**：
- 账户 ModelMapping：`{"claude-opus-4-5-20251101":"MiniMax-M2.1"}`
- 账户 AllowedModels：`MiniMax-M2.1`

**测试**：
```bash
curl http://localhost:8080/claude/v1/messages \
  -H "Authorization: Bearer sk-xxx" \
  -d '{"model": "claude-sonnet-4", ...}'
```

**预期**：账户被过滤（模型不在 ModelMapping 中）

### 场景 3：无 ModelMapping

**配置**：
- 账户 ModelMapping：空
- 账户 AllowedModels：`claude-opus-4`

**测试**：
```bash
curl http://localhost:8080/claude/v1/messages \
  -H "Authorization: Bearer sk-xxx" \
  -d '{"model": "claude-opus-4-5-20251101", ...}'
```

**预期**：请求成功（前缀匹配 AllowedModels）

## 日志示例

**成功匹配**：
```
账户 ModelMapping 命中 - ID: 10, Name: MiniMax账户, 原始模型: claude-opus-4-5-20251101 -> 目标模型: MiniMax-M2.1
```

**未匹配**：
```
账户 ModelMapping 不包含原始模型 - ID: 10, Name: MiniMax账户, ModelMapping: {"claude-opus-4-5":"MiniMax"}, OriginalModel: claude-sonnet-4
```

## 相关文档

- [代码索引](../代码索引.md) - 变更记录
- [开发日志/2025-12-27](../开发日志/2025-12-27.md) - 完整开发记录
