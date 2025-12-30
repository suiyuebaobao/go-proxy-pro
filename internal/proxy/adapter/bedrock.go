/*
 * 文件作用：AWS Bedrock API 适配器，处理 AWS Bedrock 平台的请求转发
 * 负责功能：
 *   - AWS Bedrock API 请求转发
 *   - AWS Signature V4 签名认证
 *   - Claude on Bedrock 格式转换
 *   - 流式响应处理
 * 重要程度：⭐⭐⭐⭐ 重要（Bedrock平台适配器）
 * 依赖模块：model, logger, http_client
 */
package adapter

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go-aiproxy/internal/model"
	"go-aiproxy/pkg/logger"
)

type BedrockAdapter struct{}

func init() {
	Register(&BedrockAdapter{})
}

func (a *BedrockAdapter) Name() string {
	return "bedrock"
}

func (a *BedrockAdapter) Platform() string {
	return model.PlatformClaude
}

func (a *BedrockAdapter) SupportedTypes() []string {
	return []string{model.AccountTypeBedrock}
}

// Bedrock Claude 请求格式
type bedrockRequest struct {
	AnthropicVersion string           `json:"anthropic_version"`
	MaxTokens        int              `json:"max_tokens"`
	System           string           `json:"system,omitempty"`
	Messages         []bedrockMessage `json:"messages"`
	Temperature      float64          `json:"temperature,omitempty"`
	TopP             float64          `json:"top_p,omitempty"`
	StopSequences    []string         `json:"stop_sequences,omitempty"`
}

type bedrockMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// Bedrock Claude 响应格式
type bedrockResponse struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Role         string `json:"role"`
	Content      []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence,omitempty"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Bedrock 流式响应事件
type bedrockStreamEvent struct {
	Type         string `json:"type"`
	Index        int    `json:"index,omitempty"`
	ContentBlock *struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content_block,omitempty"`
	Delta *struct {
		Type       string `json:"type"`
		Text       string `json:"text,omitempty"`
		StopReason string `json:"stop_reason,omitempty"`
	} `json:"delta,omitempty"`
	Message *bedrockResponse `json:"message,omitempty"`
	Usage   *struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage,omitempty"`
}

func (a *BedrockAdapter) Send(ctx context.Context, account *model.Account, req *Request) (*Response, error) {
	log := logger.GetLogger("proxy")

	bedrockReq := a.convertRequest(req)

	body, err := json.Marshal(bedrockReq)
	if err != nil {
		log.Error("Bedrock 序列化请求失败: %v", err)
		return nil, err
	}

	url := a.buildURL(account, req.Model, false)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		log.Error("Bedrock 创建请求失败: %v", err)
		return nil, err
	}

	// AWS Signature V4 签名
	a.signRequest(httpReq, body, account)

	log.Debug("Bedrock 请求开始 - URL: %s, AccountType: %s, AccountID: %d, Model: %s, Region: %s",
		url, account.Type, account.ID, req.Model, account.AWSRegion)
	log.Debug("Bedrock 请求体: %s", truncateBody(string(body), 500))

	client := GetHTTPClient(account)
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Error("Bedrock 请求失败 - 网络错误: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ReadResponseBody(resp)
	if err != nil {
		log.Error("Bedrock 读取响应失败: %v", err)
		return nil, err
	}

	log.Debug("Bedrock 响应状态码: %d", resp.StatusCode)
	log.Debug("Bedrock 响应体: %s", truncateBody(string(respBody), 1000))

	var bedrockResp bedrockResponse
	if err := json.Unmarshal(respBody, &bedrockResp); err != nil {
		log.Error("Bedrock 解析响应失败: %v, 原始响应: %s", err, string(respBody))
		return nil, fmt.Errorf("parse response: %w, body: %s", err, string(respBody))
	}

	if bedrockResp.Error != nil {
		log.Error("Bedrock API 返回错误 - Type: %s, Message: %s",
			bedrockResp.Error.Type, bedrockResp.Error.Message)
		return &Response{
			Error: &Error{
				Type:    bedrockResp.Error.Type,
				Message: bedrockResp.Error.Message,
			},
		}, nil
	}

	content := ""
	for _, block := range bedrockResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	log.Info("Bedrock 请求成功 - Model: %s, InputTokens: %d, OutputTokens: %d",
		req.Model, bedrockResp.Usage.InputTokens, bedrockResp.Usage.OutputTokens)

	return &Response{
		ID:           bedrockResp.ID,
		Model:        req.Model,
		Content:      content,
		StopReason:   bedrockResp.StopReason,
		InputTokens:  bedrockResp.Usage.InputTokens,
		OutputTokens: bedrockResp.Usage.OutputTokens,
	}, nil
}

func (a *BedrockAdapter) SendStream(ctx context.Context, account *model.Account, req *Request, writer io.Writer) (*StreamResult, error) {
	log := logger.GetLogger("proxy")

	bedrockReq := a.convertRequest(req)

	body, err := json.Marshal(bedrockReq)
	if err != nil {
		log.Error("Bedrock Stream 序列化请求失败: %v", err)
		return nil, err
	}

	url := a.buildURL(account, req.Model, true)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		log.Error("Bedrock Stream 创建请求失败: %v", err)
		return nil, err
	}

	// AWS Signature V4 签名
	a.signRequest(httpReq, body, account)
	httpReq.Header.Set("Accept", "application/vnd.amazon.eventstream")

	log.Debug("Bedrock Stream 请求开始 - URL: %s, AccountID: %d, Model: %s, Region: %s",
		url, account.ID, req.Model, account.AWSRegion)

	client := GetHTTPClient(account)
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Error("Bedrock Stream 请求失败 - 网络错误: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := ReadResponseBody(resp)
		log.Error("Bedrock Stream API 错误 - StatusCode: %d, Body: %s", resp.StatusCode, string(respBody))
		return nil, NewUpstreamError(resp.StatusCode, string(respBody))
	}

	log.Debug("Bedrock Stream 响应状态码: %d, 开始接收流式数据", resp.StatusCode)

	result := &StreamResult{}

	// Bedrock 使用 Amazon Event Stream 格式，这里简化处理
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

		var event bedrockStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		// 解析 usage 信息
		switch event.Type {
		case "message_start":
			// Bedrock 在 message_start 中包含 input_tokens
			if event.Message != nil {
				result.InputTokens = event.Message.Usage.InputTokens
			}
		case "message_delta":
			// Bedrock 在 message_delta 中包含 output_tokens
			if event.Usage != nil {
				result.OutputTokens = event.Usage.OutputTokens
			}
		}

		// 转换为 OpenAI 流式格式
		switch event.Type {
		case "content_block_delta":
			if event.Delta != nil && event.Delta.Text != "" {
				openAIChunk := map[string]interface{}{
					"id":      "chatcmpl-bedrock",
					"object":  "chat.completion.chunk",
					"model":   req.Model,
					"choices": []map[string]interface{}{
						{
							"index": 0,
							"delta": map[string]interface{}{
								"content": event.Delta.Text,
							},
							"finish_reason": nil,
						},
					},
				}
				chunkData, _ := json.Marshal(openAIChunk)
				writer.Write([]byte("data: " + string(chunkData) + "\n\n"))
			}
		case "message_delta":
			if event.Delta != nil && event.Delta.StopReason != "" {
				openAIChunk := map[string]interface{}{
					"id":      "chatcmpl-bedrock",
					"object":  "chat.completion.chunk",
					"model":   req.Model,
					"choices": []map[string]interface{}{
						{
							"index":         0,
							"delta":         map[string]interface{}{},
							"finish_reason": event.Delta.StopReason,
						},
					},
				}
				chunkData, _ := json.Marshal(openAIChunk)
				writer.Write([]byte("data: " + string(chunkData) + "\n\n"))
			}
		}
	}

	writer.Write([]byte("data: [DONE]\n\n"))

	if err := scanner.Err(); err != nil {
		log.Error("Bedrock Stream 读取错误: %v", err)
		return result, err
	}

	log.Info("Bedrock Stream 请求完成 - Model: %s, InputTokens: %d, OutputTokens: %d",
		req.Model, result.InputTokens, result.OutputTokens)
	return result, nil
}

func (a *BedrockAdapter) buildURL(account *model.Account, modelName string, stream bool) string {
	region := account.AWSRegion
	if region == "" {
		region = "us-east-1"
	}

	// 转换模型名为 Bedrock 格式
	modelID := a.convertModelID(modelName)

	action := "invoke"
	if stream {
		action = "invoke-with-response-stream"
	}

	return fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/%s",
		region, modelID, action)
}

func (a *BedrockAdapter) convertModelID(modelName string) string {
	// 将通用模型名转换为 Bedrock 模型 ID
	modelMap := map[string]string{
		"claude-3-5-sonnet":          "anthropic.claude-3-5-sonnet-20241022-v2:0",
		"claude-3-5-sonnet-20241022": "anthropic.claude-3-5-sonnet-20241022-v2:0",
		"claude-3-5-haiku":           "anthropic.claude-3-5-haiku-20241022-v1:0",
		"claude-3-opus":              "anthropic.claude-3-opus-20240229-v1:0",
		"claude-3-sonnet":            "anthropic.claude-3-sonnet-20240229-v1:0",
		"claude-3-haiku":             "anthropic.claude-3-haiku-20240307-v1:0",
		"claude-2.1":                 "anthropic.claude-v2:1",
		"claude-2":                   "anthropic.claude-v2",
		"claude-instant":             "anthropic.claude-instant-v1",
	}

	if id, ok := modelMap[modelName]; ok {
		return id
	}

	// 如果已经是 Bedrock 格式，直接返回
	if strings.HasPrefix(modelName, "anthropic.") {
		return modelName
	}

	// 默认使用 Claude 3.5 Sonnet
	return "anthropic.claude-3-5-sonnet-20241022-v2:0"
}

func (a *BedrockAdapter) convertRequest(req *Request) *bedrockRequest {
	messages := make([]bedrockMessage, 0, len(req.Messages))

	for _, msg := range req.Messages {
		messages = append(messages, bedrockMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	return &bedrockRequest{
		AnthropicVersion: "bedrock-2023-05-31",
		MaxTokens:        maxTokens,
		System:           req.System,
		Messages:         messages,
		Temperature:      req.Temperature,
		TopP:             req.TopP,
		StopSequences:    req.Stop,
	}
}

// AWS Signature V4 签名
func (a *BedrockAdapter) signRequest(req *http.Request, body []byte, account *model.Account) {
	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")

	region := account.AWSRegion
	if region == "" {
		region = "us-east-1"
	}
	service := "bedrock"

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("Host", req.URL.Host)

	// 如果有 session token
	if account.AWSSessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", account.AWSSessionToken)
	}

	// 计算 payload hash
	payloadHash := sha256Hash(body)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)

	// 创建规范请求
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n",
		req.Header.Get("Content-Type"),
		req.URL.Host,
		payloadHash,
		amzDate)

	signedHeaders := "content-type;host;x-amz-content-sha256;x-amz-date"
	if account.AWSSessionToken != "" {
		canonicalHeaders = fmt.Sprintf("content-type:%s\nhost:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\nx-amz-security-token:%s\n",
			req.Header.Get("Content-Type"),
			req.URL.Host,
			payloadHash,
			amzDate,
			account.AWSSessionToken)
		signedHeaders = "content-type;host;x-amz-content-sha256;x-amz-date;x-amz-security-token"
	}

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		req.Method,
		req.URL.Path,
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash)

	// 创建签名字符串
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		amzDate,
		credentialScope,
		sha256Hash([]byte(canonicalRequest)))

	// 计算签名
	signingKey := getSignatureKey(account.AWSSecretKey, dateStamp, region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	// 添加授权头
	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		account.AWSAccessKey, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)
}

func sha256Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func getSignatureKey(secretKey, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}
