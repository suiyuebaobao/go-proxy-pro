package service

import (
	"fmt"
	"strings"
	"sync"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/pkg/logger"
)

// ErrorMessageService 错误消息服务
type ErrorMessageService struct {
	repo  *repository.ErrorMessageRepository
	cache map[string]*model.ErrorMessage // key: errorType
	mu    sync.RWMutex
	log   *logger.Logger
}

var errorMsgService *ErrorMessageService
var errorMsgOnce sync.Once

// GetErrorMessageService 获取错误消息服务单例
func GetErrorMessageService() *ErrorMessageService {
	errorMsgOnce.Do(func() {
		errorMsgService = &ErrorMessageService{
			repo:  repository.NewErrorMessageRepository(),
			cache: make(map[string]*model.ErrorMessage),
			log:   logger.GetLogger("error_message"),
		}
		errorMsgService.loadCache()
	})
	return errorMsgService
}

// loadCache 加载配置到缓存
func (s *ErrorMessageService) loadCache() error {
	messages, err := s.repo.GetAll()
	if err != nil {
		s.log.Error("加载错误消息配置失败: %v", err)
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 清空旧缓存
	s.cache = make(map[string]*model.ErrorMessage)

	for i := range messages {
		s.cache[messages[i].ErrorType] = &messages[i]
	}

	s.log.Info("错误消息配置已加载，共 %d 条", len(messages))
	return nil
}

// RefreshCache 刷新缓存
func (s *ErrorMessageService) RefreshCache() error {
	return s.loadCache()
}

// GetCustomMessage 获取自定义错误消息
// 返回: customMessage, originalError（用于日志）
// 如果未启用自定义消息，返回原始错误
func (s *ErrorMessageService) GetCustomMessage(errorType string, originalError string) (customMessage string, shouldLog bool) {
	s.mu.RLock()
	msg, exists := s.cache[errorType]
	s.mu.RUnlock()

	if !exists || !msg.Enabled {
		return originalError, false
	}

	// 自定义消息启用，需要记录原始错误
	return msg.CustomMessage, true
}

// GetMessageByType 根据错误类型获取消息配置
func (s *ErrorMessageService) GetMessageByType(errorType string) *model.ErrorMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cache[errorType]
}

// GetAll 获取所有错误消息配置
func (s *ErrorMessageService) GetAll() ([]model.ErrorMessage, error) {
	return s.repo.GetAll()
}

// GetByID 根据 ID 获取
func (s *ErrorMessageService) GetByID(id uint) (*model.ErrorMessage, error) {
	return s.repo.GetByID(id)
}

// GetByCode 根据 HTTP 状态码获取
func (s *ErrorMessageService) GetByCode(code int) ([]model.ErrorMessage, error) {
	return s.repo.GetByCode(code)
}

// Update 更新错误消息配置
func (s *ErrorMessageService) Update(id uint, customMessage string, enabled bool, description string) error {
	msg, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	msg.CustomMessage = customMessage
	msg.Enabled = enabled
	msg.Description = description

	if err := s.repo.Update(msg); err != nil {
		return err
	}

	// 更新缓存
	s.mu.Lock()
	s.cache[msg.ErrorType] = msg
	s.mu.Unlock()

	s.log.Info("错误消息配置已更新: %s", msg.ErrorType)
	return nil
}

// ToggleEnabled 切换启用状态
func (s *ErrorMessageService) ToggleEnabled(id uint) error {
	if err := s.repo.ToggleEnabled(id); err != nil {
		return err
	}

	// 重新加载缓存
	return s.loadCache()
}

// EnableAll 启用所有错误消息
func (s *ErrorMessageService) EnableAll() (int64, error) {
	affected, err := s.repo.EnableAll()
	if err != nil {
		return 0, err
	}

	// 重新加载缓存
	if err := s.loadCache(); err != nil {
		s.log.Error("刷新缓存失败: %v", err)
	}

	s.log.Info("批量启用错误消息配置，影响 %d 条", affected)
	return affected, nil
}

// DisableAll 禁用所有错误消息
func (s *ErrorMessageService) DisableAll() (int64, error) {
	affected, err := s.repo.DisableAll()
	if err != nil {
		return 0, err
	}

	// 重新加载缓存
	if err := s.loadCache(); err != nil {
		s.log.Error("刷新缓存失败: %v", err)
	}

	s.log.Info("批量禁用错误消息配置，影响 %d 条", affected)
	return affected, nil
}

// InitDefaultMessages 初始化默认错误消息
func (s *ErrorMessageService) InitDefaultMessages() error {
	if err := s.repo.InitDefaultData(); err != nil {
		return err
	}
	return s.loadCache()
}

// Create 创建新的错误消息配置
func (s *ErrorMessageService) Create(msg *model.ErrorMessage) error {
	if err := s.repo.Create(msg); err != nil {
		return err
	}

	// 更新缓存
	s.mu.Lock()
	s.cache[msg.ErrorType] = msg
	s.mu.Unlock()

	return nil
}

// Delete 删除错误消息配置
func (s *ErrorMessageService) Delete(id uint) error {
	msg, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(id); err != nil {
		return err
	}

	// 从缓存删除
	s.mu.Lock()
	delete(s.cache, msg.ErrorType)
	s.mu.Unlock()

	return nil
}

// ==================== 自动学习错误信息 ====================

// AutoDiscoverError 自动发现并注册新的错误类型
// 当遇到未知错误时，自动创建一条新的错误配置记录（默认禁用）
// errorType: 错误类型标识
// originalError: 原始错误信息（用于提取关键词和描述）
// httpCode: HTTP 状态码
func (s *ErrorMessageService) AutoDiscoverError(errorType string, originalError string, httpCode int) {
	// 检查是否已存在
	s.mu.RLock()
	_, exists := s.cache[errorType]
	s.mu.RUnlock()

	if exists {
		return
	}

	// 异步创建新的错误类型记录
	go s.createDiscoveredError(errorType, originalError, httpCode)
}

// createDiscoveredError 创建自动发现的错误记录
func (s *ErrorMessageService) createDiscoveredError(errorType string, originalError string, httpCode int) {
	// 再次检查（防止并发重复创建）
	s.mu.Lock()
	if _, exists := s.cache[errorType]; exists {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	// 提取错误关键信息作为描述
	description := truncateErrorMessage(originalError, 180)
	if description == "" {
		description = "自动发现的错误类型"
	}

	// 创建新记录（默认禁用，等待管理员确认）
	msg := &model.ErrorMessage{
		Code:          httpCode,
		ErrorType:     errorType,
		CustomMessage: "服务暂时不可用", // 默认消息
		Enabled:       false,          // 默认禁用，需要管理员手动启用
		Description:   "[自动发现] " + description,
	}

	if err := s.repo.CreateWithDisabled(msg); err != nil {
		// 如果是唯一键冲突（已存在），忽略错误
		if !isDuplicateKeyError(err) {
			s.log.Error("自动创建错误类型失败: %s, %v", errorType, err)
		}
		return
	}

	// 更新缓存
	s.mu.Lock()
	s.cache[errorType] = msg
	s.mu.Unlock()

	s.log.Info("自动发现新错误类型: %s (HTTP %d) | 原始错误: %s", errorType, httpCode, truncateErrorMessage(originalError, 100))
}

// ExtractErrorType 从错误信息中提取错误类型标识
// 将错误关键词转换为 snake_case 格式的类型标识
func ExtractErrorType(errMsg string) string {
	if errMsg == "" {
		return "unknown_error"
	}

	// 转小写
	errLower := strings.ToLower(errMsg)

	// 提取关键词并生成类型标识
	var typeWords []string

	// 按优先级匹配关键词
	keywords := []struct {
		pattern string
		word    string
	}{
		{"timeout", "timeout"},
		{"deadline exceeded", "timeout"},
		{"context canceled", "canceled"},
		{"rate limit", "rate_limit"},
		{"too many requests", "rate_limit"},
		{"overloaded", "overloaded"},
		{"unauthorized", "unauthorized"},
		{"authentication", "auth_failed"},
		{"permission denied", "permission_denied"},
		{"invalid api key", "invalid_api_key"},
		{"invalid_api_key", "invalid_api_key"},
		{"not found", "not_found"},
		{"bad request", "bad_request"},
		{"internal error", "internal_error"},
		{"server error", "server_error"},
		{"connection refused", "connection_refused"},
		{"connection reset", "connection_reset"},
		{"network error", "network_error"},
		{"ssl", "ssl_error"},
		{"certificate", "cert_error"},
		{"dns", "dns_error"},
		{"model not", "model_not_found"},
		{"unsupported", "unsupported"},
		{"quota", "quota_exceeded"},
		{"limit exceeded", "limit_exceeded"},
		{"insufficient", "insufficient"},
		{"blocked", "blocked"},
		{"forbidden", "forbidden"},
		{"invalid", "invalid"},
		{"failed", "failed"},
		{"error", "error"},
	}

	for _, kw := range keywords {
		if strings.Contains(errLower, kw.pattern) {
			typeWords = append(typeWords, kw.word)
			if len(typeWords) >= 2 {
				break
			}
		}
	}

	if len(typeWords) == 0 {
		// 无法识别，使用哈希
		return "auto_" + hashString(errMsg)[:8]
	}

	// 去重
	seen := make(map[string]bool)
	var unique []string
	for _, w := range typeWords {
		if !seen[w] {
			seen[w] = true
			unique = append(unique, w)
		}
	}

	return "auto_" + strings.Join(unique, "_")
}

// truncateErrorMessage 截断错误信息
func truncateErrorMessage(s string, maxLen int) string {
	// 移除换行符
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")

	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// hashString 简单哈希函数
func hashString(s string) string {
	h := uint32(0)
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return fmt.Sprintf("%08x", h)
}

// isDuplicateKeyError 检查是否是唯一键冲突错误
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "duplicate") || strings.Contains(errStr, "unique")
}
