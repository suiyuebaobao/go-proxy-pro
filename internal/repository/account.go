/*
 * 文件作用：账户数据仓库，提供上游账户的数据库操作
 * 负责功能：
 *   - 账户CRUD操作
 *   - 按平台/类型/状态查询
 *   - 账户状态管理（限流/恢复/封号）
 *   - 健康检查调度
 *   - 账户分组管理
 * 重要程度：⭐⭐⭐⭐⭐ 核心（账户核心仓库）
 * 依赖模块：model, gorm
 */
package repository

import (
	"time"

	"go-aiproxy/internal/model"

	"gorm.io/gorm"
)

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository() *AccountRepository {
	return &AccountRepository{db: DB}
}

// Account CRUD

func (r *AccountRepository) Create(account *model.Account) error {
	return r.db.Create(account).Error
}

func (r *AccountRepository) GetByID(id uint) (*model.Account, error) {
	var account model.Account
	err := r.db.Preload("Groups").Preload("Proxy").First(&account, id).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// GetByIDs 根据 ID 列表批量获取账户
func (r *AccountRepository) GetByIDs(ids []uint) ([]model.Account, error) {
	var accounts []model.Account
	if len(ids) == 0 {
		return accounts, nil
	}
	err := r.db.Where("id IN ?", ids).Find(&accounts).Error
	return accounts, err
}

func (r *AccountRepository) Update(account *model.Account) error {
	// 使用 Select 指定所有字段，确保 nil 值也能被更新
	return r.db.Model(account).Select("*").Updates(account).Error
}

// ClearProxyID 清除账户的代理 ID
func (r *AccountRepository) ClearProxyID(id uint) error {
	// 使用原生 SQL 确保能将 proxy_id 设为 NULL
	return r.db.Exec("UPDATE accounts SET proxy_id = NULL WHERE id = ?", id).Error
}

func (r *AccountRepository) Delete(id uint) error {
	return r.db.Delete(&model.Account{}, id).Error
}

func (r *AccountRepository) List(page, pageSize int, platform, status string) ([]model.Account, int64, error) {
	var accounts []model.Account
	var total int64

	query := r.db.Model(&model.Account{})

	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Preload("Groups").Preload("Proxy").Offset(offset).Limit(pageSize).Order("priority DESC, id ASC").Find(&accounts).Error
	if err != nil {
		return nil, 0, err
	}

	return accounts, total, nil
}

func (r *AccountRepository) GetByPlatform(platform string) ([]model.Account, error) {
	var accounts []model.Account
	err := r.db.Where("platform = ? AND enabled = ? AND status = ?",
		platform, true, model.AccountStatusValid).
		Order("priority DESC, weight DESC").
		Find(&accounts).Error
	return accounts, err
}

func (r *AccountRepository) GetEnabledByType(accountType string) ([]model.Account, error) {
	var accounts []model.Account
	err := r.db.Where("type = ? AND enabled = ? AND status = ?",
		accountType, true, model.AccountStatusValid).
		Order("priority DESC, weight DESC").
		Find(&accounts).Error
	return accounts, err
}

// GetEnabledByTypePrefix 按类型前缀获取启用的账户
// 例如传入 "claude" 会匹配 "claude-official", "claude-console", "claude-bedrock" 等
func (r *AccountRepository) GetEnabledByTypePrefix(typePrefix string) ([]model.Account, error) {
	var accounts []model.Account
	err := r.db.Where("type LIKE ? AND enabled = ? AND status = ?",
		typePrefix+"%", true, model.AccountStatusValid).
		Order("priority DESC, weight DESC").
		Find(&accounts).Error
	return accounts, err
}

func (r *AccountRepository) UpdateStatus(id uint, status string, lastError string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if lastError != "" {
		updates["last_error"] = lastError
		updates["last_error_at"] = gorm.Expr("NOW()")
	}
	// 如果不是限流状态，清空限流恢复时间
	if status != model.AccountStatusRateLimited {
		updates["rate_limit_reset_at"] = nil
	}
	return r.db.Model(&model.Account{}).Where("id = ?", id).Updates(updates).Error
}

// SetEnabled 设置账户启用状态
func (r *AccountRepository) SetEnabled(id uint, enabled bool) error {
	return r.db.Model(&model.Account{}).Where("id = ?", id).Update("enabled", enabled).Error
}

// UpdateStatusWithRateLimit 更新状态并设置限流恢复时间
func (r *AccountRepository) UpdateStatusWithRateLimit(id uint, status string, lastError string, resetAt *time.Time) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if lastError != "" {
		updates["last_error"] = lastError
		updates["last_error_at"] = gorm.Expr("NOW()")
	}
	if resetAt != nil {
		updates["rate_limit_reset_at"] = resetAt
	}
	return r.db.Model(&model.Account{}).Where("id = ?", id).Updates(updates).Error
}

// RecoverRateLimitedAccounts 恢复已到期的限流账号
// 将 status=rate_limited 且 rate_limit_reset_at <= now 的账号状态改为 valid
func (r *AccountRepository) RecoverRateLimitedAccounts() (int64, error) {
	result := r.db.Model(&model.Account{}).
		Where("status = ? AND rate_limit_reset_at IS NOT NULL AND rate_limit_reset_at <= ?",
			model.AccountStatusRateLimited, time.Now()).
		Updates(map[string]interface{}{
			"status":              model.AccountStatusValid,
			"rate_limit_reset_at": nil,
		})
	return result.RowsAffected, result.Error
}

// UpdateUsageStatus 更新账号用量状态 (从 Claude API 响应头获取)
func (r *AccountRepository) UpdateUsageStatus(id uint, usageStatus string, rateLimitReset *int64) error {
	now := time.Now()
	updates := map[string]interface{}{
		"usage_status":            usageStatus,
		"usage_status_updated_at": now,
	}
	if rateLimitReset != nil {
		updates["rate_limit_reset"] = *rateLimitReset
		// 同时更新 rate_limit_reset_at（转换为时间）
		resetTime := time.Unix(*rateLimitReset, 0)
		updates["rate_limit_reset_at"] = resetTime
	}
	return r.db.Model(&model.Account{}).Where("id = ?", id).Updates(updates).Error
}

// ClaudeUsageData Claude 用量数据结构
type ClaudeUsageData struct {
	FiveHour struct {
		Utilization *float64 `json:"utilization"`
		ResetsAt    string   `json:"resets_at"`
	} `json:"five_hour"`
	SevenDay struct {
		Utilization *float64 `json:"utilization"`
		ResetsAt    string   `json:"resets_at"`
	} `json:"seven_day"`
	SevenDaySonnet struct {
		Utilization *float64 `json:"utilization"`
		ResetsAt    string   `json:"resets_at"`
	} `json:"seven_day_sonnet"`
}

// UpdateClaudeUsage 更新 Claude 账号用量数据
func (r *AccountRepository) UpdateClaudeUsage(id uint, usage *ClaudeUsageData) error {
	now := time.Now()
	updates := map[string]interface{}{
		"usage_status_updated_at": now,
	}

	// 5小时窗口
	if usage.FiveHour.Utilization != nil {
		updates["five_hour_utilization"] = *usage.FiveHour.Utilization
	}
	if usage.FiveHour.ResetsAt != "" {
		if t, err := time.Parse(time.RFC3339, usage.FiveHour.ResetsAt); err == nil {
			updates["five_hour_resets_at"] = t
		}
	}

	// 7天窗口
	if usage.SevenDay.Utilization != nil {
		updates["seven_day_utilization"] = *usage.SevenDay.Utilization
	}
	if usage.SevenDay.ResetsAt != "" {
		if t, err := time.Parse(time.RFC3339, usage.SevenDay.ResetsAt); err == nil {
			updates["seven_day_resets_at"] = t
		}
	}

	// 7天Sonnet窗口
	if usage.SevenDaySonnet.Utilization != nil {
		updates["seven_day_sonnet_utilization"] = *usage.SevenDaySonnet.Utilization
	}
	if usage.SevenDaySonnet.ResetsAt != "" {
		if t, err := time.Parse(time.RFC3339, usage.SevenDaySonnet.ResetsAt); err == nil {
			updates["seven_day_sonnet_resets_at"] = t
		}
	}

	return r.db.Model(&model.Account{}).Where("id = ?", id).Updates(updates).Error
}

func (r *AccountRepository) IncrementRequestCount(id uint) error {
	return r.db.Model(&model.Account{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"request_count": gorm.Expr("request_count + 1"),
			"last_used_at":  gorm.Expr("NOW()"),
		}).Error
}

func (r *AccountRepository) IncrementErrorCount(id uint) error {
	return r.db.Model(&model.Account{}).Where("id = ?", id).
		Update("error_count", gorm.Expr("error_count + 1")).Error
}

// UpdateLastError 只更新最后错误信息（不改变状态）
func (r *AccountRepository) UpdateLastError(id uint, lastError string) error {
	if lastError == "" {
		return nil
	}
	// 截断过长的错误信息
	if len(lastError) > 500 {
		lastError = lastError[:500] + "..."
	}
	return r.db.Model(&model.Account{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_error":    lastError,
			"last_error_at": gorm.Expr("NOW()"),
		}).Error
}

// UpdateTotalCost 更新账户总费用
func (r *AccountRepository) UpdateTotalCost(id uint, totalCost float64) error {
	return r.db.Model(&model.Account{}).Where("id = ?", id).
		Update("total_cost", totalCost).Error
}

// IncrementTotalCost 增量更新账户总费用（原子操作）
func (r *AccountRepository) IncrementTotalCost(id uint, cost float64) error {
	return r.db.Model(&model.Account{}).Where("id = ?", id).
		Update("total_cost", gorm.Expr("total_cost + ?", cost)).Error
}

// GetTotalCostByIDs 批量获取账户费用
func (r *AccountRepository) GetTotalCostByIDs(ids []uint) (map[uint]float64, error) {
	result := make(map[uint]float64)
	if len(ids) == 0 {
		return result, nil
	}

	var accounts []model.Account
	if err := r.db.Select("id, total_cost").Where("id IN ?", ids).Find(&accounts).Error; err != nil {
		return nil, err
	}

	for _, acc := range accounts {
		result[acc.ID] = acc.TotalCost
	}
	return result, nil
}

func (r *AccountRepository) GetAllEnabled() ([]model.Account, error) {
	var accounts []model.Account
	err := r.db.Where("enabled = ?", true).Find(&accounts).Error
	return accounts, err
}

func (r *AccountRepository) UpdateToken(id uint, accessToken, refreshToken string, expiry *time.Time) error {
	updates := map[string]interface{}{
		"access_token": accessToken,
	}
	if refreshToken != "" {
		updates["refresh_token"] = refreshToken
	}
	if expiry != nil {
		updates["token_expiry"] = expiry
	}
	return r.db.Model(&model.Account{}).Where("id = ?", id).Updates(updates).Error
}

// GetAccountsForHealthCheck 获取需要健康检查的账号
// 只检查 OAuth/SessionKey 类型的账号（非 API Key 类型）
// 包括 valid 和 rate_limited 状态的账号
func (r *AccountRepository) GetAccountsForHealthCheck() ([]model.Account, error) {
	var accounts []model.Account
	// 只检查这些类型：claude-official, openai-responses, gemini
	// 这些是使用 OAuth 或 SessionKey 认证的账号
	err := r.db.Where("enabled = ? AND type IN (?, ?, ?) AND status IN (?, ?)",
		true,
		model.AccountTypeClaudeOfficial, model.AccountTypeOpenAIResponses, model.AccountTypeGemini,
		model.AccountStatusValid, model.AccountStatusRateLimited).
		Preload("Proxy").
		Find(&accounts).Error
	return accounts, err
}

// IncrementConsecutiveErrorCount 增加连续错误计数
func (r *AccountRepository) IncrementConsecutiveErrorCount(id uint) (int, error) {
	var account model.Account
	err := r.db.Model(&model.Account{}).Where("id = ?", id).
		Update("consecutive_error_count", gorm.Expr("consecutive_error_count + 1")).Error
	if err != nil {
		return 0, err
	}
	// 获取更新后的值
	r.db.Select("consecutive_error_count").First(&account, id)
	return account.ConsecutiveErrorCount, nil
}

// ResetConsecutiveErrorCount 重置连续错误计数
func (r *AccountRepository) ResetConsecutiveErrorCount(id uint) error {
	return r.db.Model(&model.Account{}).Where("id = ?", id).
		Update("consecutive_error_count", 0).Error
}

// DisableAccountByHealthCheck 因健康检查失败禁用账号
func (r *AccountRepository) DisableAccountByHealthCheck(id uint, lastError string) error {
	// 截断过长的错误信息（数据库字段限制）
	if len(lastError) > 500 {
		lastError = lastError[:500] + "..."
	}
	return r.db.Model(&model.Account{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"enabled":       false,
			"status":        model.AccountStatusInvalid,
			"last_error":    lastError,
			"last_error_at": gorm.Expr("NOW()"),
		}).Error
}

// AccountGroup CRUD

type AccountGroupRepository struct {
	db *gorm.DB
}

func NewAccountGroupRepository() *AccountGroupRepository {
	return &AccountGroupRepository{db: DB}
}

func (r *AccountGroupRepository) Create(group *model.AccountGroup) error {
	return r.db.Create(group).Error
}

func (r *AccountGroupRepository) GetByID(id uint) (*model.AccountGroup, error) {
	var group model.AccountGroup
	err := r.db.Preload("Accounts").First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *AccountGroupRepository) Update(group *model.AccountGroup) error {
	return r.db.Save(group).Error
}

func (r *AccountGroupRepository) Delete(id uint) error {
	// 先删除关联
	r.db.Exec("DELETE FROM account_group_members WHERE account_group_id = ?", id)
	return r.db.Delete(&model.AccountGroup{}, id).Error
}

func (r *AccountGroupRepository) List(page, pageSize int) ([]model.AccountGroup, int64, error) {
	var groups []model.AccountGroup
	var total int64

	r.db.Model(&model.AccountGroup{}).Count(&total)

	offset := (page - 1) * pageSize
	err := r.db.Preload("Accounts").Offset(offset).Limit(pageSize).Find(&groups).Error
	if err != nil {
		return nil, 0, err
	}

	return groups, total, nil
}

func (r *AccountGroupRepository) GetAll() ([]model.AccountGroup, error) {
	var groups []model.AccountGroup
	err := r.db.Find(&groups).Error
	return groups, err
}

func (r *AccountGroupRepository) AddAccount(groupID, accountID uint) error {
	return r.db.Exec("INSERT IGNORE INTO account_group_members (account_group_id, account_id) VALUES (?, ?)",
		groupID, accountID).Error
}

func (r *AccountGroupRepository) RemoveAccount(groupID, accountID uint) error {
	return r.db.Exec("DELETE FROM account_group_members WHERE account_group_id = ? AND account_id = ?",
		groupID, accountID).Error
}

func (r *AccountGroupRepository) GetAccountsByGroup(groupID uint) ([]model.Account, error) {
	var accounts []model.Account
	err := r.db.Joins("JOIN account_group_members ON accounts.id = account_group_members.account_id").
		Where("account_group_members.account_group_id = ?", groupID).
		Find(&accounts).Error
	return accounts, err
}

// ========== 健康检测相关方法 ==========

// GetProblemAccounts 获取问题账号（需要健康检测的）
// 包括：rate_limited, token_expired, suspended, banned, overloaded 状态的账号
func (r *AccountRepository) GetProblemAccounts() ([]model.Account, error) {
	var accounts []model.Account
	// 获取所有非 valid 状态的账号（排除 disabled 状态，那是手动禁用的）
	err := r.db.Where("type IN (?, ?, ?) AND status IN (?, ?, ?, ?, ?)",
		model.AccountTypeClaudeOfficial, model.AccountTypeOpenAIResponses, model.AccountTypeGemini,
		model.AccountStatusRateLimited, model.AccountStatusTokenExpired,
		model.AccountStatusSuspended, model.AccountStatusBanned, model.AccountStatusOverloaded).
		Preload("Proxy").
		Find(&accounts).Error
	return accounts, err
}

// GetAccountsNeedingProbe 获取需要探测的账号（到达探测时间的）
func (r *AccountRepository) GetAccountsNeedingProbe() ([]model.Account, error) {
	var accounts []model.Account
	now := time.Now()
	err := r.db.Where("type IN (?, ?, ?) AND status IN (?, ?, ?, ?) AND (next_health_check_at IS NULL OR next_health_check_at <= ?)",
		model.AccountTypeClaudeOfficial, model.AccountTypeOpenAIResponses, model.AccountTypeGemini,
		model.AccountStatusRateLimited, model.AccountStatusTokenExpired,
		model.AccountStatusSuspended, model.AccountStatusBanned,
		now).
		Preload("Proxy").
		Find(&accounts).Error
	return accounts, err
}

// UpdateHealthCheckSchedule 更新账号的健康检测计划
func (r *AccountRepository) UpdateHealthCheckSchedule(id uint, nextCheckAt time.Time, intervalSeconds int) error {
	now := time.Now()
	return r.db.Model(&model.Account{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_health_check_at":   now,
			"next_health_check_at":   nextCheckAt,
			"health_check_interval":  intervalSeconds,
		}).Error
}

// IncrementSuspendedCount 增加疑似封号计数
func (r *AccountRepository) IncrementSuspendedCount(id uint) (int, error) {
	var account model.Account
	err := r.db.Model(&model.Account{}).Where("id = ?", id).
		Update("suspended_count", gorm.Expr("suspended_count + 1")).Error
	if err != nil {
		return 0, err
	}
	r.db.Select("suspended_count").First(&account, id)
	return account.SuspendedCount, nil
}

// ResetSuspendedCount 重置疑似封号计数
func (r *AccountRepository) ResetSuspendedCount(id uint) error {
	return r.db.Model(&model.Account{}).Where("id = ?", id).
		Update("suspended_count", 0).Error
}

// MarkAsSuspended 标记账号为疑似封号状态
func (r *AccountRepository) MarkAsSuspended(id uint, errMsg string) error {
	if len(errMsg) > 500 {
		errMsg = errMsg[:500]
	}
	return r.db.Model(&model.Account{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       model.AccountStatusSuspended,
			"last_error":   errMsg,
			"last_error_at": gorm.Expr("NOW()"),
		}).Error
}

// MarkAsBanned 标记账号为确认封号状态
func (r *AccountRepository) MarkAsBanned(id uint, errMsg string) error {
	if len(errMsg) > 500 {
		errMsg = errMsg[:500]
	}
	return r.db.Model(&model.Account{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       model.AccountStatusBanned,
			"enabled":      false,
			"last_error":   errMsg,
			"last_error_at": gorm.Expr("NOW()"),
		}).Error
}

// MarkAsTokenExpired 标记账号为 Token 过期状态
func (r *AccountRepository) MarkAsTokenExpired(id uint, errMsg string) error {
	if len(errMsg) > 500 {
		errMsg = errMsg[:500]
	}
	return r.db.Model(&model.Account{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       model.AccountStatusTokenExpired,
			"last_error":   errMsg,
			"last_error_at": gorm.Expr("NOW()"),
		}).Error
}

// MarkAsInvalid 标记账号为无效状态
func (r *AccountRepository) MarkAsInvalid(id uint, errMsg string) error {
	if len(errMsg) > 500 {
		errMsg = errMsg[:500]
	}
	return r.db.Model(&model.Account{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       model.AccountStatusInvalid,
			"last_error":   errMsg,
			"last_error_at": gorm.Expr("NOW()"),
		}).Error
}

// MarkAsRateLimited 标记账号为限流状态
func (r *AccountRepository) MarkAsRateLimited(id uint, resetAt *time.Time, errMsg string) error {
	if len(errMsg) > 500 {
		errMsg = errMsg[:500]
	}
	return r.db.Model(&model.Account{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":              model.AccountStatusRateLimited,
			"rate_limit_reset_at": resetAt,
			"last_error":          errMsg,
			"last_error_at":       gorm.Expr("NOW()"),
		}).Error
}

// RecoverAccount 恢复账号为有效状态
func (r *AccountRepository) RecoverAccount(id uint) error {
	return r.db.Model(&model.Account{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":                   model.AccountStatusValid,
			"enabled":                  true,
			"consecutive_error_count":  0,
			"suspended_count":          0,
			"last_error":               nil,
			"rate_limit_reset_at":      nil,
			"next_health_check_at":     nil,
			"health_check_interval":    0,
		}).Error
}

// ForceRecoverAccount 强制恢复账号（不检测，直接恢复）
func (r *AccountRepository) ForceRecoverAccount(id uint) error {
	return r.RecoverAccount(id)
}
