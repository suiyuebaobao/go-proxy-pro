package handler

import (
	"strconv"
	"time"

	"go-aiproxy/internal/config"
	"go-aiproxy/internal/service"
	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

// CacheHandler 缓存管理处理器
type CacheHandler struct {
	cacheService *service.CacheService
}

// NewCacheHandler 创建缓存管理处理器
func NewCacheHandler() *CacheHandler {
	return &CacheHandler{
		cacheService: service.NewCacheService(),
	}
}

// GetStats 获取缓存统计
func (h *CacheHandler) GetStats(c *gin.Context) {
	ctx := c.Request.Context()

	stats, err := h.cacheService.GetCacheStats(ctx)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, stats)
}

// ListSessions 列出所有会话绑定
func (h *CacheHandler) ListSessions(c *gin.Context) {
	ctx := c.Request.Context()

	offset, _ := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "20"), 10, 64)

	if limit > 100 {
		limit = 100
	}

	sessions, total, err := h.cacheService.ListAllSessions(ctx, offset, limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"sessions": sessions,
		"total":    total,
		"offset":   offset,
		"limit":    limit,
	})
}

// ListAccountsCache 列出所有有缓存的账号（聚合视图）
func (h *CacheHandler) ListAccountsCache(c *gin.Context) {
	ctx := c.Request.Context()

	accounts, err := h.cacheService.ListAccountsWithCache(ctx)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"accounts": accounts,
		"total":    len(accounts),
	})
}

// ListUsersCache 列出所有有缓存的用户（聚合视图）
func (h *CacheHandler) ListUsersCache(c *gin.Context) {
	ctx := c.Request.Context()

	users, err := h.cacheService.ListUsersWithCache(ctx)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"users": users,
		"total": len(users),
	})
}

// RemoveSession 移除会话绑定
func (h *CacheHandler) RemoveSession(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		response.BadRequest(c, "session_id is required")
		return
	}

	ctx := c.Request.Context()

	err := h.cacheService.RemoveSessionBinding(ctx, sessionID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "session removed"})
}

// ClearAccountSessions 清除账户的所有会话
func (h *CacheHandler) ClearAccountSessions(c *gin.Context) {
	accountIDStr := c.Param("id")
	accountID, err := strconv.ParseUint(accountIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid account_id")
		return
	}

	ctx := c.Request.Context()

	count, err := h.cacheService.ClearAccountSessions(ctx, uint(accountID))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"deleted_count": count,
	})
}

// ListUnavailableAccounts 列出所有不可用账户
func (h *CacheHandler) ListUnavailableAccounts(c *gin.Context) {
	ctx := c.Request.Context()

	accounts, err := h.cacheService.GetAllUnavailableAccounts(ctx)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"accounts": accounts,
		"total":    len(accounts),
	})
}

// MarkAccountUnavailable 标记账户不可用
func (h *CacheHandler) MarkAccountUnavailable(c *gin.Context) {
	accountIDStr := c.Param("id")
	accountID, err := strconv.ParseUint(accountIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid account_id")
		return
	}

	var req struct {
		Reason string `json:"reason"`
		TTL    int    `json:"ttl"` // 秒
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Reason = "manual"
	}

	ctx := c.Request.Context()

	ttl := service.DefaultTempUnavailableTTL()
	if req.TTL > 0 {
		ttl = time.Duration(req.TTL) * time.Second
	}

	err = h.cacheService.MarkAccountUnavailable(ctx, uint(accountID), req.Reason, ttl)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "account marked unavailable"})
}

// ClearAccountUnavailable 清除账户不可用标记
func (h *CacheHandler) ClearAccountUnavailable(c *gin.Context) {
	accountIDStr := c.Param("id")
	accountID, err := strconv.ParseUint(accountIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid account_id")
		return
	}

	ctx := c.Request.Context()

	err = h.cacheService.ClearAccountUnavailable(ctx, uint(accountID))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "unavailable mark cleared"})
}

// GetAccountConcurrency 获取账户并发信息
func (h *CacheHandler) GetAccountConcurrency(c *gin.Context) {
	accountIDStr := c.Param("id")
	accountID, err := strconv.ParseUint(accountIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid account_id")
		return
	}

	ctx := c.Request.Context()

	current, err := h.cacheService.GetAccountConcurrency(ctx, uint(accountID))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	limit := h.cacheService.GetAccountConcurrencyLimit(uint(accountID))

	response.Success(c, gin.H{
		"account_id": accountID,
		"current":    current,
		"limit":      limit,
	})
}

// SetAccountConcurrencyLimit 设置账户并发限制
func (h *CacheHandler) SetAccountConcurrencyLimit(c *gin.Context) {
	accountIDStr := c.Param("id")
	accountID, err := strconv.ParseUint(accountIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid account_id")
		return
	}

	var req struct {
		Limit int `json:"limit" binding:"required,min=1,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	h.cacheService.SetAccountConcurrencyLimit(uint(accountID), req.Limit)

	response.Success(c, gin.H{
		"account_id": accountID,
		"limit":      req.Limit,
	})
}

// ResetAccountConcurrency 重置账户并发计数
func (h *CacheHandler) ResetAccountConcurrency(c *gin.Context) {
	accountIDStr := c.Param("id")
	accountID, err := strconv.ParseUint(accountIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid account_id")
		return
	}

	ctx := c.Request.Context()

	err = h.cacheService.ResetAccountConcurrency(ctx, uint(accountID))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "concurrency reset"})
}

// ClearCache 清理缓存
func (h *CacheHandler) ClearCache(c *gin.Context) {
	var req struct {
		Type string `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	ctx := c.Request.Context()

	result, err := h.cacheService.ClearCache(ctx, service.ClearCacheType(req.Type))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, result)
}

// ClearUserCache 清理用户缓存
func (h *CacheHandler) ClearUserCache(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user_id")
		return
	}

	ctx := c.Request.Context()

	result, err := h.cacheService.ClearUserCache(ctx, uint(userID))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, result)
}

// GetUserConcurrency 获取用户并发信息
func (h *CacheHandler) GetUserConcurrency(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user_id")
		return
	}

	ctx := c.Request.Context()

	current, err := h.cacheService.GetUserConcurrency(ctx, uint(userID))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"user_id": userID,
		"current": current,
	})
}

// ResetUserConcurrency 重置用户并发计数
func (h *CacheHandler) ResetUserConcurrency(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user_id")
		return
	}

	ctx := c.Request.Context()

	err = h.cacheService.ResetUserConcurrency(ctx, uint(userID))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"message": "user concurrency reset"})
}

// ClearAPIKeyCache 清理 API Key 缓存
func (h *CacheHandler) ClearAPIKeyCache(c *gin.Context) {
	keyIDStr := c.Param("id")
	keyID, err := strconv.ParseUint(keyIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid api_key_id")
		return
	}

	ctx := c.Request.Context()

	result, err := h.cacheService.ClearAPIKeyCache(ctx, uint(keyID))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, result)
}

// GetCacheConfig 获取缓存配置
func (h *CacheHandler) GetCacheConfig(c *gin.Context) {
	cfg := config.Cfg.Cache
	response.Success(c, gin.H{
		"session_ttl":             cfg.GetSessionTTL(),
		"session_renewal_ttl":     cfg.GetSessionRenewalTTL(),
		"unavailable_ttl":         cfg.GetUnavailableTTL(),
		"concurrency_ttl":         cfg.GetConcurrencyTTL(),
		"default_concurrency_max": cfg.GetDefaultConcurrencyMax(),
	})
}

// UpdateCacheConfig 更新缓存配置（运行时修改，重启后恢复配置文件设置）
func (h *CacheHandler) UpdateCacheConfig(c *gin.Context) {
	var req struct {
		SessionTTL            *int `json:"session_ttl"`
		SessionRenewalTTL     *int `json:"session_renewal_ttl"`
		UnavailableTTL        *int `json:"unavailable_ttl"`
		ConcurrencyTTL        *int `json:"concurrency_ttl"`
		DefaultConcurrencyMax *int `json:"default_concurrency_max"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	cfg := &config.Cfg.Cache
	if req.SessionTTL != nil && *req.SessionTTL > 0 {
		cfg.SessionTTL = *req.SessionTTL
	}
	if req.SessionRenewalTTL != nil && *req.SessionRenewalTTL > 0 {
		cfg.SessionRenewalTTL = *req.SessionRenewalTTL
	}
	if req.UnavailableTTL != nil && *req.UnavailableTTL > 0 {
		cfg.UnavailableTTL = *req.UnavailableTTL
	}
	if req.ConcurrencyTTL != nil && *req.ConcurrencyTTL > 0 {
		cfg.ConcurrencyTTL = *req.ConcurrencyTTL
	}
	if req.DefaultConcurrencyMax != nil && *req.DefaultConcurrencyMax > 0 {
		cfg.DefaultConcurrencyMax = *req.DefaultConcurrencyMax
	}

	response.Success(c, gin.H{
		"message":                 "config updated (runtime only)",
		"session_ttl":             cfg.GetSessionTTL(),
		"session_renewal_ttl":     cfg.GetSessionRenewalTTL(),
		"unavailable_ttl":         cfg.GetUnavailableTTL(),
		"concurrency_ttl":         cfg.GetConcurrencyTTL(),
		"default_concurrency_max": cfg.GetDefaultConcurrencyMax(),
	})
}
