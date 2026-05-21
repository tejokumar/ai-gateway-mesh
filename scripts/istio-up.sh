#!/usr/bin/env sh
set -eu

NAMESPACE="${K8S_NAMESPACE:-ai-gateway-mesh}"

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

need istioctl
need kubectl

echo "Installing Istio demo profile"
istioctl install --set profile=demo -y

echo "Applying Istio routing manifests"
kubectl apply -k deploy/istio

echo "Waiting for Istio control plane and ingress"
kubectl -n istio-system rollout status deployment/istiod --timeout=180s
kubectl -n istio-system rollout status deployment/istio-ingressgateway --timeout=180s

echo "Waiting for AI Gateway deployment"
kubectl -n "$NAMESPACE" rollout status deployment/ai-gateway --timeout=120s

echo "Istio demo is ready."
echo "Run: make istio-smoke"
