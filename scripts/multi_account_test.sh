#!/bin/bash
# 多账号测试脚本

API_KEY="sk-3f6fe4a7d3eed623e4ccce34167d56073ca0c20443ad4214958a90b1c104af99"
BASE_URL="http://localhost:8080"
TOKEN=$(cat /tmp/test_token.txt)

echo "=========================================="
echo "  多账号测试"
echo "=========================================="

# 清除缓存
echo -e "\n[准备] 清除旧缓存..."
redis-cli keys "session:*" 2>/dev/null | xargs -r redis-cli del >/dev/null 2>&1
redis-cli keys "concurrency:*" 2>/dev/null | xargs -r redis-cli del >/dev/null 2>&1
redis-cli keys "user:concurrency:*" 2>/dev/null | xargs -r redis-cli del >/dev/null 2>&1
echo "缓存已清除"

# 测试1: 发送10个串行请求
echo -e "\n[测试1] 发送10个串行请求，观察账号分配和响应..."
for i in {1..10}; do
  RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/v1/messages" \
    -H "Content-Type: application/json" \
    -H "x-api-key: $API_KEY" \
    -H "anthropic-version: 2023-06-01" \
    -H "x-session-id: serial-test-$i" \
    -d '{"model":"claude-sonnet-4-20250514","max_tokens":5,"messages":[{"role":"user","content":"say ok"}]}' 2>&1)

  HTTP_CODE=$(echo "$RESP" | tail -1)
  BODY=$(echo "$RESP" | head -n -1)

  # 提取错误信息（如果有）
  if [ "$HTTP_CODE" != "200" ]; then
    ERROR=$(echo "$BODY" | grep -o '"message":"[^"]*"' | head -1)
    echo "  请求$i: HTTP $HTTP_CODE - $ERROR"
  else
    echo "  请求$i: HTTP $HTTP_CODE (成功)"
  fi
done

# 测试2: 查看账号分配
echo -e "\n[测试2] 查看缓存中的账号分配..."
CACHE=$(curl -s "$BASE_URL/api/admin/cache/accounts" -H "Authorization: Bearer $TOKEN")
echo "$CACHE" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    accounts = data['data']['accounts']
    print(f'  活跃账号数: {len(accounts)}')
    for acc in accounts:
        print(f'  - 账号ID:{acc[\"account_id\"]} | 会话数:{acc[\"session_count\"]} | 用户:{acc[\"users\"]}')
except:
    print('  解析失败')
" 2>/dev/null || echo "  $CACHE"

# 测试3: 查看账号状态
echo -e "\n[测试3] 查看账号当前状态..."
curl -s "$BASE_URL/api/admin/accounts" -H "Authorization: Bearer $TOKEN" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    for acc in data['data']['items']:
        if acc['enabled']:
            err = acc.get('last_error', '')[:50] if acc.get('last_error') else '无'
            print(f'  ID:{acc[\"id\"]} | {acc[\"name\"][:15]:15} | 请求:{acc[\"request_count\"]:3} | 错误:{acc[\"error_count\"]:3} | 最后错误:{err}')
except Exception as e:
    print(f'  解析失败: {e}')
" 2>/dev/null

# 测试4: 并发测试
echo -e "\n[测试4] 发送5个并发请求..."
RESULT_DIR=$(mktemp -d)
for i in {1..5}; do
  (
    RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/v1/messages" \
      -H "Content-Type: application/json" \
      -H "x-api-key: $API_KEY" \
      -H "anthropic-version: 2023-06-01" \
      -H "x-session-id: concurrent-$i" \
      -d '{"model":"claude-sonnet-4-20250514","max_tokens":5,"messages":[{"role":"user","content":"ok"}]}' 2>&1)
    echo "$RESP" | tail -1 > "$RESULT_DIR/$i.txt"
  ) &
done
wait

echo "  结果:"
for i in {1..5}; do
  CODE=$(cat "$RESULT_DIR/$i.txt" 2>/dev/null)
  echo "    请求$i: HTTP $CODE"
done
rm -rf "$RESULT_DIR"

# 测试5: 查看日志中的账号选择
echo -e "\n[测试5] 查看最近的调度日志..."
grep -E "(选中账户|会话粘性|账户并发)" /root/go-aiproxy/logs/scheduler-2025-12-14.log 2>/dev/null | tail -10 || echo "  无调度日志"

echo -e "\n=========================================="
echo "  测试完成"
echo "=========================================="
