/*
 * 文件作用：模型映射数据仓库，提供模型名称映射的数据库操作
 * 负责功能：
 *   - 模型映射CRUD操作
 *   - 按源模型名/优先级查询
 *   - 映射重复性检查
 *   - 全局映射表生成
 * 重要程度：⭐⭐⭐ 一般（模型映射仓库）
 * 依赖模块：model, gorm
 */
package repository

import (
	"go-aiproxy/internal/model"

	"gorm.io/gorm"
)

// ModelMappingRepository 模型映射数据访问层
type ModelMappingRepository struct {
	db *gorm.DB
}

// NewModelMappingRepository 创建模型映射仓库实例
func NewModelMappingRepository() *ModelMappingRepository {
	return &ModelMappingRepository{db: DB}
}

// Create 创建模型映射
// 如果存在已软删除的同名记录，则恢复并更新
func (r *ModelMappingRepository) Create(mapping *model.ModelMapping) error {
	// 检查是否存在已软删除的同名记录
	var existing model.ModelMapping
	err := r.db.Unscoped().Where("source_model = ? AND deleted_at IS NOT NULL", mapping.SourceModel).First(&existing).Error
	if err == nil {
		// 存在软删除记录，恢复并更新
		existing.TargetModel = mapping.TargetModel
		existing.Enabled = mapping.Enabled
		existing.Priority = mapping.Priority
		existing.Description = mapping.Description
		existing.DeletedAt = gorm.DeletedAt{} // 清除删除标记
		return r.db.Unscoped().Save(&existing).Error
	}
	// 不存在软删除记录，正常创建
	return r.db.Create(mapping).Error
}

// GetByID 根据ID获取模型映射
func (r *ModelMappingRepository) GetByID(id uint) (*model.ModelMapping, error) {
	var mapping model.ModelMapping
	err := r.db.First(&mapping, id).Error
	if err != nil {
		return nil, err
	}
	return &mapping, nil
}

// GetBySourceModel 根据源模型名获取映射
func (r *ModelMappingRepository) GetBySourceModel(sourceModel string) (*model.ModelMapping, error) {
	var mapping model.ModelMapping
	err := r.db.Where("source_model = ? AND enabled = ?", sourceModel, true).
		Order("priority DESC").
		First(&mapping).Error
	if err != nil {
		return nil, err
	}
	return &mapping, nil
}

// List 获取所有模型映射
func (r *ModelMappingRepository) List() ([]model.ModelMapping, error) {
	var mappings []model.ModelMapping
	err := r.db.Order("priority DESC, id ASC").Find(&mappings).Error
	return mappings, err
}

// ListEnabled 获取所有启用的模型映射
func (r *ModelMappingRepository) ListEnabled() ([]model.ModelMapping, error) {
	var mappings []model.ModelMapping
	err := r.db.Where("enabled = ?", true).
		Order("priority DESC, id ASC").
		Find(&mappings).Error
	return mappings, err
}

// Update 更新模型映射
func (r *ModelMappingRepository) Update(mapping *model.ModelMapping) error {
	return r.db.Save(mapping).Error
}

// Delete 删除模型映射
func (r *ModelMappingRepository) Delete(id uint) error {
	return r.db.Delete(&model.ModelMapping{}, id).Error
}

// ExistsBySourceModel 检查源模型名是否已存在
func (r *ModelMappingRepository) ExistsBySourceModel(sourceModel string, excludeID uint) (bool, error) {
	var count int64
	query := r.db.Model(&model.ModelMapping{}).Where("source_model = ?", sourceModel)
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

// ExistsBySourceAndTarget 检查源模型+目标模型组合是否已存在
func (r *ModelMappingRepository) ExistsBySourceAndTarget(sourceModel, targetModel string, excludeID uint) (bool, error) {
	var count int64
	query := r.db.Model(&model.ModelMapping{}).Where("source_model = ? AND target_model = ?", sourceModel, targetModel)
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

// GetAllMappingsMap 获取所有启用的映射，返回 map[源模型]目标模型
// 当同一源模型有多个映射时，使用优先级最高的那个
func (r *ModelMappingRepository) GetAllMappingsMap() (map[string]string, error) {
	mappings, err := r.ListEnabled()
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, m := range mappings {
		// 只保留第一个（优先级最高的），后续同源模型的映射跳过
		if _, exists := result[m.SourceModel]; !exists {
			result[m.SourceModel] = m.TargetModel
		}
	}
	return result, nil
}
