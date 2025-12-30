package scheduler

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"sync"
	"time"

	"go-aiproxy/internal/cache"
	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/pkg/logger"
)

var (
	ErrNoAvailableAccount = errors.New("no available account")
	ErrUnsupportedModel   = errors.New("unsupported model")
)

// Scheduler 调度器
type Scheduler struct {
	repo         *repository.AccountRepository
	sessionCache *cache.SessionCache
	mu           sync.RWMutex

	// 内存中的账户缓存
	accounts map[string][]*model.Account // platform -> accounts
	lastSync time.Time
}

var defaultScheduler *Scheduler
var once sync.Once

// GetScheduler 获取调度器单例
func GetScheduler() *Scheduler {
	once.Do(func() {
		defaultScheduler = &Scheduler{
			repo:         repository.NewAccountRepository(),
			sessionCache: cache.GetSessionCache(),
			accounts:     make(map[string][]*model.Account),
		}
		// 初始加载
		defaultScheduler.Refresh()

		// 启动定时恢复限流账号的任务
		go defaultScheduler.startRateLimitRecoveryTask()
	})
	return defaultScheduler
}

// GetSessionCache 获取会话缓存（供外部使用）
func (s *Scheduler) GetSessionCache() *cache.SessionCache {
	return s.sessionCache
}

// startRateLimitRecoveryTask 启动定时恢复限流账号的任务
func (s *Scheduler) startRateLimitRecoveryTask() {
	ticker := time.NewTicker(1 * time.Minute) // 每分钟检查一次
	defer ticker.Stop()

	for range ticker.C {
		recovered, err := s.repo.RecoverRateLimitedAccounts()
		if err != nil {
			continue
		}
		if recovered > 0 {
			// 刷新缓存以更新内存中的账号状态
			s.Refresh()
		}
	}
}

// Refresh 刷新账户缓存
func (s *Scheduler) Refresh() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	platforms := []string{model.PlatformClaude, model.PlatformOpenAI, model.PlatformGemini, model.PlatformOther}

	for _, platform := range platforms {
		accounts, err := s.repo.GetByPlatform(platform)
		if err != nil {
			continue
		}
		s.accounts[platform] = make([]*model.Account, len(accounts))
		for i := range accounts {
			s.accounts[platform][i] = &accounts[i]
		}
	}

	s.lastSync = time.Now()
	return nil
}

// SelectAccount 选择账户
func (s *Scheduler) SelectAccount(ctx context.Context, modelName string) (*model.Account, error) {
	return s.SelectAccountWithSession(ctx, modelName, "", 0, 0)
}

// SelectAccountWithSession 选择账户（支持会话粘性）
// userID 和 apiKeyID 用于记录会话绑定信息
func (s *Scheduler) SelectAccountWithSession(ctx context.Context, modelName string, sessionID string, userID uint, apiKeyID uint) (*model.Account, error) {
	platform := DetectPlatform(modelName)
	if platform == "" {
		return nil, ErrUnsupportedModel
	}

	// 检查会话粘性（从 Redis）
	if sessionID != "" && s.sessionCache != nil {
		binding, err := s.sessionCache.GetSessionBinding(ctx, sessionID)
		if err == nil && binding != nil && binding.Platform == platform {
			// 尝试获取绑定的账户
			s.mu.RLock()
			accounts := s.accounts[platform]
			s.mu.RUnlock()

			for _, acc := range accounts {
				if acc.ID == binding.AccountID && acc.Enabled && acc.Status == model.AccountStatusValid {
					// 账户可用，更新最后使用时间
					s.sessionCache.UpdateSessionLastUsed(ctx, sessionID)
					return acc, nil
				}
			}
			// 账户不可用，移除会话绑定
			s.sessionCache.RemoveSessionBinding(ctx, sessionID)
		}
	}

	s.mu.RLock()
	accounts := s.accounts[platform]
	s.mu.RUnlock()

	if len(accounts) == 0 {
		// 尝试刷新
		s.Refresh()

		s.mu.RLock()
		accounts = s.accounts[platform]
		s.mu.RUnlock()

		if len(accounts) == 0 {
			return nil, ErrNoAvailableAccount
		}
	}

	// 根据优先级和权重选择
	account := s.selectByWeight(accounts)

	// 绑定会话到 Redis
	if sessionID != "" && s.sessionCache != nil && account != nil {
		binding := &cache.SessionBinding{
			SessionID: sessionID,
			AccountID: account.ID,
			Platform:  account.Platform,
			Model:     modelName,
			UserID:    userID,
			APIKeyID:  apiKeyID,
		}
		s.sessionCache.SetSessionBinding(ctx, binding)
	}

	return account, nil
}

// SelectAccountByType 根据账户类型选择
func (s *Scheduler) SelectAccountByType(ctx context.Context, accountType string) (*model.Account, error) {
	accounts, err := s.repo.GetEnabledByType(accountType)
	if err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return nil, ErrNoAvailableAccount
	}

	accountPtrs := make([]*model.Account, len(accounts))
	for i := range accounts {
		accountPtrs[i] = &accounts[i]
	}

	return s.selectByWeight(accountPtrs), nil
}

// SelectAccountByTypesWithSession 根据多个账户类型选择（支持会话粘性）
func (s *Scheduler) SelectAccountByTypesWithSession(ctx context.Context, accountTypes []string, sessionID string, userID uint, apiKeyID uint) (*model.Account, error) {
	log := logger.GetLogger("scheduler")

	// 获取所有类型的账户
	var allAccounts []model.Account
	for _, accountType := range accountTypes {
		accounts, err := s.repo.GetEnabledByType(accountType)
		if err == nil {
			allAccounts = append(allAccounts, accounts...)
		}
	}

	if len(allAccounts) == 0 {
		return nil, ErrNoAvailableAccount
	}

	// 检查会话粘性（从 Redis）
	if sessionID != "" && s.sessionCache != nil {
		binding, err := s.sessionCache.GetSessionBinding(ctx, sessionID)
		if err == nil && binding != nil {
			for _, acc := range allAccounts {
				if acc.ID == binding.AccountID && acc.Enabled && acc.Status == model.AccountStatusValid {
					log.Info("会话粘性命中 - SessionID: %s, 账户ID: %d, 名称: %s", sessionID, acc.ID, acc.Name)
					s.sessionCache.UpdateSessionLastUsed(ctx, sessionID)
					return &acc, nil
				}
			}
			// 账户不可用，移除会话绑定
			s.sessionCache.RemoveSessionBinding(ctx, sessionID)
		}
	}

	// 转换为指针切片
	accountPtrs := make([]*model.Account, len(allAccounts))
	for i := range allAccounts {
		accountPtrs[i] = &allAccounts[i]
	}

	// 根据权重选择
	account := s.selectByWeight(accountPtrs)

	// 绑定会话到 Redis
	if sessionID != "" && s.sessionCache != nil && account != nil {
		binding := &cache.SessionBinding{
			SessionID: sessionID,
			AccountID: account.ID,
			Platform:  account.Platform,
			Model:     account.Type,
			UserID:    userID,
			APIKeyID:  apiKeyID,
		}
		s.sessionCache.SetSessionBinding(ctx, binding)
		log.Info("会话粘性绑定 - SessionID: %s, 账户ID: %d, 名称: %s, UserID: %d", sessionID, account.ID, account.Name, userID)
	}

	return account, nil
}

// SelectAccountByTypeWithSession 根据账户类型选择（支持会话粘性）
func (s *Scheduler) SelectAccountByTypeWithSession(ctx context.Context, accountType string, sessionID string, userID uint, apiKeyID uint) (*model.Account, error) {
	log := logger.GetLogger("scheduler")

	// 检查会话粘性（从 Redis）
	if sessionID != "" && s.sessionCache != nil {
		binding, err := s.sessionCache.GetSessionBinding(ctx, sessionID)
		if err == nil && binding != nil {
			// 验证绑定的账户类型是否匹配
			accounts, err := s.repo.GetEnabledByType(accountType)
			if err == nil {
				for _, acc := range accounts {
					if acc.ID == binding.AccountID && acc.Enabled && acc.Status == model.AccountStatusValid {
						log.Info("会话粘性命中 - SessionID: %s, 账户ID: %d, 名称: %s", sessionID, acc.ID, acc.Name)
						s.sessionCache.UpdateSessionLastUsed(ctx, sessionID)
						return &acc, nil
					}
				}
			}
			// 账户不可用或类型不匹配，移除会话绑定
			s.sessionCache.RemoveSessionBinding(ctx, sessionID)
		}
	}

	// 获取该类型的所有账户
	accounts, err := s.repo.GetEnabledByType(accountType)
	if err != nil {
		return nil, err
	}

	if len(accounts) == 0 {
		return nil, ErrNoAvailableAccount
	}

	// 转换为指针切片
	accountPtrs := make([]*model.Account, len(accounts))
	for i := range accounts {
		accountPtrs[i] = &accounts[i]
	}

	// 根据权重选择
	account := s.selectByWeight(accountPtrs)

	// 绑定会话到 Redis
	if sessionID != "" && s.sessionCache != nil && account != nil {
		binding := &cache.SessionBinding{
			SessionID: sessionID,
			AccountID: account.ID,
			Platform:  account.Platform,
			Model:     accountType, // 使用账户类型作为模型标识
			UserID:    userID,
			APIKeyID:  apiKeyID,
		}
		s.sessionCache.SetSessionBinding(ctx, binding)
		log.Info("会话粘性绑定 - SessionID: %s, 账户ID: %d, 名称: %s, UserID: %d", sessionID, account.ID, account.Name, userID)
	}

	return account, nil
}

// selectByWeight 根据权重选择账户
func (s *Scheduler) selectByWeight(accounts []*model.Account) *model.Account {
	if len(accounts) == 1 {
		return accounts[0]
	}

	// 计算总权重
	totalWeight := 0
	for _, acc := range accounts {
		// 优先级 * 权重
		totalWeight += acc.Priority * acc.Weight
	}

	if totalWeight == 0 {
		return accounts[rand.Intn(len(accounts))]
	}

	// 随机选择
	r := rand.Intn(totalWeight)
	for _, acc := range accounts {
		r -= acc.Priority * acc.Weight
		if r < 0 {
			return acc
		}
	}

	return accounts[0]
}

// MarkAccountError 标记账户错误
func (s *Scheduler) MarkAccountError(accountID uint, accountType string, err error) {
	s.MarkAccountErrorWithReset(accountID, accountType, err, nil)
}

// MarkAccountErrorWithReset 标记账户错误，支持设置限流恢复时间
func (s *Scheduler) MarkAccountErrorWithReset(accountID uint, accountType string, err error, resetAt *time.Time) {
	log := logger.GetLogger("scheduler")
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	errMsgLower := strings.ToLower(errMsg)

	// 根据错误类型决定状态
	status := model.AccountStatusValid
	if strings.Contains(errMsgLower, "rate limit") || strings.Contains(errMsg, "429") {
		status = model.AccountStatusRateLimited
		// 如果没有提供恢复时间，默认设置为1小时后
		if resetAt == nil {
			defaultReset := time.Now().Add(1 * time.Hour)
			resetAt = &defaultReset
		}
	} else if strings.Contains(errMsgLower, "overloaded") || strings.Contains(errMsg, "529") {
		status = model.AccountStatusOverloaded
	} else if strings.Contains(errMsgLower, "token") && strings.Contains(errMsgLower, "expired") ||
		strings.Contains(errMsgLower, "authentication_error") ||
		strings.Contains(errMsgLower, "oauth token has expired") ||
		strings.Contains(errMsgLower, "invalid") || strings.Contains(errMsg, "401") || strings.Contains(errMsg, "403") {
		// Token 过期、认证失败等错误 - 标记为 invalid 状态
		// 但 claude-console 类型账户不检测失效（Session Key 方式）
		if accountType != model.AccountTypeClaudeConsole {
			status = model.AccountStatusInvalid
			log.Warn("账户已失效 - AccountID: %d, Error: %s", accountID, truncateString(errMsg, 200))
		}
	}

	if resetAt != nil && status == model.AccountStatusRateLimited {
		s.repo.UpdateStatusWithRateLimit(accountID, status, errMsg, resetAt)
	} else {
		s.repo.UpdateStatus(accountID, status, errMsg)
	}

	// 账户失效时自动禁用
	if status == model.AccountStatusInvalid {
		s.repo.SetEnabled(accountID, false)
	}

	s.repo.IncrementErrorCount(accountID)
}

// MarkAccountSuccess 标记账户成功
func (s *Scheduler) MarkAccountSuccess(accountID uint) {
	s.repo.IncrementRequestCount(accountID)
	// 如果之前是错误状态，恢复正常
	s.repo.UpdateStatus(accountID, model.AccountStatusValid, "")
}

// DetectPlatform 根据模型名检测平台
func DetectPlatform(modelName string) string {
	modelLower := strings.ToLower(modelName)

	// Claude 模型
	if strings.HasPrefix(modelLower, "claude") {
		return model.PlatformClaude
	}

	// OpenAI 模型
	if strings.HasPrefix(modelLower, "gpt") ||
		strings.HasPrefix(modelLower, "o1") ||
		strings.HasPrefix(modelLower, "o3") ||
		strings.HasPrefix(modelLower, "text-") ||
		strings.HasPrefix(modelLower, "davinci") ||
		strings.HasPrefix(modelLower, "curie") ||
		strings.HasPrefix(modelLower, "babbage") ||
		strings.HasPrefix(modelLower, "ada") {
		return model.PlatformOpenAI
	}

	// Gemini 模型
	if strings.HasPrefix(modelLower, "gemini") ||
		strings.HasPrefix(modelLower, "models/gemini") {
		return model.PlatformGemini
	}

	return ""
}

// DetectAccountType 根据模型名检测账户类型（用于更精确的路由）
func DetectAccountType(modelName string) string {
	// 支持 "type,model" 格式，如 "ccr,claude-3-5-sonnet"
	if strings.Contains(modelName, ",") {
		parts := strings.SplitN(modelName, ",", 2)
		return parts[0]
	}

	return ""
}

// GetActualModel 获取实际模型名（去掉前缀）
func GetActualModel(modelName string) string {
	if strings.Contains(modelName, ",") {
		parts := strings.SplitN(modelName, ",", 2)
		return parts[1]
	}
	return modelName
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
