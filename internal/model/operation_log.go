/*
 * 文件作用：操作日志数据模型，记录管理员操作审计信息
 * 负责功能：
 *   - 操作人信息
 *   - 操作类型和目标
 *   - 请求/响应记录
 *   - 操作时间和耗时
 * 重要程度：⭐⭐ 辅助（审计数据结构）
 * 依赖模块：无
 */
package model

import (
	"time"
)

// OperationLog 操作日志模型
type OperationLog struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	UserID       uint      `gorm:"index" json:"user_id"`                    // 操作用户ID
	Username     string    `gorm:"size:100" json:"username"`                // 操作用户名
	IP           string    `gorm:"size:50" json:"ip"`                       // 操作IP
	Method       string    `gorm:"size:10" json:"method"`                   // 请求方法 GET/POST/PUT/DELETE
	Path         string    `gorm:"size:255" json:"path"`                    // 请求路径
	Module       string    `gorm:"size:50;index" json:"module"`             // 模块名称：account/user/apikey/config等
	Action       string    `gorm:"size:50;index" json:"action"`             // 操作类型：create/update/delete/login等
	TargetID     uint      `gorm:"index" json:"target_id"`                  // 操作目标ID
	TargetName   string    `gorm:"size:255" json:"target_name"`             // 操作目标名称
	Description  string    `gorm:"size:500" json:"description"`             // 操作描述
	RequestBody  string    `gorm:"type:text" json:"request_body,omitempty"` // 请求体（脱敏后）
	ResponseCode int       `gorm:"index" json:"response_code"`              // 响应状态码
	ResponseMsg  string    `gorm:"size:500" json:"response_msg,omitempty"`  // 响应消息
	Duration     int64     `json:"duration"`                                // 请求耗时(毫秒)
	UserAgent    string    `gorm:"size:500" json:"user_agent,omitempty"`    // 用户代理
	CreatedAt    time.Time `gorm:"index" json:"created_at"`                 // 创建时间
}

// TableName 表名
func (OperationLog) TableName() string {
	return "operation_logs"
}

// 模块常量
const (
	ModuleAuth     = "auth"     // 认证
	ModuleUser     = "user"     // 用户管理
	ModuleAccount  = "account"  // 账户管理
	ModuleAPIKey   = "apikey"   // API Key管理
	ModuleModel    = "model"    // 模型管理
	ModuleConfig   = "config"   // 配置管理
	ModuleCache    = "cache"    // 缓存管理
	ModuleProxy    = "proxy"    // 代理管理
	ModulePackage  = "package"  // 套餐管理
	ModuleGroup    = "group"    // 分组管理
	ModuleSystem   = "system"   // 系统
)

// 操作类型常量
const (
	ActionLogin   = "login"   // 登录
	ActionLogout  = "logout"  // 登出
	ActionCreate  = "create"  // 创建
	ActionUpdate  = "update"  // 更新
	ActionDelete  = "delete"  // 删除
	ActionEnable  = "enable"  // 启用
	ActionDisable = "disable" // 禁用
	ActionExport  = "export"  // 导出
	ActionImport  = "import"  // 导入
	ActionClear   = "clear"   // 清除
	ActionTest    = "test"    // 测试
	ActionSync    = "sync"    // 同步
)
