/*
 * 文件作用：应用配置加载，从YAML文件读取系统配置
 * 负责功能：
 *   - 配置文件解析（YAML格式）
 *   - 服务器/数据库/JWT/缓存配置
 *   - 配置默认值处理
 *   - 全局配置实例管理
 * 重要程度：⭐⭐⭐⭐ 重要（系统配置核心）
 * 依赖模块：yaml
 */
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	MySQL  MySQLConfig  `yaml:"mysql"`
	JWT    JWTConfig    `yaml:"jwt"`
	Log    LogConfig    `yaml:"log"`
	Cache  CacheConfig  `yaml:"cache"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"`
}

type LogConfig struct {
	Dir   string `yaml:"dir"`   // 日志目录
	Level string `yaml:"level"` // 日志级别: debug, info, warn, error
}

type MySQLConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	Database     string `yaml:"database"`
	Charset      string `yaml:"charset"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
	MaxOpenConns int    `yaml:"max_open_conns"`
}

func (c *MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.Database, c.Charset)
}

type JWTConfig struct {
	Secret      string `yaml:"secret"`
	ExpireHours int    `yaml:"expire_hours"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	SessionTTL            int `yaml:"session_ttl"`             // 会话绑定 TTL（分钟），默认 60
	SessionRenewalTTL     int `yaml:"session_renewal_ttl"`     // 会话续期阈值（分钟），默认 14
	UnavailableTTL        int `yaml:"unavailable_ttl"`         // 临时不可用 TTL（分钟），默认 5
	ConcurrencyTTL        int `yaml:"concurrency_ttl"`         // 并发计数 TTL（分钟），默认 5
	DefaultConcurrencyMax int `yaml:"default_concurrency_max"` // 默认并发上限，默认 5
}

// GetSessionTTL 获取会话 TTL（分钟）
func (c *CacheConfig) GetSessionTTL() int {
	if c.SessionTTL <= 0 {
		return 60
	}
	return c.SessionTTL
}

// GetSessionRenewalTTL 获取会话续期阈值（分钟）
func (c *CacheConfig) GetSessionRenewalTTL() int {
	if c.SessionRenewalTTL <= 0 {
		return 14
	}
	return c.SessionRenewalTTL
}

// GetUnavailableTTL 获取不可用 TTL（分钟）
func (c *CacheConfig) GetUnavailableTTL() int {
	if c.UnavailableTTL <= 0 {
		return 5
	}
	return c.UnavailableTTL
}

// GetConcurrencyTTL 获取并发计数 TTL（分钟）
func (c *CacheConfig) GetConcurrencyTTL() int {
	if c.ConcurrencyTTL <= 0 {
		return 5
	}
	return c.ConcurrencyTTL
}

// GetDefaultConcurrencyMax 获取默认并发上限
func (c *CacheConfig) GetDefaultConcurrencyMax() int {
	if c.DefaultConcurrencyMax <= 0 {
		return 5
	}
	return c.DefaultConcurrencyMax
}

var Cfg *Config

func Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	Cfg = &Config{}
	if err := yaml.Unmarshal(data, Cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	return nil
}
