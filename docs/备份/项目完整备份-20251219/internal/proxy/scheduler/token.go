package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"go-aiproxy/internal/model"
	"go-aiproxy/internal/repository"
)

// TokenManager OAuth Token 管理器
type TokenManager struct {
	mu       sync.RWMutex
	repo     *repository.AccountRepository

	// Token 刷新配置
	refreshThreshold time.Duration // 提前刷新时间
	refreshing       map[uint]bool // 正在刷新的账户
}

var defaultTokenManager *TokenManager
var tokenManagerOnce sync.Once

// GetTokenManager 获取 Token 管理器单例
func GetTokenManager() *TokenManager {
	tokenManagerOnce.Do(func() {
		defaultTokenManager = &TokenManager{
			repo:             repository.NewAccountRepository(),
			refreshThreshold: 5 * time.Minute, // 提前5分钟刷新
			refreshing:       make(map[uint]bool),
		}
		// 启动后台刷新协程
		go defaultTokenManager.backgroundRefresh()
	})
	return defaultTokenManager
}

// SetRefreshThreshold 设置提前刷新时间
func (m *TokenManager) SetRefreshThreshold(d time.Duration) {
	m.refreshThreshold = d
}

// CheckAndRefreshToken 检查并刷新 Token
func (m *TokenManager) CheckAndRefreshToken(ctx context.Context, account *model.Account) error {
	// 只处理有 Token 过期时间的账户
	if account.TokenExpiry == nil {
		return nil
	}

	// 检查是否需要刷新
	if time.Until(*account.TokenExpiry) > m.refreshThreshold {
		return nil
	}

	// 检查是否正在刷新
	m.mu.Lock()
	if m.refreshing[account.ID] {
		m.mu.Unlock()
		return nil
	}
	m.refreshing[account.ID] = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.refreshing, account.ID)
		m.mu.Unlock()
	}()

	// 根据账户类型刷新
	switch account.Type {
	case model.AccountTypeClaudeOfficial:
		return m.refreshClaudeOfficialToken(ctx, account)
	case model.AccountTypeGemini:
		return m.refreshGeminiToken(ctx, account)
	default:
		return nil
	}
}

// refreshClaudeOfficialToken 刷新 Claude Official Token
func (m *TokenManager) refreshClaudeOfficialToken(ctx context.Context, account *model.Account) error {
	if account.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	// Claude OAuth 刷新端点
	tokenURL := "https://claude.ai/api/auth/token"

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", account.RefreshToken)

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token,omitempty"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	// 更新账户
	expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	account.AccessToken = tokenResp.AccessToken
	account.TokenExpiry = &expiry
	if tokenResp.RefreshToken != "" {
		account.RefreshToken = tokenResp.RefreshToken
	}

	return m.repo.UpdateToken(account.ID, account.AccessToken, account.RefreshToken, &expiry)
}

// refreshGeminiToken 刷新 Gemini Token
func (m *TokenManager) refreshGeminiToken(ctx context.Context, account *model.Account) error {
	if account.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	// Google OAuth 刷新端点
	tokenURL := "https://oauth2.googleapis.com/token"

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", account.RefreshToken)
	// 需要 client_id 和 client_secret，从 APIKey 和 APISecret 获取
	if account.APIKey != "" {
		data.Set("client_id", account.APIKey)
	}
	if account.APISecret != "" {
		data.Set("client_secret", account.APISecret)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	// 更新账户
	expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	account.AccessToken = tokenResp.AccessToken
	account.TokenExpiry = &expiry

	return m.repo.UpdateToken(account.ID, account.AccessToken, account.RefreshToken, &expiry)
}

// backgroundRefresh 后台定期检查并刷新 Token
func (m *TokenManager) backgroundRefresh() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.refreshAllExpiring()
	}
}

// refreshAllExpiring 刷新所有即将过期的 Token
func (m *TokenManager) refreshAllExpiring() {
	// 获取所有启用的账户
	accounts, err := m.repo.GetAllEnabled()
	if err != nil {
		return
	}

	ctx := context.Background()
	for _, account := range accounts {
		if account.TokenExpiry != nil && time.Until(*account.TokenExpiry) < m.refreshThreshold {
			go m.CheckAndRefreshToken(ctx, &account)
		}
	}
}

// ForceRefresh 强制刷新指定账户的 Token
func (m *TokenManager) ForceRefresh(ctx context.Context, accountID uint) error {
	account, err := m.repo.GetByID(accountID)
	if err != nil {
		return err
	}

	switch account.Type {
	case model.AccountTypeClaudeOfficial:
		return m.refreshClaudeOfficialToken(ctx, account)
	case model.AccountTypeGemini:
		return m.refreshGeminiToken(ctx, account)
	default:
		return fmt.Errorf("account type %s does not support token refresh", account.Type)
	}
}
