/*
 * 文件作用：账户调度器，负责从多个AI平台账户中选择合适的账户处理请求
 * 负责功能：
 *   - 账户选择（按模型、按类型、按权重）
 *   - 会话粘性（同一会话路由到同一账户）
 *   - AllowedModels 过滤（账户可用模型限制）
 *   - ModelMapping 映射处理（模型名转换）
 *   - 账户状态管理（错误标记、限流恢复）
 *   - 定时恢复限流账户
 * 重要程度：⭐⭐⭐⭐⭐ 核心（代理转发的核心调度逻辑）
 * 依赖模块：cache, model, repository, adapter
 */
package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"strings"
	"sync"
	"time"

	"go-aiproxy/internal/cache"
	"go-aiproxy/internal/errormatch"
	"go-aiproxy/internal/model"
	"go-aiproxy/internal/proxy/adapter"
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
					// 检查账户是否允许当前模型
					if !s.isModelAllowed(acc, modelName) {
						// 模型不被允许，移除会话绑定，重新选择
						s.sessionCache.RemoveSessionBinding(ctx, sessionID)
						break
					}
					// 账户可用且模型允许，更新最后使用时间
					s.sessionCache.UpdateSessionLastUsed(ctx, sessionID)
					return acc, nil
				}
			}
			// 账户不可用或模型不允许，移除会话绑定
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

	// 根据 AllowedModels 过滤账户
	accounts = s.filterByAllowedModels(accounts, modelName)
	if len(accounts) == 0 {
		return nil, ErrNoAvailableAccount
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
// modelName 用于根据账户的 AllowedModels 进行过滤（可选，传空字符串表示不过滤）
func (s *Scheduler) SelectAccountByType(ctx context.Context, accountType string, modelName string) (*model.Account, error) {
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

	// 根据 AllowedModels 过滤账户
	accountPtrs = s.filterByAllowedModels(accountPtrs, modelName)
	if len(accountPtrs) == 0 {
		return nil, ErrNoAvailableAccount
	}

	return s.selectByWeight(accountPtrs), nil
}

// SelectAccountByTypesWithSession 根据多个账户类型选择（支持会话粘性）
// modelName 用于根据账户的 AllowedModels 进行过滤
func (s *Scheduler) SelectAccountByTypesWithSession(ctx context.Context, accountTypes []string, modelName string, sessionID string, userID uint, apiKeyID uint) (*model.Account, error) {
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

	// 转换为指针切片
	accountPtrs := make([]*model.Account, len(allAccounts))
	for i := range allAccounts {
		accountPtrs[i] = &allAccounts[i]
	}

	// 根据 AllowedModels 过滤账户
	accountPtrs = s.filterByAllowedModels(accountPtrs, modelName)
	if len(accountPtrs) == 0 {
		return nil, ErrNoAvailableAccount
	}

	// 检查会话粘性（从 Redis）
	if sessionID != "" && s.sessionCache != nil {
		binding, err := s.sessionCache.GetSessionBinding(ctx, sessionID)
		if err == nil && binding != nil {
			for _, acc := range accountPtrs {
				if acc.ID == binding.AccountID && acc.Enabled && acc.Status == model.AccountStatusValid {
					log.Info("会话粘性命中 - SessionID: %s, 账户ID: %d, 名称: %s", sessionID, acc.ID, acc.Name)
					s.sessionCache.UpdateSessionLastUsed(ctx, sessionID)
					return acc, nil
				}
			}
			// 账户不可用，移除会话绑定
			s.sessionCache.RemoveSessionBinding(ctx, sessionID)
		}
	}

	// 根据权重选择
	account := s.selectByWeight(accountPtrs)

	// 绑定会话到 Redis
	if sessionID != "" && s.sessionCache != nil && account != nil {
		binding := &cache.SessionBinding{
			SessionID: sessionID,
			AccountID: account.ID,
			Platform:  account.Platform,
			Model:     GetActualModel(modelName),
			UserID:    userID,
			APIKeyID:  apiKeyID,
		}
		s.sessionCache.SetSessionBinding(ctx, binding)
		log.Info("会话粘性绑定 - SessionID: %s, 账户ID: %d, 名称: %s, UserID: %d", sessionID, account.ID, account.Name, userID)
	}

	return account, nil
}

// SelectAccountByTypeWithSession 根据账户类型选择（支持会话粘性）
// modelName 用于根据账户的 AllowedModels 进行过滤
func (s *Scheduler) SelectAccountByTypeWithSession(ctx context.Context, accountType string, modelName string, sessionID string, userID uint, apiKeyID uint) (*model.Account, error) {
	log := logger.GetLogger("scheduler")

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

	// 根据 AllowedModels 过滤账户
	accountPtrs = s.filterByAllowedModels(accountPtrs, modelName)
	if len(accountPtrs) == 0 {
		return nil, ErrNoAvailableAccount
	}

	// 检查会话粘性（从 Redis）
	if sessionID != "" && s.sessionCache != nil {
		binding, err := s.sessionCache.GetSessionBinding(ctx, sessionID)
		if err == nil && binding != nil {
			for _, acc := range accountPtrs {
				if acc.ID == binding.AccountID && acc.Enabled && acc.Status == model.AccountStatusValid {
					log.Info("会话粘性命中 - SessionID: %s, 账户ID: %d, 名称: %s", sessionID, acc.ID, acc.Name)
					s.sessionCache.UpdateSessionLastUsed(ctx, sessionID)
					return acc, nil
				}
			}
			// 账户不可用或类型不匹配，移除会话绑定
			s.sessionCache.RemoveSessionBinding(ctx, sessionID)
		}
	}

	// 根据权重选择
	account := s.selectByWeight(accountPtrs)

	// 绑定会话到 Redis
	if sessionID != "" && s.sessionCache != nil && account != nil {
		binding := &cache.SessionBinding{
			SessionID: sessionID,
			AccountID: account.ID,
			Platform:  account.Platform,
			Model:     GetActualModel(modelName),
			UserID:    userID,
			APIKeyID:  apiKeyID,
		}
		s.sessionCache.SetSessionBinding(ctx, binding)
		log.Info("会话粘性绑定 - SessionID: %s, 账户ID: %d, 名称: %s, UserID: %d", sessionID, account.ID, account.Name, userID)
	}

	return account, nil
}

// filterByAllowedModels 根据 AllowedModels 过滤账户
// 如果账户设置了 AllowedModels，则只有请求的模型在列表中才返回该账户
// 如果账户没有设置 AllowedModels（空），则该账户可用于所有模型
func (s *Scheduler) filterByAllowedModels(accounts []*model.Account, modelName string) []*model.Account {
	return s.filterByAllowedModelsWithOriginal(accounts, modelName, "")
}

// filterByAllowedModelsWithOriginal 根据 AllowedModels 和账户 ModelMapping 过滤账户
// mappedModel: 请求的模型名
// originalModel: 原始模型名（用于检查账户的 ModelMapping）
//
// 过滤逻辑：
// 1. 如果账户配置了 ModelMapping 且包含原始模型，使用映射后的模型检查 AllowedModels
// 2. 否则直接用请求模型检查 AllowedModels
// 3. 如果账户没有设置 AllowedModels，则允许所有模型
func (s *Scheduler) filterByAllowedModelsWithOriginal(accounts []*model.Account, mappedModel string, originalModel string) []*model.Account {
	log := logger.GetLogger("scheduler")

	if mappedModel == "" {
		return accounts
	}

	var filtered []*model.Account

	for _, acc := range accounts {
		// 确定用于 AllowedModels 检查的模型名
		checkModel := mappedModel

		// 如果账户配置了 ModelMapping，检查原始模型是否在映射中
		if originalModel != "" && acc.ModelMapping != "" {
			targetModel := getAccountMappedModel(acc, originalModel)
			if targetModel != "" {
				// 原始模型在 ModelMapping 中，使用映射后的模型检查 AllowedModels
				checkModel = targetModel
				log.Debug("账户 ModelMapping 命中 - ID: %d, Name: %s, 原始模型: %s -> 目标模型: %s",
					acc.ID, acc.Name, originalModel, targetModel)
			} else {
				// 账户配置了 ModelMapping 但不包含原始模型，跳过
				log.Debug("账户 ModelMapping 不包含原始模型 - ID: %d, Name: %s, ModelMapping: %s, OriginalModel: %s",
					acc.ID, acc.Name, acc.ModelMapping, originalModel)
				continue
			}
		}

		// 检查 AllowedModels
		allowedByModel := false
		if acc.AllowedModels == "" {
			// 没有设置 AllowedModels 限制
			allowedByModel = true
		} else {
			// 检查模型是否在允许列表中
			checkModelLower := strings.ToLower(checkModel)
			allowedList := strings.Split(acc.AllowedModels, ",")
			for _, allowed := range allowedList {
				allowed = strings.TrimSpace(strings.ToLower(allowed))
				if allowed == "" {
					continue
				}
				// 支持前缀匹配，如 "claude-3" 可以匹配 "claude-3-5-sonnet"
				if strings.HasPrefix(checkModelLower, allowed) || allowed == checkModelLower {
					allowedByModel = true
					break
				}
			}
		}

		if !allowedByModel {
			log.Debug("账户 AllowedModels 不匹配 - ID: %d, Name: %s, AllowedModels: %s, CheckModel: %s",
				acc.ID, acc.Name, acc.AllowedModels, checkModel)
			continue
		}

		filtered = append(filtered, acc)
	}

	return filtered
}

// isModelAllowed 检查单个账户是否允许指定模型
func (s *Scheduler) isModelAllowed(acc *model.Account, modelName string) bool {
	if modelName == "" || acc.AllowedModels == "" {
		// 没有设置限制，允许所有模型
		return true
	}

	modelLower := strings.ToLower(modelName)
	allowedList := strings.Split(acc.AllowedModels, ",")
	for _, allowed := range allowedList {
		allowed = strings.TrimSpace(strings.ToLower(allowed))
		if allowed == "" {
			continue
		}
		// 支持前缀匹配
		if strings.HasPrefix(modelLower, allowed) || allowed == modelLower {
			return true
		}
	}
	return false
}

// parseAccountModelMapping 解析账户的模型映射 JSON
// 返回 map[sourceModel]targetModel
func parseAccountModelMapping(acc *model.Account) map[string]string {
	if acc.ModelMapping == "" {
		return nil
	}

	var mapping map[string]string
	if err := json.Unmarshal([]byte(acc.ModelMapping), &mapping); err != nil {
		return nil
	}
	return mapping
}

// hasAccountModelMapping 检查账户是否配置了指定原始模型的映射
// originalModel: 客户端请求的原始模型名（映射前）
func hasAccountModelMapping(acc *model.Account, originalModel string) bool {
	if acc.ModelMapping == "" {
		// 账户没有配置模型映射，表示不限制
		return true
	}

	mapping := parseAccountModelMapping(acc)
	if mapping == nil {
		// 解析失败，默认不限制
		return true
	}

	// 检查原始模型是否在账户的映射配置中
	originalLower := strings.ToLower(originalModel)
	for sourceModel := range mapping {
		sourceLower := strings.ToLower(sourceModel)
		// 支持前缀匹配
		if strings.HasPrefix(originalLower, sourceLower) || sourceLower == originalLower {
			return true
		}
	}

	return false
}

// getAccountMappedModel 获取账户 ModelMapping 中原始模型对应的目标模型
// 返回映射后的模型名，如果没有找到返回空字符串
func getAccountMappedModel(acc *model.Account, originalModel string) string {
	if acc.ModelMapping == "" {
		return ""
	}

	mapping := parseAccountModelMapping(acc)
	if mapping == nil {
		return ""
	}

	// 查找原始模型对应的目标模型
	originalLower := strings.ToLower(originalModel)
	for sourceModel, targetModel := range mapping {
		sourceLower := strings.ToLower(sourceModel)
		// 支持前缀匹配
		if strings.HasPrefix(originalLower, sourceLower) || sourceLower == originalLower {
			return targetModel
		}
	}

	return ""
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

	// 根据错误类型决定状态
	status := model.AccountStatusValid

	// API Key 模式（ClaudeConsole）不做限流处理，直接透传上游错误
	if accountType == model.AccountTypeClaudeConsole {
		// API Key 模式只记录错误，不改变状态，但保存错误信息
		s.repo.IncrementErrorCount(accountID)
		s.repo.UpdateLastError(accountID, errMsg)
		return
	}

	// 1. 首先尝试从 UpstreamError 获取 HTTP 状态码（类型断言）
	var httpStatusCode int
	var upstreamErr *adapter.UpstreamError
	if errors.As(err, &upstreamErr) {
		httpStatusCode = upstreamErr.StatusCode
	}

	// 2. 使用错误规则匹配器进行匹配（先匹配状态码，再匹配关键词）
	matcher := errormatch.GetErrorRuleMatcher()
	result := matcher.Match(httpStatusCode, errMsg)

	if result.Matched {
		// 匹配到规则，使用规则指定的目标状态
		status = result.TargetStatus

		// 日志记录
		if result.Rule != nil {
			log.Info("错误规则匹配 - AccountID: %d, HTTP: %d, 规则ID: %d, 目标状态: %s, 描述: %s",
				accountID, httpStatusCode, result.Rule.ID, status, result.Rule.Description)
		}

		// 如果是限流状态，设置恢复时间
		if status == model.AccountStatusRateLimited && resetAt == nil {
			defaultReset := time.Now().Add(1 * time.Hour)
			resetAt = &defaultReset
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
		log.Warn("账户已失效并禁用 - AccountID: %d, Error: %s", accountID, truncateString(errMsg, 200))
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
// 优先从数据库查询，查不到再用硬编码兜底
func DetectPlatform(modelName string) string {
	// 1. 先从数据库查询
	repo := repository.NewAIModelRepository(repository.GetDB())

	// 精确匹配
	m, err := repo.GetByName(modelName)
	if err == nil && m != nil && m.Platform != "" {
		return m.Platform
	}

	// 别名匹配
	platform, err := repo.FindPlatformByModelName(modelName)
	if err == nil && platform != "" {
		return platform
	}

	// 2. 数据库查不到，用硬编码兜底
	return detectPlatformFallback(modelName)
}

// detectPlatformFallback 硬编码的平台检测（兜底）
func detectPlatformFallback(modelName string) string {
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
		strings.HasPrefix(modelLower, "davinci") {
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
	// 支持 "type,model" 格式，如 "bedrock,claude-3-5-sonnet"
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
