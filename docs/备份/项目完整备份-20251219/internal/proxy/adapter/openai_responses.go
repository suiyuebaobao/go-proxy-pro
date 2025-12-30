package adapter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go-aiproxy/internal/model"
	"go-aiproxy/pkg/logger"
)

// OpenAIResponsesAdapter OpenAI Responses API 适配器
// 支持 /responses API 转发到上游服务
// 参考 claude-relay-service 的 openaiResponsesRelayService.js 实现
type OpenAIResponsesAdapter struct{}

func init() {
	Register(&OpenAIResponsesAdapter{})
}

func (a *OpenAIResponsesAdapter) Name() string {
	return "openai-responses"
}

func (a *OpenAIResponsesAdapter) Platform() string {
	return model.PlatformOpenAI
}

func (a *OpenAIResponsesAdapter) SupportedTypes() []string {
	return []string{model.AccountTypeOpenAIResponses}
}

// 默认 OpenAI Responses API 端点
const (
	DefaultOpenAIResponsesBaseURL = "https://chatgpt.com/backend-api/codex"
)

// OpenAIResponsesRequest OpenAI Responses API 请求格式
type OpenAIResponsesRequest struct {
	Model        string                   `json:"model"`
	Instructions string                   `json:"instructions,omitempty"`
	Input        []map[string]interface{} `json:"input,omitempty"`
	Stream       bool                     `json:"stream"`
	Reasoning    map[string]interface{}   `json:"reasoning,omitempty"`
	// 其他字段透传
	Store          bool     `json:"store,omitempty"`
	PromptCacheKey string   `json:"prompt_cache_key,omitempty"`
	Include        []string `json:"include,omitempty"`
}

// OpenAIResponsesUsageHeaders OpenAI Responses 使用量响应头
type OpenAIResponsesUsageHeaders struct {
	PrimaryUsedPercent          float64 `json:"primary_used_percent"`
	PrimaryResetAfterSeconds    int     `json:"primary_reset_after_seconds"`
	PrimaryWindowMinutes        int     `json:"primary_window_minutes"`
	SecondaryUsedPercent        float64 `json:"secondary_used_percent"`
	SecondaryResetAfterSeconds  int     `json:"secondary_reset_after_seconds"`
	SecondaryWindowMinutes      int     `json:"secondary_window_minutes"`
	PrimaryOverSecondaryPercent float64 `json:"primary_over_secondary_percent"`
}

// Send 非流式请求（OpenAI Responses API 强制使用流式，这里收集完整响应）
func (a *OpenAIResponsesAdapter) Send(ctx context.Context, account *model.Account, req *Request) (*Response, error) {
	log := logger.GetLogger("openai-responses")

	// OpenAI Responses API 强制使用流式，我们需要收集完整响应
	var buf bytes.Buffer
	result, err := a.SendStream(ctx, account, req, &buf)
	if err != nil {
		return nil, err
	}

	// 解析收集的响应
	// OpenAI Responses 响应格式需要从 SSE 中解析
	content := buf.String()

	log.Info("OpenAI Responses 非流式请求完成 - Model: %s, InputTokens: %d, OutputTokens: %d",
		req.Model, result.InputTokens, result.OutputTokens)

	return &Response{
		ID:           fmt.Sprintf("resp-%d", result.InputTokens),
		Model:        req.Model,
		Content:      content,
		StopReason:   "end_turn",
		InputTokens:  result.InputTokens,
		OutputTokens: result.OutputTokens,
	}, nil
}

// SendStream 流式请求
// 参考 claude-relay 的 openaiResponsesRelayService.js 实现
func (a *OpenAIResponsesAdapter) SendStream(ctx context.Context, account *model.Account, req *Request, writer io.Writer) (*StreamResult, error) {
	log := logger.GetLogger("openai-responses")

	// 构建目标 URL: baseURL + path
	// 类似 claude-relay: const targetUrl = `${fullAccount.baseApi}${req.path}`
	baseURL := DefaultOpenAIResponsesBaseURL
	if account.BaseURL != "" {
		baseURL = strings.TrimSuffix(account.BaseURL, "/")
	}

	// 获取请求路径，默认为 /responses
	requestPath := req.Path
	if requestPath == "" {
		requestPath = "/responses"
	}

	targetURL := baseURL + requestPath
	log.Info("OpenAI Responses 转发目标 - BaseURL: %s, Path: %s, TargetURL: %s",
		baseURL, requestPath, targetURL)

	// 构建请求体
	// 如果上游不是 chatgpt.com，可能需要转换格式
	var body []byte
	var err error

	if len(req.RawBody) > 0 {
		// 解析原始请求体
		var rawReq map[string]interface{}
		if err := json.Unmarshal(req.RawBody, &rawReq); err == nil {
			// 检查是否需要转换格式（如果有 input 但没有 messages，转换为 messages 格式）
			if input, hasInput := rawReq["input"]; hasInput {
				if _, hasMessages := rawReq["messages"]; !hasMessages {
					// 转换 input 为 messages 格式（标准 OpenAI）
					messages := a.convertInputToMessages(input)
					if len(messages) > 0 {
						rawReq["messages"] = messages
						delete(rawReq, "input")
						// 同时处理 instructions 字段
						if instructions, ok := rawReq["instructions"].(string); ok && instructions != "" {
							// 将 instructions 作为 system message 添加到开头
							systemMsg := map[string]interface{}{
								"role":    "system",
								"content": instructions,
							}
							rawReq["messages"] = append([]interface{}{systemMsg}, messages...)
							delete(rawReq, "instructions")
						}
						log.Debug("OpenAI Responses: 已转换 input 为 messages 格式")
					}
				}
			}
			body, err = json.Marshal(rawReq)
			if err != nil {
				return nil, fmt.Errorf("marshal request: %w", err)
			}
		} else {
			body = req.RawBody
		}
	} else {
		// 构建标准 OpenAI 请求
		openaiReq := a.convertToOpenAIRequest(req)
		body, err = json.Marshal(openaiReq)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 设置请求头
	a.setRequestHeaders(httpReq, account, req)

	log.Debug("OpenAI Responses 请求开始 - URL: %s, AccountID: %d, Model: %s",
		targetURL, account.ID, req.Model)

	// 发送请求 - 使用智能 HTTP 客户端（chatgpt.com 使用 Chrome TLS）
	client := GetSmartHTTPClient(account, targetURL)
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Error("OpenAI Responses 请求失败 - 网络错误: %v", err)
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 处理错误响应
	if resp.StatusCode != http.StatusOK {
		return a.handleErrorResponse(resp, account, log)
	}

	log.Debug("OpenAI Responses 响应状态码: %d, 开始接收流式数据", resp.StatusCode)

	// 提取 Usage 头部
	usageHeaders := a.extractUsageHeaders(resp.Header)
	if usageHeaders != nil {
		log.Debug("OpenAI Responses Usage - Primary: %.1f%%, Secondary: %.1f%%",
			usageHeaders.PrimaryUsedPercent, usageHeaders.SecondaryUsedPercent)
	}

	// 处理流式响应
	return a.processStreamResponse(resp, writer, log)
}

// setRequestHeaders 设置请求头
// 参考 claude-relay 的 openaiResponsesRelayService.js
func (a *OpenAIResponsesAdapter) setRequestHeaders(httpReq *http.Request, account *model.Account, req *Request) {
	// 基本头部
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	// 认证头：按优先级选择 token
	// 1. SessionKey - 用于 Cookie/Session 认证
	// 2. AccessToken - OAuth 流程获取的 access_token
	// 3. APIKey - 长期有效的 API 密钥
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
	if req.Headers != nil {
		// User-Agent - 优先使用客户端的，否则使用默认
		if ua := req.Headers["User-Agent"]; ua != "" {
			httpReq.Header.Set("User-Agent", ua)
		} else {
			httpReq.Header.Set("User-Agent", "OpenAI-Responses-Proxy/1.0")
		}

		// Session ID
		if sessionID := req.Headers["Session_id"]; sessionID != "" {
			httpReq.Header.Set("session_id", sessionID)
		}

		// Version
		if version := req.Headers["Version"]; version != "" {
			httpReq.Header.Set("version", version)
		}

		// Originator
		if originator := req.Headers["Originator"]; originator != "" {
			httpReq.Header.Set("originator", originator)
		}

		// OpenAI Beta 头部（如果客户端提供）
		if beta := req.Headers["Openai-Beta"]; beta != "" {
			httpReq.Header.Set("openai-beta", beta)
		}
	} else {
		// 默认 User-Agent
		httpReq.Header.Set("User-Agent", "OpenAI-Responses-Proxy/1.0")
	}

	// 如果是 chatgpt.com 的请求，添加特定头部
	if strings.Contains(httpReq.URL.Host, "chatgpt.com") {
		httpReq.Header.Set("openai-beta", "responses=experimental")
		// ChatGPT Account ID (如果账户有配置)
		if account.OrganizationID != "" {
			httpReq.Header.Set("chatgpt-account-id", account.OrganizationID)
		}
	}
}

// handleErrorResponse 处理错误响应
func (a *OpenAIResponsesAdapter) handleErrorResponse(resp *http.Response, account *model.Account, log *logger.Logger) (*StreamResult, error) {
	respBody, _ := ReadResponseBody(resp)
	log.Error("OpenAI Responses API 错误 - StatusCode: %d, Body: %s", resp.StatusCode, string(respBody))

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		// 429 限流
		return nil, fmt.Errorf("rate limit exceeded: %s", string(respBody))
	case http.StatusUnauthorized:
		// 401 认证失败
		return nil, fmt.Errorf("authentication failed: %s", string(respBody))
	case http.StatusPaymentRequired:
		// 402 付费要求
		return nil, fmt.Errorf("payment required: %s", string(respBody))
	case http.StatusForbidden:
		// 403 禁止访问
		return nil, fmt.Errorf("forbidden: %s", string(respBody))
	default:
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}
}

// extractUsageHeaders 提取使用量头部
func (a *OpenAIResponsesAdapter) extractUsageHeaders(headers http.Header) *OpenAIResponsesUsageHeaders {
	usage := &OpenAIResponsesUsageHeaders{}
	hasData := false

	if v := headers.Get("x-codex-primary-used-percent"); v != "" {
		fmt.Sscanf(v, "%f", &usage.PrimaryUsedPercent)
		hasData = true
	}
	if v := headers.Get("x-codex-primary-reset-after-seconds"); v != "" {
		fmt.Sscanf(v, "%d", &usage.PrimaryResetAfterSeconds)
		hasData = true
	}
	if v := headers.Get("x-codex-primary-window-minutes"); v != "" {
		fmt.Sscanf(v, "%d", &usage.PrimaryWindowMinutes)
		hasData = true
	}
	if v := headers.Get("x-codex-secondary-used-percent"); v != "" {
		fmt.Sscanf(v, "%f", &usage.SecondaryUsedPercent)
		hasData = true
	}
	if v := headers.Get("x-codex-secondary-reset-after-seconds"); v != "" {
		fmt.Sscanf(v, "%d", &usage.SecondaryResetAfterSeconds)
		hasData = true
	}
	if v := headers.Get("x-codex-secondary-window-minutes"); v != "" {
		fmt.Sscanf(v, "%d", &usage.SecondaryWindowMinutes)
		hasData = true
	}
	if v := headers.Get("x-codex-primary-over-secondary-limit-percent"); v != "" {
		fmt.Sscanf(v, "%f", &usage.PrimaryOverSecondaryPercent)
		hasData = true
	}

	if !hasData {
		return nil
	}
	return usage
}

// processStreamResponse 处理流式响应
func (a *OpenAIResponsesAdapter) processStreamResponse(resp *http.Response, writer io.Writer, log *logger.Logger) (*StreamResult, error) {
	result := &StreamResult{
		Headers: make(map[string]string),
	}

	// 保存响应头
	for k, v := range resp.Header {
		if len(v) > 0 {
			result.Headers[k] = v[0]
		}
	}

	scanner := bufio.NewScanner(resp.Body)
	// 增大缓冲区以处理大响应
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// 直接转发 SSE 数据
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			if data == "[DONE]" {
				log.Debug("OpenAI Responses Stream 接收完成")
				break
			}

			// 尝试解析 usage 信息
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(data), &event); err == nil {
				// 检查是否是 response.completed 事件（包含 usage）
				if eventType, ok := event["type"].(string); ok && eventType == "response.completed" {
					if response, ok := event["response"].(map[string]interface{}); ok {
						if usage, ok := response["usage"].(map[string]interface{}); ok {
							if inputTokens, ok := usage["input_tokens"].(float64); ok {
								result.InputTokens = int(inputTokens)
							}
							if outputTokens, ok := usage["output_tokens"].(float64); ok {
								result.OutputTokens = int(outputTokens)
							}
						}
					}
				}
			}

			// 转发给客户端
			writer.Write([]byte(line + "\n\n"))
		} else if line != "" {
			// 转发其他非空行（如 event: 行）
			writer.Write([]byte(line + "\n"))
		}
	}

	if err := scanner.Err(); err != nil {
		log.Error("OpenAI Responses Stream 读取错误: %v", err)
		return result, err
	}

	log.Info("OpenAI Responses Stream 请求完成 - InputTokens: %d, OutputTokens: %d",
		result.InputTokens, result.OutputTokens)

	return result, nil
}

// convertRequest 转换通用请求为 OpenAI Responses 格式
func (a *OpenAIResponsesAdapter) convertRequest(req *Request) *OpenAIResponsesRequest {
	openaiReq := &OpenAIResponsesRequest{
		Model:  req.Model,
		Stream: req.Stream,
	}

	// 转换 messages 为 input
	if len(req.Messages) > 0 {
		input := make([]map[string]interface{}, 0, len(req.Messages))
		for _, msg := range req.Messages {
			item := map[string]interface{}{
				"role": msg.Role,
			}
			switch v := msg.Content.(type) {
			case string:
				item["content"] = v
			default:
				item["content"] = v
			}
			input = append(input, item)
		}
		openaiReq.Input = input
	}

	// System prompt 作为 instructions
	if req.System != "" {
		openaiReq.Instructions = req.System
	}

	return openaiReq
}

// convertInputToMessages 将 Responses API 的 input 格式转换为标准 OpenAI messages 格式
func (a *OpenAIResponsesAdapter) convertInputToMessages(input interface{}) []interface{} {
	var messages []interface{}

	switch v := input.(type) {
	case []interface{}:
		for _, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				msg := map[string]interface{}{
					"role":    itemMap["role"],
					"content": itemMap["content"],
				}
				messages = append(messages, msg)
			}
		}
	case string:
		// 如果 input 是字符串，转换为单条 user message
		messages = append(messages, map[string]interface{}{
			"role":    "user",
			"content": v,
		})
	}

	return messages
}

// convertToOpenAIRequest 转换为标准 OpenAI Chat Completions 请求
func (a *OpenAIResponsesAdapter) convertToOpenAIRequest(req *Request) map[string]interface{} {
	openaiReq := map[string]interface{}{
		"model":  req.Model,
		"stream": req.Stream,
	}

	// 构建 messages
	var messages []interface{}

	// 添加 system message
	if req.System != "" {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": req.System,
		})
	}

	// 转换 messages
	for _, msg := range req.Messages {
		messages = append(messages, map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	openaiReq["messages"] = messages
	return openaiReq
}
