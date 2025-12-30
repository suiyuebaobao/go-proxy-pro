/*
 * 文件作用：数据库迁移和初始化，管理数据库表结构和默认数据
 * 负责功能：
 *   - 自动迁移所有数据表
 *   - 初始化默认管理员
 *   - 初始化默认配置项
 *   - API Key套餐绑定迁移
 * 重要程度：⭐⭐⭐⭐ 重要（数据库初始化核心）
 * 依赖模块：model, logger, gorm
 */
package repository

import (
	"go-aiproxy/internal/model"
	"go-aiproxy/pkg/logger"
)

func AutoMigrate() error {
	return DB.AutoMigrate(
		&model.User{},
		&model.Proxy{},
		&model.Account{},
		&model.AccountGroup{},
		&model.RequestLog{},
		&model.AIModel{},
		&model.APIKey{},
		&model.DailyUsage{},
		&model.SystemConfig{},
		&model.UsageRecord{},
		&model.Package{},
		&model.UserPackage{},
		&model.OperationLog{},
		// 客户端过滤相关
		&model.ClientType{},
		&model.ClientFilterRule{},
		&model.ClientFilterConfig{},
		// 错误消息配置
		&model.ErrorMessage{},
		// 错误规则配置
		&model.ErrorRule{},
		// 模型映射
		&model.ModelMapping{},
	)
}

func InitDefaultAdmin() error {
	log := logger.GetLogger("main")

	var count int64
	DB.Model(&model.User{}).Where("role = ?", "admin").Count(&count)
	if count > 0 {
		return nil
	}

	admin := &model.User{
		Username: "admin",
		Email:    "admin@aiproxy.local",
		Role:     "admin",
		Status:   "active",
	}
	if err := admin.SetPassword("admin123"); err != nil {
		return err
	}

	if err := DB.Create(admin).Error; err != nil {
		return err
	}

	log.Info("默认管理员已创建 | 用户名: admin | 密码: admin123")
	return nil
}

// InitDefaultConfigs 初始化默认系统配置
func InitDefaultConfigs() error {
	log := logger.GetLogger("main")

	for _, cfg := range model.DefaultConfigs {
		var existing model.SystemConfig
		result := DB.Where("config_key = ?", cfg.Key).First(&existing)
		if result.Error == nil {
			// 已存在，跳过
			continue
		}

		if err := DB.Create(&cfg).Error; err != nil {
			log.Error("创建默认配置失败: %s, %v", cfg.Key, err)
			continue
		}
		log.Info("默认配置已创建: %s = %s", cfg.Key, cfg.Value)
	}

	return nil
}

// InitDefaultClientFilters 初始化默认客户端过滤配置
func InitDefaultClientFilters() error {
	repo := NewClientFilterRepository()
	return repo.InitDefaultData()
}

// InitDefaultErrorMessages 初始化默认错误消息配置
func InitDefaultErrorMessages() error {
	repo := NewErrorMessageRepository()
	return repo.InitDefaultData()
}

// InitDefaultErrorRules 初始化默认错误规则
func InitDefaultErrorRules() error {
	repo := NewErrorRuleRepository()
	return repo.InitDefaultRules()
}

// MigrateAPIKeyPackageBinding 迁移未绑定套餐的 API Key
// 将所有 user_package_id 为空的 API Key 自动绑定到用户的第一个活跃套餐
func MigrateAPIKeyPackageBinding() error {
	log := logger.GetLogger("main")

	// 查找所有未绑定套餐的 API Key
	var unboundKeys []model.APIKey
	if err := DB.Where("user_package_id IS NULL").Find(&unboundKeys).Error; err != nil {
		return err
	}

	if len(unboundKeys) == 0 {
		log.Debug("没有需要迁移的 API Key")
		return nil
	}

	log.Info("发现 %d 个未绑定套餐的 API Key，开始迁移...", len(unboundKeys))

	migrated := 0
	failed := 0

	for _, key := range unboundKeys {
		// 查找该用户的第一个活跃套餐
		var userPackage model.UserPackage
		err := DB.Where("user_id = ? AND status = ?", key.UserID, "active").
			Order("created_at ASC").
			First(&userPackage).Error

		if err != nil {
			log.Warn("用户 %d 没有活跃套餐，API Key %d 无法迁移", key.UserID, key.ID)
			failed++
			continue
		}

		// 更新 API Key 绑定
		if err := DB.Model(&key).Updates(map[string]interface{}{
			"user_package_id": userPackage.ID,
			"billing_type":    userPackage.Type,
		}).Error; err != nil {
			log.Error("迁移 API Key %d 失败: %v", key.ID, err)
			failed++
			continue
		}

		log.Debug("API Key %d 已绑定到套餐 %d (用户 %d)", key.ID, userPackage.ID, key.UserID)
		migrated++
	}

	log.Info("API Key 套餐绑定迁移完成 | 成功: %d | 失败: %d", migrated, failed)
	return nil
}
