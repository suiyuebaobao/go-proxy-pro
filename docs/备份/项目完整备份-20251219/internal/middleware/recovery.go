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
