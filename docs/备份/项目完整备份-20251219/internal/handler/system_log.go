package handler

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-aiproxy/pkg/response"

	"github.com/gin-gonic/gin"
)

// SystemLogHandler 系统日志处理器
type SystemLogHandler struct {
	appLogDir    string
	serverLogDir string
	// 允许查看的服务器日志文件（白名单，安全考虑）
	allowedServerLogs map[string]string
}

// NewSystemLogHandler 创建系统日志处理器
func NewSystemLogHandler() *SystemLogHandler {
	return &SystemLogHandler{
		appLogDir:    "logs",
		serverLogDir: "/var/log",
		allowedServerLogs: map[string]string{
			"auth.log":       "SSH认证日志",
			"auth.log.1":     "SSH认证日志(旧)",
			"syslog":         "系统日志",
			"syslog.1":       "系统日志(旧)",
			"kern.log":       "内核日志",
			"kern.log.1":     "内核日志(旧)",
			"dpkg.log":       "软件包日志",
			"fail2ban.log":   "Fail2ban日志",
			"nginx/access.log":  "Nginx访问日志",
			"nginx/error.log":   "Nginx错误日志",
			"mysql/error.log":   "MySQL错误日志",
		},
	}
}

// LogFileInfo 日志文件信息
type LogFileInfo struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	ModTime    time.Time `json:"mod_time"`
	Category   string    `json:"category"`
	Date       string    `json:"date,omitempty"`
	SizeHuman  string    `json:"size_human"`
}

// LogCategory 日志分类
type LogCategory struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

// ListFiles 获取日志文件列表
func (h *SystemLogHandler) ListFiles(c *gin.Context) {
	source := c.DefaultQuery("source", "app") // app=应用日志, server=服务器日志
	category := c.Query("category")           // 可选：筛选分类
	date := c.Query("date")                   // 可选：筛选日期

	if source == "server" {
		h.listServerLogs(c)
		return
	}

	// 应用日志
	files, err := os.ReadDir(h.appLogDir)
	if err != nil {
		response.Error(c, 500, "读取日志目录失败: "+err.Error())
		return
	}

	var logFiles []LogFileInfo
	categoryMap := make(map[string]int)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if !strings.HasSuffix(name, ".log") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		// 解析分类和日期
		cat, dt := parseLogFileName(name)
		categoryMap[cat]++

		// 筛选
		if category != "" && cat != category {
			continue
		}
		if date != "" && dt != date {
			continue
		}

		logFiles = append(logFiles, LogFileInfo{
			Name:      name,
			Size:      info.Size(),
			ModTime:   info.ModTime(),
			Category:  cat,
			Date:      dt,
			SizeHuman: formatFileSize(info.Size()),
		})
	}

	// 按修改时间倒序
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].ModTime.After(logFiles[j].ModTime)
	})

	// 构建分类列表
	categories := []LogCategory{
		{Name: "", Label: "全部", Count: len(logFiles)},
	}
	categoryLabels := map[string]string{
		"auth":           "认证日志",
		"http":           "HTTP日志",
		"main":           "主程序日志",
		"proxy":          "代理日志",
		"scheduler":      "调度器日志",
		"sync":           "同步日志",
		"operation":      "操作日志",
		"middleware":     "中间件日志",
		"client_filter":  "客户端过滤",
		"error_message":  "错误消息",
		"error_response": "错误响应",
		"server":         "服务器日志",
	}

	// 按分类统计
	var cats []string
	for cat := range categoryMap {
		cats = append(cats, cat)
	}
	sort.Strings(cats)

	for _, cat := range cats {
		label := categoryLabels[cat]
		if label == "" {
			label = cat
		}
		categories = append(categories, LogCategory{
			Name:  cat,
			Label: label,
			Count: categoryMap[cat],
		})
	}

	response.Success(c, gin.H{
		"files":      logFiles,
		"categories": categories,
		"total":      len(logFiles),
	})
}

// listServerLogs 列出服务器日志
func (h *SystemLogHandler) listServerLogs(c *gin.Context) {
	var logFiles []LogFileInfo
	categoryMap := make(map[string]int)

	for fileName := range h.allowedServerLogs {
		filePath := filepath.Join(h.serverLogDir, fileName)
		info, err := os.Stat(filePath)
		if err != nil {
			continue // 文件不存在，跳过
		}

		// 解析分类
		cat := "system"
		if strings.HasPrefix(fileName, "auth") {
			cat = "ssh"
		} else if strings.HasPrefix(fileName, "nginx") {
			cat = "nginx"
		} else if strings.HasPrefix(fileName, "mysql") {
			cat = "mysql"
		} else if strings.HasPrefix(fileName, "fail2ban") {
			cat = "fail2ban"
		} else if strings.HasPrefix(fileName, "kern") {
			cat = "kernel"
		}

		categoryMap[cat]++

		logFiles = append(logFiles, LogFileInfo{
			Name:      fileName,
			Size:      info.Size(),
			ModTime:   info.ModTime(),
			Category:  cat,
			SizeHuman: formatFileSize(info.Size()),
		})
	}

	// 按修改时间倒序
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].ModTime.After(logFiles[j].ModTime)
	})

	// 构建分类列表
	categories := []LogCategory{
		{Name: "", Label: "全部", Count: len(logFiles)},
	}
	serverCategoryLabels := map[string]string{
		"ssh":      "SSH认证",
		"system":   "系统日志",
		"nginx":    "Nginx",
		"mysql":    "MySQL",
		"fail2ban": "Fail2ban",
		"kernel":   "内核",
	}

	var cats []string
	for cat := range categoryMap {
		cats = append(cats, cat)
	}
	sort.Strings(cats)

	for _, cat := range cats {
		label := serverCategoryLabels[cat]
		if label == "" {
			label = cat
		}
		categories = append(categories, LogCategory{
			Name:  cat,
			Label: label,
			Count: categoryMap[cat],
		})
	}

	response.Success(c, gin.H{
		"files":      logFiles,
		"categories": categories,
		"total":      len(logFiles),
	})
}

// ReadFile 读取日志文件内容
func (h *SystemLogHandler) ReadFile(c *gin.Context) {
	filename := c.Query("file")
	source := c.DefaultQuery("source", "app")
	if filename == "" {
		response.BadRequest(c, "缺少文件名参数")
		return
	}

	var filePath string
	if source == "server" {
		// 服务器日志：检查白名单
		if _, ok := h.allowedServerLogs[filename]; !ok {
			response.Forbidden(c, "不允许访问该日志文件")
			return
		}
		filePath = filepath.Join(h.serverLogDir, filename)
	} else {
		// 应用日志：安全检查防止路径遍历
		filename = filepath.Base(filename)
		if !strings.HasSuffix(filename, ".log") {
			response.BadRequest(c, "无效的日志文件")
			return
		}
		filePath = filepath.Join(h.appLogDir, filename)
	}

	// 检查文件是否存在
	info, err := os.Stat(filePath)
	if err != nil {
		response.NotFound(c, "日志文件不存在")
		return
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "200"))
	if page < 1 {
		page = 1
	}
	if pageSize < 10 {
		pageSize = 10
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	// 搜索关键词
	keyword := c.Query("keyword")

	// 是否倒序（查看最新的）
	reverse := c.Query("reverse") == "true"

	// 读取文件
	file, err := os.Open(filePath)
	if err != nil {
		response.Error(c, 500, "打开文件失败: "+err.Error())
		return
	}
	defer file.Close()

	var allLines []string
	scanner := bufio.NewScanner(file)
	// 增加缓冲区大小以处理长行
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// 关键词过滤
		if keyword != "" {
			if !strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
				continue
			}
		}

		allLines = append(allLines, line)
	}

	totalLines := len(allLines)

	// 如果倒序，反转数组
	if reverse {
		for i, j := 0, len(allLines)-1; i < j; i, j = i+1, j-1 {
			allLines[i], allLines[j] = allLines[j], allLines[i]
		}
	}

	// 分页
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= len(allLines) {
		start = 0
		end = 0
	}
	if end > len(allLines) {
		end = len(allLines)
	}

	var lines []string
	if start < len(allLines) {
		lines = allLines[start:end]
	}

	response.Success(c, gin.H{
		"file":        filename,
		"size":        info.Size(),
		"size_human":  formatFileSize(info.Size()),
		"mod_time":    info.ModTime(),
		"lines":       lines,
		"total_lines": totalLines,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (totalLines + pageSize - 1) / pageSize,
		"reverse":     reverse,
	})
}

// TailFile 实时查看日志末尾
func (h *SystemLogHandler) TailFile(c *gin.Context) {
	filename := c.Query("file")
	source := c.DefaultQuery("source", "app")
	if filename == "" {
		response.BadRequest(c, "缺少文件名参数")
		return
	}

	var filePath string
	if source == "server" {
		if _, ok := h.allowedServerLogs[filename]; !ok {
			response.Forbidden(c, "不允许访问该日志文件")
			return
		}
		filePath = filepath.Join(h.serverLogDir, filename)
	} else {
		filename = filepath.Base(filename)
		if !strings.HasSuffix(filename, ".log") {
			response.BadRequest(c, "无效的日志文件")
			return
		}
		filePath = filepath.Join(h.appLogDir, filename)
	}

	// 获取行数
	lines, _ := strconv.Atoi(c.DefaultQuery("lines", "100"))
	if lines < 10 {
		lines = 10
	}
	if lines > 1000 {
		lines = 1000
	}

	// 检查文件
	info, err := os.Stat(filePath)
	if err != nil {
		response.NotFound(c, "日志文件不存在")
		return
	}

	// 读取最后 N 行
	content, err := tailFile(filePath, lines)
	if err != nil {
		response.Error(c, 500, "读取文件失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"file":       filename,
		"size":       info.Size(),
		"size_human": formatFileSize(info.Size()),
		"mod_time":   info.ModTime(),
		"lines":      content,
		"count":      len(content),
	})
}

// DownloadFile 下载日志文件
func (h *SystemLogHandler) DownloadFile(c *gin.Context) {
	filename := c.Query("file")
	source := c.DefaultQuery("source", "app")
	if filename == "" {
		response.BadRequest(c, "缺少文件名参数")
		return
	}

	var filePath string
	if source == "server" {
		if _, ok := h.allowedServerLogs[filename]; !ok {
			response.Forbidden(c, "不允许访问该日志文件")
			return
		}
		filePath = filepath.Join(h.serverLogDir, filename)
	} else {
		filename = filepath.Base(filename)
		if !strings.HasSuffix(filename, ".log") {
			response.BadRequest(c, "无效的日志文件")
			return
		}
		filePath = filepath.Join(h.appLogDir, filename)
	}

	// 检查文件
	if _, err := os.Stat(filePath); err != nil {
		response.NotFound(c, "日志文件不存在")
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.File(filePath)
}

// DeleteFile 删除日志文件
func (h *SystemLogHandler) DeleteFile(c *gin.Context) {
	filename := c.Query("file")
	source := c.DefaultQuery("source", "app")
	if filename == "" {
		response.BadRequest(c, "缺少文件名参数")
		return
	}

	// 服务器日志不允许删除
	if source == "server" {
		response.Forbidden(c, "不允许删除服务器日志文件")
		return
	}

	// 应用日志：安全检查
	filename = filepath.Base(filename)
	if !strings.HasSuffix(filename, ".log") {
		response.BadRequest(c, "无效的日志文件")
		return
	}

	filePath := filepath.Join(h.appLogDir, filename)

	// 检查文件
	if _, err := os.Stat(filePath); err != nil {
		response.NotFound(c, "日志文件不存在")
		return
	}

	if err := os.Remove(filePath); err != nil {
		response.Error(c, 500, "删除文件失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "删除成功"})
}

// 解析日志文件名，返回分类和日期
func parseLogFileName(name string) (category, date string) {
	name = strings.TrimSuffix(name, ".log")
	parts := strings.Split(name, "-")

	if len(parts) >= 4 {
		// 格式: category-YYYY-MM-DD
		category = strings.Join(parts[:len(parts)-3], "-")
		date = strings.Join(parts[len(parts)-3:], "-")
	} else {
		category = name
	}

	return
}

// 格式化文件大小
func formatFileSize(size int64) string {
	const (
		B  = 1
		KB = 1024 * B
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// 读取文件最后 N 行
func tailFile(filePath string, n int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) <= n {
		return lines, nil
	}

	return lines[len(lines)-n:], nil
}
