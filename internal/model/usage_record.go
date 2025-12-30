/*
 * 文件作用：使用记录数据模型，定义Token消耗持久化结构
 * 负责功能：
 *   - 请求使用记录存储
 *   - Token计数（输入/输出/缓存）
 *   - 费用计算记录
 *   - 用户/APIKey关联
 * 重要程度：⭐⭐⭐ 一般（使用记录数据结构）
 * 依赖模块：无
 */
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
	RequestIP                string    `gorm:"size:50" json:"request_ip"`  // 请求IP地址
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
