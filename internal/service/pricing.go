/*
 * 文件作用：定价服务，计算API调用费用
 * 负责功能：
 *   - Token费用计算
 *   - 模型价格查询
 *   - 缓存Token特殊定价
 *   - 费率倍率应用
 *   - 费用明细分解
 * 重要程度：⭐⭐⭐⭐ 重要（计费核心）
 * 依赖模块：repository, model
 */
package service

import (
	"context"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"

	"gorm.io/gorm"
)

// PricingService 定价服务
type PricingService struct {
	db *gorm.DB
}

func NewPricingService() *PricingService {
	return &PricingService{
		db: repository.DB,
	}
}

// TokenUsage Token 使用量
type TokenUsage struct {
	InputTokens              int
	OutputTokens             int
	CacheCreationInputTokens int
	CacheReadInputTokens     int
}

// CostBreakdown 费用明细
type CostBreakdown struct {
	InputCost       float64 `json:"input_cost"`        // 输入费用
	OutputCost      float64 `json:"output_cost"`       // 输出费用
	CacheCreateCost float64 `json:"cache_create_cost"` // 缓存创建费用
	CacheReadCost   float64 `json:"cache_read_cost"`   // 缓存读取费用
	TotalCost       float64 `json:"total_cost"`        // 总费用（已计算倍率）
	BaseCost        float64 `json:"base_cost"`         // 基础费用（未计算倍率）
	PriceRate       float64 `json:"price_rate"`        // 使用的费率倍率
}

// GetModelPricing 获取模型定价
func (s *PricingService) GetModelPricing(ctx context.Context, modelName string) (*model.AIModel, error) {
	var aiModel model.AIModel
	err := s.db.WithContext(ctx).Where("name = ? OR aliases LIKE ?", modelName, "%"+modelName+"%").First(&aiModel).Error
	if err != nil {
		return nil, err
	}
	return &aiModel, nil
}

// IsModelEnabled 检查模型是否启用
// 返回值: enabled, exists, error
func (s *PricingService) IsModelEnabled(ctx context.Context, modelName string) (bool, bool, error) {
	var aiModel model.AIModel
	err := s.db.WithContext(ctx).Where("name = ? OR aliases LIKE ?", modelName, "%"+modelName+"%").First(&aiModel).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 模型不存在，默认允许使用（向后兼容）
			return true, false, nil
		}
		return false, false, err
	}
	return aiModel.Enabled, true, nil
}

// CalculateCost 计算请求费用
// modelName: 模型名称
// usage: Token 使用量
// priceRate: 用户费率倍率（1.0 = 原价，1.5 = 1.5倍，0 = 免费）
func (s *PricingService) CalculateCost(ctx context.Context, modelName string, usage *TokenUsage, priceRate float64) (*CostBreakdown, error) {
	// 获取模型定价
	aiModel, err := s.GetModelPricing(ctx, modelName)
	if err != nil {
		// 如果找不到模型定价，返回零费用
		return &CostBreakdown{
			PriceRate: priceRate,
		}, nil
	}

	return s.CalculateCostWithModel(aiModel, usage, priceRate), nil
}

// CalculateCostWithModel 使用已有的模型定价计算费用
func (s *PricingService) CalculateCostWithModel(aiModel *model.AIModel, usage *TokenUsage, priceRate float64) *CostBreakdown {
	// 费率倍率为0表示免费
	if priceRate == 0 {
		return &CostBreakdown{
			PriceRate: 0,
		}
	}

	// 计算基础费用（价格单位是 $/1M tokens）
	inputCost := float64(usage.InputTokens) * aiModel.InputPrice / 1000000
	outputCost := float64(usage.OutputTokens) * aiModel.OutputPrice / 1000000
	cacheCreateCost := float64(usage.CacheCreationInputTokens) * aiModel.CacheCreatePrice / 1000000
	cacheReadCost := float64(usage.CacheReadInputTokens) * aiModel.CacheReadPrice / 1000000

	baseCost := inputCost + outputCost + cacheCreateCost + cacheReadCost

	// 应用费率倍率
	finalInputCost := inputCost * priceRate
	finalOutputCost := outputCost * priceRate
	finalCacheCreateCost := cacheCreateCost * priceRate
	finalCacheReadCost := cacheReadCost * priceRate
	totalCost := baseCost * priceRate

	return &CostBreakdown{
		InputCost:       finalInputCost,
		OutputCost:      finalOutputCost,
		CacheCreateCost: finalCacheCreateCost,
		CacheReadCost:   finalCacheReadCost,
		TotalCost:       totalCost,
		BaseCost:        baseCost,
		PriceRate:       priceRate,
	}
}

// GetAllModels 获取所有模型定价
func (s *PricingService) GetAllModels(ctx context.Context) ([]model.AIModel, error) {
	var models []model.AIModel
	err := s.db.WithContext(ctx).Where("enabled = ?", true).Order("sort_order").Find(&models).Error
	return models, err
}

// UpdateModelPricing 更新模型定价（管理员）
func (s *PricingService) UpdateModelPricing(ctx context.Context, modelID uint, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&model.AIModel{}).Where("id = ?", modelID).Updates(updates).Error
}

// CreateModel 创建模型（管理员）
func (s *PricingService) CreateModel(ctx context.Context, aiModel *model.AIModel) error {
	return s.db.WithContext(ctx).Create(aiModel).Error
}

// DeleteModel 删除模型（管理员）
func (s *PricingService) DeleteModel(ctx context.Context, modelID uint) error {
	return s.db.WithContext(ctx).Delete(&model.AIModel{}, modelID).Error
}

// GetModelByID 根据 ID 获取模型
func (s *PricingService) GetModelByID(ctx context.Context, modelID uint) (*model.AIModel, error) {
	var aiModel model.AIModel
	err := s.db.WithContext(ctx).First(&aiModel, modelID).Error
	return &aiModel, err
}

// BatchUpdateModels 批量更新模型定价
func (s *PricingService) BatchUpdateModels(ctx context.Context, updates []struct {
	ID      uint
	Updates map[string]interface{}
}) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, u := range updates {
			if err := tx.Model(&model.AIModel{}).Where("id = ?", u.ID).Updates(u.Updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
