/*
 * 文件作用：用量配额检查中间件，在代理请求前校验用户/套餐余额
 * 负责功能：
 *   - 检查 API Key 绑定的套餐是否仍有可用额度
 *   - 使用内存缓存（TTL 60s）避免频繁查库
 *   - 支持通过系统配置控制策略（enforce / warn / off）
 * 重要程度：⭐⭐⭐⭐ 重要（费用控制）
 * 依赖模块：repository, model, service
 */
package middleware

import (
	"sync"
	"time"

	"fmt"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/internal/service"
	"go-aiproxy/pkg/logger"
	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

type quotaCacheEntry struct {
	allowed   bool
	expiresAt time.Time
}

var (
	quotaCache sync.Map // map[uint]*quotaCacheEntry — keyed by UserPackage ID
)

func checkPackageQuota(pkgID uint) bool {
	if entry, ok := quotaCache.Load(pkgID); ok {
		ce := entry.(*quotaCacheEntry)
		if time.Now().Before(ce.expiresAt) {
			return ce.allowed
		}
	}

	repo := repository.NewUserPackageRepository()
	pkg, err := repo.GetByID(pkgID)
	if err != nil || pkg == nil {
		cacheQuotaResult(pkgID, false)
		return false
	}

	allowed := pkg.IsValid()
	cacheQuotaResult(pkgID, allowed)
	return allowed
}

func cacheQuotaResult(pkgID uint, allowed bool) {
	quotaCache.Store(pkgID, &quotaCacheEntry{
		allowed:   allowed,
		expiresAt: time.Now().Add(60 * time.Second),
	})
}

// InvalidateQuotaCache clears quota cache for a specific package (call after usage deduction)
func InvalidateQuotaCache(pkgID uint) {
	quotaCache.Delete(pkgID)
}

// QuotaCheck 用量配额检查中间件（在 APIKeyAuth 之后使用）
func QuotaCheck() gin.HandlerFunc {
	configService := service.GetConfigService()
	log := logger.GetLogger("quota")

	return func(c *gin.Context) {
		policy := configService.GetString("quota_policy")
		if policy == "off" || policy == "" {
			c.Next()
			return
		}

		keyVal, exists := c.Get("api_key")
		if !exists {
			c.Next()
			return
		}
		apiKey := keyVal.(*model.APIKey)

		if apiKey.UserPackageID == nil {
			c.Next()
			return
		}

		pkgID := *apiKey.UserPackageID
		allowed := checkPackageQuota(pkgID)

		if !allowed {
			go service.GetAlertService().TriggerAlert("quota_exhausted",
				fmt.Sprintf("用户 #%d 配额耗尽 (套餐 #%d, Key #%d)", apiKey.UserID, pkgID, apiKey.ID))

			if policy == "enforce" {
				log.Warn("配额超限封停 | KeyID: %d | PackageID: %d | UserID: %d", apiKey.ID, pkgID, apiKey.UserID)
				response.Error(c, 402, "套餐额度/次数已用尽，请升级套餐或联系管理员")
				c.Abort()
				return
			}
			log.Warn("配额超限警告 | KeyID: %d | PackageID: %d | UserID: %d (未封停)", apiKey.ID, pkgID, apiKey.UserID)
		}

		c.Next()
	}
}
