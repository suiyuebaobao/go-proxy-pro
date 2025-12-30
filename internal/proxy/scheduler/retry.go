/*
 * 文件作用：请求重试机制，处理失败请求的自动重试和账户切换
 * 负责功能：
 *   - 请求重试配置（次数、延迟、退避系数）
 *   - 账户切换重试（失败后尝试其他账户）
 *   - 并发控制（账户并发限制）
 *   - 可重试错误判断（连接错误、限流等）
 *   - 流式/非流式请求重试
 * 重要程度：⭐⭐⭐⭐⭐ 核心（保证请求可靠性）
 * 依赖模块：cache, model, adapter
 */
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
	MaxRetries:        5,
	RetryDelay:        time.Second,
	RetryBackoff:      1.5,
	RetryableErrors:   []string{"timeout", "connection", "403", "429", "529", "503", "502"},
	SwitchOnRateLimit: true,
	SwitchOnError:     true,
}

// RetryableRequest 可重试的请求
type RetryableRequest struct {
	Scheduler     *Scheduler
	Config        RetryConfig
	SessionID     string // 会话ID，用于会话粘性
	UserID        uint   // 用户ID
	APIKeyID      uint   // API Key ID
	ClientIP      string // 客户端IP
	UserAgent     string // 客户端User-Agent
	OriginalModel string // 原始模型名（映射前），用于 AllowedModels 检查

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
func (r *RetryableRequest) WithUserInfo(userID, apiKeyID uint, clientIP, userAgent string) *RetryableRequest {
	r.UserID = userID
	r.APIKeyID = apiKeyID
	r.ClientIP = clientIP
	r.UserAgent = userAgent
	return r
}

// WithOriginalModel 设置原始模型名（映射前）
func (r *RetryableRequest) WithOriginalModel(originalModel string) *RetryableRequest {
	r.OriginalModel = originalModel
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
	startTime := time.Now()
	var lastErr error
	var lastAccount *model.Account
	var lastResp *adapter.Response
	delay := r.Config.RetryDelay

	// 记录每个账户的失败次数（用于最终标记状态）
	accountFailures := make(map[uint]int)

	// 记录请求开始
	log.InfoZ("代理请求开始",
		logger.String("model", modelName),
		logger.String("session_id", r.SessionID),
		logger.Uint("user_id", r.UserID),
		logger.Uint("api_key_id", r.APIKeyID),
		logger.String("client_ip", r.ClientIP),
		logger.Int("max_retries", r.Config.MaxRetries),
	)

	for attempt := 0; attempt <= r.Config.MaxRetries; attempt++ {
		// 选择账户（允许重试同一账户）
		account, err := r.selectNextAccountAllowRetry(ctx, modelName, accountFailures)
		if err != nil {
			if errors.Is(err, ErrNoAvailableAccount) {
				// 如果没有可用账户，检查是否还有重试机会
				if attempt < r.Config.MaxRetries {
					// 等待后重试，可能有账户恢复
					time.Sleep(delay)
					delay = time.Duration(float64(delay) * r.Config.RetryBackoff)
					continue
				}
				// 所有重试都失败，标记最后使用的账户错误
				if lastAccount != nil && lastErr != nil {
					r.Scheduler.MarkAccountError(lastAccount.ID, lastAccount.Type, lastErr)
				}
				log.ErrorZ("代理请求失败-无可用账户",
					logger.String("model", modelName),
					logger.Uint("user_id", r.UserID),
					logger.Uint("api_key_id", r.APIKeyID),
					logger.String("client_ip", r.ClientIP),
					logger.Duration("duration", time.Since(startTime)),
					logger.Int("attempts", attempt+1),
				)
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
				log.WarnZ("获取并发槽位失败",
					logger.Uint("account_id", account.ID),
					logger.String("account_name", account.Name),
					logger.Err(err),
				)
				// Redis 错误不阻止请求，继续执行
				acquired = true
			}
			if !acquired {
				log.WarnZ("账户并发已满",
					logger.Uint("account_id", account.ID),
					logger.String("account_name", account.Name),
					logger.Int("limit", concurrencyLimit),
				)
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

		// 记录开始执行
		execStart := time.Now()
		log.InfoZ("开始执行请求",
			logger.Int("attempt", attempt+1),
			logger.Int("max_attempts", r.Config.MaxRetries+1),
			logger.Uint("account_id", account.ID),
			logger.String("account_name", account.Name),
			logger.String("account_type", account.Type),
			logger.String("model", modelName),
			logger.Uint("user_id", r.UserID),
			logger.Uint("api_key_id", r.APIKeyID),
		)

		// 执行请求
		resp, err := execFunc(ctx, account)

		if err == nil && resp.Error == nil {
			// 成功
			releaseConcurrency()
			r.Scheduler.MarkAccountSuccess(account.ID)
			log.InfoZ("代理请求成功",
				logger.String("model", modelName),
				logger.Uint("account_id", account.ID),
				logger.String("account_name", account.Name),
				logger.String("account_type", account.Type),
				logger.Uint("user_id", r.UserID),
				logger.Uint("api_key_id", r.APIKeyID),
				logger.String("client_ip", r.ClientIP),
				logger.Int("input_tokens", resp.InputTokens),
				logger.Int("output_tokens", resp.OutputTokens),
				logger.Duration("exec_duration", time.Since(execStart)),
				logger.Duration("total_duration", time.Since(startTime)),
				logger.Int("attempts", attempt+1),
			)
			return &ExecuteResult{
				Response:  resp,
				AccountID: account.ID,
			}, nil
		}

		// 释放并发槽位
		releaseConcurrency()

		// 记录错误（但不立即标记账户状态）
		actualErr := err
		if err == nil && resp.Error != nil {
			actualErr = errors.New(resp.Error.Message)
		}

		lastErr = actualErr
		lastAccount = account
		lastResp = resp
		accountFailures[account.ID]++

		log.WarnZ("请求失败，准备重试",
			logger.Int("attempt", attempt+1),
			logger.Int("max_attempts", r.Config.MaxRetries+1),
			logger.Uint("account_id", account.ID),
			logger.String("account_name", account.Name),
			logger.String("account_type", account.Type),
			logger.String("model", modelName),
			logger.Uint("user_id", r.UserID),
			logger.Uint("api_key_id", r.APIKeyID),
			logger.String("client_ip", r.ClientIP),
			logger.String("error", actualErr.Error()),
			logger.Duration("exec_duration", time.Since(execStart)),
		)

		// 判断是否可以重试
		if !r.isRetryable(actualErr) {
			// 不可重试的错误，立即标记并返回
			r.Scheduler.MarkAccountError(account.ID, account.Type, actualErr)
			log.ErrorZ("代理请求失败-不可重试错误",
				logger.String("model", modelName),
				logger.Uint("account_id", account.ID),
				logger.String("account_name", account.Name),
				logger.Uint("user_id", r.UserID),
				logger.Uint("api_key_id", r.APIKeyID),
				logger.String("client_ip", r.ClientIP),
				logger.String("error", actualErr.Error()),
				logger.Duration("duration", time.Since(startTime)),
				logger.Int("attempts", attempt+1),
			)
			return &ExecuteResult{
				Response:  resp,
				AccountID: account.ID,
			}, actualErr
		}

		// 如果有多个账户，标记当前账户已尝试，下次优先选其他账户
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

	// 所有重试都失败，标记最后使用的账户错误
	if lastAccount != nil && lastErr != nil {
		r.Scheduler.MarkAccountError(lastAccount.ID, lastAccount.Type, lastErr)
	}

	log.ErrorZ("代理请求失败-重试耗尽",
		logger.String("model", modelName),
		logger.Uint("last_account_id", lastAccount.ID),
		logger.String("last_account_name", lastAccount.Name),
		logger.Uint("user_id", r.UserID),
		logger.Uint("api_key_id", r.APIKeyID),
		logger.String("client_ip", r.ClientIP),
		logger.String("error", lastErr.Error()),
		logger.Duration("duration", time.Since(startTime)),
		logger.Int("attempts", r.Config.MaxRetries+1),
	)

	return &ExecuteResult{
		Response:  lastResp,
		AccountID: lastAccount.ID,
	}, lastErr
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
	startTime := time.Now()
	var lastErr error
	var lastAccount *model.Account
	delay := r.Config.RetryDelay

	// 记录每个账户的失败次数
	accountFailures := make(map[uint]int)

	// 记录流式请求开始
	log.InfoZ("流式代理请求开始",
		logger.String("model", modelName),
		logger.String("session_id", r.SessionID),
		logger.Uint("user_id", r.UserID),
		logger.Uint("api_key_id", r.APIKeyID),
		logger.String("client_ip", r.ClientIP),
		logger.Int("max_retries", r.Config.MaxRetries),
	)

	for attempt := 0; attempt <= r.Config.MaxRetries; attempt++ {
		// 选择账户（允许重试同一账户）
		account, err := r.selectNextAccountAllowRetry(ctx, modelName, accountFailures)
		if err != nil {
			if errors.Is(err, ErrNoAvailableAccount) {
				if attempt < r.Config.MaxRetries {
					time.Sleep(delay)
					delay = time.Duration(float64(delay) * r.Config.RetryBackoff)
					continue
				}
				// 所有重试都失败，标记最后使用的账户错误
				if lastAccount != nil && lastErr != nil {
					r.Scheduler.MarkAccountError(lastAccount.ID, lastAccount.Type, lastErr)
				}
				log.ErrorZ("流式代理请求失败-无可用账户",
					logger.String("model", modelName),
					logger.Uint("user_id", r.UserID),
					logger.Uint("api_key_id", r.APIKeyID),
					logger.String("client_ip", r.ClientIP),
					logger.Duration("duration", time.Since(startTime)),
					logger.Int("attempts", attempt+1),
				)
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
				log.WarnZ("获取并发槽位失败",
					logger.Uint("account_id", account.ID),
					logger.String("account_name", account.Name),
					logger.Err(err),
				)
				// Redis 错误不阻止请求，继续执行
				acquired = true
			}
			if !acquired {
				log.WarnZ("账户并发已满",
					logger.Uint("account_id", account.ID),
					logger.String("account_name", account.Name),
					logger.Int("limit", concurrencyLimit),
				)
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

		// 记录开始执行
		execStart := time.Now()
		log.InfoZ("开始执行流式请求",
			logger.Int("attempt", attempt+1),
			logger.Int("max_attempts", r.Config.MaxRetries+1),
			logger.Uint("account_id", account.ID),
			logger.String("account_name", account.Name),
			logger.String("account_type", account.Type),
			logger.String("model", modelName),
			logger.Uint("user_id", r.UserID),
			logger.Uint("api_key_id", r.APIKeyID),
		)

		// 执行流式请求
		result, err := execFunc(ctx, account, writer)

		if err == nil {
			releaseConcurrency()
			r.Scheduler.MarkAccountSuccess(account.ID)
			log.InfoZ("流式代理请求成功",
				logger.String("model", modelName),
				logger.Uint("account_id", account.ID),
				logger.String("account_name", account.Name),
				logger.String("account_type", account.Type),
				logger.Uint("user_id", r.UserID),
				logger.Uint("api_key_id", r.APIKeyID),
				logger.String("client_ip", r.ClientIP),
				logger.Int("input_tokens", result.InputTokens),
				logger.Int("output_tokens", result.OutputTokens),
				logger.Int("cache_creation_tokens", result.CacheCreationInputTokens),
				logger.Int("cache_read_tokens", result.CacheReadInputTokens),
				logger.Duration("exec_duration", time.Since(execStart)),
				logger.Duration("total_duration", time.Since(startTime)),
				logger.Int("attempts", attempt+1),
			)
			return &StreamExecuteResult{
				Result:    result,
				AccountID: account.ID,
			}, nil
		}

		// 释放并发槽位
		releaseConcurrency()

		// 记录错误（但不立即标记账户状态）
		lastErr = err
		lastAccount = account
		accountFailures[account.ID]++

		log.WarnZ("流式请求失败，准备重试",
			logger.Int("attempt", attempt+1),
			logger.Int("max_attempts", r.Config.MaxRetries+1),
			logger.Uint("account_id", account.ID),
			logger.String("account_name", account.Name),
			logger.String("account_type", account.Type),
			logger.String("model", modelName),
			logger.Uint("user_id", r.UserID),
			logger.Uint("api_key_id", r.APIKeyID),
			logger.String("client_ip", r.ClientIP),
			logger.String("error", err.Error()),
			logger.Duration("exec_duration", time.Since(execStart)),
		)

		// 流式请求一旦开始就不应该重试（因为可能已经写入部分数据）
		// 除非是在连接阶段就失败了
		if !r.isConnectionError(err) {
			// 不可重试的错误，立即标记并返回
			r.Scheduler.MarkAccountError(account.ID, account.Type, err)
			log.ErrorZ("流式代理请求失败-不可重试错误",
				logger.String("model", modelName),
				logger.Uint("account_id", account.ID),
				logger.String("account_name", account.Name),
				logger.Uint("user_id", r.UserID),
				logger.Uint("api_key_id", r.APIKeyID),
				logger.String("client_ip", r.ClientIP),
				logger.String("error", err.Error()),
				logger.Duration("duration", time.Since(startTime)),
				logger.Int("attempts", attempt+1),
			)
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

	// 所有重试都失败，标记最后使用的账户错误
	if lastAccount != nil && lastErr != nil {
		r.Scheduler.MarkAccountError(lastAccount.ID, lastAccount.Type, lastErr)
	}

	log.ErrorZ("流式代理请求失败-重试耗尽",
		logger.String("model", modelName),
		logger.Uint("last_account_id", lastAccount.ID),
		logger.String("last_account_name", lastAccount.Name),
		logger.Uint("user_id", r.UserID),
		logger.Uint("api_key_id", r.APIKeyID),
		logger.String("client_ip", r.ClientIP),
		logger.String("error", lastErr.Error()),
		logger.Duration("duration", time.Since(startTime)),
		logger.Int("attempts", r.Config.MaxRetries+1),
	)

	return nil, lastErr
}

// selectNextAccount 选择下一个可用账户
func (r *RetryableRequest) selectNextAccount(ctx context.Context, modelName string) (*model.Account, error) {
	log := logger.GetLogger("scheduler")

	// 检测是否指定了账户类型
	accountType := DetectAccountType(modelName)
	actualModel := GetActualModel(modelName)
	platform := DetectPlatform(actualModel)

	// 获取原始模型名（用于检查账户 ModelMapping）
	originalModel := r.OriginalModel
	if originalModel == "" {
		originalModel = actualModel
	}

	log.Debug("选择账户 - 模型: %s, 账户类型: %s, 实际模型: %s, 原始模型: %s, SessionID: %s", modelName, accountType, actualModel, originalModel, r.SessionID)

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
					// 检查账户是否允许当前模型
					// 如果账户有 ModelMapping，需要用映射后的模型来检查 AllowedModels
					checkModel := actualModel
					sessionValid := true

					if acc.ModelMapping != "" {
						if mappedModel := getAccountMappedModel(acc, originalModel); mappedModel != "" {
							checkModel = mappedModel
							log.Debug("会话粘性检查使用映射模型 - 原始: %s -> 映射: %s", originalModel, mappedModel)
						} else {
							// 账户有 ModelMapping 但不包含原始模型，移除绑定
							log.Info("会话粘性账户 ModelMapping 不包含原始模型，移除绑定 - SessionID: %s, 账户ID: %d, 原始模型: %s, ModelMapping: %s",
								r.SessionID, acc.ID, originalModel, acc.ModelMapping)
							sessionCache.RemoveSessionBinding(ctx, r.SessionID)
							sessionValid = false
						}
					}

					if sessionValid && !r.Scheduler.isModelAllowed(acc, checkModel) {
						log.Info("会话粘性账户不允许该模型，移除绑定 - SessionID: %s, 账户ID: %d, 检查模型: %s, AllowedModels: %s",
							r.SessionID, acc.ID, checkModel, acc.AllowedModels)
						sessionCache.RemoveSessionBinding(ctx, r.SessionID)
						sessionValid = false
					}

					if sessionValid {
						sessionCache.UpdateSessionLastUsed(ctx, r.SessionID)
						log.Info("会话粘性命中 - SessionID: %s, 账户ID: %d, 名称: %s", r.SessionID, acc.ID, acc.Name)
						return acc, nil
					}
				} else {
					// 账户不可用，移除会话绑定并刷新调度器缓存
					log.Info("会话粘性账户不可用，移除绑定并刷新缓存 - SessionID: %s, 账户ID: %d", r.SessionID, binding.AccountID)
					sessionCache.RemoveSessionBinding(ctx, r.SessionID)
					r.Scheduler.Refresh()
				}
			}
		}
	}

	var accounts []*model.Account

	if accountType != "" {
		// 判断是平台前缀还是具体类型
		// 如果不包含 "-"，认为是平台前缀（如 claude, openai, gemini），使用前缀匹配
		// 如果包含 "-"，认为是具体类型（如 claude-console, openai-responses），使用精确匹配
		var accList []model.Account
		var e error
		if strings.Contains(accountType, "-") {
			// 具体类型，精确匹配
			accList, e = r.Scheduler.repo.GetEnabledByType(accountType)
			log.Debug("按类型精确匹配 - 类型: %s", accountType)
		} else {
			// 平台前缀，前缀匹配
			accList, e = r.Scheduler.repo.GetEnabledByTypePrefix(accountType)
			log.Debug("按类型前缀匹配 - 前缀: %s", accountType)
		}
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

	// 根据 AllowedModels 和 账户 ModelMapping 过滤账户
	accounts = r.Scheduler.filterByAllowedModelsWithOriginal(accounts, actualModel, originalModel)
	if len(accounts) == 0 {
		log.Warn("无可用账户(AllowedModels过滤后) - 模型: %s, 原始模型: %s", actualModel, originalModel)
		return nil, ErrNoAvailableAccount
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
				UserAgent: r.UserAgent,
			}
			sessionCache.SetSessionBinding(ctx, binding)
			log.Info("会话粘性绑定 - SessionID: %s, 账户ID: %d, 名称: %s, UserID: %d", r.SessionID, selected.ID, selected.Name, r.UserID)
		}
	}

	log.Info("选中账户 - ID: %d, 名称: %s, 类型: %s", selected.ID, selected.Name, selected.Type)
	return selected, nil
}

// selectNextAccountAllowRetry 选择下一个可用账户（允许重试同一账户）
// accountFailures 记录每个账户在本次请求中的失败次数
func (r *RetryableRequest) selectNextAccountAllowRetry(ctx context.Context, modelName string, accountFailures map[uint]int) (*model.Account, error) {
	log := logger.GetLogger("scheduler")

	// 检测是否指定了账户类型
	accountType := DetectAccountType(modelName)
	actualModel := GetActualModel(modelName)
	platform := DetectPlatform(actualModel)

	// 获取原始模型名（用于检查账户 ModelMapping）
	originalModel := r.OriginalModel
	if originalModel == "" {
		originalModel = actualModel
	}

	log.Debug("选择账户(允许重试) - 模型: %s, 账户类型: %s, 实际模型: %s, 原始模型: %s, SessionID: %s", modelName, accountType, actualModel, originalModel, r.SessionID)

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
					// 检查账户是否允许当前模型
					// 如果账户有 ModelMapping，需要用映射后的模型来检查 AllowedModels
					checkModel := actualModel
					sessionValid := true

					if acc.ModelMapping != "" {
						if mappedModel := getAccountMappedModel(acc, originalModel); mappedModel != "" {
							checkModel = mappedModel
							log.Debug("会话粘性检查使用映射模型 - 原始: %s -> 映射: %s", originalModel, mappedModel)
						} else {
							// 账户有 ModelMapping 但不包含原始模型，移除绑定
							log.Info("会话粘性账户 ModelMapping 不包含原始模型，移除绑定 - SessionID: %s, 账户ID: %d, 原始模型: %s, ModelMapping: %s",
								r.SessionID, acc.ID, originalModel, acc.ModelMapping)
							sessionCache.RemoveSessionBinding(ctx, r.SessionID)
							sessionValid = false
						}
					}

					if sessionValid && !r.Scheduler.isModelAllowed(acc, checkModel) {
						log.Info("会话粘性账户不允许该模型，移除绑定 - SessionID: %s, 账户ID: %d, 检查模型: %s, AllowedModels: %s",
							r.SessionID, acc.ID, checkModel, acc.AllowedModels)
						sessionCache.RemoveSessionBinding(ctx, r.SessionID)
						sessionValid = false
					}

					if sessionValid {
						sessionCache.UpdateSessionLastUsed(ctx, r.SessionID)
						log.Info("会话粘性命中 - SessionID: %s, 账户ID: %d, 名称: %s", r.SessionID, acc.ID, acc.Name)
						return acc, nil
					}
				} else {
					// 账户不可用，移除会话绑定并刷新调度器缓存
					log.Info("会话粘性账户不可用，移除绑定并刷新缓存 - SessionID: %s, 账户ID: %d", r.SessionID, binding.AccountID)
					sessionCache.RemoveSessionBinding(ctx, r.SessionID)
					r.Scheduler.Refresh()
				}
			}
		}
	}

	var accounts []*model.Account

	if accountType != "" {
		// 判断是平台前缀还是具体类型
		// 如果不包含 "-"，认为是平台前缀（如 claude, openai, gemini），使用前缀匹配
		// 如果包含 "-"，认为是具体类型（如 claude-console, openai-responses），使用精确匹配
		var accList []model.Account
		var e error
		if strings.Contains(accountType, "-") {
			// 具体类型，精确匹配
			accList, e = r.Scheduler.repo.GetEnabledByType(accountType)
		} else {
			// 平台前缀，前缀匹配
			accList, e = r.Scheduler.repo.GetEnabledByTypePrefix(accountType)
		}
		if e != nil {
			log.Error("获取账户失败 - 类型: %s, 错误: %v", accountType, e)
			return nil, e
		}
		accounts = make([]*model.Account, len(accList))
		for i := range accList {
			accounts[i] = &accList[i]
		}
	} else {
		// 根据平台获取账户
		if platform == "" {
			log.Error("不支持的模型 - 模型: %s", actualModel)
			return nil, ErrUnsupportedModel
		}

		r.Scheduler.mu.RLock()
		platformAccounts := r.Scheduler.accounts[platform]
		r.Scheduler.mu.RUnlock()

		// 刷新缓存
		if len(platformAccounts) == 0 {
			r.Scheduler.Refresh()
			r.Scheduler.mu.RLock()
			platformAccounts = r.Scheduler.accounts[platform]
			r.Scheduler.mu.RUnlock()
		}

		accounts = platformAccounts
	}

	// 根据 AllowedModels 和 账户 ModelMapping 过滤账户
	log.Debug("AllowedModels过滤前 - 账户数: %d, 映射后模型: %s, 原始模型: %s", len(accounts), actualModel, originalModel)
	for _, acc := range accounts {
		log.Debug("  账户: ID=%d, Name=%s, AllowedModels='%s', ModelMapping='%s'", acc.ID, acc.Name, acc.AllowedModels, acc.ModelMapping)
	}
	accounts = r.Scheduler.filterByAllowedModelsWithOriginal(accounts, actualModel, originalModel)
	log.Debug("AllowedModels过滤后 - 账户数: %d", len(accounts))
	if len(accounts) == 0 {
		log.Warn("无可用账户(AllowedModels过滤后) - 模型: %s", actualModel)
		return nil, ErrNoAvailableAccount
	}

	// 第一轮：尝试找未尝试过的账户
	available := make([]*model.Account, 0, len(accounts))
	allValid := make([]*model.Account, 0, len(accounts)) // 所有有效账户（包括已尝试的）

	for _, acc := range accounts {
		if !acc.Enabled {
			continue
		}
		// 如果没有明确指定账户类型，排除 openai-responses 类型
		if accountType == "" && acc.Type == model.AccountTypeOpenAIResponses {
			continue
		}
		// 跳过无效账户（已被标记为封号等）
		if acc.Status == model.AccountStatusInvalid {
			continue
		}

		// 收集所有有效账户
		allValid = append(allValid, acc)

		// 未尝试过的账户
		if !r.triedAccounts[acc.ID] {
			available = append(available, acc)
		}
	}

	// 如果有未尝试的账户，优先选择
	if len(available) > 0 {
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
					UserAgent: r.UserAgent,
				}
				sessionCache.SetSessionBinding(ctx, binding)
				log.Info("会话粘性绑定 - SessionID: %s, 账户ID: %d, 名称: %s, UserID: %d", r.SessionID, selected.ID, selected.Name, r.UserID)
			}
		}

		log.Info("选中未尝试账户 - ID: %d, 名称: %s, 类型: %s", selected.ID, selected.Name, selected.Type)
		return selected, nil
	}

	// 如果所有账户都尝试过，但还有有效账户，允许重试（单账户场景）
	if len(allValid) > 0 {
		// 选择失败次数最少的账户
		var minFailures = -1
		var selected *model.Account
		for _, acc := range allValid {
			failures := accountFailures[acc.ID]
			if minFailures < 0 || failures < minFailures {
				minFailures = failures
				selected = acc
			}
		}
		if selected != nil {
			log.Info("重试同一账户 - ID: %d, 名称: %s, 已失败次数: %d", selected.ID, selected.Name, minFailures)
			return selected, nil
		}
	}

	log.Warn("没有可用账户 - 模型: %s, 总账户数: %d", modelName, len(accounts))
	return nil, ErrNoAvailableAccount
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
// 也包括 SSE 首个事件就是错误的情况（此时尚未向客户端写入数据）
func (r *RetryableRequest) isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// 连接错误
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

	// SSE 首个事件为错误（尚未向客户端写入数据，可安全重试）
	sseRetryableErrors := []string{
		"permission_error",
		"authentication_error",
		"overloaded_error",
		"rate_limit_error",
		"api_error",
	}

	for _, sseErr := range sseRetryableErrors {
		if strings.Contains(errStr, sseErr) {
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
