/*
 * 文件作用：HTTP客户端管理，提供连接池复用和代理支持
 * 负责功能：
 *   - 全局HTTP客户端（普通/流式）
 *   - 代理客户端缓存（避免重复创建）
 *   - Chrome TLS指纹支持（绕过TLS检测）
 *   - SOCKS5/HTTP代理支持
 *   - gzip响应自动解压
 *   - 连接池参数配置
 * 重要程度：⭐⭐⭐⭐⭐ 核心（所有上游请求的基础）
 * 依赖模块：model, logger
 */
package adapter

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"go-aiproxy/internal/model"
	"go-aiproxy/pkg/logger"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/proxy"
)

// ========== 全局 HTTP 客户端池 ==========
// 预配置的客户端，复用连接池，避免每次请求都创建新客户端

var (
	// 默认客户端（无代理）- 普通请求
	defaultHTTPClient = &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
	}

	// 默认客户端（无代理）- 流式请求
	defaultStreamClient = &http.Client{
		Timeout: 600 * time.Second, // 10 分钟超时
		Transport: &http.Transport{
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   20,
			IdleConnTimeout:       120 * time.Second,
			DisableCompression:    true,  // 禁用压缩，避免流式解析问题
			ForceAttemptHTTP2:     false, // 禁用 HTTP/2
			ResponseHeaderTimeout: 0,     // 无响应头超时
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
	}

	// 代理客户端缓存（避免每次请求都创建新客户端）
	proxyClientCache     = make(map[string]*http.Client)
	proxyClientCacheLock sync.RWMutex
)

// GetHTTPClient 获取代理感知的 HTTP 客户端
// 如果账户关联了代理，则使用该代理，否则直连
// 使用客户端缓存，避免重复创建
func GetHTTPClient(account *model.Account) *http.Client {
	proxyURL := GetEffectiveProxy(account)
	if proxyURL == "" {
		return defaultHTTPClient
	}
	return getOrCreateProxyClient(proxyURL, false)
}

// GetStreamHTTPClient 获取用于流式请求的 HTTP 客户端
// 使用更长的超时时间（10分钟），适用于 SSE 流式响应
// 使用客户端缓存，避免重复创建
func GetStreamHTTPClient(account *model.Account) *http.Client {
	proxyURL := GetEffectiveProxy(account)
	if proxyURL == "" {
		return defaultStreamClient
	}
	return getOrCreateProxyClient(proxyURL, true)
}

// getOrCreateProxyClient 获取或创建代理客户端（带缓存）
// streaming: true 表示流式客户端，false 表示普通客户端
func getOrCreateProxyClient(proxyURLStr string, streaming bool) *http.Client {
	log := logger.GetLogger("proxy")

	// 缓存键：区分流式和普通客户端
	cacheKey := proxyURLStr
	if streaming {
		cacheKey = "stream:" + proxyURLStr
	}

	// 先尝试从缓存读取
	proxyClientCacheLock.RLock()
	if client, ok := proxyClientCache[cacheKey]; ok {
		proxyClientCacheLock.RUnlock()
		return client
	}
	proxyClientCacheLock.RUnlock()

	// 解析代理 URL
	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		log.Error("解析代理 URL 失败: %s, error: %v", proxyURLStr, err)
		if streaming {
			return defaultStreamClient
		}
		return defaultHTTPClient
	}

	// 根据代理类型创建 Transport
	var transport *http.Transport

	switch proxyURL.Scheme {
	case "http", "https":
		// HTTP/HTTPS 代理
		if streaming {
			transport = &http.Transport{
				Proxy:                 http.ProxyURL(proxyURL),
				MaxIdleConns:          50,
				MaxIdleConnsPerHost:   10,
				IdleConnTimeout:       120 * time.Second,
				DisableCompression:    true,
				ForceAttemptHTTP2:     false,
				ResponseHeaderTimeout: 0,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			}
		} else {
			transport = &http.Transport{
				Proxy:               http.ProxyURL(proxyURL),
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			}
		}

	case "socks5", "socks5h":
		// SOCKS5 代理
		var auth *proxy.Auth
		if proxyURL.User != nil {
			auth = &proxy.Auth{
				User: proxyURL.User.Username(),
			}
			if password, ok := proxyURL.User.Password(); ok {
				auth.Password = password
			}
		}

		dialer, err := proxy.SOCKS5("tcp", proxyURL.Host, auth, proxy.Direct)
		if err != nil {
			log.Error("创建 SOCKS5 dialer 失败: %v", err)
			if streaming {
				return defaultStreamClient
			}
			return defaultHTTPClient
		}

		if streaming {
			transport = &http.Transport{
				Dial:                  dialer.Dial,
				MaxIdleConns:          50,
				MaxIdleConnsPerHost:   10,
				IdleConnTimeout:       120 * time.Second,
				DisableCompression:    true,
				ForceAttemptHTTP2:     false,
				ResponseHeaderTimeout: 0,
			}
		} else {
			transport = &http.Transport{
				Dial:                dialer.Dial,
				MaxIdleConns:        50,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
			}
		}

	default:
		log.Error("不支持的代理类型: %s", proxyURL.Scheme)
		if streaming {
			return defaultStreamClient
		}
		return defaultHTTPClient
	}

	// 创建客户端
	timeout := 120 * time.Second
	if streaming {
		timeout = 600 * time.Second
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	// 存入缓存
	proxyClientCacheLock.Lock()
	proxyClientCache[cacheKey] = client
	proxyClientCacheLock.Unlock()

	log.Debug("创建代理客户端并缓存: %s (streaming=%v)", proxyURLStr, streaming)
	return client
}

// createStreamProxyClient 创建带代理的流式 HTTP 客户端
// Deprecated: 使用 getOrCreateProxyClient 代替
func createStreamProxyClient(proxyURLStr string) *http.Client {
	log := logger.GetLogger("proxy")

	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		log.Error("解析代理 URL 失败: %s, error: %v", proxyURLStr, err)
		return &http.Client{
			Timeout: 600 * time.Second,
			Transport: &http.Transport{
				DisableCompression:    true,
				ForceAttemptHTTP2:     false,
				ResponseHeaderTimeout: 0,
				IdleConnTimeout:       120 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
		}
	}

	transport := &http.Transport{
		Proxy:              http.ProxyURL(proxyURL),
		DisableCompression: true,  // 禁用响应压缩
		ForceAttemptHTTP2:  false, // 禁用 HTTP/2
		// 自定义 Dialer 设置 TCP KeepAlive
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second, // TCP 保活探测间隔
		}).DialContext,
		// 响应头超时设为 0，允许无限等待
		ResponseHeaderTimeout: 0,
		// 空闲连接超时
		IdleConnTimeout: 120 * time.Second,
	}

	log.Debug("使用代理 (流式): %s", proxyURLStr)
	return &http.Client{
		Transport: transport,
		Timeout:   600 * time.Second, // 10 分钟超时
	}
}

// GetEffectiveProxy 获取生效的代理 URL
// 如果账户关联了代理，则返回代理 URL，否则返回空（直连）
func GetEffectiveProxy(account *model.Account) string {
	if account == nil {
		return ""
	}

	// 检查账户是否关联了代理
	if account.Proxy != nil && account.Proxy.Enabled {
		return account.Proxy.GetURL()
	}

	return ""
}

// createProxyClient 创建带代理的 HTTP 客户端
func createProxyClient(proxyURLStr string) *http.Client {
	log := logger.GetLogger("proxy")

	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		log.Error("解析代理 URL 失败: %s, error: %v", proxyURLStr, err)
		return &http.Client{
			Timeout: 120 * time.Second,
		}
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	log.Debug("使用代理: %s", proxyURLStr)
	return &http.Client{
		Transport: transport,
		Timeout:   120 * time.Second,
	}
}

// ReadResponseBody 读取响应体，自动处理 gzip 解压
// 如果响应头包含 Content-Encoding: gzip，则自动解压
func ReadResponseBody(resp *http.Response) ([]byte, error) {
	if resp.Body == nil {
		return nil, nil
	}

	var reader io.Reader = resp.Body

	// 检查是否为 gzip 编码
	contentEncoding := resp.Header.Get("Content-Encoding")
	if strings.EqualFold(contentEncoding, "gzip") {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			// gzip 解压失败，可能数据不是真正的 gzip，尝试原始读取
			log := logger.GetLogger("proxy")
			log.Warn("gzip 解压失败，尝试原始读取: %v", err)
			// 重置 body 已经被消费，这里无法恢复，返回错误
			return nil, err
		}
		defer gzReader.Close()
		reader = gzReader
	}

	// 也检查内容是否以 gzip magic bytes 开头（有时服务端不设置 Content-Encoding）
	// 先读取到 buffer，检查前两个字节
	var buf bytes.Buffer
	_, err := io.Copy(&buf, reader)
	if err != nil {
		return nil, err
	}

	data := buf.Bytes()

	// 如果没有通过 Content-Encoding 检测到 gzip，但内容以 gzip magic bytes 开头
	if !strings.EqualFold(contentEncoding, "gzip") && len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		log := logger.GetLogger("proxy")
		log.Debug("检测到 gzip magic bytes，进行解压")
		gzReader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			log.Warn("gzip magic bytes 检测后解压失败: %v", err)
			return data, nil // 返回原始数据
		}
		defer gzReader.Close()

		decompressed, err := io.ReadAll(gzReader)
		if err != nil {
			log.Warn("gzip 解压读取失败: %v", err)
			return data, nil // 返回原始数据
		}
		return decompressed, nil
	}

	return data, nil
}

// ========== Chrome TLS 指纹支持 ==========

// ProxyConfig 代理配置（用于 Chrome TLS 客户端）
type ProxyConfig struct {
	Type     string // socks5, http, https
	Host     string
	Port     int
	Username string
	Password string
}

// GetChromeTLSClient 获取带 Chrome TLS 指纹的 HTTP 客户端
// 用于需要绕过 TLS 指纹检测的场景（如 chatgpt.com, claude.ai）
func GetChromeTLSClient(account *model.Account) *http.Client {
	var proxyConfig *ProxyConfig
	if account != nil && account.Proxy != nil && account.Proxy.Enabled {
		proxyConfig = &ProxyConfig{
			Type:     account.Proxy.Type,
			Host:     account.Proxy.Host,
			Port:     account.Proxy.Port,
			Username: account.Proxy.Username,
			Password: account.Proxy.Password,
		}
	}
	return createChromeTLSClient(proxyConfig)
}

// GetChromeTLSClientWithProxy 获取带 Chrome TLS 指纹的 HTTP 客户端（指定代理）
func GetChromeTLSClientWithProxy(proxyConfig *ProxyConfig) *http.Client {
	return createChromeTLSClient(proxyConfig)
}

// createChromeTLSClient 创建带 Chrome TLS 指纹的 HTTP 客户端
func createChromeTLSClient(proxyConfig *ProxyConfig) *http.Client {
	// 创建自定义的 DialTLS 函数，使用 Chrome TLS 指纹
	dialTLS := func(network, addr string) (net.Conn, error) {
		return dialTLSWithChrome(network, addr, proxyConfig)
	}

	transport := &http.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialTLS(network, addr)
		},
		// 对于非 TLS 连接，仍然需要配置代理
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialWithProxy(network, addr, proxyConfig)
		},
	}

	return &http.Client{
		Transport: transport,
		Timeout:   120 * time.Second,
		// 不自动跟随重定向
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// dialWithProxy 通过代理建立普通连接
func dialWithProxy(network, addr string, proxyConfig *ProxyConfig) (net.Conn, error) {
	if proxyConfig == nil || proxyConfig.Host == "" {
		return net.DialTimeout(network, addr, 30*time.Second)
	}

	switch proxyConfig.Type {
	case "socks5":
		var auth *proxy.Auth
		if proxyConfig.Username != "" {
			auth = &proxy.Auth{
				User:     proxyConfig.Username,
				Password: proxyConfig.Password,
			}
		}
		dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("%s:%d", proxyConfig.Host, proxyConfig.Port), auth, proxy.Direct)
		if err != nil {
			return nil, err
		}
		return dialer.Dial(network, addr)
	case "http", "https":
		// HTTP 代理使用 CONNECT 方法
		proxyAddr := fmt.Sprintf("%s:%d", proxyConfig.Host, proxyConfig.Port)
		conn, err := net.DialTimeout("tcp", proxyAddr, 30*time.Second)
		if err != nil {
			return nil, err
		}

		// 发送 CONNECT 请求
		connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", addr, addr)
		if proxyConfig.Username != "" {
			auth := base64.StdEncoding.EncodeToString([]byte(proxyConfig.Username + ":" + proxyConfig.Password))
			connectReq += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", auth)
		}
		connectReq += "\r\n"

		_, err = conn.Write([]byte(connectReq))
		if err != nil {
			conn.Close()
			return nil, err
		}

		// 读取响应
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			conn.Close()
			return nil, err
		}

		if !strings.Contains(string(buf[:n]), "200") {
			conn.Close()
			return nil, fmt.Errorf("proxy CONNECT failed: %s", string(buf[:n]))
		}

		return conn, nil
	default:
		return net.DialTimeout(network, addr, 30*time.Second)
	}
}

// dialTLSWithChrome 使用 Chrome TLS 指纹建立 TLS 连接
func dialTLSWithChrome(network, addr string, proxyConfig *ProxyConfig) (net.Conn, error) {
	// 先建立普通 TCP 连接（可能通过代理）
	conn, err := dialWithProxy(network, addr, proxyConfig)
	if err != nil {
		return nil, err
	}

	// 从地址中提取主机名
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	// 创建 uTLS 配置
	// 注意：只使用 HTTP/1.1，避免 HTTP/2 协议不匹配问题
	config := &utls.Config{
		ServerName:         host,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
		NextProtos:         []string{"http/1.1"}, // 强制使用 HTTP/1.1
	}

	// 创建 uTLS 连接，使用自定义的 Chrome 指纹（仅 HTTP/1.1）
	tlsConn := utls.UClient(conn, config, utls.HelloCustom)

	// 应用 Chrome 120 指纹，但修改 ALPN 为仅 HTTP/1.1
	spec, err := utls.UTLSIdToSpec(utls.HelloChrome_120)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get Chrome spec: %v", err)
	}

	// 修改 ALPN 扩展，只保留 http/1.1
	for i, ext := range spec.Extensions {
		if alpn, ok := ext.(*utls.ALPNExtension); ok {
			spec.Extensions[i] = &utls.ALPNExtension{
				AlpnProtocols: []string{"http/1.1"},
			}
			_ = alpn // 避免未使用警告
			break
		}
	}

	if err := tlsConn.ApplyPreset(&spec); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to apply spec: %v", err)
	}

	// 执行 TLS 握手
	if err := tlsConn.Handshake(); err != nil {
		conn.Close()
		return nil, err
	}

	return tlsConn, nil
}

// NeedsChromeTLS 判断目标是否需要 Chrome TLS 指纹
// 用于自动选择合适的 HTTP 客户端
func NeedsChromeTLS(targetURL string) bool {
	// 需要 Chrome TLS 的域名列表
	chromeTLSDomains := []string{
		"chatgpt.com",
		"claude.ai",
		"api.openai.com",    // OpenAI 可能检测 TLS 指纹
		"auth.openai.com",   // OpenAI OAuth
	}

	for _, domain := range chromeTLSDomains {
		if strings.Contains(targetURL, domain) {
			return true
		}
	}
	return false
}

// GetSmartHTTPClient 智能选择 HTTP 客户端
// 根据目标 URL 自动选择是否使用 Chrome TLS
func GetSmartHTTPClient(account *model.Account, targetURL string) *http.Client {
	if NeedsChromeTLS(targetURL) {
		return GetChromeTLSClient(account)
	}
	return GetHTTPClient(account)
}

// ========== 缓存管理函数 ==========

// ClearProxyClientCache 清理指定代理的客户端缓存
// 当代理配置变更时调用
func ClearProxyClientCache(proxyURL string) {
	proxyClientCacheLock.Lock()
	defer proxyClientCacheLock.Unlock()

	// 删除普通和流式客户端
	delete(proxyClientCache, proxyURL)
	delete(proxyClientCache, "stream:"+proxyURL)

	log := logger.GetLogger("proxy")
	log.Debug("清理代理客户端缓存: %s", proxyURL)
}

// ClearAllProxyClientCache 清理所有代理客户端缓存
func ClearAllProxyClientCache() {
	proxyClientCacheLock.Lock()
	defer proxyClientCacheLock.Unlock()

	count := len(proxyClientCache)
	proxyClientCache = make(map[string]*http.Client)

	log := logger.GetLogger("proxy")
	log.Info("清理所有代理客户端缓存，共 %d 个", count)
}

// GetProxyClientCacheStats 获取代理客户端缓存统计
func GetProxyClientCacheStats() map[string]interface{} {
	proxyClientCacheLock.RLock()
	defer proxyClientCacheLock.RUnlock()

	normalCount := 0
	streamCount := 0
	for key := range proxyClientCache {
		if strings.HasPrefix(key, "stream:") {
			streamCount++
		} else {
			normalCount++
		}
	}

	return map[string]interface{}{
		"total":   len(proxyClientCache),
		"normal":  normalCount,
		"stream":  streamCount,
		"proxies": getProxyCacheKeys(),
	}
}

// getProxyCacheKeys 获取缓存的代理 URL 列表（不暴露完整 URL）
func getProxyCacheKeys() []string {
	keys := make([]string, 0, len(proxyClientCache))
	seen := make(map[string]bool)

	for key := range proxyClientCache {
		// 去掉 stream: 前缀
		proxyURL := strings.TrimPrefix(key, "stream:")
		if !seen[proxyURL] {
			// 只显示代理类型和主机，不显示密码
			if u, err := url.Parse(proxyURL); err == nil {
				maskedURL := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
				keys = append(keys, maskedURL)
				seen[proxyURL] = true
			}
		}
	}

	return keys
}
