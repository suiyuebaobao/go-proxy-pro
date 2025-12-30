#!/bin/bash
# 并发测试脚本
# 测试用户并发控制、账户并发控制、会话粘性

API_KEY="sk-3f6fe4a7d3eed623e4ccce34167d56073ca0c20443ad4214958a90b1c104af99"
BASE_URL="http://localhost:8080"
ADMIN_TOKEN=$(cat /tmp/test_token.txt)

# 简单的请求体
REQUEST_BODY='{
  "model": "claude-sonnet-4-20250514",
  "max_tokens": 10,
  "messages": [{"role": "user", "content": "Say hi"}]
}'

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=========================================="
echo "  Go-AIProxy 并发测试"
echo "=========================================="
echo ""

# 测试1: 单个请求测试连通性
echo -e "${YELLOW}[测试1] 单个请求测试连通性${NC}"
echo "发送1个请求..."
RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/v1/messages" \
  -H "Content-Type: application/json" \
  -H "x-api-key: $API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -d "$REQUEST_BODY" 2>&1)

HTTP_CODE=$(echo "$RESP" | tail -1)
BODY=$(echo "$RESP" | head -n -1)

if [ "$HTTP_CODE" == "200" ]; then
  echo -e "${GREEN}[OK] 单请求成功 (HTTP $HTTP_CODE)${NC}"
else
  echo -e "${RED}[FAIL] 单请求失败 (HTTP $HTTP_CODE)${NC}"
  echo "Response: $BODY"
fi
echo ""

# 测试2: 查看缓存状态
echo -e "${YELLOW}[测试2] 查看缓存状态${NC}"
echo "账号缓存:"
curl -s "$BASE_URL/api/admin/cache/accounts" -H "Authorization: Bearer $ADMIN_TOKEN"
echo ""
echo "用户缓存:"
curl -s "$BASE_URL/api/admin/cache/users" -H "Authorization: Bearer $ADMIN_TOKEN"
echo ""
echo ""

# 测试3: 并发请求测试（用户限制为2）
echo -e "${YELLOW}[测试3] 并发请求测试 (发送5个并发请求，用户限制为2)${NC}"
echo "期望: 2个成功，3个被拒绝(429)"
echo ""

# 创建临时目录存储结果
RESULT_DIR=$(mktemp -d)

# 发送5个并发请求
for i in {1..5}; do
  (
    RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/v1/messages" \
      -H "Content-Type: application/json" \
      -H "x-api-key: $API_KEY" \
      -H "anthropic-version: 2023-06-01" \
      -H "x-session-id: test-session-$i" \
      -d "$REQUEST_BODY" 2>&1)

    HTTP_CODE=$(echo "$RESP" | tail -1)
    echo "$HTTP_CODE" > "$RESULT_DIR/result_$i.txt"
  ) &
done

# 等待所有请求完成
wait

# 统计结果
SUCCESS=0
REJECTED=0
OTHER=0

for i in {1..5}; do
  CODE=$(cat "$RESULT_DIR/result_$i.txt" 2>/dev/null || echo "000")
  if [ "$CODE" == "200" ]; then
    ((SUCCESS++))
    echo -e "  请求$i: ${GREEN}HTTP $CODE (成功)${NC}"
  elif [ "$CODE" == "429" ]; then
    ((REJECTED++))
    echo -e "  请求$i: ${YELLOW}HTTP $CODE (并发限制)${NC}"
  else
    ((OTHER++))
    echo -e "  请求$i: ${RED}HTTP $CODE (其他)${NC}"
  fi
done

echo ""
echo "统计: 成功=$SUCCESS, 被限制=$REJECTED, 其他=$OTHER"

# 清理
rm -rf "$RESULT_DIR"
echo ""

# 测试4: 再次查看缓存状态
echo -e "${YELLOW}[测试4] 再次查看缓存状态${NC}"
echo "账号缓存:"
curl -s "$BASE_URL/api/admin/cache/accounts" -H "Authorization: Bearer $ADMIN_TOKEN"
echo ""
echo "用户缓存:"
curl -s "$BASE_URL/api/admin/cache/users" -H "Authorization: Bearer $ADMIN_TOKEN"
echo ""
echo ""

# 测试5: 会话粘性测试
echo -e "${YELLOW}[测试5] 会话粘性测试${NC}"
echo "使用相同session-id发送3次请求，应该绑定到同一账户"
SESSION_ID="sticky-test-$(date +%s)"

for i in {1..3}; do
  echo "请求 $i (session: $SESSION_ID):"
  curl -s -X POST "$BASE_URL/v1/messages" \
    -H "Content-Type: application/json" \
    -H "x-api-key: $API_KEY" \
    -H "anthropic-version: 2023-06-01" \
    -H "x-session-id: $SESSION_ID" \
    -d "$REQUEST_BODY" -o /dev/null -w "HTTP %{http_code}\n"
  sleep 1
done

echo ""
echo "查看会话绑定:"
curl -s "$BASE_URL/api/admin/cache/accounts" -H "Authorization: Bearer $ADMIN_TOKEN"
echo ""
echo ""

echo "=========================================="
echo "  测试完成"
echo "=========================================="
