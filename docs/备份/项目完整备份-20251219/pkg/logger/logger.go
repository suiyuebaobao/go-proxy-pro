package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// 日志级别
const (
	LevelDebug = iota
	LevelInfo
	LevelWarn
	LevelError
)

var levelNames = map[int]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
}

// Logger 日志器
type Logger struct {
	mu          sync.Mutex
	level       int
	module      string
	logDir      string
	currentDate string
	file        *os.File
	logger      *log.Logger
}

var (
	defaultLogger *Logger
	loggers       = make(map[string]*Logger)
	mu            sync.Mutex
)

// Init 初始化日志系统
func Init(logDir string, level int) error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	defaultLogger = &Logger{
		level:  level,
		module: "main",
		logDir: logDir,
	}

	if err := defaultLogger.rotateFile(); err != nil {
		return err
	}

	// 启动日志轮转检查
	go defaultLogger.rotateChecker()

	return nil
}

// GetLogger 获取指定模块的日志器
func GetLogger(module string) *Logger {
	mu.Lock()
	defer mu.Unlock()

	if l, ok := loggers[module]; ok {
		return l
	}

	l := &Logger{
		level:  defaultLogger.level,
		module: module,
		logDir: defaultLogger.logDir,
	}
	l.rotateFile()
	go l.rotateChecker()

	loggers[module] = l
	return l
}

// rotateFile 轮转日志文件
func (l *Logger) rotateFile() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	if l.currentDate == today && l.file != nil {
		return nil
	}

	// 关闭旧文件
	if l.file != nil {
		l.file.Close()
	}

	// 创建新文件
	filename := filepath.Join(l.logDir, fmt.Sprintf("%s-%s.log", l.module, today))
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	l.file = file
	l.currentDate = today

	// 只输出到文件
	l.logger = log.New(file, "", 0)

	return nil
}

// rotateChecker 定时检查是否需要轮转
func (l *Logger) rotateChecker() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		today := time.Now().Format("2006-01-02")
		if l.currentDate != today {
			l.rotateFile()
		}
	}
}

// log 记录日志
func (l *Logger) log(level int, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	levelName := levelNames[level]
	message := fmt.Sprintf(format, args...)

	logLine := fmt.Sprintf("[%s] [%s] [%s] %s", timestamp, levelName, l.module, message)
	l.logger.Println(logLine)
}

// Debug 调试日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info 信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn 警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error 错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// 默认日志器的快捷方法
func Debug(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Debug(format, args...)
	}
}

func Info(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Info(format, args...)
	}
}

func Warn(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Warn(format, args...)
	}
}

func Error(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Error(format, args...)
	}
}

// Close 关闭日志系统
func Close() {
	mu.Lock()
	defer mu.Unlock()

	if defaultLogger != nil && defaultLogger.file != nil {
		defaultLogger.file.Close()
	}

	for _, l := range loggers {
		if l.file != nil {
			l.file.Close()
		}
	}
}

// SetLevel 设置日志级别
func SetLevel(level int) {
	if defaultLogger != nil {
		defaultLogger.level = level
	}
}

// ParseLevel 解析日志级别字符串
func ParseLevel(levelStr string) int {
	switch levelStr {
	case "debug", "DEBUG":
		return LevelDebug
	case "info", "INFO":
		return LevelInfo
	case "warn", "WARN", "warning", "WARNING":
		return LevelWarn
	case "error", "ERROR":
		return LevelError
	default:
		return LevelInfo
	}
}
