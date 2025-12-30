package model

import (
	"time"

	"gorm.io/gorm"
)

// DailyUsage 每日使用汇总（MySQL持久化）
// 每个用户每个模型每天一条记录，增量更新
type DailyUsage struct {
	ID     uint   `gorm:"primarykey" json:"id"`
	UserID uint   `gorm:"uniqueIndex:idx_user_date_model,priority:1" json:"user_id"` // 用户ID
	Date   string `gorm:"size:10;uniqueIndex:idx_user_date_model,priority:2" json:"date"` // 日期 YYYY-MM-DD
	Model  string `gorm:"size:100;uniqueIndex:idx_user_date_model,priority:3" json:"model"` // 模型名

	// Token 使用量
	RequestCount             int64 `gorm:"default:0" json:"request_count"`               // 请求次数
	InputTokens              int64 `gorm:"default:0" json:"input_tokens"`                // 输入Token
	OutputTokens             int64 `gorm:"default:0" json:"output_tokens"`               // 输出Token
	CacheCreationInputTokens int64 `gorm:"default:0" json:"cache_creation_input_tokens"` // 缓存创建Token
	CacheReadInputTokens     int64 `gorm:"default:0" json:"cache_read_input_tokens"`     // 缓存读取Token
	TotalTokens              int64 `gorm:"default:0" json:"total_tokens"`                // 总Token

	// 费用（已计算用户费率后的实际费用）
	InputCost       float64 `gorm:"type:decimal(12,6);default:0" json:"input_cost"`        // 输入费用
	OutputCost      float64 `gorm:"type:decimal(12,6);default:0" json:"output_cost"`       // 输出费用
	CacheCreateCost float64 `gorm:"type:decimal(12,6);default:0" json:"cache_create_cost"` // 缓存创建费用
	CacheReadCost   float64 `gorm:"type:decimal(12,6);default:0" json:"cache_read_cost"`   // 缓存读取费用
	TotalCost       float64 `gorm:"type:decimal(12,6);default:0" json:"total_cost"`        // 总费用

	// 时间戳
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (d *DailyUsage) TableName() string {
	return "daily_usage"
}

// DailyUsageSummary 每日汇总（不分模型）
type DailyUsageSummary struct {
	Date         string  `json:"date"`
	RequestCount int64   `json:"request_count"`
	TotalTokens  int64   `json:"total_tokens"`
	TotalCost    float64 `json:"total_cost"`
}

// ModelUsageSummary 模型使用汇总
type ModelUsageSummary struct {
	Model        string  `json:"model"`
	RequestCount int64   `json:"request_count"`
	TotalTokens  int64   `json:"total_tokens"`
	TotalCost    float64 `json:"total_cost"`
}

// UserUsageSummary 用户使用汇总
type UserUsageSummary struct {
	UserID        uint    `json:"user_id"`
	TotalRequests int64   `json:"total_requests"`
	TotalTokens   int64   `json:"total_tokens"`
	TotalCost     float64 `json:"total_cost"`
}
