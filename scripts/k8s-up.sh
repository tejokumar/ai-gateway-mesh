#!/usr/bin/env sh
set -eu

CLUSTER_NAME="${KIND_CLUSTER_NAME:-ai-gateway-mesh}"
CONTEXT_NAME="kind-${CLUSTER_NAME}"
GATEWAY_IMAGE="${IMAGE:-ai-gateway-mesh:local}"
MOCK_IMAGE="${MOCK_IMAGE:-ai-gateway-mock-openai:latest}"

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

need docker
need kind
need kubectl

if kind get clusters | grep -Fx "$CLUSTER_NAME" >/dev/null 2>&1; then
  echo "Using existing kind cluster: $CLUSTER_NAME"
  kubectl config use-context "$CONTEXT_NAME" >/dev/null
else
  echo "Creating kind cluster: $CLUSTER_NAME"
  kind create cluster --name "$CLUSTER_NAME"
fi

echo "Building images"
docker build -t "$GATEWAY_IMAGE" .
docker build -t "$MOCK_IMAGE" tests/mock-openai-server

echo "Loading images into kind"
kind load docker-image "$GATEWAY_IMAGE" --name "$CLUSTER_NAME"
kind load docker-image "$MOCK_IMAGE" --name "$CLUSTER_NAME"

echo "Applying Kubernetes manifests"
kubectl apply -k deploy/k8s

echo "Waiting for deployments"
kubectl -n ai-gateway-mesh rollout status deployment/ai-gateway --timeout=120s
kubectl -n ai-gateway-mesh rollout status deployment/vllm-small --timeout=120s
kubectl -n ai-gateway-mesh rollout status deployment/vllm-large --timeout=120s
kubectl -n ai-gateway-mesh rollout status deployment/vllm-coder --timeout=120s

echo "Kubernetes demo is ready."
echo "Run: kubectl -n ai-gateway-mesh port-forward svc/ai-gateway 18080:8080"
