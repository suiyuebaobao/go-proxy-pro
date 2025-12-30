/*
 * 文件作用：系统监控处理器，提供服务器运行状态监控
 * 负责功能：
 *   - 完整监控数据聚合
 *   - 系统资源统计（CPU/内存/磁盘）
 *   - Redis缓存统计
 *   - MySQL连接统计
 *   - 账号/用户数量统计
 *   - 今日使用量统计
 * 重要程度：⭐⭐⭐ 一般（运维监控功能）
 * 依赖模块：service
 */
package handler

import (
	"go-aiproxy/internal/service"
	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

// SystemMonitorHandler 系统监控处理器
type SystemMonitorHandler struct {
	monitorService *service.SystemMonitorService
}

// NewSystemMonitorHandler 创建系统监控处理器
func NewSystemMonitorHandler() *SystemMonitorHandler {
	return &SystemMonitorHandler{
		monitorService: service.NewSystemMonitorService(),
	}
}

// GetMonitorData 获取完整监控数据
// @Summary 获取系统监控数据
// @Description 获取系统资源、Redis、MySQL、账号、用户、今日使用等完整监控数据
// @Tags 系统监控
// @Produce json
// @Success 200 {object} response.Response{data=service.MonitorData}
// @Router /api/admin/monitor [get]
func (h *SystemMonitorHandler) GetMonitorData(c *gin.Context) {
	data, err := h.monitorService.GetMonitorData(c.Request.Context())
	if err != nil {
		response.Error(c, 500, "获取监控数据失败: "+err.Error())
		return
	}
	response.Success(c, data)
}

// GetSystemStats 获取系统资源统计
// @Summary 获取系统资源统计
// @Description 获取 CPU、内存、磁盘使用情况
// @Tags 系统监控
// @Produce json
// @Success 200 {object} response.Response{data=service.SystemStats}
// @Router /api/admin/monitor/system [get]
func (h *SystemMonitorHandler) GetSystemStats(c *gin.Context) {
	stats := h.monitorService.GetSystemStats()
	response.Success(c, stats)
}

// GetCacheStats 获取缓存统计
// @Summary 获取缓存统计
// @Description 获取内存缓存状态、会话数量
// @Tags 系统监控
// @Produce json
// @Success 200 {object} response.Response{data=service.MemoryCacheStats}
// @Router /api/admin/monitor/cache [get]
func (h *SystemMonitorHandler) GetCacheStats(c *gin.Context) {
	stats := h.monitorService.GetCacheStats()
	response.Success(c, stats)
}

// GetMySQLStats 获取 MySQL 统计
// @Summary 获取 MySQL 统计
// @Description 获取 MySQL 连接状态、表数量、数据大小
// @Tags 系统监控
// @Produce json
// @Success 200 {object} response.Response{data=service.MySQLStats}
// @Router /api/admin/monitor/mysql [get]
func (h *SystemMonitorHandler) GetMySQLStats(c *gin.Context) {
	stats := h.monitorService.GetMySQLStats()
	response.Success(c, stats)
}

// GetAccountStats 获取账号统计
// @Summary 获取账号统计
// @Description 获取账号总数、正常、限流、无效数量
// @Tags 系统监控
// @Produce json
// @Success 200 {object} response.Response{data=service.AccountStats}
// @Router /api/admin/monitor/accounts [get]
func (h *SystemMonitorHandler) GetAccountStats(c *gin.Context) {
	stats := h.monitorService.GetAccountStats()
	response.Success(c, stats)
}

// GetUserStats 获取用户统计
// @Summary 获取用户统计
// @Description 获取用户总数、活跃数、今日新增数
// @Tags 系统监控
// @Produce json
// @Success 200 {object} response.Response{data=service.UserStats}
// @Router /api/admin/monitor/users [get]
func (h *SystemMonitorHandler) GetUserStats(c *gin.Context) {
	stats := h.monitorService.GetUserStats()
	response.Success(c, stats)
}

// GetTodayUsageStats 获取今日使用统计
// @Summary 获取今日使用统计
// @Description 获取今日消费、Token 使用、请求次数等
// @Tags 系统监控
// @Produce json
// @Success 200 {object} response.Response{data=service.TodayUsageStats}
// @Router /api/admin/monitor/today [get]
func (h *SystemMonitorHandler) GetTodayUsageStats(c *gin.Context) {
	stats := h.monitorService.GetTodayUsageStats(c.Request.Context())
	response.Success(c, stats)
}
