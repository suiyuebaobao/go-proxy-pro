package model

import (
	"time"
)

// UsageRecord 使用记录（MySQL 持久化）
type UsageRecord struct {
	ID                       uint      `gorm:"primarykey" json:"id"`
	UserID                   uint      `gorm:"index;not null" json:"user_id"`
	APIKeyID                 uint      `gorm:"index" json:"api_key_id"`
	Model                    string    `gorm:"size:100;index" json:"model"`
	Platform                 string    `gorm:"size:50" json:"platform"`
	InputTokens              int       `gorm:"default:0" json:"input_tokens"`
	OutputTokens             int       `gorm:"default:0" json:"output_tokens"`
	CacheCreationInputTokens int       `gorm:"default:0" json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int       `gorm:"default:0" json:"cache_read_input_tokens"`
	TotalTokens              int       `gorm:"default:0" json:"total_tokens"`
	TotalCost                float64   `gorm:"type:decimal(10,6);default:0" json:"total_cost"`
	RequestTime              time.Time `gorm:"index" json:"request_time"` // 请求时间
	CreatedAt                time.Time `json:"created_at"`
}

func (r *UsageRecord) TableName() string {
	return "usage_records"
}
