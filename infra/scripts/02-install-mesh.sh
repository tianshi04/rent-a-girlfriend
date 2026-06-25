#!/bin/bash
set -euo pipefail

ISTIO_VERSION="1.29.2"
GATEWAY_API_VERSION="v1.5.0"

echo "==> [1/3] Gateway API CRDs ${GATEWAY_API_VERSION}..."
kubectl apply --server-side -f \
  https://github.com/kubernetes-sigs/gateway-api/releases/download/${GATEWAY_API_VERSION}/standard-install.yaml

echo "==> [2/3] Istio ${ISTIO_VERSION} ambient profile..."
istioctl install \
  --set profile=ambient \
  --set tag=${ISTIO_VERSION} \
  --skip-confirmation

echo "==> [3/3] Chờ istio-system pods ready..."
kubectl wait --for=condition=ready pod \
  --all -n istio-system \
  --timeout=120s

echo ""
echo "Done! Istio control plane installed successfully."
