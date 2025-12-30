package model

import (
	"time"
)

// SystemConfig 系统配置
type SystemConfig struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Key       string    `gorm:"column:config_key;size:100;uniqueIndex;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	Type      string    `gorm:"size:20;default:string" json:"type"` // string, int, float, bool, json
	Desc      string    `gorm:"size:255" json:"desc"`
	Category  string    `gorm:"size:50;index" json:"category"` // 分类：session, sync, system 等
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (c *SystemConfig) TableName() string {
	return "system_configs"
}

// 配置 Key 常量
const (
	// 会话相关
	ConfigSessionTTL = "session_ttl" // 会话粘性 TTL（分钟）

	// 同步相关
	ConfigSyncEnabled  = "sync_enabled"  // 是否启用同步
	ConfigSyncInterval = "sync_interval" // 同步间隔（分钟）

	// 记录相关
	ConfigRecordRetentionDays = "record_retention_days" // Redis 记录保留天数
	ConfigRecordMaxCount      = "record_max_count"      // Redis 最大记录数

	// 安全相关
	ConfigCaptchaEnabled       = "captcha_enabled"        // 是否启用验证码
	ConfigCaptchaRateLimit     = "captcha_rate_limit"     // 验证码获取频率限制（次/分钟）
	ConfigLoginRateLimitEnable = "login_rate_limit_enable" // 是否启用登录频率限制
	ConfigLoginRateLimitCount  = "login_rate_limit_count"  // 登录频率限制次数
	ConfigLoginRateLimitWindow = "login_rate_limit_window" // 登录频率限制时间窗口（分钟）
)

// 默认配置
var DefaultConfigs = []SystemConfig{
	{Key: ConfigSessionTTL, Value: "30", Type: "int", Desc: "会话粘性过期时间（分钟）", Category: "session"},
	{Key: ConfigSyncEnabled, Value: "true", Type: "bool", Desc: "是否启用使用记录同步", Category: "sync"},
	{Key: ConfigSyncInterval, Value: "5", Type: "int", Desc: "使用记录同步间隔（分钟）", Category: "sync"},
	{Key: ConfigRecordRetentionDays, Value: "30", Type: "int", Desc: "Redis 使用记录保留天数", Category: "record"},
	{Key: ConfigRecordMaxCount, Value: "1000", Type: "int", Desc: "Redis 每用户最大记录数", Category: "record"},
	// 安全配置
	{Key: ConfigCaptchaEnabled, Value: "true", Type: "bool", Desc: "是否启用登录验证码", Category: "security"},
	{Key: ConfigCaptchaRateLimit, Value: "10", Type: "int", Desc: "验证码获取频率限制（次/分钟）", Category: "security"},
	{Key: ConfigLoginRateLimitEnable, Value: "true", Type: "bool", Desc: "是否启用登录频率限制", Category: "security"},
	{Key: ConfigLoginRateLimitCount, Value: "3", Type: "int", Desc: "登录频率限制次数", Category: "security"},
	{Key: ConfigLoginRateLimitWindow, Value: "5", Type: "int", Desc: "登录频率限制时间窗口（分钟）", Category: "security"},
}
