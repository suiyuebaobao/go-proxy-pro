/*
 * 文件作用：系统配置数据模型，定义全局配置项结构
 * 负责功能：
 *   - 配置键值对存储结构
 *   - 配置类型定义（string/int/float/bool/json）
 *   - 配置分类管理
 *   - 默认配置项定义
 * 重要程度：⭐⭐⭐⭐ 重要（系统配置数据结构）
 * 依赖模块：无
 */
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
	// 计费相关
	ConfigGlobalPriceRate = "global_price_rate" // 全局价格倍率

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

	// 账号健康检查相关
	ConfigAccountHealthCheckEnabled  = "account_health_check_enabled"  // 是否启用账号健康检查
	ConfigAccountHealthCheckInterval = "account_health_check_interval" // 检查间隔（分钟）
	ConfigAccountErrorThreshold      = "account_error_threshold"       // 连续错误阈值

	// OAuth 自动重新授权相关
	ConfigOAuthAutoReauthorizeEnabled = "oauth_auto_reauthorize_enabled" // 是否启用 OAuth 自动重新授权
	ConfigOAuthReauthorizeCooldown    = "oauth_reauthorize_cooldown"     // 重新授权失败后的冷却时间（分钟）

	// 健康检测策略 - 全局开关
	ConfigHealthCheckAutoRecovery    = "health_check_auto_recovery"     // 启用自动恢复
	ConfigHealthCheckAutoTokenRefresh = "health_check_auto_token_refresh" // 启用 Token 自动刷新

	// 健康检测策略 - 限流账号
	ConfigRateLimitedProbeEnabled     = "rate_limited_probe_enabled"      // 启用主动探测（不傻等 reset_at）
	ConfigRateLimitedProbeInitInterval = "rate_limited_probe_init_interval" // 初始探测间隔（分钟）
	ConfigRateLimitedProbeMaxInterval  = "rate_limited_probe_max_interval"  // 最大探测间隔（分钟）
	ConfigRateLimitedProbeBackoff      = "rate_limited_probe_backoff"       // 间隔递增因子

	// 健康检测策略 - 疑似封号
	ConfigSuspendedProbeInterval   = "suspended_probe_interval"    // 探测间隔（分钟）
	ConfigSuspendedConfirmThreshold = "suspended_confirm_threshold" // 确认封号阈值（连续失败次数）

	// 健康检测策略 - 已封号
	ConfigBannedProbeEnabled  = "banned_probe_enabled"   // 启用复活检测
	ConfigBannedProbeInterval = "banned_probe_interval"  // 探测间隔（小时）

	// 健康检测策略 - Token 刷新
	ConfigTokenRefreshCooldown   = "token_refresh_cooldown"    // 刷新失败冷却时间（分钟）
	ConfigTokenRefreshMaxRetries = "token_refresh_max_retries" // 最大重试次数
)

// 默认配置
var DefaultConfigs = []SystemConfig{
	// 计费配置
	{Key: ConfigGlobalPriceRate, Value: "1", Type: "float", Desc: "全局价格倍率（1=原价，0=免费，2=2倍），用户倍率为1时使用此值", Category: "billing"},
	// 会话配置
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
	// 账号健康检查配置
	{Key: ConfigAccountHealthCheckEnabled, Value: "false", Type: "bool", Desc: "是否启用账号健康检查", Category: "health_check"},
	{Key: ConfigAccountHealthCheckInterval, Value: "5", Type: "int", Desc: "账号健康检查间隔（分钟）", Category: "health_check"},
	{Key: ConfigAccountErrorThreshold, Value: "5", Type: "int", Desc: "账号连续错误阈值（达到后禁用账号）", Category: "health_check"},
	// OAuth 自动重新授权配置
	{Key: ConfigOAuthAutoReauthorizeEnabled, Value: "true", Type: "bool", Desc: "是否启用 OAuth Token 失效时自动用 SessionKey 重新授权", Category: "health_check"},
	{Key: ConfigOAuthReauthorizeCooldown, Value: "30", Type: "int", Desc: "OAuth 重新授权失败后的冷却时间（分钟），避免频繁尝试", Category: "health_check"},
	// 健康检测策略 - 全局开关
	{Key: ConfigHealthCheckAutoRecovery, Value: "true", Type: "bool", Desc: "检测成功后自动恢复账号", Category: "health_check"},
	{Key: ConfigHealthCheckAutoTokenRefresh, Value: "true", Type: "bool", Desc: "Token 过期时自动刷新", Category: "health_check"},
	// 健康检测策略 - 限流账号
	{Key: ConfigRateLimitedProbeEnabled, Value: "true", Type: "bool", Desc: "启用限流账号主动探测（不傻等官方返回的恢复时间）", Category: "health_check"},
	{Key: ConfigRateLimitedProbeInitInterval, Value: "10", Type: "int", Desc: "限流账号初始探测间隔（分钟）", Category: "health_check"},
	{Key: ConfigRateLimitedProbeMaxInterval, Value: "30", Type: "int", Desc: "限流账号最大探测间隔（分钟）", Category: "health_check"},
	{Key: ConfigRateLimitedProbeBackoff, Value: "1.5", Type: "float", Desc: "限流账号探测间隔递增因子", Category: "health_check"},
	// 健康检测策略 - 疑似封号
	{Key: ConfigSuspendedProbeInterval, Value: "5", Type: "int", Desc: "疑似封号账号探测间隔（分钟）", Category: "health_check"},
	{Key: ConfigSuspendedConfirmThreshold, Value: "3", Type: "int", Desc: "确认封号阈值（连续检测失败次数）", Category: "health_check"},
	// 健康检测策略 - 已封号
	{Key: ConfigBannedProbeEnabled, Value: "true", Type: "bool", Desc: "启用封号账号复活检测", Category: "health_check"},
	{Key: ConfigBannedProbeInterval, Value: "1", Type: "int", Desc: "封号账号复活探测间隔（小时）", Category: "health_check"},
	// 健康检测策略 - Token 刷新
	{Key: ConfigTokenRefreshCooldown, Value: "30", Type: "int", Desc: "Token 刷新失败冷却时间（分钟）", Category: "health_check"},
	{Key: ConfigTokenRefreshMaxRetries, Value: "3", Type: "int", Desc: "Token 刷新最大重试次数", Category: "health_check"},
}
