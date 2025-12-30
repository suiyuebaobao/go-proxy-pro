/*
 * 文件作用：客户端过滤数据模型，定义客户端识别和过滤规则
 * 负责功能：
 *   - 客户端类型定义（Claude Code、Codex、Gemini等）
 *   - 过滤规则配置（UA、Header、Body检查）
 *   - 全局过滤配置
 *   - 预定义规则模板
 * 重要程度：⭐⭐⭐⭐ 重要（安全过滤数据结构）
 * 依赖模块：无
 */
package model

import "time"

// ClientType 客户端类型
type ClientType struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	ClientID    string    `gorm:"size:50;uniqueIndex;not null" json:"client_id"`  // 如 claude_code, codex_cli
	Name        string    `gorm:"size:100;not null" json:"name"`                  // 显示名称
	Description string    `gorm:"size:500" json:"description"`                    // 描述
	Icon        string    `gorm:"size:10" json:"icon"`                            // 图标 emoji
	Enabled     bool      `gorm:"default:true" json:"enabled"`                    // 是否启用
	Priority    int       `gorm:"default:0" json:"priority"`                      // 优先级（用于排序）
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ClientFilterRule 客户端过滤规则
type ClientFilterRule struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ClientTypeID uint      `gorm:"index;not null" json:"client_type_id"`          // 关联客户端类型
	ClientType   *ClientType `gorm:"foreignKey:ClientTypeID" json:"client_type,omitempty"`
	RuleKey      string    `gorm:"size:50;not null" json:"rule_key"`              // 规则标识
	RuleName     string    `gorm:"size:100;not null" json:"rule_name"`            // 规则名称
	Description  string    `gorm:"size:500" json:"description"`                   // 规则描述
	RuleType     string    `gorm:"size:20;not null" json:"rule_type"`             // 规则类型: header, body, user_agent, path
	Pattern      string    `gorm:"size:500" json:"pattern"`                       // 匹配模式（正则或固定值）
	FieldPath    string    `gorm:"size:200" json:"field_path"`                    // 字段路径（如 headers.x-app, body.metadata.user_id）
	Enabled      bool      `gorm:"default:true" json:"enabled"`                   // 是否启用此规则
	Required     bool      `gorm:"default:true" json:"required"`                  // 是否必须通过（false=警告但不拦截）
	Priority     int       `gorm:"default:0" json:"priority"`                     // 规则优先级
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// FilterMode 过滤模式
const (
	FilterModeSimple = "simple" // 简单模式 (claude-relay): 宽松UA检查 + 基本头检查
	FilterModeStrict = "strict" // 严格模式 (AIProxyV2): 完整UA格式 + 全部规则检查
)

// ClientFilterConfig 全局过滤配置
type ClientFilterConfig struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	FilterEnabled        bool      `gorm:"default:false" json:"filter_enabled"`          // 是否启用客户端过滤
	FilterMode           string    `gorm:"size:20;default:simple" json:"filter_mode"`    // 过滤模式: simple/strict
	DefaultAllow         bool      `gorm:"default:true" json:"default_allow"`            // 默认是否允许（无匹配客户端时）
	LogUnmatchedRequests bool      `gorm:"default:true" json:"log_unmatched_requests"`   // 是否记录未匹配的请求
	StrictMode           bool      `gorm:"default:false" json:"strict_mode"`             // 废弃，使用 FilterMode
	AllowedClients       string    `gorm:"size:500" json:"allowed_clients"`              // 全局允许的客户端列表（逗号分隔）
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// APIKeyClientFilter API Key 级别的客户端过滤配置
// 在 APIKey 模型中添加字段：AllowedClients string `gorm:"size:200" json:"allowed_clients"`

// 预定义的客户端类型 ID
const (
	ClientIDClaudeCode = "claude_code"
	ClientIDCodexCLI   = "codex_cli"
	ClientIDGeminiCLI  = "gemini_cli"
	ClientIDDroidCLI   = "droid_cli"
	ClientIDCursor     = "cursor"
	ClientIDUnknown    = "unknown"
)

// 预定义的规则类型
const (
	RuleTypeUserAgent = "user_agent"   // User-Agent 检查
	RuleTypeHeader    = "header"       // 请求头检查
	RuleTypeBody      = "body"         // 请求体检查
	RuleTypePath      = "path"         // 路径检查
	RuleTypeCustom    = "custom"       // 自定义检查（需要特殊处理）
)

// 预定义的规则标识
const (
	// Claude Code 规则
	RuleClaudeCodeUA            = "claude_code_ua"             // User-Agent 检查
	RuleClaudeCodeXApp          = "claude_code_x_app"          // x-app 头检查
	RuleClaudeCodeAnthropicVer  = "claude_code_anthropic_ver"  // anthropic-version 头检查
	RuleClaudeCodeStainlessOs   = "claude_code_stainless_os"   // x-stainless-os 头检查
	RuleClaudeCodeMetadataUser  = "claude_code_metadata_user"  // metadata.user_id 检查
	RuleClaudeCodeSystemPrompt  = "claude_code_system_prompt"  // 系统提示词检查（Dice相似度）

	// Codex CLI 规则
	RuleCodexCLIUA           = "codex_cli_ua"           // User-Agent 检查
	RuleCodexCLIOriginator   = "codex_cli_originator"   // originator 头检查
	RuleCodexCLISessionID    = "codex_cli_session_id"   // session_id 头检查
	RuleCodexCLIInstructions = "codex_cli_instructions" // instructions 检查

	// Gemini CLI 规则
	RuleGeminiCLIUA   = "gemini_cli_ua"   // User-Agent 检查
	RuleGeminiCLIPath = "gemini_cli_path" // 路径检查
)

// Claude Code System Prompt 模板（来自 AIProxyV2）
// 用于 Dice 相似度验证
var ClaudeCodeSystemPromptTemplates = []string{
	"Analyze if this message indicates a new conversation topic. If it does, extract a 2-3 word title that captures the new topic. Format your response as a JSON object with two fields: 'isNewTopic' (boolean) and 'title' (string, or null if isNewTopic is false). Only include these fields, no other text.",
	"You are Claude Code, Anthropic's official CLI for Claude.",
	"You are an interactive CLI tool that helps users",
	"You are a Claude agent, built on Anthropic's Claude Agent SDK.",
	"You are Claude Code, Anthropic's official CLI for Claude, running within the Claude Agent SDK.",
	"You are a helpful AI assistant tasked with summarizing conversations.",
	"## Insights",
	"You are an interactive CLI tool that helps users with software engineering tasks",
	"In order to encourage learning",
}

// SystemPromptSimilarityThreshold System Prompt 相似度阈值
// AIProxyV2 使用 0.5，但实测发现不相关文本也能达到 0.5+
// 提高到 0.6 以提高准确性
const SystemPromptSimilarityThreshold = 0.6

// DefaultClientTypes 默认客户端类型
var DefaultClientTypes = []ClientType{
	{
		ClientID:    ClientIDClaudeCode,
		Name:        "Claude Code",
		Description: "Claude Code 命令行工具 (claude-cli)",
		Icon:        "🤖",
		Enabled:     true,
		Priority:    100,
	},
	{
		ClientID:    ClientIDCodexCLI,
		Name:        "Codex CLI",
		Description: "Cursor/Codex 命令行工具",
		Icon:        "🔷",
		Enabled:     true,
		Priority:    90,
	},
	{
		ClientID:    ClientIDGeminiCLI,
		Name:        "Gemini CLI",
		Description: "Google Gemini 命令行工具",
		Icon:        "💎",
		Enabled:     true,
		Priority:    80,
	},
	{
		ClientID:    ClientIDDroidCLI,
		Name:        "Droid CLI",
		Description: "Factory Droid 命令行工具",
		Icon:        "🤖",
		Enabled:     true,
		Priority:    70,
	},
	{
		ClientID:    ClientIDCursor,
		Name:        "Cursor",
		Description: "Cursor IDE",
		Icon:        "📝",
		Enabled:     true,
		Priority:    60,
	},
}

// DefaultClaudeCodeRules Claude Code 默认过滤规则（参考 AIProxyV2）
var DefaultClaudeCodeRules = []ClientFilterRule{
	{
		RuleKey:     RuleClaudeCodeUA,
		RuleName:    "User-Agent 格式验证",
		Description: "验证 User-Agent 格式: claude-cli/{version} (external, {suffix})",
		RuleType:    RuleTypeUserAgent,
		Pattern:     `^claude-cli/(\d+\.\d+\.\d+)\s*\(external,\s*(cli|claude-vscode|sdk-ts|sdk-cli)(?:,\s*agent-sdk/[\w.\-]+)?\)$`,
		Enabled:     true,
		Required:    true,
		Priority:    100,
	},
	{
		RuleKey:     RuleClaudeCodeXApp,
		RuleName:    "X-App 头检查",
		Description: "验证请求包含 X-App 头",
		RuleType:    RuleTypeHeader,
		FieldPath:   "x-app",
		Pattern:     ".+",
		Enabled:     true,
		Required:    true,
		Priority:    90,
	},
	{
		RuleKey:     RuleClaudeCodeAnthropicVer,
		RuleName:    "Anthropic-Version 头检查",
		Description: "验证请求包含 Anthropic-Version 头",
		RuleType:    RuleTypeHeader,
		FieldPath:   "anthropic-version",
		Pattern:     ".+",
		Enabled:     true,
		Required:    true,
		Priority:    80,
	},
	{
		RuleKey:     RuleClaudeCodeStainlessOs,
		RuleName:    "X-Stainless-Os 头检查",
		Description: "验证请求包含 X-Stainless-Os 头",
		RuleType:    RuleTypeHeader,
		FieldPath:   "x-stainless-os",
		Pattern:     ".+",
		Enabled:     true,
		Required:    true,
		Priority:    70,
	},
	{
		RuleKey:     RuleClaudeCodeSystemPrompt,
		RuleName:    "System Prompt 相似度验证",
		Description: "使用 Dice 系数验证系统提示词与 Claude Code 模板相似度 ≥ 0.5",
		RuleType:    RuleTypeCustom,
		FieldPath:   "system",
		Pattern:     "",
		Enabled:     true,
		Required:    true,
		Priority:    60,
	},
	{
		RuleKey:     RuleClaudeCodeMetadataUser,
		RuleName:    "metadata.user_id 格式验证",
		Description: "验证格式: user_{64位hex}_account__session_{UUID}",
		RuleType:    RuleTypeBody,
		FieldPath:   "metadata.user_id",
		Pattern:     `^user_[a-fA-F0-9]{64}_account__session_[\w-]+$`,
		Enabled:     true,
		Required:    true,
		Priority:    50,
	},
}

// DefaultCodexCLIRules Codex CLI 默认过滤规则
var DefaultCodexCLIRules = []ClientFilterRule{
	{
		RuleKey:     RuleCodexCLIUA,
		RuleName:    "User-Agent 检查",
		Description: "验证 User-Agent 是否为 codex_vscode 或 codex_cli_rs 格式",
		RuleType:    RuleTypeUserAgent,
		Pattern:     `^(codex_vscode|codex_cli_rs)/[\d.]+`,
		Enabled:     true,
		Required:    true,
		Priority:    100,
	},
	{
		RuleKey:     RuleCodexCLIOriginator,
		RuleName:    "Originator 头检查",
		Description: "验证 originator 头与 User-Agent 客户端类型匹配",
		RuleType:    RuleTypeHeader,
		FieldPath:   "originator",
		Pattern:     `^(codex_vscode|codex_cli_rs)$`,
		Enabled:     true,
		Required:    true,
		Priority:    90,
	},
	{
		RuleKey:     RuleCodexCLISessionID,
		RuleName:    "Session ID 检查",
		Description: "验证 session_id 头存在且长度大于20",
		RuleType:    RuleTypeHeader,
		FieldPath:   "session_id",
		Pattern:     `.{21,}`, // 至少21个字符
		Enabled:     true,
		Required:    true,
		Priority:    80,
	},
	{
		RuleKey:     RuleCodexCLIInstructions,
		RuleName:    "Instructions 检查",
		Description: "验证请求体中的 instructions 字段前缀",
		RuleType:    RuleTypeBody,
		FieldPath:   "instructions",
		Pattern:     `^You are Codex, based on GPT-5`,
		Enabled:     true,
		Required:    true,
		Priority:    70,
	},
}

// DefaultGeminiCLIRules Gemini CLI 默认过滤规则
var DefaultGeminiCLIRules = []ClientFilterRule{
	{
		RuleKey:     RuleGeminiCLIUA,
		RuleName:    "User-Agent 检查",
		Description: "验证 User-Agent 是否为 GeminiCLI 格式",
		RuleType:    RuleTypeUserAgent,
		Pattern:     `^GeminiCLI/v?[\d.]+`,
		Enabled:     true,
		Required:    true,
		Priority:    100,
	},
	{
		RuleKey:     RuleGeminiCLIPath,
		RuleName:    "路径检查",
		Description: "验证请求路径以 /gemini 开头",
		RuleType:    RuleTypePath,
		Pattern:     `^/gemini`,
		Enabled:     true,
		Required:    true,
		Priority:    90,
	},
}
