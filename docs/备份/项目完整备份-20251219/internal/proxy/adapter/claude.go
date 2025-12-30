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

type ClaudeAdapter struct{}

func init() {
	Register(&ClaudeAdapter{})
}

func (a *ClaudeAdapter) Name() string {
	return "claude"
}

func (a *ClaudeAdapter) Platform() string {
	return model.PlatformClaude
}

func (a *ClaudeAdapter) SupportedTypes() []string {
	return []string{
		model.AccountTypeClaudeOfficial,
		model.AccountTypeClaudeConsole,
	}
}

// Send 发送非流式请求 - 透传模式
func (a *ClaudeAdapter) Send(ctx context.Context, account *model.Account, req *Request) (*Response, error) {
	log := logger.GetLogger("proxy")

	// 直接使用原始请求体，不做任何解析和转换
	body := req.RawBody
	if len(body) == 0 {
		return nil, fmt.Errorf("empty request body")
	}

	baseURL := "https://api.anthropic.com"
	if account.BaseURL != "" {
		baseURL = account.BaseURL
	}

	fullURL := baseURL + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// 透传客户端 headers + 设置认证
	a.setHeaders(httpReq, account, req.Headers)

	// 调试：记录发送的所有头
	headerLog := make([]string, 0)
	for key, values := range httpReq.Header {
		headerLog = append(headerLog, fmt.Sprintf("%s: %s", key, strings.Join(values, ",")))
	}
	log.Debug("Claude 请求 | URL: %s | AccountID: %d | Model: %s", fullURL, account.ID, req.Model)
	log.Debug("Claude 请求头 | %s", strings.Join(headerLog, " | "))

	client := GetHTTPClient(account)
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Error("Claude 网络错误: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ReadResponseBody(resp)
	if err != nil {
		return nil, err
	}

	log.Debug("Claude 响应 | StatusCode: %d | BodyLen: %d", resp.StatusCode, len(respBody))

	// 非200状态码，返回错误
	if resp.StatusCode != http.StatusOK {
		log.Error("Claude API 错误 | StatusCode: %d | Body: %s", resp.StatusCode, truncateBody(string(respBody), 500))
		return nil, fmt.Errorf("%s", string(respBody))
	}

	// 解析响应提取 usage 信息
	response, err := a.parseResponse(respBody)
	if err != nil {
		return nil, err
	}

	// 提取 Claude 限流相关响应头
	response.Headers = extractRateLimitHeaders(resp.Header)

	return response, nil
}

// SendStream 发送流式请求 - 透传模式，同时解析 usage
func (a *ClaudeAdapter) SendStream(ctx context.Context, account *model.Account, req *Request, writer io.Writer) (*StreamResult, error) {
	log := logger.GetLogger("proxy")

	body := req.RawBody
	if len(body) == 0 {
		return nil, fmt.Errorf("empty request body")
	}

	baseURL := "https://api.anthropic.com"
	if account.BaseURL != "" {
		baseURL = account.BaseURL
	}

	fullURL := baseURL + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// 透传客户端 headers + 设置认证
	a.setHeaders(httpReq, account, req.Headers)
	httpReq.Header.Set("Accept", "text/event-stream")

	log.Debug("Claude Stream 请求 | URL: %s | AccountID: %d", fullURL, account.ID)

	client := GetHTTPClient(account)
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := ReadResponseBody(resp)
		log.Error("Claude Stream 错误 | StatusCode: %d | Body: %s", resp.StatusCode, string(respBody))
		return nil, fmt.Errorf("%s", string(respBody))
	}

	// 提取 Claude 限流相关响应头
	headers := extractRateLimitHeaders(resp.Header)

	// 透传 SSE 流并解析 usage
	result := &StreamResult{
		Headers: headers,
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024) // 64KB buffer, 10MB max

	for scanner.Scan() {
		line := scanner.Bytes()
		lineStr := string(line)

		// 直接透传所有行
		writer.Write(line)
		writer.Write([]byte("\n"))

		// 解析 usage 信息
		if strings.HasPrefix(lineStr, "data: ") {
			dataStr := strings.TrimPrefix(lineStr, "data: ")
			a.parseStreamUsage(dataStr, result)
		}
	}

	if err := scanner.Err(); err != nil {
		return result, err
	}

	return result, nil
}

// parseStreamUsage 从流式数据中解析 usage 信息
func (a *ClaudeAdapter) parseStreamUsage(data string, result *StreamResult) {
	// Claude 流式响应中，usage 信息在以下事件中：
	// message_start: 包含 input_tokens
	// message_delta: 包含 output_tokens (在流结束时)

	var event struct {
		Type    string `json:"type"`
		Message struct {
			Usage struct {
				InputTokens              int `json:"input_tokens"`
				OutputTokens             int `json:"output_tokens"`
				CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
				CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			} `json:"usage"`
		} `json:"message"`
		Usage struct {
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return
	}

	switch event.Type {
	case "message_start":
		// message_start 事件包含 input_tokens
		result.InputTokens = event.Message.Usage.InputTokens
		result.CacheCreationInputTokens = event.Message.Usage.CacheCreationInputTokens
		result.CacheReadInputTokens = event.Message.Usage.CacheReadInputTokens
	case "message_delta":
		// message_delta 事件在流结束时包含 output_tokens
		result.OutputTokens = event.Usage.OutputTokens
	}
}

// setHeaders 设置请求头 - 透传 + 认证覆盖
func (a *ClaudeAdapter) setHeaders(httpReq *http.Request, account *model.Account, clientHeaders map[string]string) {
	// 1. 先透传客户端 headers（过滤敏感头）
	sensitiveHeaders := map[string]bool{
		"authorization":       true,
		"x-api-key":           true,
		"cookie":              true,
		"host":                true,
		"content-length":      true,
		"connection":          true,
		"proxy-authorization": true,
		"accept-encoding":     true, // 过滤掉以避免 gzip 响应解析问题
	}

	for key, value := range clientHeaders {
		lowerKey := strings.ToLower(key)
		if !sensitiveHeaders[lowerKey] {
			httpReq.Header.Set(key, value)
		}
	}

	// 2. 确保基本头存在
	if httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	if httpReq.Header.Get("anthropic-version") == "" {
		httpReq.Header.Set("anthropic-version", "2023-06-01")
	}

	// 3. 设置认证头（强制覆盖）
	switch account.Type {
	case model.AccountTypeClaudeOfficial:
		// OAuth 模式
		if account.AccessToken != "" {
			httpReq.Header.Set("Authorization", "Bearer "+account.AccessToken)
			// OAuth 需要添加 beta feature
			a.addOAuthBeta(httpReq)
		} else if account.SessionKey != "" {
			httpReq.Header.Set("Cookie", "sessionKey="+account.SessionKey)
		}
	case model.AccountTypeClaudeConsole:
		// API Key 模式
		if account.APIKey != "" {
			httpReq.Header.Set("x-api-key", account.APIKey)
		}
	default:
		if account.APIKey != "" {
			httpReq.Header.Set("x-api-key", account.APIKey)
		}
	}
}

// addOAuthBeta 为 OAuth 添加必需的 beta feature
func (a *ClaudeAdapter) addOAuthBeta(httpReq *http.Request) {
	existingBeta := httpReq.Header.Get("anthropic-beta")
	oauthBeta := "oauth-2025-04-20"

	if strings.Contains(existingBeta, oauthBeta) {
		return
	}

	if existingBeta != "" {
		httpReq.Header.Set("anthropic-beta", existingBeta+","+oauthBeta)
	} else {
		httpReq.Header.Set("anthropic-beta", oauthBeta)
	}
}

// parseResponse 解析响应提取 usage
func (a *ClaudeAdapter) parseResponse(respBody []byte) (*Response, error) {
	var resp struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Model   string `json:"model"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if resp.Error != nil {
		return &Response{
			Error: &Error{
				Type:    resp.Error.Type,
				Message: resp.Error.Message,
			},
		}, nil
	}

	content := ""
	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &Response{
		ID:           resp.ID,
		Model:        resp.Model,
		Content:      content,
		StopReason:   resp.StopReason,
		InputTokens:  resp.Usage.InputTokens,
		OutputTokens: resp.Usage.OutputTokens,
	}, nil
}

// truncateBody 截断响应体用于日志
func truncateBody(body string, maxLen int) string {
	if len(body) <= maxLen {
		return body
	}
	return body[:maxLen] + "..."
}

// maskKey 掩盖 API key 用于日志
func maskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// extractRateLimitHeaders 提取 Claude 限流相关响应头
func extractRateLimitHeaders(header http.Header) map[string]string {
	headers := make(map[string]string)

	// Claude 限流头（不区分大小写）
	rateLimitHeaders := []string{
		"anthropic-ratelimit-unified-5h-status", // 5小时窗口状态: allowed/allowed_warning/rejected
		"anthropic-ratelimit-unified-reset",     // 限流重置时间戳 (Unix seconds)
	}

	for _, h := range rateLimitHeaders {
		value := header.Get(h)
		if value != "" {
			headers[h] = value
		}
	}

	return headers
}
