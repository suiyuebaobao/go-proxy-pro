/*
 * 文件作用：请求日志记录器，异步记录API请求日志到数据库
 * 负责功能：
 *   - 请求日志异步写入
 *   - 日志对象构建
 *   - 单例模式延迟初始化
 * 重要程度：⭐⭐⭐ 一般（日志记录）
 * 依赖模块：model, repository
 */
package handler

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
)

// RequestLogger 请求日志记录器
type RequestLogger struct {
	repo *repository.RequestLogRepository
}

var (
	defaultRequestLogger *RequestLogger
	requestLoggerOnce    sync.Once
)

// getRequestLogger 延迟初始化请求日志记录器
func getRequestLogger() *RequestLogger {
	requestLoggerOnce.Do(func() {
		defaultRequestLogger = &RequestLogger{
			repo: repository.NewRequestLogRepository(),
		}
	})
	return defaultRequestLogger
}

// LogRequest 记录请求
func LogRequest(log *model.RequestLog) {
	go getRequestLogger().repo.Create(log)
}

// BuildRequestLog 构建请求日志
func BuildRequestLog(
	accountID uint,
	platform string,
	modelName string,
	endpoint string,
	method string,
	path string,
	requestIP string,
	userAgent string,
	sessionID string,
) *model.RequestLog {
	return &model.RequestLog{
		AccountID:  accountID,
		Platform:   platform,
		Model:      modelName,
		Endpoint:   endpoint,
		Method:     method,
		Path:       path,
		RequestIP:  requestIP,
		UserAgent:  userAgent,
		SessionID:  sessionID,
		Success:    true,
		StatusCode: 200,
		CreatedAt:  time.Now(),
	}
}

// SetRequestDetails 设置请求详情（请求头和请求体）
func SetRequestDetails(log *model.RequestLog, headers http.Header, body []byte) {
	// 过滤敏感头部
	filteredHeaders := filterSensitiveHeaders(headers)
	if headersJSON, err := json.Marshal(filteredHeaders); err == nil {
		log.RequestHeaders = string(headersJSON)
	}
	// 限制请求体大小（最大 64KB）
	if len(body) > 65536 {
		log.RequestBody = string(body[:65536]) + "...[truncated]"
	} else {
		log.RequestBody = string(body)
	}
}

// SetResponseDetails 设置响应详情（响应头和响应体）
func SetResponseDetails(log *model.RequestLog, headers http.Header, body []byte, upstreamStatusCode int) {
	if headersJSON, err := json.Marshal(headers); err == nil {
		log.ResponseHeaders = string(headersJSON)
	}
	// 限制响应体大小（最大 64KB）
	if len(body) > 65536 {
		log.ResponseBody = string(body[:65536]) + "...[truncated]"
	} else {
		log.ResponseBody = string(body)
	}
	log.UpstreamStatusCode = upstreamStatusCode
}

// SetUpstreamError 设置上游错误信息
func SetUpstreamError(log *model.RequestLog, upstreamStatusCode int, errMsg string) {
	log.UpstreamStatusCode = upstreamStatusCode
	log.UpstreamError = errMsg
}

// filterSensitiveHeaders 过滤敏感头部
func filterSensitiveHeaders(headers http.Header) map[string]string {
	result := make(map[string]string)
	sensitiveKeys := map[string]bool{
		"Authorization":   true,
		"X-Api-Key":       true,
		"Cookie":          true,
		"Set-Cookie":      true,
		"X-Session-Key":   true,
		"X-Access-Token":  true,
		"X-Refresh-Token": true,
	}
	for key, values := range headers {
		if sensitiveKeys[key] {
			result[key] = "[REDACTED]"
		} else if len(values) > 0 {
			result[key] = values[0]
		}
	}
	return result
}

// CompleteLog 完成日志记录
func CompleteLog(log *model.RequestLog, success bool, statusCode int, errMsg string, inputTokens, outputTokens int, duration time.Duration) {
	log.Success = success
	log.StatusCode = statusCode
	log.Error = errMsg
	log.InputTokens = inputTokens
	log.OutputTokens = outputTokens
	log.Duration = duration.Milliseconds()
	LogRequest(log)
}

// CompleteLogFull 完成日志记录（包含完整信息）
func CompleteLogFull(log *model.RequestLog, success bool, statusCode int, errMsg string,
	inputTokens, outputTokens, cacheCreationTokens, cacheReadTokens int,
	inputCost, outputCost, cacheCreateCost, cacheReadCost float64,
	duration time.Duration) {
	log.Success = success
	log.StatusCode = statusCode
	log.Error = errMsg
	log.InputTokens = inputTokens
	log.OutputTokens = outputTokens
	log.CacheCreationInputTokens = cacheCreationTokens
	log.CacheReadInputTokens = cacheReadTokens
	log.TotalTokens = inputTokens + outputTokens + cacheCreationTokens + cacheReadTokens
	log.InputCost = inputCost
	log.OutputCost = outputCost
	log.CacheCreateCost = cacheCreateCost
	log.CacheReadCost = cacheReadCost
	log.TotalCost = inputCost + outputCost + cacheCreateCost + cacheReadCost
	log.Duration = duration.Milliseconds()
	LogRequest(log)
}
