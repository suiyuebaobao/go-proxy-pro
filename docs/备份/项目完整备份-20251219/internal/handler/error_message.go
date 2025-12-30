package handler

import (
	"go-aiproxy/internal/model"
	"go-aiproxy/internal/service"
	"go-aiproxy/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ErrorMessageHandler struct {
	service *service.ErrorMessageService
}

func NewErrorMessageHandler() *ErrorMessageHandler {
	return &ErrorMessageHandler{
		service: service.GetErrorMessageService(),
	}
}

// List 获取所有错误消息配置
// @Summary 获取错误消息配置列表
// @Tags 管理员-错误消息
// @Security Bearer
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/admin/error-messages [get]
func (h *ErrorMessageHandler) List(c *gin.Context) {
	messages, err := h.service.GetAll()
	if err != nil {
		response.InternalError(c, "获取错误消息配置失败: "+err.Error())
		return
	}
	response.Success(c, messages)
}

// GetByCode 根据 HTTP 状态码获取错误消息配置
// @Summary 根据状态码获取错误消息
// @Tags 管理员-错误消息
// @Security Bearer
// @Produce json
// @Param code path int true "HTTP 状态码"
// @Success 200 {object} response.Response
// @Router /api/admin/error-messages/code/{code} [get]
func (h *ErrorMessageHandler) GetByCode(c *gin.Context) {
	code, err := strconv.Atoi(c.Param("code"))
	if err != nil {
		response.BadRequest(c, "无效的状态码")
		return
	}

	messages, err := h.service.GetByCode(code)
	if err != nil {
		response.InternalError(c, "获取错误消息配置失败: "+err.Error())
		return
	}
	response.Success(c, messages)
}

// Get 获取单个错误消息配置
// @Summary 获取单个错误消息配置
// @Tags 管理员-错误消息
// @Security Bearer
// @Produce json
// @Param id path int true "错误消息 ID"
// @Success 200 {object} response.Response
// @Router /api/admin/error-messages/{id} [get]
func (h *ErrorMessageHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}

	msg, err := h.service.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "错误消息配置不存在")
		return
	}
	response.Success(c, msg)
}

// UpdateRequest 更新请求
type UpdateErrorMessageRequest struct {
	CustomMessage string `json:"custom_message" binding:"required"`
	Enabled       bool   `json:"enabled"`
	Description   string `json:"description"`
}

// Update 更新错误消息配置
// @Summary 更新错误消息配置
// @Tags 管理员-错误消息
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "错误消息 ID"
// @Param body body UpdateErrorMessageRequest true "更新内容"
// @Success 200 {object} response.Response
// @Router /api/admin/error-messages/{id} [put]
func (h *ErrorMessageHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}

	var req UpdateErrorMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.service.Update(uint(id), req.CustomMessage, req.Enabled, req.Description); err != nil {
		response.InternalError(c, "更新失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// ToggleEnabled 切换启用状态
// @Summary 切换错误消息启用状态
// @Tags 管理员-错误消息
// @Security Bearer
// @Produce json
// @Param id path int true "错误消息 ID"
// @Success 200 {object} response.Response
// @Router /api/admin/error-messages/{id}/toggle [put]
func (h *ErrorMessageHandler) ToggleEnabled(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}

	if err := h.service.ToggleEnabled(uint(id)); err != nil {
		response.InternalError(c, "切换状态失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// InitDefault 初始化默认错误消息配置
// @Summary 初始化默认错误消息配置
// @Tags 管理员-错误消息
// @Security Bearer
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/admin/error-messages/init [post]
func (h *ErrorMessageHandler) InitDefault(c *gin.Context) {
	if err := h.service.InitDefaultMessages(); err != nil {
		response.InternalError(c, "初始化失败: "+err.Error())
		return
	}
	response.Success(c, nil)
}

// RefreshCache 刷新缓存
// @Summary 刷新错误消息缓存
// @Tags 管理员-错误消息
// @Security Bearer
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/admin/error-messages/refresh [post]
func (h *ErrorMessageHandler) RefreshCache(c *gin.Context) {
	if err := h.service.RefreshCache(); err != nil {
		response.InternalError(c, "刷新缓存失败: "+err.Error())
		return
	}
	response.Success(c, nil)
}

// CreateRequest 创建请求
type CreateErrorMessageRequest struct {
	Code          int    `json:"code" binding:"required"`
	ErrorType     string `json:"error_type" binding:"required"`
	CustomMessage string `json:"custom_message" binding:"required"`
	Enabled       bool   `json:"enabled"`
	Description   string `json:"description"`
}

// Create 创建错误消息配置
// @Summary 创建错误消息配置
// @Tags 管理员-错误消息
// @Security Bearer
// @Accept json
// @Produce json
// @Param body body CreateErrorMessageRequest true "创建内容"
// @Success 200 {object} response.Response
// @Router /api/admin/error-messages [post]
func (h *ErrorMessageHandler) Create(c *gin.Context) {
	var req CreateErrorMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	msg := &model.ErrorMessage{
		Code:          req.Code,
		ErrorType:     req.ErrorType,
		CustomMessage: req.CustomMessage,
		Enabled:       req.Enabled,
		Description:   req.Description,
	}

	if err := h.service.Create(msg); err != nil {
		response.InternalError(c, "创建失败: "+err.Error())
		return
	}

	response.Success(c, msg)
}

// Delete 删除错误消息配置
// @Summary 删除错误消息配置
// @Tags 管理员-错误消息
// @Security Bearer
// @Produce json
// @Param id path int true "错误消息 ID"
// @Success 200 {object} response.Response
// @Router /api/admin/error-messages/{id} [delete]
func (h *ErrorMessageHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		response.InternalError(c, "删除失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// EnableAll 启用所有错误消息
// @Summary 启用所有错误消息
// @Tags 管理员-错误消息
// @Security Bearer
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/admin/error-messages/enable-all [put]
func (h *ErrorMessageHandler) EnableAll(c *gin.Context) {
	affected, err := h.service.EnableAll()
	if err != nil {
		response.InternalError(c, "批量启用失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"affected": affected,
		"message":  "已启用所有错误消息配置",
	})
}

// DisableAll 禁用所有错误消息
// @Summary 禁用所有错误消息
// @Tags 管理员-错误消息
// @Security Bearer
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/admin/error-messages/disable-all [put]
func (h *ErrorMessageHandler) DisableAll(c *gin.Context) {
	affected, err := h.service.DisableAll()
	if err != nil {
		response.InternalError(c, "批量禁用失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"affected": affected,
		"message":  "已禁用所有错误消息配置",
	})
}
