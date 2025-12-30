/*
 * 文件作用：Google Gemini API 适配器，处理 Gemini 平台的请求转发
 * 负责功能：
 *   - Gemini API 请求转发
 *   - OpenAI 格式到 Gemini 格式转换
 *   - 流式SSE响应处理
 *   - Usage数据解析
 * 重要程度：⭐⭐⭐⭐ 重要（Gemini平台适配器）
 * 依赖模块：model, logger, http_client
 */
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
	"time"

	"go-aiproxy/internal/model"
	"go-aiproxy/pkg/logger"
)

type GeminiAdapter struct{}

func init() {
	Register(&GeminiAdapter{})
}

func (a *GeminiAdapter) Name() string {
	return "gemini"
}

func (a *GeminiAdapter) Platform() string {
	return model.PlatformGemini
}

func (a *GeminiAdapter) SupportedTypes() []string {
	return []string{model.AccountTypeGemini, model.AccountTypeGeminiAPI}
}

// Gemini 请求格式
type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	SystemInstruction *geminiContent        `json:"systemInstruction,omitempty"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text,omitempty"`
}

type geminiGenerationConfig struct {
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	Temperature     float64  `json:"temperature,omitempty"`
	TopP            float64  `json:"topP,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

// Gemini 响应格式
type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

func (a *GeminiAdapter) Send(ctx context.Context, account *model.Account, req *Request) (*Response, error) {
	log := logger.GetLogger("proxy")

	geminiReq := a.convertRequest(req)

	body, err := json.Marshal(geminiReq)
	if err != nil {
		log.Error("Gemini 序列化请求失败: %v", err)
		return nil, err
	}

	url := a.buildURL(account, req.Model, false)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		log.Error("Gemini 创建请求失败: %v", err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// 记录请求日志 (隐藏 API Key)
	safeURL := strings.Split(url, "?")[0]
	log.Debug("Gemini 请求开始 - URL: %s, AccountType: %s, AccountID: %d, Model: %s",
		safeURL, account.Type, account.ID, req.Model)
	log.Debug("Gemini 请求体: %s", truncateBody(string(body), 500))

	client := GetHTTPClient(account)
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Error("Gemini 请求失败 - 网络错误: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ReadResponseBody(resp)
	if err != nil {
		log.Error("Gemini 读取响应失败: %v", err)
		return nil, err
	}

	log.Debug("Gemini 响应状态码: %d", resp.StatusCode)
	log.Debug("Gemini 响应体: %s", truncateBody(string(respBody), 1000))

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		log.Error("Gemini 解析响应失败: %v, 原始响应: %s", err, string(respBody))
		return nil, fmt.Errorf("parse response: %w, body: %s", err, string(respBody))
	}

	if geminiResp.Error != nil {
		log.Error("Gemini API 返回错误 - Code: %d, Status: %s, Message: %s",
			geminiResp.Error.Code, geminiResp.Error.Status, geminiResp.Error.Message)
		return &Response{
			Error: &Error{
				Type:    geminiResp.Error.Status,
				Message: geminiResp.Error.Message,
			},
		}, nil
	}

	content := ""
	stopReason := ""
	if len(geminiResp.Candidates) > 0 {
		candidate := geminiResp.Candidates[0]
		for _, part := range candidate.Content.Parts {
			content += part.Text
		}
		stopReason = candidate.FinishReason
	}

	log.Info("Gemini 请求成功 - Model: %s, InputTokens: %d, OutputTokens: %d",
		req.Model, geminiResp.UsageMetadata.PromptTokenCount, geminiResp.UsageMetadata.CandidatesTokenCount)

	return &Response{
		ID:           "", // Gemini 不返回 ID
		Model:        req.Model,
		Content:      content,
		StopReason:   a.convertStopReason(stopReason),
		InputTokens:  geminiResp.UsageMetadata.PromptTokenCount,
		OutputTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
	}, nil
}

func (a *GeminiAdapter) SendStream(ctx context.Context, account *model.Account, req *Request, writer io.Writer) (*StreamResult, error) {
	log := logger.GetLogger("proxy")

	geminiReq := a.convertRequest(req)

	body, err := json.Marshal(geminiReq)
	if err != nil {
		log.Error("Gemini Stream 序列化请求失败: %v", err)
		return nil, err
	}

	url := a.buildURL(account, req.Model, true)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		log.Error("Gemini Stream 创建请求失败: %v", err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	safeURL := strings.Split(url, "?")[0]
	log.Info("Gemini Stream 请求开始 | URL: %s | AccountID: %d | Model: %s",
		safeURL, account.ID, req.Model)

	// 使用流式 HTTP 客户端（10分钟超时）
	client := GetStreamHTTPClient(account)
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Error("Gemini Stream 请求失败 - 网络错误: %v", err)
		// 发送 SSE 错误事件给客户端
		a.sendSSEError(writer, "upstream_connection_failed", err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := ReadResponseBody(resp)
		log.Error("Gemini Stream API 错误 - StatusCode: %d, Body: %s", resp.StatusCode, string(respBody))
		// 发送 SSE 错误事件给客户端
		a.sendSSEError(writer, fmt.Sprintf("upstream_error_%d", resp.StatusCode), string(respBody))
		return nil, NewUpstreamError(resp.StatusCode, string(respBody))
	}

	log.Info("Gemini Stream 开始传输 | StatusCode: %d | AccountID: %d", resp.StatusCode, account.ID)

	result := &StreamResult{}

	// 获取 Flusher 接口用于及时刷新数据
	flusher, hasFlusher := writer.(http.Flusher)

	// 监控 context 取消（客户端断开）
	streamDone := make(chan struct{})
	defer close(streamDone)

	go func() {
		select {
		case <-ctx.Done():
			// 客户端断开或超时，关闭上游连接
			log.Info("Gemini Stream 客户端断开或超时，关闭上游连接")
			resp.Body.Close()
		case <-streamDone:
			// 正常完成
		}
	}()

	// SSE 心跳机制：防止长时间无数据导致连接被中间件关闭
	const heartbeatInterval = 15 * time.Second
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	defer heartbeatTicker.Stop()

	dataReceived := make(chan struct{}, 1)

	go func() {
		for {
			select {
			case <-heartbeatTicker.C:
				select {
				case <-dataReceived:
				default:
					if _, err := writer.Write([]byte(": keepalive\n\n")); err == nil {
						if hasFlusher {
							flusher.Flush()
						}
						log.Info("Gemini Stream 发送心跳保活 | AccountID: %d", account.ID)
					} else {
						log.Warn("Gemini Stream 心跳发送失败: %v | AccountID: %d", err, account.ID)
					}
				}
			case <-streamDone:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// Gemini 流式响应格式不同，需要转换为 OpenAI 格式
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		// 通知心跳 goroutine 有数据
		select {
		case dataReceived <- struct{}{}:
		default:
		}

		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			log.Info("Gemini Stream context 已取消，停止转发")
			return result, ctx.Err()
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "" {
			continue
		}

		var chunk geminiResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		// 解析 usage
		if chunk.UsageMetadata.PromptTokenCount > 0 {
			result.InputTokens = chunk.UsageMetadata.PromptTokenCount
		}
		if chunk.UsageMetadata.CandidatesTokenCount > 0 {
			result.OutputTokens = chunk.UsageMetadata.CandidatesTokenCount
		}

		// 转换为 OpenAI 流式格式
		if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
			openAIChunk := map[string]interface{}{
				"id":      "chatcmpl-gemini",
				"object":  "chat.completion.chunk",
				"model":   req.Model,
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"delta": map[string]interface{}{
							"content": chunk.Candidates[0].Content.Parts[0].Text,
						},
						"finish_reason": nil,
					},
				},
			}

			if chunk.Candidates[0].FinishReason != "" {
				openAIChunk["choices"].([]map[string]interface{})[0]["finish_reason"] = a.convertStopReason(chunk.Candidates[0].FinishReason)
			}

			chunkData, _ := json.Marshal(openAIChunk)
			_, writeErr := writer.Write([]byte("data: " + string(chunkData) + "\n\n"))
			if writeErr != nil {
				log.Warn("Gemini Stream 写入客户端失败: %v", writeErr)
				return result, writeErr
			}

			// 立即刷新，确保客户端及时收到数据
			if hasFlusher {
				flusher.Flush()
			}
		}
	}

	if err := scanner.Err(); err != nil {
		// 检查是否是因为 context 取消导致的错误
		if ctx.Err() != nil {
			log.Info("Gemini Stream 因 context 取消而结束: %v", ctx.Err())
			return result, ctx.Err()
		}
		log.Error("Gemini Stream 读取上游错误: %v", err)
		a.sendSSEError(writer, "stream_read_error", err.Error())
		return result, err
	}

	log.Info("Gemini Stream 传输完成 | Model: %s | AccountID: %d | InputTokens: %d | OutputTokens: %d",
		req.Model, account.ID, result.InputTokens, result.OutputTokens)
	return result, nil
}

// sendSSEError 发送 SSE 错误事件给客户端
func (a *GeminiAdapter) sendSSEError(writer io.Writer, errorType, message string) {
	flusher, hasFlusher := writer.(http.Flusher)

	// 发送 SSE 格式的错误事件
	errorEvent := fmt.Sprintf("event: error\ndata: {\"type\":\"error\",\"error\":{\"type\":\"%s\",\"message\":\"%s\"}}\n\n",
		errorType, strings.ReplaceAll(message, "\"", "\\\""))

	writer.Write([]byte(errorEvent))

	if hasFlusher {
		flusher.Flush()
	}
}

func (a *GeminiAdapter) buildURL(account *model.Account, modelName string, stream bool) string {
	baseURL := "https://generativelanguage.googleapis.com/v1beta"
	if account.BaseURL != "" {
		baseURL = account.BaseURL
	}

	// 规范化模型名
	if !strings.HasPrefix(modelName, "models/") {
		modelName = "models/" + modelName
	}

	action := "generateContent"
	if stream {
		action = "streamGenerateContent"
	}

	url := fmt.Sprintf("%s/%s:%s", baseURL, modelName, action)

	// API Key 认证
	if account.APIKey != "" {
		url += "?key=" + account.APIKey
	}

	return url
}

func (a *GeminiAdapter) convertRequest(req *Request) *geminiRequest {
	contents := make([]geminiContent, 0, len(req.Messages))

	for _, msg := range req.Messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}

		text := ""
		switch v := msg.Content.(type) {
		case string:
			text = v
		case []interface{}:
			for _, block := range v {
				if b, ok := block.(map[string]interface{}); ok {
					if b["type"] == "text" {
						text += b["text"].(string)
					}
				}
			}
		}

		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: text}},
		})
	}

	geminiReq := &geminiRequest{
		Contents: contents,
	}

	if req.System != "" {
		geminiReq.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.System}},
		}
	}

	if req.MaxTokens > 0 || req.Temperature > 0 || req.TopP > 0 || len(req.Stop) > 0 {
		geminiReq.GenerationConfig = &geminiGenerationConfig{
			MaxOutputTokens: req.MaxTokens,
			Temperature:     req.Temperature,
			TopP:            req.TopP,
			StopSequences:   req.Stop,
		}
	}

	return geminiReq
}

func (a *GeminiAdapter) convertStopReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY":
		return "content_filter"
	default:
		return reason
	}
}
