/*
 * 文件作用：代理转发核心处理器，处理所有AI平台的API请求转发
 * 负责功能：
 *   - Claude API 转发（/claude/v1/messages）
 *   - OpenAI API 转发（/openai/v1/chat/completions）
 *   - Gemini API 转发
 *   - 流式/非流式响应处理
 *   - 请求重试和账户切换
 *   - 使用量记录和费用统计
 *   - 限流头解析和账户状态更新
 * 重要程度：⭐⭐⭐⭐⭐ 核心（代理转发的主要入口）
 * 依赖模块：scheduler, adapter, service, model
 */
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type ProxyHandler struct {
	scheduler       *scheduler.Scheduler
	retryConfig     *scheduler.RetryConfig
	usageService    *service.UsageService
	pricingService  *service.PricingService
	userRepo        *repository.UserRepository
	dailyUsageRepo  *repository.DailyUsageRepository
	apiKeyService   *service.APIKeyService
	accountRepo     *repository.AccountRepository
	userPackageRepo *repository.UserPackageRepository
}

func NewProxyHandler() *ProxyHandler {
	return &ProxyHandler{
		scheduler:       scheduler.GetScheduler(),
		retryConfig:     &scheduler.DefaultRetryConfig,
		usageService:    service.NewUsageService(),
		pricingService:  service.NewPricingService(),
		userRepo:        repository.NewUserRepository(),
		dailyUsageRepo:  repository.NewDailyUsageRepository(),
		apiKeyService:   service.NewAPIKeyService(),
		accountRepo:     repository.NewAccountRepository(),
		userPackageRepo: repository.NewUserPackageRepository(),
	}
}

// RateWriter 倍率写入器，包装 io.Writer 并在写入时修改 token 值
type RateWriter struct {
	writer io.Writer
	rate   float64
}

// NewRateWriter 创建倍率写入器
func NewRateWriter(w io.Writer, rate float64) *RateWriter {
	return &RateWriter{writer: w, rate: rate}
}

// Write 实现 io.Writer 接口，写入时修改 token 值
func (rw *RateWriter) Write(p []byte) (n int, err error) {
	if rw.rate == 1.0 {
		return rw.writer.Write(p)
	}
	modified := applyRateToSSEChunk(p, rw.rate)
	// 返回原始长度，避免调用者认为写入不完整
	_, err = rw.writer.Write(modified)
	return len(p), err
}

// Flush 实现 http.Flusher 接口（如果底层 writer 支持）
func (rw *RateWriter) Flush() {
	if f, ok := rw.writer.(interface{ Flush() }); ok {
		f.Flush()
	}
}

// applyRateToSSEChunk 将倍率应用到 SSE 数据块中的 token 字段
func applyRateToSSEChunk(chunk []byte, rate float64) []byte {
	content := string(chunk)

	// 匹配所有 token 相关的字段并乘以倍率
	// 包括 OpenAI、Claude 和 Gemini 的字段名
	tokenFields := []string{
		// OpenAI 格式
		"prompt_tokens",
		"completion_tokens",
		"total_tokens",
		"cached_tokens",
		// Claude 格式
		"input_tokens",
		"output_tokens",
		"cache_creation_input_tokens",
		"cache_read_input_tokens",
		// Gemini 格式
		"promptTokenCount",
		"candidatesTokenCount",
		"totalTokenCount",
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

// getSessionID 获取会话ID
// 优先使用请求头中的 x-session-id（Claude Code 每个窗口会发不同的 session）
// 如果没有则使用 API Key ID
func (h *ProxyHandler) getSessionID(c *gin.Context) string {
	// 优先使用 Claude Code 的 x-session-id
	if sessionID := c.GetHeader("x-session-id"); sessionID != "" {
		// 加上 API Key ID 前缀，避免不同用户的 session 冲突
		if apiKeyID, ok := c.Get("api_key_id"); ok {
			if id, ok := apiKeyID.(uint); ok {
				return fmt.Sprintf("apikey:%d:%s", id, sessionID)
			}
		}
		return sessionID
	}
	// 回退到 API Key ID
	if apiKeyID, ok := c.Get("api_key_id"); ok {
		if id, ok := apiKeyID.(uint); ok {
			return fmt.Sprintf("apikey:%d", id)
		}
	}
	return ""
}

// getUserInfo 获取用户信息（用于会话绑定）
func (h *ProxyHandler) getUserInfo(c *gin.Context) (userID, apiKeyID uint, clientIP, userAgent string) {
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
	clientIP = c.ClientIP()
	userAgent = c.GetHeader("User-Agent")
	return
}

// createRetryRequest 创建带用户信息的重试请求
func (h *ProxyHandler) createRetryRequest(c *gin.Context) *scheduler.RetryableRequest {
	userID, apiKeyID, clientIP, userAgent := h.getUserInfo(c)
	return scheduler.NewRetryableRequest(h.scheduler, h.retryConfig).
		WithSessionID(h.getSessionID(c)).
		WithUserInfo(userID, apiKeyID, clientIP, userAgent)
}

// checkModelEnabled 检查模型是否启用
// 如果模型被禁用，返回错误响应并返回 false
func (h *ProxyHandler) checkModelEnabled(c *gin.Context, modelName string) bool {
	log := logger.GetLogger("proxy")
	enabled, exists, err := h.pricingService.IsModelEnabled(c.Request.Context(), modelName)
	if err != nil {
		log.Error("检查模型状态失败: %v", err)
		// 出错时默认允许，避免影响正常使用
		return true
	}
	if exists && !enabled {
		log.Warn("模型已禁用: %s", modelName)
		response.Forbidden(c, fmt.Sprintf("模型 %s 已被禁用", modelName))
		return false
	}
	return true
}

// OpenAI 非流式响应（带重试）
// originalModel: 客户端请求的原始模型名（映射前），用于账户 ModelMapping 检查
func (h *ProxyHandler) handleOpenAINonStreamWithRetry(c *gin.Context, req *adapter.Request, accountType string, originalModel string) {
	retryReq := h.createRetryRequest(c).WithOriginalModel(originalModel)

	modelName := req.Model
	if accountType != "" {
		modelName = accountType + "," + req.Model
	}

	result, err := retryReq.ExecuteWithRetry(
		c.Request.Context(),
		modelName,
		func(ctx context.Context, account *model.Account) (*adapter.Response, error) {
			adp := adapter.Get(account.Type)
			if adp == nil {
				return nil, adapter.ErrNoAdapter
			}
			return adp.Send(ctx, account, req)
		},
	)

	if err != nil {
		// 根据错误类型返回自定义错误
		errorType, statusCode := getProxyErrorTypeAndCode(err)
		response.CustomError(c, statusCode, errorType, err.Error())
		return
	}

	resp := result.Response
	if resp.Error != nil {
		response.CustomError(c, http.StatusBadRequest, model.ErrorTypeBadRequest, resp.Error.Message)
		return
	}

	// 获取倍率（由中间件设置）
	priceRate := 1.0
	if rate, ok := c.Get("api_key_price_rate"); ok {
		if r, ok := rate.(float64); ok {
			priceRate = r
		}
	}

	// 应用倍率到返回给用户的 token 值
	ratedInputTokens := int(float64(resp.InputTokens) * priceRate)
	ratedOutputTokens := int(float64(resp.OutputTokens) * priceRate)

	// 构建响应体用于日志记录（使用倍率后的 token）
	responseBody, _ := json.Marshal(gin.H{
		"id":      resp.ID,
		"object":  "chat.completion",
		"model":   resp.Model,
		"choices": []gin.H{
			{
				"index": 0,
				"message": gin.H{
					"role":    "assistant",
					"content": resp.Content,
				},
				"finish_reason": convertStopReason(resp.StopReason),
			},
		},
		"usage": gin.H{
			"prompt_tokens":     ratedInputTokens,
			"completion_tokens": ratedOutputTokens,
			"total_tokens":      ratedInputTokens + ratedOutputTokens,
		},
	})

	// 获取请求体
	var requestBody []byte
	if rb, ok := c.Get("request_body"); ok {
		requestBody = rb.([]byte)
	}

	// 记录使用统计（使用原始模型名）
	h.recordNonStreamUsage(c, originalModel, resp, requestBody, responseBody, 200, result.AccountID)

	// 返回 OpenAI 格式（使用倍率后的 token）
	c.JSON(http.StatusOK, gin.H{
		"id":      resp.ID,
		"object":  "chat.completion",
		"model":   resp.Model,
		"choices": []gin.H{
			{
				"index": 0,
				"message": gin.H{
					"role":    "assistant",
					"content": resp.Content,
				},
				"finish_reason": convertStopReason(resp.StopReason),
			},
		},
		"usage": gin.H{
			"prompt_tokens":     ratedInputTokens,
			"completion_tokens": ratedOutputTokens,
			"total_tokens":      ratedInputTokens + ratedOutputTokens,
		},
	})
}

// OpenAI 流式响应（带重试）
// originalModel: 客户端请求的原始模型名（映射前），用于账户 ModelMapping 检查
func (h *ProxyHandler) handleOpenAIStreamWithRetry(c *gin.Context, req *adapter.Request, accountType string, originalModel string) {
	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // 禁用 Nginx 缓冲

	writer := c.Writer

	// 立即刷新头部，确保客户端知道这是流式响应
	writer.Flush()

	// 获取倍率（由中间件设置）
	priceRate := 1.0
	if rate, ok := c.Get("api_key_price_rate"); ok {
		if r, ok := rate.(float64); ok {
			priceRate = r
		}
	}

	// 使用 RateWriter 包装 writer，在写入时修改 token 值
	rateWriter := NewRateWriter(writer, priceRate)

	// 使用 TailWriter 捕获末尾 2KB 响应（包装 RateWriter）
	tailWriter := adapter.NewTailWriter(rateWriter, 2048)

	retryReq := h.createRetryRequest(c).WithOriginalModel(originalModel)

	modelName := req.Model
	if accountType != "" {
		modelName = accountType + "," + req.Model
	}

	result, err := retryReq.ExecuteStreamWithRetry(
		c.Request.Context(),
		modelName,
		func(ctx context.Context, account *model.Account, w io.Writer) (*adapter.StreamResult, error) {
			adp := adapter.Get(account.Type)
			if adp == nil {
				return nil, adapter.ErrNoAdapter
			}
			return adp.SendStream(ctx, account, req, w)
		},
		tailWriter,
	)

	if err != nil {
		errEvent := map[string]interface{}{
			"error": map[string]string{
				"message": err.Error(),
				"type":    "api_error",
			},
		}
		data, _ := json.Marshal(errEvent)
		writer.Write([]byte("data: " + string(data) + "\n\n"))
		return
	}

	// 获取请求体
	var requestBody []byte
	if rb, ok := c.Get("request_body"); ok {
		requestBody = rb.([]byte)
	}

	// 获取响应末尾内容
	responseTail := tailWriter.Tail()

	// 记录使用统计（使用原始模型名）
	if result != nil && result.Result != nil {
		h.recordUsage(c, originalModel, result.Result, true, requestBody, responseTail, 200, result.AccountID)
	}

	writer.Write([]byte("data: [DONE]\n\n"))
}

// Claude 非流式响应（带重试）
// originalModel: 客户端请求的原始模型名（映射前），用于账户 ModelMapping 检查
func (h *ProxyHandler) handleClaudeNonStreamWithRetry(c *gin.Context, req *adapter.Request, accountType string, originalModel string) {
	retryReq := h.createRetryRequest(c).WithOriginalModel(originalModel)

	modelName := req.Model
	if accountType != "" {
		modelName = accountType + "," + req.Model
	}

	result, err := retryReq.ExecuteWithRetry(
		c.Request.Context(),
		modelName,
		func(ctx context.Context, account *model.Account) (*adapter.Response, error) {
			adp := adapter.Get(account.Type)
			if adp == nil {
				return nil, adapter.ErrNoAdapter
			}
			return adp.Send(ctx, account, req)
		},
	)

	if err != nil {
		// 使用自定义错误消息
		errorType, statusCode := getProxyErrorTypeAndCode(err)
		customMsg, _ := getCustomErrorMessage(errorType, err.Error())
		c.JSON(statusCode, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "api_error",
				"message": customMsg,
			},
		})
		return
	}

	resp := result.Response
	if resp.Error != nil {
		customMsg, _ := getCustomErrorMessage(model.ErrorTypeBadRequest, resp.Error.Message)
		c.JSON(http.StatusBadRequest, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    resp.Error.Type,
				"message": customMsg,
			},
		})
		return
	}

	// 获取倍率（由中间件设置）
	priceRate := 1.0
	if rate, ok := c.Get("api_key_price_rate"); ok {
		if r, ok := rate.(float64); ok {
			priceRate = r
		}
	}

	// 应用倍率到返回给用户的 token 值
	ratedInputTokens := int(float64(resp.InputTokens) * priceRate)
	ratedOutputTokens := int(float64(resp.OutputTokens) * priceRate)

	// 构建响应体用于日志记录（使用倍率后的 token）
	responseBody, _ := json.Marshal(gin.H{
		"id":          resp.ID,
		"type":        "message",
		"role":        "assistant",
		"model":       resp.Model,
		"content":     []gin.H{{"type": "text", "text": resp.Content}},
		"stop_reason": resp.StopReason,
		"usage": gin.H{
			"input_tokens":  ratedInputTokens,
			"output_tokens": ratedOutputTokens,
		},
	})

	// 获取请求体
	var requestBody []byte
	if rb, ok := c.Get("request_body"); ok {
		requestBody = rb.([]byte)
	}

	// 记录使用统计（使用原始模型名）
	h.recordNonStreamUsage(c, originalModel, resp, requestBody, responseBody, 200, result.AccountID)

	// 更新账号用量状态（从响应头获取）
	h.updateAccountUsageStatus(result.AccountID, resp.Headers)

	// 返回 Claude 格式（使用倍率后的 token）
	c.JSON(http.StatusOK, gin.H{
		"id":          resp.ID,
		"type":        "message",
		"role":        "assistant",
		"model":       resp.Model,
		"content":     []gin.H{{"type": "text", "text": resp.Content}},
		"stop_reason": resp.StopReason,
		"usage": gin.H{
			"input_tokens":  ratedInputTokens,
			"output_tokens": ratedOutputTokens,
		},
	})
}

// Claude 流式响应（带重试）
// originalModel: 客户端请求的原始模型名（映射前），用于账户 ModelMapping 检查
func (h *ProxyHandler) handleClaudeStreamWithRetry(c *gin.Context, req *adapter.Request, accountType string, originalModel string) {
	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // 禁用 Nginx 缓冲

	writer := c.Writer

	// 立即刷新头部，确保客户端知道这是流式响应
	writer.Flush()

	// 获取倍率（由中间件设置）
	priceRate := 1.0
	if rate, ok := c.Get("api_key_price_rate"); ok {
		if r, ok := rate.(float64); ok {
			priceRate = r
		}
	}

	log := logger.GetLogger("proxy")
	log.Debug("Claude Stream 倍率 | Rate: %.2f | Model: %s", priceRate, req.Model)

	// 使用 RateWriter 包装 writer，在写入时修改 token 值
	rateWriter := NewRateWriter(writer, priceRate)

	// 使用 TailWriter 捕获末尾 2KB 响应（包装 RateWriter）
	tailWriter := adapter.NewTailWriter(rateWriter, 2048)

	retryReq := h.createRetryRequest(c).WithOriginalModel(originalModel)

	modelName := req.Model
	if accountType != "" {
		modelName = accountType + "," + req.Model
	}

	result, err := retryReq.ExecuteStreamWithRetry(
		c.Request.Context(),
		modelName,
		func(ctx context.Context, account *model.Account, w io.Writer) (*adapter.StreamResult, error) {
			adp := adapter.Get(account.Type)
			if adp == nil {
				return nil, adapter.ErrNoAdapter
			}
			return adp.SendStream(ctx, account, req, w)
		},
		tailWriter,
	)

	if err != nil {
		writer.Write([]byte("event: error\n"))
		errData, _ := json.Marshal(gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "api_error",
				"message": err.Error(),
			},
		})
		writer.Write([]byte("data: " + string(errData) + "\n\n"))
		return
	}

	// 获取请求体
	var requestBody []byte
	if rb, ok := c.Get("request_body"); ok {
		requestBody = rb.([]byte)
	}

	// 获取响应末尾内容
	responseTail := tailWriter.Tail()

	// 记录使用统计（使用原始模型名）
	if result != nil && result.Result != nil {
		h.recordUsage(c, originalModel, result.Result, true, requestBody, responseTail, 200, result.AccountID)
		// 更新账号用量状态（从响应头获取）
		h.updateAccountUsageStatus(result.AccountID, result.Result.Headers)
	}
}

// ========== 平台特定路由处理器 ==========

// ClaudeMessages Claude 平台专用接口 POST /claude/v1/messages
// 强制只从 Claude 平台账户中选择，不做平台自动检测
func (h *ProxyHandler) ClaudeMessages(c *gin.Context) {
	// 1. 读取原始请求体（不做任何解析）
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": "failed to read request body",
			},
		})
		return
	}

	log := logger.GetLogger("proxy")
	log.Debug("ClaudeMessages 原始请求体 | 长度: %d | 前500字符: %s", len(rawBody), truncateForLog(string(rawBody), 500))

	// 保存原始请求体到 context 用于日志记录
	c.Set("request_body", rawBody)

	// 2. 只提取必要字段用于路由（model, stream）
	var basic struct {
		Model  string `json:"model"`
		Stream bool   `json:"stream"`
	}
	if err := json.Unmarshal(rawBody, &basic); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": "invalid JSON: " + err.Error(),
			},
		})
		return
	}

	// 3. 提取客户端 headers
	clientHeaders := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			clientHeaders[key] = values[0]
		}
	}

	// 4. 强制使用 Claude 平台（不自动检测）
	accountType := "claude"
	actualModel := scheduler.GetActualModel(basic.Model) // 去掉可能的 "type," 前缀

	// 5. 检查模型是否启用（不再做全局模型映射，只在账号级别映射）
	if !h.checkModelEnabled(c, actualModel) {
		return
	}

	// 6. 构建透传请求（模型映射由调度器在账号级别处理）
	req := &adapter.Request{
		Model:   actualModel,
		Stream:  basic.Stream,
		RawBody: rawBody,
		Headers: clientHeaders,
	}

	if req.Stream {
		h.handleClaudeStreamWithRetry(c, req, accountType, actualModel)
	} else {
		h.handleClaudeNonStreamWithRetry(c, req, accountType, actualModel)
	}
}

// OpenAIChatCompletions OpenAI 平台专用接口 POST /openai/v1/chat/completions
// 强制只从 OpenAI 平台账户中选择，不做平台自动检测
func (h *ProxyHandler) OpenAIChatCompletions(c *gin.Context) {
	// 读取原始请求体用于日志记录
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.CustomBadRequest(c, "failed to read request body")
		return
	}

	var req adapter.Request
	if err := json.Unmarshal(rawBody, &req); err != nil {
		response.CustomBadRequest(c, err.Error())
		return
	}

	// 保存原始请求体（用于适配器需要透传的场景）
	req.RawBody = rawBody

	// 保存原始请求体到 context
	c.Set("request_body", rawBody)

	// 强制使用 OpenAI 平台（不自动检测）
	accountType := "openai"
	actualModel := scheduler.GetActualModel(req.Model) // 去掉可能的 "type," 前缀

	// 使用原始模型名（不再做全局模型映射，只在账号级别映射）
	req.Model = actualModel

	// 检查模型是否启用
	if !h.checkModelEnabled(c, actualModel) {
		return
	}

	if req.Stream {
		h.handleOpenAIStreamWithRetry(c, &req, accountType, actualModel)
	} else {
		h.handleOpenAINonStreamWithRetry(c, &req, accountType, actualModel)
	}
}

// convertStopReason 转换停止原因为 OpenAI 格式
func convertStopReason(reason string) string {
	switch reason {
	case "end_turn", "stop_sequence":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	default:
		return reason
	}
}

// GeminiChat Gemini 原生格式接口 POST /gemini/v1/chat
func (h *ProxyHandler) GeminiChat(c *gin.Context) {
	// 读取原始请求体用于日志记录
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    400,
				"message": "failed to read request body",
				"status":  "INVALID_ARGUMENT",
			},
		})
		return
	}

	var req adapter.Request
	if err := json.Unmarshal(rawBody, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    400,
				"message": err.Error(),
				"status":  "INVALID_ARGUMENT",
			},
		})
		return
	}

	// 保存原始请求体到 context
	c.Set("request_body", rawBody)

	// 强制使用 Gemini 平台
	if req.Model == "" {
		req.Model = "gemini-pro"
	}

	// 保存原始模型名（不再做全局模型映射，只在账号级别映射）
	originalModel := req.Model

	// 检查模型是否启用
	if !h.checkModelEnabled(c, req.Model) {
		return
	}

	if req.Stream {
		h.handleGeminiStream(c, &req, originalModel)
	} else {
		h.handleGeminiNonStream(c, &req, originalModel)
	}
}

func (h *ProxyHandler) handleGeminiNonStream(c *gin.Context, req *adapter.Request, originalModel string) {
	retryReq := h.createRetryRequest(c)

	result, err := retryReq.ExecuteWithRetry(
		c.Request.Context(),
		req.Model,
		func(ctx context.Context, account *model.Account) (*adapter.Response, error) {
			adp := adapter.Get(account.Type)
			if adp == nil {
				return nil, adapter.ErrNoAdapter
			}
			return adp.Send(ctx, account, req)
		},
	)

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{
				"code":    502,
				"message": err.Error(),
				"status":  "UNAVAILABLE",
			},
		})
		return
	}

	resp := result.Response
	if resp.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    400,
				"message": resp.Error.Message,
				"status":  resp.Error.Type,
			},
		})
		return
	}

	// 获取倍率（由中间件设置）
	priceRate := 1.0
	if rate, ok := c.Get("api_key_price_rate"); ok {
		if r, ok := rate.(float64); ok {
			priceRate = r
		}
	}

	// 应用倍率到返回给用户的 token 值
	ratedInputTokens := int(float64(resp.InputTokens) * priceRate)
	ratedOutputTokens := int(float64(resp.OutputTokens) * priceRate)

	// 构建响应体用于日志记录（使用倍率后的 token）
	responseBody, _ := json.Marshal(gin.H{
		"candidates": []gin.H{
			{
				"content": gin.H{
					"parts": []gin.H{{"text": resp.Content}},
					"role":  "model",
				},
				"finishReason": convertGeminiStopReason(resp.StopReason),
			},
		},
		"usageMetadata": gin.H{
			"promptTokenCount":     ratedInputTokens,
			"candidatesTokenCount": ratedOutputTokens,
			"totalTokenCount":      ratedInputTokens + ratedOutputTokens,
		},
	})

	// 获取请求体
	var requestBody []byte
	if rb, ok := c.Get("request_body"); ok {
		requestBody = rb.([]byte)
	}

	// 记录使用统计（使用原始模型名）
	h.recordNonStreamUsage(c, originalModel, resp, requestBody, responseBody, 200, result.AccountID)

	// 返回 Gemini 原生格式（使用倍率后的 token）
	c.JSON(http.StatusOK, gin.H{
		"candidates": []gin.H{
			{
				"content": gin.H{
					"parts": []gin.H{{"text": resp.Content}},
					"role":  "model",
				},
				"finishReason": convertGeminiStopReason(resp.StopReason),
			},
		},
		"usageMetadata": gin.H{
			"promptTokenCount":     ratedInputTokens,
			"candidatesTokenCount": ratedOutputTokens,
			"totalTokenCount":      ratedInputTokens + ratedOutputTokens,
		},
	})
}

func (h *ProxyHandler) handleGeminiStream(c *gin.Context, req *adapter.Request, originalModel string) {
	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // 禁用 Nginx 缓冲

	writer := c.Writer

	// 立即刷新头部，确保客户端知道这是流式响应
	writer.Flush()

	// 获取倍率（由中间件设置）
	priceRate := 1.0
	if rate, ok := c.Get("api_key_price_rate"); ok {
		if r, ok := rate.(float64); ok {
			priceRate = r
		}
	}

	// 使用 RateWriter 包装 writer，在写入时修改 token 值
	rateWriter := NewRateWriter(writer, priceRate)

	// 使用 TailWriter 捕获末尾 2KB 响应（包装 RateWriter）
	tailWriter := adapter.NewTailWriter(rateWriter, 2048)

	retryReq := h.createRetryRequest(c)

	result, err := retryReq.ExecuteStreamWithRetry(
		c.Request.Context(),
		req.Model,
		func(ctx context.Context, account *model.Account, w io.Writer) (*adapter.StreamResult, error) {
			adp := adapter.Get(account.Type)
			if adp == nil {
				return nil, adapter.ErrNoAdapter
			}
			return adp.SendStream(ctx, account, req, w)
		},
		tailWriter,
	)

	if err != nil {
		errData, _ := json.Marshal(gin.H{
			"error": gin.H{
				"code":    502,
				"message": err.Error(),
				"status":  "UNAVAILABLE",
			},
		})
		writer.Write([]byte("data: " + string(errData) + "\n\n"))
		return
	}

	// 获取请求体
	var requestBody []byte
	if rb, ok := c.Get("request_body"); ok {
		requestBody = rb.([]byte)
	}

	// 获取响应末尾内容
	responseTail := tailWriter.Tail()

	// 记录使用统计（使用原始模型名）
	if result != nil && result.Result != nil {
		h.recordUsage(c, originalModel, result.Result, true, requestBody, responseTail, 200, result.AccountID)
	}
}

func convertGeminiStopReason(reason string) string {
	switch reason {
	case "stop", "end_turn":
		return "STOP"
	case "length", "max_tokens":
		return "MAX_TOKENS"
	case "content_filter":
		return "SAFETY"
	default:
		return reason
	}
}

// updateAccountUsageStatus 更新账号用量状态（从 Claude 响应头获取 + 调用 OAuth Usage API）
func (h *ProxyHandler) updateAccountUsageStatus(accountID uint, headers map[string]string) {
	if accountID == 0 {
		return
	}

	log := logger.GetLogger("proxy")

	// 获取 5H 窗口状态（从响应头）
	var usageStatus string
	var rateLimitReset *int64
	if len(headers) > 0 {
		usageStatus = headers["anthropic-ratelimit-unified-5h-status"]
		if resetStr := headers["anthropic-ratelimit-unified-reset"]; resetStr != "" {
			if reset, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
				rateLimitReset = &reset
			}
		}
	}

	// 异步处理：更新状态 + 获取详细用量
	go func() {
		// 1. 更新响应头中的状态
		if usageStatus != "" {
			if err := h.accountRepo.UpdateUsageStatus(accountID, usageStatus, rateLimitReset); err != nil {
				log.ErrorZ("更新账号用量状态失败",
					logger.Uint("account_id", accountID),
					logger.String("usage_status", usageStatus),
					logger.Err(err),
				)
			} else {
				log.DebugZ("更新账号用量状态",
					logger.Uint("account_id", accountID),
					logger.String("usage_status", usageStatus),
				)
			}
		}

		// 2. 获取账号信息
		account, err := h.accountRepo.GetByID(accountID)
		if err != nil {
			log.ErrorZ("获取账号失败",
				logger.Uint("account_id", accountID),
				logger.Err(err),
			)
			return
		}

		// 3. 只有有 access_token 的账号才调用 OAuth Usage API（排除纯 API Key 认证的账号）
		if account.AccessToken == "" {
			return
		}

		// 4. 调用 OAuth Usage API 获取详细用量
		usageData, err := h.fetchClaudeOAuthUsage(account)
		if err != nil {
			log.DebugZ("获取 Claude OAuth 用量失败",
				logger.Uint("account_id", accountID),
				logger.String("account_name", account.Name),
				logger.Err(err),
			)
			return
		}

		// 5. 更新详细用量到数据库
		if err := h.accountRepo.UpdateClaudeUsage(accountID, usageData); err != nil {
			log.ErrorZ("更新账号详细用量失败",
				logger.Uint("account_id", accountID),
				logger.String("account_name", account.Name),
				logger.Err(err),
			)
		} else {
			log.DebugZ("更新账号详细用量",
				logger.Uint("account_id", accountID),
				logger.String("account_name", account.Name),
				logger.Float64("usage_5h_percent", safeFloat(usageData.FiveHour.Utilization)*100),
				logger.Float64("usage_7d_percent", safeFloat(usageData.SevenDay.Utilization)*100),
			)
		}
	}()
}

// safeFloat 安全获取 float64 指针值
func safeFloat(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

// fetchClaudeOAuthUsage 调用 Claude OAuth Usage API 获取用量数据
func (h *ProxyHandler) fetchClaudeOAuthUsage(account *model.Account) (*repository.ClaudeUsageData, error) {
	log := logger.GetLogger("proxy")

	// 获取 HTTP 客户端（优先使用账户代理，否则使用默认代理）
	var client *http.Client
	if account.Proxy != nil && account.Proxy.Enabled {
		// 账户有配置代理，使用账户代理
		client = adapter.GetHTTPClient(account)
		log.DebugZ("OAuth Usage API 使用账户代理",
			logger.Uint("account_id", account.ID),
			logger.String("proxy_name", account.Proxy.Name),
		)
	} else {
		// 账户没有代理，尝试使用默认代理
		defaultProxy, err := service.GetProxyService().GetDefaultProxy()
		if err != nil {
			log.DebugZ("获取默认代理失败",
				logger.Uint("account_id", account.ID),
				logger.Err(err),
			)
		}
		if defaultProxy != nil && defaultProxy.Enabled {
			proxyConfig := &adapter.ProxyConfig{
				Type:     defaultProxy.Type,
				Host:     defaultProxy.Host,
				Port:     defaultProxy.Port,
				Username: defaultProxy.Username,
				Password: defaultProxy.Password,
			}
			client = adapter.GetChromeTLSClientWithProxy(proxyConfig)
			log.DebugZ("OAuth Usage API 使用默认代理",
				logger.Uint("account_id", account.ID),
				logger.String("proxy_name", defaultProxy.Name),
			)
		} else {
			// 没有代理，直连（可能会被 Cloudflare 拦截）
			client = &http.Client{Timeout: 30 * time.Second}
			log.DebugZ("OAuth Usage API 无代理，直连",
				logger.Uint("account_id", account.ID),
			)
		}
	}

	req, err := http.NewRequest("GET", "https://api.anthropic.com/api/oauth/usage", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+account.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("User-Agent", "claude-cli/2.0.53 (external, cli)")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OAuth Usage API 返回 %d", resp.StatusCode)
	}

	var usageData repository.ClaudeUsageData
	if err := json.NewDecoder(resp.Body).Decode(&usageData); err != nil {
		return nil, err
	}

	return &usageData, nil
}

// recordUsage 记录使用统计（异步执行）
func (h *ProxyHandler) recordUsage(c *gin.Context, modelName string, usage *adapter.StreamResult, isStream bool, requestBody []byte, responseBody []byte, upstreamStatusCode int, accountID uint) {
	log := logger.GetLogger("proxy")

	// 从 context 获取 API Key 信息
	apiKeyID, _ := c.Get("api_key_id")
	userID, _ := c.Get("api_key_user_id")
	packageID, _ := c.Get("api_key_package_id")
	billingType, _ := c.Get("api_key_billing_type")

	// 获取倍率（由中间件设置）
	priceRate := 1.0
	if rate, ok := c.Get("api_key_price_rate"); ok {
		if r, ok := rate.(float64); ok {
			priceRate = r
		}
	}

	var uid, keyID, pkgID uint
	var pkgType string
	if userID != nil {
		uid = userID.(uint)
	}
	if apiKeyID != nil {
		keyID = apiKeyID.(uint)
	}
	if packageID != nil {
		pkgID = packageID.(uint)
	}
	if billingType != nil {
		pkgType = billingType.(string)
	}

	// 如果没有用户信息，不记录统计
	if uid == 0 {
		log.Debug("无用户信息，跳过使用统计记录")
		return
	}

	// 应用倍率到 token（用于日志记录和费用计算）
	ratedInputTokens := int(float64(usage.InputTokens) * priceRate)
	ratedOutputTokens := int(float64(usage.OutputTokens) * priceRate)
	ratedCacheCreationTokens := int(float64(usage.CacheCreationInputTokens) * priceRate)
	ratedCacheReadTokens := int(float64(usage.CacheReadInputTokens) * priceRate)

	log.InfoZ("使用统计",
		logger.String("model", modelName),
		logger.Int("原始input", usage.InputTokens),
		logger.Int("原始output", usage.OutputTokens),
		logger.Float64("倍率", priceRate),
		logger.Int("计费input", ratedInputTokens),
		logger.Int("计费output", ratedOutputTokens),
	)

	// 异步记录使用统计
	go func() {
		ctx := context.Background()

		// 计算费用（使用倍率后的 token）
		tokenUsage := &service.TokenUsage{
			InputTokens:              ratedInputTokens,
			OutputTokens:             ratedOutputTokens,
			CacheCreationInputTokens: ratedCacheCreationTokens,
			CacheReadInputTokens:     ratedCacheReadTokens,
		}
		costBreakdown, err := h.pricingService.CalculateCost(ctx, modelName, tokenUsage, 1.0) // 倍率已应用到token，这里用1.0
		if err != nil {
			log.ErrorZ("计算费用失败",
				logger.Uint("user_id", uid),
				logger.String("model", modelName),
				logger.Err(err),
			)
			return
		}

		// 构建请求日志（使用倍率后的 token）
		requestLog := &model.RequestLog{
			AccountID:                accountID,
			UserID:                   &uid,
			APIKeyID:                 &keyID,
			Platform:                 scheduler.DetectPlatform(modelName),
			Model:                    modelName,
			Endpoint:                 c.Request.URL.Path,
			Method:                   c.Request.Method,
			Path:                     c.Request.URL.Path,
			RequestIP:                c.ClientIP(),
			UserAgent:                c.GetHeader("User-Agent"),
			InputTokens:              ratedInputTokens,
			OutputTokens:             ratedOutputTokens,
			CacheCreationInputTokens: ratedCacheCreationTokens,
			CacheReadInputTokens:     ratedCacheReadTokens,
			TotalTokens:              ratedInputTokens + ratedOutputTokens + ratedCacheCreationTokens + ratedCacheReadTokens,
			InputCost:                costBreakdown.InputCost,
			OutputCost:               costBreakdown.OutputCost,
			CacheCreateCost:          costBreakdown.CacheCreateCost,
			CacheReadCost:            costBreakdown.CacheReadCost,
			TotalCost:                costBreakdown.TotalCost,
			Success:                  true,
			StatusCode:               200,
			UpstreamStatusCode:       upstreamStatusCode,
			CreatedAt:                time.Now(),
		}

		// 记录请求头和请求体
		SetRequestDetails(requestLog, c.Request.Header, requestBody)

		// 记录响应体
		// 非流式：完整响应（最大64KB）
		// 流式：末尾内容（用于查看 usage/cache 等信息）
		if len(responseBody) > 0 {
			if len(responseBody) > 65536 {
				requestLog.ResponseBody = string(responseBody[:65536]) + "...[truncated]"
			} else if isStream {
				// 流式响应标记为末尾内容
				requestLog.ResponseBody = "[stream tail] " + string(responseBody)
			} else {
				requestLog.ResponseBody = string(responseBody)
			}
		}

		// 直接保存请求日志到数据库
		LogRequest(requestLog)

		// 记录到 Redis（倍率已应用，这里用 1.0）
		if err := h.usageService.RecordRequest(ctx, uid, keyID, requestLog, 1.0); err != nil {
			log.ErrorZ("记录使用统计失败",
				logger.Uint("user_id", uid),
				logger.Uint("api_key_id", keyID),
				logger.Err(err),
			)
			return
		}

		// 记录模型使用统计（使用倍率后的 token）
		totalTokens := int64(ratedInputTokens + ratedOutputTokens + ratedCacheCreationTokens + ratedCacheReadTokens)
		if err := h.usageService.IncrementModelUsage(ctx, uid, modelName, totalTokens, costBreakdown.TotalCost); err != nil {
			log.ErrorZ("记录模型使用统计失败",
				logger.Uint("user_id", uid),
				logger.String("model", modelName),
				logger.Int64("total_tokens", totalTokens),
				logger.Err(err),
			)
		}

		// 记录账户费用到 Redis
		if accountID > 0 {
			if err := h.usageService.IncrementAccountCost(ctx, accountID, costBreakdown.TotalCost); err != nil {
				log.ErrorZ("记录账户费用失败",
					logger.Uint("account_id", accountID),
					logger.Float64("total_cost", costBreakdown.TotalCost),
					logger.Err(err),
				)
			}
		}

		// 增量更新 MySQL 每日汇总（使用倍率后的 token）
		dailyUsage := &model.DailyUsage{
			RequestCount:             1,
			InputTokens:              int64(ratedInputTokens),
			OutputTokens:             int64(ratedOutputTokens),
			CacheCreationInputTokens: int64(ratedCacheCreationTokens),
			CacheReadInputTokens:     int64(ratedCacheReadTokens),
			TotalTokens:              totalTokens,
			InputCost:                costBreakdown.InputCost,
			OutputCost:               costBreakdown.OutputCost,
			CacheCreateCost:          costBreakdown.CacheCreateCost,
			CacheReadCost:            costBreakdown.CacheReadCost,
			TotalCost:                costBreakdown.TotalCost,
		}
		if err := h.dailyUsageRepo.IncrementUsage(uid, modelName, dailyUsage); err != nil {
			log.ErrorZ("更新每日汇总失败",
				logger.Uint("user_id", uid),
				logger.String("model", modelName),
				logger.Err(err),
			)
		}

		// 更新 API Key 使用统计（MySQL）
		if keyID > 0 {
			if err := h.apiKeyService.IncrementUsage(keyID, totalTokens, costBreakdown.TotalCost); err != nil {
				log.ErrorZ("更新 API Key 使用统计失败",
					logger.Uint("api_key_id", keyID),
					logger.Int64("total_tokens", totalTokens),
					logger.Float64("total_cost", costBreakdown.TotalCost),
					logger.Err(err),
				)
			}
		}

		// 更新绑定的套餐使用量（只扣绑定的套餐）
		if pkgID > 0 {
			// 获取套餐信息用于惰性重置检查
			userPackage, err := h.userPackageRepo.GetByID(pkgID)
			if err == nil && userPackage != nil {
				// 先检查周期是否需要重置（惰性重置）
				if userPackage.ResetPeriodUsageIfNeeded() {
					// 如果重置了，需要先保存重置后的状态
					h.userPackageRepo.Update(userPackage)
				}
				// 增加使用量
				if err := h.userPackageRepo.IncrementUsage(pkgID, pkgType, costBreakdown.TotalCost); err != nil {
					log.ErrorZ("更新用户套餐使用量失败",
						logger.Uint("user_id", uid),
						logger.Uint("package_id", pkgID),
						logger.String("package_type", pkgType),
						logger.Float64("total_cost", costBreakdown.TotalCost),
						logger.Err(err),
					)
				}
			}
		}

		log.InfoZ("使用统计记录成功",
			logger.Uint("user_id", uid),
			logger.Uint("api_key_id", keyID),
			logger.Uint("account_id", accountID),
			logger.Uint("package_id", pkgID),
			logger.String("package_type", pkgType),
			logger.String("model", modelName),
			logger.Int("input_tokens", ratedInputTokens),
			logger.Int("output_tokens", ratedOutputTokens),
			logger.Int("cache_creation_tokens", ratedCacheCreationTokens),
			logger.Int("cache_read_tokens", ratedCacheReadTokens),
			logger.Float64("input_cost", costBreakdown.InputCost),
			logger.Float64("output_cost", costBreakdown.OutputCost),
			logger.Float64("total_cost", costBreakdown.TotalCost),
			logger.Float64("price_rate", priceRate),
			logger.String("client_ip", c.ClientIP()),
		)
	}()
}

// recordNonStreamUsage 记录非流式请求的使用统计
func (h *ProxyHandler) recordNonStreamUsage(c *gin.Context, modelName string, resp *adapter.Response, requestBody []byte, responseBody []byte, upstreamStatusCode int, accountID uint) {
	usage := &adapter.StreamResult{
		InputTokens:  resp.InputTokens,
		OutputTokens: resp.OutputTokens,
	}
	h.recordUsage(c, modelName, usage, false, requestBody, responseBody, upstreamStatusCode, accountID)
}

// getProxyErrorTypeAndCode 根据错误判断错误类型和HTTP状态码
// 如果是未知错误，会自动发现并注册到数据库
func getProxyErrorTypeAndCode(err error) (string, int) {
	if err == nil {
		return model.ErrorTypeUpstreamError, http.StatusBadGateway
	}

	// 优先根据上游状态码判断
	var upstreamErr *adapter.UpstreamError
	if errors.As(err, &upstreamErr) {
		switch upstreamErr.StatusCode {
		case 401:
			return model.ErrorTypeUpstreamAuthFailed, http.StatusBadGateway
		case 403:
			return model.ErrorTypeUpstreamForbidden, http.StatusForbidden
		case 429:
			return model.ErrorTypeUpstreamRateLimit, http.StatusTooManyRequests
		case 500, 502, 503:
			// 继续根据消息内容判断
		}
	}

	errMsg := err.Error()
	errMsgLower := strings.ToLower(errMsg)

	switch {
	// 无可用账户
	case errMsg == "no available account":
		return model.ErrorTypeNoAvailableAccount, http.StatusServiceUnavailable
	case strings.Contains(errMsg, "all accounts failed"):
		return model.ErrorTypeAllAccountsFailed, http.StatusBadGateway

	// 不支持的模型/适配器
	case errMsg == "no adapter found for account type":
		return model.ErrorTypeUnsupportedModel, http.StatusBadGateway
	case strings.Contains(errMsgLower, "unsupported model"):
		return model.ErrorTypeUnsupportedModel, http.StatusBadGateway

	// 超时相关
	case strings.Contains(errMsgLower, "timeout"):
		return model.ErrorTypeUpstreamTimeout, http.StatusBadGateway
	case strings.Contains(errMsgLower, "deadline exceeded"):
		return model.ErrorTypeUpstreamTimeout, http.StatusBadGateway
	case strings.Contains(errMsgLower, "context canceled"):
		return model.ErrorTypeUpstreamTimeout, http.StatusBadGateway

	// 上游限流
	case strings.Contains(errMsgLower, "rate limit"):
		return model.ErrorTypeUpstreamRateLimit, http.StatusTooManyRequests
	case strings.Contains(errMsgLower, "too many requests"):
		return model.ErrorTypeUpstreamRateLimit, http.StatusTooManyRequests
	case strings.Contains(errMsgLower, "overloaded"):
		return model.ErrorTypeUpstreamRateLimit, http.StatusTooManyRequests

	// Token 刷新失败
	case strings.Contains(errMsgLower, "token refresh"):
		return model.ErrorTypeTokenRefreshFailed, http.StatusBadGateway
	case strings.Contains(errMsgLower, "refresh token"):
		return model.ErrorTypeTokenRefreshFailed, http.StatusBadGateway

	default:
		// 自动发现未知错误类型
		autoType := service.ExtractErrorType(errMsg)
		service.GetErrorMessageService().AutoDiscoverError(autoType, errMsg, 502)
		return autoType, http.StatusBadGateway
	}
}

// getCustomErrorMessage 获取自定义错误消息
func getCustomErrorMessage(errorType, originalError string) (string, bool) {
	errorMsgService := service.GetErrorMessageService()
	customMessage, shouldLog := errorMsgService.GetCustomMessage(errorType, originalError)
	return customMessage, shouldLog
}

// truncateForLog 截断字符串用于日志
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
