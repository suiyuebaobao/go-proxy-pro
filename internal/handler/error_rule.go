/*
 * 文件作用：错误规则处理器，管理错误匹配和处理规则
 * 负责功能：
 *   - 错误规则CRUD
 *   - 规则启用/禁用
 *   - 默认规则重置
 *   - 规则缓存刷新
 * 重要程度：⭐⭐⭐ 一般（错误处理增强）
 * 依赖模块：service, errormatch
 */
package handler

import (
	"net/http"
	"strconv"

	"go-aiproxy/internal/errormatch"
	"go-aiproxy/internal/service"

	"github.com/gin-gonic/gin"
)

type ErrorRuleHandler struct {
	service *service.ErrorRuleService
}

func NewErrorRuleHandler() *ErrorRuleHandler {
	return &ErrorRuleHandler{
		service: service.NewErrorRuleService(),
	}
}

// Create 创建错误规则
func (h *ErrorRuleHandler) Create(c *gin.Context) {
	var req service.CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	rule, err := h.service.Create(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    rule,
	})
}

// Get 获取错误规则
func (h *ErrorRuleHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	rule, err := h.service.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "规则不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    rule,
	})
}

// Update 更新错误规则
func (h *ErrorRuleHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	var req service.UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	rule, err := h.service.Update(uint(id), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    rule,
	})
}

// Delete 删除错误规则
func (h *ErrorRuleHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的ID",
		})
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}

// List 列表查询错误规则
func (h *ErrorRuleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	rules, total, err := h.service.List(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"items": rules,
			"total": total,
			"page":  page,
			"size":  pageSize,
		},
	})
}

// ResetToDefault 重置为默认规则
func (h *ErrorRuleHandler) ResetToDefault(c *gin.Context) {
	if err := h.service.ResetToDefault(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "重置失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
	})
}

// RefreshCache 刷新缓存
func (h *ErrorRuleHandler) RefreshCache(c *gin.Context) {
	errormatch.GetErrorRuleMatcher().Refresh()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"rule_count": errormatch.GetErrorRuleMatcher().GetRuleCount(),
		},
	})
}

// EnableAll 启用所有规则
func (h *ErrorRuleHandler) EnableAll(c *gin.Context) {
	affected, err := h.service.EnableAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "操作失败: " + err.Error(),
		})
		return
	}

	// 刷新缓存
	errormatch.GetErrorRuleMatcher().Refresh()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"affected": affected,
		},
	})
}

// DisableAll 禁用所有规则
func (h *ErrorRuleHandler) DisableAll(c *gin.Context) {
	affected, err := h.service.DisableAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "操作失败: " + err.Error(),
		})
		return
	}

	// 刷新缓存
	errormatch.GetErrorRuleMatcher().Refresh()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"affected": affected,
		},
	})
}
