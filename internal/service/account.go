/*
 * 文件作用：账户管理服务，处理AI平台账户的业务逻辑
 * 负责功能：
 *   - 账户CRUD操作
 *   - 账户分组管理
 *   - 账户状态更新
 *   - 调度器缓存刷新通知
 * 重要程度：⭐⭐⭐⭐ 重要（账户管理核心）
 * 依赖模块：repository, scheduler, model
 */
package service

import (
	"errors"
	"sync"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/proxy/scheduler"
	"go-aiproxy/internal/repository"
	"go-aiproxy/pkg/logger"
)

var (
	accountLog     *logger.Logger
	accountLogOnce sync.Once
)

func getAccountLog() *logger.Logger {
	accountLogOnce.Do(func() {
		accountLog = logger.GetLogger("main")
	})
	return accountLog
}

type AccountService struct {
	repo      *repository.AccountRepository
	groupRepo *repository.AccountGroupRepository
}

func NewAccountService() *AccountService {
	return &AccountService{
		repo:      repository.NewAccountRepository(),
		groupRepo: repository.NewAccountGroupRepository(),
	}
}

// Account requests

type CreateAccountRequest struct {
	Name               string `json:"name" binding:"required"`
	Type               string `json:"type" binding:"required"`
	Enabled            bool   `json:"enabled"`
	Priority           int    `json:"priority"`
	Weight             int    `json:"weight"`
	MaxConcurrency     int    `json:"max_concurrency"`
	APIKey             string `json:"api_key"`
	APISecret          string `json:"api_secret"`
	AccessToken        string `json:"access_token"`
	RefreshToken       string `json:"refresh_token"`
	SessionKey         string `json:"session_key"`
	OrganizationID     string `json:"organization_id"`
	SubscriptionLevel  string `json:"subscription_level"`
	OpusAccess         bool   `json:"opus_access"`
	AWSAccessKey       string `json:"aws_access_key"`
	AWSSecretKey       string `json:"aws_secret_key"`
	AWSRegion          string `json:"aws_region"`
	AWSSessionToken    string `json:"aws_session_token"`
	AzureEndpoint      string `json:"azure_endpoint"`
	AzureDeploymentName string `json:"azure_deployment_name"`
	AzureAPIVersion    string `json:"azure_api_version"`
	BaseURL            string `json:"base_url"`
	ModelMapping       string `json:"model_mapping"`
	AllowedModels      string `json:"allowed_models"`
	ProxyID            *uint  `json:"proxy_id"`
}

type UpdateAccountRequest struct {
	Name               string `json:"name"`
	Enabled            *bool  `json:"enabled"`
	Priority           *int   `json:"priority"`
	Weight             *int   `json:"weight"`
	MaxConcurrency     *int   `json:"max_concurrency"`
	Status             string `json:"status"`
	APIKey             string `json:"api_key"`
	APISecret          string `json:"api_secret"`
	AccessToken        string `json:"access_token"`
	RefreshToken       string `json:"refresh_token"`
	SessionKey         string `json:"session_key"`
	OrganizationID     string `json:"organization_id"`
	SubscriptionLevel  string `json:"subscription_level"`
	OpusAccess         *bool  `json:"opus_access"`
	AWSAccessKey       string `json:"aws_access_key"`
	AWSSecretKey       string `json:"aws_secret_key"`
	AWSRegion          string `json:"aws_region"`
	AWSSessionToken    string `json:"aws_session_token"`
	AzureEndpoint      string `json:"azure_endpoint"`
	AzureDeploymentName string `json:"azure_deployment_name"`
	AzureAPIVersion    string `json:"azure_api_version"`
	BaseURL            string `json:"base_url"`
	ModelMapping       string `json:"model_mapping"`
	AllowedModels      string `json:"allowed_models"`
	ProxyID            *uint  `json:"proxy_id"`
	ClearProxy         bool   `json:"clear_proxy"`         // 是否清除代理（设置为 true 时清空 proxy_id）
	ClearModelMapping  bool   `json:"clear_model_mapping"` // 是否清除模型映射
	ClearAllowedModels bool   `json:"clear_allowed_models"` // 是否清除允许的模型列表
}

// Account operations

func (s *AccountService) Create(req *CreateAccountRequest) (*model.Account, error) {
	getAccountLog().Info("[account] 创建账户请求 | Name: %s | Type: %s | Platform: %s", req.Name, req.Type, model.GetPlatformByType(req.Type))

	// 验证账户类型
	platform := model.GetPlatformByType(req.Type)
	if platform == "" {
		getAccountLog().Info("[account] 创建账户失败 | Name: %s | 原因: 无效的账户类型", req.Name)
		return nil, errors.New("invalid account type")
	}

	account := &model.Account{
		Name:               req.Name,
		Type:               req.Type,
		Platform:           platform,
		Status:             model.AccountStatusValid,
		Enabled:            req.Enabled,
		Priority:           req.Priority,
		Weight:             req.Weight,
		MaxConcurrency:     req.MaxConcurrency,
		APIKey:             req.APIKey,
		APISecret:          req.APISecret,
		AccessToken:        req.AccessToken,
		RefreshToken:       req.RefreshToken,
		SessionKey:         req.SessionKey,
		OrganizationID:     req.OrganizationID,
		SubscriptionLevel:  req.SubscriptionLevel,
		OpusAccess:         req.OpusAccess,
		AWSAccessKey:     req.AWSAccessKey,
		AWSSecretKey:     req.AWSSecretKey,
		AWSRegion:          req.AWSRegion,
		AWSSessionToken:    req.AWSSessionToken,
		AzureEndpoint:      req.AzureEndpoint,
		AzureDeploymentName: req.AzureDeploymentName,
		AzureAPIVersion:    req.AzureAPIVersion,
		BaseURL:            req.BaseURL,
		ModelMapping:       req.ModelMapping,
		AllowedModels:      req.AllowedModels,
		ProxyID:            req.ProxyID,
	}

	if account.Priority == 0 {
		account.Priority = 50
	}
	if account.Weight == 0 {
		account.Weight = 100
	}
	if account.MaxConcurrency == 0 {
		account.MaxConcurrency = 5 // 默认并发限制
	}

	if err := s.repo.Create(account); err != nil {
		getAccountLog().Error("[account] 创建账户失败 | Name: %s | 原因: %v", req.Name, err)
		return nil, err
	}

	// 刷新调度器缓存
	scheduler.GetScheduler().Refresh()

	getAccountLog().Info("[account] 创建账户成功 | AccountID: %d | Name: %s | Type: %s", account.ID, account.Name, account.Type)
	return account, nil
}

func (s *AccountService) GetByID(id uint) (*model.Account, error) {
	return s.repo.GetByID(id)
}

func (s *AccountService) Update(id uint, req *UpdateAccountRequest) (*model.Account, error) {
	account, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		account.Name = req.Name
	}
	if req.Enabled != nil {
		account.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		account.Priority = *req.Priority
	}
	if req.Weight != nil {
		account.Weight = *req.Weight
	}
	if req.MaxConcurrency != nil {
		account.MaxConcurrency = *req.MaxConcurrency
	}
	if req.Status != "" {
		account.Status = req.Status
	}
	if req.APIKey != "" {
		account.APIKey = req.APIKey
	}
	if req.APISecret != "" {
		account.APISecret = req.APISecret
	}
	if req.AccessToken != "" {
		account.AccessToken = req.AccessToken
	}
	if req.RefreshToken != "" {
		account.RefreshToken = req.RefreshToken
	}
	if req.SessionKey != "" {
		account.SessionKey = req.SessionKey
	}
	if req.OrganizationID != "" {
		account.OrganizationID = req.OrganizationID
	}
	if req.SubscriptionLevel != "" {
		account.SubscriptionLevel = req.SubscriptionLevel
	}
	if req.OpusAccess != nil {
		account.OpusAccess = *req.OpusAccess
	}
	if req.AWSAccessKey != "" {
		account.AWSAccessKey = req.AWSAccessKey
	}
	if req.AWSSecretKey != "" {
		account.AWSSecretKey = req.AWSSecretKey
	}
	if req.AWSRegion != "" {
		account.AWSRegion = req.AWSRegion
	}
	if req.AWSSessionToken != "" {
		account.AWSSessionToken = req.AWSSessionToken
	}
	if req.AzureEndpoint != "" {
		account.AzureEndpoint = req.AzureEndpoint
	}
	if req.AzureDeploymentName != "" {
		account.AzureDeploymentName = req.AzureDeploymentName
	}
	if req.AzureAPIVersion != "" {
		account.AzureAPIVersion = req.AzureAPIVersion
	}
	if req.BaseURL != "" {
		account.BaseURL = req.BaseURL
	}
	if req.ModelMapping != "" {
		account.ModelMapping = req.ModelMapping
	} else if req.ClearModelMapping {
		account.ModelMapping = ""
	}
	if req.AllowedModels != "" {
		account.AllowedModels = req.AllowedModels
	} else if req.ClearAllowedModels {
		account.AllowedModels = ""
	}
	// 处理代理：ClearProxy 优先级高于 ProxyID
	clearProxyAfterUpdate := false
	if req.ClearProxy {
		clearProxyAfterUpdate = true
		account.ProxyID = nil
	} else if req.ProxyID != nil {
		account.ProxyID = req.ProxyID
	}

	if err := s.repo.Update(account); err != nil {
		return nil, err
	}

	// 如果需要清除代理，在 Update 之后单独执行（避免被 Update 覆盖）
	if clearProxyAfterUpdate {
		if err := s.repo.ClearProxyID(id); err != nil {
			return nil, err
		}
		account.ProxyID = nil
	}

	// 刷新调度器缓存
	scheduler.GetScheduler().Refresh()

	return account, nil
}

func (s *AccountService) Delete(id uint) error {
	getAccountLog().Info("[account] 删除账户请求 | AccountID: %d", id)
	if err := s.repo.Delete(id); err != nil {
		getAccountLog().Error("[account] 删除账户失败 | AccountID: %d | 原因: %v", id, err)
		return err
	}

	// 刷新调度器缓存
	scheduler.GetScheduler().Refresh()

	getAccountLog().Info("[account] 删除账户成功 | AccountID: %d", id)
	return nil
}

func (s *AccountService) List(page, pageSize int, platform, status string) ([]model.Account, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(page, pageSize, platform, status)
}

func (s *AccountService) UpdateStatus(id uint, status, lastError string) error {
	getAccountLog().Info("[account] 更新账户状态 | AccountID: %d | Status: %s | LastError: %s", id, status, lastError)
	if err := s.repo.UpdateStatus(id, status, lastError); err != nil {
		getAccountLog().Error("[account] 更新账户状态失败 | AccountID: %d | 原因: %v", id, err)
		return err
	}
	getAccountLog().Info("[account] 更新账户状态成功 | AccountID: %d | Status: %s", id, status)
	return nil
}

func (s *AccountService) GetByPlatform(platform string) ([]model.Account, error) {
	return s.repo.GetByPlatform(platform)
}

// AccountGroup operations

type CreateGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Platform    string `json:"platform"`
	IsDefault   bool   `json:"is_default"`
}

type UpdateGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Platform    string `json:"platform"`
	IsDefault   *bool  `json:"is_default"`
}

func (s *AccountService) CreateGroup(req *CreateGroupRequest) (*model.AccountGroup, error) {
	group := &model.AccountGroup{
		Name:        req.Name,
		Description: req.Description,
		Platform:    req.Platform,
		IsDefault:   req.IsDefault,
	}

	if err := s.groupRepo.Create(group); err != nil {
		return nil, err
	}

	return group, nil
}

func (s *AccountService) GetGroupByID(id uint) (*model.AccountGroup, error) {
	return s.groupRepo.GetByID(id)
}

func (s *AccountService) UpdateGroup(id uint, req *UpdateGroupRequest) (*model.AccountGroup, error) {
	group, err := s.groupRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		group.Name = req.Name
	}
	if req.Description != "" {
		group.Description = req.Description
	}
	if req.Platform != "" {
		group.Platform = req.Platform
	}
	if req.IsDefault != nil {
		group.IsDefault = *req.IsDefault
	}

	if err := s.groupRepo.Update(group); err != nil {
		return nil, err
	}

	return group, nil
}

func (s *AccountService) DeleteGroup(id uint) error {
	return s.groupRepo.Delete(id)
}

func (s *AccountService) ListGroups(page, pageSize int) ([]model.AccountGroup, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.groupRepo.List(page, pageSize)
}

func (s *AccountService) GetAllGroups() ([]model.AccountGroup, error) {
	return s.groupRepo.GetAll()
}

func (s *AccountService) AddAccountToGroup(groupID, accountID uint) error {
	return s.groupRepo.AddAccount(groupID, accountID)
}

func (s *AccountService) RemoveAccountFromGroup(groupID, accountID uint) error {
	return s.groupRepo.RemoveAccount(groupID, accountID)
}
