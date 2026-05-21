#!/usr/bin/env sh
set -eu

NAMESPACE="${K8S_NAMESPACE:-ai-gateway-mesh}"
LOCAL_PORT="${AI_GATEWAY_LOCAL_PORT:-18080}"
BASE_URL="${AI_GATEWAY_BASE_URL:-http://127.0.0.1:${LOCAL_PORT}}"
PF_LOG="${TMPDIR:-/tmp}/ai-gateway-mesh-port-forward.log"

cleanup() {
  if [ "${PF_PID:-}" ]; then
    kill "$PF_PID" >/dev/null 2>&1 || true
    wait "$PF_PID" 2>/dev/null || true
  fi
}

trap cleanup EXIT INT TERM

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

need curl
need kubectl

echo "Checking Kubernetes rollouts"
kubectl -n "$NAMESPACE" rollout status deployment/ai-gateway --timeout=120s
kubectl -n "$NAMESPACE" rollout status deployment/vllm-small --timeout=120s
kubectl -n "$NAMESPACE" rollout status deployment/vllm-large --timeout=120s
kubectl -n "$NAMESPACE" rollout status deployment/vllm-coder --timeout=120s

echo "Starting temporary port-forward on ${BASE_URL}"
kubectl -n "$NAMESPACE" port-forward "svc/ai-gateway" "${LOCAL_PORT}:8080" >"$PF_LOG" 2>&1 &
PF_PID="$!"

ready=0
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if curl -fsS "${BASE_URL}/healthz" >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 1
done

if [ "$ready" -ne 1 ]; then
  echo "gateway did not become reachable through port-forward" >&2
  echo "port-forward log:" >&2
  cat "$PF_LOG" >&2 || true
  exit 1
fi

echo "Checking health and readiness"
curl -fsS "${BASE_URL}/healthz" >/dev/null
curl -fsS "${BASE_URL}/readyz" >/dev/null

echo "Checking admin backends"
curl -fsS "${BASE_URL}/admin/backends" | grep -q '"api_key_env":"configured"'

echo "Sending non-streaming chat completion"
curl -fsS "${BASE_URL}/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{"model":"auto","messages":[{"role":"user","content":"Explain Kubernetes service mesh in one sentence"}],"stream":false}' \
  | grep -q '"object": "chat.completion"'

echo "Sending streaming chat completion"
curl -fsS -N "${BASE_URL}/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{"model":"auto","messages":[{"role":"user","content":"Write a tiny Go function"}],"stream":true}' \
  | grep -q 'data: \[DONE\]'

echo "Checking metrics"
curl -fsS "${BASE_URL}/metrics" | grep -q 'ai_gateway_requests_total'

echo "Kubernetes smoke test passed."
