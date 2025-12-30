package repository

import (
	"go-aiproxy/internal/model"

	"gorm.io/gorm"
)

type ClientFilterRepository struct {
	db *gorm.DB
}

func NewClientFilterRepository() *ClientFilterRepository {
	return &ClientFilterRepository{db: DB}
}

// ==================== ClientType ====================

// CreateClientType 创建客户端类型
func (r *ClientFilterRepository) CreateClientType(ct *model.ClientType) error {
	return r.db.Create(ct).Error
}

// GetClientTypeByID 根据 ID 获取客户端类型
func (r *ClientFilterRepository) GetClientTypeByID(id uint) (*model.ClientType, error) {
	var ct model.ClientType
	err := r.db.First(&ct, id).Error
	if err != nil {
		return nil, err
	}
	return &ct, nil
}

// GetClientTypeByClientID 根据 client_id 获取客户端类型
func (r *ClientFilterRepository) GetClientTypeByClientID(clientID string) (*model.ClientType, error) {
	var ct model.ClientType
	err := r.db.Where("client_id = ?", clientID).First(&ct).Error
	if err != nil {
		return nil, err
	}
	return &ct, nil
}

// ListClientTypes 获取所有客户端类型
func (r *ClientFilterRepository) ListClientTypes() ([]model.ClientType, error) {
	var types []model.ClientType
	err := r.db.Order("priority DESC, id ASC").Find(&types).Error
	return types, err
}

// ListEnabledClientTypes 获取所有启用的客户端类型
func (r *ClientFilterRepository) ListEnabledClientTypes() ([]model.ClientType, error) {
	var types []model.ClientType
	err := r.db.Where("enabled = ?", true).Order("priority DESC, id ASC").Find(&types).Error
	return types, err
}

// UpdateClientType 更新客户端类型
func (r *ClientFilterRepository) UpdateClientType(ct *model.ClientType) error {
	return r.db.Model(&model.ClientType{}).Where("id = ?", ct.ID).Updates(map[string]interface{}{
		"client_id":   ct.ClientID,
		"name":        ct.Name,
		"description": ct.Description,
		"icon":        ct.Icon,
		"enabled":     ct.Enabled,
		"priority":    ct.Priority,
	}).Error
}

// DeleteClientType 删除客户端类型
func (r *ClientFilterRepository) DeleteClientType(id uint) error {
	return r.db.Delete(&model.ClientType{}, id).Error
}

// ==================== ClientFilterRule ====================

// CreateRule 创建过滤规则
func (r *ClientFilterRepository) CreateRule(rule *model.ClientFilterRule) error {
	return r.db.Create(rule).Error
}

// GetRuleByID 根据 ID 获取规则
func (r *ClientFilterRepository) GetRuleByID(id uint) (*model.ClientFilterRule, error) {
	var rule model.ClientFilterRule
	err := r.db.Preload("ClientType").First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// GetRuleByKey 根据规则标识获取规则
func (r *ClientFilterRepository) GetRuleByKey(clientTypeID uint, ruleKey string) (*model.ClientFilterRule, error) {
	var rule model.ClientFilterRule
	err := r.db.Where("client_type_id = ? AND rule_key = ?", clientTypeID, ruleKey).First(&rule).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// ListRulesByClientTypeID 获取指定客户端类型的所有规则
func (r *ClientFilterRepository) ListRulesByClientTypeID(clientTypeID uint) ([]model.ClientFilterRule, error) {
	var rules []model.ClientFilterRule
	err := r.db.Where("client_type_id = ?", clientTypeID).Order("priority DESC, id ASC").Find(&rules).Error
	return rules, err
}

// ListEnabledRulesByClientTypeID 获取指定客户端类型的启用规则
func (r *ClientFilterRepository) ListEnabledRulesByClientTypeID(clientTypeID uint) ([]model.ClientFilterRule, error) {
	var rules []model.ClientFilterRule
	err := r.db.Where("client_type_id = ? AND enabled = ?", clientTypeID, true).Order("priority DESC, id ASC").Find(&rules).Error
	return rules, err
}

// ListAllRules 获取所有规则（带客户端类型信息）
func (r *ClientFilterRepository) ListAllRules() ([]model.ClientFilterRule, error) {
	var rules []model.ClientFilterRule
	err := r.db.Preload("ClientType").Order("client_type_id ASC, priority DESC, id ASC").Find(&rules).Error
	return rules, err
}

// UpdateRule 更新规则
func (r *ClientFilterRepository) UpdateRule(rule *model.ClientFilterRule) error {
	return r.db.Model(&model.ClientFilterRule{}).Where("id = ?", rule.ID).Updates(map[string]interface{}{
		"rule_key":     rule.RuleKey,
		"rule_name":    rule.RuleName,
		"description":  rule.Description,
		"rule_type":    rule.RuleType,
		"pattern":      rule.Pattern,
		"field_path":   rule.FieldPath,
		"enabled":      rule.Enabled,
		"required":     rule.Required,
		"priority":     rule.Priority,
	}).Error
}

// DeleteRule 删除规则
func (r *ClientFilterRepository) DeleteRule(id uint) error {
	return r.db.Delete(&model.ClientFilterRule{}, id).Error
}

// DeleteRulesByClientTypeID 删除指定客户端类型的所有规则
func (r *ClientFilterRepository) DeleteRulesByClientTypeID(clientTypeID uint) error {
	return r.db.Where("client_type_id = ?", clientTypeID).Delete(&model.ClientFilterRule{}).Error
}

// ==================== ClientFilterConfig ====================

// GetConfig 获取全局配置（只有一条记录）
func (r *ClientFilterRepository) GetConfig() (*model.ClientFilterConfig, error) {
	var config model.ClientFilterConfig
	err := r.db.First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 返回默认配置
			return &model.ClientFilterConfig{
				FilterEnabled:        false,
				DefaultAllow:         true,
				LogUnmatchedRequests: true,
				StrictMode:           false,
			}, nil
		}
		return nil, err
	}
	return &config, nil
}

// SaveConfig 保存全局配置
func (r *ClientFilterRepository) SaveConfig(config *model.ClientFilterConfig) error {
	// 确保只有一条记录
	var existing model.ClientFilterConfig
	err := r.db.First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(config).Error
	}
	// 只更新可变字段
	return r.db.Model(&existing).Updates(map[string]interface{}{
		"filter_enabled":         config.FilterEnabled,
		"filter_mode":            config.FilterMode,
		"default_allow":          config.DefaultAllow,
		"log_unmatched_requests": config.LogUnmatchedRequests,
		"strict_mode":            config.StrictMode,
		"allowed_clients":        config.AllowedClients,
	}).Error
}

// ==================== 初始化默认数据 ====================

// InitDefaultData 初始化默认的客户端类型和规则
func (r *ClientFilterRepository) InitDefaultData() error {
	// 检查是否已有数据
	var count int64
	r.db.Model(&model.ClientType{}).Count(&count)
	if count > 0 {
		return nil // 已有数据，不再初始化
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		// 创建默认客户端类型
		clientTypeMap := make(map[string]uint)
		for _, ct := range model.DefaultClientTypes {
			if err := tx.Create(&ct).Error; err != nil {
				return err
			}
			clientTypeMap[ct.ClientID] = ct.ID
		}

		// 创建 Claude Code 规则
		if clientTypeID, ok := clientTypeMap[model.ClientIDClaudeCode]; ok {
			for _, rule := range model.DefaultClaudeCodeRules {
				rule.ClientTypeID = clientTypeID
				if err := tx.Create(&rule).Error; err != nil {
					return err
				}
			}
		}

		// 创建 Codex CLI 规则
		if clientTypeID, ok := clientTypeMap[model.ClientIDCodexCLI]; ok {
			for _, rule := range model.DefaultCodexCLIRules {
				rule.ClientTypeID = clientTypeID
				if err := tx.Create(&rule).Error; err != nil {
					return err
				}
			}
		}

		// 创建 Gemini CLI 规则
		if clientTypeID, ok := clientTypeMap[model.ClientIDGeminiCLI]; ok {
			for _, rule := range model.DefaultGeminiCLIRules {
				rule.ClientTypeID = clientTypeID
				if err := tx.Create(&rule).Error; err != nil {
					return err
				}
			}
		}

		// 创建默认全局配置
		config := &model.ClientFilterConfig{
			FilterEnabled:        false, // 默认关闭
			DefaultAllow:         true,
			LogUnmatchedRequests: true,
			StrictMode:           false,
		}
		return tx.Create(config).Error
	})
}
