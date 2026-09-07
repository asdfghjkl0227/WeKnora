#!/usr/bin/env bash
# CI 初始化：注册管理员账号，并配置 embedding + LLM 模型。
#
# 依赖环境变量（由 GitHub Secrets 提供）：
#   WEKNORA_BASE_URL          服务地址
#   WEKNORA_ADMIN_EMAIL       管理员邮箱
#   WEKNORA_ADMIN_PASSWORD    管理员密码
#   EMBEDDING_MODEL_NAME      embedding 模型名
#   EMBEDDING_BASE_URL        embedding 服务地址
#   EMBEDDING_API_KEY         embedding 服务密钥
#   LLM_MODEL_NAME            LLM 模型名
#   LLM_BASE_URL              LLM 服务地址
#   LLM_API_KEY               LLM 服务密钥
set -euo pipefail

BASE_URL="${WEKNORA_BASE_URL:-http://localhost:8080}"
EMAIL="${WEKNORA_ADMIN_EMAIL:?缺少 WEKNORA_ADMIN_EMAIL}"
PASSWORD="${WEKNORA_ADMIN_PASSWORD:?缺少 WEKNORA_ADMIN_PASSWORD}"

# 1. 注册管理员（账号已存在时忽略报错）
curl -sf -X POST "$BASE_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"admin\",\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" >/dev/null 2>&1 || true

# 2. 登录拿 token
TOKEN=$(curl -sf -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" \
  | python3 -c "import sys,json;print(json.load(sys.stdin)['token'])")

# 3. 配置 embedding 模型
if [ -n "${EMBEDDING_MODEL_NAME:-}" ] && [ -n "${EMBEDDING_API_KEY:-}" ]; then
  curl -sf -X POST "$BASE_URL/api/v1/models" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"$EMBEDDING_MODEL_NAME\",\"type\":\"Embedding\",\"source\":\"remote\",\"parameters\":{\"base_url\":\"${EMBEDDING_BASE_URL:-}\",\"api_key\":\"$EMBEDDING_API_KEY\"}}" >/dev/null
  echo "已配置 embedding 模型：$EMBEDDING_MODEL_NAME"
fi

# 4. 配置 LLM 模型
if [ -n "${LLM_MODEL_NAME:-}" ] && [ -n "${LLM_API_KEY:-}" ]; then
  curl -sf -X POST "$BASE_URL/api/v1/models" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"$LLM_MODEL_NAME\",\"type\":\"KnowledgeQA\",\"source\":\"remote\",\"parameters\":{\"base_url\":\"${LLM_BASE_URL:-}\",\"api_key\":\"$LLM_API_KEY\"}}" >/dev/null
  echo "已配置 LLM 模型：$LLM_MODEL_NAME"
fi
