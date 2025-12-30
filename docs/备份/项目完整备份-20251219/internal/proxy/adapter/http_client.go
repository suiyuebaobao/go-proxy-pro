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
	"time"

	"go-aiproxy/internal/model"
	"go-aiproxy/pkg/logger"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/proxy"
)

// GetHTTPClient 获取代理感知的 HTTP 客户端
// 如果账户关联了代理，则使用该代理，否则直连
func GetHTTPClient(account *model.Account) *http.Client {
	proxyURL := GetEffectiveProxy(account)
	if proxyURL == "" {
		return &http.Client{
			Timeout: 120 * time.Second,
		}
	}
	return createProxyClient(proxyURL)
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
