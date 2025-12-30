/*
 * 文件作用：系统配置服务，管理运行时动态配置
 * 负责功能：
 *   - 配置缓存管理
 *   - 配置读取（按key/分类）
 *   - 配置更新
 *   - 预定义配置项获取方法
 *   - 配置变更通知
 * 重要程度：⭐⭐⭐⭐ 重要（配置管理核心）
 * 依赖模块：repository, model
 */
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

// GetGlobalPriceRate 获取全局价格倍率
func (s *ConfigService) GetGlobalPriceRate() float64 {
	val := s.GetFloat(model.ConfigGlobalPriceRate)
	if val <= 0 {
		return 1.0 // 默认 1.0（原价）
	}
	return val
}

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

// ========== 账号健康检查配置便捷方法 ==========

// GetAccountHealthCheckEnabled 获取是否启用账号健康检查
func (s *ConfigService) GetAccountHealthCheckEnabled() bool {
	return s.GetBool(model.ConfigAccountHealthCheckEnabled)
}

// GetAccountHealthCheckInterval 获取账号健康检查间隔
func (s *ConfigService) GetAccountHealthCheckInterval() time.Duration {
	duration := s.GetDuration(model.ConfigAccountHealthCheckInterval)
	if duration < time.Minute {
		return 5 * time.Minute // 默认 5 分钟
	}
	return duration
}

// GetAccountErrorThreshold 获取账号连续错误阈值
func (s *ConfigService) GetAccountErrorThreshold() int {
	val := s.GetInt(model.ConfigAccountErrorThreshold)
	if val <= 0 {
		return 5 // 默认值
	}
	return val
}

// ========== OAuth 自动重新授权配置便捷方法 ==========

// GetOAuthAutoReauthorizeEnabled 获取是否启用 OAuth 自动重新授权
func (s *ConfigService) GetOAuthAutoReauthorizeEnabled() bool {
	return s.GetBool(model.ConfigOAuthAutoReauthorizeEnabled)
}

// GetOAuthReauthorizeCooldown 获取 OAuth 重新授权冷却时间
func (s *ConfigService) GetOAuthReauthorizeCooldown() time.Duration {
	duration := s.GetDuration(model.ConfigOAuthReauthorizeCooldown)
	if duration < time.Minute {
		return 30 * time.Minute // 默认 30 分钟
	}
	return duration
}

// ========== 健康检测策略配置便捷方法 ==========

// GetHealthCheckAutoRecovery 获取是否启用自动恢复
func (s *ConfigService) GetHealthCheckAutoRecovery() bool {
	return s.GetBool(model.ConfigHealthCheckAutoRecovery)
}

// GetHealthCheckAutoTokenRefresh 获取是否启用 Token 自动刷新
func (s *ConfigService) GetHealthCheckAutoTokenRefresh() bool {
	return s.GetBool(model.ConfigHealthCheckAutoTokenRefresh)
}

// ========== 限流账号检测配置 ==========

// GetRateLimitedProbeEnabled 获取是否启用限流账号主动探测
func (s *ConfigService) GetRateLimitedProbeEnabled() bool {
	return s.GetBool(model.ConfigRateLimitedProbeEnabled)
}

// GetRateLimitedProbeInitInterval 获取限流账号初始探测间隔
func (s *ConfigService) GetRateLimitedProbeInitInterval() time.Duration {
	duration := s.GetDuration(model.ConfigRateLimitedProbeInitInterval)
	if duration < time.Minute {
		return 10 * time.Minute // 默认 10 分钟
	}
	return duration
}

// GetRateLimitedProbeMaxInterval 获取限流账号最大探测间隔
func (s *ConfigService) GetRateLimitedProbeMaxInterval() time.Duration {
	duration := s.GetDuration(model.ConfigRateLimitedProbeMaxInterval)
	if duration < time.Minute {
		return 30 * time.Minute // 默认 30 分钟
	}
	return duration
}

// GetRateLimitedProbeBackoff 获取限流账号探测间隔递增因子
func (s *ConfigService) GetRateLimitedProbeBackoff() float64 {
	val := s.GetFloat(model.ConfigRateLimitedProbeBackoff)
	if val <= 1 {
		return 1.5 // 默认 1.5
	}
	return val
}

// ========== 疑似封号检测配置 ==========

// GetSuspendedProbeInterval 获取疑似封号账号探测间隔
func (s *ConfigService) GetSuspendedProbeInterval() time.Duration {
	duration := s.GetDuration(model.ConfigSuspendedProbeInterval)
	if duration < time.Minute {
		return 5 * time.Minute // 默认 5 分钟
	}
	return duration
}

// GetSuspendedConfirmThreshold 获取确认封号阈值
func (s *ConfigService) GetSuspendedConfirmThreshold() int {
	val := s.GetInt(model.ConfigSuspendedConfirmThreshold)
	if val <= 0 {
		return 3 // 默认 3 次
	}
	return val
}

// ========== 已封号账号检测配置 ==========

// GetBannedProbeEnabled 获取是否启用封号账号复活检测
func (s *ConfigService) GetBannedProbeEnabled() bool {
	return s.GetBool(model.ConfigBannedProbeEnabled)
}

// GetBannedProbeInterval 获取封号账号复活探测间隔
func (s *ConfigService) GetBannedProbeInterval() time.Duration {
	val := s.GetInt(model.ConfigBannedProbeInterval)
	if val <= 0 {
		return 1 * time.Hour // 默认 1 小时
	}
	return time.Duration(val) * time.Hour
}

// ========== Token 刷新配置 ==========

// GetTokenRefreshCooldown 获取 Token 刷新失败冷却时间
func (s *ConfigService) GetTokenRefreshCooldown() time.Duration {
	duration := s.GetDuration(model.ConfigTokenRefreshCooldown)
	if duration < time.Minute {
		return 30 * time.Minute // 默认 30 分钟
	}
	return duration
}

// GetTokenRefreshMaxRetries 获取 Token 刷新最大重试次数
func (s *ConfigService) GetTokenRefreshMaxRetries() int {
	val := s.GetInt(model.ConfigTokenRefreshMaxRetries)
	if val <= 0 {
		return 3 // 默认 3 次
	}
	return val
}
