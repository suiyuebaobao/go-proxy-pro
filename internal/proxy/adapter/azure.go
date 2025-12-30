/*
 * 文件作用：Azure OpenAI API 适配器，处理 Azure 平台的请求转发
 * 负责功能：
 *   - Azure OpenAI API 请求转发
 *   - Azure 特有的 API 版本和部署名处理
 *   - 流式SSE响应处理
 *   - Usage数据解析
 * 重要程度：⭐⭐⭐⭐ 重要（Azure平台适配器）
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

type AzureOpenAIAdapter struct{}

func init() {
	Register(&AzureOpenAIAdapter{})
}

func (a *AzureOpenAIAdapter) Name() string {
	return "azure-openai"
}

func (a *AzureOpenAIAdapter) Platform() string {
	return model.PlatformOpenAI
}

func (a *AzureOpenAIAdapter) SupportedTypes() []string {
	return []string{model.AccountTypeAzureOpenAI}
}

func (a *AzureOpenAIAdapter) Send(ctx context.Context, account *model.Account, req *Request) (*Response, error) {
	log := logger.GetLogger("proxy")

	openAIReq := convertToOpenAIRequest(req)
	openAIReq.Stream = false

	body, err := json.Marshal(openAIReq)
	if err != nil {
		log.Error("Azure OpenAI 序列化请求失败: %v", err)
		return nil, err
	}

	url := a.buildURL(account)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		log.Error("Azure OpenAI 创建请求失败: %v", err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", account.APIKey)

	log.Debug("Azure OpenAI 请求开始 - URL: %s, AccountType: %s, AccountID: %d, Deployment: %s",
		url, account.Type, account.ID, account.AzureDeploymentName)
	log.Debug("Azure OpenAI 请求头 - api-key: %s...", maskKey(account.APIKey))
	log.Debug("Azure OpenAI 请求体: %s", truncateBody(string(body), 500))

	client := GetHTTPClient(account)
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Error("Azure OpenAI 请求失败 - 网络错误: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ReadResponseBody(resp)
	if err != nil {
		log.Error("Azure OpenAI 读取响应失败: %v", err)
		return nil, err
	}

	log.Debug("Azure OpenAI 响应状态码: %d", resp.StatusCode)
	log.Debug("Azure OpenAI 响应体: %s", truncateBody(string(respBody), 1000))

	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		log.Error("Azure OpenAI 解析响应失败: %v, 原始响应: %s", err, string(respBody))
		return nil, fmt.Errorf("parse response: %w, body: %s", err, string(respBody))
	}

	if openAIResp.Error != nil {
		log.Error("Azure OpenAI API 返回错误 - Type: %s, Message: %s",
			openAIResp.Error.Type, openAIResp.Error.Message)
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

	log.Info("Azure OpenAI 请求成功 - Model: %s, InputTokens: %d, OutputTokens: %d",
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

func (a *AzureOpenAIAdapter) SendStream(ctx context.Context, account *model.Account, req *Request, writer io.Writer) (*StreamResult, error) {
	log := logger.GetLogger("proxy")

	openAIReq := convertToOpenAIRequest(req)
	openAIReq.Stream = true

	body, err := json.Marshal(openAIReq)
	if err != nil {
		log.Error("Azure OpenAI Stream 序列化请求失败: %v", err)
		return nil, err
	}

	url := a.buildURL(account)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		log.Error("Azure OpenAI Stream 创建请求失败: %v", err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", account.APIKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	log.Debug("Azure OpenAI Stream 请求开始 - URL: %s, AccountID: %d, Deployment: %s",
		url, account.ID, account.AzureDeploymentName)

	client := GetHTTPClient(account)
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Error("Azure OpenAI Stream 请求失败 - 网络错误: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := ReadResponseBody(resp)
		log.Error("Azure OpenAI Stream API 错误 - StatusCode: %d, Body: %s", resp.StatusCode, string(respBody))
		return nil, NewUpstreamError(resp.StatusCode, string(respBody))
	}

	log.Debug("Azure OpenAI Stream 响应状态码: %d, 开始接收流式数据", resp.StatusCode)

	result := &StreamResult{}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			writer.Write([]byte(line + "\n\n"))
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				log.Debug("Azure OpenAI Stream 接收完成")
				break
			}
			// 尝试解析 usage
			var chunk struct {
				Usage struct {
					PromptTokens     int `json:"prompt_tokens"`
					CompletionTokens int `json:"completion_tokens"`
				} `json:"usage"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				if chunk.Usage.PromptTokens > 0 {
					result.InputTokens = chunk.Usage.PromptTokens
				}
				if chunk.Usage.CompletionTokens > 0 {
					result.OutputTokens = chunk.Usage.CompletionTokens
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Error("Azure OpenAI Stream 读取错误: %v", err)
		return result, err
	}

	log.Info("Azure OpenAI Stream 请求完成 - Deployment: %s", account.AzureDeploymentName)
	return result, nil
}

func (a *AzureOpenAIAdapter) buildURL(account *model.Account) string {
	endpoint := account.AzureEndpoint
	deployment := account.AzureDeploymentName
	apiVersion := account.AzureAPIVersion

	if apiVersion == "" {
		apiVersion = "2024-02-01"
	}

	return fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		strings.TrimSuffix(endpoint, "/"), deployment, apiVersion)
}

// 共用的请求转换函数
func convertToOpenAIRequest(req *Request) *openAIRequest {
	messages := make([]openAIMessage, 0, len(req.Messages)+1)

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
