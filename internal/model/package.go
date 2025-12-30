/*
 * 文件作用：套餐数据模型，定义套餐模板和用户套餐结构
 * 负责功能：
 *   - 套餐模板定义（订阅/额度）
 *   - 用户套餐分配
 *   - 额度限制配置
 *   - 模型访问权限
 * 重要程度：⭐⭐⭐ 一般（套餐数据结构）
 * 依赖模块：gorm
 */
package model

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Package 套餐定义（管理员创建的套餐模板）
type Package struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`                       // 套餐名称
	Type        string         `gorm:"size:20;not null;index" json:"type"`                  // subscription(订阅包月) / quota(额度)
	Price       float64        `gorm:"type:decimal(10,2);default:0" json:"price"`           // 价格（美元）
	Duration    int            `gorm:"default:30" json:"duration"`                          // 有效期天数

	// 订阅类型的额度限制（美元）
	DailyQuota  float64        `gorm:"type:decimal(10,4);default:0" json:"daily_quota"`     // 每日额度（0=不限）
	WeeklyQuota float64        `gorm:"type:decimal(10,4);default:0" json:"weekly_quota"`    // 每周额度（0=不限）
	MonthlyQuota float64       `gorm:"type:decimal(10,4);default:0" json:"monthly_quota"`   // 每月额度（0=不限）

	// 额度类型的总额度
	QuotaAmount float64        `gorm:"type:decimal(10,4);default:0" json:"quota_amount"`    // 总额度（额度类型使用，美元）

	// 模型限制
	AllowedModels string       `gorm:"type:text" json:"allowed_models"`                     // 允许的模型（逗号分隔，空=全部）

	Description string         `gorm:"size:500" json:"description"`                         // 套餐描述
	Status      string         `gorm:"size:20;default:active" json:"status"`                // active/disabled
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (p *Package) TableName() string {
	return "packages"
}

// UserPackage 用户购买的套餐
type UserPackage struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	UserID       uint           `gorm:"index;not null" json:"user_id"`
	PackageID    uint           `gorm:"index;not null" json:"package_id"`
	Package      *Package       `gorm:"foreignKey:PackageID" json:"package,omitempty"`
	Name         string         `gorm:"size:100;not null" json:"name"`                    // 套餐名称（冗余存储）
	Type         string         `gorm:"size:20;not null;index" json:"type"`               // subscription / quota
	Status       string         `gorm:"size:20;default:active;index" json:"status"`       // active/expired/exhausted/disabled

	// 时间相关
	StartTime    *time.Time     `json:"start_time,omitempty"`                             // 开始时间
	ExpireTime   *time.Time     `gorm:"index" json:"expire_time,omitempty"`               // 到期时间

	// 订阅类型的额度限制（从套餐模板复制）
	DailyQuota   float64        `gorm:"type:decimal(10,4);default:0" json:"daily_quota"`  // 每日额度限制
	WeeklyQuota  float64        `gorm:"type:decimal(10,4);default:0" json:"weekly_quota"` // 每周额度限制
	MonthlyQuota float64        `gorm:"type:decimal(10,4);default:0" json:"monthly_quota"`// 每月额度限制

	// 订阅类型的周期使用量
	DailyUsed    float64        `gorm:"type:decimal(10,4);default:0" json:"daily_used"`   // 今日已用
	WeeklyUsed   float64        `gorm:"type:decimal(10,4);default:0" json:"weekly_used"`  // 本周已用
	MonthlyUsed  float64        `gorm:"type:decimal(10,4);default:0" json:"monthly_used"` // 本月已用
	LastResetDay string         `gorm:"size:10" json:"last_reset_day"`                    // 上次重置日期 (YYYY-MM-DD)
	LastResetWeek string        `gorm:"size:10" json:"last_reset_week"`                   // 上次重置周 (YYYY-WW)
	LastResetMonth string       `gorm:"size:7" json:"last_reset_month"`                   // 上次重置月 (YYYY-MM)

	// 额度类型字段
	QuotaTotal   float64        `gorm:"type:decimal(10,4);default:0" json:"quota_total"`  // 总额度（美元）
	QuotaUsed    float64        `gorm:"type:decimal(10,4);default:0" json:"quota_used"`   // 已用额度（美元）

	// 模型限制
	AllowedModels string        `gorm:"type:text" json:"allowed_models"`                  // 允许的模型（逗号分隔）

	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (up *UserPackage) TableName() string {
	return "user_packages"
}

// QuotaRemaining 剩余额度（仅额度类型）
func (up *UserPackage) QuotaRemaining() float64 {
	if up.Type != "quota" {
		return 0
	}
	return up.QuotaTotal - up.QuotaUsed
}

// ResetPeriodUsageIfNeeded 如果需要，重置周期使用量
func (up *UserPackage) ResetPeriodUsageIfNeeded() bool {
	if up.Type != "subscription" {
		return false
	}

	now := time.Now()
	today := now.Format("2006-01-02")
	year, week := now.ISOWeek()
	thisWeek := now.Format("2006") + "-" + padWeek(week)
	thisMonth := now.Format("2006-01")

	changed := false

	// 重置每日
	if up.LastResetDay != today {
		up.DailyUsed = 0
		up.LastResetDay = today
		changed = true
	}

	// 重置每周
	if up.LastResetWeek != thisWeek {
		up.WeeklyUsed = 0
		up.LastResetWeek = thisWeek
		_ = year // 使用变量
		changed = true
	}

	// 重置每月
	if up.LastResetMonth != thisMonth {
		up.MonthlyUsed = 0
		up.LastResetMonth = thisMonth
		changed = true
	}

	return changed
}

func padWeek(week int) string {
	return fmt.Sprintf("%02d", week)
}

// CanUse 检查是否可以使用指定金额
func (up *UserPackage) CanUse(amount float64) bool {
	if up.Status != "active" {
		return false
	}

	// 检查是否过期
	if up.ExpireTime != nil && time.Now().After(*up.ExpireTime) {
		return false
	}

	if up.Type == "subscription" {
		// 订阅类型检查周期限额
		if up.DailyQuota > 0 && up.DailyUsed+amount > up.DailyQuota {
			return false
		}
		if up.WeeklyQuota > 0 && up.WeeklyUsed+amount > up.WeeklyQuota {
			return false
		}
		if up.MonthlyQuota > 0 && up.MonthlyUsed+amount > up.MonthlyQuota {
			return false
		}
		return true
	} else if up.Type == "quota" {
		// 额度类型检查总额度
		return up.QuotaUsed+amount <= up.QuotaTotal
	}

	return false
}

// RecordUsage 记录使用量
func (up *UserPackage) RecordUsage(amount float64) {
	if up.Type == "subscription" {
		up.DailyUsed += amount
		up.WeeklyUsed += amount
		up.MonthlyUsed += amount
	} else if up.Type == "quota" {
		up.QuotaUsed += amount
	}
}

// IsValid 检查套餐是否有效
func (up *UserPackage) IsValid() bool {
	if up.Status != "active" {
		return false
	}

	// 检查是否过期
	if up.ExpireTime != nil && time.Now().After(*up.ExpireTime) {
		return false
	}

	if up.Type == "quota" {
		// 检查额度是否耗尽
		if up.QuotaUsed >= up.QuotaTotal {
			return false
		}
	}

	return true
}
