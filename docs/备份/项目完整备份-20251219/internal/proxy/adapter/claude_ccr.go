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
)

// ClaudeCCRAdapter CCR 认证适配器
type ClaudeCCRAdapter struct{}

func init() {
	Register(&ClaudeCCRAdapter{})
}

func (a *ClaudeCCRAdapter) Name() string {
	return "claude-ccr"
}

func (a *ClaudeCCRAdapter) Platform() string {
	return model.PlatformClaude
}

func (a *ClaudeCCRAdapter) SupportedTypes() []string {
	return []string{model.AccountTypeCCR}
}

func (a *ClaudeCCRAdapter) Send(ctx context.Context, account *model.Account, req *Request) (*Response, error) {
	claudeReq := a.convertRequest(req)
	claudeReq.Stream = false

	body, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, err
	}

	// CCR 使用自定义 BaseURL
	baseURL := account.BaseURL
	if baseURL == "" {
		return nil, fmt.Errorf("CCR account requires base_url")
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	a.setHeaders(httpReq, account)

	client := GetHTTPClient(account)
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := ReadResponseBody(resp)
	if err != nil {
		return nil, err
	}

	var claudeResp struct {
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
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		return nil, fmt.Errorf("parse response: %w, body: %s", err, string(respBody))
	}

	if claudeResp.Error != nil {
		return &Response{
			Error: &Error{
				Type:    claudeResp.Error.Type,
				Message: claudeResp.Error.Message,
			},
		}, nil
	}

	content := ""
	for _, block := range claudeResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &Response{
		ID:           claudeResp.ID,
		Model:        claudeResp.Model,
		Content:      content,
		StopReason:   claudeResp.StopReason,
		InputTokens:  claudeResp.Usage.InputTokens,
		OutputTokens: claudeResp.Usage.OutputTokens,
	}, nil
}

func (a *ClaudeCCRAdapter) SendStream(ctx context.Context, account *model.Account, req *Request, writer io.Writer) (*StreamResult, error) {
	claudeReq := a.convertRequest(req)
	claudeReq.Stream = true

	body, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, err
	}

	baseURL := account.BaseURL
	if baseURL == "" {
		return nil, fmt.Errorf("CCR account requires base_url")
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	a.setHeaders(httpReq, account)
	httpReq.Header.Set("Accept", "text/event-stream")

	client := GetHTTPClient(account)
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := ReadResponseBody(resp)
		return nil, fmt.Errorf("CCR API error: %s", string(respBody))
	}

	result := &StreamResult{}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event:") || strings.HasPrefix(line, "data:") {
			writer.Write([]byte(line + "\n"))
			// 解析 usage
			if strings.HasPrefix(line, "data: ") {
				a.parseStreamUsage(strings.TrimPrefix(line, "data: "), result)
			}
		} else if line == "" {
			writer.Write([]byte("\n"))
		}
	}

	return result, scanner.Err()
}

// parseStreamUsage 从流式数据中解析 usage 信息
func (a *ClaudeCCRAdapter) parseStreamUsage(data string, result *StreamResult) {
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
		result.InputTokens = event.Message.Usage.InputTokens
		result.CacheCreationInputTokens = event.Message.Usage.CacheCreationInputTokens
		result.CacheReadInputTokens = event.Message.Usage.CacheReadInputTokens
	case "message_delta":
		result.OutputTokens = event.Usage.OutputTokens
	}
}

func (a *ClaudeCCRAdapter) setHeaders(req *http.Request, account *model.Account) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	// CCR 使用 API Key 认证
	if account.APIKey != "" {
		req.Header.Set("x-api-key", account.APIKey)
	}

	// 支持额外的认证头
	if account.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+account.AccessToken)
	}
}

func (a *ClaudeCCRAdapter) convertRequest(req *Request) *ClaudeRequest {
	messages := make([]ClaudeMessage, 0, len(req.Messages))

	for _, msg := range req.Messages {
		messages = append(messages, ClaudeMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	return &ClaudeRequest{
		Model:         req.Model,
		Messages:      messages,
		MaxTokens:     maxTokens,
		System:        req.System,
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		Stream:        req.Stream,
		StopSequences: req.Stop,
	}
}
