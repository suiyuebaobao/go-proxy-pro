/*
 * 文件作用：错误消息数据模型，定义自定义错误响应配置
 * 负责功能：
 *   - 错误类型定义（认证、权限、限流等）
 *   - 自定义错误消息配置
 *   - 默认错误消息模板
 *   - 原始/自定义消息映射
 * 重要程度：⭐⭐⭐ 一般（错误消息数据结构）
 * 依赖模块：无
 */
package model

import "time"

// ErrorMessage 错误消息自定义配置
type ErrorMessage struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	Code            int       `gorm:"not null;index" json:"code"`                     // HTTP 状态码: 400/401/403/429/500/502/503
	ErrorType       string    `gorm:"size:50;uniqueIndex;not null" json:"error_type"` // 错误类型标识
	CustomMessage   string    `gorm:"size:500;not null" json:"custom_message"`        // 自定义返回消息
	OriginalMessage string    `gorm:"-" json:"original_message"`                      // 原始默认消息（不存数据库）
	Enabled         bool      `gorm:"default:true" json:"enabled"`                    // 是否启用自定义消息
	Description     string    `gorm:"size:200" json:"description"`                    // 说明（给管理员看）
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (ErrorMessage) TableName() string {
	return "error_messages"
}

// 预定义的错误类型常量
const (
	// 400 Bad Request
	ErrorTypeBadRequest     = "bad_request"
	ErrorTypeInvalidModel   = "invalid_model"   // 无效的模型名称
	ErrorTypeInvalidRequest = "invalid_request" // 请求格式错误

	// 401 Unauthorized
	ErrorTypeAuthFailed  = "auth_failed"
	ErrorTypeKeyDisabled = "key_disabled"
	ErrorTypeKeyExpired  = "key_expired"
	ErrorTypeKeyInvalid  = "key_invalid"

	// 403 Forbidden
	ErrorTypeForbidden        = "forbidden"
	ErrorTypeClientNotAllowed = "client_not_allowed"
	ErrorTypePlatformForbid   = "platform_forbidden"
	ErrorTypeModelForbidden   = "model_forbidden"
	ErrorTypePackageExpired   = "package_expired"   // 套餐已过期
	ErrorTypeQuotaExceeded    = "quota_exceeded"    // 配额超限（通用）
	ErrorTypeDailyLimit       = "daily_limit"       // 日限额超限
	ErrorTypeMonthlyQuota     = "monthly_quota"     // 月配额超限
	ErrorTypeIPBlocked        = "ip_blocked"        // IP 被封禁

	// 429 Too Many Requests
	ErrorTypeRateLimit            = "rate_limit"
	ErrorTypeUserConcurrencyLimit = "user_concurrency_limit"
	ErrorTypeAccountConcurrency   = "account_concurrency_limit"

	// 500 Internal Server Error
	ErrorTypeInternalError = "internal_error"

	// 502 Bad Gateway
	ErrorTypeUpstreamError       = "upstream_error"
	ErrorTypeUpstreamTimeout     = "upstream_timeout"      // 上游超时
	ErrorTypeUpstreamRateLimit   = "upstream_rate_limit"   // 上游限流
	ErrorTypeUpstreamAuthFailed  = "upstream_auth_failed"  // 上游认证失败
	ErrorTypeUpstreamForbidden   = "upstream_forbidden"    // 上游权限不足
	ErrorTypeTokenRefreshFailed  = "token_refresh_failed"  // Token 刷新失败
	ErrorTypeAllAccountsFailed   = "all_accounts_failed"   // 所有账户都失败
	ErrorTypeUnsupportedModel    = "unsupported_model"     // 不支持的模型

	// 503 Service Unavailable
	ErrorTypeNoAvailableAccount = "no_available_account"
	ErrorTypeServiceUnavailable = "service_unavailable"
	ErrorTypeMaintenanceMode    = "maintenance_mode" // 维护模式
)

// DefaultErrorMessages 默认错误消息配置
var DefaultErrorMessages = []ErrorMessage{
	// 400 Bad Request
	{Code: 400, ErrorType: ErrorTypeBadRequest, CustomMessage: "请求参数错误", Enabled: true, Description: "通用请求参数错误"},
	{Code: 400, ErrorType: ErrorTypeInvalidModel, CustomMessage: "无效的模型名称", Enabled: true, Description: "请求的模型名称无效"},
	{Code: 400, ErrorType: ErrorTypeInvalidRequest, CustomMessage: "请求格式错误", Enabled: true, Description: "请求体格式不正确"},

	// 401 Unauthorized
	{Code: 401, ErrorType: ErrorTypeAuthFailed, CustomMessage: "认证失败", Enabled: true, Description: "API Key 认证失败"},
	{Code: 401, ErrorType: ErrorTypeKeyDisabled, CustomMessage: "API Key 已禁用", Enabled: true, Description: "API Key 被禁用"},
	{Code: 401, ErrorType: ErrorTypeKeyExpired, CustomMessage: "API Key 已过期", Enabled: true, Description: "API Key 已过期"},
	{Code: 401, ErrorType: ErrorTypeKeyInvalid, CustomMessage: "无效的 API Key", Enabled: true, Description: "API Key 格式无效或不存在"},

	// 403 Forbidden
	{Code: 403, ErrorType: ErrorTypeForbidden, CustomMessage: "无权访问", Enabled: true, Description: "通用权限不足"},
	{Code: 403, ErrorType: ErrorTypeClientNotAllowed, CustomMessage: "客户端未授权", Enabled: true, Description: "客户端类型不在允许列表"},
	{Code: 403, ErrorType: ErrorTypePlatformForbid, CustomMessage: "平台访问受限", Enabled: true, Description: "套餐不支持该平台"},
	{Code: 403, ErrorType: ErrorTypeModelForbidden, CustomMessage: "模型访问受限", Enabled: true, Description: "套餐不支持该模型"},
	{Code: 403, ErrorType: ErrorTypePackageExpired, CustomMessage: "套餐已过期，请续费", Enabled: true, Description: "用户套餐已过期"},
	{Code: 403, ErrorType: ErrorTypeQuotaExceeded, CustomMessage: "配额已用尽", Enabled: true, Description: "通用配额超限"},
	{Code: 403, ErrorType: ErrorTypeDailyLimit, CustomMessage: "今日额度已用完，请明天再试", Enabled: true, Description: "日请求限额超限"},
	{Code: 403, ErrorType: ErrorTypeMonthlyQuota, CustomMessage: "本月额度已用完，请下月再试", Enabled: true, Description: "月配额超限"},
	{Code: 403, ErrorType: ErrorTypeIPBlocked, CustomMessage: "访问受限", Enabled: true, Description: "IP 地址被封禁"},

	// 429 Too Many Requests
	{Code: 429, ErrorType: ErrorTypeRateLimit, CustomMessage: "请求过于频繁，请稍后重试", Enabled: true, Description: "通用速率限制"},
	{Code: 429, ErrorType: ErrorTypeUserConcurrencyLimit, CustomMessage: "并发请求过多，请稍后重试", Enabled: true, Description: "用户并发数超限"},
	{Code: 429, ErrorType: ErrorTypeAccountConcurrency, CustomMessage: "系统繁忙，请稍后重试", Enabled: true, Description: "账户并发数超限"},

	// 500 Internal Server Error
	{Code: 500, ErrorType: ErrorTypeInternalError, CustomMessage: "服务器内部错误", Enabled: true, Description: "通用服务器错误"},

	// 502 Bad Gateway
	{Code: 502, ErrorType: ErrorTypeUpstreamError, CustomMessage: "上游服务暂时不可用", Enabled: true, Description: "上游 API 返回错误"},
	{Code: 502, ErrorType: ErrorTypeUpstreamTimeout, CustomMessage: "请求超时，请重试", Enabled: true, Description: "上游 API 响应超时"},
	{Code: 502, ErrorType: ErrorTypeUpstreamRateLimit, CustomMessage: "服务繁忙，请稍后重试", Enabled: true, Description: "上游 API 限流"},
	{Code: 502, ErrorType: ErrorTypeUpstreamAuthFailed, CustomMessage: "上游服务认证失败", Enabled: true, Description: "上游账户认证失败（401）"},
	{Code: 403, ErrorType: ErrorTypeUpstreamForbidden, CustomMessage: "上游服务权限不足", Enabled: true, Description: "上游账户权限不足（403）"},
	{Code: 502, ErrorType: ErrorTypeTokenRefreshFailed, CustomMessage: "服务暂时不可用", Enabled: true, Description: "Token 刷新失败"},
	{Code: 502, ErrorType: ErrorTypeAllAccountsFailed, CustomMessage: "服务暂时不可用，请稍后重试", Enabled: true, Description: "所有上游账户都请求失败"},
	{Code: 502, ErrorType: ErrorTypeUnsupportedModel, CustomMessage: "不支持的模型", Enabled: true, Description: "请求的模型暂不支持"},

	// 503 Service Unavailable
	{Code: 503, ErrorType: ErrorTypeNoAvailableAccount, CustomMessage: "服务暂时不可用，请稍后重试", Enabled: true, Description: "没有可用的上游账户"},
	{Code: 503, ErrorType: ErrorTypeMaintenanceMode, CustomMessage: "系统维护中，请稍后再试", Enabled: true, Description: "系统处于维护模式"},
	{Code: 503, ErrorType: ErrorTypeServiceUnavailable, CustomMessage: "服务暂时不可用", Enabled: true, Description: "通用服务不可用"},
}

// OriginalErrorMessages 原始英文错误消息示例（上游API典型返回）
var OriginalErrorMessages = map[string]string{
	// 400 Bad Request
	ErrorTypeBadRequest:     "Bad request: invalid parameters",
	ErrorTypeInvalidModel:   "The model 'xxx' does not exist",
	ErrorTypeInvalidRequest: "Invalid request body",

	// 401 Unauthorized
	ErrorTypeAuthFailed:  "Invalid API key provided",
	ErrorTypeKeyDisabled: "This API key has been disabled",
	ErrorTypeKeyExpired:  "API key has expired",
	ErrorTypeKeyInvalid:  "Incorrect API key provided",

	// 403 Forbidden
	ErrorTypeForbidden:        "Permission denied",
	ErrorTypeClientNotAllowed: "Client not allowed",
	ErrorTypePlatformForbid:   "Platform access forbidden",
	ErrorTypeModelForbidden:   "Model access forbidden for this account",
	ErrorTypePackageExpired:   "Subscription has expired",
	ErrorTypeQuotaExceeded:    "Quota exceeded",
	ErrorTypeDailyLimit:       "Daily limit exceeded",
	ErrorTypeMonthlyQuota:     "Monthly quota exceeded",
	ErrorTypeIPBlocked:        "IP address blocked",

	// 429 Too Many Requests
	ErrorTypeRateLimit:            "Rate limit exceeded. Please retry after X seconds",
	ErrorTypeUserConcurrencyLimit: "Too many concurrent requests",
	ErrorTypeAccountConcurrency:   "Account concurrency limit reached",

	// 500 Internal Server Error
	ErrorTypeInternalError: "Internal server error",

	// 502 Bad Gateway - 上游错误
	ErrorTypeUpstreamError:      "Upstream service error",
	ErrorTypeUpstreamTimeout:    "Request timeout",
	ErrorTypeUpstreamRateLimit:  "429 Too Many Requests / Rate limit reached",
	ErrorTypeUpstreamAuthFailed: "401 Unauthorized / Authentication failed",
	ErrorTypeUpstreamForbidden:  "403 Forbidden / Permission denied",
	ErrorTypeTokenRefreshFailed: "Failed to refresh access token",
	ErrorTypeAllAccountsFailed:  "All upstream accounts failed",
	ErrorTypeUnsupportedModel:   "Model not supported",

	// 503 Service Unavailable
	ErrorTypeNoAvailableAccount: "No available upstream account",
	ErrorTypeMaintenanceMode:    "Service under maintenance",
	ErrorTypeServiceUnavailable: "Service temporarily unavailable",
}

// GetOriginalMessage 根据错误类型获取原始英文错误消息
func GetOriginalMessage(errorType string) string {
	if msg, ok := OriginalErrorMessages[errorType]; ok {
		return msg
	}
	return ""
}

// FillOriginalMessage 填充原始消息字段
func (m *ErrorMessage) FillOriginalMessage() {
	m.OriginalMessage = GetOriginalMessage(m.ErrorType)
}
