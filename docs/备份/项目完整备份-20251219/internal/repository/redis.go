package repository

import (
	"context"
	"strings"

	"go-aiproxy/internal/config"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

func InitRedis() error {
	RDB = redis.NewClient(&redis.Options{
		Addr:     config.Cfg.Redis.Addr(),
		Password: config.Cfg.Redis.Password,
		DB:       config.Cfg.Redis.DB,
	})

	ctx := context.Background()
	if err := RDB.Ping(ctx).Err(); err != nil {
		return err
	}

	return nil
}

// GetRedisInfo 获取 Redis 服务器信息
func GetRedisInfo() (map[string]string, error) {
	ctx := context.Background()
	info, err := RDB.Info(ctx).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			result[parts[0]] = strings.TrimSpace(parts[1])
		}
	}
	return result, nil
}

// CloseRedis 关闭 Redis 连接
func CloseRedis() error {
	if RDB != nil {
		return RDB.Close()
	}
	return nil
}
