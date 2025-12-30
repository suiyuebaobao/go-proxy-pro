package model

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Proxy 代理配置
type Proxy struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`              // 代理名称
	Type        string         `gorm:"size:20;not null;default:http" json:"type"`  // 代理类型: http, https, socks5
	Host        string         `gorm:"size:200;not null" json:"host"`              // 代理主机
	Port        int            `gorm:"not null" json:"port"`                       // 代理端口
	Username    string         `gorm:"size:100" json:"username,omitempty"`         // 认证用户名
	Password    string         `gorm:"size:100" json:"password,omitempty"`         // 认证密码
	Enabled     bool           `gorm:"default:true" json:"enabled"`                // 是否启用
	IsDefault   bool           `gorm:"default:false" json:"is_default"`            // 是否为默认代理（用于OAuth认证）
	TestStatus  string         `gorm:"size:20" json:"test_status"`                 // 测试状态: success, failed, 空表示未测试
	TestLatency int            `gorm:"default:0" json:"test_latency"`              // 测试延迟(ms)
	TestError   string         `gorm:"size:500" json:"test_error,omitempty"`       // 测试错误信息
	LastTestAt  *time.Time     `json:"last_test_at,omitempty"`                     // 最后测试时间
	Remark      string         `gorm:"size:500" json:"remark,omitempty"`           // 备注
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// ProxyType 代理类型常量
const (
	ProxyTypeHTTP   = "http"
	ProxyTypeHTTPS  = "https"
	ProxyTypeSOCKS5 = "socks5"
)

// GetURL 获取代理完整 URL
func (p *Proxy) GetURL() string {
	if p.Username != "" && p.Password != "" {
		return fmt.Sprintf("%s://%s:%s@%s:%d", p.Type, p.Username, p.Password, p.Host, p.Port)
	}
	return fmt.Sprintf("%s://%s:%d", p.Type, p.Host, p.Port)
}
