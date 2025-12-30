/*
 * 文件作用：模型映射服务，处理模型名称的转换映射
 * 负责功能：
 *   - 模型映射CRUD
 *   - 映射缓存管理
 *   - 模型名转换（源模型->目标模型）
 *   - 缓存刷新
 * 重要程度：⭐⭐⭐ 一般（模型映射功能）
 * 依赖模块：repository, model
 */
package service

import (
	"errors"
	"sync"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/pkg/logger"
)

// ModelMappingService 模型映射服务
type ModelMappingService struct {
	repo  *repository.ModelMappingRepository
	cache map[string]string // 源模型 -> 目标模型 的缓存
	mu    sync.RWMutex
}

var (
	modelMappingServiceInstance *ModelMappingService
	modelMappingServiceOnce     sync.Once
)

// NewModelMappingService 获取模型映射服务单例
func NewModelMappingService() *ModelMappingService {
	modelMappingServiceOnce.Do(func() {
		modelMappingServiceInstance = &ModelMappingService{
			repo:  repository.NewModelMappingRepository(),
			cache: make(map[string]string),
		}
		// 初始化时加载缓存
		modelMappingServiceInstance.RefreshCache()
	})
	return modelMappingServiceInstance
}

// RefreshCache 刷新缓存
func (s *ModelMappingService) RefreshCache() {
	mappings, err := s.repo.GetAllMappingsMap()
	if err != nil {
		logger.Error("刷新模型映射缓存失败: %v", err)
		return
	}

	s.mu.Lock()
	s.cache = mappings
	s.mu.Unlock()

	logger.Info("模型映射缓存已刷新，共 %d 条映射", len(mappings))
}

// MapModel 映射模型名称
// 如果存在映射则返回目标模型，否则返回原模型名
func (s *ModelMappingService) MapModel(sourceModel string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if targetModel, ok := s.cache[sourceModel]; ok {
		logger.Debug("模型映射: %s -> %s", sourceModel, targetModel)
		return targetModel
	}
	return sourceModel
}

// HasMapping 检查是否存在映射
func (s *ModelMappingService) HasMapping(sourceModel string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.cache[sourceModel]
	return ok
}

// Create 创建模型映射
// 允许同一源模型创建多条映射规则（映射到不同目标），供不同账户选择使用
func (s *ModelMappingService) Create(req *model.CreateModelMappingRequest) (*model.ModelMapping, error) {
	// 检查是否存在完全相同的映射（源模型+目标模型都相同）
	exists, err := s.repo.ExistsBySourceAndTarget(req.SourceModel, req.TargetModel, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrModelMappingExists
	}

	mapping := &model.ModelMapping{
		SourceModel: req.SourceModel,
		TargetModel: req.TargetModel,
		Enabled:     true,
		Priority:    req.Priority,
		Description: req.Description,
	}

	if req.Enabled != nil {
		mapping.Enabled = *req.Enabled
	}

	if err := s.repo.Create(mapping); err != nil {
		return nil, err
	}

	// 刷新缓存
	s.RefreshCache()

	return mapping, nil
}

// GetByID 根据ID获取模型映射
func (s *ModelMappingService) GetByID(id uint) (*model.ModelMapping, error) {
	return s.repo.GetByID(id)
}

// List 获取所有模型映射
func (s *ModelMappingService) List() ([]model.ModelMapping, error) {
	return s.repo.List()
}

// Update 更新模型映射
func (s *ModelMappingService) Update(id uint, req *model.UpdateModelMappingRequest) (*model.ModelMapping, error) {
	mapping, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	newSourceModel := mapping.SourceModel
	newTargetModel := mapping.TargetModel

	if req.SourceModel != "" {
		newSourceModel = req.SourceModel
	}
	if req.TargetModel != "" {
		newTargetModel = req.TargetModel
	}

	// 如果源模型或目标模型有变化，检查是否存在完全相同的映射
	if newSourceModel != mapping.SourceModel || newTargetModel != mapping.TargetModel {
		exists, err := s.repo.ExistsBySourceAndTarget(newSourceModel, newTargetModel, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrModelMappingExists
		}
		mapping.SourceModel = newSourceModel
		mapping.TargetModel = newTargetModel
	}

	if req.Enabled != nil {
		mapping.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		mapping.Priority = *req.Priority
	}
	if req.Description != "" {
		mapping.Description = req.Description
	}

	if err := s.repo.Update(mapping); err != nil {
		return nil, err
	}

	// 刷新缓存
	s.RefreshCache()

	return mapping, nil
}

// Delete 删除模型映射
func (s *ModelMappingService) Delete(id uint) error {
	if err := s.repo.Delete(id); err != nil {
		return err
	}

	// 刷新缓存
	s.RefreshCache()

	return nil
}

// ToggleEnabled 切换启用状态
func (s *ModelMappingService) ToggleEnabled(id uint) (*model.ModelMapping, error) {
	mapping, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	mapping.Enabled = !mapping.Enabled

	if err := s.repo.Update(mapping); err != nil {
		return nil, err
	}

	// 刷新缓存
	s.RefreshCache()

	return mapping, nil
}

// GetCacheStats 获取缓存统计
func (s *ModelMappingService) GetCacheStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"count":    len(s.cache),
		"mappings": s.cache,
	}
}

// 错误定义
var (
	ErrModelMappingExists = errors.New("该映射规则已存在（相同的源模型和目标模型组合）")
)
