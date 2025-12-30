/*
 * 文件作用：错误规则服务，管理错误匹配和处理规则
 * 负责功能：
 *   - 错误规则CRUD
 *   - 规则缓存刷新
 *   - 默认规则初始化
 *   - 规则启用/禁用
 * 重要程度：⭐⭐⭐ 一般（错误处理增强）
 * 依赖模块：repository, errormatch, model
 */
package service

import (
	"go-aiproxy/internal/errormatch"
	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
)

type ErrorRuleService struct {
	repo *repository.ErrorRuleRepository
}

func NewErrorRuleService() *ErrorRuleService {
	return &ErrorRuleService{
		repo: repository.NewErrorRuleRepository(),
	}
}

// CreateRuleRequest 创建规则请求
type CreateRuleRequest struct {
	HTTPStatusCode int    `json:"http_status_code"`
	Keyword        string `json:"keyword"`
	TargetStatus   string `json:"target_status" binding:"required"`
	Priority       int    `json:"priority"`
	Enabled        bool   `json:"enabled"`
	Description    string `json:"description"`
}

// UpdateRuleRequest 更新规则请求
type UpdateRuleRequest struct {
	HTTPStatusCode *int    `json:"http_status_code"`
	Keyword        *string `json:"keyword"`
	TargetStatus   string  `json:"target_status"`
	Priority       *int    `json:"priority"`
	Enabled        *bool   `json:"enabled"`
	Description    string  `json:"description"`
}

// Create 创建规则
func (s *ErrorRuleService) Create(req *CreateRuleRequest) (*model.ErrorRule, error) {
	rule := &model.ErrorRule{
		HTTPStatusCode: req.HTTPStatusCode,
		Keyword:        req.Keyword,
		TargetStatus:   req.TargetStatus,
		Priority:       req.Priority,
		Enabled:        req.Enabled,
		Description:    req.Description,
	}

	if err := s.repo.Create(rule); err != nil {
		return nil, err
	}

	// 刷新缓存
	errormatch.GetErrorRuleMatcher().Refresh()

	return rule, nil
}

// GetByID 根据ID获取规则
func (s *ErrorRuleService) GetByID(id uint) (*model.ErrorRule, error) {
	return s.repo.GetByID(id)
}

// Update 更新规则
func (s *ErrorRuleService) Update(id uint, req *UpdateRuleRequest) (*model.ErrorRule, error) {
	rule, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.HTTPStatusCode != nil {
		rule.HTTPStatusCode = *req.HTTPStatusCode
	}
	if req.Keyword != nil {
		rule.Keyword = *req.Keyword
	}
	if req.TargetStatus != "" {
		rule.TargetStatus = req.TargetStatus
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.Description != "" {
		rule.Description = req.Description
	}

	if err := s.repo.Update(rule); err != nil {
		return nil, err
	}

	// 刷新缓存
	errormatch.GetErrorRuleMatcher().Refresh()

	return rule, nil
}

// Delete 删除规则
func (s *ErrorRuleService) Delete(id uint) error {
	if err := s.repo.Delete(id); err != nil {
		return err
	}

	// 刷新缓存
	errormatch.GetErrorRuleMatcher().Refresh()

	return nil
}

// List 列表查询
func (s *ErrorRuleService) List(page, pageSize int) ([]model.ErrorRule, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(page, pageSize)
}

// GetAllEnabled 获取所有启用的规则
func (s *ErrorRuleService) GetAllEnabled() ([]model.ErrorRule, error) {
	return s.repo.GetAllEnabled()
}

// InitDefaultRules 初始化默认规则
func (s *ErrorRuleService) InitDefaultRules() error {
	if err := s.repo.InitDefaultRules(); err != nil {
		return err
	}
	// 刷新缓存
	errormatch.GetErrorRuleMatcher().Refresh()
	return nil
}

// ResetToDefault 重置为默认规则
func (s *ErrorRuleService) ResetToDefault() error {
	if err := s.repo.ResetToDefault(); err != nil {
		return err
	}
	// 刷新缓存
	errormatch.GetErrorRuleMatcher().Refresh()
	return nil
}

// EnableAll 启用所有规则
func (s *ErrorRuleService) EnableAll() (int64, error) {
	affected, err := s.repo.EnableAll()
	if err != nil {
		return 0, err
	}
	// 刷新缓存
	errormatch.GetErrorRuleMatcher().Refresh()
	return affected, nil
}

// DisableAll 禁用所有规则
func (s *ErrorRuleService) DisableAll() (int64, error) {
	affected, err := s.repo.DisableAll()
	if err != nil {
		return 0, err
	}
	// 刷新缓存
	errormatch.GetErrorRuleMatcher().Refresh()
	return affected, nil
}
