/*
 * 文件作用：OAuth认证服务，处理Claude OAuth授权流程
 * 负责功能：
 *   - 授权URL生成
 *   - 授权码交换Token
 *   - TLS指纹伪装
 *   - PKCE验证器生成
 * 重要程度：⭐⭐⭐⭐ 重要（OAuth授权核心）
 * 依赖模块：model, logger
 */
package service

import (
	"bytes"
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
	"time"

	"go-aiproxy/internal/model"
	"go-aiproxy/pkg/logger"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/proxy"
)

const (
	ClaudeAIURL      = "https://claude.ai"
	OAuthClientID    = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	OAuthRedirectURI = "https://console.anthropic.com/oauth/code/callback"
	OAuthTokenURL    = "https://console.anthropic.com/v1/oauth/token"
)

// OAuthAuthService OAuth 认证服务
type OAuthAuthService struct{}

// NewOAuthAuthService 创建 OAuth 认证服务
func NewOAuthAuthService() *OAuthAuthService {
	return &OAuthAuthService{}
}

// ReauthorizeWithSessionKey 使用 SessionKey 重新获取 OAuth Token
// 返回值：
// - *OAuthTokenResult: 成功时返回新 Token
// - error: 失败时返回错误
// - 特殊错误 ErrAccountBanned 表示账号已被封禁，不应再尝试刷新
func (s *OAuthAuthService) ReauthorizeWithSessionKey(ctx context.Context, account *model.Account) (*OAuthTokenResult, error) {
	log := logger.GetLogger("oauth_auth")

	sessionKey := account.SessionKey
	if sessionKey == "" {
		return nil, fmt.Errorf("账号没有 SessionKey")
	}

	// 获取代理配置
	proxyConfig := s.getProxyConfig(account)

	// 1. 先验证 SessionKey 是否有效（检查账号是否被封）
	log.Info("[%s] 验证 SessionKey 有效性...", account.Name)
	orgUUID, err := s.getOrganizationInfo(ctx, sessionKey, proxyConfig)
	if err != nil {
		// SessionKey 无效，可能账号被封禁
		log.Warn("[%s] SessionKey 验证失败，账号可能已被封禁: %v", account.Name, err)
		return nil, &AccountBannedError{
			AccountName: account.Name,
			Reason:      fmt.Sprintf("SessionKey 验证失败: %v", err),
		}
	}
	log.Info("[%s] SessionKey 有效，组织 UUID: %s", account.Name, orgUUID)

	// 2. 执行 OAuth 授权
	verifier, challenge := s.generatePKCE()
	state := s.generateRandomString(32)

	log.Info("[%s] 执行 OAuth 授权...", account.Name)
	authCode, err := s.authorizeWithCookie(ctx, sessionKey, orgUUID, challenge, state, proxyConfig)
	if err != nil {
		return nil, fmt.Errorf("OAuth 授权失败: %v", err)
	}
	log.Info("[%s] 获取到授权码", account.Name)

	// 3. 交换 Token
	log.Info("[%s] 交换 Token...", account.Name)
	tokenData, err := s.exchangeToken(ctx, authCode, verifier, state, proxyConfig)
	if err != nil {
		return nil, fmt.Errorf("Token 交换失败: %v", err)
	}

	log.Info("[%s] OAuth 重新授权成功，scope: %s", account.Name, tokenData.Scope)
	return tokenData, nil
}

// AccountBannedError 账号被封禁错误
type AccountBannedError struct {
	AccountName string
	Reason      string
}

func (e *AccountBannedError) Error() string {
	return fmt.Sprintf("账号 %s 可能已被封禁: %s", e.AccountName, e.Reason)
}

// IsAccountBannedError 判断是否为账号封禁错误
func IsAccountBannedError(err error) bool {
	_, ok := err.(*AccountBannedError)
	return ok
}

// OAuthTokenResult OAuth Token 结果
type OAuthTokenResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
	OrgUUID      string `json:"organization_uuid"`
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	Type     string
	Host     string
	Port     int
	Username string
	Password string
}

func (s *OAuthAuthService) getProxyConfig(account *model.Account) *ProxyConfig {
	// 优先使用账号关联的代理
	if account.Proxy != nil && account.Proxy.Enabled {
		return &ProxyConfig{
			Type:     account.Proxy.Type,
			Host:     account.Proxy.Host,
			Port:     account.Proxy.Port,
			Username: account.Proxy.Username,
			Password: account.Proxy.Password,
		}
	}

	// 获取默认代理
	defaultProxy, err := GetProxyService().GetDefaultProxy()
	if err == nil && defaultProxy != nil {
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

func (s *OAuthAuthService) dialWithProxy(network, addr string, proxyConfig *ProxyConfig) (net.Conn, error) {
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
	default:
		return net.DialTimeout(network, addr, 30*time.Second)
	}
}

func (s *OAuthAuthService) dialTLSWithChrome(network, addr string, proxyConfig *ProxyConfig) (net.Conn, error) {
	conn, err := s.dialWithProxy(network, addr, proxyConfig)
	if err != nil {
		return nil, err
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	config := &utls.Config{
		ServerName:         host,
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
		NextProtos:         []string{"http/1.1"},
	}

	tlsConn := utls.UClient(conn, config, utls.HelloCustom)

	spec, err := utls.UTLSIdToSpec(utls.HelloChrome_120)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get Chrome spec: %v", err)
	}

	for i, ext := range spec.Extensions {
		if _, ok := ext.(*utls.ALPNExtension); ok {
			spec.Extensions[i] = &utls.ALPNExtension{
				AlpnProtocols: []string{"http/1.1"},
			}
			break
		}
	}

	if err := tlsConn.ApplyPreset(&spec); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to apply spec: %v", err)
	}

	if err := tlsConn.Handshake(); err != nil {
		conn.Close()
		return nil, err
	}

	return tlsConn, nil
}

func (s *OAuthAuthService) createHTTPClient(proxyConfig *ProxyConfig) *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return s.dialWithProxy(network, addr, proxyConfig)
		},
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return s.dialTLSWithChrome(network, addr, proxyConfig)
		},
		DisableKeepAlives: true,
	}
	return &http.Client{Transport: transport, Timeout: 60 * time.Second}
}

func (s *OAuthAuthService) setChromeHeaders(req *http.Request, cookie string) {
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Origin", ClaudeAIURL)
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Referer", ClaudeAIURL+"/new")
	req.Header.Set("sec-ch-ua", `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
}

func (s *OAuthAuthService) getOrganizationInfo(ctx context.Context, sessionKey string, proxyConfig *ProxyConfig) (string, error) {
	client := s.createHTTPClient(proxyConfig)
	req, _ := http.NewRequestWithContext(ctx, "GET", ClaudeAIURL+"/api/organizations", nil)

	cookie := sessionKey
	if !strings.HasPrefix(sessionKey, "sessionKey=") {
		cookie = "sessionKey=" + sessionKey
	}
	s.setChromeHeaders(req, cookie)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		bodyLen := len(body)
		if bodyLen > 500 {
			bodyLen = 500
		}
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body)[:bodyLen])
	}

	var orgs []struct {
		UUID         string   `json:"uuid"`
		Capabilities []string `json:"capabilities"`
	}
	json.Unmarshal(body, &orgs)

	for _, org := range orgs {
		for _, cap := range org.Capabilities {
			if cap == "chat" {
				return org.UUID, nil
			}
		}
	}
	return "", fmt.Errorf("未找到有效组织")
}

func (s *OAuthAuthService) authorizeWithCookie(ctx context.Context, sessionKey, orgUUID, challenge, state string, proxyConfig *ProxyConfig) (string, error) {
	client := s.createHTTPClient(proxyConfig)
	authorizeURL := fmt.Sprintf("https://claude.ai/v1/oauth/%s/authorize", orgUUID)

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

	req, _ := http.NewRequestWithContext(ctx, "POST", authorizeURL, bytes.NewReader(jsonBody))
	cookie := sessionKey
	if !strings.HasPrefix(sessionKey, "sessionKey=") {
		cookie = "sessionKey=" + sessionKey
	}
	s.setChromeHeaders(req, cookie)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		bodyLen := len(body)
		if bodyLen > 500 {
			bodyLen = 500
		}
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body)[:bodyLen])
	}

	var result struct {
		RedirectURI string `json:"redirect_uri"`
		Code        string `json:"code"`
	}
	json.Unmarshal(body, &result)

	if result.RedirectURI != "" {
		u, _ := url.Parse(result.RedirectURI)
		code := u.Query().Get("code")
		respState := u.Query().Get("state")
		if code != "" {
			if respState != "" {
				return code + "#" + respState, nil
			}
			return code, nil
		}
	}

	if result.Code != "" {
		return result.Code, nil
	}

	return "", fmt.Errorf("未获取到授权码: %s", string(body))
}

func (s *OAuthAuthService) exchangeToken(ctx context.Context, code, verifier, state string, proxyConfig *ProxyConfig) (*OAuthTokenResult, error) {
	client := s.createHTTPClient(proxyConfig)

	parts := strings.Split(code, "#")
	authCode := parts[0]
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

	req, _ := http.NewRequestWithContext(ctx, "POST", OAuthTokenURL, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "claude-cli/2.0.30 (external, cli)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var tokenData struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		Scope        string `json:"scope"`
		Organization struct {
			UUID string `json:"uuid"`
		} `json:"organization"`
	}
	json.Unmarshal(body, &tokenData)

	return &OAuthTokenResult{
		AccessToken:  tokenData.AccessToken,
		RefreshToken: tokenData.RefreshToken,
		ExpiresIn:    tokenData.ExpiresIn,
		Scope:        tokenData.Scope,
		OrgUUID:      tokenData.Organization.UUID,
	}, nil
}

func (s *OAuthAuthService) generatePKCE() (verifier, challenge string) {
	b := make([]byte, 32)
	rand.Read(b)
	verifier = base64.RawURLEncoding.EncodeToString(b)
	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(hash[:])
	return
}

func (s *OAuthAuthService) generateRandomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)[:length]
}

// 单例
var oauthAuthService *OAuthAuthService

func GetOAuthAuthService() *OAuthAuthService {
	if oauthAuthService == nil {
		oauthAuthService = NewOAuthAuthService()
	}
	return oauthAuthService
}
