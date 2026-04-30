/*
 * 文件作用：告警管理 API 处理器
 * 负责功能：
 *   - 告警规则 CRUD 接口
 *   - 告警测试发送接口
 *   - 告警历史日志查询接口
 * 重要程度：⭐⭐⭐ 一般（告警管理 HTTP 层）
 * 依赖模块：service, repository, model, response
 */
package handler

import (
	"net/http"
	"strconv"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/internal/service"
	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

type AlertHandler struct {
	repo    *repository.AlertRepository
	service *service.AlertService
}

func NewAlertHandler() *AlertHandler {
	return &AlertHandler{
		repo:    repository.NewAlertRepository(),
		service: service.GetAlertService(),
	}
}

// ListRules 获取所有告警规则
func (h *AlertHandler) ListRules(c *gin.Context) {
	rules, err := h.repo.ListRules()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"items": rules})
}

// CreateRule 创建告警规则
func (h *AlertHandler) CreateRule(c *gin.Context) {
	var rule model.AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.repo.CreateRule(&rule); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, rule)
}

// UpdateRule 更新告警规则
func (h *AlertHandler) UpdateRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	existing, err := h.repo.GetRuleByID(uint(id))
	if err != nil {
		response.Error(c, http.StatusNotFound, "rule not found")
		return
	}

	if err := c.ShouldBindJSON(existing); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	existing.ID = uint(id)

	if err := h.repo.UpdateRule(existing); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, existing)
}

// DeleteRule 删除告警规则
func (h *AlertHandler) DeleteRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.repo.DeleteRule(uint(id)); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(c, nil)
}

// TestSend 测试发送告警
func (h *AlertHandler) TestSend(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid id")
		return
	}

	rule, err := h.repo.GetRuleByID(uint(id))
	if err != nil {
		response.Error(c, http.StatusNotFound, "rule not found")
		return
	}

	if err := h.service.TestSend(rule); err != nil {
		response.Error(c, http.StatusInternalServerError, "发送失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"message": "测试发送成功"})
}

// ListLogs 获取告警历史
func (h *AlertHandler) ListLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	logs, total, err := h.repo.ListLogs(page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	response.SuccessWithPagination(c, logs, total, page, pageSize)
}
