#!/usr/bin/env sh
set -eu

NAMESPACE="${K8S_NAMESPACE:-ai-gateway-mesh}"
PROM_PORT="${PROMETHEUS_LOCAL_PORT:-19090}"
GRAFANA_PORT="${GRAFANA_LOCAL_PORT:-3301}"
PROM_URL="http://127.0.0.1:${PROM_PORT}"
GRAFANA_URL="http://127.0.0.1:${GRAFANA_PORT}"
PROM_LOG="${TMPDIR:-/tmp}/ai-gateway-mesh-prometheus-port-forward.log"
GRAFANA_LOG="${TMPDIR:-/tmp}/ai-gateway-mesh-grafana-port-forward.log"

cleanup() {
  if [ "${PROM_PF_PID:-}" ]; then
    kill "$PROM_PF_PID" >/dev/null 2>&1 || true
    wait "$PROM_PF_PID" 2>/dev/null || true
  fi
  if [ "${GRAFANA_PF_PID:-}" ]; then
    kill "$GRAFANA_PF_PID" >/dev/null 2>&1 || true
    wait "$GRAFANA_PF_PID" 2>/dev/null || true
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

echo "Checking observability rollouts"
kubectl -n "$NAMESPACE" rollout status deployment/prometheus --timeout=120s
kubectl -n "$NAMESPACE" rollout status deployment/grafana --timeout=120s

echo "Starting temporary Prometheus port-forward on ${PROM_URL}"
kubectl -n "$NAMESPACE" port-forward svc/prometheus "${PROM_PORT}:9090" >"$PROM_LOG" 2>&1 &
PROM_PF_PID="$!"

echo "Starting temporary Grafana port-forward on ${GRAFANA_URL}"
kubectl -n "$NAMESPACE" port-forward svc/grafana "${GRAFANA_PORT}:3000" >"$GRAFANA_LOG" 2>&1 &
GRAFANA_PF_PID="$!"

prom_ready=0
grafana_ready=0
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if curl -fsS "${PROM_URL}/-/ready" >/dev/null 2>&1; then
    prom_ready=1
  fi
  if curl -fsS "${GRAFANA_URL}/api/health" >/dev/null 2>&1; then
    grafana_ready=1
  fi
  if [ "$prom_ready" -eq 1 ] && [ "$grafana_ready" -eq 1 ]; then
    break
  fi
  sleep 1
done

if [ "$prom_ready" -ne 1 ]; then
  echo "Prometheus did not become reachable" >&2
  cat "$PROM_LOG" >&2 || true
  exit 1
fi

if [ "$grafana_ready" -ne 1 ]; then
  echo "Grafana did not become reachable" >&2
  cat "$GRAFANA_LOG" >&2 || true
  exit 1
fi

echo "Checking Prometheus scrape target"
curl -fsS "${PROM_URL}/api/v1/targets" | grep -q 'ai-gateway-mesh'

echo "Checking Grafana datasource"
curl -fsS -u admin:ai-gateway "${GRAFANA_URL}/api/datasources/uid/prometheus" | grep -q '"name":"Prometheus"'

echo "Checking Grafana dashboard provisioning"
curl -fsS -u admin:ai-gateway "${GRAFANA_URL}/api/dashboards/uid/ai-gateway-mesh" | grep -q '"title":"AI Gateway Mesh"'

echo "Kubernetes observability smoke test passed."
