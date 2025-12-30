package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
	"golang.org/x/net/proxy"
)

func main() {
	// 读取 token
	tokenBytes, err := os.ReadFile("/tmp/openai_token.txt")
	if err != nil {
		fmt.Printf("读取 token 失败: %v\n", err)
		os.Exit(1)
	}
	token := strings.TrimSpace(string(tokenBytes))
	fmt.Printf("Token 长度: %d\n", len(token))

	// 检查代理环境变量
	proxyEnv := os.Getenv("https_proxy")
	if proxyEnv == "" {
		proxyEnv = os.Getenv("HTTPS_PROXY")
	}
	if proxyEnv == "" {
		proxyEnv = os.Getenv("all_proxy")
	}
	if proxyEnv == "" {
		proxyEnv = os.Getenv("ALL_PROXY")
	}

	if proxyEnv != "" {
		fmt.Printf("使用代理: %s\n", proxyEnv)
	} else {
		fmt.Println("未检测到代理，直连模式")
	}

	// 构建请求体 - 使用 claude-relay 完全一样的 instructions
	reqBody := map[string]interface{}{
		"model":  "gpt-5.2",
		"stream": true,
		"instructions": "You are Codex, based on GPT-5. You are running as a coding agent in the Codex CLI on a user's computer.\n\n## General\n\n- When searching for text or files, prefer using `rg` or `rg --files` respectively because `rg` is much faster than alternatives like `grep`. (If the `rg` command is not found, then use alternatives.)\n\n## Editing constraints\n\n- Default to ASCII when editing or creating files. Only introduce non-ASCII or other Unicode characters when there is a clear justification and the file already uses them.\n- Add succinct code comments that explain what is going on if code is not self-explanatory. You should not add comments like \"Assigns the value to the variable\", but a brief comment might be useful ahead of a complex code block that the user would otherwise have to spend time parsing out. Usage of these comments should be rare.\n- Try to use apply_patch for single file edits, but it is fine to explore other options to make the edit if it does not work well. Do not use apply_patch for changes that are auto-generated (i.e. generating package.json or running a lint or format command like gofmt) or when scripting is more efficient (such as search and replacing a string across a codebase).\n- You may be in a dirty git worktree.\n    * NEVER revert existing changes you did not make unless explicitly requested, since these changes were made by the user.\n    * If asked to make a commit or code edits and there are unrelated changes to your work or changes that you didn't make in those files, don't revert those changes.\n    * If the changes are in files you've touched recently, you should read carefully and understand how you can work with the changes rather than reverting them.\n    * If the changes are in unrelated files, just ignore them and don't revert them.\n- Do not amend a commit unless explicitly requested to do so.\n- While you are working, you might notice unexpected changes that you didn't make. If this happens, STOP IMMEDIATELY and ask the user how they would like to proceed.\n- **NEVER** use destructive commands like `git reset --hard` or `git checkout --` unless specifically requested or approved by the user.\n\n## Presenting your work\n\n- Default: be very concise; friendly coding teammate tone.\n- For code changes: Lead with a quick explanation of the change, and then give more details on the context covering where and why a change was made.\n- Don't dump large files you've written; reference paths only.",
		"input": []map[string]interface{}{
			{"role": "user", "content": "say hi"},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("序列化请求失败: %v\n", err)
		os.Exit(1)
	}

	// 创建带 Chrome TLS 指纹的 HTTP 客户端
	client := createChromeTLSClient(proxyEnv)

	// 创建 HTTP 请求
	targetURL := "https://chatgpt.com/backend-api/codex/responses"
	req, err := http.NewRequestWithContext(context.Background(), "POST", targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		fmt.Printf("创建请求失败: %v\n", err)
		os.Exit(1)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "codex_cli_rs/0.1.0")
	req.Header.Set("openai-beta", "responses=experimental")
	req.Header.Set("chatgpt-account-id", "650ff8e7-5c6d-4f3b-8feb-a3c867621d76")

	fmt.Printf("\n发送请求到: %s\n", targetURL)
	fmt.Printf("Model: %s\n", reqBody["model"])
	fmt.Println("---")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	fmt.Printf("响应状态: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("协议: %s\n", resp.Proto)
	fmt.Println("---")

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("错误响应: %s\n", string(body))
		os.Exit(1)
	}

	// 读取流式响应
	fmt.Println("流式响应:")
	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				fmt.Println("\n[DONE]")
				break
			}
			var event map[string]interface{}
			if err := json.Unmarshal([]byte(data), &event); err == nil {
				if eventType, ok := event["type"].(string); ok {
					if eventType == "response.output_text.delta" {
						if delta, ok := event["delta"].(string); ok {
							fmt.Print(delta)
						}
					} else if eventType == "response.completed" {
						if response, ok := event["response"].(map[string]interface{}); ok {
							if usage, ok := response["usage"].(map[string]interface{}); ok {
								fmt.Printf("\n\nUsage: input=%v, output=%v\n",
									usage["input_tokens"], usage["output_tokens"])
							}
						}
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("读取响应错误: %v\n", err)
	}

	fmt.Println("\n测试完成!")
}

// createChromeTLSClient 创建带 Chrome TLS 指纹的 HTTP 客户端
func createChromeTLSClient(proxyURL string) *http.Client {
	dialTLS := func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
		// 解析地址
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			host = addr
		}

		var conn net.Conn

		if proxyURL != "" {
			// 使用代理
			pURL, err := url.Parse(proxyURL)
			if err != nil {
				return nil, fmt.Errorf("parse proxy url: %w", err)
			}

			dialer, err := proxy.FromURL(pURL, proxy.Direct)
			if err != nil {
				return nil, fmt.Errorf("create proxy dialer: %w", err)
			}

			conn, err = dialer.Dial(network, addr)
			if err != nil {
				return nil, fmt.Errorf("proxy dial: %w", err)
			}
		} else {
			// 直连
			var d net.Dialer
			conn, err = d.DialContext(ctx, network, addr)
			if err != nil {
				return nil, fmt.Errorf("dial: %w", err)
			}
		}

		// 使用 uTLS Chrome 指纹
		tlsConn := utls.UClient(conn, &utls.Config{
			ServerName: host,
		}, utls.HelloChrome_Auto)

		if err := tlsConn.Handshake(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("tls handshake: %w", err)
		}

		return tlsConn, nil
	}

	// 创建 HTTP/2 Transport
	transport := &http2.Transport{
		DialTLSContext: dialTLS,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   120 * time.Second,
	}
}

// 备用：标准 TLS 客户端（不使用 Chrome 指纹）
func createStandardClient(proxyURL string) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{},
		Proxy: func(req *http.Request) (*url.URL, error) {
			if proxyURL != "" {
				return url.Parse(proxyURL)
			}
			return nil, nil
		},
	}

	// 配置 HTTP/2
	http2.ConfigureTransport(transport)

	return &http.Client{
		Transport: transport,
		Timeout:   120 * time.Second,
	}
}
