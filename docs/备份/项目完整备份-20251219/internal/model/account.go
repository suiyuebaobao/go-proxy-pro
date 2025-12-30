package model

import (
	"time"

	"gorm.io/gorm"
)

// 账户类型常量
const (
	AccountTypeClaudeOfficial  = "claude-official"   // Claude Official OAuth
	AccountTypeClaudeConsole   = "claude-console"    // Claude Console
	AccountTypeBedrock         = "bedrock"           // AWS Bedrock
	AccountTypeCCR             = "ccr"               // Claude CCR
	AccountTypeOpenAI          = "openai"            // OpenAI
	AccountTypeOpenAIResponses = "openai-responses"  // OpenAI Responses API
	AccountTypeAzureOpenAI     = "azure-openai"      // Azure OpenAI
	AccountTypeGemini          = "gemini"            // Google Gemini OAuth
	AccountTypeGeminiAPI       = "gemini-api"        // Gemini API Key
	AccountTypeDroid           = "droid"             // Droid
)

// 平台常量
const (
	PlatformClaude = "claude"
	PlatformOpenAI = "openai"
	PlatformGemini = "gemini"
	PlatformOther  = "other"
)

// 账户状态常量
const (
	AccountStatusValid       = "valid"        // 正常
	AccountStatusInvalid     = "invalid"      // 无效/失效
	AccountStatusRateLimited = "rate_limited" // 限流中
	AccountStatusOverloaded  = "overloaded"   // 过载
)

// Account 账户模型
type Account struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Name      string         `gorm:"size:100;not null" json:"name"`           // 账户名称
	Type      string         `gorm:"size:30;not null;index" json:"type"`      // 账户类型
	Platform  string         `gorm:"size:20;not null;index" json:"platform"`  // 平台
	Status    string         `gorm:"size:20;default:valid" json:"status"`     // 状态
	Enabled   bool           `gorm:"default:true" json:"enabled"`             // 是否启用
	Priority  int            `gorm:"default:50" json:"priority"`              // 优先级 1-100
	Weight    int            `gorm:"default:100" json:"weight"`               // 权重

	// 通用认证字段
	APIKey      string `gorm:"size:500" json:"api_key,omitempty"`       // API Key
	APISecret   string `gorm:"size:500" json:"api_secret,omitempty"`    // API Secret
	AccessToken string `gorm:"size:2000" json:"access_token,omitempty"` // Access Token
	RefreshToken string `gorm:"size:2000" json:"refresh_token,omitempty"` // Refresh Token
	TokenExpiry *time.Time `json:"token_expiry,omitempty"`               // Token 过期时间

	// Claude Official 专用
	SessionKey        string `gorm:"type:text" json:"session_key,omitempty"`        // Session Key
	OrganizationID    string `gorm:"size:100" json:"organization_id,omitempty"`    // 组织 ID
	SubscriptionLevel string `gorm:"size:20" json:"subscription_level,omitempty"`  // 订阅级别: free/pro/team
	OpusAccess        bool   `gorm:"default:false" json:"opus_access"`             // 是否有 Opus 权限

	// AWS Bedrock 专用
	AWSAccessKey    string `gorm:"size:100" json:"aws_access_key,omitempty"`
	AWSSecretKey    string `gorm:"size:200" json:"aws_secret_key,omitempty"`
	AWSRegion       string `gorm:"size:30" json:"aws_region,omitempty"`
	AWSSessionToken string `gorm:"size:2000" json:"aws_session_token,omitempty"`

	// Azure OpenAI 专用
	AzureEndpoint      string `gorm:"size:200" json:"azure_endpoint,omitempty"`
	AzureDeploymentName string `gorm:"size:100" json:"azure_deployment_name,omitempty"`
	AzureAPIVersion    string `gorm:"size:20" json:"azure_api_version,omitempty"`

	// 通用配置
	BaseURL        string `gorm:"size:200" json:"base_url,omitempty"`        // 自定义 Base URL
	ProxyID        *uint  `gorm:"index" json:"proxy_id,omitempty"`           // 关联的代理 ID
	ModelMapping   string `gorm:"type:text" json:"model_mapping,omitempty"`  // 模型映射 JSON
	AllowedModels  string `gorm:"type:text" json:"allowed_models,omitempty"` // 允许的模型列表
	MaxConcurrency int    `gorm:"default:5" json:"max_concurrency"`          // 最大并发数

	// 关联对象
	Proxy *Proxy `gorm:"foreignKey:ProxyID" json:"proxy,omitempty"` // 代理配置

	// 统计字段
	RequestCount     int64      `gorm:"default:0" json:"request_count"`             // 请求次数
	ErrorCount       int64      `gorm:"default:0" json:"error_count"`               // 错误次数
	TotalCost        float64    `gorm:"default:0" json:"total_cost"`                // 总费用（从 Redis 同步）
	LastUsedAt       *time.Time `json:"last_used_at,omitempty"`                     // 最后使用时间
	LastErrorAt      *time.Time `json:"last_error_at,omitempty"`                    // 最后错误时间
	LastError        string     `gorm:"size:500" json:"last_error,omitempty"`       // 最后错误信息
	RateLimitResetAt *time.Time `json:"rate_limit_reset_at,omitempty"`              // 限流恢复时间

	// Claude 用量字段 (从 OAuth Usage API 获取)
	UsageStatus          string     `gorm:"size:30" json:"usage_status,omitempty"`            // 5H窗口状态: allowed/allowed_warning/rejected
	UsageStatusUpdatedAt *time.Time `json:"usage_status_updated_at,omitempty"`                // 用量状态更新时间
	RateLimitReset       *int64     `json:"rate_limit_reset,omitempty"`                       // 限流重置时间戳

	// Claude 用量百分比 (从 OAuth Usage API 获取)
	FiveHourUtilization       *float64   `json:"five_hour_utilization"`        // 5小时窗口用量百分比 (0-100)
	FiveHourResetsAt          *time.Time `json:"five_hour_resets_at"`          // 5小时窗口重置时间
	SevenDayUtilization       *float64   `json:"seven_day_utilization"`        // 7天窗口用量百分比 (0-100)
	SevenDayResetsAt          *time.Time `json:"seven_day_resets_at"`          // 7天窗口重置时间
	SevenDaySonnetUtilization *float64   `json:"seven_day_sonnet_utilization"` // 7天Sonnet窗口用量百分比 (0-100)
	SevenDaySonnetResetsAt    *time.Time `json:"seven_day_sonnet_resets_at"`   // 7天Sonnet窗口重置时间

	// 时间戳
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Groups []AccountGroup `gorm:"many2many:account_group_members;" json:"groups,omitempty"`
}

func (a *Account) TableName() string {
	return "accounts"
}

// GetPlatformByType 根据账户类型获取平台
func GetPlatformByType(accountType string) string {
	switch accountType {
	case AccountTypeClaudeOfficial, AccountTypeClaudeConsole, AccountTypeBedrock, AccountTypeCCR:
		return PlatformClaude
	case AccountTypeOpenAI, AccountTypeOpenAIResponses, AccountTypeAzureOpenAI:
		return PlatformOpenAI
	case AccountTypeGemini, AccountTypeGeminiAPI:
		return PlatformGemini
	default:
		return PlatformOther
	}
}

// AccountGroup 账户分组
type AccountGroup struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Name        string         `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Description string         `gorm:"size:500" json:"description,omitempty"`
	Platform    string         `gorm:"size:20" json:"platform,omitempty"` // 限定平台
	IsDefault   bool           `gorm:"default:false" json:"is_default"`   // 是否默认分组
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Accounts []Account `gorm:"many2many:account_group_members;" json:"accounts,omitempty"`
}

func (g *AccountGroup) TableName() string {
	return "account_groups"
}
