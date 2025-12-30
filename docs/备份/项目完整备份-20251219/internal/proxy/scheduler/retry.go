package scheduler

import (
	"context"
	"errors"
	"io"
	"strings"
	"time"

	"go-aiproxy/internal/cache"
	"go-aiproxy/internal/model"
	"go-aiproxy/internal/proxy/adapter"
	"go-aiproxy/pkg/logger"
)

var (
	ErrAllAccountsFailed    = errors.New("all accounts failed")
	ErrMaxRetriesExceeded   = errors.New("max retries exceeded")
	ErrAccountConcurrencyFull = errors.New("account concurrency limit reached")
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries        int           // 最大重试次数
	RetryDelay        time.Duration // 重试延迟
	RetryBackoff      float64       // 退避系数
	RetryableErrors   []string      // 可重试的错误类型
	SwitchOnRateLimit bool          // 限流时是否切换账户
	SwitchOnError     bool          // 错误时是否切换账户
}

// DefaultRetryConfig 默认重试配置
var DefaultRetryConfig = RetryConfig{
	MaxRetries:        3,
	RetryDelay:        time.Second,
	RetryBackoff:      2.0,
	RetryableErrors:   []string{"timeout", "connection", "429", "529", "503", "502"},
	SwitchOnRateLimit: true,
	SwitchOnError:     true,
}

// RetryableRequest 可重试的请求
type RetryableRequest struct {
	Scheduler *Scheduler
	Config    RetryConfig
	SessionID string // 会话ID，用于会话粘性
	UserID    uint   // 用户ID
	APIKeyID  uint   // API Key ID
	ClientIP  string // 客户端IP

	// 已尝试的账户 ID，避免重复使用
	triedAccounts map[uint]bool
}

// NewRetryableRequest 创建可重试请求
func NewRetryableRequest(scheduler *Scheduler, config *RetryConfig) *RetryableRequest {
	cfg := DefaultRetryConfig
	if config != nil {
		cfg = *config
	}

	return &RetryableRequest{
		Scheduler:     scheduler,
		Config:        cfg,
		triedAccounts: make(map[uint]bool),
	}
}

// WithSessionID 设置会话ID（用于会话粘性）
func (r *RetryableRequest) WithSessionID(sessionID string) *RetryableRequest {
	r.SessionID = sessionID
	return r
}

// WithUserInfo 设置用户信息
func (r *RetryableRequest) WithUserInfo(userID, apiKeyID uint, clientIP string) *RetryableRequest {
	r.UserID = userID
	r.APIKeyID = apiKeyID
	r.ClientIP = clientIP
	return r
}

// ExecuteResult 执行结果
type ExecuteResult struct {
	Response  *adapter.Response
	AccountID uint
}

// ExecuteWithRetry 带重试的执行
func (r *RetryableRequest) ExecuteWithRetry(
	ctx context.Context,
	modelName string,
	execFunc func(ctx context.Context, account *model.Account) (*adapter.Response, error),
) (*ExecuteResult, error) {
	log := logger.GetLogger("scheduler")
	var lastErr error
	delay := r.Config.RetryDelay

	for attempt := 0; attempt <= r.Config.MaxRetries; attempt++ {
		// 选择账户
		account, err := r.selectNextAccount(ctx, modelName)
		if err != nil {
			if errors.Is(err, ErrNoAvailableAccount) {
				// 如果没有可用账户，检查是否还有重试机会
				if attempt < r.Config.MaxRetries {
					// 等待后重试，可能有账户恢复
					time.Sleep(delay)
					delay = time.Duration(float64(delay) * r.Config.RetryBackoff)
					// 重置已尝试账户列表，允许重新尝试
					r.triedAccounts = make(map[uint]bool)
					continue
				}
				return nil, ErrAllAccountsFailed
			}
			return nil, err
		}

		// 尝试获取并发槽位
		sessionCache := r.Scheduler.GetSessionCache()
		var acquired bool
		if sessionCache != nil {
			concurrencyLimit := account.MaxConcurrency
			if concurrencyLimit <= 0 {
				concurrencyLimit = 5 // 默认值
			}
			acquired, _, err = sessionCache.AcquireConcurrencyWithLimit(ctx, account.ID, concurrencyLimit)
			if err != nil {
				log.Warn("获取并发槽位失败: accountID=%d, error=%v", account.ID, err)
				// Redis 错误不阻止请求，继续执行
				acquired = true
			}
			if !acquired {
				log.Debug("账户并发已满: accountID=%d, limit=%d", account.ID, concurrencyLimit)
				// 标记该账户已尝试，选择下一个
				r.triedAccounts[account.ID] = true
				continue
			}
		}

		// 确保释放并发槽位
		releaseConcurrency := func() {
			if sessionCache != nil && acquired {
				sessionCache.ReleaseConcurrency(ctx, account.ID)
			}
		}

		// 执行请求
		resp, err := execFunc(ctx, account)

		if err == nil && resp.Error == nil {
			// 成功
			releaseConcurrency()
			r.Scheduler.MarkAccountSuccess(account.ID)
			return &ExecuteResult{
				Response:  resp,
				AccountID: account.ID,
			}, nil
		}

		// 释放并发槽位
		releaseConcurrency()

		// 记录错误
		actualErr := err
		if err == nil && resp.Error != nil {
			actualErr = errors.New(resp.Error.Message)
		}

		r.Scheduler.MarkAccountError(account.ID, account.Type, actualErr)
		lastErr = actualErr

		// 判断是否可以重试
		if !r.isRetryable(actualErr) {
			return &ExecuteResult{
				Response:  resp,
				AccountID: account.ID,
			}, actualErr
		}

		// 标记该账户已尝试
		r.triedAccounts[account.ID] = true

		// 如果不是最后一次尝试，等待后重试
		if attempt < r.Config.MaxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				delay = time.Duration(float64(delay) * r.Config.RetryBackoff)
			}
		}
	}

	return nil, lastErr
}

// StreamExecuteResult 流式执行结果
type StreamExecuteResult struct {
	Result    *adapter.StreamResult
	AccountID uint
}

// ExecuteStreamWithRetry 带重试的流式执行
func (r *RetryableRequest) ExecuteStreamWithRetry(
	ctx context.Context,
	modelName string,
	execFunc func(ctx context.Context, account *model.Account, writer io.Writer) (*adapter.StreamResult, error),
	writer io.Writer,
) (*StreamExecuteResult, error) {
	log := logger.GetLogger("scheduler")
	var lastErr error
	delay := r.Config.RetryDelay

	for attempt := 0; attempt <= r.Config.MaxRetries; attempt++ {
		// 选择账户
		account, err := r.selectNextAccount(ctx, modelName)
		if err != nil {
			if errors.Is(err, ErrNoAvailableAccount) {
				if attempt < r.Config.MaxRetries {
					time.Sleep(delay)
					delay = time.Duration(float64(delay) * r.Config.RetryBackoff)
					r.triedAccounts = make(map[uint]bool)
					continue
				}
				return nil, ErrAllAccountsFailed
			}
			return nil, err
		}

		// 尝试获取并发槽位
		sessionCache := r.Scheduler.GetSessionCache()
		var acquired bool
		if sessionCache != nil {
			concurrencyLimit := account.MaxConcurrency
			if concurrencyLimit <= 0 {
				concurrencyLimit = 5 // 默认值
			}
			acquired, _, err = sessionCache.AcquireConcurrencyWithLimit(ctx, account.ID, concurrencyLimit)
			if err != nil {
				log.Warn("获取并发槽位失败: accountID=%d, error=%v", account.ID, err)
				// Redis 错误不阻止请求，继续执行
				acquired = true
			}
			if !acquired {
				log.Debug("账户并发已满: accountID=%d, limit=%d", account.ID, concurrencyLimit)
				// 标记该账户已尝试，选择下一个
				r.triedAccounts[account.ID] = true
				continue
			}
		}

		// 确保释放并发槽位
		releaseConcurrency := func() {
			if sessionCache != nil && acquired {
				sessionCache.ReleaseConcurrency(ctx, account.ID)
			}
		}

		// 执行流式请求
		result, err := execFunc(ctx, account, writer)

		if err == nil {
			releaseConcurrency()
			r.Scheduler.MarkAccountSuccess(account.ID)
			return &StreamExecuteResult{
				Result:    result,
				AccountID: account.ID,
			}, nil
		}

		// 释放并发槽位
		releaseConcurrency()

		r.Scheduler.MarkAccountError(account.ID, account.Type, err)
		lastErr = err

		// 流式请求一旦开始就不应该重试（因为可能已经写入部分数据）
		// 除非是在连接阶段就失败了
		if !r.isConnectionError(err) {
			return nil, err
		}

		r.triedAccounts[account.ID] = true

		if attempt < r.Config.MaxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				delay = time.Duration(float64(delay) * r.Config.RetryBackoff)
			}
		}
	}

	return nil, lastErr
}

// selectNextAccount 选择下一个可用账户
func (r *RetryableRequest) selectNextAccount(ctx context.Context, modelName string) (*model.Account, error) {
	log := logger.GetLogger("scheduler")

	// 检测是否指定了账户类型
	accountType := DetectAccountType(modelName)
	actualModel := GetActualModel(modelName)
	platform := DetectPlatform(actualModel)

	log.Debug("选择账户 - 模型: %s, 账户类型: %s, 实际模型: %s, SessionID: %s", modelName, accountType, actualModel, r.SessionID)

	// 【会话粘性】首次尝试时检查会话绑定（从 Redis）
	if r.SessionID != "" && len(r.triedAccounts) == 0 {
		sessionCache := r.Scheduler.GetSessionCache()
		if sessionCache != nil {
			binding, err := sessionCache.GetSessionBinding(ctx, r.SessionID)
			targetPlatform := platform
			if accountType != "" {
				targetPlatform = accountType
			}
			if err == nil && binding != nil && binding.Platform == targetPlatform {
				// 尝试获取绑定的账户
				acc, err := r.Scheduler.repo.GetByID(binding.AccountID)
				if err == nil && acc != nil && acc.Enabled && acc.Status == model.AccountStatusValid {
					sessionCache.UpdateSessionLastUsed(ctx, r.SessionID)
					log.Info("会话粘性命中 - SessionID: %s, 账户ID: %d, 名称: %s", r.SessionID, acc.ID, acc.Name)
					return acc, nil
				}
				// 账户不可用，移除会话绑定并刷新调度器缓存
				log.Info("会话粘性账户不可用，移除绑定并刷新缓存 - SessionID: %s, 账户ID: %d", r.SessionID, binding.AccountID)
				sessionCache.RemoveSessionBinding(ctx, r.SessionID)
				r.Scheduler.Refresh()
			}
		}
	}

	var accounts []*model.Account

	if accountType != "" {
		// 根据类型获取账户
		accList, e := r.Scheduler.repo.GetEnabledByType(accountType)
		if e != nil {
			log.Error("获取账户失败 - 类型: %s, 错误: %v", accountType, e)
			return nil, e
		}
		accounts = make([]*model.Account, len(accList))
		for i := range accList {
			accounts[i] = &accList[i]
		}
		log.Debug("按类型获取账户 - 类型: %s, 账户数量: %d", accountType, len(accounts))
	} else {
		// 根据平台获取账户
		if platform == "" {
			log.Error("不支持的模型 - 模型: %s", actualModel)
			return nil, ErrUnsupportedModel
		}

		log.Debug("检测到平台: %s", platform)

		r.Scheduler.mu.RLock()
		platformAccounts := r.Scheduler.accounts[platform]
		r.Scheduler.mu.RUnlock()

		// 刷新缓存
		if len(platformAccounts) == 0 {
			log.Debug("平台 %s 无缓存账户，刷新缓存", platform)
			r.Scheduler.Refresh()
			r.Scheduler.mu.RLock()
			platformAccounts = r.Scheduler.accounts[platform]
			r.Scheduler.mu.RUnlock()
		}

		accounts = platformAccounts
		log.Debug("按平台获取账户 - 平台: %s, 账户数量: %d", platform, len(accounts))
	}

	// 过滤掉已尝试的账户和非正常状态的账户
	available := make([]*model.Account, 0, len(accounts))
	for _, acc := range accounts {
		if r.triedAccounts[acc.ID] {
			log.Debug("跳过已尝试账户 - ID: %d, 名称: %s", acc.ID, acc.Name)
			continue
		}
		if !acc.Enabled {
			log.Debug("跳过禁用账户 - ID: %d, 名称: %s", acc.ID, acc.Name)
			continue
		}
		// 如果没有明确指定账户类型，排除 openai-responses 类型（它需要特殊处理）
		if accountType == "" && acc.Type == model.AccountTypeOpenAIResponses {
			log.Debug("跳过 openai-responses 账户（需明确指定类型） - ID: %d, 名称: %s", acc.ID, acc.Name)
			continue
		}
		// 跳过无效账户
		if acc.Status == model.AccountStatusInvalid {
			log.Debug("跳过无效账户 - ID: %d, 名称: %s, 状态: %s, 错误: %s",
				acc.ID, acc.Name, acc.Status, acc.LastError)
			continue
		}
		// 如果配置了切换，跳过限流和过载的账户
		if r.Config.SwitchOnRateLimit {
			if acc.Status == model.AccountStatusRateLimited || acc.Status == model.AccountStatusOverloaded {
				log.Debug("跳过限流/过载账户 - ID: %d, 名称: %s, 状态: %s",
					acc.ID, acc.Name, acc.Status)
				continue
			}
		}
		log.Debug("可用账户 - ID: %d, 名称: %s, 类型: %s, 状态: %s",
			acc.ID, acc.Name, acc.Type, acc.Status)
		available = append(available, acc)
	}

	if len(available) == 0 {
		log.Warn("没有可用账户 - 模型: %s, 总账户数: %d", modelName, len(accounts))
		return nil, ErrNoAvailableAccount
	}

	selected := r.Scheduler.selectByWeight(available)

	// 【会话粘性】绑定新选中的账户（到 Redis）
	if r.SessionID != "" {
		sessionCache := r.Scheduler.GetSessionCache()
		if sessionCache != nil {
			binding := &cache.SessionBinding{
				SessionID: r.SessionID,
				AccountID: selected.ID,
				Platform:  selected.Platform,
				Model:     modelName,
				UserID:    r.UserID,
				APIKeyID:  r.APIKeyID,
				ClientIP:  r.ClientIP,
			}
			sessionCache.SetSessionBinding(ctx, binding)
			log.Info("会话粘性绑定 - SessionID: %s, 账户ID: %d, 名称: %s, UserID: %d", r.SessionID, selected.ID, selected.Name, r.UserID)
		}
	}

	log.Info("选中账户 - ID: %d, 名称: %s, 类型: %s", selected.ID, selected.Name, selected.Type)
	return selected, nil
}

// isRetryable 判断错误是否可重试
func (r *RetryableRequest) isRetryable(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	for _, retryable := range r.Config.RetryableErrors {
		if strings.Contains(errStr, strings.ToLower(retryable)) {
			return true
		}
	}

	return false
}

// isConnectionError 判断是否是连接错误（流式请求开始前的错误）
func (r *RetryableRequest) isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	connectionErrors := []string{
		"connection refused",
		"connection reset",
		"no such host",
		"timeout",
		"dial",
		"network",
	}

	for _, connErr := range connectionErrors {
		if strings.Contains(errStr, connErr) {
			return true
		}
	}

	return false
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	accountID     uint
	failureCount  int
	lastFailure   time.Time
	state         CircuitState

	// 配置
	FailureThreshold int           // 失败阈值
	RecoveryTime     time.Duration // 恢复时间
}

type CircuitState int

const (
	CircuitClosed CircuitState = iota // 正常
	CircuitOpen                       // 熔断
	CircuitHalfOpen                   // 半开
)

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(accountID uint) *CircuitBreaker {
	return &CircuitBreaker{
		accountID:        accountID,
		state:            CircuitClosed,
		FailureThreshold: 5,
		RecoveryTime:     time.Minute * 5,
	}
}

// Allow 检查是否允许请求
func (cb *CircuitBreaker) Allow() bool {
	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// 检查是否到了恢复时间
		if time.Since(cb.lastFailure) > cb.RecoveryTime {
			cb.state = CircuitHalfOpen
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	}
	return true
}

// RecordSuccess 记录成功
func (cb *CircuitBreaker) RecordSuccess() {
	cb.failureCount = 0
	cb.state = CircuitClosed
}

// RecordFailure 记录失败
func (cb *CircuitBreaker) RecordFailure() {
	cb.failureCount++
	cb.lastFailure = time.Now()

	if cb.failureCount >= cb.FailureThreshold {
		cb.state = CircuitOpen
	}
}

// GetState 获取状态
func (cb *CircuitBreaker) GetState() CircuitState {
	return cb.state
}
