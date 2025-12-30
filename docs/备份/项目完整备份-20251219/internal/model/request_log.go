package model

import (
	"time"

	"gorm.io/gorm"
)

// RequestLog 请求日志
type RequestLog struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	AccountID uint           `gorm:"index" json:"account_id"`         // 使用的账户ID
	UserID    *uint          `gorm:"index" json:"user_id,omitempty"`  // 发起请求的用户ID
	Platform  string         `gorm:"size:20;index" json:"platform"`   // 平台
	Model     string         `gorm:"size:100;index" json:"model"`     // 模型名
	Endpoint  string         `gorm:"size:100" json:"endpoint"`        // 请求端点

	// 请求信息
	Method     string `gorm:"size:10" json:"method"`                    // HTTP方法
	Path       string `gorm:"size:200" json:"path"`                     // 请求路径
	RequestIP  string `gorm:"size:50" json:"request_ip"`                // 请求IP
	UserAgent  string `gorm:"size:500" json:"user_agent,omitempty"`     // User-Agent
	SessionID  string `gorm:"size:100;index" json:"session_id,omitempty"` // 会话ID

	// 完整请求/响应记录
	RequestHeaders  string `gorm:"type:text" json:"request_headers,omitempty"`   // 请求头 JSON
	RequestBody     string `gorm:"type:longtext" json:"request_body,omitempty"`  // 请求体
	ResponseHeaders string `gorm:"type:text" json:"response_headers,omitempty"`  // 响应头 JSON
	ResponseBody    string `gorm:"type:longtext" json:"response_body,omitempty"` // 响应体

	// Token 使用
	InputTokens              int `gorm:"default:0" json:"input_tokens"`                // 输入Token
	OutputTokens             int `gorm:"default:0" json:"output_tokens"`               // 输出Token
	CacheCreationInputTokens int `gorm:"default:0" json:"cache_creation_input_tokens"` // 缓存创建Token
	CacheReadInputTokens     int `gorm:"default:0" json:"cache_read_input_tokens"`     // 缓存读取Token
	TotalTokens              int `gorm:"default:0" json:"total_tokens"`                // 总Token数

	// 费用信息（已计算倍率后的实际费用，用户可见）
	InputCost       float64 `gorm:"type:decimal(10,6);default:0" json:"input_cost"`        // 输入费用
	OutputCost      float64 `gorm:"type:decimal(10,6);default:0" json:"output_cost"`       // 输出费用
	CacheCreateCost float64 `gorm:"type:decimal(10,6);default:0" json:"cache_create_cost"` // 缓存创建费用
	CacheReadCost   float64 `gorm:"type:decimal(10,6);default:0" json:"cache_read_cost"`   // 缓存读取费用
	TotalCost       float64 `gorm:"type:decimal(10,6);default:0" json:"total_cost"`        // 总费用

	// API Key 信息（用于统计）
	APIKeyID *uint `gorm:"index" json:"api_key_id,omitempty"` // API Key ID

	// 响应信息
	StatusCode int    `gorm:"default:200" json:"status_code"`      // HTTP状态码
	Success    bool   `gorm:"default:true" json:"success"`         // 是否成功
	Error      string `gorm:"size:1000" json:"error,omitempty"`    // 错误信息
	Duration   int64  `gorm:"default:0" json:"duration"`           // 请求耗时(毫秒)

	// 上游响应信息
	UpstreamStatusCode int    `gorm:"default:0" json:"upstream_status_code"`       // 上游HTTP状态码
	UpstreamError      string `gorm:"size:2000" json:"upstream_error,omitempty"`   // 上游错误信息

	// 时间戳
	CreatedAt time.Time      `gorm:"index" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Account *Account `gorm:"foreignKey:AccountID" json:"account,omitempty"`
}

func (r *RequestLog) TableName() string {
	return "request_logs"
}

// RequestLogSummary 请求日志摘要统计
type RequestLogSummary struct {
	TotalRequests            int64   `json:"total_requests"`
	SuccessRequests          int64   `json:"success_requests"`
	FailedRequests           int64   `json:"failed_requests"`
	TotalInputTokens         int64   `json:"total_input_tokens"`
	TotalOutputTokens        int64   `json:"total_output_tokens"`
	TotalCacheCreationTokens int64   `json:"total_cache_creation_tokens"`
	TotalCacheReadTokens     int64   `json:"total_cache_read_tokens"`
	TotalTokens              int64   `json:"total_tokens"`
	TotalCost                float64 `json:"total_cost"`    // 总费用
	AvgDuration              float64 `json:"avg_duration"`  // 平均耗时(毫秒)
}

// AccountLoadStats 账户负载统计
type AccountLoadStats struct {
	AccountID    uint       `json:"account_id"`
	AccountName  string     `json:"account_name"`
	Platform     string     `json:"platform"`
	RequestCount int64      `json:"request_count"`
	SuccessCount int64      `json:"success_count"`
	ErrorCount   int64      `json:"error_count"`
	TotalTokens  int64      `json:"total_tokens"`
	TotalCost    float64    `json:"total_cost"`
	AvgDuration  float64    `json:"avg_duration"`
	LastUsedAt   *time.Time `json:"last_used_at"`
}

// UserUsageStats 用户使用统计（用户可见）
type UserUsageStats struct {
	UserID       uint    `json:"user_id"`
	TotalCost    float64 `json:"total_cost"`     // 总费用
	TotalTokens  int64   `json:"total_tokens"`   // 总Token
	TotalRequests int64  `json:"total_requests"` // 总请求数
	// 按日期统计
	DailyStats []DailyUsageStats `json:"daily_stats,omitempty"`
}

// DailyUsageStats 每日使用统计
type DailyUsageStats struct {
	Date                     string  `json:"date"`                        // 日期 YYYY-MM-DD
	RequestCount             int64   `json:"request_count"`               // 请求数
	InputTokens              int64   `json:"input_tokens"`                // 输入Token
	OutputTokens             int64   `json:"output_tokens"`               // 输出Token
	CacheCreationInputTokens int64   `json:"cache_creation_input_tokens"` // 缓存创建Token
	CacheReadInputTokens     int64   `json:"cache_read_input_tokens"`     // 缓存读取Token
	TotalTokens              int64   `json:"total_tokens"`                // 总Token
	TotalCost                float64 `json:"total_cost"`                  // 总费用
}

// ModelUsageStats 按模型统计
type ModelUsageStats struct {
	Model        string  `json:"model"`
	RequestCount int64   `json:"request_count"`
	TotalTokens  int64   `json:"total_tokens"`
	TotalCost    float64 `json:"total_cost"`
}

// UnavailableAccount 临时不可用账户
type UnavailableAccount struct {
	AccountID    uint   `json:"account_id"`
	Reason       string `json:"reason"`
	RemainingTTL int64  `json:"remaining_ttl"` // 剩余秒数
}
