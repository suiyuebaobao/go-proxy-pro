/*
 * 文件作用：频率限制服务，基于IP的请求频率控制
 * 负责功能：
 *   - 登录频率限制
 *   - 验证码获取频率限制
 *   - 滑动窗口计数
 *   - 自动清理过期记录
 * 重要程度：⭐⭐⭐ 一般（安全防护）
 * 依赖模块：无
 */
package service

import (
	"fmt"
	"sync"
	"time"
)

// RateLimiter IP 频率限制器
type RateLimiter struct {
	mu       sync.RWMutex
	attempts map[string]*attemptRecord
}

type attemptRecord struct {
	count     int
	firstTime time.Time
}

var (
	loginRateLimiter   *RateLimiter
	captchaRateLimiter *RateLimiter
	rateLimiterOnce    sync.Once
)

// GetLoginRateLimiter 获取登录频率限制器
func GetLoginRateLimiter() *RateLimiter {
	rateLimiterOnce.Do(func() {
		loginRateLimiter = &RateLimiter{
			attempts: make(map[string]*attemptRecord),
		}
		captchaRateLimiter = &RateLimiter{
			attempts: make(map[string]*attemptRecord),
		}
		// 定期清理过期记录
		go loginRateLimiter.cleanup()
		go captchaRateLimiter.cleanup()
	})
	return loginRateLimiter
}

// GetCaptchaRateLimiter 获取验证码频率限制器
func GetCaptchaRateLimiter() *RateLimiter {
	GetLoginRateLimiter() // 确保初始化
	return captchaRateLimiter
}

// Check 检查是否允许操作
// ip: 客户端 IP
// limit: 限制次数
// window: 时间窗口（分钟）
// 返回: 是否允许, 剩余等待时间(秒)
func (r *RateLimiter) Check(ip string, limit int, window int) (bool, int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	windowDuration := time.Duration(window) * time.Minute
	now := time.Now()

	record, exists := r.attempts[ip]
	if !exists {
		// 首次访问
		r.attempts[ip] = &attemptRecord{
			count:     1,
			firstTime: now,
		}
		return true, 0
	}

	// 检查是否过了时间窗口
	if now.Sub(record.firstTime) >= windowDuration {
		// 重置计数
		record.count = 1
		record.firstTime = now
		return true, 0
	}

	// 在时间窗口内
	if record.count >= limit {
		// 超过限制
		waitSeconds := int(windowDuration.Seconds() - now.Sub(record.firstTime).Seconds())
		return false, waitSeconds
	}

	// 增加计数
	record.count++
	return true, 0
}

// Reset 重置某个 IP 的计数
func (r *RateLimiter) Reset(ip string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.attempts, ip)
}

// cleanup 定期清理过期记录
func (r *RateLimiter) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		r.mu.Lock()
		now := time.Now()
		for ip, record := range r.attempts {
			// 清理超过 30 分钟的记录
			if now.Sub(record.firstTime) > 30*time.Minute {
				delete(r.attempts, ip)
			}
		}
		r.mu.Unlock()
	}
}

// GetRateLimitError 获取频率限制错误信息
func GetRateLimitError(waitSeconds int) string {
	if waitSeconds >= 60 {
		return fmt.Sprintf("请求过于频繁，请 %d 分钟后再试", waitSeconds/60+1)
	}
	return fmt.Sprintf("请求过于频繁，请 %d 秒后再试", waitSeconds)
}
