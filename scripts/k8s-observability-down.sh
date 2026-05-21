#!/usr/bin/env sh
set -eu

if ! command -v kubectl >/dev/null 2>&1; then
  echo "kubectl not found; nothing to delete"
  exit 0
fi

echo "Deleting Kubernetes observability stack"
kubectl delete -k deploy/k8s-observability --ignore-not-found=true
