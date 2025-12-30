/*
 * 文件作用：API Key认证中间件，验证代理请求的API密钥
 * 负责功能：
 *   - API Key 解析（支持多种Header格式）
 *   - API Key 有效性验证
 *   - 用户/API Key 信息注入上下文
 *   - 费率倍率应用
 *   - 请求日志记录
 * 重要程度：⭐⭐⭐⭐⭐ 核心（代理认证核心）
 * 依赖模块：service, repository, model
 */
package middleware

import (
	"strings"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/internal/service"
	"go-aiproxy/pkg/logger"
	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth API Key 认证中间件
func APIKeyAuth() gin.HandlerFunc {
	apiKeyService := service.NewAPIKeyService()
	userRepo := repository.NewUserRepository()
	configService := service.GetConfigService()
	log := logger.GetLogger("auth")

	return func(c *gin.Context) {
		// 从 Header 获取 API Key（支持多种格式）
		apiKey := c.GetHeader("Authorization")
		if apiKey == "" {
			apiKey = c.GetHeader("X-API-Key")
		}
		if apiKey == "" {
			apiKey = c.GetHeader("x-api-key") // Claude 标准格式
		}

		if apiKey == "" {
			log.Debug("API Key 认证失败 | IP: %s | 原因: 缺少API Key", c.ClientIP())
			response.CustomUnauthorizedAbort(c, model.ErrorTypeAuthFailed, "缺少 API Key，请在 Authorization 或 x-api-key header 中提供")
			return
		}

		// 移除 Bearer 前缀
		if strings.HasPrefix(apiKey, "Bearer ") {
			apiKey = apiKey[7:]
		}

		// 验证 API Key
		key, err := apiKeyService.ValidateKey(apiKey)
		if err != nil {
			log.Debug("API Key 认证失败 | IP: %s | Key: %s... | 原因: %v", c.ClientIP(), maskAPIKey(apiKey), err)
			// 根据错误内容确定错误类型
			errorType := getAPIKeyErrorType(err.Error())
			response.CustomUnauthorizedAbort(c, errorType, err.Error())
			return
		}

		log.Debug("API Key 认证成功 | IP: %s | KeyID: %d | UserID: %d", c.ClientIP(), key.ID, key.UserID)

		// 将 API Key 信息存储到 Context 中
		c.Set("api_key", key)
		c.Set("api_key_id", key.ID)
		c.Set("api_key_user_id", key.UserID)
		c.Set("api_key_allowed_platforms", key.AllowedPlatforms)
		c.Set("api_key_allowed_models", key.AllowedModels)
		c.Set("api_key_rate_limit", key.RateLimit)

		// 添加套餐信息（用于扣费）
		if key.UserPackageID != nil {
			c.Set("api_key_package_id", *key.UserPackageID)
			c.Set("api_key_billing_type", key.BillingType)
		}

		// 计算有效倍率
		// 优先级：全局倍率 → 用户倍率
		// 规则：如果全局倍率=1（默认），则读取用户倍率
		globalRate := configService.GetGlobalPriceRate()
		priceRate := globalRate
		user, err := userRepo.GetByID(key.UserID)
		if err == nil && user != nil {
			c.Set("user", user)

			// 全局倍率是默认值1，才看用户倍率
			if priceRate == 1.0 && user.PriceRate != 1.0 {
				priceRate = user.PriceRate
			}
		}
		c.Set("api_key_price_rate", priceRate)

		// 调试日志：每次请求都输出倍率信息
		log.Info("API Key 认证 | KeyID: %d | GlobalRate: %.2f | FinalRate: %.2f | Path: %s",
			key.ID, globalRate, priceRate, c.Request.URL.Path)

		c.Next()
	}
}

// maskAPIKey 遮蔽API Key用于日志
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:8]
}

// GetAPIKey 从 Context 获取 API Key 信息
func GetAPIKey(c *gin.Context) *model.APIKey {
	if key, exists := c.Get("api_key"); exists {
		return key.(*model.APIKey)
	}
	return nil
}

// CheckPlatformAccess 检查平台访问权限
func CheckPlatformAccess(c *gin.Context, platform string) bool {
	allowed, exists := c.Get("api_key_allowed_platforms")
	if !exists {
		return true // 没有设置 API Key，允许访问
	}

	allowedStr := allowed.(string)
	if allowedStr == "all" || allowedStr == "" {
		return true
	}

	// 检查是否在允许的平台列表中
	platforms := strings.Split(allowedStr, ",")
	for _, p := range platforms {
		if strings.TrimSpace(p) == platform {
			return true
		}
	}

	return false
}

// CheckModelAccess 检查模型访问权限
func CheckModelAccess(c *gin.Context, modelName string) bool {
	allowed, exists := c.Get("api_key_allowed_models")
	if !exists {
		return true
	}

	allowedStr := allowed.(string)
	if allowedStr == "" {
		return true // 空字符串表示允许所有模型
	}

	// 检查是否在允许的模型列表中
	models := strings.Split(allowedStr, ",")
	for _, m := range models {
		if strings.TrimSpace(m) == modelName {
			return true
		}
	}

	return false
}

// getAPIKeyErrorType 根据错误信息判断错误类型
func getAPIKeyErrorType(errMsg string) string {
	switch {
	case strings.Contains(errMsg, "禁用"):
		return model.ErrorTypeKeyDisabled
	case strings.Contains(errMsg, "过期"):
		return model.ErrorTypeKeyExpired
	case strings.Contains(errMsg, "无效"):
		return model.ErrorTypeKeyInvalid
	default:
		return model.ErrorTypeAuthFailed
	}
}
