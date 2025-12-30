package handler

import (
	"strconv"
	"time"

	"go-aiproxy/internal/repository"
	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

type OperationLogHandler struct {
	repo *repository.OperationLogRepository
}

func NewOperationLogHandler() *OperationLogHandler {
	return &OperationLogHandler{
		repo: repository.NewOperationLogRepository(),
	}
}

// List 查询操作日志列表
func (h *OperationLogHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	filters := make(map[string]interface{})

	// 用户ID过滤
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
			filters["user_id"] = uint(userID)
		}
	}

	// 模块过滤
	if module := c.Query("module"); module != "" {
		filters["module"] = module
	}

	// 操作类型过滤
	if action := c.Query("action"); action != "" {
		filters["action"] = action
	}

	// 时间范围过滤
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse("2006-01-02", startTimeStr); err == nil {
			filters["start_time"] = startTime
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse("2006-01-02", endTimeStr); err == nil {
			// 设置为当天结束
			filters["end_time"] = endTime.Add(24*time.Hour - time.Second)
		}
	}

	// 搜索
	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}

	logs, total, err := h.repo.List(page, pageSize, filters)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"items": logs,
		"total": total,
		"page":  page,
	})
}

// Get 获取单条操作日志详情
func (h *OperationLogHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	log, err := h.repo.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "log not found")
		return
	}

	response.Success(c, log)
}

// GetStats 获取操作日志统计
func (h *OperationLogHandler) GetStats(c *gin.Context) {
	// 默认统计最近7天
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days < 1 || days > 90 {
		days = 7
	}

	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	stats, err := h.repo.GetStats(startTime, endTime)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, stats)
}

// Cleanup 清理旧日志
func (h *OperationLogHandler) Cleanup(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "90"))
	if days < 7 {
		days = 7 // 最少保留7天
	}

	deleted, err := h.repo.DeleteOldLogs(days)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"deleted": deleted,
		"message": "清理完成",
	})
}
