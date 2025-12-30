/*
 * 文件作用：HTTP请求日志中间件，记录所有HTTP请求的详细信息
 * 负责功能：
 *   - 请求ID生成和传递
 *   - 请求/响应时间记录
 *   - 请求体大小统计
 *   - 敏感信息脱敏（token/password）
 * 重要程度：⭐⭐⭐ 一般（调试和监控）
 * 依赖模块：logger
 */
package middleware

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"strings"
	"time"

	"go-aiproxy/pkg/logger"

	"github.com/gin-gonic/gin"
)

const (
	// RequestIDHeader HTTP请求头中的request_id字段名
	RequestIDHeader = "X-Request-ID"
	// RequestIDCtxKey Gin Context中的request_id字段名
	RequestIDCtxKey = "request_id"
)

// generateRequestID 生成唯一的请求ID
func generateRequestID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// responseBodyWriter 用于捕获响应体大小
type responseBodyWriter struct {
	gin.ResponseWriter
	size int
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

// getRealClientIP 获取真实客户端IP（支持代理）
func getRealClientIP(c *gin.Context) string {
	// 优先检查 X-Forwarded-For
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For 可能包含多个IP，取第一个
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 其次检查 X-Real-IP
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}

	// 检查 CF-Connecting-IP (Cloudflare)
	if cfip := c.GetHeader("CF-Connecting-IP"); cfip != "" {
		return cfip
	}

	// 最后使用 Gin 的 ClientIP
	return c.ClientIP()
}

// Logger HTTP请求日志中间件
// 功能：生成request_id、注入context、记录详细结构化日志
func Logger() gin.HandlerFunc {
	log := logger.GetLogger("http")

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 获取或生成 request_id
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = generateRequestID()
		}

		// 注入到 Gin Context
		c.Set(RequestIDCtxKey, requestID)

		// 注入到 Go Context（用于日志）
		ctx := logger.SetRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		// 设置响应头
		c.Header(RequestIDHeader, requestID)

		// 获取请求体大小
		var requestBodySize int64
		if c.Request.Body != nil {
			// 读取请求体以获取大小，然后重新设置
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			requestBodySize = int64(len(bodyBytes))
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// 包装 ResponseWriter 以捕获响应体大小
		rbw := &responseBodyWriter{ResponseWriter: c.Writer, size: 0}
		c.Writer = rbw

		// 处理请求
		c.Next()

		// 计算耗时
		latency := time.Since(start)
		status := c.Writer.Status()

		// 获取详细信息
		clientIP := getRealClientIP(c)
		method := c.Request.Method
		userAgent := c.Request.UserAgent()
		referer := c.Request.Referer()
		contentType := c.GetHeader("Content-Type")
		host := c.Request.Host
		protocol := c.Request.Proto
		responseBodySize := rbw.size

		// 获取用户信息（如果已认证）
		var userID uint
		var username string
		var apiKeyID uint
		if uid, exists := c.Get("user_id"); exists {
			if id, ok := uid.(uint); ok {
				userID = id
			}
		}
		if uname, exists := c.Get("username"); exists {
			if name, ok := uname.(string); ok {
				username = name
			}
		}
		if keyID, exists := c.Get("api_key_id"); exists {
			if id, ok := keyID.(uint); ok {
				apiKeyID = id
			}
		}

		// 获取错误信息
		var errMsg string
		if len(c.Errors) > 0 {
			errMsg = c.Errors.String()
		}

		// 构建完整路径
		fullPath := path
		if query != "" {
			fullPath = path + "?" + query
		}

		// 构建日志字段
		fields := []logger.Field{
			logger.String("request_id", requestID),
			logger.String("client_ip", clientIP),
			logger.String("method", method),
			logger.String("path", fullPath),
			logger.Int("status", status),
			logger.Duration("latency", latency),
			logger.String("host", host),
			logger.String("protocol", protocol),
			logger.String("user_agent", userAgent),
			logger.String("content_type", contentType),
			logger.Int64("request_size", requestBodySize),
			logger.Int("response_size", responseBodySize),
		}

		// 添加可选字段
		if referer != "" {
			fields = append(fields, logger.String("referer", referer))
		}
		if userID > 0 {
			fields = append(fields, logger.Uint("user_id", userID))
		}
		if username != "" {
			fields = append(fields, logger.String("username", username))
		}
		if apiKeyID > 0 {
			fields = append(fields, logger.Uint("api_key_id", apiKeyID))
		}
		if errMsg != "" {
			fields = append(fields, logger.String("error", errMsg))
		}

		// 添加代理相关信息（如果存在）
		if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
			fields = append(fields, logger.String("x_forwarded_for", xff))
		}
		if xri := c.GetHeader("X-Real-IP"); xri != "" {
			fields = append(fields, logger.String("x_real_ip", xri))
		}

		// 根据状态码选择日志级别
		if status >= 500 {
			log.ErrorZ("HTTP请求", fields...)
		} else if status >= 400 {
			log.WarnZ("HTTP请求", fields...)
		} else {
			log.InfoZ("HTTP请求", fields...)
		}
	}
}

// GetRequestID 从Gin Context获取request_id
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get(RequestIDCtxKey); exists {
		if requestID, ok := id.(string); ok {
			return requestID
		}
	}
	return ""
}
