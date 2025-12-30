/*
 * 文件作用：内存缓存实现，提供会话存储、并发管理和不可用标记
 * 负责功能：
 *   - 会话绑定存储（SessionStore）
 *   - 并发计数管理（ConcurrencyManager）
 *   - 账户不可用标记（UnavailableMarker）
 *   - 过期数据自动清理
 * 重要程度：⭐⭐⭐⭐ 重要（内存缓存核心）
 * 依赖模块：config
 */
package cache

import (
	"context"
	"sort"
	"sync"
	"time"

	"go-aiproxy/internal/config"
)

// ==================== 会话存储 ====================

// MemorySessionBinding 内存中的会话绑定信息
type MemorySessionBinding struct {
	SessionID  string
	AccountID  uint
	Platform   string
	Model      string
	UserID     uint
	APIKeyID   uint
	ClientIP   string
	UserAgent  string
	BoundAt    time.Time
	LastUsedAt time.Time
	ExpireAt   time.Time
}

// IsExpired 检查是否过期
func (b *MemorySessionBinding) IsExpired() bool {
	return time.Now().After(b.ExpireAt)
}

// SessionStore 会话存储（替代 Redis 的会话功能）
type SessionStore struct {
	bindings  sync.Map // sessionID -> *MemorySessionBinding
	byAccount sync.Map // accountID -> *sessionSet (session IDs)
	byUser    sync.Map // userID -> *sessionSet (session IDs)

	cleanupInterval time.Duration
	stopCleanup     chan struct{}
	cleanupOnce     sync.Once
	cleanupWg       sync.WaitGroup
}

// sessionSet 线程安全的 session ID 集合
type sessionSet struct {
	mu   sync.RWMutex
	data map[string]struct{}
}

func newSessionSet() *sessionSet {
	return &sessionSet{
		data: make(map[string]struct{}),
	}
}

func (s *sessionSet) Add(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[sessionID] = struct{}{}
}

func (s *sessionSet) Remove(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, sessionID)
}

func (s *sessionSet) List() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]string, 0, len(s.data))
	for id := range s.data {
		result = append(result, id)
	}
	return result
}

func (s *sessionSet) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// 全局会话存储单例
var (
	globalSessionStore *SessionStore
	sessionStoreOnce   sync.Once
)

// GetSessionStore 获取会话存储单例
func GetSessionStore() *SessionStore {
	sessionStoreOnce.Do(func() {
		globalSessionStore = &SessionStore{
			cleanupInterval: 1 * time.Minute,
			stopCleanup:     make(chan struct{}),
		}
		globalSessionStore.StartCleanup()
	})
	return globalSessionStore
}

// StartCleanup 启动定期清理
func (s *SessionStore) StartCleanup() {
	s.cleanupOnce.Do(func() {
		s.cleanupWg.Add(1)
		go func() {
			defer s.cleanupWg.Done()
			ticker := time.NewTicker(s.cleanupInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					s.cleanExpired()
				case <-s.stopCleanup:
					return
				}
			}
		}()
	})
}

// StopCleanup 停止定期清理
func (s *SessionStore) StopCleanup() {
	select {
	case <-s.stopCleanup:
		// 已经关闭
	default:
		close(s.stopCleanup)
	}
	s.cleanupWg.Wait()
}

// cleanExpired 清理过期会话
func (s *SessionStore) cleanExpired() {
	now := time.Now()
	var expiredSessions []string

	s.bindings.Range(func(key, value interface{}) bool {
		binding := value.(*MemorySessionBinding)
		if now.After(binding.ExpireAt) {
			expiredSessions = append(expiredSessions, binding.SessionID)
		}
		return true
	})

	for _, sessionID := range expiredSessions {
		s.Remove(sessionID)
	}
}

// getSessionTTL 获取会话 TTL
func getSessionTTL() time.Duration {
	return time.Duration(config.Cfg.Cache.GetSessionTTL()) * time.Minute
}

// getSessionRenewalThreshold 获取续期阈值
func getSessionRenewalThreshold() time.Duration {
	return time.Duration(config.Cfg.Cache.GetSessionRenewalTTL()) * time.Minute
}

// Get 获取会话绑定
func (s *SessionStore) Get(sessionID string) *MemorySessionBinding {
	value, ok := s.bindings.Load(sessionID)
	if !ok {
		return nil
	}

	binding := value.(*MemorySessionBinding)
	if binding.IsExpired() {
		s.Remove(sessionID)
		return nil
	}

	// 智能续期：剩余时间不足阈值时续期
	remaining := time.Until(binding.ExpireAt)
	if remaining > 0 && remaining < getSessionRenewalThreshold() {
		binding.ExpireAt = time.Now().Add(getSessionTTL())
	}

	return binding
}

// Set 设置会话绑定
func (s *SessionStore) Set(binding *MemorySessionBinding) {
	now := time.Now()
	if binding.BoundAt.IsZero() {
		binding.BoundAt = now
	}
	binding.LastUsedAt = now
	binding.ExpireAt = now.Add(getSessionTTL())

	s.bindings.Store(binding.SessionID, binding)

	// 维护账户索引
	if binding.AccountID > 0 {
		setVal, _ := s.byAccount.LoadOrStore(binding.AccountID, newSessionSet())
		setVal.(*sessionSet).Add(binding.SessionID)
	}

	// 维护用户索引
	if binding.UserID > 0 {
		setVal, _ := s.byUser.LoadOrStore(binding.UserID, newSessionSet())
		setVal.(*sessionSet).Add(binding.SessionID)
	}
}

// UpdateLastUsed 更新最后使用时间
func (s *SessionStore) UpdateLastUsed(sessionID string) bool {
	value, ok := s.bindings.Load(sessionID)
	if !ok {
		return false
	}

	binding := value.(*MemorySessionBinding)
	if binding.IsExpired() {
		s.Remove(sessionID)
		return false
	}

	now := time.Now()
	binding.LastUsedAt = now
	// 会话粘性更符合直觉的语义：最后使用后 TTL（滑动过期）。
	// 之前的“接近过期才续期”策略更适合 Redis TTL 场景，但在内存缓存下会让前端看到 TTL 一直倒计时，误以为“几分钟就没了”。
	binding.ExpireAt = now.Add(getSessionTTL())

	return true
}

// Remove 移除会话绑定
func (s *SessionStore) Remove(sessionID string) {
	value, ok := s.bindings.LoadAndDelete(sessionID)
	if !ok {
		return
	}

	binding := value.(*MemorySessionBinding)

	// 从账户索引移除
	if binding.AccountID > 0 {
		if setVal, ok := s.byAccount.Load(binding.AccountID); ok {
			setVal.(*sessionSet).Remove(sessionID)
		}
	}

	// 从用户索引移除
	if binding.UserID > 0 {
		if setVal, ok := s.byUser.Load(binding.UserID); ok {
			setVal.(*sessionSet).Remove(sessionID)
		}
	}
}

// GetByAccount 获取账户的所有会话
func (s *SessionStore) GetByAccount(accountID uint) []*MemorySessionBinding {
	setVal, ok := s.byAccount.Load(accountID)
	if !ok {
		return nil
	}

	sessionIDs := setVal.(*sessionSet).List()
	var result []*MemorySessionBinding

	for _, sessionID := range sessionIDs {
		if binding := s.Get(sessionID); binding != nil {
			result = append(result, binding)
		}
	}

	return result
}

// GetByUser 获取用户的所有会话
func (s *SessionStore) GetByUser(userID uint) []*MemorySessionBinding {
	setVal, ok := s.byUser.Load(userID)
	if !ok {
		return nil
	}

	sessionIDs := setVal.(*sessionSet).List()
	var result []*MemorySessionBinding

	for _, sessionID := range sessionIDs {
		if binding := s.Get(sessionID); binding != nil {
			result = append(result, binding)
		}
	}

	return result
}

// ClearByAccount 清除账户的所有会话
func (s *SessionStore) ClearByAccount(accountID uint) int {
	setVal, ok := s.byAccount.Load(accountID)
	if !ok {
		return 0
	}

	sessionIDs := setVal.(*sessionSet).List()
	for _, sessionID := range sessionIDs {
		s.Remove(sessionID)
	}

	s.byAccount.Delete(accountID)
	return len(sessionIDs)
}

// ClearByUser 清除用户的所有会话
func (s *SessionStore) ClearByUser(userID uint) int {
	setVal, ok := s.byUser.Load(userID)
	if !ok {
		return 0
	}

	sessionIDs := setVal.(*sessionSet).List()
	for _, sessionID := range sessionIDs {
		s.Remove(sessionID)
	}

	s.byUser.Delete(userID)
	return len(sessionIDs)
}

// ListAll 列出所有会话（带分页）
func (s *SessionStore) ListAll(offset, limit int) ([]*MemorySessionBinding, int) {
	var all []*MemorySessionBinding

	s.bindings.Range(func(key, value interface{}) bool {
		binding := value.(*MemorySessionBinding)
		if !binding.IsExpired() {
			all = append(all, binding)
		}
		return true
	})

	// 稳定排序，避免 sync.Map 遍历顺序不稳定导致分页“会话消失/跳页”的错觉：
	// 1) 最后使用时间（新->旧）
	// 2) 绑定时间（新->旧）
	// 3) SessionID（字典序）
	sort.Slice(all, func(i, j int) bool {
		li, lj := all[i].LastUsedAt, all[j].LastUsedAt
		if !li.Equal(lj) {
			return li.After(lj)
		}
		bi, bj := all[i].BoundAt, all[j].BoundAt
		if !bi.Equal(bj) {
			return bi.After(bj)
		}
		return all[i].SessionID < all[j].SessionID
	})

	total := len(all)

	// 分页
	if offset >= total {
		return nil, total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	return all[offset:end], total
}

// Count 统计会话数量
func (s *SessionStore) Count() int {
	count := 0
	s.bindings.Range(func(key, value interface{}) bool {
		binding := value.(*MemorySessionBinding)
		if !binding.IsExpired() {
			count++
		}
		return true
	})
	return count
}

// ClearAll 清除所有会话
func (s *SessionStore) ClearAll() int {
	count := 0
	s.bindings.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	s.bindings = sync.Map{}
	s.byAccount = sync.Map{}
	s.byUser = sync.Map{}

	return count
}

// ==================== 并发控制 ====================

// ConcurrencyCounter 带TTL的并发计数器
type ConcurrencyCounter struct {
	mu    sync.Mutex
	slots []time.Time // 每个槽位的获取时间
	limit int
}

// Count 获取当前有效并发数（排除过期槽位）
func (c *ConcurrencyCounter) Count(ttl time.Duration) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cleanExpiredLocked(ttl)
	return len(c.slots)
}

// Acquire 获取并发槽位
func (c *ConcurrencyCounter) Acquire(limit int, ttl time.Duration) (bool, int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 先清理过期槽位
	c.cleanExpiredLocked(ttl)

	if len(c.slots) >= limit {
		return false, len(c.slots)
	}

	c.slots = append(c.slots, time.Now())
	return true, len(c.slots)
}

// Release 释放并发槽位（移除最老的一个）
func (c *ConcurrencyCounter) Release() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.slots) > 0 {
		c.slots = c.slots[1:] // 移除最老的槽位
	}
}

// Reset 重置计数器
func (c *ConcurrencyCounter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.slots = nil
}

// cleanExpiredLocked 清理过期槽位（需要持有锁）
func (c *ConcurrencyCounter) cleanExpiredLocked(ttl time.Duration) {
	if ttl <= 0 || len(c.slots) == 0 {
		return
	}

	now := time.Now()
	validStart := 0
	for i, t := range c.slots {
		if now.Sub(t) < ttl {
			validStart = i
			break
		}
		validStart = i + 1
	}

	if validStart > 0 {
		c.slots = c.slots[validStart:]
	}
}

// ConcurrencyManager 并发控制管理器（带TTL支持）
type ConcurrencyManager struct {
	accountCounters sync.Map // accountID -> *ConcurrencyCounter
	userCounters    sync.Map // userID -> *ConcurrencyCounter
	accountLimits   sync.Map // accountID -> int (自定义限制)

	cleanupInterval time.Duration
	stopCleanup     chan struct{}
	cleanupOnce     sync.Once
	cleanupWg       sync.WaitGroup
}

// 全局并发管理器单例
var (
	globalConcurrencyManager *ConcurrencyManager
	concurrencyManagerOnce   sync.Once
)

// GetConcurrencyManager 获取并发管理器单例
func GetConcurrencyManager() *ConcurrencyManager {
	concurrencyManagerOnce.Do(func() {
		globalConcurrencyManager = &ConcurrencyManager{
			cleanupInterval: 1 * time.Minute,
			stopCleanup:     make(chan struct{}),
		}
		globalConcurrencyManager.StartCleanup()
	})
	return globalConcurrencyManager
}

// StartCleanup 启动定期清理
func (m *ConcurrencyManager) StartCleanup() {
	m.cleanupOnce.Do(func() {
		m.cleanupWg.Add(1)
		go func() {
			defer m.cleanupWg.Done()
			ticker := time.NewTicker(m.cleanupInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					m.cleanExpired()
				case <-m.stopCleanup:
					return
				}
			}
		}()
	})
}

// StopCleanup 停止定期清理
func (m *ConcurrencyManager) StopCleanup() {
	select {
	case <-m.stopCleanup:
	default:
		close(m.stopCleanup)
	}
	m.cleanupWg.Wait()
}

// cleanExpired 清理所有过期槽位
func (m *ConcurrencyManager) cleanExpired() {
	ttl := getConcurrencyTTL()

	m.accountCounters.Range(func(key, value interface{}) bool {
		counter := value.(*ConcurrencyCounter)
		counter.Count(ttl) // Count 会触发清理
		return true
	})

	m.userCounters.Range(func(key, value interface{}) bool {
		counter := value.(*ConcurrencyCounter)
		counter.Count(ttl) // Count 会触发清理
		return true
	})
}

// getConcurrencyTTL 获取并发TTL
func getConcurrencyTTL() time.Duration {
	return time.Duration(config.Cfg.Cache.GetConcurrencyTTL()) * time.Minute
}

// getDefaultConcurrencyMax 获取默认并发上限
func getDefaultConcurrencyMax() int {
	return config.Cfg.Cache.GetDefaultConcurrencyMax()
}

// SetAccountLimit 设置账户并发限制
func (m *ConcurrencyManager) SetAccountLimit(accountID uint, limit int) {
	m.accountLimits.Store(accountID, limit)
}

// GetAccountLimit 获取账户并发限制
func (m *ConcurrencyManager) GetAccountLimit(accountID uint) int {
	if val, ok := m.accountLimits.Load(accountID); ok {
		return val.(int)
	}
	return getDefaultConcurrencyMax()
}

// getOrCreateAccountCounter 获取或创建账户计数器
func (m *ConcurrencyManager) getOrCreateAccountCounter(accountID uint) *ConcurrencyCounter {
	val, _ := m.accountCounters.LoadOrStore(accountID, &ConcurrencyCounter{})
	return val.(*ConcurrencyCounter)
}

// getOrCreateUserCounter 获取或创建用户计数器
func (m *ConcurrencyManager) getOrCreateUserCounter(userID uint) *ConcurrencyCounter {
	val, _ := m.userCounters.LoadOrStore(userID, &ConcurrencyCounter{})
	return val.(*ConcurrencyCounter)
}

// AcquireAccount 获取账户并发槽位
func (m *ConcurrencyManager) AcquireAccount(ctx context.Context, accountID uint) (bool, int64) {
	limit := m.GetAccountLimit(accountID)
	return m.AcquireAccountWithLimit(ctx, accountID, limit)
}

// AcquireAccountWithLimit 获取账户并发槽位（指定限制）
func (m *ConcurrencyManager) AcquireAccountWithLimit(ctx context.Context, accountID uint, limit int) (bool, int64) {
	counter := m.getOrCreateAccountCounter(accountID)
	ttl := getConcurrencyTTL()
	acquired, count := counter.Acquire(limit, ttl)
	return acquired, int64(count)
}

// ReleaseAccount 释放账户并发槽位
func (m *ConcurrencyManager) ReleaseAccount(ctx context.Context, accountID uint) {
	counter := m.getOrCreateAccountCounter(accountID)
	counter.Release()
}

// GetAccountConcurrency 获取账户当前并发数
func (m *ConcurrencyManager) GetAccountConcurrency(accountID uint) int64 {
	counter := m.getOrCreateAccountCounter(accountID)
	ttl := getConcurrencyTTL()
	return int64(counter.Count(ttl))
}

// ResetAccountConcurrency 重置账户并发计数
func (m *ConcurrencyManager) ResetAccountConcurrency(accountID uint) {
	if val, ok := m.accountCounters.Load(accountID); ok {
		val.(*ConcurrencyCounter).Reset()
	}
}

// AcquireUser 获取用户并发槽位
func (m *ConcurrencyManager) AcquireUser(ctx context.Context, userID uint, limit int) (bool, int64) {
	if limit <= 0 {
		limit = 10 // 默认用户并发限制
	}

	counter := m.getOrCreateUserCounter(userID)
	ttl := getConcurrencyTTL()
	acquired, count := counter.Acquire(limit, ttl)
	return acquired, int64(count)
}

// ReleaseUser 释放用户并发槽位
func (m *ConcurrencyManager) ReleaseUser(ctx context.Context, userID uint) {
	counter := m.getOrCreateUserCounter(userID)
	counter.Release()
}

// GetUserConcurrency 获取用户当前并发数
func (m *ConcurrencyManager) GetUserConcurrency(userID uint) int64 {
	counter := m.getOrCreateUserCounter(userID)
	ttl := getConcurrencyTTL()
	return int64(counter.Count(ttl))
}

// ResetUserConcurrency 重置用户并发计数
func (m *ConcurrencyManager) ResetUserConcurrency(userID uint) {
	if val, ok := m.userCounters.Load(userID); ok {
		val.(*ConcurrencyCounter).Reset()
	}
}

// Stats 获取并发管理器统计
func (m *ConcurrencyManager) Stats() (accountCount, userCount int) {
	m.accountCounters.Range(func(_, _ interface{}) bool {
		accountCount++
		return true
	})
	m.userCounters.Range(func(_, _ interface{}) bool {
		userCount++
		return true
	})
	return
}

// ==================== 不可用标记 ====================

// unavailableMark 不可用标记
type unavailableMark struct {
	Reason   string
	ExpireAt time.Time
}

// UnavailableMarker 不可用标记管理器（替代 Redis String + TTL）
type UnavailableMarker struct {
	marks           sync.Map // accountID -> *unavailableMark
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
	cleanupOnce     sync.Once
	cleanupWg       sync.WaitGroup
}

// 全局不可用标记管理器单例
var (
	globalUnavailableMarker *UnavailableMarker
	unavailableMarkerOnce   sync.Once
)

// GetUnavailableMarker 获取不可用标记管理器单例
func GetUnavailableMarker() *UnavailableMarker {
	unavailableMarkerOnce.Do(func() {
		globalUnavailableMarker = &UnavailableMarker{
			cleanupInterval: 1 * time.Minute,
			stopCleanup:     make(chan struct{}),
		}
		globalUnavailableMarker.StartCleanup()
	})
	return globalUnavailableMarker
}

// getUnavailableTTL 获取不可用标记 TTL
func getUnavailableTTL() time.Duration {
	return time.Duration(config.Cfg.Cache.GetUnavailableTTL()) * time.Minute
}

// StartCleanup 启动定期清理
func (m *UnavailableMarker) StartCleanup() {
	m.cleanupOnce.Do(func() {
		m.cleanupWg.Add(1)
		go func() {
			defer m.cleanupWg.Done()
			ticker := time.NewTicker(m.cleanupInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					m.cleanExpired()
				case <-m.stopCleanup:
					return
				}
			}
		}()
	})
}

// StopCleanup 停止定期清理
func (m *UnavailableMarker) StopCleanup() {
	select {
	case <-m.stopCleanup:
	default:
		close(m.stopCleanup)
	}
	m.cleanupWg.Wait()
}

// cleanExpired 清理过期标记
func (m *UnavailableMarker) cleanExpired() {
	now := time.Now()
	m.marks.Range(func(key, value interface{}) bool {
		mark := value.(*unavailableMark)
		if now.After(mark.ExpireAt) {
			m.marks.Delete(key)
		}
		return true
	})
}

// Mark 标记账户不可用
func (m *UnavailableMarker) Mark(accountID uint, reason string, ttl time.Duration) {
	if ttl == 0 {
		ttl = getUnavailableTTL()
	}
	m.marks.Store(accountID, &unavailableMark{
		Reason:   reason,
		ExpireAt: time.Now().Add(ttl),
	})
}

// IsUnavailable 检查账户是否不可用
func (m *UnavailableMarker) IsUnavailable(accountID uint) (bool, string) {
	value, ok := m.marks.Load(accountID)
	if !ok {
		return false, ""
	}

	mark := value.(*unavailableMark)
	if time.Now().After(mark.ExpireAt) {
		m.marks.Delete(accountID)
		return false, ""
	}

	return true, mark.Reason
}

// Clear 清除账户不可用标记
func (m *UnavailableMarker) Clear(accountID uint) {
	m.marks.Delete(accountID)
}

// ListAll 列出所有不可用账户
func (m *UnavailableMarker) ListAll() map[uint]struct {
	Reason       string
	RemainingTTL int64
} {
	result := make(map[uint]struct {
		Reason       string
		RemainingTTL int64
	})

	now := time.Now()
	m.marks.Range(func(key, value interface{}) bool {
		accountID := key.(uint)
		mark := value.(*unavailableMark)
		if now.Before(mark.ExpireAt) {
			result[accountID] = struct {
				Reason       string
				RemainingTTL int64
			}{
				Reason:       mark.Reason,
				RemainingTTL: int64(mark.ExpireAt.Sub(now).Seconds()),
			}
		}
		return true
	})

	return result
}

// Count 统计不可用账户数量
func (m *UnavailableMarker) Count() int {
	count := 0
	now := time.Now()
	m.marks.Range(func(key, value interface{}) bool {
		mark := value.(*unavailableMark)
		if now.Before(mark.ExpireAt) {
			count++
		}
		return true
	})
	return count
}

// ClearAll 清除所有不可用标记
func (m *UnavailableMarker) ClearAll() int {
	count := 0
	m.marks.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	m.marks = sync.Map{}
	return count
}

// ==================== 统一管理 ====================

// MemoryCache 内存缓存统一管理
type MemoryCache struct {
	Sessions    *SessionStore
	Concurrency *ConcurrencyManager
	Unavailable *UnavailableMarker
}

// 全局内存缓存单例
var (
	globalMemoryCache *MemoryCache
	memoryCacheOnce   sync.Once
)

// GetMemoryCache 获取内存缓存单例
func GetMemoryCache() *MemoryCache {
	memoryCacheOnce.Do(func() {
		globalMemoryCache = &MemoryCache{
			Sessions:    GetSessionStore(),
			Concurrency: GetConcurrencyManager(),
			Unavailable: GetUnavailableMarker(),
		}
	})
	return globalMemoryCache
}

// Stop 停止所有清理任务
func (c *MemoryCache) Stop() {
	c.Sessions.StopCleanup()
	c.Unavailable.StopCleanup()
}

// Stats 获取缓存统计
func (c *MemoryCache) Stats() map[string]interface{} {
	accountConcurrency, userConcurrency := GetConcurrencyManager().Stats()
	return map[string]interface{}{
		"session_count":             c.Sessions.Count(),
		"unavailable_count":         c.Unavailable.Count(),
		"account_concurrency_count": accountConcurrency,
		"user_concurrency_count":    userConcurrency,
	}
}

// ClearAll 清除所有缓存
func (c *MemoryCache) ClearAll() map[string]int {
	return map[string]int{
		"sessions":    c.Sessions.ClearAll(),
		"unavailable": c.Unavailable.ClearAll(),
	}
}
