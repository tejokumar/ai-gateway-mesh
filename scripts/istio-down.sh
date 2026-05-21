#!/usr/bin/env sh
set -eu

if ! command -v kubectl >/dev/null 2>&1; then
  echo "kubectl not found; nothing to delete"
  exit 0
fi

echo "Deleting AI Gateway Istio routing resources"
kubectl delete -k deploy/istio --ignore-not-found=true

if command -v istioctl >/dev/null 2>&1; then
  echo "Uninstalling Istio"
  istioctl uninstall --purge -y
fi

kubectl delete namespace istio-system --ignore-not-found=true
