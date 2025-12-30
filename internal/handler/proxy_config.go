/*
 * 文件作用：代理配置处理器，管理HTTP/SOCKS5代理服务器配置
 * 负责功能：
 *   - 代理配置列表查询
 *   - 代理配置CRUD
 *   - 代理启用/禁用
 *   - 默认代理设置
 *   - 代理连通性测试
 * 重要程度：⭐⭐⭐ 一般（代理配置管理）
 * 依赖模块：service, model
 */
package handler

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/proxy"
)

// ListProxyConfigs 获取代理配置列表
func ListProxyConfigs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	keyword := c.Query("keyword")

	proxies, total, err := service.GetProxyService().List(page, pageSize, keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": proxies,
		"total": total,
		"page":  page,
	})
}

// GetProxyConfig 获取单个代理配置
func GetProxyConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	proxy, err := service.GetProxyService().GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if proxy == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "代理不存在"})
		return
	}

	c.JSON(http.StatusOK, proxy)
}

// CreateProxyConfig 创建代理配置
func CreateProxyConfig(c *gin.Context) {
	var proxy model.Proxy
	if err := c.ShouldBindJSON(&proxy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置默认值
	if proxy.Type == "" {
		proxy.Type = model.ProxyTypeHTTP
	}

	if err := service.GetProxyService().Create(&proxy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, proxy)
}

// UpdateProxyConfig 更新代理配置
func UpdateProxyConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	existing, err := service.GetProxyService().GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "代理不存在"})
		return
	}

	var proxy model.Proxy
	if err := c.ShouldBindJSON(&proxy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	proxy.ID = uint(id)
	proxy.CreatedAt = existing.CreatedAt

	if err := service.GetProxyService().Update(&proxy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, proxy)
}

// DeleteProxyConfig 删除代理配置
func DeleteProxyConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	if err := service.GetProxyService().Delete(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ToggleProxyConfigEnabled 切换代理配置启用状态
func ToggleProxyConfigEnabled(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	if err := service.GetProxyService().ToggleEnabled(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "操作成功"})
}

// GetEnabledProxyConfigs 获取所有启用的代理配置 (用于下拉选择)
func GetEnabledProxyConfigs(c *gin.Context) {
	proxies, err := service.GetProxyService().GetEnabledProxies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": proxies})
}

// SetDefaultProxyConfig 设置默认代理
func SetDefaultProxyConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	// 验证代理存在且启用
	proxy, err := service.GetProxyService().GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if proxy == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "代理不存在"})
		return
	}
	if !proxy.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能将启用的代理设置为默认"})
		return
	}

	if err := service.GetProxyService().SetDefault(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "设置成功"})
}

// ClearDefaultProxyConfig 清除默认代理
func ClearDefaultProxyConfig(c *gin.Context) {
	if err := service.GetProxyService().ClearDefault(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "清除成功"})
}

// GetDefaultProxyConfig 获取当前默认代理
func GetDefaultProxyConfig(c *gin.Context) {
	proxy, err := service.GetProxyService().GetDefaultProxy()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": proxy})
}

// TestProxyConnectivityRequest 测试代理连通性请求
type TestProxyConnectivityRequest struct {
	ID       uint   `json:"id"`       // 代理ID（可选，有则保存测试结果）
	Type     string `json:"type"`     // http, https, socks5
	Host     string `json:"host"`     // 代理主机
	Port     int    `json:"port"`     // 代理端口
	Username string `json:"username"` // 用户名（可选）
	Password string `json:"password"` // 密码（可选）
}

// TestProxyConnectivity 测试代理连通性
func TestProxyConnectivity(c *gin.Context) {
	var req TestProxyConnectivityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Host == "" || req.Port == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "主机和端口不能为空"})
		return
	}

	if req.Type == "" {
		req.Type = "http"
	}

	start := time.Now()

	// 测试目标 URL（使用 Google 或 Cloudflare 来测试代理，更稳定）
	testURLs := []string{
		"https://www.google.com/generate_204",
		"https://cp.cloudflare.com/",
		"https://www.gstatic.com/generate_204",
	}

	var err error
	var resp *http.Response
	var lastErr error

	for _, testURL := range testURLs {
		switch req.Type {
		case "socks5":
			resp, err = testSocks5Proxy(req.Host, req.Port, req.Username, req.Password, testURL)
		case "http", "https":
			resp, err = testHTTPProxy(req.Type, req.Host, req.Port, req.Username, req.Password, testURL)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的代理类型: " + req.Type})
			return
		}

		if err == nil && resp != nil {
			// 204 No Content 或 200 OK 都表示成功
			if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				latency := int(time.Since(start).Milliseconds())

				// 如果有 ID，保存测试结果到数据库
				if req.ID > 0 {
					service.GetProxyService().UpdateTestStatus(req.ID, "success", latency, "")
				}

				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"message": "连接成功",
					"latency": latency,
				})
				return
			}
			resp.Body.Close()
		}
		if err != nil {
			lastErr = err
		}
	}

	latency := int(time.Since(start).Milliseconds())

	if lastErr != nil {
		// 如果有 ID，保存测试失败结果到数据库
		if req.ID > 0 {
			service.GetProxyService().UpdateTestStatus(req.ID, "failed", latency, lastErr.Error())
		}

		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   lastErr.Error(),
			"latency": latency,
		})
		return
	}

	// 如果有 ID，保存测试失败结果到数据库
	if req.ID > 0 {
		service.GetProxyService().UpdateTestStatus(req.ID, "failed", latency, "无法连接到测试目标")
	}

	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"error":   "无法连接到测试目标",
		"latency": latency,
	})
}

// testHTTPProxy 测试 HTTP/HTTPS 代理
func testHTTPProxy(proxyType, host string, port int, username, password, testURL string) (*http.Response, error) {
	proxyURL := fmt.Sprintf("%s://%s:%d", proxyType, host, port)
	if username != "" && password != "" {
		proxyURL = fmt.Sprintf("%s://%s:%s@%s:%d", proxyType, username, password, host, port)
	}

	proxyURLParsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("解析代理 URL 失败: %v", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURLParsed),
		},
		Timeout: 10 * time.Second,
	}

	return client.Get(testURL)
}

// testSocks5Proxy 测试 SOCKS5 代理
func testSocks5Proxy(host string, port int, username, password, testURL string) (*http.Response, error) {
	var auth *proxy.Auth
	if username != "" && password != "" {
		auth = &proxy.Auth{
			User:     username,
			Password: password,
		}
	}

	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%d", host, port), auth, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("创建 SOCKS5 代理失败: %v", err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			},
		},
		Timeout: 10 * time.Second,
	}

	return client.Get(testURL)
}
