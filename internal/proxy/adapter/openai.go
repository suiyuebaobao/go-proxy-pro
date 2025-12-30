/*
 * 文件作用：OpenAI API 适配器，处理 OpenAI 平台的请求转发
 * 负责功能：
 *   - OpenAI Chat Completions API 转发
 *   - 流式SSE响应处理
 *   - Usage数据解析（输入/输出Token）
 *   - 错误响应处理
 * 重要程度：⭐⭐⭐⭐⭐ 核心（OpenAI平台核心适配器）
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

	"go-aiproxy/internal/model"
	"go-aiproxy/pkg/logger"
)

type OpenAIAdapter struct{}

func init() {
	Register(&OpenAIAdapter{})
}

func (a *OpenAIAdapter) Name() string {
	return "openai"
}

func (a *OpenAIAdapter) Platform() string {
	return model.PlatformOpenAI
}

func (a *OpenAIAdapter) SupportedTypes() []string {
	return []string{model.AccountTypeOpenAI, model.AccountTypeOpenAIResponses}
}

// OpenAI 请求格式
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAI 响应格式
type openAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int           `json:"index"`
		Message      openAIMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
		Delta        openAIMessage `json:"delta,omitempty"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func (a *OpenAIAdapter) Send(ctx context.Context, account *model.Account, req *Request) (*Response, error) {
	log := logger.GetLogger("proxy")

	// 构建 OpenAI 请求
	openAIReq := a.convertRequest(req)
	openAIReq.Stream = false

	body, err := json.Marshal(openAIReq)
	if err != nil {
		log.Error("OpenAI 序列化请求失败: %v", err)
		return nil, err
	}

	baseURL := "https://api.openai.com"
	if account.BaseURL != "" {
		baseURL = account.BaseURL
	}

	fullURL := baseURL + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewReader(body))
	if err != nil {
		log.Error("OpenAI 创建请求失败: %v", err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+account.APIKey)

	// 记录请求日志
	log.Debug("OpenAI 请求开始 - URL: %s, AccountType: %s, AccountID: %d, Model: %s",
		fullURL, account.Type, account.ID, req.Model)
	log.Debug("OpenAI 请求头 - Authorization: Bearer %s...", maskKey(account.APIKey))
	log.Debug("OpenAI 请求体: %s", truncateBody(string(body), 500))

	client := GetHTTPClient(account)
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Error("OpenAI 请求失败 - 网络错误: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ReadResponseBody(resp)
	if err != nil {
		log.Error("OpenAI 读取响应失败: %v", err)
		return nil, err
	}

	// 记录响应日志
	log.Debug("OpenAI 响应状态码: %d", resp.StatusCode)
	log.Debug("OpenAI 响应体: %s", truncateBody(string(respBody), 1000))

	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		log.Error("OpenAI 解析响应失败: %v, 原始响应: %s", err, string(respBody))
		return nil, fmt.Errorf("parse response: %w, body: %s", err, string(respBody))
	}

	if openAIResp.Error != nil {
		log.Error("OpenAI API 返回错误 - Type: %s, Message: %s", openAIResp.Error.Type, openAIResp.Error.Message)
		return &Response{
			Error: &Error{
				Type:    openAIResp.Error.Type,
				Message: openAIResp.Error.Message,
			},
		}, nil
	}

	content := ""
	stopReason := ""
	if len(openAIResp.Choices) > 0 {
		content = openAIResp.Choices[0].Message.Content
		stopReason = openAIResp.Choices[0].FinishReason
	}

	log.Info("OpenAI 请求成功 - Model: %s, InputTokens: %d, OutputTokens: %d",
		openAIResp.Model, openAIResp.Usage.PromptTokens, openAIResp.Usage.CompletionTokens)

	return &Response{
		ID:           openAIResp.ID,
		Model:        openAIResp.Model,
		Content:      content,
		StopReason:   stopReason,
		InputTokens:  openAIResp.Usage.PromptTokens,
		OutputTokens: openAIResp.Usage.CompletionTokens,
	}, nil
}

func (a *OpenAIAdapter) SendStream(ctx context.Context, account *model.Account, req *Request, writer io.Writer) (*StreamResult, error) {
	log := logger.GetLogger("proxy")

	// 构建 OpenAI 请求
	openAIReq := a.convertRequest(req)
	openAIReq.Stream = true

	body, err := json.Marshal(openAIReq)
	if err != nil {
		log.Error("OpenAI Stream 序列化请求失败: %v", err)
		return nil, err
	}

	baseURL := "https://api.openai.com"
	if account.BaseURL != "" {
		baseURL = account.BaseURL
	}

	fullURL := baseURL + "/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewReader(body))
	if err != nil {
		log.Error("OpenAI Stream 创建请求失败: %v", err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+account.APIKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	log.Debug("OpenAI Stream 请求开始 - URL: %s, AccountID: %d, Model: %s",
		fullURL, account.ID, req.Model)

	client := GetHTTPClient(account)
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Error("OpenAI Stream 请求失败 - 网络错误: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := ReadResponseBody(resp)
		log.Error("OpenAI Stream API 错误 - StatusCode: %d, Body: %s", resp.StatusCode, string(respBody))
		return nil, NewUpstreamError(resp.StatusCode, string(respBody))
	}

	log.Debug("OpenAI Stream 响应状态码: %d, 开始接收流式数据", resp.StatusCode)

	result := &StreamResult{}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			log.Debug("OpenAI Stream 接收完成")
			break
		}

		var chunk openAIResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		// 解析 usage（OpenAI 流式响应最后一个 chunk 可能包含 usage）
		if chunk.Usage.PromptTokens > 0 {
			result.InputTokens = chunk.Usage.PromptTokens
		}
		if chunk.Usage.CompletionTokens > 0 {
			result.OutputTokens = chunk.Usage.CompletionTokens
		}

		// 直接转发 OpenAI 格式
		writer.Write([]byte(line + "\n\n"))
	}

	if err := scanner.Err(); err != nil {
		log.Error("OpenAI Stream 读取错误: %v", err)
		return result, err
	}

	log.Info("OpenAI Stream 请求完成 - Model: %s", req.Model)
	return result, nil
}

func (a *OpenAIAdapter) convertRequest(req *Request) *openAIRequest {
	messages := make([]openAIMessage, 0, len(req.Messages))

	// 如果有 system prompt，添加到消息开头
	if req.System != "" {
		messages = append(messages, openAIMessage{
			Role:    "system",
			Content: req.System,
		})
	}

	for _, msg := range req.Messages {
		content := ""
		switch v := msg.Content.(type) {
		case string:
			content = v
		case []interface{}:
			// 处理多模态内容
			for _, block := range v {
				if b, ok := block.(map[string]interface{}); ok {
					if b["type"] == "text" {
						content += b["text"].(string)
					}
				}
			}
		}
		messages = append(messages, openAIMessage{
			Role:    msg.Role,
			Content: content,
		})
	}

	return &openAIRequest{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
		Stop:        req.Stop,
	}
}
