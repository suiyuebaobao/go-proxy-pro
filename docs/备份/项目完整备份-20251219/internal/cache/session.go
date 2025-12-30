package cache

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-aiproxy/internal/config"
	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"

	"github.com/redis/go-redis/v9"
)

// SessionCache 会话缓存服务
type SessionCache struct {
	rdb *redis.Client
}

var (
	defaultSessionCache *SessionCache
	sessionCacheOnce    sync.Once
)

// GetSessionCache 获取会话缓存单例
func GetSessionCache() *SessionCache {
	sessionCacheOnce.Do(func() {
		defaultSessionCache = &SessionCache{
			rdb: repository.RDB,
		}
	})
	return defaultSessionCache
}

// Redis Key 前缀
const (
	keySessionBinding   = "session:binding:%s" // session:binding:{sessionID} -> accountID
	keySessionByAccount = "session:account:%d" // session:account:{accountID} -> set of sessionIDs
	keyTempUnavailable  = "unavailable:%d"     // unavailable:{accountID}
	keyConcurrency      = "concurrency:%d"     // concurrency:{accountID}
)

// 获取配置的 TTL 值
func getSessionBindingTTL() time.Duration {
	return time.Duration(config.Cfg.Cache.GetSessionTTL()) * time.Minute
}

func getSessionRenewalTTL() time.Duration {
	return time.Duration(config.Cfg.Cache.GetSessionRenewalTTL()) * time.Minute
}

func getUnavailableTTL() time.Duration {
	return time.Duration(config.Cfg.Cache.GetUnavailableTTL()) * time.Minute
}

func getConcurrencyTTL() time.Duration {
	return time.Duration(config.Cfg.Cache.GetConcurrencyTTL()) * time.Minute
}

func getDefaultConcurrencyMax() int {
	return config.Cfg.Cache.GetDefaultConcurrencyMax()
}

// 导出的默认配置
var (
	DefaultTempUnavailableTTL = func() time.Duration { return getUnavailableTTL() }
	DefaultSessionBindingTTL  = func() time.Duration { return getSessionBindingTTL() }
	DefaultConcurrencyLimit   = func() int { return getDefaultConcurrencyMax() }
)

// ==================== 会话绑定 ====================

// SessionBinding 会话绑定信息
type SessionBinding struct {
	SessionID  string    `json:"session_id"`
	AccountID  uint      `json:"account_id"`
	Platform   string    `json:"platform"`
	Model      string    `json:"model,omitempty"`
	UserID     uint      `json:"user_id"`
	APIKeyID   uint      `json:"api_key_id"`
	ClientIP   string    `json:"client_ip"`
	BoundAt    time.Time `json:"bound_at"`
	LastUsedAt time.Time `json:"last_used_at"`
}

// GetSessionBinding 获取会话绑定
func (s *SessionCache) GetSessionBinding(ctx context.Context, sessionID string) (*SessionBinding, error) {
	key := fmt.Sprintf(keySessionBinding, sessionID)

	result, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}

	binding := &SessionBinding{
		SessionID: sessionID,
		Platform:  result["platform"],
		Model:     result["model"],
		ClientIP:  result["client_ip"],
	}

	if v, ok := result["account_id"]; ok {
		id, _ := strconv.ParseUint(v, 10, 32)
		binding.AccountID = uint(id)
	}
	if v, ok := result["user_id"]; ok {
		id, _ := strconv.ParseUint(v, 10, 32)
		binding.UserID = uint(id)
	}
	if v, ok := result["api_key_id"]; ok {
		id, _ := strconv.ParseUint(v, 10, 32)
		binding.APIKeyID = uint(id)
	}
	if v, ok := result["bound_at"]; ok {
		ts, _ := strconv.ParseInt(v, 10, 64)
		binding.BoundAt = time.Unix(ts, 0)
	}
	if v, ok := result["last_used_at"]; ok {
		ts, _ := strconv.ParseInt(v, 10, 64)
		binding.LastUsedAt = time.Unix(ts, 0)
	}

	// 智能续期：剩余时间不足阈值时续期
	ttl, err := s.rdb.TTL(ctx, key).Result()
	if err == nil && ttl > 0 && ttl < getSessionRenewalTTL() {
		s.rdb.Expire(ctx, key, getSessionBindingTTL())
	}

	return binding, nil
}

// SetSessionBinding 设置会话绑定
func (s *SessionCache) SetSessionBinding(ctx context.Context, binding *SessionBinding) error {
	key := fmt.Sprintf(keySessionBinding, binding.SessionID)
	now := time.Now()

	if binding.BoundAt.IsZero() {
		binding.BoundAt = now
	}
	binding.LastUsedAt = now

	pipe := s.rdb.Pipeline()

	pipe.HSet(ctx, key,
		"account_id", binding.AccountID,
		"platform", binding.Platform,
		"model", binding.Model,
		"user_id", binding.UserID,
		"api_key_id", binding.APIKeyID,
		"client_ip", binding.ClientIP,
		"bound_at", binding.BoundAt.Unix(),
		"last_used_at", binding.LastUsedAt.Unix(),
	)
	pipe.Expire(ctx, key, getSessionBindingTTL())

	// 维护账户->会话的索引
	accountKey := fmt.Sprintf(keySessionByAccount, binding.AccountID)
	pipe.SAdd(ctx, accountKey, binding.SessionID)
	pipe.Expire(ctx, accountKey, getSessionBindingTTL())

	_, err := pipe.Exec(ctx)
	return err
}

// UpdateSessionLastUsed 更新会话最后使用时间
func (s *SessionCache) UpdateSessionLastUsed(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf(keySessionBinding, sessionID)
	pipe := s.rdb.Pipeline()
	pipe.HSet(ctx, key, "last_used_at", time.Now().Unix())

	// 智能续期
	ttl, _ := s.rdb.TTL(ctx, key).Result()
	if ttl > 0 && ttl < getSessionRenewalTTL() {
		pipe.Expire(ctx, key, getSessionBindingTTL())
	}

	_, err := pipe.Exec(ctx)
	return err
}

// RemoveSessionBinding 移除会话绑定
func (s *SessionCache) RemoveSessionBinding(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf(keySessionBinding, sessionID)

	// 先获取账户ID，用于清理索引
	accountIDStr, _ := s.rdb.HGet(ctx, key, "account_id").Result()

	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, key)

	if accountIDStr != "" {
		accountID, _ := strconv.ParseUint(accountIDStr, 10, 32)
		accountKey := fmt.Sprintf(keySessionByAccount, accountID)
		pipe.SRem(ctx, accountKey, sessionID)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// GetAccountSessions 获取账户的所有会话
func (s *SessionCache) GetAccountSessions(ctx context.Context, accountID uint) ([]string, error) {
	key := fmt.Sprintf(keySessionByAccount, accountID)
	return s.rdb.SMembers(ctx, key).Result()
}

// ClearAccountSessions 清除账户的所有会话
func (s *SessionCache) ClearAccountSessions(ctx context.Context, accountID uint) (int64, error) {
	sessions, err := s.GetAccountSessions(ctx, accountID)
	if err != nil {
		return 0, err
	}

	if len(sessions) == 0 {
		return 0, nil
	}

	pipe := s.rdb.Pipeline()
	for _, sessionID := range sessions {
		key := fmt.Sprintf(keySessionBinding, sessionID)
		pipe.Del(ctx, key)
	}

	accountKey := fmt.Sprintf(keySessionByAccount, accountID)
	pipe.Del(ctx, accountKey)

	_, err = pipe.Exec(ctx)
	return int64(len(sessions)), err
}

// ListAllSessions 列出所有会话绑定
func (s *SessionCache) ListAllSessions(ctx context.Context, offset, limit int64) ([]SessionBinding, int64, error) {
	pattern := "session:binding:*"
	var sessions []SessionBinding
	var cursor uint64
	var total int64
	var count int64

	for {
		keys, newCursor, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, 0, err
		}

		for _, key := range keys {
			total++

			// 分页处理
			if total <= offset {
				continue
			}
			if count >= limit {
				cursor = newCursor
				continue
			}

			// 提取 sessionID
			sessionID := strings.TrimPrefix(key, "session:binding:")

			binding, err := s.GetSessionBinding(ctx, sessionID)
			if err != nil || binding == nil {
				continue
			}

			sessions = append(sessions, *binding)
			count++
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
	}

	return sessions, total, nil
}

// ==================== 临时不可用标记 ====================

// MarkAccountUnavailable 标记账户临时不可用
func (s *SessionCache) MarkAccountUnavailable(ctx context.Context, accountID uint, reason string, ttl time.Duration) error {
	key := fmt.Sprintf(keyTempUnavailable, accountID)
	if ttl == 0 {
		ttl = getUnavailableTTL()
	}
	return s.rdb.Set(ctx, key, reason, ttl).Err()
}

// IsAccountUnavailable 检查账户是否临时不可用
func (s *SessionCache) IsAccountUnavailable(ctx context.Context, accountID uint) (bool, string, error) {
	key := fmt.Sprintf(keyTempUnavailable, accountID)
	reason, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, reason, nil
}

// ClearAccountUnavailable 清除账户不可用标记
func (s *SessionCache) ClearAccountUnavailable(ctx context.Context, accountID uint) error {
	key := fmt.Sprintf(keyTempUnavailable, accountID)
	return s.rdb.Del(ctx, key).Err()
}

// GetAllUnavailableAccounts 获取所有不可用账户
func (s *SessionCache) GetAllUnavailableAccounts(ctx context.Context) ([]model.UnavailableAccount, error) {
	pattern := "unavailable:*"
	var accounts []model.UnavailableAccount
	var cursor uint64

	for {
		keys, newCursor, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			// 提取账户ID
			parts := strings.Split(key, ":")
			if len(parts) != 2 {
				continue
			}
			accountID, _ := strconv.ParseUint(parts[1], 10, 32)

			reason, _ := s.rdb.Get(ctx, key).Result()
			ttl, _ := s.rdb.TTL(ctx, key).Result()

			accounts = append(accounts, model.UnavailableAccount{
				AccountID:    uint(accountID),
				Reason:       reason,
				RemainingTTL: int64(ttl.Seconds()),
			})
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
	}

	return accounts, nil
}

// ==================== 并发控制 ====================

var (
	concurrencyLimits   = make(map[uint]int)
	concurrencyLimitsMu sync.RWMutex
)

// SetAccountConcurrencyLimit 设置账户并发限制（运行时覆盖）
func (s *SessionCache) SetAccountConcurrencyLimit(accountID uint, limit int) {
	concurrencyLimitsMu.Lock()
	defer concurrencyLimitsMu.Unlock()
	concurrencyLimits[accountID] = limit
}

// GetAccountConcurrencyLimit 获取账户并发限制
func (s *SessionCache) GetAccountConcurrencyLimit(accountID uint) int {
	concurrencyLimitsMu.RLock()
	defer concurrencyLimitsMu.RUnlock()
	if limit, ok := concurrencyLimits[accountID]; ok {
		return limit
	}
	return getDefaultConcurrencyMax()
}

// AcquireConcurrency 获取并发槽位（使用默认限制）
func (s *SessionCache) AcquireConcurrency(ctx context.Context, accountID uint) (bool, int64, error) {
	limit := s.GetAccountConcurrencyLimit(accountID)
	return s.AcquireConcurrencyWithLimit(ctx, accountID, limit)
}

// AcquireConcurrencyWithLimit 获取并发槽位（指定限制）
func (s *SessionCache) AcquireConcurrencyWithLimit(ctx context.Context, accountID uint, limit int) (bool, int64, error) {
	key := fmt.Sprintf(keyConcurrency, accountID)

	// 使用 INCR 原子增加
	current, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}

	// 设置过期时间（防止泄漏）
	s.rdb.Expire(ctx, key, getConcurrencyTTL())

	if int(current) > limit {
		// 超限，回退
		s.rdb.Decr(ctx, key)
		return false, current - 1, nil
	}

	return true, current, nil
}

// ReleaseConcurrency 释放并发槽位
func (s *SessionCache) ReleaseConcurrency(ctx context.Context, accountID uint) error {
	key := fmt.Sprintf(keyConcurrency, accountID)
	current, err := s.rdb.Decr(ctx, key).Result()
	if err != nil {
		return err
	}
	// 防止负数
	if current < 0 {
		s.rdb.Set(ctx, key, 0, getConcurrencyTTL())
	}
	return nil
}

// GetAccountConcurrency 获取账户当前并发数
func (s *SessionCache) GetAccountConcurrency(ctx context.Context, accountID uint) (int64, error) {
	key := fmt.Sprintf(keyConcurrency, accountID)
	val, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

// ResetAccountConcurrency 重置账户并发计数
func (s *SessionCache) ResetAccountConcurrency(ctx context.Context, accountID uint) error {
	key := fmt.Sprintf(keyConcurrency, accountID)
	return s.rdb.Del(ctx, key).Err()
}

// ==================== 用户并发控制 ====================

const keyUserConcurrency = "user:concurrency:%d" // user:concurrency:{userID}

// AcquireUserConcurrency 获取用户并发槽位
func (s *SessionCache) AcquireUserConcurrency(ctx context.Context, userID uint, limit int) (bool, int64, error) {
	if limit <= 0 {
		limit = 10 // 默认用户并发限制
	}

	key := fmt.Sprintf(keyUserConcurrency, userID)

	// 使用 INCR 原子增加
	current, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}

	// 设置过期时间（防止泄漏）
	s.rdb.Expire(ctx, key, getConcurrencyTTL())

	if int(current) > limit {
		// 超限，回退
		s.rdb.Decr(ctx, key)
		return false, current - 1, nil
	}

	return true, current, nil
}

// ReleaseUserConcurrency 释放用户并发槽位
func (s *SessionCache) ReleaseUserConcurrency(ctx context.Context, userID uint) error {
	key := fmt.Sprintf(keyUserConcurrency, userID)
	current, err := s.rdb.Decr(ctx, key).Result()
	if err != nil {
		return err
	}
	// 防止负数
	if current < 0 {
		s.rdb.Set(ctx, key, 0, getConcurrencyTTL())
	}
	return nil
}

// GetUserConcurrency 获取用户当前并发数
func (s *SessionCache) GetUserConcurrency(ctx context.Context, userID uint) (int64, error) {
	key := fmt.Sprintf(keyUserConcurrency, userID)
	val, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

// ResetUserConcurrency 重置用户并发计数
func (s *SessionCache) ResetUserConcurrency(ctx context.Context, userID uint) error {
	key := fmt.Sprintf(keyUserConcurrency, userID)
	return s.rdb.Del(ctx, key).Err()
}
