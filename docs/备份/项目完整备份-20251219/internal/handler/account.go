package handler

import (
	"context"
	"strconv"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/internal/service"
	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

type AccountHandler struct {
	service      *service.AccountService
	usageService *service.UsageService
	cacheService *service.CacheService
}

func NewAccountHandler() *AccountHandler {
	return &AccountHandler{
		service:      service.NewAccountService(),
		usageService: service.NewUsageService(),
		cacheService: service.NewCacheService(),
	}
}

// Account endpoints

func (h *AccountHandler) Create(c *gin.Context) {
	var req service.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	account, err := h.service.Create(&req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Created(c, account)
}

func (h *AccountHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid account id")
		return
	}

	account, err := h.service.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "account not found")
		return
	}

	response.Success(c, account)
}

func (h *AccountHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid account id")
		return
	}

	var req service.UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	account, err := h.service.Update(uint(id), &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, account)
}

func (h *AccountHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid account id")
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *AccountHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	platform := c.Query("platform")
	status := c.Query("status")

	accounts, total, err := h.service.List(page, pageSize, platform, status)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// 获取所有账户的今日用量
	accountIDs := make([]uint, len(accounts))
	for i, acc := range accounts {
		accountIDs[i] = acc.ID
	}

	logRepo := repository.NewRequestLogRepository()
	usageMap, _ := logRepo.GetAccountsTodayUsage(accountIDs)

	// 获取账户总费用：先查 Redis，没有数据再从 MySQL 查
	ctx := context.Background()
	redisCostMap, _ := h.usageService.GetAccountsCost(ctx, accountIDs)

	// 检查哪些账户在 Redis 中没有数据
	var missingIDs []uint
	for _, id := range accountIDs {
		if _, ok := redisCostMap[id]; !ok {
			missingIDs = append(missingIDs, id)
		}
	}

	// 对于 Redis 中没有的，从 MySQL 查询
	var mysqlCostMap map[uint]float64
	if len(missingIDs) > 0 {
		mysqlCostMap, _ = logRepo.GetAccountsTotalCost(missingIDs)
	}

	// 构建带用量信息的响应
	type AccountWithUsage struct {
		model.Account
		TodayTokens        int64   `json:"today_tokens"`
		TodayCount         int64   `json:"today_count"`
		TotalCost          float64 `json:"total_cost"`
		CurrentConcurrency int64   `json:"current_concurrency"`
	}

	items := make([]AccountWithUsage, len(accounts))
	for i, acc := range accounts {
		items[i] = AccountWithUsage{
			Account: acc,
		}
		if usage, ok := usageMap[acc.ID]; ok {
			items[i].TodayTokens = usage.TodayTokens
			items[i].TodayCount = usage.TodayCount
		}
		// 优先使用 Redis 数据，没有再用 MySQL 数据
		if cost, ok := redisCostMap[acc.ID]; ok {
			items[i].TotalCost = cost
		} else if cost, ok := mysqlCostMap[acc.ID]; ok {
			items[i].TotalCost = cost
		}
		// 获取当前并发数
		if concurrent, err := h.cacheService.GetAccountConcurrency(ctx, acc.ID); err == nil {
			items[i].CurrentConcurrency = concurrent
		}
	}

	response.Success(c, gin.H{
		"items": items,
		"total": total,
		"page":  page,
	})
}

func (h *AccountHandler) UpdateStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid account id")
		return
	}

	var req struct {
		Status    string `json:"status" binding:"required"`
		LastError string `json:"last_error"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.service.UpdateStatus(uint(id), req.Status, req.LastError); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *AccountHandler) GetTypes(c *gin.Context) {
	types := []gin.H{
		{"value": model.AccountTypeClaudeOfficial, "label": "Claude Official", "platform": "claude"},
		{"value": model.AccountTypeClaudeConsole, "label": "Claude Console", "platform": "claude"},
		{"value": model.AccountTypeBedrock, "label": "AWS Bedrock", "platform": "claude"},
		{"value": model.AccountTypeCCR, "label": "Claude CCR", "platform": "claude"},
		{"value": model.AccountTypeOpenAI, "label": "OpenAI", "platform": "openai"},
		{"value": model.AccountTypeOpenAIResponses, "label": "OpenAI Responses", "platform": "openai"},
		{"value": model.AccountTypeAzureOpenAI, "label": "Azure OpenAI", "platform": "openai"},
		{"value": model.AccountTypeGemini, "label": "Gemini OAuth", "platform": "gemini"},
		{"value": model.AccountTypeGeminiAPI, "label": "Gemini API", "platform": "gemini"},
		{"value": model.AccountTypeDroid, "label": "Droid", "platform": "other"},
	}
	response.Success(c, types)
}

// AccountGroup endpoints

func (h *AccountHandler) CreateGroup(c *gin.Context) {
	var req service.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	group, err := h.service.CreateGroup(&req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Created(c, group)
}

func (h *AccountHandler) GetGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid group id")
		return
	}

	group, err := h.service.GetGroupByID(uint(id))
	if err != nil {
		response.NotFound(c, "group not found")
		return
	}

	response.Success(c, group)
}

func (h *AccountHandler) UpdateGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid group id")
		return
	}

	var req service.UpdateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	group, err := h.service.UpdateGroup(uint(id), &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, group)
}

func (h *AccountHandler) DeleteGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid group id")
		return
	}

	if err := h.service.DeleteGroup(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *AccountHandler) ListGroups(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	groups, total, err := h.service.ListGroups(page, pageSize)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"items": groups,
		"total": total,
		"page":  page,
	})
}

func (h *AccountHandler) GetAllGroups(c *gin.Context) {
	groups, err := h.service.GetAllGroups()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, groups)
}

func (h *AccountHandler) AddAccountToGroup(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid group id")
		return
	}

	var req struct {
		AccountID uint `json:"account_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.service.AddAccountToGroup(uint(groupID), req.AccountID); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

func (h *AccountHandler) RemoveAccountFromGroup(c *gin.Context) {
	groupID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid group id")
		return
	}

	accountID, err := strconv.ParseUint(c.Param("accountId"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid account id")
		return
	}

	if err := h.service.RemoveAccountFromGroup(uint(groupID), uint(accountID)); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}
