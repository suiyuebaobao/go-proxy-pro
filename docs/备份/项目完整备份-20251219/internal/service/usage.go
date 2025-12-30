package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"

	"github.com/redis/go-redis/v9"
)

// UsageService 使用统计服务
type UsageService struct {
	rdb *redis.Client
}

func NewUsageService() *UsageService {
	return &UsageService{
		rdb: repository.RDB,
	}
}

// UsageData Token 使用数据
type UsageData struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
	TotalTokens              int64 `json:"total_tokens"`
	Requests                 int64 `json:"requests"`
}

// CostData 费用数据
type CostData struct {
	InputCost       float64 `json:"input_cost"`
	OutputCost      float64 `json:"output_cost"`
	CacheCreateCost float64 `json:"cache_create_cost"`
	CacheReadCost   float64 `json:"cache_read_cost"`
	TotalCost       float64 `json:"total_cost"`
}

// UsageRecord 单次请求的使用记录
type UsageRecord struct {
	ID                       string    `json:"id"`
	Model                    string    `json:"model"`
	RequestIP                string    `json:"request_ip"`
	InputTokens              int       `json:"input_tokens"`
	OutputTokens             int       `json:"output_tokens"`
	CacheCreationInputTokens int       `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int       `json:"cache_read_input_tokens"`
	TotalTokens              int       `json:"total_tokens"`
	TotalCost                float64   `json:"total_cost"`
	Timestamp                time.Time `json:"timestamp"`
}

// Redis Key 前缀
const (
	// Token 使用量 Keys
	keyUsageTotal      = "usage:total:%d"      // usage:total:{userID}
	keyUsageDaily      = "usage:daily:%d:%s"   // usage:daily:{userID}:{date}
	keyUsageMonthly    = "usage:monthly:%d:%s" // usage:monthly:{userID}:{month}
	keyUsageByKey      = "usage:key:%d"        // usage:key:{apiKeyID}
	keyUsageKeyDaily   = "usage:key:daily:%d:%s"
	keyUsageKeyMonthly = "usage:key:monthly:%d:%s"

	// 费用 Keys
	keyCostTotal      = "cost:total:%d"      // cost:total:{userID}
	keyCostDaily      = "cost:daily:%d:%s"   // cost:daily:{userID}:{date}
	keyCostMonthly    = "cost:monthly:%d:%s" // cost:monthly:{userID}:{month}
	keyCostByKey      = "cost:key:%d"        // cost:key:{apiKeyID}
	keyCostKeyDaily   = "cost:key:daily:%d:%s"
	keyCostKeyMonthly = "cost:key:monthly:%d:%s"

	// 账户费用 Keys
	keyCostAccount      = "cost:account:%d"      // cost:account:{accountID} 账户总费用
	keyCostAccountDaily = "cost:account:daily:%d:%s" // cost:account:daily:{accountID}:{date}

	// 使用记录 Keys
	keyUsageRecords    = "usage:records:%d"     // usage:records:{userID} (list)
	keyUsageRecordsKey = "usage:records:key:%d" // usage:records:key:{apiKeyID} (list)

	// 模型统计 Keys
	keyModelUsage = "usage:model:%d:%s" // usage:model:{userID}:{model}

	// 过期时间
	dailyExpiration   = 90 * 24 * time.Hour  // 90天
	monthlyExpiration = 365 * 24 * time.Hour // 365天
	recordExpiration  = 30 * 24 * time.Hour  // 30天
	maxRecords        = 1000                 // 最多保留1000条记录
)

// IncrementUsage 增加使用量（原子操作）
func (s *UsageService) IncrementUsage(ctx context.Context, userID, apiKeyID uint, data *UsageData) error {
	now := time.Now()
	date := now.Format("2006-01-02")
	month := now.Format("2006-01")

	pipe := s.rdb.Pipeline()

	// 用户级别统计
	userTotalKey := fmt.Sprintf(keyUsageTotal, userID)
	userDailyKey := fmt.Sprintf(keyUsageDaily, userID, date)
	userMonthlyKey := fmt.Sprintf(keyUsageMonthly, userID, month)

	s.incrUsageHash(pipe, userTotalKey, data)
	s.incrUsageHash(pipe, userDailyKey, data)
	s.incrUsageHash(pipe, userMonthlyKey, data)

	// 设置过期时间
	pipe.Expire(ctx, userDailyKey, dailyExpiration)
	pipe.Expire(ctx, userMonthlyKey, monthlyExpiration)

	// API Key 级别统计
	if apiKeyID > 0 {
		keyTotalKey := fmt.Sprintf(keyUsageByKey, apiKeyID)
		keyDailyKey := fmt.Sprintf(keyUsageKeyDaily, apiKeyID, date)
		keyMonthlyKey := fmt.Sprintf(keyUsageKeyMonthly, apiKeyID, month)

		s.incrUsageHash(pipe, keyTotalKey, data)
		s.incrUsageHash(pipe, keyDailyKey, data)
		s.incrUsageHash(pipe, keyMonthlyKey, data)

		pipe.Expire(ctx, keyDailyKey, dailyExpiration)
		pipe.Expire(ctx, keyMonthlyKey, monthlyExpiration)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// incrUsageHash 在 pipeline 中增加 hash 字段
func (s *UsageService) incrUsageHash(pipe redis.Pipeliner, key string, data *UsageData) {
	ctx := context.Background()
	pipe.HIncrBy(ctx, key, "input_tokens", data.InputTokens)
	pipe.HIncrBy(ctx, key, "output_tokens", data.OutputTokens)
	pipe.HIncrBy(ctx, key, "cache_creation_input_tokens", data.CacheCreationInputTokens)
	pipe.HIncrBy(ctx, key, "cache_read_input_tokens", data.CacheReadInputTokens)
	pipe.HIncrBy(ctx, key, "total_tokens", data.TotalTokens)
	pipe.HIncrBy(ctx, key, "requests", data.Requests)
}

// IncrementCost 增加费用（原子操作）
func (s *UsageService) IncrementCost(ctx context.Context, userID, apiKeyID uint, data *CostData) error {
	now := time.Now()
	date := now.Format("2006-01-02")
	month := now.Format("2006-01")

	pipe := s.rdb.Pipeline()

	// 用户级别统计
	userTotalKey := fmt.Sprintf(keyCostTotal, userID)
	userDailyKey := fmt.Sprintf(keyCostDaily, userID, date)
	userMonthlyKey := fmt.Sprintf(keyCostMonthly, userID, month)

	s.incrCostHash(pipe, userTotalKey, data)
	s.incrCostHash(pipe, userDailyKey, data)
	s.incrCostHash(pipe, userMonthlyKey, data)

	pipe.Expire(ctx, userDailyKey, dailyExpiration)
	pipe.Expire(ctx, userMonthlyKey, monthlyExpiration)

	// API Key 级别统计
	if apiKeyID > 0 {
		keyTotalKey := fmt.Sprintf(keyCostByKey, apiKeyID)
		keyDailyKey := fmt.Sprintf(keyCostKeyDaily, apiKeyID, date)
		keyMonthlyKey := fmt.Sprintf(keyCostKeyMonthly, apiKeyID, month)

		s.incrCostHash(pipe, keyTotalKey, data)
		s.incrCostHash(pipe, keyDailyKey, data)
		s.incrCostHash(pipe, keyMonthlyKey, data)

		pipe.Expire(ctx, keyDailyKey, dailyExpiration)
		pipe.Expire(ctx, keyMonthlyKey, monthlyExpiration)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// incrCostHash 在 pipeline 中增加费用 hash 字段（使用整数存储，精度为0.000001）
func (s *UsageService) incrCostHash(pipe redis.Pipeliner, key string, data *CostData) {
	ctx := context.Background()
	// 将浮点数转换为整数存储（乘以1000000），避免浮点数精度问题
	pipe.HIncrBy(ctx, key, "input_cost", int64(data.InputCost*1000000))
	pipe.HIncrBy(ctx, key, "output_cost", int64(data.OutputCost*1000000))
	pipe.HIncrBy(ctx, key, "cache_create_cost", int64(data.CacheCreateCost*1000000))
	pipe.HIncrBy(ctx, key, "cache_read_cost", int64(data.CacheReadCost*1000000))
	pipe.HIncrBy(ctx, key, "total_cost", int64(data.TotalCost*1000000))
}

// AddUsageRecord 添加使用记录
func (s *UsageService) AddUsageRecord(ctx context.Context, userID, apiKeyID uint, record *UsageRecord) error {
	recordJSON, err := json.Marshal(record)
	if err != nil {
		return err
	}

	pipe := s.rdb.Pipeline()

	// 用户记录
	userRecordsKey := fmt.Sprintf(keyUsageRecords, userID)
	pipe.LPush(ctx, userRecordsKey, recordJSON)
	pipe.LTrim(ctx, userRecordsKey, 0, maxRecords-1)
	pipe.Expire(ctx, userRecordsKey, recordExpiration)

	// API Key 记录
	if apiKeyID > 0 {
		keyRecordsKey := fmt.Sprintf(keyUsageRecordsKey, apiKeyID)
		pipe.LPush(ctx, keyRecordsKey, recordJSON)
		pipe.LTrim(ctx, keyRecordsKey, 0, maxRecords-1)
		pipe.Expire(ctx, keyRecordsKey, recordExpiration)
	}

	_, err = pipe.Exec(ctx)
	return err
}

// GetUserUsage 获取用户总使用量
func (s *UsageService) GetUserUsage(ctx context.Context, userID uint) (*UsageData, error) {
	key := fmt.Sprintf(keyUsageTotal, userID)
	return s.getUsageData(ctx, key)
}

// GetUserDailyUsage 获取用户某天的使用量
func (s *UsageService) GetUserDailyUsage(ctx context.Context, userID uint, date string) (*UsageData, error) {
	key := fmt.Sprintf(keyUsageDaily, userID, date)
	return s.getUsageData(ctx, key)
}

// GetUserMonthlyUsage 获取用户某月的使用量
func (s *UsageService) GetUserMonthlyUsage(ctx context.Context, userID uint, month string) (*UsageData, error) {
	key := fmt.Sprintf(keyUsageMonthly, userID, month)
	return s.getUsageData(ctx, key)
}

// GetUserCost 获取用户总费用
func (s *UsageService) GetUserCost(ctx context.Context, userID uint) (*CostData, error) {
	key := fmt.Sprintf(keyCostTotal, userID)
	return s.getCostData(ctx, key)
}

// GetUserDailyCost 获取用户某天的费用
func (s *UsageService) GetUserDailyCost(ctx context.Context, userID uint, date string) (*CostData, error) {
	key := fmt.Sprintf(keyCostDaily, userID, date)
	return s.getCostData(ctx, key)
}

// GetUserMonthlyCost 获取用户某月的费用
func (s *UsageService) GetUserMonthlyCost(ctx context.Context, userID uint, month string) (*CostData, error) {
	key := fmt.Sprintf(keyCostMonthly, userID, month)
	return s.getCostData(ctx, key)
}

// GetAPIKeyUsage 获取 API Key 的使用量
func (s *UsageService) GetAPIKeyUsage(ctx context.Context, apiKeyID uint) (*UsageData, error) {
	key := fmt.Sprintf(keyUsageByKey, apiKeyID)
	return s.getUsageData(ctx, key)
}

// GetAPIKeyCost 获取 API Key 的费用
func (s *UsageService) GetAPIKeyCost(ctx context.Context, apiKeyID uint) (*CostData, error) {
	key := fmt.Sprintf(keyCostByKey, apiKeyID)
	return s.getCostData(ctx, key)
}

// GetUserRecords 获取用户使用记录
func (s *UsageService) GetUserRecords(ctx context.Context, userID uint, offset, limit int64) ([]UsageRecord, error) {
	key := fmt.Sprintf(keyUsageRecords, userID)
	return s.getRecords(ctx, key, offset, limit)
}

// GetAPIKeyRecords 获取 API Key 使用记录
func (s *UsageService) GetAPIKeyRecords(ctx context.Context, apiKeyID uint, offset, limit int64) ([]UsageRecord, error) {
	key := fmt.Sprintf(keyUsageRecordsKey, apiKeyID)
	return s.getRecords(ctx, key, offset, limit)
}

// getUsageData 从 Redis Hash 获取使用量数据
func (s *UsageService) getUsageData(ctx context.Context, key string) (*UsageData, error) {
	result, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	data := &UsageData{}
	if v, ok := result["input_tokens"]; ok {
		data.InputTokens, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := result["output_tokens"]; ok {
		data.OutputTokens, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := result["cache_creation_input_tokens"]; ok {
		data.CacheCreationInputTokens, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := result["cache_read_input_tokens"]; ok {
		data.CacheReadInputTokens, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := result["total_tokens"]; ok {
		data.TotalTokens, _ = strconv.ParseInt(v, 10, 64)
	}
	if v, ok := result["requests"]; ok {
		data.Requests, _ = strconv.ParseInt(v, 10, 64)
	}

	return data, nil
}

// getCostData 从 Redis Hash 获取费用数据
func (s *UsageService) getCostData(ctx context.Context, key string) (*CostData, error) {
	result, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	data := &CostData{}
	// 从整数存储恢复为浮点数（除以1000000）
	if v, ok := result["input_cost"]; ok {
		val, _ := strconv.ParseInt(v, 10, 64)
		data.InputCost = float64(val) / 1000000
	}
	if v, ok := result["output_cost"]; ok {
		val, _ := strconv.ParseInt(v, 10, 64)
		data.OutputCost = float64(val) / 1000000
	}
	if v, ok := result["cache_create_cost"]; ok {
		val, _ := strconv.ParseInt(v, 10, 64)
		data.CacheCreateCost = float64(val) / 1000000
	}
	if v, ok := result["cache_read_cost"]; ok {
		val, _ := strconv.ParseInt(v, 10, 64)
		data.CacheReadCost = float64(val) / 1000000
	}
	if v, ok := result["total_cost"]; ok {
		val, _ := strconv.ParseInt(v, 10, 64)
		data.TotalCost = float64(val) / 1000000
	}

	return data, nil
}

// getRecords 从 Redis List 获取记录
func (s *UsageService) getRecords(ctx context.Context, key string, offset, limit int64) ([]UsageRecord, error) {
	results, err := s.rdb.LRange(ctx, key, offset, offset+limit-1).Result()
	if err != nil {
		return nil, err
	}

	records := make([]UsageRecord, 0, len(results))
	for _, r := range results {
		var record UsageRecord
		if err := json.Unmarshal([]byte(r), &record); err != nil {
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

// GetUserDailyStats 获取用户指定日期范围的每日统计
func (s *UsageService) GetUserDailyStats(ctx context.Context, userID uint, startDate, endDate string) ([]model.DailyUsageStats, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, err
	}

	var stats []model.DailyUsageStats

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		date := d.Format("2006-01-02")

		usage, err := s.GetUserDailyUsage(ctx, userID, date)
		if err != nil {
			continue
		}

		cost, err := s.GetUserDailyCost(ctx, userID, date)
		if err != nil {
			continue
		}

		// 只返回有数据的日期
		if usage.Requests > 0 {
			stats = append(stats, model.DailyUsageStats{
				Date:                     date,
				RequestCount:             usage.Requests,
				InputTokens:              usage.InputTokens,
				OutputTokens:             usage.OutputTokens,
				CacheCreationInputTokens: usage.CacheCreationInputTokens,
				CacheReadInputTokens:     usage.CacheReadInputTokens,
				TotalTokens:              usage.TotalTokens,
				TotalCost:                cost.TotalCost,
			})
		}
	}

	return stats, nil
}

// RecordRequest 记录一次请求（综合方法，同时更新使用量、费用和记录）
func (s *UsageService) RecordRequest(ctx context.Context, userID, apiKeyID uint, log *model.RequestLog, priceRate float64) error {
	// 计算 token 统计
	usageData := &UsageData{
		InputTokens:              int64(log.InputTokens),
		OutputTokens:             int64(log.OutputTokens),
		CacheCreationInputTokens: int64(log.CacheCreationInputTokens),
		CacheReadInputTokens:     int64(log.CacheReadInputTokens),
		TotalTokens:              int64(log.TotalTokens),
		Requests:                 1,
	}

	// 费用数据（已经计算好倍率）
	costData := &CostData{
		InputCost:       log.InputCost,
		OutputCost:      log.OutputCost,
		CacheCreateCost: log.CacheCreateCost,
		CacheReadCost:   log.CacheReadCost,
		TotalCost:       log.TotalCost,
	}

	// 使用记录
	record := &UsageRecord{
		ID:                       fmt.Sprintf("%d", log.ID),
		Model:                    log.Model,
		RequestIP:                log.RequestIP,
		InputTokens:              log.InputTokens,
		OutputTokens:             log.OutputTokens,
		CacheCreationInputTokens: log.CacheCreationInputTokens,
		CacheReadInputTokens:     log.CacheReadInputTokens,
		TotalTokens:              log.TotalTokens,
		TotalCost:                log.TotalCost,
		Timestamp:                log.CreatedAt,
	}

	// 并行执行
	errChan := make(chan error, 3)

	go func() {
		errChan <- s.IncrementUsage(ctx, userID, apiKeyID, usageData)
	}()

	go func() {
		errChan <- s.IncrementCost(ctx, userID, apiKeyID, costData)
	}()

	go func() {
		errChan <- s.AddUsageRecord(ctx, userID, apiKeyID, record)
	}()

	// 收集错误
	for i := 0; i < 3; i++ {
		if err := <-errChan; err != nil {
			return err
		}
	}

	return nil
}

// IncrementModelUsage 增加模型使用统计
func (s *UsageService) IncrementModelUsage(ctx context.Context, userID uint, modelName string, tokens int64, cost float64) error {
	key := fmt.Sprintf(keyModelUsage, userID, modelName)
	pipe := s.rdb.Pipeline()
	pipe.HIncrBy(ctx, key, "tokens", tokens)
	pipe.HIncrBy(ctx, key, "cost", int64(cost*1000000))
	pipe.HIncrBy(ctx, key, "requests", 1)
	_, err := pipe.Exec(ctx)
	return err
}

// GetUserModelStats 获取用户按模型的统计
func (s *UsageService) GetUserModelStats(ctx context.Context, userID uint) ([]model.ModelUsageStats, error) {
	// 使用 SCAN 查找所有模型统计 key
	pattern := fmt.Sprintf("usage:model:%d:*", userID)
	var cursor uint64
	var stats []model.ModelUsageStats

	for {
		keys, newCursor, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			// 从 key 中提取模型名
			// key 格式: usage:model:{userID}:{model}
			parts := len(fmt.Sprintf("usage:model:%d:", userID))
			if len(key) <= parts {
				continue
			}
			modelName := key[parts:]

			result, err := s.rdb.HGetAll(ctx, key).Result()
			if err != nil {
				continue
			}

			stat := model.ModelUsageStats{Model: modelName}
			if v, ok := result["tokens"]; ok {
				stat.TotalTokens, _ = strconv.ParseInt(v, 10, 64)
			}
			if v, ok := result["cost"]; ok {
				val, _ := strconv.ParseInt(v, 10, 64)
				stat.TotalCost = float64(val) / 1000000
			}
			if v, ok := result["requests"]; ok {
				stat.RequestCount, _ = strconv.ParseInt(v, 10, 64)
			}

			stats = append(stats, stat)
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
	}

	return stats, nil
}

// IncrementAccountCost 增加账户费用（原子操作）
func (s *UsageService) IncrementAccountCost(ctx context.Context, accountID uint, cost float64) error {
	if accountID == 0 {
		return nil
	}

	now := time.Now()
	date := now.Format("2006-01-02")

	pipe := s.rdb.Pipeline()

	// 账户总费用
	totalKey := fmt.Sprintf(keyCostAccount, accountID)
	pipe.HIncrBy(ctx, totalKey, "total_cost", int64(cost*1000000))
	pipe.HIncrBy(ctx, totalKey, "requests", 1)

	// 账户每日费用
	dailyKey := fmt.Sprintf(keyCostAccountDaily, accountID, date)
	pipe.HIncrBy(ctx, dailyKey, "total_cost", int64(cost*1000000))
	pipe.HIncrBy(ctx, dailyKey, "requests", 1)
	pipe.Expire(ctx, dailyKey, dailyExpiration)

	_, err := pipe.Exec(ctx)
	return err
}

// GetAccountCost 获取账户总费用（从 Redis）
func (s *UsageService) GetAccountCost(ctx context.Context, accountID uint) (float64, error) {
	key := fmt.Sprintf(keyCostAccount, accountID)
	result, err := s.rdb.HGet(ctx, key, "total_cost").Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return 0, nil
		}
		return 0, err
	}

	val, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return 0, err
	}

	return float64(val) / 1000000, nil
}

// GetAccountsCost 批量获取账户总费用（从 Redis）
func (s *UsageService) GetAccountsCost(ctx context.Context, accountIDs []uint) (map[uint]float64, error) {
	result := make(map[uint]float64)
	if len(accountIDs) == 0 {
		return result, nil
	}

	pipe := s.rdb.Pipeline()
	cmds := make(map[uint]*redis.StringCmd)

	for _, id := range accountIDs {
		key := fmt.Sprintf(keyCostAccount, id)
		cmds[id] = pipe.HGet(ctx, key, "total_cost")
	}

	_, err := pipe.Exec(ctx)
	if err != nil && err.Error() != "redis: nil" {
		// Pipeline exec 可能有部分错误，继续处理
	}

	for id, cmd := range cmds {
		val, err := cmd.Result()
		if err != nil {
			continue
		}
		intVal, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			continue
		}
		result[id] = float64(intVal) / 1000000
	}

	return result, nil
}

// GetAccountDailyCost 获取账户某天的费用
func (s *UsageService) GetAccountDailyCost(ctx context.Context, accountID uint, date string) (float64, error) {
	key := fmt.Sprintf(keyCostAccountDaily, accountID, date)
	result, err := s.rdb.HGet(ctx, key, "total_cost").Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return 0, nil
		}
		return 0, err
	}

	val, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return 0, err
	}

	return float64(val) / 1000000, nil
}
