/*
 * 文件作用：会话缓存服务，管理会话绑定和并发控制
 * 负责功能：
 *   - 会话-账户绑定（实现会话粘性）
 *   - 账户并发计数管理
 *   - 账户不可用标记管理
 *   - 用户并发计数管理
 *   - API Key使用量计数
 * 重要程度：⭐⭐⭐⭐ 重要（会话管理核心）
 * 依赖模块：model
 */
package cache

import (
	"context"
	"sync"
	"time"

	"go-aiproxy/internal/model"
)

// SessionCache 会话缓存服务（使用内存缓存）
type SessionCache struct {
	sessionStore       *SessionStore
	concurrencyManager *ConcurrencyManager
	unavailableMarker  *UnavailableMarker
}

var (
	defaultSessionCache *SessionCache
	sessionCacheOnce    sync.Once
)

// GetSessionCache 获取会话缓存单例
func GetSessionCache() *SessionCache {
	sessionCacheOnce.Do(func() {
		defaultSessionCache = &SessionCache{
			sessionStore:       GetSessionStore(),
			concurrencyManager: GetConcurrencyManager(),
			unavailableMarker:  GetUnavailableMarker(),
		}
	})
	return defaultSessionCache
}

// 导出的默认配置
var (
	DefaultTempUnavailableTTL = func() time.Duration { return getUnavailableTTL() }
	DefaultSessionBindingTTL  = func() time.Duration { return getSessionTTL() }
	DefaultConcurrencyLimit   = func() int { return getDefaultConcurrencyMax() }
)

// ==================== 会话绑定 ====================

// SessionBinding 会话绑定信息
type SessionBinding struct {
	SessionID    string    `json:"session_id"`
	AccountID    uint      `json:"account_id"`
	Platform     string    `json:"platform"`
	Model        string    `json:"model,omitempty"`
	UserID       uint      `json:"user_id"`
	APIKeyID     uint      `json:"api_key_id"`
	ClientIP     string    `json:"client_ip"`
	UserAgent    string    `json:"user_agent"`
	BoundAt      time.Time `json:"bound_at"`
	LastUsedAt   time.Time `json:"last_used_at"`
	ExpireAt     time.Time `json:"expire_at"`
	RemainingTTL int64     `json:"remaining_ttl"` // 剩余秒数
}

// GetSessionBinding 获取会话绑定
func (s *SessionCache) GetSessionBinding(ctx context.Context, sessionID string) (*SessionBinding, error) {
	binding := s.sessionStore.Get(sessionID)
	if binding == nil {
		return nil, nil
	}

	remainingTTL := int64(0)
	if time.Now().Before(binding.ExpireAt) {
		remainingTTL = int64(time.Until(binding.ExpireAt).Seconds())
	}

	return &SessionBinding{
		SessionID:    binding.SessionID,
		AccountID:    binding.AccountID,
		Platform:     binding.Platform,
		Model:        binding.Model,
		UserID:       binding.UserID,
		APIKeyID:     binding.APIKeyID,
		ClientIP:     binding.ClientIP,
		UserAgent:    binding.UserAgent,
		BoundAt:      binding.BoundAt,
		LastUsedAt:   binding.LastUsedAt,
		ExpireAt:     binding.ExpireAt,
		RemainingTTL: remainingTTL,
	}, nil
}

// SetSessionBinding 设置会话绑定
func (s *SessionCache) SetSessionBinding(ctx context.Context, binding *SessionBinding) error {
	memBinding := &MemorySessionBinding{
		SessionID:  binding.SessionID,
		AccountID:  binding.AccountID,
		Platform:   binding.Platform,
		Model:      binding.Model,
		UserID:     binding.UserID,
		APIKeyID:   binding.APIKeyID,
		ClientIP:   binding.ClientIP,
		UserAgent:  binding.UserAgent,
		BoundAt:    binding.BoundAt,
		LastUsedAt: binding.LastUsedAt,
	}
	s.sessionStore.Set(memBinding)
	return nil
}

// UpdateSessionLastUsed 更新会话最后使用时间
func (s *SessionCache) UpdateSessionLastUsed(ctx context.Context, sessionID string) error {
	s.sessionStore.UpdateLastUsed(sessionID)
	return nil
}

// RemoveSessionBinding 移除会话绑定
func (s *SessionCache) RemoveSessionBinding(ctx context.Context, sessionID string) error {
	s.sessionStore.Remove(sessionID)
	return nil
}

// GetAccountSessions 获取账户的所有会话
func (s *SessionCache) GetAccountSessions(ctx context.Context, accountID uint) ([]string, error) {
	bindings := s.sessionStore.GetByAccount(accountID)
	sessionIDs := make([]string, len(bindings))
	for i, b := range bindings {
		sessionIDs[i] = b.SessionID
	}
	return sessionIDs, nil
}

// ClearAccountSessions 清除账户的所有会话
func (s *SessionCache) ClearAccountSessions(ctx context.Context, accountID uint) (int64, error) {
	count := s.sessionStore.ClearByAccount(accountID)
	return int64(count), nil
}

// ListAllSessions 列出所有会话绑定
func (s *SessionCache) ListAllSessions(ctx context.Context, offset, limit int64) ([]SessionBinding, int64, error) {
	bindings, total := s.sessionStore.ListAll(int(offset), int(limit))

	now := time.Now()
	result := make([]SessionBinding, len(bindings))
	for i, b := range bindings {
		remainingTTL := int64(0)
		if now.Before(b.ExpireAt) {
			remainingTTL = int64(b.ExpireAt.Sub(now).Seconds())
		}
		result[i] = SessionBinding{
			SessionID:    b.SessionID,
			AccountID:    b.AccountID,
			Platform:     b.Platform,
			Model:        b.Model,
			UserID:       b.UserID,
			APIKeyID:     b.APIKeyID,
			ClientIP:     b.ClientIP,
			UserAgent:    b.UserAgent,
			BoundAt:      b.BoundAt,
			LastUsedAt:   b.LastUsedAt,
			ExpireAt:     b.ExpireAt,
			RemainingTTL: remainingTTL,
		}
	}
	return result, int64(total), nil
}

// ==================== 临时不可用标记 ====================

// MarkAccountUnavailable 标记账户临时不可用
func (s *SessionCache) MarkAccountUnavailable(ctx context.Context, accountID uint, reason string, ttl time.Duration) error {
	s.unavailableMarker.Mark(accountID, reason, ttl)
	return nil
}

// IsAccountUnavailable 检查账户是否临时不可用
func (s *SessionCache) IsAccountUnavailable(ctx context.Context, accountID uint) (bool, string, error) {
	unavailable, reason := s.unavailableMarker.IsUnavailable(accountID)
	return unavailable, reason, nil
}

// ClearAccountUnavailable 清除账户不可用标记
func (s *SessionCache) ClearAccountUnavailable(ctx context.Context, accountID uint) error {
	s.unavailableMarker.Clear(accountID)
	return nil
}

// GetAllUnavailableAccounts 获取所有不可用账户
func (s *SessionCache) GetAllUnavailableAccounts(ctx context.Context) ([]model.UnavailableAccount, error) {
	all := s.unavailableMarker.ListAll()
	result := make([]model.UnavailableAccount, 0, len(all))
	for accountID, info := range all {
		result = append(result, model.UnavailableAccount{
			AccountID:    accountID,
			Reason:       info.Reason,
			RemainingTTL: info.RemainingTTL,
		})
	}
	return result, nil
}

// ==================== 并发控制 ====================

// SetAccountConcurrencyLimit 设置账户并发限制
func (s *SessionCache) SetAccountConcurrencyLimit(accountID uint, limit int) {
	s.concurrencyManager.SetAccountLimit(accountID, limit)
}

// GetAccountConcurrencyLimit 获取账户并发限制
func (s *SessionCache) GetAccountConcurrencyLimit(accountID uint) int {
	return s.concurrencyManager.GetAccountLimit(accountID)
}

// AcquireConcurrency 获取并发槽位（使用默认限制）
func (s *SessionCache) AcquireConcurrency(ctx context.Context, accountID uint) (bool, int64, error) {
	acquired, current := s.concurrencyManager.AcquireAccount(ctx, accountID)
	return acquired, current, nil
}

// AcquireConcurrencyWithLimit 获取并发槽位（指定限制）
func (s *SessionCache) AcquireConcurrencyWithLimit(ctx context.Context, accountID uint, limit int) (bool, int64, error) {
	acquired, current := s.concurrencyManager.AcquireAccountWithLimit(ctx, accountID, limit)
	return acquired, current, nil
}

// ReleaseConcurrency 释放并发槽位
func (s *SessionCache) ReleaseConcurrency(ctx context.Context, accountID uint) error {
	s.concurrencyManager.ReleaseAccount(ctx, accountID)
	return nil
}

// GetAccountConcurrency 获取账户当前并发数
func (s *SessionCache) GetAccountConcurrency(ctx context.Context, accountID uint) (int64, error) {
	return s.concurrencyManager.GetAccountConcurrency(accountID), nil
}

// ResetAccountConcurrency 重置账户并发计数
func (s *SessionCache) ResetAccountConcurrency(ctx context.Context, accountID uint) error {
	s.concurrencyManager.ResetAccountConcurrency(accountID)
	return nil
}

// ==================== 用户并发控制 ====================

// AcquireUserConcurrency 获取用户并发槽位
func (s *SessionCache) AcquireUserConcurrency(ctx context.Context, userID uint, limit int) (bool, int64, error) {
	acquired, current := s.concurrencyManager.AcquireUser(ctx, userID, limit)
	return acquired, current, nil
}

// ReleaseUserConcurrency 释放用户并发槽位
func (s *SessionCache) ReleaseUserConcurrency(ctx context.Context, userID uint) error {
	s.concurrencyManager.ReleaseUser(ctx, userID)
	return nil
}

// GetUserConcurrency 获取用户当前并发数
func (s *SessionCache) GetUserConcurrency(ctx context.Context, userID uint) (int64, error) {
	return s.concurrencyManager.GetUserConcurrency(userID), nil
}

// ResetUserConcurrency 重置用户并发计数
func (s *SessionCache) ResetUserConcurrency(ctx context.Context, userID uint) error {
	s.concurrencyManager.ResetUserConcurrency(userID)
	return nil
}
