/*
 * 文件作用：验证码处理器，处理图形验证码的生成和状态查询
 * 负责功能：
 *   - 验证码图片生成
 *   - 验证码获取频率限制
 *   - 验证码功能状态查询
 * 重要程度：⭐⭐ 辅助（安全功能）
 * 依赖模块：service
 */
package handler

import (
	"net/http"

	"go-aiproxy/internal/service"
	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

// CaptchaHandler 验证码处理器
type CaptchaHandler struct {
	service *service.CaptchaService
}

// NewCaptchaHandler 创建验证码处理器
func NewCaptchaHandler() *CaptchaHandler {
	return &CaptchaHandler{
		service: service.GetCaptchaService(),
	}
}

// Generate 生成验证码
// @Summary 生成验证码
// @Tags 认证
// @Produce json
// @Success 200 {object} response.Response{data=service.CaptchaResponse}
// @Router /api/auth/captcha [get]
func (h *CaptchaHandler) Generate(c *gin.Context) {
	configSvc := service.GetConfigService()
	clientIP := c.ClientIP()

	// 检查验证码获取频率限制
	limiter := service.GetCaptchaRateLimiter()
	allowed, waitSeconds := limiter.Check(clientIP, configSvc.GetCaptchaRateLimit(), 1) // 每分钟限制
	if !allowed {
		response.TooManyRequests(c, service.GetRateLimitError(waitSeconds))
		return
	}

	// 检查是否启用验证码
	if !configSvc.GetCaptchaEnabled() {
		// 验证码未启用，返回空数据
		response.Success(c, gin.H{
			"captcha_id": "",
			"image":      "",
			"enabled":    false,
		})
		return
	}

	captcha, err := h.service.Generate()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "生成验证码失败")
		return
	}

	response.Success(c, gin.H{
		"captcha_id": captcha.CaptchaID,
		"image":      captcha.Image,
		"enabled":    true,
	})
}

// GetStatus 获取验证码配置状态
// @Summary 获取验证码配置状态
// @Tags 认证
// @Produce json
// @Router /api/auth/captcha/status [get]
func (h *CaptchaHandler) GetStatus(c *gin.Context) {
	configSvc := service.GetConfigService()
	response.Success(c, gin.H{
		"captcha_enabled":          configSvc.GetCaptchaEnabled(),
		"login_rate_limit_enabled": configSvc.GetLoginRateLimitEnabled(),
	})
}
