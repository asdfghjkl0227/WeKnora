#!/usr/bin/env bash
# 在 CI 中运行一次评测：登录 → 发起评测 → 轮询直到完成 → 输出指标 JSON 文件。
#
# 依赖环境变量：
#   WEKNORA_BASE_URL          服务地址（默认 http://localhost:8080）
#   WEKNORA_ADMIN_EMAIL       管理员邮箱
#   WEKNORA_ADMIN_PASSWORD    管理员密码
#   EVALUATION_RESULT_FILE    结果输出文件（默认 evaluation-result.json）
#   EVALUATION_MAX_WAIT_SECONDS 最长等待秒数（默认 300）
set -euo pipefail

BASE_URL="${WEKNORA_BASE_URL:-http://localhost:8080}"
EMAIL="${WEKNORA_ADMIN_EMAIL:?缺少 WEKNORA_ADMIN_EMAIL}"
PASSWORD="${WEKNORA_ADMIN_PASSWORD:?缺少 WEKNORA_ADMIN_PASSWORD}"
RESULT_FILE="${EVALUATION_RESULT_FILE:-evaluation-result.json}"
MAX_WAIT_SECONDS="${EVALUATION_MAX_WAIT_SECONDS:-300}"

# 1. 登录拿 token
TOKEN=$(curl -sf -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")

# 2. 发起评测（使用内置默认数据集 + 默认模型）
RESP=$(curl -sf -X POST "$BASE_URL/api/v1/evaluation" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}')
TASK_ID=$(printf '%s' "$RESP" | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['task']['id'])")
echo "评测任务 ID: $TASK_ID"

# 3. 轮询直到完成（status: 0=pending 1=running 2=success 3=failed）
START=$(date +%s)
while true; do
  RESULT=$(curl -sf "$BASE_URL/api/v1/evaluation?task_id=$TASK_ID" \
    -H "Authorization: Bearer $TOKEN")
  STATUS=$(printf '%s' "$RESULT" | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['task']['status'])")

  if [ "$STATUS" = "2" ]; then
    printf '%s' "$RESULT" | python3 -c "import sys,json;print(json.dumps(json.load(sys.stdin)['data']['metric'], ensure_ascii=False, indent=2))" > "$RESULT_FILE"
    echo "评测完成，指标结果如下："
    cat "$RESULT_FILE"
    exit 0
  fi

  if [ "$STATUS" = "3" ]; then
    echo "评测失败：$RESULT" >&2
    exit 1
  fi

  NOW=$(date +%s)
  if [ $((NOW - START)) -gt "$MAX_WAIT_SECONDS" ]; then
    echo "评测超时（超过 ${MAX_WAIT_SECONDS} 秒）" >&2
    exit 1
  fi
  sleep 5
done
