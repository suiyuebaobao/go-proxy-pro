/*
 * 文件作用：错误规则数据模型，定义上游错误匹配和处理规则
 * 负责功能：
 *   - 错误匹配规则定义
 *   - HTTP状态码/关键词匹配
 *   - 账户状态转换配置
 *   - 默认错误规则模板
 * 重要程度：⭐⭐⭐ 一般（错误规则数据结构）
 * 依赖模块：无
 */
package model

import "time"

// ErrorRule 错误匹配规则
type ErrorRule struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	HTTPStatusCode int       `json:"http_status_code" gorm:"index;comment:HTTP状态码，0表示任意"`
	Keyword        string    `json:"keyword" gorm:"size:255;comment:错误关键词，空表示任意"`
	TargetStatus   string    `json:"target_status" gorm:"size:50;not null;comment:目标账户状态"`
	Priority       int       `json:"priority" gorm:"default:0;comment:优先级，越大越先匹配"`
	Enabled        bool      `json:"enabled" gorm:"default:true"`
	Description    string    `json:"description" gorm:"size:500;comment:规则描述"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// 目标状态常量
const (
	TargetStatusInvalid     = "invalid"      // 账户失效，自动禁用
	TargetStatusRateLimited = "rate_limited" // 限流，1小时后恢复
	TargetStatusOverloaded  = "overloaded"   // 过载，临时不可用
	TargetStatusValid       = "valid"        // 正常（忽略错误）
)

// DefaultErrorRules 默认错误规则
var DefaultErrorRules = []ErrorRule{
	// HTTP 状态码规则（高优先级）
	{HTTPStatusCode: 401, Keyword: "", TargetStatus: TargetStatusInvalid, Priority: 100, Enabled: true, Description: "HTTP 401 认证失败"},

	// 403 封号相关（最高优先级，确认封号才标记失效）
	{HTTPStatusCode: 403, Keyword: "account has been suspended", TargetStatus: TargetStatusInvalid, Priority: 120, Enabled: true, Description: "账户已被封禁"},
	{HTTPStatusCode: 403, Keyword: "account is disabled", TargetStatus: TargetStatusInvalid, Priority: 120, Enabled: true, Description: "账户已禁用"},
	{HTTPStatusCode: 403, Keyword: "account has been disabled", TargetStatus: TargetStatusInvalid, Priority: 120, Enabled: true, Description: "账户已禁用"},
	{HTTPStatusCode: 403, Keyword: "banned", TargetStatus: TargetStatusInvalid, Priority: 120, Enabled: true, Description: "账户被封"},
	{HTTPStatusCode: 403, Keyword: "billing", TargetStatus: TargetStatusInvalid, Priority: 110, Enabled: true, Description: "账单问题"},

	// 403 临时性错误（标记为限流，允许重试切换账户）
	{HTTPStatusCode: 403, Keyword: "permission_error", TargetStatus: TargetStatusRateLimited, Priority: 90, Enabled: true, Description: "HTTP 403 权限错误（临时）"},
	{HTTPStatusCode: 403, Keyword: "permission denied", TargetStatus: TargetStatusRateLimited, Priority: 90, Enabled: true, Description: "HTTP 403 权限被拒（临时）"},
	{HTTPStatusCode: 403, Keyword: "", TargetStatus: TargetStatusRateLimited, Priority: 80, Enabled: true, Description: "HTTP 403 其他错误（临时）"},

	// 其他 HTTP 状态码
	{HTTPStatusCode: 429, Keyword: "", TargetStatus: TargetStatusRateLimited, Priority: 100, Enabled: true, Description: "HTTP 429 限流"},
	{HTTPStatusCode: 529, Keyword: "", TargetStatus: TargetStatusOverloaded, Priority: 100, Enabled: true, Description: "HTTP 529 过载"},
	{HTTPStatusCode: 503, Keyword: "", TargetStatus: TargetStatusOverloaded, Priority: 100, Enabled: true, Description: "HTTP 503 服务不可用"},

	// 关键词规则（中等优先级）
	{HTTPStatusCode: 0, Keyword: "rate limit", TargetStatus: TargetStatusRateLimited, Priority: 50, Enabled: true, Description: "限流关键词"},
	{HTTPStatusCode: 0, Keyword: "rate_limit", TargetStatus: TargetStatusRateLimited, Priority: 50, Enabled: true, Description: "限流关键词"},
	{HTTPStatusCode: 0, Keyword: "overloaded", TargetStatus: TargetStatusOverloaded, Priority: 50, Enabled: true, Description: "过载关键词"},
	{HTTPStatusCode: 0, Keyword: "overloaded_error", TargetStatus: TargetStatusOverloaded, Priority: 50, Enabled: true, Description: "过载错误"},
	{HTTPStatusCode: 0, Keyword: "authentication_error", TargetStatus: TargetStatusInvalid, Priority: 50, Enabled: true, Description: "认证错误"},
	{HTTPStatusCode: 0, Keyword: "api_error", TargetStatus: TargetStatusOverloaded, Priority: 50, Enabled: true, Description: "API错误（临时）"},
	{HTTPStatusCode: 0, Keyword: "invalid_api_key", TargetStatus: TargetStatusInvalid, Priority: 50, Enabled: true, Description: "无效API Key"},
	{HTTPStatusCode: 0, Keyword: "invalid api key", TargetStatus: TargetStatusInvalid, Priority: 50, Enabled: true, Description: "无效API Key"},
	{HTTPStatusCode: 0, Keyword: "token expired", TargetStatus: TargetStatusInvalid, Priority: 50, Enabled: true, Description: "Token过期"},
	{HTTPStatusCode: 0, Keyword: "oauth token has expired", TargetStatus: TargetStatusInvalid, Priority: 50, Enabled: true, Description: "OAuth Token过期"},
	{HTTPStatusCode: 0, Keyword: "please run /login", TargetStatus: TargetStatusInvalid, Priority: 50, Enabled: true, Description: "需要重新登录"},
	{HTTPStatusCode: 0, Keyword: "all retries failed", TargetStatus: TargetStatusInvalid, Priority: 50, Enabled: true, Description: "所有重试失败"},
}
