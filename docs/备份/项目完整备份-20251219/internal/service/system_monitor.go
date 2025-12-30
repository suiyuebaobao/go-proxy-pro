package service

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"

	"github.com/redis/go-redis/v9"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"gorm.io/gorm"
)

// SystemMonitorService 系统监控服务
type SystemMonitorService struct {
	rdb             *redis.Client
	db              *gorm.DB
	accountRepo     *repository.AccountRepository
	userRepo        *repository.UserRepository
	dailyUsageRepo  *repository.DailyUsageRepository
	requestLogRepo  *repository.RequestLogRepository
}

// NewSystemMonitorService 创建系统监控服务
func NewSystemMonitorService() *SystemMonitorService {
	return &SystemMonitorService{
		rdb:             repository.RDB,
		db:              repository.GetDB(),
		accountRepo:     repository.NewAccountRepository(),
		userRepo:        repository.NewUserRepository(),
		dailyUsageRepo:  repository.NewDailyUsageRepository(),
		requestLogRepo:  repository.NewRequestLogRepository(),
	}
}

// SystemStats 系统资源统计
type SystemStats struct {
	// CPU
	CPUCores     int     `json:"cpu_cores"`      // CPU 核心数
	CPUUsage     float64 `json:"cpu_usage"`      // CPU 使用率 (%)

	// 内存
	MemoryTotal  uint64  `json:"memory_total"`   // 内存总量 (bytes)
	MemoryUsed   uint64  `json:"memory_used"`    // 已用内存 (bytes)
	MemoryFree   uint64  `json:"memory_free"`    // 可用内存 (bytes)
	MemoryUsage  float64 `json:"memory_usage"`   // 内存使用率 (%)

	// 磁盘
	DiskTotal    uint64  `json:"disk_total"`     // 磁盘总量 (bytes)
	DiskUsed     uint64  `json:"disk_used"`      // 已用磁盘 (bytes)
	DiskFree     uint64  `json:"disk_free"`      // 可用磁盘 (bytes)
	DiskUsage    float64 `json:"disk_usage"`     // 磁盘使用率 (%)
}

// RedisStats Redis 统计
type RedisStats struct {
	KeyCount     int64  `json:"key_count"`       // Key 数量
	MemoryUsed   int64  `json:"memory_used"`     // 已用内存 (bytes)
	MemoryPeak   int64  `json:"memory_peak"`     // 内存峰值 (bytes)
	Connected    bool   `json:"connected"`       // 是否连接
}

// MySQLStats MySQL 统计
type MySQLStats struct {
	TableCount   int    `json:"table_count"`     // 表数量
	DataSize     int64  `json:"data_size"`       // 数据大小 (bytes)
	IndexSize    int64  `json:"index_size"`      // 索引大小 (bytes)
	TotalSize    int64  `json:"total_size"`      // 总大小 (bytes)
	Connected    bool   `json:"connected"`       // 是否连接
}

// AccountStats 账号统计
type AccountStats struct {
	Total        int64 `json:"total"`           // 总数
	Active       int64 `json:"active"`          // 正常可用
	RateLimited  int64 `json:"rate_limited"`    // 被限流
	Invalid      int64 `json:"invalid"`         // 无效/禁用
}

// UserStats 用户统计
type UserStats struct {
	Total        int64 `json:"total"`           // 总用户数
	Active       int64 `json:"active"`          // 活跃用户数（今日有请求）
	NewToday     int64 `json:"new_today"`       // 今日新增
}

// TodayUsageStats 今日使用统计
type TodayUsageStats struct {
	TotalCost           float64 `json:"total_cost"`            // 今日消费
	TotalTokens         int64   `json:"total_tokens"`          // 今日总 token
	InputTokens         int64   `json:"input_tokens"`          // 输入 token
	OutputTokens        int64   `json:"output_tokens"`         // 输出 token
	CacheCreationTokens int64   `json:"cache_creation_tokens"` // 缓存创建 token
	CacheReadTokens     int64   `json:"cache_read_tokens"`     // 缓存读取 token
	RequestCount        int64   `json:"request_count"`         // 请求次数
}

// MonitorData 完整监控数据
type MonitorData struct {
	System     SystemStats     `json:"system"`
	Redis      RedisStats      `json:"redis"`
	MySQL      MySQLStats      `json:"mysql"`
	Accounts   AccountStats    `json:"accounts"`
	Users      UserStats       `json:"users"`
	TodayUsage TodayUsageStats `json:"today_usage"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// GetMonitorData 获取完整监控数据
func (s *SystemMonitorService) GetMonitorData(ctx context.Context) (*MonitorData, error) {
	data := &MonitorData{
		UpdatedAt: time.Now(),
	}

	// 并行获取各项数据
	data.System = s.GetSystemStats()
	data.Redis = s.GetRedisStats(ctx)
	data.MySQL = s.GetMySQLStats()
	data.Accounts = s.GetAccountStats()
	data.Users = s.GetUserStats()
	data.TodayUsage = s.GetTodayUsageStats(ctx)

	return data, nil
}

// GetSystemStats 获取系统资源统计
func (s *SystemMonitorService) GetSystemStats() SystemStats {
	stats := SystemStats{}

	// CPU
	stats.CPUCores = runtime.NumCPU()
	if cpuPercent, err := cpu.Percent(100*time.Millisecond, false); err == nil && len(cpuPercent) > 0 {
		stats.CPUUsage = cpuPercent[0]
	}

	// 内存
	if memInfo, err := mem.VirtualMemory(); err == nil {
		stats.MemoryTotal = memInfo.Total
		stats.MemoryUsed = memInfo.Used
		stats.MemoryFree = memInfo.Available
		stats.MemoryUsage = memInfo.UsedPercent
	}

	// 磁盘 (使用当前工作目录所在的磁盘)
	wd, _ := os.Getwd()
	if diskInfo, err := disk.Usage(wd); err == nil {
		stats.DiskTotal = diskInfo.Total
		stats.DiskUsed = diskInfo.Used
		stats.DiskFree = diskInfo.Free
		stats.DiskUsage = diskInfo.UsedPercent
	}

	return stats
}

// GetRedisStats 获取 Redis 统计
func (s *SystemMonitorService) GetRedisStats(ctx context.Context) RedisStats {
	stats := RedisStats{}

	if s.rdb == nil {
		return stats
	}

	// 测试连接
	if err := s.rdb.Ping(ctx).Err(); err != nil {
		stats.Connected = false
		return stats
	}
	stats.Connected = true

	// 获取 key 数量
	if dbSize, err := s.rdb.DBSize(ctx).Result(); err == nil {
		stats.KeyCount = dbSize
	}

	// 获取内存信息
	if info, err := s.rdb.Info(ctx, "memory").Result(); err == nil {
		// 解析 used_memory 和 used_memory_peak
		var usedMem, peakMem int64
		fmt.Sscanf(info, "%*[^:]used_memory:%d", &usedMem)
		fmt.Sscanf(info, "%*[^:]used_memory_peak:%d", &peakMem)

		// 更精确的解析
		lines := splitLines(info)
		for _, line := range lines {
			if len(line) > 12 && line[:12] == "used_memory:" {
				fmt.Sscanf(line, "used_memory:%d", &stats.MemoryUsed)
			}
			if len(line) > 17 && line[:17] == "used_memory_peak:" {
				fmt.Sscanf(line, "used_memory_peak:%d", &stats.MemoryPeak)
			}
		}
	}

	return stats
}

// GetMySQLStats 获取 MySQL 统计
func (s *SystemMonitorService) GetMySQLStats() MySQLStats {
	stats := MySQLStats{}

	if s.db == nil {
		return stats
	}

	// 测试连接
	sqlDB, err := s.db.DB()
	if err != nil {
		stats.Connected = false
		return stats
	}
	if err := sqlDB.Ping(); err != nil {
		stats.Connected = false
		return stats
	}
	stats.Connected = true

	// 获取当前数据库名
	var dbName string
	s.db.Raw("SELECT DATABASE()").Scan(&dbName)

	// 获取表数量
	var tableCount int
	s.db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ?", dbName).Scan(&tableCount)
	stats.TableCount = tableCount

	// 获取数据库大小
	type DBSize struct {
		DataSize  int64
		IndexSize int64
	}
	var dbSize DBSize
	s.db.Raw(`
		SELECT
			COALESCE(SUM(data_length), 0) as data_size,
			COALESCE(SUM(index_length), 0) as index_size
		FROM information_schema.tables
		WHERE table_schema = ?
	`, dbName).Scan(&dbSize)

	stats.DataSize = dbSize.DataSize
	stats.IndexSize = dbSize.IndexSize
	stats.TotalSize = dbSize.DataSize + dbSize.IndexSize

	return stats
}

// GetAccountStats 获取账号统计
func (s *SystemMonitorService) GetAccountStats() AccountStats {
	stats := AccountStats{}

	// 总数
	s.db.Model(&model.Account{}).Count(&stats.Total)

	// 正常可用 (enabled=true AND status=valid)
	s.db.Model(&model.Account{}).
		Where("enabled = ? AND status = ?", true, model.AccountStatusValid).
		Count(&stats.Active)

	// 被限流 (status=rate_limited)
	s.db.Model(&model.Account{}).
		Where("status = ?", model.AccountStatusRateLimited).
		Count(&stats.RateLimited)

	// 无效/禁用 (enabled=false OR status=invalid)
	s.db.Model(&model.Account{}).
		Where("enabled = ? OR status = ?", false, model.AccountStatusInvalid).
		Count(&stats.Invalid)

	return stats
}

// GetUserStats 获取用户统计
func (s *SystemMonitorService) GetUserStats() UserStats {
	stats := UserStats{}

	// 总用户数
	s.db.Model(&model.User{}).Count(&stats.Total)

	// 今日新增
	today := time.Now().Format("2006-01-02")
	s.db.Model(&model.User{}).
		Where("DATE(created_at) = ?", today).
		Count(&stats.NewToday)

	// 活跃用户数（今日有请求记录）
	s.db.Model(&model.RequestLog{}).
		Where("DATE(created_at) = ?", today).
		Distinct("user_id").
		Count(&stats.Active)

	return stats
}

// GetTodayUsageStats 获取今日使用统计
func (s *SystemMonitorService) GetTodayUsageStats(ctx context.Context) TodayUsageStats {
	stats := TodayUsageStats{}

	today := time.Now().Format("2006-01-02")

	// 从 MySQL daily_usages 表获取今日汇总
	type DailySummary struct {
		TotalCost           float64
		InputTokens         int64
		OutputTokens        int64
		CacheCreationTokens int64
		CacheReadTokens     int64
		RequestCount        int64
	}
	var summary DailySummary

	s.db.Model(&model.DailyUsage{}).
		Select(`
			COALESCE(SUM(total_cost), 0) as total_cost,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(SUM(cache_creation_tokens), 0) as cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens), 0) as cache_read_tokens,
			COALESCE(SUM(request_count), 0) as request_count
		`).
		Where("date = ?", today).
		Scan(&summary)

	stats.TotalCost = summary.TotalCost
	stats.InputTokens = summary.InputTokens
	stats.OutputTokens = summary.OutputTokens
	stats.CacheCreationTokens = summary.CacheCreationTokens
	stats.CacheReadTokens = summary.CacheReadTokens
	stats.RequestCount = summary.RequestCount
	stats.TotalTokens = summary.InputTokens + summary.OutputTokens + summary.CacheCreationTokens + summary.CacheReadTokens

	return stats
}

// splitLines 分割字符串为行
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
