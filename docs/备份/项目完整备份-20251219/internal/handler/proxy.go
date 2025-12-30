package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// getSessionID 获取会话ID（使用 API Key ID 作为 session）
func (h *ProxyHandler) getSessionID(c *gin.Context) string {
	if apiKeyID, ok := c.Get("api_key_id"); ok {
		if id, ok := apiKeyID.(uint); ok {
			return fmt.Sprintf("apikey:%d", id)
		}
	}
	return ""
}

// getUserInfo 获取用户信息（用于会话绑定）
func (h *ProxyHandler) getUserInfo(c *gin.Context) (userID, apiKeyID uint, clientIP string) {
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
	return
}

// createRetryRequest 创建带用户信息的重试请求
func (h *ProxyHandler) createRetryRequest(c *gin.Context) *scheduler.RetryableRequest {
	userID, apiKeyID, clientIP := h.getUserInfo(c)
	return scheduler.NewRetryableRequest(h.scheduler, h.retryConfig).
		WithSessionID(h.getSessionID(c)).
		WithUserInfo(userID, apiKeyID, clientIP)
}

// ChatCompletions OpenAI 兼容接口 POST /v1/chat/completions
func (h *ProxyHandler) ChatCompletions(c *gin.Context) {
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

	// 保存原始请求体到 context
	c.Set("request_body", rawBody)

	// 检测账户类型（支持 "type,model" 格式）
	accountType := scheduler.DetectAccountType(req.Model)
	actualModel := scheduler.GetActualModel(req.Model)
	req.Model = actualModel

	if req.Stream {
		h.handleOpenAIStreamWithRetry(c, &req, accountType)
	} else {
		h.handleOpenAINonStreamWithRetry(c, &req, accountType)
	}
}

// Messages Claude 消息接口 POST /v1/messages (透传模式)
func (h *ProxyHandler) Messages(c *gin.Context) {
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

	// 4. 构建透传请求
	req := &adapter.Request{
		Model:   basic.Model,
		Stream:  basic.Stream,
		RawBody: rawBody,
		Headers: clientHeaders,
	}

	// 5. 检测账户类型
	accountType := scheduler.DetectAccountType(req.Model)
	actualModel := scheduler.GetActualModel(req.Model)
	req.Model = actualModel

	if req.Stream {
		h.handleClaudeStreamWithRetry(c, req, accountType)
	} else {
		h.handleClaudeNonStreamWithRetry(c, req, accountType)
	}
}

// OpenAI 非流式响应（带重试）
func (h *ProxyHandler) handleOpenAINonStreamWithRetry(c *gin.Context, req *adapter.Request, accountType string) {
	retryReq := h.createRetryRequest(c)

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
		errorType := getProxyErrorType(err)
		response.CustomError(c, http.StatusBadGateway, errorType, err.Error())
		return
	}

	resp := result.Response
	if resp.Error != nil {
		response.CustomError(c, http.StatusBadRequest, model.ErrorTypeBadRequest, resp.Error.Message)
		return
	}

	// 构建响应体用于日志记录
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
			"prompt_tokens":     resp.InputTokens,
			"completion_tokens": resp.OutputTokens,
			"total_tokens":      resp.InputTokens + resp.OutputTokens,
		},
	})

	// 获取请求体
	var requestBody []byte
	if rb, ok := c.Get("request_body"); ok {
		requestBody = rb.([]byte)
	}

	// 记录使用统计
	h.recordNonStreamUsage(c, req.Model, resp, requestBody, responseBody, 200, result.AccountID)

	// 返回 OpenAI 格式
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
			"prompt_tokens":     resp.InputTokens,
			"completion_tokens": resp.OutputTokens,
			"total_tokens":      resp.InputTokens + resp.OutputTokens,
		},
	})
}

// OpenAI 流式响应（带重试）
func (h *ProxyHandler) handleOpenAIStreamWithRetry(c *gin.Context, req *adapter.Request, accountType string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	writer := c.Writer
	retryReq := h.createRetryRequest(c)

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
		writer,
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

	// 记录使用统计
	if result != nil && result.Result != nil {
		h.recordUsage(c, req.Model, result.Result, true, requestBody, nil, 200, result.AccountID)
	}

	writer.Write([]byte("data: [DONE]\n\n"))
}

// Claude 非流式响应（带重试）
func (h *ProxyHandler) handleClaudeNonStreamWithRetry(c *gin.Context, req *adapter.Request, accountType string) {
	retryReq := h.createRetryRequest(c)

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
		errorType := getProxyErrorType(err)
		customMsg, _ := getCustomErrorMessage(errorType, err.Error())
		c.JSON(http.StatusBadGateway, gin.H{
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

	// 构建响应体用于日志记录
	responseBody, _ := json.Marshal(gin.H{
		"id":          resp.ID,
		"type":        "message",
		"role":        "assistant",
		"model":       resp.Model,
		"content":     []gin.H{{"type": "text", "text": resp.Content}},
		"stop_reason": resp.StopReason,
		"usage": gin.H{
			"input_tokens":  resp.InputTokens,
			"output_tokens": resp.OutputTokens,
		},
	})

	// 获取请求体
	var requestBody []byte
	if rb, ok := c.Get("request_body"); ok {
		requestBody = rb.([]byte)
	}

	// 记录使用统计
	h.recordNonStreamUsage(c, req.Model, resp, requestBody, responseBody, 200, result.AccountID)

	// 更新账号用量状态（从响应头获取）
	h.updateAccountUsageStatus(result.AccountID, resp.Headers)

	// 返回 Claude 格式
	c.JSON(http.StatusOK, gin.H{
		"id":          resp.ID,
		"type":        "message",
		"role":        "assistant",
		"model":       resp.Model,
		"content":     []gin.H{{"type": "text", "text": resp.Content}},
		"stop_reason": resp.StopReason,
		"usage": gin.H{
			"input_tokens":  resp.InputTokens,
			"output_tokens": resp.OutputTokens,
		},
	})
}

// Claude 流式响应（带重试）
func (h *ProxyHandler) handleClaudeStreamWithRetry(c *gin.Context, req *adapter.Request, accountType string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	writer := c.Writer
	retryReq := h.createRetryRequest(c)

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
		writer,
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

	// 记录使用统计
	if result != nil && result.Result != nil {
		h.recordUsage(c, req.Model, result.Result, true, requestBody, nil, 200, result.AccountID)
		// 更新账号用量状态（从响应头获取）
		h.updateAccountUsageStatus(result.AccountID, result.Result.Headers)
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

	if req.Stream {
		h.handleGeminiStream(c, &req)
	} else {
		h.handleGeminiNonStream(c, &req)
	}
}

func (h *ProxyHandler) handleGeminiNonStream(c *gin.Context, req *adapter.Request) {
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

	// 构建响应体用于日志记录
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
			"promptTokenCount":     resp.InputTokens,
			"candidatesTokenCount": resp.OutputTokens,
			"totalTokenCount":      resp.InputTokens + resp.OutputTokens,
		},
	})

	// 获取请求体
	var requestBody []byte
	if rb, ok := c.Get("request_body"); ok {
		requestBody = rb.([]byte)
	}

	// 记录使用统计
	h.recordNonStreamUsage(c, req.Model, resp, requestBody, responseBody, 200, result.AccountID)

	// 返回 Gemini 原生格式
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
			"promptTokenCount":     resp.InputTokens,
			"candidatesTokenCount": resp.OutputTokens,
			"totalTokenCount":      resp.InputTokens + resp.OutputTokens,
		},
	})
}

func (h *ProxyHandler) handleGeminiStream(c *gin.Context, req *adapter.Request) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	writer := c.Writer
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
		writer,
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

	// 记录使用统计
	if result != nil && result.Result != nil {
		h.recordUsage(c, req.Model, result.Result, true, requestBody, nil, 200, result.AccountID)
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

// TestProxyRequest 代理测试请求
type TestProxyRequest struct {
	Model       string `json:"model"`
	Message     string `json:"message"`
	System      string `json:"system,omitempty"`
	MaxTokens   int    `json:"max_tokens,omitempty"`
	AccountType string `json:"account_type,omitempty"`
	ClientMode  string `json:"client_mode,omitempty"` // http, claude_code, sdk
}

// TestProxy 管理员代理测试接口 POST /api/admin/proxy/test
func (h *ProxyHandler) TestProxy(c *gin.Context) {
	var req TestProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if req.Model == "" {
		req.Model = "claude-3-5-sonnet-20241022"
	}
	if req.Message == "" {
		req.Message = "Hello! Please say hi."
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 1024
	}

	// 构建代理请求
	proxyReq := &adapter.Request{
		Model:     req.Model,
		System:    req.System,
		MaxTokens: req.MaxTokens,
		Messages: []adapter.Message{
			{Role: "user", Content: req.Message},
		},
	}

	// 检测账户类型
	accountType := req.AccountType
	if accountType == "" {
		accountType = scheduler.DetectAccountType(req.Model)
	}
	actualModel := scheduler.GetActualModel(req.Model)
	proxyReq.Model = actualModel

	// 执行请求
	retryReq := h.createRetryRequest(c)

	modelName := proxyReq.Model
	if accountType != "" {
		modelName = accountType + "," + proxyReq.Model
	}

	result, err := retryReq.ExecuteWithRetry(
		c.Request.Context(),
		modelName,
		func(ctx context.Context, account *model.Account) (*adapter.Response, error) {
			adp := adapter.Get(account.Type)
			if adp == nil {
				return nil, adapter.ErrNoAdapter
			}
			return adp.Send(ctx, account, proxyReq)
		},
	)

	if err != nil {
		response.Error(c, http.StatusBadGateway, "代理请求失败: "+err.Error())
		return
	}

	resp := result.Response
	if resp.Error != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error": gin.H{
				"type":    resp.Error.Type,
				"message": resp.Error.Message,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":            resp.ID,
			"model":         resp.Model,
			"content":       resp.Content,
			"stop_reason":   resp.StopReason,
			"input_tokens":  resp.InputTokens,
			"output_tokens": resp.OutputTokens,
		},
	})
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
				log.Error("更新账号用量状态失败: accountID=%d, error=%v", accountID, err)
			} else {
				log.Debug("更新账号用量状态: accountID=%d, status=%s", accountID, usageStatus)
			}
		}

		// 2. 获取账号信息
		account, err := h.accountRepo.GetByID(accountID)
		if err != nil {
			log.Error("获取账号失败: accountID=%d, error=%v", accountID, err)
			return
		}

		// 3. 只有有 access_token 的账号才调用 OAuth Usage API（排除纯 API Key 认证的账号）
		if account.AccessToken == "" {
			return
		}

		// 4. 调用 OAuth Usage API 获取详细用量
		usageData, err := h.fetchClaudeOAuthUsage(account)
		if err != nil {
			log.Debug("获取 Claude OAuth 用量失败: accountID=%d, error=%v", accountID, err)
			return
		}

		// 5. 更新详细用量到数据库
		if err := h.accountRepo.UpdateClaudeUsage(accountID, usageData); err != nil {
			log.Error("更新账号详细用量失败: accountID=%d, error=%v", accountID, err)
		} else {
			log.Debug("更新账号详细用量: accountID=%d, 5H=%.1f%%, 7D=%.1f%%",
				accountID,
				safeFloat(usageData.FiveHour.Utilization),
				safeFloat(usageData.SevenDay.Utilization))
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
	client := adapter.GetHTTPClient(account)

	req, err := http.NewRequest("GET", "https://api.anthropic.com/api/oauth/usage", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+account.AccessToken)
	req.Header.Set("Content-Type", "application/json")

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

	// 异步记录使用统计
	go func() {
		ctx := context.Background()

		// 获取用户费率倍率
		user, err := h.userRepo.GetByID(uid)
		if err != nil {
			log.Error("获取用户信息失败: %v", err)
			return
		}
		priceRate := user.PriceRate
		if priceRate == 0 {
			priceRate = 1.0 // 默认 1.0 倍
		}

		// 计算费用
		tokenUsage := &service.TokenUsage{
			InputTokens:              usage.InputTokens,
			OutputTokens:             usage.OutputTokens,
			CacheCreationInputTokens: usage.CacheCreationInputTokens,
			CacheReadInputTokens:     usage.CacheReadInputTokens,
		}
		costBreakdown, err := h.pricingService.CalculateCost(ctx, modelName, tokenUsage, priceRate)
		if err != nil {
			log.Error("计算费用失败: %v", err)
			return
		}

		// 构建请求日志
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
			InputTokens:              usage.InputTokens,
			OutputTokens:             usage.OutputTokens,
			CacheCreationInputTokens: usage.CacheCreationInputTokens,
			CacheReadInputTokens:     usage.CacheReadInputTokens,
			TotalTokens:              usage.InputTokens + usage.OutputTokens + usage.CacheCreationInputTokens + usage.CacheReadInputTokens,
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

		// 记录响应体（非流式）
		if !isStream && len(responseBody) > 0 {
			if len(responseBody) > 65536 {
				requestLog.ResponseBody = string(responseBody[:65536]) + "...[truncated]"
			} else {
				requestLog.ResponseBody = string(responseBody)
			}
		}

		// 直接保存请求日志到数据库
		LogRequest(requestLog)

		// 记录到 Redis
		if err := h.usageService.RecordRequest(ctx, uid, keyID, requestLog, priceRate); err != nil {
			log.Error("记录使用统计失败: %v", err)
			return
		}

		// 记录模型使用统计
		totalTokens := int64(usage.InputTokens + usage.OutputTokens + usage.CacheCreationInputTokens + usage.CacheReadInputTokens)
		if err := h.usageService.IncrementModelUsage(ctx, uid, modelName, totalTokens, costBreakdown.TotalCost); err != nil {
			log.Error("记录模型使用统计失败: %v", err)
		}

		// 记录账户费用到 Redis
		if accountID > 0 {
			if err := h.usageService.IncrementAccountCost(ctx, accountID, costBreakdown.TotalCost); err != nil {
				log.Error("记录账户费用失败: %v", err)
			}
		}

		// 增量更新 MySQL 每日汇总
		dailyUsage := &model.DailyUsage{
			RequestCount:             1,
			InputTokens:              int64(usage.InputTokens),
			OutputTokens:             int64(usage.OutputTokens),
			CacheCreationInputTokens: int64(usage.CacheCreationInputTokens),
			CacheReadInputTokens:     int64(usage.CacheReadInputTokens),
			TotalTokens:              totalTokens,
			InputCost:                costBreakdown.InputCost,
			OutputCost:               costBreakdown.OutputCost,
			CacheCreateCost:          costBreakdown.CacheCreateCost,
			CacheReadCost:            costBreakdown.CacheReadCost,
			TotalCost:                costBreakdown.TotalCost,
		}
		if err := h.dailyUsageRepo.IncrementUsage(uid, modelName, dailyUsage); err != nil {
			log.Error("更新每日汇总失败: %v", err)
		}

		// 更新 API Key 使用统计（MySQL）
		if keyID > 0 {
			if err := h.apiKeyService.IncrementUsage(keyID, totalTokens, costBreakdown.TotalCost); err != nil {
				log.Error("更新 API Key 使用统计失败: %v", err)
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
					log.Error("更新用户套餐使用量失败: %v", err)
				}
			}
		}

		log.Debug("使用统计记录成功 - UserID: %d, Model: %s, TotalCost: %.6f", uid, modelName, costBreakdown.TotalCost)
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

// getProxyErrorType 根据错误判断错误类型
// 如果是未知错误，会自动发现并注册到数据库
func getProxyErrorType(err error) string {
	if err == nil {
		return model.ErrorTypeUpstreamError
	}
	errMsg := err.Error()
	errMsgLower := strings.ToLower(errMsg)

	switch {
	// 无可用账户
	case errMsg == "no available account":
		return model.ErrorTypeNoAvailableAccount
	case strings.Contains(errMsg, "all accounts failed"):
		return model.ErrorTypeAllAccountsFailed

	// 不支持的模型/适配器
	case errMsg == "no adapter found for account type":
		return model.ErrorTypeUnsupportedModel
	case strings.Contains(errMsgLower, "unsupported model"):
		return model.ErrorTypeUnsupportedModel

	// 超时相关
	case strings.Contains(errMsgLower, "timeout"):
		return model.ErrorTypeUpstreamTimeout
	case strings.Contains(errMsgLower, "deadline exceeded"):
		return model.ErrorTypeUpstreamTimeout
	case strings.Contains(errMsgLower, "context canceled"):
		return model.ErrorTypeUpstreamTimeout

	// 上游限流
	case strings.Contains(errMsgLower, "rate limit"):
		return model.ErrorTypeUpstreamRateLimit
	case strings.Contains(errMsgLower, "too many requests"):
		return model.ErrorTypeUpstreamRateLimit
	case strings.Contains(errMsgLower, "overloaded"):
		return model.ErrorTypeUpstreamRateLimit

	// 上游认证失败
	case strings.Contains(errMsgLower, "unauthorized"):
		return model.ErrorTypeUpstreamAuthFailed
	case strings.Contains(errMsgLower, "invalid api key"):
		return model.ErrorTypeUpstreamAuthFailed
	case strings.Contains(errMsgLower, "authentication"):
		return model.ErrorTypeUpstreamAuthFailed
	case strings.Contains(errMsgLower, "invalid_api_key"):
		return model.ErrorTypeUpstreamAuthFailed
	case strings.Contains(errMsgLower, "permission denied"):
		return model.ErrorTypeUpstreamAuthFailed

	// Token 刷新失败
	case strings.Contains(errMsgLower, "token refresh"):
		return model.ErrorTypeTokenRefreshFailed
	case strings.Contains(errMsgLower, "refresh token"):
		return model.ErrorTypeTokenRefreshFailed

	default:
		// 自动发现未知错误类型
		autoType := service.ExtractErrorType(errMsg)
		service.GetErrorMessageService().AutoDiscoverError(autoType, errMsg, 502)
		return autoType
	}
}

// getCustomErrorMessage 获取自定义错误消息
func getCustomErrorMessage(errorType, originalError string) (string, bool) {
	errorMsgService := service.GetErrorMessageService()
	customMessage, shouldLog := errorMsgService.GetCustomMessage(errorType, originalError)
	return customMessage, shouldLog
}
