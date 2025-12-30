/*
 * 文件作用：请求日志处理器，提供API请求日志的查询和统计
 * 负责功能：
 *   - 请求日志列表查询（分页、筛选）
 *   - 请求汇总统计
 *   - 账户负载统计
 *   - 按时间范围查询
 * 重要程度：⭐⭐⭐ 一般（日志查询功能）
 * 依赖模块：repository
 */
package handler

import (
	"net/http"
	"strconv"
	"time"

	"go-aiproxy/internal/repository"
	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

type RequestLogHandler struct {
	repo *repository.RequestLogRepository
}

func NewRequestLogHandler() *RequestLogHandler {
	return &RequestLogHandler{
		repo: repository.NewRequestLogRepository(),
	}
}

// List 获取请求日志列表
func (h *RequestLogHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filters := make(map[string]interface{})

	if accountID, _ := strconv.ParseUint(c.Query("account_id"), 10, 32); accountID > 0 {
		filters["account_id"] = uint(accountID)
	}
	if platform := c.Query("platform"); platform != "" {
		filters["platform"] = platform
	}
	if model := c.Query("model"); model != "" {
		filters["model"] = model
	}
	if success := c.Query("success"); success != "" {
		filters["success"] = success == "true"
	}
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filters["start_time"] = t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filters["end_time"] = t
		}
	}

	logs, total, err := h.repo.List(page, pageSize, filters)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.SuccessWithPagination(c, logs, total, page, pageSize)
}

// GetSummary 获取请求统计摘要
func (h *RequestLogHandler) GetSummary(c *gin.Context) {
	// 默认最近24小时
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	if start := c.Query("start_time"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			startTime = t
		}
	}
	if end := c.Query("end_time"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			endTime = t
		}
	}

	summary, err := h.repo.GetSummary(startTime, endTime)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, summary)
}

// GetAccountLoadStats 获取账户负载统计
func (h *RequestLogHandler) GetAccountLoadStats(c *gin.Context) {
	// 默认最近24小时
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	if start := c.Query("start_time"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			startTime = t
		}
	}
	if end := c.Query("end_time"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			endTime = t
		}
	}

	stats, err := h.repo.GetAccountLoadStats(startTime, endTime)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, stats)
}
