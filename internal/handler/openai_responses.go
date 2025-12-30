/*
 * 文件作用：OpenAI Responses API处理器，处理Codex CLI的请求
 * 负责功能：
 *   - OpenAI Responses API 转发
 *   - Codex CLI 专用接口处理
 *   - 流式/非流式响应转换
 *   - 模型映射和费用统计
 * 重要程度：⭐⭐⭐⭐ 重要（Codex CLI专用接口）
 * 依赖模块：scheduler, adapter, service, repository
 */
package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/proxy/adapter"
	"go-aiproxy/internal/proxy/scheduler"
	"go-aiproxy/internal/repository"
	"go-aiproxy/internal/service"
	"go-aiproxy/pkg/logger"
	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

// OpenAIResponsesHandler 处理 OpenAI Responses API 请求
// 参考 claude-relay 的 openaiRoutes.js 实现
type OpenAIResponsesHandler struct {
	scheduler           *scheduler.Scheduler
	usageService        *service.UsageService
	pricingService      *service.PricingService
	dailyUsageRepo      *repository.DailyUsageRepository
	modelMappingService *service.ModelMappingService
}

// DefaultCodexInstructions 默认的 Codex CLI instructions
// 参考 claude-relay 的 openaiRoutes.js
const DefaultCodexInstructions = `You are Codex, based on GPT-5. You are running as a coding agent in the Codex CLI on a user's computer.

## General

- When searching for text or files, prefer using ` + "`rg`" + ` or ` + "`rg --files`" + ` respectively because ` + "`rg`" + ` is much faster than alternatives like ` + "`grep`" + `. (If the ` + "`rg`" + ` command is not found, then use alternatives.)

## Editing constraints

- Default to ASCII when editing or creating files. Only introduce non-ASCII or other Unicode characters when there is a clear justification and the file already uses them.
- You may be in a dirty git worktree.
    * NEVER revert existing changes you did not make unless explicitly requested, since these changes were made by the user.
    * If asked to make a commit or code edits and there are unrelated changes to your work or changes that you didn't make in those files, don't revert those changes.
- Do not amend a commit unless explicitly requested to do so.
- **NEVER** use destructive commands like ` + "`git reset --hard`" + ` or ` + "`git checkout --`" + ` unless specifically requested or approved by the user.

## Presenting your work

- Default: be very concise; friendly coding teammate tone.
- For code changes: Lead with a quick explanation of the change, and then give more details on the context covering where and why a change was made.
- Don't dump large files you've written; reference paths only.
`

// NewOpenAIResponsesHandler 创建 OpenAI Responses Handler
func NewOpenAIResponsesHandler() *OpenAIResponsesHandler {
	return &OpenAIResponsesHandler{
		scheduler:           scheduler.GetScheduler(),
		usageService:        service.NewUsageService(),
		pricingService:      service.NewPricingService(),
		dailyUsageRepo:      repository.NewDailyUsageRepository(),
		modelMappingService: service.NewModelMappingService(),
	}
}

// checkModelEnabled 检查模型是否启用
// 如果模型被禁用，返回错误响应并返回 false
func (h *OpenAIResponsesHandler) checkModelEnabled(c *gin.Context, modelName string) bool {
	log := logger.GetLogger("openai-responses")
	enabled, exists, err := h.pricingService.IsModelEnabled(c.Request.Context(), modelName)
	if err != nil {
		log.Error("检查模型状态失败: %v", err)
		// 出错时默认允许，避免影响正常使用
		return true
	}
	if exists && !enabled {
		log.Warn("模型已禁用: %s", modelName)
		response.CustomError(c, http.StatusForbidden, "MODEL_DISABLED", "模型 "+modelName+" 已被禁用")
		return false
	}
	return true
}

// HandleResponses 处理 /responses 和 /v1/responses 请求
// 参考 claude-relay: router.post('/responses', authenticateApiKey, handleResponses)
func (h *OpenAIResponsesHandler) HandleResponses(c *gin.Context) {
	log := logger.GetLogger("openai-responses")

	// 读取原始请求体
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.CustomBadRequest(c, "failed to read request body")
		return
	}

	// 解析请求体获取基本信息
	var reqBody map[string]interface{}
	if err := json.Unmarshal(rawBody, &reqBody); err != nil {
		response.CustomBadRequest(c, "invalid JSON body")
		return
	}

	// 提取模型和流式标志
	modelName := "gpt-4"
	if m, ok := reqBody["model"].(string); ok && m != "" {
		modelName = m
	}

	// 应用模型映射
	originalModel := modelName
	modelName = h.modelMappingService.MapModel(modelName)
	if modelName != originalModel {
		// 更新请求体中的模型名
		reqBody["model"] = modelName
		// 需要重新序列化
		rawBody, _ = json.Marshal(reqBody)
		log.Info("模型映射: %s -> %s", originalModel, modelName)
	}

	// 检查模型是否启用
	if !h.checkModelEnabled(c, modelName) {
		return
	}

	isStream := true // 默认流式
	if s, ok := reqBody["stream"].(bool); ok {
		isStream = s
	}

	// 检测是否为 Codex CLI 请求（参考 claude-relay）
	userAgent := c.GetHeader("User-Agent")
	isCodexCLI := strings.HasPrefix(userAgent, "codex_vscode/") || strings.HasPrefix(userAgent, "codex_cli_rs/")

	// 如果不是 Codex CLI 请求，进行适配（参考 claude-relay）
	if !isCodexCLI {
		log.Info("非 Codex CLI 请求，应用适配")

		// 移除不需要的字段
		fieldsToRemove := []string{"temperature", "top_p", "max_output_tokens", "user", "text_formatting", "truncation", "text", "service_tier"}
		for _, field := range fieldsToRemove {
			delete(reqBody, field)
		}

		// 设置默认的 Codex instructions
		reqBody["instructions"] = DefaultCodexInstructions

		// 确保 stream 为 true
		reqBody["stream"] = true
		isStream = true

		// 重新序列化请求体
		rawBody, err = json.Marshal(reqBody)
		if err != nil {
			response.CustomBadRequest(c, "failed to marshal request body")
			return
		}
	}

	// 获取请求路径
	requestPath := c.Request.URL.Path
	log.Info("OpenAI Responses 请求 - Model: %s, Stream: %v, Path: %s, IsCodexCLI: %v", modelName, isStream, requestPath, isCodexCLI)

	// 获取用户信息
	userID, apiKeyID := h.getUserInfo(c)

	// 构建 sessionID 用于会话粘性
	// 参考 claude-relay 的 sessionHelper.js 实现，基于请求内容生成会话哈希
	sessionID := h.generateSessionHash(c, reqBody)
	log.Info("会话哈希 - SessionID: %s", sessionID)

	// 选择账户（支持 openai-responses 和 openai 两种类型，支持会话粘性）
	ctx := context.Background()
	accountTypes := []string{model.AccountTypeOpenAIResponses, model.AccountTypeOpenAI}
	account, err := h.scheduler.SelectAccountByTypesWithSession(ctx, accountTypes, modelName, sessionID, userID, apiKeyID)
	if err != nil {
		log.Error("选择账户失败: %v", err)
		response.CustomError(c, http.StatusServiceUnavailable, "no_available_account", err.Error())
		return
	}

	log.Info("选中账户 - ID: %d, Name: %s, BaseURL: %s", account.ID, account.Name, account.BaseURL)

	// 构建目标 URL: baseURL + path
	// 参考 claude-relay: const targetUrl = `${fullAccount.baseApi}${req.path}`
	baseURL := account.BaseURL
	if baseURL == "" {
		baseURL = "https://chatgpt.com/backend-api/codex"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	// 处理平台前缀路由：去掉 /openai 前缀
	forwardPath := requestPath
	if strings.HasPrefix(requestPath, "/openai/") {
		forwardPath = strings.TrimPrefix(requestPath, "/openai")
	}
	targetURL := baseURL + forwardPath

	log.Info("转发目标 - TargetURL: %s", targetURL)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(rawBody))
	if err != nil {
		log.Error("创建请求失败: %v", err)
		response.CustomError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	// 设置请求头
	h.setRequestHeaders(httpReq, c, account)

	// 发送请求 - 流式请求使用流式客户端（10分钟超时）
	var client *http.Client
	if isStream {
		client = adapter.GetStreamHTTPClient(account)
	} else {
		client = adapter.GetHTTPClient(account)
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Error("请求失败 - 网络错误: %v", err)
		response.CustomError(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}
	defer resp.Body.Close()

	// 处理错误响应
	if resp.StatusCode != http.StatusOK {
		h.handleErrorResponse(c, resp, account, log)
		return
	}

	// 记录开始时间
	startTime := time.Now()

	// 处理响应
	if isStream {
		h.handleStreamResponse(c, resp, account, userID, apiKeyID, modelName, log)
	} else {
		h.handleNormalResponse(c, resp, account, userID, apiKeyID, modelName, log)
	}

	log.Info("请求完成 - 耗时: %v", time.Since(startTime))
}

// setRequestHeaders 设置请求头
func (h *OpenAIResponsesHandler) setRequestHeaders(httpReq *http.Request, c *gin.Context, account *model.Account) {
	// 基本头部
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	// 认证令牌优先级: SessionKey > AccessToken > APIKey
	authToken := ""
	if account.SessionKey != "" {
		authToken = account.SessionKey
	} else if account.AccessToken != "" {
		authToken = account.AccessToken
	} else if account.APIKey != "" {
		authToken = account.APIKey
	}
	httpReq.Header.Set("Authorization", "Bearer "+authToken)

	// 透传客户端头部
	if ua := c.GetHeader("User-Agent"); ua != "" {
		httpReq.Header.Set("User-Agent", ua)
	}

	if sessionID := c.GetHeader("Session_id"); sessionID != "" {
		httpReq.Header.Set("session_id", sessionID)
	}

	if version := c.GetHeader("Version"); version != "" {
		httpReq.Header.Set("version", version)
	}

	if beta := c.GetHeader("Openai-Beta"); beta != "" {
		httpReq.Header.Set("openai-beta", beta)
	}

	// 如果是 chatgpt.com 请求，添加特定头部
	if strings.Contains(httpReq.URL.Host, "chatgpt.com") {
		httpReq.Header.Set("openai-beta", "responses=experimental")
		if account.OrganizationID != "" {
			httpReq.Header.Set("chatgpt-account-id", account.OrganizationID)
		}
	}
}

// handleErrorResponse 处理错误响应
func (h *OpenAIResponsesHandler) handleErrorResponse(c *gin.Context, resp *http.Response, account *model.Account, log *logger.Logger) {
	respBody, _ := io.ReadAll(resp.Body)
	log.Error("API 错误 - StatusCode: %d, Body: %s", resp.StatusCode, string(respBody))

	// 尝试解析错误响应
	var errorResp map[string]interface{}
	if err := json.Unmarshal(respBody, &errorResp); err == nil {
		c.JSON(resp.StatusCode, errorResp)
		return
	}

	// 返回原始错误
	c.Data(resp.StatusCode, "application/json", respBody)
}

// handleStreamResponse 处理流式响应
// 参考 claude-relay: openaiResponsesRelayService._handleStreamResponse
// 直接转发原始字节流，同时解析 usage 数据
func (h *OpenAIResponsesHandler) handleStreamResponse(c *gin.Context, resp *http.Response, account *model.Account, userID, apiKeyID uint, modelName string, log *logger.Logger) {
	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 立即刷新头部
	c.Writer.Flush()

	// 获取倍率
	priceRate := 1.0
	if rate, ok := c.Get("api_key_price_rate"); ok {
		if r, ok := rate.(float64); ok {
			priceRate = r
		}
	}

	var inputTokens, outputTokens int
	var cacheReadTokens, cacheCreationTokens int
	var actualModel string
	var buffer strings.Builder

	ctx := c.Request.Context()

	// 监控 context 取消（客户端断开）
	streamDone := make(chan struct{})
	defer close(streamDone)

	go func() {
		select {
		case <-ctx.Done():
			// 客户端断开或超时，关闭上游连接
			log.Info("OpenAI Responses Stream 客户端断开或超时，关闭上游连接")
			resp.Body.Close()
		case <-streamDone:
			// 正常完成
		}
	}()

	// 直接转发原始字节，同时解析 usage
	buf := make([]byte, 32*1024)
	for {
		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			log.Info("OpenAI Responses Stream context 已取消，停止转发")
			goto done
		default:
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			chunk := buf[:n]

			// 如果倍率不为1，需要修改 token 数量后再转发
			if priceRate != 1.0 {
				chunk = h.applyRateToSSEChunk(chunk, priceRate)
			}

			// 转发给客户端
			_, writeErr := c.Writer.Write(chunk)
			if writeErr != nil {
				log.Warn("OpenAI Responses Stream 写入客户端失败: %v", writeErr)
				goto done
			}
			c.Writer.Flush()

			// 同时解析 usage 数据（解析原始数据，不是修改后的）
			buffer.Write(buf[:n])
			h.parseSSEForUsage(&buffer, &actualModel, &inputTokens, &outputTokens, &cacheReadTokens, &cacheCreationTokens, log)
		}

		if err != nil {
			if err != io.EOF {
				// 检查是否是因为 context 取消导致的错误
				if ctx.Err() != nil {
					log.Info("OpenAI Responses Stream 因 context 取消而结束: %v", ctx.Err())
				} else {
					log.Error("Stream 读取错误: %v", err)
				}
			}
			break
		}
	}

done:
	// 处理剩余 buffer
	if buffer.Len() > 0 {
		h.parseSSEForUsage(&buffer, &actualModel, &inputTokens, &outputTokens, &cacheReadTokens, &cacheCreationTokens, log)
	}

	// 记录使用量
	if actualModel == "" {
		actualModel = modelName
	}

	// 应用倍率到 token（用于日志记录和费用计算）
	ratedInputTokens := int(float64(inputTokens) * priceRate)
	ratedOutputTokens := int(float64(outputTokens) * priceRate)
	ratedCacheReadTokens := int(float64(cacheReadTokens) * priceRate)
	ratedCacheCreationTokens := int(float64(cacheCreationTokens) * priceRate)

	log.Info("Stream 完成 - Model: %s, 原始Token(in:%d/out:%d), 倍率:%.2f, 计费Token(in:%d/out:%d)",
		actualModel, inputTokens, outputTokens, priceRate, ratedInputTokens, ratedOutputTokens)

	// 标记账户成功（更新 last_used_at 和 request_count）
	h.scheduler.MarkAccountSuccess(account.ID)

	// 记录使用统计（使用倍率后的 token）
	if ratedInputTokens > 0 || ratedOutputTokens > 0 {
		h.recordUsage(c, userID, apiKeyID, account.ID, actualModel, ratedInputTokens, ratedOutputTokens, ratedCacheReadTokens, ratedCacheCreationTokens)
	}
}

// parseSSEForUsage 从 SSE 数据中解析 usage 信息
// 参考 claude-relay: openaiResponsesRelayService 的 usage 解析
func (h *OpenAIResponsesHandler) parseSSEForUsage(buffer *strings.Builder, actualModel *string, inputTokens, outputTokens, cacheReadTokens, cacheCreationTokens *int, log *logger.Logger) {
	data := buffer.String()

	// 查找完整的 SSE 事件（以 \n\n 分隔）
	for {
		idx := strings.Index(data, "\n\n")
		if idx == -1 {
			break
		}

		event := data[:idx]
		data = data[idx+2:]

		// 解析事件
		lines := strings.Split(event, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "data: ") {
				jsonStr := strings.TrimPrefix(line, "data: ")
				if jsonStr == "[DONE]" {
					continue
				}

				var eventData map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &eventData); err != nil {
					continue
				}

				// 检查 response.completed 事件
				if eventType, ok := eventData["type"].(string); ok && eventType == "response.completed" {
					if resp, ok := eventData["response"].(map[string]interface{}); ok {
						if m, ok := resp["model"].(string); ok {
							*actualModel = m
							log.Debug("捕获实际模型: %s", m)
						}
						if usage, ok := resp["usage"].(map[string]interface{}); ok {
							// 基础 token
							if it, ok := usage["input_tokens"].(float64); ok {
								*inputTokens = int(it)
							}
							if ot, ok := usage["output_tokens"].(float64); ok {
								*outputTokens = int(ot)
							}

							// 缓存 token - 参考 claude-relay
							// 1. 从 input_tokens_details.cached_tokens 获取缓存读取 token
							if details, ok := usage["input_tokens_details"].(map[string]interface{}); ok {
								if ct, ok := details["cached_tokens"].(float64); ok {
									*cacheReadTokens = int(ct)
								}
								// 2. 从 input_tokens_details.cache_creation_input_tokens 获取缓存创建 token
								if cct, ok := details["cache_creation_input_tokens"].(float64); ok {
									*cacheCreationTokens = int(cct)
								}
								// 兼容其他可能的字段名
								if cct, ok := details["cache_creation_tokens"].(float64); ok && *cacheCreationTokens == 0 {
									*cacheCreationTokens = int(cct)
								}
							}

							// 3. 从顶级 cache_creation_input_tokens 获取（兼容旧格式）
							if cct, ok := usage["cache_creation_input_tokens"].(float64); ok && *cacheCreationTokens == 0 {
								*cacheCreationTokens = int(cct)
							}

							log.Debug("捕获 usage: input=%d, output=%d, cacheRead=%d, cacheCreation=%d",
								*inputTokens, *outputTokens, *cacheReadTokens, *cacheCreationTokens)
						}
					}
				}
			}
		}
	}

	// 更新 buffer，保留未处理的数据
	buffer.Reset()
	buffer.WriteString(data)
}

// handleNormalResponse 处理非流式响应
func (h *OpenAIResponsesHandler) handleNormalResponse(c *gin.Context, resp *http.Response, account *model.Account, userID, apiKeyID uint, modelName string, log *logger.Logger) {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("读取响应失败: %v", err)
		response.CustomError(c, http.StatusBadGateway, "upstream_error", err.Error())
		return
	}

	// 获取倍率
	priceRate := 1.0
	if rate, ok := c.Get("api_key_price_rate"); ok {
		if r, ok := rate.(float64); ok {
			priceRate = r
		}
	}

	// 解析响应获取 usage
	var respData map[string]interface{}
	var inputTokens, outputTokens int
	var cacheReadTokens, cacheCreationTokens int
	var actualModel string

	if err := json.Unmarshal(respBody, &respData); err == nil {
		if m, ok := respData["model"].(string); ok {
			actualModel = m
		}
		if usage, ok := respData["usage"].(map[string]interface{}); ok {
			if it, ok := usage["input_tokens"].(float64); ok {
				inputTokens = int(it)
			}
			if ot, ok := usage["output_tokens"].(float64); ok {
				outputTokens = int(ot)
			}

			// 缓存 token - 参考 claude-relay
			if details, ok := usage["input_tokens_details"].(map[string]interface{}); ok {
				if ct, ok := details["cached_tokens"].(float64); ok {
					cacheReadTokens = int(ct)
				}
				if cct, ok := details["cache_creation_input_tokens"].(float64); ok {
					cacheCreationTokens = int(cct)
				}
				if cct, ok := details["cache_creation_tokens"].(float64); ok && cacheCreationTokens == 0 {
					cacheCreationTokens = int(cct)
				}
			}
			if cct, ok := usage["cache_creation_input_tokens"].(float64); ok && cacheCreationTokens == 0 {
				cacheCreationTokens = int(cct)
			}

			// 如果倍率不为1，修改响应中的 token 数量
			if priceRate != 1.0 {
				h.applyRateToUsageMap(usage, priceRate)
				// 重新序列化响应
				respBody, _ = json.Marshal(respData)
			}
		}
	}

	if actualModel == "" {
		actualModel = modelName
	}

	// 应用倍率到 token（用于日志记录和费用计算）
	ratedInputTokens := int(float64(inputTokens) * priceRate)
	ratedOutputTokens := int(float64(outputTokens) * priceRate)
	ratedCacheReadTokens := int(float64(cacheReadTokens) * priceRate)
	ratedCacheCreationTokens := int(float64(cacheCreationTokens) * priceRate)

	log.Info("非流式响应 - Model: %s, 原始Token(in:%d/out:%d), 倍率:%.2f, 计费Token(in:%d/out:%d)",
		actualModel, inputTokens, outputTokens, priceRate, ratedInputTokens, ratedOutputTokens)

	// 标记账户成功（更新 last_used_at 和 request_count）
	h.scheduler.MarkAccountSuccess(account.ID)

	// 记录使用统计（使用倍率后的 token）
	if ratedInputTokens > 0 || ratedOutputTokens > 0 {
		h.recordUsage(c, userID, apiKeyID, account.ID, actualModel, ratedInputTokens, ratedOutputTokens, ratedCacheReadTokens, ratedCacheCreationTokens)
	}

	// 返回响应（已应用倍率）
	c.Data(resp.StatusCode, "application/json", respBody)
}

// applyRateToUsageMap 将倍率应用到 usage map 中的 token 字段
func (h *OpenAIResponsesHandler) applyRateToUsageMap(usage map[string]interface{}, rate float64) {
	tokenFields := []string{"input_tokens", "output_tokens", "cached_tokens", "cache_creation_input_tokens", "cache_read_input_tokens", "total_tokens"}
	for _, field := range tokenFields {
		if val, ok := usage[field].(float64); ok {
			usage[field] = int(val * rate)
		}
	}
	// 处理嵌套的 input_tokens_details
	if details, ok := usage["input_tokens_details"].(map[string]interface{}); ok {
		h.applyRateToUsageMap(details, rate)
	}
	// 处理嵌套的 output_tokens_details
	if details, ok := usage["output_tokens_details"].(map[string]interface{}); ok {
		h.applyRateToUsageMap(details, rate)
	}
}

// applyRateToSSEChunk 将倍率应用到 SSE 数据块中的 token 字段
func (h *OpenAIResponsesHandler) applyRateToSSEChunk(chunk []byte, rate float64) []byte {
	content := string(chunk)

	// 匹配所有 token 相关的字段并乘以倍率
	tokenFields := []string{
		"input_tokens",
		"output_tokens",
		"cached_tokens",
		"cache_creation_input_tokens",
		"cache_read_input_tokens",
		"total_tokens",
	}

	for _, field := range tokenFields {
		// 匹配 "field": 数字 或 "field":数字
		pattern := regexp.MustCompile(`"` + field + `"\s*:\s*(\d+)`)
		content = pattern.ReplaceAllStringFunc(content, func(match string) string {
			// 提取数字
			numPattern := regexp.MustCompile(`(\d+)`)
			numStr := numPattern.FindString(match)
			if num, err := strconv.Atoi(numStr); err == nil {
				newNum := int(float64(num) * rate)
				return strings.Replace(match, numStr, strconv.Itoa(newNum), 1)
			}
			return match
		})
	}

	return []byte(content)
}

// recordUsage 记录使用量到 Redis 和 MySQL
func (h *OpenAIResponsesHandler) recordUsage(c *gin.Context, userID, apiKeyID, accountID uint, modelName string, inputTokens, outputTokens, cacheReadTokens, cacheCreationTokens int) {
	log := logger.GetLogger("openai-responses")
	log.Info("Usage - User: %d, APIKey: %d, Account: %d, Model: %s, Input: %d, Output: %d, CacheRead: %d, CacheCreation: %d",
		userID, apiKeyID, accountID, modelName, inputTokens, outputTokens, cacheReadTokens, cacheCreationTokens)

	ctx := context.Background()

	// 获取价格倍率
	priceRate := 1.0
	if rate, ok := c.Get("api_key_price_rate"); ok {
		if r, ok := rate.(float64); ok {
			priceRate = r
		}
	}

	// 应用倍率到 token（用于日志记录和费用计算）
	ratedInputTokens := int(float64(inputTokens) * priceRate)
	ratedOutputTokens := int(float64(outputTokens) * priceRate)
	ratedCacheCreationTokens := int(float64(cacheCreationTokens) * priceRate)
	ratedCacheReadTokens := int(float64(cacheReadTokens) * priceRate)

	// 计算费用（使用倍率后的 token）
	tokenUsage := &service.TokenUsage{
		InputTokens:              ratedInputTokens,
		OutputTokens:             ratedOutputTokens,
		CacheReadInputTokens:     ratedCacheReadTokens,
		CacheCreationInputTokens: ratedCacheCreationTokens,
	}
	costBreakdown, err := h.pricingService.CalculateCost(ctx, modelName, tokenUsage, 1.0) // 倍率已应用到token，这里用1.0
	if err != nil {
		log.Error("计算费用失败: %v", err)
		costBreakdown = &service.CostBreakdown{}
	}

	// 使用辅助函数构建请求日志
	requestLog := BuildRequestLog(
		accountID,
		"openai",
		modelName,
		c.Request.URL.Path,
		c.Request.Method,
		c.Request.URL.Path,
		c.ClientIP(),
		c.GetHeader("User-Agent"),
		"",
	)

	// 设置用户信息
	uid := userID
	keyID := apiKeyID
	requestLog.UserID = &uid
	requestLog.APIKeyID = &keyID

	// 使用 CompleteLogFull 完成日志记录（会自动调用 LogRequest 写入 MySQL）
	CompleteLogFull(requestLog, true, 200, "",
		ratedInputTokens, ratedOutputTokens, ratedCacheCreationTokens, ratedCacheReadTokens,
		costBreakdown.InputCost, costBreakdown.OutputCost, costBreakdown.CacheCreateCost, costBreakdown.CacheReadCost,
		0)

	// 记录到 Redis（倍率已应用，这里用 1.0）
	if err := h.usageService.RecordRequest(ctx, userID, apiKeyID, requestLog, 1.0); err != nil {
		log.Error("记录使用统计失败: %v", err)
	}

	// 记录模型使用统计（使用倍率后的 token）
	totalTokens := int64(ratedInputTokens + ratedOutputTokens)
	if err := h.usageService.IncrementModelUsage(ctx, userID, modelName, totalTokens, costBreakdown.TotalCost); err != nil {
		log.Error("记录模型使用统计失败: %v", err)
	}

	// 记录账户费用到 Redis
	if accountID > 0 {
		if err := h.usageService.IncrementAccountCost(ctx, accountID, costBreakdown.TotalCost); err != nil {
			log.Error("记录账户费用失败: %v", err)
		}
	}

	// 增量更新 MySQL 每日汇总（使用倍率后的 token）
	dailyUsage := &model.DailyUsage{
		RequestCount:             1,
		InputTokens:              int64(ratedInputTokens),
		OutputTokens:             int64(ratedOutputTokens),
		CacheCreationInputTokens: int64(ratedCacheCreationTokens),
		CacheReadInputTokens:     int64(ratedCacheReadTokens),
		TotalTokens:              int64(ratedInputTokens + ratedOutputTokens),
		InputCost:                costBreakdown.InputCost,
		OutputCost:               costBreakdown.OutputCost,
		CacheCreateCost:          costBreakdown.CacheCreateCost,
		CacheReadCost:            costBreakdown.CacheReadCost,
		TotalCost:                costBreakdown.TotalCost,
	}
	if err := h.dailyUsageRepo.IncrementUsage(userID, modelName, dailyUsage); err != nil {
		log.Error("更新每日汇总失败: %v", err)
	}

	log.Info("使用记录已保存 - Cost: %.6f", costBreakdown.TotalCost)
}

// getUserInfo 获取用户信息
func (h *OpenAIResponsesHandler) getUserInfo(c *gin.Context) (userID, apiKeyID uint) {
	if uid, ok := c.Get("api_key_user_id"); ok {
		if id, ok := uid.(uint); ok {
			userID = id
		}
	}
	if kid, ok := c.Get("api_key_id"); ok {
		if id, ok := kid.(uint); ok {
			apiKeyID = id
		}
	}
	return
}

// generateSessionHash 生成会话哈希，用于粘性会话保持
// 参考 claude-relay 的 sessionHelper.js 实现
// 优先级：
//  1. 客户端提供的 Session_id 请求头
//  2. 请求体中的 instructions 字段（类似 system prompt）
//  3. 第一条 input 消息内容
func (h *OpenAIResponsesHandler) generateSessionHash(c *gin.Context, reqBody map[string]interface{}) string {
	log := logger.GetLogger("openai-responses")

	// 1. 最高优先级：使用客户端提供的 Session_id 请求头
	// 注意：Gin 的 GetHeader 不区分大小写，但请求头名称可能被规范化
	sessionHeader := c.GetHeader("Session_id")
	if sessionHeader == "" {
		sessionHeader = c.GetHeader("Session-Id")
	}
	if sessionHeader == "" {
		sessionHeader = c.Request.Header.Get("Session_id")
	}

	if sessionHeader != "" {
		log.Debug("使用 Session_id 请求头生成哈希: %s", sessionHeader)
		hash := sha256.Sum256([]byte(sessionHeader))
		return hex.EncodeToString(hash[:])[:32]
	}

	// 2. 使用 instructions 字段（类似 claude-relay 的 system prompt）
	if instructions, ok := reqBody["instructions"].(string); ok && instructions != "" {
		log.Debug("使用 instructions 生成哈希: %s...", instructions[:min(len(instructions), 50)])
		hash := sha256.Sum256([]byte(instructions))
		return hex.EncodeToString(hash[:])[:32]
	}

	// 3. Fallback: 使用第一条 input 消息内容
	if input, ok := reqBody["input"].([]interface{}); ok && len(input) > 0 {
		if firstMsg, ok := input[0].(map[string]interface{}); ok {
			// 尝试获取 content 字段
			var content string
			if c, ok := firstMsg["content"].(string); ok {
				content = c
			} else if contentArr, ok := firstMsg["content"].([]interface{}); ok {
				// content 可能是数组格式
				for _, item := range contentArr {
					if itemMap, ok := item.(map[string]interface{}); ok {
						if itemType, ok := itemMap["type"].(string); ok && itemType == "text" {
							if text, ok := itemMap["text"].(string); ok {
								content += text
							}
						}
					}
				}
			}

			if content != "" {
				hash := sha256.Sum256([]byte(content))
				return hex.EncodeToString(hash[:])[:32]
			}
		}
	}

	// 无法生成会话哈希，返回空字符串（调度器会使用随机选择）
	return ""
}
