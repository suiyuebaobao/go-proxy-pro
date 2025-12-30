package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/pkg/logger"

	"github.com/gin-gonic/gin"
)

// 路由到模块和操作的映射
type RouteMapping struct {
	PathPattern *regexp.Regexp
	Module      string
	Action      string
	GetTargetID func(c *gin.Context) uint
	GetTargetName func(c *gin.Context, body map[string]interface{}) string
	Description func(c *gin.Context, body map[string]interface{}) string
}

var routeMappings []RouteMapping

func init() {
	// 初始化路由映射
	routeMappings = []RouteMapping{
		// 认证
		{regexp.MustCompile(`^/api/auth/login$`), model.ModuleAuth, model.ActionLogin, nil, getLoginUsername, descLogin},
		{regexp.MustCompile(`^/api/auth/register$`), model.ModuleAuth, model.ActionCreate, nil, getRegisterUsername, descRegister},

		// 用户管理
		{regexp.MustCompile(`^/api/admin/users$`), model.ModuleUser, model.ActionCreate, nil, getUserName, descCreateUser},
		{regexp.MustCompile(`^/api/admin/users/(\d+)$`), model.ModuleUser, model.ActionUpdate, getPathID, nil, descUpdateUser},
		{regexp.MustCompile(`^/api/admin/users/(\d+)$`), model.ModuleUser, model.ActionDelete, getPathID, nil, descDeleteUser},
		{regexp.MustCompile(`^/api/admin/users/batch-price-rate$`), model.ModuleUser, model.ActionUpdate, nil, nil, descBatchUpdateRate},
		{regexp.MustCompile(`^/api/admin/users/all-price-rate$`), model.ModuleUser, model.ActionUpdate, nil, nil, descAllUpdateRate},

		// 账户管理
		{regexp.MustCompile(`^/api/admin/accounts$`), model.ModuleAccount, model.ActionCreate, nil, getAccountName, descCreateAccount},
		{regexp.MustCompile(`^/api/admin/accounts/(\d+)$`), model.ModuleAccount, model.ActionUpdate, getPathID, nil, descUpdateAccount},
		{regexp.MustCompile(`^/api/admin/accounts/(\d+)$`), model.ModuleAccount, model.ActionDelete, getPathID, nil, descDeleteAccount},
		{regexp.MustCompile(`^/api/admin/accounts/(\d+)/status$`), model.ModuleAccount, model.ActionUpdate, getPathID, nil, descUpdateAccountStatus},

		// 账户分组
		{regexp.MustCompile(`^/api/admin/account-groups$`), model.ModuleGroup, model.ActionCreate, nil, getGroupName, descCreateGroup},
		{regexp.MustCompile(`^/api/admin/account-groups/(\d+)$`), model.ModuleGroup, model.ActionUpdate, getPathID, nil, descUpdateGroup},
		{regexp.MustCompile(`^/api/admin/account-groups/(\d+)$`), model.ModuleGroup, model.ActionDelete, getPathID, nil, descDeleteGroup},
		{regexp.MustCompile(`^/api/admin/account-groups/(\d+)/accounts$`), model.ModuleGroup, model.ActionUpdate, getPathID, nil, descAddAccountToGroup},
		{regexp.MustCompile(`^/api/admin/account-groups/(\d+)/accounts/(\d+)$`), model.ModuleGroup, model.ActionDelete, getPathID, nil, descRemoveAccountFromGroup},

		// API Key 管理
		{regexp.MustCompile(`^/api/api-keys$`), model.ModuleAPIKey, model.ActionCreate, nil, getAPIKeyName, descCreateAPIKey},
		{regexp.MustCompile(`^/api/api-keys/(\d+)$`), model.ModuleAPIKey, model.ActionUpdate, getPathID, nil, descUpdateAPIKey},
		{regexp.MustCompile(`^/api/api-keys/(\d+)$`), model.ModuleAPIKey, model.ActionDelete, getPathID, nil, descDeleteAPIKey},
		{regexp.MustCompile(`^/api/api-keys/(\d+)/toggle$`), model.ModuleAPIKey, model.ActionUpdate, getPathID, nil, descToggleAPIKey},
		{regexp.MustCompile(`^/api/admin/users/(\d+)/api-keys$`), model.ModuleAPIKey, model.ActionCreate, nil, getAPIKeyName, descAdminCreateAPIKey},
		{regexp.MustCompile(`^/api/admin/users/(\d+)/api-keys/(\d+)$`), model.ModuleAPIKey, model.ActionDelete, getSecondPathID, nil, descAdminDeleteAPIKey},
		{regexp.MustCompile(`^/api/admin/users/(\d+)/api-keys/(\d+)/toggle$`), model.ModuleAPIKey, model.ActionUpdate, getSecondPathID, nil, descAdminToggleAPIKey},

		// 模型管理
		{regexp.MustCompile(`^/api/admin/models$`), model.ModuleModel, model.ActionCreate, nil, getModelName, descCreateModel},
		{regexp.MustCompile(`^/api/admin/models/(\d+)$`), model.ModuleModel, model.ActionUpdate, getPathID, nil, descUpdateModel},
		{regexp.MustCompile(`^/api/admin/models/(\d+)$`), model.ModuleModel, model.ActionDelete, getPathID, nil, descDeleteModel},
		{regexp.MustCompile(`^/api/admin/models/(\d+)/toggle$`), model.ModuleModel, model.ActionUpdate, getPathID, nil, descToggleModel},
		{regexp.MustCompile(`^/api/admin/models/init-defaults$`), model.ModuleModel, model.ActionCreate, nil, nil, descInitModels},
		{regexp.MustCompile(`^/api/admin/models/reset-defaults$`), model.ModuleModel, model.ActionUpdate, nil, nil, descResetModels},

		// 配置管理
		{regexp.MustCompile(`^/api/admin/configs$`), model.ModuleConfig, model.ActionUpdate, nil, nil, descUpdateConfig},
		{regexp.MustCompile(`^/api/admin/configs/sync/trigger$`), model.ModuleConfig, model.ActionSync, nil, nil, descTriggerSync},
		{regexp.MustCompile(`^/api/admin/cache/config$`), model.ModuleCache, model.ActionUpdate, nil, nil, descUpdateCacheConfig},

		// 缓存管理
		{regexp.MustCompile(`^/api/admin/cache/clear$`), model.ModuleCache, model.ActionClear, nil, nil, descClearCache},
		{regexp.MustCompile(`^/api/admin/cache/sessions/(.+)$`), model.ModuleCache, model.ActionDelete, nil, nil, descRemoveSession},
		{regexp.MustCompile(`^/api/admin/cache/users/(\d+)$`), model.ModuleCache, model.ActionClear, getPathID, nil, descClearUserCache},
		{regexp.MustCompile(`^/api/admin/cache/api-keys/(\d+)$`), model.ModuleCache, model.ActionClear, getPathID, nil, descClearAPIKeyCache},
		{regexp.MustCompile(`^/api/admin/accounts/(\d+)/cache/sessions$`), model.ModuleCache, model.ActionClear, getPathID, nil, descClearAccountSessions},
		{regexp.MustCompile(`^/api/admin/accounts/(\d+)/cache/unavailable$`), model.ModuleCache, model.ActionUpdate, getPathID, nil, descMarkAccountUnavailable},
		{regexp.MustCompile(`^/api/admin/accounts/(\d+)/cache/concurrency$`), model.ModuleCache, model.ActionUpdate, getPathID, nil, descSetConcurrency},

		// 代理配置
		{regexp.MustCompile(`^/api/admin/proxy-configs$`), model.ModuleProxy, model.ActionCreate, nil, getProxyName, descCreateProxy},
		{regexp.MustCompile(`^/api/admin/proxy-configs/(\d+)$`), model.ModuleProxy, model.ActionUpdate, getPathID, nil, descUpdateProxy},
		{regexp.MustCompile(`^/api/admin/proxy-configs/(\d+)$`), model.ModuleProxy, model.ActionDelete, getPathID, nil, descDeleteProxy},
		{regexp.MustCompile(`^/api/admin/proxy-configs/(\d+)/toggle$`), model.ModuleProxy, model.ActionUpdate, getPathID, nil, descToggleProxy},
		{regexp.MustCompile(`^/api/admin/proxy-configs/(\d+)/default$`), model.ModuleProxy, model.ActionUpdate, getPathID, nil, descSetDefaultProxy},
		{regexp.MustCompile(`^/api/admin/proxy-configs/default$`), model.ModuleProxy, model.ActionDelete, nil, nil, descClearDefaultProxy},
		{regexp.MustCompile(`^/api/admin/proxy-configs/test$`), model.ModuleProxy, model.ActionTest, nil, nil, descTestProxy},

		// 套餐管理
		{regexp.MustCompile(`^/api/admin/packages$`), model.ModulePackage, model.ActionCreate, nil, getPackageName, descCreatePackage},
		{regexp.MustCompile(`^/api/admin/packages/(\d+)$`), model.ModulePackage, model.ActionUpdate, getPathID, nil, descUpdatePackage},
		{regexp.MustCompile(`^/api/admin/packages/(\d+)$`), model.ModulePackage, model.ActionDelete, getPathID, nil, descDeletePackage},
		{regexp.MustCompile(`^/api/admin/user-packages/user/(\d+)$`), model.ModulePackage, model.ActionCreate, getPathID, nil, descAssignPackage},
		{regexp.MustCompile(`^/api/admin/user-packages/(\d+)$`), model.ModulePackage, model.ActionUpdate, getPathID, nil, descUpdateUserPackage},
		{regexp.MustCompile(`^/api/admin/user-packages/(\d+)$`), model.ModulePackage, model.ActionDelete, getPathID, nil, descDeleteUserPackage},

		// 个人资料
		{regexp.MustCompile(`^/api/profile$`), model.ModuleUser, model.ActionUpdate, nil, nil, descUpdateProfile},
		{regexp.MustCompile(`^/api/profile/password$`), model.ModuleUser, model.ActionUpdate, nil, nil, descChangePassword},

		// OAuth
		{regexp.MustCompile(`^/api/admin/oauth/generate-url$`), model.ModuleAccount, model.ActionCreate, nil, nil, descGenerateOAuthURL},
		{regexp.MustCompile(`^/api/admin/oauth/exchange$`), model.ModuleAccount, model.ActionCreate, nil, nil, descExchangeOAuth},
		{regexp.MustCompile(`^/api/admin/oauth/cookie-auth$`), model.ModuleAccount, model.ActionCreate, nil, nil, descCookieAuth},
	}
}

// 辅助函数
func getPathID(c *gin.Context) uint {
	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 32)
	return uint(id)
}

func getSecondPathID(c *gin.Context) uint {
	// 用于 /users/:id/api-keys/:keyId 这样的路径
	idStr := c.Param("keyId")
	id, _ := strconv.ParseUint(idStr, 10, 32)
	return uint(id)
}

func getLoginUsername(c *gin.Context, body map[string]interface{}) string {
	if username, ok := body["username"].(string); ok {
		return username
	}
	return ""
}

func getRegisterUsername(c *gin.Context, body map[string]interface{}) string {
	if username, ok := body["username"].(string); ok {
		return username
	}
	return ""
}

func getUserName(c *gin.Context, body map[string]interface{}) string {
	if name, ok := body["username"].(string); ok {
		return name
	}
	return ""
}

func getAccountName(c *gin.Context, body map[string]interface{}) string {
	if name, ok := body["name"].(string); ok {
		return name
	}
	return ""
}

func getGroupName(c *gin.Context, body map[string]interface{}) string {
	if name, ok := body["name"].(string); ok {
		return name
	}
	return ""
}

func getAPIKeyName(c *gin.Context, body map[string]interface{}) string {
	if name, ok := body["name"].(string); ok {
		return name
	}
	return ""
}

func getModelName(c *gin.Context, body map[string]interface{}) string {
	if name, ok := body["name"].(string); ok {
		return name
	}
	return ""
}

func getProxyName(c *gin.Context, body map[string]interface{}) string {
	if name, ok := body["name"].(string); ok {
		return name
	}
	return ""
}

func getPackageName(c *gin.Context, body map[string]interface{}) string {
	if name, ok := body["name"].(string); ok {
		return name
	}
	return ""
}

// 描述函数
func descLogin(c *gin.Context, body map[string]interface{}) string {
	return "用户登录"
}

func descRegister(c *gin.Context, body map[string]interface{}) string {
	return "用户注册"
}

func descCreateUser(c *gin.Context, body map[string]interface{}) string {
	if name, ok := body["username"].(string); ok {
		return "创建用户: " + name
	}
	return "创建用户"
}

func descUpdateUser(c *gin.Context, body map[string]interface{}) string {
	return "更新用户 #" + c.Param("id")
}

func descDeleteUser(c *gin.Context, body map[string]interface{}) string {
	return "删除用户 #" + c.Param("id")
}

func descBatchUpdateRate(c *gin.Context, body map[string]interface{}) string {
	return "批量更新用户费率"
}

func descAllUpdateRate(c *gin.Context, body map[string]interface{}) string {
	return "更新所有用户费率"
}

func descCreateAccount(c *gin.Context, body map[string]interface{}) string {
	if name, ok := body["name"].(string); ok {
		return "创建账户: " + name
	}
	return "创建账户"
}

func descUpdateAccount(c *gin.Context, body map[string]interface{}) string {
	return "更新账户 #" + c.Param("id")
}

func descDeleteAccount(c *gin.Context, body map[string]interface{}) string {
	return "删除账户 #" + c.Param("id")
}

func descUpdateAccountStatus(c *gin.Context, body map[string]interface{}) string {
	status := ""
	if s, ok := body["status"].(string); ok {
		status = s
	}
	return "更新账户 #" + c.Param("id") + " 状态为: " + status
}

func descCreateGroup(c *gin.Context, body map[string]interface{}) string {
	if name, ok := body["name"].(string); ok {
		return "创建分组: " + name
	}
	return "创建分组"
}

func descUpdateGroup(c *gin.Context, body map[string]interface{}) string {
	return "更新分组 #" + c.Param("id")
}

func descDeleteGroup(c *gin.Context, body map[string]interface{}) string {
	return "删除分组 #" + c.Param("id")
}

func descAddAccountToGroup(c *gin.Context, body map[string]interface{}) string {
	return "添加账户到分组 #" + c.Param("id")
}

func descRemoveAccountFromGroup(c *gin.Context, body map[string]interface{}) string {
	return "从分组 #" + c.Param("id") + " 移除账户"
}

func descCreateAPIKey(c *gin.Context, body map[string]interface{}) string {
	if name, ok := body["name"].(string); ok {
		return "创建 API Key: " + name
	}
	return "创建 API Key"
}

func descUpdateAPIKey(c *gin.Context, body map[string]interface{}) string {
	return "更新 API Key #" + c.Param("id")
}

func descDeleteAPIKey(c *gin.Context, body map[string]interface{}) string {
	return "删除 API Key #" + c.Param("id")
}

func descToggleAPIKey(c *gin.Context, body map[string]interface{}) string {
	return "切换 API Key #" + c.Param("id") + " 状态"
}

func descAdminCreateAPIKey(c *gin.Context, body map[string]interface{}) string {
	if name, ok := body["name"].(string); ok {
		return "为用户 #" + c.Param("id") + " 创建 API Key: " + name
	}
	return "为用户 #" + c.Param("id") + " 创建 API Key"
}

func descAdminDeleteAPIKey(c *gin.Context, body map[string]interface{}) string {
	return "删除用户 #" + c.Param("id") + " 的 API Key #" + c.Param("keyId")
}

func descAdminToggleAPIKey(c *gin.Context, body map[string]interface{}) string {
	return "切换用户 #" + c.Param("id") + " 的 API Key #" + c.Param("keyId") + " 状态"
}

func descCreateModel(c *gin.Context, body map[string]interface{}) string {
	if name, ok := body["name"].(string); ok {
		return "创建模型: " + name
	}
	return "创建模型"
}

func descUpdateModel(c *gin.Context, body map[string]interface{}) string {
	return "更新模型 #" + c.Param("id")
}

func descDeleteModel(c *gin.Context, body map[string]interface{}) string {
	return "删除模型 #" + c.Param("id")
}

func descToggleModel(c *gin.Context, body map[string]interface{}) string {
	return "切换模型 #" + c.Param("id") + " 状态"
}

func descInitModels(c *gin.Context, body map[string]interface{}) string {
	return "初始化默认模型"
}

func descResetModels(c *gin.Context, body map[string]interface{}) string {
	return "重置默认模型"
}

func descUpdateConfig(c *gin.Context, body map[string]interface{}) string {
	return "更新系统配置"
}

func descTriggerSync(c *gin.Context, body map[string]interface{}) string {
	return "手动触发数据同步"
}

func descUpdateCacheConfig(c *gin.Context, body map[string]interface{}) string {
	return "更新缓存配置"
}

func descClearCache(c *gin.Context, body map[string]interface{}) string {
	if t, ok := body["type"].(string); ok {
		return "清理缓存: " + t
	}
	return "清理缓存"
}

func descRemoveSession(c *gin.Context, body map[string]interface{}) string {
	return "移除会话"
}

func descClearUserCache(c *gin.Context, body map[string]interface{}) string {
	return "清理用户 #" + c.Param("id") + " 缓存"
}

func descClearAPIKeyCache(c *gin.Context, body map[string]interface{}) string {
	return "清理 API Key #" + c.Param("id") + " 缓存"
}

func descClearAccountSessions(c *gin.Context, body map[string]interface{}) string {
	return "清理账户 #" + c.Param("id") + " 会话"
}

func descMarkAccountUnavailable(c *gin.Context, body map[string]interface{}) string {
	return "标记账户 #" + c.Param("id") + " 不可用"
}

func descSetConcurrency(c *gin.Context, body map[string]interface{}) string {
	return "设置账户 #" + c.Param("id") + " 并发限制"
}

func descCreateProxy(c *gin.Context, body map[string]interface{}) string {
	if name, ok := body["name"].(string); ok {
		return "创建代理: " + name
	}
	return "创建代理"
}

func descUpdateProxy(c *gin.Context, body map[string]interface{}) string {
	return "更新代理 #" + c.Param("id")
}

func descDeleteProxy(c *gin.Context, body map[string]interface{}) string {
	return "删除代理 #" + c.Param("id")
}

func descToggleProxy(c *gin.Context, body map[string]interface{}) string {
	return "切换代理 #" + c.Param("id") + " 状态"
}

func descSetDefaultProxy(c *gin.Context, body map[string]interface{}) string {
	return "设置默认代理 #" + c.Param("id")
}

func descClearDefaultProxy(c *gin.Context, body map[string]interface{}) string {
	return "清除默认代理"
}

func descTestProxy(c *gin.Context, body map[string]interface{}) string {
	return "测试代理连通性"
}

func descCreatePackage(c *gin.Context, body map[string]interface{}) string {
	if name, ok := body["name"].(string); ok {
		return "创建套餐: " + name
	}
	return "创建套餐"
}

func descUpdatePackage(c *gin.Context, body map[string]interface{}) string {
	return "更新套餐 #" + c.Param("id")
}

func descDeletePackage(c *gin.Context, body map[string]interface{}) string {
	return "删除套餐 #" + c.Param("id")
}

func descAssignPackage(c *gin.Context, body map[string]interface{}) string {
	return "为用户 #" + c.Param("user_id") + " 分配套餐"
}

func descUpdateUserPackage(c *gin.Context, body map[string]interface{}) string {
	return "更新用户套餐 #" + c.Param("id")
}

func descDeleteUserPackage(c *gin.Context, body map[string]interface{}) string {
	return "删除用户套餐 #" + c.Param("id")
}

func descUpdateProfile(c *gin.Context, body map[string]interface{}) string {
	return "更新个人资料"
}

func descChangePassword(c *gin.Context, body map[string]interface{}) string {
	return "修改密码"
}

func descGenerateOAuthURL(c *gin.Context, body map[string]interface{}) string {
	if platform, ok := body["platform"].(string); ok {
		return "生成 " + platform + " OAuth 授权链接"
	}
	return "生成 OAuth 授权链接"
}

func descExchangeOAuth(c *gin.Context, body map[string]interface{}) string {
	if platform, ok := body["platform"].(string); ok {
		return "交换 " + platform + " OAuth Token"
	}
	return "交换 OAuth Token"
}

func descCookieAuth(c *gin.Context, body map[string]interface{}) string {
	if platform, ok := body["platform"].(string); ok {
		return platform + " Cookie 认证"
	}
	return "Cookie 认证"
}

// 敏感字段脱敏（password 不脱敏，用于安全审计查看登录尝试的密码）
var sensitiveFields = []string{"token", "secret", "api_key", "session_key", "access_token", "refresh_token"}

func sanitizeBody(body map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})
	for k, v := range body {
		lowerK := strings.ToLower(k)
		isSensitive := false
		for _, field := range sensitiveFields {
			if strings.Contains(lowerK, field) {
				isSensitive = true
				break
			}
		}
		if isSensitive {
			sanitized[k] = "******"
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
}

// responseWriter 包装 gin.ResponseWriter 以捕获响应
type responseWriter struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// OperationLogger 操作日志中间件
func OperationLogger() gin.HandlerFunc {
	repo := repository.NewOperationLogRepository()

	return func(c *gin.Context) {
		// 只记录写操作（POST/PUT/DELETE）和登录
		method := c.Request.Method
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			c.Next()
			return
		}

		path := c.Request.URL.Path

		// 跳过代理转发接口
		if strings.HasPrefix(path, "/v1/") || strings.HasPrefix(path, "/gemini/") {
			c.Next()
			return
		}

		// 查找匹配的路由映射
		var mapping *RouteMapping
		for i := range routeMappings {
			m := &routeMappings[i]
			if m.PathPattern.MatchString(path) {
				// 检查方法是否匹配
				if (method == "POST" && (m.Action == model.ActionCreate || m.Action == model.ActionLogin || m.Action == model.ActionSync || m.Action == model.ActionClear || m.Action == model.ActionTest)) ||
					(method == "PUT" && (m.Action == model.ActionUpdate || m.Action == model.ActionEnable || m.Action == model.ActionDisable)) ||
					(method == "DELETE" && (m.Action == model.ActionDelete || m.Action == model.ActionClear)) {
					mapping = m
					break
				}
			}
		}

		if mapping == nil {
			c.Next()
			return
		}

		startTime := time.Now()

		// 读取请求体
		var bodyBytes []byte
		var bodyMap map[string]interface{}
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			json.Unmarshal(bodyBytes, &bodyMap)
		}

		// 包装 ResponseWriter
		rw := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBuffer(nil),
			status:         200,
		}
		c.Writer = rw

		// 处理请求
		c.Next()

		// 获取用户信息
		userID, _ := c.Get("user_id")
		username, _ := c.Get("username")

		userIDUint := uint(0)
		if userID != nil {
			userIDUint = userID.(uint)
		}
		usernameStr := ""
		if username != nil {
			usernameStr = username.(string)
		}

		// 解析响应
		var respData map[string]interface{}
		json.Unmarshal(rw.body.Bytes(), &respData)
		respCode := 0
		respMsg := ""
		if code, ok := respData["code"].(float64); ok {
			respCode = int(code)
		}
		if msg, ok := respData["message"].(string); ok {
			respMsg = msg
		}

		// 构建日志
		opLog := &model.OperationLog{
			UserID:       userIDUint,
			Username:     usernameStr,
			IP:           c.ClientIP(),
			Method:       method,
			Path:         path,
			Module:       mapping.Module,
			Action:       mapping.Action,
			ResponseCode: respCode,
			ResponseMsg:  respMsg,
			Duration:     time.Since(startTime).Milliseconds(),
			UserAgent:    c.Request.UserAgent(),
		}

		// 获取目标ID
		if mapping.GetTargetID != nil {
			opLog.TargetID = mapping.GetTargetID(c)
		}

		// 获取目标名称
		if mapping.GetTargetName != nil {
			opLog.TargetName = mapping.GetTargetName(c, bodyMap)
		}

		// 获取描述
		if mapping.Description != nil {
			opLog.Description = mapping.Description(c, bodyMap)
		}

		// 脱敏后的请求体
		if bodyMap != nil {
			sanitized := sanitizeBody(bodyMap)
			if sanitizedBytes, err := json.Marshal(sanitized); err == nil {
				opLog.RequestBody = string(sanitizedBytes)
			}
		}

		// 写入文件日志
		fileLog := logger.GetLogger("operation")
		result := "成功"
		if respCode != 0 {
			result = "失败"
		}

		// 登录操作时记录密码（用于安全审计）
		if mapping.Action == model.ActionLogin && bodyMap != nil {
			password := ""
			if pwd, ok := bodyMap["password"].(string); ok {
				password = pwd
			}
			fileLog.Info("[%s] %s | User: %s(ID:%d) | IP: %s | %s %s | Target: %s(ID:%d) | Password: %s | Result: %s | Duration: %dms",
				opLog.Module,
				opLog.Description,
				usernameStr,
				userIDUint,
				c.ClientIP(),
				method,
				path,
				opLog.TargetName,
				opLog.TargetID,
				password,
				result,
				opLog.Duration,
			)
		} else {
			fileLog.Info("[%s] %s | User: %s(ID:%d) | IP: %s | %s %s | Target: %s(ID:%d) | Result: %s | Duration: %dms",
				opLog.Module,
				opLog.Description,
				usernameStr,
				userIDUint,
				c.ClientIP(),
				method,
				path,
				opLog.TargetName,
				opLog.TargetID,
				result,
				opLog.Duration,
			)
		}

		// 异步写入数据库
		go repo.Create(opLog)
	}
}
