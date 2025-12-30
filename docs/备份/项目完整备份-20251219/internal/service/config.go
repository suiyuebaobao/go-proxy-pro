package service

import (
	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"strconv"
	"sync"
	"time"
)

// ConfigService 系统配置服务
type ConfigService struct {
	repo  *repository.SystemConfigRepository
	cache map[string]string
	mu    sync.RWMutex
}

var configService *ConfigService
var configOnce sync.Once

// GetConfigService 获取配置服务单例
func GetConfigService() *ConfigService {
	configOnce.Do(func() {
		configService = &ConfigService{
			repo:  repository.NewSystemConfigRepository(),
			cache: make(map[string]string),
		}
		configService.loadCache()
	})
	return configService
}

// loadCache 加载配置到缓存
func (s *ConfigService) loadCache() error {
	configs, err := s.repo.GetAll()
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, cfg := range configs {
		s.cache[cfg.Key] = cfg.Value
	}
	return nil
}

// RefreshCache 刷新缓存
func (s *ConfigService) RefreshCache() error {
	return s.loadCache()
}

// GetString 获取字符串配置
func (s *ConfigService) GetString(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cache[key]
}

// GetInt 获取整数配置
func (s *ConfigService) GetInt(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, _ := strconv.Atoi(s.cache[key])
	return val
}

// GetFloat 获取浮点数配置
func (s *ConfigService) GetFloat(key string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, _ := strconv.ParseFloat(s.cache[key], 64)
	return val
}

// GetBool 获取布尔配置
func (s *ConfigService) GetBool(key string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cache[key] == "true"
}

// GetDuration 获取时间间隔配置（分钟）
func (s *ConfigService) GetDuration(key string) time.Duration {
	return time.Duration(s.GetInt(key)) * time.Minute
}

// Set 设置配置
func (s *ConfigService) Set(key, value string) error {
	if err := s.repo.Update(key, value); err != nil {
		return err
	}

	s.mu.Lock()
	s.cache[key] = value
	s.mu.Unlock()

	return nil
}

// BatchSet 批量设置配置
func (s *ConfigService) BatchSet(configs map[string]string) error {
	if err := s.repo.BatchUpdate(configs); err != nil {
		return err
	}

	s.mu.Lock()
	for key, value := range configs {
		s.cache[key] = value
	}
	s.mu.Unlock()

	return nil
}

// GetAll 获取所有配置
func (s *ConfigService) GetAll() ([]model.SystemConfig, error) {
	return s.repo.GetAll()
}

// GetByCategory 获取分类配置
func (s *ConfigService) GetByCategory(category string) ([]model.SystemConfig, error) {
	return s.repo.GetByCategory(category)
}

// ========== 便捷方法 ==========

// GetSessionTTL 获取会话 TTL
func (s *ConfigService) GetSessionTTL() time.Duration {
	return s.GetDuration(model.ConfigSessionTTL)
}

// GetSyncEnabled 获取是否启用同步
func (s *ConfigService) GetSyncEnabled() bool {
	return s.GetBool(model.ConfigSyncEnabled)
}

// GetSyncInterval 获取同步间隔
func (s *ConfigService) GetSyncInterval() time.Duration {
	return s.GetDuration(model.ConfigSyncInterval)
}

// GetRecordRetentionDays 获取记录保留天数
func (s *ConfigService) GetRecordRetentionDays() int {
	return s.GetInt(model.ConfigRecordRetentionDays)
}

// GetRecordMaxCount 获取最大记录数
func (s *ConfigService) GetRecordMaxCount() int {
	return s.GetInt(model.ConfigRecordMaxCount)
}

// ========== 安全配置便捷方法 ==========

// GetCaptchaEnabled 获取是否启用验证码
func (s *ConfigService) GetCaptchaEnabled() bool {
	return s.GetBool(model.ConfigCaptchaEnabled)
}

// GetCaptchaRateLimit 获取验证码获取频率限制（次/分钟）
func (s *ConfigService) GetCaptchaRateLimit() int {
	val := s.GetInt(model.ConfigCaptchaRateLimit)
	if val <= 0 {
		return 10 // 默认值
	}
	return val
}

// GetLoginRateLimitEnabled 获取是否启用登录频率限制
func (s *ConfigService) GetLoginRateLimitEnabled() bool {
	return s.GetBool(model.ConfigLoginRateLimitEnable)
}

// GetLoginRateLimitCount 获取登录频率限制次数
func (s *ConfigService) GetLoginRateLimitCount() int {
	val := s.GetInt(model.ConfigLoginRateLimitCount)
	if val <= 0 {
		return 3 // 默认值
	}
	return val
}

// GetLoginRateLimitWindow 获取登录频率限制时间窗口（分钟）
func (s *ConfigService) GetLoginRateLimitWindow() int {
	val := s.GetInt(model.ConfigLoginRateLimitWindow)
	if val <= 0 {
		return 5 // 默认值
	}
	return val
}
