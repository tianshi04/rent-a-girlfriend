#!/bin/bash
set -euo pipefail

# Text formatting helper
info() {
  echo -e "\033[34m[INFO] $1\033[0m"
}
success() {
  echo -e "\033[32m[SUCCESS] $1\033[0m"
}
error() {
  echo -e "\033[31m[ERROR] $1\033[0m" >&2
}

NAMESPACE="identity-service"
RELEASE_NAME="identity-service"
CHART_DIR="$(dirname "$0")/deployments"
IMAGE_TAG="${1:-latest}"
IMAGE_REPO="${2:-local-registry:5000/identity-service}"

info "Starting deployment of identity-service (tag: ${IMAGE_TAG}) via Helm..."

# 1. Ensure the namespace exists
if ! kubectl get namespace "${NAMESPACE}" >/dev/null 2>&1; then
  info "Namespace '${NAMESPACE}' does not exist. Creating..."
  kubectl create namespace "${NAMESPACE}"
fi

# Ensure the namespace has helm ownership labels and annotations so helm can manage it
info "Applying Helm ownership metadata to namespace '${NAMESPACE}'..."
kubectl label namespace "${NAMESPACE}" app.kubernetes.io/managed-by=Helm --overwrite
kubectl annotate namespace "${NAMESPACE}" meta.helm.sh/release-name="${RELEASE_NAME}" --overwrite
kubectl annotate namespace "${NAMESPACE}" meta.helm.sh/release-namespace="${NAMESPACE}" --overwrite

# Ensure the namespace is enrolled in Istio Ambient mesh
info "Enrolling namespace '${NAMESPACE}' into Istio ambient mesh..."
kubectl label namespace "${NAMESPACE}" istio.io/dataplane-mode=ambient --overwrite

# Deploy/Apply Waypoint Proxy for this namespace
info "Applying Istio Waypoint Proxy for namespace '${NAMESPACE}'..."
istioctl waypoint apply --namespace "${NAMESPACE}" --enroll-namespace --wait

# 2. Deploy via Helm
info "Executing helm upgrade --install..."

HELM_ARGS=(
  --namespace "${NAMESPACE}"
  --set image.repository="${IMAGE_REPO}"
  --set image.tag="${IMAGE_TAG}"
  --wait
)

# Automatically load secrets.yaml if it exists locally
SECRETS_FILE="${CHART_DIR}/secrets.yaml"
if [ -f "${SECRETS_FILE}" ]; then
  info "Found local secrets file: '${SECRETS_FILE}'. Appending to Helm command..."
  HELM_ARGS+=("-f" "${SECRETS_FILE}")
fi

helm upgrade --install "${RELEASE_NAME}" "${CHART_DIR}" "${HELM_ARGS[@]}"

# 3. Apply global Istio mesh configurations (PeerAuthentication, RequestAuthentication, EnvoyFilters)
info "Applying global Istio mesh policies..."
kubectl apply -f "$(dirname "$0")/../../infra/istio/"

success "identity-service successfully deployed to namespace '${NAMESPACE}'!"
