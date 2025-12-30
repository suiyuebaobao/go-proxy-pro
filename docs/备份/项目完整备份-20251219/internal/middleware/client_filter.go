package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/service"
	"go-aiproxy/pkg/logger"
	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

// ClientFilter 客户端过滤中间件
func ClientFilter() gin.HandlerFunc {
	filterService := service.GetClientFilterService()
	log := logger.GetLogger("client_filter")

	return func(c *gin.Context) {
		// 检查过滤是否启用
		if !filterService.IsFilterEnabled() {
			c.Next()
			return
		}

		// 构建请求上下文
		reqCtx := buildRequestContext(c)

		// 执行验证
		result := filterService.ValidateRequest(reqCtx)

		// 记录验证结果
		if result.Allowed {
			if result.ClientType != "" {
				log.Debug("客户端验证通过 | IP: %s | 类型: %s (%s) | 规则通过: %d | 警告: %d",
					c.ClientIP(), result.ClientType, result.ClientName,
					len(result.MatchedRules), len(result.Warnings))
			}
		} else {
			log.Warn("客户端验证失败 | IP: %s | 类型: %s | 原因: %s | UA: %s",
				c.ClientIP(), result.ClientType, result.Details["reason"], reqCtx.UserAgent)
		}

		// 将验证结果存储到 Context
		c.Set("client_filter_result", result)
		c.Set("client_type", result.ClientType)
		c.Set("client_name", result.ClientName)

		// 如果不允许，拒绝请求
		if !result.Allowed {
			response.CustomForbiddenAbort(c, model.ErrorTypeClientNotAllowed, buildErrorMessage(result))
			return
		}

		c.Next()
	}
}

// buildRequestContext 构建请求上下文
func buildRequestContext(c *gin.Context) *service.RequestContext {
	ctx := &service.RequestContext{
		UserAgent: c.GetHeader("User-Agent"),
		Path:      c.Request.URL.Path,
		Headers:   make(map[string]string),
		Body:      make(map[string]interface{}),
	}

	// 收集所有请求头
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			ctx.Headers[strings.ToLower(key)] = values[0]
		}
	}

	// 解析请求体（仅 POST/PUT/PATCH）
	if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err == nil && len(bodyBytes) > 0 {
			// 重置 Body 以便后续处理
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			// 解析 JSON
			var bodyMap map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &bodyMap); err == nil {
				ctx.Body = bodyMap
			}
		}
	}

	return ctx
}

// buildErrorMessage 构建错误消息
func buildErrorMessage(result *service.ValidationResult) string {
	msg := "客户端验证失败"

	if reason, ok := result.Details["reason"]; ok {
		msg = reason
	}

	if len(result.FailedRules) > 0 {
		failedNames := make([]string, 0, len(result.FailedRules))
		for _, rule := range result.FailedRules {
			if rule.Required {
				failedNames = append(failedNames, rule.RuleName)
			}
		}
		if len(failedNames) > 0 {
			msg += " (" + strings.Join(failedNames, ", ") + ")"
		}
	}

	return msg
}

// GetClientFilterResult 从 Context 获取客户端过滤结果
func GetClientFilterResult(c *gin.Context) *service.ValidationResult {
	if result, exists := c.Get("client_filter_result"); exists {
		return result.(*service.ValidationResult)
	}
	return nil
}

// GetClientType 从 Context 获取客户端类型
func GetClientType(c *gin.Context) string {
	if clientType, exists := c.Get("client_type"); exists {
		return clientType.(string)
	}
	return ""
}

// ClientFilterOptional 可选的客户端过滤中间件（仅记录，不拦截）
func ClientFilterOptional() gin.HandlerFunc {
	filterService := service.GetClientFilterService()
	log := logger.GetLogger("client_filter")

	return func(c *gin.Context) {
		// 检查过滤是否启用
		if !filterService.IsFilterEnabled() {
			c.Next()
			return
		}

		// 构建请求上下文
		reqCtx := buildRequestContext(c)

		// 执行验证
		result := filterService.ValidateRequest(reqCtx)

		// 仅记录，不拦截
		if !result.Allowed {
			log.Info("客户端验证警告（非阻塞）| IP: %s | 类型: %s | 原因: %s",
				c.ClientIP(), result.ClientType, result.Details["reason"])
		}

		// 将验证结果存储到 Context
		c.Set("client_filter_result", result)
		c.Set("client_type", result.ClientType)
		c.Set("client_name", result.ClientName)

		c.Next()
	}
}

// CheckAllowedClients 检查 API Key 的客户端限制
// 在 API Key 验证后调用，检查当前客户端是否在 API Key 允许的客户端列表中
func CheckAllowedClients() gin.HandlerFunc {
	log := logger.GetLogger("client_filter")

	return func(c *gin.Context) {
		// 获取 API Key 信息
		apiKey := GetAPIKey(c)
		if apiKey == nil || apiKey.AllowedClients == "" {
			c.Next()
			return
		}

		// 获取客户端类型
		clientType := GetClientType(c)
		if clientType == "" {
			c.Next()
			return
		}

		// 检查客户端是否被允许
		allowedClients := strings.Split(apiKey.AllowedClients, ",")
		allowed := false
		for _, ac := range allowedClients {
			if strings.TrimSpace(ac) == clientType {
				allowed = true
				break
			}
		}

		if !allowed {
			log.Warn("API Key 客户端限制 | KeyID: %d | 客户端: %s | 允许: %s",
				apiKey.ID, clientType, apiKey.AllowedClients)
			response.CustomForbiddenAbort(c, model.ErrorTypeClientNotAllowed,
				"此 API Key 不允许 "+clientType+" 客户端访问")
			return
		}

		c.Next()
	}
}
