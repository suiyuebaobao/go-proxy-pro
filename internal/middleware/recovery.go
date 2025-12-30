/*
 * 文件作用：Panic恢复中间件，捕获未处理的panic防止服务崩溃
 * 负责功能：
 *   - 捕获panic
 *   - 记录错误堆栈
 *   - 返回500错误响应
 * 重要程度：⭐⭐⭐⭐ 重要（服务稳定性保障）
 * 依赖模块：logger
 */
package middleware

import (
	"net/http"
	"runtime/debug"

	"go-aiproxy/pkg/logger"

	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	log := logger.GetLogger("http")

	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("PANIC | %v | Stack: %s", err, debug.Stack())
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": "Internal Server Error",
				})
			}
		}()
		c.Next()
	}
}
