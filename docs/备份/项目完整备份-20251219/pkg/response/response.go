package response

import (
	"net/http"
	"sync"

	"go-aiproxy/pkg/logger"

	"github.com/gin-gonic/gin"
)

var (
	errorLog     *logger.Logger
	errorLogOnce sync.Once
)

// getErrorLog 懒加载获取错误日志
func getErrorLog() *logger.Logger {
	errorLogOnce.Do(func() {
		errorLog = logger.GetLogger("error_response")
	})
	return errorLog
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:    0,
		Message: "created",
		Data:    data,
	})
}

func Error(c *gin.Context, code int, message string) {
	c.JSON(code, Response{
		Code:    code,
		Message: message,
	})
}

func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message)
}

func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message)
}

func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message)
}

func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, message)
}

func TooManyRequests(c *gin.Context, message string) {
	Error(c, http.StatusTooManyRequests, message)
}

// ErrorWithLog 返回自定义错误消息并记录原始错误
// customMessage: 返回给用户的消息
// originalError: 记录到日志的原始错误（不返回给用户）
// errorType: 错误类型标识（用于日志分类）
func ErrorWithLog(c *gin.Context, code int, customMessage, originalError, errorType string) {
	// 记录原始错误到日志
	if originalError != "" && originalError != customMessage {
		requestID := c.GetString("request_id")
		clientIP := c.ClientIP()
		path := c.Request.URL.Path

		log := getErrorLog()
		if log != nil {
			log.Warn("[%s] %s | IP: %s | Path: %s | Type: %s | Original: %s | Return: %s",
				getCodeLabel(code), requestID, clientIP, path, errorType, originalError, customMessage)
		}
	}

	// 返回自定义消息给用户
	c.JSON(code, Response{
		Code:    code,
		Message: customMessage,
	})
}

// BadRequestWithLog 400 错误，记录原始错误
func BadRequestWithLog(c *gin.Context, customMessage, originalError, errorType string) {
	ErrorWithLog(c, http.StatusBadRequest, customMessage, originalError, errorType)
}

// UnauthorizedWithLog 401 错误，记录原始错误
func UnauthorizedWithLog(c *gin.Context, customMessage, originalError, errorType string) {
	ErrorWithLog(c, http.StatusUnauthorized, customMessage, originalError, errorType)
}

// ForbiddenWithLog 403 错误，记录原始错误
func ForbiddenWithLog(c *gin.Context, customMessage, originalError, errorType string) {
	ErrorWithLog(c, http.StatusForbidden, customMessage, originalError, errorType)
}

// TooManyRequestsWithLog 429 错误，记录原始错误
func TooManyRequestsWithLog(c *gin.Context, customMessage, originalError, errorType string) {
	ErrorWithLog(c, http.StatusTooManyRequests, customMessage, originalError, errorType)
}

// InternalErrorWithLog 500 错误，记录原始错误
func InternalErrorWithLog(c *gin.Context, customMessage, originalError, errorType string) {
	ErrorWithLog(c, http.StatusInternalServerError, customMessage, originalError, errorType)
}

// BadGatewayWithLog 502 错误，记录原始错误
func BadGatewayWithLog(c *gin.Context, customMessage, originalError, errorType string) {
	ErrorWithLog(c, http.StatusBadGateway, customMessage, originalError, errorType)
}

// ServiceUnavailableWithLog 503 错误，记录原始错误
func ServiceUnavailableWithLog(c *gin.Context, customMessage, originalError, errorType string) {
	ErrorWithLog(c, http.StatusServiceUnavailable, customMessage, originalError, errorType)
}

// getCodeLabel 获取状态码标签
func getCodeLabel(code int) string {
	switch code {
	case 400:
		return "BAD_REQUEST"
	case 401:
		return "UNAUTHORIZED"
	case 403:
		return "FORBIDDEN"
	case 429:
		return "TOO_MANY_REQUESTS"
	case 500:
		return "INTERNAL_ERROR"
	case 502:
		return "BAD_GATEWAY"
	case 503:
		return "SERVICE_UNAVAILABLE"
	default:
		return "ERROR"
	}
}

type PaginationResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type PaginationData struct {
	Items interface{} `json:"items"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"page_size"`
}

func SuccessWithPagination(c *gin.Context, items interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: PaginationData{
			Items: items,
			Total: total,
			Page:  page,
			Size:  pageSize,
		},
	})
}
