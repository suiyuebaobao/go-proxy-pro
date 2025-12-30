package main

import (
	"fmt"
	"os"
)

const (
	ClaudeAIURL      = "https://claude.ai"
	OAuthClientID    = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	OAuthRedirectURI = "https://console.anthropic.com/oauth/code/callback"
	OAuthTokenURL    = "https://console.anthropic.com/v1/oauth/token"
)

func main() {
	// 从环境变量获取 Session Key
	sessionKey := os.Getenv("ANTHROPIC_SESSION_KEY")
	if sessionKey == "" {
		fmt.Println("错误: 请设置环境变量 ANTHROPIC_SESSION_KEY")
		fmt.Println("\n使用方法:")
		fmt.Println("  export ANTHROPIC_SESSION_KEY='your-session-key'")
		fmt.Println("  ./oauth_tool")
		return
	}

	fmt.Println("OAuth 工具已准备就绪")
	fmt.Printf("Session Key: %s...%s\n", sessionKey[:10], sessionKey[len(sessionKey)-10:])
	fmt.Println("\n注意: 此工具需要完整实现才能使用")
	fmt.Println("当前版本仅作为示例")
}
