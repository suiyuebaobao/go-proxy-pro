/*
 * 文件作用：AI模型数据仓库，提供模型配置的数据库操作
 * 负责功能：
 *   - AI模型CRUD操作
 *   - 按平台/别名查询
 *   - 默认模型初始化
 *   - 模型平台映射生成
 * 重要程度：⭐⭐⭐ 一般（AI模型仓库）
 * 依赖模块：model, gorm
 */
package repository

import (
	"strings"

	"go-aiproxy/internal/model"

	"gorm.io/gorm"
)

type AIModelRepository struct {
	db *gorm.DB
}

func NewAIModelRepository(db *gorm.DB) *AIModelRepository {
	return &AIModelRepository{db: db}
}

// List 获取模型列表
func (r *AIModelRepository) List(platform string, enabled *bool) ([]model.AIModel, error) {
	var models []model.AIModel
	query := r.db.Order("sort_order ASC, id ASC")

	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	err := query.Find(&models).Error
	return models, err
}

// GetByID 根据 ID 获取模型
func (r *AIModelRepository) GetByID(id uint) (*model.AIModel, error) {
	var m model.AIModel
	err := r.db.First(&m, id).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetByName 根据名称获取模型
func (r *AIModelRepository) GetByName(name string) (*model.AIModel, error) {
	var m model.AIModel
	err := r.db.Where("name = ?", name).First(&m).Error
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// Create 创建模型
func (r *AIModelRepository) Create(m *model.AIModel) error {
	return r.db.Create(m).Error
}

// Update 更新模型
func (r *AIModelRepository) Update(m *model.AIModel) error {
	return r.db.Save(m).Error
}

// Delete 删除模型
func (r *AIModelRepository) Delete(id uint) error {
	return r.db.Delete(&model.AIModel{}, id).Error
}

// BatchCreate 批量创建模型
func (r *AIModelRepository) BatchCreate(models []model.AIModel) error {
	return r.db.CreateInBatches(models, 100).Error
}

// InitDefaultModels 初始化默认模型
func (r *AIModelRepository) InitDefaultModels() error {
	var count int64
	r.db.Model(&model.AIModel{}).Count(&count)
	if count > 0 {
		return nil // 已有数据，不初始化
	}
	return r.BatchCreate(model.DefaultModels)
}

// ResetDefaultModels 重置为默认模型（删除所有现有模型并重新创建）
func (r *AIModelRepository) ResetDefaultModels() error {
	// 删除所有模型（硬删除）
	if err := r.db.Unscoped().Where("1=1").Delete(&model.AIModel{}).Error; err != nil {
		return err
	}
	// 创建默认模型
	return r.BatchCreate(model.DefaultModels)
}

// GetPlatforms 获取所有平台
func (r *AIModelRepository) GetPlatforms() ([]string, error) {
	var platforms []string
	err := r.db.Model(&model.AIModel{}).Distinct("platform").Pluck("platform", &platforms).Error
	return platforms, err
}

// GetAllModelPlatformMappings 获取所有模型->平台的映射（包括别名）
// 返回 map[modelName]platform，包含主名称和别名
func (r *AIModelRepository) GetAllModelPlatformMappings() (map[string]string, error) {
	var models []model.AIModel
	err := r.db.Where("enabled = ?", true).Find(&models).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, m := range models {
		// 主名称
		result[m.Name] = m.Platform

		// 别名
		if m.Aliases != "" {
			aliases := splitAliases(m.Aliases)
			for _, alias := range aliases {
				if alias != "" {
					result[alias] = m.Platform
				}
			}
		}
	}
	return result, nil
}

// FindPlatformByModelName 根据模型名称查找平台（支持别名和前缀匹配）
func (r *AIModelRepository) FindPlatformByModelName(modelName string) (string, error) {
	// 精确匹配主名称
	var m model.AIModel
	err := r.db.Where("name = ? AND enabled = ?", modelName, true).First(&m).Error
	if err == nil {
		return m.Platform, nil
	}

	// 模糊匹配别名（aliases 字段包含逗号分隔的别名）
	var models []model.AIModel
	err = r.db.Where("enabled = ?", true).Find(&models).Error
	if err != nil {
		return "", err
	}

	for _, m := range models {
		if m.Aliases != "" {
			aliases := splitAliases(m.Aliases)
			for _, alias := range aliases {
				if alias == modelName {
					return m.Platform, nil
				}
			}
		}
	}

	return "", nil // 未找到
}

// splitAliases 分割别名字符串
func splitAliases(aliases string) []string {
	var result []string
	for _, s := range strings.Split(aliases, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}
