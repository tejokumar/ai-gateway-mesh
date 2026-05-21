#!/usr/bin/env sh
set -eu

NAMESPACE="${K8S_NAMESPACE:-ai-gateway-mesh}"
LOCAL_PORT="${ISTIO_INGRESS_LOCAL_PORT:-18081}"
BASE_URL="${ISTIO_BASE_URL:-http://127.0.0.1:${LOCAL_PORT}}"
HOST="${ISTIO_GATEWAY_HOST:-ai-gateway.local}"
PF_LOG="${TMPDIR:-/tmp}/ai-gateway-mesh-istio-port-forward.log"

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

echo "Checking Istio and gateway rollouts"
kubectl -n istio-system rollout status deployment/istiod --timeout=180s
kubectl -n istio-system rollout status deployment/istio-ingressgateway --timeout=180s
kubectl -n "$NAMESPACE" rollout status deployment/ai-gateway --timeout=120s

echo "Starting temporary Istio ingress port-forward on ${BASE_URL}"
kubectl -n istio-system port-forward svc/istio-ingressgateway "${LOCAL_PORT}:80" >"$PF_LOG" 2>&1 &
PF_PID="$!"

ready=0
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if curl -fsS -H "Host: ${HOST}" "${BASE_URL}/healthz" >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 1
done

if [ "$ready" -ne 1 ]; then
  echo "Istio ingress did not become reachable" >&2
  echo "port-forward log:" >&2
  cat "$PF_LOG" >&2 || true
  exit 1
fi

echo "Checking health and readiness through Istio"
curl -fsS -H "Host: ${HOST}" "${BASE_URL}/healthz" >/dev/null
curl -fsS -H "Host: ${HOST}" "${BASE_URL}/readyz" >/dev/null

echo "Sending non-streaming chat completion through Istio"
curl -fsS -H "Host: ${HOST}" "${BASE_URL}/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{"model":"auto","messages":[{"role":"user","content":"Explain Istio ingress for an AI gateway"}],"stream":false}' \
  | grep -q '"object": "chat.completion"'

echo "Sending streaming chat completion through Istio"
curl -fsS -N -H "Host: ${HOST}" "${BASE_URL}/v1/chat/completions" \
  -H "Content-Type: application/json" \
  -d '{"model":"auto","messages":[{"role":"user","content":"Write a tiny mesh haiku"}],"stream":true}' \
  | grep -q 'data: \[DONE\]'

echo "Checking metrics through Istio"
curl -fsS -H "Host: ${HOST}" "${BASE_URL}/metrics" | grep -q 'ai_gateway_requests_total'

echo "Istio smoke test passed."
