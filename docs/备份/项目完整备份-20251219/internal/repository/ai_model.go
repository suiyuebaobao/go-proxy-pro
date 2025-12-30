package repository

import (
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
