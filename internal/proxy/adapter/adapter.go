/*
 * 文件作用：适配器接口定义和通用工具，定义所有AI平台适配器的统一接口
 * 负责功能：
 *   - Adapter 接口定义（Send/SendStream）
 *   - 适配器注册表管理
 *   - UpstreamError 上游错误类型
 *   - StreamResult 流式结果封装
 *   - TailWriter 流式响应末尾捕获
 *   - 通用响应头处理
 * 重要程度：⭐⭐⭐⭐⭐ 核心（所有适配器的基础接口）
 * 依赖模块：model
 */
package adapter

import (
	"context"
	"errors"
	"fmt"
	"io"

	"go-aiproxy/internal/model"
)

var (
	ErrNoAdapter = errors.New("no adapter found for account type")
)

// UpstreamError 上游错误（包含状态码）
type UpstreamError struct {
	StatusCode int
	Message    string
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("[HTTP %d] %s", e.StatusCode, e.Message)
}

// NewUpstreamError 创建上游错误
func NewUpstreamError(statusCode int, message string) *UpstreamError {
	return &UpstreamError{
		StatusCode: statusCode,
		Message:    message,
	}
}

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

// TailWriter 包装 Writer，同时捕获末尾 N 字节
type TailWriter struct {
	w       io.Writer
	tail    []byte
	maxSize int
}

// NewTailWriter 创建 TailWriter，捕获末尾 maxSize 字节
func NewTailWriter(w io.Writer, maxSize int) *TailWriter {
	return &TailWriter{
		w:       w,
		tail:    make([]byte, 0, maxSize),
		maxSize: maxSize,
	}
}

// Write 实现 io.Writer 接口
func (t *TailWriter) Write(p []byte) (n int, err error) {
	// 先写入原始 writer
	n, err = t.w.Write(p)
	if err != nil {
		return n, err
	}

	// 追加到 tail 缓冲区
	t.tail = append(t.tail, p[:n]...)

	// 如果超过最大大小，只保留末尾部分
	if len(t.tail) > t.maxSize {
		t.tail = t.tail[len(t.tail)-t.maxSize:]
	}

	return n, nil
}

// Tail 获取捕获的末尾内容
func (t *TailWriter) Tail() []byte {
	return t.tail
}

// Flush 实现 http.Flusher 接口（如果底层 writer 支持）
func (t *TailWriter) Flush() {
	if f, ok := t.w.(interface{ Flush() }); ok {
		f.Flush()
	}
}
