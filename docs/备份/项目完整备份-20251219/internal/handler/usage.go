package handler

import (
	"strconv"
	"time"

	"go-aiproxy/internal/repository"
	"go-aiproxy/internal/service"
	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

// UsageHandler 使用统计处理器
type UsageHandler struct {
	usageService     *service.UsageService
	pricingService   *service.PricingService
	dailyUsageRepo   *repository.DailyUsageRepository
	usageRecordRepo  *repository.UsageRecordRepository
	userPackageRepo  *repository.UserPackageRepository
}

// NewUsageHandler 创建使用统计处理器
func NewUsageHandler() *UsageHandler {
	return &UsageHandler{
		usageService:     service.NewUsageService(),
		pricingService:   service.NewPricingService(),
		dailyUsageRepo:   repository.NewDailyUsageRepository(),
		usageRecordRepo:  repository.NewUsageRecordRepository(),
		userPackageRepo:  repository.NewUserPackageRepository(),
	}
}

// GetUserUsageSummary 获取用户使用量汇总（用户可见）
func (h *UsageHandler) GetUserUsageSummary(c *gin.Context) {
	userID := c.GetUint("user_id")

	ctx := c.Request.Context()

	// 获取总使用量
	usage, err := h.usageService.GetUserUsage(ctx, userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// 获取总费用
	cost, err := h.usageService.GetUserCost(ctx, userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"total_tokens":                 usage.TotalTokens,
		"input_tokens":                 usage.InputTokens,
		"output_tokens":                usage.OutputTokens,
		"cache_creation_input_tokens":  usage.CacheCreationInputTokens,
		"cache_read_input_tokens":      usage.CacheReadInputTokens,
		"total_requests":               usage.Requests,
		"total_cost":                   cost.TotalCost,
		"input_cost":                   cost.InputCost,
		"output_cost":                  cost.OutputCost,
		"cache_create_cost":            cost.CacheCreateCost,
		"cache_read_cost":              cost.CacheReadCost,
	})
}

// GetUserDailyUsage 获取用户某天的使用量
func (h *UsageHandler) GetUserDailyUsage(c *gin.Context) {
	userID := c.GetUint("user_id")
	date := c.DefaultQuery("date", time.Now().Format("2006-01-02"))

	ctx := c.Request.Context()

	usage, err := h.usageService.GetUserDailyUsage(ctx, userID, date)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	cost, err := h.usageService.GetUserDailyCost(ctx, userID, date)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"date":                         date,
		"total_tokens":                 usage.TotalTokens,
		"input_tokens":                 usage.InputTokens,
		"output_tokens":                usage.OutputTokens,
		"cache_creation_input_tokens":  usage.CacheCreationInputTokens,
		"cache_read_input_tokens":      usage.CacheReadInputTokens,
		"total_requests":               usage.Requests,
		"total_cost":                   cost.TotalCost,
	})
}

// GetUserMonthlyUsage 获取用户某月的使用量
func (h *UsageHandler) GetUserMonthlyUsage(c *gin.Context) {
	userID := c.GetUint("user_id")
	month := c.DefaultQuery("month", time.Now().Format("2006-01"))

	ctx := c.Request.Context()

	usage, err := h.usageService.GetUserMonthlyUsage(ctx, userID, month)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	cost, err := h.usageService.GetUserMonthlyCost(ctx, userID, month)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"month":                        month,
		"total_tokens":                 usage.TotalTokens,
		"input_tokens":                 usage.InputTokens,
		"output_tokens":                usage.OutputTokens,
		"cache_creation_input_tokens":  usage.CacheCreationInputTokens,
		"cache_read_input_tokens":      usage.CacheReadInputTokens,
		"total_requests":               usage.Requests,
		"total_cost":                   cost.TotalCost,
	})
}

// GetUserDailyStats 获取用户日期范围的统计
func (h *UsageHandler) GetUserDailyStats(c *gin.Context) {
	userID := c.GetUint("user_id")

	// 默认最近30天
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	if start := c.Query("start_date"); start != "" {
		if t, err := time.Parse("2006-01-02", start); err == nil {
			startDate = t
		}
	}
	if end := c.Query("end_date"); end != "" {
		if t, err := time.Parse("2006-01-02", end); err == nil {
			endDate = t
		}
	}

	ctx := c.Request.Context()

	stats, err := h.usageService.GetUserDailyStats(ctx, userID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"start_date": startDate.Format("2006-01-02"),
		"end_date":   endDate.Format("2006-01-02"),
		"stats":      stats,
	})
}

// GetUserUsageRecords 获取用户使用记录列表（Redis 优先，MySQL 兜底）
func (h *UsageHandler) GetUserUsageRecords(c *gin.Context) {
	userID := c.GetUint("user_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	source := c.DefaultQuery("source", "auto") // auto, redis, mysql

	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	ctx := c.Request.Context()

	// 根据 source 参数决定数据来源
	if source == "mysql" {
		// 强制从 MySQL 读取
		records, total, err := h.usageRecordRepo.GetByUserID(userID, offset, pageSize)
		if err != nil {
			response.InternalError(c, err.Error())
			return
		}
		response.Success(c, gin.H{
			"items":  records,
			"total":  total,
			"page":   page,
			"page_size": pageSize,
			"source": "mysql",
		})
		return
	}

	// 尝试从 Redis 读取
	records, err := h.usageService.GetUserRecords(ctx, userID, int64(offset), int64(pageSize))
	if err == nil && len(records) > 0 {
		response.Success(c, gin.H{
			"items":  records,
			"total":  len(records), // Redis 不保存总数，返回当前数量
			"page":   page,
			"page_size": pageSize,
			"source": "redis",
		})
		return
	}

	// Redis 无数据，从 MySQL 读取
	mysqlRecords, total, err := h.usageRecordRepo.GetByUserID(userID, offset, pageSize)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"items":  mysqlRecords,
		"total":  total,
		"page":   page,
		"page_size": pageSize,
		"source": "mysql",
	})
}

// GetUserModelStats 获取用户按模型的统计
func (h *UsageHandler) GetUserModelStats(c *gin.Context) {
	userID := c.GetUint("user_id")

	ctx := c.Request.Context()

	stats, err := h.usageService.GetUserModelStats(ctx, userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"models": stats,
	})
}

// GetAPIKeyUsage 获取 API Key 的使用量
func (h *UsageHandler) GetAPIKeyUsage(c *gin.Context) {
	userID := c.GetUint("user_id")
	keyID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid api key id")
		return
	}

	// 验证 API Key 属于当前用户（这里应该在 service 层做验证）
	_ = userID // TODO: 添加权限验证

	ctx := c.Request.Context()

	usage, err := h.usageService.GetAPIKeyUsage(ctx, uint(keyID))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	cost, err := h.usageService.GetAPIKeyCost(ctx, uint(keyID))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"api_key_id":                   keyID,
		"total_tokens":                 usage.TotalTokens,
		"input_tokens":                 usage.InputTokens,
		"output_tokens":                usage.OutputTokens,
		"cache_creation_input_tokens":  usage.CacheCreationInputTokens,
		"cache_read_input_tokens":      usage.CacheReadInputTokens,
		"total_requests":               usage.Requests,
		"total_cost":                   cost.TotalCost,
	})
}

// GetModels 获取所有模型定价（用户可见，用于展示）
func (h *UsageHandler) GetModels(c *gin.Context) {
	ctx := c.Request.Context()

	models, err := h.pricingService.GetAllModels(ctx)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"models": models,
	})
}

// AdminGetUserUsageSummary 管理员获取指定用户的使用量汇总
func (h *UsageHandler) AdminGetUserUsageSummary(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	ctx := c.Request.Context()

	// 获取总使用量
	usage, err := h.usageService.GetUserUsage(ctx, uint(userID))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// 获取总费用
	cost, err := h.usageService.GetUserCost(ctx, uint(userID))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// 获取按模型统计
	modelStats, err := h.usageService.GetUserModelStats(ctx, uint(userID))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// 转换为前端需要的格式
	modelUsage := make([]gin.H, 0, len(modelStats))
	for _, stat := range modelStats {
		modelUsage = append(modelUsage, gin.H{
			"model":    stat.Model,
			"requests": stat.RequestCount,
			"tokens":   stat.TotalTokens,
			"cost":     stat.TotalCost,
		})
	}

	// 获取用户套餐使用情况
	userPackages, _ := h.userPackageRepo.GetByUserID(uint(userID))
	packageUsage := make([]gin.H, 0, len(userPackages))
	for _, up := range userPackages {
		pkgInfo := gin.H{
			"id":     up.ID,
			"name":   up.Name,
			"type":   up.Type,
			"status": up.Status,
		}

		if up.Type == "subscription" {
			pkgInfo["daily_quota"] = up.DailyQuota
			pkgInfo["daily_used"] = up.DailyUsed
			pkgInfo["daily_remain"] = up.DailyQuota - up.DailyUsed
			pkgInfo["weekly_quota"] = up.WeeklyQuota
			pkgInfo["weekly_used"] = up.WeeklyUsed
			pkgInfo["weekly_remain"] = up.WeeklyQuota - up.WeeklyUsed
			pkgInfo["monthly_quota"] = up.MonthlyQuota
			pkgInfo["monthly_used"] = up.MonthlyUsed
			pkgInfo["monthly_remain"] = up.MonthlyQuota - up.MonthlyUsed
			pkgInfo["expire_time"] = up.ExpireTime
		} else if up.Type == "quota" {
			pkgInfo["quota_total"] = up.QuotaTotal
			pkgInfo["quota_used"] = up.QuotaUsed
			pkgInfo["quota_remain"] = up.QuotaTotal - up.QuotaUsed
			pkgInfo["expire_time"] = up.ExpireTime
		}

		packageUsage = append(packageUsage, pkgInfo)
	}

	response.Success(c, gin.H{
		"total_requests": usage.Requests,
		"total_tokens":   usage.TotalTokens,
		"total_cost":     cost.TotalCost,
		"model_usage":    modelUsage,
		"package_usage":  packageUsage,
	})
}

// GetUserDailyStatsRange 获取用户日期范围内的每日统计（用于图表）
func (h *UsageHandler) GetUserDailyStatsRange(c *gin.Context) {
	userID := c.GetUint("user_id")

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		// 默认最近7天
		endDate = time.Now().Format("2006-01-02")
		startDate = time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	}

	ctx := c.Request.Context()

	// 解析日期范围
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		response.BadRequest(c, "invalid start_date format")
		return
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		response.BadRequest(c, "invalid end_date format")
		return
	}

	// 获取每日数据
	dailyData := make([]gin.H, 0)
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		date := d.Format("2006-01-02")
		usage, _ := h.usageService.GetUserDailyUsage(ctx, userID, date)
		cost, _ := h.usageService.GetUserDailyCost(ctx, userID, date)

		dailyData = append(dailyData, gin.H{
			"date":          date,
			"requests":      usage.Requests,
			"input_tokens":  usage.InputTokens,
			"output_tokens": usage.OutputTokens,
			"total_tokens":  usage.TotalTokens,
			"cost":          cost.TotalCost,
		})
	}

	response.Success(c, dailyData)
}

// GetUserUsageSummaryWithToday 获取用户使用量汇总（包含今日费用）
func (h *UsageHandler) GetUserUsageSummaryWithToday(c *gin.Context) {
	userID := c.GetUint("user_id")

	ctx := c.Request.Context()

	// 获取总使用量
	usage, err := h.usageService.GetUserUsage(ctx, userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// 获取总费用
	cost, err := h.usageService.GetUserCost(ctx, userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// 获取今日费用
	today := time.Now().Format("2006-01-02")
	todayCost, _ := h.usageService.GetUserDailyCost(ctx, userID, today)

	response.Success(c, gin.H{
		"total_tokens":   usage.TotalTokens,
		"total_requests": usage.Requests,
		"total_cost":     cost.TotalCost,
		"today_cost":     todayCost.TotalCost,
	})
}

// ========== MySQL 每日汇总查询接口 ==========

// GetUserDailySummaryFromDB 从 MySQL 获取用户每日汇总（用于历史查询）
func (h *UsageHandler) GetUserDailySummaryFromDB(c *gin.Context) {
	userID := c.GetUint("user_id")

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		// 默认最近30天
		endDate = time.Now().Format("2006-01-02")
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}

	summaries, err := h.dailyUsageRepo.GetUserDailySummary(userID, startDate, endDate)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"start_date": startDate,
		"end_date":   endDate,
		"daily":      summaries,
	})
}

// GetUserModelSummaryFromDB 从 MySQL 获取用户模型使用汇总
func (h *UsageHandler) GetUserModelSummaryFromDB(c *gin.Context) {
	userID := c.GetUint("user_id")

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		// 默认全部
		endDate = time.Now().Format("2006-01-02")
		startDate = "2000-01-01"
	}

	summaries, err := h.dailyUsageRepo.GetUserModelSummary(userID, startDate, endDate)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"models": summaries,
	})
}

// GetUserTotalUsageFromDB 从 MySQL 获取用户总使用汇总
func (h *UsageHandler) GetUserTotalUsageFromDB(c *gin.Context) {
	userID := c.GetUint("user_id")

	summary, err := h.dailyUsageRepo.GetUserTotalUsage(userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// 获取今日汇总
	todaySummary, _ := h.dailyUsageRepo.GetTodayUsage(userID)

	response.Success(c, gin.H{
		"total_requests": summary.TotalRequests,
		"total_tokens":   summary.TotalTokens,
		"total_cost":     summary.TotalCost,
		"today_requests": todaySummary.TotalRequests,
		"today_tokens":   todaySummary.TotalTokens,
		"today_cost":     todaySummary.TotalCost,
	})
}

// AdminGetUserDailySummaryFromDB 管理员获取指定用户的每日汇总
func (h *UsageHandler) AdminGetUserDailySummaryFromDB(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		// 默认最近30天
		endDate = time.Now().Format("2006-01-02")
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}

	// 获取每日汇总
	dailySummaries, err := h.dailyUsageRepo.GetUserDailySummary(uint(userID), startDate, endDate)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// 获取模型汇总
	modelSummaries, err := h.dailyUsageRepo.GetUserModelSummary(uint(userID), startDate, endDate)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// 获取总汇总
	totalSummary, err := h.dailyUsageRepo.GetUserTotalUsage(uint(userID))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"start_date":     startDate,
		"end_date":       endDate,
		"daily":          dailySummaries,
		"models":         modelSummaries,
		"total_requests": totalSummary.TotalRequests,
		"total_tokens":   totalSummary.TotalTokens,
		"total_cost":     totalSummary.TotalCost,
	})
}

// AdminGetUserUsageRecords 管理员获取指定用户的使用记录（Redis 优先，MySQL 兜底）
func (h *UsageHandler) AdminGetUserUsageRecords(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	source := c.DefaultQuery("source", "auto") // auto, redis, mysql
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	modelFilter := c.Query("model")

	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	ctx := c.Request.Context()

	// 如果有日期筛选或强制 MySQL，直接从 MySQL 查询
	if source == "mysql" || (startDate != "" && endDate != "") {
		records, total, err := h.usageRecordRepo.GetByUserIDWithFilters(uint(userID), offset, pageSize, startDate, endDate, modelFilter)
		if err != nil {
			response.InternalError(c, err.Error())
			return
		}
		response.Success(c, gin.H{
			"user_id":   userID,
			"items":     records,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"source":    "mysql",
		})
		return
	}

	// 尝试从 Redis 读取
	records, err := h.usageService.GetUserRecords(ctx, uint(userID), int64(offset), int64(pageSize))
	if err == nil && len(records) > 0 {
		// 如果有模型筛选，在内存中过滤
		if modelFilter != "" {
			filtered := make([]service.UsageRecord, 0)
			for _, r := range records {
				if r.Model == modelFilter {
					filtered = append(filtered, r)
				}
			}
			records = filtered
		}

		response.Success(c, gin.H{
			"user_id":   userID,
			"items":     records,
			"total":     len(records),
			"page":      page,
			"page_size": pageSize,
			"source":    "redis",
		})
		return
	}

	// Redis 无数据，从 MySQL 读取
	mysqlRecords, total, err := h.usageRecordRepo.GetByUserIDWithFilters(uint(userID), offset, pageSize, startDate, endDate, modelFilter)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"user_id":   userID,
		"items":     mysqlRecords,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"source":    "mysql",
	})
}

// AdminGetAllUsageSummary 管理员获取所有用户的使用汇总（从 MySQL）
func (h *UsageHandler) AdminGetAllUsageSummary(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	if startDate == "" || endDate == "" {
		endDate = time.Now().Format("2006-01-02")
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}

	// 获取总汇总
	totalSummary, err := h.dailyUsageRepo.GetTotalSummary(startDate, endDate)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// 获取每日汇总
	dailySummaries, err := h.dailyUsageRepo.GetDailySummaryAll(startDate, endDate)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// 获取模型汇总
	modelSummaries, err := h.dailyUsageRepo.GetModelSummary(startDate, endDate)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"start_date":     startDate,
		"end_date":       endDate,
		"total_requests": totalSummary.TotalRequests,
		"total_tokens":   totalSummary.TotalTokens,
		"total_cost":     totalSummary.TotalCost,
		"daily":          dailySummaries,
		"models":         modelSummaries,
	})
}
