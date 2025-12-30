package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// 静态文件目录（相对于可执行文件或工作目录）
var staticDir = "web/dist"

func init() {
	// 支持通过环境变量自定义静态文件目录
	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		staticDir = dir
	}
}

func RegisterStatic(r *gin.Engine) {
	// 检查静态文件目录是否存在
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		// 静态文件目录不存在，跳过静态文件服务
		// 这样可以支持纯 API 模式运行
		return
	}

	indexPath := filepath.Join(staticDir, "index.html")

	// 处理根路径
	r.GET("/", func(c *gin.Context) {
		c.File(indexPath)
	})

	// 处理静态资源
	r.Static("/assets", filepath.Join(staticDir, "assets"))

	// 处理其他静态文件（favicon.ico 等）
	r.GET("/favicon.ico", func(c *gin.Context) {
		faviconPath := filepath.Join(staticDir, "favicon.ico")
		if _, err := os.Stat(faviconPath); err == nil {
			c.File(faviconPath)
		} else {
			c.Status(http.StatusNotFound)
		}
	})

	// SPA 路由：非 API 请求返回 index.html
	r.NoRoute(func(c *gin.Context) {
		// API 请求返回 404
		if strings.HasPrefix(c.Request.URL.Path, "/api") ||
			strings.HasPrefix(c.Request.URL.Path, "/v1") {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "Not Found",
			})
			return
		}

		// 其他请求返回 index.html（SPA 路由）
		c.File(indexPath)
	})
}
