/*
 * 文件作用：客户端过滤处理器，管理客户端类型和过滤规则
 * 负责功能：
 *   - 全局过滤配置管理
 *   - 客户端类型定义（Claude Code/Cursor/Cline等）
 *   - 过滤规则CRUD
 *   - 规则测试验证
 *   - 缓存刷新
 * 重要程度：⭐⭐⭐ 一般（客户端过滤功能）
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

type ClientFilterHandler struct {
	service *service.ClientFilterService
}

func NewClientFilterHandler() *ClientFilterHandler {
	return &ClientFilterHandler{
		service: service.GetClientFilterService(),
	}
}

// ==================== 全局配置 ====================

// GetConfig 获取全局配置
func (h *ClientFilterHandler) GetConfig(c *gin.Context) {
	config := h.service.GetConfig()
	response.Success(c, config)
}

// UpdateConfig 更新全局配置
func (h *ClientFilterHandler) UpdateConfig(c *gin.Context) {
	var config model.ClientFilterConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.BadRequest(c, "无效的配置数据")
		return
	}

	if err := h.service.SaveConfig(&config); err != nil {
		response.InternalError(c, "保存配置失败: "+err.Error())
		return
	}

	response.Success(c, config)
}

// ==================== 客户端类型管理 ====================

// ListClientTypes 获取所有客户端类型
func (h *ClientFilterHandler) ListClientTypes(c *gin.Context) {
	types, err := h.service.GetAllClientTypes()
	if err != nil {
		response.InternalError(c, "获取客户端类型失败: "+err.Error())
		return
	}

	// 为每个类型加载规则数量
	type ClientTypeWithRuleCount struct {
		model.ClientType
		RuleCount int `json:"rule_count"`
	}

	result := make([]ClientTypeWithRuleCount, 0, len(types))
	for _, ct := range types {
		rules, _ := h.service.GetRulesByClientType(ct.ID)
		result = append(result, ClientTypeWithRuleCount{
			ClientType: ct,
			RuleCount:  len(rules),
		})
	}

	response.Success(c, result)
}

// GetClientType 获取单个客户端类型
func (h *ClientFilterHandler) GetClientType(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}

	ct, err := h.service.GetClientType(uint(id))
	if err != nil {
		response.NotFound(c, "客户端类型不存在")
		return
	}

	response.Success(c, ct)
}

// CreateClientType 创建客户端类型
func (h *ClientFilterHandler) CreateClientType(c *gin.Context) {
	var ct model.ClientType
	if err := c.ShouldBindJSON(&ct); err != nil {
		response.BadRequest(c, "无效的数据")
		return
	}

	if ct.ClientID == "" {
		response.BadRequest(c, "client_id 不能为空")
		return
	}
	if ct.Name == "" {
		response.BadRequest(c, "name 不能为空")
		return
	}

	if err := h.service.CreateClientType(&ct); err != nil {
		response.InternalError(c, "创建客户端类型失败: "+err.Error())
		return
	}

	response.Created(c, ct)
}

// UpdateClientType 更新客户端类型
func (h *ClientFilterHandler) UpdateClientType(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}

	var ct model.ClientType
	if err := c.ShouldBindJSON(&ct); err != nil {
		response.BadRequest(c, "无效的数据")
		return
	}

	ct.ID = uint(id)
	if err := h.service.UpdateClientType(&ct); err != nil {
		response.InternalError(c, "更新客户端类型失败: "+err.Error())
		return
	}

	response.Success(c, ct)
}

// DeleteClientType 删除客户端类型
func (h *ClientFilterHandler) DeleteClientType(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}

	if err := h.service.DeleteClientType(uint(id)); err != nil {
		response.InternalError(c, "删除客户端类型失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// ToggleClientType 切换客户端类型启用状态
func (h *ClientFilterHandler) ToggleClientType(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}

	ct, err := h.service.GetClientType(uint(id))
	if err != nil {
		response.NotFound(c, "客户端类型不存在")
		return
	}

	ct.Enabled = !ct.Enabled
	if err := h.service.UpdateClientType(ct); err != nil {
		response.InternalError(c, "更新客户端类型失败: "+err.Error())
		return
	}

	response.Success(c, ct)
}

// ==================== 过滤规则管理 ====================

// ListRules 获取所有规则
func (h *ClientFilterHandler) ListRules(c *gin.Context) {
	// 检查是否按客户端类型过滤
	clientTypeIDStr := c.Query("client_type_id")
	if clientTypeIDStr != "" {
		clientTypeID, err := strconv.ParseUint(clientTypeIDStr, 10, 64)
		if err != nil {
			response.BadRequest(c, "无效的 client_type_id")
			return
		}
		rules, err := h.service.GetRulesByClientType(uint(clientTypeID))
		if err != nil {
			response.InternalError(c, "获取规则失败: "+err.Error())
			return
		}
		response.Success(c, rules)
		return
	}

	// 获取所有规则
	rules, err := h.service.GetAllRules()
	if err != nil {
		response.InternalError(c, "获取规则失败: "+err.Error())
		return
	}

	response.Success(c, rules)
}

// GetRule 获取单个规则
func (h *ClientFilterHandler) GetRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}

	rule, err := h.service.GetRule(uint(id))
	if err != nil {
		response.NotFound(c, "规则不存在")
		return
	}

	response.Success(c, rule)
}

// CreateRule 创建规则
func (h *ClientFilterHandler) CreateRule(c *gin.Context) {
	var rule model.ClientFilterRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		response.BadRequest(c, "无效的数据")
		return
	}

	if rule.ClientTypeID == 0 {
		response.BadRequest(c, "client_type_id 不能为空")
		return
	}
	if rule.RuleKey == "" {
		response.BadRequest(c, "rule_key 不能为空")
		return
	}
	if rule.RuleName == "" {
		response.BadRequest(c, "rule_name 不能为空")
		return
	}
	if rule.RuleType == "" {
		response.BadRequest(c, "rule_type 不能为空")
		return
	}

	if err := h.service.CreateRule(&rule); err != nil {
		response.InternalError(c, "创建规则失败: "+err.Error())
		return
	}

	response.Created(c, rule)
}

// UpdateRule 更新规则
func (h *ClientFilterHandler) UpdateRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}

	var rule model.ClientFilterRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		response.BadRequest(c, "无效的数据")
		return
	}

	rule.ID = uint(id)
	if err := h.service.UpdateRule(&rule); err != nil {
		response.InternalError(c, "更新规则失败: "+err.Error())
		return
	}

	response.Success(c, rule)
}

// DeleteRule 删除规则
func (h *ClientFilterHandler) DeleteRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}

	if err := h.service.DeleteRule(uint(id)); err != nil {
		response.InternalError(c, "删除规则失败: "+err.Error())
		return
	}

	response.Success(c, nil)
}

// ToggleRule 切换规则启用状态
func (h *ClientFilterHandler) ToggleRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的 ID")
		return
	}

	rule, err := h.service.GetRule(uint(id))
	if err != nil {
		response.NotFound(c, "规则不存在")
		return
	}

	rule.Enabled = !rule.Enabled
	if err := h.service.UpdateRule(rule); err != nil {
		response.InternalError(c, "更新规则失败: "+err.Error())
		return
	}

	response.Success(c, rule)
}

// ==================== 测试 ====================

// TestValidation 测试验证
func (h *ClientFilterHandler) TestValidation(c *gin.Context) {
	var reqCtx service.RequestContext
	if err := c.ShouldBindJSON(&reqCtx); err != nil {
		response.BadRequest(c, "无效的请求数据")
		return
	}

	result := h.service.ValidateRequest(&reqCtx)
	response.Success(c, result)
}

// ReloadCache 重新加载缓存
func (h *ClientFilterHandler) ReloadCache(c *gin.Context) {
	if err := h.service.ReloadCache(); err != nil {
		response.InternalError(c, "重新加载缓存失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "缓存已重新加载"})
}
