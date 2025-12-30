package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"go-aiproxy/internal/config"
	"go-aiproxy/internal/handler"
	"go-aiproxy/internal/middleware"
	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
	"go-aiproxy/internal/service"
	"go-aiproxy/pkg/logger"

	"github.com/gin-gonic/gin"
)

func main() {
	// 全局 panic 恢复
	defer func() {
		if r := recover(); r != nil {
			// 尝试获取日志实例记录 panic
			if log := logger.GetLogger("main"); log != nil {
				log.Error("=== 服务崩溃 (PANIC) ===")
				log.Error("Panic 原因: %v", r)
				log.Error("堆栈信息:\n%s", string(debug.Stack()))
				log.Error("=== 服务异常终止 ===")
			}
			// 同时输出到 stderr
			fmt.Fprintf(os.Stderr, "\n=== 服务崩溃 (PANIC) ===\n")
			fmt.Fprintf(os.Stderr, "Panic 原因: %v\n", r)
			fmt.Fprintf(os.Stderr, "堆栈信息:\n%s\n", string(debug.Stack()))
			os.Exit(1)
		}
	}()

	startTime := time.Now()

	// 加载配置
	configPath := "configs/config.yaml"
	if err := config.Load(configPath); err != nil {
		panic(fmt.Sprintf("加载配置失败: %v", err))
	}

	// 初始化日志系统
	logDir := config.Cfg.Log.Dir
	if logDir == "" {
		logDir = "logs"
	}
	logLevel := logger.ParseLevel(config.Cfg.Log.Level)
	if err := logger.Init(logDir, logLevel); err != nil {
		panic(fmt.Sprintf("初始化日志失败: %v", err))
	}
	defer logger.Close()

	log := logger.GetLogger("main")

	// 打印启动横幅和系统信息
	log.Info("Go-AIProxy 服务启动 | 系统: %s/%s | CPU: %d核 | Go: %s | PID: %d | 工作目录: %s",
		runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), runtime.Version(), os.Getpid(), getWorkDir())

	// 配置信息
	log.Info("配置加载 | 文件: %s | 日志: %s(%s) | 模式: %s | 端口: %d | 网卡: %s",
		configPath, logDir, config.Cfg.Log.Level, config.Cfg.Server.Mode, config.Cfg.Server.Port, getNetworkIPs())

	// 初始化数据库
	log.Info("MySQL 连接中 | %s@%s:%d/%s | 字符集: %s | 连接池: %d-%d",
		config.Cfg.MySQL.User, config.Cfg.MySQL.Host, config.Cfg.MySQL.Port,
		config.Cfg.MySQL.Database, config.Cfg.MySQL.Charset,
		config.Cfg.MySQL.MaxIdleConns, config.Cfg.MySQL.MaxOpenConns)

	mysqlStart := time.Now()
	if err := repository.InitMySQL(); err != nil {
		log.Error("MySQL 连接失败: %v | 请检查: 1.服务是否启动 2.地址端口是否正确 3.用户密码是否正确 4.数据库是否存在 5.防火墙设置", err)
		panic(err)
	}
	log.Info("MySQL 连接成功 | 耗时: %v", time.Since(mysqlStart))

	// 初始化 Redis
	log.Info("Redis 连接中 | %s:%d | DB: %d | 密码: %s",
		config.Cfg.Redis.Host, config.Cfg.Redis.Port, config.Cfg.Redis.DB, maskRedisPassword(config.Cfg.Redis.Password))

	redisStart := time.Now()
	if err := repository.InitRedis(); err != nil {
		log.Error("Redis 连接失败: %v | 请检查: 1.服务是否启动 2.地址端口是否正确 3.密码是否正确 4.防火墙设置", err)
		panic(err)
	}

	// 获取 Redis 信息
	redisInfo := getRedisInfoString()
	log.Info("Redis 连接成功 | 耗时: %v | %s", time.Since(redisStart), redisInfo)

	// 数据库迁移
	migrateStart := time.Now()
	if err := repository.AutoMigrate(); err != nil {
		log.Error("数据库迁移失败: %v", err)
		panic(err)
	}
	log.Info("数据库迁移完成 | 耗时: %v", time.Since(migrateStart))

	// 初始化默认管理员
	if err := repository.InitDefaultAdmin(); err != nil {
		log.Warn("初始化默认管理员: %v", err)
	}

	// 初始化默认系统配置
	if err := repository.InitDefaultConfigs(); err != nil {
		log.Warn("初始化默认配置: %v", err)
	}

	// 迁移未绑定套餐的 API Key
	if err := repository.MigrateAPIKeyPackageBinding(); err != nil {
		log.Warn("API Key 套餐绑定迁移: %v", err)
	}

	// 初始化默认客户端过滤配置
	if err := repository.InitDefaultClientFilters(); err != nil {
		log.Warn("初始化客户端过滤配置: %v", err)
	}

	// 初始化默认错误消息配置
	if err := repository.InitDefaultErrorMessages(); err != nil {
		log.Warn("初始化错误消息配置: %v", err)
	}

	// 初始化配置服务
	configService := service.GetConfigService()
	log.Info("会话粘性 TTL: %d分钟", config.Cfg.Cache.GetSessionTTL())

	// 启动同步服务
	syncService := service.GetSyncService()
	if configService.GetSyncEnabled() {
		syncService.Start()
		log.Info("同步服务已启动 | 间隔: %v", configService.GetSyncInterval())
	}

	// 设置配置变更回调
	handler.SetConfigChangeCallback(func(key, value string) {
		switch key {
		case model.ConfigSessionTTL:
			log.Info("会话 TTL 配置已更新: %s", value)
		case model.ConfigSyncEnabled, model.ConfigSyncInterval:
			syncService.OnConfigChange(key, value)
		}
	})

	// 设置同步触发回调
	handler.SetSyncTriggerCallback(syncService.TriggerSync)
	handler.SetSyncStatusCallback(syncService.GetStatus)

	// JWT 配置
	log.Info("JWT 配置 | 密钥: %s | 过期: %d小时", maskJWTSecret(config.Cfg.JWT.Secret), config.Cfg.JWT.ExpireHours)

	// 设置 Gin 为 release 模式，避免debug日志输出到控制台
	gin.SetMode(gin.ReleaseMode)

	// 创建路由
	r := gin.New()

	// 基础中间件
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())

	// 注册路由
	handler.RegisterRoutes(r)

	// 统计路由数量
	routes := r.Routes()
	log.Info("路由注册完成 | 路由数量: %d", len(routes))

	// 启动完成信息
	log.Info("服务启动完成 | 总耗时: %v | 监听: 0.0.0.0:%d | 访问: %s",
		time.Since(startTime), config.Cfg.Server.Port, getAccessURLs(config.Cfg.Server.Port))

	// 创建 HTTP 服务器
	addr := fmt.Sprintf(":%d", config.Cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// 在 goroutine 中启动服务器
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("服务启动失败: %v", err)
			os.Exit(1)
		}
	}()

	// 设置信号监听
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

	// 等待信号
	sig := <-quit
	log.Info("=== 收到关闭信号: %v ===", sig)
	log.Info("信号说明: %s", getSignalDescription(sig))

	// 记录关闭原因
	log.Info("服务运行时长: %v", time.Since(startTime))
	log.Info("正在优雅关闭服务...")

	// 停止同步服务
	if syncService != nil {
		syncService.Stop()
		log.Info("同步服务已停止")
	}

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 优雅关闭 HTTP 服务器
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("服务关闭出错: %v", err)
	}

	// 关闭数据库连接
	if err := repository.CloseMySQL(); err != nil {
		log.Error("关闭 MySQL 连接出错: %v", err)
	} else {
		log.Info("MySQL 连接已关闭")
	}

	// 关闭 Redis 连接
	if err := repository.CloseRedis(); err != nil {
		log.Error("关闭 Redis 连接出错: %v", err)
	} else {
		log.Info("Redis 连接已关闭")
	}

	log.Info("=== 服务已正常关闭 ===")
}

// getWorkDir 获取工作目录
func getWorkDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return dir
}

// getNetworkIPs 获取所有网卡IP
func getNetworkIPs() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "unknown"
	}

	var ips []string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				ips = append(ips, fmt.Sprintf("%s(%s)", iface.Name, ipnet.IP.String()))
			}
		}
	}
	if len(ips) == 0 {
		return "none"
	}
	return strings.Join(ips, ", ")
}

// getAccessURLs 获取访问地址
func getAccessURLs(port int) string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Sprintf("http://localhost:%d", port)
	}

	urls := []string{fmt.Sprintf("http://localhost:%d", port)}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				urls = append(urls, fmt.Sprintf("http://%s:%d", ipnet.IP.String(), port))
			}
		}
	}
	return strings.Join(urls, ", ")
}

// getRedisInfoString 获取 Redis 信息字符串
func getRedisInfoString() string {
	info, err := repository.GetRedisInfo()
	if err != nil {
		return "获取信息失败"
	}
	return fmt.Sprintf("版本: %s | 模式: %s | 内存: %s | 客户端: %s",
		info["redis_version"], info["redis_mode"], info["used_memory_human"], info["connected_clients"])
}

// maskRedisPassword 遮蔽 Redis 密码
func maskRedisPassword(password string) string {
	if password == "" {
		return "(无)"
	}
	if len(password) <= 4 {
		return "****"
	}
	return password[:2] + "****"
}

// maskJWTSecret 遮蔽 JWT 密钥
func maskJWTSecret(secret string) string {
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:4] + "****"
}

// getSignalDescription 获取信号描述
func getSignalDescription(sig os.Signal) string {
	switch sig {
	case syscall.SIGINT:
		return "SIGINT (Ctrl+C 中断)"
	case syscall.SIGTERM:
		return "SIGTERM (终止信号，通常来自 kill 命令或系统关机)"
	case syscall.SIGHUP:
		return "SIGHUP (终端挂起或控制进程终止)"
	case syscall.SIGQUIT:
		return "SIGQUIT (Ctrl+\\ 退出)"
	default:
		return fmt.Sprintf("未知信号: %v", sig)
	}
}
