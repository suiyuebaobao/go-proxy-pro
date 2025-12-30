/*
 * 文件作用：用户管理处理器，处理用户认证和用户管理
 * 负责功能：
 *   - 用户登录/注册
 *   - 用户列表查询（含余额统计）
 *   - 用户创建/更新/删除
 *   - 密码修改
 *   - JWT Token 生成
 * 重要程度：⭐⭐⭐⭐ 重要（用户管理核心）
 * 依赖模块：service
 */
package handler

import (
	"strconv"

	"go-aiproxy/internal/service"
	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler() *UserHandler {
	return &UserHandler{
		service: service.NewUserService(),
	}
}

func (h *UserHandler) Login(c *gin.Context) {
	configSvc := service.GetConfigService()
	clientIP := c.ClientIP()

	// 检查登录频率限制
	if configSvc.GetLoginRateLimitEnabled() {
		limiter := service.GetLoginRateLimiter()
		allowed, waitSeconds := limiter.Check(
			clientIP,
			configSvc.GetLoginRateLimitCount(),
			configSvc.GetLoginRateLimitWindow(),
		)
		if !allowed {
			response.TooManyRequests(c, service.GetRateLimitError(waitSeconds))
			return
		}
	}

	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 验证验证码（如果启用）
	if configSvc.GetCaptchaEnabled() {
		if req.CaptchaID == "" || req.Captcha == "" {
			response.BadRequest(c, "请输入验证码")
			return
		}
		captchaService := service.GetCaptchaService()
		if !captchaService.Verify(req.CaptchaID, req.Captcha) {
			response.BadRequest(c, "验证码错误")
			return
		}
	}

	result, err := h.service.Login(&req)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	// 登录成功，重置该 IP 的登录频率限制
	if configSvc.GetLoginRateLimitEnabled() {
		service.GetLoginRateLimiter().Reset(clientIP)
	}

	response.Success(c, result)
}

func (h *UserHandler) Register(c *gin.Context) {
	configSvc := service.GetConfigService()

	var req service.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 验证验证码（如果启用）
	if configSvc.GetCaptchaEnabled() {
		if req.CaptchaID == "" || req.Captcha == "" {
			response.BadRequest(c, "请输入验证码")
			return
		}
		captchaService := service.GetCaptchaService()
		if !captchaService.Verify(req.CaptchaID, req.Captcha) {
			response.BadRequest(c, "验证码错误")
			return
		}
	}

	user, err := h.service.Register(&req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Created(c, user)
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	user, err := h.service.GetByID(userID)
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}

	response.Success(c, user)
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 普通用户不能修改 role 和 status
	req.Role = ""
	req.Status = ""

	user, err := h.service.Update(userID, &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, user)
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req service.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.service.ChangePassword(userID, &req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// Admin endpoints

func (h *UserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	users, total, err := h.service.ListWithKeyBalances(page, pageSize)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"items": users,
		"total": total,
		"page":  page,
	})
}

// Create 管理员创建用户
func (h *UserHandler) Create(c *gin.Context) {
	var req service.AdminCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, err := h.service.AdminCreateUser(&req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Created(c, user)
}

func (h *UserHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	user, err := h.service.GetByID(uint(id))
	if err != nil {
		response.NotFound(c, "user not found")
		return
	}

	response.Success(c, user)
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req service.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, err := h.service.Update(uint(id), &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, user)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, nil)
}

// BatchUpdatePriceRate 批量更新用户费率倍率
func (h *UserHandler) BatchUpdatePriceRate(c *gin.Context) {
	var req service.BatchUpdatePriceRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.service.BatchUpdatePriceRate(&req); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"message":       "费率倍率更新成功",
		"affected_count": len(req.UserIDs),
		"price_rate":    req.PriceRate,
	})
}

// UpdateAllPriceRate 更新所有用户费率倍率
func (h *UserHandler) UpdateAllPriceRate(c *gin.Context) {
	var req service.UpdateAllPriceRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.service.UpdateAllPriceRate(req.PriceRate); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"message":    "所有用户费率倍率更新成功",
		"price_rate": req.PriceRate,
	})
}

// ListAllUsers 获取所有用户列表（不分页，用于批量操作）
func (h *UserHandler) ListAllUsers(c *gin.Context) {
	users, err := h.service.ListAll()
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"items": users,
		"total": len(users),
	})
}
