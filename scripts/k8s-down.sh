#!/usr/bin/env sh
set -eu

CLUSTER_NAME="${KIND_CLUSTER_NAME:-ai-gateway-mesh}"

if ! command -v kind >/dev/null 2>&1; then
  echo "kind not found; nothing to delete"
  exit 0
fi

if kind get clusters | grep -Fx "$CLUSTER_NAME" >/dev/null 2>&1; then
  echo "Deleting kind cluster: $CLUSTER_NAME"
  kind delete cluster --name "$CLUSTER_NAME"
else
  echo "kind cluster not found: $CLUSTER_NAME"
fi
