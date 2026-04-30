/*
 * 文件作用：告警系统数据模型定义
 * 负责功能：
 *   - AlertRule 告警规则模型（条件、渠道、静默期）
 *   - AlertLog 告警历史记录模型
 * 重要程度：⭐⭐⭐ 一般（告警子系统数据层）
 * 依赖模块：gorm
 */
package model

import (
	"time"

	"gorm.io/gorm"
)

// AlertRule 告警规则
type AlertRule struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	Name           string         `gorm:"size:100;not null" json:"name"`
	ConditionType  string         `gorm:"size:50;not null" json:"condition_type"`   // account_banned, rate_limited, quota_exhausted, cpu_high, memory_high, disk_high, error_spike
	ConditionValue string         `gorm:"size:200" json:"condition_value"`          // JSON: {"threshold": 90} or {}
	ChannelType    string         `gorm:"size:20;not null" json:"channel_type"`     // telegram, webhook, email
	ChannelConfig  string         `gorm:"type:text;not null" json:"channel_config"` // JSON: channel-specific config
	SilenceMinutes int            `gorm:"default:30" json:"silence_minutes"`
	Enabled        bool           `gorm:"default:true" json:"enabled"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AlertRule) TableName() string { return "alert_rules" }

// AlertLog 告警日志
type AlertLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	RuleID    uint      `gorm:"index" json:"rule_id"`
	RuleName  string    `gorm:"size:100" json:"rule_name"`
	AlertType string    `gorm:"size:50" json:"alert_type"`
	Message   string    `gorm:"type:text" json:"message"`
	Channel   string    `gorm:"size:20" json:"channel"`
	Status    string    `gorm:"size:20;default:sent" json:"status"` // sent, failed
	Error     string    `gorm:"size:500" json:"error,omitempty"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

func (AlertLog) TableName() string { return "alert_logs" }
