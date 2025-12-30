package handler

import (
	"net/http"
	"strconv"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

type AIModelHandler struct {
	repo *repository.AIModelRepository
}

func NewAIModelHandler(repo *repository.AIModelRepository) *AIModelHandler {
	return &AIModelHandler{repo: repo}
}

// List 获取模型列表
func (h *AIModelHandler) List(c *gin.Context) {
	platform := c.Query("platform")
	enabledStr := c.Query("enabled")

	var enabled *bool
	if enabledStr != "" {
		b := enabledStr == "true" || enabledStr == "1"
		enabled = &b
	}

	models, err := h.repo.List(platform, enabled)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取模型列表失败")
		return
	}

	response.Success(c, models)
}

// Get 获取单个模型
func (h *AIModelHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	m, err := h.repo.GetByID(uint(id))
	if err != nil {
		response.Error(c, http.StatusNotFound, "模型不存在")
		return
	}

	response.Success(c, m)
}

// Create 创建模型
func (h *AIModelHandler) Create(c *gin.Context) {
	var m model.AIModel
	if err := c.ShouldBindJSON(&m); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	// 检查名称是否已存在
	existing, _ := h.repo.GetByName(m.Name)
	if existing != nil {
		response.Error(c, http.StatusConflict, "模型名称已存在")
		return
	}

	if err := h.repo.Create(&m); err != nil {
		response.Error(c, http.StatusInternalServerError, "创建模型失败")
		return
	}

	response.Success(c, m)
}

// Update 更新模型
func (h *AIModelHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	existing, err := h.repo.GetByID(uint(id))
	if err != nil {
		response.Error(c, http.StatusNotFound, "模型不存在")
		return
	}

	var updates model.AIModel
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.Error(c, http.StatusBadRequest, "无效的请求数据")
		return
	}

	// 更新字段
	existing.Name = updates.Name
	existing.DisplayName = updates.DisplayName
	existing.Platform = updates.Platform
	existing.Provider = updates.Provider
	existing.Description = updates.Description
	existing.Category = updates.Category
	existing.ContextSize = updates.ContextSize
	existing.MaxOutput = updates.MaxOutput
	existing.InputPrice = updates.InputPrice
	existing.OutputPrice = updates.OutputPrice
	existing.Enabled = updates.Enabled
	existing.IsDefault = updates.IsDefault
	existing.SortOrder = updates.SortOrder
	existing.Aliases = updates.Aliases
	existing.Capabilities = updates.Capabilities

	if err := h.repo.Update(existing); err != nil {
		response.Error(c, http.StatusInternalServerError, "更新模型失败")
		return
	}

	response.Success(c, existing)
}

// Delete 删除模型
func (h *AIModelHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := h.repo.Delete(uint(id)); err != nil {
		response.Error(c, http.StatusInternalServerError, "删除模型失败")
		return
	}

	response.Success(c, nil)
}

// GetPlatforms 获取平台列表
func (h *AIModelHandler) GetPlatforms(c *gin.Context) {
	platforms, err := h.repo.GetPlatforms()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取平台列表失败")
		return
	}

	response.Success(c, platforms)
}

// InitDefaults 初始化默认模型
func (h *AIModelHandler) InitDefaults(c *gin.Context) {
	if err := h.repo.InitDefaultModels(); err != nil {
		response.Error(c, http.StatusInternalServerError, "初始化默认模型失败")
		return
	}

	response.Success(c, gin.H{"message": "初始化成功"})
}

// ResetDefaults 重置为默认模型
func (h *AIModelHandler) ResetDefaults(c *gin.Context) {
	if err := h.repo.ResetDefaultModels(); err != nil {
		response.Error(c, http.StatusInternalServerError, "重置默认模型失败")
		return
	}

	response.Success(c, gin.H{"message": "重置成功"})
}

// ToggleEnabled 切换启用状态
func (h *AIModelHandler) ToggleEnabled(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	m, err := h.repo.GetByID(uint(id))
	if err != nil {
		response.Error(c, http.StatusNotFound, "模型不存在")
		return
	}

	m.Enabled = !m.Enabled
	if err := h.repo.Update(m); err != nil {
		response.Error(c, http.StatusInternalServerError, "更新状态失败")
		return
	}

	response.Success(c, m)
}
