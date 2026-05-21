#!/usr/bin/env sh
set -eu

NAMESPACE="${K8S_NAMESPACE:-ai-gateway-mesh}"

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

need kubectl

echo "Ensuring namespace exists"
kubectl apply -f deploy/k8s/namespace.yaml

echo "Applying Kubernetes observability stack"
kubectl apply -k deploy/k8s-observability

echo "Waiting for observability deployments"
kubectl -n "$NAMESPACE" rollout status deployment/prometheus --timeout=120s
kubectl -n "$NAMESPACE" rollout status deployment/grafana --timeout=120s

echo "Kubernetes observability is ready."
echo "Run: make k8s-grafana-port-forward"
echo "Grafana login: admin / ai-gateway"
