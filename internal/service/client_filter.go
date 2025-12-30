/*
 * 文件作用：客户端过滤服务，验证和识别请求来源客户端
 * 负责功能：
 *   - 客户端类型定义管理
 *   - 过滤规则管理
 *   - 请求验证（Header/Body匹配）
 *   - 正则表达式缓存
 *   - 验证结果生成
 * 重要程度：⭐⭐⭐⭐ 重要（安全过滤核心）
 * 依赖模块：repository, model, logger
 */
package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/pkg/logger"
)

// ClientFilterService 客户端过滤服务
type ClientFilterService struct {
	repo   *repository.ClientFilterRepository
	cache  *clientFilterCache
	logger *logger.Logger
}

// clientFilterCache 规则缓存
type clientFilterCache struct {
	sync.RWMutex
	config      *model.ClientFilterConfig
	clientTypes map[string]*model.ClientType      // key: client_id
	rules       map[uint][]model.ClientFilterRule // key: client_type_id
	regexCache  map[string]*regexp.Regexp         // 正则表达式缓存
	regexMu     sync.RWMutex                      // 正则缓存专用锁
}

// ValidationResult 验证结果
type ValidationResult struct {
	Allowed       bool                 `json:"allowed"`        // 是否允许
	ClientType    string               `json:"client_type"`    // 识别的客户端类型
	ClientName    string               `json:"client_name"`    // 客户端名称
	MatchedRules  []RuleMatchResult    `json:"matched_rules"`  // 匹配的规则
	FailedRules   []RuleMatchResult    `json:"failed_rules"`   // 失败的规则
	Warnings      []string             `json:"warnings"`       // 警告信息
	Details       map[string]string    `json:"details"`        // 详细信息
}

// RuleMatchResult 规则匹配结果
type RuleMatchResult struct {
	RuleKey     string `json:"rule_key"`
	RuleName    string `json:"rule_name"`
	RuleType    string `json:"rule_type"`
	Pattern     string `json:"pattern,omitempty"`
	FieldPath   string `json:"field_path,omitempty"`
	ActualValue string `json:"actual_value,omitempty"`
	Required    bool   `json:"required"`
	Passed      bool   `json:"passed"`
	Message     string `json:"message,omitempty"`
}

// RequestContext 请求上下文（用于验证）
type RequestContext struct {
	UserAgent string                 `json:"user_agent"`
	Headers   map[string]string      `json:"headers"`
	Path      string                 `json:"path"`
	Body      map[string]interface{} `json:"body"`
}

var (
	clientFilterService     *ClientFilterService
	clientFilterServiceOnce sync.Once
)

// GetClientFilterService 获取客户端过滤服务单例
func GetClientFilterService() *ClientFilterService {
	clientFilterServiceOnce.Do(func() {
		clientFilterService = &ClientFilterService{
			repo:   repository.NewClientFilterRepository(),
			logger: logger.GetLogger("client_filter"),
			cache: &clientFilterCache{
				clientTypes: make(map[string]*model.ClientType),
				rules:       make(map[uint][]model.ClientFilterRule),
				regexCache:  make(map[string]*regexp.Regexp),
			},
		}
		// 加载初始数据
		clientFilterService.ReloadCache()
	})
	return clientFilterService
}

// ReloadCache 重新加载缓存
func (s *ClientFilterService) ReloadCache() error {
	s.cache.Lock()
	defer s.cache.Unlock()

	// 加载配置
	config, err := s.repo.GetConfig()
	if err != nil {
		s.logger.Error("加载过滤配置失败: %v", err)
		return err
	}
	s.cache.config = config

	// 加载客户端类型
	types, err := s.repo.ListClientTypes()
	if err != nil {
		s.logger.Error("加载客户端类型失败: %v", err)
		return err
	}
	s.cache.clientTypes = make(map[string]*model.ClientType)
	for i := range types {
		s.cache.clientTypes[types[i].ClientID] = &types[i]
	}

	// 加载规则
	s.cache.rules = make(map[uint][]model.ClientFilterRule)
	for _, ct := range types {
		rules, err := s.repo.ListRulesByClientTypeID(ct.ID)
		if err != nil {
			s.logger.Error("加载客户端 %s 的规则失败: %v", ct.ClientID, err)
			continue
		}
		s.cache.rules[ct.ID] = rules
	}

	// 清空正则缓存（会重新编译）
	s.cache.regexCache = make(map[string]*regexp.Regexp)

	s.logger.Info("客户端过滤缓存已重新加载 | 客户端类型: %d | 规则总数: %d",
		len(s.cache.clientTypes), s.countTotalRules())

	return nil
}

func (s *ClientFilterService) countTotalRules() int {
	count := 0
	for _, rules := range s.cache.rules {
		count += len(rules)
	}
	return count
}

// IsFilterEnabled 检查过滤是否启用
func (s *ClientFilterService) IsFilterEnabled() bool {
	s.cache.RLock()
	defer s.cache.RUnlock()
	return s.cache.config != nil && s.cache.config.FilterEnabled
}

// GetConfig 获取当前配置
func (s *ClientFilterService) GetConfig() *model.ClientFilterConfig {
	s.cache.RLock()
	defer s.cache.RUnlock()
	return s.cache.config
}

// ValidateRequest 验证请求
func (s *ClientFilterService) ValidateRequest(ctx *RequestContext) *ValidationResult {
	result := &ValidationResult{
		Allowed:      true,
		MatchedRules: make([]RuleMatchResult, 0),
		FailedRules:  make([]RuleMatchResult, 0),
		Warnings:     make([]string, 0),
		Details:      make(map[string]string),
	}

	s.cache.RLock()
	config := s.cache.config
	s.cache.RUnlock()

	// 如果过滤未启用，直接允许
	if config == nil || !config.FilterEnabled {
		result.Details["reason"] = "过滤功能未启用"
		return result
	}

	// 识别客户端类型
	clientType := s.identifyClientType(ctx)
	if clientType != nil {
		result.ClientType = clientType.ClientID
		result.ClientName = clientType.Name
		result.Details["client_icon"] = clientType.Icon

		// 检查客户端是否被允许
		if !clientType.Enabled {
			result.Allowed = false
			result.Details["reason"] = "客户端类型已禁用"
			return result
		}

		// 验证规则
		s.validateRules(ctx, clientType, result)
	} else {
		result.ClientType = model.ClientIDUnknown
		result.ClientName = "未知客户端"

		// 检查默认策略
		if !config.DefaultAllow {
			result.Allowed = false
			result.Details["reason"] = "未识别的客户端，默认拒绝"
		}

		if config.LogUnmatchedRequests {
			result.Warnings = append(result.Warnings, "未能识别客户端类型")
		}
	}

	return result
}

// identifyClientType 识别客户端类型
func (s *ClientFilterService) identifyClientType(ctx *RequestContext) *model.ClientType {
	s.cache.RLock()
	defer s.cache.RUnlock()

	// 按优先级排序的客户端类型
	var sortedTypes []*model.ClientType
	for _, ct := range s.cache.clientTypes {
		if ct.Enabled {
			sortedTypes = append(sortedTypes, ct)
		}
	}

	// 按优先级排序（高优先级优先）
	for i := 0; i < len(sortedTypes); i++ {
		for j := i + 1; j < len(sortedTypes); j++ {
			if sortedTypes[j].Priority > sortedTypes[i].Priority {
				sortedTypes[i], sortedTypes[j] = sortedTypes[j], sortedTypes[i]
			}
		}
	}

	// 尝试匹配每种客户端类型
	for _, ct := range sortedTypes {
		if s.matchClientType(ctx, ct) {
			return ct
		}
	}

	return nil
}

// matchClientType 检查是否匹配特定客户端类型
func (s *ClientFilterService) matchClientType(ctx *RequestContext, ct *model.ClientType) bool {
	// 根据过滤模式使用不同的 User-Agent 匹配规则
	config := s.cache.config
	filterMode := model.FilterModeSimple
	if config != nil && config.FilterMode != "" {
		filterMode = config.FilterMode
	}

	// Claude Code 特殊处理
	if ct.ClientID == model.ClientIDClaudeCode {
		if filterMode == model.FilterModeSimple {
			// 简单模式: 宽松的 UA 检查 (claude-relay 风格)
			return s.matchPattern(`^claude-cli/\d+\.\d+\.\d+`, ctx.UserAgent)
		} else {
			// 严格模式: 完整的 UA 格式检查 (AIProxyV2 风格)
			return s.matchPattern(`^claude-cli/(\d+\.\d+\.\d+)\s*\(external,\s*(cli|claude-vscode|sdk-ts|sdk-cli)(?:,\s*agent-sdk/[\w.\-]+)?\)$`, ctx.UserAgent)
		}
	}

	// 其他客户端使用数据库中定义的规则
	rules := s.cache.rules[ct.ID]
	if len(rules) == 0 {
		return false
	}

	for _, rule := range rules {
		if rule.RuleType == model.RuleTypeUserAgent && rule.Enabled {
			if s.matchPattern(rule.Pattern, ctx.UserAgent) {
				return true
			}
		}
	}

	return false
}

// validateRules 验证客户端的所有规则
func (s *ClientFilterService) validateRules(ctx *RequestContext, ct *model.ClientType, result *ValidationResult) {
	s.cache.RLock()
	config := s.cache.config
	s.cache.RUnlock()

	filterMode := model.FilterModeSimple
	if config != nil && config.FilterMode != "" {
		filterMode = config.FilterMode
	}

	// Claude Code 使用内置规则集，根据模式选择
	if ct.ClientID == model.ClientIDClaudeCode {
		s.validateClaudeCodeRules(ctx, filterMode, result)
		return
	}

	// 其他客户端使用数据库中的规则
	s.cache.RLock()
	rules := s.cache.rules[ct.ID]
	s.cache.RUnlock()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		ruleResult := s.validateSingleRule(ctx, &rule)

		if ruleResult.Passed {
			result.MatchedRules = append(result.MatchedRules, ruleResult)
		} else {
			result.FailedRules = append(result.FailedRules, ruleResult)

			if rule.Required {
				result.Allowed = false
				result.Details["reason"] = "规则验证失败: " + rule.RuleName
			} else {
				result.Warnings = append(result.Warnings, "可选规则未通过: "+rule.RuleName)
			}
		}
	}
}

// validateClaudeCodeRules Claude Code 专用验证（根据模式选择规则集）
func (s *ClientFilterService) validateClaudeCodeRules(ctx *RequestContext, filterMode string, result *ValidationResult) {
	// 简单模式规则 (claude-relay 风格):
	// 1. x-app 头存在
	// 2. anthropic-beta 头存在
	// 3. anthropic-version 头存在
	// 4. System Prompt 相似度
	// 5. metadata.user_id 格式

	// 严格模式额外规则 (AIProxyV2 风格):
	// + x-stainless-os 头存在
	// (User-Agent 已在 matchClientType 中用严格正则验证)

	// 公共规则
	rules := []struct {
		Key      string
		Name     string
		Check    func() (bool, string, string)
		Required bool
	}{
		{
			Key:  "x_app",
			Name: "X-App 头检查",
			Check: func() (bool, string, string) {
				val := s.getHeaderValue(ctx.Headers, "x-app")
				return val != "", val, "验证请求包含 X-App 头"
			},
			Required: true,
		},
		{
			Key:  "anthropic_version",
			Name: "Anthropic-Version 头检查",
			Check: func() (bool, string, string) {
				val := s.getHeaderValue(ctx.Headers, "anthropic-version")
				return val != "", val, "验证请求包含 Anthropic-Version 头"
			},
			Required: true,
		},
	}

	// 简单模式额外检查 anthropic-beta
	if filterMode == model.FilterModeSimple {
		rules = append(rules, struct {
			Key      string
			Name     string
			Check    func() (bool, string, string)
			Required bool
		}{
			Key:  "anthropic_beta",
			Name: "Anthropic-Beta 头检查",
			Check: func() (bool, string, string) {
				val := s.getHeaderValue(ctx.Headers, "anthropic-beta")
				return val != "", val, "验证请求包含 Anthropic-Beta 头"
			},
			Required: true,
		})
	}

	// 严格模式额外检查 x-stainless-os
	if filterMode == model.FilterModeStrict {
		rules = append(rules, struct {
			Key      string
			Name     string
			Check    func() (bool, string, string)
			Required bool
		}{
			Key:  "x_stainless_os",
			Name: "X-Stainless-Os 头检查",
			Check: func() (bool, string, string) {
				val := s.getHeaderValue(ctx.Headers, "x-stainless-os")
				return val != "", val, "验证请求包含 X-Stainless-Os 头"
			},
			Required: true,
		})
	}

	// System Prompt 相似度验证
	rules = append(rules, struct {
		Key      string
		Name     string
		Check    func() (bool, string, string)
		Required bool
	}{
		Key:  "system_prompt",
		Name: "System Prompt 相似度验证",
		Check: func() (bool, string, string) {
			ruleResult := &RuleMatchResult{}
			passed := s.validateClaudeCodeSystemPrompt(ctx, ruleResult)
			return passed, ruleResult.ActualValue, ruleResult.Message
		},
		Required: true,
	})

	// metadata.user_id 格式验证
	rules = append(rules, struct {
		Key      string
		Name     string
		Check    func() (bool, string, string)
		Required bool
	}{
		Key:  "metadata_user_id",
		Name: "metadata.user_id 格式验证",
		Check: func() (bool, string, string) {
			val := s.getBodyValue(ctx.Body, "metadata.user_id")
			pattern := `^user_[a-fA-F0-9]{64}_account__session_[\w-]+$`
			passed := s.matchPattern(pattern, val)
			msg := "验证格式: user_{64位hex}_account__session_{UUID}"
			if !passed && val != "" {
				msg = "格式不匹配: " + truncateString(val, 50)
			}
			return passed, val, msg
		},
		Required: true,
	})

	// 执行验证
	for _, rule := range rules {
		passed, actualValue, message := rule.Check()

		ruleResult := RuleMatchResult{
			RuleKey:     rule.Key,
			RuleName:    rule.Name,
			RuleType:    model.RuleTypeHeader,
			ActualValue: actualValue,
			Required:    rule.Required,
			Passed:      passed,
			Message:     message,
		}

		if passed {
			result.MatchedRules = append(result.MatchedRules, ruleResult)
		} else {
			result.FailedRules = append(result.FailedRules, ruleResult)
			if rule.Required {
				result.Allowed = false
				result.Details["reason"] = "规则验证失败: " + rule.Name
			}
		}
	}
}

// validateSingleRule 验证单个规则
func (s *ClientFilterService) validateSingleRule(ctx *RequestContext, rule *model.ClientFilterRule) RuleMatchResult {
	result := RuleMatchResult{
		RuleKey:   rule.RuleKey,
		RuleName:  rule.RuleName,
		RuleType:  rule.RuleType,
		Pattern:   rule.Pattern,
		FieldPath: rule.FieldPath,
		Required:  rule.Required,
		Passed:    false,
	}

	switch rule.RuleType {
	case model.RuleTypeUserAgent:
		result.ActualValue = ctx.UserAgent
		result.Passed = s.matchPattern(rule.Pattern, ctx.UserAgent)

	case model.RuleTypeHeader:
		value := s.getHeaderValue(ctx.Headers, rule.FieldPath)
		result.ActualValue = value
		if rule.Pattern == "" || rule.Pattern == ".+" {
			result.Passed = value != ""
		} else {
			result.Passed = s.matchPattern(rule.Pattern, value)
		}

	case model.RuleTypeBody:
		value := s.getBodyValue(ctx.Body, rule.FieldPath)
		result.ActualValue = value
		if rule.Pattern == "" || rule.Pattern == ".+" {
			result.Passed = value != ""
		} else {
			result.Passed = s.matchPattern(rule.Pattern, value)
		}

	case model.RuleTypePath:
		result.ActualValue = ctx.Path
		result.Passed = s.matchPattern(rule.Pattern, ctx.Path)

	case model.RuleTypeCustom:
		result.Passed = s.validateCustomRule(ctx, rule, &result)
	}

	// 如果 custom 规则已设置详细消息，不覆盖
	if result.Message == "" {
		if result.Passed {
			result.Message = "验证通过"
		} else {
			result.Message = "验证失败"
		}
	}

	return result
}

// matchPattern 使用正则表达式匹配
func (s *ClientFilterService) matchPattern(pattern, value string) bool {
	if pattern == "" {
		return true
	}

	// 先尝试读取缓存
	s.cache.regexMu.RLock()
	re, ok := s.cache.regexCache[pattern]
	s.cache.regexMu.RUnlock()

	if !ok {
		// 需要编译正则表达式
		s.cache.regexMu.Lock()
		// 双重检查
		re, ok = s.cache.regexCache[pattern]
		if !ok {
			var err error
			re, err = regexp.Compile(pattern)
			if err != nil {
				s.logger.Error("编译正则表达式失败: %s, %v", pattern, err)
				s.cache.regexMu.Unlock()
				return false
			}
			s.cache.regexCache[pattern] = re
		}
		s.cache.regexMu.Unlock()
	}

	return re.MatchString(value)
}

// getHeaderValue 获取请求头值（不区分大小写）
func (s *ClientFilterService) getHeaderValue(headers map[string]string, key string) string {
	// 直接匹配
	if v, ok := headers[key]; ok {
		return v
	}
	// 小写匹配
	keyLower := strings.ToLower(key)
	for k, v := range headers {
		if strings.ToLower(k) == keyLower {
			return v
		}
	}
	return ""
}

// getBodyValue 获取请求体中的嵌套值
func (s *ClientFilterService) getBodyValue(body map[string]interface{}, path string) string {
	if body == nil || path == "" {
		return ""
	}

	parts := strings.Split(path, ".")
	current := interface{}(body)

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			current = v[part]
		default:
			return ""
		}
	}

	switch v := current.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		// 尝试 JSON 序列化
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
		return ""
	}
}

// validateCustomRule 验证自定义规则
func (s *ClientFilterService) validateCustomRule(ctx *RequestContext, rule *model.ClientFilterRule, result *RuleMatchResult) bool {
	switch rule.RuleKey {
	case model.RuleClaudeCodeSystemPrompt:
		return s.validateClaudeCodeSystemPrompt(ctx, result)
	default:
		result.Message = "未知的自定义规则"
		return false
	}
}

// validateClaudeCodeSystemPrompt 验证 Claude Code 系统提示词（使用 Dice 相似度算法）
func (s *ClientFilterService) validateClaudeCodeSystemPrompt(ctx *RequestContext, result *RuleMatchResult) bool {
	// 获取系统提示词
	systemMessages := s.extractSystemMessages(ctx.Body)

	if len(systemMessages) == 0 {
		result.Message = "未找到系统提示词（空消息直接通过）"
		return true // 空消息直接通过，与 AIProxyV2 一致
	}

	// 对每个系统消息计算最高相似度
	maxScore := 0.0
	matchedTemplate := ""

	for _, msg := range systemMessages {
		if msg == "" {
			continue
		}

		for _, template := range model.ClaudeCodeSystemPromptTemplates {
			score := s.calculateDiceSimilarity(msg, template)
			if score > maxScore {
				maxScore = score
				matchedTemplate = template
			}
		}

		// 如果达到阈值，直接通过
		if maxScore >= model.SystemPromptSimilarityThreshold {
			result.ActualValue = truncateString(msg, 100)
			result.Message = fmt.Sprintf("相似度 %.2f >= %.2f (匹配模板: %s)",
				maxScore, model.SystemPromptSimilarityThreshold, truncateString(matchedTemplate, 50))
			return true
		}
	}

	result.ActualValue = truncateString(systemMessages[0], 100)
	result.Message = fmt.Sprintf("相似度 %.2f < %.2f (最佳匹配: %s)",
		maxScore, model.SystemPromptSimilarityThreshold, truncateString(matchedTemplate, 50))
	return false
}

// extractSystemMessages 从请求体提取系统消息
func (s *ClientFilterService) extractSystemMessages(body map[string]interface{}) []string {
	var messages []string

	// 方式1: 直接 system 字段（字符串）
	if system, ok := body["system"].(string); ok && system != "" {
		messages = append(messages, system)
	}

	// 方式2: system 字段是数组（Anthropic 格式）
	if systemArr, ok := body["system"].([]interface{}); ok {
		for _, item := range systemArr {
			if m, ok := item.(map[string]interface{}); ok {
				if text, ok := m["text"].(string); ok && text != "" {
					messages = append(messages, text)
				}
			}
		}
	}

	return messages
}

// calculateDiceSimilarity 计算 Sørensen-Dice 相似度系数
// 算法与 AIProxyV2 完全一致
func (s *ClientFilterService) calculateDiceSimilarity(s1, s2 string) float64 {
	// 归一化处理
	s1 = s.normalizePrompt(s1)
	s2 = s.normalizePrompt(s2)

	if s1 == s2 {
		return 1.0
	}
	if s1 == "" || s2 == "" {
		return 0.0
	}

	// 提取 bigrams
	bigrams1 := s.extractBigrams(s1)
	bigrams2 := s.extractBigrams(s2)

	if len(bigrams1) == 0 || len(bigrams2) == 0 {
		return 0.0
	}

	// 计算交集
	intersection := 0
	for bigram, count1 := range bigrams1 {
		if count2, exists := bigrams2[bigram]; exists {
			if count1 < count2 {
				intersection += count1
			} else {
				intersection += count2
			}
		}
	}

	// Dice 系数 = 2 * |交集| / (|集合1| + |集合2|)
	totalBigrams := len(bigrams1) + len(bigrams2)
	if totalBigrams == 0 {
		return 0.0
	}

	return 2.0 * float64(intersection) / float64(totalBigrams)
}

// extractBigrams 提取字符串的所有 bigrams（二元组）
func (s *ClientFilterService) extractBigrams(str string) map[string]int {
	runes := []rune(str)
	bigrams := make(map[string]int)

	for i := 0; i < len(runes)-1; i++ {
		bigram := string(runes[i : i+2])
		bigrams[bigram]++
	}

	return bigrams
}

// normalizePrompt 归一化 Prompt 文本
func (s *ClientFilterService) normalizePrompt(str string) string {
	// 替换占位符
	str = strings.ReplaceAll(str, "__PLACEHOLDER__", " ")

	// 折叠空白字符
	re := regexp.MustCompile(`\s+`)
	str = re.ReplaceAllString(str, " ")

	return strings.TrimSpace(str)
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ==================== 管理操作 ====================

// GetAllClientTypes 获取所有客户端类型
func (s *ClientFilterService) GetAllClientTypes() ([]model.ClientType, error) {
	return s.repo.ListClientTypes()
}

// GetClientType 获取客户端类型
func (s *ClientFilterService) GetClientType(id uint) (*model.ClientType, error) {
	return s.repo.GetClientTypeByID(id)
}

// UpdateClientType 更新客户端类型
func (s *ClientFilterService) UpdateClientType(ct *model.ClientType) error {
	if err := s.repo.UpdateClientType(ct); err != nil {
		return err
	}
	return s.ReloadCache()
}

// GetRulesByClientType 获取客户端类型的所有规则
func (s *ClientFilterService) GetRulesByClientType(clientTypeID uint) ([]model.ClientFilterRule, error) {
	return s.repo.ListRulesByClientTypeID(clientTypeID)
}

// GetAllRules 获取所有规则
func (s *ClientFilterService) GetAllRules() ([]model.ClientFilterRule, error) {
	return s.repo.ListAllRules()
}

// GetRule 获取规则
func (s *ClientFilterService) GetRule(id uint) (*model.ClientFilterRule, error) {
	return s.repo.GetRuleByID(id)
}

// UpdateRule 更新规则
func (s *ClientFilterService) UpdateRule(rule *model.ClientFilterRule) error {
	if err := s.repo.UpdateRule(rule); err != nil {
		return err
	}
	return s.ReloadCache()
}

// CreateRule 创建规则
func (s *ClientFilterService) CreateRule(rule *model.ClientFilterRule) error {
	if err := s.repo.CreateRule(rule); err != nil {
		return err
	}
	return s.ReloadCache()
}

// DeleteRule 删除规则
func (s *ClientFilterService) DeleteRule(id uint) error {
	if err := s.repo.DeleteRule(id); err != nil {
		return err
	}
	return s.ReloadCache()
}

// SaveConfig 保存配置
func (s *ClientFilterService) SaveConfig(config *model.ClientFilterConfig) error {
	if err := s.repo.SaveConfig(config); err != nil {
		return err
	}
	return s.ReloadCache()
}

// CreateClientType 创建客户端类型
func (s *ClientFilterService) CreateClientType(ct *model.ClientType) error {
	if err := s.repo.CreateClientType(ct); err != nil {
		return err
	}
	return s.ReloadCache()
}

// DeleteClientType 删除客户端类型
func (s *ClientFilterService) DeleteClientType(id uint) error {
	// 先删除该类型下的所有规则
	if err := s.repo.DeleteRulesByClientTypeID(id); err != nil {
		return err
	}
	if err := s.repo.DeleteClientType(id); err != nil {
		return err
	}
	return s.ReloadCache()
}
