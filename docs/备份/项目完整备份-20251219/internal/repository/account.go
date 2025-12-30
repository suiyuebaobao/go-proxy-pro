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

func (r *AccountRepository) Update(account *model.Account) error {
	return r.db.Save(account).Error
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

// UpdateTotalCost 更新账户总费用（从 Redis 同步）
func (r *AccountRepository) UpdateTotalCost(id uint, totalCost float64) error {
	return r.db.Model(&model.Account{}).Where("id = ?", id).
		Update("total_cost", totalCost).Error
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
