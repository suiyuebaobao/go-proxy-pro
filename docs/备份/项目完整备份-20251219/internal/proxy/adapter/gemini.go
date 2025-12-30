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
	log.Debug("Gemini Stream 请求开始 - URL: %s, AccountID: %d, Model: %s",
		safeURL, account.ID, req.Model)

	client := GetHTTPClient(account)
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Error("Gemini Stream 请求失败 - 网络错误: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := ReadResponseBody(resp)
		log.Error("Gemini Stream API 错误 - StatusCode: %d, Body: %s", resp.StatusCode, string(respBody))
		return nil, fmt.Errorf("Gemini API error: %s", string(respBody))
	}

	log.Debug("Gemini Stream 响应状态码: %d, 开始接收流式数据", resp.StatusCode)

	result := &StreamResult{}
	// Gemini 流式响应格式不同，需要转换为 OpenAI 格式
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
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
			writer.Write([]byte("data: " + string(chunkData) + "\n\n"))
		}
	}

	if err := scanner.Err(); err != nil {
		log.Error("Gemini Stream 读取错误: %v", err)
		return result, err
	}

	log.Info("Gemini Stream 请求完成 - Model: %s", req.Model)
	return result, nil
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
