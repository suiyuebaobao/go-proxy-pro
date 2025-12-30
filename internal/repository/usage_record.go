/*
 * 文件作用：使用记录数据仓库，提供请求记录的数据库操作
 * 负责功能：
 *   - 使用记录批量创建
 *   - 按用户/APIKey/日期查询
 *   - 统计数据聚合查询
 *   - 历史记录清理
 * 重要程度：⭐⭐⭐ 一般（使用记录存储仓库）
 * 依赖模块：model, gorm
 */
package repository

import (
	"go-aiproxy/internal/model"
	"time"

	"gorm.io/gorm"
)

type UsageRecordRepository struct {
	db *gorm.DB
}

func NewUsageRecordRepository() *UsageRecordRepository {
	return &UsageRecordRepository{db: DB}
}

// BatchCreate 批量创建使用记录
func (r *UsageRecordRepository) BatchCreate(records []model.UsageRecord) error {
	if len(records) == 0 {
		return nil
	}
	return r.db.CreateInBatches(records, 100).Error
}

// Create 创建单条记录
func (r *UsageRecordRepository) Create(record *model.UsageRecord) error {
	return r.db.Create(record).Error
}

// GetByUserID 获取用户使用记录（分页）
func (r *UsageRecordRepository) GetByUserID(userID uint, offset, limit int) ([]model.UsageRecord, int64, error) {
	return r.GetByUserIDWithFilters(userID, offset, limit, "", "", "")
}

// GetByUserIDWithFilters 获取用户使用记录（带筛选）
func (r *UsageRecordRepository) GetByUserIDWithFilters(userID uint, offset, limit int, startDate, endDate, modelFilter string) ([]model.UsageRecord, int64, error) {
	var records []model.UsageRecord
	var total int64

	query := r.db.Model(&model.UsageRecord{}).Where("user_id = ?", userID)

	// 日期筛选
	if startDate != "" && endDate != "" {
		start, _ := time.Parse("2006-01-02", startDate)
		end, _ := time.Parse("2006-01-02", endDate)
		end = end.Add(24 * time.Hour) // 包含结束日期
		query = query.Where("request_time >= ? AND request_time < ?", start, end)
	}

	// 模型筛选
	if modelFilter != "" {
		query = query.Where("model = ?", modelFilter)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("request_time DESC").Offset(offset).Limit(limit).Find(&records).Error
	return records, total, err
}

// GetByAPIKeyID 获取 API Key 使用记录（分页）
func (r *UsageRecordRepository) GetByAPIKeyID(apiKeyID uint, offset, limit int) ([]model.UsageRecord, int64, error) {
	var records []model.UsageRecord
	var total int64

	query := r.db.Model(&model.UsageRecord{}).Where("api_key_id = ?", apiKeyID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("request_time DESC").Offset(offset).Limit(limit).Find(&records).Error
	return records, total, err
}

// GetByDateRange 获取指定日期范围的记录
func (r *UsageRecordRepository) GetByDateRange(userID uint, startDate, endDate string, offset, limit int) ([]model.UsageRecord, int64, error) {
	var records []model.UsageRecord
	var total int64

	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	end = end.Add(24 * time.Hour) // 包含结束日期

	query := r.db.Model(&model.UsageRecord{}).
		Where("user_id = ? AND request_time >= ? AND request_time < ?", userID, start, end)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("request_time DESC").Offset(offset).Limit(limit).Find(&records).Error
	return records, total, err
}

// GetLatestByUserID 获取用户最新的记录 ID（用于增量同步）
func (r *UsageRecordRepository) GetLatestByUserID(userID uint) (*model.UsageRecord, error) {
	var record model.UsageRecord
	err := r.db.Where("user_id = ?", userID).Order("request_time DESC").First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// DeleteOldRecords 删除旧记录（可选，用于清理历史数据）
func (r *UsageRecordRepository) DeleteOldRecords(beforeDate time.Time) (int64, error) {
	result := r.db.Where("request_time < ?", beforeDate).Delete(&model.UsageRecord{})
	return result.RowsAffected, result.Error
}

// GetUserDailyStats 获取用户日统计（从 MySQL 聚合）
func (r *UsageRecordRepository) GetUserDailyStats(userID uint, startDate, endDate string) ([]model.DailyUsageStats, error) {
	var stats []model.DailyUsageStats

	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	end = end.Add(24 * time.Hour)

	err := r.db.Model(&model.UsageRecord{}).
		Select("DATE(request_time) as date, COUNT(*) as request_count, "+
			"SUM(input_tokens) as input_tokens, SUM(output_tokens) as output_tokens, "+
			"SUM(cache_creation_input_tokens) as cache_creation_input_tokens, "+
			"SUM(cache_read_input_tokens) as cache_read_input_tokens, "+
			"SUM(total_tokens) as total_tokens, SUM(total_cost) as total_cost").
		Where("user_id = ? AND request_time >= ? AND request_time < ?", userID, start, end).
		Group("DATE(request_time)").
		Order("date DESC").
		Find(&stats).Error

	return stats, err
}

// GetUserModelStats 获取用户按模型的统计（从 MySQL 聚合）
func (r *UsageRecordRepository) GetUserModelStats(userID uint) ([]model.ModelUsageStats, error) {
	var stats []model.ModelUsageStats

	err := r.db.Model(&model.UsageRecord{}).
		Select("model, COUNT(*) as request_count, SUM(total_tokens) as total_tokens, SUM(total_cost) as total_cost").
		Where("user_id = ?", userID).
		Group("model").
		Order("total_cost DESC").
		Find(&stats).Error

	return stats, err
}
