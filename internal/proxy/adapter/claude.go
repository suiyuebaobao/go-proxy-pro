/*
 * 文件作用：Claude API 适配器，处理 Anthropic Claude 平台的请求转发
 * 负责功能：
 *   - Claude Official API 请求转发
 *   - Claude OAuth/SessionKey 认证
 *   - 流式SSE响应处理和Usage解析
 *   - Thinking Block Signature 错误自动重试
 *   - 限流响应头提取（5H/7D利用率）
 *   - 账户 ModelMapping 模型转换
 * 重要程度：⭐⭐⭐⭐⭐ 核心（Claude平台核心适配器）
 * 依赖模块：model, logger, http_client
 */
package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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

	// 调试：记录请求体长度和前 500 字符
	log.Debug("Claude 请求体 | 长度: %d | 前500字符: %s", len(body), truncateBody(string(body), 500))

	// 执行请求（支持 signature 错误自动重试）
	return a.doSendWithRetry(ctx, account, req, body, false)
}

// doSendWithRetry 执行非流式请求，支持 signature 错误自动重试
func (a *ClaudeAdapter) doSendWithRetry(ctx context.Context, account *model.Account, req *Request, body []byte, isRetry bool) (*Response, error) {
	log := logger.GetLogger("proxy")

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
	log.Debug("Claude 请求 | URL: %s | AccountID: %d | Model: %s | isRetry: %v", fullURL, account.ID, req.Model, isRetry)
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

	// 非200状态码，检查是否是 signature 错误
	if resp.StatusCode != http.StatusOK {
		errStr := string(respBody)
		log.Error("Claude API 错误 | StatusCode: %d | Body: %s", resp.StatusCode, truncateBody(errStr, 500))

		// 检测 signature 错误，自动移除 thinking block 并重试
		if !isRetry && isSignatureError(errStr) {
			log.Warn("检测到 thinking block signature 错误，尝试移除 thinking block 并重试")
			newBody, removed := removeThinkingBlocks(body)
			if removed {
				log.Info("已移除 thinking block，重试请求 | 原长度: %d | 新长度: %d", len(body), len(newBody))
				return a.doSendWithRetry(ctx, account, req, newBody, true)
			}
		}

		return nil, NewUpstreamError(resp.StatusCode, errStr)
	}

	// 解析响应提取 usage 信息
	response, err := a.parseResponse(respBody)
	if err != nil {
		return nil, err
	}

	// 仅对 OAuth/SessionKey 模式提取限流头，API Key 模式不需要
	if account.Type != model.AccountTypeClaudeConsole {
		response.Headers = extractRateLimitHeaders(resp.Header)
	}

	return response, nil
}

// SendStream 发送流式请求 - 透传模式，同时解析 usage
func (a *ClaudeAdapter) SendStream(ctx context.Context, account *model.Account, req *Request, writer io.Writer) (*StreamResult, error) {
	body := req.RawBody
	if len(body) == 0 {
		return nil, fmt.Errorf("empty request body")
	}

	// 执行流式请求（支持 signature 错误自动重试）
	return a.doSendStreamWithRetry(ctx, account, req, body, writer, false)
}

// doSendStreamWithRetry 执行流式请求，支持 signature 错误自动重试
func (a *ClaudeAdapter) doSendStreamWithRetry(ctx context.Context, account *model.Account, req *Request, body []byte, writer io.Writer, isRetry bool) (*StreamResult, error) {
	log := logger.GetLogger("proxy")

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

	log.Info("Claude Stream 请求开始 | URL: %s | AccountID: %d | AccountName: %s | isRetry: %v", fullURL, account.ID, account.Name, isRetry)

	client := GetStreamHTTPClient(account)
	resp, err := client.Do(httpReq)
	if err != nil {
		// 发送 SSE 错误事件给客户端
		a.sendSSEError(writer, "upstream_connection_failed", err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	log.Info("Claude Stream 上游响应 | StatusCode: %d | AccountID: %d", resp.StatusCode, account.ID)

	if resp.StatusCode != http.StatusOK {
		respBody, _ := ReadResponseBody(resp)
		errStr := string(respBody)
		log.Error("Claude Stream 上游错误 | StatusCode: %d | Body: %s | AccountID: %d", resp.StatusCode, errStr, account.ID)

		// 检测 signature 错误，自动移除 thinking block 并重试
		if !isRetry && isSignatureError(errStr) {
			log.Warn("Claude Stream 检测到 thinking block signature 错误，尝试移除 thinking block 并重试")
			newBody, removed := removeThinkingBlocks(body)
			if removed {
				log.Info("已移除 thinking block，重试流式请求 | 原长度: %d | 新长度: %d", len(body), len(newBody))
				return a.doSendStreamWithRetry(ctx, account, req, newBody, writer, true)
			}
		}

		// 发送 SSE 错误事件给客户端
		a.sendSSEError(writer, fmt.Sprintf("upstream_error_%d", resp.StatusCode), errStr)
		return nil, NewUpstreamError(resp.StatusCode, errStr)
	}

	// 透传 SSE 流并解析 usage
	result := &StreamResult{}

	// 仅对 OAuth/SessionKey 模式提取限流头，API Key 模式不需要
	if account.Type != model.AccountTypeClaudeConsole {
		result.Headers = extractRateLimitHeaders(resp.Header)
	}

	// 获取 Flusher 接口用于及时刷新数据
	flusher, hasFlusher := writer.(http.Flusher)

	// 监控 context 取消（客户端断开）
	streamDone := make(chan struct{})
	defer close(streamDone)

	go func() {
		select {
		case <-ctx.Done():
			// 客户端断开或超时，关闭上游连接
			log.Info("Claude Stream 客户端断开或超时，关闭上游连接")
			resp.Body.Close()
		case <-streamDone:
			// 正常完成
		}
	}()

	// SSE 心跳机制：防止长时间无数据导致连接被中间件关闭
	// 参考 claude-relay-service 实现
	const heartbeatInterval = 15 * time.Second
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	// 用于通知心跳 goroutine 有新数据
	dataReceived := make(chan struct{}, 1)
	heartbeatDone := make(chan struct{})

	// 心跳发送 goroutine
	go func() {
		defer close(heartbeatDone)
		for {
			select {
			case <-heartbeatTicker.C:
				// 检查是否有最近的数据，如果没有则发送心跳
				select {
				case <-dataReceived:
					// 有数据，重置等待
				default:
					// 没有数据，发送心跳（空行）
					if _, err := writer.Write([]byte(": keepalive\n\n")); err == nil {
						if hasFlusher {
							flusher.Flush()
						}
						log.Info("Claude Stream 发送心跳保活 | AccountID: %d", account.ID)
					} else {
						log.Warn("Claude Stream 心跳发送失败: %v | AccountID: %d", err, account.ID)
					}
				}
			case <-streamDone:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	log.Info("Claude Stream 开始传输 | AccountID: %d | 心跳间隔: %v", account.ID, heartbeatInterval)

	// 使用 buffer 处理不完整行
	var buffer string
	lineCount := 0
	var debugLines []string // 记录前几行用于调试

	// 首个事件检测：用于在写入客户端前检测 SSE 错误
	// 如果首个事件是错误，返回错误以触发重试（不写入客户端）
	var firstEventChecked bool
	var pendingLines []string // 缓冲首个事件的行（event: 和 data:）

	// 使用较大的读取缓冲区
	readBuf := make([]byte, 32*1024) // 32KB

	for {
		// 通知心跳 goroutine 有数据
		select {
		case dataReceived <- struct{}{}:
		default:
		}

		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			log.Info("Claude Stream context 已取消，停止转发 | 已传输行数: %d", lineCount)
			return result, ctx.Err()
		default:
		}

		n, readErr := resp.Body.Read(readBuf)
		if n > 0 {
			buffer += string(readBuf[:n])

			// 按换行符分割，保留最后的不完整行
			lines := strings.Split(buffer, "\n")
			buffer = lines[len(lines)-1] // 保留最后的不完整行

			// 处理完整的行（除了最后一行）
			for i := 0; i < len(lines)-1; i++ {
				line := lines[i]
				lineCount++

				// 记录前 10 行用于调试
				if lineCount <= 10 {
					debugLines = append(debugLines, line)
				}

				// === 首个事件检测逻辑 ===
				if !firstEventChecked {
					pendingLines = append(pendingLines, line)

					// 检测到 data: 行，检查是否是错误事件
					if strings.HasPrefix(line, "data: ") {
						dataStr := strings.TrimPrefix(line, "data: ")

						// 解析检查是否是错误事件
						var errEvent struct {
							Type  string `json:"type"`
							Error *struct {
								Type    string `json:"type"`
								Message string `json:"message"`
							} `json:"error"`
						}
						if json.Unmarshal([]byte(dataStr), &errEvent) == nil && errEvent.Type == "error" && errEvent.Error != nil {
							errMsg := fmt.Sprintf("%s: %s", errEvent.Error.Type, errEvent.Error.Message)
							log.Error("Claude Stream 首个事件为错误 | Type: %s | Message: %s | AccountID: %d",
								errEvent.Error.Type, errEvent.Error.Message, account.ID)

							// 检测 signature 错误，自动移除 thinking block 并重试
							if !isRetry && isSignatureError(errMsg) {
								log.Warn("Claude Stream SSE 首个事件检测到 signature 错误，尝试移除 thinking block 并重试")
								newBody, removed := removeThinkingBlocks(body)
								if removed {
									log.Info("已移除 thinking block，重试流式请求 | 原长度: %d | 新长度: %d", len(body), len(newBody))
									return a.doSendStreamWithRetry(ctx, account, req, newBody, writer, true)
								}
							}

							return result, NewUpstreamError(500, errMsg)
						}

						// 首个事件不是错误，写入所有缓冲的行
						firstEventChecked = true

						// 【修复】解析第一个 data 事件的 usage 信息（message_start 包含 input_tokens）
						a.parseStreamUsage(dataStr, result)

						for _, pendingLine := range pendingLines {
							writer.Write([]byte(pendingLine + "\n"))
						}
						pendingLines = nil
						if hasFlusher {
							flusher.Flush()
						}
					}
					continue // 继续缓冲直到看到 data: 行
				}

				// === 正常转发逻辑 ===
				// 解析 data 行获取 usage 信息（不阻塞转发）
				if strings.HasPrefix(line, "data: ") {
					dataStr := strings.TrimPrefix(line, "data: ")
					a.parseStreamUsage(dataStr, result)
				}

				// 立即转发到客户端
				_, writeErr := writer.Write([]byte(line + "\n"))
				if writeErr != nil {
					log.Warn("Claude Stream 写入客户端失败: %v | 已传输行数: %d", writeErr, lineCount)
					return result, writeErr
				}
			}

			// 立即刷新，确保客户端及时收到数据
			if hasFlusher && firstEventChecked {
				flusher.Flush()
			}

			// 每 100 行记录一次进度（调试用）
			if lineCount > 0 && lineCount%100 == 0 {
				log.Debug("Claude Stream 进度 | 已传输行数: %d", lineCount)
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				// 处理缓冲区中剩余的数据
				if buffer != "" {
					lineCount++
					if lineCount <= 10 {
						debugLines = append(debugLines, buffer)
					}
					if firstEventChecked {
						writer.Write([]byte(buffer + "\n"))
						if hasFlusher {
							flusher.Flush()
						}
					}
				}
				// 如果还有未写入的缓冲行（首个事件检测期间的）
				if !firstEventChecked && len(pendingLines) > 0 {
					for _, pendingLine := range pendingLines {
						writer.Write([]byte(pendingLine + "\n"))
					}
					if hasFlusher {
						flusher.Flush()
					}
				}
				break
			}
			// 检查是否是因为 context 取消导致的错误
			if ctx.Err() != nil {
				log.Info("Claude Stream 因 context 取消而结束: %v", ctx.Err())
				return result, ctx.Err()
			}
			log.Error("Claude Stream 读取上游错误: %v", readErr)
			a.sendSSEError(writer, "stream_read_error", readErr.Error())
			return result, readErr
		}
	}

	// 如果行数很少（可能是错误响应），记录详细内容用于调试
	if lineCount <= 10 {
		log.Warn("Claude Stream 行数异常少 | 总行数: %d | 内容: %v", lineCount, debugLines)
	}
	log.Info("Claude Stream 传输完成 | 总行数: %d | InputTokens: %d | OutputTokens: %d", lineCount, result.InputTokens, result.OutputTokens)

	return result, nil
}

// sendSSEError 发送 SSE 错误事件给客户端
func (a *ClaudeAdapter) sendSSEError(writer io.Writer, errorType, message string) {
	flusher, hasFlusher := writer.(http.Flusher)

	// 发送 SSE 格式的错误事件
	errorEvent := fmt.Sprintf("event: error\ndata: {\"type\":\"error\",\"error\":{\"type\":\"%s\",\"message\":\"%s\"}}\n\n",
		errorType, strings.ReplaceAll(message, "\"", "\\\""))

	writer.Write([]byte(errorEvent))

	if hasFlusher {
		flusher.Flush()
	}
}

// parseStreamUsage 从流式数据中解析 usage 信息
func (a *ClaudeAdapter) parseStreamUsage(data string, result *StreamResult) {
	// Claude 流式响应中，usage 信息在以下事件中：
	// message_start: 包含 input_tokens（Claude 标准格式）
	// message_delta: 包含 output_tokens (在流结束时)
	//
	// 注意：GLM 等兼容 API 可能在 message_delta 中返回完整的 usage 信息
	// 包括 input_tokens 和 cache_read_input_tokens

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
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return
	}

	switch event.Type {
	case "message_start":
		// message_start 事件包含 input_tokens（Claude 标准格式）
		if event.Message.Usage.InputTokens > 0 {
			result.InputTokens = event.Message.Usage.InputTokens
		}
		if event.Message.Usage.CacheCreationInputTokens > 0 {
			result.CacheCreationInputTokens = event.Message.Usage.CacheCreationInputTokens
		}
		if event.Message.Usage.CacheReadInputTokens > 0 {
			result.CacheReadInputTokens = event.Message.Usage.CacheReadInputTokens
		}
	case "message_delta":
		// message_delta 事件在流结束时包含 usage 信息
		// Claude 标准格式只有 output_tokens
		// GLM 等兼容 API 可能包含完整的 usage 信息
		if event.Usage.OutputTokens > 0 {
			result.OutputTokens = event.Usage.OutputTokens
		}
		// 兼容 GLM：如果 message_delta 中有 input_tokens，使用它
		if event.Usage.InputTokens > 0 && result.InputTokens == 0 {
			result.InputTokens = event.Usage.InputTokens
		}
		// 兼容 GLM：如果 message_delta 中有缓存信息，使用它
		if event.Usage.CacheCreationInputTokens > 0 && result.CacheCreationInputTokens == 0 {
			result.CacheCreationInputTokens = event.Usage.CacheCreationInputTokens
		}
		if event.Usage.CacheReadInputTokens > 0 && result.CacheReadInputTokens == 0 {
			result.CacheReadInputTokens = event.Usage.CacheReadInputTokens
		}
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

// isSignatureError 检测是否是 thinking block signature 错误
// 当请求包含来自不同账户的 thinking block 时，Claude API 会返回此错误
func isSignatureError(errStr string) bool {
	return strings.Contains(errStr, "Invalid") &&
		strings.Contains(errStr, "signature") &&
		strings.Contains(errStr, "thinking")
}

// removeThinkingBlocks 从请求体中移除 thinking blocks
// 返回新的请求体和是否有移除操作
func removeThinkingBlocks(body []byte) ([]byte, bool) {
	log := logger.GetLogger("proxy")

	// 解析 JSON
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		log.Warn("removeThinkingBlocks: 解析请求体失败: %v", err)
		return body, false
	}

	// 获取 messages 数组
	messages, ok := req["messages"].([]interface{})
	if !ok {
		log.Debug("removeThinkingBlocks: 没有 messages 字段")
		return body, false
	}

	removed := false

	// 遍历每个 message
	for i, msg := range messages {
		msgMap, ok := msg.(map[string]interface{})
		if !ok {
			continue
		}

		// 获取 content
		content, ok := msgMap["content"]
		if !ok {
			continue
		}

		// content 可能是字符串或数组
		contentArray, ok := content.([]interface{})
		if !ok {
			// 如果是字符串，跳过
			continue
		}

		// 过滤掉 thinking blocks
		newContent := make([]interface{}, 0, len(contentArray))
		for _, block := range contentArray {
			blockMap, ok := block.(map[string]interface{})
			if !ok {
				newContent = append(newContent, block)
				continue
			}

			blockType, _ := blockMap["type"].(string)
			if blockType == "thinking" {
				// 跳过 thinking block
				removed = true
				log.Debug("removeThinkingBlocks: 移除 thinking block from message %d", i)
				continue
			}

			newContent = append(newContent, block)
		}

		// 更新 content
		if len(newContent) != len(contentArray) {
			msgMap["content"] = newContent
		}
	}

	if !removed {
		log.Debug("removeThinkingBlocks: 没有找到 thinking blocks")
		return body, false
	}

	// 重新序列化
	newBody, err := json.Marshal(req)
	if err != nil {
		log.Warn("removeThinkingBlocks: 序列化失败: %v", err)
		return body, false
	}

	log.Info("removeThinkingBlocks: 成功移除 thinking blocks | 原长度: %d | 新长度: %d", len(body), len(newBody))
	return newBody, true
}
