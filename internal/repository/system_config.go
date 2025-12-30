/*
 * 文件作用：系统配置数据仓库，提供配置项的数据库操作
 * 负责功能：
 *   - 配置项CRUD操作
 *   - 按Key/分类查询配置
 *   - 批量更新配置
 *   - 配置缓存管理
 * 重要程度：⭐⭐⭐⭐ 重要（系统配置核心仓库）
 * 依赖模块：model, gorm
 */
package repository

import (
	"go-aiproxy/internal/model"

	"gorm.io/gorm"
)

type SystemConfigRepository struct {
	db *gorm.DB
}

func NewSystemConfigRepository() *SystemConfigRepository {
	return &SystemConfigRepository{db: DB}
}

// GetByKey 根据 key 获取配置
func (r *SystemConfigRepository) GetByKey(key string) (*model.SystemConfig, error) {
	var config model.SystemConfig
	err := r.db.Where("config_key = ?", key).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// GetByCategory 根据分类获取配置列表
func (r *SystemConfigRepository) GetByCategory(category string) ([]model.SystemConfig, error) {
	var configs []model.SystemConfig
	err := r.db.Where("category = ?", category).Find(&configs).Error
	return configs, err
}

// GetAll 获取所有配置
func (r *SystemConfigRepository) GetAll() ([]model.SystemConfig, error) {
	var configs []model.SystemConfig
	err := r.db.Order("category, config_key").Find(&configs).Error
	return configs, err
}

// Update 更新配置
func (r *SystemConfigRepository) Update(key, value string) error {
	return r.db.Model(&model.SystemConfig{}).Where("config_key = ?", key).Update("value", value).Error
}

// BatchUpdate 批量更新配置
func (r *SystemConfigRepository) BatchUpdate(configs map[string]string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for key, value := range configs {
			if err := tx.Model(&model.SystemConfig{}).Where("config_key = ?", key).Update("value", value).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Create 创建配置
func (r *SystemConfigRepository) Create(config *model.SystemConfig) error {
	return r.db.Create(config).Error
}

// Delete 删除配置
func (r *SystemConfigRepository) Delete(key string) error {
	return r.db.Where("config_key = ?", key).Delete(&model.SystemConfig{}).Error
}
