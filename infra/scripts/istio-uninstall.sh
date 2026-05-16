#!/bin/bash
set -euo pipefail

NAMESPACE="default"

echo "==> [1/4] Xóa Waypoint Proxy..."
istioctl waypoint delete --all --namespace ${NAMESPACE} || true

echo "==> [2/4] Xóa label ambient trên namespace..."
kubectl label namespace ${NAMESPACE} \
  istio.io/dataplane-mode- \
  istio.io/use-waypoint- \
  2>/dev/null || true

echo "==> [3/4] Uninstall Istio..."
istioctl uninstall --purge -y

echo "==> [4/4] Xóa namespace istio-system..."
kubectl delete namespace istio-system --ignore-not-found

echo "Done."
