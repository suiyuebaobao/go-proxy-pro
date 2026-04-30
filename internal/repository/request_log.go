/*
 * 文件作用：请求日志数据仓库，提供代理请求记录的数据库操作
 * 负责功能：
 *   - 请求日志创建和查询
 *   - 多条件过滤（账户/平台/模型/时间）
 *   - 请求统计汇总
 *   - 账户负载分析
 * 重要程度：⭐⭐⭐ 一般（请求日志仓库）
 * 依赖模块：model, gorm
 */
package repository

import (
	"time"

	"go-aiproxy/internal/model"

	"gorm.io/gorm"
)

type RequestLogRepository struct {
	db *gorm.DB
}

func NewRequestLogRepository() *RequestLogRepository {
	return &RequestLogRepository{db: DB}
}

func (r *RequestLogRepository) Create(log *model.RequestLog) error {
	return r.db.Create(log).Error
}

func (r *RequestLogRepository) List(page, pageSize int, filters map[string]interface{}) ([]model.RequestLog, int64, error) {
	var logs []model.RequestLog
	var total int64

	query := r.db.Model(&model.RequestLog{})

	if accountID, ok := filters["account_id"].(uint); ok && accountID > 0 {
		query = query.Where("account_id = ?", accountID)
	}
	if platform, ok := filters["platform"].(string); ok && platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if modelName, ok := filters["model"].(string); ok && modelName != "" {
		query = query.Where("model LIKE ?", "%"+modelName+"%")
	}
	if userID, ok := filters["user_id"].(uint); ok && userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if success, ok := filters["success"].(bool); ok {
		query = query.Where("success = ?", success)
	}
	if statusCode, ok := filters["status_code"].(int); ok {
		query = query.Where("status_code = ?", statusCode)
	}
	if minDur, ok := filters["min_duration"].(int64); ok {
		query = query.Where("duration >= ?", minDur)
	}
	if maxDur, ok := filters["max_duration"].(int64); ok {
		query = query.Where("duration <= ?", maxDur)
	}
	if startTime, ok := filters["start_time"].(time.Time); ok {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime, ok := filters["end_time"].(time.Time); ok {
		query = query.Where("created_at <= ?", endTime)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Preload("Account").Offset(offset).Limit(pageSize).Order("created_at DESC").
		Select("id, account_id, user_id, platform, model, endpoint, method, path, request_ip, user_agent, session_id, input_tokens, output_tokens, cache_creation_input_tokens, cache_read_input_tokens, total_tokens, input_cost, output_cost, cache_create_cost, cache_read_cost, total_cost, api_key_id, status_code, success, error, duration, upstream_status_code, upstream_error, created_at").
		Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetByID 获取单条日志详情（含完整请求体/响应体）
func (r *RequestLogRepository) GetByID(id uint) (*model.RequestLog, error) {
	var log model.RequestLog
	err := r.db.Preload("Account").First(&log, id).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *RequestLogRepository) GetSummary(startTime, endTime time.Time) (*model.RequestLogSummary, error) {
	var summary model.RequestLogSummary

	query := r.db.Model(&model.RequestLog{}).
		Where("created_at BETWEEN ? AND ?", startTime, endTime)

	err := query.Select(`
		COUNT(*) as total_requests,
		SUM(CASE WHEN success = true THEN 1 ELSE 0 END) as success_requests,
		SUM(CASE WHEN success = false THEN 1 ELSE 0 END) as failed_requests,
		COALESCE(SUM(input_tokens), 0) as total_input_tokens,
		COALESCE(SUM(output_tokens), 0) as total_output_tokens,
		COALESCE(AVG(duration), 0) as avg_duration
	`).Scan(&summary).Error

	return &summary, err
}

func (r *RequestLogRepository) GetAccountLoadStats(startTime, endTime time.Time) ([]model.AccountLoadStats, error) {
	var stats []model.AccountLoadStats

	err := r.db.Model(&model.RequestLog{}).
		Select(`
			request_logs.account_id,
			accounts.name as account_name,
			accounts.platform,
			COUNT(*) as request_count,
			SUM(CASE WHEN request_logs.success = true THEN 1 ELSE 0 END) as success_count,
			SUM(CASE WHEN request_logs.success = false THEN 1 ELSE 0 END) as error_count,
			COALESCE(SUM(request_logs.input_tokens + request_logs.output_tokens), 0) as total_tokens,
			COALESCE(AVG(request_logs.duration), 0) as avg_duration,
			MAX(request_logs.created_at) as last_used_at
		`).
		Joins("LEFT JOIN accounts ON accounts.id = request_logs.account_id").
		Where("request_logs.created_at BETWEEN ? AND ?", startTime, endTime).
		Group("request_logs.account_id, accounts.name, accounts.platform").
		Order("request_count DESC").
		Scan(&stats).Error

	return stats, err
}

func (r *RequestLogRepository) CleanOldLogs(before time.Time) (int64, error) {
	result := r.db.Where("created_at < ?", before).Delete(&model.RequestLog{})
	return result.RowsAffected, result.Error
}

// AccountTodayUsage 账户今日用量统计
type AccountTodayUsage struct {
	AccountID   uint    `json:"account_id"`
	TodayTokens int64   `json:"today_tokens"`
	TodayCount  int64   `json:"today_count"`
	TodayCost   float64 `json:"today_cost"`
}

// GetAccountsTodayUsage 获取多个账户的今日用量
func (r *RequestLogRepository) GetAccountsTodayUsage(accountIDs []uint) (map[uint]*AccountTodayUsage, error) {
	if len(accountIDs) == 0 {
		return make(map[uint]*AccountTodayUsage), nil
	}

	// 获取今日开始时间
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var results []AccountTodayUsage
	err := r.db.Model(&model.RequestLog{}).
		Select(`
			account_id,
			COALESCE(SUM(input_tokens + output_tokens), 0) as today_tokens,
			COUNT(*) as today_count,
			COALESCE(SUM(cost), 0) as today_cost
		`).
		Where("account_id IN ? AND created_at >= ?", accountIDs, todayStart).
		Group("account_id").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// 转换为 map
	usageMap := make(map[uint]*AccountTodayUsage)
	for i := range results {
		usageMap[results[i].AccountID] = &results[i]
	}

	return usageMap, nil
}

// AccountTotalCost 账户总费用统计
type AccountTotalCost struct {
	AccountID uint    `json:"account_id"`
	TotalCost float64 `json:"total_cost"`
}

// GetAccountsTotalCost 获取多个账户的总费用
func (r *RequestLogRepository) GetAccountsTotalCost(accountIDs []uint) (map[uint]float64, error) {
	if len(accountIDs) == 0 {
		return make(map[uint]float64), nil
	}

	var results []AccountTotalCost
	err := r.db.Model(&model.RequestLog{}).
		Select(`
			account_id,
			COALESCE(SUM(total_cost), 0) as total_cost
		`).
		Where("account_id IN ?", accountIDs).
		Group("account_id").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// 转换为 map
	costMap := make(map[uint]float64)
	for _, r := range results {
		costMap[r.AccountID] = r.TotalCost
	}

	return costMap, nil
}

// HourlyTrendItem 小时趋势数据
type HourlyTrendItem struct {
	Hour         string  `json:"hour"`
	RequestCount int64   `json:"request_count"`
	TotalTokens  int64   `json:"total_tokens"`
	TotalCost    float64 `json:"total_cost"`
}

// GetHourlyTrend 获取最近 24 小时逐小时趋势
func (r *RequestLogRepository) GetHourlyTrend() ([]HourlyTrendItem, error) {
	var items []HourlyTrendItem
	since := time.Now().Add(-24 * time.Hour)

	err := r.db.Model(&model.RequestLog{}).
		Select("DATE_FORMAT(created_at, '%Y-%m-%d %H:00') as hour, COUNT(*) as request_count, COALESCE(SUM(total_tokens), 0) as total_tokens, COALESCE(SUM(total_cost), 0) as total_cost").
		Where("created_at >= ?", since).
		Group("hour").
		Order("hour ASC").
		Scan(&items).Error

	return items, err
}
