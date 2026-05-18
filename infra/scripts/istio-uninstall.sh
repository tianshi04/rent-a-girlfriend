#!/bin/bash
set -euo pipefail

echo "==> [1/2] Uninstall Istio..."
istioctl uninstall --purge -y

echo "==> [2/2] Xóa namespace istio-system..."
kubectl delete namespace istio-system --ignore-not-found

echo "Done."
