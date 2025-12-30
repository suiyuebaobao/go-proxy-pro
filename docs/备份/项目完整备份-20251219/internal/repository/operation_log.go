package repository

import (
	"go-aiproxy/internal/model"
	"time"

	"gorm.io/gorm"
)

type OperationLogRepository struct {
	db *gorm.DB
}

func NewOperationLogRepository() *OperationLogRepository {
	return &OperationLogRepository{db: DB}
}

// Create 创建操作日志
func (r *OperationLogRepository) Create(log *model.OperationLog) error {
	return r.db.Create(log).Error
}

// List 分页查询操作日志
func (r *OperationLogRepository) List(page, pageSize int, filters map[string]interface{}) ([]model.OperationLog, int64, error) {
	var logs []model.OperationLog
	var total int64

	query := r.db.Model(&model.OperationLog{})

	// 应用过滤条件
	if userID, ok := filters["user_id"]; ok && userID.(uint) > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if module, ok := filters["module"]; ok && module.(string) != "" {
		query = query.Where("module = ?", module)
	}
	if action, ok := filters["action"]; ok && action.(string) != "" {
		query = query.Where("action = ?", action)
	}
	if startTime, ok := filters["start_time"]; ok && !startTime.(time.Time).IsZero() {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime, ok := filters["end_time"]; ok && !endTime.(time.Time).IsZero() {
		query = query.Where("created_at <= ?", endTime)
	}
	if search, ok := filters["search"]; ok && search.(string) != "" {
		searchPattern := "%" + search.(string) + "%"
		query = query.Where("username LIKE ? OR description LIKE ? OR target_name LIKE ? OR path LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetByID 根据ID获取日志
func (r *OperationLogRepository) GetByID(id uint) (*model.OperationLog, error) {
	var log model.OperationLog
	if err := r.db.First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// DeleteOldLogs 删除指定天数之前的日志
func (r *OperationLogRepository) DeleteOldLogs(days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days)
	result := r.db.Where("created_at < ?", cutoff).Delete(&model.OperationLog{})
	return result.RowsAffected, result.Error
}

// GetStats 获取日志统计
func (r *OperationLogRepository) GetStats(startTime, endTime time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 总数
	var total int64
	r.db.Model(&model.OperationLog{}).Where("created_at BETWEEN ? AND ?", startTime, endTime).Count(&total)
	stats["total"] = total

	// 按模块统计
	type ModuleStat struct {
		Module string `json:"module"`
		Count  int64  `json:"count"`
	}
	var moduleStats []ModuleStat
	r.db.Model(&model.OperationLog{}).
		Select("module, COUNT(*) as count").
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Group("module").
		Scan(&moduleStats)
	stats["by_module"] = moduleStats

	// 按操作类型统计
	type ActionStat struct {
		Action string `json:"action"`
		Count  int64  `json:"count"`
	}
	var actionStats []ActionStat
	r.db.Model(&model.OperationLog{}).
		Select("action, COUNT(*) as count").
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Group("action").
		Scan(&actionStats)
	stats["by_action"] = actionStats

	// 按用户统计（Top 10）
	type UserStat struct {
		UserID   uint   `json:"user_id"`
		Username string `json:"username"`
		Count    int64  `json:"count"`
	}
	var userStats []UserStat
	r.db.Model(&model.OperationLog{}).
		Select("user_id, username, COUNT(*) as count").
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Group("user_id, username").
		Order("count DESC").
		Limit(10).
		Scan(&userStats)
	stats["by_user"] = userStats

	return stats, nil
}
