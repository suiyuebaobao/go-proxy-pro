/*
 * 文件作用：用户并发控制中间件，限制每个用户的最大并发请求数
 * 负责功能：
 *   - 用户并发数检查
 *   - 并发计数器管理
 *   - 请求完成后释放计数
 *   - 超限拒绝请求
 * 重要程度：⭐⭐⭐⭐ 重要（资源保护）
 * 依赖模块：cache, repository, model
 */
package middleware

import (
	"go-aiproxy/internal/cache"
	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/pkg/logger"
	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

// UserConcurrencyControl 用户并发控制中间件
// 限制每个用户的最大并发请求数
func UserConcurrencyControl() gin.HandlerFunc {
	sessionCache := cache.GetSessionCache()
	userRepo := repository.NewUserRepository()
	log := logger.GetLogger("middleware")

	return func(c *gin.Context) {
		// 获取用户ID
		userID, exists := c.Get("api_key_user_id")
		if !exists || userID == nil {
			// 没有用户信息，不做限制
			c.Next()
			return
		}

		uid, ok := userID.(uint)
		if !ok || uid == 0 {
			c.Next()
			return
		}

		// 获取用户信息以获取并发限制
		user, err := userRepo.GetByID(uid)
		if err != nil {
			log.Warn("获取用户信息失败: userID=%d, error=%v", uid, err)
			c.Next()
			return
		}

		limit := user.MaxConcurrency
		if limit <= 0 {
			limit = 10 // 默认限制
		}

		// 尝试获取并发槽位
		acquired, current, err := sessionCache.AcquireUserConcurrency(c.Request.Context(), uid, limit)
		if err != nil {
			log.Warn("获取用户并发槽位失败: userID=%d, error=%v", uid, err)
			// Redis 错误不阻止请求
			c.Next()
			return
		}

		if !acquired {
			log.Info("用户并发超限: userID=%d, current=%d, limit=%d", uid, current, limit)
			response.CustomTooManyRequestsAbort(c, model.ErrorTypeUserConcurrencyLimit,
				"Too many concurrent requests. Please try again later.")
			return
		}

		// 确保释放并发槽位
		c.Set("user_concurrency_acquired", true)
		c.Set("user_concurrency_uid", uid)

		// 使用 defer 释放并发槽位
		defer func() {
			if err := sessionCache.ReleaseUserConcurrency(c.Request.Context(), uid); err != nil {
				log.Warn("释放用户并发槽位失败: userID=%d, error=%v", uid, err)
			}
		}()

		c.Next()
	}
}
