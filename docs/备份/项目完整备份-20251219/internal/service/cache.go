package service

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

// CacheService 缓存管理服务
type CacheService struct {
	rdb *redis.Client
}

func NewCacheService() *CacheService {
	return &CacheService{
		rdb: repository.RDB,
	}
}

// Redis Key 前缀
const (
	// 粘性会话 Keys
	keySessionBinding   = "session:binding:%s" // session:binding:{sessionID} -> accountID
	keySessionByAccount = "session:account:%d" // session:account:{accountID} -> set of sessionIDs

	// 临时不可用标记 Keys
	keyTempUnavailable = "unavailable:%d" // unavailable:{accountID}

	// 并发控制 Keys
	keyConcurrency = "concurrency:%d" // concurrency:{accountID}

	// 缓存统计 Key 前缀
	keyCachePrefix = "cache:"
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

// 导出的默认配置（兼容旧代码）
var (
	DefaultTempUnavailableTTL = func() time.Duration { return getUnavailableTTL() }
	DefaultSessionBindingTTL  = func() time.Duration { return getSessionBindingTTL() }
	DefaultConcurrencyLimit   = func() int { return getDefaultConcurrencyMax() }
)

// ==================== 粘性会话管理 ====================

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
func (s *CacheService) GetSessionBinding(ctx context.Context, sessionID string) (*SessionBinding, error) {
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
func (s *CacheService) SetSessionBinding(ctx context.Context, binding *SessionBinding) error {
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
func (s *CacheService) UpdateSessionLastUsed(ctx context.Context, sessionID string) error {
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
func (s *CacheService) RemoveSessionBinding(ctx context.Context, sessionID string) error {
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
func (s *CacheService) GetAccountSessions(ctx context.Context, accountID uint) ([]string, error) {
	key := fmt.Sprintf(keySessionByAccount, accountID)
	return s.rdb.SMembers(ctx, key).Result()
}

// ClearAccountSessions 清除账户的所有会话
func (s *CacheService) ClearAccountSessions(ctx context.Context, accountID uint) (int64, error) {
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

// ==================== 临时不可用标记 ====================

// MarkAccountUnavailable 标记账户临时不可用
func (s *CacheService) MarkAccountUnavailable(ctx context.Context, accountID uint, reason string, ttl time.Duration) error {
	key := fmt.Sprintf(keyTempUnavailable, accountID)
	if ttl == 0 {
		ttl = getUnavailableTTL()
	}
	return s.rdb.Set(ctx, key, reason, ttl).Err()
}

// IsAccountUnavailable 检查账户是否临时不可用
func (s *CacheService) IsAccountUnavailable(ctx context.Context, accountID uint) (bool, string, error) {
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
func (s *CacheService) ClearAccountUnavailable(ctx context.Context, accountID uint) error {
	key := fmt.Sprintf(keyTempUnavailable, accountID)
	return s.rdb.Del(ctx, key).Err()
}

// GetAllUnavailableAccounts 获取所有不可用账户
func (s *CacheService) GetAllUnavailableAccounts(ctx context.Context) ([]model.UnavailableAccount, error) {
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

// SetAccountConcurrencyLimit 设置账户并发限制
func (s *CacheService) SetAccountConcurrencyLimit(accountID uint, limit int) {
	concurrencyLimitsMu.Lock()
	defer concurrencyLimitsMu.Unlock()
	concurrencyLimits[accountID] = limit
}

// GetAccountConcurrencyLimit 获取账户并发限制
func (s *CacheService) GetAccountConcurrencyLimit(accountID uint) int {
	concurrencyLimitsMu.RLock()
	defer concurrencyLimitsMu.RUnlock()
	if limit, ok := concurrencyLimits[accountID]; ok {
		return limit
	}
	return getDefaultConcurrencyMax()
}

// AcquireConcurrency 获取并发槽位
func (s *CacheService) AcquireConcurrency(ctx context.Context, accountID uint) (bool, int64, error) {
	key := fmt.Sprintf(keyConcurrency, accountID)
	limit := s.GetAccountConcurrencyLimit(accountID)

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
func (s *CacheService) ReleaseConcurrency(ctx context.Context, accountID uint) error {
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
func (s *CacheService) GetAccountConcurrency(ctx context.Context, accountID uint) (int64, error) {
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
func (s *CacheService) ResetAccountConcurrency(ctx context.Context, accountID uint) error {
	key := fmt.Sprintf(keyConcurrency, accountID)
	return s.rdb.Del(ctx, key).Err()
}

// ==================== 用户并发控制 ====================

const keyUserConcurrency = "user:concurrency:%d" // user:concurrency:{userID}

// GetUserConcurrency 获取用户当前并发数
func (s *CacheService) GetUserConcurrency(ctx context.Context, userID uint) (int64, error) {
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
func (s *CacheService) ResetUserConcurrency(ctx context.Context, userID uint) error {
	key := fmt.Sprintf(keyUserConcurrency, userID)
	return s.rdb.Del(ctx, key).Err()
}

// ==================== 缓存管理统计 ====================

// CacheStats 缓存统计信息
type CacheStats struct {
	SessionCount      int64 `json:"session_count"`
	UnavailableCount  int64 `json:"unavailable_count"`
	UsageKeyCount     int64 `json:"usage_key_count"`
	CostKeyCount      int64 `json:"cost_key_count"`
	TotalKeyCount     int64 `json:"total_key_count"`
	MemoryUsed        int64 `json:"memory_used"`
	MemoryUsedHuman   string `json:"memory_used_human"`
}

// GetCacheStats 获取缓存统计
func (s *CacheService) GetCacheStats(ctx context.Context) (*CacheStats, error) {
	stats := &CacheStats{}

	// 统计各类 key 数量
	sessionCount, _ := s.countKeys(ctx, "session:binding:*")
	unavailableCount, _ := s.countKeys(ctx, "unavailable:*")
	usageCount, _ := s.countKeys(ctx, "usage:*")
	costCount, _ := s.countKeys(ctx, "cost:*")

	stats.SessionCount = sessionCount
	stats.UnavailableCount = unavailableCount
	stats.UsageKeyCount = usageCount
	stats.CostKeyCount = costCount

	// 获取总 key 数量
	dbSize, _ := s.rdb.DBSize(ctx).Result()
	stats.TotalKeyCount = dbSize

	// 获取内存使用
	info, err := repository.GetRedisInfo()
	if err == nil {
		if mem, ok := info["used_memory"]; ok {
			stats.MemoryUsed, _ = strconv.ParseInt(mem, 10, 64)
		}
		if memHuman, ok := info["used_memory_human"]; ok {
			stats.MemoryUsedHuman = memHuman
		}
	}

	return stats, nil
}

// countKeys 统计匹配的 key 数量
func (s *CacheService) countKeys(ctx context.Context, pattern string) (int64, error) {
	var count int64
	var cursor uint64

	for {
		keys, newCursor, err := s.rdb.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return 0, err
		}
		count += int64(len(keys))
		cursor = newCursor
		if cursor == 0 {
			break
		}
	}

	return count, nil
}

// ==================== 手动缓存清理 ====================

// ClearCacheType 缓存类型
type ClearCacheType string

const (
	ClearCacheAll         ClearCacheType = "all"
	ClearCacheSessions    ClearCacheType = "sessions"
	ClearCacheUnavailable ClearCacheType = "unavailable"
	ClearCacheUsage       ClearCacheType = "usage"
	ClearCacheCost        ClearCacheType = "cost"
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

	var pattern string
	switch cacheType {
	case ClearCacheAll:
		// 清理所有业务缓存（保留系统缓存）
		patterns := []string{"session:*", "unavailable:*", "usage:*", "cost:*", "concurrency:*"}
		var totalDeleted int64
		for _, p := range patterns {
			deleted, err := s.deleteByPattern(ctx, p)
			if err != nil {
				result.Error = err.Error()
				return result, err
			}
			totalDeleted += deleted
		}
		result.DeletedCount = totalDeleted
		return result, nil

	case ClearCacheSessions:
		pattern = "session:*"
	case ClearCacheUnavailable:
		pattern = "unavailable:*"
	case ClearCacheUsage:
		pattern = "usage:*"
	case ClearCacheCost:
		pattern = "cost:*"
	case ClearCacheConcurrency:
		pattern = "concurrency:*"
	default:
		return nil, fmt.Errorf("unknown cache type: %s", cacheType)
	}

	deleted, err := s.deleteByPattern(ctx, pattern)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	result.DeletedCount = deleted
	return result, nil
}

// deleteByPattern 按模式删除 key
func (s *CacheService) deleteByPattern(ctx context.Context, pattern string) (int64, error) {
	var deleted int64
	var cursor uint64

	for {
		keys, newCursor, err := s.rdb.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return deleted, err
		}

		if len(keys) > 0 {
			count, err := s.rdb.Del(ctx, keys...).Result()
			if err != nil {
				return deleted, err
			}
			deleted += count
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
	}

	return deleted, nil
}

// ClearUserCache 清理指定用户的缓存
func (s *CacheService) ClearUserCache(ctx context.Context, userID uint) (*ClearCacheResult, error) {
	result := &ClearCacheResult{Type: "user"}

	patterns := []string{
		fmt.Sprintf("usage:total:%d", userID),
		fmt.Sprintf("usage:daily:%d:*", userID),
		fmt.Sprintf("usage:monthly:%d:*", userID),
		fmt.Sprintf("usage:records:%d", userID),
		fmt.Sprintf("usage:model:%d:*", userID),
		fmt.Sprintf("cost:total:%d", userID),
		fmt.Sprintf("cost:daily:%d:*", userID),
		fmt.Sprintf("cost:monthly:%d:*", userID),
	}

	var totalDeleted int64
	for _, pattern := range patterns {
		// 如果是精确 key（不含通配符），直接删除
		if !strings.Contains(pattern, "*") {
			count, _ := s.rdb.Del(ctx, pattern).Result()
			totalDeleted += count
		} else {
			deleted, err := s.deleteByPattern(ctx, pattern)
			if err != nil {
				result.Error = err.Error()
				return result, err
			}
			totalDeleted += deleted
		}
	}

	result.DeletedCount = totalDeleted
	return result, nil
}

// ClearAPIKeyCache 清理指定 API Key 的缓存
func (s *CacheService) ClearAPIKeyCache(ctx context.Context, apiKeyID uint) (*ClearCacheResult, error) {
	result := &ClearCacheResult{Type: "apikey"}

	patterns := []string{
		fmt.Sprintf("usage:key:%d", apiKeyID),
		fmt.Sprintf("usage:key:daily:%d:*", apiKeyID),
		fmt.Sprintf("usage:key:monthly:%d:*", apiKeyID),
		fmt.Sprintf("usage:records:key:%d", apiKeyID),
		fmt.Sprintf("cost:key:%d", apiKeyID),
		fmt.Sprintf("cost:key:daily:%d:*", apiKeyID),
		fmt.Sprintf("cost:key:monthly:%d:*", apiKeyID),
	}

	var totalDeleted int64
	for _, pattern := range patterns {
		if !strings.Contains(pattern, "*") {
			count, _ := s.rdb.Del(ctx, pattern).Result()
			totalDeleted += count
		} else {
			deleted, err := s.deleteByPattern(ctx, pattern)
			if err != nil {
				result.Error = err.Error()
				return result, err
			}
			totalDeleted += deleted
		}
	}

	result.DeletedCount = totalDeleted
	return result, nil
}

// ==================== 会话列表查询 ====================

// AccountCacheInfo 账号缓存信息（聚合）
type AccountCacheInfo struct {
	AccountID   uint             `json:"account_id"`
	AccountName string           `json:"account_name"`
	Concurrency int64            `json:"concurrency"`
	MaxConcurrency int           `json:"max_concurrency"`
	SessionCount int             `json:"session_count"`
	Sessions    []SessionBinding `json:"sessions"`
	Users       []uint           `json:"users"` // 使用此账号的用户ID列表
}

// UserCacheInfo 用户缓存信息（聚合）
type UserCacheInfo struct {
	UserID       uint             `json:"user_id"`
	UserName     string           `json:"user_name"`
	Concurrency  int64            `json:"concurrency"`
	MaxConcurrency int            `json:"max_concurrency"`
	SessionCount int              `json:"session_count"`
	Sessions     []SessionBinding `json:"sessions"`
	Accounts     []uint           `json:"accounts"` // 此用户使用的账号ID列表
}

// ListAccountsWithCache 列出所有有缓存的账号（聚合视图）
func (s *CacheService) ListAccountsWithCache(ctx context.Context) ([]AccountCacheInfo, error) {
	// 获取所有会话
	sessions, _, err := s.ListAllSessions(ctx, 0, 1000)
	if err != nil {
		return nil, err
	}

	// 按账号聚合
	accountMap := make(map[uint]*AccountCacheInfo)
	for _, sess := range sessions {
		if _, ok := accountMap[sess.AccountID]; !ok {
			concurrency, _ := s.GetAccountConcurrency(ctx, sess.AccountID)
			accountMap[sess.AccountID] = &AccountCacheInfo{
				AccountID:      sess.AccountID,
				Concurrency:    concurrency,
				MaxConcurrency: s.GetAccountConcurrencyLimit(sess.AccountID),
				Sessions:       []SessionBinding{},
				Users:          []uint{},
			}
		}
		info := accountMap[sess.AccountID]
		info.Sessions = append(info.Sessions, sess)
		info.SessionCount = len(info.Sessions)

		// 添加用户（去重）
		found := false
		for _, uid := range info.Users {
			if uid == sess.UserID {
				found = true
				break
			}
		}
		if !found && sess.UserID > 0 {
			info.Users = append(info.Users, sess.UserID)
		}
	}

	// 转为列表
	result := make([]AccountCacheInfo, 0, len(accountMap))
	for _, info := range accountMap {
		result = append(result, *info)
	}

	return result, nil
}

// ListUsersWithCache 列出所有有缓存的用户（聚合视图）
func (s *CacheService) ListUsersWithCache(ctx context.Context) ([]UserCacheInfo, error) {
	// 获取所有会话
	sessions, _, err := s.ListAllSessions(ctx, 0, 1000)
	if err != nil {
		return nil, err
	}

	// 按用户聚合
	userMap := make(map[uint]*UserCacheInfo)
	for _, sess := range sessions {
		if sess.UserID == 0 {
			continue
		}
		if _, ok := userMap[sess.UserID]; !ok {
			concurrency, _ := s.GetUserConcurrency(ctx, sess.UserID)
			userMap[sess.UserID] = &UserCacheInfo{
				UserID:      sess.UserID,
				Concurrency: concurrency,
				Sessions:    []SessionBinding{},
				Accounts:    []uint{},
			}
		}
		info := userMap[sess.UserID]
		info.Sessions = append(info.Sessions, sess)
		info.SessionCount = len(info.Sessions)

		// 添加账号（去重）
		found := false
		for _, aid := range info.Accounts {
			if aid == sess.AccountID {
				found = true
				break
			}
		}
		if !found && sess.AccountID > 0 {
			info.Accounts = append(info.Accounts, sess.AccountID)
		}
	}

	// 转为列表
	result := make([]UserCacheInfo, 0, len(userMap))
	for _, info := range userMap {
		result = append(result, *info)
	}

	return result, nil
}

// ListAllSessions 列出所有会话绑定
func (s *CacheService) ListAllSessions(ctx context.Context, offset, limit int64) ([]SessionBinding, int64, error) {
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
