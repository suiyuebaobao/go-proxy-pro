package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"go-aiproxy/internal/model"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type APIKeyRepository struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewAPIKeyRepository() *APIKeyRepository {
	return &APIKeyRepository{db: DB, rdb: RDB}
}

// Create 创建 API Key
func (r *APIKeyRepository) Create(key *model.APIKey) error {
	return r.db.Create(key).Error
}

// GetByID 根据 ID 获取 API Key
func (r *APIKeyRepository) GetByID(id uint) (*model.APIKey, error) {
	var key model.APIKey
	err := r.db.First(&key, id).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// GetByHash 根据哈希获取 API Key
func (r *APIKeyRepository) GetByHash(hash string) (*model.APIKey, error) {
	var key model.APIKey
	err := r.db.Preload("User").Where("key_hash = ?", hash).First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// ListByUserID 获取用户的所有 API Key（包含绑定的套餐信息）
func (r *APIKeyRepository) ListByUserID(userID uint) ([]model.APIKey, error) {
	var keys []model.APIKey
	err := r.db.Preload("UserPackage").Where("user_id = ?", userID).Order("created_at DESC").Find(&keys).Error
	return keys, err
}

// Update 更新 API Key
func (r *APIKeyRepository) Update(key *model.APIKey) error {
	return r.db.Save(key).Error
}

// Delete 删除 API Key
func (r *APIKeyRepository) Delete(id uint) error {
	return r.db.Delete(&model.APIKey{}, id).Error
}

// IncrementUsage 增加使用统计
func (r *APIKeyRepository) IncrementUsage(id uint, tokens int64, cost float64) error {
	return r.db.Model(&model.APIKey{}).Where("id = ?", id).Updates(map[string]interface{}{
		"request_count": gorm.Expr("request_count + 1"),
		"tokens_used":   gorm.Expr("tokens_used + ?", tokens),
		"cost_used":     gorm.Expr("cost_used + ?", cost),
		"last_used_at":  gorm.Expr("NOW()"),
	}).Error
}

// CountByUserID 统计用户的 API Key 数量
func (r *APIKeyRepository) CountByUserID(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&model.APIKey{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// CountByUserIDAndBillingType 统计用户指定计费类型的 API Key 数量
func (r *APIKeyRepository) CountByUserIDAndBillingType(userID uint, billingType string) (int64, error) {
	var count int64
	err := r.db.Model(&model.APIKey{}).Where("user_id = ? AND billing_type = ?", userID, billingType).Count(&count).Error
	return count, err
}

// GetByUserIDAndBillingType 获取用户指定计费类型的 API Key
func (r *APIKeyRepository) GetByUserIDAndBillingType(userID uint, billingType string) (*model.APIKey, error) {
	var key model.APIKey
	err := r.db.Where("user_id = ? AND billing_type = ?", userID, billingType).First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// ListAllWithUser 获取所有 API Key 并带用户信息和套餐信息（管理员用）
func (r *APIKeyRepository) ListAllWithUser(page, pageSize int) ([]model.APIKey, int64, error) {
	var keys []model.APIKey
	var total int64

	err := r.db.Model(&model.APIKey{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err = r.db.Preload("User").Preload("UserPackage").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&keys).Error
	return keys, total, err
}

// GetAPIKeyLogs 从 Redis 获取 API Key 的使用日志
func (r *APIKeyRepository) GetAPIKeyLogs(keyID uint, page, pageSize int) ([]map[string]interface{}, int64, error) {
	ctx := context.Background()
	key := fmt.Sprintf("usage:records:key:%d", keyID)

	// 获取总数
	total, err := r.rdb.LLen(ctx, key).Result()
	if err != nil {
		return nil, 0, err
	}

	// 分页获取
	offset := int64((page - 1) * pageSize)
	results, err := r.rdb.LRange(ctx, key, offset, offset+int64(pageSize)-1).Result()
	if err != nil {
		return nil, 0, err
	}

	logs := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		var record map[string]interface{}
		if err := json.Unmarshal([]byte(r), &record); err != nil {
			continue
		}
		logs = append(logs, record)
	}

	return logs, total, nil
}
