/*
 * 文件作用：账号健康检查服务，定期检测AI平台账号的可用性
 * 负责功能：
 *   - 定时健康检查调度
 *   - 单个账号健康检测
 *   - 账号状态自动恢复
 *   - Token刷新
 *   - OAuth重新授权冷却控制
 * 重要程度：⭐⭐⭐⭐ 重要（账号可用性保障）
 * 依赖模块：repository, adapter, scheduler, logger
 */
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/proxy/adapter"
	"go-aiproxy/internal/proxy/scheduler"
	"go-aiproxy/internal/repository"
	"go-aiproxy/pkg/logger"
)

// AccountHealthCheckService 账号健康检查服务
type AccountHealthCheckService struct {
	accountRepo   *repository.AccountRepository
	configService *ConfigService
	log           *logger.Logger

	mu           sync.Mutex
	running      bool
	stopChan     chan struct{}
	lastCheck    time.Time
	checkedCount int
	failedCount  int
	lastError    error

	// OAuth 重新授权冷却记录
	reauthorizeCooldown map[uint]time.Time
	cooldownMu          sync.RWMutex
}

var healthCheckService *AccountHealthCheckService
var healthCheckOnce sync.Once

// GetAccountHealthCheckService 获取健康检查服务单例
func GetAccountHealthCheckService() *AccountHealthCheckService {
	healthCheckOnce.Do(func() {
		healthCheckService = &AccountHealthCheckService{
			accountRepo:         repository.NewAccountRepository(),
			configService:       GetConfigService(),
			log:                 logger.GetLogger("health_check"),
			stopChan:            make(chan struct{}),
			reauthorizeCooldown: make(map[uint]time.Time),
		}
	})
	return healthCheckService
}

// Start 启动健康检查任务
func (s *AccountHealthCheckService) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	// 启动两个检测循环
	go s.normalAccountLoop()  // 正常账号检测循环（较慢）
	go s.problemAccountLoop() // 问题账号检测循环（较快）

	s.log.Info("账号健康检查服务已启动（分级检测模式）")
}

// Stop 停止健康检查任务
func (s *AccountHealthCheckService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	close(s.stopChan)
	s.running = false
	s.log.Info("账号健康检查服务已停止")
}

// Restart 重启健康检查任务（配置变更时调用）
func (s *AccountHealthCheckService) Restart() {
	s.Stop()
	time.Sleep(100 * time.Millisecond)
	s.Start()
}

// normalAccountLoop 正常账号检测循环（使用配置的间隔）
func (s *AccountHealthCheckService) normalAccountLoop() {
	// 首次启动时等待一个间隔再检查
	interval := s.configService.GetAccountHealthCheckInterval()
	if interval < time.Minute {
		interval = 5 * time.Minute
	}

	select {
	case <-time.After(interval / 2):
	case <-s.stopChan:
		return
	}

	// 执行第一次检查
	if s.configService.GetAccountHealthCheckEnabled() {
		s.doNormalCheck()
	}

	for {
		interval = s.configService.GetAccountHealthCheckInterval()
		if interval < time.Minute {
			interval = 5 * time.Minute
		}

		select {
		case <-time.After(interval):
			if s.configService.GetAccountHealthCheckEnabled() {
				s.doNormalCheck()
			}
		case <-s.stopChan:
			return
		}
	}
}

// problemAccountLoop 问题账号检测循环（每分钟检查一次是否有账号需要探测）
func (s *AccountHealthCheckService) problemAccountLoop() {
	// 等待服务稳定后开始
	select {
	case <-time.After(30 * time.Second):
	case <-s.stopChan:
		return
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if s.configService.GetAccountHealthCheckEnabled() {
				s.doProblemAccountCheck()
			}
		case <-s.stopChan:
			return
		}
	}
}

// doProblemAccountCheck 检测问题账号
func (s *AccountHealthCheckService) doProblemAccountCheck() {
	// 获取需要探测的账号（到达检测时间的）
	accounts, err := s.accountRepo.GetAccountsNeedingProbe()
	if err != nil {
		s.log.Error("获取问题账号列表失败: %v", err)
		return
	}

	if len(accounts) == 0 {
		return
	}

	s.log.Debug("发现 %d 个问题账号需要探测", len(accounts))

	// 获取全局默认代理
	defaultProxy, _ := GetProxyService().GetDefaultProxy()

	// 并发检查，限制并发数
	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup

	for _, account := range accounts {
		if account.Proxy == nil && defaultProxy != nil {
			account.Proxy = defaultProxy
		}

		wg.Add(1)
		go func(acc model.Account) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			s.checkProblemAccount(&acc)
		}(account)
	}

	wg.Wait()
}

// checkProblemAccount 检测单个问题账号，根据状态采取不同策略
func (s *AccountHealthCheckService) checkProblemAccount(account *model.Account) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch account.Status {
	case model.AccountStatusRateLimited, model.AccountStatusOverloaded:
		s.handleRateLimitedAccount(ctx, account)

	case model.AccountStatusTokenExpired:
		s.handleTokenExpiredAccount(ctx, account)

	case model.AccountStatusSuspended:
		s.handleSuspendedAccount(ctx, account)

	case model.AccountStatusBanned:
		s.handleBannedAccount(ctx, account)
	}
}

// handleRateLimitedAccount 处理限流账号
func (s *AccountHealthCheckService) handleRateLimitedAccount(ctx context.Context, account *model.Account) {
	if !s.configService.GetRateLimitedProbeEnabled() {
		return
	}

	healthy, errMsg := s.checkAccount(account)

	if healthy {
		// 恢复成功
		if s.configService.GetHealthCheckAutoRecovery() {
			if err := s.accountRepo.RecoverAccount(account.ID); err != nil {
				s.log.Error("[%s] 恢复账号失败: %v", account.Name, err)
			} else {
				s.log.Info("[%s] 限流账号已恢复正常", account.Name)
				scheduler.GetScheduler().Refresh()
			}
		}
	} else {
		// 仍然限流，计算下次探测时间（间隔递增）
		currentInterval := account.HealthCheckInterval
		if currentInterval == 0 {
			currentInterval = int(s.configService.GetRateLimitedProbeInitInterval().Seconds())
		} else {
			// 间隔递增
			backoff := s.configService.GetRateLimitedProbeBackoff()
			currentInterval = int(float64(currentInterval) * backoff)
			maxInterval := int(s.configService.GetRateLimitedProbeMaxInterval().Seconds())
			if currentInterval > maxInterval {
				currentInterval = maxInterval
			}
		}

		nextCheck := time.Now().Add(time.Duration(currentInterval) * time.Second)
		if err := s.accountRepo.UpdateHealthCheckSchedule(account.ID, nextCheck, currentInterval); err != nil {
			s.log.Error("[%s] 更新检测计划失败: %v", account.Name, err)
		}

		s.log.Debug("[%s] 限流中，下次探测: %v 后 (原因: %s)",
			account.Name, time.Duration(currentInterval)*time.Second, truncateMsg(errMsg, 100))
	}
}

// handleTokenExpiredAccount 处理 Token 过期账号
func (s *AccountHealthCheckService) handleTokenExpiredAccount(ctx context.Context, account *model.Account) {
	if !s.configService.GetHealthCheckAutoTokenRefresh() {
		return
	}

	// 检查冷却时间
	if s.isInCooldown(account.ID) {
		s.log.Debug("[%s] Token 刷新在冷却中", account.Name)
		return
	}

	// 尝试刷新 Token
	if account.SessionKey != "" {
		reauthorized, err := s.tryReauthorizeWithSessionKey(ctx, account)
		if reauthorized {
			s.log.Info("[%s] Token 刷新成功，账号已恢复", account.Name)
			s.clearCooldown(account.ID)

			if err := s.accountRepo.RecoverAccount(account.ID); err != nil {
				s.log.Error("[%s] 恢复账号失败: %v", account.Name, err)
			}
			scheduler.GetScheduler().Refresh()
			return
		}

		// 刷新失败，设置冷却
		s.setCooldown(account.ID)

		// 检查是否账号被封
		if IsAccountBannedError(err) {
			s.log.Warn("[%s] Token 刷新失败，账号疑似被封: %v", account.Name, err)
			s.accountRepo.MarkAsSuspended(account.ID, err.Error())
			return
		}

		s.log.Warn("[%s] Token 刷新失败: %v", account.Name, err)
	}

	// 没有 SessionKey 或刷新失败，安排下次检测
	cooldown := s.configService.GetTokenRefreshCooldown()
	nextCheck := time.Now().Add(cooldown)
	s.accountRepo.UpdateHealthCheckSchedule(account.ID, nextCheck, int(cooldown.Seconds()))
}

// handleSuspendedAccount 处理疑似封号账号
func (s *AccountHealthCheckService) handleSuspendedAccount(ctx context.Context, account *model.Account) {
	healthy, errMsg := s.checkAccount(account)

	if healthy {
		// 恢复成功
		if s.configService.GetHealthCheckAutoRecovery() {
			if err := s.accountRepo.RecoverAccount(account.ID); err != nil {
				s.log.Error("[%s] 恢复账号失败: %v", account.Name, err)
			} else {
				s.log.Info("[%s] 疑似封号账号已恢复正常", account.Name)
				scheduler.GetScheduler().Refresh()
			}
		}
	} else {
		// 检测失败，增加计数
		count, err := s.accountRepo.IncrementSuspendedCount(account.ID)
		if err != nil {
			s.log.Error("[%s] 增加疑似封号计数失败: %v", account.Name, err)
			return
		}

		threshold := s.configService.GetSuspendedConfirmThreshold()
		s.log.Warn("[%s] 疑似封号检测失败 [%d/%d]: %s", account.Name, count, threshold, truncateMsg(errMsg, 100))

		if count >= threshold {
			// 确认封号
			if err := s.accountRepo.MarkAsBanned(account.ID, errMsg); err != nil {
				s.log.Error("[%s] 标记封号失败: %v", account.Name, err)
			} else {
				s.log.Warn("[%s] 连续 %d 次检测失败，确认封号", account.Name, count)
				scheduler.GetScheduler().Refresh()
			}
		} else {
			// 安排下次检测
			interval := s.configService.GetSuspendedProbeInterval()
			nextCheck := time.Now().Add(interval)
			s.accountRepo.UpdateHealthCheckSchedule(account.ID, nextCheck, int(interval.Seconds()))
		}
	}
}

// handleBannedAccount 处理已封号账号（复活检测）
func (s *AccountHealthCheckService) handleBannedAccount(ctx context.Context, account *model.Account) {
	if !s.configService.GetBannedProbeEnabled() {
		return
	}

	healthy, errMsg := s.checkAccount(account)

	if healthy {
		// 意外恢复！
		if s.configService.GetHealthCheckAutoRecovery() {
			if err := s.accountRepo.RecoverAccount(account.ID); err != nil {
				s.log.Error("[%s] 恢复账号失败: %v", account.Name, err)
			} else {
				s.log.Info("[%s] 封号账号意外恢复正常！", account.Name)
				scheduler.GetScheduler().Refresh()
			}
		}
	} else {
		// 仍然封号，安排下次检测
		interval := s.configService.GetBannedProbeInterval()
		nextCheck := time.Now().Add(interval)
		s.accountRepo.UpdateHealthCheckSchedule(account.ID, nextCheck, int(interval.Seconds()))
		s.log.Debug("[%s] 封号账号检测仍失败，下次检测: %v 后", account.Name, interval)
		_ = errMsg // 忽略错误信息
	}
}

// TriggerCheck 手动触发健康检查
func (s *AccountHealthCheckService) TriggerCheck() {
	go s.doNormalCheck()
}

// TriggerSingleCheck 手动触发单个账号检查
func (s *AccountHealthCheckService) TriggerSingleCheck(accountID uint) (bool, string) {
	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		return false, fmt.Sprintf("获取账号失败: %v", err)
	}

	// 获取全局默认代理
	if account.Proxy == nil {
		defaultProxy, _ := GetProxyService().GetDefaultProxy()
		if defaultProxy != nil {
			account.Proxy = defaultProxy
		}
	}

	healthy, errMsg := s.checkAccount(account)

	if healthy {
		// 自动恢复
		if s.configService.GetHealthCheckAutoRecovery() {
			if err := s.accountRepo.RecoverAccount(accountID); err != nil {
				s.log.Error("[%s] 恢复账号失败: %v", account.Name, err)
			} else {
				s.log.Info("[%s] 手动检测通过，账号已恢复", account.Name)
				scheduler.GetScheduler().Refresh()
			}
		}
		return true, "检测通过，账号正常"
	}

	// 检测失败，根据错误类型更新账号状态
	s.updateAccountStatusByError(account, errMsg)

	return false, errMsg
}

// updateAccountStatusByError 根据错误信息更新账号状态
func (s *AccountHealthCheckService) updateAccountStatusByError(account *model.Account, errMsg string) {
	errLower := strings.ToLower(errMsg)

	// 判断错误类型并更新状态
	if strings.Contains(errLower, "401") || strings.Contains(errLower, "认证失败") ||
		strings.Contains(errLower, "token") || strings.Contains(errLower, "expired") ||
		strings.Contains(errLower, "unauthorized") {
		// Token 过期
		if account.Status != model.AccountStatusTokenExpired {
			if err := s.accountRepo.MarkAsTokenExpired(account.ID, errMsg); err != nil {
				s.log.Error("[%s] 标记 Token 过期失败: %v", account.Name, err)
			} else {
				s.log.Warn("[%s] 检测失败，标记为 Token 过期: %s", account.Name, truncateMsg(errMsg, 100))
				scheduler.GetScheduler().Refresh()
			}
		}
	} else if strings.Contains(errLower, "403") || strings.Contains(errLower, "封") ||
		strings.Contains(errLower, "banned") || strings.Contains(errLower, "suspended") ||
		strings.Contains(errLower, "disabled") || strings.Contains(errLower, "permission") {
		// 疑似封号
		if account.Status != model.AccountStatusSuspended && account.Status != model.AccountStatusBanned {
			if err := s.accountRepo.MarkAsSuspended(account.ID, errMsg); err != nil {
				s.log.Error("[%s] 标记疑似封号失败: %v", account.Name, err)
			} else {
				s.log.Warn("[%s] 检测失败，标记为疑似封号: %s", account.Name, truncateMsg(errMsg, 100))
				scheduler.GetScheduler().Refresh()
			}
		}
	} else if strings.Contains(errLower, "429") || strings.Contains(errLower, "rate") ||
		strings.Contains(errLower, "限流") {
		// 限流（通常 429 会被判定为健康，但以防万一）
		if account.Status != model.AccountStatusRateLimited {
			resetAt := time.Now().Add(30 * time.Minute) // 默认 30 分钟后重试
			if err := s.accountRepo.MarkAsRateLimited(account.ID, &resetAt, errMsg); err != nil {
				s.log.Error("[%s] 标记限流失败: %v", account.Name, err)
			} else {
				s.log.Warn("[%s] 检测失败，标记为限流: %s", account.Name, truncateMsg(errMsg, 100))
				scheduler.GetScheduler().Refresh()
			}
		}
	} else {
		// 其他错误，标记为无效
		if account.Status != model.AccountStatusInvalid {
			if err := s.accountRepo.MarkAsInvalid(account.ID, errMsg); err != nil {
				s.log.Error("[%s] 标记无效失败: %v", account.Name, err)
			} else {
				s.log.Warn("[%s] 检测失败，标记为无效: %s", account.Name, truncateMsg(errMsg, 100))
				scheduler.GetScheduler().Refresh()
			}
		}
	}
}

// ForceRecover 强制恢复账号（跳过检测）
func (s *AccountHealthCheckService) ForceRecover(accountID uint) error {
	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		return fmt.Errorf("获取账号失败: %v", err)
	}

	if err := s.accountRepo.RecoverAccount(accountID); err != nil {
		return fmt.Errorf("恢复账号失败: %v", err)
	}

	s.log.Info("[%s] 账号已强制恢复", account.Name)
	scheduler.GetScheduler().Refresh()
	return nil
}

// RefreshToken 手动刷新 Token
func (s *AccountHealthCheckService) RefreshToken(accountID uint) error {
	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		return fmt.Errorf("获取账号失败: %v", err)
	}

	if account.SessionKey == "" {
		return fmt.Errorf("账号没有 SessionKey，无法刷新 Token")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reauthorized, reauthorizeErr := s.tryReauthorizeWithSessionKey(ctx, account)
	if reauthorized {
		s.log.Info("[%s] Token 刷新成功", account.Name)
		s.clearCooldown(accountID)

		// 如果账号状态不是 valid，恢复它
		if account.Status != model.AccountStatusValid {
			s.accountRepo.RecoverAccount(accountID)
		}

		scheduler.GetScheduler().Refresh()
		return nil
	}

	return fmt.Errorf("Token 刷新失败: %v", reauthorizeErr)
}

// doNormalCheck 执行正常账号健康检查（原有逻辑）
func (s *AccountHealthCheckService) doNormalCheck() {
	s.log.Info("开始正常账号健康检查...")
	startTime := time.Now()

	// 获取需要健康检查的账号（只检查 valid 状态的）
	accounts, err := s.accountRepo.GetAccountsForHealthCheck()
	if err != nil {
		s.lastError = err
		s.log.Error("获取账号列表失败: %v", err)
		return
	}

	if len(accounts) == 0 {
		s.log.Info("没有需要检查的正常账号")
		return
	}

	// 获取全局默认代理
	defaultProxy, err := GetProxyService().GetDefaultProxy()
	if err != nil {
		s.log.Warn("获取默认代理失败: %v", err)
	}

	checkedCount := 0
	failedCount := 0
	threshold := s.configService.GetAccountErrorThreshold()

	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	for _, account := range accounts {
		if account.Proxy == nil && defaultProxy != nil {
			account.Proxy = defaultProxy
		}
		wg.Add(1)
		go func(acc model.Account) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			healthy, errMsg := s.checkAccount(&acc)
			checkedCount++

			if healthy {
				if acc.ConsecutiveErrorCount > 0 {
					if err := s.accountRepo.ResetConsecutiveErrorCount(acc.ID); err != nil {
						s.log.Error("[%s] 重置错误计数失败: %v", acc.Name, err)
					}
				}
			} else {
				failedCount++
				newCount, err := s.accountRepo.IncrementConsecutiveErrorCount(acc.ID)
				if err != nil {
					s.log.Error("[%s] 增加错误计数失败: %v", acc.Name, err)
					return
				}

				s.log.Warn("[%s] 健康检查失败 [%d/%d]: %s",
					acc.Name, newCount, threshold, truncateMsg(errMsg, 200))

				if newCount >= threshold {
					// 不再直接禁用，而是标记为疑似封号
					if err := s.accountRepo.MarkAsSuspended(acc.ID, errMsg); err != nil {
						s.log.Error("[%s] 标记疑似封号失败: %v", acc.Name, err)
					} else {
						s.log.Warn("[%s] 连续错误达到阈值 %d，标记为疑似封号", acc.Name, threshold)
						scheduler.GetScheduler().Refresh()
					}
				}
			}
		}(account)
	}

	wg.Wait()

	s.lastCheck = time.Now()
	s.checkedCount = checkedCount
	s.failedCount = failedCount
	s.lastError = nil

	duration := time.Since(startTime)
	s.log.Info("正常账号健康检查完成，共检查 %d 个，失败 %d 个，耗时 %v",
		checkedCount, failedCount, duration)
}

// checkAccount 检查单个账号的健康状态
// 返回: (是否健康, 错误信息)
func (s *AccountHealthCheckService) checkAccount(account *model.Account) (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	switch account.Type {
	case model.AccountTypeClaudeOfficial:
		return s.checkClaudeOfficial(ctx, account)
	case model.AccountTypeOpenAIResponses:
		return s.checkOpenAIResponses(ctx, account)
	case model.AccountTypeGemini:
		return s.checkGemini(ctx, account)
	default:
		// 不支持的账号类型，跳过检查
		return true, ""
	}
}

// checkClaudeOfficial 检查 Claude Official 账号
// 支持两种认证方式：
// 1. OAuth (AccessToken): 通过 /api/oauth/usage 验证
// 2. SessionKey: 通过 /api/organizations 验证
// 如果两种方式都有，优先用 OAuth，OAuth 失败时尝试用 SessionKey 重新授权
func (s *AccountHealthCheckService) checkClaudeOfficial(ctx context.Context, account *model.Account) (bool, string) {
	// 优先使用 OAuth (AccessToken) 验证
	if account.AccessToken != "" {
		healthy, errMsg := s.checkClaudeOAuth(ctx, account)
		if healthy {
			return true, ""
		}

		// OAuth 失败，如果有 SessionKey 则尝试重新授权
		if account.SessionKey != "" {
			// 检查是否启用自动重新授权
			if !s.configService.GetOAuthAutoReauthorizeEnabled() {
				s.log.Debug("[%s] OAuth 验证失败，自动重新授权已禁用，尝试 SessionKey 验证", account.Name)
				return s.checkClaudeSessionKey(ctx, account)
			}

			// 检查是否在冷却时间内
			if s.isInCooldown(account.ID) {
				s.log.Debug("[%s] OAuth 验证失败，重新授权在冷却中，尝试 SessionKey 验证", account.Name)
				return s.checkClaudeSessionKey(ctx, account)
			}

			s.log.Info("[%s] OAuth 验证失败 (%s)，尝试用 SessionKey 重新授权", account.Name, errMsg)

			// 尝试重新授权
			reauthorized, reauthorizeErr := s.tryReauthorizeWithSessionKey(ctx, account)
			if reauthorized {
				s.log.Info("[%s] 重新授权成功，账号已恢复", account.Name)
				// 清除冷却时间
				s.clearCooldown(account.ID)
				return true, ""
			}

			// 设置冷却时间
			s.setCooldown(account.ID)

			// 检查是否账号被封
			if IsAccountBannedError(reauthorizeErr) {
				s.log.Warn("[%s] 账号可能已被封禁: %v", account.Name, reauthorizeErr)
				return false, reauthorizeErr.Error()
			}

			// 重新授权失败，再尝试 SessionKey 验证
			s.log.Debug("[%s] 重新授权失败 (%v)，尝试 SessionKey 验证", account.Name, reauthorizeErr)
			return s.checkClaudeSessionKey(ctx, account)
		}
		// 没有 SessionKey，返回 OAuth 的错误
		return false, errMsg
	}

	// 只有 SessionKey
	if account.SessionKey != "" {
		return s.checkClaudeSessionKey(ctx, account)
	}

	// 两者都没有，返回错误
	return false, "AccessToken 和 SessionKey 都为空"
}

// isInCooldown 检查账号是否在重新授权冷却时间内
func (s *AccountHealthCheckService) isInCooldown(accountID uint) bool {
	s.cooldownMu.RLock()
	defer s.cooldownMu.RUnlock()

	lastAttempt, exists := s.reauthorizeCooldown[accountID]
	if !exists {
		return false
	}

	cooldownDuration := s.configService.GetOAuthReauthorizeCooldown()
	return time.Since(lastAttempt) < cooldownDuration
}

// setCooldown 设置账号的重新授权冷却时间
func (s *AccountHealthCheckService) setCooldown(accountID uint) {
	s.cooldownMu.Lock()
	defer s.cooldownMu.Unlock()
	s.reauthorizeCooldown[accountID] = time.Now()
}

// clearCooldown 清除账号的重新授权冷却时间
func (s *AccountHealthCheckService) clearCooldown(accountID uint) {
	s.cooldownMu.Lock()
	defer s.cooldownMu.Unlock()
	delete(s.reauthorizeCooldown, accountID)
}

// tryReauthorizeWithSessionKey 尝试用 SessionKey 重新授权获取新的 OAuth Token
// 返回: (是否成功, 错误)
func (s *AccountHealthCheckService) tryReauthorizeWithSessionKey(ctx context.Context, account *model.Account) (bool, error) {
	oauthService := GetOAuthAuthService()

	tokenResult, err := oauthService.ReauthorizeWithSessionKey(ctx, account)
	if err != nil {
		return false, err
	}

	// 更新账号的 Token
	expiry := time.Now().Add(time.Duration(tokenResult.ExpiresIn) * time.Second)
	err = s.accountRepo.UpdateToken(account.ID, tokenResult.AccessToken, tokenResult.RefreshToken, &expiry)
	if err != nil {
		s.log.Error("[%s] 更新 Token 失败: %v", account.Name, err)
		return false, err
	}

	// 同时更新内存中的账号信息
	account.AccessToken = tokenResult.AccessToken
	account.RefreshToken = tokenResult.RefreshToken
	account.TokenExpiry = &expiry

	// 如果返回了组织 UUID，也更新
	if tokenResult.OrgUUID != "" && account.OrganizationID != tokenResult.OrgUUID {
		s.log.Info("[%s] 更新组织 UUID: %s -> %s", account.Name, account.OrganizationID, tokenResult.OrgUUID)
		account.OrganizationID = tokenResult.OrgUUID
		// 这里可以额外调用 repo 更新组织 ID
	}

	s.log.Info("[%s] Token 已更新，过期时间: %v, scope: %s",
		account.Name, expiry.Format(time.RFC3339), tokenResult.Scope)

	// 刷新调度器缓存
	scheduler.GetScheduler().Refresh()

	return true, nil
}

// checkClaudeOAuth 检查 Claude OAuth 账号
// 通过调用 /api/oauth/usage 来验证账号有效性
func (s *AccountHealthCheckService) checkClaudeOAuth(ctx context.Context, account *model.Account) (bool, string) {
	client := adapter.GetSmartHTTPClient(account, "https://api.anthropic.com")

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.anthropic.com/api/oauth/usage", nil)
	if err != nil {
		return false, fmt.Sprintf("创建请求失败: %v", err)
	}

	// 设置必要的请求头 (参考 claude-relay-service)
	req.Header.Set("Authorization", "Bearer "+account.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("User-Agent", "claude-cli/2.0.53 (external, cli)")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// 检查状态码
	if resp.StatusCode == 200 {
		s.log.Debug("[%s] OAuth 验证成功", account.Name)
		return true, ""
	}

	// 403 表示认证失败（可能是 Setup Token 或账号被封）
	if resp.StatusCode == 403 {
		// 尝试解析错误信息
		var errResp struct {
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			return false, fmt.Sprintf("OAuth 认证失败: %s", errResp.Error.Message)
		}
		return false, fmt.Sprintf("OAuth 认证失败 (HTTP 403)")
	}

	// 401 表示认证失败
	if resp.StatusCode == 401 {
		return false, fmt.Sprintf("认证失败 (HTTP 401): %s", string(body))
	}

	// 429 表示限流，账号仍然有效
	if resp.StatusCode == 429 {
		s.log.Debug("[%s] OAuth 验证: 限流中但账号有效", account.Name)
		return true, ""
	}

	// 其他错误
	return false, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
}

// checkClaudeSessionKey 检查 Claude SessionKey 账号
// 通过调用 /api/organizations 来验证账号有效性
func (s *AccountHealthCheckService) checkClaudeSessionKey(ctx context.Context, account *model.Account) (bool, string) {
	client := adapter.GetSmartHTTPClient(account, "https://claude.ai")

	req, err := http.NewRequestWithContext(ctx, "GET", "https://claude.ai/api/organizations", nil)
	if err != nil {
		return false, fmt.Sprintf("创建请求失败: %v", err)
	}

	// 设置必要的请求头
	req.Header.Set("Cookie", fmt.Sprintf("sessionKey=%s", account.SessionKey))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体（用于错误信息）
	body, _ := io.ReadAll(resp.Body)

	// 检查状态码
	if resp.StatusCode == 200 {
		s.log.Debug("[%s] SessionKey 验证成功", account.Name)
		return true, ""
	}

	// 401/403 表示认证失败
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return false, fmt.Sprintf("认证失败 (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// 429 表示限流，账号仍然有效
	if resp.StatusCode == 429 {
		s.log.Debug("[%s] SessionKey 验证: 限流中但账号有效", account.Name)
		return true, ""
	}

	// 其他错误
	return false, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
}

// checkOpenAIResponses 检查 OpenAI Responses 账号
// 支持两种认证方式：
// 1. API Key: 通过 /v1/models 验证
// 2. OAuth Token (access_token/session_key): 通过 ChatGPT backend-api 验证
func (s *AccountHealthCheckService) checkOpenAIResponses(ctx context.Context, account *model.Account) (bool, string) {
	// 如果使用 API Key，通过 /v1/models 验证
	if account.APIKey != "" {
		client := adapter.GetSmartHTTPClient(account, "https://api.openai.com")

		baseURL := "https://api.openai.com"
		if account.BaseURL != "" {
			baseURL = account.BaseURL
		}

		req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/v1/models", nil)
		if err != nil {
			return false, fmt.Sprintf("创建请求失败: %v", err)
		}

		req.Header.Set("Authorization", "Bearer "+account.APIKey)
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return false, fmt.Sprintf("请求失败: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == 200 {
			return true, ""
		}

		// 401/403 表示认证失败
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			var errResp struct {
				Error struct {
					Message string `json:"message"`
					Code    string `json:"code"`
				} `json:"error"`
			}
			if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
				return false, fmt.Sprintf("认证失败: %s", errResp.Error.Message)
			}
			return false, fmt.Sprintf("认证失败 (HTTP %d)", resp.StatusCode)
		}

		// 429 表示限流，账号仍然有效
		if resp.StatusCode == 429 {
			return true, ""
		}

		return false, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// 如果使用 OAuth Token（access_token 或 session_key），通过 ChatGPT backend-api 验证
	token := account.AccessToken
	if token == "" {
		token = account.SessionKey
	}
	if token != "" {
		return s.checkChatGPTOAuth(ctx, account, token)
	}

	// 没有任何认证信息
	return false, "APIKey、AccessToken 和 SessionKey 都为空"
}

// checkChatGPTOAuth 检查 ChatGPT OAuth Token
// 通过调用 ChatGPT backend-api 来验证 token 有效性
func (s *AccountHealthCheckService) checkChatGPTOAuth(ctx context.Context, account *model.Account, token string) (bool, string) {
	client := adapter.GetSmartHTTPClient(account, "https://chatgpt.com")

	// ChatGPT 账户信息检查 API
	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://chatgpt.com/backend-api/accounts/check/v4-2023-04-27", nil)
	if err != nil {
		return false, fmt.Sprintf("创建请求失败: %v", err)
	}

	// 设置请求头（模拟浏览器）
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// 检查状态码
	if resp.StatusCode == 200 {
		// 解析响应，检查是否有账户信息
		var result struct {
			Accounts map[string]interface{} `json:"accounts"`
		}
		if json.Unmarshal(body, &result) == nil {
			// 过滤掉 "default" key
			accountCount := 0
			for key := range result.Accounts {
				if key != "default" {
					accountCount++
				}
			}
			if accountCount > 0 {
				s.log.Debug("[%s] ChatGPT OAuth 验证成功，账户数: %d", account.Name, accountCount)
				return true, ""
			}
			return false, "未找到有效账户"
		}
		// 即使解析失败，200 也认为成功
		s.log.Debug("[%s] ChatGPT OAuth 验证成功", account.Name)
		return true, ""
	}

	// 401/403 表示认证失败
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		// 尝试解析错误信息
		var errResp struct {
			Detail string `json:"detail"`
		}
		errMsg := ""
		if json.Unmarshal(body, &errResp) == nil && errResp.Detail != "" {
			errMsg = errResp.Detail
		} else {
			errMsg = string(body)
			if len(errMsg) > 200 {
				errMsg = errMsg[:200] + "..."
			}
		}
		return false, fmt.Sprintf("认证失败 (HTTP %d): %s", resp.StatusCode, errMsg)
	}

	// 429 表示限流，账号仍然有效
	if resp.StatusCode == 429 {
		s.log.Debug("[%s] ChatGPT OAuth 验证: 限流中但账号有效", account.Name)
		return true, ""
	}

	// 其他错误
	errMsg := string(body)
	if len(errMsg) > 200 {
		errMsg = errMsg[:200] + "..."
	}
	return false, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, errMsg)
}

// checkGemini 检查 Gemini OAuth 账号
// 通过调用模型列表接口来验证账号有效性
func (s *AccountHealthCheckService) checkGemini(ctx context.Context, account *model.Account) (bool, string) {
	// 使用 AccessToken 认证
	if account.AccessToken == "" {
		return false, "AccessToken 为空"
	}

	client := adapter.GetHTTPClient(account)

	// Gemini API 端点
	baseURL := "https://generativelanguage.googleapis.com"
	if account.BaseURL != "" {
		baseURL = account.BaseURL
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/v1beta/models", nil)
	if err != nil {
		return false, fmt.Sprintf("创建请求失败: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+account.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 200 {
		return true, ""
	}

	// 401/403 表示认证失败
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		// 尝试解析 Google API 错误格式
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Status  string `json:"status"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			return false, fmt.Sprintf("认证失败: %s", errResp.Error.Message)
		}
		return false, fmt.Sprintf("认证失败 (HTTP %d)", resp.StatusCode)
	}

	// 429 表示限流，账号仍然有效
	if resp.StatusCode == 429 {
		return true, ""
	}

	return false, fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
}

// GetStatus 获取健康检查服务状态
func (s *AccountHealthCheckService) GetStatus() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 获取问题账号数量
	problemAccounts, _ := s.accountRepo.GetProblemAccounts()

	status := map[string]interface{}{
		"enabled":       s.configService.GetAccountHealthCheckEnabled(),
		"interval":      s.configService.GetAccountHealthCheckInterval().Minutes(),
		"threshold":     s.configService.GetAccountErrorThreshold(),
		"running":       s.running,
		"last_check":    nil,
		"checked_count": s.checkedCount,
		"failed_count":  s.failedCount,
		"last_error":    nil,
		// 新增：问题账号统计
		"problem_account_count": len(problemAccounts),
		// 新增：各状态配置
		"config": map[string]interface{}{
			"auto_recovery":             s.configService.GetHealthCheckAutoRecovery(),
			"auto_token_refresh":        s.configService.GetHealthCheckAutoTokenRefresh(),
			"rate_limited_probe":        s.configService.GetRateLimitedProbeEnabled(),
			"rate_limited_init_interval": s.configService.GetRateLimitedProbeInitInterval().Minutes(),
			"rate_limited_max_interval":  s.configService.GetRateLimitedProbeMaxInterval().Minutes(),
			"rate_limited_backoff":       s.configService.GetRateLimitedProbeBackoff(),
			"suspended_probe_interval":   s.configService.GetSuspendedProbeInterval().Minutes(),
			"suspended_confirm_threshold": s.configService.GetSuspendedConfirmThreshold(),
			"banned_probe":               s.configService.GetBannedProbeEnabled(),
			"banned_probe_interval":      s.configService.GetBannedProbeInterval().Hours(),
			"token_refresh_cooldown":     s.configService.GetTokenRefreshCooldown().Minutes(),
			"token_refresh_max_retries":  s.configService.GetTokenRefreshMaxRetries(),
		},
	}

	if !s.lastCheck.IsZero() {
		status["last_check"] = s.lastCheck.Format(time.RFC3339)
	}

	if s.lastError != nil {
		status["last_error"] = s.lastError.Error()
	}

	return status
}

// OnConfigChange 配置变更回调
func (s *AccountHealthCheckService) OnConfigChange(key, value string) {
	switch key {
	case model.ConfigAccountHealthCheckEnabled:
		if value == "true" {
			s.Start()
		} else {
			s.Stop()
		}
	case model.ConfigAccountHealthCheckInterval:
		// 重启以应用新的间隔
		if s.running {
			s.Restart()
		}
	}
}

// truncateMsg 截断过长的消息
func truncateMsg(msg string, maxLen int) string {
	if len(msg) > maxLen {
		return msg[:maxLen] + "..."
	}
	return msg
}
