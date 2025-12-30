package adapter

import (
	"encoding/json"
	"fmt"
)

// FormatConverter 格式转换器
type FormatConverter struct{}

// OpenAI 格式定义
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	TopP        float64         `json:"top_p,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int    `json:"index"`
		Message      OpenAIMessage `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Claude 格式定义
type ClaudeRequest struct {
	Model         string          `json:"model"`
	MaxTokens     int             `json:"max_tokens"`
	System        string          `json:"system,omitempty"`
	Messages      []ClaudeMessage `json:"messages"`
	Temperature   float64         `json:"temperature,omitempty"`
	TopP          float64         `json:"top_p,omitempty"`
	Stream        bool            `json:"stream,omitempty"`
	StopSequences []string        `json:"stop_sequences,omitempty"`
}

type ClaudeMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []ContentBlock
}

type ClaudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ClaudeResponse struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Role         string `json:"role"`
	Content      []ClaudeContentBlock `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Gemini 格式定义
type GeminiRequest struct {
	Contents          []GeminiContent         `json:"contents"`
	SystemInstruction *GeminiContent          `json:"systemInstruction,omitempty"`
	GenerationConfig  *GeminiGenerationConfig `json:"generationConfig,omitempty"`
}

type GeminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text,omitempty"`
}

type GeminiGenerationConfig struct {
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	Temperature     float64  `json:"temperature,omitempty"`
	TopP            float64  `json:"topP,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []GeminiPart `json:"parts"`
			Role  string       `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

// ======================== OpenAI -> Other ========================

// OpenAIToClaude 将 OpenAI 请求转换为 Claude 请求
func (c *FormatConverter) OpenAIToClaude(req *OpenAIRequest) *ClaudeRequest {
	messages := make([]ClaudeMessage, 0, len(req.Messages))
	var system string

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			system = msg.Content
			continue
		}
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
		MaxTokens:     maxTokens,
		System:        system,
		Messages:      messages,
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		Stream:        req.Stream,
		StopSequences: req.Stop,
	}
}

// OpenAIToGemini 将 OpenAI 请求转换为 Gemini 请求
func (c *FormatConverter) OpenAIToGemini(req *OpenAIRequest) *GeminiRequest {
	contents := make([]GeminiContent, 0, len(req.Messages))
	var systemInstruction *GeminiContent

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemInstruction = &GeminiContent{
				Parts: []GeminiPart{{Text: msg.Content}},
			}
			continue
		}

		role := msg.Role
		if role == "assistant" {
			role = "model"
		}

		contents = append(contents, GeminiContent{
			Role:  role,
			Parts: []GeminiPart{{Text: msg.Content}},
		})
	}

	geminiReq := &GeminiRequest{
		Contents:          contents,
		SystemInstruction: systemInstruction,
	}

	if req.MaxTokens > 0 || req.Temperature > 0 || req.TopP > 0 || len(req.Stop) > 0 {
		geminiReq.GenerationConfig = &GeminiGenerationConfig{
			MaxOutputTokens: req.MaxTokens,
			Temperature:     req.Temperature,
			TopP:            req.TopP,
			StopSequences:   req.Stop,
		}
	}

	return geminiReq
}

// ======================== Claude -> Other ========================

// ClaudeToOpenAI 将 Claude 请求转换为 OpenAI 请求
func (c *FormatConverter) ClaudeToOpenAI(req *ClaudeRequest) *OpenAIRequest {
	messages := make([]OpenAIMessage, 0, len(req.Messages)+1)

	if req.System != "" {
		messages = append(messages, OpenAIMessage{
			Role:    "system",
			Content: req.System,
		})
	}

	for _, msg := range req.Messages {
		content := c.extractTextContent(msg.Content)
		messages = append(messages, OpenAIMessage{
			Role:    msg.Role,
			Content: content,
		})
	}

	return &OpenAIRequest{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stream:      req.Stream,
		Stop:        req.StopSequences,
	}
}

// ClaudeToGemini 将 Claude 请求转换为 Gemini 请求
func (c *FormatConverter) ClaudeToGemini(req *ClaudeRequest) *GeminiRequest {
	contents := make([]GeminiContent, 0, len(req.Messages))
	var systemInstruction *GeminiContent

	if req.System != "" {
		systemInstruction = &GeminiContent{
			Parts: []GeminiPart{{Text: req.System}},
		}
	}

	for _, msg := range req.Messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}

		content := c.extractTextContent(msg.Content)
		contents = append(contents, GeminiContent{
			Role:  role,
			Parts: []GeminiPart{{Text: content}},
		})
	}

	geminiReq := &GeminiRequest{
		Contents:          contents,
		SystemInstruction: systemInstruction,
	}

	if req.MaxTokens > 0 || req.Temperature > 0 || req.TopP > 0 || len(req.StopSequences) > 0 {
		geminiReq.GenerationConfig = &GeminiGenerationConfig{
			MaxOutputTokens: req.MaxTokens,
			Temperature:     req.Temperature,
			TopP:            req.TopP,
			StopSequences:   req.StopSequences,
		}
	}

	return geminiReq
}

// ======================== Gemini -> Other ========================

// GeminiToOpenAI 将 Gemini 请求转换为 OpenAI 请求
func (c *FormatConverter) GeminiToOpenAI(req *GeminiRequest) *OpenAIRequest {
	messages := make([]OpenAIMessage, 0, len(req.Contents)+1)

	if req.SystemInstruction != nil && len(req.SystemInstruction.Parts) > 0 {
		text := ""
		for _, part := range req.SystemInstruction.Parts {
			text += part.Text
		}
		messages = append(messages, OpenAIMessage{
			Role:    "system",
			Content: text,
		})
	}

	for _, content := range req.Contents {
		role := content.Role
		if role == "model" {
			role = "assistant"
		}

		text := ""
		for _, part := range content.Parts {
			text += part.Text
		}

		messages = append(messages, OpenAIMessage{
			Role:    role,
			Content: text,
		})
	}

	openAIReq := &OpenAIRequest{
		Messages: messages,
	}

	if req.GenerationConfig != nil {
		openAIReq.MaxTokens = req.GenerationConfig.MaxOutputTokens
		openAIReq.Temperature = req.GenerationConfig.Temperature
		openAIReq.TopP = req.GenerationConfig.TopP
		openAIReq.Stop = req.GenerationConfig.StopSequences
	}

	return openAIReq
}

// GeminiToClaude 将 Gemini 请求转换为 Claude 请求
func (c *FormatConverter) GeminiToClaude(req *GeminiRequest) *ClaudeRequest {
	messages := make([]ClaudeMessage, 0, len(req.Contents))
	var system string

	if req.SystemInstruction != nil && len(req.SystemInstruction.Parts) > 0 {
		for _, part := range req.SystemInstruction.Parts {
			system += part.Text
		}
	}

	for _, content := range req.Contents {
		role := content.Role
		if role == "model" {
			role = "assistant"
		}

		text := ""
		for _, part := range content.Parts {
			text += part.Text
		}

		messages = append(messages, ClaudeMessage{
			Role:    role,
			Content: text,
		})
	}

	claudeReq := &ClaudeRequest{
		Messages: messages,
		System:   system,
	}

	if req.GenerationConfig != nil {
		claudeReq.MaxTokens = req.GenerationConfig.MaxOutputTokens
		claudeReq.Temperature = req.GenerationConfig.Temperature
		claudeReq.TopP = req.GenerationConfig.TopP
		claudeReq.StopSequences = req.GenerationConfig.StopSequences
	}

	if claudeReq.MaxTokens == 0 {
		claudeReq.MaxTokens = 4096
	}

	return claudeReq
}

// ======================== Response Conversions ========================

// ClaudeResponseToOpenAI 将 Claude 响应转换为 OpenAI 响应
func (c *FormatConverter) ClaudeResponseToOpenAI(resp *ClaudeResponse) *OpenAIResponse {
	content := ""
	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	finishReason := c.claudeStopReasonToOpenAI(resp.StopReason)

	return &OpenAIResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Model:   resp.Model,
		Choices: []struct {
			Index        int           `json:"index"`
			Message      OpenAIMessage `json:"message"`
			FinishReason string        `json:"finish_reason"`
		}{
			{
				Index: 0,
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: finishReason,
			},
		},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}

// GeminiResponseToOpenAI 将 Gemini 响应转换为 OpenAI 响应
func (c *FormatConverter) GeminiResponseToOpenAI(resp *GeminiResponse) *OpenAIResponse {
	content := ""
	finishReason := ""

	if len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		for _, part := range candidate.Content.Parts {
			content += part.Text
		}
		finishReason = c.geminiStopReasonToOpenAI(candidate.FinishReason)
	}

	return &OpenAIResponse{
		Object: "chat.completion",
		Choices: []struct {
			Index        int           `json:"index"`
			Message      OpenAIMessage `json:"message"`
			FinishReason string        `json:"finish_reason"`
		}{
			{
				Index: 0,
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: content,
				},
				FinishReason: finishReason,
			},
		},
		Usage: struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		}{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		},
	}
}

// OpenAIResponseToClaude 将 OpenAI 响应转换为 Claude 响应
func (c *FormatConverter) OpenAIResponseToClaude(resp *OpenAIResponse) *ClaudeResponse {
	content := ""
	stopReason := ""

	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
		stopReason = c.openAIStopReasonToClaude(resp.Choices[0].FinishReason)
	}

	return &ClaudeResponse{
		ID:   resp.ID,
		Type: "message",
		Role: "assistant",
		Content: []ClaudeContentBlock{
			{
				Type: "text",
				Text: content,
			},
		},
		Model:      resp.Model,
		StopReason: stopReason,
		Usage: struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		}{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}
}

// ======================== Helper Functions ========================

func (c *FormatConverter) extractTextContent(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		text := ""
		for _, block := range v {
			if b, ok := block.(map[string]interface{}); ok {
				if b["type"] == "text" {
					if t, ok := b["text"].(string); ok {
						text += t
					}
				}
			}
		}
		return text
	case []ClaudeContentBlock:
		text := ""
		for _, block := range v {
			if block.Type == "text" {
				text += block.Text
			}
		}
		return text
	default:
		// 尝试 JSON 解析
		if data, err := json.Marshal(content); err == nil {
			var blocks []ClaudeContentBlock
			if json.Unmarshal(data, &blocks) == nil {
				text := ""
				for _, block := range blocks {
					if block.Type == "text" {
						text += block.Text
					}
				}
				return text
			}
		}
		return fmt.Sprintf("%v", content)
	}
}

func (c *FormatConverter) claudeStopReasonToOpenAI(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	default:
		return reason
	}
}

func (c *FormatConverter) geminiStopReasonToOpenAI(reason string) string {
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

func (c *FormatConverter) openAIStopReasonToClaude(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "content_filter":
		return "content_filter"
	default:
		return reason
	}
}

// NewFormatConverter 创建格式转换器
func NewFormatConverter() *FormatConverter {
	return &FormatConverter{}
}
