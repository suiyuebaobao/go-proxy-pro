package response

import (
	"net/http"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/service"

	"github.com/gin-gonic/gin"
)

// CustomError 返回自定义错误消息
// 根据 errorType 查找自定义消息配置，如果启用则返回自定义消息，否则返回 originalError
// code: HTTP 状态码
// errorType: 错误类型标识（如 model.ErrorTypeAuthFailed）
// originalError: 原始错误消息
func CustomError(c *gin.Context, code int, errorType, originalError string) {
	errorMsgService := service.GetErrorMessageService()
	customMessage, shouldLog := errorMsgService.GetCustomMessage(errorType, originalError)

	if shouldLog {
		// 使用自定义消息，需要记录原始错误
		ErrorWithLog(c, code, customMessage, originalError, errorType)
	} else {
		// 未启用自定义消息，直接返回原始错误
		Error(c, code, originalError)
	}
}

// CustomErrorAbort 返回自定义错误消息并中断请求
func CustomErrorAbort(c *gin.Context, code int, errorType, originalError string) {
	CustomError(c, code, errorType, originalError)
	c.Abort()
}

// ========== 便捷方法 ==========

// CustomBadRequest 400 错误
func CustomBadRequest(c *gin.Context, originalError string) {
	CustomError(c, http.StatusBadRequest, model.ErrorTypeBadRequest, originalError)
}

// CustomUnauthorized 401 错误 - 认证失败
func CustomUnauthorized(c *gin.Context, errorType, originalError string) {
	CustomError(c, http.StatusUnauthorized, errorType, originalError)
}

// CustomForbidden 403 错误 - 权限不足
func CustomForbidden(c *gin.Context, errorType, originalError string) {
	CustomError(c, http.StatusForbidden, errorType, originalError)
}

// CustomTooManyRequests 429 错误 - 请求过多
func CustomTooManyRequests(c *gin.Context, errorType, originalError string) {
	CustomError(c, http.StatusTooManyRequests, errorType, originalError)
}

// CustomInternalError 500 错误 - 服务器内部错误
func CustomInternalError(c *gin.Context, originalError string) {
	CustomError(c, http.StatusInternalServerError, model.ErrorTypeInternalError, originalError)
}

// CustomBadGateway 502 错误 - 上游服务错误
func CustomBadGateway(c *gin.Context, originalError string) {
	CustomError(c, http.StatusBadGateway, model.ErrorTypeUpstreamError, originalError)
}

// CustomServiceUnavailable 503 错误 - 服务不可用
func CustomServiceUnavailable(c *gin.Context, errorType, originalError string) {
	CustomError(c, http.StatusServiceUnavailable, errorType, originalError)
}

// ========== 带中断的便捷方法 ==========

// CustomBadRequestAbort 400 错误并中断
func CustomBadRequestAbort(c *gin.Context, originalError string) {
	CustomErrorAbort(c, http.StatusBadRequest, model.ErrorTypeBadRequest, originalError)
}

// CustomUnauthorizedAbort 401 错误并中断
func CustomUnauthorizedAbort(c *gin.Context, errorType, originalError string) {
	CustomErrorAbort(c, http.StatusUnauthorized, errorType, originalError)
}

// CustomForbiddenAbort 403 错误并中断
func CustomForbiddenAbort(c *gin.Context, errorType, originalError string) {
	CustomErrorAbort(c, http.StatusForbidden, errorType, originalError)
}

// CustomTooManyRequestsAbort 429 错误并中断
func CustomTooManyRequestsAbort(c *gin.Context, errorType, originalError string) {
	CustomErrorAbort(c, http.StatusTooManyRequests, errorType, originalError)
}

// CustomInternalErrorAbort 500 错误并中断
func CustomInternalErrorAbort(c *gin.Context, originalError string) {
	CustomErrorAbort(c, http.StatusInternalServerError, model.ErrorTypeInternalError, originalError)
}

// CustomBadGatewayAbort 502 错误并中断
func CustomBadGatewayAbort(c *gin.Context, originalError string) {
	CustomErrorAbort(c, http.StatusBadGateway, model.ErrorTypeUpstreamError, originalError)
}

// CustomServiceUnavailableAbort 503 错误并中断
func CustomServiceUnavailableAbort(c *gin.Context, errorType, originalError string) {
	CustomErrorAbort(c, http.StatusServiceUnavailable, errorType, originalError)
}
