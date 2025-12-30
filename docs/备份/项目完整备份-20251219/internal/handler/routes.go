package handler

import (
	"go-aiproxy/internal/middleware"
	"go-aiproxy/internal/repository"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 全局操作日志中间件（放在认证之后，记录所有写操作）
	r.Use(middleware.OperationLogger())

	// 公开接口
	userHandler := NewUserHandler()
	accountHandler := NewAccountHandler()
	proxyHandler := NewProxyHandler()
	requestLogHandler := NewRequestLogHandler()
	oauthHandler := NewOAuthHandler()
	usageHandler := NewUsageHandler()
	cacheHandler := NewCacheHandler()
	operationLogHandler := NewOperationLogHandler()

	// 模型管理
	modelRepo := repository.NewAIModelRepository(repository.GetDB())
	modelHandler := NewAIModelHandler(modelRepo)
	// 初始化默认模型
	modelRepo.InitDefaultModels()

	captchaHandler := NewCaptchaHandler()

	auth := r.Group("/api/auth")
	{
		auth.GET("/captcha", captchaHandler.Generate)
		auth.GET("/captcha/status", captchaHandler.GetStatus)
		auth.POST("/login", userHandler.Login)
		auth.POST("/register", userHandler.Register)
	}

	// ========== 代理转发接口 (需要 API Key 认证) ==========
	proxyGroup := r.Group("")
	proxyGroup.Use(middleware.APIKeyAuth())
	proxyGroup.Use(middleware.ClientFilter())           // 客户端过滤
	proxyGroup.Use(middleware.CheckAllowedClients())    // API Key 客户端限制检查
	proxyGroup.Use(middleware.UserConcurrencyControl()) // 用户并发控制
	{
		// OpenAI 兼容接口
		proxyGroup.POST("/v1/chat/completions", proxyHandler.ChatCompletions)

		// Claude 兼容接口 (支持两种路径)
		proxyGroup.POST("/v1/messages", proxyHandler.Messages)
		proxyGroup.POST("/api/v1/messages", proxyHandler.Messages)

		// Gemini 原生格式接口
		proxyGroup.POST("/gemini/v1/chat", proxyHandler.GeminiChat)
	}

	// API Key Handler
	apiKeyHandler := NewAPIKeyHandler()

	// 需要认证的接口
	api := r.Group("/api")
	api.Use(middleware.JWTAuth())
	{
		// 用户个人接口
		profile := api.Group("/profile")
		{
			profile.GET("", userHandler.GetProfile)
			profile.PUT("", userHandler.UpdateProfile)
			profile.PUT("/password", userHandler.ChangePassword)
		}

		// 用户 API Key 管理
		apiKeys := api.Group("/api-keys")
		{
			apiKeys.GET("", apiKeyHandler.List)
			apiKeys.POST("", apiKeyHandler.Create)
			apiKeys.GET("/:id", apiKeyHandler.Get)
			apiKeys.PUT("/:id", apiKeyHandler.Update)
			apiKeys.DELETE("/:id", apiKeyHandler.Delete)
			apiKeys.PUT("/:id/toggle", apiKeyHandler.ToggleStatus)
			apiKeys.GET("/:id/usage", usageHandler.GetAPIKeyUsage) // API Key 使用统计
		}

		// 用户使用统计（用户只能看到自己的，只能看费用不能看倍率）
		usage := api.Group("/usage")
		{
			usage.GET("/summary", usageHandler.GetUserUsageSummaryWithToday) // 总使用量汇总（含今日）
			usage.GET("/daily", usageHandler.GetUserDailyStatsRange)         // 日期范围每日统计
			usage.GET("/monthly", usageHandler.GetUserMonthlyUsage)          // 某月使用量
			usage.GET("/stats", usageHandler.GetUserDailyStats)              // 日期范围统计
			usage.GET("/records", usageHandler.GetUserUsageRecords)          // 使用记录列表
			usage.GET("/models", usageHandler.GetUserModelStats)             // 按模型统计

			// MySQL 持久化数据查询（历史汇总）
			usage.GET("/db/summary", usageHandler.GetUserTotalUsageFromDB)    // 从 MySQL 获取总汇总
			usage.GET("/db/daily", usageHandler.GetUserDailySummaryFromDB)    // 从 MySQL 获取每日汇总
			usage.GET("/db/models", usageHandler.GetUserModelSummaryFromDB)   // 从 MySQL 获取模型汇总
		}

		// 模型价格查询（用户可见）
		api.GET("/models", usageHandler.GetModels)

		// 用户套餐查询
		packageHandler := NewPackageHandler()
		api.GET("/packages", packageHandler.GetAvailablePackages)       // 可购买的套餐
		api.GET("/my-packages", packageHandler.GetMyPackages)           // 我的所有套餐
		api.GET("/my-packages/active", packageHandler.GetMyActivePackages) // 我的有效套餐

		// 管理员接口
		admin := api.Group("/admin")
		admin.Use(middleware.AdminRequired())
		{
			// 用户管理
			users := admin.Group("/users")
			{
				users.GET("", userHandler.List)
				users.POST("", userHandler.Create)                                    // 创建用户
				users.GET("/all", userHandler.ListAllUsers)                       // 获取所有用户（不分页）
				users.GET("/:id", userHandler.Get)
				users.PUT("/:id", userHandler.Update)
				users.DELETE("/:id", userHandler.Delete)
				users.POST("/batch-price-rate", userHandler.BatchUpdatePriceRate) // 批量更新费率
				users.POST("/all-price-rate", userHandler.UpdateAllPriceRate)     // 全部更新费率
				users.GET("/:id/usage/summary", usageHandler.AdminGetUserUsageSummary) // 用户使用统计
				users.GET("/:id/usage/db", usageHandler.AdminGetUserDailySummaryFromDB)    // 用户使用统计（MySQL）
				users.GET("/:id/usage/records", usageHandler.AdminGetUserUsageRecords)     // 用户使用记录（Redis）
				users.GET("/:id/concurrency", cacheHandler.GetUserConcurrency)             // 获取用户并发信息
				users.DELETE("/:id/concurrency", cacheHandler.ResetUserConcurrency)        // 重置用户并发计数
				// 用户 API Key 管理
				users.GET("/:id/api-keys", apiKeyHandler.AdminList)                  // 获取用户的 API Key 列表
				users.POST("/:id/api-keys", apiKeyHandler.AdminCreate)               // 为用户创建 API Key
				users.DELETE("/:id/api-keys/:keyId", apiKeyHandler.AdminDelete)      // 删除 API Key
				users.PUT("/:id/api-keys/:keyId/toggle", apiKeyHandler.AdminToggleStatus) // 切换 API Key 状态
			}

			// API Key 管理（所有用户的）
			adminAPIKeys := admin.Group("/api-keys")
			{
				adminAPIKeys.GET("", apiKeyHandler.AdminListAll)           // 获取所有 API Key
				adminAPIKeys.GET("/:id/logs", apiKeyHandler.AdminGetAPIKeyLogs) // 获取 API Key 使用日志
			}

			// 账户管理
			accounts := admin.Group("/accounts")
			{
				accounts.GET("/types", accountHandler.GetTypes)
				accounts.GET("", accountHandler.List)
				accounts.POST("", accountHandler.Create)
				accounts.GET("/:id", accountHandler.Get)
				accounts.PUT("/:id", accountHandler.Update)
				accounts.DELETE("/:id", accountHandler.Delete)
				accounts.PUT("/:id/status", accountHandler.UpdateStatus)
			}

			// 账户分组管理
			groups := admin.Group("/account-groups")
			{
				groups.GET("", accountHandler.ListGroups)
				groups.GET("/all", accountHandler.GetAllGroups)
				groups.POST("", accountHandler.CreateGroup)
				groups.GET("/:id", accountHandler.GetGroup)
				groups.PUT("/:id", accountHandler.UpdateGroup)
				groups.DELETE("/:id", accountHandler.DeleteGroup)
				groups.POST("/:id/accounts", accountHandler.AddAccountToGroup)
				groups.DELETE("/:id/accounts/:accountId", accountHandler.RemoveAccountFromGroup)
			}

			// OAuth 授权
			oauth := admin.Group("/oauth")
			{
				oauth.POST("/generate-url", oauthHandler.GenerateURL)
				oauth.POST("/exchange", oauthHandler.Exchange)
				oauth.POST("/cookie-auth", oauthHandler.CookieAuth)
			}

			// 请求日志
			logs := admin.Group("/logs")
			{
				logs.GET("", requestLogHandler.List)
				logs.GET("/summary", requestLogHandler.GetSummary)
				logs.GET("/account-load", requestLogHandler.GetAccountLoadStats)
				logs.GET("/usage-summary", usageHandler.AdminGetAllUsageSummary) // 所有用户使用汇总（MySQL）
			}

			// 操作日志
			opLogs := admin.Group("/operation-logs")
			{
				opLogs.GET("", operationLogHandler.List)
				opLogs.GET("/stats", operationLogHandler.GetStats)
				opLogs.GET("/:id", operationLogHandler.Get)
				opLogs.DELETE("/cleanup", operationLogHandler.Cleanup)
			}

			// 模型管理
			models := admin.Group("/models")
			{
				models.GET("", modelHandler.List)
				models.GET("/platforms", modelHandler.GetPlatforms)
				models.POST("", modelHandler.Create)
				models.GET("/:id", modelHandler.Get)
				models.PUT("/:id", modelHandler.Update)
				models.DELETE("/:id", modelHandler.Delete)
				models.PUT("/:id/toggle", modelHandler.ToggleEnabled)
				models.POST("/init-defaults", modelHandler.InitDefaults)
				models.POST("/reset-defaults", modelHandler.ResetDefaults)
			}

			// 代理测试（管理员内部测试，不需要API Key）
			admin.POST("/proxy/test", proxyHandler.TestProxy)

			// 缓存管理
			cache := admin.Group("/cache")
			{
				cache.GET("/stats", cacheHandler.GetStats)                          // 获取缓存统计
				cache.GET("/sessions", cacheHandler.ListSessions)                    // 列出所有会话
				cache.DELETE("/sessions/:sessionId", cacheHandler.RemoveSession)     // 移除会话
				cache.GET("/accounts", cacheHandler.ListAccountsCache)               // 列出有缓存的账号（聚合）
				cache.GET("/users", cacheHandler.ListUsersCache)                     // 列出有缓存的用户（聚合）
				cache.GET("/unavailable", cacheHandler.ListUnavailableAccounts)      // 列出不可用账户
				cache.POST("/clear", cacheHandler.ClearCache)                        // 按类型清理缓存
				cache.DELETE("/users/:id", cacheHandler.ClearUserCache)              // 清理用户缓存
				cache.DELETE("/api-keys/:id", cacheHandler.ClearAPIKeyCache)         // 清理 API Key 缓存
				cache.GET("/config", cacheHandler.GetCacheConfig)                    // 获取缓存配置
				cache.PUT("/config", cacheHandler.UpdateCacheConfig)                 // 更新缓存配置
			}

			// 账户缓存管理（并发控制和不可用标记）
			accountCache := admin.Group("/accounts/:id/cache")
			{
				accountCache.DELETE("/sessions", cacheHandler.ClearAccountSessions)       // 清除账户会话
				accountCache.POST("/unavailable", cacheHandler.MarkAccountUnavailable)    // 标记账户不可用
				accountCache.DELETE("/unavailable", cacheHandler.ClearAccountUnavailable) // 清除不可用标记
				accountCache.GET("/concurrency", cacheHandler.GetAccountConcurrency)      // 获取并发信息
				accountCache.PUT("/concurrency", cacheHandler.SetAccountConcurrencyLimit) // 设置并发限制
				accountCache.DELETE("/concurrency", cacheHandler.ResetAccountConcurrency) // 重置并发计数
			}

			// 系统配置管理
			configHandler := NewConfigHandler()
			configs := admin.Group("/configs")
			{
				configs.GET("", configHandler.GetAll)                      // 获取所有配置
				configs.GET("/category/:category", configHandler.GetByCategory) // 获取分类配置
				configs.PUT("", configHandler.Update)                      // 更新配置
				configs.GET("/sync/status", configHandler.GetSyncStatus)   // 获取同步状态
				configs.POST("/sync/trigger", configHandler.TriggerSync)   // 手动触发同步
			}

			// 套餐管理
			adminPkgHandler := NewPackageHandler()
			packages := admin.Group("/packages")
			{
				packages.GET("", adminPkgHandler.ListPackages)           // 获取所有套餐
				packages.POST("", adminPkgHandler.CreatePackage)         // 创建套餐
				packages.PUT("/:id", adminPkgHandler.UpdatePackage)      // 更新套餐
				packages.DELETE("/:id", adminPkgHandler.DeletePackage)   // 删除套餐
			}

			// 用户套餐管理
			userPackages := admin.Group("/user-packages")
			{
				userPackages.GET("/user/:user_id", adminPkgHandler.ListUserPackages)  // 获取用户的套餐
				userPackages.POST("/user/:user_id", adminPkgHandler.AssignPackage)    // 给用户分配套餐
				userPackages.PUT("/:id", adminPkgHandler.UpdateUserPackage)           // 更新用户套餐
				userPackages.DELETE("/:id", adminPkgHandler.DeleteUserPackage)        // 删除用户套餐
			}

			// 代理配置管理
			proxyConfigs := admin.Group("/proxy-configs")
			{
				proxyConfigs.GET("", ListProxyConfigs)              // 获取代理列表
				proxyConfigs.GET("/enabled", GetEnabledProxyConfigs) // 获取启用的代理（用于下拉选择）
				proxyConfigs.GET("/default", GetDefaultProxyConfig)  // 获取默认代理
				proxyConfigs.DELETE("/default", ClearDefaultProxyConfig) // 清除默认代理
				proxyConfigs.POST("", CreateProxyConfig)            // 创建代理
				proxyConfigs.POST("/test", TestProxyConnectivity)   // 测试代理连通性
				proxyConfigs.GET("/:id", GetProxyConfig)            // 获取单个代理
				proxyConfigs.PUT("/:id", UpdateProxyConfig)         // 更新代理
				proxyConfigs.DELETE("/:id", DeleteProxyConfig)      // 删除代理
				proxyConfigs.PUT("/:id/toggle", ToggleProxyConfigEnabled) // 切换启用状态
				proxyConfigs.PUT("/:id/default", SetDefaultProxyConfig)   // 设置为默认代理
			}

			// 系统监控
			monitorHandler := NewSystemMonitorHandler()
			monitor := admin.Group("/monitor")
			{
				monitor.GET("", monitorHandler.GetMonitorData)              // 获取完整监控数据
				monitor.GET("/system", monitorHandler.GetSystemStats)       // 系统资源
				monitor.GET("/redis", monitorHandler.GetRedisStats)         // Redis 统计
				monitor.GET("/mysql", monitorHandler.GetMySQLStats)         // MySQL 统计
				monitor.GET("/accounts", monitorHandler.GetAccountStats)    // 账号统计
				monitor.GET("/users", monitorHandler.GetUserStats)          // 用户统计
				monitor.GET("/today", monitorHandler.GetTodayUsageStats)    // 今日使用统计
			}

			// 错误消息管理
			errorMsgHandler := NewErrorMessageHandler()
			errorMessages := admin.Group("/error-messages")
			{
				errorMessages.GET("", errorMsgHandler.List)
				errorMessages.GET("/code/:code", errorMsgHandler.GetByCode)
				errorMessages.GET("/:id", errorMsgHandler.Get)
				errorMessages.POST("", errorMsgHandler.Create)
				errorMessages.PUT("/:id", errorMsgHandler.Update)
				errorMessages.DELETE("/:id", errorMsgHandler.Delete)
				errorMessages.PUT("/:id/toggle", errorMsgHandler.ToggleEnabled)
				errorMessages.POST("/init", errorMsgHandler.InitDefault)
				errorMessages.POST("/refresh", errorMsgHandler.RefreshCache)
				errorMessages.PUT("/enable-all", errorMsgHandler.EnableAll)
				errorMessages.PUT("/disable-all", errorMsgHandler.DisableAll)
			}

			// 系统日志查看
			systemLogHandler := NewSystemLogHandler()
			sysLogs := admin.Group("/system-logs")
			{
				sysLogs.GET("/files", systemLogHandler.ListFiles)       // 获取日志文件列表
				sysLogs.GET("/read", systemLogHandler.ReadFile)         // 读取日志内容
				sysLogs.GET("/tail", systemLogHandler.TailFile)         // 查看日志末尾
				sysLogs.GET("/download", systemLogHandler.DownloadFile) // 下载日志文件
				sysLogs.DELETE("/file", systemLogHandler.DeleteFile)    // 删除日志文件
			}

			// 客户端过滤管理
			clientFilterHandler := NewClientFilterHandler()
			clientFilter := admin.Group("/client-filter")
			{
				// 全局配置
				clientFilter.GET("/config", clientFilterHandler.GetConfig)
				clientFilter.PUT("/config", clientFilterHandler.UpdateConfig)
				clientFilter.POST("/reload", clientFilterHandler.ReloadCache)
				clientFilter.POST("/test", clientFilterHandler.TestValidation)

				// 客户端类型管理
				clientTypes := clientFilter.Group("/client-types")
				{
					clientTypes.GET("", clientFilterHandler.ListClientTypes)
					clientTypes.POST("", clientFilterHandler.CreateClientType)
					clientTypes.GET("/:id", clientFilterHandler.GetClientType)
					clientTypes.PUT("/:id", clientFilterHandler.UpdateClientType)
					clientTypes.DELETE("/:id", clientFilterHandler.DeleteClientType)
					clientTypes.PUT("/:id/toggle", clientFilterHandler.ToggleClientType)
				}

				// 过滤规则管理
				rules := clientFilter.Group("/rules")
				{
					rules.GET("", clientFilterHandler.ListRules)
					rules.POST("", clientFilterHandler.CreateRule)
					rules.GET("/:id", clientFilterHandler.GetRule)
					rules.PUT("/:id", clientFilterHandler.UpdateRule)
					rules.DELETE("/:id", clientFilterHandler.DeleteRule)
					rules.PUT("/:id/toggle", clientFilterHandler.ToggleRule)
				}
			}
		}
	}

	// ========== OpenAI Responses API 路由 ==========
	// 注意：必须在静态文件处理之前注册，否则会被静态文件处理器拦截
	// 参考 claude-relay: router.post('/responses', authenticateApiKey, handleResponses)
	openaiResponsesHandler := NewOpenAIResponsesHandler()
	responsesGroup := r.Group("")
	responsesGroup.Use(middleware.APIKeyAuth())
	responsesGroup.Use(middleware.ClientFilter())
	responsesGroup.Use(middleware.CheckAllowedClients())
	responsesGroup.Use(middleware.UserConcurrencyControl())
	{
		// OpenAI Responses API 端点
		responsesGroup.POST("/responses", openaiResponsesHandler.HandleResponses)
		responsesGroup.POST("/v1/responses", openaiResponsesHandler.HandleResponses)
		responsesGroup.POST("/responses/compact", openaiResponsesHandler.HandleResponses)
		responsesGroup.POST("/v1/responses/compact", openaiResponsesHandler.HandleResponses)
	}

	// 静态文件（前端）- 放在最后作为兜底
	RegisterStatic(r)
}
