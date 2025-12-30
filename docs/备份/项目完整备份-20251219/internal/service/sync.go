package service

import (
	"context"
	"encoding/json"
	"fmt"
	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/pkg/logger"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// SyncService 同步服务
type SyncService struct {
	rdb             *redis.Client
	usageRecordRepo *repository.UsageRecordRepository
	accountRepo     *repository.AccountRepository
	configService   *ConfigService
	log             *logger.Logger

	mu          sync.Mutex
	running     bool
	stopChan    chan struct{}
	lastSync    time.Time
	syncedCount int64
	lastError   error
}

var syncService *SyncService
var syncOnce sync.Once

// GetSyncService 获取同步服务单例
func GetSyncService() *SyncService {
	syncOnce.Do(func() {
		syncService = &SyncService{
			rdb:             repository.RDB,
			usageRecordRepo: repository.NewUsageRecordRepository(),
			accountRepo:     repository.NewAccountRepository(),
			configService:   GetConfigService(),
			log:             logger.GetLogger("sync"),
			stopChan:        make(chan struct{}),
		}
	})
	return syncService
}

// Start 启动同步任务
func (s *SyncService) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	go s.syncLoop()
	s.log.Info("同步服务已启动")
}

// Stop 停止同步任务
func (s *SyncService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	close(s.stopChan)
	s.running = false
	s.log.Info("同步服务已停止")
}

// Restart 重启同步任务（配置变更时调用）
func (s *SyncService) Restart() {
	s.Stop()
	time.Sleep(100 * time.Millisecond)
	s.Start()
}

// syncLoop 同步循环
func (s *SyncService) syncLoop() {
	// 首次启动时执行一次同步
	s.doSync()

	for {
		interval := s.configService.GetSyncInterval()
		if interval < time.Minute {
			interval = time.Minute // 最小1分钟
		}

		select {
		case <-time.After(interval):
			if s.configService.GetSyncEnabled() {
				s.doSync()
			}
		case <-s.stopChan:
			return
		}
	}
}

// TriggerSync 手动触发同步
func (s *SyncService) TriggerSync() {
	go s.doSync()
}

// doSync 执行同步
func (s *SyncService) doSync() {
	s.log.Info("开始同步使用记录到 MySQL...")
	startTime := time.Now()

	ctx := context.Background()

	// 1. 同步使用记录
	totalSynced := s.syncUsageRecords(ctx)

	// 2. 同步账户费用
	s.syncAccountCosts(ctx)

	s.lastSync = time.Now()
	s.syncedCount = totalSynced
	s.lastError = nil

	duration := time.Since(startTime)
	s.log.Info("同步完成，共同步 %d 条记录，耗时 %v", totalSynced, duration)
}

// syncUsageRecords 同步使用记录
func (s *SyncService) syncUsageRecords(ctx context.Context) int64 {
	// 获取所有用户的使用记录 keys
	pattern := "usage:records:*"
	var cursor uint64
	var totalSynced int64

	for {
		keys, newCursor, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			s.lastError = err
			s.log.Error("扫描 Redis keys 失败: %v", err)
			return totalSynced
		}

		for _, key := range keys {
			// 跳过 API Key 的记录（usage:records:key:*）
			if len(key) > 18 && key[14:18] == "key:" {
				continue
			}

			synced, err := s.syncUserRecords(ctx, key)
			if err != nil {
				s.log.Error("同步 %s 失败: %v", key, err)
				continue
			}
			totalSynced += synced
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
	}

	return totalSynced
}

// syncAccountCosts 同步账户费用从 Redis 到 MySQL
func (s *SyncService) syncAccountCosts(ctx context.Context) {
	// 扫描所有账户费用 keys: cost:account:*
	pattern := "cost:account:*"
	var cursor uint64
	var syncedCount int

	for {
		keys, newCursor, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			s.log.Error("扫描账户费用 keys 失败: %v", err)
			return
		}

		for _, key := range keys {
			// 跳过每日费用 keys (cost:account:daily:*)
			if len(key) > 20 && key[13:19] == "daily:" {
				continue
			}

			// 从 key 中提取 accountID: cost:account:{accountID}
			var accountID uint
			_, err := fmt.Sscanf(key, "cost:account:%d", &accountID)
			if err != nil {
				continue
			}

			// 获取 Redis 中的费用
			result, err := s.rdb.HGet(ctx, key, "total_cost").Result()
			if err != nil {
				continue
			}

			costInt, err := strconv.ParseInt(result, 10, 64)
			if err != nil {
				continue
			}

			totalCost := float64(costInt) / 1000000

			// 更新 MySQL
			if err := s.accountRepo.UpdateTotalCost(accountID, totalCost); err != nil {
				s.log.Error("同步账户费用失败: accountID=%d, error=%v", accountID, err)
				continue
			}

			syncedCount++
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
	}

	if syncedCount > 0 {
		s.log.Info("同步账户费用完成，共更新 %d 个账户", syncedCount)
	}
}

// syncUserRecords 同步单个用户的记录
func (s *SyncService) syncUserRecords(ctx context.Context, key string) (int64, error) {
	// 从 key 中提取 userID: usage:records:{userID}
	var userID uint
	_, err := fmt.Sscanf(key, "usage:records:%d", &userID)
	if err != nil {
		return 0, err
	}

	// 获取最后同步的记录时间（用于增量同步）
	lastSyncKey := fmt.Sprintf("sync:last:%d", userID)
	lastSyncTime := time.Time{}
	if val, err := s.rdb.Get(ctx, lastSyncKey).Result(); err == nil {
		lastSyncTime, _ = time.Parse(time.RFC3339, val)
	}

	// 获取 Redis 中的记录
	records, err := s.rdb.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return 0, err
	}

	var toSync []model.UsageRecord
	for _, r := range records {
		var record UsageRecord
		if err := json.Unmarshal([]byte(r), &record); err != nil {
			continue
		}

		// 只同步新记录
		if record.Timestamp.After(lastSyncTime) {
			// 提取 API Key ID
			apiKeyID := uint(0)
			if record.ID != "" {
				// ID 格式可能是 requestLogID
				if id, err := strconv.ParseUint(record.ID, 10, 64); err == nil {
					apiKeyID = uint(id)
				}
			}

			toSync = append(toSync, model.UsageRecord{
				UserID:                   userID,
				APIKeyID:                 apiKeyID,
				Model:                    record.Model,
				InputTokens:              record.InputTokens,
				OutputTokens:             record.OutputTokens,
				CacheCreationInputTokens: record.CacheCreationInputTokens,
				CacheReadInputTokens:     record.CacheReadInputTokens,
				TotalTokens:              record.TotalTokens,
				TotalCost:                record.TotalCost,
				RequestTime:              record.Timestamp,
			})
		}
	}

	if len(toSync) == 0 {
		return 0, nil
	}

	// 批量写入 MySQL
	if err := s.usageRecordRepo.BatchCreate(toSync); err != nil {
		return 0, err
	}

	// 更新最后同步时间
	s.rdb.Set(ctx, lastSyncKey, time.Now().Format(time.RFC3339), 24*time.Hour)

	return int64(len(toSync)), nil
}

// GetStatus 获取同步状态
func (s *SyncService) GetStatus() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := map[string]interface{}{
		"enabled":      s.configService.GetSyncEnabled(),
		"interval":     s.configService.GetSyncInterval().Minutes(),
		"running":      s.running,
		"last_sync":    nil,
		"synced_count": s.syncedCount,
		"last_error":   nil,
	}

	if !s.lastSync.IsZero() {
		status["last_sync"] = s.lastSync.Format(time.RFC3339)
	}

	if s.lastError != nil {
		status["last_error"] = s.lastError.Error()
	}

	return status
}

// OnConfigChange 配置变更回调
func (s *SyncService) OnConfigChange(key, value string) {
	switch key {
	case model.ConfigSyncEnabled:
		if value == "true" {
			s.Start()
		} else {
			s.Stop()
		}
	case model.ConfigSyncInterval:
		// 重启以应用新的间隔
		if s.running {
			s.Restart()
		}
	}
}
