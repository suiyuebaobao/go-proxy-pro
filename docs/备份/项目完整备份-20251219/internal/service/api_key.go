package service

import (
	"errors"
	"sync"
	"time"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/pkg/logger"
)

var (
	apiKeyLog     *logger.Logger
	apiKeyLogOnce sync.Once
)

func getAPIKeyLog() *logger.Logger {
	apiKeyLogOnce.Do(func() {
		apiKeyLog = logger.GetLogger("main")
	})
	return apiKeyLog
}

type APIKeyService struct {
	repo            *repository.APIKeyRepository
	userPackageRepo *repository.UserPackageRepository
}

func NewAPIKeyService() *APIKeyService {
	return &APIKeyService{
		repo:            repository.NewAPIKeyRepository(),
		userPackageRepo: repository.NewUserPackageRepository(),
	}
}

// CreateAPIKeyRequest 创建 API Key 请求
type CreateAPIKeyRequest struct {
	Name             string     `json:"name" binding:"required"`
	UserPackageID    uint       `json:"user_package_id" binding:"required"` // 必须绑定用户套餐
	AllowedPlatforms string     `json:"allowed_platforms"`
	AllowedModels    string     `json:"allowed_models"`
	RateLimit        int        `json:"rate_limit"`
	DailyLimit       int        `json:"daily_limit"`
	MonthlyQuota     float64    `json:"monthly_quota"`
	ExpiresAt        *time.Time `json:"expires_at"`
}

// CreateAPIKeyResponse 创建 API Key 响应 (只在创建时返回完整 key)
type CreateAPIKeyResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Key       string `json:"key"`        // 只在创建时返回
	KeyPrefix string `json:"key_prefix"` // 用于显示
}

// Create 创建新的 API Key
func (s *APIKeyService) Create(userID uint, req *CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	getAPIKeyLog().Info("[apikey] 创建 API Key 请求 | UserID: %d | Name: %s | PackageID: %d", userID, req.Name, req.UserPackageID)

	// 验证套餐存在且属于该用户
	userPackage, err := s.userPackageRepo.GetByID(req.UserPackageID)
	if err != nil {
		getAPIKeyLog().Info("[apikey] 创建 API Key 失败 | UserID: %d | 原因: 套餐不存在", userID)
		return nil, errors.New("指定的套餐不存在")
	}
	if userPackage.UserID != userID {
		getAPIKeyLog().Info("[apikey] 创建 API Key 失败 | UserID: %d | 原因: 套餐不属于该用户", userID)
		return nil, errors.New("该套餐不属于当前用户")
	}
	if userPackage.Status != "active" {
		getAPIKeyLog().Info("[apikey] 创建 API Key 失败 | UserID: %d | 原因: 套餐未激活", userID)
		return nil, errors.New("套餐未激活，无法创建 API Key")
	}

	// 从套餐获取计费类型
	billingType := userPackage.Type

	// 生成新的 API Key
	key, hash, prefix, err := model.GenerateAPIKey()
	if err != nil {
		getAPIKeyLog().Error("[apikey] 创建 API Key 失败 | UserID: %d | 原因: 生成 Key 失败: %v", userID, err)
		return nil, errors.New("生成 API Key 失败")
	}

	// 设置默认值
	rateLimit := 60
	if req.RateLimit > 0 {
		rateLimit = req.RateLimit
	}

	allowedPlatforms := "all"
	if req.AllowedPlatforms != "" {
		allowedPlatforms = req.AllowedPlatforms
	}

	packageID := req.UserPackageID
	apiKey := &model.APIKey{
		UserID:           userID,
		Name:             req.Name,
		KeyHash:          hash,
		KeyFull:          key,
		KeyPrefix:        prefix,
		Status:           "active",
		BillingType:      billingType,
		UserPackageID:    &packageID,
		AllowedPlatforms: allowedPlatforms,
		AllowedModels:    req.AllowedModels,
		RateLimit:        rateLimit,
		DailyLimit:       req.DailyLimit,
		MonthlyQuota:     req.MonthlyQuota,
		ExpiresAt:        req.ExpiresAt,
	}

	if err := s.repo.Create(apiKey); err != nil {
		getAPIKeyLog().Error("[apikey] 创建 API Key 失败 | UserID: %d | 原因: 数据库错误: %v", userID, err)
		return nil, err
	}

	getAPIKeyLog().Info("[apikey] 创建 API Key 成功 | UserID: %d | KeyID: %d | Name: %s | PackageID: %d", userID, apiKey.ID, apiKey.Name, req.UserPackageID)

	return &CreateAPIKeyResponse{
		ID:        apiKey.ID,
		Name:      apiKey.Name,
		Key:       key, // 只在创建时返回完整 key
		KeyPrefix: prefix,
	}, nil
}

// List 获取用户的所有 API Key
func (s *APIKeyService) List(userID uint) ([]model.APIKey, error) {
	return s.repo.ListByUserID(userID)
}

// GetByID 根据 ID 获取 API Key
func (s *APIKeyService) GetByID(id uint, userID uint) (*model.APIKey, error) {
	key, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	// 验证所有权
	if key.UserID != userID {
		return nil, errors.New("无权访问此 API Key")
	}
	return key, nil
}

// Delete 删除 API Key
func (s *APIKeyService) Delete(id uint, userID uint) error {
	getAPIKeyLog().Info("[apikey] 删除 API Key 请求 | UserID: %d | KeyID: %d", userID, id)
	key, err := s.repo.GetByID(id)
	if err != nil {
		getAPIKeyLog().Error("[apikey] 删除 API Key 失败 | UserID: %d | KeyID: %d | 原因: 查询失败: %v", userID, id, err)
		return err
	}
	// 验证所有权
	if key.UserID != userID {
		getAPIKeyLog().Info("[apikey] 删除 API Key 失败 | UserID: %d | KeyID: %d | 原因: 无权限", userID, id)
		return errors.New("无权删除此 API Key")
	}
	if err := s.repo.Delete(id); err != nil {
		getAPIKeyLog().Error("[apikey] 删除 API Key 失败 | UserID: %d | KeyID: %d | 原因: 数据库错误: %v", userID, id, err)
		return err
	}
	getAPIKeyLog().Info("[apikey] 删除 API Key 成功 | UserID: %d | KeyID: %d", userID, id)
	return nil
}

// UpdateStatus 更新 API Key 状态
func (s *APIKeyService) UpdateStatus(id uint, userID uint, status string) error {
	key, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	// 验证所有权
	if key.UserID != userID {
		return errors.New("无权修改此 API Key")
	}
	key.Status = status
	return s.repo.Update(key)
}

// ValidateKey 验证 API Key 并返回 Key 信息
func (s *APIKeyService) ValidateKey(keyStr string) (*model.APIKey, error) {
	hash := model.HashAPIKey(keyStr)
	key, err := s.repo.GetByHash(hash)
	if err != nil {
		return nil, errors.New("无效的 API Key")
	}

	if !key.IsActive() {
		if key.Status == "disabled" {
			return nil, errors.New("API Key 已被禁用")
		}
		if key.IsExpired() {
			return nil, errors.New("API Key 已过期")
		}
		return nil, errors.New("API Key 不可用")
	}

	return key, nil
}

// IncrementUsage 记录使用统计
func (s *APIKeyService) IncrementUsage(id uint, tokens int64, cost float64) error {
	return s.repo.IncrementUsage(id, tokens, cost)
}

// UpdateAPIKeyRequest 更新 API Key 请求
type UpdateAPIKeyRequest struct {
	Name             string     `json:"name"`
	AllowedPlatforms string     `json:"allowed_platforms"`
	AllowedModels    string     `json:"allowed_models"`
	RateLimit        int        `json:"rate_limit"`
	DailyLimit       int        `json:"daily_limit"`
	MonthlyQuota     float64    `json:"monthly_quota"`
	ExpiresAt        *time.Time `json:"expires_at"`
	Status           string     `json:"status"`
}

// Update 更新 API Key
func (s *APIKeyService) Update(id uint, userID uint, req *UpdateAPIKeyRequest) (*model.APIKey, error) {
	key, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	// 验证所有权
	if key.UserID != userID {
		return nil, errors.New("无权修改此 API Key")
	}

	// 更新字段
	if req.Name != "" {
		key.Name = req.Name
	}
	if req.AllowedPlatforms != "" {
		key.AllowedPlatforms = req.AllowedPlatforms
	}
	if req.AllowedModels != "" {
		key.AllowedModels = req.AllowedModels
	}
	if req.RateLimit > 0 {
		key.RateLimit = req.RateLimit
	}
	if req.DailyLimit >= 0 {
		key.DailyLimit = req.DailyLimit
	}
	if req.MonthlyQuota >= 0 {
		key.MonthlyQuota = req.MonthlyQuota
	}
	if req.ExpiresAt != nil {
		key.ExpiresAt = req.ExpiresAt
	}
	if req.Status != "" {
		key.Status = req.Status
	}

	if err := s.repo.Update(key); err != nil {
		return nil, err
	}

	return key, nil
}

// ========== 管理员专用方法 ==========

// AdminListByUserID 管理员获取指定用户的 API Key 列表
func (s *APIKeyService) AdminListByUserID(userID uint) ([]model.APIKey, error) {
	return s.repo.ListByUserID(userID)
}

// AdminCreate 管理员为指定用户创建 API Key
func (s *APIKeyService) AdminCreate(userID uint, req *CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	getAPIKeyLog().Info("[apikey] 管理员创建 API Key 请求 | UserID: %d | Name: %s | PackageID: %d", userID, req.Name, req.UserPackageID)

	// 验证套餐存在且属于该用户
	userPackage, err := s.userPackageRepo.GetByID(req.UserPackageID)
	if err != nil {
		getAPIKeyLog().Info("[apikey] 管理员创建 API Key 失败 | UserID: %d | 原因: 套餐不存在", userID)
		return nil, errors.New("指定的套餐不存在")
	}
	if userPackage.UserID != userID {
		getAPIKeyLog().Info("[apikey] 管理员创建 API Key 失败 | UserID: %d | 原因: 套餐不属于该用户", userID)
		return nil, errors.New("该套餐不属于指定用户")
	}

	// 从套餐获取计费类型
	billingType := userPackage.Type

	// 生成新的 API Key
	key, hash, prefix, err := model.GenerateAPIKey()
	if err != nil {
		getAPIKeyLog().Error("[apikey] 管理员创建 API Key 失败 | UserID: %d | 原因: 生成 Key 失败: %v", userID, err)
		return nil, errors.New("生成 API Key 失败")
	}

	// 设置默认值
	rateLimit := 60
	if req.RateLimit > 0 {
		rateLimit = req.RateLimit
	}

	allowedPlatforms := "all"
	if req.AllowedPlatforms != "" {
		allowedPlatforms = req.AllowedPlatforms
	}

	packageID := req.UserPackageID
	apiKey := &model.APIKey{
		UserID:           userID,
		Name:             req.Name,
		KeyHash:          hash,
		KeyFull:          key,
		KeyPrefix:        prefix,
		Status:           "active",
		BillingType:      billingType,
		UserPackageID:    &packageID,
		AllowedPlatforms: allowedPlatforms,
		AllowedModels:    req.AllowedModels,
		RateLimit:        rateLimit,
		DailyLimit:       req.DailyLimit,
		MonthlyQuota:     req.MonthlyQuota,
		ExpiresAt:        req.ExpiresAt,
	}

	if err := s.repo.Create(apiKey); err != nil {
		getAPIKeyLog().Error("[apikey] 管理员创建 API Key 失败 | UserID: %d | 原因: 数据库错误: %v", userID, err)
		return nil, err
	}

	getAPIKeyLog().Info("[apikey] 管理员创建 API Key 成功 | UserID: %d | KeyID: %d | Name: %s | PackageID: %d", userID, apiKey.ID, apiKey.Name, req.UserPackageID)

	return &CreateAPIKeyResponse{
		ID:        apiKey.ID,
		Name:      apiKey.Name,
		Key:       key,
		KeyPrefix: prefix,
	}, nil
}

// AdminDelete 管理员删除 API Key（无需验证所有权）
func (s *APIKeyService) AdminDelete(id uint) error {
	getAPIKeyLog().Info("[apikey] 管理员删除 API Key 请求 | KeyID: %d", id)
	if err := s.repo.Delete(id); err != nil {
		getAPIKeyLog().Error("[apikey] 管理员删除 API Key 失败 | KeyID: %d | 原因: %v", id, err)
		return err
	}
	getAPIKeyLog().Info("[apikey] 管理员删除 API Key 成功 | KeyID: %d", id)
	return nil
}

// AdminToggleStatus 管理员切换 API Key 状态
func (s *APIKeyService) AdminToggleStatus(id uint) (*model.APIKey, error) {
	getAPIKeyLog().Info("[apikey] 管理员切换 API Key 状态请求 | KeyID: %d", id)
	key, err := s.repo.GetByID(id)
	if err != nil {
		getAPIKeyLog().Error("[apikey] 管理员切换 API Key 状态失败 | KeyID: %d | 原因: 查询失败: %v", id, err)
		return nil, err
	}

	oldStatus := key.Status
	if key.Status == "active" {
		key.Status = "disabled"
	} else {
		key.Status = "active"
	}

	if err := s.repo.Update(key); err != nil {
		getAPIKeyLog().Error("[apikey] 管理员切换 API Key 状态失败 | KeyID: %d | 原因: 更新失败: %v", id, err)
		return nil, err
	}

	getAPIKeyLog().Info("[apikey] 管理员切换 API Key 状态成功 | KeyID: %d | %s -> %s", id, oldStatus, key.Status)
	return key, nil
}

// AdminListAll 管理员获取所有 API Key（带用户信息）
func (s *APIKeyService) AdminListAll(page, pageSize int) ([]model.APIKey, int64, error) {
	return s.repo.ListAllWithUser(page, pageSize)
}

// GetAPIKeyLogs 获取 API Key 的使用日志（从 Redis）
func (s *APIKeyService) GetAPIKeyLogs(keyID uint, page, pageSize int) ([]map[string]interface{}, int64, error) {
	return s.repo.GetAPIKeyLogs(keyID, page, pageSize)
}
