/*
 * 文件作用：缓存管理服务，提供Redis缓存的高层封装
 * 负责功能：
 *   - 缓存统计信息获取
 *   - 会话缓存管理
 *   - 账户/用户缓存管理
 *   - 并发计数管理
 *   - 不可用账户标记管理
 * 重要程度：⭐⭐⭐⭐ 重要（缓存管理核心）
 * 依赖模块：cache, repository, model
 */
package service

import (
	"context"
	"sync"
	"time"

	"go-aiproxy/internal/cache"
	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
)

// CacheService 缓存管理服务（使用内存缓存）
type CacheService struct {
	sessionCache   *cache.SessionCache
	memoryCache    *cache.MemoryCache
	accountRepo    *repository.AccountRepository
	userRepo       *repository.UserRepository
}

var (
	cacheServiceInstance *CacheService
	cacheServiceOnce     sync.Once
)

func NewCacheService() *CacheService {
	cacheServiceOnce.Do(func() {
		cacheServiceInstance = &CacheService{
			sessionCache: cache.GetSessionCache(),
			memoryCache:  cache.GetMemoryCache(),
			accountRepo:  repository.NewAccountRepository(),
			userRepo:     repository.NewUserRepository(),
		}
	})
	return cacheServiceInstance
}

// 导出的默认配置（兼容旧代码）
var (
	DefaultTempUnavailableTTL = cache.DefaultTempUnavailableTTL
	DefaultSessionBindingTTL  = cache.DefaultSessionBindingTTL
	DefaultConcurrencyLimit   = cache.DefaultConcurrencyLimit
)

// ==================== 粘性会话管理 ====================

// SessionBinding 会话绑定信息（复用 cache 包的定义）
type SessionBinding = cache.SessionBinding

// GetSessionBinding 获取会话绑定
func (s *CacheService) GetSessionBinding(ctx context.Context, sessionID string) (*SessionBinding, error) {
	return s.sessionCache.GetSessionBinding(ctx, sessionID)
}

// SetSessionBinding 设置会话绑定
func (s *CacheService) SetSessionBinding(ctx context.Context, binding *SessionBinding) error {
	return s.sessionCache.SetSessionBinding(ctx, binding)
}

// UpdateSessionLastUsed 更新会话最后使用时间
func (s *CacheService) UpdateSessionLastUsed(ctx context.Context, sessionID string) error {
	return s.sessionCache.UpdateSessionLastUsed(ctx, sessionID)
}

// RemoveSessionBinding 移除会话绑定
func (s *CacheService) RemoveSessionBinding(ctx context.Context, sessionID string) error {
	return s.sessionCache.RemoveSessionBinding(ctx, sessionID)
}

// GetAccountSessions 获取账户的所有会话
func (s *CacheService) GetAccountSessions(ctx context.Context, accountID uint) ([]string, error) {
	return s.sessionCache.GetAccountSessions(ctx, accountID)
}

// ClearAccountSessions 清除账户的所有会话
func (s *CacheService) ClearAccountSessions(ctx context.Context, accountID uint) (int64, error) {
	return s.sessionCache.ClearAccountSessions(ctx, accountID)
}

// ==================== 临时不可用标记 ====================

// MarkAccountUnavailable 标记账户临时不可用
func (s *CacheService) MarkAccountUnavailable(ctx context.Context, accountID uint, reason string, ttl time.Duration) error {
	return s.sessionCache.MarkAccountUnavailable(ctx, accountID, reason, ttl)
}

// IsAccountUnavailable 检查账户是否临时不可用
func (s *CacheService) IsAccountUnavailable(ctx context.Context, accountID uint) (bool, string, error) {
	return s.sessionCache.IsAccountUnavailable(ctx, accountID)
}

// ClearAccountUnavailable 清除账户不可用标记
func (s *CacheService) ClearAccountUnavailable(ctx context.Context, accountID uint) error {
	return s.sessionCache.ClearAccountUnavailable(ctx, accountID)
}

// GetAllUnavailableAccounts 获取所有不可用账户
func (s *CacheService) GetAllUnavailableAccounts(ctx context.Context) ([]model.UnavailableAccount, error) {
	return s.sessionCache.GetAllUnavailableAccounts(ctx)
}

// ==================== 并发控制 ====================

// SetAccountConcurrencyLimit 设置账户并发限制
func (s *CacheService) SetAccountConcurrencyLimit(accountID uint, limit int) {
	s.sessionCache.SetAccountConcurrencyLimit(accountID, limit)
}

// GetAccountConcurrencyLimit 获取账户并发限制
func (s *CacheService) GetAccountConcurrencyLimit(accountID uint) int {
	return s.sessionCache.GetAccountConcurrencyLimit(accountID)
}

// AcquireConcurrency 获取并发槽位
func (s *CacheService) AcquireConcurrency(ctx context.Context, accountID uint) (bool, int64, error) {
	return s.sessionCache.AcquireConcurrency(ctx, accountID)
}

// ReleaseConcurrency 释放并发槽位
func (s *CacheService) ReleaseConcurrency(ctx context.Context, accountID uint) error {
	return s.sessionCache.ReleaseConcurrency(ctx, accountID)
}

// GetAccountConcurrency 获取账户当前并发数
func (s *CacheService) GetAccountConcurrency(ctx context.Context, accountID uint) (int64, error) {
	return s.sessionCache.GetAccountConcurrency(ctx, accountID)
}

// ResetAccountConcurrency 重置账户并发计数
func (s *CacheService) ResetAccountConcurrency(ctx context.Context, accountID uint) error {
	return s.sessionCache.ResetAccountConcurrency(ctx, accountID)
}

// ==================== 用户并发控制 ====================

// GetUserConcurrency 获取用户当前并发数
func (s *CacheService) GetUserConcurrency(ctx context.Context, userID uint) (int64, error) {
	return s.sessionCache.GetUserConcurrency(ctx, userID)
}

// ResetUserConcurrency 重置用户并发计数
func (s *CacheService) ResetUserConcurrency(ctx context.Context, userID uint) error {
	return s.sessionCache.ResetUserConcurrency(ctx, userID)
}

// ==================== 缓存管理统计 ====================

// CacheStats 缓存统计信息
type CacheStats struct {
	SessionCount      int64  `json:"session_count"`
	UnavailableCount  int64  `json:"unavailable_count"`
	UsageKeyCount     int64  `json:"usage_key_count"`    // 不再使用 Redis，保留兼容
	CostKeyCount      int64  `json:"cost_key_count"`     // 不再使用 Redis，保留兼容
	TotalKeyCount     int64  `json:"total_key_count"`    // 内存缓存项数
	MemoryUsed        int64  `json:"memory_used"`        // 不再使用 Redis
	MemoryUsedHuman   string `json:"memory_used_human"`  // 不再使用 Redis
}

// GetCacheStats 获取缓存统计
func (s *CacheService) GetCacheStats(ctx context.Context) (*CacheStats, error) {
	stats := s.memoryCache.Stats()

	sessionCount := int64(0)
	if v, ok := stats["session_count"].(int); ok {
		sessionCount = int64(v)
	}
	unavailableCount := int64(0)
	if v, ok := stats["unavailable_count"].(int); ok {
		unavailableCount = int64(v)
	}

	return &CacheStats{
		SessionCount:     sessionCount,
		UnavailableCount: unavailableCount,
		TotalKeyCount:    sessionCount + unavailableCount,
		MemoryUsedHuman:  "N/A (内存缓存)",
	}, nil
}

// ==================== 手动缓存清理 ====================

// ClearCacheType 缓存类型
type ClearCacheType string

const (
	ClearCacheAll         ClearCacheType = "all"
	ClearCacheSessions    ClearCacheType = "sessions"
	ClearCacheUnavailable ClearCacheType = "unavailable"
	ClearCacheUsage       ClearCacheType = "usage"       // 不再使用，数据在 MySQL
	ClearCacheCost        ClearCacheType = "cost"        // 不再使用，数据在 MySQL
	ClearCacheConcurrency ClearCacheType = "concurrency"
)

// ClearCacheResult 清理结果
type ClearCacheResult struct {
	Type         ClearCacheType `json:"type"`
	DeletedCount int64          `json:"deleted_count"`
	Error        string         `json:"error,omitempty"`
}

// ClearCache 清理指定类型的缓存
func (s *CacheService) ClearCache(ctx context.Context, cacheType ClearCacheType) (*ClearCacheResult, error) {
	result := &ClearCacheResult{Type: cacheType}

	switch cacheType {
	case ClearCacheAll:
		cleared := s.memoryCache.ClearAll()
		result.DeletedCount = int64(cleared["sessions"] + cleared["unavailable"])
		return result, nil

	case ClearCacheSessions:
		count := s.memoryCache.Sessions.ClearAll()
		result.DeletedCount = int64(count)
		return result, nil

	case ClearCacheUnavailable:
		count := s.memoryCache.Unavailable.ClearAll()
		result.DeletedCount = int64(count)
		return result, nil

	case ClearCacheUsage, ClearCacheCost:
		// 使用量和费用数据已经在 MySQL 中，这里不需要清理
		result.DeletedCount = 0
		return result, nil

	case ClearCacheConcurrency:
		// 并发计数器不支持批量清理，需要逐个清理
		// 这里简化处理，不支持批量清理并发计数
		result.DeletedCount = 0
		return result, nil

	default:
		result.Error = "unknown cache type"
		return result, nil
	}
}

// ClearUserCache 清理指定用户的缓存
func (s *CacheService) ClearUserCache(ctx context.Context, userID uint) (*ClearCacheResult, error) {
	result := &ClearCacheResult{Type: "user"}
	count := s.memoryCache.Sessions.ClearByUser(userID)
	result.DeletedCount = int64(count)
	return result, nil
}

// ClearAPIKeyCache 清理指定 API Key 的缓存
func (s *CacheService) ClearAPIKeyCache(ctx context.Context, apiKeyID uint) (*ClearCacheResult, error) {
	// API Key 缓存数据现在在 MySQL，这里只返回空结果
	result := &ClearCacheResult{Type: "apikey"}
	result.DeletedCount = 0
	return result, nil
}

// ==================== 会话列表查询 ====================

// SimpleUserInfo 简单用户信息
type SimpleUserInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// SimpleAccountInfo 简单账号信息
type SimpleAccountInfo struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// AccountCacheInfo 账号缓存信息（聚合）
type AccountCacheInfo struct {
	AccountID      uint             `json:"account_id"`
	AccountName    string           `json:"account_name"`
	Concurrency    int64            `json:"concurrency"`
	MaxConcurrency int              `json:"max_concurrency"`
	SessionCount   int              `json:"session_count"`
	Sessions       []SessionBinding `json:"sessions"`
	Users          []SimpleUserInfo `json:"users"`
}

// UserCacheInfo 用户缓存信息（聚合）
type UserCacheInfo struct {
	UserID         uint               `json:"user_id"`
	UserName       string             `json:"user_name"`
	Concurrency    int64              `json:"concurrency"`
	MaxConcurrency int                `json:"max_concurrency"`
	SessionCount   int                `json:"session_count"`
	Sessions       []SessionBinding   `json:"sessions"`
	Accounts       []SimpleAccountInfo `json:"accounts"`
}

// ListAccountsWithCache 列出所有有缓存的账号（聚合视图）
func (s *CacheService) ListAccountsWithCache(ctx context.Context) ([]AccountCacheInfo, error) {
	sessions, _, err := s.ListAllSessions(ctx, 0, 1000)
	if err != nil {
		return nil, err
	}

	// 收集所有账号ID和用户ID
	accountIDs := make([]uint, 0)
	accountIDSet := make(map[uint]bool)
	userIDs := make([]uint, 0)
	userIDSet := make(map[uint]bool)
	for _, sess := range sessions {
		if !accountIDSet[sess.AccountID] {
			accountIDSet[sess.AccountID] = true
			accountIDs = append(accountIDs, sess.AccountID)
		}
		if sess.UserID > 0 && !userIDSet[sess.UserID] {
			userIDSet[sess.UserID] = true
			userIDs = append(userIDs, sess.UserID)
		}
	}

	// 批量查询账号信息
	accountInfoMap := make(map[uint]*model.Account)
	if len(accountIDs) > 0 {
		accounts, err := s.accountRepo.GetByIDs(accountIDs)
		if err == nil {
			for i := range accounts {
				accountInfoMap[accounts[i].ID] = &accounts[i]
			}
		}
	}

	// 批量查询用户信息（用于显示使用此账号的用户名）
	userInfoMap := make(map[uint]*model.User)
	if len(userIDs) > 0 {
		users, err := s.userRepo.GetByIDs(userIDs)
		if err == nil {
			for i := range users {
				userInfoMap[users[i].ID] = &users[i]
			}
		}
	}

	// 临时存储每个账号的用户ID列表（用于去重）
	accountUserIDs := make(map[uint]map[uint]bool)

	accountMap := make(map[uint]*AccountCacheInfo)
	for _, sess := range sessions {
		if _, ok := accountMap[sess.AccountID]; !ok {
			concurrency, _ := s.GetAccountConcurrency(ctx, sess.AccountID)

			// 从数据库获取账号名称和最大并发数
			accountName := ""
			maxConcurrency := 5 // 默认值
			if acc, ok := accountInfoMap[sess.AccountID]; ok {
				accountName = acc.Name
				maxConcurrency = acc.MaxConcurrency
			}

			accountMap[sess.AccountID] = &AccountCacheInfo{
				AccountID:      sess.AccountID,
				AccountName:    accountName,
				Concurrency:    concurrency,
				MaxConcurrency: maxConcurrency,
				Sessions:       []SessionBinding{},
				Users:          []SimpleUserInfo{},
			}
			accountUserIDs[sess.AccountID] = make(map[uint]bool)
		}
		info := accountMap[sess.AccountID]
		info.Sessions = append(info.Sessions, sess)
		info.SessionCount = len(info.Sessions)

		// 添加用户（去重）
		if sess.UserID > 0 && !accountUserIDs[sess.AccountID][sess.UserID] {
			accountUserIDs[sess.AccountID][sess.UserID] = true
			userName := ""
			if user, ok := userInfoMap[sess.UserID]; ok {
				userName = user.Username
			}
			info.Users = append(info.Users, SimpleUserInfo{
				ID:   sess.UserID,
				Name: userName,
			})
		}
	}

	result := make([]AccountCacheInfo, 0, len(accountMap))
	for _, info := range accountMap {
		result = append(result, *info)
	}
	return result, nil
}

// ListUsersWithCache 列出所有有缓存的用户（聚合视图）
func (s *CacheService) ListUsersWithCache(ctx context.Context) ([]UserCacheInfo, error) {
	sessions, _, err := s.ListAllSessions(ctx, 0, 1000)
	if err != nil {
		return nil, err
	}

	// 收集所有用户ID和账号ID
	userIDs := make([]uint, 0)
	userIDSet := make(map[uint]bool)
	accountIDs := make([]uint, 0)
	accountIDSet := make(map[uint]bool)
	for _, sess := range sessions {
		if sess.UserID > 0 && !userIDSet[sess.UserID] {
			userIDSet[sess.UserID] = true
			userIDs = append(userIDs, sess.UserID)
		}
		if sess.AccountID > 0 && !accountIDSet[sess.AccountID] {
			accountIDSet[sess.AccountID] = true
			accountIDs = append(accountIDs, sess.AccountID)
		}
	}

	// 批量查询用户信息
	userInfoMap := make(map[uint]*model.User)
	if len(userIDs) > 0 {
		users, err := s.userRepo.GetByIDs(userIDs)
		if err == nil {
			for i := range users {
				userInfoMap[users[i].ID] = &users[i]
			}
		}
	}

	// 批量查询账号信息（用于显示用户使用的账号名）
	accountInfoMap := make(map[uint]*model.Account)
	if len(accountIDs) > 0 {
		accounts, err := s.accountRepo.GetByIDs(accountIDs)
		if err == nil {
			for i := range accounts {
				accountInfoMap[accounts[i].ID] = &accounts[i]
			}
		}
	}

	// 临时存储每个用户的账号ID列表（用于去重）
	userAccountIDs := make(map[uint]map[uint]bool)

	userMap := make(map[uint]*UserCacheInfo)
	for _, sess := range sessions {
		if sess.UserID == 0 {
			continue
		}
		if _, ok := userMap[sess.UserID]; !ok {
			concurrency, _ := s.GetUserConcurrency(ctx, sess.UserID)

			// 从数据库获取用户名和最大并发数
			userName := ""
			maxConcurrency := 10 // 默认值
			if user, ok := userInfoMap[sess.UserID]; ok {
				userName = user.Username
				maxConcurrency = user.MaxConcurrency
			}

			userMap[sess.UserID] = &UserCacheInfo{
				UserID:         sess.UserID,
				UserName:       userName,
				Concurrency:    concurrency,
				MaxConcurrency: maxConcurrency,
				Sessions:       []SessionBinding{},
				Accounts:       []SimpleAccountInfo{},
			}
			userAccountIDs[sess.UserID] = make(map[uint]bool)
		}
		info := userMap[sess.UserID]
		info.Sessions = append(info.Sessions, sess)
		info.SessionCount = len(info.Sessions)

		// 添加账号（去重）
		if sess.AccountID > 0 && !userAccountIDs[sess.UserID][sess.AccountID] {
			userAccountIDs[sess.UserID][sess.AccountID] = true
			accountName := ""
			if acc, ok := accountInfoMap[sess.AccountID]; ok {
				accountName = acc.Name
			}
			info.Accounts = append(info.Accounts, SimpleAccountInfo{
				ID:   sess.AccountID,
				Name: accountName,
			})
		}
	}

	result := make([]UserCacheInfo, 0, len(userMap))
	for _, info := range userMap {
		result = append(result, *info)
	}
	return result, nil
}

// ListAllSessions 列出所有会话绑定
func (s *CacheService) ListAllSessions(ctx context.Context, offset, limit int64) ([]SessionBinding, int64, error) {
	return s.sessionCache.ListAllSessions(ctx, offset, limit)
}
