#!/bin/bash
set -euo pipefail

ISTIO_VERSION="1.29.2"
GATEWAY_API_VERSION="v1.5.0"
NAMESPACE="default"

echo "==> [1/5] Gateway API CRDs ${GATEWAY_API_VERSION}..."
kubectl apply --server-side -f \
  https://github.com/kubernetes-sigs/gateway-api/releases/download/${GATEWAY_API_VERSION}/standard-install.yaml

echo "==> [2/5] Istio ${ISTIO_VERSION} ambient profile..."
istioctl install \
  --set profile=ambient \
  --set tag=${ISTIO_VERSION} \
  --skip-confirmation

echo "==> [3/5] Chờ istio-system pods ready..."
kubectl wait --for=condition=ready pod \
  --all -n istio-system \
  --timeout=120s

echo "==> [4/5] Enroll namespace '${NAMESPACE}' vào ambient mesh..."
kubectl label namespace ${NAMESPACE} \
  istio.io/dataplane-mode=ambient \
  --overwrite

echo "==> [5/5] Deploy Waypoint Proxy cho namespace '${NAMESPACE}'..."
istioctl waypoint apply \
  --namespace ${NAMESPACE} \
  --enroll-namespace \
  --wait

echo ""
echo "Done! Verify: istioctl waypoint status -n ${NAMESPACE}"
