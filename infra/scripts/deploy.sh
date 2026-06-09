#!/bin/bash
set -euo pipefail

ENV=${1:-dev}

if [[ "$ENV" != "dev" && "$ENV" != "prod" ]]; then
  echo "Error: Environment must be 'dev' or 'prod'"
  exit 1
fi

echo "Deploying base infra and Kafka for environment: ${ENV}..."

# Add Homebrew to PATH so Kustomize can find the helm binary
export PATH="/home/linuxbrew/.linuxbrew/bin:$PATH"

# Deploy using Kustomize and pipe to kubectl apply
kubectl kustomize "infra/k8s/envs/${ENV}/" --enable-helm | kubectl apply -f -

echo "Base infra and Kafka deployed successfully for ${ENV}"