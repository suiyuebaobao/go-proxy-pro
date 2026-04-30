/*
 * 文件作用：IP 访问控制中间件，实现全局黑名单和 API Key 级白名单
 * 负责功能：
 *   - 全局 IP 黑名单检查
 *   - API Key 级 IP 白名单检查
 *   - 支持单 IP 和 CIDR 网段匹配
 * 重要程度：⭐⭐⭐⭐ 重要（安全访问控制）
 * 依赖模块：service, model
 */
package middleware

import (
	"net"
	"strings"
	"sync"
	"time"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/service"
	"go-aiproxy/pkg/logger"
	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

var (
	ipBlacklistCache   []string
	ipBlacklistMu      sync.RWMutex
	ipBlacklistUpdated time.Time
)

func refreshIPBlacklist(configService *service.ConfigService) []string {
	ipBlacklistMu.RLock()
	if time.Since(ipBlacklistUpdated) < 30*time.Second && ipBlacklistCache != nil {
		defer ipBlacklistMu.RUnlock()
		return ipBlacklistCache
	}
	ipBlacklistMu.RUnlock()

	ipBlacklistMu.Lock()
	defer ipBlacklistMu.Unlock()

	raw := configService.GetString("ip_blacklist")
	var list []string
	if raw != "" {
		for _, s := range strings.Split(raw, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				list = append(list, s)
			}
		}
	}
	ipBlacklistCache = list
	ipBlacklistUpdated = time.Now()
	return list
}

func ipMatchesList(clientIP string, list []string) bool {
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}
	for _, entry := range list {
		if strings.Contains(entry, "/") {
			_, cidr, err := net.ParseCIDR(entry)
			if err == nil && cidr.Contains(ip) {
				return true
			}
		} else {
			if entry == clientIP {
				return true
			}
		}
	}
	return false
}

// IPAccessControl IP 访问控制中间件（在 APIKeyAuth 之后使用）
func IPAccessControl() gin.HandlerFunc {
	configService := service.GetConfigService()
	log := logger.GetLogger("ip_acl")

	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		// 1. 全局黑名单检查
		blacklist := refreshIPBlacklist(configService)
		if len(blacklist) > 0 && ipMatchesList(clientIP, blacklist) {
			log.Warn("IP 黑名单拦截 | IP: %s", clientIP)
			response.Error(c, 403, "IP 地址被禁止访问")
			c.Abort()
			return
		}

		// 2. API Key 级白名单检查
		if key, exists := c.Get("api_key"); exists {
			apiKey := key.(*model.APIKey)
			if apiKey.AllowedIPs != "" {
				var whitelist []string
				for _, s := range strings.Split(apiKey.AllowedIPs, ",") {
					s = strings.TrimSpace(s)
					if s != "" {
						whitelist = append(whitelist, s)
					}
				}
				if len(whitelist) > 0 && !ipMatchesList(clientIP, whitelist) {
					log.Warn("IP 白名单拦截 | IP: %s | KeyID: %d", clientIP, apiKey.ID)
					response.Error(c, 403, "当前 IP 不在 API Key 允许的访问范围内")
					c.Abort()
					return
				}
			}
		}

		c.Next()
	}
}
