/*
 * 文件作用：API Key 速率限制中间件，实现 RPM/RPD 滑动窗口
 * 负责功能：
 *   - 每分钟请求数（RPM）限制
 *   - 每日请求数（RPD）限制
 *   - 基于 sync.Map 的滑动窗口计数
 *   - 超限返回 HTTP 429 + Retry-After
 * 重要程度：⭐⭐⭐⭐ 重要（流量控制）
 * 依赖模块：model
 */
package middleware

import (
	"fmt"
	"sync"
	"time"

	"go-aiproxy/internal/model"
	"go-aiproxy/pkg/logger"
	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

type slidingWindow struct {
	mu         sync.Mutex
	timestamps []int64
}

func (w *slidingWindow) countAndAdd(windowSec int64, now int64) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	cutoff := now - windowSec
	start := 0
	for start < len(w.timestamps) && w.timestamps[start] <= cutoff {
		start++
	}
	w.timestamps = w.timestamps[start:]
	count := len(w.timestamps)
	w.timestamps = append(w.timestamps, now)
	return count
}

func (w *slidingWindow) rollback() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.timestamps) > 0 {
		w.timestamps = w.timestamps[:len(w.timestamps)-1]
	}
}

var (
	rpmWindows sync.Map // map[uint]*slidingWindow
	rpdWindows sync.Map
)

func getWindow(store *sync.Map, keyID uint) *slidingWindow {
	val, _ := store.LoadOrStore(keyID, &slidingWindow{})
	return val.(*slidingWindow)
}

func init() {
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			cleanOldEntries(&rpmWindows, 60)
			cleanOldEntries(&rpdWindows, 86400)
		}
	}()
}

func cleanOldEntries(store *sync.Map, windowSec int64) {
	cutoff := time.Now().Unix() - windowSec
	store.Range(func(key, value interface{}) bool {
		w := value.(*slidingWindow)
		w.mu.Lock()
		allOld := len(w.timestamps) == 0 || (len(w.timestamps) > 0 && w.timestamps[len(w.timestamps)-1] <= cutoff)
		w.mu.Unlock()
		if allOld {
			store.Delete(key)
		}
		return true
	})
}

// RateLimiter API Key 速率限制中间件（在 APIKeyAuth 之后使用）
func RateLimiter() gin.HandlerFunc {
	log := logger.GetLogger("rate_limit")

	return func(c *gin.Context) {
		keyVal, exists := c.Get("api_key")
		if !exists {
			c.Next()
			return
		}
		apiKey := keyVal.(*model.APIKey)
		now := time.Now().Unix()

		// RPM 检查
		if apiKey.RateLimit > 0 {
			w := getWindow(&rpmWindows, apiKey.ID)
			count := w.countAndAdd(60, now)
			if count >= apiKey.RateLimit {
				w.rollback()
				log.Warn("RPM 超限 | KeyID: %d | Count: %d | Limit: %d", apiKey.ID, count, apiKey.RateLimit)
				c.Header("Retry-After", "60")
				c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", apiKey.RateLimit))
				c.Header("X-RateLimit-Remaining", "0")
				response.Error(c, 429, fmt.Sprintf("请求速率超限，每分钟最多 %d 次请求", apiKey.RateLimit))
				c.Abort()
				return
			}
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", apiKey.RateLimit))
			c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", apiKey.RateLimit-count-1))
		}

		// RPD 检查
		if apiKey.DailyLimit > 0 {
			w := getWindow(&rpdWindows, apiKey.ID)
			count := w.countAndAdd(86400, now)
			if count >= apiKey.DailyLimit {
				w.rollback()
				log.Warn("RPD 超限 | KeyID: %d | Count: %d | Limit: %d", apiKey.ID, count, apiKey.DailyLimit)
				c.Header("Retry-After", "3600")
				c.Header("X-RateLimit-Limit-Day", fmt.Sprintf("%d", apiKey.DailyLimit))
				c.Header("X-RateLimit-Remaining-Day", "0")
				response.Error(c, 429, fmt.Sprintf("每日请求次数超限，最多 %d 次/天", apiKey.DailyLimit))
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
