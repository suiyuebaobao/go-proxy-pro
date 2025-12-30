package handler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"go-aiproxy/internal/service"
	"go-aiproxy/pkg/logger"

	"github.com/gin-gonic/gin"
	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/proxy"
)

// OAuth 配置常量 - 使用与 clove 项目相同的配置
const (
	// Claude OAuth 配置
	ClaudeAIURL       = "https://claude.ai"
	OAuthClientID     = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	OAuthRedirectURI  = "https://console.anthropic.com/oauth/code/callback"
	OAuthTokenURL     = "https://console.anthropic.com/v1/oauth/token"
	// OAuth 授权端点 URL: /v1/oauth/{organization_uuid}/authorize
	OAuthAuthorizeURL = "https://claude.ai/v1/oauth/%s/authorize"

	// OpenAI OAuth 配置 - 参考 claude-relay 的 openaiAccounts.js
	// Codex CLI 的官方 CLIENT_ID
	OpenAIClientID     = "app_EMoamEEZ73f0CkXaXp7hrann"
	OpenAITokenURL     = "https://auth.openai.com/oauth/token"
	OpenAIAuthorizeURL = "https://auth.openai.com/oauth/authorize" // 注意：路径是 /oauth/authorize 不是 /authorize
	OpenAIRedirectURI  = "http://localhost:1455/auth/callback"
	OpenAIScope        = "openid profile email offline_access" // offline_access 用于获取 refresh_token
)

// OAuthHandler OAuth 处理器
type OAuthHandler struct {
	sessions sync.Map // 存储 OAuth 会话状态
}

// OAuthSession OAuth 会话
type OAuthSession struct {
	Verifier  string    `json:"verifier"`
	Challenge string    `json:"challenge"`
	State     string    `json:"state"`
	Platform  string    `json:"platform"`
	CreatedAt time.Time `json:"created_at"`
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	Type     string `json:"type"`     // socks5, http, https
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// NewOAuthHandler 创建 OAuth 处理器
func NewOAuthHandler() *OAuthHandler {
	return &OAuthHandler{}
}

// getEffectiveProxy 获取有效代理配置（优先使用请��指定的，否则使用默认代理）
func getEffectiveProxy(requestProxy *ProxyConfig) *ProxyConfig {
	// 如果请求中指定了代理且有效，使用请求的代理
	if requestProxy != nil && requestProxy.Host != "" && requestProxy.Port > 0 {
		return requestProxy
	}

	// 否则尝试获取默认代理
	defaultProxy, err := service.GetProxyService().GetDefaultProxy()
	if err != nil {
		logger.Warn("[oauth] 获取默认代理失败: %v", err)
		return nil
	}

	if defaultProxy != nil {
		logger.Info("[oauth] 使用默认代理: %s (%s:%d)", defaultProxy.Name, defaultProxy.Host, defaultProxy.Port)
		return &ProxyConfig{
			Type:     defaultProxy.Type,
			Host:     defaultProxy.Host,
			Port:     defaultProxy.Port,
			Username: defaultProxy.Username,
			Password: defaultProxy.Password,
		}
	}

	return nil
}

// GenerateURL 生成 OAuth 授权 URL
func (h *OAuthHandler) GenerateURL(c *gin.Context) {
	var req struct {
		Platform string       `json:"platform"`
		Proxy    *ProxyConfig `json:"proxy,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 生成 PKCE
	verifier, challenge := generatePKCE()
	state := generateRandomString(32)
	sessionID := generateRandomString(16)

	// 保存会话
	h.sessions.Store(sessionID, &OAuthSession{
		Verifier:  verifier,
		Challenge: challenge,
		State:     state,
		Platform:  req.Platform,
		CreatedAt: time.Now(),
	})

	// 根据平台生成不同的授权 URL
	var authURL string
	switch req.Platform {
	case "claude", "claude-official":
		// Claude OAuth URL 需要通过 API 获取组织 UUID 后才能生成完整 URL
		// 这里返回一个提示，让前端使用 SessionKey 方式
		authURL = fmt.Sprintf(
			"%s/oauth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=user:profile+user:inference&state=%s&code_challenge=%s&code_challenge_method=S256",
			ClaudeAIURL,
			OAuthClientID,
			url.QueryEscape(OAuthRedirectURI),
			state,
			challenge,
		)
	case "openai":
		// OpenAI OAuth - 使用 Codex CLI 的官方 CLIENT_ID
		// 参考 claude-relay 的 openaiAccounts.js
		// 关键参数：
		// - id_token_add_organizations=true : 获取组织信息
		// - codex_cli_simplified_flow=true : 使用 Codex CLI 的简化流程
		authURL = fmt.Sprintf(
			"%s?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s&code_challenge=%s&code_challenge_method=S256&id_token_add_organizations=true&codex_cli_simplified_flow=true",
			OpenAIAuthorizeURL,
			OpenAIClientID,
			url.QueryEscape(OpenAIRedirectURI),
			url.QueryEscape(OpenAIScope),
			state,
			challenge,
		)
	case "gemini":
		// Gemini/Google OAuth
		authURL = fmt.Sprintf(
			"https://accounts.google.com/o/oauth2/v2/auth?response_type=code&client_id=YOUR_CLIENT_ID&redirect_uri=%s&scope=openid+email&state=%s&code_challenge=%s&code_challenge_method=S256",
			url.QueryEscape("http://localhost:1455/auth/callback"),
			state,
			challenge,
		)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的平台"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"auth_url":   authURL,
			"session_id": sessionID,
		},
	})
}

// Exchange 交换授权码
func (h *OAuthHandler) Exchange(c *gin.Context) {
	var req struct {
		Platform  string       `json:"platform"`
		Code      string       `json:"code"`
		SessionID string       `json:"session_id"`
		Proxy     *ProxyConfig `json:"proxy,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 获取会话
	sessionData, ok := h.sessions.Load(req.SessionID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的会话 ID"})
		return
	}
	session := sessionData.(*OAuthSession)

	// 删除会话（一次性使用）
	h.sessions.Delete(req.SessionID)

	// 检查会话是否过期（10分钟）
	if time.Since(session.CreatedAt) > 10*time.Minute {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话已过期"})
		return
	}

	// 获取有效代理
	effectiveProxy := getEffectiveProxy(req.Proxy)

	// 根据平台交换 token
	var tokenData map[string]interface{}
	var err error

	switch req.Platform {
	case "claude", "claude-official":
		tokenData, err = exchangeClaudeToken(req.Code, session.Verifier, effectiveProxy)
	case "openai":
		tokenData, err = exchangeOpenAIToken(req.Code, session.Verifier, effectiveProxy)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的平台"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Token 交换失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": tokenData})
}

// CookieAuth 使用 Cookie/SessionKey 进行授权
func (h *OAuthHandler) CookieAuth(c *gin.Context) {
	var req struct {
		Platform   string       `json:"platform"`
		SessionKey string       `json:"session_key"`
		Proxy      *ProxyConfig `json:"proxy,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	if req.SessionKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SessionKey 不能为空"})
		return
	}

	// 获取有效代理（优先请求指定的，否则使用默认代理）
	effectiveProxy := getEffectiveProxy(req.Proxy)

	switch req.Platform {
	case "claude", "claude-official":
		// 使用 SessionKey 获取 OAuth Token
		result, err := authenticateWithSessionKey(req.SessionKey, effectiveProxy)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("授权失败: %v", err)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": result})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "此平台不支持 SessionKey 授权"})
	}
}

// authenticateWithSessionKey 使用 SessionKey 进行 Claude OAuth 授权
func authenticateWithSessionKey(sessionKey string, proxyConfig *ProxyConfig) (map[string]interface{}, error) {
	logger.Info("[oauth] 开始 SessionKey 认证流程")

	// 1. 获取组织信息
	logger.Info("[oauth] 步骤1: 获取组织信息")
	orgUUID, capabilities, err := getOrganizationInfo(sessionKey, proxyConfig)
	if err != nil {
		logger.Error("[oauth] 获取组织信息失败: %v", err)
		return nil, fmt.Errorf("获取组织信息失败: %v", err)
	}
	logger.Info("[oauth] 获取到组织 UUID: %s, 能力: %v", orgUUID, capabilities)

	// 2. 生成 PKCE
	verifier, challenge := generatePKCE()
	state := generateRandomString(32)
	logger.Info("[oauth] 步骤2: 生成 PKCE, state: %s", state[:8]+"...")

	// 3. 使用 Cookie 获取授权码
	logger.Info("[oauth] 步骤3: 获取授权码")
	authCode, err := authorizeWithCookie(sessionKey, orgUUID, challenge, state, proxyConfig)
	if err != nil {
		logger.Error("[oauth] 获取授权码失败: %v", err)
		return nil, fmt.Errorf("获取授权码失败: %v", err)
	}
	logger.Info("[oauth] 获取到授权码: %s...", authCode[:min(20, len(authCode))])

	// 4. 交换 Token
	logger.Info("[oauth] 步骤4: 交换 Token")
	tokenData, err := exchangeClaudeToken(authCode, verifier, proxyConfig)
	if err != nil {
		logger.Error("[oauth] Token 交换失败: %v", err)
		return nil, fmt.Errorf("Token 交换失败: %v", err)
	}
	logger.Info("[oauth] Token 交换成功")

	// 记录返回的 token 信息（隐藏敏感部分）
	if accessToken, ok := tokenData["access_token"].(string); ok && len(accessToken) > 20 {
		logger.Info("[oauth] 获取到 access_token: %s...%s (长度: %d)", accessToken[:10], accessToken[len(accessToken)-5:], len(accessToken))
	}
	if refreshToken, ok := tokenData["refresh_token"].(string); ok && len(refreshToken) > 20 {
		logger.Info("[oauth] 获取到 refresh_token: %s...%s (长度: %d)", refreshToken[:10], refreshToken[len(refreshToken)-5:], len(refreshToken))
	}
	if expiresIn, ok := tokenData["expires_in"]; ok {
		logger.Info("[oauth] token 过期时间: %v 秒", expiresIn)
	}
	if tokenType, ok := tokenData["token_type"]; ok {
		logger.Info("[oauth] token 类型: %v", tokenType)
	}

	// 添加额外信息
	tokenData["organization_uuid"] = orgUUID
	tokenData["capabilities"] = capabilities

	return tokenData, nil
}

// setChromeHeaders 设置完整的 Chrome 浏览器请求头
func setChromeHeaders(req *http.Request, cookie string) {
	// Chrome 120 完整请求头模拟
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Origin", ClaudeAIURL)
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Referer", ClaudeAIURL+"/new")

	// Chrome 特有的 sec-ch-* 请求头
	req.Header.Set("sec-ch-ua", `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	// 完整的 Chrome User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
}

// formatHeaders 格式化请求头用于日志（隐藏敏感信息）
func formatHeaders(headers http.Header) string {
	result := make([]string, 0)
	for key, values := range headers {
		value := strings.Join(values, ",")
		// 隐藏 Cookie 的值
		if strings.ToLower(key) == "cookie" {
			if len(value) > 20 {
				value = value[:20] + "..."
			}
		}
		result = append(result, fmt.Sprintf("%s: %s", key, value))
	}
	return strings.Join(result, " | ")
}

// getOrganizationInfo 获取组织信息
func getOrganizationInfo(sessionKey string, proxyConfig *ProxyConfig) (string, []string, error) {
	client := createHTTPClient(proxyConfig)

	req, err := http.NewRequest("GET", ClaudeAIURL+"/api/organizations", nil)
	if err != nil {
		return "", nil, err
	}

	// 设置 Cookie
	cookie := sessionKey
	if !strings.HasPrefix(sessionKey, "sessionKey=") {
		cookie = "sessionKey=" + sessionKey
	}

	// 设置完整的 Chrome 请求头
	setChromeHeaders(req, cookie)

	// 记录请求头用于调试
	logger.Debug("[oauth] 组织信息请求头: %v", formatHeaders(req.Header))

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var orgs []struct {
		UUID         string   `json:"uuid"`
		Capabilities []string `json:"capabilities"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return "", nil, err
	}

	// 找到具有 chat 能力的组织
	for _, org := range orgs {
		for _, cap := range org.Capabilities {
			if cap == "chat" {
				return org.UUID, org.Capabilities, nil
			}
		}
	}

	return "", nil, fmt.Errorf("未找到有效的组织")
}

// authorizeWithCookie 使用 Cookie 获取授权码
func authorizeWithCookie(sessionKey, orgUUID, challenge, state string, proxyConfig *ProxyConfig) (string, error) {
	client := createHTTPClient(proxyConfig)

	authorizeURL := fmt.Sprintf(OAuthAuthorizeURL, orgUUID)
	logger.Info("[oauth] 授权 URL: %s", authorizeURL)

	payload := map[string]interface{}{
		"response_type":         "code",
		"client_id":             OAuthClientID,
		"organization_uuid":     orgUUID,
		"redirect_uri":          OAuthRedirectURI,
		"scope":                 "user:profile user:inference",
		"state":                 state,
		"code_challenge":        challenge,
		"code_challenge_method": "S256",
	}

	jsonBody, _ := json.Marshal(payload)
	logger.Debug("[oauth] 授权请求 payload: %s", string(jsonBody))

	req, err := http.NewRequest("POST", authorizeURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", err
	}

	// 设置 Cookie
	cookie := sessionKey
	if !strings.HasPrefix(sessionKey, "sessionKey=") {
		cookie = "sessionKey=" + sessionKey
	}

	// 设置完整的 Chrome 请求头
	setChromeHeaders(req, cookie)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		logger.Error("[oauth] 授权请求失败: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	logger.Info("[oauth] 授权响应状态: %d, 长度: %d", resp.StatusCode, len(body))

	if resp.StatusCode != http.StatusOK {
		// 检查是否是 HTML 错误页面
		bodyStr := string(body)
		if len(bodyStr) > 500 {
			bodyStr = bodyStr[:500] + "..."
		}
		logger.Error("[oauth] 授权失败响应: %s", bodyStr)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, bodyStr)
	}

	var result struct {
		RedirectURI string `json:"redirect_uri"`
		Code        string `json:"code"` // 有些 API 直接返回 code
	}

	if err := json.Unmarshal(body, &result); err != nil {
		logger.Error("[oauth] 解析授权响应失败: %v, body: %s", err, string(body))
		return "", err
	}

	// 优先使用直接返回的 code
	if result.Code != "" {
		logger.Info("[oauth] 直接获取到 code")
		return result.Code, nil
	}

	// 从 redirect_uri 提取 code
	if result.RedirectURI == "" {
		return "", fmt.Errorf("响应中没有 redirect_uri 或 code")
	}

	parsedURL, err := url.Parse(result.RedirectURI)
	if err != nil {
		return "", err
	}

	code := parsedURL.Query().Get("code")
	responseState := parsedURL.Query().Get("state")

	if code == "" {
		return "", fmt.Errorf("未获取到授权码")
	}

	// 返回 code#state 格式
	if responseState != "" {
		return code + "#" + responseState, nil
	}
	return code, nil
}

// exchangeClaudeToken 交换 Claude Token
func exchangeClaudeToken(code, verifier string, proxyConfig *ProxyConfig) (map[string]interface{}, error) {
	client := createHTTPClient(proxyConfig)

	// 解析 code#state 格式
	parts := strings.Split(code, "#")
	authCode := parts[0]
	var state string
	if len(parts) > 1 {
		state = parts[1]
	}

	payload := map[string]interface{}{
		"code":          authCode,
		"grant_type":    "authorization_code",
		"client_id":     OAuthClientID,
		"redirect_uri":  OAuthRedirectURI,
		"code_verifier": verifier,
	}
	if state != "" {
		payload["state"] = state
	}

	jsonBody, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", OAuthTokenURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenData); err != nil {
		return nil, err
	}

	return tokenData, nil
}

// exchangeOpenAIToken 交换 OpenAI Token
// 参考 claude-relay 的 openaiAccounts.js exchange-code
func exchangeOpenAIToken(code, verifier string, proxyConfig *ProxyConfig) (map[string]interface{}, error) {
	client := createHTTPClient(proxyConfig)

	// 准备请求数据 - 使用 form urlencoded 格式
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", OpenAIClientID)
	data.Set("code", code)
	data.Set("redirect_uri", OpenAIRedirectURI)
	data.Set("code_verifier", verifier)

	req, err := http.NewRequest("POST", OpenAITokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	logger.Info("[oauth] OpenAI Token 交换请求 - URL: %s", OpenAITokenURL)

	resp, err := client.Do(req)
	if err != nil {
		logger.Error("[oauth] OpenAI Token 请求失败: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	logger.Debug("[oauth] OpenAI Token 响应状态: %d, 长度: %d", resp.StatusCode, len(body))

	if resp.StatusCode != http.StatusOK {
		logger.Error("[oauth] OpenAI Token 交换失败: HTTP %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenData map[string]interface{}
	if err := json.Unmarshal(body, &tokenData); err != nil {
		return nil, err
	}

	// 记录返回的 token 信息（隐藏敏感部分）
	if accessToken, ok := tokenData["access_token"].(string); ok && len(accessToken) > 20 {
		logger.Info("[oauth] OpenAI 获取到 access_token: %s...%s (长度: %d)", accessToken[:10], accessToken[len(accessToken)-5:], len(accessToken))
	}
	if refreshToken, ok := tokenData["refresh_token"].(string); ok && len(refreshToken) > 20 {
		logger.Info("[oauth] OpenAI 获取到 refresh_token: %s...%s (长度: %d)", refreshToken[:10], refreshToken[len(refreshToken)-5:], len(refreshToken))
	}
	if expiresIn, ok := tokenData["expires_in"]; ok {
		logger.Info("[oauth] OpenAI token 过期时间: %v 秒", expiresIn)
	}

	return tokenData, nil
}

// createHTTPClient 创建带 Chrome TLS 指纹的 HTTP 客户端
func createHTTPClient(proxyConfig *ProxyConfig) *http.Client {
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
		Timeout:   30 * time.Second,
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

// generatePKCE 生成 PKCE 验证器和挑战码
func generatePKCE() (verifier, challenge string) {
	// 生成 32 字节随机数作为 verifier
	bytes := make([]byte, 32)
	rand.Read(bytes)
	verifier = base64.RawURLEncoding.EncodeToString(bytes)

	// 生成 SHA256 哈希作为 challenge
	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])

	return verifier, challenge
}

// generateRandomString 生成随机字符串
func generateRandomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.RawURLEncoding.EncodeToString(bytes)[:length]
}
