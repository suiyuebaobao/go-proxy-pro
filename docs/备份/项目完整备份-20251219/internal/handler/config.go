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

	// 通知相关组件更新配置
	h.notifyConfigChange(req.Configs)

	c.JSON(http.StatusOK, gin.H{"message": "配置更新成功"})
}

// notifyConfigChange 通知配置变更
func (h *ConfigHandler) notifyConfigChange(configs map[string]string) {
	// 检查是否更新了会话 TTL
	if _, ok := configs[model.ConfigSessionTTL]; ok {
		// 更新会话管理器的 TTL
		// 这里需要引入 scheduler 包，但会造成循环依赖
		// 所以使用回调机制
		if configChangeCallback != nil {
			configChangeCallback(model.ConfigSessionTTL, configs[model.ConfigSessionTTL])
		}
	}

	// 检查是否更新了同步配置
	if _, ok := configs[model.ConfigSyncInterval]; ok {
		if configChangeCallback != nil {
			configChangeCallback(model.ConfigSyncInterval, configs[model.ConfigSyncInterval])
		}
	}

	if _, ok := configs[model.ConfigSyncEnabled]; ok {
		if configChangeCallback != nil {
			configChangeCallback(model.ConfigSyncEnabled, configs[model.ConfigSyncEnabled])
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

// TriggerSync 手动触发同步
func (h *ConfigHandler) TriggerSync(c *gin.Context) {
	if syncTriggerCallback != nil {
		go syncTriggerCallback()
		c.JSON(http.StatusOK, gin.H{"message": "同步任务已触发"})
		return
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "同步服务未就绪"})
}

// SyncTriggerCallback 同步触发回调
type SyncTriggerCallback func()

var syncTriggerCallback SyncTriggerCallback

// SetSyncTriggerCallback 设置同步触发回调
func SetSyncTriggerCallback(callback SyncTriggerCallback) {
	syncTriggerCallback = callback
}

// GetSyncStatus 获取同步状态
func (h *ConfigHandler) GetSyncStatus(c *gin.Context) {
	if syncStatusCallback != nil {
		status := syncStatusCallback()
		c.JSON(http.StatusOK, status)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"enabled":   h.configService.GetSyncEnabled(),
		"interval":  h.configService.GetSyncInterval().Minutes(),
		"last_sync": nil,
		"status":    "unknown",
	})
}

// SyncStatusCallback 同步状态回调
type SyncStatusCallback func() map[string]interface{}

var syncStatusCallback SyncStatusCallback

// SetSyncStatusCallback 设置同步状态回调
func SetSyncStatusCallback(callback SyncStatusCallback) {
	syncStatusCallback = callback
}
