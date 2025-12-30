package adapter

import (
	"context"
	"errors"
	"io"

	"go-aiproxy/internal/model"
)

var (
	ErrNoAdapter = errors.New("no adapter found for account type")
)

// Request 统一请求结构
type Request struct {
	Model       string        `json:"model"`
	Messages    []Message     `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	TopP        float64       `json:"top_p,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	Stop        []string      `json:"stop,omitempty"`
	System      string        `json:"system,omitempty"`
	Tools       []interface{} `json:"tools,omitempty"`

	// 原始请求体（用于直接转发）
	RawBody []byte `json:"-"`
	// 客户端请求头（用于透传）
	Headers map[string]string `json:"-"`
	// 原始请求路径（用于 Codex 等透传场景）
	Path string `json:"-"`
}

// Message 消息结构
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string 或 []ContentBlock
}

// ContentBlock 内容块
type ContentBlock struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	Source *struct {
		Type      string `json:"type"`
		MediaType string `json:"media_type"`
		Data      string `json:"data"`
	} `json:"source,omitempty"`
}

// Response 统一响应结构
type Response struct {
	ID           string            `json:"id"`
	Model        string            `json:"model"`
	Content      string            `json:"content"`
	StopReason   string            `json:"stop_reason,omitempty"`
	InputTokens  int               `json:"input_tokens"`
	OutputTokens int               `json:"output_tokens"`
	Error        *Error            `json:"error,omitempty"`
	Headers      map[string]string `json:"-"` // 响应头（用于获取限流信息等）
}

// Error 错误结构
type Error struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// StreamEvent 流式事件
type StreamEvent struct {
	Type         string `json:"type"`
	Delta        string `json:"delta,omitempty"`
	StopReason   string `json:"stop_reason,omitempty"`
	InputTokens  int    `json:"input_tokens,omitempty"`
	OutputTokens int    `json:"output_tokens,omitempty"`
	Error        *Error `json:"error,omitempty"`
}

// StreamResult 流式响应结果（包含 usage 信息）
type StreamResult struct {
	InputTokens              int               `json:"input_tokens"`
	OutputTokens             int               `json:"output_tokens"`
	CacheCreationInputTokens int               `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int               `json:"cache_read_input_tokens,omitempty"`
	Headers                  map[string]string `json:"-"` // 响应头（用于获取限流信息等）
}

// Adapter 适配器接口
type Adapter interface {
	// Name 返回适配器名称
	Name() string

	// Platform 返回平台标识
	Platform() string

	// SupportedTypes 返回支持的账户类型
	SupportedTypes() []string

	// Send 发送请求（非流式）
	Send(ctx context.Context, account *model.Account, req *Request) (*Response, error)

	// SendStream 发送请求（流式），返回 StreamResult 包含 token 使用量
	SendStream(ctx context.Context, account *model.Account, req *Request, writer io.Writer) (*StreamResult, error)
}

// AdapterFactory 适配器工厂
type AdapterFactory struct {
	adapters map[string]Adapter
}

var defaultFactory = &AdapterFactory{
	adapters: make(map[string]Adapter),
}

// Register 注册适配器
func Register(adapter Adapter) {
	for _, t := range adapter.SupportedTypes() {
		defaultFactory.adapters[t] = adapter
	}
}

// Get 获取适配器
func Get(accountType string) Adapter {
	return defaultFactory.adapters[accountType]
}

// GetByPlatform 根据平台获取适配器
func GetByPlatform(platform string) []Adapter {
	seen := make(map[string]bool)
	var result []Adapter
	for _, adapter := range defaultFactory.adapters {
		if adapter.Platform() == platform && !seen[adapter.Name()] {
			seen[adapter.Name()] = true
			result = append(result, adapter)
		}
	}
	return result
}
