/*
 * 文件作用：告警规则与告警日志的数据访问层
 * 负责功能：
 *   - 告警规则 CRUD 操作
 *   - 按条件类型查询已启用规则
 *   - 告警日志写入与分页查询
 *   - 过期日志清理
 * 重要程度：⭐⭐⭐ 一般（告警子系统数据访问）
 * 依赖模块：model, gorm
 */
package repository

import (
	"go-aiproxy/internal/model"

	"gorm.io/gorm"
)

type AlertRepository struct {
	db *gorm.DB
}

func NewAlertRepository() *AlertRepository {
	return &AlertRepository{db: GetDB()}
}

// ========== AlertRule CRUD ==========

func (r *AlertRepository) CreateRule(rule *model.AlertRule) error {
	return r.db.Create(rule).Error
}

func (r *AlertRepository) UpdateRule(rule *model.AlertRule) error {
	return r.db.Save(rule).Error
}

func (r *AlertRepository) DeleteRule(id uint) error {
	return r.db.Delete(&model.AlertRule{}, id).Error
}

func (r *AlertRepository) GetRuleByID(id uint) (*model.AlertRule, error) {
	var rule model.AlertRule
	err := r.db.First(&rule, id).Error
	return &rule, err
}

func (r *AlertRepository) ListRules() ([]model.AlertRule, error) {
	var rules []model.AlertRule
	err := r.db.Order("created_at DESC").Find(&rules).Error
	return rules, err
}

func (r *AlertRepository) GetEnabledRulesByCondition(conditionType string) ([]model.AlertRule, error) {
	var rules []model.AlertRule
	err := r.db.Where("enabled = ? AND condition_type = ?", true, conditionType).Find(&rules).Error
	return rules, err
}

// ========== AlertLog ==========

func (r *AlertRepository) CreateLog(log *model.AlertLog) error {
	return r.db.Create(log).Error
}

func (r *AlertRepository) ListLogs(page, pageSize int) ([]model.AlertLog, int64, error) {
	var logs []model.AlertLog
	var total int64

	r.db.Model(&model.AlertLog{}).Count(&total)

	offset := (page - 1) * pageSize
	err := r.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error
	return logs, total, err
}

func (r *AlertRepository) CleanupOldLogs(days int) (int64, error) {
	result := r.db.Where("created_at < DATE_SUB(NOW(), INTERVAL ? DAY)", days).Delete(&model.AlertLog{})
	return result.RowsAffected, result.Error
}
