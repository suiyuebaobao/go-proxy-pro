/*
 * 文件作用：错误规则数据仓库，提供上游错误匹配规则的数据库操作
 * 负责功能：
 *   - 错误规则CRUD操作
 *   - 按状态码/优先级查询
 *   - 批量规则管理
 *   - 默认规则初始化
 * 重要程度：⭐⭐⭐ 一般（错误规则仓库）
 * 依赖模块：model, gorm
 */
package repository

import (
	"go-aiproxy/internal/model"

	"gorm.io/gorm"
)

type ErrorRuleRepository struct {
	db *gorm.DB
}

func NewErrorRuleRepository() *ErrorRuleRepository {
	return &ErrorRuleRepository{db: DB}
}

// Create 创建规则
func (r *ErrorRuleRepository) Create(rule *model.ErrorRule) error {
	return r.db.Create(rule).Error
}

// GetByID 根据ID获取规则
func (r *ErrorRuleRepository) GetByID(id uint) (*model.ErrorRule, error) {
	var rule model.ErrorRule
	err := r.db.First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// Update 更新规则
func (r *ErrorRuleRepository) Update(rule *model.ErrorRule) error {
	return r.db.Save(rule).Error
}

// Delete 删除规则
func (r *ErrorRuleRepository) Delete(id uint) error {
	return r.db.Delete(&model.ErrorRule{}, id).Error
}

// List 列表查询
func (r *ErrorRuleRepository) List(page, pageSize int) ([]model.ErrorRule, int64, error) {
	var rules []model.ErrorRule
	var total int64

	r.db.Model(&model.ErrorRule{}).Count(&total)

	offset := (page - 1) * pageSize
	err := r.db.Order("priority DESC, id ASC").Offset(offset).Limit(pageSize).Find(&rules).Error
	if err != nil {
		return nil, 0, err
	}

	return rules, total, nil
}

// GetAllEnabled 获取所有启用的规则（按优先级排序）
func (r *ErrorRuleRepository) GetAllEnabled() ([]model.ErrorRule, error) {
	var rules []model.ErrorRule
	err := r.db.Where("enabled = ?", true).Order("priority DESC, id ASC").Find(&rules).Error
	return rules, err
}

// GetByStatusCode 获取指定状态码的规则
func (r *ErrorRuleRepository) GetByStatusCode(statusCode int) ([]model.ErrorRule, error) {
	var rules []model.ErrorRule
	err := r.db.Where("enabled = ? AND (http_status_code = ? OR http_status_code = 0)", true, statusCode).
		Order("priority DESC, id ASC").Find(&rules).Error
	return rules, err
}

// InitDefaultRules 初始化默认规则
func (r *ErrorRuleRepository) InitDefaultRules() error {
	// 检查是否已有规则
	var count int64
	r.db.Model(&model.ErrorRule{}).Count(&count)
	if count > 0 {
		return nil // 已有规则，不再初始化
	}

	// 插入默认规则
	for _, rule := range model.DefaultErrorRules {
		if err := r.db.Create(&rule).Error; err != nil {
			return err
		}
	}
	return nil
}

// BatchCreate 批量创建规则
func (r *ErrorRuleRepository) BatchCreate(rules []model.ErrorRule) error {
	return r.db.CreateInBatches(rules, 100).Error
}

// DeleteAll 删除所有规则
func (r *ErrorRuleRepository) DeleteAll() error {
	return r.db.Where("1 = 1").Delete(&model.ErrorRule{}).Error
}

// ResetToDefault 重置为默认规则
func (r *ErrorRuleRepository) ResetToDefault() error {
	if err := r.DeleteAll(); err != nil {
		return err
	}
	for _, rule := range model.DefaultErrorRules {
		if err := r.db.Create(&rule).Error; err != nil {
			return err
		}
	}
	return nil
}

// EnableAll 启用所有规则
func (r *ErrorRuleRepository) EnableAll() (int64, error) {
	result := r.db.Model(&model.ErrorRule{}).Where("enabled = ?", false).Update("enabled", true)
	return result.RowsAffected, result.Error
}

// DisableAll 禁用所有规则
func (r *ErrorRuleRepository) DisableAll() (int64, error) {
	result := r.db.Model(&model.ErrorRule{}).Where("enabled = ?", true).Update("enabled", false)
	return result.RowsAffected, result.Error
}
