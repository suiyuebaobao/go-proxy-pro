/*
 * 文件作用：静态文件服务，提供前端SPA应用的静态资源托管
 * 负责功能：
 *   - 前端静态资源托管
 *   - SPA路由回退处理
 *   - 环境变量配置静态目录
 * 重要程度：⭐⭐ 辅助（前端托管）
 * 依赖模块：无
 */
package handler

import (
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
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
	assetsRoot := filepath.Join(staticDir, "assets")

	// 处理根路径
	r.GET("/", func(c *gin.Context) {
		// SPA 入口页不要缓存，避免发布后出现“旧 index.html + 新 assets”不一致
		c.Header("Cache-Control", "no-store, max-age=0")
		c.File(indexPath)
	})

	// 处理静态资源（优先返回预压缩 .gz，避免在线 gzip 的 chunked 传输）
	serveAsset := func(c *gin.Context) {
		rel := strings.TrimPrefix(c.Param("filepath"), "/")
		if rel == "" || rel == "." {
			c.Status(http.StatusNotFound)
			return
		}

		clean := path.Clean("/" + rel)
		clean = strings.TrimPrefix(clean, "/")
		if clean == "" || clean == "." || strings.HasPrefix(clean, "..") {
			c.Status(http.StatusNotFound)
			return
		}

		origPath := filepath.Join(assetsRoot, filepath.FromSlash(clean))
		assetsRootAbs, _ := filepath.Abs(assetsRoot)
		origAbs, _ := filepath.Abs(origPath)
		if assetsRootAbs != "" && origAbs != "" && !strings.HasPrefix(origAbs, assetsRootAbs+string(os.PathSeparator)) {
			c.Status(http.StatusNotFound)
			return
		}

		if _, err := os.Stat(origPath); err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		// 带 hash 的静态资源可长缓存，显著提升后台打开速度
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Header("Vary", "Accept-Encoding")

		if ct := mime.TypeByExtension(filepath.Ext(origPath)); ct != "" {
			c.Header("Content-Type", ct)
		}

		acceptEnc := c.GetHeader("Accept-Encoding")
		if strings.Contains(acceptEnc, "gzip") {
			gzPath := origPath + ".gz"
			if st, err := os.Stat(gzPath); err == nil && st.Mode().IsRegular() {
				c.Header("Content-Encoding", "gzip")
				c.Header("Content-Length", strconv.FormatInt(st.Size(), 10))
				c.File(gzPath)
				return
			}
		}

		c.File(origPath)
	}
	r.GET("/assets/*filepath", serveAsset)
	r.HEAD("/assets/*filepath", serveAsset)

	// 处理其他静态文件（favicon.ico 等）
	faviconSvgHandler := func(c *gin.Context) {
		faviconPath := filepath.Join(staticDir, "favicon.svg")
		if _, err := os.Stat(faviconPath); err == nil {
			c.Header("Cache-Control", "public, max-age=86400")
			c.File(faviconPath)
			return
		}
		c.Status(http.StatusNoContent)
	}
	r.GET("/favicon.svg", faviconSvgHandler)
	r.HEAD("/favicon.svg", faviconSvgHandler)

	faviconHandler := func(c *gin.Context) {
		faviconPath := filepath.Join(staticDir, "favicon.ico")
		if _, err := os.Stat(faviconPath); err == nil {
			c.Header("Cache-Control", "public, max-age=86400")
			c.File(faviconPath)
		} else {
			// 没有 favicon.ico 时，尽量给浏览器一个可用的 icon（避免 404 噪音）
			if _, err := os.Stat(filepath.Join(staticDir, "favicon.svg")); err == nil {
				c.Redirect(http.StatusFound, "/favicon.svg")
				return
			}
			c.Status(http.StatusNoContent)
		}
	}
	r.GET("/favicon.ico", faviconHandler)
	r.HEAD("/favicon.ico", faviconHandler)

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
		c.Header("Cache-Control", "no-store, max-age=0")
		c.File(indexPath)
	})
}
