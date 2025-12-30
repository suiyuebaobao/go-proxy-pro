/*
 * 文件作用：系统配置处理器，处理动态配置的CRUD操作
 * 负责功能：
 *   - 获取所有配置（按分类分组）
 *   - 获取指定分类配置
 *   - 更新配置项
 * 重要程度：⭐⭐⭐ 一般（管理后台功能）
 * 依赖模块：service, model
 */
package handler

import (
	"go-aiproxy/internal/model"
	"go-aiproxy/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ConfigHandler struct {
	configService *service.ConfigService
}

func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{
		configService: service.GetConfigService(),
	}
}

// GetAll 获取所有配置
func (h *ConfigHandler) GetAll(c *gin.Context) {
	configs, err := h.configService.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 按分类分组
	grouped := make(map[string][]model.SystemConfig)
	for _, cfg := range configs {
		grouped[cfg.Category] = append(grouped[cfg.Category], cfg)
	}

	c.JSON(http.StatusOK, gin.H{
		"items":   configs,
		"grouped": grouped,
	})
}

// GetByCategory 获取分类配置
func (h *ConfigHandler) GetByCategory(c *gin.Context) {
	category := c.Param("category")
	configs, err := h.configService.GetByCategory(category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": configs})
}

// UpdateRequest 更新配置请求
type UpdateConfigRequest struct {
	Configs map[string]string `json:"configs" binding:"required"`
}

// Update 更新配置
func (h *ConfigHandler) Update(c *gin.Context) {
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	if err := h.configService.BatchSet(req.Configs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 刷新配置缓存（确保新配置立即生效）
	h.configService.RefreshCache()

	// 通知相关组件更新配置
	h.notifyConfigChange(req.Configs)

	c.JSON(http.StatusOK, gin.H{"message": "配置更新成功"})
}

// notifyConfigChange 通知配置变更
func (h *ConfigHandler) notifyConfigChange(configs map[string]string) {
	// 检查是否更新了会话 TTL
	if _, ok := configs[model.ConfigSessionTTL]; ok {
		if configChangeCallback != nil {
			configChangeCallback(model.ConfigSessionTTL, configs[model.ConfigSessionTTL])
		}
	}

	// 检查是否更新了健康检查配置
	if _, ok := configs[model.ConfigAccountHealthCheckEnabled]; ok {
		if configChangeCallback != nil {
			configChangeCallback(model.ConfigAccountHealthCheckEnabled, configs[model.ConfigAccountHealthCheckEnabled])
		}
	}

	if _, ok := configs[model.ConfigAccountHealthCheckInterval]; ok {
		if configChangeCallback != nil {
			configChangeCallback(model.ConfigAccountHealthCheckInterval, configs[model.ConfigAccountHealthCheckInterval])
		}
	}
}

// ConfigChangeCallback 配置变更回调函数类型
type ConfigChangeCallback func(key, value string)

var configChangeCallback ConfigChangeCallback

// SetConfigChangeCallback 设置配置变更回调
func SetConfigChangeCallback(callback ConfigChangeCallback) {
	configChangeCallback = callback
}
