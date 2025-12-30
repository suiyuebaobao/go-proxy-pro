package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	baseURL = "http://localhost:8080"
	apiKey  = "sk-3f6fe4a7d3eed623e4ccce34167d56073ca0c20443ad4214958a90b1c104af99"
)

func main() {
	fmt.Println("=== OpenAI Responses 并发测试 ===")
	fmt.Println("测试完成后请查看日志: logs/openai-responses-*.log")
	fmt.Println()

	// 测试1: 同一会话的粘性测试
	fmt.Println("【测试1】粘性会话测试 - 同一 Session ID 连续5次请求")
	testStickySession("sticky-test-001", 5)
	fmt.Println()

	// 测试2: 不同会话的负载均衡测试
	fmt.Println("【测试2】负载均衡测试 - 10个不同 Session ID")
	testLoadBalance(10)
	fmt.Println()

	// 测试3: 并发请求测试
	fmt.Println("【测试3】并发请求测试 - 10个并发请求")
	testConcurrent(10)
	fmt.Println()

	fmt.Println("=== 测试完成 ===")
	fmt.Println("请检查日志中 '选中账户' 的分布情况")
}

// testStickySession 测试粘性会话
func testStickySession(sessionID string, count int) {
	for i := 0; i < count; i++ {
		success, duration, err := sendRequest(sessionID)
		if success {
			fmt.Printf("  请求 %d: 成功, 耗时=%v\n", i+1, duration)
		} else {
			fmt.Printf("  请求 %d: 失败 - %s\n", i+1, err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// testLoadBalance 测试负载均衡
func testLoadBalance(count int) {
	for i := 0; i < count; i++ {
		sessionID := fmt.Sprintf("lb-session-%03d", i)
		success, duration, err := sendRequest(sessionID)
		if success {
			fmt.Printf("  请求 %d (session=%s): 成功, 耗时=%v\n", i+1, sessionID, duration)
		} else {
			fmt.Printf("  请求 %d: 失败 - %s\n", i+1, err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// testConcurrent 测试并发
func testConcurrent(count int) {
	var wg sync.WaitGroup
	successCount := 0
	failCount := 0
	var mu sync.Mutex

	start := time.Now()

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sessionID := fmt.Sprintf("concurrent-%03d", idx)
			success, _, _ := sendRequest(sessionID)
			mu.Lock()
			if success {
				successCount++
			} else {
				failCount++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(start)

	fmt.Printf("  成功: %d, 失败: %d, 总耗时: %v\n", successCount, failCount, totalDuration)
}

// sendRequest 发送测试请求
func sendRequest(sessionID string) (bool, time.Duration, string) {
	start := time.Now()

	reqBody := map[string]interface{}{
		"model":  "gpt-5.2",
		"stream": true,
		"instructions": "You are a helpful assistant.",
		"input": []map[string]interface{}{
			{"role": "user", "content": "hi"},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", baseURL+"/v1/responses", bytes.NewReader(bodyBytes))
	if err != nil {
		return false, time.Since(start), err.Error()
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", "codex_cli_rs/0.1.0")
	req.Header.Set("Session_id", sessionID)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, time.Since(start), err.Error()
	}
	defer resp.Body.Close()

	// 读取并丢弃响应体
	io.Copy(io.Discard, resp.Body)

	duration := time.Since(start)

	if resp.StatusCode == 200 {
		return true, duration, ""
	}
	return false, duration, fmt.Sprintf("status %d", resp.StatusCode)
}
