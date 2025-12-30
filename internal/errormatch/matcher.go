/*
 * 文件作用：错误规则匹配器，根据配置规则识别上游错误类型
 * 负责功能：
 *   - 错误规则缓存管理
 *   - HTTP状态码/关键词匹配
 *   - 目标账户状态确定
 *   - 规则优先级处理
 * 重要程度：⭐⭐⭐⭐ 重要（错误处理核心）
 * 依赖模块：model, repository, logger
 */
package errormatch

import (
	"strings"
	"sync"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/pkg/logger"
)

// ErrorRuleMatcher 错误规则匹配器（带缓存）
type ErrorRuleMatcher struct {
	repo  *repository.ErrorRuleRepository
	cache []model.ErrorRule
	mu    sync.RWMutex
}

var (
	defaultMatcher *ErrorRuleMatcher
	matcherOnce    sync.Once
)

// GetErrorRuleMatcher 获取错误规则匹配器单例
func GetErrorRuleMatcher() *ErrorRuleMatcher {
	matcherOnce.Do(func() {
		defaultMatcher = &ErrorRuleMatcher{
			repo: repository.NewErrorRuleRepository(),
		}
		defaultMatcher.Refresh()
	})
	return defaultMatcher
}

// Refresh 刷新缓存
func (m *ErrorRuleMatcher) Refresh() {
	log := logger.GetLogger("error_rule")
	rules, err := m.repo.GetAllEnabled()
	if err != nil {
		log.Error("刷新错误规则缓存失败: %v", err)
		return
	}

	m.mu.Lock()
	m.cache = rules
	m.mu.Unlock()

	log.Info("错误规则缓存已刷新，共 %d 条规则", len(rules))
}

// MatchResult 匹配结果
type MatchResult struct {
	Matched      bool
	TargetStatus string
	Rule         *model.ErrorRule
}

// Match 匹配错误
// httpStatusCode: HTTP状态码（0表示未知）
// errMsg: 错误信息
func (m *ErrorRuleMatcher) Match(httpStatusCode int, errMsg string) *MatchResult {
	m.mu.RLock()
	rules := m.cache
	m.mu.RUnlock()

	errMsgLower := strings.ToLower(errMsg)

	// 规则已按优先级排序，找到第一个匹配的就返回
	for _, rule := range rules {
		if m.matchRule(&rule, httpStatusCode, errMsgLower) {
			return &MatchResult{
				Matched:      true,
				TargetStatus: rule.TargetStatus,
				Rule:         &rule,
			}
		}
	}

	return &MatchResult{Matched: false}
}

// matchRule 检查单条规则是否匹配
func (m *ErrorRuleMatcher) matchRule(rule *model.ErrorRule, httpStatusCode int, errMsgLower string) bool {
	// 检查HTTP状态码
	statusCodeMatch := false
	if rule.HTTPStatusCode == 0 {
		// 规则不限制状态码
		statusCodeMatch = true
	} else if rule.HTTPStatusCode == httpStatusCode {
		// 状态码匹配
		statusCodeMatch = true
	}

	if !statusCodeMatch {
		return false
	}

	// 检查关键词
	keywordMatch := false
	if rule.Keyword == "" {
		// 规则不限制关键词
		keywordMatch = true
	} else if strings.Contains(errMsgLower, strings.ToLower(rule.Keyword)) {
		// 关键词匹配
		keywordMatch = true
	}

	return keywordMatch
}

// GetRuleCount 获取规则数量
func (m *ErrorRuleMatcher) GetRuleCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.cache)
}
