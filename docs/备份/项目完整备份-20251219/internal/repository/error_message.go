package repository

import (
	"go-aiproxy/internal/model"

	"gorm.io/gorm"
)

type ErrorMessageRepository struct {
	db *gorm.DB
}

func NewErrorMessageRepository() *ErrorMessageRepository {
	return &ErrorMessageRepository{db: DB}
}

// GetAll 获取所有错误消息配置
func (r *ErrorMessageRepository) GetAll() ([]model.ErrorMessage, error) {
	var messages []model.ErrorMessage
	err := r.db.Order("code, error_type").Find(&messages).Error
	return messages, err
}

// GetByID 根据 ID 获取
func (r *ErrorMessageRepository) GetByID(id uint) (*model.ErrorMessage, error) {
	var msg model.ErrorMessage
	err := r.db.First(&msg, id).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetByErrorType 根据错误类型获取
func (r *ErrorMessageRepository) GetByErrorType(errorType string) (*model.ErrorMessage, error) {
	var msg model.ErrorMessage
	err := r.db.Where("error_type = ?", errorType).First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetByCode 根据 HTTP 状态码获取
func (r *ErrorMessageRepository) GetByCode(code int) ([]model.ErrorMessage, error) {
	var messages []model.ErrorMessage
	err := r.db.Where("code = ?", code).Find(&messages).Error
	return messages, err
}

// GetEnabled 获取所有启用的错误消息
func (r *ErrorMessageRepository) GetEnabled() ([]model.ErrorMessage, error) {
	var messages []model.ErrorMessage
	err := r.db.Where("enabled = ?", true).Find(&messages).Error
	return messages, err
}

// Create 创建错误消息配置
func (r *ErrorMessageRepository) Create(msg *model.ErrorMessage) error {
	return r.db.Create(msg).Error
}

// CreateWithDisabled 创建禁用状态的错误消息配置（用于自动发现）
func (r *ErrorMessageRepository) CreateWithDisabled(msg *model.ErrorMessage) error {
	// 先创建记录
	if err := r.db.Create(msg).Error; err != nil {
		return err
	}
	// 强制更新 enabled 为 false（绕过 GORM 零值问题）
	return r.db.Model(msg).UpdateColumn("enabled", false).Error
}

// Update 更新错误消息配置
func (r *ErrorMessageRepository) Update(msg *model.ErrorMessage) error {
	return r.db.Save(msg).Error
}

// UpdateCustomMessage 更新自定义消息
func (r *ErrorMessageRepository) UpdateCustomMessage(id uint, customMessage string) error {
	return r.db.Model(&model.ErrorMessage{}).Where("id = ?", id).Update("custom_message", customMessage).Error
}

// ToggleEnabled 切换启用状态
func (r *ErrorMessageRepository) ToggleEnabled(id uint) error {
	return r.db.Model(&model.ErrorMessage{}).Where("id = ?", id).
		Update("enabled", gorm.Expr("NOT enabled")).Error
}

// Delete 删除错误消息配置
func (r *ErrorMessageRepository) Delete(id uint) error {
	return r.db.Delete(&model.ErrorMessage{}, id).Error
}

// EnableAll 启用所有错误消息
func (r *ErrorMessageRepository) EnableAll() (int64, error) {
	result := r.db.Model(&model.ErrorMessage{}).Where("enabled = ?", false).Update("enabled", true)
	return result.RowsAffected, result.Error
}

// DisableAll 禁用所有错误消息
func (r *ErrorMessageRepository) DisableAll() (int64, error) {
	result := r.db.Model(&model.ErrorMessage{}).Where("enabled = ?", true).Update("enabled", false)
	return result.RowsAffected, result.Error
}

// InitDefaultData 初始化默认数据
func (r *ErrorMessageRepository) InitDefaultData() error {
	for _, msg := range model.DefaultErrorMessages {
		var existing model.ErrorMessage
		result := r.db.Where("error_type = ?", msg.ErrorType).First(&existing)
		if result.Error == nil {
			// 已存在，跳过
			continue
		}
		if err := r.db.Create(&msg).Error; err != nil {
			return err
		}
	}
	return nil
}
