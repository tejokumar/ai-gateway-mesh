#!/usr/bin/env sh
set -eu

BASE_URL="${AI_GATEWAY_BASE_URL:-http://127.0.0.1:8080}"

post_chat() {
  model="$1"
  prompt="$2"
  stream="$3"

  curl -sS "$BASE_URL/v1/chat/completions" \
    -H "Content-Type: application/json" \
    -d "{\"model\":\"$model\",\"messages\":[{\"role\":\"user\",\"content\":\"$prompt\"}],\"stream\":$stream}" >/dev/null
}

echo "Sending demo traffic to $BASE_URL"

post_chat "auto" "Explain service mesh in simple terms" "false"
post_chat "auto" "Write golang code for an HTTP health check" "false"
post_chat "auto" "Explain kubernetes scheduling" "true"
post_chat "general-large" "Summarize LLM gateway fallback behavior" "false"

echo "Done. Open Grafana: http://127.0.0.1:3300/d/ai-gateway-mesh/ai-gateway-mesh"
