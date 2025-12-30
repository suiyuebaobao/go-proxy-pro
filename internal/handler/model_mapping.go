/*
 * 文件作用：模型映射处理器，处理模型名称映射的CRUD操作
 * 负责功能：
 *   - 模型映射列表查询
 *   - 模型映射创建/更新/删除
 *   - 模型映射启用/禁用
 *   - 映射缓存刷新
 *   - 缓存统计查询
 * 重要程度：⭐⭐⭐ 一般（模型映射功能）
 * 依赖模块：service, model
 */
package handler

import (
	"strconv"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/service"
	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

// ModelMappingHandler 模型映射处理器
type ModelMappingHandler struct {
	service *service.ModelMappingService
}

// NewModelMappingHandler 创建模型映射处理器
func NewModelMappingHandler() *ModelMappingHandler {
	return &ModelMappingHandler{
		service: service.NewModelMappingService(),
	}
}

// List 获取所有模型映射
// GET /api/admin/model-mappings
func (h *ModelMappingHandler) List(c *gin.Context) {
	mappings, err := h.service.List()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"mappings": mappings})
}

// Create 创建模型映射
// POST /api/admin/model-mappings
func (h *ModelMappingHandler) Create(c *gin.Context) {
	var req model.CreateModelMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomBadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	mapping, err := h.service.Create(&req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, mapping)
}

// Get 获取单个模型映射
// GET /api/admin/model-mappings/:id
func (h *ModelMappingHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.CustomBadRequest(c, "无效的ID")
		return
	}

	mapping, err := h.service.GetByID(uint(id))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, mapping)
}

// Update 更新模型映射
// PUT /api/admin/model-mappings/:id
func (h *ModelMappingHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.CustomBadRequest(c, "无效的ID")
		return
	}

	var req model.UpdateModelMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.CustomBadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	mapping, err := h.service.Update(uint(id), &req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, mapping)
}

// Delete 删除模型映射
// DELETE /api/admin/model-mappings/:id
func (h *ModelMappingHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.CustomBadRequest(c, "无效的ID")
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "删除成功"})
}

// Toggle 切换启用状态
// POST /api/admin/model-mappings/:id/toggle
func (h *ModelMappingHandler) Toggle(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.CustomBadRequest(c, "无效的ID")
		return
	}

	mapping, err := h.service.ToggleEnabled(uint(id))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, mapping)
}

// RefreshCache 刷新缓存
// POST /api/admin/model-mappings/refresh
func (h *ModelMappingHandler) RefreshCache(c *gin.Context) {
	h.service.RefreshCache()
	response.Success(c, gin.H{"message": "缓存已刷新"})
}

// GetCacheStats 获取缓存统计
// GET /api/admin/model-mappings/cache
func (h *ModelMappingHandler) GetCacheStats(c *gin.Context) {
	stats := h.service.GetCacheStats()
	response.Success(c, stats)
}
